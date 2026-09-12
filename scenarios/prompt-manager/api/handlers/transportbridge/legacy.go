// Package transportbridge provides the narrow compatibility seam used while
// prompt-manager's transport-owning packages are moved from net/http to
// Connect. It invokes an existing in-process handler; it never performs a
// loopback network request and is not exposed as a REST route.
package transportbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// Result is the complete response emitted by an in-process legacy handler.
type Result struct {
	Status int
	Header http.Header
	Body   []byte
}

// DecodeValue preserves an established JSON response inside protobuf Value.
// It is reserved for domain projections whose keys are themselves governed
// data (for example authored operating-model documents), not as a substitute
// for stable entity schemas.
func DecodeValue(body []byte) (*structpb.Value, error) {
	var decoded any
	if len(bytes.TrimSpace(body)) == 0 {
		return structpb.NewNullValue(), nil
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decode JSON projection: %w", err))
	}
	value, err := structpb.NewValue(decoded)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("translate JSON projection: %w", err))
	}
	return value, nil
}

// ValueBody converts an optional protobuf Value into its JSON-compatible Go
// representation for an in-process legacy domain handler.
func ValueBody(value *structpb.Value) any {
	if value == nil {
		return nil
	}
	return value.AsInterface()
}

// Decode translates the legacy handler's JSON representation into the
// generated protobuf response. Unknown fields fail closed so contract drift is
// discovered during tests instead of silently discarded.
func Decode(body []byte, response proto.Message) error {
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(body, response); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("translate response to protobuf: %w", err))
	}
	return nil
}

// DecodeWrapped wraps a legacy top-level object or array in the named response
// field before decoding it as protobuf JSON.
func DecodeWrapped(body []byte, field string, response proto.Message) error {
	wrapper, err := json.Marshal(map[string]json.RawMessage{field: body})
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("wrap response: %w", err))
	}
	return Decode(wrapper, response)
}

// ProtoBody renders a generated message as protobuf JSON for an established
// domain handler that still accepts its legacy request shape.
func ProtoBody(message proto.Message) (json.RawMessage, error) {
	if message == nil {
		return json.RawMessage("{}"), nil
	}
	raw, err := protojson.Marshal(message)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode protobuf request: %w", err))
	}
	return raw, nil
}

// MaskedBody applies a standard protobuf FieldMask to a message and returns
// the flat partial-object shape expected by legacy PATCH-style domain logic.
func MaskedBody(message proto.Message, paths []string) (map[string]any, error) {
	raw, err := ProtoBody(message)
	if err != nil {
		return nil, err
	}
	all := map[string]any{}
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decode protobuf request JSON: %w", err))
	}
	if len(paths) == 0 {
		return all, nil
	}
	filtered := make(map[string]any, len(paths))
	for _, path := range paths {
		key := lowerCamel(path)
		if value, ok := all[key]; ok {
			filtered[key] = value
		}
	}
	return filtered, nil
}

func lowerCamel(value string) string {
	parts := strings.Split(value, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// InvokeProto invokes a legacy handler in-process and translates its successful
// JSON response into a generated protobuf message. It is the shared migration
// seam for Connect domains whose transport-neutral behavior still lives behind
// an established net/http handler.
func InvokeProto[T any](ctx context.Context, headers http.Header, handler http.HandlerFunc, method, target string, request proto.Message, response *T) (*connect.Response[T], error) {
	var body any
	if request != nil {
		encoded, err := protojson.Marshal(request)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode protobuf request: %w", err))
		}
		body = json.RawMessage(encoded)
	}
	return InvokeJSON(ctx, headers, handler, method, target, body, nil, response)
}

// InvokeJSON invokes a handler with an already-adapted legacy JSON body and
// route variables, then decodes the response into the generated message.
func InvokeJSON[T any](ctx context.Context, headers http.Header, handler http.HandlerFunc, method, target string, body any, vars map[string]string, response *T) (*connect.Response[T], error) {
	result, err := Invoke(ctx, headers, handler, method, target, body, vars)
	if err != nil {
		return nil, err
	}
	message, ok := any(response).(proto.Message)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("generated response is not a protobuf message"))
	}
	if err := Decode(result.Body, message); err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

// InvokeWrappedJSON adapts a legacy top-level array into a named repeated
// protobuf response field.
func InvokeWrappedJSON[T any](ctx context.Context, headers http.Header, handler http.HandlerFunc, method, target string, body any, vars map[string]string, field string, response *T) (*connect.Response[T], error) {
	result, err := Invoke(ctx, headers, handler, method, target, body, vars)
	if err != nil {
		return nil, err
	}
	message, ok := any(response).(proto.Message)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("generated response is not a protobuf message"))
	}
	if err := DecodeWrapped(result.Body, field, message); err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

// Invoke adapts one transport-neutral Connect call to an existing handler.
// Callers are responsible for translating the successful JSON body to their
// generated response message.
func Invoke(ctx context.Context, headers http.Header, handler http.HandlerFunc, method, target string, body any, vars map[string]string) (Result, error) {
	var payload io.Reader
	if body != nil {
		var encoded bytes.Buffer
		if err := json.NewEncoder(&encoded).Encode(body); err != nil {
			return Result{}, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("encode request: %w", err))
		}
		payload = &encoded
	}
	req, err := http.NewRequestWithContext(ctx, method, target, payload)
	if err != nil {
		return Result{}, connect.NewError(connect.CodeInternal, fmt.Errorf("construct request: %w", err))
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if len(vars) > 0 {
		req = mux.SetURLVars(req, vars)
	}

	recorder := newResponseRecorder()
	handler(recorder, req)
	result := Result{Status: recorder.status, Header: recorder.header.Clone(), Body: recorder.body.Bytes()}
	if result.Status >= http.StatusBadRequest {
		message := string(bytes.TrimSpace(result.Body))
		if message == "" {
			message = http.StatusText(result.Status)
		}
		return result, connect.NewError(codeForStatus(result.Status), fmt.Errorf("%s", message))
	}
	return result, nil
}

type responseRecorder struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func newResponseRecorder() *responseRecorder {
	return &responseRecorder{header: make(http.Header), status: http.StatusOK}
}

func (r *responseRecorder) Header() http.Header    { return r.header }
func (r *responseRecorder) WriteHeader(status int) { r.status = status }
func (r *responseRecorder) Write(p []byte) (int, error) {
	return r.body.Write(p)
}

func codeForStatus(status int) connect.Code {
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return connect.CodeInvalidArgument
	case http.StatusUnauthorized:
		return connect.CodeUnauthenticated
	case http.StatusForbidden:
		return connect.CodePermissionDenied
	case http.StatusNotFound:
		return connect.CodeNotFound
	case http.StatusConflict:
		return connect.CodeAlreadyExists
	case http.StatusTooManyRequests:
		return connect.CodeResourceExhausted
	case http.StatusNotImplemented:
		return connect.CodeUnimplemented
	case http.StatusServiceUnavailable:
		return connect.CodeUnavailable
	case http.StatusGatewayTimeout:
		return connect.CodeDeadlineExceeded
	default:
		return connect.CodeInternal
	}
}
