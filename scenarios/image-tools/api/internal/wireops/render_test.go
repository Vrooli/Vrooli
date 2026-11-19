package wireops

import (
	"os"
	"reflect"
	"testing"

	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
	"google.golang.org/protobuf/reflect/protoreflect"

	"image-tools/internal/ops"
)

// TestGeneratedBindingsAreUpToDate is the drift gate for the deterministic wire
// adapter: a parameter added to ops.proto, or a field renamed on ops.Params,
// fails here until `make capability-gen` regenerates the table.
func TestGeneratedBindingsAreUpToDate(t *testing.T) {
	want, err := RenderBindings()
	if err != nil {
		t.Fatalf("RenderBindings: %v", err)
	}
	got, err := os.ReadFile("bindings.generated.go")
	if err != nil {
		t.Fatalf("read generated bindings: %v (run `make capability-gen`)", err)
	}
	if string(got) != want {
		t.Fatalf("generated wire bindings are stale; run `make capability-gen` from the scenario root")
	}
}

// TestEveryWireParameterIsBound proves the direction the drift gate cannot: a
// field that exists on an operation's message but is missing from the binding
// table would be accepted at the wire and silently dropped before the engine.
//
// Every scalar field is given a distinctive value, so a bound field must come
// out non-zero on the compute struct. The count of non-zero struct fields must
// therefore equal the count of message fields.
func TestEveryWireParameterIsBound(t *testing.T) {
	oneof := (&opsv1.OpParams{}).ProtoReflect().Descriptor().Oneofs().ByName("op")
	if oneof == nil {
		t.Fatal("OpParams has no op oneof")
	}
	for i := 0; i < oneof.Fields().Len(); i++ {
		member := oneof.Fields().Get(i)
		op := string(member.Name())
		if layeredOperations[op] {
			continue
		}
		pb := &opsv1.OpParams{}
		target := pb.ProtoReflect().Mutable(member).Message()
		sub := member.Message()
		for j := 0; j < sub.Fields().Len(); j++ {
			setDistinctive(target, sub.Fields().Get(j))
		}
		p, err := Translate(op, pb)
		if err != nil {
			t.Errorf("%s: Translate: %v", op, err)
			continue
		}
		if got, want := nonZeroFields(p), sub.Fields().Len(); got != want {
			t.Errorf("%s: %d of %d parameters reached the engine: %s", op, got, want, describe(p))
		}
	}
}

func nonZeroFields(p *ops.Params) int {
	v := reflect.ValueOf(p).Elem()
	count := 0
	for i := 0; i < v.NumField(); i++ {
		if !v.Field(i).IsZero() {
			count++
		}
	}
	return count
}

func describe(p *ops.Params) string {
	v := reflect.ValueOf(p).Elem()
	set := make([]string, 0, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		if !v.Field(i).IsZero() {
			set = append(set, v.Type().Field(i).Name)
		}
	}
	return "bound " + join(set)
}

func join(v []string) string {
	out := ""
	for i, s := range v {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}

func setDistinctive(msg protoreflect.Message, fd protoreflect.FieldDescriptor) {
	switch {
	case fd.IsList():
		switch fd.Kind() {
		case protoreflect.StringKind:
			msg.Mutable(fd).List().Append(protoreflect.ValueOfString("wire-bound-sentinel"))
		case protoreflect.Int32Kind:
			msg.Mutable(fd).List().Append(protoreflect.ValueOfInt32(7))
		}
	case fd.Kind() == protoreflect.StringKind:
		msg.Set(fd, protoreflect.ValueOfString("wire-bound-sentinel"))
	case fd.Kind() == protoreflect.BoolKind:
		msg.Set(fd, protoreflect.ValueOfBool(true))
	case fd.Kind() == protoreflect.DoubleKind, fd.Kind() == protoreflect.FloatKind:
		msg.Set(fd, protoreflect.ValueOfFloat64(0.5))
	default:
		msg.Set(fd, protoreflect.ValueOfInt64(3))
	}
}
