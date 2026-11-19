package repohealth

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/operatordecision"
)

// Option keys the Notification Hub returns when the operator answers.
const (
	OptAccept          = "accept_exception"
	OptKopiaQuarantine = "quarantine_repository_temp"
	OptRepair          = "repair_drive"
	OptFresh           = "fresh_repository"
	OptLeave           = "leave_blocked"
)

// Offers is which actions the code can truly perform for a diagnosis. An action
// that cannot run is never offered.
type Offers struct {
	Accept, KopiaQuarantine, Repair, Fresh bool
}

// OffersFor decides what to offer. repairPlannable is whether the control plane
// could build an identity-bound unmount/repair/remount plan for this drive.
//
// A repair that the platform's tool cannot perform is never offered. On Linux,
// NTFS repair is a dirty-flag reset: it can re-enable a read-only mount, but it
// cannot repair a metadata record. Offering it for unreadable metadata promised
// a fix that never happened and repeatedly told the operator the drive had been
// repaired.
func OffersFor(d Diagnosis, repairPlannable bool) Offers {
	repair := repairPlannable && d.repairCanResolve()
	switch d.Cause {
	case CauseLeftoverTempUnreadable:
		canQuarantine := d.DestinationDevice != nil && d.DestinationDevice.StrongIdentity() && quarantinableTemps(d.Unreadable)
		return Offers{Accept: true, KopiaQuarantine: canQuarantine, Repair: repair, Fresh: true}
	case CauseContentUnreadable, CauseSnapshotsUnlistable, CauseRepositoryUnopenable, CauseUsageUnexplained:
		return Offers{Repair: repair, Fresh: true}
	case CauseVolumeReadOnly:
		return Offers{Repair: repairPlannable}
	default:
		return Offers{}
	}
}

// quarantinableTemps permits one verified decision to cover a complete set of
// exact orphan session temps. Every entry must be an EIO unreadable, orphaned
// s/...f.tmp object; a mixed or ambiguous set remains blocked for inspection.
func quarantinableTemps(entries []Entry) bool {
	if len(entries) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		clean := path.Clean(strings.ReplaceAll(entry.RelPath, `\\`, "/"))
		class, kind := ClassifyName(entry.RelPath)
		if clean != entry.RelPath || !strings.HasPrefix(clean, "s/") || class != ClassLeftoverTemp || kind != BlobSessionMarker || entry.Class != ClassLeftoverTemp || entry.BlobKind != BlobSessionMarker || entry.Problem != ProblemIO || !entry.Orphan {
			return false
		}
		if _, ok := seen[clean]; ok {
			return false
		}
		seen[clean] = struct{}{}
	}
	return true
}

// Askable reports whether a diagnosis leads to a question. A cleared failure and
// an unmounted volume (owned by the existing drive recovery) do not.
func (o Offers) Askable() bool { return o.Accept || o.KopiaQuarantine || o.Repair || o.Fresh }

// repairCanResolve reports whether this platform's repair tool can plausibly
// fix the diagnosed cause. Clearing a dirty flag re-enables a read-only mount,
// but it cannot repair an unreadable metadata record or unreadable saved data —
// so recommending a repair for that case would promise what the tool cannot do.
func (d Diagnosis) repairCanResolve() bool {
	if d.RepairLimitation == "" {
		return true
	}
	return d.Cause == CauseVolumeReadOnly
}

