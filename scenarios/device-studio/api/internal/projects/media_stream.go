package projects

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"unicode"

	"github.com/vrooli/api-core/blobstore"
)

// StagedSourceMediaImport owns one unique, unreferenced routed object until
// Commit publishes metadata or Abort removes that object.
type StagedSourceMediaImport interface {
	Media() SourceMedia
	Commit(context.Context) (SourceMedia, error)
	Abort(context.Context) error
}

type InspectionProjection struct {
	ProjectID         string
	MediaID           string
	SourceSHA256      string
	JobID             string
	AttemptID         string
	Readiness         Readiness
	IssueCode         string
	IssueMessage      string
	Receipt           *InspectionReceipt
	ReceiptStorageKey string
}

// BindInspectionAttempt pins or advances the current attempt for one immutable
// source job. expectedAttempt is empty only for the first queued attempt.
func (s *sourceMediaService) BindInspectionAttempt(ctx context.Context, projectID, mediaID, sourceDigest, jobID, expectedAttempt, attemptID string) (SourceMedia, error) {
	return s.repo.BindInspectionAttempt(ctx, projectID, mediaID, sourceDigest, jobID, expectedAttempt, attemptID)
}

func (s *sourceMediaService) StoreInspectionArtifacts(ctx context.Context, projectID, mediaID, jobID, attemptID, receiptSHA, clockSHA string, receipt, clock []byte) (string, string, error) {
	row, err := s.repo.GetSourceMedia(ctx, projectID, mediaID)
	if err != nil {
		return "", "", err
	}
	if row.Kind != MediaKindVideo || row.InspectionJobID != jobID || row.InspectionAttemptID != attemptID || !validSourceDigest(receiptSHA) || digestBytes(receipt) != receiptSHA || len(receipt) == 0 || len(receipt) > 64<<10 {
		return "", "", ErrInspectionConflict
	}
	receiptID := digestBytes(receipt)
	clockRef := "source-clock.v1.bin"
	if len(clock) == 0 {
		if clockSHA != "" {
			return "", "", ErrInspectionInvalid
		}
	} else if !validSourceDigest(clockSHA) || digestBytes(clock) != clockSHA || len(clock) > 1<<20 {
		return "", "", ErrInspectionInvalid
	}
	receiptKey := InspectionReceiptObjectKey(projectID, mediaID, jobID, receiptID)
	clockKey := InspectionClockObjectKey(projectID, mediaID, jobID, clockRef)
	if err := s.store.Put(ctx, receiptKey, bytes.NewReader(receipt), "application/json"); err != nil {
		return "", "", err
	}
	if len(clock) > 0 {
		if err := s.store.Put(ctx, clockKey, bytes.NewReader(clock), "application/octet-stream"); err != nil {
			return "", "", err
		}
	}
	return receiptKey, clockRef, nil
}

func (s *sourceMediaService) OpenInspectionArtifacts(ctx context.Context, projectID, mediaID, jobID, attemptID, receiptSHA, clockSHA string) ([]byte, []byte, error) {
	row, err := s.repo.GetSourceMedia(ctx, projectID, mediaID)
	if err != nil {
		return nil, nil, err
	}
	if row.Kind != MediaKindVideo || row.InspectionJobID != jobID || row.InspectionAttemptID != attemptID || !validSourceDigest(receiptSHA) {
		return nil, nil, ErrInspectionConflict
	}
	receiptKey := InspectionReceiptObjectKey(projectID, mediaID, jobID, receiptSHA)
	receipt, err := s.readInspectionObject(ctx, receiptKey, 64<<10)
	if err != nil || digestBytes(receipt) != receiptSHA {
		return nil, nil, errors.Join(errors.New("inspection receipt object failed verification"), err)
	}
	var clock []byte
	if clockSHA != "" {
		if !validSourceDigest(clockSHA) {
			return nil, nil, ErrInspectionInvalid
		}
		clockKey := InspectionClockObjectKey(projectID, mediaID, jobID, "source-clock.v1.bin")
		clock, err = s.readInspectionObject(ctx, clockKey, 1<<20)
		if err != nil || digestBytes(clock) != clockSHA {
			return nil, nil, errors.Join(errors.New("inspection clock object failed verification"), err)
		}
	}
	return receipt, clock, nil
}

