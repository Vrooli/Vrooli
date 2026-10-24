// Package measures is the shared contract library for Vrooli "measures": named,
// typed, parameterized analytical queries declared once per scenario.
//
// This file implements the Phase 0 descriptor reader. It reads the committed
// proto descriptor image (a self-contained FileDescriptorSet emitted by
// `buf build`, see packages/proto/gen/descriptor/image.binpb) and, given a
// (service, method) pair, returns the request message's parameter schema —
// including buf.validate constraints (numeric bounds, enum membership, uuid
// format, length bounds), proto3 optionality, repeatedness, and the field's
// leading comment. Later phases assemble a MeasureDeclaration on top of this.
package measures

import (
	"fmt"
	"strings"

	validate "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/vrooli/vrooli/packages/proto/descriptorimage"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// ParamSchema is a runtime-introspectable description of a single request-message
// field (a measure parameter). It is derived purely from the proto descriptor +
// its buf.validate constraints; nothing here is duplicated in the CLI manifest.
type ParamSchema struct {
	// Name is the proto field name (snake_case, as authored).
	Name string

	// Type is the canonical parameter type. For most fields it is the proto
	// kind ("string", "int32", "int64", "bool", "double", "enum", "message",
	// ...). A field whose message type is the shared vrooli.measures.v1.TimeWindow
	// is reported as the canonical type "time_window".
	Type string

	// MessageType is the fully-qualified message name when Type == "message"
	// (or "time_window"); empty otherwise. Lets later phases recognize canonical
	// shared types without re-parsing.
	MessageType string

	// Repeated is true for `repeated` fields (and map fields).
	Repeated bool

	// Optional is true for proto3 explicit-presence fields (`optional`).
	Optional bool

	// Required is true when the field carries (buf.validate.field).required.
	Required bool

	// EnumValues lists the permitted enum value names for an enum field: the
	// enum's declared values, minus any excluded by a buf.validate `not_in`
	// constraint. Empty for non-enum fields.
	EnumValues []string

	// Min / Max carry numeric bounds from buf.validate (gte/gt -> Min,
	// lte/lt -> Max). Nil when unconstrained or non-numeric.
	Min *int64
	Max *int64

	// MinLen / MaxLen carry string length bounds from buf.validate
	// (min_len / max_len). Nil when unconstrained or non-string.
	MinLen *uint64
	MaxLen *uint64

	// Format is a string-format hint derived from buf.validate (currently
	// "uuid" when string.uuid is set). Empty otherwise.
	Format string

	// Description is the field's leading comment from the .proto source,
	// trimmed. Empty when the descriptor carried no source info or comment.
	Description string
}

// TimeWindowMessageName is the fully-qualified name of the shared canonical
// time-window type. A request field of this message type is surfaced with the
// canonical Type "time_window".
const TimeWindowMessageName = "vrooli.measures.v1.TimeWindow"

// SchemaReader resolves request-message param schemas from a proto descriptor
// image. Construct it once (it parses the whole image) and reuse it.
type SchemaReader struct {
	files *protoregistry.Files
}

// NewSchemaReaderFromFile loads a FileDescriptorSet image (e.g.
// packages/proto/gen/descriptor/image.binpb) from disk.
func NewSchemaReaderFromFile(path string) (*SchemaReader, error) {
	source, err := descriptorimage.New(descriptorimage.Config{DescriptorPath: path})
	if err != nil {
		return nil, err
	}
	snapshot, err := source.Snapshot()
	if err != nil {
		return nil, fmt.Errorf("measures: load descriptor image %q: %w", path, err)
	}
	return NewSchemaReaderFromBytes(snapshot.DescriptorBytes())
}

// NewSchemaReaderFromBytes loads a serialized FileDescriptorSet image from
// memory. The image must be self-contained (all transitive imports included),
// which is exactly what `buf build -o image.binpb` produces.
func NewSchemaReaderFromBytes(b []byte) (*SchemaReader, error) {
	fdset := &descriptorpb.FileDescriptorSet{}
	// Unmarshal against the global registry so the buf.validate.field extension
	// (registered by the blank-imported validate package) is decoded into typed
	// field options rather than left as unknown bytes.
	if err := proto.Unmarshal(b, fdset); err != nil {
		return nil, fmt.Errorf("measures: unmarshal descriptor image: %w", err)
	}
	files, err := protodesc.NewFiles(fdset)
	if err != nil {
		return nil, fmt.Errorf("measures: build descriptor registry: %w", err)
	}
	return &SchemaReader{files: files}, nil
}

// RequestParams returns the parameter schema for the request message of
// (service, method). `service` may be a fully-qualified service name
// (e.g. "agent_manager.v1.AgentManagerService") or a bare service name
// (e.g. "StatsService") — the latter is matched across all packages, which is
// what a CLI manifest `binding` supplies. `method` is the RPC method name.
func (r *SchemaReader) RequestParams(service, method string) ([]ParamSchema, error) {
	md, err := r.findMethod(service, method)
	if err != nil {
		return nil, err
	}
	return paramsForMessage(md.Input()), nil
}

// findMethod resolves a MethodDescriptor by (service, method).
func (r *SchemaReader) findMethod(service, method string) (protoreflect.MethodDescriptor, error) {
	if strings.Contains(service, ".") {
		// Fully-qualified: resolve directly.
		full := protoreflect.FullName(service + "." + method)
		d, err := r.files.FindDescriptorByName(full)
		if err != nil {
			return nil, fmt.Errorf("measures: method %q not found: %w", full, err)
		}
		md, ok := d.(protoreflect.MethodDescriptor)
		if !ok {
			return nil, fmt.Errorf("measures: %q is not a method", full)
		}
		return md, nil
	}

	// Bare service name: scan every file's services for a matching short name.
	var found protoreflect.MethodDescriptor
	var matches int
	r.files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		svcs := fd.Services()
		for i := 0; i < svcs.Len(); i++ {
			sd := svcs.Get(i)
			if string(sd.Name()) != service {
				continue
			}
			if md := sd.Methods().ByName(protoreflect.Name(method)); md != nil {
				found = md
				matches++
			}
		}
		return true
	})
	switch {
	case matches == 0:
		return nil, fmt.Errorf("measures: no service %q with method %q found in descriptor image", service, method)
	case matches > 1:
		return nil, fmt.Errorf("measures: ambiguous service %q.%q (%d matches); qualify the service with its proto package", service, method, matches)
	}
	return found, nil
}

