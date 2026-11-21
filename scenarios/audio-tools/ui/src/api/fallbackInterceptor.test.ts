import { describe, expect, it, vi } from "vitest";

import {
  capabilityFromServicePath,
  createFallbackInterceptor,
  parseFallbackHeader,
} from "./fallbackInterceptor";

describe("parseFallbackHeader", () => {
  it("parses from/to/reason", () => {
    expect(parseFallbackHeader("from=byok;to=vrooli;reason=provider_unavailable")).toEqual({
      from: "byok",
      to: "vrooli",
      reason: "provider_unavailable",
    });
  });

  it("tolerates missing reason", () => {
    expect(parseFallbackHeader("from=local;to=byok")).toEqual({ from: "local", to: "byok", reason: "" });
  });

  it("returns null for empty/invalid headers", () => {
    expect(parseFallbackHeader(null)).toBeNull();
    expect(parseFallbackHeader(undefined)).toBeNull();
    expect(parseFallbackHeader("")).toBeNull();
    expect(parseFallbackHeader("just=garbage")).toBeNull();
  });
});

describe("capabilityFromServicePath", () => {
  it("extracts stt/tts/summarize from typeName", () => {
    expect(capabilityFromServicePath("vrooli.audio_tools.v1.stt.STTService")).toBe("stt");
    expect(capabilityFromServicePath("vrooli.audio_tools.v1.tts.TTSService")).toBe("tts");
    expect(capabilityFromServicePath("vrooli.audio_tools.v1.summarize.SummarizeService")).toBe("summarize");
  });

  it("returns null for unknown services", () => {
    expect(capabilityFromServicePath("vrooli.audio_tools.v1.session.SessionService")).toBeNull();
    expect(capabilityFromServicePath("")).toBeNull();
  });
});

describe("createFallbackInterceptor", () => {
  function makeResponse(headerValue: string | null, typeName = "vrooli.audio_tools.v1.stt.STTService") {
    const header = new Headers();
    if (headerValue !== null) header.set("x-audio-tools-fallback", headerValue);
    return {
      stream: false as const,
      service: { typeName },
      method: { name: "Transcribe" },
      header,
      trailer: new Headers(),
      message: {},
    } as unknown as import("@connectrpc/connect").UnaryResponse;
  }

  it("fires once on first hit and debounces within window", () => {
    const notify = vi.fn();
    let now = 1_000_000;
    const interceptor = createFallbackInterceptor({ now: () => now, notify });
    const wrapped = interceptor(() => Promise.resolve(makeResponse("from=byok;to=vrooli;reason=oops")));

    const req = { service: { typeName: "vrooli.audio_tools.v1.stt.STTService" }, method: { name: "Transcribe" } } as unknown as Parameters<typeof wrapped>[0];

    return (async () => {
      await wrapped(req);
      expect(notify).toHaveBeenCalledTimes(1);
      // Second hit within 60s — no new toast.
      now += 1_000;
      await wrapped(req);
      expect(notify).toHaveBeenCalledTimes(1);
      // After the 60s debounce window — fires again.
      now += 60_000;
      await wrapped(req);
      expect(notify).toHaveBeenCalledTimes(2);
    })();
  });

  it("does not fire when header is absent", async () => {
    const notify = vi.fn();
    const interceptor = createFallbackInterceptor({ notify });
    const wrapped = interceptor(() => Promise.resolve(makeResponse(null)));
    const req = { service: { typeName: "vrooli.audio_tools.v1.stt.STTService" }, method: { name: "Transcribe" } } as unknown as Parameters<typeof wrapped>[0];
    await wrapped(req);
    expect(notify).not.toHaveBeenCalled();
  });

  it("passes capability derived from the response service typeName", async () => {
    const notify = vi.fn();
    const interceptor = createFallbackInterceptor({ notify });
    const wrapped = interceptor(() =>
      Promise.resolve(makeResponse("from=byok;to=local;reason=x", "vrooli.audio_tools.v1.summarize.SummarizeService")),
    );
    const req = { service: { typeName: "vrooli.audio_tools.v1.summarize.SummarizeService" }, method: { name: "Summarize" } } as unknown as Parameters<typeof wrapped>[0];
    await wrapped(req);
    expect(notify).toHaveBeenCalledWith("summarize", expect.objectContaining({ from: "byok", to: "local" }));
  });
});
