package kafka

import (
	"context"
	"fmt"
	v1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	jianzhiuniquev1 "github.com/jianzhiunique/kafka-operator/pkg/apis/jianzhiunique/v1"
	"github.com/jianzhiunique/kafka-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	v1beta12 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"math/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strconv"
	"time"

	_ "github.com/jianzhiunique/kafka-operator/pkg/utils"
	_ "github.com/pravega/zookeeper-operator/pkg/apis"
	"github.com/pravega/zookeeper-operator/pkg/apis/zookeeper/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
)

var log = logf.Log.WithName("controller_kafka")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Kafka Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKafka{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("kafka-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Kafka
	err = c.Watch(&source.Kind{Type: &jianzhiuniquev1.Kafka{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource ZookeeperCluster and requeue the owner Kafka
	err = c.Watch(&source.Kind{Type: &v1beta1.ZookeeperCluster{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &jianzhiuniquev1.Kafka{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource StatefulSet and requeue the owner Kafka
	err = c.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &jianzhiuniquev1.Kafka{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Service and requeue the owner Kafka
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &jianzhiuniquev1.Kafka{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Deployment and requeue the owner Kafka
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &jianzhiuniquev1.Kafka{},
	})
	if err != nil {
		return err
	}

	//解除对ingress的监听，防止删除共用ingress？但需要考虑删除逻辑
	// Watch for changes to secondary resource Ingress and requeue the owner Kafka
	/*err = c.Watch(&source.Kind{Type: &v1beta12.Ingress{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &jianzhiuniquev1.Kafka{},
	})
	if err != nil {
		return err
	}*/

	// Watch for changes to secondary resource Deployment and requeue the owner Kafka
	err = c.Watch(&source.Kind{Type: &v1.ServiceMonitor{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &jianzhiuniquev1.Kafka{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileKafka implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKafka{}

// ReconcileKafka reconciles a Kafka object
type ReconcileKafka struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	log    logr.Logger
}

const ReconcileTime = 30 * time.Second

type reconcileFun func(cluster *jianzhiuniquev1.Kafka) error

// Reconcile reads that state of the cluster for a Kafka object and makes changes based on the state read
// and what is in the Kafka.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKafka) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	r.log = log.WithValues("Namespace", request.Namespace, "Name", request.Name)
	r.log.Info("调谐中")

	// Fetch the Kafka instance
	// 获取Kafka CR
	instance := &jianzhiuniquev1.Kafka{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, fmt.Errorf("获取CR失败: %s", err)
	}

	//check if default values will be used
	//设置默认的配置
	changed := utils.CheckCR(instance)

	if changed {
		r.log.Info("设置默认配置")
		if err := r.client.Update(context.TODO(), instance); err != nil {
			return reconcile.Result{}, fmt.Errorf("设置默认配置失败: %s", err)
		}
		//retry reconcile
		return reconcile.Result{Requeue: true}, nil
	}

	// 生成常量值
	if instance.Status.KafkaProxyUrl == "" {
		instance.Status.KafkaProxyUrl = "kfk-mqp-svc-" + instance.Name + ":8080"
	}

	if instance.Status.KafkaUrl == "" {
		instance.Status.KafkaUrl = "kfk-svc-" + instance.Name
	}
	if instance.Status.KafkaPort == "" {
		instance.Status.KafkaPort = "9092"
	}
	if instance.Status.KafkaManagerUsername == "" {
		instance.Status.KafkaManagerUsername = "admin"
	}

	if instance.Status.KafkaManagerPassword == "" {
		instance.Status.KafkaManagerPassword = GetRandomString(16)
	}

	if instance.Status.KafkaManagerPath == "" {
		instance.Status.KafkaManagerPath = "/kfk-" + instance.Namespace + "-" + instance.Name + "/"
	}

	if instance.Status.KafkaToolsPath == "" {
		instance.Status.KafkaToolsPath = "/" + instance.Namespace + "-" + instance.Name + "-kafka/"
	}

	if instance.Status.KafkaManagerUrl == "" {
		if instance.Spec.KafkaManagerHostAlias == "" {
			instance.Status.KafkaManagerUrl = instance.Spec.KafkaManagerHost + instance.Status.KafkaToolsPath
		} else {
			instance.Status.KafkaManagerUrl = instance.Spec.KafkaManagerHostAlias + instance.Status.KafkaToolsPath
		}
	}

	if instance.Status.ZkUrl == "" {
		instance.Status.ZkUrl = "kfk-zk-" + instance.Name + "-client:2181"
	}

	//sts_name.svc_name.namespace.svc.cluster.local
	var i int32
	if instance.Status.KafkaUrlAll == "" {
		for i = 0; i < instance.Spec.Size; i++ {
			instance.Status.KafkaUrlAll += "kfk-sts-" + instance.Name + "-" + strconv.FormatInt(int64(i), 10) + "."
			instance.Status.KafkaUrlAll += "kfk-svc-" + instance.Name + "."
			instance.Status.KafkaUrlAll += instance.Namespace + ".svc.cluster.local:9092"
			if i != instance.Spec.Size-1 {
				instance.Status.KafkaUrlAll += ","
			}
		}
	}

	if instance.Status.ZkUrlAll == "" {
		for i = 0; i < instance.Spec.ZkSize; i++ {
			instance.Status.ZkUrlAll += "kfk-zk-" + instance.Name + "-" + strconv.FormatInt(int64(i), 10) + "."
			instance.Status.ZkUrlAll += "kfk-zk-" + instance.Name + "-headless."
			instance.Status.ZkUrlAll += instance.Namespace + ".svc.cluster.local:2181"
			if i != instance.Spec.ZkSize-1 {
				instance.Status.ZkUrlAll += ","
			}
		}
	}

	//生成常量值后马上保存一次状态
	if err = r.reconcileClusterStatus(instance); err != nil {
		return reconcile.Result{}, err
	}

	// reconcile
	//由于ingress原因reconcileKafkaManager与reconcileMQManagementTools有先后顺序
	for _, fun := range []reconcileFun{
		r.reconcileFinalizers,
		r.reconcileZooKeeper,
		r.reconcileKafka,
		r.reconcileKafkaManager,
		r.reconcileMQManagementTools,
		r.reconcileKafkaProxy,
		r.reconcileKafkaExporter,
		r.reconcileServiceMonitor,
	} {
		if err = fun(instance); err != nil {
			r.reconcileClusterStatus(instance)
			return reconcile.Result{}, err
		} else {
			r.reconcileClusterStatus(instance)
		}
	}

	return reconcile.Result{}, nil
}

func GetRandomString(l int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

func (r *ReconcileKafka) reconcileFinalizers(instance *jianzhiuniquev1.Kafka) (err error) {
	r.log.Info("最终调谐")
	// instance is not deleted
	if instance.DeletionTimestamp.IsZero() {
		if !utils.ContainsString(instance.ObjectMeta.Finalizers, utils.KafkaFinalizer) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, utils.KafkaFinalizer)
			if err = r.client.Update(context.TODO(), instance); err != nil {
				return err
			}
		}
		return r.cleanupOrphanPVCs(instance)
	} else {
		r.log.Info("最终调谐", "时间戳", instance.DeletionTimestamp)
		// instance is deleted
		if utils.ContainsString(instance.ObjectMeta.Finalizers, utils.KafkaFinalizer) {
			if err = r.cleanUpAllPVCs(instance); err != nil {
				return err
			}

			//删除ingress path
			foundIngress := &v1beta12.Ingress{}
			err = r.client.Get(context.TODO(), types.NamespacedName{Name: "mq-ingress", Namespace: instance.Spec.IngressNamespace}, foundIngress)

			if err != nil && errors.IsNotFound(err) {

			} else if err != nil {

			} else {
				utils.DeleteKafkaManagerPathFromIngress(instance, foundIngress)
				utils.DeleteKafkaToolsPathFromIngress(instance, foundIngress)
				err = r.client.Update(context.TODO(), foundIngress)
				if err != nil {
					return fmt.Errorf("删除ingress path出现问题: %s", err)
				}
			}

			instance.ObjectMeta.Finalizers = utils.RemoveString(instance.ObjectMeta.Finalizers, utils.KafkaFinalizer)
			if err = r.client.Update(context.TODO(), instance); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *ReconcileKafka) getPVCCount(instance *jianzhiuniquev1.Kafka) (pvcCount int, err error) {
	pvcList, err := r.getPVCList(instance)
	if err != nil {
		return -1, err
	}
	pvcCount = len(pvcList.Items)
	return pvcCount, nil
}

func (r *ReconcileKafka) cleanupOrphanPVCs(instance *jianzhiuniquev1.Kafka) (err error) {
	// this check should make sure we do not delete the PVCs before the STS has scaled down
	// 确保不会在伸缩时删掉PVC
	if instance.Status.Replicas == instance.Spec.Size {
		pvcCount, err := r.getPVCCount(instance)
		if err != nil {
			return err
		}

		if pvcCount > int(instance.Spec.Size) {
			r.log.Info("清理孤儿PVC", "当前PVC数", pvcCount, "副本数", instance.Status.Replicas)
			pvcList, err := r.getPVCList(instance)
			if err != nil {
				return err
			}
			for _, pvcItem := range pvcList.Items {
				// delete only Orphan PVCs
				if utils.IsPVCOrphan(pvcItem.Name, instance.Spec.Size) {
					r.deletePVC(pvcItem)
				}
			}
		}
	}
	return nil
}

func (r *ReconcileKafka) getPVCList(instance *jianzhiuniquev1.Kafka) (pvList corev1.PersistentVolumeClaimList, err error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "kfk-pod-" + instance.Name},
	})
	pvclistOps := &client.ListOptions{
		Namespace:     instance.Namespace,
		LabelSelector: selector,
	}
	pvcList := &corev1.PersistentVolumeClaimList{}
	err = r.client.List(context.TODO(), pvcList, pvclistOps)
	return *pvcList, err
}

func (r *ReconcileKafka) cleanUpAllPVCs(instance *jianzhiuniquev1.Kafka) (err error) {
	pvcList, err := r.getPVCList(instance)
	if err != nil {
		return err
	}
	for _, pvcItem := range pvcList.Items {
		r.deletePVC(pvcItem)
	}
	return nil
}

func (r *ReconcileKafka) deletePVC(pvcItem corev1.PersistentVolumeClaim) {
	pvcDelete := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcItem.Name,
			Namespace: pvcItem.Namespace,
		},
	}
	r.log.Info("删除PVC", "名称", pvcItem.Name)
	err := r.client.Delete(context.TODO(), pvcDelete)
	if err != nil {
		r.log.Error(err, "删除PVC失败", "名称", pvcDelete.Name)
	}
}

func (r *ReconcileKafka) reconcileZooKeeper(instance *jianzhiuniquev1.Kafka) (err error) {
	r.log.Info("调谐zk")
	zk := utils.NewZkForCR(instance)
	// Set zk instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, zk, r.scheme); err != nil {
		return fmt.Errorf("设置zk控制器引用失败: %s", err)
	}

	//检查zk是否存在
	foundzk := &v1beta1.ZookeeperCluster{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: zk.Name, Namespace: zk.Namespace}, foundzk)
	//如果sts不存在,新建
	if err != nil && errors.IsNotFound(err) {
		r.log.Info("创建zk", "名称", zk.Name)

		err = r.client.Create(context.TODO(), zk)
		if err != nil {
			return fmt.Errorf("创建zk失败: %s", err)
		}
		//创建zk成功,继续进行
	} else if err != nil {
		//有异常
		return fmt.Errorf("获取zk失败: %s", err)
	} else {
		//检查是否有变化，如果有变化，则Update
		//对于zk，目前只更新节点数
		if zk.Spec.Replicas != foundzk.Spec.Replicas {
			foundzk.Spec.Replicas = zk.Spec.Replicas
			err = r.client.Update(context.TODO(), foundzk)
			if err != nil {
				return fmt.Errorf("更新zk失败: %s", err)
			}
		}
	}

	//检查zk是否就绪
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: zk.Name, Namespace: zk.Namespace}, foundzk)
	if err != nil {
		return fmt.Errorf("检查可用获取zk失败: %s", err)
	}
	if foundzk.Status.ReadyReplicas != zk.Spec.Replicas {
		instance.Status.Progress = float32(foundzk.Status.ReadyReplicas) / float32(zk.Spec.Replicas) * 0.3
		r.log.Info("Zk未就绪", "名称", zk.Name)
		return fmt.Errorf("Zk未就绪")
	}
	r.log.Info("Zk已就绪", "名称", zk.Name)

	return nil
}

func (r *ReconcileKafka) reconcileKafka(instance *jianzhiuniquev1.Kafka) (err error) {
	r.log.Info("调谐kafka")
	sts := utils.NewStsForCR(instance)
	// Set Kafka instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, sts, r.scheme); err != nil {
		return fmt.Errorf("设置kafka控制器引用失败: %s", err)
	}

	//检查sts是否存在
	found := &appsv1.StatefulSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: sts.Name, Namespace: sts.Namespace}, found)
	//如果sts不存在,新建
	if err != nil && errors.IsNotFound(err) {
		r.log.Info("创建sts", "名称", sts.Name)
		err = r.client.Create(context.TODO(), sts)
		if err != nil {
			return fmt.Errorf("创建sts失败: %s", err)
		}
	} else if err != nil {
		//有异常
		return fmt.Errorf("获取sts失败: %s", err)
	} else {
		utils.SyncKafkaSts(found, sts)
		err = r.client.Update(context.TODO(), found)
		if err != nil {
			return fmt.Errorf("更新sts失败: %s", err)
		}
	}

	//检查kfk是否可用
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: sts.Name, Namespace: sts.Namespace}, found)
	if err != nil {
		return fmt.Errorf("检查可用获取sts失败: %s", err)
	}
	if found.Status.ReadyReplicas != instance.Spec.Size {
		r.log.Info("kafka未就绪", "名称", sts.Name)
		instance.Status.Progress = float32(found.Status.ReadyReplicas)/float32(found.Status.Replicas)*0.3 + 0.3
		return fmt.Errorf("kafka未就绪")
	}
	r.log.Info("kafka已就绪", "名称", sts.Name)
	//update KafkaReplicas when kafka is ready
	instance.Status.Replicas = instance.Spec.Size

	//创建kafka service
	svc := utils.NewSvcForCR(instance)
	if err := controllerutil.SetControllerReference(instance, svc, r.scheme); err != nil {
		return fmt.Errorf("设置kafka svc控制器引用失败: %s", err)
	}
	foundSvc := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, foundSvc)

	if err != nil && errors.IsNotFound(err) {
		r.log.Info("创建kafka headless svc", "名称", svc.Name)
		err = r.client.Create(context.TODO(), svc)
		if err != nil {
			return fmt.Errorf("创建kafka headless svc失败: %s", err)
		}
	} else if err != nil {
		return fmt.Errorf("获取kafka headless svc失败: %s", err)
	}

	return nil
}

