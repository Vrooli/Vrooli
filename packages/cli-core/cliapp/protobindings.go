package cliapp

// This file contains the shared manifest-to-Connect dispatcher. It owns
// request construction only; callers may retain scenario-specific response
// rendering through ProtoBindingOptions.Render.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Renderer formats a successful response for a human or machine caller.
// Request construction remains owned by ProtoBindings.
type Renderer func(RunContext, proto.Message) error

// ProtoBindingOptions configures the generic dispatcher.
type ProtoBindingOptions struct {
	Render    map[string]Renderer
	Normalize map[string]func([]byte) ([]byte, error)
}

// ResolvedField describes the proto path selected for one manifest argument.
// Path has one element for a top-level field and two for the supported
// one-level envelope form.
type ResolvedField struct {
	Path []protoreflect.FieldDescriptor
	Kind string
}

// scalarDecodableWellKnown lists the well-known messages protojson encodes as
// a bare JSON scalar, so a single CLI string can legitimately populate them.
// Struct, ListValue, Any, and Empty are deliberately absent: none of them
// accepts a bare scalar, so an argument targeting one needs an explicit
// json_inline or json_file decoder.
var scalarDecodableWellKnown = map[string]struct{}{
	"google.protobuf.Timestamp":   {},
	"google.protobuf.Duration":    {},
	"google.protobuf.FieldMask":   {},
	"google.protobuf.StringValue": {},
	"google.protobuf.BytesValue":  {},
	"google.protobuf.BoolValue":   {},
	"google.protobuf.Int32Value":  {},
	"google.protobuf.Int64Value":  {},
	"google.protobuf.UInt32Value": {},
	"google.protobuf.UInt64Value": {},
	"google.protobuf.FloatValue":  {},
	"google.protobuf.DoubleValue": {},
	"google.protobuf.Value":       {},
}

// ScalarDecodableField reports whether CLI string values can populate field
// without a structured decoder.
//
// Scalars, enums, and the well-known scalar-encoded messages qualify. So does a
// repeated scalar or enum: a repeatable flag or a comma-split positional
// supplies those element by element.
//
// A message element never qualifies, because no sequence of bare strings
// carries field structure. That is what makes `--name` bound to a
// `DeviceProfile` field a defect rather than a style choice, and it is the
// whole difference between this check and the callability check — the field
// resolves either way.
//
// Callers pair this with the argument's decode kind: json_inline and json_file
// supply the structure protojson needs, so they are exempt.
func ScalarDecodableField(field protoreflect.FieldDescriptor) bool {
	if field == nil {
		return false
	}
	if field.IsMap() {
		return false
	}
	if field.Kind() != protoreflect.MessageKind && field.Kind() != protoreflect.GroupKind {
		return true
	}
	_, ok := scalarDecodableWellKnown[string(field.Message().FullName())]
	return ok
}

// StructuredDecodeKind reports whether kind supplies the structure a message
// field needs. It is the exemption side of ScalarDecodableField.
func StructuredDecodeKind(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "json_inline", "json_file":
		return true
	default:
		return false
	}
}

