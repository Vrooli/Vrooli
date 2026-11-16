package courier

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/operatordecision"
)

// Every decision reaches the operator as a page. An item's `Explainer:` line
// names its own page, sent with the images it names inlined; any other item,
// or one whose page cannot travel, goes with the summary page the desk builds
// from the item. The desk sends only a real page from inside the research home.

const page = "<!doctype html><title>Compare</title><h1>Keep or propose</h1>"

var av4 = AskPrefix + commission + "/AV4"

// withExplainer gives AV4 an `Explainer:` line, writing html to page.html in its
// commission folder when html is not empty.
func (r *rig) withExplainer(t *testing.T, line, html string) string {
	t.Helper()
	path := filepath.Join(r.research, "cohorts", commission, "page.html")
	if html != "" {
		r.write(t, path, []byte(html))
	}
	r.setItems(t, strings.Replace(items, "Options: keep", "Explainer: "+line+"\nOptions: keep", 1))
	return path
}

func (r *rig) write(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestADecisionWithoutItsOwnPageGoesWithTheDesksSummaryPage(t *testing.T) {
	r := newRig(t, "desk")
	r.pass(t)
	if len(r.hub.raised) != 1 {
		t.Fatalf("one ask: %+v", r.hub.raised)
	}
	got := r.hub.raised[0].Explainer
	for _, want := range []string{"<!doctype html>", "<h1>yt-dlp target choice</h1>", "Keep or propose.", "Keep it uninstalled", "Ask for a maintained target proposal", "Recommended", `data-od-answer="keep"`, "Keep yt-dlp uninstalled and continue AV5–AV7.", noAnswer} {
		if !strings.Contains(got, want) {
			t.Fatalf("the summary page holds %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Recommended: keep") || strings.Contains(got, commission) {
		t.Fatalf("the page holds the operator's text, not the item's field lines or IDs:\n%s", got)
	}
	if strings.Contains(r.hub.raised[0].TechnicalDetails, "Explainer:") {
		t.Fatalf("no item page is claimed: %s", r.hub.raised[0].TechnicalDetails)
	}
	r.pass(t)
	if len(r.hub.raised) != 1 {
		t.Fatalf("the summary page is a function of the item, so an unchanged item is not sent again: %d asks", len(r.hub.raised))
	}
}

func TestAnItemWithAnExplainerGoesOutWithItsPage(t *testing.T) {
	r := newRig(t, "desk")
	path := r.withExplainer(t, "page.html", page)

	r.pass(t)
	if len(r.hub.raised) != 1 || r.hub.raised[0].Explainer != page {
		t.Fatalf("the ask carries the page the item names: %+v", r.hub.raised)
	}
	spec := r.hub.raised[0]
	if !strings.Contains(spec.TechnicalDetails, "Explainer: "+path+" (sha256 ") {
		t.Fatalf("the technical details name the page and its hash:\n%s", spec.TechnicalDetails)
	}
	if strings.Contains(spec.BigPicture, "page.html") {
		t.Fatalf("the operator's text never holds the Explainer: line: %q", spec.BigPicture)
	}

	r.pass(t)
	if len(r.hub.raised) != 1 {
		t.Fatalf("an unchanged page is not sent again: %d asks", len(r.hub.raised))
	}

	revised := page + "<p>Revised after the last unit.</p>"
	r.write(t, path, []byte(revised))
	if got := r.step(r.pass(t), av4); got.Action != ActRefreshAsk {
		t.Fatalf("a changed page is a changed request, so the Hub's page is replaced: %+v", got)
	}
	if len(r.hub.raised) != 2 || r.hub.raised[1].Explainer != revised || r.hub.open(av4) != 1 {
		t.Fatalf("the revised page replaces the open ask's, with one ask open: %+v", r.hub.raised)
	}
}

// pngBytes is a PNG signature padded to n bytes.
func pngBytes(n int) []byte {
	return append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{7}, n-8)...)
}

func TestTheImagesAPageNamesTravelInsideIt(t *testing.T) {
	r := newRig(t, "desk")
	dir := filepath.Join(r.research, "cohorts", commission)
	r.write(t, filepath.Join(dir, "img", "before.png"), pngBytes(64))
	r.write(t, filepath.Join(dir, "img", "after.webp"), []byte("RIFF\x00\x00\x00\x00WEBPVP8 "))
	outside := filepath.Join(t.TempDir(), "secret.png")
	r.write(t, outside, pngBytes(16))
	html := `<!doctype html><img src="img/before.png" alt="Before"><div style="background:url('img/after.webp')"></div>` +
		`<img src="https://example.com/x.png"><img src="#frag"><img src="` + outside + `"><img src="img/missing.png">`
	r.withExplainer(t, "page.html", html)

	plan := r.pass(t)
	got := r.hub.raised[0].Explainer
	if !strings.Contains(got, `src="data:image/png;base64,`+base64.StdEncoding.EncodeToString(pngBytes(64))+`"`) || !strings.Contains(got, "url('data:image/webp;base64,") {
		t.Fatalf("local images inside the home are inlined in place:\n%s", got)
	}
	for _, kept := range []string{`src="https://example.com/x.png"`, `src="#frag"`, `src="` + outside + `"`, `src="img/missing.png"`} {
		if !strings.Contains(got, kept) {
			t.Fatalf("%s is left as written", kept)
		}
	}
	why := r.step(plan, av4).Why
	for _, want := range []string{"sent with its page, without 2 image(s)", outside + " (is outside the research home", "img/missing.png (cannot be read"} {
		if !strings.Contains(why, want) {
			t.Fatalf("the step names what did not travel (%q): %q", want, why)
		}
	}
	if strings.Contains(why, "example.com") {
		t.Fatalf("a network reference is the page's own business, not a local image left out: %q", why)
	}
}

func TestAnImageThatWouldOverflowTheDecisionIsLeftOutAndNamed(t *testing.T) {
	r := newRig(t, "desk")
	dir := filepath.Join(r.research, "cohorts", commission)
	r.write(t, filepath.Join(dir, "small.png"), pngBytes(1024))
	r.write(t, filepath.Join(dir, "huge.png"), pngBytes(operatordecision.MaxExplainerBytes))
	r.withExplainer(t, "page.html", `<!doctype html><img src="small.png"><img src="huge.png">`)

	plan := r.pass(t)
	spec := r.hub.raised[0]
	if !strings.Contains(spec.Explainer, "data:image/png") || !strings.Contains(spec.Explainer, `src="huge.png"`) {
		t.Fatalf("what fits is inlined, what does not is left as written")
	}
	if facts, err := askFacts(spec); err != nil || facts.Explainer != spec.Explainer {
		t.Fatalf("the page still fits one decision: %v", err)
	}
	if why := r.step(plan, av4).Why; !strings.Contains(why, "huge.png (") || !strings.Contains(why, "more than one decision carries") {
		t.Fatalf("the step names the image left out and why: %q", why)
	}
}

func TestAPageThatCannotBeSentNeverStopsItsItem(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "outside.html")
	if err := os.WriteFile(outside, []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, line := range map[string]string{
		"outside the research home": outside,
		"missing":                   "gone.html",
		"not html":                  "notes.md",
	} {
		t.Run(name, func(t *testing.T) {
			r := newRig(t, "desk")
			r.write(t, filepath.Join(r.research, "cohorts", commission, "notes.md"), []byte(page))
			r.withExplainer(t, line, "")
			plan := r.pass(t)
			if len(r.hub.raised) != 1 || !strings.Contains(r.hub.raised[0].Explainer, noAnswer) {
				t.Fatalf("the item goes with its summary page: %+v", r.hub.raised)
			}
			if why := r.step(plan, av4).Why; !strings.Contains(why, "summary page instead of its own") {
				t.Fatalf("the step says the page was left out and why: %q", why)
			}
		})
	}
}

func TestAPageNeverLeavesTheResearchHomeBySymlink(t *testing.T) {
	r := newRig(t, "desk")
	outside := filepath.Join(t.TempDir(), "secret.html")
	r.write(t, outside, []byte(page))
	if err := os.Symlink(outside, filepath.Join(r.research, "cohorts", commission, "page.html")); err != nil {
		t.Skip("symlinks are not available here")
	}
	r.withExplainer(t, "page.html", "")
	r.pass(t)
	if len(r.hub.raised) != 1 || r.hub.raised[0].Explainer == page {
		t.Fatalf("a link out of the research home is not followed: %+v", r.hub.raised)
	}
}

func TestLoadPageRefusesWhatIsNotAPageThatFitsOneDecision(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	for name, path := range map[string]string{
		"empty":           write("empty.html", ""),
		"not a page":      write("key.html", "-----BEGIN PRIVATE KEY-----\nabc\n"),
		"over the file":   write("big.html", "<html>"+strings.Repeat("x", maxPageBytes)),
		"over a decision": write("wide.html", "<html>"+strings.Repeat("<b>", operatordecision.MaxExplainerBytes/12)),
		"directory":       dir,
	} {
		if _, err := loadPage(path, dir); err == nil {
			t.Errorf("%s: a file that is not a page within the limit must be refused", name)
		}
	}
	got, err := loadPage(write("ok.HTML", "  <HTML><body>fine"), dir)
	if err != nil || got.SHA256 == "" {
		t.Fatalf("a page is read whatever the case of its extension and tags: %v", err)
	}
}

func TestTheLongestItemsSummaryPageFitsOneDecision(t *testing.T) {
	r := newRig(t, "desk")
	r.setItems(t, "\n## Needs you\n\n### AV4 — "+strings.Repeat("Q", 500)+"\n"+strings.Repeat("Why & <how> it matters. ", 400)+
		"\nRecommended: keep — "+strings.Repeat("k", 190)+"\nReason: "+strings.Repeat("r", 290)+"\nOptions: keep — "+strings.Repeat("k", 190)+"; propose — "+strings.Repeat("p", 190)+"\n")
	r.pass(t)
	if len(r.hub.raised) != 1 {
		t.Fatalf("one ask: %+v", r.hub.raised)
	}
	if _, err := askFacts(r.hub.raised[0]); err != nil {
		t.Fatalf("a full-size item's page is a valid decision: %v", err)
	}
}

func TestAnItemThatBreaksTheGrammarSaysSoOnItsPlanRow(t *testing.T) {
	r := newRig(t, "desk")
	r.setItems(t, strings.Replace(items, "Options: keep — Keep it uninstalled;", "Options: keep — Keep it uninstalled; Maybe later;", 1))
	if why := r.step(r.pass(t), av4).Why; !strings.Contains(why, `malformed option "Maybe later"`) || !strings.Contains(why, "needs_you.py validate") {
		t.Fatalf("the plan row names the problem and the check: %q", why)
	}
}
