/**
 * Local experience telemetry for scenario UIs.
 *
 * Server: `createScenarioServer` mounts these endpoints and injects the agent
 * on loopback requests by default. Import from `@vrooli/api-base/experience`
 * when hosting a UI outside the shared server.
 *
 * Analysis: `readSessions` + `analyzeExperience` turn append-only JSONL into a
 * friction report. The `vrooli-experience` binary wraps the same functions.
 */

export {
  EXPERIENCE_AGENT_ATTRIBUTE,
  EXPERIENCE_DEFAULTS,
  EXPERIENCE_PROTOCOL_VERSION,
} from './types.js'
export type {
  ExperienceActionEvent,
  ExperienceAdvisoryFinding,
  ExperienceAdvisoryKind,
  ExperienceBatch,
  ExperienceBatchRequest,
  ExperienceCapabilityDeclaration,
  ExperienceCapabilityMetric,
  ExperienceControlMetric,
  ExperienceCorrectionCandidate,
  ExperienceElementRef,
  ExperienceErrorEvent,
  ExperienceEvent,
  ExperienceFieldEvent,
  ExperienceFieldMetric,
  ExperienceFlowMetric,
  ExperienceFrictionEvent,
  ExperienceFrictionKind,
  ExperienceFrictionMetric,
  ExperienceFrictionReport,
  ExperienceJourneyDiff,
  ExperienceKeyEvent,
  ExperienceObservedBehavior,
  ExperienceObservedFlow,
  ExperiencePacketObservedBlock,
  ExperiencePerformanceEvent,
  ExperiencePointerFeatures,
  ExperiencePolicy,
  ExperienceRecommendation,
  ExperienceRouteEvent,
  ExperienceRouteMetric,
  ExperienceSession,
  ExperienceSessionEndEvent,
  ExperienceSessionStartEvent,
  ExperienceTelemetryOptions,
  ExperienceTotals,
  ExperienceVisibilityEvent,
  ResolvedExperienceTelemetry,
} from './types.js'

export {
  experienceDataDir,
  experienceInjectionConfig,
  experienceScenarioName,
  isLoopbackHost,
  readExperiencePolicy,
  resolveExperienceTelemetry,
  shouldInstrumentRequest,
} from './gate.js'
export type { ExperienceRequestLike } from './gate.js'

export { buildExperienceScriptTag, injectExperienceTelemetry } from './inject.js'
export type { ExperienceInjectionConfig } from './inject.js'

export { createExperienceTelemetryRouter } from './router.js'

export { installExperienceTelemetry } from './standalone.js'
export type { StandaloneExperienceOptions } from './standalone.js'

export {
  appendBatch,
  ensureSessionDir,
  isSessionId,
  listSessionFiles,
  pruneSessions,
  readSessions,
  sessionStatus,
} from './store.js'
export type { PruneResult, ReadSessionsResult, SessionFileInfo, SessionStatus } from './store.js'

export {
  analyzeExperience,
  buildSessions,
  loadDeclaredSpec,
  renderFrictionMarkdown,
  routeMatches,
} from './report.js'
export type { AnalyzeOptions, AnalyzeResult } from './report.js'

export {
  canonicalJson,
  computeReportDigest,
  createObservedEvidence,
  createPacketObservedBlock,
  deriveAdvisoryFindings,
  deriveCorrectionCandidates,
  validateReport,
} from './contracts.js'
export type { ObservedEvidenceOptions } from './contracts.js'
