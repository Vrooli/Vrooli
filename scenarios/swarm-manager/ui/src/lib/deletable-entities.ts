/**
 * Deletable Entity Registry
 *
 * Single source of truth for every entity type that supports the unified
 * three-level delete-confirmation system (none / simple / strong). The
 * registry drives:
 *   - the Settings "Delete Confirmation" UI (iterated to render a row per type)
 *   - DEFAULT_SETTINGS / normalize fallbacks (default level per type)
 *   - the useDeleteConfirm hook (resolves a type's configured level)
 *
 * Map keys persisted in settings are the `DeletableEntityType` strings below.
 * The API mirrors this list in `api/internal/settings/model.go`
 * (`deletableEntityDefaults`) — keep the two in sync when adding a type.
 */

import type { DeleteConfirmLevel } from "../types/settings";

/**
 * Every entity type whose delete confirmation is configurable. Adding a new
 * deletable entity = add one entry here (and the Go mirror), no proto changes.
 */
export type DeletableEntityType =
  | "session"
  | "scenario"
  | "backlog"
  | "initiative"
  | "capture"
  | "backlogFile";

export interface DeletableEntityMeta {
  /** Stable key persisted in settings and used as the map key. */
  type: DeletableEntityType;
  /** Plural label shown in the Settings UI. */
  label: string;
  /** Helper text shown under the label in the Settings UI. */
  description: string;
  /** Confirmation level applied when the user has not chosen one. */
  defaultLevel: DeleteConfirmLevel;
}

export const DELETABLE_ENTITIES: readonly DeletableEntityMeta[] = [
  {
    type: "session",
    label: "Sessions",
    description: "Agent sessions and their conversation history.",
    defaultLevel: "simple",
  },
  {
    type: "scenario",
    label: "Scenarios",
    description: "Scenario apps — high-stakes; archive before deleting.",
    defaultLevel: "strong",
  },
  {
    type: "backlog",
    label: "Backlog Items",
    description: "Ideas, fixes, research, execute, and chore items.",
    defaultLevel: "simple",
  },
  {
    type: "initiative",
    label: "Initiatives",
    description: "Initiative groupings and their metadata.",
    defaultLevel: "strong",
  },
  {
    type: "capture",
    label: "Captures",
    description: "Captured notes and observations.",
    defaultLevel: "none",
  },
  {
    type: "backlogFile",
    label: "Backlog Files",
    description: "Files and directories attached to backlog items.",
    defaultLevel: "simple",
  },
] as const;

/** Quick lookup of a registry entry by type. */
export const DELETABLE_ENTITY_BY_TYPE: Readonly<
  Record<DeletableEntityType, DeletableEntityMeta>
> = Object.fromEntries(
  DELETABLE_ENTITIES.map((e) => [e.type, e]),
) as Record<DeletableEntityType, DeletableEntityMeta>;

/** All entity-type keys, in registry order. */
export const DELETABLE_ENTITY_TYPES: readonly DeletableEntityType[] =
  DELETABLE_ENTITIES.map((e) => e.type);

/**
 * Default level map built from the registry. Used by DEFAULT_SETTINGS and as
 * the normalize fallback so every known entity always resolves to a level.
 */
export function defaultDeleteConfirmationLevels(): Record<DeletableEntityType, DeleteConfirmLevel> {
  return DELETABLE_ENTITIES.reduce(
    (acc, e) => {
      acc[e.type] = e.defaultLevel;
      return acc;
    },
    {} as Record<DeletableEntityType, DeleteConfirmLevel>,
  );
}

/** Registry default level for a single entity type. */
export function defaultLevelFor(type: DeletableEntityType): DeleteConfirmLevel {
  return DELETABLE_ENTITY_BY_TYPE[type].defaultLevel;
}
