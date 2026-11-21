package courier

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/operatordecision"

	"research-desk/internal/home"
)

// Every decision reaches the operator as a page: the one beside its brief in
// its own folder, sent with the images it names inlined. A decision whose
// folder is not ready for a cold executor is never sent. A page too wide for
// one decision goes as the summary page the desk builds from the item and its
// brief. The desk sends only a real page from inside the research home.

const page = "<!doctype html><title>Compare</title><h1>Keep or propose</h1>"

var av4 = AskPrefix + commission + "/AV4"

// av4Folder is AV4's decision folder, beside its commission's STATE.md.
func (r *rig) av4Folder() string {
	return filepath.Join(r.research, "cohorts", commission, "decisions", "AV4")
}

// withExplainer gives AV4 the `Explainer:` line, writing html to its page.html
// when html is not empty.
func (r *rig) withExplainer(t *testing.T, line, html string) string {
	t.Helper()
	path := filepath.Join(r.av4Folder(), "page.html")
	if html != "" {
		r.write(t, path, []byte(html))
	}
	r.setItems(t, strings.Replace(items, "Explainer: decisions/AV4/page.html", "Explainer: "+line, 1))
	return path
}

// O7(f): F7's packet was a workstream's brief and PF-KEYS asked for keys
// nobody had checked; the executors re-derived both. Such a decision never
// reaches the operator, and goes once its folder is ready.
func TestADecisionGoesOnlyWhenItsFolderIsReady(t *testing.T) {
	for name, c := range map[string]struct {
		change func(t *testing.T, r *rig)
		want   []string
	}{
		"no page": {func(t *testing.T, r *rig) { os.Remove(filepath.Join(r.av4Folder(), "page.html")) }, []string{"its page does not exist", "no decision-page writer"}},
		"a workstream's brief": {func(t *testing.T, r *rig) {
			r.setItems(t, strings.Replace(items, "decisions/AV4/", "work/W1/", 2))
		}, []string{"never a workstream's brief"}},
		"a thin brief": {func(t *testing.T, r *rig) {
			os.WriteFile(filepath.Join(r.av4Folder(), home.BriefFile), []byte("# W1 — Build the thing\n"), 0o644)
		}, []string{"no `Checked:", "`keep`, `propose`, `other`"}},
		"a premise checked weeks ago": {func(t *testing.T, r *rig) {
			os.WriteFile(filepath.Join(r.av4Folder(), home.BriefFile), append([]byte("Checked: 2026-10-01 still open, read in STATE.md\n"), readyBrief("keep", "propose")[len("Checked: 2026-11-12 the choice is still open, read in the commission's STATE.md\n"):]...), 0o644)
		}, []string{"is over 7 days old"}},
	} {
		t.Run(name, func(t *testing.T) {
			r := newRig(t, "desk")
			c.change(t, r)
			s := r.step(r.pass(t), av4)
			if len(r.hub.raised) != 0 || s.Reason != home.ReasonPacketNotReady || s.Action != ActNone {
				t.Fatalf("not sent: %+v", s)
			}
			for _, want := range c.want {
				if !strings.Contains(s.Why, want) {
					t.Fatalf("the plan row says %q: %q", want, s.Why)
				}
			}
			r.ready(t, r.av4Folder(), "keep", "propose")
			r.setItems(t, items)
			r.pass(t)
			if len(r.hub.raised) != 1 || r.hub.raised[0].Explainer != page {
				t.Fatalf("made ready, it goes with its page: %+v", r.hub.raised)
			}
		})
	}
}

