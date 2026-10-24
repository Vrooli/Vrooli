package goalhome

import (
	"bytes"
	"errors"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
)

// Retired register names belong to import, not the runtime format.
const (
	GoalFile        = "GOAL.md"
	QueueFile       = "QUEUE.md"
	WorkaroundsFile = "WORKAROUNDS.md"
)

// LoadLegacy reads the retired register set for explicit effort migrate only.
// Runtime boards and writers never fall back to this format.
func LoadLegacy(fsys fs.FS) (*Home, error) {
	home := &Home{FileBytes: make(map[string]int), EpochsComplete: true}
	goal, err := home.read(fsys, GoalFile)
	if err != nil {
		return nil, err
	}
	home.Goal = parseLegacyGoal(goal)
	if home.QueueData, err = home.read(fsys, QueueFile); err != nil {
		return nil, err
	}
	home.Queue = ParseQueue(home.QueueData)
	feedback, err := home.read(fsys, FeedbackFile)
	if err != nil {
		return nil, err
	}
	home.Feedback = ParseFeedback(feedback)
	workarounds, err := home.read(fsys, WorkaroundsFile)
	if err != nil {
		return nil, err
	}
	home.Workarounds = ParseWorkarounds(workarounds)
	entries, err := fs.ReadDir(fsys, EpochsDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	for _, entry := range entries {
		m := epochNamePattern.FindStringSubmatch(entry.Name())
		if m == nil || entry.IsDir() {
			continue
		}
		data, err := fs.ReadFile(fsys, path.Join(EpochsDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		home.FileBytes[path.Join(EpochsDir, entry.Name())] = len(data)
		epoch, err := ParseEpoch(bytes.NewReader(data))
		if err != nil {
			home.Warnings = append(home.Warnings, Finding{Code: "epoch-unreadable", File: path.Join(EpochsDir, entry.Name()), Detail: err.Error()})
			continue
		}
		number, _ := strconv.Atoi(m[1])
		home.Epochs = append(home.Epochs, EpochFile{Name: entry.Name(), Number: number, Epoch: epoch})
	}
	sort.Slice(home.Epochs, func(i, j int) bool { return home.Epochs[i].Number < home.Epochs[j].Number })
	return home, nil
}

// parseLegacyGoal reads the retired GOAL.md fields for import.
func parseLegacyGoal(data []byte) Goal {
	doc := parseDocument(data)
	goal := Goal{Bytes: len(data), Fields: doc.fields()}
	if m := destinationPattern.FindStringSubmatch(goal.Fields["destination"]); m != nil {
		target, _, _ := ParseNumber(m[3])
		goal.Destination = &Destination{Metric: strings.TrimSpace(m[1]), Op: m[2], Target: target}
	}
	if raw := goal.Fields["current"]; raw != "" {
		if value, rest, ok := ParseNumber(raw); ok {
			current := &Measurement{Value: value}
			if _, when, found := strings.Cut(rest, "@"); found {
				if fields := strings.Fields(when); len(fields) > 0 {
					current.At, _ = parseWhen(fields[0])
				}
			}
			goal.Current = current
		}
	}
	goal.DestinationSet, _ = parseWhen(goal.Fields["destination set"])
	return goal
}
