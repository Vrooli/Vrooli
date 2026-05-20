// Package protodispatch provides a generic, manifest-driven Connect-RPC
// dispatcher for BAS CLI command groups.
//
// As BAS migrates off hand-rolled REST handlers (per
// plans:bas-migration-to-proto-connect-rpc) every CLI domain wires every
// manifest command to a proto method on the corresponding generated
// Connect service. Hand-writing one strongly-typed handler per RPC is
// expensive for the ~113-command surface, so this package uses
// protoreflect + dynamicpb to invoke any RPC by its
// "Service.Method" binding key without per-domain boilerplate.
//
// Each domain's register.go calls Bindings(serviceFQN) which returns a
// map[bindingKey]func(RunContext) error suitable for
// cliapp.LoadFromManifest. The handler reads a single optional --request
// flag whose value is a JSON-encoded request body (matching the proto JSON
// representation of the RPC's request message; empty / missing means a
// zero-value request) and prints the JSON-encoded response.
//
// Domains that need richer flag→proto mapping or human-readable rendering
// can replace these generic handlers with hand-coded ones; the manifest
// is the single source of truth either way.
package protodispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/vrooli/cli-core/cliapp"
)

// Bindings returns a bindings map for cliapp.LoadFromManifest that
// dispatches every RPC of the named fully-qualified proto service via a
// generic JSON-in/JSON-out handler.
//
// serviceFQN must match the proto definition's fully-qualified name (for
// example "browser_automation_studio.v1.scenarios.ScenariosService"). The
// service is looked up in protoregistry.GlobalFiles, so the generated
// Go package for that service must be imported somewhere in the binary
// (the cli's domains.go is the single import gate).
//
// The returned map keys are "<ManifestService>.<ManifestMethod>" — i.e.
// the proto's short service name plus its method name, matching
// ManifestBinding.BindingKey().
func Bindings(core *cliapp.ScenarioApp, serviceFQN protoreflect.FullName) (map[string]func(cliapp.RunContext) error, error) {
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(serviceFQN)
	if err != nil {
		return nil, fmt.Errorf("protodispatch: lookup service %q: %w", serviceFQN, err)
	}
	svc, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("protodispatch: descriptor %q is not a service (got %T)", serviceFQN, desc)
	}

	bindings := make(map[string]func(cliapp.RunContext) error, svc.Methods().Len())
	for i := 0; i < svc.Methods().Len(); i++ {
		m := svc.Methods().Get(i)
		method := m
		key := fmt.Sprintf("%s.%s", svc.Name(), method.Name())
		bindings[key] = makeHandler(core, svc, method)
	}
	return bindings, nil
}

func makeHandler(core *cliapp.ScenarioApp, svc protoreflect.ServiceDescriptor, method protoreflect.MethodDescriptor) func(cliapp.RunContext) error {
	procedure := fmt.Sprintf("/%s/%s", svc.FullName(), method.Name())
	return func(rc cliapp.RunContext) error {
		req := dynamicpb.NewMessage(method.Input())

		// Allow --request '<json>' overrides; otherwise leave zero-valued.
		// Required-flag enforcement remains the parser's job per the
		// manifest's ArgSchema; positionals and named flags are auto-
		// populated into the request body when their names match
		// top-level field names of the request message.
		if err := hydrateFromContext(rc, method.Input(), req); err != nil {
			return fmt.Errorf("%s: build request: %w", procedure, err)
		}

		httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
		client := connect.NewClient[dynamicpb.Message, dynamicpb.Message](
			httpClient,
			baseURL+procedure,
			connect.WithSchema(method),
			connect.WithResponseInitializer(func(_ connect.Spec, msg any) error {
				dm, ok := msg.(*dynamicpb.Message)
				if !ok {
					return fmt.Errorf("protodispatch: response initializer received %T, want *dynamicpb.Message", msg)
				}
				*dm = *dynamicpb.NewMessage(method.Output())
				return nil
			}),
		)

		ctx := context.Background()
		resp, err := client.CallUnary(ctx, connect.NewRequest(req))
		if err != nil {
			return cliapp.WrapAPIError(procedure, err, nil)
		}
		if resp == nil || resp.Msg == nil {
			return fmt.Errorf("%s: server returned no response", procedure)
		}
		// When the user passed --json, emit the raw protojson body (bytes
		// fields included). Otherwise, swap bytes payloads for a length
		// summary so the human format doesn't dump kilobytes of base64
		// to a terminal (the original motivation: ai preview-screenshot
		// returning ~75 KB of raw screenshotPng).
		if rc.JSON() {
			return renderProtoJSON(rc.Stdout(), resp.Msg)
		}
		return renderProtoJSONRedacted(rc.Stdout(), resp.Msg)
	}
}