func (s *sourceMediaService) readInspectionObject(ctx context.Context, key string, max int64) ([]byte, error) {
	r, _, err := s.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	b, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, errors.New("inspection artifact exceeds owner limit")
	}
	return b, nil
}

var (
	ErrInspectionConflict = errors.New("source media inspection projection conflicts with current source state")
	ErrInspectionInvalid  = errors.New("source media inspection projection is invalid")
)

// InspectionReceiptObjectKey derives a private routed-object key from validated opaque IDs.
func InspectionReceiptObjectKey(projectID, mediaID, jobID, receiptID string) string {
	return "projects/" + projectID + "/media/" + mediaID + "/inspection/" + jobID + "/receipts/" + receiptID + ".json"
}

// InspectionClockObjectKey derives a private routed-object key for an immutable clock artifact.
func InspectionClockObjectKey(projectID, mediaID, jobID, clockArtifactID string) string {
	return "projects/" + projectID + "/media/" + mediaID + "/inspection/" + jobID + "/clocks/" + clockArtifactID + ".bin"
}

func (s *sourceMediaService) BeginImport(ctx context.Context, projectID, filename string, declaredSize int64, body io.Reader) (StagedSourceMediaImport, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, ErrInvalidSourceMedia{Reason: "project_id is required"}
	}
	if s.projects == nil || s.repo == nil || s.store == nil {
		return nil, errors.New("source media service is not configured")
	}
	if _, err := s.projects.Get(ctx, projectID); err != nil {
		return nil, err
	}
	if body == nil {
		return nil, ErrInvalidSourceMedia{Reason: "file is required"}
	}
	if declaredSize == 0 || declaredSize < -1 {
		return nil, ErrInvalidSourceMedia{Reason: "file is empty or declared size is invalid"}
	}
	if declaredSize > s.limits.MaxVideoBytes {
		return nil, ErrSourceMediaTooLarge{Limit: s.limits.MaxVideoBytes}
	}
	filename, err := sanitizeMediaFilename(filename, s.limits.MaxFilename)
	if err != nil {
		return nil, err
	}
	var prefix [mediaSniffPrefixBytes]byte
	prefixSize, prefixErr := io.ReadFull(body, prefix[:])
	if prefixErr != nil && !errors.Is(prefixErr, io.EOF) && !errors.Is(prefixErr, io.ErrUnexpectedEOF) {
		return nil, fmt.Errorf("read source media prefix: %w", prefixErr)
	}
	if prefixSize == 0 {
		return nil, ErrInvalidSourceMedia{Reason: "file is empty"}
	}
	kind, mediaType, err := sniffSourceMedia(prefix[:prefixSize])
	if err != nil {
		return nil, err
	}
	maxBytes := s.limits.MaxBytes
	if kind == MediaKindVideo {
		maxBytes = s.limits.MaxVideoBytes
	}
	if declaredSize > maxBytes {
		return nil, ErrSourceMediaTooLarge{Limit: maxBytes}
	}
	mediaID, objectID := strings.TrimSpace(s.newID()), strings.TrimSpace(s.newID())
	if mediaID == "" || objectID == "" {
		return nil, errors.New("source media id source returned an empty id")
	}
	key := "projects/" + projectID + "/media/" + objectID
	stream := &digestLimitReader{reader: io.MultiReader(bytes.NewReader(prefix[:prefixSize]), body), max: maxBytes, hash: sha256.New()}
	if err := s.store.Put(ctx, key, stream, mediaType); err != nil {
		cleanupErr := s.cleanupObject(ctx, key)
		return nil, errors.Join(fmt.Errorf("store source media bytes: %w", err), cleanupErr)
	}
	if !stream.eof {
		cleanupErr := s.cleanupObject(ctx, key)
		return nil, errors.Join(errors.New("source media store did not consume the complete input"), cleanupErr)
	}
	if stream.count == 0 {
		return nil, errors.Join(ErrInvalidSourceMedia{Reason: "file is empty"}, s.cleanupObject(ctx, key))
	}
	if stream.count > maxBytes {
		return nil, errors.Join(ErrSourceMediaTooLarge{Limit: maxBytes}, s.cleanupObject(ctx, key))
	}
	if declaredSize >= 0 && stream.count != declaredSize {
		return nil, errors.Join(ErrInvalidSourceMedia{Reason: "declared file size did not match bytes read"}, s.cleanupObject(ctx, key))
	}
	media := SourceMedia{
		ID: mediaID, ProjectID: projectID, SHA256: hex.EncodeToString(stream.hash.Sum(nil)),
		SizeBytes: stream.count, MediaType: mediaType, Kind: kind, OriginalFilename: filename,
		ImportedAt: s.now().UTC(), ImportMethod: "local_upload", storageKey: key,
	}
	if kind == MediaKindImage {
		data, err := s.readStagedImage(ctx, key, media.SizeBytes, media.SHA256)
		if err != nil {
			return nil, errors.Join(fmt.Errorf("verify staged image: %w", err), s.cleanupObject(ctx, key))
		}
		info, err := inspectSourceImage(data, SourceMediaLimits{MaxPixels: s.limits.MaxPixels, MaxSide: s.limits.MaxSide})
		if err != nil {
			return nil, errors.Join(err, s.cleanupObject(ctx, key))
		}
		media.Width, media.Height, media.RasterKnown = uint32(info.Width), uint32(info.Height), true
		media.Orientation, media.ColorModel, media.ColorProfileStatus = info.Orientation, info.ColorModel, info.ColorProfileStatus
		media.OrientationKnown, media.ColorModelKnown, media.ColorProfileKnown = true, true, true
		media.Decoder = "go-image/" + runtime.Version()
		media.Readiness = ReadinessReady
	} else {
		media.Readiness = ReadinessPending
		media.Decoder = "pending-inspection"
	}
	return &stagedMediaImport{service: s, media: media}, nil
}

