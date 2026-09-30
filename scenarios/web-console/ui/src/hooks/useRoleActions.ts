import { useCallback } from "react";

import {
  createRole as createRoleAPI,
  deleteRole as deleteRoleAPI,
  updateRole as updateRoleAPI,
  type RoleDTO,
} from "../api/workspaceRoles";
import { useWorkspaceStore, type RoleMeta } from "../stores/useWorkspaceStore";

// [REQ:P0-014e] Waiting Roles

/** Translate the wire shape into the store's shape, in exactly one place. */
export function roleFromDTO(dto: RoleDTO): RoleMeta {
  return {
    id: dto.id,
    groupId: dto.group_id,
    label: dto.label,
    command: dto.command,
    workingDir: dto.working_dir,
    incomingPrompt: dto.incoming_prompt,
    backend: dto.backend,
    targetId: dto.target_id,
    sessionId: dto.session_id,
    sortOrder: dto.sort_order,
  };
}

export interface CreateRoleOptions {
  groupId: string;
  label: string;
  command?: string;
  workingDir?: string;
  incomingPrompt?: string;
  backend?: string;
  targetId?: string;
  /** Set to attach the role to an already-running session. */
  sessionId?: string | null;
}

/**
 * Role operations, each pairing the local store mutation with its backend
 * sync — the same shape as useGroupActions, so a reader who knows one knows
 * the other.
 *
 * Creation is SERVER-FIRST and awaited: the backend mints the role id, and a
 * client-fabricated id would produce a role that no later update could reach.
 */
export function useRoleActions() {
  const addRole = useWorkspaceStore((s) => s.addRole);
  const updateRoleLocal = useWorkspaceStore((s) => s.updateRole);
  const removeRoleLocal = useWorkspaceStore((s) => s.removeRole);
  const setRoleSessionLocal = useWorkspaceStore((s) => s.setRoleSession);

  const createRole = useCallback(async (options: CreateRoleOptions): Promise<RoleMeta | null> => {
    try {
      const dto = await createRoleAPI({
        group_id: options.groupId,
        label: options.label,
        command: options.command ?? "",
        working_dir: options.workingDir ?? "",
        incoming_prompt: options.incomingPrompt ?? "",
        backend: options.backend ?? "",
        target_id: options.targetId ?? "",
        session_id: options.sessionId ?? null,
      });
      const role = roleFromDTO(dto);
      addRole(role);
      return role;
    } catch (error) {
      console.error("Failed to create role:", error);
      return null;
    }
  }, [addRole]);

  const updateRole = useCallback((roleId: string, update: Partial<Omit<RoleMeta, "id">>) => {
    updateRoleLocal(roleId, update);
    void updateRoleAPI(roleId, {
      ...(update.label !== undefined ? { label: update.label } : {}),
      ...(update.command !== undefined ? { command: update.command } : {}),
      ...(update.workingDir !== undefined ? { working_dir: update.workingDir } : {}),
      ...(update.incomingPrompt !== undefined ? { incoming_prompt: update.incomingPrompt } : {}),
      ...(update.backend !== undefined ? { backend: update.backend } : {}),
      ...(update.targetId !== undefined ? { target_id: update.targetId } : {}),
      ...(update.sortOrder !== undefined ? { sort_order: update.sortOrder } : {}),
      ...(update.groupId !== undefined ? { group_id: update.groupId } : {}),
      ...(update.sessionId !== undefined ? { session_id: update.sessionId } : {}),
    }).catch((error: unknown) => { console.error("Failed to sync role update:", error); });
  }, [updateRoleLocal]);

  const removeRole = useCallback((roleId: string) => {
    removeRoleLocal(roleId);
    void deleteRoleAPI(roleId).catch((error: unknown) => {
      console.error("Failed to sync role delete:", error);
    });
  }, [removeRoleLocal]);

  /**
   * Point a role at a session, or (with null) return it to waiting.
   *
   * This is the one call the session-reconcile effect makes when a started
   * role's session finally appears, so it must stay cheap and idempotent.
   */
  const setRoleSession = useCallback((roleId: string, sessionId: string | null) => {
    setRoleSessionLocal(roleId, sessionId);
    void updateRoleAPI(roleId, { session_id: sessionId }).catch((error: unknown) => {
      console.error("Failed to sync role session:", error);
    });
  }, [setRoleSessionLocal]);

  return { createRole, updateRole, removeRole, setRoleSession };
}
