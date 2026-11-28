/**
 * data layer — everything that talks to the server.
 * Imports sim, config and the generated proto client. Never scene, hud or engine.
 */
export { createWorldApi, createWorldClient, createWorldTransport, type WorldApi } from './worldClient'
export { eventToSignals, diffRuns, parseSeconds, initialAdapterState, type AdapterState, type KnownRun } from './adapters'
export { startFeed, backoffMs, type FeedHandle, type FeedStatus, type FeedMode, type FeedDeps } from './feed'
export { useWorldRuntime, pollRunningAgents, type WorldRuntime, type RuntimeOptions } from './runtime'
export { useWorldRoster, agentToInput, type WorldRoster } from './roster'
export { syntheticRoster } from './synthetic'
export { createWorldActions, type WorldActions } from './actions'
export { useWorldPreferences, preferencesFromWire, DEFAULT_PREFERENCES, type WorldPreferences } from './config'
export { useLayoutPersistence, overridesFromWire, overridesToWire, type LayoutPersistence } from './layout'