func (r *ReconcileKafka) reconcileKafkaManager(instance *jianzhiuniquev1.Kafka) (err error) {
	r.log.Info("调谐kafka manager")
	km := utils.NewKafkaManagerForCR(instance)

	if err := controllerutil.SetControllerReference(instance, km, r.scheme); err != nil {
		return fmt.Errorf("设置kafka manager控制器引用失败: %s", err)
	}
	foundKm := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: km.Name, Namespace: km.Namespace}, foundKm)

	if err != nil && errors.IsNotFound(err) {
		r.log.Info("创建kafka manager", "名称", km.Name)
		err = r.client.Create(context.TODO(), km)
		if err != nil {
			return fmt.Errorf("创建kafka manager失败: %s", err)
		}
	} else if err != nil {
		return fmt.Errorf("获取kafka manager失败: %s", err)
	}

	kmsvc := utils.NewKmSvcForCR(instance)
	if err := controllerutil.SetControllerReference(instance, kmsvc, r.scheme); err != nil {
		return fmt.Errorf("设置kafka manager svc控制器引用失败: %s", err)
	}
	foundKmSvc := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: kmsvc.Name, Namespace: kmsvc.Namespace}, foundKmSvc)

	if err != nil && errors.IsNotFound(err) {
		r.log.Info("创建kafka manager svc", "名称", kmsvc.Name)
		err = r.client.Create(context.TODO(), kmsvc)
		if err != nil {
			return fmt.Errorf("创建kafka manager svc失败: %s", err)
		}
	} else if err != nil {
		return fmt.Errorf("获取kafka manager svc失败: %s", err)
	}

	//如果资源所在的ns 与 ingress所在的ns不同，需要额外创建ExternalName类型的svc
	if instance.Namespace != instance.Spec.IngressNamespace {
		external := utils.NewKmExternalSvcForCR(instance)
		//关联控制
		if err := controllerutil.SetControllerReference(instance, external, r.scheme); err != nil {
			return fmt.Errorf("设置Kafka manager external svc控制器引用失败: %s", err)
		}
		//检查是否已经存在
		foundExternal := &corev1.Service{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: external.Name, Namespace: external.Namespace}, foundExternal)

		if err != nil && errors.IsNotFound(err) {
			//如果不存在新建
			r.log.Info("创建Kafka manager external svc", "名称", external.Name)
			err = r.client.Create(context.TODO(), external)
			if err != nil {
				return fmt.Errorf("创建Kafka manager external svc失败: %s", err)
			}
		} else if err != nil {
			//如果发生错误重新调谐
			return fmt.Errorf("获取Kafka manager external svc失败: %s", err)
		}
	}

	kmi := utils.NewIngressForCRIfNotExists(instance)
	/*if err := controllerutil.SetControllerReference(instance, kmi, r.scheme); err != nil {
		return fmt.Errorf("SET Kafka Owner fail : %s", err)
	}*/
	foundKmi := &v1beta12.Ingress{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: kmi.Name, Namespace: kmi.Namespace}, foundKmi)

	if err != nil && errors.IsNotFound(err) {
		r.log.Info("创建kafka manager ingress", "名称", kmi.Name)
		err = r.client.Create(context.TODO(), kmi)
		if err != nil {
			return fmt.Errorf("创建kafka manager ingress失败: %s", err)
		}
		instance.Status.Progress = 0.65
	} else if err != nil {
		return fmt.Errorf("获取kafka manager ingress失败: %s", err)
	} else {
		utils.AppendKafkaManagerPathToIngress(instance, foundKmi)
		err = r.client.Update(context.TODO(), foundKmi)
		if err != nil {
			return fmt.Errorf("追加ingress path失败: %s", err)
		}
		instance.Status.Progress = 0.65
	}

	return nil
}

