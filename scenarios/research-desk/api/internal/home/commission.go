package home

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TopicRef names one topic a commission covers and the evidence for it.
type TopicRef struct {
	Topic  string
	Basis  string // packet, card, charter-mission-link, run-folder
	Source string
}

// RunFolder is one topic's run folder: topics/<t>/runs/<name>. It belongs to
// the commission whose id is <name> or a prefix of it followed by "-"; the
// longest such id wins.
type RunFolder struct {
	Topic      string
	Path       string
	Checkpoint *FileFact
	// LastWrite is the newest file anywhere inside, from a bounded walk.
	LastWrite Time
}

// runOwner returns the commission id that owns a run folder name, or "".
func runOwner(name string, ids []string) string {
	owner := ""
	for _, id := range ids {
		if (name == id || strings.HasPrefix(name, id+"-")) && len(id) > len(owner) {
			owner = id
		}
	}
	return owner
}

// Commission is everything the desk derives about one commission folder.
type Commission struct {
	ID     string
	Dir    string
	Files  []FileFact
	Title  Text
	Claim  Text
	Topics []TopicRef
	// TopicsEvidence is unknown when no topic evidence exists.
	TopicsEvidence Evidence
	Runs           []RunFolder
	PacketComplete bool
	// notStartedEv is the never-started evidence for a complete packet with no session.
	notStartedEv Evidence
	Packet       *PacketRecord
	Spec         *PacketSpec
	Launch       *LaunchRecord
	// Operator is OPERATOR.md, nil when absent.
	Operator *OperatorLog
	// Steering says whether the charter tells the scheduler to read OPERATOR.md.
	Steering Flag
	// Decisions says whether the charter leaves delivery of its needs-you
	// items to the desk. An older charter's scheduler delivers them itself.
	Decisions Flag
	// NeedsYou and Legacy are STATE.md's "## Needs you" section.
	NeedsYou []NeedsYouItem
	Legacy   []Legacy
	// RecordErrors lists LAUNCH.json or PACKET.json files that exist but
	// cannot be read; the folder still displays.
	RecordErrors []string
	// Unreadable lists the folder, STATE.md, OPERATOR.md, the charter or the
	// kernel when they exist but cannot be read: what they hold
	// is unknown, and the courier sends nothing for this commission meanwhile.
	Unreadable []Evidence

	Matches      []Match
	Session      *Match // newest exact match
	Allowance    time.Duration
	AllowanceEv  Evidence
	Started      Time
	AllowanceEnd Time
	// Ended is when the commission's interval ended (discarded, goal
	// complete or allowance ended, the earliest the evidence gives); unknown
	// while it has not.
	Ended Time
	// LastWrite is the newest write up to Ended; TouchedAfterEnd the newest
	// after it, never counted as activity.
	LastWrite       Time
	TouchedAfterEnd Time
	State           State
	StateRule       string
	StateEvidence   []Evidence
	Stalled         bool
	// NeverStarted says, for an unknown commission, whether the operator may
	// assert it never started; zero for every other state.
	NeverStarted NeverStarted
}

// NeverStarted is whether an unknown commission may be launched on the
// operator's word that it never started: only a complete packet with no
// launch record, no exact session and no goal naming its charter in any
// Codex home the desk can read.
type NeverStarted struct {
	Assertable bool
	Reason     string
}

// LaunchedBy returns "desk" or "hand" from LAUNCH.json, or "".
func (c *Commission) LaunchedBy() string {
	if c.Launch == nil {
		return ""
	}
	return c.Launch.LaunchedBy
}

// File returns a top-level file of the folder by name.
func (c *Commission) File(name string) (FileFact, bool) {
	for _, f := range c.Files {
		if f.Name == name {
			return f, true
		}
	}
	return FileFact{}, false
}

var (
	cardHeading   = regexp.MustCompile(`(?m)^### Card `)
	cardTopicLink = regexp.MustCompile(`topics/(` + NamePattern + `)/`)
	// charterTopicLink is a charter link that names a topic: to its
	// MISSION.md, to an older layout's README.md, or to the folder itself.
	// A deeper path (a run folder, a background file) does not.
	charterTopicLink = regexp.MustCompile("topics/(" + NamePattern + ")/(?:(MISSION\\.md|README\\.md)|[\\s)`\"'>\\]#,;:.]|$)")
	kernelHours      = regexp.MustCompile(`(\d+)\s*h(?:ours?)?\s+allowance`)
)