type stagedMediaImport struct {
	service *sourceMediaService
	media   SourceMedia
	done    bool
}

func (s *stagedMediaImport) Media() SourceMedia { return s.media }

func (s *stagedMediaImport) Commit(ctx context.Context) (SourceMedia, error) {
	if s.done {
		return SourceMedia{}, errors.New("staged source media import is already finalized")
	}
	created, inserted, err := s.service.repo.CreateSourceMedia(ctx, s.media)
	if err != nil {
		s.done = true
		persistErr := fmt.Errorf("persist source media metadata: %w", err)
		readCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
		defer cancel()
		current, readErr := s.service.repo.GetSourceMediaByDigest(readCtx, s.media.ProjectID, s.media.SHA256)
		if readErr == nil {
			if current.storageKey == s.media.storageKey {
				return SourceMedia{}, errors.Join(persistErr, errors.New("metadata write outcome is uncertain; retaining its referenced object"))
			}
			return SourceMedia{}, errors.Join(persistErr, s.service.cleanupObject(ctx, s.media.storageKey))
		}
		if errors.Is(readErr, ErrSourceMediaNotFound{ID: s.media.SHA256}) {
			return SourceMedia{}, errors.Join(persistErr, s.service.cleanupObject(ctx, s.media.storageKey))
		}
		return SourceMedia{}, errors.Join(persistErr, fmt.Errorf("metadata write outcome is uncertain; retaining staged object after readback failed: %w", readErr))
	}
	if inserted {
		s.done = true
		return created, nil
	}
	if err := verifyStoredBlob(ctx, s.service.store, created.storageKey, created.SizeBytes, created.SHA256, max(s.service.limits.MaxBytes, s.service.limits.MaxVideoBytes)); err != nil {
		s.done = true
		return SourceMedia{}, errors.Join(fmt.Errorf("verify existing source media bytes: %w", err), s.service.cleanupObject(ctx, s.media.storageKey))
	}
	if created.SizeBytes != s.media.SizeBytes || created.SHA256 != s.media.SHA256 || created.Kind != s.media.Kind || created.MediaType != s.media.MediaType {
		s.done = true
		return SourceMedia{}, errors.Join(ErrSourceMediaDigestCollision{SHA256: s.media.SHA256}, s.service.cleanupObject(ctx, s.media.storageKey))
	}
	if err := s.service.cleanupObject(ctx, s.media.storageKey); err != nil {
		s.done = true
		return SourceMedia{}, fmt.Errorf("remove duplicate staging bytes: %w", err)
	}
	s.done = true
	return created, nil
}

func (s *stagedMediaImport) Abort(ctx context.Context) error {
	if s.done {
		return nil
	}
	s.done = true
	return s.service.cleanupObject(ctx, s.media.storageKey)
}

