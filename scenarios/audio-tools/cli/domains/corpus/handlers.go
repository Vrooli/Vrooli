package corpus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	corpusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/corpus"
	corpusconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/corpus/corpus_v1connect"
)

type handlers struct {
	core      *cliapp.ScenarioApp
	client    corpusconnect.CorpusServiceClient
	readAudio func(string) ([]byte, error)
}

type sourceManifestRecord struct {
	CorpusRevision   string
	ManifestSHA256   string
	SourceID         string `json:"source_id"`
	SpeakerPseudonym string `json:"speaker_pseudonym"`
	AccentGroup      string `json:"accent_group"`
	Split            string `json:"split"`
	TranscriptSHA256 string `json:"transcript_sha256"`
	LicenseReference string `json:"license_reference"`
	Variant          string
	ContentSHA256    string
	MeasuredSNRDB    *float64
}

type sourceManifest struct {
	CorpusRevision string                 `json:"corpus_revision"`
	Records        []sourceManifestRecord `json:"records"`
	Imports        []sourceManifestImport `json:"imports"`
}

type sourceManifestImport struct {
	ClipID        string   `json:"clip_id"`
	SourceID      string   `json:"source_id"`
	Variant       string   `json:"variant"`
	ContentSHA256 string   `json:"content_sha256"`
	MeasuredSNRDB *float64 `json:"measured_snr_db"`
}

type lineageBackfillResult struct {
	DryRun           bool  `json:"dry_run"`
	WouldUpdateCount int32 `json:"would_update_count"`
	UpdatedCount     int32 `json:"updated_count"`
}

func lineageBackfillSummary(result lineageBackfillResult) string {
	if result.DryRun {
		return fmt.Sprintf("Dry run: %d clips would be updated; no rows changed.\n", result.WouldUpdateCount)
	}
	return fmt.Sprintf("Applied source lineage to %d clips.\n", result.UpdatedCount)
}

func readSourceManifest(path string) (*sourceManifest, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read source manifest: %w", err)
	}
	if len(b) > 4<<20 {
		return nil, "", fmt.Errorf("source manifest exceeds 4 MiB")
	}
	var manifest sourceManifest
	decoder := json.NewDecoder(strings.NewReader(string(b)))
	if err := decoder.Decode(&manifest); err != nil {
		return nil, "", fmt.Errorf("decode source manifest: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, "", fmt.Errorf("source manifest must contain one JSON value")
	}
	if strings.TrimSpace(manifest.CorpusRevision) == "" {
		return nil, "", fmt.Errorf("source manifest requires corpus_revision")
	}
	sum := sha256.Sum256(b)
	return &manifest, hex.EncodeToString(sum[:]), nil
}

func loadSourceManifestRecord(path, sourceID string) (*sourceManifestRecord, error) {
	return resolveSourceManifestRecord(path, sourceID, "")
}

func resolveSourceManifestRecord(path, sourceID, contentSHA256 string) (*sourceManifestRecord, error) {
	path, sourceID = strings.TrimSpace(path), strings.TrimSpace(sourceID)
	if path == "" && sourceID == "" {
		return nil, nil
	}
	if path == "" || sourceID == "" {
		return nil, fmt.Errorf("--source-manifest and --source-id must be supplied together")
	}
	manifest, manifestSHA256, err := readSourceManifest(path)
	if err != nil {
		return nil, err
	}
	var found *sourceManifestRecord
	for i := range manifest.Records {
		if manifest.Records[i].SourceID != sourceID {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("source manifest contains duplicate source id %q", sourceID)
		}
		r := manifest.Records[i]
		if strings.TrimSpace(r.SpeakerPseudonym) == "" || strings.TrimSpace(r.AccentGroup) == "" || strings.TrimSpace(r.LicenseReference) == "" || len(r.TranscriptSHA256) != 64 {
			return nil, fmt.Errorf("source manifest record %q lacks required lineage fields", sourceID)
		}
		r.CorpusRevision = manifest.CorpusRevision
		r.ManifestSHA256 = manifestSHA256
		if contentSHA256 != "" {
			matches := 0
			for _, imported := range manifest.Imports {
				if imported.SourceID == sourceID && imported.ContentSHA256 == contentSHA256 {
					matches++
					r.Variant, r.ContentSHA256, r.MeasuredSNRDB = imported.Variant, imported.ContentSHA256, imported.MeasuredSNRDB
				}
			}
			if matches != 1 {
				return nil, fmt.Errorf("source manifest must contain exactly one import for source %q and content digest %s; found %d", sourceID, contentSHA256, matches)
			}
		}
		found = &r
	}
	if found == nil {
		return nil, fmt.Errorf("source id %q was not found in source manifest", sourceID)
	}
	return found, nil
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:      core,
		client:    corpusconnect.NewCorpusServiceClient(httpClient, baseURL),
		readAudio: os.ReadFile,
	}
}

