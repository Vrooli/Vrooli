package home

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Ceilings is what the skill's templates/limits.json says each role reads and
// may read. limits.json is the only list; the desk measures it.
type Ceilings struct {
	CharterBytes int
	Roles        []RoleReads
	// Fixed holds the skill-text items (every item that is not a commission
	// file), measured when limits.json was read.
	Fixed map[string]FileFact
	// Unmeasured says why a fixed item could not be measured.
	Unmeasured map[string]string
	// OwnerTopic names the files in topics/<topic>/ each owner also reads
	// (its MISSION.md). They are operator-authored: measured per topic and
	// added to the owner's total, never held to the owner_start ceiling.
	OwnerTopic []string
	// MissionBytes is the size past which a topic's MISSION.md is flagged
	// for tidying; never a launch gate.
	MissionBytes int64
	Source       string
}

// MissionFile is a topic's mission, the owner's charter.
const MissionFile = "MISSION.md"

// RoleReads is one read_sets list and its ceiling.
type RoleReads struct {
	// Key is the limits.json key: scheduler_start, scheduler_unit, owner_start.
	Key          string
	Items        []string
	CeilingBytes int64
}

// OwnerStartKey is the read set every topic owner reads at start; it is
// reported once per commission topic.
const OwnerStartKey = "owner_start"

// CommissionItem is the commission file a read-set item names: "charter" is
// the charter, a bare *.md name is that file in the commission folder. Any
// other item is skill text and returns "".
func CommissionItem(item string) string {
	if item == "charter" {
		return CharterFile
	}
	if strings.HasSuffix(item, ".md") && !strings.Contains(item, "/") {
		return item
	}
	return ""
}

// ReadSet is what one role reads to start or to run one unit, as limits.json
// lists it, with this commission's files in place of its commission items.
type ReadSet struct {
	Role            string // scheduler-start, scheduler-unit, owner-start:<topic>
	Files           []FileFact
	Bytes           int64
	EstimatedTokens int64
	Flags           []string
}

// datedHeading is a level-2+ heading carrying a date: an amendment appended
// to the charter rather than folded into it.
var datedHeading = regexp.MustCompile(`^#{2,6} .*\b20\d\d-\d\d-\d\d\b`)

// ReadSets measures each limits.json read set for this commission, an
// owner's with its topic's files added after the skill text is held to its
// ceiling. Without limits.json there is no list, so none (the caller says
// why).
func (s *Snapshot) ReadSets(c *Commission) []ReadSet {
	sets := c.readSets(s.Ceilings)
	for i := range sets {
		if topic, ok := strings.CutPrefix(sets[i].Role, ownerRole+":"); ok {
			s.addOwnerTopic(&sets[i], topic)
		}
	}
	return sets
}

// OwnerReading is what the owner of each topic reads at start, for topics
// not yet in a commission (a spec's preflight).
func (s *Snapshot) OwnerReading(topics []string) []ReadSet {
	c := &Commission{}
	for _, t := range topics {
		c.Topics = append(c.Topics, TopicRef{Topic: t})
	}
	var out []ReadSet
	for _, set := range s.ReadSets(c) {
		if strings.HasPrefix(set.Role, ownerRole+":") {
			out = append(out, set)
		}
	}
	return out
}

// ownerRole is the owner read set's role name.
var ownerRole = strings.ReplaceAll(OwnerStartKey, "_", "-")

