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

// OllamaProviderTransport identifies how the Ollama provider is reached. Calls
// go through the resource-ollama gateway CLI, not direct HTTP — the host-wide
// semaphore can only bound parallelism when every scenario funnels through the
// CLI. Surface this as the provider's URL so the existing provider-listing API
// remains stable for clients.
const OllamaProviderTransport = "resource-ollama://gateway"

// OpenRouterURL is the fixed endpoint for the OpenRouter fallback provider.
const OpenRouterURL = "https://openrouter.ai/api/v1"
