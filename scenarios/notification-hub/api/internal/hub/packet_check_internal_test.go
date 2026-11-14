package hub

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInternalIDsFlagsRequesterLabelsAndSparesOrdinaryText(t *testing.T) {
	for name, tc := range map[string]struct {
		text string
		want []string
	}{
		"letter and hyphen digits":         {"Should we pick S-14 now?", []string{"S-14"}},
		"letter and digits":                {"Do O7 first or later?", []string{"O7"}},
		"long segmented id":                {"Ship F49-MOR-CHECKOUT this week?", []string{"F49-MOR-CHECKOUT"}},
		"decision id":                      {"Close BAS-D-065 or keep it open?", []string{"BAS-D-065"}},
		"prefixed number":                  {"Is BAS-065 still wanted?", []string{"BAS-065"}},
		"uuid":                             {"Retry 3f2a9c1b-0d4e-4f6a-8b7c-1e2d3c4b5a69?", []string{"3f2a9c1b-0d4e-4f6a-8b7c-1e2d3c4b5a69"}},
		"several in order, repeats once":   {"S-14 then O7, then S-14 again", []string{"S-14", "O7"}},
		"id at the start and end":          {"O7", []string{"O7"}},
		"inside brackets and quotes":       {`Skip "S-14" (O7)?`, []string{"S-14", "O7"}},
		"nested label reported as longest": {"Work on F49-MOR-CHECKOUT", []string{"F49-MOR-CHECKOUT"}},

		"plain words":            {"Should the backup run tonight or wait until the drive is back?", nil},
		"year":                   {"Renew the certificate before 2026?", nil},
		"date":                   {"Keep logs until 2026-11-13?", nil},
		"time of day":            {"Run at 10:30 or 22:00?", nil},
		"money":                  {"Spend $50 or $500 on the upgrade?", nil},
		"lowercase version":      {"Upgrade to v2 or stay on v1.4?", nil},
		"capital version":        {"Upgrade to V2.1 or stay?", nil},
		"quarter":                {"Plan for Q3 or Q4?", nil},
		"half year":              {"Ship in H1 or H2?", nil},
		"apple chip":             {"Should the build target M1 or M3?", nil},
		"latency percentile":     {"Is a P95 of 300 ms acceptable?", nil},
		"paper size":             {"Print on A4?", nil},
		"storage product":        {"Move the archive to S3?", nil},
		"encoding standard":      {"Use UTF-8 and SHA-256 for the export?", nil},
		"model name":             {"Switch to GPT-4 for summaries?", nil},
		"cve":                    {"Patch CVE-2026-1234 now?", nil},
		"gpu names":              {"Rent an A100 or an H100?", nil},
		"mixed letters digits":   {"Is the B2B plan or the E2E suite first?", nil},
		"all caps words":         {"Is the DO-NOT-DISTURB mode OK?", nil},
		"iso week":               {"Deploy in 2026-W45?", nil},
		"hyphenated lower":       {"Use the follow-up-2 plan?", nil},
		"percent and ranges":     {"Raise the limit 10-20% or 5-10?", nil},
		"acronym with plural":    {"Review the APIs and CLIs?", nil},
		"sentence ending in a-b": {"Pick option a-1 or b-2?", nil},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, internalIDs(tc.text))
		})
	}
}

func TestBracketedTag(t *testing.T) {
	require.Equal(t, "[BAS]", bracketedTag("[BAS] Close the goal?"))
	require.Equal(t, "[S-14]", bracketedTag("  [S-14] retry?"))
	require.Empty(t, bracketedTag("Close the goal [now]?"), "a bracket inside the sentence is not a prefix tag")
	require.Empty(t, bracketedTag("Close the goal?"))
}