// ResolveArgField applies the shared resolution ladder:
// name, declared alias, one-level envelope auto-descent, then bind.
// A declared bind is an explicit override of inferred field selection.
func ResolveArgField(md protoreflect.MessageDescriptor, argName string, schema ArgSchema) (ResolvedField, error) {
	if md == nil {
		return ResolvedField{}, fmt.Errorf("argument %q: request descriptor is nil", argName)
	}
	canonical := argName
	var bind FlagBind
	for _, f := range schema.Flags {
		if f.Name == argName {
			canonical, bind = f.Name, f.Bind
			break
		}
		for _, alias := range f.Aliases {
			if alias == argName {
				canonical, bind = f.Name, f.Bind
				break
			}
		}
	}
	if bind.IsZero() {
		for _, p := range schema.Positionals {
			if p.Name == argName {
				canonical, bind = p.Name, p.Bind
				break
			}
		}
	}
	if !bind.IsZero() {
		fd := fieldByName(md, bind.Field)
		if fd == nil {
			return ResolvedField{}, fmt.Errorf("argument %q: bind field %q is not present on %s", argName, bind.Field, md.FullName())
		}
		return ResolvedField{Path: []protoreflect.FieldDescriptor{fd}, Kind: bindKind(bind.Kind)}, nil
	}

	if fd := fieldByName(md, canonical); fd != nil {
		return ResolvedField{Path: []protoreflect.FieldDescriptor{fd}, Kind: "raw_string"}, nil
	}

	var candidates []struct {
		parent protoreflect.FieldDescriptor
		field  protoreflect.FieldDescriptor
	}
	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		parent := fields.Get(i)
		if parent.IsList() || parent.IsMap() || parent.Kind() != protoreflect.MessageKind {
			continue
		}
		if strings.HasPrefix(string(parent.Message().FullName()), "google.protobuf.") {
			continue
		}
		if nested := fieldByName(parent.Message(), canonical); nested != nil {
			candidates = append(candidates, struct {
				parent protoreflect.FieldDescriptor
				field  protoreflect.FieldDescriptor
			}{parent, nested})
		}
	}
	if len(candidates) == 1 {
		return ResolvedField{Path: []protoreflect.FieldDescriptor{candidates[0].parent, candidates[0].field}, Kind: "raw_string"}, nil
	}
	if len(candidates) > 1 {
		names := make([]string, len(candidates))
		for i, candidate := range candidates {
			names[i] = string(candidate.parent.Name()) + "." + string(candidate.field.Name())
		}
		return ResolvedField{}, fmt.Errorf("argument %q: auto-descent is ambiguous across %s", argName, strings.Join(names, ", "))
	}
	return ResolvedField{}, fmt.Errorf("argument %q: no proto field matches %q on %s", argName, canonical, md.FullName())
}

func bindKind(kind string) string {
	if kind == "" {
		return "raw_string"
	}
	return kind
}

func fieldByName(md protoreflect.MessageDescriptor, name string) protoreflect.FieldDescriptor {
	want := normalizeArgName(name)
	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if normalizeArgName(string(fd.Name())) == want || normalizeArgName(fd.JSONName()) == want {
			return fd
		}
	}
	return nil
}

func normalizeArgName(name string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), "-", "_")
}

// ProtoBindings returns handlers for every method on serviceFQN. The manifest
// remains responsible for selecting which methods are exposed in a group.
func ProtoBindings(core *ScenarioApp, serviceFQN protoreflect.FullName, options ProtoBindingOptions) (map[string]func(RunContext) error, error) {
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(serviceFQN)
	if err != nil {
		return nil, fmt.Errorf("lookup proto service %q: %w", serviceFQN, err)
	}
	svc, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor %q is not a service", serviceFQN)
	}
	out := make(map[string]func(RunContext) error, svc.Methods().Len())
	for i := 0; i < svc.Methods().Len(); i++ {
		method := svc.Methods().Get(i)
		key := string(svc.Name()) + "." + string(method.Name())
		m := method
		out[key] = func(ctx RunContext) error {
			return invokeProtoBinding(ctx, core, svc, m, key, options)
		}
	}
	return out, nil
}

