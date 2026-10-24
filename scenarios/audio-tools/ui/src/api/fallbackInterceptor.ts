import type { Interceptor } from "@connectrpc/connect";

import { pushToast } from "../components/ui/toast";

/**
 * Connect interceptor that observes the `x-audio-tools-fallback` response
 * header emitted by audio-tools STT/TTS/Summarize handlers when a request
 * was served from a non-primary provider tier.
 *
 * It surfaces a toast and debounces per capability so a flapping local
 * provider can't spam the UI. The debounce is in-memory (Map<capability,
 * lastFiredMs>) which is intentional — fallback state is ephemeral and
 * shouldn't persist across reloads.
 */

const FALLBACK_HEADER = "x-audio-tools-fallback";
const DEBOUNCE_MS = 60_000;

export interface FallbackToken {
  from: string;
  to: string;
  reason: string;
}

export function parseFallbackHeader(raw: string | null | undefined): FallbackToken | null {
  if (!raw) return null;
  const parts: Record<string, string> = {};
  for (const seg of raw.split(";")) {
    const [k, v] = seg.split("=", 2);
    if (!k || v === undefined) continue;
    parts[k.trim().toLowerCase()] = v.trim();
  }
  if (!parts.from || !parts.to) return null;
  return { from: parts.from, to: parts.to, reason: parts.reason ?? "" };
}

// Capability is the lowercase RPC method's parent service slug.
// Connect service paths look like "/vrooli.audio_tools.v1.stt.STTService/Transcribe".
export function capabilityFromServicePath(path: string | undefined): string | null {
  if (!path) return null;
  const m = /\.(stt|tts|summarize)\./i.exec(path);
  const grp = m?.[1];
  return grp ? grp.toLowerCase() : null;
}

export interface InterceptorOptions {
  now?: () => number;
  notify?: (capability: string, token: FallbackToken) => void;
}

export function createFallbackInterceptor(opts: InterceptorOptions = {}): Interceptor {
  const now = opts.now ?? (() => Date.now());
  const notify = opts.notify ?? defaultNotify;
  const lastFired = new Map<string, number>();

  function maybeFire(capability: string, token: FallbackToken) {
    const t = now();
    const prev = lastFired.get(capability) ?? 0;
    if (t - prev < DEBOUNCE_MS) return;
    lastFired.set(capability, t);
    notify(capability, token);
  }

  return (next) => async (req) => {
    const res = await next(req);
    try {
      const header = res.header.get(FALLBACK_HEADER) ?? res.trailer.get(FALLBACK_HEADER);
      const token = parseFallbackHeader(header);
      if (token) {
        const cap =
          capabilityFromServicePath(res.service.typeName) ??
          capabilityFromServicePath(req.service.typeName) ??
          "unknown";
        maybeFire(cap, token);
      }
    } catch {
      // Header inspection must never break the response path.
    }
    return res;
  };
}

function defaultNotify(capability: string, token: FallbackToken) {
  const upper = capability.toUpperCase();
  const fromUpper = token.from.toUpperCase();
  const toUpper = token.to.toUpperCase();
  const reason = token.reason ? ` (${token.reason})` : "";
  pushToast({
    title: `${upper} fell back to ${toUpper}`,
    body: `Request fell back from ${fromUpper} to ${toUpper}${reason}.`,
    href: `/status#${capability}`,
    hrefLabel: "View status",
  });
}
