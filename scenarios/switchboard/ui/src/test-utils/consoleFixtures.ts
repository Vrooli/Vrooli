/**
 * Typed fixtures for the console API plus a `fetch` stub that answers by
 * path. Tests describe the world in one object and the pages render against
 * it, so a shape change in `api/console.ts` fails here first.
 */
import { vi } from "vitest";

import type { Agent, AgentDetail, ChannelListing, Contact, ContactDetail, Gate, Overview, Thread, ThreadDetail } from "../api/console";

const NOW = new Date("2026-09-01T12:00:00Z");
const iso = (minutesAgo: number) => new Date(NOW.getTime() - minutesAgo * 60_000).toISOString();

export function makeGate(overrides: Partial<Gate> = {}): Gate {
  return {
    id: "gate-1",
    thread_id: "thread-1",
    owner_id: "owner",
    scope: "calendar.write",
    withheld: "Adding the dentist appointment",
    unblock: "Grant calendar.write once",
    created_at: iso(5),
    expires_at: new Date(Date.now() + 20 * 60_000).toISOString(),
    status: "pending",
    grant_once: true,
    ...overrides,
  };
}

export function makeThread(overrides: Partial<Thread> = {}): Thread {
  return {
    id: "thread-1",
    channel_id: "in-app",
    channel_display_name: "Switchboard app",
    channel_accent: "#0891b2",
    thread_key: "5f2c",
    is_group: false,
    agent_id: "household-planner",
    agent_display_name: "Household Planner",
    ceiling_tier: "owner",
    participant_count: 1,
    message_count: 2,
    pending_gates: 0,
    last_message: { text: "Added to the calendar.", author_kind: "agent", sender_address: "agent", received_at: iso(3) },
    budget: { thread_id: "thread-1", channel_id: "in-app", thread_key: "5f2c", agent_id: "household-planner", turn_budget: 20, used: 2, spend_cap_cents: 0, spent_cents: 0, window_started_at: iso(30), exhausted: false },
    created_at: iso(60),
    updated_at: iso(3),
    ...overrides,
  };
}

export function makeThreadDetail(overrides: Partial<ThreadDetail> = {}): ThreadDetail {
  const thread = overrides.thread ?? makeThread();
  return {
    thread,
    messages: [
      { id: 1, remote_id: "m1", author_kind: "human", sender_address: "owner", text: "Book the dentist for Tuesday", received_at: iso(4) },
      { id: 2, remote_id: "m2", author_kind: "agent", sender_address: "agent", text: "Added to the calendar.", received_at: iso(3) },
    ],
    participants: [{ contact_id: "c-owner", address: "owner", display_name: "You", tier: "owner" }],
    gates: [],
    ...overrides,
  };
}

export function makeAgent(overrides: Partial<Agent> = {}): Agent {
  return {
    id: "household-planner",
    display_name: "Household Planner",
    description: "Keeps the family calendar.",
    status: "active",
    appearance: { body: "#1E3A8A", head: "#2563EB", accent: "#BFDBFE" },
    tags: ["home"],
    bindings: [{ id: "b1", channel_id: "in-app", channel_display_name: "Switchboard app", channel_accent: "#0891b2", address: "owner", thread_key: "5f2c", live: true }],
    grant: { scopes: ["read"], owner_only: ["owner"], source: "default" },
    activity: { turns_24h: 4, refusals_24h: 1, threads: 1 },
    ...overrides,
  };
}

export function makeAgentDetail(overrides: Partial<AgentDetail> = {}): AgentDetail {
  return {
    ...makeAgent(),
    activity_log: [{ kind: "turn", thread_id: "thread-1", channel_id: "in-app", text: "Added to the calendar.", at: iso(3) }],
    ...overrides,
  };
}

export function makeContact(overrides: Partial<Contact> = {}): Contact {
  return {
    id: "c-sam",
    channel_id: "telegram",
    channel_display_name: "Telegram",
    channel_accent: "#2AABEE",
    address: "@sam",
    display_name: "Sam",
    tier: "known",
    first_seen: iso(600),
    last_seen: iso(10),
    message_count: 12,
    room_count: 2,
    ...overrides,
  };
}

