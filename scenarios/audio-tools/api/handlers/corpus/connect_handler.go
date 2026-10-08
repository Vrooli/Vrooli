package corpus

import (
	"context"
	"errors"

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
	clip, err := svc.CreateClip(ctx, intcorpus.CreateClipInput{
		Audio:         m.GetAudio(),
		ReferenceText: m.GetReferenceText(),
		Tags:          m.GetTags(),
		DurationMs:    m.GetDurationMs(),
		SampleRateHz:  int(m.GetSampleRateHz()),
		Format:        m.GetFormat(),
		Source:        sourceFromProto(m.GetSource()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&corpusv1.CreateClipResponse{Clip: clipToProto(clip)}), nil
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

// mapClipError maps a not-found to connect.CodeNotFound; everything else is
// Internal.
func mapClipError(err error) error {
	var nf intcorpus.ErrClipNotFound
	if errors.As(err, &nf) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
