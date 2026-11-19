package capability

import (
	"os"
	"path/filepath"
	"testing"

	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// TestGeneratedOpsCLICommandsUpToDate is the drift gate for the deterministic
// CLI surface: a parameter added to ops.proto, or an operation added to the
// manifest set, fails here until `make capability-gen` regenerates the table.
func TestGeneratedOpsCLICommandsUpToDate(t *testing.T) {
	want, err := RenderOpsCLI()
	if err != nil {
		t.Fatalf("RenderOpsCLI: %v", err)
	}
	path := filepath.Join("..", "..", "..", "cli", "domains", "ops", "commands.generated.go")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated ops CLI table %s: %v (run `make capability-gen`)", path, err)
	}
	if string(got) != want {
		t.Fatalf("generated ops CLI command table is stale; run `make capability-gen` from the scenario root")
	}
}

// TestDeterministicWireCoverage is the bidirectional contract between the
// deterministic operation set and the REST run edge. Every deterministic
// operation must have a typed OpParams member (the edge refuses an operation it
// cannot translate, so a manifest without one would advertise a broken run),
// and every member must be either generated as a flag command or named as a
// layered view the CLI builds by hand. The exception list cannot rot: an entry
// that names a generated operation, or none at all, fails here.
func TestDeterministicWireCoverage(t *testing.T) {
	oneof := (&opsv1.OpParams{}).ProtoReflect().Descriptor().Oneofs().ByName("op")
	if oneof == nil {
		t.Fatal("OpParams has no op oneof")
	}
	specs, err := opsRunSpecs()
	if err != nil {
		t.Fatalf("opsRunSpecs: %v", err)
	}
	generated := make(map[string]struct{}, len(specs))
	for _, s := range specs {
		generated[s.op] = struct{}{}
	}
	declared := 0
	for _, m := range registry.manifests {
		if m.Engine != EngineCompute {
			continue
		}
		declared++
		if oneof.Fields().ByName(protoreflect.Name(m.ID)) == nil {
			t.Errorf("operation %q has no OpParams member; the REST run edge cannot execute it", m.ID)
			continue
		}
		_, layered := layeredOps[m.ID]
		_, ok := generated[m.ID]
		if ok && layered {
			t.Errorf("operation %q is both generated and listed as a layered view", m.ID)
		}
		if !ok && !layered {
			t.Errorf("operation %q is neither generated as a command nor listed as a layered view", m.ID)
		}
	}
	if declared != oneof.Fields().Len() {
		t.Errorf("%d deterministic operations, %d OpParams members: every member must be a manifest", declared, oneof.Fields().Len())
	}
}
