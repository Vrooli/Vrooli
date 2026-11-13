import { keepPreviousData, useQuery } from "@tanstack/react-query";
import type { Workstream } from "@vrooli/proto-types/research-desk/v1/shared/commission_pb";
import {
  ControlAction,
  LaunchService,
  Resolution,
  type GetLaunchOptionsResponse,
  type PreflightResponse,
} from "@vrooli/proto-types/research-desk/v1/launch/launch_pb";

import { serviceClient } from "./client";

/** Launch domain: packets, launch and control without chat (SYSTEM.md §5–§6). */
const client = serviceClient(LaunchService);

export function useLaunchOptions() {
  return useQuery({ queryKey: ["launch", "options"], queryFn: () => client.getLaunchOptions({}) });
}

/**
 * Suggested stance quotes and each workstream's title for the chosen topics
 *. Kept while the topics change, so the form does not flicker.
 */
export function useLaunchSuggestions(topics: readonly string[]) {
  return useQuery({
    queryKey: ["launch", "options", [...topics].sort()],
    queryFn: () => client.getLaunchOptions({ topics: [...topics] }),
    enabled: topics.length > 0,
    placeholderData: keepPreviousData,
  });
}

/** What this launch does about the earlier commissions that block it, and whether the operator says an unknown packet never started. */
export interface Decisions {
  choices: Record<string, BlockerChoice | "">;
  assertNeverStarted?: boolean;
}

function decided({ choices, assertNeverStarted = false }: Decisions) {
  return { ...blockerLists(choices), assertNeverStarted };
}

/** packet.py preview of a spec, judged with the operator's choices: renders in memory and writes nothing. */
export function usePreflightSpec(specJson: string | null, choices: Decisions["choices"]) {
  const asked = decided({ choices });
  return useQuery({
    queryKey: ["launch", "preflight", "spec", specJson, asked.supersedes, asked.runAlongside],
    queryFn: () => client.preflight({ target: { case: "specJson", value: specJson ?? "" }, ...asked }),
    enabled: specJson !== null,
    placeholderData: keepPreviousData,
  });
}

/** packet.py check of a built commission folder, judged with the operator's choices; writes nothing. */
export function usePreflightPrepared(commissionId: string, decisions: Decisions, enabled = true) {
  const asked = decided(decisions);
  return useQuery({
    queryKey: ["launch", "preflight", "commission", commissionId, asked.supersedes, asked.runAlongside, asked.assertNeverStarted],
    queryFn: () => client.preflight({ target: { case: "commissionId", value: commissionId }, ...asked }),
    enabled,
    placeholderData: keepPreviousData,
  });
}

/** The charter path and the two texts to paste for a manual launch: a read. */
export function useLaunchMessages(commissionId: string, enabled: boolean) {
  return useQuery({
    queryKey: ["launch", "messages", commissionId],
    queryFn: () => client.getLaunchMessages({ commissionId }),
    enabled,
  });
}

export function useLaunchRecord(commissionId: string) {
  return useQuery({
    queryKey: ["launch", "record", commissionId],
    queryFn: () => client.getLaunch({ commissionId }),
    refetchInterval: (query) => (query.state.data?.launch?.running ? 2_000 : 30_000),
  });
}

export function useLaunchTest() {
  return useQuery({ queryKey: ["launch", "test"], queryFn: () => client.getLaunchTest({}), refetchInterval: (q) => (q.state.data?.running ? 3_000 : false) });
}

export const launchApi = {
  /** Builds a packet; the choices are written into its charter ("Supersedes …", "Runs alongside …"), so its launch needs none. */
  buildFromSpec: (specJson: string, choices: Decisions["choices"] = {}) => client.build({ source: { case: "specJson", value: specJson }, ...blockerLists(choices) }),
  relaunch: (commissionId: string) => client.build({ source: { case: "relaunchOf", value: commissionId } }),
  /**
   * Starts the automatic launch. A packet built with its choices needs none;
   * one launched as it is carries them here, with the never-started
   * assertion for an unknown packet.
   */
  launch: (commissionId: string, decisions: Decisions = { choices: {} }) => client.launch({ commissionId, ...decided(decisions) }),
  control: (commissionId: string, action: ControlAction, reason = "") => client.control({ commissionId, action, reason }),
  testLaunch: () => client.testLaunch({}),
};

/** What a launch does about an earlier commission still holding one of its topics. */
export type BlockerChoice = "supersede" | "alongside";

function fromResolution(r: Resolution): BlockerChoice | null {
  return r === Resolution.SUPERSEDE ? "supersede" : r === Resolution.RUN_ALONGSIDE ? "alongside" : null;
}

/** The operator's choices as the request carries them: the commissions to supersede and to run alongside, each sorted. */
export function blockerLists(choices: Record<string, BlockerChoice | "">): { supersedes: string[]; runAlongside: string[] } {
  const named = (want: BlockerChoice) =>
    Object.entries(choices)
      .filter(([, choice]) => choice === want)
      .map(([commissionId]) => commissionId)
      .sort((a, b) => a.localeCompare(b));
  return { supersedes: named("supersede"), runAlongside: named("alongside") };
}

/** One earlier commission this launch must decide about, across every topic it holds. */
export interface Blocker {
  commissionId: string;
  /** The verdict's state word: live, paused, limited or unknown. */
  state: string;
  topics: string[];
  /** The API's own sentence for the first topic, kept for "Why?". */
  message: string;
  choices: BlockerChoice[];
  /** The desk's choice, already selected; null when the operator must choose (a live supervisor). */
  recommended: BlockerChoice | null;
  /** Why the desk recommends what it does, or why it does not. */
  reason: string;
}

/**
 * The earlier commissions this launch decides about: every verdict that
 * offers choices, once per commission with the topics it shares.
 * A commission already decided stays listed, so the choice can be changed;
 * a quiet one offers no choices and is not listed here.
 */
export function blockersOf(result: PreflightResponse): Blocker[] {
  const out = new Map<string, Blocker>();
  for (const v of result.topics) {
    const choices = v.choices.map(fromResolution).filter((c): c is BlockerChoice => c !== null);
    if (choices.length === 0) continue;
    const found = out.get(v.commissionId);
    if (found) {
      if (!found.topics.includes(v.topic)) found.topics.push(v.topic);
      continue;
    }
    out.set(v.commissionId, {
      commissionId: v.commissionId,
      state: v.state,
      topics: [v.topic],
      message: v.message,
      choices,
      recommended: fromResolution(v.recommended),
      reason: v.recommendedReason,
    });
  }
  return [...out.values()];
}

/** A stance quote the desk suggests, with the file it came from. */
export interface SuggestedQuote {
  text: string;
  source: string;
}

/** The quotes the desk suggests for the topics asked about, and why there are none when there are none. */
export function suggestedQuotes(suggestions: GetLaunchOptionsResponse | undefined): { quotes: SuggestedQuote[]; reason: string } {
  return {
    quotes: (suggestions?.stanceQuotes ?? []).map((q) => ({ text: q.text, source: q.source })),
    reason: suggestions?.stanceQuotesReason ?? "",
  };
}

/** A workstream as the operator reads it: its title beside its ID, else its showcase line. */
export function workstreamWords(w: Workstream): { id: string; title: string } {
  return { id: w.id, title: w.title || w.showcase };
}

export { ControlAction };
export { Resolution };
export type {
  Gate,
  GetLaunchOptionsResponse,
  LaunchRecord,
  PacketLimits,
  PacketResult,
  PreflightResponse,
  StateExcerpt,
  TopicVerdict,
} from "@vrooli/proto-types/research-desk/v1/launch/launch_pb";