// Recommended is the option to suggest, with the plain reason.
func Recommended(d Diagnosis, o Offers) (key, reason string) {
	switch {
	case o.KopiaQuarantine:
		return OptKopiaQuarantine, fmt.Sprintf("Kopia can move these %d exact unreadable session %s into same-volume quarantine while preserving their bytes; it will not recreate any final blob.", len(d.Unreadable), operatordecision.Plural(len(d.Unreadable), "temp", "temps"))
	case o.Accept:
		return OptAccept, fmt.Sprintf("They are unfinished uploads, not saved backup data, and your %d earlier backups still list normally. This restarts protection soonest and changes nothing on the drive.", d.SnapshotCount)
	case o.Fresh && !d.repairCanResolve():
		// The only repair the platform offers cannot fix this. Point at the
		// durable fix instead of repeating a step already known to fail.
		reason := "A repair cannot fix this: " + d.RepairLimitation + "."
		if !d.PriorRepairAt.IsZero() {
			reason = "The repair on " + d.PriorRepairAt.Format("2 Jan") + " could not fix this, because " + d.RepairLimitation + "."
		}
		return OptFresh, reason + " A fresh backup set leaves the existing backups read-only and untouched."
	case o.Repair:
		return OptRepair, "It is the only fix I can run myself. Unreadable saved data can make a backup impossible to restore, so it should not be ignored."
	default:
		return OptLeave, "Nothing I can safely run applies here; the drive needs a person or an agent to look at it."
	}
}

