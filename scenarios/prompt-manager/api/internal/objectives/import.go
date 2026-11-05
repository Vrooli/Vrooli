package objectives

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"prompt-manager/internal/memberflow"
)

// This file owns the one-way migration from the operator's authored
// declarations (docs/director-swarm/strategy/OBJECTIVES.md plus each team.json
// `objectivesServed` block) into the objective domain.
//
// It deliberately reuses the memberflow parse rather than re-reading the prose:
// the migration must preserve exactly the ids, titles, qualifiers, evidence
// expectations and acknowledgements the existing reader already derives. What
// it adds is (a) a reviewed preview with source fingerprints and conflicts,
// (b) a recoverable snapshot of the raw declaration bytes, and (c) an
// acknowledgement-meaning-preserving carry across the digest-scheme change
// recorded by decision 90d7aa13-71c9-47b0-9296-b7e6208af5ad.

const (
	importSeverityError   = "error"
	importSeverityWarning = "warning"
)

// ImportConflict classifies one source disagreement the preview surfaced.
type ImportConflict struct {
	Kind        string `json:"kind"`
	Severity    string `json:"severity"`
	ObjectiveID string `json:"objectiveId,omitempty"`
	TeamID      string `json:"teamId,omitempty"`
	Detail      string `json:"detail"`
}

// SourceFile is one operator declaration file captured for the snapshot.
type SourceFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Body   string `json:"body"`
}

// ImportSource is the recoverable, self-contained operator input to an import.
type ImportSource struct {
	Digest string       `json:"digest"`
	Files  []SourceFile `json:"files"`
}

// ImportedObjective is one parsed objective with both digest schemes so the
// reviewer can see exactly where a scheme change altered the token.
type ImportedObjective struct {
	ID                    string `json:"id"`
	Title                 string `json:"title"`
	Class                 string `json:"class"`
	EvidenceSource        string `json:"evidenceSource,omitempty"`
	GapMarker             string `json:"gapMarker,omitempty"`
	GlobalOrder           int    `json:"globalOrder"`
	LegacyMeaningRevision string `json:"legacyMeaningRevision"`
	NewMeaningRevision    string `json:"newMeaningRevision"`
}

// ImportedAttachment is one team link with the acknowledgement the import
// policy resolves for it. AcknowledgementCarried is true when the source
// acknowledgement matched the legacy digest and was carried by recomputing the
// new meaning revision instead of being left stale.
type ImportedAttachment struct {
	ObjectiveID                string `json:"objectiveId"`
	TeamID                     string `json:"teamId"`
	Role                       string `json:"role,omitempty"`
	Coverage                   string `json:"coverage,omitempty"`
	Note                       string `json:"note,omitempty"`
	Priority                   int    `json:"priority"`
	LegacyAcknowledgedRevision string `json:"legacyAcknowledgedRevision,omitempty"`
	NewAcknowledgedRevision    string `json:"newAcknowledgedRevision,omitempty"`
	RestatementPending         bool   `json:"restatementPending"`
	AcknowledgementCarried     bool   `json:"acknowledgementCarried"`
}

// AcknowledgementCarry records one digest-scheme migration of an ack.
type AcknowledgementCarry struct {
	ObjectiveID        string `json:"objectiveId"`
	TeamID             string `json:"teamId"`
	OldDigest          string `json:"oldDigest"`
	NewDigest          string `json:"newDigest"`
	RestatementPending bool   `json:"restatementPending"`
}

// ImportPreview is the reviewed, reproducible projection of the operator
// source before it is applied.
type ImportPreview struct {
	Source       ImportSource         `json:"source"`
	Objectives   []ImportedObjective  `json:"objectives"`
	Attachments  []ImportedAttachment `json:"attachments"`
	Conflicts    []ImportConflict     `json:"conflicts,omitempty"`
	CoverageGaps []string             `json:"coverageGaps,omitempty"`
}

// MigrationReceipt is the durable record of one applied import.
type MigrationReceipt struct {
	SourceDigest            string                 `json:"sourceDigest"`
	AppliedAt               string                 `json:"appliedAt,omitempty"`
	ObjectivesImported      int                    `json:"objectivesImported"`
	AttachmentsImported     int                    `json:"attachmentsImported"`
	AcknowledgementsCarried int                    `json:"acknowledgementsCarried"`
	RestatementPending      int                    `json:"restatementPending"`
	Conflicts               []ImportConflict       `json:"conflicts,omitempty"`
	AcknowledgementCarries  []AcknowledgementCarry `json:"acknowledgementCarries,omitempty"`
	SnapshotStored          bool                   `json:"snapshotStored"`
	AlreadyApplied          bool                   `json:"alreadyApplied"`
}

// ImportConflictError carries the error-class conflicts that stopped an import.
type ImportConflictError struct {
	Conflicts []ImportConflict
}