// hydrateFromContext populates a request message from RunContext's parsed
// flags and positionals. It is intentionally minimal:
//
//   - If the user passes --request '<json>', the body is decoded via
//     protojson into the request message (canonical wire-shape entry).
//   - Else, every declared flag/positional whose name matches a top-level
//     field name of the request (using the proto field's JSON name, the
//     proto field name, or a kebab-case variant) is copied in as a
//     string-typed value.
//
// More sophisticated mappings (nested messages, enums, oneofs) require a
// hand-coded handler in the domain package; this generic path is for
// list/get/simple-CRUD RPCs and falls back gracefully via --request JSON.
func hydrateFromContext(rc cliapp.RunContext, md protoreflect.MessageDescriptor, msg *dynamicpb.Message) error {
	// --request '<json>' is the canonical full-body entry point. The
	// flag is reserved by convention; manifests do not have to declare
	// it. We swallow the panic from RunContext.Flag if the schema didn't
	// declare it.
	if body := safeFlag(rc, "request"); body != "" {
		if err := protojson.Unmarshal([]byte(body), msg); err != nil {
			return fmt.Errorf("--request: %w", err)
		}
		return nil
	}

	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		names := candidateNames(fd)
		for _, name := range names {
			val, ok := lookupValue(rc, name)
			if !ok {
				continue
			}
			if err := setScalar(msg, fd, val); err != nil {
				return fmt.Errorf("field %s: %w", fd.Name(), err)
			}
			break
		}
	}
	return nil
}

func candidateNames(fd protoreflect.FieldDescriptor) []string {
	jsonName := fd.JSONName()
	protoName := string(fd.Name())
	kebab := snakeToKebab(protoName)
	return []string{jsonName, protoName, kebab}
}

func snakeToKebab(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '_' {
			out = append(out, '-')
		} else {
			out = append(out, s[i])
		}
	}
	return string(out)
}

func lookupValue(rc cliapp.RunContext, name string) (string, bool) {
	if v, ok := safeFlagWithOK(rc, name); ok && v != "" {
		return v, true
	}
	if v, ok := safePositionalWithOK(rc, name); ok && v != "" {
		return v, true
	}
	return "", false
}

func safeFlag(rc cliapp.RunContext, name string) (out string) {
	defer func() { _ = recover() }()
	return rc.Flag(name)
}

func safeFlagWithOK(rc cliapp.RunContext, name string) (out string, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			out, ok = "", false
		}
	}()
	v := rc.Flag(name)
	return v, true
}

func safePositionalWithOK(rc cliapp.RunContext, name string) (out string, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			out, ok = "", false
		}
	}()
	v := rc.Positional(name)
	return v, true
}

func setScalar(msg *dynamicpb.Message, fd protoreflect.FieldDescriptor, raw string) error {
	if fd.IsList() || fd.IsMap() {
		// Lists/maps require --request JSON; skip silently here.
		return nil
	}
	switch fd.Kind() {
	case protoreflect.StringKind:
		msg.Set(fd, protoreflect.ValueOfString(raw))
	case protoreflect.BoolKind:
		msg.Set(fd, protoreflect.ValueOfBool(raw == "true" || raw == "1" || raw == "yes"))
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		var n int32
		if _, err := fmt.Sscanf(raw, "%d", &n); err != nil {
			return err
		}
		msg.Set(fd, protoreflect.ValueOfInt32(n))
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		var n int64
		if _, err := fmt.Sscanf(raw, "%d", &n); err != nil {
			return err
		}
		msg.Set(fd, protoreflect.ValueOfInt64(n))
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		var n uint32
		if _, err := fmt.Sscanf(raw, "%d", &n); err != nil {
			return err
		}
		msg.Set(fd, protoreflect.ValueOfUint32(n))
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		var n uint64
		if _, err := fmt.Sscanf(raw, "%d", &n); err != nil {
			return err
		}
		msg.Set(fd, protoreflect.ValueOfUint64(n))
	case protoreflect.FloatKind:
		var n float32
		if _, err := fmt.Sscanf(raw, "%f", &n); err != nil {
			return err
		}
		msg.Set(fd, protoreflect.ValueOfFloat32(n))
	case protoreflect.DoubleKind:
		var n float64
		if _, err := fmt.Sscanf(raw, "%f", &n); err != nil {
			return err
		}
		msg.Set(fd, protoreflect.ValueOfFloat64(n))
	case protoreflect.EnumKind:
		ev := fd.Enum().Values().ByName(protoreflect.Name(raw))
		if ev == nil {
			// Try by number.
			var n int32
			if _, err := fmt.Sscanf(raw, "%d", &n); err == nil {
				msg.Set(fd, protoreflect.ValueOfEnum(protoreflect.EnumNumber(n)))
				return nil
			}
			return fmt.Errorf("unknown enum value %q", raw)
		}
		msg.Set(fd, protoreflect.ValueOfEnum(ev.Number()))
	default:
		// Message-typed fields require --request JSON; skip.
	}
	return nil
}

