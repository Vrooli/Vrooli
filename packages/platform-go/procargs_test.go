package platform

import (
	"encoding/binary"
	"errors"
	"reflect"
	"testing"
)

// procArgsBlock builds a KERN_PROCARGS2 image the way darwin lays it out.
func procArgsBlock(argc uint32, executable string, padding int, strs ...string) []byte {
	block := binary.LittleEndian.AppendUint32(nil, argc)
	block = append(block, executable...)
	block = append(block, 0)
	block = append(block, make([]byte, padding)...)
	for _, s := range strs {
		block = append(block, s...)
		block = append(block, 0)
	}
	return block
}

func TestParseProcArgs2ReadsExecutableArgsAndEnvironment(t *testing.T) {
	block := procArgsBlock(2, "/usr/local/opt/postgresql/bin/postgres", 7,
		"postgres", "-D", "HOME=/Users/op", "VROOLI_TOKEN=a=b=c", "NOEQUALS", "",
		"executable_path=/usr/local/opt/postgresql/bin/postgres")
	got, err := parseProcArgs2(block)
	if err != nil {
		t.Fatalf("parseProcArgs2() error = %v", err)
	}
	if got.Executable != "/usr/local/opt/postgresql/bin/postgres" {
		t.Errorf("Executable = %q", got.Executable)
	}
	if !reflect.DeepEqual(got.Args, []string{"postgres", "-D"}) {
		t.Errorf("Args = %q", got.Args)
	}
	want := map[string]string{"HOME": "/Users/op", "VROOLI_TOKEN": "a=b=c"}
	if !reflect.DeepEqual(got.Env, want) {
		t.Errorf("Env = %v, want %v (apple strings after the empty entry are not environment)", got.Env, want)
	}
}

func TestParseProcArgs2KeepsCompleteEntriesOfATruncatedBlock(t *testing.T) {
	block := procArgsBlock(1, "/bin/sleep", 0, "sleep", "A=1")
	block = append(block, "B=partial"...)
	got, err := parseProcArgs2(block)
	if err != nil {
		t.Fatalf("parseProcArgs2() error = %v", err)
	}
	if !reflect.DeepEqual(got.Env, map[string]string{"A": "1"}) {
		t.Errorf("Env = %v, want only the complete entry", got.Env)
	}
}

func TestParseProcArgs2AcceptsZeroArgumentsAndNoEnvironment(t *testing.T) {
	got, err := parseProcArgs2(procArgsBlock(0, "/bin/daemon", 3))
	if err != nil {
		t.Fatalf("parseProcArgs2() error = %v", err)
	}
	if got.Executable != "/bin/daemon" || len(got.Args) != 0 || len(got.Env) != 0 {
		t.Errorf("parseProcArgs2() = %+v", got)
	}
}

func TestParseProcArgs2RejectsMalformedBlocks(t *testing.T) {
	cases := map[string][]byte{
		"empty":                   nil,
		"short header":            {1, 0},
		"unterminated executable": append(binary.LittleEndian.AppendUint32(nil, 0), "/bin/sh"...),
		"empty executable":        procArgsBlock(0, "", 0),
		"argc beyond block":       procArgsBlock(1<<31, "/bin/sh", 0, "sh"),
		"argv shorter than argc":  procArgsBlock(3, "/bin/sh", 0, "sh", "-c"),
		"unterminated final argv": append(procArgsBlock(2, "/bin/sh", 0, "sh"), "-c"...),
	}
	for name, block := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseProcArgs2(block); !errors.Is(err, errMalformedProcArgs) {
				t.Fatalf("parseProcArgs2() error = %v, want errMalformedProcArgs", err)
			}
		})
	}
}