func (e *ImportConflictError) Error() string {
	return fmt.Sprintf("%s: %d blocking conflict(s)", ErrImportConflict.Error(), len(e.Conflicts))
}

// Is exposes the sentinel to errors.Is.
func (e *ImportConflictError) Is(target error) bool { return target == ErrImportConflict }

// PreviewImportFromRepo builds a reproducible import preview from the working
// tree. repoRoot is the repository root; configDir is the store directory that
// contains `teams/`. It reads only; it never mutates source or storage.
func PreviewImportFromRepo(repoRoot, configDir string) (ImportPreview, error) {
	registry, err := memberflow.LoadObjectives(repoRoot)
	if err != nil {
		return ImportPreview{}, fmt.Errorf("objectives: load operator objectives: %w", err)
	}
	declared, teamPaths, err := memberflow.LoadTeamObjectives(configDir)
	if err != nil {
		return ImportPreview{}, fmt.Errorf("objectives: load team declarations: %w", err)
	}
	files, digest, err := captureSourceFiles(repoRoot, teamPaths)
	if err != nil {
		return ImportPreview{}, err
	}
	preview := ImportPreview{Source: ImportSource{Digest: digest, Files: files}}

	legacyByID := map[string]string{}
	newByID := map[string]string{}
	orderByID := map[string]int{}
	for i, o := range registry.Objectives {
		id := strings.ToUpper(strings.TrimSpace(o.ID))
		domain := Objective{
			ID:             o.ID,
			Title:          o.Title,
			Class:          Class(strings.ToLower(strings.TrimSpace(o.Class))),
			EvidenceSource: o.EvidenceSource,
			GapMarker:      o.GapMarker,
			GlobalOrder:    i,
		}
		newRev := ComputeMeaningRevision(domain)
		legacyByID[id] = strings.TrimSpace(o.Revision)
		newByID[id] = newRev
		orderByID[id] = i
		preview.Objectives = append(preview.Objectives, ImportedObjective{
			ID:                    o.ID,
			Title:                 o.Title,
			Class:                 strings.ToLower(strings.TrimSpace(o.Class)),
			EvidenceSource:        o.EvidenceSource,
			GapMarker:             o.GapMarker,
			GlobalOrder:           i,
			LegacyMeaningRevision: legacyByID[id],
			NewMeaningRevision:    newRev,
		})
	}

	seenPair := map[string]bool{}
	for _, teamID := range sortedTeamIDs(declared) {
		decls := declared[teamID]
		// A team that declares no objectives at all is undeclared, not empty:
		// importing an attachment for it would invent intent.
		if decls == nil {
			continue
		}
		for priority, d := range decls {
			id := strings.ToUpper(strings.TrimSpace(d.ID))
			legacy, known := legacyByID[id]
			if !known {
				preview.Conflicts = append(preview.Conflicts, ImportConflict{
					Kind:        "unknown-objective",
					Severity:    importSeverityError,
					ObjectiveID: id,
					TeamID:      teamID,
					Detail:      fmt.Sprintf("team %q declares objective %q which is not defined in %s", teamID, d.ID, memberflow.ObjectivesDocPath),
				})
				continue
			}
			pair := id + "\x00" + teamID
			if seenPair[pair] {
				preview.Conflicts = append(preview.Conflicts, ImportConflict{
					Kind:        "duplicate-declaration",
					Severity:    importSeverityError,
					ObjectiveID: id,
					TeamID:      teamID,
					Detail:      fmt.Sprintf("team %q declares objective %q more than once", teamID, d.ID),
				})
				continue
			}
			seenPair[pair] = true

			att := ImportedAttachment{
				ObjectiveID:                d.ID,
				TeamID:                     teamID,
				Role:                       strings.ToLower(strings.TrimSpace(d.Role)),
				Coverage:                   strings.ToLower(strings.TrimSpace(d.Coverage)),
				Note:                       strings.TrimSpace(d.Note),
				Priority:                   priority,
				LegacyAcknowledgedRevision: strings.TrimSpace(d.AcknowledgedRevision),
			}
			newRev := newByID[id]
			switch {
			case att.LegacyAcknowledgedRevision != "" && att.LegacyAcknowledgedRevision == legacy:
				// The team confirmed the exact statement the source still
				// carries. Recompute the new token so the confirmed fact
				// survives the digest-scheme change.
				att.NewAcknowledgedRevision = newRev
				att.AcknowledgementCarried = true
			default:
				// Either never acknowledged, or already stale before the
				// migration. Preserve that state; do not advance it.
				att.NewAcknowledgedRevision = att.LegacyAcknowledgedRevision
				preview.Conflicts = append(preview.Conflicts, ImportConflict{
					Kind:        "acknowledgement-stale",
					Severity:    importSeverityWarning,
					ObjectiveID: id,
					TeamID:      teamID,
					Detail:      fmt.Sprintf("team %q acknowledgement %q does not match the legacy meaning %q; restatement stays pending", teamID, att.LegacyAcknowledgedRevision, legacy),
				})
			}
			att.RestatementPending = att.NewAcknowledgedRevision != newRev
			preview.Attachments = append(preview.Attachments, att)
		}
	}

	// Coverage gaps: the table names a team that does not declare the objective
	// back. This is the asymmetry the coverage rule reports as a real gap.
	for _, o := range registry.Objectives {
		id := strings.ToUpper(strings.TrimSpace(o.ID))
		for _, ref := range o.ServedBy {
			if !seenPair[id+"\x00"+ref.TeamID] {
				preview.CoverageGaps = append(preview.CoverageGaps, fmt.Sprintf("%s<->%s", o.ID, ref.TeamID))
			}
		}
	}
	sort.Strings(preview.CoverageGaps)

	// A declaration that the table does not name is a one-sided link too, but
	// the declaration is the team's own obligation, so it is a warning rather
	// than a blocking conflict.
	for _, att := range preview.Attachments {
		if !tableNamesTeam(registry, att.ObjectiveID, att.TeamID) {
			preview.Conflicts = append(preview.Conflicts, ImportConflict{
				Kind:        "declaration-without-table",
				Severity:    importSeverityWarning,
				ObjectiveID: strings.ToUpper(strings.TrimSpace(att.ObjectiveID)),
				TeamID:      att.TeamID,
				Detail:      fmt.Sprintf("team %q declares an objective the table does not name for it", att.TeamID),
			})
		}
	}
	return preview, nil
}

