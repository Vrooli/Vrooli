// Package targetproxy forwards typed Connect requests to a selected node.
// Target selection is a proto field, so the procedure address remains exactly
// the generated value used by local and remote calls.
package targetproxy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/nodereach"
	"github.com/vrooli/vrooli/internal/clock"
	hostv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/host/hostv1connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type Reacher interface {
	CallScenario(context.Context, nodereach.ScenarioRequest) ([]byte, error)
}

var now = clock.Real{}.Now

func Interceptor(reacher Reacher) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
			target := targetValue(request.Any())
			if target == "" || target == "local" {
				return next(ctx, request)
			}
			if remoteTargetMutationUnsupported(request.Spec().Procedure) {
				return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("procedure %s is supported only for the local host target", request.Spec().Procedure))
			}
			if reacher == nil {
				return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("node %q is unavailable", target))
			}
			if principal, ok := identity.PrincipalFromContext(ctx); ok && principal.Source == identity.SourcePersonalLocal {
				// The runtime-owned local session proves admission to this local
				// process. It is intentionally not a portable node credential.
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("remote target requires an explicit operator credential"))
			}
			authorization := strings.TrimSpace(request.Header().Get("Authorization"))
			message, ok := request.Any().(proto.Message)
			if !ok {
				return nil, connect.NewError(connect.CodeInternal, errors.New("target request is not a protobuf message"))
			}
			body, err := marshalLocalTargetRequest(message)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal target request: %w", err))
			}
			timeout := 30 * time.Second
			if deadline, ok := ctx.Deadline(); ok {
				timeout = deadline.Sub(now())
			}
			responseBody, err := reacher.CallScenario(ctx, nodereach.ScenarioRequest{
				NodeID: target, Scenario: "vrooli-onboarding", Procedure: request.Spec().Procedure,
				Authorization: authorization, Body: body, Timeout: timeout, MaxResponse: 8 << 20,
			})
			if err != nil {
				return nil, mapError(target, err)
			}
			method, ok := request.Spec().Schema.(protoreflect.MethodDescriptor)
			if !ok || method.Output() == nil {
				return nil, connect.NewError(connect.CodeInternal, errors.New("target procedure has no protobuf response descriptor"))
			}
			output := dynamicpb.NewMessage(method.Output())
			if err := proto.Unmarshal(responseBody, output); err != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unmarshal target response: %w", err))
			}
			retargetResponse(output.ProtoReflect(), target)
			return connect.NewResponse(output), nil
		}
	})
}

// These writes mutate local host/operator state. Keep them out of the remote
// target proxy even though the node-local payload must be normalized to
// target=local for procedures that are supported remotely.
func remoteTargetMutationUnsupported(procedure string) bool {
	switch procedure {
	case hostv1connect.HostServicePatchHostSafeguardConfigProcedure,
		hostv1connect.HostServiceSetNotificationRecipientProcedure:
		return true
	default:
		return false
	}
}

// The target node owns the local scenario state. Forward the request's selected
// target separately through Bridge, then translate the payload target to local
// so the receiving node executes the owner handler instead of proxying it again.
func marshalLocalTargetRequest(message proto.Message) ([]byte, error) {
	forwarded := proto.Clone(message)
	fields := forwarded.ProtoReflect().Descriptor().Fields()
	field := fields.ByName("target")
	if field == nil || field.Kind() != protoreflect.StringKind {
		return nil, errors.New("target request has no string target field")
	}
	forwarded.ProtoReflect().Set(field, protoreflect.ValueOfString("local"))
	return proto.Marshal(forwarded)
}

// Restore the caller's selected node ID in owner responses. The target node
// correctly identifies itself as local, while the originating UI remains
// scoped to the remote target it selected. Walk message fields only; leave
// arbitrary string maps and user-authored context untouched.
func retargetResponse(message protoreflect.Message, target string) {
	fields := message.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if field.Name() == "target" && field.Kind() == protoreflect.StringKind && !field.IsList() && !field.IsMap() {
			if message.Get(field).String() == "local" {
				message.Set(field, protoreflect.ValueOfString(target))
			}
			continue
		}
		if field.Kind() != protoreflect.MessageKind || field.IsMap() {
			continue
		}
		if field.IsList() {
			values := message.Get(field).List()
			for item := 0; item < values.Len(); item++ {
				retargetResponse(values.Get(item).Message(), target)
			}
		} else if message.Has(field) {
			retargetResponse(message.Get(field).Message(), target)
		}
	}
}

func targetValue(value any) string {
	message, ok := value.(protoreflect.ProtoMessage)
	if !ok {
		return ""
	}
	fields := message.ProtoReflect().Descriptor().Fields()
	field := fields.ByName("target")
	if field == nil || field.Kind() != protoreflect.StringKind {
		return ""
	}
	return strings.TrimSpace(message.ProtoReflect().Get(field).String())
}

func mapError(target string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return connect.NewError(connect.CodeDeadlineExceeded, fmt.Errorf("node %q request exceeded its deadline", target))
	}
	var reachErr *nodereach.Error
	if errors.As(err, &reachErr) {
		switch reachErr.Kind {
		case nodereach.ErrMissingScope:
			scope := strings.TrimSpace(reachErr.Scope)
			if scope == "" {
				scope = "vrooli-onboarding"
			}
			return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("node %q is missing scope %q", target, scope))
		case nodereach.ErrNodeUnavailable, nodereach.ErrNodeNotFound:
			return connect.NewError(connect.CodeUnavailable, fmt.Errorf("node %q is unavailable", target))
		case nodereach.ErrUnauthenticated:
			return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("Bridge did not accept the owner credential used to reach node %q", target))
		}
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "deadline exceeded") || strings.Contains(message, "timeout") {
		return connect.NewError(connect.CodeDeadlineExceeded, fmt.Errorf("node %q request exceeded its deadline", target))
	}
	if strings.Contains(message, "missing_scope") || strings.Contains(message, "missing scope") || strings.Contains(message, "grant") {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("node %q is missing scope %q", target, "vrooli-onboarding"))
	}
	return connect.NewError(connect.CodeUnavailable, fmt.Errorf("node %q is unavailable: %v", target, err))
}