// readCommission reads the files of one commission folder. Session-derived
// fields are filled in later by derive.
func readCommission(researchDir, id string, runs []RunFolder, topicOf func(string) (string, bool)) *Commission {
	dir := filepath.Join(researchDir, "cohorts", id)
	c := &Commission{ID: id, Dir: dir, Runs: runs}
	var err error
	if c.Files, err = topFiles(dir); err != nil {
		c.Unreadable = append(c.Unreadable, Unreadable(dir, err))
	}

	if title, ok := firstHeading(filepath.Join(dir, CharterFile)); ok {
		c.Title = title
	} else {
		c.Title = Text{Evidence: Unknown("no " + CharterFile + " with a title")}
	}
	c.Claim = readClaim(dir)

	_, hasCharter := c.File(CharterFile)
	_, hasStance := c.File(StanceFile)
	_, hasKernel := c.File(KernelFile)
	c.PacketComplete = hasCharter && hasStance && hasKernel

	launchPath := filepath.Join(dir, LaunchFile)
	if record, err := ReadLaunch(launchPath); err != nil {
		c.RecordErrors = append(c.RecordErrors, err.Error())
	} else {
		c.Launch = record
	}
	packetPath := filepath.Join(dir, PacketFile)
	if record, spec, err := ReadPacket(packetPath); err != nil {
		c.RecordErrors = append(c.RecordErrors, err.Error())
	} else {
		c.Packet, c.Spec = record, spec
	}

	if c.Operator, err = readOperatorFile(filepath.Join(dir, OperatorFile)); err != nil {
		c.Unreadable = append(c.Unreadable, Unreadable(filepath.Join(dir, OperatorFile), err))
	}
	c.Steering, c.Decisions = readCharterFlags(filepath.Join(dir, CharterFile))
	if c.NeedsYou, c.Legacy, err = readNeedsYou(filepath.Join(dir, StateFile), StateNeedsSection); err != nil {
		c.Unreadable = append(c.Unreadable, Unreadable(filepath.Join(dir, StateFile), err))
	}

	c.Topics, c.TopicsEvidence = commissionTopics(c, topicOf)
	c.Allowance, c.AllowanceEv = commissionAllowance(c)
	return c
}

// readClaim is the supervisor's own status line: STATE.md, else STATUS.md.
func readClaim(dir string) Text {
	for _, name := range []string{StateFile, "STATUS.md"} {
		if line, ok := firstParagraphLine(filepath.Join(dir, name)); ok {
			return line
		}
	}
	return Text{Evidence: Unknown("no STATE.md or STATUS.md")}
}

// commissionTopics derives topics from, in order of strength, the packet
// spec, the first topic link in each card, any topic MISSION.md the charter
// names, and the topics' run folders. The union is returned; each topic keeps
// its strongest basis. Looser mentions are not evidence.
func commissionTopics(c *Commission, topicOf func(string) (string, bool)) ([]TopicRef, Evidence) {
	var refs []TopicRef
	seen := map[string]bool{}
	add := func(topic, basis, source string) {
		if seen[topic] || !FolderID.MatchString(topic) {
			return
		}
		seen[topic] = true
		refs = append(refs, TopicRef{Topic: topic, Basis: basis, Source: source})
	}
	if c.Spec != nil {
		for _, topic := range c.Spec.Topics {
			add(topic, "packet", filepath.Join(c.Dir, PacketFile))
		}
	}
	charterPath := filepath.Join(c.Dir, CharterFile)
	data, err := os.ReadFile(charterPath)
	if err != nil && !absent(err) {
		c.Unreadable = append(c.Unreadable, Unreadable(charterPath, err))
	}
	if err == nil {
		locs := cardHeading.FindAllIndex(data, -1)
		for i, loc := range locs {
			end := len(data)
			if i+1 < len(locs) {
				end = locs[i+1][0]
			} else if next := strings.Index(string(data[loc[1]:]), "\n## "); next >= 0 {
				end = loc[1] + next
			}
			if sub := cardTopicLink.FindSubmatch(data[loc[0]:end]); sub != nil {
				if slug, ok := topicOf(string(sub[1])); ok {
					add(slug, "card", charterPath)
				}
			}
		}
		for _, sub := range charterTopicLink.FindAllSubmatch(data, -1) {
			slug, ok := topicOf(string(sub[1]))
			if !ok {
				continue
			}
			if string(sub[2]) == "MISSION.md" {
				add(slug, "charter-mission-link", charterPath)
			} else {
				add(slug, "charter-topic-link", charterPath)
			}
		}
	}
	for _, run := range c.Runs {
		add(run.Topic, "run-folder", run.Path)
	}
	if len(refs) == 0 && err != nil && !absent(err) {
		return nil, Unreadable(charterPath, err)
	}
	if len(refs) == 0 {
		return nil, Unknown("no PACKET.json topics, no topic link in a card, no topic MISSION.md, README.md or folder link in the charter, and no run folders")
	}
	return refs, Evidence{Known: true, Source: refs[0].Source}
}

