import { describe, it, expect, vi, beforeEach } from "vitest";

const getProviderConfigRpc = vi.fn();
const listBYOKCredentialsRpc = vi.fn();
const getVoiceOverridesRpc = vi.fn();
const updateProviderConfigRpc = vi.fn();
const upsertBYOKCredentialRpc = vi.fn();
const deleteBYOKCredentialRpc = vi.fn();
const setVoiceOverrideRpc = vi.fn();

vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({
    getProviderConfig: (req: unknown) => getProviderConfigRpc(req),
    listBYOKCredentials: (req: unknown) => listBYOKCredentialsRpc(req),
    getVoiceOverrides: (req: unknown) => getVoiceOverridesRpc(req),
    updateProviderConfig: (req: unknown) => updateProviderConfigRpc(req),
    upsertBYOKCredential: (req: unknown) => upsertBYOKCredentialRpc(req),
    deleteBYOKCredential: (req: unknown) => deleteBYOKCredentialRpc(req),
    setVoiceOverride: (req: unknown) => setVoiceOverrideRpc(req),
  }),
}));

import { ApiError } from "../api/client";
import {
  getProviderConfig,
  listByokCredentials,
  getVoiceOverrides,
  updateProviderConfig,
  upsertByokCredential,
  deleteByokCredential,
  setVoiceOverride,
  normalizeConnectError,
} from "./settings";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("getProviderConfig", () => {
  it("decodes a populated config", async () => {
    getProviderConfigRpc.mockResolvedValue({
      config: {
        byokEnabled: true,
        vrooliEnabled: false,
        localEnabled: true,
        whisperUrl: "http://whisper",
        kokoroUrl: "",
        ollamaUrl: "http://ollama",
      },
    });
    const r = await getProviderConfig();
    expect(r.ok).toBe(true);
    if (r.ok) {
      expect(r.data).toEqual({
        byokEnabled: true,
        vrooliEnabled: false,
        localEnabled: true,
        whisperUrl: "http://whisper",
        kokoroUrl: undefined,
        ollamaUrl: "http://ollama",
      });
    }
  });

  it("defaults everything when config is absent", async () => {
    getProviderConfigRpc.mockResolvedValue({});
    const r = await getProviderConfig();
    expect(r.ok).toBe(true);
    if (r.ok) {
      expect(r.data.byokEnabled).toBe(false);
      expect(r.data.whisperUrl).toBeUndefined();
    }
  });

  it("maps a thrown connect error into a failure envelope", async () => {
    getProviderConfigRpc.mockRejectedValue({ code: "unavailable", message: "down" });
    const r = await getProviderConfig();
    expect(r.ok).toBe(false);
    if (!r.ok) {
      expect(r.error.code).toBe("unavailable");
      expect(r.error.status).toBe(500);
    }
  });
});

describe("listByokCredentials", () => {
  it("decodes credential rows with ISO timestamps", async () => {
    listBYOKCredentialsRpc.mockResolvedValue({
      credentials: [
        {
          providerId: "openai",
          capability: "stt",
          fingerprint: "abcd",
          createdAt: { seconds: 1_700_000_000n, nanos: 0 },
        },
      ],
    });
    const r = await listByokCredentials();
    expect(r.ok).toBe(true);
    if (r.ok) {
      expect(r.data[0]!.providerId).toBe("openai");
      expect(r.data[0]!.createdAt).toContain("T");
    }
  });
});

describe("getVoiceOverrides", () => {
  it("decodes override rows", async () => {
    getVoiceOverridesRpc.mockResolvedValue({
      overrides: [{ canonicalVoice: "alloy", tierProvider: "byok:openai", adapterVoice: "nova" }],
    });
    const r = await getVoiceOverrides();
    expect(r.ok).toBe(true);
    if (r.ok) {
      expect(r.data).toEqual([
        { canonicalVoice: "alloy", tierProvider: "byok:openai", adapterVoice: "nova" },
      ]);
    }
  });
});

