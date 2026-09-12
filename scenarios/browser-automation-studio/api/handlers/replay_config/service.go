package replay_config

import (
	"context"
	"encoding/json"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/vrooli/browser-automation-studio/database"
	replayconfigv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/replay_config"
)

const settingsKey = "replay_config.v1"

type service struct {
	deps Deps
}

func (s *service) Get(
	ctx context.Context,
	_ *connect.Request[replayconfigv1.GetReplayConfigRequest],
) (*connect.Response[replayconfigv1.GetReplayConfigResponse], error) {
	cfg, err := s.load(ctx)
	if err != nil {
		s.deps.Logger.WithError(err).Error("replay_config.Get failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	pb, err := structpb.NewStruct(cfg)
	if err != nil {
		s.deps.Logger.WithError(err).Error("replay_config.Get encode failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&replayconfigv1.GetReplayConfigResponse{Config: pb}), nil
}

func (s *service) Put(
	ctx context.Context,
	req *connect.Request[replayconfigv1.PutReplayConfigRequest],
) (*connect.Response[replayconfigv1.PutReplayConfigResponse], error) {
	cfg := req.Msg.GetConfig()
	if cfg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errMissingConfig)
	}
	asMap := cfg.AsMap()
	payload, err := json.Marshal(asMap)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidConfig)
	}
	if err := s.deps.Store.SetSetting(ctx, settingsKey, string(payload)); err != nil {
		s.deps.Logger.WithError(err).Error("replay_config.Put persist failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&replayconfigv1.PutReplayConfigResponse{Config: cfg}), nil
}

func (s *service) Reset(
	ctx context.Context,
	_ *connect.Request[replayconfigv1.ResetReplayConfigRequest],
) (*connect.Response[replayconfigv1.ResetReplayConfigResponse], error) {
	if err := s.deps.Store.DeleteSetting(ctx, settingsKey); err != nil {
		s.deps.Logger.WithError(err).Error("replay_config.Reset failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	empty, _ := structpb.NewStruct(map[string]any{})
	return connect.NewResponse(&replayconfigv1.ResetReplayConfigResponse{Config: empty}), nil
}

func (s *service) load(ctx context.Context) (map[string]any, error) {
	value, err := s.deps.Store.GetSetting(ctx, settingsKey)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	if value == "" {
		return map[string]any{}, nil
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(value), &cfg); err != nil {
		return nil, err
	}
	if cfg == nil {
		return map[string]any{}, nil
	}
	return cfg, nil
}
