package releases

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	ErrInvalid  = errors.New("invalid release edition")
	ErrNotFound = errors.New("release not found")
	ErrConflict = errors.New("release revision conflict")
	ErrNotReady = errors.New("release is missing showcase evidence")
	hashRE      = regexp.MustCompile(`^[0-9a-f]{64}$`)
	idRE        = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,159}$`)
	localeRE    = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type ArtistResolver interface {
	ResolveArtist(context.Context, string, string, string) (bool, error)
}

type AssetVerifier interface {
	VerifyAsset(context.Context, Asset) (bool, error)
}

type DerivationVerifier interface {
	VerifyDerivation(context.Context, Derivation) (bool, error)
}

type Store struct {
	db             SQLExecutor
	artistResolver ArtistResolver
	assetVerifier  AssetVerifier
	derivVerifier  DerivationVerifier
}

func NewStore(db SQLExecutor) *Store { return &Store{db: db} }

func NewStoreWithEvidenceAdapters(db SQLExecutor, artists ArtistResolver, assets AssetVerifier) *Store {
	return &Store{db: db, artistResolver: artists, assetVerifier: assets}
}

func NewStoreWithLineageAdapter(db SQLExecutor, artists ArtistResolver, assets AssetVerifier, derivations DerivationVerifier) *Store {
	return &Store{db: db, artistResolver: artists, assetVerifier: assets, derivVerifier: derivations}
}

type LocalizedText struct {
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle,omitempty"`
	Description string `json:"description,omitempty"`
}

type TrackOccurrence struct {
	Order         int      `json:"order"`
	TrackSHA256   string   `json:"track_sha256"`
	TitleOverride string   `json:"title_override,omitempty"`
	Languages     []string `json:"languages,omitempty"`
}

// RecordingProvenance is populated from the exact catalog row and pinned into
// the release revision. API callers cannot supply the take or receipt identity.
type RecordingProvenance struct {
	TrackSHA256 string `json:"track_sha256"`
	TakeID      string `json:"take_id"`
	ReceiptRef  string `json:"receipt_ref"`
	Approval    string `json:"operator_track_approval"`
}

type Credit struct {
	ID             string `json:"credit_id"`
	Scope          string `json:"scope"` // release or track
	TrackOrder     int    `json:"track_order,omitempty"`
	SubjectOwner   string `json:"subject_owner,omitempty"`
	SubjectID      string `json:"subject_id,omitempty"`
	SubjectVersion string `json:"subject_version,omitempty"`
	Role           string `json:"role"`
	CreditedAs     string `json:"credited_as"`
	ProvenanceRef  string `json:"provenance_ref,omitempty"`
}

type Asset struct {
	ID            string `json:"asset_id"`
	Role          string `json:"role"`
	OwnerScenario string `json:"owner_scenario"`
	Ref           string `json:"asset_ref"`
	SHA256        string `json:"asset_sha256"`
	OperationID   string `json:"operation_id,omitempty"`
	ModelID       string `json:"model_id,omitempty"`
	ModelRevision string `json:"model_revision,omitempty"`
	LicenseLane   string `json:"license_lane,omitempty"`
	RightsStatus  string `json:"rights_status"`
	ProvenanceRef string `json:"provenance_ref"`
}

type TrackRights struct {
	TrackSHA256   string `json:"track_sha256"`
	SourceLicense string `json:"source_license,omitempty"`
	UsageScope    string `json:"usage_scope"`
	Decision      string `json:"decision_status"`
	DecisionID    string `json:"decision_id,omitempty"`
	EvidenceRef   string `json:"evidence_ref,omitempty"`
	Note          string `json:"note,omitempty"`
}

type RightsDecision struct {
	ID                        string    `json:"decision_id"`
	ReleaseID                 string    `json:"release_id"`
	Revision                  int64     `json:"revision"`
	TrackSHA256               string    `json:"track_sha256"`
	SourceLicense             string    `json:"source_license"`
	UsageScope                string    `json:"usage_scope"`
	Decision                  string    `json:"decision"`
	EvidenceRef               string    `json:"evidence_ref"`
	RecordingProvenanceSHA256 string    `json:"recording_provenance_sha256"`
	DecisionSource            string    `json:"reviewer_source"`
	Note                      string    `json:"note"`
	CreatedAt                 time.Time `json:"created_at"`
}

type Derivation struct {
	ID                  string  `json:"derivation_id"`
	Kind                string  `json:"kind"` // remix or sample
	SourceTrackSHA256   string  `json:"source_track_sha256"`
	DerivedTrackSHA256  string  `json:"derived_track_sha256"`
	OperationOwner      string  `json:"operation_owner"`
	OperationID         string  `json:"operation_id"`
	SourceStartSeconds  float64 `json:"source_start_seconds"`
	SourceEndSeconds    float64 `json:"source_end_seconds"`
	DerivedStartSeconds float64 `json:"derived_start_seconds"`
	DerivedEndSeconds   float64 `json:"derived_end_seconds"`
	InheritedRights     string  `json:"inherited_rights_scope"`
	EvidenceRef         string  `json:"evidence_ref"`
}

type Edition struct {
	ReleaseID             string                   `json:"release_id"`
	Revision              int64                    `json:"revision"`
	Kind                  string                   `json:"kind"`
	Status                string                   `json:"status"`
	Title                 string                   `json:"title"`
	DefaultLocale         string                   `json:"default_locale"`
	ArtistOwner           string                   `json:"artist_owner,omitempty"`
	ArtistSubjectID       string                   `json:"artist_subject_id,omitempty"`
	ArtistSubjectVersion  string                   `json:"artist_subject_version,omitempty"`
	LocalizedMetadata     map[string]LocalizedText `json:"localized_metadata"`
	Tracks                []TrackOccurrence        `json:"tracks"`
	Recordings            []RecordingProvenance    `json:"recording_provenance,omitempty"`
	Credits               []Credit                 `json:"credits"`
	Assets                []Asset                  `json:"assets"`
	TrackRights           []TrackRights            `json:"track_rights"`
	RightsDecisions       []RightsDecision         `json:"rights_decisions,omitempty"`
	Derivations           []Derivation             `json:"derivations"`
	VerifiedDerivationIDs []string                 `json:"verified_derivation_ids,omitempty"`
	CreatedAt             time.Time                `json:"created_at"`
}

