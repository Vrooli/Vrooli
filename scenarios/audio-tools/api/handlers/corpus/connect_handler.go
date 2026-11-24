//go:build audio_lab

package corpus

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"connectrpc.com/connect"

	intcorpus "audio-tools/internal/corpus"

	corpusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/corpus"
)

type connectHandler struct{ deps Deps }

// NewConnectHandler builds the CorpusService Connect handler. Deps.Logger
// and Deps.Clock are required seams; nil values panic.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		panic("corpus.NewConnectHandler requires Deps.Logger")
	}
	if d.Clock == nil {
		panic("corpus.NewConnectHandler requires Deps.Clock")
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) service() (*intcorpus.Service, error) {
	if h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("corpus service not configured (no database)"))
	}
	return h.deps.Service, nil
}

func (h *connectHandler) CreateClip(ctx context.Context, req *connect.Request[corpusv1.CreateClipRequest]) (*connect.Response[corpusv1.CreateClipResponse], error) {
	svc, err := h.service()
	if err != nil {
		return nil, err
	}
	m := req.Msg
	if len(m.GetAudio()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("audio bytes are required"))
	}
	if timed := m.GetTimedReference(); timed != nil && len(timed.GetWords()) > intcorpus.MaxTimedReferenceWords {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("timed reference exceeds maximum word count"))
	}
	timedReference := timedReferenceFromProto(m.GetTimedReference())
	if err := timedReference.Validate(m.GetDurationMs()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if timedReference != nil {
		encoded, err := json.Marshal(timedReference)
		if err != nil || len(encoded) > intcorpus.MaxTimedReferenceBytes {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("timed reference exceeds encoded size limit"))
		}
	}
	clip, err := svc.CreateClip(ctx, intcorpus.CreateClipInput{
		Audio:            m.GetAudio(),
		ReferenceText:    m.GetReferenceText(),
		Tags:             m.GetTags(),
		DurationMs:       m.GetDurationMs(),
		SampleRateHz:     int(m.GetSampleRateHz()),
		Format:           m.GetFormat(),
		Source:           sourceFromProto(m.GetSource()),
		CorpusRevision:   m.GetCorpusRevision(),
		LicenseReference: m.GetLicenseReference(),
		SpeakerPseudonym: m.GetSpeakerPseudonym(),
		Split:            splitFromProto(m.GetSplit()),
		NoiseCondition:   noiseConditionFromProto(m.GetNoiseCondition()),
		SNRDB:            m.GetSnrDb(),
		TimedReference:   timedReference,
		ReferenceReview:  referenceReviewFromProto(m.GetReferenceReview()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&corpusv1.CreateClipResponse{Clip: clipToProto(clip)}), nil
}

func (h *connectHandler) ReviewReferenceGroup(ctx context.Context, req *connect.Request[corpusv1.ReviewReferenceGroupRequest]) (*connect.Response[corpusv1.ReviewReferenceGroupResponse], error) {
	svc, err := h.service()
	if err != nil {
		return nil, err
	}
	m := req.Msg
	if strings.TrimSpace(m.GetReviewer()) == "" || strings.TrimSpace(m.GetProtocol()) == "" || len(m.GetMembers()) < 2 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("group review requires at least two variants, reviewer, and protocol"))
	}
	members := make([]intcorpus.ReviewMember, 0, len(m.GetMembers()))
	for _, member := range m.GetMembers() {
		if member == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("group review member is required"))
		}
		members = append(members, intcorpus.ReviewMember{ID: member.GetId(), ExpectedReferenceSHA256: member.GetExpectedReferenceSha256()})
	}
	clips, err := svc.ReviewReferenceGroup(ctx, members, m.GetReferenceText(), m.GetReviewer(), m.GetProtocol())
	if err != nil {
		var stale intcorpus.ErrStaleReference
		if errors.As(err, &stale) {
			return nil, connect.NewError(connect.CodeAborted, err)
		}
		var missing intcorpus.ErrClipNotFound
		if errors.As(err, &missing) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	out := &corpusv1.ReviewReferenceGroupResponse{Clips: make([]*corpusv1.Clip, 0, len(clips)), AffectedCount: int32(len(clips))}
	for _, clip := range clips {
		out.Clips = append(out.Clips, clipToProto(clip))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) BackfillSourceLineage(ctx context.Context, req *connect.Request[corpusv1.BackfillSourceLineageRequest]) (*connect.Response[corpusv1.BackfillSourceLineageResponse], error) {
	svc, err := h.service()
	if err != nil {
		return nil, err
	}
	entries := make([]intcorpus.LineageEntry, 0, len(req.Msg.GetEntries()))
	for _, entry := range req.Msg.GetEntries() {
		if entry == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("lineage backfill entry is required"))
		}
		entries = append(entries, intcorpus.LineageEntry{ClipID: entry.GetClipId(), CorpusRevision: entry.GetCorpusRevision(), ContentSHA256: entry.GetContentSha256(), ManifestTranscriptSHA256: entry.GetManifestTranscriptSha256(), SourceID: entry.GetSourceId(), SpeakerPseudonym: entry.GetSpeakerPseudonym(), AccentGroup: entry.GetAccentGroup(), Split: string(splitFromProto(entry.GetSplit())), LicenseReference: entry.GetLicenseReference(), ManifestSHA256: entry.GetManifestSha256()})
	}
	count, err := svc.BackfillSourceLineage(ctx, entries, req.Msg.GetDryRun())
	if err != nil {
		var missing intcorpus.ErrClipNotFound
		if errors.As(err, &missing) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	response := &corpusv1.BackfillSourceLineageResponse{}
	if req.Msg.GetDryRun() {
		response.WouldUpdateCount = int32(count)
	} else {
		response.UpdatedCount = int32(count)
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) ListClips(ctx context.Context, req *connect.Request[corpusv1.ListClipsRequest]) (*connect.Response[corpusv1.ListClipsResponse], error) {
	svc, err := h.service()
	if err != nil {
		return nil, err
	}
	m := req.Msg
	clips, err := svc.ListClips(ctx, intcorpus.ListFilter{
		TagContains: m.GetTagContains(),
		Limit:       int(m.GetLimit()),
		Offset:      int(m.GetOffset()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &corpusv1.ListClipsResponse{Clips: make([]*corpusv1.Clip, 0, len(clips))}
	for _, c := range clips {
		out.Clips = append(out.Clips, clipToProto(c))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) GetClip(ctx context.Context, req *connect.Request[corpusv1.GetClipRequest]) (*connect.Response[corpusv1.GetClipResponse], error) {
	svc, err := h.service()
	if err != nil {
		return nil, err
	}
	clip, err := svc.GetClip(ctx, req.Msg.GetId())
	if err != nil {
		return nil, mapClipError(err)
	}
	return connect.NewResponse(&corpusv1.GetClipResponse{Clip: clipToProto(clip)}), nil
}

func (h *connectHandler) GetClipAudio(ctx context.Context, req *connect.Request[corpusv1.GetClipAudioRequest]) (*connect.Response[corpusv1.GetClipAudioResponse], error) {
	svc, err := h.service()
	if err != nil {
		return nil, err
	}
	audio, clip, err := svc.GetClipAudio(ctx, req.Msg.GetId())
	if err != nil {
		return nil, mapClipError(err)
	}
	return connect.NewResponse(&corpusv1.GetClipAudioResponse{Audio: audio, Clip: clipToProto(clip)}), nil
}

func (h *connectHandler) DeleteClip(ctx context.Context, req *connect.Request[corpusv1.DeleteClipRequest]) (*connect.Response[corpusv1.DeleteClipResponse], error) {
	svc, err := h.service()
	if err != nil {
		return nil, err
	}
	if err := svc.DeleteClip(ctx, req.Msg.GetId()); err != nil {
		return nil, mapClipError(err)
	}
	return connect.NewResponse(&corpusv1.DeleteClipResponse{}), nil
}

func (h *connectHandler) ReviewReference(ctx context.Context, req *connect.Request[corpusv1.ReviewReferenceRequest]) (*connect.Response[corpusv1.ReviewReferenceResponse], error) {
	svc, err := h.service()
	if err != nil {
		return nil, err
	}
	m := req.Msg
	if strings.TrimSpace(m.GetReviewer()) == "" || strings.TrimSpace(m.GetProtocol()) == "" || strings.TrimSpace(m.GetExpectedReferenceSha256()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("expected digest, reviewer, and protocol are required"))
	}
	clip, err := svc.ReviewReference(ctx, m.GetId(), m.GetExpectedReferenceSha256(), m.GetReferenceText(), m.GetReviewer(), m.GetProtocol())
	if err != nil {
		var stale intcorpus.ErrStaleReference
		if errors.As(err, &stale) {
			return nil, connect.NewError(connect.CodeAborted, err)
		}
		var missing intcorpus.ErrClipNotFound
		if errors.As(err, &missing) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&corpusv1.ReviewReferenceResponse{Clip: clipToProto(clip)}), nil
}

// mapClipError maps a not-found to connect.CodeNotFound; everything else is
// Internal.
func mapClipError(err error) error {
	var nf intcorpus.ErrClipNotFound
	if errors.As(err, &nf) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
