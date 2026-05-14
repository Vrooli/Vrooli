package session

// AnsiResponderFunc returns server-origin replies for ANSI device queries
// (DA1/DA3/DECRQM 2026) found in chunk, or nil if there are none.
// Package main installs the real responder during init; the default is no-op
// so the package builds in isolation (tests / future standalone callers).
//
// The Phase 3 plan turns this into a session.Observer that subscribes to the
// emulator's parsed ControlEvent stream; until then the responder runs inline
// in readLoop the same way it did pre-split, just through this seam.
var AnsiResponderFunc = func(chunk []byte) []byte { return nil }

// SanitizeForClientFunc strips control sequences from PTY output before it
// is broadcast to client channels. Package main installs the real
// implementation during init; the default is identity so the package builds
// without package-main's ansi_strip helper.
var SanitizeForClientFunc = func(data []byte) []byte { return data }