type offeredOption struct {
	key, label, effect string
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
		return fmt.Sprintf("%d unfinished temporary %s on the drive can't be read", n, operatordecision.Plural(n, "file", "files"))
	case CauseContentUnreadable:
		return fmt.Sprintf("%d %s inside the backup set can't be read", n, operatordecision.Plural(n, "file", "files"))
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
		return fmt.Sprintf("Your last good backup was %s (%d %s ago).", when, days, operatordecision.Plural(days, "day", "days"))
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

// offered lists the offered options in a fixed order, each with what choosing
// it does.
func offered(d Diagnosis, o Offers) []offeredOption {
	var opts []offeredOption
	n := len(d.Unreadable)
	if o.Accept {
		opts = append(opts, offeredOption{
			key:    OptAccept,
			label:  fmt.Sprintf("Skip only %s and resume backups. Low risk; I keep warning you while %s.", operatordecision.Plural(n, "that unreadable file", fmt.Sprintf("those %d unreadable files", n)), operatordecision.Plural(n, "it exists", "they exist")),
			effect: "I record an exception for each exact file listed above and start a backup. Nothing on the drive is deleted, renamed or written. Any other unreadable file, or a change to these, stops backups and asks again.",
		})
	}
	if o.KopiaQuarantine {
		opts = append(opts, offeredOption{
			key:    OptKopiaQuarantine,
			label:  fmt.Sprintf("Quarantine only these %d exact unreadable session %s. Kopia preserves their bytes in same-volume quarantine and writes no final blob; backups retry after revalidation.", n, operatordecision.Plural(n, "temp", "temps")),
			effect: "The backup engine moves exactly these files into a quarantine folder on the same drive, keeping their bytes, and backups retry after the drive is checked again.",
		})
	}
	if o.Repair {
		label := "Take the drive offline and repair it. Backups pause meanwhile; a repair can discard damaged data."
		if d.RepairLimitation != "" {
			// Only reachable for a read-only mount, which is the one condition
			// a dirty-flag reset genuinely fixes.
			label = "Take the drive offline and clear the filesystem dirty state. Backups pause meanwhile. This is the one condition this filesystem's Linux tool can fix; it will not repair damaged metadata."
		} else if !d.PriorRepairAt.IsZero() && (d.Cause == CauseLeftoverTempUnreadable || d.Cause == CauseContentUnreadable) {
			label = "Take the drive offline and repair it. Backups pause meanwhile; a repair can discard damaged data" +
				fmt.Sprintf(", and the repair on %s did not clear this.", d.PriorRepairAt.Format("2 Jan"))
		}
		opts = append(opts, offeredOption{
			key: OptRepair, label: label,
			effect: "The drive is unmounted, its filesystem repaired, and mounted again, only if it is still the same drive and no backup is running to it. A backup starts afterwards.",
		})
	}
	if o.Fresh {
		label := "Start a fresh backup set and keep the old one read-only. Uses more space; an agent has to set it up, so this only asks for that."
		effect := "Nothing changes yet. The answer records the request, and I then ask you where the new backup set should go."
		if d.RepairRemedy != "" {
			label = "Start a fresh backup set and keep the old one read-only — the durable fix here."
			effect += " To keep using this drive instead: " + d.RepairRemedy + "."
		}
		opts = append(opts, offeredOption{key: OptFresh, label: label, effect: effect})
	}
	return append(opts, offeredOption{
		key: OptLeave, label: "Do nothing for now. Backups stay stopped and I ask again tomorrow.",
		effect: "Nothing changes. Backups stay stopped and this question comes back tomorrow.",
	})
}

// BuildFacts is the question the operator reads. The visible text carries no
// hashes or internal identifiers; those are confined to TechnicalDetails.
func BuildFacts(d Diagnosis, o Offers, incidentID string, generation int) operatordecision.Facts {
	recommended, reason := Recommended(d, o)
	opts := offered(d, o)
	parts := []string{fmt.Sprintf("Backups to %s have stopped: %s.", d.DriveSentence(), d.problemSentence()), d.lastGoodSentence()}
	parts = append(parts, "What should I do?")
	options := make([]operatordecision.Option, 0, len(opts))
	effects := make(map[string]string, len(opts))
	for _, opt := range opts {
		options = append(options, operatordecision.Option{Key: opt.key, Label: opt.label})
		effects[opt.key] = opt.effect
	}
	facts := operatordecision.Facts{
		Subject: d.subject(), Question: strings.Join(parts, " "), Options: options,
		Recommended: recommended, RecommendationReason: reason,
		Reversible: false, Urgency: "high",
		BigPicture:       bigPicture(d),
		TechnicalDetails: technicalDetails(d, o),
		Correlation: map[string]string{
			"incident_id": incidentID, "generation": fmt.Sprint(generation), "destination_id": d.DestinationID,
		},
	}
	facts.Explainer = page(d, facts, effects)
	return facts
}

// lastGoodSentence is the short form for the question: when the last backup
// worked and how much has no other copy.
func (d Diagnosis) lastGoodSentence() string {
	s := "No backup has ever succeeded."
	if !d.LastSuccessAt.IsZero() {
		if days := d.DaysStopped(); days <= 0 {
			s = "Last good backup today."
		} else {
			s = fmt.Sprintf("Last good backup %s (%d %s ago).", d.LastSuccessAt.Format("2 Jan"), days, operatordecision.Plural(days, "day", "days"))
		}
	}
	if d.Plans.TargetTotal > 0 && d.Plans.Unprotected > 0 {
		s = strings.TrimSuffix(s, ".") + fmt.Sprintf("; %d of the %d items have no other copy.", d.Plans.Unprotected, d.Plans.TargetTotal)
	}
	return s
}

func bigPicture(d Diagnosis) string {
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
	b.WriteString("\n\nOnly you can choose, because each fix trades risk to the drive against how soon backups resume. If you decline or do not answer, backups stay blocked and I remind you every day until this is fixed. Nothing is changed on the drive until you choose.")
	return b.String()
}

// entryWords names an unreadable file's kind in everyday words.
func entryWords(e Entry) string {
	switch {
	case e.Class == ClassLeftoverTemp:
		return "Unfinished upload (not saved backup data)"
	case e.BlobKind == BlobDataPack:
		return "Saved backup data"
	case e.BlobKind == BlobMetadataPack, e.BlobKind == BlobIndex:
		return "The backup set's index"
	}
	return "Other file in the backup set"
}

func problemWords(p Problem) string {
	switch p {
	case ProblemIO:
		return "The drive returns a read error"
	case ProblemPermission:
		return "Access is denied"
	}
	return "Cannot be read"
}

// page shows the stopped backups, what has no other copy, the files that
// cannot be read, and what each answer does.
func page(d Diagnosis, facts operatordecision.Facts, effects map[string]string) string {
	p := operatordecision.NewPage(d.subject()).
		Lead(fmt.Sprintf("%s. %s", Capitalize(d.problemSentence()), d.stoppedSentence()))
	rows := []operatordecision.Row{{Label: "Drive", Value: d.DriveSentence()}}
	switch {
	case !d.Volume.Mounted:
		rows = append(rows, operatordecision.Row{Label: "Drive state", Value: "Not mounted"})
	case d.Volume.ReadOnly:
		rows = append(rows, operatordecision.Row{Label: "Drive state", Value: "Mounted read-only"})
	default:
		rows = append(rows, operatordecision.Row{Label: "Drive state", Value: "Mounted and writable"})
	}
	if d.SnapshotsListed {
		rows = append(rows, operatordecision.Row{Label: "Saved backups", Value: fmt.Sprintf("%d still listed", d.SnapshotCount)})
	}
	if d.Plans.TargetTotal > 0 {
		rows = append(rows, operatordecision.Row{Label: "Without another copy", Value: fmt.Sprintf("%d of %d items", d.Plans.Unprotected, d.Plans.TargetTotal)})
	}
	rows = append(rows, operatordecision.Row{Label: "Checked", Value: d.CheckedAt.Format("2 Jan 15:04 MST")})
	p.Facts(rows...)
	tone := "warning"
	if d.Plans.Unprotected > 0 && d.DaysStopped() >= 2 {
		tone = "danger"
	}
	p.Callout(tone, "At risk:", d.unprotectedSentence()+" A disk failure now would lose whatever changed since the last good backup.")
	if names := d.Plans.UnprotectedNames; len(names) > 0 {
		list := make([][]string, 0, len(names))
		for _, name := range names {
			list = append(list, []string{name})
		}
		p.Section("Items with no other backup copy", "").Table([]string{"Item"}, list)
	}
	if len(d.Unreadable) > 0 {
		sorted := append([]Entry(nil), d.Unreadable...)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].RelPath < sorted[j].RelPath })
		kinds := map[string]int{}
		order := []string{}
		for _, e := range sorted {
			key := entryWords(e) + "\x00" + problemWords(e.Problem)
			if kinds[key] == 0 {
				order = append(order, key)
			}
			kinds[key]++
		}
		table := make([][]string, 0, len(order))
		for _, key := range order {
			kind, problem, _ := strings.Cut(key, "\x00")
			table = append(table, []string{kind, problem, fmt.Sprint(kinds[key])})
		}
		p.Section("Files the backup engine cannot read", "Their exact names are in the technical details.").
			Table([]string{"What it is", "Problem", "Files"}, table)
	}
	p.Options(facts, effects).
		IfNoAnswer("Backups stay blocked and I remind you every day. Nothing on the drive changes.")
	return p.HTML()
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
	if d.DestinationDevice != nil {
		fmt.Fprintf(&b, "Observed device identity: filesystem=%s UUID=%s serial=%s\n", d.DestinationDevice.Filesystem, d.DestinationDevice.UUID, d.DestinationDevice.Serial)
	}
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
	if d.RepairLimitation != "" {
		fmt.Fprintf(&b, "- Repair ceiling on this filesystem: %s.\n", d.RepairLimitation)
		if d.RepairRemedy != "" {
			fmt.Fprintf(&b, "  For a real repair: %s.\n", d.RepairRemedy)
		}
		if !d.PriorRepairAt.IsZero() {
			fmt.Fprintf(&b, "  A repair already completed on %s and did not clear this condition.\n", d.PriorRepairAt.Format("2 Jan 2006"))
		}
	}
	if o.Repair {
		b.WriteString("- Repair: unmount, offline filesystem repair, and read/write remount through the control-plane volume owner, bound to this drive's identity. Refused if a backup is running to this destination. A backup is triggered afterwards.\n")
	}
	if o.Fresh {
		b.WriteString("- Fresh set: nothing is changed automatically; the answer records a request and an agent must register the new destination and re-point the plan.\n")
	}
	b.WriteString("- Do nothing: no change; the question returns after a day.\n")
	return operatordecision.Clip(b.String(), operatordecision.MaxTechnicalDetailsCodePoints)
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
