package capacity_test

import (
	"testing"

	"image-tools/internal/adapters"
	"image-tools/internal/capacity"
	"image-tools/internal/models"
)

func TestDownloadsDeclareTransferSizedCPUNotAFullCore(t *testing.T) {
	def := capacity.SupportProfileFor("resize")
	for _, op := range []string{models.InstallJobOperation, adapters.InstallJobOperation} {
		p := capacity.SupportProfileFor(op)
		if p.CPUMilli >= def.CPUMilli || p.CPUMilli <= 0 {
			t.Fatalf("%s declares %d milli CPU; a network-bound transfer must declare less than the %d milli default but not zero", op, p.CPUMilli, def.CPUMilli)
		}
		if p.RAMBytes <= 0 || p.Reason == "" {
			t.Fatalf("%s profile is incomplete: %+v", op, p)
		}
	}
}

func TestEveryProfiledOperationIsARealOperation(t *testing.T) {
	known := map[string]bool{models.InstallJobOperation: true, adapters.InstallJobOperation: true}
	for _, op := range capacity.SupportProfileOperations() {
		if !known[op] {
			t.Fatalf("support_profiles.json profiles %q, which no job runs under; fix the name or remove the row", op)
		}
	}
}

func TestUnprofiledOperationsKeepTheFullCoreDefault(t *testing.T) {
	if p := capacity.SupportProfileFor("some_other_op"); p.CPUMilli != 1000 || p.RAMBytes != 256<<20 {
		t.Fatalf("default profile changed: %+v", p)
	}
}
