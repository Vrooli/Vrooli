/**
 * Covers lines 72-94: capabilityFromServicePath null → "unknown" sentinel,
 * header read from trailer fallback, and the defaultNotify path via pushToast.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

// pushToast is the side-effect in defaultNotify — stub it.
const pushToastMock = vi.fn();
vi.mock("../components/ui/toast", () => ({
  pushToast: (opts: unknown) => pushToastMock(opts),
}));

import {
  createFallbackInterceptor,
  capabilityFromServicePath,
} from "./fallbackInterceptor";

beforeEach(() => {
  vi.clearAllMocks();
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("capabilityFromServicePath — edge cases", () => {
  it("returns null for undefined", () => {
    expect(capabilityFromServicePath(undefined)).toBeNull();
  });

  it("extracts 'tts' case-insensitively", () => {
    expect(capabilityFromServicePath("vrooli.audio_tools.v1.TTS.TTSService")).toBe("tts");
  });
});

describe("createFallbackInterceptor — defaultNotify (lines 83-94)", () => {
  function makeResponse(
    headerValue: string | null,
    typeName = "",
    inTrailer = false,
  ) {
    const header = new Headers();
    const trailer = new Headers();
    if (headerValue !== null) {
      if (inTrailer) {
        trailer.set("x-audio-tools-fallback", headerValue);
      } else {
        header.set("x-audio-tools-fallback", headerValue);
      }
    }
    return {
      stream: false as const,
      service: { typeName },
      method: { name: "Op" },
      header,
      trailer,
      message: {},
    } as unknown as import("@connectrpc/connect").UnaryResponse;
  }

  it("calls pushToast with capability/from/to/reason when no custom notify is provided", async () => {
    const interceptor = createFallbackInterceptor();
    const headerVal = "from=byok;to=vrooli;reason=provider_unavailable";
    const wrapped = interceptor(
      () =>
        Promise.resolve(
          makeResponse(headerVal, "vrooli.audio_tools.v1.stt.STTService"),
        ),
    );
    const req = {
      service: { typeName: "vrooli.audio_tools.v1.stt.STTService" },
      method: { name: "Transcribe" },
    } as unknown as Parameters<typeof wrapped>[0];
    await wrapped(req);
    expect(pushToastMock).toHaveBeenCalledTimes(1);
    const call = pushToastMock.mock.calls[0]![0] as {
      title: string;
      body: string;
      href: string;
      hrefLabel: string;
    };
    expect(call.title).toContain("STT");
    expect(call.title).toContain("VROOLI");
    expect(call.body).toContain("BYOK");
    expect(call.body).toContain("provider_unavailable");
    expect(call.href).toBe("/status#stt");
    expect(call.hrefLabel).toBe("View status");
  });

  it("omits the reason parenthetical when reason is empty", async () => {
    const interceptor = createFallbackInterceptor();
    const wrapped = interceptor(
      () =>
        Promise.resolve(
          makeResponse("from=byok;to=local", "vrooli.audio_tools.v1.tts.TTSService"),
        ),
    );
    const req = {
      service: { typeName: "vrooli.audio_tools.v1.tts.TTSService" },
      method: { name: "Synthesize" },
    } as unknown as Parameters<typeof wrapped>[0];
    await wrapped(req);
    const call = pushToastMock.mock.calls[0]![0] as { body: string };
    // No trailing "()" from empty reason
    expect(call.body).not.toMatch(/\(\)/);
  });

  it("reads the fallback header from the trailer when absent in headers", async () => {
    const interceptor = createFallbackInterceptor({ notify: pushToastMock });
    const wrapped = interceptor(
      () =>
        Promise.resolve(
          makeResponse(
            "from=local;to=vrooli;reason=offline",
            "vrooli.audio_tools.v1.summarize.SummarizeService",
            true, // put header in trailer
          ),
        ),
    );
    const req = {
      service: { typeName: "vrooli.audio_tools.v1.summarize.SummarizeService" },
      method: { name: "Summarize" },
    } as unknown as Parameters<typeof wrapped>[0];
    await wrapped(req);
    expect(pushToastMock).toHaveBeenCalledTimes(1);
    expect(pushToastMock).toHaveBeenCalledWith(
      "summarize",
      expect.objectContaining({ from: "local", to: "vrooli", reason: "offline" }),
    );
  });

  it("falls back to 'unknown' capability when service typeName has no STT/TTS/summarize segment", async () => {
    const interceptor = createFallbackInterceptor({ notify: pushToastMock });
    // Neither res.service.typeName nor req.service.typeName matches → "unknown"
    const wrapped = interceptor(
      () =>
        Promise.resolve(
          makeResponse("from=byok;to=local;reason=x", ""),
        ),
    );
    const req = {
      service: { typeName: "" },
      method: { name: "Op" },
    } as unknown as Parameters<typeof wrapped>[0];
    await wrapped(req);
    expect(pushToastMock).toHaveBeenCalledWith(
      "unknown",
      expect.objectContaining({ from: "byok" }),
    );
  });

  it("derives capability from req.service.typeName when res.service.typeName is empty", async () => {
    const interceptor = createFallbackInterceptor({ notify: pushToastMock });
    // res has no typeName match; req does
    const wrapped = interceptor(
      () => Promise.resolve(makeResponse("from=byok;to=local", "")),
    );
    const req = {
      service: { typeName: "vrooli.audio_tools.v1.tts.TTSService" },
      method: { name: "Synthesize" },
    } as unknown as Parameters<typeof wrapped>[0];
    await wrapped(req);
    expect(pushToastMock).toHaveBeenCalledWith(
      "tts",
      expect.objectContaining({ from: "byok" }),
    );
  });

  it("swallows header inspection errors without breaking the response", async () => {
    const interceptor = createFallbackInterceptor({ notify: pushToastMock });
    // Return a response whose header.get throws
    const badResponse = {
      stream: false as const,
      service: { typeName: "vrooli.audio_tools.v1.stt.STTService" },
      method: { name: "Transcribe" },
      header: {
        get: () => {
          throw new Error("header-error");
        },
      },
      trailer: new Headers(),
      message: {},
    } as unknown as import("@connectrpc/connect").UnaryResponse;

    const wrapped = interceptor(() => Promise.resolve(badResponse));
    const req = {
      service: { typeName: "vrooli.audio_tools.v1.stt.STTService" },
      method: { name: "Transcribe" },
    } as unknown as Parameters<typeof wrapped>[0];
    // Should not throw
    await expect(wrapped(req)).resolves.toBe(badResponse);
    // No notify fired because exception was caught
    expect(pushToastMock).not.toHaveBeenCalled();
  });
});