// commissionAllowance reads the allowance from the packet spec, else from
// the kernel's "<N>h allowance".
func commissionAllowance(c *Commission) (time.Duration, Evidence) {
	if c.Spec != nil && c.Spec.AllowanceHours > 0 {
		fact, _ := c.File(PacketFile)
		return time.Duration(c.Spec.AllowanceHours) * time.Hour, From(fact.Path, fact.Modified)
	}
	if fact, ok := c.File(KernelFile); ok {
		data, err := os.ReadFile(fact.Path)
		if err != nil {
			return 0, Unreadable(fact.Path, err)
		}
		if sub := kernelHours.FindSubmatch(data); sub != nil {
			if hours, err := strconv.Atoi(string(sub[1])); err == nil && hours > 0 {
				return time.Duration(hours) * time.Hour, From(fact.Path, fact.Modified)
			}
		}
		return 0, Unknown(KernelFile + " names no \"<N>h allowance\"")
	}
	return 0, Unknown("no PACKET.json allowance and no " + KernelFile)
}

// derive fills the session-derived fields: matches, start, allowance end,
// last write and state.
func (c *Commission) derive(env deriveEnv) {
	c.Matches = recognize(c.ID, charterForms(env.researchDir, env.userHome, c.ID), c.Launch, filepath.Join(c.Dir, LaunchFile), env.records, windowStart(c.Files))
	hasRelated := false
	for i := range c.Matches {
		m := &c.Matches[i]
		m.WindowOpen = env.windows[filepath.Join(env.codexDir, m.SessionID)]
		if m.Kind == MatchExact && c.Session == nil {
			c.Session = m
		}
		if m.Kind == MatchRelated {
			hasRelated = true
		}
	}

	c.Started = Time{Evidence: Unknown("no exact session")}
	c.AllowanceEnd = Time{Evidence: Unknown("no exact session")}
	if c.Session != nil {
		c.Started = Time{Value: c.Session.At, Evidence: From(c.Session.Source, c.Session.At)}
		if c.AllowanceEv.Known {
			c.AllowanceEnd = Time{Value: c.Session.At.Add(c.Allowance), Evidence: Evidence{Known: true, Source: c.AllowanceEv.Source, AsOf: c.AllowanceEv.AsOf, Reason: "session start plus allowance"}}
		} else {
			c.AllowanceEnd = Time{Evidence: c.AllowanceEv}
		}
	}

	in := stateInput{
		Now:            env.now,
		Discarded:      c.Launch != nil && c.Launch.Discarded != nil,
		PacketComplete: c.PacketComplete,
		HasLaunch:      c.Launch != nil,
		HasRelated:     hasRelated,
		Exact:          c.Session,
		ProcsKnown:     env.procsKnown,
		ProcsReason:    env.procsReason,
		AllowanceEnd:   c.AllowanceEnd,
	}
	for _, e := range c.Unreadable {
		if e.Source == c.Dir {
			in.Unlisted = e.Reason
		}
	}
	if c.Session != nil {
		in.WindowOpen = c.Session.WindowOpen
	}
	if c.Session == nil && c.PacketComplete && !hasRelated && c.Launch == nil {
		in.NotStarted = c.notStarted(env)
	}
	c.notStartedEv = in.NotStarted
	c.State, c.StateRule = deriveState(in)
	if c.State == StateUnknown {
		c.NeverStarted = c.neverStarted(env)
	}
	c.StateEvidence = c.stateEvidence(env)
	c.Ended = c.end(env.now)
	c.LastWrite, c.TouchedAfterEnd = c.writes(env)
	c.Stalled = c.State == StateLive && c.LastWrite.Known && env.now.Sub(c.LastWrite.Value) > StallAfter
}