func (h *handlers) listPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(ctx cliapp.OperationContext) (*corpusv1.ListClipsResponse, error) {
		limit := 0
		if v := strings.TrimSpace(ctx.Flag("limit")); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("--limit must be an integer: %q", v)
			}
			limit = n
		}
		resp, err := h.client.ListClips(context.Background(), connect.NewRequest(&corpusv1.ListClipsRequest{
			TagContains: ctx.Flag("tag"),
			Limit:       int32(limit),
		}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list-clips", err, nil)
		}
		return resp.Msg, nil
	}, renderList, func(ctx cliapp.RunContext, resp *corpusv1.ListClipsResponse) error {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp)
	})
}

func renderList(ctx cliapp.RunContext, resp *corpusv1.ListClipsResponse) error {
	clips := resp.GetClips()
	if len(clips) == 0 {
		fmt.Fprintln(ctx.Stdout(), "No clips in the corpus. Record some in the Dictation Studio UI or `corpus import` a PCM file.")
		return nil
	}
	fmt.Fprintf(ctx.Stdout(), "%-38s  %-9s  %-7s  %-8s  %s\n", "ID", "SOURCE", "DUR_MS", "TAGS", "REFERENCE")
	for _, c := range clips {
		ref := c.GetReferenceText()
		if len(ref) > 48 {
			ref = ref[:45] + "..."
		}
		fmt.Fprintf(ctx.Stdout(), "%-38s  %-9s  %-7d  %-8s  %s\n",
			c.GetId(), sourceLabel(c.GetSource()), c.GetDurationMs(), strings.Join(c.GetTags(), ","), ref)
	}
	return nil
}