type Review struct {
	ID                   string    `json:"review_id"`
	ReleaseID            string    `json:"release_id"`
	Revision             int64     `json:"revision"`
	Decision             string    `json:"decision"`
	EditionSHA256        string    `json:"edition_sha256"`
	HeardAllTracks       bool      `json:"heard_all_tracks"`
	ReviewedTrackSHA256s []string  `json:"reviewed_track_sha256s"`
	DecisionSource       string    `json:"reviewer_source"`
	Note                 string    `json:"note"`
	CreatedAt            time.Time `json:"created_at"`
}

type CheckStatus string

const (
	CheckSupported CheckStatus = "supported"
	CheckMissing   CheckStatus = "missing"
	CheckBlocked   CheckStatus = "blocked"
)

type EligibilityCheck struct {
	Property string      `json:"property"`
	Status   CheckStatus `json:"status"`
	Evidence []string    `json:"evidence,omitempty"`
	Missing  []string    `json:"missing,omitempty"`
}

type Eligibility struct {
	ReadyForOperatorJudgment bool               `json:"ready_for_operator_judgment"`
	ShowcaseReady            bool               `json:"showcase_ready"`
	Checks                   []EligibilityCheck `json:"checks"`
}

func AssessShowcase(e Edition, reviews []Review, artistResolved, assetsVerified bool) Eligibility {
	checks := []EligibilityCheck{}
	unique := map[string]bool{}
	validHashes := e.Kind == "ep" && len(e.Tracks) >= 4 && len(e.Tracks) <= 6
	for _, track := range e.Tracks {
		if !hashRE.MatchString(track.TrackSHA256) || unique[track.TrackSHA256] || track.Order < 1 || track.Order > len(e.Tracks) {
			validHashes = false
		}
		unique[track.TrackSHA256] = true
	}
	trackStatus, trackMissing := CheckSupported, []string(nil)
	if !validHashes {
		trackStatus = CheckMissing
		trackMissing = append(trackMissing, "four to six distinct exact-hash track occurrences")
	}
	checks = append(checks, EligibilityCheck{Property: "ordered_track_identity", Status: trackStatus, Evidence: hashEvidence(e.Tracks), Missing: trackMissing})
	recordingStatus, recordingMissing := CheckSupported, []string(nil)
	if !hasRecordingProvenance(e) {
		recordingStatus = CheckBlocked
		recordingMissing = []string{"every exact catalog hash must pin a liked operator record, take id, and source receipt"}
	}
	checks = append(checks, EligibilityCheck{Property: "track_recording_provenance", Status: recordingStatus, Evidence: recordingEvidence(e.Recordings), Missing: recordingMissing})

	artistStatus, artistEvidence, artistMissing := CheckMissing, []string(nil), []string(nil)
	if e.ArtistOwner != "" && e.ArtistSubjectID != "" && e.ArtistSubjectVersion != "" {
		artistEvidence = []string{e.ArtistOwner + ":" + e.ArtistSubjectID + "@" + e.ArtistSubjectVersion}
		if artistResolved {
			artistStatus = CheckSupported
		} else {
			artistStatus = CheckBlocked
			artistMissing = []string{"resolve pinned artist subject through its canonical owner"}
		}
	} else {
		artistMissing = []string{"versioned fictional-artist subject reference"}
	}
	checks = append(checks, EligibilityCheck{Property: "fictional_artist_continuity", Status: artistStatus, Evidence: artistEvidence, Missing: artistMissing})

	locales := make([]string, 0, len(e.LocalizedMetadata))
	for locale, text := range e.LocalizedMetadata {
		if localeRE.MatchString(locale) && strings.TrimSpace(text.Title) != "" {
			locales = append(locales, locale)
		}
	}
	sort.Strings(locales)
	localeStatus, localeMissing := CheckSupported, []string(nil)
	if len(locales) < 2 {
		localeStatus = CheckMissing
		localeMissing = []string{"release metadata in at least two explicitly populated locales"}
	}
	checks = append(checks, EligibilityCheck{Property: "multilingual_release_metadata", Status: localeStatus, Evidence: locales, Missing: localeMissing})

	creditStatus, creditMissing := CheckSupported, []string(nil)
	if !completeCredits(e) {
		creditStatus = CheckMissing
		creditMissing = []string{"release-level primary artist credit", "per-track performer and composer/author credits"}
	}
	checks = append(checks, EligibilityCheck{Property: "complete_credits", Status: creditStatus, Missing: creditMissing})

	coverFound := false
	for _, asset := range e.Assets {
		if asset.Role == "cover_front" && hashRE.MatchString(asset.SHA256) && asset.OwnerScenario != "" && asset.Ref != "" && asset.OperationID != "" && asset.ProvenanceRef != "" && asset.RightsStatus != "unknown" && asset.RightsStatus != "review_required" && asset.RightsStatus != "blocked" && assetsVerified {
			coverFound = true
		}
	}
	coverStatus, coverMissing := CheckSupported, []string(nil)
	if !coverFound {
		coverStatus = CheckMissing
		coverMissing = []string{"existing exact-hash cover asset, source operation, and resolved rights"}
	}
	checks = append(checks, EligibilityCheck{Property: "cover_asset_provenance", Status: coverStatus, Missing: coverMissing})

	rightsStatus, rightsMissing := CheckSupported, []string(nil)
	rightsByHash := make(map[string]TrackRights, len(e.TrackRights))
	for _, rights := range e.TrackRights {
		rightsByHash[rights.TrackSHA256] = rights
	}
	for _, track := range e.Tracks {
		rights, ok := rightsByHash[track.TrackSHA256]
		if !ok || rights.Decision != "operator_confirmed" || rights.DecisionID == "" || rights.SourceLicense == "" || rights.EvidenceRef == "" || rights.UsageScope == "unknown" || rights.UsageScope == "blocked" {
			rightsMissing = append(rightsMissing, "operator rights decision for track "+track.TrackSHA256)
		}
	}
	if len(rightsMissing) > 0 {
		rightsStatus = CheckBlocked
	}
	checks = append(checks, EligibilityCheck{Property: "source_rights", Status: rightsStatus, Missing: rightsMissing})

	derivationStatus, derivationMissing := CheckSupported, []string(nil)
	if len(e.Derivations) == 0 {
		derivationStatus = CheckMissing
		derivationMissing = []string{"at least one exact-hash remix or sample relationship with operation evidence"}
	}
	for _, derivation := range e.Derivations {
		derivedIncluded := false
		for _, track := range e.Tracks {
			if track.TrackSHA256 == derivation.DerivedTrackSHA256 {
				derivedIncluded = true
				break
			}
		}
		if !derivedIncluded || !contains(e.VerifiedDerivationIDs, derivation.ID) || !hashRE.MatchString(derivation.SourceTrackSHA256) || !hashRE.MatchString(derivation.DerivedTrackSHA256) || derivation.OperationOwner == "" || derivation.OperationID == "" || derivation.EvidenceRef == "" || derivation.InheritedRights == "unknown" || derivation.InheritedRights == "blocked" || !rightsDoNotWiden(e.TrackRights, derivation) {
			derivationStatus = CheckBlocked
			derivationMissing = append(derivationMissing, "verified operation lineage to an included exact-hash track and no rights widening")
		}
	}
	checks = append(checks, EligibilityCheck{Property: "remix_sample_lineage", Status: derivationStatus, Missing: uniqueStrings(derivationMissing)})

	reviewed := false
	digest, digestErr := editionDigest(e)
	for _, review := range reviews {
		if digestErr == nil && review.ReleaseID == e.ReleaseID && review.Revision == e.Revision && review.Decision == "approved" && review.EditionSHA256 == digest && review.HeardAllTracks && sameReviewedTracks(e.Tracks, review.ReviewedTrackSHA256s) {
			reviewed = true
		}
	}
	reviewStatus, reviewMissing := CheckSupported, []string(nil)
	if !reviewed {
		reviewStatus = CheckMissing
		reviewMissing = []string{"operator's one-batch judgment pinned to this exact edition digest"}
	}
	checks = append(checks, EligibilityCheck{Property: "operator_ep_judgment", Status: reviewStatus, Missing: reviewMissing})

	readyForJudgment, ready := true, true
	for _, check := range checks {
		if check.Status != CheckSupported {
			if check.Property != "operator_ep_judgment" {
				readyForJudgment = false
			}
			ready = false
		}
	}
	return Eligibility{ReadyForOperatorJudgment: readyForJudgment, ShowcaseReady: ready, Checks: checks}
}

