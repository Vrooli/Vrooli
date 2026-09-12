import { beforeEach, describe, expect, it, vi } from "vitest";
import { fetchHealth } from "../health";
import { uploadFile } from "../uploads";
import { getTTSHookStatus, recordTTSHookAck, recordTTSPlaybackEvent, updateTTSHookConfig } from "../ttsHook";
import { decodeApiError, makeApiError } from "../client";

describe("REST exception wrappers", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("reads and updates TTS hook state", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue({ ok: true, json: async () => ({ config: { autoEnabled: true, backend: "auto", startMuted: true } }) } as Response);
    await expect(getTTSHookStatus()).resolves.toMatchObject({ config: { autoEnabled: true } });
    await expect(updateTTSHookConfig({ autoEnabled: false })).resolves.toMatchObject({ config: { autoEnabled: true } });
    await recordTTSHookAck({ eventId: "e", source: "ui", sessionId: "s", stage: "start" });
    await recordTTSPlaybackEvent({ source: "ui", stage: "stop" });
    expect(fetchMock).toHaveBeenCalledTimes(4);
    fetchMock.mockResolvedValue({ ok: false, status: 500, statusText: "bad", text: async () => "no" } as Response);
    await expect(getTTSHookStatus()).rejects.toThrow("tts-hook 500: no");
    await expect(recordTTSHookAck({ eventId: "e", source: "ui", sessionId: "s", stage: "error" })).rejects.toThrow("tts-hook/ack 500: no");
  });

  it("fetches health and uploads a file", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue({ ok: true, json: async () => ({ status: "ok", version: "1" }) } as Response);
    await expect(fetchHealth()).resolves.toBeTruthy();
    fetchMock.mockResolvedValue({ ok: true, json: async () => ({ path: "/tmp/a.png" }) } as Response);
    await expect(uploadFile("s1", new Blob(["x"], { type: "text/plain" }), "a.txt")).resolves.toBe("/tmp/a.png");
    fetchMock.mockResolvedValue({ ok: false, status: 413, statusText: "large", text: async () => JSON.stringify({ error: "too large" }) } as Response);
    await expect(uploadFile("s1", new Blob(["x"]))).rejects.toBeTruthy();
  });

  it("preserves structured API errors and supplies a safe fallback envelope", async () => {
    const structured = makeApiError("not_found", "missing", 404);
    expect(structured).toMatchObject({ name: "ApiError", code: "not_found", status: 404, message: "not_found: missing" });
    await expect(decodeApiError({ status: 409, json: async () => ({ code: "conflict", message: "already exists" }) } as Response)).resolves.toMatchObject({ code: "conflict", status: 409 });
    await expect(decodeApiError({ status: 502, json: async () => { throw new Error("invalid json"); } } as unknown as Response)).resolves.toMatchObject({ code: "internal", status: 502 });
  });
});