type digestLimitReader struct {
	reader io.Reader
	max    int64
	count  int64
	hash   hashWriter
	eof    bool
}

type hashWriter interface {
	Write([]byte) (int, error)
	Sum([]byte) []byte
}

func (r *digestLimitReader) Read(p []byte) (int, error) {
	if r.count > r.max {
		return 0, ErrSourceMediaTooLarge{Limit: r.max}
	}
	remaining := r.max + 1 - r.count
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}
	n, err := r.reader.Read(p)
	if n > 0 {
		r.count += int64(n)
		_, _ = r.hash.Write(p[:n])
	}
	if errors.Is(err, io.EOF) {
		r.eof = true
	}
	if r.count > r.max {
		return n, ErrSourceMediaTooLarge{Limit: r.max}
	}
	return n, err
}

func (s *sourceMediaService) readStagedImage(ctx context.Context, key string, size int64, digest string) ([]byte, error) {
	if size <= 0 || size > s.limits.MaxBytes {
		return nil, ErrSourceMediaTooLarge{Limit: s.limits.MaxBytes}
	}
	reader, _, err := s.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(reader, s.limits.MaxBytes+1))
	closeErr := reader.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return nil, err
	}
	if int64(len(data)) != size || digestBytes(data) != digest {
		return nil, errors.New("staged image size or digest does not match source bytes")
	}
	return data, nil
}

func verifyStoredBlob(ctx context.Context, store blobstore.BlobStore, key string, size int64, digest string, maxBytes int64) error {
	reader, _, err := store.Get(ctx, key)
	if err != nil {
		return err
	}
	if size <= 0 || size > maxBytes {
		_ = reader.Close()
		return ErrSourceMediaTooLarge{Limit: maxBytes}
	}
	h := sha256.New()
	written, err := io.Copy(h, io.LimitReader(reader, size+1))
	closeErr := reader.Close()
	err = errors.Join(err, closeErr)
	if err != nil {
		return err
	}
	if written != size || hex.EncodeToString(h.Sum(nil)) != digest {
		return errors.New("stored source media size or digest differs from metadata")
	}
	return nil
}

func sniffSourceMedia(prefix []byte) (MediaKind, string, error) {
	switch {
	case len(prefix) >= 8 && bytes.Equal(prefix[:8], []byte{137, 'P', 'N', 'G', 13, 10, 26, 10}):
		return MediaKindImage, "image/png", nil
	case len(prefix) >= 2 && prefix[0] == 0xff && prefix[1] == 0xd8:
		return MediaKindImage, "image/jpeg", nil
	case len(prefix) >= 12 && string(prefix[4:8]) == "ftyp":
		if string(prefix[8:12]) == "qt  " {
			return MediaKindVideo, "video/quicktime", nil
		}
		return MediaKindVideo, "video/mp4", nil
	case len(prefix) >= 4 && bytes.Equal(prefix[:4], []byte{0x1a, 0x45, 0xdf, 0xa3}):
		switch {
		case bytes.Contains(prefix, []byte("webm")):
			return MediaKindVideo, "video/webm", nil
		case bytes.Contains(prefix, []byte("matroska")):
			return MediaKindVideo, "video/x-matroska", nil
		default:
			return "", "", ErrInvalidSourceMedia{Reason: "EBML media is missing a supported WebM or Matroska doctype"}
		}
	default:
		return "", "", ErrInvalidSourceMedia{Reason: "only PNG, JPEG, MP4, QuickTime, WebM, and Matroska sources are supported"}
	}
}

