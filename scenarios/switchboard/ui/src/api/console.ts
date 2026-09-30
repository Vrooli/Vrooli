/**
 * Console read/write API for the operator surfaces.
 *
 * Every shape here mirrors a JSON endpoint mounted by `api/handlers/console`.
 * Field names are snake_case on the wire and are kept that way here so a
 * response can be compared against a curl output without a mental mapping.
 */
import { buildApiUrl, buildWsUrl } from "@vrooli/api-base";

import { API_BASE } from "./client";
import { authHeaders } from "./session";

export type TrustTier = "stranger" | "known" | "trusted" | "owner";
export const TRUST_TIERS: readonly TrustTier[] = ["stranger", "known", "trusted", "owner"];

export type AuthorKind = "human" | "agent";
export type Availability = "available" | "unavailable" | "unknown";
export type GateStatus = "pending" | "granted" | "denied" | "expired";

export interface Gate {
  id: string;
  thread_id: string;
  owner_id: string;
  scope: string;
  withheld: string;
  unblock: string;
  created_at: string;
  expires_at: string;
  status: GateStatus;
  grant_once: boolean;
}

export interface ThreadBudget {
  thread_id: string;
  channel_id: string;
  thread_key: string;
  agent_id: string;
  turn_budget: number;
  used: number;
  spend_cap_cents: number;
  spent_cents: number;
  window_started_at: string;
  exhausted: boolean;
}

export interface Refusal {
  thread_id: string;
  channel_id: string;
  channel_display_name?: string;
  channel_accent?: string;
  thread_key: string;
  sender_address: string;
  agent_id: string;
  reason: string;
  at: string;
}

export interface ChannelHealth {
  id: string;
  display_name: string;
  accent?: string;
  availability: Availability;
  reason?: string | null;
  implemented: boolean;
  friction: number;
  bindings: number;
  threads: number;
}

export interface Overview {
  generated_at: string;
  gates: Gate[];
  refusals: Refusal[];
  channels: ChannelHealth[];
  budget: { threads_under_pressure: ThreadBudget[] };
}

export interface MessageMedia {
  name: string;
  mime: string;
  size: number;
  url?: string;
}

export interface LastMessage {
  text: string;
  author_kind: AuthorKind;
  sender_address: string;
  display_name?: string;
  received_at: string;
}

export interface Thread {
  id: string;
  channel_id: string;
  channel_display_name?: string;
  channel_accent?: string;
  thread_key: string;
  is_group: boolean;
  agent_id: string;
  agent_display_name?: string;
  ceiling_tier: TrustTier;
  participant_count: number;
  message_count: number;
  pending_gates: number;
  last_message: LastMessage | null;
  budget: ThreadBudget;
  created_at: string;
  updated_at: string;
}

export interface Message {
  id: number | string;
  remote_id: string;
  author_kind: AuthorKind;
  sender_address: string;
  display_name?: string;
  text: string;
  reply_to_remote_id?: string;
  received_at: string;
  media?: MessageMedia[] | null;
}

export interface Participant {
  contact_id: string;
  address: string;
  display_name?: string;
  tier: TrustTier;
}

export interface ThreadDetail {
  thread: Thread;
  messages: Message[];
  participants: Participant[];
  gates: Gate[];
  run_id?: string;
}

export interface Appearance {
  body: string;
  head: string;
  accent: string;
}

export interface AgentBinding {
  id: string;
  channel_id: string;
  channel_display_name?: string;
  channel_accent?: string;
  address: string;
  thread_key: string;
  live: boolean;
}

export interface AgentGrant {
  scopes: string[];
  program_bindings?: string[];
  owner_only: string[];
  source: "descriptor" | "default";
}

export interface Agent {
  id: string;
  display_name: string;
  description?: string;
  status?: string;
  appearance?: Appearance | null;
  tags?: string[];
  bindings: AgentBinding[];
  grant: AgentGrant;
  activity?: { turns_24h: number; refusals_24h: number; threads: number };
  broken?: string;
}

export interface AgentActivityEntry {
  kind: "turn" | "refusal" | "suppressed" | "gate";
  thread_id: string;
  channel_id: string;
  text?: string;
  reason?: string;
  at: string;
}

export interface AgentRoster {
  source: { ok: boolean; reason?: string };
  agents: Agent[];
}

export interface AgentDetail extends Agent {
  activity_log: AgentActivityEntry[];
}

export interface AgentDraft {
  display_name: string;
  description: string;
  scopes: string[];
  owner_only_scopes: string[];
}

export interface Contact {
  id: string;
  channel_id: string;
  channel_display_name?: string;
  channel_accent?: string;
  address: string;
  display_name?: string;
  tier: TrustTier;
  first_seen: string;
  last_seen: string;
  message_count: number;
  room_count: number;
}

export interface ContactRoom {
  thread_id: string;
  channel_id: string;
  channel_display_name?: string;
  channel_accent?: string;
  thread_key: string;
  is_group: boolean;
  ceiling_tier: TrustTier;
  participant_count: number;
}

