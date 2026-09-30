package skills

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	clitest "prompt-manager/cli/internal/testutil"
)

func TestProjectionDirsAcceptsMultiTargetEnvironment(t *testing.T) {
	sep := string(os.PathListSeparator)
	got := projectionDirs("/one" + sep + " /two " + sep + "/one" + sep)
	if !reflect.DeepEqual(got, []string{"/one", "/two"}) {
		t.Fatalf("projectionDirs = %#v", got)
	}
}

func TestRefreshRequiresPreviewBeforeApply(t *testing.T) {
	ctx := clitest.NewContext(t)
	if err := cmdRefresh(ctx, []string{"--apply"}); err == nil {
		t.Fatal("apply without preview accepted")
	}
	ctx.RequireNoRequests()
}

func TestRefreshSendsScopeAndReportsPartialFailure(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("POST", "/skills/refresh", map[string]any{"digest": "sha256:review", "rows": []map[string]any{{"runtime": "codex", "skill": "alpha", "status": "modified", "error": "local edits", "applied": false}}})
	out, _, err := clitest.Output(t, func() error {
		return cmdRefresh(ctx, []string{"--runtime=codex", "--skills=alpha", "--apply", "--expected-digest=sha256:review", "--json"})
	})
	if err == nil || !strings.Contains(out, "local edits") {
		t.Fatalf("partial result hidden: %s %v", out, err)
	}
	payload := ctx.LastRequest().Payload.(map[string]any)
	if payload["runtime"] != "codex" || payload["expectedDigest"] != "sha256:review" {
		t.Fatalf("request: %#v", payload)
	}
}

func TestResidentProjectionTokensIgnoreSkillBody(t *testing.T) {
	content := []byte("---\nname: short\ndescription: compact description\n---\n\n" + strings.Repeat("body ", 1000))
	if got := residentProjectionTokens(content); got >= len(content)/4 {
		t.Fatalf("residentProjectionTokens counted the lazy body: got %d for %d bytes", got, len(content))
	}
}

func TestCommandsRegistersSkillCommand(t *testing.T) {
	groups := Commands(nil)
	if len(groups) != 1 || groups[0].Title != "Skills" {
		t.Fatalf("unexpected command groups: %+v", groups)
	}
	if len(groups[0].Commands) != 1 || groups[0].Commands[0].Name != "skill" {
		t.Fatalf("unexpected skill command: %+v", groups[0].Commands)
	}
}

func TestUsageTextDocumentsReadAndVariants(t *testing.T) {
	text := usageText()
	for _, want := range []string{"read", "variants"} {
		if !strings.Contains(text, want) {
			t.Fatalf("usage text missing %q: %s", want, text)
		}
	}
}

func TestCmdListSendsFiltersAndPrintsResults(t *testing.T) {
	ctx := clitest.NewContext(t)
	rating := 4
	ctx.Respond("GET", "/skills", []SkillResponse{{
		ID:                  "skill-test",
		Name:                "Testing Skill",
		Folder:              "core",
		UsageCount:          7,
		Tags:                []string{"testing", "quality"},
		EffectivenessRating: &rating,
	}})

	stdout, _, err := clitest.Output(t, func() error {
		return cmdList(ctx, []string{"--folder=core", "--tag=testing"})
	})
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}

	req := ctx.LastRequest()
	if req.Method != "GET" || req.Path != "/skills" {
		t.Fatalf("unexpected request: %+v", req)
	}
	if got := req.Query.Get("folder"); got != "core" {
		t.Fatalf("folder query = %q, want core", got)
	}
	if got := req.Query.Get("tag"); got != "testing" {
		t.Fatalf("tag query = %q, want testing", got)
	}
	for _, want := range []string{"Testing Skill", "used 7 times", "[skill-test]"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q: %s", want, stdout)
		}
	}
}

func TestCmdShowRejectsMissingIDBeforeAPI(t *testing.T) {
	ctx := clitest.NewContext(t)
	if err := cmdShow(ctx, nil); err == nil {
		t.Fatal("expected missing skill id to fail")
	}
	ctx.RequireNoRequests()
}

func TestCmdListSurfacesAPIError(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Fail("GET", "/skills", errors.New("service unavailable"))

	err := cmdList(ctx, nil)
	if err == nil {
		t.Fatal("expected API error")
	}
	if !strings.Contains(err.Error(), "failed to list skills") || !strings.Contains(err.Error(), "service unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCmdReadPostsContractPayload(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("POST", "/skills/read", ReadResponse{
		Combined: "<skills />",
		Output:   "combined",
	})

	stdout, _, err := clitest.Output(t, func() error {
		return cmdRead(ctx, []string{"--resolve=name", "--output=combined", "--format=xml", "--with-scope", "unit-testing"})
	})
	if err != nil {
		t.Fatalf("cmdRead: %v", err)
	}
	if stdout != "<skills />" {
		t.Fatalf("stdout = %q, want combined skill output", stdout)
	}

	req := ctx.LastRequest()
	if req.Method != "POST" || req.Path != "/skills/read" {
		t.Fatalf("unexpected request: %+v", req)
	}
	payload, ok := req.Payload.(ReadRequest)
	if !ok {
		t.Fatalf("payload type = %T, want ReadRequest", req.Payload)
	}
	if strings.Join(payload.Identifiers, ",") != "unit-testing" || payload.Resolve != "name" || payload.Output != "combined" || payload.Format != "xml" || !payload.WithScope {
		t.Fatalf("unexpected read payload: %+v", payload)
	}
}

func TestThirdPartyHeaderNamesSourceAndExternalTools(t *testing.T) {
	skill := SkillResponse{ID: "brag", SkillDir: "/packs/vendor/brag", Origin: &SkillOriginView{Kind: "imported", SourceURL: "https://github.com/latent-spaces/brag", Commit: "1f8d9ade", License: "MIT"}, ExternalTools: []ExternalToolView{{Name: "hyperframes", URL: "https://hyperframes.heygen.com/", Purpose: "renders video"}}}
	skill.Origin.Review.Verdict = "passed"
	header := thirdPartyHeader(skill)
	for _, want := range []string{"THIRD-PARTY SKILL brag", "https://github.com/latent-spaces/brag", "license MIT", "/packs/vendor/brag", "not provided or managed by Vrooli", "hyperframes: renders video"} {
		if !strings.Contains(header, want) {
			t.Fatalf("header missing %q:\n%s", want, header)
		}
	}
	if thirdPartyHeader(SkillResponse{ID: "authored"}) != "" {
		t.Fatal("authored skills must not get a third-party header")
	}
}

func TestParseExternalToolRequiresName(t *testing.T) {
	tool, err := parseExternalTool("node | https://nodejs.org | runs hyperframes (>=22)")
	if err != nil || tool.Name != "node" || tool.URL != "https://nodejs.org" || tool.Purpose != "runs hyperframes (>=22)" {
		t.Fatalf("unexpected parse: %#v %v", tool, err)
	}
	if _, err := parseExternalTool(" |https://x"); err == nil {
		t.Fatal("empty name must be rejected")
	}
}