// addOwnerTopic adds the topic files an owner reads to its set and flags a
// mission past MissionBytes with the recommended tidy.
func (s *Snapshot) addOwnerTopic(set *ReadSet, topic string) {
	set.Files, set.Flags = slices.Clone(set.Files), slices.Clone(set.Flags)
	t := s.Topic(topic)
	for _, item := range s.Ceilings.OwnerTopic {
		if t == nil {
			set.Flags = append(set.Flags, fmt.Sprintf("topics/%s/%s not counted: no topic %s", topic, item, topic))
			continue
		}
		path := filepath.Join(t.Dir, item)
		fact, ok := stat(path)
		if !ok {
			set.Flags = append(set.Flags, fmt.Sprintf("%s not counted: no such file", path))
			continue
		}
		set.Files = append(set.Files, fact)
		set.Bytes += fact.Bytes
		if item == MissionFile && s.Ceilings.MissionBytes > 0 && fact.Bytes > s.Ceilings.MissionBytes {
			set.Flags = append(set.Flags, fmt.Sprintf("the %s mission is %d bytes, over the %d bytes a mission should stay under (%s); the owner reads all of it at start. Recommended: the next architect pass tidies the mission.", topic, fact.Bytes, s.Ceilings.MissionBytes, s.Ceilings.Source))
		}
	}
	set.EstimatedTokens = set.Bytes / 4
}

// readSets measures the read sets without topic files.
func (c *Commission) readSets(ceilings *Ceilings) []ReadSet {
	if ceilings == nil {
		return nil
	}
	var sets []ReadSet
	charterFlagged := false
	for _, role := range ceilings.Roles {
		set := ReadSet{}
		readsCharter := false
		for _, item := range role.Items {
			if name := CommissionItem(item); name != "" {
				readsCharter = readsCharter || name == CharterFile
				if fact, ok := c.File(name); ok {
					set.Files = append(set.Files, fact)
				} else if name == CharterFile {
					set.Flags = append(set.Flags, "no "+CharterFile+" charter")
				}
				continue
			}
			if fact, ok := ceilings.Fixed[item]; ok {
				set.Files = append(set.Files, fact)
			} else {
				set.Flags = append(set.Flags, fmt.Sprintf("%s not counted: %s", item, ceilings.Unmeasured[item]))
			}
		}
		for _, f := range set.Files {
			set.Bytes += f.Bytes
		}
		set.EstimatedTokens = set.Bytes / 4
		if role.CeilingBytes > 0 && set.Bytes > role.CeilingBytes {
			set.Flags = append(set.Flags, fmt.Sprintf("%d bytes is over the %d-byte %s ceiling (%s)", set.Bytes, role.CeilingBytes, role.Key, ceilings.Source))
		}
		if readsCharter && !charterFlagged {
			charterFlagged = true
			set.Flags = append(set.Flags, c.charterFlags(ceilings)...)
		}
		name := strings.ReplaceAll(role.Key, "_", "-")
		if role.Key != OwnerStartKey || len(c.Topics) == 0 {
			set.Role = name
			sets = append(sets, set)
			continue
		}
		for _, ref := range c.Topics {
			owner := set
			owner.Role = name + ":" + ref.Topic
			sets = append(sets, owner)
		}
	}
	return sets
}

// charterFlags are the charter's gate and appended-amendment checks.
func (c *Commission) charterFlags(ceilings *Ceilings) []string {
	charter, ok := c.File(CharterFile)
	if !ok {
		return nil
	}
	var flags []string
	if charter.Bytes > int64(ceilings.CharterBytes) {
		flags = append(flags, fmt.Sprintf("charter %d bytes is over the %d-byte gate (%s)", charter.Bytes, ceilings.CharterBytes, ceilings.Source))
	}
	n, first, err := datedHeadings(charter.Path)
	switch {
	case err != nil:
		flags = append(flags, "dated amendment headings unknown: "+Unreadable(charter.Path, err).Reason)
	case n > 0:
		flags = append(flags, fmt.Sprintf("%d dated amendment headings in the charter, first at line %d", n, first))
	}
	return flags
}

// datedHeadings counts dated amendment headings and the first one's line.
func datedHeadings(path string) (count, firstLine int, err error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for line := 1; scanner.Scan(); line++ {
		if datedHeading.MatchString(scanner.Text()) {
			if count == 0 {
				firstLine = line
			}
			count++
		}
	}
	return count, firstLine, scanner.Err()
}
