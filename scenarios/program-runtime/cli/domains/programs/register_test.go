package programs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
)

func TestProgramListReportHasOneRowPerProgram(t *testing.T) {
	report := (&handlers{}).listReport(nil, &programsv1.ListProgramsResponse{Programs: []*programsv1.Program{{Id: "p1", SessionId: "s1"}, {Id: "p2", SessionId: "s2"}}})
	if len(report.Results) != 2 {
		t.Fatalf("results=%d, want 2", len(report.Results))
	}
}

func TestGroupName(t *testing.T) {
	if GroupName != "programs" {
		t.Fatal(GroupName)
	}
}

func TestProgramSourceReadsFileAndRejectsAmbiguousInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "program.py")
	if err := os.WriteFile(path, []byte("print('file')\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := programSource("", path)
	if err != nil || source != "print('file')\n" {
		t.Fatalf("source=%q err=%v", source, err)
	}
	if _, err := programSource("print('inline')", path); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("ambiguous source error=%v", err)
	}
}

func TestProgramSourceReadsStdin(t *testing.T) {
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := write.WriteString("print('stdin')"); err != nil {
		t.Fatal(err)
	}
	_ = write.Close()
	previous := os.Stdin
	os.Stdin = read
	t.Cleanup(func() { os.Stdin = previous; _ = read.Close() })
	source, err := programSource("", "-")
	if err != nil || source != "print('stdin')" {
		t.Fatalf("source=%q err=%v", source, err)
	}
}

func TestProgramReportSummarisesLearningReceipt(t *testing.T) {
	receipt := `{"task_id":"t","attempt_id":"a","outcome":"verified_success","delivery":"delivered","attempts":[{"recall_status":"matched"}]}`
	report := (&handlers{}).programReport(nil, &programsv1.GetProgramResponse{Program: &programsv1.Program{Id: "p1", LearningJson: receipt}})
	if len(report.Summary) != 2 || report.Summary[1] != "Learning: outcome=verified_success delivery=delivered recall_status=matched." {
		t.Fatalf("summary=%q", report.Summary)
	}
	plain := (&handlers{}).programReport(nil, &programsv1.GetProgramResponse{Program: &programsv1.Program{Id: "p1"}})
	if len(plain.Summary) != 1 {
		t.Fatalf("summary=%q, want no learning line without a receipt", plain.Summary)
	}
	waited := (&handlers{}).waitReport(nil, &programsv1.WaitForProgramResponse{Terminal: true, Program: &programsv1.Program{Id: "p1", LearningJson: receipt}})
	if len(waited.Summary) != 2 || !strings.HasPrefix(waited.Summary[1], "Learning: ") {
		t.Fatalf("summary=%q", waited.Summary)
	}
}
