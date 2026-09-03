/**
 * sim layer — deterministic, renderer-free world model.
 * API: createWorld / step / buildView, or createWorldStore for a live world.
 */
export type * from './model'
export { createWorld, rebuildLayout, variantFor } from './world'
export { step, type StepTuning } from './tick'
export { buildView, createViewSelector, equipmentTier, type WorldView, type ActorView, type SummaryView, type TeamView } from './view/select'
export { createWorldStore, type WorldStore } from './store'
export { hashState } from './hash'
export { generateLayout, applyOverrides, scatterTrees, roomId, deskId, deskSeatId, tableId, COMMONS_ID, CAMPFIRE_ID, BOARD_ID, type GeneratedLayout } from './layout/generate'
export { buildNavGrid, isWalkable, nearestWalkable, worldToCell, cellToWorld } from './nav/grid'
export { findPath, smoothPath, lineOfSight, PathCache } from './nav/astar'
export { moveAlongPath, turnToward, wrapAngle, headingTo } from './motion/move'
export * as seatMath from './layout/seatMath'
export { emptyHistory, upsertOverride, removeOverride, commit, undo, redo, canUndo, canRedo, snapPosition, invertOverride, type OverrideHistory } from './layout/overrides'
export { checkWorldInvariants, checkSeats, checkBounds, checkPlaces, checkSeparation, checkRestingInPlace, type Violation, type InvariantRule, type InvariantTuning } from './invariants'
export { atHome, insideCommons } from './idle/behaviors'