func (r *ReconcileKafka) reconcileKafkaProxy(instance *jianzhiuniquev1.Kafka) (err error) {
	r.log.Info("调谐kafka proxy")
	//check
	dep := utils.NewProxyForCR(instance)
	if err := controllerutil.SetControllerReference(instance, dep, r.scheme); err != nil {
		return fmt.Errorf("设置proxy控制器引用失败: %s", err)
	}
	found := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		r.log.Info("创建proxy", "名称", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			return fmt.Errorf("创建proxy失败: %s", err)
		}
	} else if err != nil {
		return fmt.Errorf("获取proxy失败: %s", err)
	}

	//check svc
	svc := utils.NewMqpSvcForCR(instance)
	if err := controllerutil.SetControllerReference(instance, svc, r.scheme); err != nil {
		return fmt.Errorf("设置proxy svc控制器引用失败: %s", err)
	}
	foundSvc := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, foundSvc)

	if err != nil && errors.IsNotFound(err) {
		r.log.Info("创建proxy svc", "名称", svc.Name)
		err = r.client.Create(context.TODO(), svc)
		if err != nil {
			return fmt.Errorf("创建proxy svc失败: %s", err)
		}
		instance.Status.Progress = 1.0
	} else if err != nil {
		return fmt.Errorf("获取proxy svc失败: %s", err)
	}

	instance.Status.Progress = 1.0

	return nil
}