// paramsForMessage extracts a ParamSchema per field of a message descriptor.
func paramsForMessage(msg protoreflect.MessageDescriptor) []ParamSchema {
	fields := msg.Fields()
	out := make([]ParamSchema, 0, fields.Len())
	srcLocs := msg.ParentFile().SourceLocations()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		out = append(out, paramForField(fd, srcLocs))
	}
	return out
}

func paramForField(fd protoreflect.FieldDescriptor, srcLocs protoreflect.SourceLocations) ParamSchema {
	p := ParamSchema{
		Name:     string(fd.Name()),
		Type:     fieldType(fd),
		Repeated: fd.Cardinality() == protoreflect.Repeated,
		Optional: fd.HasOptionalKeyword(),
	}
	if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
		p.MessageType = string(fd.Message().FullName())
	}
	if fd.Kind() == protoreflect.EnumKind {
		p.EnumValues = enumValueNames(fd.Enum(), nil)
	}
	// Leading comment from source info (present when buf build kept source info).
	if loc := srcLocs.ByDescriptor(fd); loc.Path != nil {
		if c := strings.TrimSpace(loc.LeadingComments); c != "" {
			p.Description = c
		}
	}
	applyValidateRules(fd, &p)
	return p
}

// fieldType maps a field to its canonical param type string.
func fieldType(fd protoreflect.FieldDescriptor) string {
	switch fd.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if string(fd.Message().FullName()) == TimeWindowMessageName {
			return "time_window"
		}
		return "message"
	case protoreflect.EnumKind:
		return "enum"
	default:
		return fd.Kind().String()
	}
}