func (h *handlers) importPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ActionCustomRenderers(func(ctx cliapp.OperationContext) (importResult, error) {
		audioPath := ctx.Positional("audio-file")
		if audioPath == "" {
			return importResult{}, fmt.Errorf("audio-file is required")
		}
		audio, err := h.readAudio(audioPath)
		if err != nil {
			return importResult{}, fmt.Errorf("read audio file: %w", err)
		}
		reference := ctx.Flag("reference")
		if rf := strings.TrimSpace(ctx.Flag("reference-file")); rf != "" {
			b, err := os.ReadFile(rf)
			if err != nil {
				return importResult{}, fmt.Errorf("read reference file: %w", err)
			}
			reference = string(b)
		}
		reference = strings.TrimSpace(reference)
		if reference == "" {
			return importResult{}, fmt.Errorf("a reference transcript is required (--reference or --reference-file)")
		}
		audioSum := sha256.Sum256(audio)
		manifestSource, err := resolveSourceManifestRecord(ctx.Flag("source-manifest"), ctx.Flag("source-id"), hex.EncodeToString(audioSum[:]))
		if err != nil {
			return importResult{}, err
		}
		var timedReference *corpusv1.TimedReference
		if path := strings.TrimSpace(ctx.Flag("timed-reference-file")); path != "" {
			file, err := os.Open(path)
			if err != nil {
				return importResult{}, fmt.Errorf("read timed reference file: %w", err)
			}
			b, readErr := io.ReadAll(io.LimitReader(file, (4<<20)+1))
			closeErr := file.Close()
			if readErr != nil {
				return importResult{}, fmt.Errorf("read timed reference file: %w", readErr)
			}
			if closeErr != nil {
				return importResult{}, fmt.Errorf("close timed reference file: %w", closeErr)
			}
			timedReference, err = parseTimedReference(b)
			if err != nil {
				return importResult{}, err
			}
		}
		var referenceReview *corpusv1.ReferenceReview
		if path := strings.TrimSpace(ctx.Flag("reference-review-file")); path != "" {
			b, err := os.ReadFile(path)
			if err != nil {
				return importResult{}, fmt.Errorf("read reference review file: %w", err)
			}
			referenceReview, err = parseReferenceReview(b)
			if err != nil {
				return importResult{}, err
			}
		}

		format := strings.TrimSpace(ctx.Flag("format"))
		if format == "" {
			format = "pcm_s16le"
		}
		sampleRate := 16000
		if v := strings.TrimSpace(ctx.Flag("sample-rate")); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				return importResult{}, fmt.Errorf("--sample-rate must be an integer: %q", v)
			}
			sampleRate = n
		}
		source := sourceFromContext(ctx)
		split := corpusv1.CorpusSplit_CORPUS_SPLIT_UNSPECIFIED
		switch strings.TrimSpace(ctx.Flag("split")) {
		case "", "unspecified":
		case "dev":
			split = corpusv1.CorpusSplit_CORPUS_SPLIT_DEV
		case "held_out":
			split = corpusv1.CorpusSplit_CORPUS_SPLIT_HELD_OUT
		default:
			return importResult{}, fmt.Errorf("--split must be dev or held_out")
		}
		noise := corpusv1.NoiseCondition_NOISE_CONDITION_UNSPECIFIED
		switch strings.TrimSpace(ctx.Flag("noise-condition")) {
		case "", "unspecified":
		case "clean":
			noise = corpusv1.NoiseCondition_NOISE_CONDITION_CLEAN
		case "noisy":
			noise = corpusv1.NoiseCondition_NOISE_CONDITION_NOISY
		default:
			return importResult{}, fmt.Errorf("--noise-condition must be clean or noisy")
		}
		var snrDB float64
		if v := strings.TrimSpace(ctx.Flag("snr-db")); v != "" {
			snrDB, err = strconv.ParseFloat(v, 64)
			if err != nil || snrDB < 0 {
				return importResult{}, fmt.Errorf("--snr-db must be a non-negative number: %q", v)
			}
		}
		var tags []string
		if t := strings.TrimSpace(ctx.Flag("tags")); t != "" {
			for _, tag := range strings.Split(t, ",") {
				if s := strings.TrimSpace(tag); s != "" {
					tags = append(tags, s)
				}
			}
		}
		corpusRevision := strings.TrimSpace(ctx.Flag("corpus-revision"))
		licenseReference := strings.TrimSpace(ctx.Flag("license-reference"))
		speakerPseudonym := strings.TrimSpace(ctx.Flag("speaker-pseudonym"))
		if manifestSource != nil {
			sum := sha256.Sum256([]byte(reference))
			transcriptDigest := hex.EncodeToString(sum[:])
			if transcriptDigest != manifestSource.TranscriptSHA256 {
				return importResult{}, fmt.Errorf("reference transcript digest does not match source manifest record %q", manifestSource.SourceID)
			}
			corpusRevision, licenseReference, speakerPseudonym = manifestSource.CorpusRevision, manifestSource.LicenseReference, manifestSource.SpeakerPseudonym
			switch manifestSource.Split {
			case "dev":
				split = corpusv1.CorpusSplit_CORPUS_SPLIT_DEV
			case "held_out":
				split = corpusv1.CorpusSplit_CORPUS_SPLIT_HELD_OUT
			default:
				return importResult{}, fmt.Errorf("source manifest has unsupported split %q", manifestSource.Split)
			}
			switch manifestSource.Variant {
			case "clean", "dev_clean":
				if (manifestSource.Variant == "dev_clean") != (manifestSource.Split == "dev") {
					return importResult{}, fmt.Errorf("source manifest variant %q conflicts with source split %q", manifestSource.Variant, manifestSource.Split)
				}
				noise = corpusv1.NoiseCondition_NOISE_CONDITION_CLEAN
				snrDB = 0
			case "noisy_10db", "noisy_20db":
				if manifestSource.Split != "held_out" {
					return importResult{}, fmt.Errorf("noisy source variants must be held_out")
				}
				if manifestSource.MeasuredSNRDB == nil {
					return importResult{}, fmt.Errorf("manifest import %q lacks measured SNR", manifestSource.Variant)
				}
				wantSNR := 10.0
				if manifestSource.Variant == "noisy_20db" {
					wantSNR = 20.0
				}
				if math.IsNaN(*manifestSource.MeasuredSNRDB) || math.IsInf(*manifestSource.MeasuredSNRDB, 0) || math.Abs(*manifestSource.MeasuredSNRDB-wantSNR) > 0.01 {
					return importResult{}, fmt.Errorf("manifest import %q has inconsistent measured SNR", manifestSource.Variant)
				}
				noise = corpusv1.NoiseCondition_NOISE_CONDITION_NOISY
				snrDB = *manifestSource.MeasuredSNRDB
			case "":
				return importResult{}, fmt.Errorf("source manifest import variant is required")
			default:
				return importResult{}, fmt.Errorf("source manifest has unsupported import variant %q", manifestSource.Variant)
			}
		}
		// Derive duration from PCM byte length for s16le mono.
		var durationMs int64
		if format == "pcm_s16le" || format == "pcm" {
			durationMs = int64(len(audio)) * 1000 / int64(sampleRate*2)
		}

		resp, err := h.client.CreateClip(context.Background(), connect.NewRequest(&corpusv1.CreateClipRequest{
			Audio:            audio,
			ReferenceText:    reference,
			Tags:             tags,
			DurationMs:       durationMs,
			SampleRateHz:     int32(sampleRate),
			Format:           format,
			Source:           source,
			CorpusRevision:   corpusRevision,
			LicenseReference: licenseReference,
			SpeakerPseudonym: speakerPseudonym,
			Split:            split,
			NoiseCondition:   noise,
			SnrDb:            snrDB,
			TimedReference:   timedReference,
			ReferenceReview:  referenceReview,
		}))
		if err != nil {
			return importResult{}, cliapp.WrapAPIError("create-clip", err, nil)
		}
		if manifestSource != nil {
			createdClip := resp.Msg.GetClip()
			if createdClip == nil {
				return importResult{}, fmt.Errorf("create-clip response is missing clip metadata; lineage was not applied")
			}
			if createdClip.GetContentSha256() != manifestSource.ContentSHA256 {
				return importResult{}, fmt.Errorf("created clip content digest differs from selected source-manifest import; clip remains ungrouped")
			}
			_, err := h.client.BackfillSourceLineage(context.Background(), connect.NewRequest(&corpusv1.BackfillSourceLineageRequest{Entries: []*corpusv1.SourceLineageBackfillEntry{{ClipId: createdClip.GetId(), CorpusRevision: manifestSource.CorpusRevision, ContentSha256: createdClip.GetContentSha256(), ManifestTranscriptSha256: manifestSource.TranscriptSHA256, SourceId: manifestSource.SourceID, SpeakerPseudonym: manifestSource.SpeakerPseudonym, AccentGroup: manifestSource.AccentGroup, Split: split, LicenseReference: manifestSource.LicenseReference, ManifestSha256: manifestSource.ManifestSHA256}}}))
			if err != nil {
				return importResult{}, cliapp.WrapAPIError("backfill-source-lineage", err, nil)
			}
			updated, getErr := h.client.GetClip(context.Background(), connect.NewRequest(&corpusv1.GetClipRequest{Id: createdClip.GetId()}))
			if getErr != nil {
				return importResult{}, cliapp.WrapAPIError("get-lineage-clip", getErr, nil)
			}
			resp.Msg.Clip = updated.Msg.GetClip()
		}
		return importResult{response: resp.Msg, audioBytes: len(audio)}, nil
	}, renderImport, renderImportJSON)
}