// Import applies a reviewed preview through the single objective service. The
// snapshot and every mutation share one transaction where the store supports
// it; a repeated import of the same source digest is a no-op that returns the
// stored receipt.
func (s *Service) Import(ctx context.Context, preview ImportPreview) (MigrationReceipt, error) {
	if err := validateImport(preview); err != nil {
		return MigrationReceipt{}, err
	}
	snapshot, err := json.Marshal(preview.Source)
	if err != nil {
		return MigrationReceipt{}, fmt.Errorf("objectives: encode snapshot: %w", err)
	}
	var receipt MigrationReceipt
	err = s.repo.WithTx(ctx, func(tx Repository) error {
		if existing, ok, err := tx.GetImportReceipt(ctx, preview.Source.Digest); err != nil {
			return err
		} else if ok {
			if err := json.Unmarshal([]byte(existing), &receipt); err != nil {
				return fmt.Errorf("objectives: decode prior receipt: %w", err)
			}
			receipt.AlreadyApplied = true
			return nil
		}
		var importErr error
		receipt, importErr = importInto(ctx, tx, preview, string(snapshot))
		return importErr
	})
	if err != nil {
		return MigrationReceipt{}, err
	}
	return receipt, nil
}

// importInto performs the import against one repository (normally a
// transaction). The receipt is written last, so an interrupted import leaves no
// receipt and the next attempt converges.
func importInto(ctx context.Context, repo Repository, preview ImportPreview, snapshot string) (MigrationReceipt, error) {
	if err := repo.PutImportSnapshot(ctx, preview.Source.Digest, snapshot); err != nil {
		return MigrationReceipt{}, fmt.Errorf("objectives: store import snapshot: %w", err)
	}

	for _, io := range preview.Objectives {
		obj := Objective{
			ID:             io.ID,
			Title:          io.Title,
			Class:          Class(strings.ToLower(strings.TrimSpace(io.Class))),
			EvidenceSource: io.EvidenceSource,
			GapMarker:      io.GapMarker,
			GlobalOrder:    io.GlobalOrder,
		}
		obj.MeaningRevision = ComputeMeaningRevision(obj)
		// Mirror UpsertObjective's normalization: a declared evidence source is
		// what makes an objective measurable. Deriving HasEvidence here rather
		// than trusting a caller keeps the persisted flag consistent with the
		// text, so the unmeasurable signal never fires on an objective that
		// names an instrument. HasEvidence is excluded from the meaning digest,
		// so this cannot invalidate a carried acknowledgement.
		obj.HasEvidence = strings.TrimSpace(obj.EvidenceSource) != ""
		if existing, ok, err := repo.GetObjective(ctx, obj.ID); err != nil {
			return MigrationReceipt{}, err
		} else if ok {
			obj.GlobalOrder = existing.GlobalOrder
		}
		if err := repo.PutObjective(ctx, obj); err != nil {
			return MigrationReceipt{}, fmt.Errorf("objectives: import objective %q: %w", obj.ID, err)
		}
	}

	teams := map[string]bool{}
	for _, ia := range preview.Attachments {
		att := Attachment{
			ObjectiveID:          ia.ObjectiveID,
			TeamID:               ia.TeamID,
			Role:                 strings.ToLower(strings.TrimSpace(ia.Role)),
			Coverage:             strings.ToLower(strings.TrimSpace(ia.Coverage)),
			Note:                 strings.TrimSpace(ia.Note),
			Priority:             ia.Priority,
			AcknowledgedRevision: ia.NewAcknowledgedRevision,
		}
		if err := repo.PutAttachment(ctx, att); err != nil {
			return MigrationReceipt{}, fmt.Errorf("objectives: import attachment %s/%s: %w", ia.TeamID, ia.ObjectiveID, err)
		}
		teams[ia.TeamID] = true
	}
	for teamID := range teams {
		atts, err := repo.ListAttachments(ctx, teamID)
		if err != nil {
			return MigrationReceipt{}, err
		}
		if err := repo.PutTeamAttachmentRevision(ctx, teamID, ComputeAttachmentRevision(atts)); err != nil {
			return MigrationReceipt{}, fmt.Errorf("objectives: persist team revision %q: %w", teamID, err)
		}
	}

	receipt := buildReceipt(preview)
	receipt.SnapshotStored = true
	encoded, err := json.Marshal(receipt)
	if err != nil {
		return MigrationReceipt{}, fmt.Errorf("objectives: encode receipt: %w", err)
	}
	if err := repo.PutImportReceipt(ctx, preview.Source.Digest, string(encoded)); err != nil {
		return MigrationReceipt{}, fmt.Errorf("objectives: store receipt: %w", err)
	}
	return receipt, nil
}