export interface ContactDetail {
  contact: Contact;
  rooms: ContactRoom[];
}

export interface TierChange {
  contact: Contact;
  affected_rooms: { thread_id: string; channel_id: string; thread_key: string; previous_ceiling: TrustTier; new_ceiling: TrustTier }[];
}

export interface ChannelListing {
  descriptor: {
    id: string;
    displayName: string;
    transport?: string;
    accent?: string;
    cost?: string;
    supports?: { text?: boolean; images?: boolean; files?: boolean; groups?: boolean; threads?: boolean };
    limits?: { maxTextBytes?: number; maxMediaBytes?: number };
    setup: { friction: number };
    requires?: { key: string; description: string }[];
  };
  availability: Availability;
  reason?: string | null;
  implemented: boolean;
}

export class ConsoleApiError extends Error {
  readonly status: number;
  constructor(message: string, status: number) {
    super(message);
    this.name = "ConsoleApiError";
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  headers.set("Accept", "application/json");
  for (const [key, value] of Object.entries(authHeaders())) headers.set(key, value);
  if (init?.body) headers.set("Content-Type", "application/json");
  const res = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), {
    cache: "no-store",
    credentials: "same-origin",
    ...init,
    headers,
  });
  if (!res.ok) {
    let detail = "";
    try {
      detail = (await res.text()).trim();
    } catch {
      detail = "";
    }
    throw new ConsoleApiError(detail || `${res.status} ${res.statusText}`, res.status);
  }
  if (res.status === 204 || res.headers.get("Content-Length") === "0") return undefined as T;
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export const consoleApi = {
  overview: (signal?: AbortSignal) => request<Overview>("/api/v1/overview", { signal }),
  threads: (signal?: AbortSignal) => request<Thread[]>("/api/v1/threads", { signal }),
  thread: (id: string, signal?: AbortSignal) => request<ThreadDetail>(`/api/v1/threads/${encodeURIComponent(id)}`, { signal }),
  startInAppThread: (agentId: string) =>
    request<{ thread_id: string; thread_key: string; channel_id: string }>("/api/v1/channels/in-app/threads", {
      method: "POST",
      body: JSON.stringify({ agent_id: agentId }),
    }),
  agents: (signal?: AbortSignal) => request<AgentRoster>("/api/v1/agents", { signal }),
  agent: (id: string, signal?: AbortSignal) => request<AgentDetail>(`/api/v1/agents/${encodeURIComponent(id)}`, { signal }),
  draftAgent: (description: string) => request<AgentDraft>("/api/v1/agents/draft", { method: "POST", body: JSON.stringify({ description }) }),
  createAgent: (draft: AgentDraft) => request<{ id: string; display_name: string }>("/api/v1/agents", { method: "POST", body: JSON.stringify(draft) }),
  contacts: (signal?: AbortSignal) => request<Contact[]>("/api/v1/contacts", { signal }),
  contact: (id: string, signal?: AbortSignal) => request<ContactDetail>(`/api/v1/contacts/${encodeURIComponent(id)}`, { signal }),
  updateContact: (id: string, patch: { tier?: TrustTier; display_name?: string }) =>
    request<TierChange>(`/api/v1/contacts/${encodeURIComponent(id)}`, { method: "PUT", body: JSON.stringify(patch) }),
  gates: (status: GateStatus = "pending", signal?: AbortSignal) => request<Gate[]>(`/api/v1/gates?status=${status}`, { signal }),
  answerGate: (id: string, grant: boolean) =>
    request<Gate>(`/api/v1/gates/${encodeURIComponent(id)}/answer`, { method: "POST", body: JSON.stringify({ grant }) }),
  channels: (signal?: AbortSignal) => request<ChannelListing[]>("/api/v1/channels", { signal }),
  createBinding: (input: { agentId: string; channelId: string; address: string; threadKey?: string }) =>
    request<unknown>("/vrooli.switchboard.v1.channels.ChannelService/CreateBinding", {
      method: "POST",
      body: JSON.stringify({ agentId: input.agentId, channelId: input.channelId, address: input.address, threadKey: input.threadKey ?? "" }),
    }),
};

/** WebSocket URL for the in-app channel adapter, scoped to one thread key. */
export function inAppSocketUrl(threadKey: string): string {
  return buildWsUrl(`/api/v1/channels/socket?thread_key=${encodeURIComponent(threadKey)}`);
}

/** Query keys, kept in one place so invalidation never drifts from fetching. */
export const consoleKeys = {
  overview: ["console", "overview"] as const,
  threads: ["console", "threads"] as const,
  thread: (id: string) => ["console", "thread", id] as const,
  agents: ["console", "agents"] as const,
  agent: (id: string) => ["console", "agent", id] as const,
  contacts: ["console", "contacts"] as const,
  contact: (id: string) => ["console", "contact", id] as const,
  channels: ["console", "channels"] as const,
};
