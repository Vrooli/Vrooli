package handlers

import (
	"net/url"
	"strings"

	"github.com/vrooli/api-core/uiselectors"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Only recordings of the declared application's origin adopt its identities.
// External websites, frames, and ambiguous aliases keep their recorded locator.
func symbolizeRecordingSelectors(flow *basworkflows.WorkflowDefinitionV2, manifest map[string]interface{}, origin string) {
	target, err := url.Parse(origin)
	if err != nil || target.Host == "" || flow == nil {
		return
	}
	inTarget := false
	var visit func(protoreflect.Message)
	visit = func(m protoreflect.Message) {
		m.Range(func(f protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			if f.IsMap() || f.IsList() {
				return true
			}
			if f.Kind() == protoreflect.MessageKind {
				visit(v.Message())
				return true
			}
			if f.Kind() == protoreflect.StringKind && (f.Name() == "selector" || strings.HasSuffix(string(f.Name()), "_selector")) {
				if ref := uiselectors.ReferenceFor(manifest, v.String()); ref != "" {
					m.Set(f, protoreflect.ValueOfString(ref))
				}
			}
			return true
		})
	}
	for _, node := range flow.Nodes {
		action := node.Action
		if action == nil {
			continue
		}
		if nav := action.GetNavigate(); nav != nil {
			u, e := url.Parse(nav.Url)
			inTarget = e == nil && u.Scheme == target.Scheme && u.Host == target.Host
			continue
		}
		// Frame-local identity is a separate namespace until its ownership is known.
		if strings.Contains(action.Type.String(), "FRAME") {
			inTarget = false
			continue
		}
		if inTarget {
			visit(action.ProtoReflect())
		}
	}
}