// validateImport refuses an import whose error-class conflicts would erase or
// duplicate intent. Warnings are recorded, not blocking.
func validateImport(preview ImportPreview) error {
	var blocking []ImportConflict
	for _, c := range preview.Conflicts {
		if c.Severity == importSeverityError {
			blocking = append(blocking, c)
		}
	}
	if len(blocking) > 0 {
		return &ImportConflictError{Conflicts: blocking}
	}
	return nil
}

func buildReceipt(preview ImportPreview) MigrationReceipt {
	receipt := MigrationReceipt{
		SourceDigest:        preview.Source.Digest,
		AppliedAt:           time.Now().UTC().Format(time.RFC3339),
		ObjectivesImported:  len(preview.Objectives),
		AttachmentsImported: len(preview.Attachments),
		Conflicts:           preview.Conflicts,
	}
	for _, a := range preview.Attachments {
		if a.AcknowledgementCarried {
			receipt.AcknowledgementsCarried++
		}
		if a.RestatementPending {
			receipt.RestatementPending++
		}
		receipt.AcknowledgementCarries = append(receipt.AcknowledgementCarries, AcknowledgementCarry{
			ObjectiveID:        a.ObjectiveID,
			TeamID:             a.TeamID,
			OldDigest:          a.LegacyAcknowledgedRevision,
			NewDigest:          a.NewAcknowledgedRevision,
			RestatementPending: a.RestatementPending,
		})
	}
	return receipt
}

// captureSourceFiles reads the objective document and every team.json, and
// returns both the snapshot bodies and a digest over their (path, sha256)
// pairs. The digest is the migration's idempotency key.
func captureSourceFiles(repoRoot string, teamPaths map[string]string) ([]SourceFile, string, error) {
	type entry struct {
		path string
		file SourceFile
	}
	var entries []entry
	docPath := filepath.Join(repoRoot, memberflow.ObjectivesDocPath)
	body, err := os.ReadFile(docPath)
	if err != nil {
		return nil, "", fmt.Errorf("objectives: read %q: %w", docPath, err)
	}
	entries = append(entries, entry{path: memberflow.ObjectivesDocPath, file: SourceFile{
		Path:   memberflow.ObjectivesDocPath,
		SHA256: sha256Hex(body),
		Body:   string(body),
	}})

	for _, teamID := range sortedTeamIDs(teamPaths) {
		path := teamPaths[teamID]
		b, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, "", fmt.Errorf("objectives: read %q: %w", path, err)
		}
		entries = append(entries, entry{path: path, file: SourceFile{
			Path:   filepath.ToSlash(path),
			SHA256: sha256Hex(b),
			Body:   string(b),
		}})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })

	hash := sha256.New()
	files := make([]SourceFile, 0, len(entries))
	for _, e := range entries {
		fmt.Fprintf(hash, "%s\x00%s\n", filepath.ToSlash(e.path), e.file.SHA256)
		files = append(files, e.file)
	}
	return files, hex.EncodeToString(hash.Sum(nil)), nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func tableNamesTeam(registry memberflow.ObjectiveRegistry, objectiveID, teamID string) bool {
	for _, ref := range registry.TeamsFor(objectiveID) {
		if ref.TeamID == teamID {
			return true
		}
	}
	return false
}

func sortedTeamIDs[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
