package incidents

import (
	"fmt"
	"html"
	"io"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/operatordecision"
)

// maxScriptOnPage bounds how much of a generated script the page shows; the
// whole script stays at its artifact path.
const maxScriptOnPage = 8 << 10

// readArtifact reads the start of a generated artifact for the page. A test
// replaces it; an unreadable artifact is simply not shown.
var readArtifact = func(path string) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxScriptOnPage+1))
	if err != nil {
		return "", false
	}
	return string(data), true
}

// remediationDecisionFacts asks whether to run one generated fix: the cover
// names the problem, the fix and its risk; the page shows what autoheal saw,
// what the fix does, its checks and fallback, and the script itself.
func remediationDecisionFacts(incident Incident, candidate RemediationCandidate, deadline time.Time) operatordecision.Facts {
	risk := remediationRiskLabel(candidate.RiskLevel)
	problem, fix := operatordecision.Clip(incident.Title, 80), operatordecision.Clip(candidate.Title, 80)
	if fix == "" {
		fix = "a generated fix"
	}
	admin := ""
	if candidate.RequiresPrivilege {
		admin = " It runs with administrator rights."
	}
	about := operatordecision.Clip(firstNonEmpty(incident.Diagnosis, incident.Summary), 400)
	if about != "" {
		about += "\n\n"
	}
	var artifacts []RemediationArtifact
	for _, artifact := range incident.RemediationArtifacts {
		if artifact.RemediationID == candidate.ID {
			artifacts = append(artifacts, artifact)
		}
	}
	tech := strings.Builder{}
	fmt.Fprintf(&tech, "Incident: %s (fingerprint %s, severity %s, seen %d times). Candidate: %s, template %s, risk %s, applicability %s, platforms %s.\n",
		incident.ID, incident.Fingerprint, incident.Severity, incident.ObservationCount, candidate.ID, candidate.TemplateID, risk, candidate.Applicability, strings.Join(candidate.Platforms, ","))
	for _, artifact := range artifacts {
		fmt.Fprintf(&tech, "Artifact %s: %s (generated %s)\n", artifact.ID, artifact.Path, artifact.GeneratedAt.UTC().Format(time.RFC3339))
	}
	if candidate.ArtifactPolicy != "" {
		fmt.Fprintf(&tech, "Artifact policy: %s\n", candidate.ArtifactPolicy)
	}
	if candidate.DecisionPrompt != "" {
		fmt.Fprintf(&tech, "Candidate prompt: %s\n", candidate.DecisionPrompt)
	}
	facts := operatordecision.Facts{
		Subject:  operatordecision.Clip("Approve a fix for: "+problem, 110),
		Question: fmt.Sprintf("Autoheal found “%s” and proposes “%s” (%s risk). Run it?", problem, fix, risk),
		Options: []operatordecision.Option{
			{Key: "approve", Label: "Run the fix once, with its checks before and after"},
			{Key: "reject", Label: "Do not run it: the incident stays open for a person"},
		},
		Recommended:          "reject",
		RecommendationReason: "A generated host script can affect the system; read the script and its checks on the page before approving.",
		Reversible:           false,
		Urgency:              "high",
		BigPicture: fmt.Sprintf("%sAutoheal generated a script to fix this.%s Only you can approve it, because it changes the machine.\n\n"+
			"If you approve, its checks run first, then the script, then checks confirm the fix. If you decline or do not answer by %s, nothing runs and the incident stays open.",
			about, admin, deadline.UTC().Format("15:04 MST on 2 Jan")),
		TechnicalDetails: tech.String(),
		Deadline:         deadline.UTC(),
		SubjectKey:       RemediationDecisionSubjectKey(incident.ID),
	}

	page := operatordecision.NewPage(fmt.Sprintf("Run “%s”?", fix))
	if incident.Summary != "" {
		page.Lead(incident.Summary)
	}
	rows := []operatordecision.Row{{Label: "Problem", Value: incident.Title}, {Label: "Severity", Value: string(incident.Severity)}}
	if !incident.DetectedAt.IsZero() {
		rows = append(rows, operatordecision.Row{Label: "First seen", Value: incident.DetectedAt.UTC().Format("2 Jan 15:04 MST")})
	}
	if incident.ObservationCount > 0 {
		rows = append(rows, operatordecision.Row{Label: "Seen", Value: fmt.Sprintf("%d times", incident.ObservationCount)})
	}
	if incident.Confidence != "" {
		rows = append(rows, operatordecision.Row{Label: "Diagnosis confidence", Value: incident.Confidence})
	}
	page.Facts(rows...)
	if incident.Diagnosis != "" {
		page.Section("What autoheal found", incident.Diagnosis)
	}
	var evidence [][]string
	for i, item := range incident.EvidenceItems {
		if i == 12 {
			break
		}
		evidence = append(evidence, []string{item.Summary, item.Source, string(item.Severity)})
	}
	page.Table([]string{"Evidence", "From", "Severity"}, evidence)

	tone := "warning"
	if strings.EqualFold(candidate.RiskLevel, "high") || strings.EqualFold(candidate.RiskLevel, "critical") {
		tone = "danger"
	}
	page.Section("The proposed fix", fix).
		Facts(
			operatordecision.Row{Label: "Risk", Value: risk},
			operatordecision.Row{Label: "Administrator rights", Value: yesNo(candidate.RequiresPrivilege)},
			operatordecision.Row{Label: "Runs on", Value: firstNonEmpty(strings.Join(candidate.Platforms, ", "), "this machine")},
		).
		Callout(tone, "Changes the machine:", "the script runs on this computer. Read it below before approving.")
	var steps [][]string
	for _, check := range candidate.PreflightChecks {
		steps = append(steps, []string{"Before", check})
	}
	if candidate.Simulation != "" {
		steps = append(steps, []string{"Dry run", candidate.Simulation})
	}
	for _, check := range candidate.PostChecks {
		steps = append(steps, []string{"After", check})
	}
	for _, fallback := range candidate.RollbackOrFallback {
		steps = append(steps, []string{"If it fails", fallback})
	}
	page.Table([]string{"When", "Check or fallback"}, steps)
	for _, artifact := range artifacts {
		if script, ok := readArtifact(artifact.Path); ok {
			note := ""
			if len(script) > maxScriptOnPage {
				script, note = strings.ToValidUTF8(script[:maxScriptOnPage], ""), " (start only; the whole script is at the path in the technical details)"
			}
			page.Section("The script"+note, "").Raw(`<pre style="overflow-x:auto;white-space:pre"><code>` + html.EscapeString(strings.ToValidUTF8(script, "")) + "</code></pre>")
		}
	}
	facts.Explainer = page.Options(facts, map[string]string{
		"approve": "The checks run first. If they pass, the script runs once, then the after-checks confirm the fix. If anything fails, the fallback above applies.",
		"reject":  "Nothing runs. The incident stays open for a person to handle.",
	}).
		IfNoAnswer(fmt.Sprintf("At %s the request closes. Nothing runs and the incident stays open.", deadline.UTC().Format("15:04 MST on 2 Jan"))).
		HTML()
	return facts
}

func yesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