func (h *handlers) backfillLineagePrimitive() cliapp.PrimitiveHandler {
	return cliapp.ActionCustomRenderers(func(ctx cliapp.OperationContext) (lineageBackfillResult, error) {
		path := strings.TrimSpace(ctx.Flag("source-manifest"))
		if path == "" {
			return lineageBackfillResult{}, fmt.Errorf("--source-manifest is required")
		}
		manifest, digest, err := readSourceManifest(path)
		if err != nil {
			return lineageBackfillResult{}, err
		}
		records := make(map[string]sourceManifestRecord, len(manifest.Records))
		for _, record := range manifest.Records {
			if _, exists := records[record.SourceID]; exists {
				return lineageBackfillResult{}, fmt.Errorf("source manifest contains duplicate source id %q", record.SourceID)
			}
			record.CorpusRevision = manifest.CorpusRevision
			record.ManifestSHA256 = digest
			records[record.SourceID] = record
		}
		entries := make([]*corpusv1.SourceLineageBackfillEntry, 0, len(manifest.Imports))
		seen := map[string]struct{}{}
		for _, imported := range manifest.Imports {
			record, ok := records[imported.SourceID]
			if !ok {
				return lineageBackfillResult{}, fmt.Errorf("import %q references unknown source id %q", imported.ClipID, imported.SourceID)
			}
			if imported.ClipID == "" || len(imported.ContentSHA256) != 64 {
				return lineageBackfillResult{}, fmt.Errorf("import for source %q lacks clip id or content digest", imported.SourceID)
			}
			if _, ok := seen[imported.ClipID]; ok {
				return lineageBackfillResult{}, fmt.Errorf("source manifest has duplicate clip id %q", imported.ClipID)
			}
			seen[imported.ClipID] = struct{}{}
			split := corpusv1.CorpusSplit_CORPUS_SPLIT_UNSPECIFIED
			if record.Split == "held_out" {
				split = corpusv1.CorpusSplit_CORPUS_SPLIT_HELD_OUT
			} else if record.Split == "dev" {
				split = corpusv1.CorpusSplit_CORPUS_SPLIT_DEV
			} else {
				return lineageBackfillResult{}, fmt.Errorf("source manifest has unsupported split %q", record.Split)
			}
			entries = append(entries, &corpusv1.SourceLineageBackfillEntry{ClipId: imported.ClipID, CorpusRevision: record.CorpusRevision, ContentSha256: imported.ContentSHA256, ManifestTranscriptSha256: record.TranscriptSHA256, SourceId: record.SourceID, SpeakerPseudonym: record.SpeakerPseudonym, AccentGroup: record.AccentGroup, Split: split, LicenseReference: record.LicenseReference, ManifestSha256: digest})
		}
		if len(entries) == 0 {
			return lineageBackfillResult{}, fmt.Errorf("source manifest has no clip imports")
		}
		resp, err := h.client.BackfillSourceLineage(context.Background(), connect.NewRequest(&corpusv1.BackfillSourceLineageRequest{Entries: entries, DryRun: !ctx.BoolFlag("apply")}))
		if err != nil {
			return lineageBackfillResult{}, cliapp.WrapAPIError("backfill-source-lineage", err, nil)
		}
		result := lineageBackfillResult{DryRun: !ctx.BoolFlag("apply")}
		if result.DryRun {
			result.WouldUpdateCount = resp.Msg.GetWouldUpdateCount()
		} else {
			result.UpdatedCount = resp.Msg.GetUpdatedCount()
		}
		return result, nil
	}, func(ctx cliapp.RunContext, result lineageBackfillResult) error {
		_, err := fmt.Fprint(ctx.Stdout(), lineageBackfillSummary(result))
		return err
	}, func(ctx cliapp.RunContext, result lineageBackfillResult) error {
		return json.NewEncoder(ctx.Stdout()).Encode(result)
	})
}

