/**
 * Zod schemas for miscellaneous API response types.
 *
 * Covers health, docs, desktop records, probing, wine, telemetry,
 * and port endpoints. These schemas provide runtime validation at the
 * API boundary so callers receive verified data rather than trusting
 * untyped JSON.
 */

import { z } from "zod";

// ==================== Docs ====================

export const DocsDocumentSchema = z.object({
  path: z.string(),
  title: z.string(),
  description: z.string().optional(),
});

export const DocsSectionSchema = z.object({
  id: z.string(),
  title: z.string(),
  icon: z.string().optional(),
  description: z.string().optional(),
  documents: z.array(DocsDocumentSchema),
});

export const DocsNavigationSchema = z.object({
  primary: z.array(z.string()).optional(),
  secondary: z.array(z.string()).optional(),
});

export const DocsManifestSchema = z.object({
  version: z.string(),
  title: z.string(),
  description: z.string().optional(),
  defaultDocument: z.string(),
  sections: z.array(DocsSectionSchema),
  navigation: DocsNavigationSchema.optional(),
});

export const DocsContentResponseSchema = z.object({
  path: z.string(),
  content: z.string(),
});

// ==================== Proxy / Bundle ====================

export const ProxyHintSchema = z.object({
  url: z.string(),
  source: z.string(),
  confidence: z.string(),
  message: z.string(),
});

export const ProxyHintsResponseSchema = z.object({
  scenario: z.string(),
  hints: z.array(ProxyHintSchema),
});

export const BundleManifestResponseSchema = z.object({
  path: z.string(),
  // Validated as present in the JSON; the explicit return type on
  // fetchBundleManifest() narrows this to the hand-written interface.
  manifest: z.any(),
});

// ==================== Wine ====================

export const WineInstallMethodSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string(),
  requires_sudo: z.boolean(),
  steps: z.array(z.string()),
  estimated_time: z.string(),
});

export const WineCheckResponseSchema = z.object({
  installed: z.boolean(),
  version: z.string().optional(),
  platform: z.string().optional(),
  required_for: z.array(z.string()).optional(),
  install_methods: z.array(WineInstallMethodSchema).optional(),
  recommended_method: z.string().optional(),
});

export const WineInstallStatusSchema = z.object({
  install_id: z.string(),
  status: z.string(),
  method: z.string(),
  started_at: z.string(),
  completed_at: z.string().optional(),
  log: z.array(z.string()),
  error_log: z.array(z.string()),
});

// ==================== Telemetry ====================

export const TelemetryInsightSessionSchema = z.object({
  session_id: z.string().optional(),
  status: z.string(),
  started_at: z.string().optional(),
  ready_at: z.string().optional(),
  completed_at: z.string().optional(),
  reason: z.string().optional(),
});

export const TelemetryInsightSmokeTestSchema = z.object({
  session_id: z.string().optional(),
  status: z.string(),
  started_at: z.string().optional(),
  completed_at: z.string().optional(),
  error: z.string().optional(),
});

export const TelemetryInsightErrorSchema = z.object({
  event: z.string(),
  timestamp: z.string(),
  message: z.string().optional(),
});

export const TelemetryInsightsSchema = z.object({
  scenario_name: z.string(),
  exists: z.boolean(),
  last_session: TelemetryInsightSessionSchema.optional(),
  last_smoke_test: TelemetryInsightSmokeTestSchema.optional(),
  last_error: TelemetryInsightErrorSchema.optional(),
});

export const TelemetrySummarySchema = z.object({
  scenario_name: z.string(),
  exists: z.boolean(),
  file_path: z.string().optional(),
  file_size_bytes: z.number().optional(),
  event_count: z.number().optional(),
  last_ingested_at: z.string().optional(),
});

export const TelemetryEventSchema = z.object({
  event: z.string().optional(),
  timestamp: z.string().optional(),
  level: z.string().optional(),
  session_id: z.string().optional(),
  session_kind: z.string().optional(),
  details: z.record(z.unknown()).optional(),
  detail: z.string().optional(),
}).passthrough();

export const TelemetryTailEntrySchema = z.object({
  raw: z.string(),
  event: TelemetryEventSchema.optional(),
  error: z.string().optional(),
});

export const TelemetryTailResponseSchema = z.object({
  scenario_name: z.string(),
  exists: z.boolean(),
  limit: z.number(),
  total_lines: z.number().optional(),
  entries: z.array(TelemetryTailEntrySchema).optional(),
});

// ==================== Port ====================

export const ScenarioPortResponseSchema = z.object({
  scenario: z.string(),
  port_name: z.string(),
  host: z.string(),
  port: z.number(),
  url: z.string(),
});

// ==================== Inline response types ====================
// Named schemas for small response shapes that were previously inline `as` casts.

export const MoveRecordResponseSchema = z.object({
  record_id: z.string(),
  from: z.string(),
  to: z.string(),
  status: z.string(),
});

export const StatusResponseSchema = z.object({
  status: z.string(),
});

export const InstallIdResponseSchema = z.object({
  install_id: z.string(),
});

export const OutputPathResponseSchema = z.object({
  output_path: z.string(),
});
