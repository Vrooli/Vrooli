package platform

import (
	"testing"
	"unicode/utf16"
)

func nativeWindowsEnvFixture(entries ...string) []uint16 {
	var b []uint16
	for _, entry := range entries {
		b = append(b, utf16.Encode([]rune(entry))...)
		b = append(b, 0)
	}
	b = append(b, 0)
	if len(b) == 1 {
		b = append(b, 0)
	}
	return b
}
func TestWindowsEnvironmentObservationRejectsAmbiguityBeforeMapConversion(t *testing.T) {
	for _, entries := range [][]string{{"PATH=a", "PATH=b"}, {"Path=a", "pAtH=b"}, {"K=a", "K=b"}, {"malformed"}, {"=missingvalue"}} {
		if values, err := parseWindowsProcessEnvironment(nativeWindowsEnvFixture(entries...)); err == nil {
			t.Errorf("accepted ambiguous entries %q: %v", entries, values)
		}
	}
	for _, block := range [][]uint16{{'A', '=', 0}, {0, 0, 'B', '=', 0, 0}, {'A', '=', 0xd800, 0, 0}, {'A', '=', 0xdc00, 0, 0}} {
		if _, err := parseWindowsProcessEnvironment(block); err == nil {
			t.Errorf("accepted malformed UTF16/framing %v", block)
		}
	}
}
func TestWindowsEnvironmentObservationRetainsHiddenDriveVariables(t *testing.T) {
	values, err := parseWindowsProcessEnvironment(nativeWindowsEnvFixture("PATH=private", "=C:=C:\\private"))
	if err != nil || values["PATH"] != "private" || values["=C:"] != "C:\\private" {
		t.Fatalf("values=%v err=%v", values, err)
	}
}
func TestDarwinEnvironmentObservationPreservesArgvButRefusesIncompleteEnvironment(t *testing.T) {
	for name, block := range map[string][]byte{
		"duplicate": procArgsBlock(1, "/bin/service", 1, "service", "TOKEN=one", "TOKEN=two", ""),
		"malformed": procArgsBlock(1, "/bin/service", 1, "service", "NOEQUALS", ""),
		"truncated": append(procArgsBlock(1, "/bin/service", 1, "service", "TOKEN=one"), []byte("PARTIAL")...),
	} {
		t.Run(name, func(t *testing.T) {
			args, err := parseProcArgs2(block)
			if err != nil || len(args.Args) != 1 || args.Args[0] != "service" {
				t.Fatalf("lost independent argv fact: %+v %v", args, err)
			}
			if args.environmentErr == nil {
				t.Fatal("incomplete environment became verified map")
			}
		})
	}
}
