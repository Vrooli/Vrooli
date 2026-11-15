package skills

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	clitest "prompt-manager/cli/internal/testutil"
)

const researchID = "research/experience-engineering/present-ux-decisions"

func researchHomeWithSkill(t *testing.T, body string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("VROOLI_RESEARCH_HOME", home)
	path := filepath.Join(home, "topics", "experience-engineering", "skills", "present-ux-decisions", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("---\nname: present-ux-decisions\ndescription: d\n---\n"+body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestResearchReadDetectsOnlySkillReadsNamingAResearchID(t *testing.T) {
	if got, ok := ResearchRead([]string{"skill", "read", researchID, "--lines", "1-5"}); !ok || got[0] != researchID {
		t.Fatalf("ResearchRead = %v, %v", got, ok)
	}
	for _, args := range [][]string{{"skill", "read", "adaptive-research"}, {"skill", "show", researchID}, {"skill", "read"}, {"team", "read", researchID}} {
		if _, ok := ResearchRead(args); ok {
			t.Fatalf("%v must keep the normal API path", args)
		}
	}
}

func TestReadFallsBackToTheResearchFileWhenTheAPIIsUnreachable(t *testing.T) {
	path := researchHomeWithSkill(t, "line one\nline two\nline three\n")
	ctx := clitest.NewContext(t)
	ctx.Fail("POST", "/skills/read", errors.New("dial tcp 127.0.0.1:1: connect: connection refused"))

	out, stderr, err := clitest.Output(t, func() error { return cmdRead(ctx, []string{researchID}) })
	if err != nil || out != "line one\nline two\nline three\n" {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if real, _ := filepath.EvalSymlinks(path); !strings.Contains(stderr, "from "+real) || strings.Count(stderr, "\n") != 1 {
		t.Fatalf("stderr must be one line naming the file: %q", stderr)
	}

	paged, _, err := clitest.Output(t, func() error { return cmdRead(ctx, []string{researchID, "--lines", "2-2"}) })
	if err != nil || !strings.HasPrefix(paged, "     2\tline two\n") || !strings.Contains(paged, "next: prompt-manager skill read "+researchID+" --lines 3-3") {
		t.Fatalf("paged fallback = %q, %v", paged, err)
	}

	if err := os.WriteFile(path, []byte("---\nname: present-ux-decisions\ndescription: d\n---\nedited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fresh, _, _ := clitest.Output(t, func() error { return cmdRead(ctx, []string{researchID}) })
	if fresh != "edited\n" {
		t.Fatalf("an edit must be read on the next call: %q", fresh)
	}
}

func TestReadFallsBackWhenTheAPIDoesNotServeAResearchSkill(t *testing.T) {
	researchHomeWithSkill(t, "from file\n")
	ctx := clitest.NewContext(t)
	ctx.Respond("POST", "/skills/read", ReadResponse{
		Skills:  []SkillResponse{{ID: "catalog", Content: "catalog body"}},
		Missing: []ReadIssue{{Identifier: researchID, Reason: "not_found"}},
		Output:  "skills",
	})
	out, stderr, err := clitest.Output(t, func() error { return cmdRead(ctx, []string{"catalog", researchID}) })
	if err != nil || !strings.Contains(out, "catalog body") || !strings.Contains(out, "from file") {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if strings.Contains(stderr, "Missing skills") || !strings.Contains(stderr, "did not serve it") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestReadUsesTheAPIWhenItServesTheResearchSkill(t *testing.T) {
	researchHomeWithSkill(t, "from file\n")
	ctx := clitest.NewContext(t)
	ctx.Respond("POST", "/skills/read", ReadResponse{Skills: []SkillResponse{{ID: researchID, Content: "from api\n"}}, Output: "skills"})
	out, stderr, err := clitest.Output(t, func() error { return cmdRead(ctx, []string{researchID}) })
	if err != nil || out != "from api\n" || stderr != "" {
		t.Fatalf("out=%q stderr=%q err=%v", out, stderr, err)
	}
}

func TestReadReportsAMissingResearchFileAndRefusesMixedReadsWhenTheAPIIsDown(t *testing.T) {
	researchHomeWithSkill(t, "x\n")
	ctx := clitest.NewContext(t)
	ctx.Fail("POST", "/skills/read", errors.New("connection refused"))

	_, stderr, err := clitest.Output(t, func() error {
		return cmdRead(ctx, []string{"research/experience-engineering/absent", "--strict"})
	})
	if err == nil || !strings.Contains(stderr, "research/experience-engineering/absent (not_found: ") {
		t.Fatalf("missing file: stderr=%q err=%v", stderr, err)
	}

	if _, _, err := clitest.Output(t, func() error { return cmdRead(ctx, []string{"catalog", researchID}) }); err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("a catalog skill cannot be read without the API: %v", err)
	}
}
