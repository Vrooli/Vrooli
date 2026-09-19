// Capture rules domain: Connect-RPC client, types, and decoders for the
// patterns that decide when a handoff is SUGGESTED.
//
// A rule never sends anything. This module is read by the matcher and the
// rules editor; the send path in hooks/useHandoff.ts must never import it.
//
// [REQ:P0-014h] Handoff Capture Rules

import { createClient } from "@connectrpc/connect";
import { HandoffRulesService } from "@vrooli/proto-types/web-console/v1/handoffrules/handoffrules_pb";

import { transport } from "./client";

export const handoffRulesClient = createClient(HandoffRulesService, transport);

/**
 * What a rule's pattern is matched against.
 *
 * `file_path`    — a glob against paths a session mentioned.
 * `message_text` — a regular expression against message text; the first
 *                  capture group becomes the payload, or the whole match when
 *                  the pattern has no group.
 */
export type RuleSource = "file_path" | "message_text";

export interface HandoffRuleDTO {
  id: string;
  name: string;
  enabled: boolean;
  source: RuleSource;
  pattern: string;
  /** Where a match may render. Empty means every surface that can render one. */
  surfaces: string[];
  sort_order: number;
}

function decodeRule(r: {
  id: string;
  name: string;
  enabled: boolean;
  source: string;
  pattern: string;
  surfaces: string[];
  sortOrder: number;
}): HandoffRuleDTO {
  return {
    id: r.id,
    name: r.name,
    enabled: r.enabled,
    // An unrecognised source falls back to the narrower matcher rather than
    // the regular-expression one, so a malformed row cannot run arbitrary
    // patterns over message text.
    source: r.source === "message_text" ? "message_text" : "file_path",
    pattern: r.pattern,
    surfaces: r.surfaces,
    sort_order: r.sortOrder,
  };
}

export async function listHandoffRules(): Promise<HandoffRuleDTO[]> {
  const resp = await handoffRulesClient.listRules({});
  return resp.rules.map(decodeRule);
}

export type UpsertHandoffRuleInput = Omit<HandoffRuleDTO, "id" | "sort_order"> & {
  id?: string;
  sort_order?: number;
};

export async function upsertHandoffRule(input: UpsertHandoffRuleInput): Promise<HandoffRuleDTO> {
  const resp = await handoffRulesClient.upsertRule({
    id: input.id ?? "",
    name: input.name,
    enabled: input.enabled,
    source: input.source,
    pattern: input.pattern,
    surfaces: input.surfaces,
    sortOrder: input.sort_order ?? 0,
  });
  if (!resp.rule) throw new Error("handoffrules.upsertRule: missing rule in response");
  return decodeRule(resp.rule);
}

export async function deleteHandoffRule(id: string): Promise<void> {
  await handoffRulesClient.deleteRule({ id });
}
