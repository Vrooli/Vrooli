package home

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Topic is everything the desk derives about one topic folder.
type Topic struct {
	Slug string
	// Aliases are other names under topics/ that link to this topic's
	// folder (fraud-ppp → public-data-fraud); a link through one names this
	// topic.
	Aliases []string
	Dir     string
	Title   Text
	Mission *FileFact
	// LastResearchWrite is the newest write in the topic's run folders;
	// mission edits (Mission) and reviews (LastReview) are separate.
	// Topics sort by it.
	LastResearchWrite ResearchWrite
	// LastReview is the newest review with a section for this topic.
	LastReview    *ReviewTouch
	CommissionIDs []string // newest last write first
	// Live, Paused and Limited are the commissions with a supervisor window
	// open, by state; never summed into one "running".
	Live, Paused, Limited []string
	Check                 MissionCheck
	// Unreadable lists the runs folder when it exists but cannot be read
	//: the topic's run folders, and so their commissions' runs,
	// are then unknown.
	Unreadable []Evidence
}

// MissionCheck is one topic's summary from mission_check.py --json, the
// skill's MISSION.md parser. The desk never parses MISSION.md itself.
type MissionCheck struct {
	Mode            string
	Open            int
	Active          int
	WaitingOperator int
	Landed          int
	Errors          []string
	Warnings        []string
	Workstreams     []Workstream
	Evidence
}

// Workstream is one row of mission_check.py's workstream_rows.
type Workstream struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Showcase string `json:"showcase"`
	// Title is the row's destination: what the workstream lands.
	Title     string `json:"destination"`
	Magnitude string `json:"magnitude"`
}

// ResearchWrite is the newest write in a topic's run folders and the
// commission owning that folder.
type ResearchWrite struct {
	Topic string
	Time
	// CommissionID owns the run folder; empty for an unowned folder.
	CommissionID string
	State        State
	// Note says when the owner is paused or ended: a write then is a fact
	// the operator needs.
	Note string
}

// ReviewTouch is the newest review file naming a topic.
type ReviewTouch struct {
	ReviewID string
	File     FileFact
}

// writeNote is what a run-folder write at `at` under a commission means, or
// "". A write before an ended commission's end is ordinary.
func writeNote(c *Commission, at time.Time) string {
	switch c.State {
	case StatePaused:
		// The goal row changes when the goal is paused; a write before that
		// was made while it ran.
		if c.Session != nil && !c.Session.GoalUpdated.IsZero() && !at.After(c.Session.GoalUpdated) {
			return ""
		}
		return "written while paused: an owner may still be running"
	case StateLimited:
		return "written while limited: an owner may still be running"
	case StateEnded:
		if c.Ended.Known && !at.After(c.Ended.Value) {
			return ""
		}
		return "written after the commission ended: an owner may still be running"
	case StateStopped:
		return "written while the supervisor window is closed: an owner may still be running"
	}
	return ""
}

// MissionChecker runs the skill's mission_check.py over the research home
// and returns its JSON output.
type MissionChecker interface {
	MissionCheck(ctx context.Context, researchDir string) (output []byte, source string, err error)
}

// Topics is the topic list and how a folder name under topics/ resolves.
type topicIndex struct {
	topics []*Topic
	// names maps every listed folder name, aliases included, to its topic.
	names map[string]string
}

// of resolves a folder name under topics/ to its topic's slug.
func (x topicIndex) of(name string) (string, bool) {
	slug, ok := x.names[name]
	return slug, ok
}

// readTopics lists every folder under topics/ whose name follows NamePattern,
// and their run folders keyed by folder name. A symbolic link
// to another listed topic folder is that topic's alias, not a second topic.
// Run folders are walked later, once their owner's interval end is known.
func readTopics(researchDir string) (topicIndex, map[string][]RunFolder, error) {
	topicsDir := filepath.Join(researchDir, "topics")
	entries, err := os.ReadDir(topicsDir)
	if err != nil {
		return topicIndex{}, nil, err
	}
	realTopics, _ := filepath.EvalSymlinks(topicsDir)
	runs := map[string][]RunFolder{}
	index := topicIndex{names: map[string]string{}}
	aliases := map[string]string{}
	for _, entry := range entries {
		if !FolderID.MatchString(entry.Name()) {
			continue
		}
		path := filepath.Join(topicsDir, entry.Name())
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			// A link to a sibling folder that is itself listed is an alias.
			if target, err := filepath.EvalSymlinks(path); err == nil && filepath.Dir(target) == realTopics && FolderID.MatchString(filepath.Base(target)) {
				aliases[entry.Name()] = filepath.Base(target)
				continue
			}
		}
		t := &Topic{Slug: entry.Name(), Dir: path}
		index.topics = append(index.topics, t)
		index.names[t.Slug] = t.Slug
		missionPath := filepath.Join(t.Dir, "MISSION.md")
		if fact, ok := stat(missionPath); ok {
			t.Mission = &fact
		}
		if title, ok := firstHeading(missionPath); ok {
			t.Title = title
		} else if title, ok := firstHeading(filepath.Join(t.Dir, "README.md")); ok {
			t.Title = title
		} else {
			t.Title = Text{Evidence: Unknown("no MISSION.md or README.md title")}
		}
		runsDir := filepath.Join(t.Dir, "runs")
		runEntries, err := os.ReadDir(runsDir)
		if err != nil && !absent(err) {
			t.Unreadable = append(t.Unreadable, Unreadable(runsDir, err))
		}
		for _, runEntry := range runEntries {
			if !runEntry.IsDir() {
				continue
			}
			run := RunFolder{Topic: t.Slug, Path: filepath.Join(runsDir, runEntry.Name())}
			if fact, ok := stat(filepath.Join(run.Path, "CHECKPOINT.md")); ok {
				run.Checkpoint = &fact
			}
			runs[runEntry.Name()] = append(runs[runEntry.Name()], run)
		}
	}
	for alias, slug := range aliases {
		index.names[alias] = slug
		for _, t := range index.topics {
			if t.Slug == slug {
				t.Aliases = append(t.Aliases, alias)
				sort.Strings(t.Aliases)
			}
		}
	}
	return index, runs, nil
}

