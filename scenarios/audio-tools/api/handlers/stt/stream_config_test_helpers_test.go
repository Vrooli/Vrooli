package stt

import "context"

type staticStreamConfig struct{ raw string }

func (s staticStreamConfig) Get(context.Context) (string, bool, error) { return s.raw, true, nil }
func (s staticStreamConfig) Set(context.Context, string) error         { return nil }