func validateInspectionProjection(p InspectionProjection) error {
	if strings.TrimSpace(p.ProjectID) == "" || strings.TrimSpace(p.MediaID) == "" || strings.TrimSpace(p.JobID) == "" || strings.TrimSpace(p.AttemptID) == "" || !validSourceDigest(p.SourceSHA256) {
		return fmt.Errorf("%w: project, media, source digest, job and attempt are required", ErrInspectionInvalid)
	}
	if !validPathComponent(p.ProjectID) || !validPathComponent(p.MediaID) || !validPathComponent(p.JobID) || !validPathComponent(p.AttemptID) || strings.IndexFunc(p.ProjectID+p.MediaID+p.JobID+p.AttemptID, unicode.IsControl) >= 0 {
		return fmt.Errorf("%w: project, media or job identity is invalid", ErrInspectionInvalid)
	}
	if len(p.IssueCode) > 64 || len(p.IssueMessage) > 512 || strings.IndexFunc(p.IssueCode+p.IssueMessage, unicode.IsControl) >= 0 {
		return fmt.Errorf("%w: issue text exceeds limits or contains controls", ErrInspectionInvalid)
	}
	switch p.Readiness {
	case ReadinessReady:
		if p.Receipt == nil || p.IssueCode != "" || p.IssueMessage != "" || p.Receipt.Classification != "accepted" {
			return fmt.Errorf("%w: ready media requires a receipt and no issue", ErrInspectionInvalid)
		}
		if err := validateInspectionReceipt(*p.Receipt, p, ReadinessReady); err != nil {
			return err
		}
		if p.ReceiptStorageKey != InspectionReceiptObjectKey(p.ProjectID, p.MediaID, p.JobID, p.Receipt.ID) {
			return fmt.Errorf("%w: receipt object is outside its source namespace", ErrInspectionInvalid)
		}
	case ReadinessRejected:
		if p.IssueCode == "" || strings.TrimSpace(p.IssueMessage) == "" {
			return fmt.Errorf("%w: rejected media requires a typed issue", ErrInspectionInvalid)
		}
		if p.Receipt == nil {
			if !validInspectionExecutionIssueCode(p.IssueCode) || p.ReceiptStorageKey != "" {
				return fmt.Errorf("%w: receipt-free rejection requires a known execution outcome", ErrInspectionInvalid)
			}
			return nil
		}
		if p.Receipt.Classification != "rejected" {
			return fmt.Errorf("%w: rejected media requires a rejected receipt", ErrInspectionInvalid)
		}
		if p.Receipt.IssueCode != p.IssueCode || p.Receipt.IssueMessage != p.IssueMessage || !validInspectionIssueCode(p.IssueCode) {
			return fmt.Errorf("%w: rejected receipt issue differs from the projected issue", ErrInspectionInvalid)
		}
		if err := validateInspectionReceipt(*p.Receipt, p, ReadinessRejected); err != nil {
			return err
		}
		if p.ReceiptStorageKey != InspectionReceiptObjectKey(p.ProjectID, p.MediaID, p.JobID, p.Receipt.ID) {
			return fmt.Errorf("%w: receipt object is outside its source namespace", ErrInspectionInvalid)
		}
	case ReadinessReconciliationRequired:
		if p.IssueCode == "" || strings.TrimSpace(p.IssueMessage) == "" || p.Receipt != nil || p.ReceiptStorageKey != "" {
			return fmt.Errorf("%w: unresolved media requires an issue and no inspection receipt", ErrInspectionInvalid)
		}
	default:
		return fmt.Errorf("%w: only a known inspection result can be projected", ErrInspectionInvalid)
	}
	return nil
}

func validInspectionExecutionIssueCode(code string) bool {
	return code == "inspection_execution_failed" || code == "inspection_execution_cancelled"
}

func validPathComponent(value string) bool {
	return value != "" && value != "." && value != ".." && len(value) <= 128 && !strings.ContainsAny(value, "/\\")
}

func validAudioCodec(codec string) bool {
	if len(codec) == 0 || len(codec) > 64 {
		return false
	}
	for i, c := range codec {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || (i > 0 && (c == '_' || c == '-' || c == '.'))) {
			return false
		}
	}
	return true
}