func TestLocalPathsFindsFilePathsButNotUrlsOrRoutes(t *testing.T) {
	for name, tc := range map[string]struct {
		text string
		want []string
	}{
		"absolute under a host root": {"See /home/matthew/Vrooli/docs/plan.md for details.", []string{"/home/matthew/Vrooli/docs/plan.md"}},
		"tmp file":                   {"Output is in /tmp/run-42/out.txt.", []string{"/tmp/run-42/out.txt"}},
		"in parentheses":             {"the plan (/var/lib/app/state.json)", []string{"/var/lib/app/state.json"}},
		"in backticks":               {"edit `/etc/hosts` first", []string{"/etc/hosts"}},
		"home relative":              {"open ~/notes/todo.md", []string{"~/notes/todo.md"}},
		"dot relative":               {"run ./scripts/fix.sh and ../out/log.txt", []string{"./scripts/fix.sh", "../out/log.txt"}},
		"windows":                    {`copy C:\Users\me\file.txt over`, []string{`C:\Users\me\file.txt`}},
		"other root with extension":  {"Read /project/docs/plan.md carefully", []string{"/project/docs/plan.md"}},
		"two in order":               {"/home/a/b.txt and /home/a/c.txt", []string{"/home/a/b.txt", "/home/a/c.txt"}},

		"https url":             {"See https://example.com/docs/plan.md for details", nil},
		"http url with path":    {"Visit http://example.com/a/b/c.html", nil},
		"api route":             {"Call /api/v1/asks to list them", nil},
		"single segment route":  {"The /health page is green", nil},
		"fraction and ratios":   {"About 3/4 or 24/7, and/or both", nil},
		"slashes in a date":     {"On 2026/11/13 we decide", nil},
		"word with slash":       {"Pick read/write access", nil},
		"protocol relative":     {"load //cdn.example.com/lib.js", nil},
		"sentence period only":  {"Done. Now what?", nil},
		"extension without dir": {"Edit plan.md and notes.txt", nil},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, localPaths(tc.text))
		})
	}
}

func TestIsLocalContextURL(t *testing.T) {
	for _, local := range []string{
		"/home/matthew/report.md", "~/report.md", "./report.md", "../report.md", "file:///tmp/report.md",
		"http://localhost:3000/asks", "https://LOCALHOST/x", "http://127.0.0.1:8080/", "http://127.9.9.9/", "http://[::1]:8080/", "http://0.0.0.0/", "http://app.localhost/x",
	} {
		require.True(t, isLocalContextURL(local), local)
	}
	for _, public := range []string{
		"", "   ", "https://example.com/report", "http://192.168.1.20/report", "https://notification-hub.example.net/asks/1", "mailto:a@b.c",
	} {
		require.False(t, isLocalContextURL(public), public)
	}
}

func coverSpec() AskSpec {
	return AskSpec{
		Question:             "Should the nightly backup skip the Elements drive until it is plugged back in?",
		BigPicture:           "The drive has been unplugged since Tuesday, so tonight's backup would fail.",
		Recommended:          "skip",
		RecommendationReason: "Skipping avoids a failed run and a noisy alert.",
	}
}

func TestCheckPacketAcceptsAWellWrittenPacketSilently(t *testing.T) {
	require.Empty(t, CheckPacket(coverSpec(), nil, nil))
}

func TestCheckPacketCoverRules(t *testing.T) {
	long := coverSpec()
	long.Question = strings.Repeat("Should we keep going with this plan? ", 6)
	noPicture := coverSpec()
	noPicture.BigPicture = "  "
	noReason := coverSpec()
	noReason.RecommendationReason = ""
	noRecommendation := coverSpec()
	noRecommendation.Recommended, noRecommendation.RecommendationReason = "", ""
	id := coverSpec()
	id.Question = "Do S-14 and O7 before F49-MOR-CHECKOUT?"
	tag := coverSpec()
	tag.Question = "[BAS] Close the goal?"
	pathQuestion := coverSpec()
	pathQuestion.Question = "Is /home/matthew/Vrooli/docs/plan.md still right?"
	pathPicture := coverSpec()
	pathPicture.BigPicture = "The notes live at ~/notes/todo.md and /tmp/run/out.txt."
	local := coverSpec()
	local.ContextURL = "http://localhost:3000/asks/1"
	details := coverSpec()
	details.TechnicalDetails = "Log: /var/log/backup/run.log and /tmp/attached.txt"

	for name, tc := range map[string]struct {
		spec        AskSpec
		attachments []PacketAttachment
		want        []string
		message     string
	}{
		"internal ids":            {id, nil, []string{"internal_id_in_question@question"}, `"S-14", "O7", "F49-MOR-CHECKOUT"`},
		"bracket tag":             {tag, nil, []string{"internal_id_in_question@question"}, "[BAS]"},
		"path in question":        {pathQuestion, nil, []string{"path_in_question@question"}, "plan.md"},
		"paths in big picture":    {pathPicture, nil, []string{"path_in_question@big_picture"}, `"~/notes/todo.md", "/tmp/run/out.txt"`},
		"long question":           {long, nil, []string{"long_question@question"}, "characters"},
		"missing big picture":     {noPicture, nil, []string{"missing_big_picture@big_picture"}, "big picture"},
		"missing reason":          {noReason, nil, []string{"missing_recommendation_reason@recommendation_reason"}, "reason"},
		"no recommendation":       {noRecommendation, nil, nil, ""},
		"local context url":       {local, nil, []string{"local_context_url@context_url"}, "phone"},
		"unattached technical":    {details, []PacketAttachment{{Name: "attached", SourcePath: "/tmp/attached.txt"}}, []string{"unattached_path@technical_details"}, "/var/log/backup/run.log"},
		"attached folder matches": {details, []PacketAttachment{{Name: "logs", SourcePath: "/var/log/backup"}, {Name: "attached", SourcePath: "/tmp/attached.txt"}}, nil, ""},
	} {
		t.Run(name, func(t *testing.T) {
			warnings := CheckPacket(tc.spec, tc.attachments, nil)
			require.Equal(t, tc.want, warningCodes(warnings))
			for _, warning := range warnings {
				require.Equal(t, WarningAdvice, warning.Severity, "the packet check only advises")
				require.Contains(t, warning.Message, tc.message)
			}
		})
	}

	// A path the packet names in the question is advice about the question even
	// when the file is attached: the reader still meets a path in the sentence.
	attached := []PacketAttachment{{Name: "plan", SourcePath: "/home/matthew/Vrooli/docs/plan.md"}}
	require.Equal(t, []string{"path_in_question@question"}, warningCodes(CheckPacket(pathQuestion, attached, nil)))
}