// notStarted is the evidence that nothing started a complete packet: no goal
// in any readable Codex home names its charter, and neither STATE.md nor
// LEDGER.md was written after the packet was built. Known only
// when every home was read.
func (c *Commission) notStarted(env deriveEnv) Evidence {
	switch {
	case !env.records.Known:
		return Unknown(env.records.Reason)
	case len(env.records.Unreadable) > 0:
		return Unknown(fmt.Sprintf("%d Web Console session files unreadable, first %s", len(env.records.Unreadable), env.records.Unreadable[0]))
	}
	other := env.otherGoals()
	if len(other.Unreadable) > 0 {
		return Unknown(fmt.Sprintf("%d other Codex homes unreadable, first %s", len(other.Unreadable), other.Unreadable[0]))
	}
	if found := c.goalNaming(env, other); found != "" {
		return Unknown(found)
	}
	built := c.Built()
	if !built.Known {
		return Unknown(built.Reason)
	}
	for _, name := range []string{StateFile, LedgerFile} {
		if fact, ok := c.File(name); ok && fact.Modified.After(built.Value) {
			return Unknown(fmt.Sprintf("%s was written at %s, after the packet was built at %s", name, stamp(fact.Modified), stamp(built.Value)))
		}
	}
	return Evidence{
		Known: true, Source: fmt.Sprintf("%s and %d other Codex homes", env.codexDir, other.Homes), AsOf: env.now,
		Reason: "no goal names this charter; STATE.md and LEDGER.md unchanged since the build at " + stamp(built.Value),
	}
}

// goalNaming says which goal in another Codex home names this commission's
// charter, or "".
func (c *Commission) goalNaming(env deriveEnv, other OtherGoals) string {
	forms := charterForms(env.researchDir, env.userHome, c.ID)
	for _, goal := range other.Goals {
		for _, form := range forms {
			if strings.Contains(goal.Objective, form) {
				return fmt.Sprintf("a goal in %s (thread %s, %s) names this charter", goal.Source, goal.ThreadID, orUnknown(goal.Status))
			}
		}
	}
	return ""
}

// neverStarted says whether the operator may assert that this unknown
// commission never started, and why. A session or goal found
// anywhere the desk can read refuses it; homes it cannot read are named.
func (c *Commission) neverStarted(env deriveEnv) NeverStarted {
	refuse := func(why string) NeverStarted { return NeverStarted{Reason: why} }
	switch {
	case !c.PacketComplete:
		return refuse("the packet is not complete: it needs " + CharterFile + ", " + StanceFile + " and " + KernelFile)
	case c.Launch != nil:
		return refuse(LaunchFile + " records a launch attempt; relaunch it as a new commission")
	case c.Session != nil:
		return refuse(fmt.Sprintf("session %s (goal %s) names this charter", c.Session.SessionID, orUnknown(c.Session.GoalStatus)))
	case len(c.Matches) > 0:
		m := c.Matches[0]
		return refuse(fmt.Sprintf("session %s mentions this commission (%s, %s)", m.SessionID, m.Basis, m.Source))
	case !env.records.Known:
		return refuse("Web Console's Codex sessions could not be read: " + env.records.Reason)
	}
	other := env.otherGoals()
	if found := c.goalNaming(env, other); found != "" {
		return refuse(found)
	}
	reason := fmt.Sprintf("no session or goal names this charter in %s or %d other Codex homes", env.codexDir, other.Homes)
	var unread []string
	if n := len(env.records.Unreadable); n > 0 {
		unread = append(unread, fmt.Sprintf("%d Web Console session files (first %s)", n, env.records.Unreadable[0]))
	}
	if n := len(other.Unreadable); n > 0 {
		unread = append(unread, fmt.Sprintf("%d other Codex homes (first %s)", n, other.Unreadable[0]))
	}
	if len(unread) > 0 {
		reason += "; not read: " + strings.Join(unread, ", ")
	}
	return NeverStarted{Assertable: true, Reason: reason}
}

// Built is when packet.py built the packet: PACKET.json's built_at, else
// KERNEL.txt's time (an older packet; build writes it once).
func (c *Commission) Built() Time {
	if c.Packet != nil {
		if at, err := time.Parse(time.RFC3339, c.Packet.BuiltAt); err == nil {
			fact, _ := c.File(PacketFile)
			return Time{Value: at, Evidence: Evidence{Known: true, Source: fact.Path, AsOf: fact.Modified, Reason: "built_at"}}
		}
	}
	if fact, ok := c.File(KernelFile); ok {
		return Time{Value: fact.Modified, Evidence: From(fact.Path, fact.Modified)}
	}
	return Time{Evidence: Unknown("the packet's build time is unknown: no PACKET.json built_at and no " + KernelFile)}
}

