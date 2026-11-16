package evidenceretention

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"agent-manager/internal/decisionpage"

	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/api-core/retention"
)

// classWords names each regenerable class for an operator who has never seen
// the class keys: what the files are and how they come back.
var classWords = map[string][2]string{
	string(ClassBuildBinary):     {"Compiled programs", "Rebuilt from source by the next build."},
	string(ClassDependencyTree):  {"Downloaded libraries", "Downloaded again by the next install."},
	string(ClassPlatformRelease): {"Desktop and mobile app builds", "Rebuilt by the next packaging run."},
	string(ClassReleaseArchive):  {"Zipped release copies", "Recreated by the next release."},
	string(ClassRenderFrames):    {"Rendered video frames", "Rendered again from their source."},
}

func className(class string) string {
	if w, ok := classWords[class]; ok {
		return w[0]
	}
	return decisionpage.Name(class)
}

// classBreakdown is the stored preview grouped by class: size, files, paths.
type classTotal struct {
	Bytes int64 `json:"bytes"`
	Files int   `json:"files"`
	Paths int   `json:"paths"`
}

func breakdown(preview Receipt) map[string]classTotal {
	totals := map[string]classTotal{}
	for _, r := range preview.Removed {
		t := totals[string(r.Class)]
		t.Bytes += r.Bytes
		t.Files += r.Files
		t.Paths++
		totals[string(r.Class)] = t
	}
	return totals
}

// approvalFacts builds the operator ask entirely from the stored ask, so a
// republish of the same generation yields the same event.
func approvalFacts(ask *operatordecision.PendingAsk) operatordecision.Facts {
	x := ask.Extra
	efforts := decodeList(x[extraEfforts])
	classes := decodeList(x[extraClasses])
	paths := decodeList(x[extraPaths])
	skipped := decodeList(x[extraSkipped])
	bytes, _ := strconv.ParseInt(x[extraPreviewBytes], 10, 64)
	files, _ := strconv.Atoi(x[extraPreviewFiles])
	size := retention.FormatBytes(bytes)
	var byClass map[string]classTotal
	_ = decodeJSON(x[extraBreakdown], &byClass)

	kinds := []string{}
	for _, class := range classes {
		kinds = append(kinds, strings.ToLower(className(class)))
	}
	kindText := "rebuildable build files"
	if len(kinds) > 0 {
		kindText = joinWords(kinds)
	}
	project := "a finished project"
	if len(efforts) == 1 {
		project = "the finished " + decisionpage.Name(efforts[0]) + " project"
	} else if len(efforts) > 1 {
		project = fmt.Sprintf("%d finished projects", len(efforts))
	}

	var tech strings.Builder
	fmt.Fprintf(&tech, "Requested by: %s\nReason: %s\nEfforts: %s\nClasses: %s\nWould remove: %s in %d files across %d paths\n",
		x[extraRequestedBy], x[extraReason], strings.Join(efforts, ", "), strings.Join(classes, ", "), size, files, len(paths))
	tech.WriteString("Paths (relative to the evidence home):\n")
	for i, p := range paths {
		if i == 200 {
			fmt.Fprintf(&tech, "  ... and %d more\n", len(paths)-i)
			break
		}
		fmt.Fprintf(&tech, "  - %s\n", p)
	}
	if len(skipped) > 0 {
		tech.WriteString("Kept by the preview:\n")
		for i, s := range skipped {
			if i == 50 {
				fmt.Fprintf(&tech, "  ... and %d more\n", len(skipped)-i)
				break
			}
			fmt.Fprintf(&tech, "  - %s\n", s)
		}
	}

	correlation := map[string]string{
		correlationRequestID:                      ask.ID,
		operatordecision.CorrelationAgentAwaitKey: ask.ID,
	}
	if x[extraRunID] != "" {
		correlation[operatordecision.CorrelationRaisedByRunID] = x[extraRunID]
	}
	if x[extraSession] != "" {
		correlation[operatordecision.CorrelationWebConsoleSessionID] = x[extraSession]
	}
	f := operatordecision.Facts{
		SubjectKey: subjectKeyFor(ask.ID),
		Subject:    fmt.Sprintf("Delete %s of rebuildable files from %s?", size, project),
		Question: operatordecision.Clip(fmt.Sprintf("An agent asks to permanently delete %s of %s left by %s. They can be rebuilt but not restored. Delete them?",
			size, kindText, project), operatordecision.MaxQuestion),
		BigPicture: "These files were produced while building and checking the project's results. They are kept only so the results can be rebuilt or re-checked exactly. " +
			"Deleting frees the space; the files can be rebuilt later but never restored as they are. Agent Manager deletes only the listed files that are still unused when it runs, and records you as the approver. " +
			"Keeping them deletes nothing. If you do not answer, nothing is deleted.",
		TechnicalDetails: operatordecision.Clip(tech.String(), operatordecision.MaxTechnicalDetailsCodePoints),
		Options: []operatordecision.Option{
			{Key: approveAnswer, Label: "Delete them and free " + size},
			{Key: "deny", Label: "Keep them"},
		},
		// Deletion is irreversible: recommend deny, never auto-apply at a deadline.
		Recommended:          "deny",
		RecommendationReason: "Deletion cannot be undone; approve only when the project is finished and nobody needs to rebuild its results exactly.",
		Reversible:           false,
		Urgency:              "normal",
		Correlation:          correlation,
	}

	page := operatordecision.NewPage(f.Subject).
		Lead("An agent wants to free disk space by deleting files that a finished project left behind. You decide because deletion cannot be undone.")
	rows := []operatordecision.Row{{Label: "Frees", Value: fmt.Sprintf("%s in %s", size, operatordecision.Count(files, "file", "files"))}}
	if len(efforts) > 0 {
		names := make([]string, 0, len(efforts))
		for _, e := range efforts {
			names = append(names, decisionpage.Name(e))
		}
		rows = append(rows, operatordecision.Row{Label: "From", Value: strings.Join(names, ", ")})
	}
	if reason := strings.TrimSpace(x[extraReason]); reason != "" {
		rows = append(rows, operatordecision.Row{Label: "The agent's reason", Value: operatordecision.Clip(reason, 400)})
	}
	page.Facts(rows...)
	if len(byClass) > 0 {
		keys := make([]string, 0, len(byClass))
		for class := range byClass {
			keys = append(keys, class)
		}
		sort.Slice(keys, func(i, j int) bool { return byClass[keys[i]].Bytes > byClass[keys[j]].Bytes })
		table := [][]string{}
		for _, class := range keys {
			t := byClass[class]
			table = append(table, []string{className(class), retention.FormatBytes(t.Bytes), strconv.Itoa(t.Files), classWords[class][1]})
		}
		page.Section("What would be deleted", "").Table([]string{"Kind of file", "Size", "Files", "How it comes back"}, table)
	}
	if len(skipped) > 0 {
		page.Callout("", "Some files are kept:", operatordecision.Count(len(skipped), "item was", "items were")+" left out because something still uses them or they are not rebuildable.")
	}
	page.Options(f, map[string]string{
		approveAnswer: "Agent Manager deletes exactly these files now and writes a receipt naming you. Nothing else is touched.",
		"deny":        "Nothing is deleted. The space stays used.",
	})
	f.Explainer = page.IfNoAnswer("Nothing is deleted.").HTML()
	return f
}

func joinWords(words []string) string {
	switch len(words) {
	case 0:
		return ""
	case 1:
		return words[0]
	}
	return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
}