const goodExplainer = `<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<style>body{font:16px system-ui} .hero{background:url(data:image/png;base64,AAAA)}</style></head>
<body><h1>Plan</h1><img src="chart.png" alt="chart"><a href="#details">details</a>
<a href="https://example.com/docs">docs</a><button data-hub-open="plan.md">Open plan</button>
<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"></svg>
<script>document.querySelector('button').addEventListener('click', () => hub.open("report.pdf"))</script>
</body></html>`

func explainerFor(page string) *PacketExplainer {
	return &PacketExplainer{Field: "attachments[0]", Name: "explainer.html", SourcePath: "/tmp/explainer.html", Size: int64(len(page)), HTML: []byte(page)}
}

func explainerAttachments() []PacketAttachment {
	return []PacketAttachment{{Name: "explainer.html"}, {Name: "chart.png"}, {Name: "plan.md"}, {Name: "report.pdf"}}
}

func TestCheckPacketAcceptsAGoodExplainer(t *testing.T) {
	require.Empty(t, CheckPacket(coverSpec(), explainerAttachments(), explainerFor(goodExplainer)))
}

func TestCheckPacketExplainerRules(t *testing.T) {
	for name, tc := range map[string]struct {
		page    string
		want    []string
		message string
	}{
		"stylesheet link":    {strings.Replace(goodExplainer, "<style>", `<link rel="stylesheet" href="https://cdn.example.com/site.css"><style>`, 1), []string{"explainer_network"}, "https://cdn.example.com/site.css"},
		"protocol relative":  {strings.Replace(goodExplainer, `<img src="chart.png"`, `<img src="//cdn.example.com/chart.png"`, 1), []string{"explainer_network"}, "//cdn.example.com/chart.png"},
		"remote image":       {strings.Replace(goodExplainer, `src="chart.png"`, `src="http://example.com/chart.png"`, 1), []string{"explainer_network"}, "http://example.com/chart.png"},
		"css url":            {strings.Replace(goodExplainer, "url(data:image/png;base64,AAAA)", "url('https://example.com/bg.png')", 1), []string{"explainer_network"}, "https://example.com/bg.png"},
		"css import":         {strings.Replace(goodExplainer, "<style>", `<style>@import url("https://fonts.example.com/f.css");`, 1), []string{"explainer_network"}, "https://fonts.example.com/f.css"},
		"fetch":              {strings.Replace(goodExplainer, "</script>", `fetch("https://api.example.com/data")</script>`, 1), []string{"explainer_network"}, "https://api.example.com/data"},
		"srcset":             {strings.Replace(goodExplainer, `<img src="chart.png"`, `<img src="chart.png" srcset="chart.png 1x, https://example.com/chart@2x.png 2x"`, 1), []string{"explainer_network"}, "chart@2x.png"},
		"no viewport":        {strings.Replace(goodExplainer, `<meta name="viewport" content="width=device-width, initial-scale=1">`, "", 1), []string{"explainer_no_viewport"}, "viewport"},
		"form":               {strings.Replace(goodExplainer, "<h1>Plan</h1>", `<h1>Plan</h1><form><input name="x"><textarea></textarea></form>`, 1), []string{"explainer_form"}, "<form>, <input>, <textarea>"},
		"select":             {strings.Replace(goodExplainer, "<h1>Plan</h1>", `<select><option>a</option></select>`, 1), []string{"explainer_form"}, "<select>"},
		"unknown image":      {strings.Replace(goodExplainer, `src="chart.png"`, `src="missing.png"`, 1), []string{"explainer_unknown_attachment"}, "missing.png"},
		"unknown open":       {strings.Replace(goodExplainer, `data-hub-open="plan.md"`, `data-hub-open="other.md"`, 1), []string{"explainer_unknown_attachment"}, "other.md"},
		"unknown hub.open":   {strings.Replace(goodExplainer, `hub.open("report.pdf")`, `hub.open('gone.pdf')`, 1), []string{"explainer_unknown_attachment"}, "gone.pdf"},
		"unknown link":       {strings.Replace(goodExplainer, `<a href="#details">`, `<a href="page2.html">`, 1), []string{"explainer_unknown_attachment"}, "page2.html"},
		"folder entry known": {strings.Replace(goodExplainer, `src="chart.png"`, `src="chart.png/inner/x.png"`, 1), nil, ""},
		"several problems":   {"<html><body><img src=\"https://x.test/a.png\"><input></body></html>", []string{"explainer_network", "explainer_no_viewport", "explainer_form"}, ""},
	} {
		t.Run(name, func(t *testing.T) {
			warnings := CheckPacket(coverSpec(), explainerAttachments(), explainerFor(tc.page))
			var got []string
			for _, warning := range warnings {
				got = append(got, warning.Code)
				require.Equal(t, "attachments[0]", warning.Field)
				require.Equal(t, WarningAdvice, warning.Severity)
			}
			require.Equal(t, tc.want, got)
			if tc.message != "" {
				require.Contains(t, warnings[0].Message, tc.message)
			}
		})
	}
}