func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// stateEvidence names the inputs the state rules read.
func (c *Commission) stateEvidence(env deriveEnv) []Evidence {
	var out []Evidence
	if c.Launch != nil {
		if fact, ok := c.File(LaunchFile); ok {
			out = append(out, From(fact.Path, fact.Modified))
		}
	}
	if c.Session != nil {
		out = append(out, Evidence{Known: true, Source: c.Session.Source, AsOf: c.Session.At, Reason: "goal status " + orUnknown(c.Session.GoalStatus)})
		if env.procsKnown {
			reason := "no Codex window for session " + c.Session.SessionID
			if c.Session.WindowOpen {
				reason = "Codex window open for session " + c.Session.SessionID
			}
			out = append(out, Evidence{Known: true, Source: "/proc", AsOf: env.now, Reason: reason})
		} else {
			out = append(out, Unknown(env.procsReason))
		}
	} else if !env.records.Known {
		out = append(out, Unknown(env.records.Reason))
	}
	if ev := c.notStartedEv; ev.Known || ev.Reason != "" {
		out = append(out, ev)
	}
	if c.AllowanceEnd.Known {
		out = append(out, c.AllowanceEnd.Evidence)
	}
	return out
}

// end is the earliest end the evidence gives: discarded, goal complete, or
// allowance ended. A commission with a supervisor window open has not ended
// whatever its allowance says.
func (c *Commission) end(now time.Time) Time {
	end := Time{Evidence: Unknown("not ended: not discarded, goal not complete, allowance not ended")}
	if c.State.Running() {
		return Time{Evidence: Unknown("not ended: supervisor window open (" + string(c.State) + ")")}
	}
	earlier := func(t Time) {
		if t.Known && (!end.Known || t.Value.Before(end.Value)) {
			end = t
		}
	}
	if c.Launch != nil && c.Launch.Discarded != nil {
		fact, _ := c.File(LaunchFile)
		earlier(Time{Value: c.Launch.Discarded.At, Evidence: Evidence{Known: true, Source: fact.Path, AsOf: fact.Modified, Reason: "discarded"}})
	}
	if c.Session != nil && !c.Session.Ended.IsZero() {
		earlier(Time{Value: c.Session.Ended, Evidence: Evidence{Known: true, Source: c.Session.Source, AsOf: c.Session.Ended, Reason: "goal marked complete"}})
	}
	if c.AllowanceEnd.Known && c.AllowanceEnd.Value.Before(now) {
		t := c.AllowanceEnd
		t.Reason = "allowance ended"
		earlier(t)
	}
	return end
}

// writes splits the commission's writes (its folder's files, its run
// folders and its exact session's transcript) at the interval end: the
// newest up to the end is activity, the newest after it is "touched after
// end".
func (c *Commission) writes(env deriveEnv) (Time, Time) {
	var cutoff time.Time
	if c.Ended.Known {
		cutoff = c.Ended.Value
	}
	last := Time{Evidence: Unknown("no files")}
	after := Time{Evidence: Unknown("nothing written after the end")}
	add := func(t Time) {
		if !cutoff.IsZero() && t.Value.After(cutoff) {
			after = later(after, t)
		} else {
			last = later(last, t)
		}
	}
	for _, f := range c.Files {
		add(Time{Value: f.Modified, Evidence: From(f.Path, f.Modified)})
	}
	for i := range c.Runs {
		run := &c.Runs[i]
		found := env.walk(run.Path, cutoff)
		run.LastWrite = found.All
		last = later(last, found.Before)
		after = later(after, found.After)
	}
	if c.Session != nil {
		if rollout, ok := rolloutFor(env.codexDir, c.Session.SessionID, c.Session.ThreadID); ok {
			add(Time{Value: rollout.Modified, Evidence: From(rollout.Path, rollout.Modified)})
		}
	}
	return last, after
}

// deriveEnv is what derive reads besides the folder itself.
type deriveEnv struct {
	now         time.Time
	researchDir string
	codexDir    string
	userHome    string
	records     CodexRecords
	windows     map[string]bool
	procsKnown  bool
	procsReason string
	// walk finds a folder's newest file, split at a cutoff (walk.go).
	walk func(dir string, cutoff time.Time) newestWrite
	// others reads the goals of the Codex homes outside Web Console's
	// session folder, once per snapshot and only when needed.
	others func() OtherGoals
}

// otherGoals reads the other Codex homes' goals, or none without a reader.
func (env deriveEnv) otherGoals() OtherGoals {
	if env.others == nil {
		return OtherGoals{}
	}
	return env.others()
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

// sortCommissions orders commissions by last write, newest first; unknown last.
func sortCommissions(list []*Commission) {
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i].LastWrite, list[j].LastWrite
		if a.Known != b.Known {
			return a.Known
		}
		if !a.Value.Equal(b.Value) {
			return a.Value.After(b.Value)
		}
		return list[i].ID < list[j].ID
	})
}