func (r *ReconcileKafka) reconcileKafkaExporter(instance *jianzhiuniquev1.Kafka) (err error) {
	r.log.Info("调谐kafka exporter")
	//check
	dep := utils.NewExporterForCR(instance)
	if err := controllerutil.SetControllerReference(instance, dep, r.scheme); err != nil {
		return fmt.Errorf("设置exporter控制器引用失败: %s", err)
	}
	found := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		r.log.Info("创建exporter", "名称", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			return fmt.Errorf("创建exporter失败: %s", err)
		}
	} else if err != nil {
		return fmt.Errorf("获取exporter失败: %s", err)
	}

	//check svc
	svc := utils.NewExporterSvcForCR(instance)
	if err := controllerutil.SetControllerReference(instance, svc, r.scheme); err != nil {
		return fmt.Errorf("设置exporter svc控制器引用失败: %s", err)
	}
	foundSvc := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, foundSvc)

	if err != nil && errors.IsNotFound(err) {
		r.log.Info("创建exporter svc", "名称", svc.Name)
		err = r.client.Create(context.TODO(), svc)
		if err != nil {
			return fmt.Errorf("创建exporter svc 失败: %s", err)
		}
		instance.Status.Progress = 1.0
	} else if err != nil {
		return fmt.Errorf("获取exporter svc失败: %s", err)
	}
	instance.Status.Progress = 1.0

	return nil
}

