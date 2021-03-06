// +build !ignore_autogenerated

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1

import (
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"kafka-operator/pkg/apis/jianzhiunique/v1.Kafka":       schema_pkg_apis_jianzhiunique_v1_Kafka(ref),
		"kafka-operator/pkg/apis/jianzhiunique/v1.KafkaSpec":   schema_pkg_apis_jianzhiunique_v1_KafkaSpec(ref),
		"kafka-operator/pkg/apis/jianzhiunique/v1.KafkaStatus": schema_pkg_apis_jianzhiunique_v1_KafkaStatus(ref),
	}
}

func schema_pkg_apis_jianzhiunique_v1_Kafka(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "Kafka is the Schema for the kafkas API",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("kafka-operator/pkg/apis/jianzhiunique/v1.KafkaSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("kafka-operator/pkg/apis/jianzhiunique/v1.KafkaStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta", "kafka-operator/pkg/apis/jianzhiunique/v1.KafkaSpec", "kafka-operator/pkg/apis/jianzhiunique/v1.KafkaStatus"},
	}
}

func schema_pkg_apis_jianzhiunique_v1_KafkaSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "KafkaSpec defines the desired state of Kafka",
				Type:        []string{"object"},
			},
		},
	}
}

func schema_pkg_apis_jianzhiunique_v1_KafkaStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "KafkaStatus defines the observed state of Kafka",
				Type:        []string{"object"},
			},
		},
	}
}
