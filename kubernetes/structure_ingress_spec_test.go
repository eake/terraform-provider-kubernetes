package kubernetes

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/networking/v1"
)

// Test Flatteners
func TestFlattenIngressRule(t *testing.T) {
	p := v1.ServiceBackendPort{
		Name:   "foo",
		Number: 1234,
	}
	s := v1.IngressServiceBackend{
		Name: "foo",
		Port: p,
	}
	r := v1.HTTPIngressRuleValue{
		Paths: []v1.HTTPIngressPath{
			{
				Path: "/foo/bar",
				Backend: v1.IngressBackend{
					Service: &s,
				},
			},
		},
	}

	in := []v1.IngressRule{
		{
			Host: "the-app-name.staging.live.domain-replaced.tld",
			IngressRuleValue: v1.IngressRuleValue{
				HTTP: (*v1.HTTPIngressRuleValue)(nil),
			},
		},
		{
			Host: "",
			IngressRuleValue: v1.IngressRuleValue{
				HTTP: (*v1.HTTPIngressRuleValue)(&r),
			},
		},
	}
	out := []interface{}{
		map[string]interface{}{
			"host": "the-app-name.staging.live.domain-replaced.tld",
			"http": []interface{}{},
		},
		map[string]interface{}{
			"host": "",
			"http": []interface{}{
				map[string]interface{}{
					"path": []interface{}{
						map[string]interface{}{
							"path": "/foo/bar",
							"backend": []interface{}{
								map[string]interface{}{
									"service_name": "foo",
									"service_port": "1234",
								},
							},
						},
					},
				},
			},
		},
	}

	flatRules := flattenIngressRule(in)

	if len(flatRules) < len(out) {
		t.Error("Failed to flatten ingress rules")
	}

	for i, v := range flatRules {
		control := v.(map[string]interface{})
		sample := out[i]

		if !reflect.DeepEqual(control, sample) {
			t.Errorf("Unexpected result:\n\tWant:%s\n\tGot:%s\n", control, sample)
		}
	}
}
