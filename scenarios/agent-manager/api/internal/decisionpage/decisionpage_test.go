package decisionpage

import (
	"strings"
	"testing"
)

func TestNameReadsLikeWords(t *testing.T) {
	for slug, want := range map[string]string{
		"device-studio-delivery-2026-10-20": "Device studio delivery",
		"local-inference-maturity":          "Local inference maturity",
		"prompt-manager":                    "Prompt manager",
		"build_binary":                      "Build binary",
		"code.review":                       "Code review",
		"research-run-20261114":             "Research run",
	} {
		if got := Name(slug); got != want {
			t.Errorf("Name(%q) = %q, want %q", slug, got, want)
		}
	}
}

func TestDiffShowsEachFileColouredAndSaysWhatItCut(t *testing.T) {
	unified := "diff --git a/api/main.go b/api/main.go\n--- a/api/main.go\n+++ b/api/main.go\n@@ -1,2 +1,2 @@\n-old <line>\n+new line\n same\n" +
		"diff --git a/ui/app.tsx b/ui/app.tsx\n--- a/ui/app.tsx\n+++ b/ui/app.tsx\n@@ -1 +1 @@\n-" + strings.Repeat("x", 400) + "\n+y\n"
	got := Diff(unified, 200)
	for _, want := range []string{
		"<code>api/main.go</code>", `<span class="dp-add">+1</span>`, `<span class="dp-del">−1</span>`,
		`<span class="dp-del">-old &lt;line&gt;</span>`, `<span class="dp-add">+new line</span>`, "1 file more not shown",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("diff lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "+++ b/") {
		t.Error("diff headers repeat the file name the summary already shows")
	}
}