func validateInspectionReceipt(r InspectionReceipt, p InspectionProjection, readiness Readiness) error {
	if readiness == ReadinessReady && (r.VideoStreamIndex == nil || *r.VideoStreamIndex < 0 || uint64(*r.VideoStreamIndex) >= uint64(r.StreamCount)) {
		return fmt.Errorf("%w: accepted receipt requires a selected video stream", ErrInspectionInvalid)
	}
	if r.Version == 1 && r.Audio != nil {
		return fmt.Errorf("%w: legacy receipt cannot contain audio facts", ErrInspectionInvalid)
	}
	if r.Version == 2 && readiness == ReadinessReady {
		if r.Audio == nil || r.Audio.Version != 1 || r.Audio.Streams == nil || len(r.Audio.Streams) > 7 {
			return fmt.Errorf("%w: accepted receipt requires versioned audio facts", ErrInspectionInvalid)
		}
		seen := make(map[uint32]bool)
		for _, stream := range r.Audio.Streams {
			if stream.OriginalStreamIndex >= r.StreamCount || seen[stream.OriginalStreamIndex] || (r.VideoStreamIndex != nil && int64(stream.OriginalStreamIndex) == *r.VideoStreamIndex) || !validAudioCodec(stream.Codec) {
				return fmt.Errorf("%w: invalid audio stream facts", ErrInspectionInvalid)
			}
			seen[stream.OriginalStreamIndex] = true
		}
	}
	if (r.Version != 1 && r.Version != 2) || strings.TrimSpace(r.ID) == "" || !validSourceDigest(r.SHA256) || r.SourceSHA256 != p.SourceSHA256 || r.JobID != p.JobID || r.SourceSizeBytes <= 0 || r.SourceSizeBytes > MaxSourceVideoBytes || r.AttemptID == "" || r.ToolProfileID == "" || r.InspectionProtocolVersion == 0 {
		return fmt.Errorf("%w: receipt is not bound to the source and job", ErrInspectionInvalid)
	}
	if !validPathComponent(r.ID) || (r.ClockArtifactRef != "" && !validPathComponent(r.ClockArtifactRef)) || strings.ContainsAny(r.AttemptID+r.ToolProfileID, "/\\\x00\r\n") || len(r.AttemptID) > 128 || len(r.ToolProfileID) > 128 || len(r.ClockArtifactRef) > 256 {
		return fmt.Errorf("%w: receipt identity is invalid", ErrInspectionInvalid)
	}
	if readiness == ReadinessRejected {
		if r.Classification != "rejected" || r.IssueCode == "" || strings.TrimSpace(r.IssueMessage) == "" || len(r.IssueCode) > 64 || len(r.IssueMessage) > 512 || strings.IndexFunc(r.IssueCode+r.IssueMessage, unicode.IsControl) >= 0 || hasAcceptedInspectionFacts(r) {
			return fmt.Errorf("%w: rejected receipt has an invalid issue or fabricated accepted facts", ErrInspectionInvalid)
		}
		return nil
	}
	if readiness != ReadinessReady || r.Classification != "accepted" || r.IssueCode != "" || r.IssueMessage != "" {
		return fmt.Errorf("%w: receipt classification does not match projected readiness", ErrInspectionInvalid)
	}
	if r.VideoWidth == 0 || r.VideoHeight == 0 || r.VideoWidth > 3840 || r.VideoHeight > 3840 || uint64(r.VideoWidth)*uint64(r.VideoHeight) > 3840*2160 {
		return fmt.Errorf("%w: receipt raster facts are outside the source envelope", ErrInspectionInvalid)
	}
	durationSeconds := int64(0)
	if r.DurationDenominator > 0 && r.DurationNumerator > 0 {
		durationSeconds = r.DurationNumerator / r.DurationDenominator
	}
	if r.DurationNumerator <= 0 || r.DurationDenominator <= 0 || durationSeconds > 600 || (durationSeconds == 600 && r.DurationNumerator%r.DurationDenominator != 0) || r.FrameCount == 0 || r.FrameCount > 36000 || r.StreamCount == 0 || r.StreamCount > 8 || !validSourceDigest(r.ClockSHA256) || r.ClockArtifactRef == "" {
		return fmt.Errorf("%w: receipt timing or stream facts are outside the source envelope", ErrInspectionInvalid)
	}
	switch r.RotationDegrees {
	case 0, 90, 180, 270, -90, -180, -270:
	default:
		return fmt.Errorf("%w: receipt rotation is unsupported", ErrInspectionInvalid)
	}
	return nil
}

func hasAcceptedInspectionFacts(r InspectionReceipt) bool {
	return r.Audio != nil || r.Container != "" || r.VideoStreamIndex != nil || r.Codec != "" || r.CodedWidth != 0 || r.CodedHeight != 0 || r.VideoWidth != 0 || r.VideoHeight != 0 || r.SampleAspectNum != 0 || r.SampleAspectDen != 0 || r.RotationDegrees != 0 || r.ColorPolicy != "" || r.ColorPrimaries != "" || r.ColorTransfer != "" || r.ColorMatrix != "" || r.ColorRange != "" || r.TimeBaseNumerator != 0 || r.TimeBaseDenominator != 0 || r.StreamStartPTS != nil || r.SourceEndPTS != nil || r.LastFrameDurationTicks != 0 || r.DurationNumerator != 0 || r.DurationDenominator != 0 || r.FrameCount != 0 || r.StreamCount != 0 || r.EOFPolicy != "" || r.ClockSHA256 != "" || r.ClockArtifactRef != "" || r.ClockArtifactSizeBytes != 0
}