func TestAPageTooWideForOneDecisionGoesAsTheDesksSummaryPage(t *testing.T) {
	r := newRig(t, "desk")
	r.withExplainer(t, "decisions/AV4/page.html", "<html>"+strings.Repeat("<b>", operatordecision.MaxExplainerBytes/12))
	plan := r.pass(t)
	if len(r.hub.raised) != 1 {
		t.Fatalf("one ask: %+v", r.hub.raised)
	}
	got := r.hub.raised[0].Explainer
	for _, want := range []string{"<!doctype html>", "<h1>yt-dlp target choice</h1>", "Keep or propose.", "Keep it uninstalled", "Ask for a maintained target proposal", "Recommended", `data-od-answer="keep"`, "Keep yt-dlp uninstalled and continue AV5–AV7.", "settles it this way", noAnswer} {
		if !strings.Contains(got, want) {
			t.Fatalf("the summary page holds %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Recommended: keep") || strings.Contains(got, commission) {
		t.Fatalf("the page holds the operator's text, not the item's field lines or IDs:\n%s", got)
	}
	if why := r.step(plan, av4).Why; !strings.Contains(why, "summary page instead of its own") {
		t.Fatalf("the step says its page was left out and why: %q", why)
	}
	r.pass(t)
	if len(r.hub.raised) != 1 {
		t.Fatalf("the summary page is a function of the item, so an unchanged item is not sent again: %d asks", len(r.hub.raised))
	}
}

func TestAnItemWithAnExplainerGoesOutWithItsPage(t *testing.T) {
	r := newRig(t, "desk")
	path := r.withExplainer(t, "decisions/AV4/page.html", page)

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
	dir := r.av4Folder()
	r.write(t, filepath.Join(dir, "img", "before.png"), pngBytes(64))
	r.write(t, filepath.Join(dir, "img", "after.webp"), []byte("RIFF\x00\x00\x00\x00WEBPVP8 "))
	outside := filepath.Join(t.TempDir(), "secret.png")
	r.write(t, outside, pngBytes(16))
	html := `<!doctype html><img src="img/before.png" alt="Before"><div style="background:url('img/after.webp')"></div>` +
		`<img src="https://example.com/x.png"><img src="#frag"><img src="` + outside + `"><img src="img/missing.png">`
	r.withExplainer(t, "decisions/AV4/page.html", html)

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
	dir := r.av4Folder()
	r.write(t, filepath.Join(dir, "small.png"), pngBytes(1024))
	r.write(t, filepath.Join(dir, "huge.png"), pngBytes(operatordecision.MaxExplainerBytes))
	r.withExplainer(t, "decisions/AV4/page.html", `<!doctype html><img src="small.png"><img src="huge.png">`)

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

func TestAPageThatIsNotBesideItsBriefStopsItsDecision(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "outside.html")
	if err := os.WriteFile(outside, []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct{ line, want string }{
		"outside the research home": {outside, "must sit beside the brief"},
		"missing, with no writer":   {"decisions/AV4/gone.html", "its page does not exist"},
		"not html":                  {"decisions/AV4/notes.md", "must name an .html page"},
	} {
		t.Run(name, func(t *testing.T) {
			r := newRig(t, "desk")
			r.write(t, filepath.Join(r.av4Folder(), "notes.md"), []byte(page))
			r.withExplainer(t, c.line, "")
			s := r.step(r.pass(t), av4)
			if len(r.hub.raised) != 0 || s.Reason != home.ReasonPacketNotReady || !strings.Contains(s.Why, c.want) {
				t.Fatalf("not sent, and the row says %q: %+v", c.want, s)
			}
		})
	}
}

func TestAPageNeverLeavesTheResearchHomeBySymlink(t *testing.T) {
	r := newRig(t, "desk")
	outside := filepath.Join(t.TempDir(), "secret.html")
	r.write(t, outside, []byte(page))
	path := filepath.Join(r.av4Folder(), "page.html")
	os.Remove(path)
	if err := os.Symlink(outside, path); err != nil {
		t.Skip("symlinks are not available here")
	}
	if s := r.step(r.pass(t), av4); len(r.hub.raised) != 0 || !strings.Contains(s.Why, "outside the research home") {
		t.Fatalf("a link out of the research home is not followed: %+v", s)
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
	r.write(t, filepath.Join(r.av4Folder(), "page.html"), []byte("<html>"+strings.Repeat("<b>", operatordecision.MaxExplainerBytes/12)))
	r.setItems(t, "\n## Needs you\n\n### AV4 — "+strings.Repeat("Q", 500)+"\n"+strings.Repeat("Why & <how> it matters. ", 400)+
		"\nRecommended: keep — "+strings.Repeat("k", 190)+"\nReason: "+strings.Repeat("r", 290)+"\nOptions: keep — "+strings.Repeat("k", 190)+"; propose — "+strings.Repeat("p", 190)+
		"\nPacket: decisions/AV4/BRIEF.md\nExplainer: decisions/AV4/page.html\n")
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