func rightsDoNotWiden(rights []TrackRights, derivation Derivation) bool {
	var sourceScope, derivedScope string
	sourceApproved, derivedApproved := false, false
	for _, item := range rights {
		if item.TrackSHA256 == derivation.SourceTrackSHA256 {
			sourceScope = item.UsageScope
			sourceApproved = item.Decision == "operator_confirmed" && item.DecisionID != ""
		}
		if item.TrackSHA256 == derivation.DerivedTrackSHA256 {
			derivedScope = item.UsageScope
			derivedApproved = item.Decision == "operator_confirmed" && item.DecisionID != ""
		}
	}
	permission := func(scope string) int {
		switch scope {
		case "commercial":
			return 2
		case "personal_only":
			return 1
		case "blocked":
			return -1
		default:
			return 0
		}
	}
	return sourceApproved && derivedApproved && sourceScope != "unknown" && sourceScope != "blocked" && derivation.InheritedRights == sourceScope && derivedScope != "unknown" && derivedScope != "blocked" && permission(derivedScope) <= permission(sourceScope)
}

func completeCredits(e Edition) bool {
	releaseArtist := false
	artistSubjectMatches := false
	byTrack := make(map[int]map[string]bool)
	for _, credit := range e.Credits {
		if credit.Scope == "release" && credit.Role == "primary_artist" && credit.CreditedAs != "" {
			releaseArtist = true
			artistSubjectMatches = e.ArtistOwner != "" && e.ArtistSubjectID != "" && credit.SubjectOwner == e.ArtistOwner && credit.SubjectID == e.ArtistSubjectID && credit.SubjectVersion == e.ArtistSubjectVersion
		}
		if credit.Scope == "track" && credit.TrackOrder > 0 && credit.CreditedAs != "" {
			if byTrack[credit.TrackOrder] == nil {
				byTrack[credit.TrackOrder] = map[string]bool{}
			}
			if credit.SubjectOwner != "" && credit.SubjectID != "" && credit.SubjectVersion != "" {
				byTrack[credit.TrackOrder][credit.Role] = true
			}
		}
	}
	if !releaseArtist || !artistSubjectMatches || len(e.Tracks) == 0 {
		return false
	}
	for _, track := range e.Tracks {
		roles := byTrack[track.Order]
		if !roles["performer"] || (!roles["composer"] && !roles["author"]) {
			return false
		}
	}
	return true
}

func hasRecordingProvenance(e Edition) bool {
	byHash := make(map[string]RecordingProvenance, len(e.Recordings))
	for _, record := range e.Recordings {
		byHash[record.TrackSHA256] = record
	}
	for _, track := range e.Tracks {
		record, ok := byHash[track.TrackSHA256]
		if !ok || record.Approval != "liked" || record.TakeID == "" || record.ReceiptRef == "" {
			return false
		}
	}
	return true
}

