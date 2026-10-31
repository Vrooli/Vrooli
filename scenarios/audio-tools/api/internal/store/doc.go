// Package store provides typed read/write access to the audio-tools
// SQLite database. Each concern (provider config, BYOK credentials,
// voice overrides, usage rows, wake-word templates, speaker profiles,
// STT stream config, TTS config, playback events) lives in its own
// file. Handlers depend on the store interfaces — not directly on
// database/sql — so they remain unit-testable with an in-memory fake.
package store
