package main

import (
	"github.com/vrooli/api-core/storage"

	inttts "web-console/internal/tts"
)

// getTTSConfig / setTTSConfig provide thread-safe access to the Server's
// in-memory TTS config. The config type itself, defaults, patch type, and
// load/save helpers live in internal/tts.
func (s *Server) getTTSConfig() inttts.Config {
	s.ttsConfigMu.RLock()
	defer s.ttsConfigMu.RUnlock()
	return s.ttsConfig
}

func (s *Server) setTTSConfig(cfg inttts.Config) {
	s.ttsConfigMu.Lock()
	defer s.ttsConfigMu.Unlock()
	s.ttsConfig = cfg
}

func (s *Server) getTTSSummarizeConfig() inttts.SummarizeConfig {
	s.ttsSummarizeMu.RLock()
	defer s.ttsSummarizeMu.RUnlock()
	return s.ttsSummarizeConfig
}

func (s *Server) setTTSSummarizeConfig(cfg inttts.SummarizeConfig) {
	s.ttsSummarizeMu.Lock()
	defer s.ttsSummarizeMu.Unlock()
	s.ttsSummarizeConfig = cfg
}

// resolveTTSSummarizeConfigPath returns the summarize config file path using api-core/storage.
func resolveTTSSummarizeConfigPath() string {
	return mustResolveScenarioStoragePath(storage.ClassState, "tts-summarize-config.json")
}