// applyMissionCheck attaches mission_check.py's per-topic summary.
func applyMissionCheck(topics []*Topic, output []byte, source string, at time.Time, err error) {
	if err != nil {
		for _, t := range topics {
			t.Check.Evidence = Unknown("mission_check.py unavailable: " + err.Error())
		}
		return
	}
	var rows []struct {
		Topic       string         `json:"topic"`
		Mode        string         `json:"mode"`
		Workstreams map[string]int `json:"workstreams"`
		Rows        []Workstream   `json:"workstream_rows"`
		Errors      []string       `json:"errors"`
		Warnings    []string       `json:"warnings"`
	}
	if jsonErr := json.Unmarshal(output, &rows); jsonErr != nil {
		for _, t := range topics {
			t.Check.Evidence = Unknown("mission_check.py output is not the expected JSON list: " + jsonErr.Error())
		}
		return
	}
	byTopic := map[string]int{}
	for i, row := range rows {
		byTopic[row.Topic] = i
	}
	for _, t := range topics {
		i, ok := byTopic[t.Slug]
		if !ok {
			t.Check.Evidence = Unknown("mission_check.py did not report this topic")
			continue
		}
		row := rows[i]
		t.Check = MissionCheck{
			Mode:            row.Mode,
			Open:            row.Workstreams["open"],
			Active:          row.Workstreams["active"],
			WaitingOperator: row.Workstreams["waiting-operator"],
			Landed:          row.Workstreams["landed"],
			Errors:          row.Errors,
			Warnings:        row.Warnings,
			Workstreams:     row.Rows,
			Evidence:        Evidence{Known: true, Source: source, AsOf: at},
		}
	}
}

// linkTopics fills each topic's commissions from the commissions' topic refs,
// its last research write from every run folder, owned or not, and its
// newest review. Commissions arrive sorted newest first.
func linkTopics(topics []*Topic, commissions []*Commission, unowned []RunFolder, reviews []*Review) {
	bySlug := map[string]*Topic{}
	for _, t := range topics {
		bySlug[t.Slug] = t
		t.LastResearchWrite = ResearchWrite{Topic: t.Slug, Time: Time{Evidence: Unknown("no file in any run folder")}}
	}
	activity := func(run RunFolder, c *Commission) {
		t := bySlug[run.Topic]
		if t == nil || !run.LastWrite.Known || (t.LastResearchWrite.Known && !run.LastWrite.Value.After(t.LastResearchWrite.Value)) {
			return
		}
		w := ResearchWrite{Topic: t.Slug, Time: run.LastWrite}
		if c != nil {
			w.CommissionID, w.State, w.Note = c.ID, c.State, writeNote(c, run.LastWrite.Value)
		}
		t.LastResearchWrite = w
	}
	for _, run := range unowned {
		activity(run, nil)
	}
	for _, r := range reviews {
		if r.Review == nil {
			continue
		}
		for _, slug := range r.Topics {
			if t := bySlug[slug]; t != nil && (t.LastReview == nil || r.Review.Modified.After(t.LastReview.File.Modified)) {
				t.LastReview = &ReviewTouch{ReviewID: r.ID, File: *r.Review}
			}
		}
	}
	for _, c := range commissions {
		for _, run := range c.Runs {
			activity(run, c)
		}
		for _, ref := range c.Topics {
			t, ok := bySlug[ref.Topic]
			if !ok {
				continue
			}
			t.CommissionIDs = append(t.CommissionIDs, c.ID)
			switch c.State {
			case StateLive:
				t.Live = append(t.Live, c.ID)
			case StatePaused:
				t.Paused = append(t.Paused, c.ID)
			case StateLimited:
				t.Limited = append(t.Limited, c.ID)
			}
		}
	}
	sort.SliceStable(topics, func(i, j int) bool {
		a, b := topics[i].LastResearchWrite, topics[j].LastResearchWrite
		if a.Known != b.Known {
			return a.Known
		}
		return a.Value.After(b.Value)
	})
}

// LatestResearchWrite is the newest write in any topic's run folders.
func (s *Snapshot) LatestResearchWrite() ResearchWrite {
	out := ResearchWrite{Time: Time{Evidence: Unknown("no file in any run folder")}}
	for _, t := range s.Topics {
		if w := t.LastResearchWrite; w.Known && (!out.Known || w.Value.After(out.Value)) {
			out = w
		}
	}
	return out
}