func editionDigest(e Edition) (string, error) {
	encoded, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func hashEvidence(tracks []TrackOccurrence) []string {
	out := make([]string, 0, len(tracks))
	for _, track := range tracks {
		out = append(out, fmt.Sprintf("%02d:%s", track.Order, track.TrackSHA256))
	}
	return out
}

func recordingEvidence(records []RecordingProvenance) []string {
	out := make([]string, 0, len(records))
	for _, record := range records {
		out = append(out, record.TrackSHA256+" take="+record.TakeID+" receipt="+record.ReceiptRef+" approval="+record.Approval)
	}
	sort.Strings(out)
	return out
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func sameReviewedTracks(tracks []TrackOccurrence, hashes []string) bool {
	if len(tracks) == 0 || len(tracks) != len(hashes) {
		return false
	}
	for i := range tracks {
		if tracks[i].TrackSHA256 != hashes[i] {
			return false
		}
	}
	return true
}

func (s *Store) SaveDraft(ctx context.Context, e Edition) (Edition, error) {
	if err := validate(e); err != nil {
		return Edition{}, err
	}
	// Recording provenance is always derived from the exact catalog row inside
	// this transaction; caller supplied take and receipt fields are ignored.
	e.Recordings = nil
	e.RightsDecisions = nil
	e.VerifiedDerivationIDs = nil
	localized, err := json.Marshal(e.LocalizedMetadata)
	if err != nil {
		return Edition{}, err
	}
	if e.Status == "" {
		e.Status = "draft"
	}
	if e.Status != "draft" {
		return Edition{}, ErrInvalid
	}
	now := time.Now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Edition{}, err
	}
	defer tx.Rollback()
	var maxRevision int64
	err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(revision),0) FROM music_release_versions WHERE release_id=?`, e.ReleaseID).Scan(&maxRevision)
	if err != nil {
		return Edition{}, err
	}
	if e.Revision != maxRevision+1 {
		return Edition{}, ErrConflict
	}
	if maxRevision == 0 {
		if _, err := tx.ExecContext(ctx, `INSERT INTO music_releases (release_id,created_at) VALUES (?,?)`, e.ReleaseID, now.Format(time.RFC3339Nano)); err != nil {
			return Edition{}, err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO music_release_versions (release_id,revision,kind,status,title,default_locale,artist_owner,artist_subject_id,artist_subject_version,localized_metadata_json,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, e.ReleaseID, e.Revision, e.Kind, e.Status, e.Title, e.DefaultLocale, e.ArtistOwner, e.ArtistSubjectID, e.ArtistSubjectVersion, string(localized), now.Format(time.RFC3339Nano))
	if err != nil {
		return Edition{}, err
	}
	for _, track := range e.Tracks {
		var provenance RecordingProvenance
		err = tx.QueryRowContext(ctx, `SELECT sha256,take_id,receipt_path,approval FROM music_catalog_tracks WHERE sha256=?`, track.TrackSHA256).Scan(&provenance.TrackSHA256, &provenance.TakeID, &provenance.ReceiptRef, &provenance.Approval)
		if errors.Is(err, sql.ErrNoRows) {
			return Edition{}, ErrNotFound
		}
		if err != nil {
			return Edition{}, err
		}
		if strings.TrimSpace(provenance.TakeID) == "" || strings.TrimSpace(provenance.ReceiptRef) == "" {
			provenance.Approval = "unverified"
		}
		languages, err := json.Marshal(track.Languages)
		if err != nil {
			return Edition{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO music_release_track_occurrences (release_id,revision,track_order,track_sha256,title_override,language_tags_json) VALUES (?,?,?,?,?,?)`, e.ReleaseID, e.Revision, track.Order, track.TrackSHA256, track.TitleOverride, string(languages)); err != nil {
			return Edition{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO music_release_recording_provenance (release_id,revision,track_order,track_sha256,take_id,receipt_ref,operator_track_approval) VALUES (?,?,?,?,?,?,?)`, e.ReleaseID, e.Revision, track.Order, track.TrackSHA256, provenance.TakeID, provenance.ReceiptRef, provenance.Approval); err != nil {
			return Edition{}, err
		}
		e.Recordings = append(e.Recordings, provenance)
	}
	for _, credit := range e.Credits {
		if _, err := tx.ExecContext(ctx, `INSERT INTO music_release_credits (credit_id,release_id,revision,scope,track_order,subject_owner,subject_id,role,credited_as,provenance_ref) VALUES (?,?,?,?,?,?,?,?,?,?)`, credit.ID, e.ReleaseID, e.Revision, credit.Scope, credit.TrackOrder, credit.SubjectOwner, credit.SubjectID, credit.Role, credit.CreditedAs, credit.ProvenanceRef); err != nil {
			return Edition{}, err
		}
		if credit.SubjectVersion != "" {
			if _, err := tx.ExecContext(ctx, `INSERT INTO music_release_credit_identity_versions (release_id,revision,credit_id,subject_version) VALUES (?,?,?,?)`, e.ReleaseID, e.Revision, credit.ID, credit.SubjectVersion); err != nil {
				return Edition{}, err
			}
		}
	}
	for _, asset := range e.Assets {
		if _, err := tx.ExecContext(ctx, `INSERT INTO music_release_assets (asset_id,release_id,revision,role,owner_scenario,asset_ref,asset_sha256,operation_id,model_id,model_revision,license_lane,rights_status,provenance_ref) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`, asset.ID, e.ReleaseID, e.Revision, asset.Role, asset.OwnerScenario, asset.Ref, asset.SHA256, asset.OperationID, asset.ModelID, asset.ModelRevision, asset.LicenseLane, asset.RightsStatus, asset.ProvenanceRef); err != nil {
			return Edition{}, err
		}
	}
	for i := range e.TrackRights {
		rights := &e.TrackRights[i]
		rights.Decision = "review_required"
		rights.DecisionID = ""
		if _, err := tx.ExecContext(ctx, `INSERT INTO music_release_track_rights (release_id,revision,track_sha256,source_license,usage_scope,decision_status,evidence_ref,note) VALUES (?,?,?,?,?,?,?,?)`, e.ReleaseID, e.Revision, rights.TrackSHA256, rights.SourceLicense, rights.UsageScope, rights.Decision, rights.EvidenceRef, rights.Note); err != nil {
			return Edition{}, err
		}
	}
	for _, derivation := range e.Derivations {
		if _, err := tx.ExecContext(ctx, `INSERT INTO music_release_derivations (derivation_id,release_id,revision,kind,source_track_sha256,derived_track_sha256,operation_owner,operation_id,source_start_seconds,source_end_seconds,derived_start_seconds,derived_end_seconds,inherited_rights_scope,evidence_ref) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, derivation.ID, e.ReleaseID, e.Revision, derivation.Kind, derivation.SourceTrackSHA256, derivation.DerivedTrackSHA256, derivation.OperationOwner, derivation.OperationID, derivation.SourceStartSeconds, derivation.SourceEndSeconds, derivation.DerivedStartSeconds, derivation.DerivedEndSeconds, derivation.InheritedRights, derivation.EvidenceRef); err != nil {
			return Edition{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Edition{}, err
	}
	e.CreatedAt = now
	return e, nil
}

func (s *Store) Get(ctx context.Context, releaseID string, revision int64) (Edition, error) {
	var e Edition
	var localized, created string
	query := `SELECT release_id,revision,kind,status,title,default_locale,artist_owner,artist_subject_id,artist_subject_version,localized_metadata_json,created_at FROM music_release_versions WHERE release_id=?`
	args := []any{releaseID}
	if revision > 0 {
		query += ` AND revision=?`
		args = append(args, revision)
	} else {
		query += ` ORDER BY revision DESC LIMIT 1`
	}
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&e.ReleaseID, &e.Revision, &e.Kind, &e.Status, &e.Title, &e.DefaultLocale, &e.ArtistOwner, &e.ArtistSubjectID, &e.ArtistSubjectVersion, &localized, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return Edition{}, ErrNotFound
	}
	if err != nil {
		return Edition{}, err
	}
	if err := json.Unmarshal([]byte(localized), &e.LocalizedMetadata); err != nil {
		return Edition{}, err
	}
	e.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return Edition{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT track_order,track_sha256,title_override,language_tags_json FROM music_release_track_occurrences WHERE release_id=? AND revision=? ORDER BY track_order`, e.ReleaseID, e.Revision)
	if err != nil {
		return Edition{}, err
	}
	for rows.Next() {
		var item TrackOccurrence
		var languages string
		if err := rows.Scan(&item.Order, &item.TrackSHA256, &item.TitleOverride, &languages); err != nil {
			rows.Close()
			return Edition{}, err
		}
		if err := json.Unmarshal([]byte(languages), &item.Languages); err != nil {
			rows.Close()
			return Edition{}, err
		}
		e.Tracks = append(e.Tracks, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Edition{}, err
	}
	if err := rows.Close(); err != nil {
		return Edition{}, err
	}
	e.Recordings, err = s.getRecordingProvenance(ctx, e.ReleaseID, e.Revision)
	if err != nil {
		return Edition{}, err
	}
	e.Credits, err = s.getCredits(ctx, e.ReleaseID, e.Revision)
	if err != nil {
		return Edition{}, err
	}
	e.Assets, err = s.getAssets(ctx, e.ReleaseID, e.Revision)
	if err != nil {
		return Edition{}, err
	}
	e.TrackRights, err = s.getTrackRights(ctx, e.ReleaseID, e.Revision)
	if err != nil {
		return Edition{}, err
	}
	e.RightsDecisions, err = s.ListRightsDecisions(ctx, e.ReleaseID, e.Revision)
	if err != nil {
		return Edition{}, err
	}
	latestRights := map[string]RightsDecision{}
	for _, decision := range e.RightsDecisions {
		if prior, ok := latestRights[decision.TrackSHA256]; !ok || decision.CreatedAt.After(prior.CreatedAt) || (decision.CreatedAt.Equal(prior.CreatedAt) && decision.ID > prior.ID) {
			latestRights[decision.TrackSHA256] = decision
		}
	}
	for i := range e.TrackRights {
		if decision, ok := latestRights[e.TrackRights[i].TrackSHA256]; ok {
			e.TrackRights[i].DecisionID = decision.ID
			if decision.Decision == "confirmed" && decision.SourceLicense == e.TrackRights[i].SourceLicense && decision.UsageScope == e.TrackRights[i].UsageScope && decision.EvidenceRef == e.TrackRights[i].EvidenceRef {
				e.TrackRights[i].Decision = "operator_confirmed"
			} else {
				e.TrackRights[i].Decision = "blocked"
			}
		}
	}
	e.Derivations, err = s.getDerivations(ctx, e.ReleaseID, e.Revision)
	return e, err
}

func (s *Store) getRecordingProvenance(ctx context.Context, releaseID string, revision int64) ([]RecordingProvenance, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT track_sha256,take_id,receipt_ref,operator_track_approval FROM music_release_recording_provenance WHERE release_id=? AND revision=? ORDER BY track_order`, releaseID, revision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RecordingProvenance{}
	for rows.Next() {
		var item RecordingProvenance
		if err := rows.Scan(&item.TrackSHA256, &item.TakeID, &item.ReceiptRef, &item.Approval); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) getCredits(ctx context.Context, releaseID string, revision int64) ([]Credit, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT c.credit_id,c.scope,c.track_order,c.subject_owner,c.subject_id,COALESCE(v.subject_version,''),c.role,c.credited_as,c.provenance_ref FROM music_release_credits c LEFT JOIN music_release_credit_identity_versions v ON v.release_id=c.release_id AND v.revision=c.revision AND v.credit_id=c.credit_id WHERE c.release_id=? AND c.revision=? ORDER BY c.scope,c.track_order,c.credit_id`, releaseID, revision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Credit{}
	for rows.Next() {
		var item Credit
		if err := rows.Scan(&item.ID, &item.Scope, &item.TrackOrder, &item.SubjectOwner, &item.SubjectID, &item.SubjectVersion, &item.Role, &item.CreditedAs, &item.ProvenanceRef); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) getAssets(ctx context.Context, releaseID string, revision int64) ([]Asset, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT asset_id,role,owner_scenario,asset_ref,asset_sha256,operation_id,model_id,model_revision,license_lane,rights_status,provenance_ref FROM music_release_assets WHERE release_id=? AND revision=? ORDER BY role,asset_id`, releaseID, revision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Asset{}
	for rows.Next() {
		var item Asset
		if err := rows.Scan(&item.ID, &item.Role, &item.OwnerScenario, &item.Ref, &item.SHA256, &item.OperationID, &item.ModelID, &item.ModelRevision, &item.LicenseLane, &item.RightsStatus, &item.ProvenanceRef); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) getTrackRights(ctx context.Context, releaseID string, revision int64) ([]TrackRights, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT track_sha256,source_license,usage_scope,decision_status,evidence_ref,note FROM music_release_track_rights WHERE release_id=? AND revision=? ORDER BY track_sha256`, releaseID, revision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TrackRights{}
	for rows.Next() {
		var item TrackRights
		if err := rows.Scan(&item.TrackSHA256, &item.SourceLicense, &item.UsageScope, &item.Decision, &item.EvidenceRef, &item.Note); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) ListRightsDecisions(ctx context.Context, releaseID string, revision int64) ([]RightsDecision, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT decision_id,release_id,revision,track_sha256,source_license,usage_scope,decision,evidence_ref,recording_provenance_sha256,reviewer_source,note,created_at FROM music_release_rights_decisions WHERE release_id=? AND revision=? ORDER BY track_sha256,created_at,decision_id`, releaseID, revision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RightsDecision{}
	for rows.Next() {
		var item RightsDecision
		var created string
		if err := rows.Scan(&item.ID, &item.ReleaseID, &item.Revision, &item.TrackSHA256, &item.SourceLicense, &item.UsageScope, &item.Decision, &item.EvidenceRef, &item.RecordingProvenanceSHA256, &item.DecisionSource, &item.Note, &created); err != nil {
			return nil, err
		}
		if item.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) RecordRightsDecision(ctx context.Context, decision RightsDecision) (RightsDecision, error) {
	if !idRE.MatchString(decision.ID) || !idRE.MatchString(decision.ReleaseID) || decision.Revision < 1 || !hashRE.MatchString(decision.TrackSHA256) || strings.TrimSpace(decision.SourceLicense) != decision.SourceLicense || decision.SourceLicense == "" || (decision.UsageScope != "personal_only" && decision.UsageScope != "commercial" && decision.UsageScope != "blocked") || (decision.Decision != "confirmed" && decision.Decision != "denied") || strings.TrimSpace(decision.EvidenceRef) != decision.EvidenceRef || decision.EvidenceRef == "" || strings.TrimSpace(decision.Note) != decision.Note || len(decision.Note) > 2000 {
		return RightsDecision{}, ErrInvalid
	}
	edition, err := s.Get(ctx, decision.ReleaseID, decision.Revision)
	if err != nil {
		return RightsDecision{}, err
	}
	var proposed *TrackRights
	for i := range edition.TrackRights {
		if edition.TrackRights[i].TrackSHA256 == decision.TrackSHA256 {
			proposed = &edition.TrackRights[i]
			break
		}
	}
	if proposed == nil || proposed.SourceLicense != decision.SourceLicense || proposed.UsageScope != decision.UsageScope || proposed.EvidenceRef != decision.EvidenceRef {
		return RightsDecision{}, ErrNotReady
	}
	var recording *RecordingProvenance
	for i := range edition.Recordings {
		if edition.Recordings[i].TrackSHA256 == decision.TrackSHA256 {
			recording = &edition.Recordings[i]
			break
		}
	}
	if recording == nil || recording.Approval != "liked" || recording.ReceiptRef != decision.EvidenceRef || (decision.Decision == "confirmed" && decision.UsageScope == "blocked") {
		return RightsDecision{}, ErrNotReady
	}
	encoded, err := json.Marshal(*recording)
	if err != nil {
		return RightsDecision{}, err
	}
	provenanceHash := sha256.Sum256(encoded)
	decision.RecordingProvenanceSHA256 = hex.EncodeToString(provenanceHash[:])
	decision.DecisionSource = "operator"
	decision.CreatedAt = time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `INSERT INTO music_release_rights_decisions (decision_id,release_id,revision,track_sha256,source_license,usage_scope,decision,evidence_ref,recording_provenance_sha256,reviewer_source,note,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, decision.ID, decision.ReleaseID, decision.Revision, decision.TrackSHA256, decision.SourceLicense, decision.UsageScope, decision.Decision, decision.EvidenceRef, decision.RecordingProvenanceSHA256, decision.DecisionSource, decision.Note, decision.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return RightsDecision{}, err
	}
	return decision, nil
}

func (s *Store) getDerivations(ctx context.Context, releaseID string, revision int64) ([]Derivation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT derivation_id,kind,source_track_sha256,derived_track_sha256,operation_owner,operation_id,source_start_seconds,source_end_seconds,derived_start_seconds,derived_end_seconds,inherited_rights_scope,evidence_ref FROM music_release_derivations WHERE release_id=? AND revision=? ORDER BY derivation_id`, releaseID, revision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Derivation{}
	for rows.Next() {
		var item Derivation
		if err := rows.Scan(&item.ID, &item.Kind, &item.SourceTrackSHA256, &item.DerivedTrackSHA256, &item.OperationOwner, &item.OperationID, &item.SourceStartSeconds, &item.SourceEndSeconds, &item.DerivedStartSeconds, &item.DerivedEndSeconds, &item.InheritedRights, &item.EvidenceRef); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) ListReviews(ctx context.Context, releaseID string, revision int64) ([]Review, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT r.review_id,r.release_id,r.revision,r.decision,COALESCE(p.edition_sha256,''),COALESCE(a.heard_all_tracks,0),COALESCE(a.reviewed_track_hashes_json,'[]'),r.reviewer_source,r.note,r.created_at FROM music_release_reviews r LEFT JOIN music_release_review_pins p ON p.review_id=r.review_id LEFT JOIN music_release_review_attestations a ON a.review_id=r.review_id WHERE r.release_id=? AND (?=0 OR r.revision=?) ORDER BY r.revision,r.created_at,r.review_id`, releaseID, revision, revision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Review{}
	for rows.Next() {
		var item Review
		var created string
		var heardAllTracks bool
		var reviewedHashes string
		if err := rows.Scan(&item.ID, &item.ReleaseID, &item.Revision, &item.Decision, &item.EditionSHA256, &heardAllTracks, &reviewedHashes, &item.DecisionSource, &item.Note, &created); err != nil {
			return nil, err
		}
		item.HeardAllTracks = heardAllTracks
		if err := json.Unmarshal([]byte(reviewedHashes), &item.ReviewedTrackSHA256s); err != nil {
			return nil, err
		}
		if item.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) Assess(ctx context.Context, releaseID string, revision int64) (Edition, []Review, Eligibility, error) {
	e, err := s.Get(ctx, releaseID, revision)
	if err != nil {
		return Edition{}, nil, Eligibility{}, err
	}
	reviews, err := s.ListReviews(ctx, e.ReleaseID, e.Revision)
	if err != nil {
		return Edition{}, nil, Eligibility{}, err
	}
	artistResolved := false
	if s.artistResolver != nil && e.ArtistOwner != "" {
		artistResolved, err = s.artistResolver.ResolveArtist(ctx, e.ArtistOwner, e.ArtistSubjectID, e.ArtistSubjectVersion)
		if err != nil {
			return Edition{}, nil, Eligibility{}, err
		}
		if artistResolved {
			for _, credit := range e.Credits {
				if credit.SubjectOwner == "" {
					continue
				}
				resolved, resolveErr := s.artistResolver.ResolveArtist(ctx, credit.SubjectOwner, credit.SubjectID, credit.SubjectVersion)
				if resolveErr != nil {
					return Edition{}, nil, Eligibility{}, resolveErr
				}
				if !resolved {
					artistResolved = false
					break
				}
			}
		}
	}
	assetsVerified := false
	if s.assetVerifier != nil {
		for _, asset := range e.Assets {
			if asset.Role != "cover_front" {
				continue
			}
			verified, verifyErr := s.assetVerifier.VerifyAsset(ctx, asset)
			if verifyErr != nil {
				return Edition{}, nil, Eligibility{}, verifyErr
			}
			if verified {
				assetsVerified = true
				break
			}
		}
	}
	e.VerifiedDerivationIDs = nil
	if s.derivVerifier != nil {
		for _, derivation := range e.Derivations {
			verified, verifyErr := s.derivVerifier.VerifyDerivation(ctx, derivation)
			if verifyErr != nil {
				return Edition{}, nil, Eligibility{}, verifyErr
			}
			if verified {
				e.VerifiedDerivationIDs = append(e.VerifiedDerivationIDs, derivation.ID)
			}
		}
	}
	return e, reviews, AssessShowcase(e, reviews, artistResolved, assetsVerified), nil
}

func (s *Store) RecordReview(ctx context.Context, review Review) (Review, error) {
	if !idRE.MatchString(review.ID) || !idRE.MatchString(review.ReleaseID) || review.Revision < 1 || (review.Decision != "approved" && review.Decision != "changes_requested") || len(review.Note) > 4000 {
		return Review{}, ErrInvalid
	}
	edition, _, eligibility, err := s.Assess(ctx, review.ReleaseID, review.Revision)
	if err != nil {
		return Review{}, err
	}
	if review.Decision == "approved" && !eligibility.ReadyForOperatorJudgment {
		return Review{}, ErrNotReady
	}
	if !review.HeardAllTracks || !sameReviewedTracks(edition.Tracks, review.ReviewedTrackSHA256s) {
		return Review{}, ErrNotReady
	}
	review.EditionSHA256, err = editionDigest(edition)
	if err != nil {
		return Review{}, err
	}
	now := time.Now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Review{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO music_release_reviews (review_id,release_id,revision,decision,reviewer_source,note,created_at) VALUES (?,?,?,?,'operator',?,?)`, review.ID, review.ReleaseID, review.Revision, review.Decision, review.Note, now.Format(time.RFC3339Nano))
	if err != nil {
		return Review{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO music_release_review_pins (review_id,release_id,revision,edition_sha256) VALUES (?,?,?,?)`, review.ID, review.ReleaseID, review.Revision, review.EditionSHA256)
	if err != nil {
		return Review{}, err
	}
	reviewedHashes, err := json.Marshal(review.ReviewedTrackSHA256s)
	if err != nil {
		return Review{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO music_release_review_attestations (review_id,heard_all_tracks,reviewed_track_hashes_json) VALUES (?,?,?)`, review.ID, review.HeardAllTracks, string(reviewedHashes))
	if err != nil {
		return Review{}, err
	}
	if err = tx.Commit(); err != nil {
		return Review{}, err
	}
	review.CreatedAt, review.DecisionSource = now, "operator"
	return review, nil
}

func validate(e Edition) error {
	if !idRE.MatchString(e.ReleaseID) || e.Revision < 1 || (e.Kind != "single" && e.Kind != "ep" && e.Kind != "album") || (e.Status != "" && e.Status != "draft") || strings.TrimSpace(e.Title) != e.Title || e.Title == "" || len(e.Title) > 512 || !localeRE.MatchString(e.DefaultLocale) || len(e.Tracks) == 0 || len(e.Tracks) > 100 {
		return fmt.Errorf("%w: release identity, kind, draft title, locale or track count is invalid", ErrInvalid)
	}
	artistParts := []string{e.ArtistOwner, e.ArtistSubjectID, e.ArtistSubjectVersion}
	artistCount := 0
	for _, part := range artistParts {
		if part != "" {
			artistCount++
			if len(part) > 200 || strings.TrimSpace(part) != part || strings.ContainsAny(part, "\r\n\x00") {
				return ErrInvalid
			}
		}
	}
	if artistCount != 0 && artistCount != 3 {
		return fmt.Errorf("%w: external artist owner, subject id and version must be supplied together", ErrInvalid)
	}
	defaultText, ok := e.LocalizedMetadata[e.DefaultLocale]
	if !ok || strings.TrimSpace(defaultText.Title) == "" {
		return fmt.Errorf("%w: default locale needs explicit title metadata", ErrInvalid)
	}
	for locale, text := range e.LocalizedMetadata {
		if !localeRE.MatchString(locale) || strings.TrimSpace(text.Title) == "" || len(text.Title) > 512 || len(text.Subtitle) > 512 || len(text.Description) > 4000 {
			return fmt.Errorf("%w: localized metadata is invalid", ErrInvalid)
		}
	}
	orders, hashes, creditIDs, rightsHashes, assetIDs, derivationIDs := map[int]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, track := range e.Tracks {
		if track.Order < 1 || orders[track.Order] || !hashRE.MatchString(track.TrackSHA256) || strings.TrimSpace(track.TitleOverride) != track.TitleOverride || len(track.TitleOverride) > 512 {
			return fmt.Errorf("%w: track occurrence order, hash or title is invalid", ErrInvalid)
		}
		orders[track.Order], hashes[track.TrackSHA256] = true, true
		for _, language := range track.Languages {
			if !localeRE.MatchString(language) {
				return fmt.Errorf("%w: invalid track language tag", ErrInvalid)
			}
		}
	}
	for order := 1; order <= len(e.Tracks); order++ {
		if !orders[order] {
			return fmt.Errorf("%w: track order must be contiguous from one", ErrInvalid)
		}
	}
	for _, credit := range e.Credits {
		subjectFields := 0
		for _, field := range []string{credit.SubjectOwner, credit.SubjectID, credit.SubjectVersion} {
			if field != "" {
				subjectFields++
				if strings.TrimSpace(field) != field || len(field) > 200 {
					return fmt.Errorf("%w: credit subject identity is invalid", ErrInvalid)
				}
			}
		}
		if !idRE.MatchString(credit.ID) || creditIDs[credit.ID] || (credit.Scope != "release" && credit.Scope != "track") || strings.TrimSpace(credit.Role) != credit.Role || credit.Role == "" || len(credit.Role) > 80 || strings.TrimSpace(credit.CreditedAs) != credit.CreditedAs || credit.CreditedAs == "" || len(credit.CreditedAs) > 256 || (credit.Scope == "release" && credit.TrackOrder != 0) || (credit.Scope == "track" && (credit.TrackOrder < 1 || !orders[credit.TrackOrder])) || (subjectFields != 0 && subjectFields != 3) {
			return fmt.Errorf("%w: credit is invalid or not bound to an edition occurrence", ErrInvalid)
		}
		creditIDs[credit.ID] = true
	}
	for _, asset := range e.Assets {
		if !idRE.MatchString(asset.ID) || assetIDs[asset.ID] || strings.TrimSpace(asset.Role) != asset.Role || asset.Role == "" || len(asset.Role) > 80 || strings.TrimSpace(asset.OwnerScenario) != asset.OwnerScenario || asset.OwnerScenario == "" || strings.TrimSpace(asset.Ref) != asset.Ref || asset.Ref == "" || !hashRE.MatchString(asset.SHA256) || (asset.RightsStatus != "unknown" && asset.RightsStatus != "review_required" && asset.RightsStatus != "personal_use" && asset.RightsStatus != "commercial_cleared" && asset.RightsStatus != "blocked") || (asset.RightsStatus == "commercial_cleared" && asset.ProvenanceRef == "") || (asset.ProvenanceRef == "") {
			return fmt.Errorf("%w: asset must keep exact hash, owner reference, and rights provenance", ErrInvalid)
		}
		assetIDs[asset.ID] = true
	}
	for _, rights := range e.TrackRights {
		if !hashes[rights.TrackSHA256] || rightsHashes[rights.TrackSHA256] || (rights.UsageScope != "unknown" && rights.UsageScope != "personal_only" && rights.UsageScope != "commercial" && rights.UsageScope != "blocked") || (rights.Decision != "unknown" && rights.Decision != "review_required" && rights.Decision != "operator_confirmed" && rights.Decision != "blocked") || (rights.UsageScope == "commercial" && (rights.Decision != "operator_confirmed" || rights.EvidenceRef == "")) || strings.TrimSpace(rights.EvidenceRef) != rights.EvidenceRef || strings.TrimSpace(rights.Note) != rights.Note || len(rights.Note) > 2000 {
			return fmt.Errorf("%w: track rights must be exact-hash-bound and cannot overstate an unreviewed license", ErrInvalid)
		}
		rightsHashes[rights.TrackSHA256] = true
	}
	for _, derivation := range e.Derivations {
		if !idRE.MatchString(derivation.ID) || derivationIDs[derivation.ID] || (derivation.Kind != "remix" && derivation.Kind != "sample") || !hashRE.MatchString(derivation.SourceTrackSHA256) || !hashRE.MatchString(derivation.DerivedTrackSHA256) || derivation.SourceTrackSHA256 == derivation.DerivedTrackSHA256 || strings.TrimSpace(derivation.OperationOwner) != derivation.OperationOwner || derivation.OperationOwner == "" || !idRE.MatchString(derivation.OperationID) || !finiteNonnegative(derivation.SourceStartSeconds) || derivation.SourceEndSeconds <= derivation.SourceStartSeconds || !finiteNonnegative(derivation.DerivedStartSeconds) || derivation.DerivedEndSeconds <= derivation.DerivedStartSeconds || (derivation.InheritedRights != "unknown" && derivation.InheritedRights != "personal_only" && derivation.InheritedRights != "commercial" && derivation.InheritedRights != "blocked") || strings.TrimSpace(derivation.EvidenceRef) == "" {
			return fmt.Errorf("%w: derivation needs two exact hashes, source spans, operation receipt and inherited rights", ErrInvalid)
		}
		derivationIDs[derivation.ID] = true
	}
	return nil
}

func finiteNonnegative(value float64) bool {
	return value >= 0 && value < 1e12 && value == value
}