describe("updateProviderConfig", () => {
  it("builds an update mask only from provided fields and decodes the result", async () => {
    updateProviderConfigRpc.mockResolvedValue({
      config: { byokEnabled: true, vrooliEnabled: true, localEnabled: false, whisperUrl: "http://w" },
    });
    const r = await updateProviderConfig({
      byokEnabled: true,
      vrooliEnabled: true,
      localEnabled: false,
      whisperUrl: "http://w",
      kokoroUrl: "http://k",
      ollamaUrl: "http://o",
      lpbsBaseUrl: "http://l",
    });
    const req = updateProviderConfigRpc.mock.calls[0]![0];
    expect(req.updateMask.paths).toEqual([
      "byok_enabled",
      "vrooli_enabled",
      "local_enabled",
      "whisper_url",
      "kokoro_url",
      "ollama_url",
      "lpbs_base_url",
    ]);
    expect(req.config.byokEnabled).toBe(true);
    expect(r.ok).toBe(true);
  });

  it("sends an empty mask when nothing changes", async () => {
    updateProviderConfigRpc.mockResolvedValue({ config: {} });
    await updateProviderConfig({});
    const req = updateProviderConfigRpc.mock.calls[0]![0];
    expect(req.updateMask.paths).toEqual([]);
  });

  it("propagates connect errors", async () => {
    updateProviderConfigRpc.mockRejectedValue(new Error("nope"));
    const r = await updateProviderConfig({ byokEnabled: false });
    expect(r.ok).toBe(false);
  });
});

describe("upsertByokCredential", () => {
  it("encodes the api key as a oneof and decodes the saved row", async () => {
    upsertBYOKCredentialRpc.mockResolvedValue({
      credential: {
        providerId: "openai",
        capability: "tts",
        fingerprint: "ffff",
        createdAt: { seconds: 1_700_000_000n, nanos: 0 },
      },
    });
    const r = await upsertByokCredential("openai", "tts", "sk-123");
    const req = upsertBYOKCredentialRpc.mock.calls[0]![0];
    expect(req.secret).toEqual({ case: "apiKey", value: "sk-123" });
    expect(r.ok).toBe(true);
    if (r.ok) expect(r.data.fingerprint).toBe("ffff");
  });

  it("fails when no credential is returned", async () => {
    upsertBYOKCredentialRpc.mockResolvedValue({});
    const r = await upsertByokCredential("openai", "tts", "sk");
    expect(r.ok).toBe(false);
    if (!r.ok) expect(r.error.code).toBe("internal");
  });
});

describe("deleteByokCredential", () => {
  it("calls delete with the identifying pair", async () => {
    deleteBYOKCredentialRpc.mockResolvedValue({});
    const r = await deleteByokCredential("openai", "stt");
    expect(deleteBYOKCredentialRpc).toHaveBeenCalledWith({ providerId: "openai", capability: "stt" });
    expect(r.ok).toBe(true);
  });

  it("maps errors", async () => {
    deleteBYOKCredentialRpc.mockRejectedValue(new Error("x"));
    const r = await deleteByokCredential("a", "b");
    expect(r.ok).toBe(false);
  });
});

describe("setVoiceOverride", () => {
  it("sends the override and decodes the new list", async () => {
    setVoiceOverrideRpc.mockResolvedValue({
      overrides: [{ canonicalVoice: "alloy", tierProvider: "local", adapterVoice: "af" }],
    });
    const r = await setVoiceOverride("alloy", "local", "af");
    expect(setVoiceOverrideRpc).toHaveBeenCalledWith({
      override: { canonicalVoice: "alloy", tierProvider: "local", adapterVoice: "af" },
    });
    expect(r.ok).toBe(true);
    if (r.ok) expect(r.data).toHaveLength(1);
  });
});

describe("normalizeConnectError", () => {
  it("returns an ApiError unchanged", () => {
    const e = new ApiError({ code: "x", message: "y" } as never, 418);
    expect(normalizeConnectError(e)).toBe(e);
  });

  it("maps unimplemented to status 501", () => {
    const out = normalizeConnectError({ code: "unimplemented", message: "no" });
    expect(out.status).toBe(501);
    expect(out.code).toBe("unimplemented");
  });

  it("defaults to internal/500 for a code-less throw", () => {
    const out = normalizeConnectError(new Error("plain"));
    expect(out.code).toBe("internal");
    expect(out.status).toBe(500);
    expect(out.message).toContain("plain");
  });

  it("stringifies a numeric code", () => {
    const out = normalizeConnectError({ code: 14, message: "n" });
    expect(out.code).toBe("14");
  });

  it("stringifies a non-Error throw", () => {
    const out = normalizeConnectError("raw");
    expect(out.message).toContain("raw");
    expect(out.code).toBe("internal");
  });
});