// renderProtoJSONRedacted renders a response message as JSON with every
// bytes-typed field replaced by a `"<N bytes redacted; pass --json for raw>"`
// placeholder. This is the default for human output so commands
// returning embedded binary payloads (screenshots, captured files,
// PDFs) don't fire kilobytes of base64 at a TTY.
//
// Implementation: marshal to JSON via protojson, decode into a generic
// map, walk the message descriptor + map in lockstep to overwrite bytes
// fields by their protojson key, then re-marshal the redacted map. We
// keep the protojson key (JSONName/camelCase) as the lookup so we
// remain shape-compatible with protojson output.
func renderProtoJSONRedacted(w io.Writer, msg proto.Message) error {
	opts := protojson.MarshalOptions{Multiline: true, Indent: "  ", EmitUnpopulated: true}
	raw, err := opts.Marshal(msg)
	if err != nil {
		return renderProtoJSON(w, msg)
	}
	var generic map[string]interface{}
	if err := json.Unmarshal(raw, &generic); err != nil {
		// Couldn't decode — fall back to the raw protojson output.
		_, _ = w.Write(raw)
		_, _ = w.Write([]byte{'\n'})
		return nil
	}

	redactBytesInJSON(msg.ProtoReflect().Descriptor(), generic)

	out, err := json.MarshalIndent(generic, "", "  ")
	if err != nil {
		return renderProtoJSON(w, msg)
	}
	_, _ = w.Write(out)
	_, _ = w.Write([]byte{'\n'})
	return nil
}

// redactBytesInJSON walks `node` (a protojson-shaped JSON object) using
// `md` as the proto schema for that level. Any field whose descriptor
// has Kind=BytesKind gets its JSON value replaced with a summary string.
// Nested messages, lists, and maps recurse.
func redactBytesInJSON(md protoreflect.MessageDescriptor, node map[string]interface{}) {
	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		key := fd.JSONName()
		val, ok := node[key]
		if !ok {
			continue
		}
		if val == nil {
			continue
		}

		switch {
		case fd.IsList():
			if fd.Kind() == protoreflect.BytesKind {
				arr, _ := val.([]interface{})
				totalBytes := 0
				for _, item := range arr {
					if s, ok := item.(string); ok {
						totalBytes += approxBytesFromBase64Len(len(s))
					}
				}
				node[key] = fmt.Sprintf("<%d entries, %d bytes total redacted; pass --json for raw>", len(arr), totalBytes)
				continue
			}
			if fd.Kind() == protoreflect.MessageKind {
				arr, _ := val.([]interface{})
				for _, item := range arr {
					if obj, ok := item.(map[string]interface{}); ok {
						redactBytesInJSON(fd.Message(), obj)
					}
				}
			}
		case fd.IsMap():
			if fd.MapValue().Kind() == protoreflect.MessageKind {
				if mp, ok := val.(map[string]interface{}); ok {
					for _, v := range mp {
						if obj, ok := v.(map[string]interface{}); ok {
							redactBytesInJSON(fd.MapValue().Message(), obj)
						}
					}
				}
			}
		default:
			switch fd.Kind() {
			case protoreflect.BytesKind:
				s, _ := val.(string)
				node[key] = fmt.Sprintf("<%d bytes redacted; pass --json for raw>", approxBytesFromBase64Len(len(s)))
			case protoreflect.MessageKind:
				if obj, ok := val.(map[string]interface{}); ok {
					redactBytesInJSON(fd.Message(), obj)
				}
			}
		}
	}
}

// approxBytesFromBase64Len returns the number of bytes a base64-encoded
// string of length n decodes to, ignoring exact padding (off by ±2).
// We avoid an actual decode just to count.
func approxBytesFromBase64Len(n int) int {
	// 4 base64 chars → 3 bytes.
	return (n * 3) / 4
}

func renderProtoJSON(w io.Writer, msg proto.Message) error {
	opts := protojson.MarshalOptions{Multiline: true, Indent: "  ", EmitUnpopulated: true}
	data, err := opts.Marshal(msg)
	if err != nil {
		// Fall back to plain JSON so the user still sees something useful.
		alt, jerr := json.MarshalIndent(map[string]string{"warning": err.Error()}, "", "  ")
		if jerr != nil {
			return err
		}
		_, _ = w.Write(alt)
		_, _ = w.Write([]byte{'\n'})
		return nil
	}
	_, _ = w.Write(data)
	_, _ = w.Write([]byte{'\n'})
	return nil
}

// Ensure http is referenced so go vet does not complain when build tags
// trim the connect import path; harmless at runtime.
var _ = http.NoBody
