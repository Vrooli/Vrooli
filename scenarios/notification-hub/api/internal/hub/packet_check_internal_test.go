package hub

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

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
		Packet:               "/home/me/.vrooli/research/work/BACKUP-SKIP/BRIEF.md",
	}
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
	idsBelow := coverSpec()
	idsBelow.BigPicture = "This moves O5 M1 through the delivery step. Nothing is spent."
	idsBelow.RecommendationReason = "It closes S-18 sooner."
	idsBelow.Options = append(idsBelow.Options, AskOption{Key: "later", Label: "Wait for F11 first"})
	pathQuestion := coverSpec()
	pathQuestion.Question = "Is /home/matthew/Vrooli/docs/plan.md still right?"
	pathPicture := coverSpec()
	pathPicture.BigPicture = "The notes live at ~/notes/todo.md and /tmp/run/out.txt."
	local := coverSpec()
	local.ContextURL = "http://localhost:3000/asks/1"
	details := coverSpec()
	details.TechnicalDetails = "Log: /var/log/backup/run.log and /tmp/attached.txt"
	noPacket := coverSpec()
	noPacket.Packet = ""
	scenarioDecision := noPacket
	scenarioDecision.SourceEventType = "storage_manager.deletion.decision_requested.v1"

	for name, tc := range map[string]struct {
		spec        AskSpec
		attachments []PacketAttachment
		want        []string
		message     string
	}{
		"internal ids":              {id, nil, []string{"internal_id_in_question@question"}, `"S-14", "O7", "F49-MOR-CHECKOUT"`},
		"bracket tag":               {tag, nil, []string{"internal_id_in_question@question"}, "[BAS]"},
		"ids below the question":    {idsBelow, nil, []string{"internal_id_in_question@big_picture", "internal_id_in_question@recommendation_reason", fmt.Sprintf("internal_id_in_question@options[%d].label", len(idsBelow.Options)-1)}, "technical_details"},
		"path in question":          {pathQuestion, nil, []string{"path_in_question@question"}, "plan.md"},
		"paths in big picture":      {pathPicture, nil, []string{"path_in_question@big_picture"}, `"~/notes/todo.md", "/tmp/run/out.txt"`},
		"long question":             {long, nil, []string{"long_question@question"}, "characters"},
		"missing big picture":       {noPicture, nil, []string{"missing_big_picture@big_picture"}, "big picture"},
		"missing reason":            {noReason, nil, []string{"missing_recommendation_reason@recommendation_reason"}, "reason"},
		"no recommendation":         {noRecommendation, nil, nil, ""},
		"local context url":         {local, nil, []string{"local_context_url@context_url"}, "phone"},
		"unattached technical":      {details, []PacketAttachment{{Name: "attached", SourcePath: "/tmp/attached.txt"}}, []string{"unattached_path@technical_details"}, "/var/log/backup/run.log"},
		"agent decision, no packet": {noPacket, nil, []string{"packet_missing@packet"}, "--packet"},
		"scenario decision":         {scenarioDecision, nil, nil, ""},
		"attached folder matches":   {details, []PacketAttachment{{Name: "logs", SourcePath: "/var/log/backup"}, {Name: "attached", SourcePath: "/tmp/attached.txt"}}, nil, ""},
	} {
		t.Run(name, func(t *testing.T) {
			// Each case has a good page, so only the cover rule under test speaks.
			warnings := CheckPacket(tc.spec, append(tc.attachments, explainerAttachments()...), explainerFor(goodExplainer))
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
	require.Equal(t, []string{"path_in_question@question"}, warningCodes(CheckPacket(pathQuestion, append(attached, explainerAttachments()...), explainerFor(goodExplainer))))
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
		"several problems":   {"<html><body><img src=\"https://x.test/a.png\"><input></body></html>", []string{"explainer_network", "explainer_no_viewport", "explainer_form", "explainer_restates"}, ""},
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
		// A page of one word is read as a page, and restates the cover.
		require.Equal(t, []string{"explainer_no_viewport@f", "explainer_restates@f"}, warningCodes(CheckPacket(coverSpec(), nil, page)), page.Name)
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