func TestCheckPacketExplainerIgnoresCommentsScriptStringsAndSvgNamespaces(t *testing.T) {
	page := `<!doctype html><html><head><meta name="viewport" content="width=device-width">
<!-- <link href="https://cdn.example.com/x.css"> <input> <img src="ghost.png"> -->
</head><body>
<script>const html = '<a href="stale.png"><input></a>'; document.body.insertAdjacentHTML('beforeend', html)</script>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><use href="#icon"/></svg>
<a href="mailto:me@example.com">mail</a><a href="javascript:void(0)">noop</a><img src="data:image/gif;base64,R0lGODlhAQABAAAAACw=">
</body></html>`
	require.Empty(t, CheckPacket(coverSpec(), nil, explainerFor(page)))
}

func TestCheckPacketExplainerFormatRules(t *testing.T) {
	notHTML := &PacketExplainer{Field: "attachments[2]", Name: "plan.pdf", SourcePath: "/tmp/plan.pdf", MediaType: "application/pdf", Size: 10, HTML: []byte("%PDF-1.7")}
	require.Equal(t, []string{"explainer_not_html@attachments[2]"}, warningCodes(CheckPacket(coverSpec(), nil, notHTML)))
	folder := &PacketExplainer{Field: "attachments[0]", Name: "site", IsFolder: true}
	require.Equal(t, []string{"explainer_not_html@attachments[0]"}, warningCodes(CheckPacket(coverSpec(), nil, folder)))

	// HTML is recognized by extension, media type, or its first bytes, so a
	// page named "plan" with no extension is still checked as a page.
	for _, page := range []*PacketExplainer{
		{Field: "f", Name: "plan.HTM", HTML: []byte("<p>x</p>")},
		{Field: "f", Name: "plan", MediaType: "text/html; charset=utf-8", HTML: []byte("<p>x</p>")},
		{Field: "f", Name: "plan", HTML: []byte("\n  <!DOCTYPE html><p>x</p>")},
	} {
		require.Equal(t, []string{"explainer_no_viewport@f"}, warningCodes(CheckPacket(coverSpec(), nil, page)), page.Name)
	}

	big := &PacketExplainer{Field: "attachments[0]", Name: "huge.html", Size: MaxExplainerBytes + 1}
	warnings := CheckPacket(coverSpec(), nil, big)
	require.Equal(t, []string{"explainer_too_large@attachments[0]"}, warningCodes(warnings), "a page over the limit is not scanned")
	require.Contains(t, warnings[0].Message, "2 MiB")
	atLimit := &PacketExplainer{Field: "f", Name: "edge.html", Size: MaxExplainerBytes, HTML: []byte(goodExplainer)}
	require.NotContains(t, warningCodes(CheckPacket(coverSpec(), explainerAttachments(), atLimit)), "explainer_too_large@f")

	// An explainer the Hub could not read gets no content advice.
	unreadable := &PacketExplainer{Field: "attachments[0]", Name: "plan.html", Size: 100}
	require.Empty(t, CheckPacket(coverSpec(), nil, unreadable))
}
