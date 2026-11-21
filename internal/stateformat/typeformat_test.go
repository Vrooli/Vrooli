package stateformat

import (
	"reflect"
	"testing"
)

type shapeFirst struct {
	Alpha string `json:"alpha"`
	Beta  int    `json:"beta"`
}

type shapeReordered struct {
	Beta  int    `json:"beta"`
	Alpha string `json:"alpha"`
}

type shapeRenamed struct {
	Alpha string `json:"alpha"`
	Beta  int    `json:"gamma"`
}

type shapeAdded struct {
	Alpha string `json:"alpha"`
	Beta  int    `json:"beta"`
	Extra bool   `json:"extra"`
}

type shapeSkipped struct {
	Alpha string `json:"alpha"`
	Beta  int    `json:"beta"`
	Cache string `json:"-"`
	note  string
}

type shapeUntagged struct {
	Alpha string
}

type embeddedAlpha struct {
	Inner string `json:"inner"`
}

type embeddedBeta struct {
	Inner string `json:"inner"`
}

type shapeEmbeddedAlpha struct {
	embeddedAlpha
	Outer string `json:"outer"`
}

type shapeEmbeddedBeta struct {
	embeddedBeta
	Outer string `json:"outer"`
}

type shapeContainers struct {
	Pointer *shapeFirst             `json:"pointer"`
	Any     any                     `json:"any"`
	Slice   []shapeFirst            `json:"slice"`
	Map     map[string]shapeFirst   `json:"map"`
	Bytes   []byte                  `json:"bytes"`
	List    []map[string][]*float64 `json:"list"`
}

type recursiveNode struct {
	Name     string          `json:"name"`
	Children []recursiveNode `json:"children"`
}

func digestTypesOf(t *testing.T, types ...reflect.Type) string {
	t.Helper()
	digest, err := DigestTypes(types...)
	if err != nil {
		t.Fatalf("digest %v: %v", types, err)
	}
	return digest
}

func TestDigestTypesIgnoresNamesAndFieldOrder(t *testing.T) {
	first := digestTypesOf(t, reflect.TypeFor[shapeFirst]())
	if reordered := digestTypesOf(t, reflect.TypeFor[shapeReordered]()); first != reordered {
		t.Fatalf("moving a field changed the stored shape: %s != %s", first, reordered)
	}
	// The same shape declared under a different type name is the same format.
	firstPointer := digestTypesOf(t, reflect.TypeFor[*shapeFirst]())
	if first != firstPointer {
		t.Fatalf("a pointer to a stored type changed the shape: %s != %s", first, firstPointer)
	}
}

func TestDigestTypesTracksTheWireShape(t *testing.T) {
	base := digestTypesOf(t, reflect.TypeFor[shapeFirst]())
	if renamed := digestTypesOf(t, reflect.TypeFor[shapeRenamed]()); renamed == base {
		t.Fatal("renaming a JSON field kept the stored shape")
	}
	if added := digestTypesOf(t, reflect.TypeFor[shapeAdded]()); added == base {
		t.Fatal("adding a JSON field kept the stored shape")
	}
}

func TestDigestTypesIgnoresFieldsTheCodecDrops(t *testing.T) {
	base := digestTypesOf(t, reflect.TypeFor[shapeFirst]())
	if skipped := digestTypesOf(t, reflect.TypeFor[shapeSkipped]()); skipped != base {
		t.Fatalf("a field the codec never writes changed the shape: %s != %s", skipped, base)
	}
	if untagged := digestTypesOf(t, reflect.TypeFor[shapeUntagged]()); untagged == digestTypesOf(t, reflect.TypeFor[shapeFirst]()) {
		t.Fatal("an untagged field is described as if it were tagged")
	}
}

// An embedded type is flattened by the codec, so its Go type name must not leak
// into the digest: two identically shaped embedded types are the same format.
func TestDigestTypesKeepsEmbeddedTypeNamesOutOfTheShape(t *testing.T) {
	alpha := digestTypesOf(t, reflect.TypeFor[shapeEmbeddedAlpha]())
	beta := digestTypesOf(t, reflect.TypeFor[shapeEmbeddedBeta]())
	if alpha != beta {
		t.Fatalf("renaming an embedded type changed the stored shape: %s != %s", alpha, beta)
	}
}

func TestDigestTypesDescribesContainersAndRecursion(t *testing.T) {
	if _, err := DescribeTypes(reflect.TypeFor[shapeContainers]()); err != nil {
		t.Fatalf("containers: %v", err)
	}
	first := digestTypesOf(t, reflect.TypeFor[recursiveNode]())
	second := digestTypesOf(t, reflect.TypeFor[recursiveNode]())
	if first != second {
		t.Fatal("a recursive type has an unstable digest")
	}
}

func TestDigestTypesIsStableAcrossRunsAndIndependentPerType(t *testing.T) {
	combined := digestTypesOf(t, reflect.TypeFor[shapeFirst](), reflect.TypeFor[recursiveNode]())
	if again := digestTypesOf(t, reflect.TypeFor[shapeFirst](), reflect.TypeFor[recursiveNode]()); combined != again {
		t.Fatal("the same types digest differently between calls")
	}
	alone := digestTypesOf(t, reflect.TypeFor[shapeFirst]())
	if combined == alone {
		t.Fatal("adding a stored type did not change the digest")
	}
}

func TestDigestTypesRefusesAnUnknownShape(t *testing.T) {
	if _, err := DigestTypes(reflect.TypeFor[chan int]()); err == nil {
		t.Fatal("a channel was accepted as a stored type")
	}
	if _, err := DigestTypes(nil); err == nil {
		t.Fatal("a nil type was accepted")
	}
}