export function makeContactDetail(overrides: Partial<ContactDetail> = {}): ContactDetail {
  return {
    contact: makeContact(),
    rooms: [
      { thread_id: "thread-2", channel_id: "telegram", channel_display_name: "Telegram", thread_key: "chat-9", is_group: true, ceiling_tier: "known", participant_count: 3 },
      { thread_id: "thread-3", channel_id: "telegram", channel_display_name: "Telegram", thread_key: "chat-1", is_group: false, ceiling_tier: "known", participant_count: 1 },
    ],
    ...overrides,
  };
}

export function makeChannelListing(overrides: Partial<ChannelListing> & { id?: string } = {}): ChannelListing {
  const { id = "in-app", ...rest } = overrides;
  return {
    descriptor: { id, displayName: id === "in-app" ? "Switchboard app" : id, accent: "#0891b2", cost: "free", supports: { text: true, images: true, files: true, groups: true, threads: true }, limits: { maxTextBytes: 100000 }, setup: { friction: 0 } },
    availability: "available",
    implemented: true,
    reason: null,
    ...rest,
  };
}

export function makeOverview(overrides: Partial<Overview> = {}): Overview {
  return {
    generated_at: iso(0),
    gates: [],
    refusals: [],
    channels: [
      { id: "in-app", display_name: "Switchboard app", accent: "#0891b2", availability: "available", implemented: true, friction: 0, bindings: 1, threads: 1 },
      { id: "telegram", display_name: "Telegram", accent: "#2AABEE", availability: "unavailable", reason: "configure a Telegram bot token", implemented: true, friction: 2, bindings: 0, threads: 0 },
    ],
    budget: { threads_under_pressure: [] },
    ...overrides,
  };
}

export type RouteBody = Response | object | unknown[];
export type RouteValue = RouteBody | ((init: RequestInit | undefined, url: URL) => RouteBody);
export type RouteTable = Record<string, RouteValue>;

/** Read a request body the way the server would; non-string bodies are not used by the console client. */
export function bodyOf(init: RequestInit | undefined): unknown {
  return typeof init?.body === "string" ? JSON.parse(init.body) : undefined;
}

/**
 * Install a `fetch` stub that resolves each console path to a fixture. Keys
 * are matched on `pathname` (query string ignored); unknown paths 404 so a
 * page that calls something unexpected fails loudly.
 */
export function stubConsoleFetch(routes: RouteTable) {
  const calls: Array<{ path: string; init?: RequestInit }> = [];
  const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
    const url = new URL(typeof input === "string" ? input : input instanceof URL ? input.href : input.url, "http://localhost");
    calls.push({ path: url.pathname, init });
    const handler = routes[url.pathname];
    if (handler === undefined) {
      return Promise.resolve(new Response("not found", { status: 404 }));
    }
    const body = typeof handler === "function" ? handler(init, url) : handler;
    if (body instanceof Response) return Promise.resolve(body);
    return Promise.resolve(new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } }));
  });
  vi.stubGlobal("fetch", fetchMock);
  return { fetchMock, calls };
}

export const defaultRoutes = (): RouteTable => ({
  "/api/v1/overview": makeOverview(),
  "/api/v1/threads": [makeThread()],
  "/api/v1/threads/thread-1": makeThreadDetail(),
  "/api/v1/agents": { source: { ok: true }, agents: [makeAgent()] },
  "/api/v1/agents/household-planner": makeAgentDetail(),
  "/api/v1/contacts": [makeContact()],
  "/api/v1/contacts/c-sam": makeContactDetail(),
  "/api/v1/channels": [makeChannelListing(), makeChannelListing({ id: "telegram", availability: "unavailable", reason: "configure a Telegram bot token", descriptor: { id: "telegram", displayName: "Telegram", accent: "#2AABEE", cost: "byok", setup: { friction: 2 }, supports: { text: true }, limits: { maxTextBytes: 4096 } } })],
  "/api/v1/gates": [],
  "/health": { status: "ok", service: "switchboard", timestamp: iso(0) },
});