func validInspectionIssueCode(code string) bool {
	switch code {
	case "audio_stream_unsupported", "stream_count_unsupported", "stream_index_invalid", "stream_index_ambiguous", "video_stream_selection_ambiguous", "video_raster_unsupported", "sample_aspect_ratio_unsupported", "rotation_unsupported", "color_profile_unsupported", "container_unsupported", "time_base_unavailable", "stream_start_unavailable", "frame_count_unsupported", "frame_pts_unavailable", "frame_pts_ambiguous", "source_eof_unavailable", "duration_unsupported", "decode_timestamp_mismatch":
		return true
	default:
		return false
	}
}

func validSourceDigest(digest string) bool {
	if len(digest) != 64 {
		return false
	}
	_, err := hex.DecodeString(digest)
	return err == nil
}

type verifiedMediaReader struct {
	reader       io.ReadCloser
	expectedSize int64
	expectedSHA  string
	maxBytes     int64
	count        int64
	hash         hashWriter
	checked      bool
	verifyErr    error
	finalByte    byte
	finalRead    bool
}

var ErrSourceMediaUnverified = errors.New("source media stream was closed before digest verification")

func (r *verifiedMediaReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.hash == nil {
		r.hash = sha256.New()
	}
	if r.verifyErr != nil {
		return 0, r.verifyErr
	}
	if r.checked {
		return 0, io.EOF
	}
	if !r.finalRead {
		remaining := r.expectedSize - r.count
		if remaining <= 0 {
			r.verifyErr = errors.New("source media size accounting is invalid")
			return 0, r.verifyErr
		}
		if remaining > 1 {
			if int64(len(p)) > remaining-1 {
				p = p[:remaining-1]
			}
			n, err := r.reader.Read(p)
			if n > 0 {
				r.count += int64(n)
				_, _ = r.hash.Write(p[:n])
			}
			if errors.Is(err, io.EOF) {
				r.verifyErr = errors.New("source media ended before its recorded size")
				return n, r.verifyErr
			}
			if err != nil {
				r.verifyErr = err
			}
			return n, err
		}
		var final [1]byte
		n, err := r.reader.Read(final[:])
		if n == 0 {
			if errors.Is(err, io.EOF) {
				r.verifyErr = errors.New("source media ended before its recorded size")
			} else if err != nil {
				r.verifyErr = err
			}
			return 0, r.verifyErrOr(err)
		}
		r.finalByte = final[0]
		r.finalRead = true
		r.count++
		_, _ = r.hash.Write(final[:])
		if err != nil && !errors.Is(err, io.EOF) {
			r.verifyErr = err
			return 0, r.verifyErr
		}
		if errors.Is(err, io.EOF) {
			return r.publishFinalByte(p)
		}
	}
	var extra [1]byte
	n, err := r.reader.Read(extra[:])
	if n > 0 {
		r.verifyErr = errors.New("source media contains more bytes than its recorded size")
		return 0, r.verifyErr
	}
	if err != nil && !errors.Is(err, io.EOF) {
		r.verifyErr = err
		return 0, err
	}
	if !errors.Is(err, io.EOF) {
		return 0, nil
	}
	return r.publishFinalByte(p)
}

func (r *verifiedMediaReader) publishFinalByte(p []byte) (int, error) {
	if r.count != r.expectedSize || r.count > r.maxBytes || hex.EncodeToString(r.hash.Sum(nil)) != r.expectedSHA {
		r.verifyErr = errors.New("source media bytes do not match stored digest and size")
		return 0, r.verifyErr
	}
	r.checked = true
	p[0] = r.finalByte
	return 1, nil
}

func (r *verifiedMediaReader) verifyErrOr(fallback error) error {
	if r.verifyErr != nil {
		return r.verifyErr
	}
	return fallback
}

func (r *verifiedMediaReader) Close() error {
	if !r.checked && r.verifyErr == nil {
		r.verifyErr = ErrSourceMediaUnverified
	}
	return errors.Join(r.verifyErr, r.reader.Close())
}

func digestBytes(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
