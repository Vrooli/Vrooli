// Wake-word administration handlers backed by the wakeword store.
package stt

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	"audio-tools/internal/store"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

const wakeWordID = "default"

func (h *connectHandler) GetWakeWordConfig(ctx context.Context, _ *connect.Request[sttv1.GetWakeWordConfigRequest]) (*connect.Response[sttv1.GetWakeWordConfigResponse], error) {
	if h.deps.Wakeword == nil {
		return connect.NewResponse(&sttv1.GetWakeWordConfigResponse{Config: &sttv1.WakeWordConfig{}}), nil
	}
	t, ok, err := h.deps.Wakeword.Get(ctx, wakeWordID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	cfg := &sttv1.WakeWordConfig{Configured: ok}
	if ok && t.Phrase != "" {
		tmpl := &sttv1.WakeWordTemplate{}
		if err := protojson.Unmarshal([]byte(t.Phrase), tmpl); err == nil {
			cfg.Template = tmpl
		}
	}
	return connect.NewResponse(&sttv1.GetWakeWordConfigResponse{Config: cfg}), nil
}

func (h *connectHandler) UpdateWakeWordTemplate(ctx context.Context, req *connect.Request[sttv1.UpdateWakeWordTemplateRequest]) (*connect.Response[sttv1.UpdateWakeWordTemplateResponse], error) {
	if h.deps.Wakeword == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("wakeword store not configured"))
	}
	tmpl := req.Msg.GetTemplate()
	if tmpl == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("template required"))
	}
	raw, err := protojson.Marshal(tmpl)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := h.deps.Wakeword.Upsert(ctx, store.WakeWordTemplate{
		ID: wakeWordID, Phrase: string(raw),
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&sttv1.UpdateWakeWordTemplateResponse{
		Config: &sttv1.WakeWordConfig{Configured: true, Template: tmpl},
	}), nil
}

func (h *connectHandler) DeleteWakeWordTemplate(ctx context.Context, _ *connect.Request[sttv1.DeleteWakeWordTemplateRequest]) (*connect.Response[sttv1.DeleteWakeWordTemplateResponse], error) {
	if h.deps.Wakeword == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("wakeword store not configured"))
	}
	if _, err := h.deps.Wakeword.Delete(ctx, wakeWordID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&sttv1.DeleteWakeWordTemplateResponse{
		Config: &sttv1.WakeWordConfig{Configured: false},
	}), nil
}