func parseReferenceReview(b []byte) (*corpusv1.ReferenceReview, error) {
	if len(b) > 16<<10 {
		return nil, fmt.Errorf("reference review file exceeds 16 KiB")
	}
	var input struct {
		Reviewer   string `json:"reviewer"`
		Protocol   string `json:"protocol"`
		ReviewedAt string `json:"reviewed_at"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(b)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return nil, fmt.Errorf("decode reference review JSON: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("reference review file must contain one JSON value")
	}
	if strings.TrimSpace(input.Reviewer) == "" || strings.TrimSpace(input.Protocol) == "" || strings.TrimSpace(input.ReviewedAt) == "" {
		return nil, fmt.Errorf("reference review requires reviewer, protocol, and reviewed_at")
	}
	return &corpusv1.ReferenceReview{Reviewer: input.Reviewer, Protocol: input.Protocol, ReviewedAt: input.ReviewedAt}, nil
}

func sourceFromContext(ctx cliapp.OperationContext) corpusv1.ClipSource {
	if ctx.BoolFlag("scripted") {
		return corpusv1.ClipSource_CLIP_SOURCE_SCRIPTED
	}
	return corpusv1.ClipSource_CLIP_SOURCE_FREE_FORM
}

func parseTimedReference(b []byte) (*corpusv1.TimedReference, error) {
	if len(b) > 4<<20 {
		return nil, fmt.Errorf("timed reference file exceeds 4 MiB")
	}
	var input timedReferenceFile
	decoder := json.NewDecoder(strings.NewReader(string(b)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return nil, fmt.Errorf("decode timed reference JSON: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("timed reference file must contain one JSON value")
	}
	if len(input.Words) == 0 || len(input.Words) > 20_000 {
		return nil, fmt.Errorf("timed reference requires 1..20000 words")
	}
	words := make([]*corpusv1.TimedWord, len(input.Words))
	for i, word := range input.Words {
		words[i] = &corpusv1.TimedWord{Word: word.Word, StartSeconds: word.StartSeconds, EndSeconds: word.EndSeconds}
	}
	p := input.Provenance
	return &corpusv1.TimedReference{Words: words, Provenance: &corpusv1.TimingProvenance{Status: p.Status, SourceReference: p.SourceReference, AlignmentTool: p.AlignmentTool, AlignmentModel: p.AlignmentModel, Reviewer: p.Reviewer, ReviewMethod: p.ReviewMethod, ReviewedAt: p.ReviewedAt, Notes: p.Notes}}, nil
}

type timedReferenceFile struct {
	Words []struct {
		Word         string  `json:"word"`
		StartSeconds float64 `json:"start_seconds"`
		EndSeconds   float64 `json:"end_seconds"`
	} `json:"words"`
	Provenance struct {
		Status          string `json:"status"`
		SourceReference string `json:"source_reference"`
		AlignmentTool   string `json:"alignment_tool"`
		AlignmentModel  string `json:"alignment_model"`
		Reviewer        string `json:"reviewer"`
		ReviewMethod    string `json:"review_method"`
		ReviewedAt      string `json:"reviewed_at"`
		Notes           string `json:"notes"`
	} `json:"provenance"`
}

type importResult struct {
	response   *corpusv1.CreateClipResponse
	audioBytes int
}

func renderImport(ctx cliapp.RunContext, result importResult) error {
	resp := result.response
	fmt.Fprintf(ctx.Stdout(), "Imported clip %s (%d bytes, %dms, %s).\n",
		resp.GetClip().GetId(), result.audioBytes, resp.GetClip().GetDurationMs(), resp.GetClip().GetFormat())
	return nil
}

func renderImportJSON(ctx cliapp.RunContext, result importResult) error {
	return cliapp.PrintProtoJSON(ctx.Stdout(), result.response)
}

func (h *handlers) getPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(ctx cliapp.OperationContext) (*corpusv1.GetClipResponse, error) {
		id := ctx.Positional("id")
		resp, err := h.client.GetClip(context.Background(), connect.NewRequest(&corpusv1.GetClipRequest{Id: id}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get-clip", err, nil)
		}
		return resp.Msg, nil
	}, renderGet, renderGet)
}

func renderGet(ctx cliapp.RunContext, resp *corpusv1.GetClipResponse) error {
	c := resp.GetClip()
	fmt.Fprintf(ctx.Stdout(), "id             = %s\n", c.GetId())
	fmt.Fprintf(ctx.Stdout(), "source         = %s\n", sourceLabel(c.GetSource()))
	fmt.Fprintf(ctx.Stdout(), "duration_ms    = %d\n", c.GetDurationMs())
	fmt.Fprintf(ctx.Stdout(), "sample_rate_hz = %d\n", c.GetSampleRateHz())
	fmt.Fprintf(ctx.Stdout(), "format         = %s\n", c.GetFormat())
	fmt.Fprintf(ctx.Stdout(), "tags           = %s\n", strings.Join(c.GetTags(), ", "))
	fmt.Fprintf(ctx.Stdout(), "blob_key       = %s\n", c.GetBlobKey())
	fmt.Fprintf(ctx.Stdout(), "reference      = %s\n", c.GetReferenceText())
	if timed := c.GetTimedReference(); timed != nil {
		p := timed.GetProvenance()
		fmt.Fprintf(ctx.Stdout(), "timed_words    = %d\n", len(timed.GetWords()))
		fmt.Fprintf(ctx.Stdout(), "timing_status  = %s\n", p.GetStatus())
		fmt.Fprintf(ctx.Stdout(), "timing_source  = %s\n", p.GetSourceReference())
		fmt.Fprintf(ctx.Stdout(), "alignment      = %s / %s\n", p.GetAlignmentTool(), p.GetAlignmentModel())
		if p.GetStatus() == "human_reviewed" {
			fmt.Fprintf(ctx.Stdout(), "reviewer       = %s\n", p.GetReviewer())
			fmt.Fprintf(ctx.Stdout(), "review_method  = %s\n", p.GetReviewMethod())
			fmt.Fprintf(ctx.Stdout(), "reviewed_at    = %s\n", p.GetReviewedAt())
		}
	}
	return nil
}

func (h *handlers) deletePrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(ctx cliapp.OperationContext) (*corpusv1.DeleteClipResponse, error) {
		id := ctx.Positional("id")
		resp, err := h.client.DeleteClip(context.Background(), connect.NewRequest(&corpusv1.DeleteClipRequest{Id: id}))
		if err != nil {
			return nil, cliapp.WrapAPIError("delete-clip", err, nil)
		}
		return resp.Msg, nil
	}, func(ctx cliapp.RunContext, resp *corpusv1.DeleteClipResponse) error {
		fmt.Fprintf(ctx.Stdout(), "Deleted clip %s.\n", ctx.Positional("id"))
		return nil
	}, func(ctx cliapp.RunContext, resp *corpusv1.DeleteClipResponse) error {
		fmt.Fprintf(ctx.Stdout(), "Deleted clip %s.\n", ctx.Positional("id"))
		return nil
	})
}

func sourceLabel(s corpusv1.ClipSource) string {
	switch s {
	case corpusv1.ClipSource_CLIP_SOURCE_SCRIPTED:
		return "scripted"
	default:
		return "free_form"
	}
}
