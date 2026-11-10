package repohealth

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/operatordecision"
)

// Option keys the Notification Hub returns when the operator answers.
const (
	OptAccept = "accept_exception"
	OptRepair = "repair_drive"
	OptFresh  = "fresh_repository"
	OptLeave  = "leave_blocked"
)

// Offers is which actions the code can truly perform for a diagnosis. An action
// that cannot run is never offered.
type Offers struct {
	Accept, Repair, Fresh bool
}

// OffersFor decides what to offer. repairPlannable is whether the control plane
// could build an identity-bound unmount/repair/remount plan for this drive.
func OffersFor(d Diagnosis, repairPlannable bool) Offers {
	switch d.Cause {
	case CauseLeftoverTempUnreadable:
		return Offers{Accept: true, Repair: repairPlannable, Fresh: true}
	case CauseContentUnreadable, CauseSnapshotsUnlistable, CauseRepositoryUnopenable, CauseUsageUnexplained:
		return Offers{Repair: repairPlannable, Fresh: true}
	case CauseVolumeReadOnly:
		return Offers{Repair: repairPlannable}
	default:
		return Offers{}
	}
}

// Askable reports whether a diagnosis leads to a question. A cleared failure and
// an unmounted volume (owned by the existing drive recovery) do not.
func (o Offers) Askable() bool { return o.Accept || o.Repair || o.Fresh }

// Recommended is the option to suggest, with the plain reason.
func Recommended(d Diagnosis, o Offers) (key, reason string) {
	switch {
	case o.Accept:
		return OptAccept, fmt.Sprintf("They are unfinished uploads, not saved backup data, and your %d earlier backups still list normally. This restarts protection soonest and changes nothing on the drive.", d.SnapshotCount)
	case o.Repair:
		return OptRepair, "It is the only fix I can run myself. Unreadable saved data can make a backup impossible to restore, so it should not be ignored."
	default:
		return OptLeave, "Nothing I can safely run applies here; the drive needs a person or an agent to look at it."
	}
}

type letteredOption struct {
	key, action, label string
}

