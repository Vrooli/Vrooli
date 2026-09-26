// Role domain client: named positions inside a group.
//
// Roles live on WorkspaceService (a role IS workspace layout), so this module
// reuses the workspace client rather than opening a second transport. The
// snake_case DTO shape matches the rest of the workspace API surface; the
// camelCase proto fields are translated in decodeRole.
//
// [REQ:P0-014e] Waiting Roles

import { workspaceClient } from "./workspace";

/**
 * A named position inside a group.
 *
 * `session_id` is null while the role is WAITING: it holds a command and no
 * process. That distinction is the whole reason the type exists, so it is
 * modelled as `string | null` rather than an empty string — a caller has to
 * decide what to do about it.
 *
 * `command` is a plain string, never an enum of known agents.
 */
export interface RoleDTO {
  id: string;
  group_id: string;
  label: string;
  command: string;
  working_dir: string;
  /** May contain at most one `{{payload}}` placeholder. Lives on the receiving role. */
  incoming_prompt: string;
  backend: string;
  target_id: string;
  /** Null while waiting. */
  session_id: string | null;
  sort_order: number;
}

/** The fields a caller may set when creating a role. */
export type CreateRoleInput = Partial<Omit<RoleDTO, "id" | "group_id">> & { group_id: string };

/** The fields a caller may change. Omitted keys are left untouched. */
export type UpdateRoleInput = Partial<Omit<RoleDTO, "id">>;

interface RoleWire {
  id: string;
  groupId: string;
  label: string;
  command: string;
  workingDir: string;
  incomingPrompt: string;
  backend: string;
  targetId: string;
  sessionId: string;
  sortOrder: number;
}

export function decodeRole(r: RoleWire): RoleDTO {
  return {
    id: r.id,
    group_id: r.groupId,
    label: r.label,
    command: r.command,
    working_dir: r.workingDir,
    incoming_prompt: r.incomingPrompt,
    backend: r.backend,
    target_id: r.targetId,
    // The wire carries "" for a waiting role; the UI carries null, so no
    // component can mistake the empty string for a session id.
    session_id: r.sessionId === "" ? null : r.sessionId,
    sort_order: r.sortOrder,
  };
}

export async function listRoles(groupId?: string): Promise<RoleDTO[]> {
  const resp = await workspaceClient.listRoles({ groupId: groupId ?? "" });
  return resp.roles.map(decodeRole);
}

export async function createRole(input: CreateRoleInput): Promise<RoleDTO> {
  const resp = await workspaceClient.createRole({
    groupId: input.group_id,
    label: input.label ?? "",
    command: input.command ?? "",
    workingDir: input.working_dir ?? "",
    incomingPrompt: input.incoming_prompt ?? "",
    backend: input.backend ?? "",
    targetId: input.target_id ?? "",
    sessionId: input.session_id ?? "",
    sortOrder: input.sort_order ?? 0,
  });
  if (!resp.role) throw new Error("workspace.createRole: missing role in response");
  return decodeRole(resp.role);
}

export async function updateRole(id: string, update: UpdateRoleInput): Promise<RoleDTO> {
  const req: Parameters<typeof workspaceClient.updateRole>[0] = { id };
  if (update.label !== undefined) {
    req.label = update.label;
    req.hasLabel = true;
  }
  if (update.command !== undefined) {
    req.command = update.command;
    req.hasCommand = true;
  }
  if (update.working_dir !== undefined) {
    req.workingDir = update.working_dir;
    req.hasWorkingDir = true;
  }
  if (update.incoming_prompt !== undefined) {
    req.incomingPrompt = update.incoming_prompt;
    req.hasIncomingPrompt = true;
  }
  if (update.session_id !== undefined) {
    // null means "return this role to waiting", which the wire spells as an
    // empty string with the flag set. Dropping the flag would instead mean
    // "leave it alone", so the distinction is load-bearing.
    req.sessionId = update.session_id ?? "";
    req.hasSessionId = true;
  }
  if (update.sort_order !== undefined) {
    req.sortOrder = update.sort_order;
    req.hasSortOrder = true;
  }
  if (update.backend !== undefined) {
    req.backend = update.backend;
    req.hasBackend = true;
  }
  if (update.target_id !== undefined) {
    req.targetId = update.target_id;
    req.hasTargetId = true;
  }
  if (update.group_id !== undefined) {
    req.groupId = update.group_id;
    req.hasGroupId = true;
  }
  const resp = await workspaceClient.updateRole(req);
  if (!resp.role) throw new Error("workspace.updateRole: missing role in response");
  return decodeRole(resp.role);
}

export async function deleteRole(id: string): Promise<void> {
  await workspaceClient.deleteRole({ id });
}
