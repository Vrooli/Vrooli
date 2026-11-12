import { vi } from "vitest";

/** One request the desk sent: its path and decoded JSON body. */
export interface WireCall {
  path: string;
  body: unknown;
}

function decode(body: BodyInit | null | undefined): unknown {
  if (body === undefined || body === null) return undefined;
  // Connect sends JSON as bytes; jsdom and Node have separate Uint8Array realms.
  if (typeof body === "string") return body ? JSON.parse(body) : undefined;
  if (ArrayBuffer.isView(body)) return JSON.parse(new TextDecoder().decode(body));
  throw new Error("stubWire: the desk sends JSON strings or bytes only");
}

/** A Connect error answer: return one from a `stubWire` reply to refuse that request. */
export class WireError {
  readonly code: string;
  readonly message: string;
  readonly status: number;
  constructor(code: string, message: string, status = 400) {
    this.code = code;
    this.message = message;
    this.status = status;
  }
}

/**
 * Replaces `fetch` so tests see exactly what the desk would send, and answer
 * with `reply(path, body)` (an empty message by default). Nothing reaches a
 * real API, so a test can drive an outward action to its request safely.
 */
export function stubWire(reply: (path: string, body: unknown) => unknown = () => ({})): WireCall[] {
  const calls: WireCall[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
      const path = new URL(url, "http://desk.test").pathname;
      const body = decode(init?.body);
      calls.push({ path, body });
      const out = reply(path, body);
      if (out instanceof WireError) {
        return Promise.resolve(new Response(JSON.stringify({ code: out.code, message: out.message }), { status: out.status, headers: { "Content-Type": "application/json" } }));
      }
      return Promise.resolve(new Response(typeof out === "string" ? out : JSON.stringify(out ?? {}), { status: 200, headers: { "Content-Type": "application/json" } }));
    }),
  );
  return calls;
}
