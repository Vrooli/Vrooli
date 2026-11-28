package store

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SkillRetirement records the evidence and custody needed to remove a skill
// from discovery without destroying its authored content or version history.
// The caller is responsible for establishing that the skill has no live,
// external, or retained callers before invoking Retire.
type SkillRetirement struct {
	Action    string   `json:"action"`
	SkillID   string   `json:"skillId"`
	Pack      string   `json:"pack"`
	RetiredAt string   `json:"retiredAt"`
	RetiredBy string   `json:"retiredBy"`
	Reason    string   `json:"reason"`
	Evidence  []string `json:"evidence"`
	Revision  int      `json:"revision"`
}

var errInvalidSkillRetirement = errors.New("skill retirement requires actor, reason, and evidence references")

// Retire removes an eligible authored skill from active packs while preserving
// its complete directory under skills/retired. Only the separate retirement
// record is added; existing source, versions, and history are left intact.
// Caller analysis is an owner workflow responsibility and must be reflected in
// the supplied evidence references.
func (s *FileSkillStore) Retire(ctx context.Context, id, actor, reason string, evidence []string) (*SkillRetirement, error) {
	original := s
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return nil, err
	}
	actor, reason = strings.TrimSpace(actor), strings.TrimSpace(reason)
	cleanEvidence := make([]string, 0, len(evidence))
	for _, ref := range evidence {
		if ref = strings.TrimSpace(ref); ref != "" {
			cleanEvidence = append(cleanEvidence, ref)
		}
	}
	if actor == "" || reason == "" || len(cleanEvidence) == 0 {
		return nil, errInvalidSkillRetirement
	}
	skill, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if skill.SourceDir != "" {
		return nil, fmt.Errorf("cannot retire scenario-owned skill %s; retire it at %s", id, skill.SourceDir)
	}
	if skill.Origin != nil && skill.Origin.Kind == OriginImported {
		return nil, fmt.Errorf("cannot retire imported skill %s through authored-skill lifecycle", id)
	}
	if skill.Pack != "core" && skill.Pack != "local" {
		return nil, fmt.Errorf("cannot retire skill %s from pack %s", id, skill.Pack)
	}
	source := filepath.Join(s.packsDir(), skill.Pack, id)
	destination := filepath.Join(s.skillsDir(), "retired", skill.Pack, id)
	if _, err := os.Stat(destination); err == nil {
		return nil, fmt.Errorf("retired skill destination already exists: %s", id)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("checking retired destination: %w", err)
	}
	record := &SkillRetirement{
		SkillID: id, Pack: skill.Pack, RetiredAt: time.Now().UTC().Format(time.RFC3339),
		RetiredBy: actor, Reason: reason, Evidence: cleanEvidence, Revision: skill.Revision,
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return nil, fmt.Errorf("creating retirement custody: %w", err)
	}
	// Append the disposition into the source directory before the atomic rename,
	// so a successful move can never lack its custody record. JSONL retains each
	// retire/restore cycle instead of overwriting prior lifecycle history.
	record.Action = "retired"
	if err := AppendJSONL(filepath.Join(source, "lifecycle.jsonl"), record); err != nil {
		return nil, fmt.Errorf("writing retirement record: %w", err)
	}
	if err := os.Rename(source, destination); err != nil {
		// Keep the failed attempt's audit entry: the source remains active, and
		// the caller can distinguish an attempted move from an accomplished one.
		return nil, fmt.Errorf("moving skill into retirement custody: %w", err)
	}
	original.recordWrite(ctx)
	return record, nil
}

// Restore returns a retired skill to its original pack. The retirement record
// remains with the skill as provenance; version history is never rewritten.
func (s *FileSkillStore) Restore(ctx context.Context, id, actor, reason string) error {
	original := s
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return err
	}
	actor, reason = strings.TrimSpace(actor), strings.TrimSpace(reason)
	if actor == "" || reason == "" {
		return errors.New("skill restore requires actor and reason")
	}
	for _, pack := range []string{"core", "local"} {
		source := filepath.Join(s.skillsDir(), "retired", pack, id)
		lifecyclePath := filepath.Join(source, "lifecycle.jsonl")
		file, err := os.Open(lifecyclePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("reading lifecycle history for %s: %w", id, err)
		}
		var last SkillRetirement
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var entry SkillRetirement
			if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
				_ = file.Close()
				return fmt.Errorf("parsing lifecycle history for %s: %w", id, err)
			}
			last = entry
		}
		if err := scanner.Err(); err != nil {
			_ = file.Close()
			return fmt.Errorf("reading lifecycle history for %s: %w", id, err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("closing lifecycle history for %s: %w", id, err)
		}
		if last.Action != "retired" || last.SkillID != id || last.Pack != pack {
			return fmt.Errorf("skill %s is not currently retired", id)
		}
		destination := filepath.Join(s.packsDir(), pack, id)
		if _, err := os.Stat(destination); err == nil {
			return fmt.Errorf("cannot restore %s: active skill destination already exists", id)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("checking active skill destination: %w", err)
		}
		last.Action = "restored"
		last.RetiredAt = time.Now().UTC().Format(time.RFC3339)
		last.RetiredBy = actor
		last.Reason = reason
		last.Evidence = nil
		if err := AppendJSONL(lifecyclePath, last); err != nil {
			return fmt.Errorf("recording restore for %s: %w", id, err)
		}
		if err := os.Rename(source, destination); err != nil {
			return fmt.Errorf("restoring skill %s: %w", id, err)
		}
		original.recordWrite(ctx)
		return nil
	}
	return fmt.Errorf("retired skill not found: %s", id)
}