// applyValidateRules reads the (buf.validate.field) extension off the field
// options and folds the relevant constraints into the ParamSchema.
func applyValidateRules(fd protoreflect.FieldDescriptor, p *ParamSchema) {
	opts := fd.Options()
	if opts == nil {
		return
	}
	if !proto.HasExtension(opts, validate.E_Field) {
		return
	}
	rules, ok := proto.GetExtension(opts, validate.E_Field).(*validate.FieldRules)
	if !ok || rules == nil {
		return
	}
	if rules.GetRequired() {
		p.Required = true
	}
	switch r := rules.GetType().(type) {
	case *validate.FieldRules_Int32:
		setIntBounds(p, int64(r.Int32.GetGte()), r.Int32.HasGte(), int64(r.Int32.GetGt()), r.Int32.HasGt(),
			int64(r.Int32.GetLte()), r.Int32.HasLte(), int64(r.Int32.GetLt()), r.Int32.HasLt())
	case *validate.FieldRules_Int64:
		setIntBounds(p, r.Int64.GetGte(), r.Int64.HasGte(), r.Int64.GetGt(), r.Int64.HasGt(),
			r.Int64.GetLte(), r.Int64.HasLte(), r.Int64.GetLt(), r.Int64.HasLt())
	case *validate.FieldRules_Uint32:
		setIntBounds(p, int64(r.Uint32.GetGte()), r.Uint32.HasGte(), int64(r.Uint32.GetGt()), r.Uint32.HasGt(),
			int64(r.Uint32.GetLte()), r.Uint32.HasLte(), int64(r.Uint32.GetLt()), r.Uint32.HasLt())
	case *validate.FieldRules_Uint64:
		setIntBounds(p, int64(r.Uint64.GetGte()), r.Uint64.HasGte(), int64(r.Uint64.GetGt()), r.Uint64.HasGt(),
			int64(r.Uint64.GetLte()), r.Uint64.HasLte(), int64(r.Uint64.GetLt()), r.Uint64.HasLt())
	case *validate.FieldRules_String_:
		applyStringRules(r.String_, p)
	case *validate.FieldRules_Enum:
		applyEnumRules(fd, r.Enum, p)
	}
}

func applyStringRules(s *validate.StringRules, p *ParamSchema) {
	if s == nil {
		return
	}
	if s.HasMinLen() {
		v := s.GetMinLen()
		p.MinLen = &v
	}
	if s.HasMaxLen() {
		v := s.GetMaxLen()
		p.MaxLen = &v
	}
	if s.GetUuid() {
		p.Format = "uuid"
	}
}

func applyEnumRules(fd protoreflect.FieldDescriptor, e *validate.EnumRules, p *ParamSchema) {
	if e == nil {
		return
	}
	// Re-derive the permitted set honoring not_in (defined_only is already
	// implied by enumerating the declared values).
	p.EnumValues = enumValueNames(fd.Enum(), e.GetNotIn())
}

// enumValueNames returns the enum's declared value names, excluding any numbers
// in `exclude`.
func enumValueNames(ed protoreflect.EnumDescriptor, exclude []int32) []string {
	excluded := make(map[int32]struct{}, len(exclude))
	for _, n := range exclude {
		excluded[n] = struct{}{}
	}
	vals := ed.Values()
	names := make([]string, 0, vals.Len())
	for i := 0; i < vals.Len(); i++ {
		v := vals.Get(i)
		if _, skip := excluded[int32(v.Number())]; skip {
			continue
		}
		names = append(names, string(v.Name()))
	}
	return names
}

// setIntBounds folds the four comparison rules into Min/Max. gte/gt -> Min,
// lte/lt -> Max. (gt/lt are exclusive; we record the boundary value as-is — the
// inclusivity nuance is not needed by the param-extraction consumers.)
func setIntBounds(p *ParamSchema, gte int64, hasGte bool, gt int64, hasGt bool, lte int64, hasLte bool, lt int64, hasLt bool) {
	switch {
	case hasGte:
		v := gte
		p.Min = &v
	case hasGt:
		v := gt
		p.Min = &v
	}
	switch {
	case hasLte:
		v := lte
		p.Max = &v
	case hasLt:
		v := lt
		p.Max = &v
	}
}