func plural1(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// DriveSentence names the drive the way a person would: "the Elements drive".
func (d Diagnosis) DriveSentence() string {
	if strings.TrimSpace(d.Volume.Label) != "" {
		return "the " + d.Volume.Label + " drive"
	}
	return d.DestinationName
}

func (d Diagnosis) problemSentence() string {
	n := len(d.Unreadable)
	switch d.Cause {
	case CauseLeftoverTempUnreadable:
		return fmt.Sprintf("%d unfinished temporary %s on the drive can't be read", n, plural1(n, "file", "files"))
	case CauseContentUnreadable:
		return fmt.Sprintf("%d %s inside the backup set can't be read", n, plural1(n, "file", "files"))
	case CauseSnapshotsUnlistable:
		return "the backup set's list of saved backups can't be read"
	case CauseRepositoryUnopenable:
		return "the backup set can't be opened"
	case CauseVolumeReadOnly:
		return "the drive is mounted read-only"
	default:
		return "the backup set's size can't be measured"
	}
}

func (d Diagnosis) stoppedSentence() string {
	if d.LastSuccessAt.IsZero() {
		return "No successful backup is on record."
	}
	days := d.DaysStopped()
	when := d.LastSuccessAt.Format("2 Jan")
	switch {
	case days <= 0:
		return "Your last good backup was today."
	default:
		return fmt.Sprintf("Your last good backup was %s (%d %s ago).", when, days, plural1(days, "day", "days"))
	}
}

func (d Diagnosis) unprotectedSentence() string {
	if d.Plans.TargetTotal == 0 {
		return ""
	}
	if d.Plans.Unprotected == 0 {
		return "Everything in the plan also has another copy."
	}
	return fmt.Sprintf("%d of the %d items in the plan have no other backup copy.", d.Plans.Unprotected, d.Plans.TargetTotal)
}

func (d Diagnosis) subject() string {
	days := d.DaysStopped()
	if days >= 2 {
		return fmt.Sprintf("Backups to %s have been stopped for %d days", d.DriveSentence(), days)
	}
	return fmt.Sprintf("Backups to %s are blocked", d.DriveSentence())
}

// lettered lists the offered options in a fixed order with their letters.
func lettered(d Diagnosis, o Offers, recommended string) []letteredOption {
	var opts []letteredOption
	n := len(d.Unreadable)
	if o.Accept {
		opts = append(opts, letteredOption{
			key:    OptAccept,
			action: fmt.Sprintf("skip only %s and resume", plural1(n, "that file", fmt.Sprintf("those %d files", n))),
			label:  fmt.Sprintf("Skip only %s and resume backups. Low risk; I keep warning you while %s.", plural1(n, "that unreadable file", fmt.Sprintf("those %d unreadable files", n)), plural1(n, "it exists", "they exist")),
		})
	}
	if o.Repair {
		label := "Take the drive offline and repair it. Backups pause meanwhile; a repair can discard damaged data"
		if !d.PriorRepairAt.IsZero() && (d.Cause == CauseLeftoverTempUnreadable || d.Cause == CauseContentUnreadable) {
			label += fmt.Sprintf(", and the repair on %s did not clear this.", d.PriorRepairAt.Format("2 Jan"))
		} else {
			label += "."
		}
		opts = append(opts, letteredOption{key: OptRepair, action: "take the drive offline and repair it", label: label})
	}
	if o.Fresh {
		opts = append(opts, letteredOption{
			key: OptFresh, action: "start a fresh backup set",
			label: "Start a fresh backup set on this drive and keep the old one read-only. Uses more space; an agent has to set it up, so this only asks for that.",
		})
	}
	opts = append(opts, letteredOption{
		key: OptLeave, action: "do nothing for now",
		label: "Do nothing for now. Backups stay stopped and I ask again tomorrow.",
	})
	for i := range opts {
		letter := string(rune('A' + i))
		prefix := letter + ": "
		if opts[i].key == recommended {
			prefix = letter + " (recommended): "
		}
		opts[i].label = prefix + opts[i].label
		if opts[i].key == recommended {
			opts[i].action = letter + " (recommended): " + opts[i].action
		} else {
			opts[i].action = letter + ": " + opts[i].action
		}
	}
	return opts
}

// BuildFacts is the question the operator reads. The visible text carries no
// hashes or internal identifiers; those are confined to TechnicalDetails.
func BuildFacts(d Diagnosis, o Offers, incidentID string, generation int) operatordecision.Facts {
	recommended, reason := Recommended(d, o)
	opts := lettered(d, o, recommended)

	parts := []string{fmt.Sprintf("Backups to %s are blocked: %s.", d.DriveSentence(), d.problemSentence()), d.stoppedSentence()}
	if s := d.unprotectedSentence(); s != "" {
		parts = append(parts, s)
	}
	actions := make([]string, 0, len(opts))
	for _, opt := range opts {
		actions = append(actions, opt.action+".")
	}
	parts = append(parts, "What should I do? "+strings.Join(actions, " "))
	question := strings.Join(parts, " ")

	options := make([]operatordecision.Option, 0, len(opts))
	for _, opt := range opts {
		options = append(options, operatordecision.Option{Key: opt.key, Label: opt.label})
	}
	return operatordecision.Facts{
		Subject: d.subject(), Question: question, Options: options,
		Recommended: recommended, RecommendationReason: reason,
		Reversible: false, Urgency: "high",
		BigPicture:       bigPicture(d, opts),
		TechnicalDetails: technicalDetails(d, o),
		Correlation: map[string]string{
			"incident_id": incidentID, "generation": fmt.Sprint(generation), "destination_id": d.DestinationID,
		},
	}
}

func bigPicture(d Diagnosis, opts []letteredOption) string {
	var b strings.Builder
	fmt.Fprintf(&b, "What is wrong: %s.\n\n", Capitalize(d.problemSentence()))
	b.WriteString(d.stoppedSentence())
	if s := d.unprotectedSentence(); s != "" {
		b.WriteString(" " + s)
	}
	if len(d.Plans.UnprotectedNames) > 0 {
		names := d.Plans.UnprotectedNames
		if len(names) > 4 {
			names = names[:4]
		}
		fmt.Fprintf(&b, " For example: %s.", strings.Join(names, ", "))
	}
	b.WriteString("\n\nYour options:\n")
	for _, opt := range opts {
		b.WriteString("- " + opt.label + "\n")
	}
	b.WriteString("\nIf you decline or do not answer, backups stay blocked and I remind you every day until this is fixed. Nothing is changed on the drive until you choose.")
	return b.String()
}

// Capitalize upper-cases the first letter of a sentence.
func Capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func technicalDetails(d Diagnosis, o Offers) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Cause: %s\nChecked: %s\nDestination: %s (%s)\nRepository directory: %s\n",
		d.Cause, d.CheckedAt.Format(time.RFC3339), d.DestinationName, d.DestinationID, d.RepositoryRoot)
	fmt.Fprintf(&b, "Drive: mounted=%t read_only=%t %s\n", d.Volume.Mounted, d.Volume.ReadOnly, d.Volume.Detail)
	fmt.Fprintf(&b, "Repository opens: %t. Snapshots listed: %t (%d). Latest snapshot: %s. Last success: %s (%s).\n",
		d.RepositoryOpens, d.SnapshotsListed, d.SnapshotCount, fmtTime(d.LatestSnapshotAt), fmtTime(d.LastSuccessAt), d.LastSuccessSource)
	if d.UsageError != "" {
		fmt.Fprintf(&b, "Engine usage error: %s\n", d.UsageError)
	}
	if d.OpenError != "" {
		fmt.Fprintf(&b, "Engine open error: %s\n", d.OpenError)
	}
	if d.SnapshotError != "" {
		fmt.Fprintf(&b, "Engine snapshot error: %s\n", d.SnapshotError)
	}
	if d.ScanRan {
		fmt.Fprintf(&b, "Read-only walk of the repository: %d files, %d bytes, %d readable leftover temporary files, %.0f s.\n",
			d.ScanFiles, d.ScanBytes, d.ReadableTempFiles, d.ScanSeconds)
	}
	if d.ScanError != "" {
		fmt.Fprintf(&b, "Walk error: %s\n", d.ScanError)
	}
	writeEntries(&b, "Unreadable entries not covered by an exception", d.Unreadable)
	writeEntries(&b, "Unreadable entries already covered by a recorded exception", d.Accepted)
	for _, p := range d.Plans.Plans {
		fmt.Fprintf(&b, "Plan %s (%s): enabled=%t schedule=%q targets=%d unprotected=%d\n", p.PlanName, p.PlanID, p.Enabled, p.Schedule, p.TargetCount, p.UnprotectedTargets)
	}
	b.WriteString("\nHow each option is applied (only after a verified human answer):\n")
	if o.Accept {
		b.WriteString("- Skip: records an exception for each exact path above (path, read-failure kind, approver, time). Usage measurement and preflight honour an exception only for that exact path while it stays an unreadable leftover temporary file. Any other unreadable entry, or a changed path, blocks again and asks again. Nothing is deleted, renamed or written in the repository. A backup is then triggered for the plan(s).\n")
	}
	if o.Repair {
		b.WriteString("- Repair: unmount, offline filesystem repair, and read/write remount through the control-plane volume owner, bound to this drive's identity. Refused if a backup is running to this destination. A backup is triggered afterwards.\n")
	}
	if o.Fresh {
		b.WriteString("- Fresh set: nothing is changed automatically; the answer records a request and an agent must register the new destination and re-point the plan.\n")
	}
	b.WriteString("- Do nothing: no change; the question returns after a day.\n")
	text := b.String()
	if len([]rune(text)) > operatordecision.MaxTechnicalDetailsCodePoints {
		text = string([]rune(text)[:operatordecision.MaxTechnicalDetailsCodePoints-40]) + "\n[truncated]"
	}
	return text
}

func writeEntries(b *strings.Builder, title string, entries []Entry) {
	if len(entries) == 0 {
		return
	}
	sorted := append([]Entry(nil), entries...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].RelPath < sorted[j].RelPath })
	fmt.Fprintf(b, "%s (%d):\n", title, len(sorted))
	for i, e := range sorted {
		if i == 60 {
			fmt.Fprintf(b, "  ... and %d more\n", len(sorted)-60)
			break
		}
		fmt.Fprintf(b, "  %s [%s, %s, %s]\n", e.RelPath, e.Class, e.BlobKind, e.Problem)
	}
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	return t.UTC().Format(time.RFC3339)
}
