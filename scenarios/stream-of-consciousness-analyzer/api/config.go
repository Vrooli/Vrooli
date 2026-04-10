// DOC: docs/reference/configuration.md
package main

import "time"

// --- Application Metadata ---

// AppVersion is the semantic version of the Stream of Consciousness Analyzer API.
const AppVersion = "1.0.0"

// --- Export Configuration ---

// ExportFormatVersion identifies the graph export schema version.
// Consumers of the export endpoint use this to select the correct parser.
const ExportFormatVersion = "vrooli-graph-v1"

// --- Request Timeouts ---

// RequestTimeout bounds how long any single API request may run.
// Long-running operations (export, suggestions) share this limit.
const RequestTimeout = 30 * time.Second

// --- LLM Provider Defaults ---

// DefaultOllamaURL is the default endpoint for the local Ollama instance.
// Override at runtime with the OLLAMA_URL environment variable.
const DefaultOllamaURL = "http://localhost:11434"

// OpenRouterURL is the fixed endpoint for the OpenRouter fallback provider.
const OpenRouterURL = "https://openrouter.ai/api/v1"