func invokeProtoBinding(ctx RunContext, core *ScenarioApp, svc protoreflect.ServiceDescriptor, method protoreflect.MethodDescriptor, key string, options ProtoBindingOptions) error {
	request := dynamicpb.NewMessage(method.Input())
	if err := hydrateProtoRequest(ctx, method.Input(), request, options.Normalize[key]); err != nil {
		return fmt.Errorf("%s: build request: %w", key, err)
	}
	httpClient, baseURL := NewConnectHTTPClient(core)
	client := connect.NewClient[dynamicpb.Message, dynamicpb.Message](httpClient, strings.TrimRight(baseURL, "/")+"/"+string(svc.FullName())+"/"+string(method.Name()), connect.WithSchema(method), connect.WithResponseInitializer(func(_ connect.Spec, msg any) error {
		dm, ok := msg.(*dynamicpb.Message)
		if !ok {
			return fmt.Errorf("response initializer received %T", msg)
		}
		*dm = *dynamicpb.NewMessage(method.Output())
		return nil
	}))
	resp, err := client.CallUnary(context.Background(), connect.NewRequest(request))
	if err != nil {
		return WrapAPIError(key, err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("%s: server returned no response", key)
	}
	if render := options.Render[key]; render != nil {
		return render(ctx, concreteRendererMessage(resp.Msg, key))
	}
	return PrintProtoJSON(ctx.Stdout(), resp.Msg)
}

// concreteRendererMessage preserves the typed-message contract of renderer
// overrides while keeping the transport dispatcher dynamic. Connect needs a
// dynamic message because the manifest selects methods at runtime, but
// scenario renderers use generated getters and type assertions. The generated
// type is resolved from the global registry and populated through wire bytes
// so this conversion has the same semantics as the Connect response.
func concreteRendererMessage(message proto.Message, binding string) proto.Message {
	if message == nil {
		return nil
	}
	name := message.ProtoReflect().Descriptor().FullName()
	typeInfo, err := protoregistry.GlobalTypes.FindMessageByName(name)
	if err != nil {
		log.Printf("cliapp: renderer %s using dynamic response %s: concrete type is not registered: %v", binding, name, err)
		return message
	}
	encoded, err := proto.Marshal(message)
	if err != nil {
		log.Printf("cliapp: renderer %s using dynamic response %s: marshal failed: %v", binding, name, err)
		return message
	}
	concrete := typeInfo.New().Interface()
	if err := proto.Unmarshal(encoded, concrete); err != nil {
		log.Printf("cliapp: renderer %s using dynamic response %s: unmarshal into concrete type failed: %v", binding, name, err)
		return message
	}
	return concrete
}

func hydrateProtoRequest(ctx RunContext, md protoreflect.MessageDescriptor, msg *dynamicpb.Message, normalize func([]byte) ([]byte, error)) error {
	if ctx.FlagDeclared("request") && ctx.Flag("request") != "" {
		return protojson.Unmarshal([]byte(ctx.Flag("request")), msg)
	}
	schema := ctx.Schema()
	for _, p := range schema.Positionals {
		if p.LocalOnly {
			continue
		}
		values := []string{ctx.Positional(p.Name)}
		if p.Repeated {
			values = ctx.Positionals(p.Name)
		}
		for _, value := range values {
			if value == "" {
				continue
			}
			resolved, err := ResolveArgField(md, p.Name, schema)
			if err != nil {
				return err
			}
			if err := applyResolvedValue(msg, resolved, value, normalize); err != nil {
				return fmt.Errorf("positional %q: %w", p.Name, err)
			}
		}
	}
	for _, f := range schema.Flags {
		if f.LocalOnly {
			continue
		}
		if f.Bool {
			if !ctx.BoolFlag(f.Name) {
				continue
			}
		} else if !ctx.FlagProvided(f.Name) && f.Default == "" {
			continue
		}
		value := ctx.Flag(f.Name)
		resolved, err := ResolveArgField(md, f.Name, schema)
		if err != nil {
			return err
		}
		if err := applyResolvedValue(msg, resolved, value, normalize); err != nil {
			return fmt.Errorf("flag --%s: %w", f.Name, err)
		}
	}
	return nil
}

func applyResolvedValue(root *dynamicpb.Message, resolved ResolvedField, raw string, normalize func([]byte) ([]byte, error)) error {
	if len(resolved.Path) == 0 {
		return fmt.Errorf("empty proto field path")
	}
	msg := root
	for _, parent := range resolved.Path[:len(resolved.Path)-1] {
		value := msg.Mutable(parent)
		msg, _ = value.Message().(*dynamicpb.Message)
		if msg == nil {
			return fmt.Errorf("field %s is not a dynamic message", parent.Name())
		}
	}
	return setProtoField(msg, resolved.Path[len(resolved.Path)-1], raw, resolved.Kind, normalize)
}

func setProtoField(msg *dynamicpb.Message, fd protoreflect.FieldDescriptor, raw, kind string, normalize func([]byte) ([]byte, error)) error {
	if kind == "json_file" {
		body, err := os.ReadFile(raw)
		if err != nil {
			return err
		}
		if normalize != nil {
			body, err = normalize(body)
			if err != nil {
				return err
			}
		}
		return setJSONField(msg, fd, body)
	}
	if kind == "json_inline" {
		return setJSONField(msg, fd, []byte(raw))
	}
	if fd.IsList() || fd.IsMap() {
		return fmt.Errorf("field %s is repeated or map; use json_inline", fd.Name())
	}
	return setScalarField(msg, fd, raw)
}

func setJSONField(msg *dynamicpb.Message, fd protoreflect.FieldDescriptor, body []byte) error {
	if fd.IsList() {
		var values []json.RawMessage
		if err := json.Unmarshal(body, &values); err != nil {
			return err
		}
		for _, raw := range values {
			if fd.Kind() == protoreflect.MessageKind {
				sub := dynamicpb.NewMessage(fd.Message())
				if err := protojson.Unmarshal(raw, sub); err != nil {
					return err
				}
				msg.Mutable(fd).List().Append(protoreflect.ValueOfMessage(sub))
				continue
			}
			var scalar any
			if err := json.Unmarshal(raw, &scalar); err != nil {
				return err
			}
			tmp := dynamicpb.NewMessage(msg.Descriptor())
			if err := setScalarField(tmp, fd, fmt.Sprint(scalar)); err != nil {
				return err
			}
			msg.Mutable(fd).List().Append(tmp.Get(fd))
		}
		return nil
	}
	if fd.Kind() == protoreflect.StringKind {
		return setScalarField(msg, fd, string(body))
	}
	if fd.Kind() == protoreflect.BytesKind {
		msg.Set(fd, protoreflect.ValueOfBytes(body))
		return nil
	}
	if fd.Kind() == protoreflect.MessageKind {
		sub := dynamicpb.NewMessage(fd.Message())
		if err := protojson.Unmarshal(body, sub); err != nil {
			return err
		}
		msg.Set(fd, protoreflect.ValueOfMessage(sub))
		return nil
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return err
	}
	return setScalarField(msg, fd, fmt.Sprint(value))
}

func setScalarField(msg *dynamicpb.Message, fd protoreflect.FieldDescriptor, raw string) error {
	switch fd.Kind() {
	case protoreflect.StringKind:
		msg.Set(fd, protoreflect.ValueOfString(raw))
	case protoreflect.BytesKind:
		msg.Set(fd, protoreflect.ValueOfBytes([]byte(raw)))
	case protoreflect.BoolKind:
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		msg.Set(fd, protoreflect.ValueOfBool(value))
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		value, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return err
		}
		msg.Set(fd, protoreflect.ValueOfInt32(int32(value)))
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return err
		}
		msg.Set(fd, protoreflect.ValueOfInt64(value))
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		value, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return err
		}
		msg.Set(fd, protoreflect.ValueOfUint32(uint32(value)))
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return err
		}
		msg.Set(fd, protoreflect.ValueOfUint64(value))
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}
		if fd.Kind() == protoreflect.FloatKind {
			msg.Set(fd, protoreflect.ValueOfFloat32(float32(value)))
		} else {
			msg.Set(fd, protoreflect.ValueOfFloat64(value))
		}
	case protoreflect.EnumKind:
		ev := fd.Enum().Values().ByName(protoreflect.Name(raw))
		if ev == nil {
			want := normalizeArgName(raw)
			values := fd.Enum().Values()
			for i := 0; i < values.Len(); i++ {
				candidate := values.Get(i)
				name := normalizeArgName(strings.TrimPrefix(strings.ToLower(string(candidate.Name())), ""))
				if name == want || strings.HasSuffix(name, "_"+want) || strings.Contains(name, "_"+want+"_") {
					ev = candidate
					break
				}
			}
		}
		if ev == nil {
			n, err := strconv.ParseInt(raw, 10, 32)
			if err != nil {
				return fmt.Errorf("unknown enum value %q", raw)
			}
			msg.Set(fd, protoreflect.ValueOfEnum(protoreflect.EnumNumber(n)))
		} else {
			msg.Set(fd, protoreflect.ValueOfEnum(ev.Number()))
		}
	default:
		return fmt.Errorf("field %s has unsupported kind %s", fd.Name(), fd.Kind())
	}
	return nil
}
