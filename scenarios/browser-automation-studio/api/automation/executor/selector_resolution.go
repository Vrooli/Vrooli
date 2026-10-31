package executor

import (
	"fmt"
	"github.com/vrooli/api-core/uiselectors"
	"github.com/vrooli/browser-automation-studio/automation/state"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func resolveRuntimeSelectors(action *basactions.ActionDefinition, execution *state.ExecutionState) (*basactions.ActionDefinition, error) {
	if action == nil {
		return nil, nil
	}
	copy := proto.Clone(action).(*basactions.ActionDefinition)
	interpolate := func(s string) (string, error) {
		return "", fmt.Errorf("execution state required for selector parameters")
	}
	if execution != nil {
		interpolate = state.NewInterpolator(execution).InterpolateStringStrict
	}
	var visit func(protoreflect.Message) error
	visit = func(m protoreflect.Message) error {
		var failure error
		m.Range(func(f protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			if f.IsMap() {
				return true
			}
			if f.IsList() {
				if f.Kind() == protoreflect.MessageKind {
					for j := 0; j < v.List().Len(); j++ {
						if failure = visit(v.List().Get(j).Message()); failure != nil {
							return false
						}
					}
				}
				return true
			}
			if f.Kind() == protoreflect.MessageKind {
				failure = visit(v.Message())
				return failure == nil
			}
			if f.Kind() == protoreflect.StringKind {
				var s string
				s, failure = uiselectors.ResolveDeferred(v.String(), interpolate)
				if failure == nil && s != v.String() {
					m.Set(f, protoreflect.ValueOfString(s))
				}
			}
			return failure == nil
		})
		return failure
	}
	return copy, visit(copy.ProtoReflect())
}