func (r *ReconcileKafka) reconcileServiceMonitor(instance *jianzhiuniquev1.Kafka) (err error) {
	r.log.Info("调谐service monitor")
	svcm := utils.NewSvcMonitorForCR(instance)
	if err := controllerutil.SetControllerReference(instance, svcm, r.scheme); err != nil {
		return fmt.Errorf("设置 service monitor控制器引用失败:%s", err)
	}
	foundSvcm := &v1.ServiceMonitor{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: svcm.Name, Namespace: svcm.Namespace}, foundSvcm)

	if err != nil && errors.IsNotFound(err) {
		r.log.Info("创建service monitor", "名称", svcm.Name)
		err = r.client.Create(context.TODO(), svcm)
		if err != nil {
			return fmt.Errorf("创建service monitor失败: %s", err)
		}
		instance.Status.Progress = 1.0
	} else if err != nil {
		return fmt.Errorf("获取service monitor失败: %s", err)
	}

	return nil
}

func (r *ReconcileKafka) reconcileClusterStatus(instance *jianzhiuniquev1.Kafka) (err error) {
	r.log.Info("调谐状态")
	return r.client.Status().Update(context.TODO(), instance)
}

func (r *ReconcileKafka) reconcileMQManagementTools(instance *jianzhiuniquev1.Kafka) error {
	r.log.Info("调谐kafka tools")
	//check
	dep := utils.NewToolsForCR(instance)
	if err := controllerutil.SetControllerReference(instance, dep, r.scheme); err != nil {
		return fmt.Errorf("设置管理工具控制器引用失败: %s", err)
	}
	found := &appsv1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		r.log.Info("创建管理工具", "名称", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			return fmt.Errorf("创建管理工具失败: %s", err)
		}
		instance.Status.Progress = 0.7
	} else if err != nil {
		return fmt.Errorf("获取管理工具失败: %s", err)
	}

	//check svc
	svc := utils.NewToolsSvcForCR(instance)
	if err := controllerutil.SetControllerReference(instance, svc, r.scheme); err != nil {
		return fmt.Errorf("设置管理工具svc控制器引用失败: %s", err)
	}
	foundSvc := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, foundSvc)

	if err != nil && errors.IsNotFound(err) {
		r.log.Info("创建管理工具 svc", "名称", svc.Name)
		err = r.client.Create(context.TODO(), svc)
		if err != nil {
			return fmt.Errorf("创建管理工具 svc失败: %s", err)
		}
		instance.Status.Progress = 0.75
	} else if err != nil {
		return fmt.Errorf("获取管理工具 svc失败: %s", err)
	}

	//如果资源所在的ns 与 ingress所在的ns不同，需要额外创建ExternalName类型的svc
	if instance.Namespace != instance.Spec.IngressNamespace {
		external := utils.NewToolsExternalSvcForCR(instance)
		//关联控制
		if err := controllerutil.SetControllerReference(instance, external, r.scheme); err != nil {
			return fmt.Errorf("设置管理工具external svc控制器引用失败: %s", err)
		}
		//检查是否已经存在
		foundExternal := &corev1.Service{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: external.Name, Namespace: external.Namespace}, foundExternal)

		if err != nil && errors.IsNotFound(err) {
			//如果不存在新建
			r.log.Info("创建管理工具external svc", "名称", external.Name)
			err = r.client.Create(context.TODO(), external)
			if err != nil {
				return fmt.Errorf("创建管理工具external svc失败: %s", err)
			}
		} else if err != nil {
			//如果发生错误重新调谐
			return fmt.Errorf("获取管理工具external svc失败: %s", err)
		}
	}

	//check ingress
	rmi := utils.NewIngressForCRIfNotExists(instance)
	/*if err := controllerutil.SetControllerReference(instance, rmi, r.scheme); err != nil {
		return fmt.Errorf("SET ingress Owner fail : %s", err)
	}*/
	foundKmi := &v1beta12.Ingress{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: rmi.Name, Namespace: rmi.Namespace}, foundKmi)

	if err != nil && errors.IsNotFound(err) {
		r.log.Info("未找到ingress", "名称", rmi.Name)
		return fmt.Errorf("未找到ingress")
	} else if err != nil {
		return fmt.Errorf("获取ingress失败: %s", err)
	} else {
		utils.AppendKafkaToolsPathToIngress(instance, foundKmi)
		err = r.client.Update(context.TODO(), foundKmi)
		if err != nil {
			return fmt.Errorf("追加管理工具ingress path失败: %s", err)
		}
		instance.Status.Progress = 0.8
	}

	return nil
}
