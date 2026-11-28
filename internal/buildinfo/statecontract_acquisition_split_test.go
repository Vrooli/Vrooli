package buildinfo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/statecontract"
)

func TestAcquisitionGeneratorChangesContract(t *testing.T) {
	root := t.TempDir()
	writeStateContractFixture(t, root)
	writeTestFile(t, root, "go.mod", "module example.com/root\n")
	manifest := func() statecontract.Manifest {
		t.Helper()
		stamp, err := PersistentStateCodecStamp(root)
		if err != nil {
			t.Fatal(err)
		}
		var acquisition string
		for _, part := range strings.Split(stamp, ";") {
			if strings.HasPrefix(part, "srcs=") {
				for _, item := range strings.Split(strings.TrimPrefix(part, "srcs="), ",") {
					if strings.HasPrefix(item, "acquisition:") {
						acquisition = strings.TrimPrefix(item, "acquisition:")
					}
				}
			}
		}
		if acquisition == "" {
			t.Fatal("acquisition missing from real build stamp")
		}
		stamp, err = statecontract.EncodeStamp(statecontract.Stamp{Packages: []string{"example.com/root"}, Sources: map[string]string{"acquisition": acquisition}})
		if err != nil {
			t.Fatal(err)
		}
		prior := statecontract.CodecStamp
		statecontract.CodecStamp = stamp
		defer func() { statecontract.CodecStamp = prior }()
		got, err := statecontract.Compute([]statecontract.Owner{{Name: "acquisition", WholeSource: true}})
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	original := manifest()
	writeTestFile(t, root, "packages/binaryfetch/cmd/acquisition-schema/main.go", "package main\n// changed generator\n")
	changed := manifest()
	if original.Contract == changed.Contract || original.Owners[0].Digest == changed.Owners[0].Digest {
		t.Fatal("generator change did not alter acquisition contract")
	}
	writeTestFile(t, root, "packages/binaryfetch/cmd/acquisition-schema/input.txt", "dynamic input")
	if manifest().Contract == changed.Contract {
		t.Fatal("generator data file did not alter acquisition contract")
	}
	link := filepath.Join(root, "packages/binaryfetch/cmd/acquisition-schema/link")
	if err := os.Symlink("input.txt", link); err != nil {
		t.Fatal(err)
	}
	if _, err := acquisitionGeneratorDigest(root); err == nil {
		t.Fatal("symlink accepted in generator inventory")
	}
}
