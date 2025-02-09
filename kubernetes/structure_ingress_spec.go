package kubernetes

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	//"k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Flatteners

func flattenIngressRule(in []v1.IngressRule) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for i, r := range in {
		m := make(map[string]interface{})

		m["host"] = r.Host
		m["http"] = flattenIngressRuleHttp(r.HTTP)
		att[i] = m
	}
	return att
}

func flattenIngressRuleHttp(in *v1.HTTPIngressRuleValue) []interface{} {
	if in == nil {
		return []interface{}{}
	}
	pathAtts := make([]interface{}, len(in.Paths), len(in.Paths))
	for i, p := range in.Paths {
		path := map[string]interface{}{
			"path":    p.Path,
			"backend": flattenIngressBackend(&p.Backend),
		}
		pathAtts[i] = path
	}

	httpAtt := map[string]interface{}{
		"path": pathAtts,
	}

	return []interface{}{httpAtt}
}

func flattenIngressBackend(in *v1.IngressBackend) []interface{} {
	att := make([]interface{}, 1, 1)

	m := make(map[string]interface{})
	m["service_name"] = in.Service.Name
	m["service_port"] = strconv.Itoa(int(in.Service.Port.Number))

	att[0] = m

	return att
}

func flattenIngressSpec(in v1.IngressSpec) []interface{} {
	att := make(map[string]interface{})

	if in.IngressClassName != nil {
		att["ingress_class_name"] = in.IngressClassName
	}

	if in.DefaultBackend != nil {
		att["backend"] = flattenIngressBackend(in.DefaultBackend)
	}

	if len(in.Rules) > 0 {
		att["rule"] = flattenIngressRule(in.Rules)
	}

	if len(in.TLS) > 0 {
		att["tls"] = flattenIngressTLS(in.TLS)
	}

	return []interface{}{att}
}

func flattenIngressTLS(in []v1.IngressTLS) []interface{} {
	att := make([]interface{}, len(in), len(in))

	for i, v := range in {
		m := make(map[string]interface{})
		m["hosts"] = v.Hosts
		m["secret_name"] = v.SecretName

		att[i] = m
	}

	return att
}

// Expanders

func expandIngressRule(l []interface{}) []v1.IngressRule {
	if len(l) == 0 || l[0] == nil {
		return []v1.IngressRule{}
	}
	obj := make([]v1.IngressRule, len(l), len(l))
	for i, n := range l {
		cfg := n.(map[string]interface{})

		var paths []v1.HTTPIngressPath

		if httpCfg, ok := cfg["http"]; ok {
			httpList := httpCfg.([]interface{})
			for _, h := range httpList {
				http := h.(map[string]interface{})
				if v, ok := http["path"]; ok {
					pathList := v.([]interface{})
					paths = make([]v1.HTTPIngressPath, len(pathList), len(pathList))
					for i, path := range pathList {
						p := path.(map[string]interface{})
						hip := v1.HTTPIngressPath{
							Path:    p["path"].(string),
							Backend: *expandIngressBackend(p["backend"].([]interface{})),
						}
						paths[i] = hip
					}
				}
			}
		}

		obj[i] = v1.IngressRule{
			Host: cfg["host"].(string),
			IngressRuleValue: v1.IngressRuleValue{
				HTTP: &v1.HTTPIngressRuleValue{
					Paths: paths,
				},
			},
		}
	}
	return obj
}

func expandIngressSpec(l []interface{}) v1.IngressSpec {
	if len(l) == 0 || l[0] == nil {
		return v1.IngressSpec{}
	}
	in := l[0].(map[string]interface{})
	obj := v1.IngressSpec{}

	if v, ok := in["ingress_class_name"].(string); ok && len(v) > 0 {
		obj.IngressClassName = &v
	}

	if v, ok := in["backend"].([]interface{}); ok && len(v) > 0 {
		obj.DefaultBackend = expandIngressBackend(v)
	}

	if v, ok := in["rule"].([]interface{}); ok && len(v) > 0 {
		obj.Rules = expandIngressRule(v)
	}

	if v, ok := in["tls"].([]interface{}); ok && len(v) > 0 {
		obj.TLS = expandIngressTLS(v)
	}

	return obj
}

func expandIngressBackend(l []interface{}) *v1.IngressBackend {
	if len(l) == 0 || l[0] == nil {
		return &v1.IngressBackend{}
	}
	in := l[0].(map[string]interface{})
	obj := &v1.IngressBackend{}

	if v, ok := in["service_name"].(string); ok {
		fmt.Println(in)
		obj.Service.Name = v
	}

	if v, ok := in["service_port"].(string); ok {
		obj.Service.Port.Number = intstr.Parse(v).IntVal
	}

	return obj
}

func expandIngressTLS(l []interface{}) []v1.IngressTLS {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tlsList := make([]v1.IngressTLS, len(l), len(l))
	for i, t := range l {
		in := t.(map[string]interface{})
		obj := v1.IngressTLS{}

		if v, ok := in["hosts"]; ok {
			obj.Hosts = expandStringSlice(v.([]interface{}))
		}

		if v, ok := in["secret_name"].(string); ok {
			obj.SecretName = v
		}
		tlsList[i] = obj
	}

	return tlsList
}

// Patch Ops

func patchIngressSpec(keyPrefix, pathPrefix string, d *schema.ResourceData) PatchOperations {
	ops := make([]PatchOperation, 0, 0)
	if d.HasChange(keyPrefix + "backend") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "backend",
			Value: expandIngressBackend(d.Get(keyPrefix + "backend").([]interface{})),
		})
	}

	if d.HasChange(keyPrefix + "rule") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "rules",
			Value: expandIngressRule(d.Get(keyPrefix + "rule").([]interface{})),
		})
	}

	if d.HasChange(keyPrefix + "tls") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "tls",
			Value: expandIngressTLS(d.Get(keyPrefix + "tls").([]interface{})),
		})
	}

	return ops
}
