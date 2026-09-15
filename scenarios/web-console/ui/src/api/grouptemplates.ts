// Group templates domain: Connect-RPC client, types, and decoders for saved
// role recipes. Creating a group from a template creates the group and its
// roles in one action.
//
// A template carries no privilege. A shipped example is an ordinary row and
// is deletable like any other, so nothing here has an is-builtin flag.
//
// [REQ:P0-014g] Group Templates

import { createClient } from "@connectrpc/connect";
import { GroupTemplatesService } from "@vrooli/proto-types/web-console/v1/grouptemplates/grouptemplates_pb";

import { transport } from "./client";

export const groupTemplatesClient = createClient(GroupTemplatesService, transport);

/**
 * Whether creating a group from a template starts a process for this role.
 *
 * Only `eager` costs a process. Anything else waits, so a template can hold a
 * long role list without the operator paying for every agent up front.
 */
export type StartMode = "eager" | "waiting";

export interface TemplateRoleDTO {
  label: string;
  command: string;
  working_dir: string;
  /** May contain at most one `{{payload}}` placeholder. */
  incoming_prompt: string;
  backend: string;
  target_id: string;
  start_mode: StartMode;
}

export interface GroupTemplateDTO {
  id: string;
  name: string;
  color: string;
  /** An ordered list of any length from zero upward. Nothing constrains it to two. */
  roles: TemplateRoleDTO[];
  use_count: number;
}

interface TemplateRoleWire {
  label: string;
  command: string;
  workingDir: string;
  incomingPrompt: string;
  backend: string;
  targetId: string;
  startMode: string;
}

function decodeRole(r: TemplateRoleWire): TemplateRoleDTO {
  return {
    label: r.label,
    command: r.command,
    working_dir: r.workingDir,
    incoming_prompt: r.incomingPrompt,
    backend: r.backend,
    target_id: r.targetId,
    // An unrecognised mode waits. Defaulting the other way would let a
    // malformed row cost the operator an unexpected process.
    start_mode: r.startMode === "eager" ? "eager" : "waiting",
  };
}

function decodeTemplate(t: {
  id: string;
  name: string;
  color: string;
  roles: TemplateRoleWire[];
  useCount: number;
}): GroupTemplateDTO {
  return {
    id: t.id,
    name: t.name,
    color: t.color,
    roles: t.roles.map(decodeRole),
    use_count: t.useCount,
  };
}

export async function listGroupTemplates(): Promise<GroupTemplateDTO[]> {
  const resp = await groupTemplatesClient.listTemplates({});
  return resp.templates.map(decodeTemplate);
}

export interface UpsertGroupTemplateInput {
  id?: string;
  name: string;
  color?: string;
  roles: TemplateRoleDTO[];
  /** Omit to leave the counter alone — editing content must not reset it. */
  use_count?: number;
}

export async function upsertGroupTemplate(input: UpsertGroupTemplateInput): Promise<GroupTemplateDTO> {
  const resp = await groupTemplatesClient.upsertTemplate({
    id: input.id ?? "",
    name: input.name,
    color: input.color ?? "",
    roles: input.roles.map((r) => ({
      label: r.label,
      command: r.command,
      workingDir: r.working_dir,
      incomingPrompt: r.incoming_prompt,
      backend: r.backend,
      targetId: r.target_id,
      startMode: r.start_mode,
    })),
    useCount: input.use_count ?? 0,
    hasUseCount: input.use_count !== undefined,
  });
  if (!resp.template) throw new Error("grouptemplates.upsertTemplate: missing template in response");
  return decodeTemplate(resp.template);
}

export async function deleteGroupTemplate(id: string): Promise<void> {
  await groupTemplatesClient.deleteTemplate({ id });
}
