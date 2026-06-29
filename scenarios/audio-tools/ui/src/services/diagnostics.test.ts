import { describe, it, expect, vi, beforeEach } from "vitest";

import { ProviderTier } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";
import {
  Capability,
  SuiteOverall_Status,
} from "@vrooli/proto-types/audio-tools/v1/diagnostics/diagnostics_pb";
import { SummarizeLevel } from "@vrooli/proto-types/audio-tools/v1/summarize/summarize_pb";

const summarizeRpc = vi.fn();
const runSuiteRpc = vi.fn();
const getLastRunRpc = vi.fn();

vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({
    summarize: (req: unknown) => summarizeRpc(req),
    runSuite: (req: unknown) => runSuiteRpc(req),
    getLastRun: (req: unknown) => getLastRunRpc(req),
  }),
}));

const uploadFileMock = vi.fn();
vi.mock("../api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/client")>();
  return { ...actual, uploadFile: (path: string, fd: FormData) => uploadFileMock(path, fd) };
});

import { summarize, runSuite, getLastSuiteRun, transcribe } from "./diagnostics";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("summarize", () => {
  it("maps the requested level and decodes the response with a provider trace", async () => {
    summarizeRpc.mockResolvedValue({
      text: "short",
      promptTokens: 100,
      outputTokens: 20,
      providerTier: ProviderTier.VROOLI,
      providerId: "vrooli",
      modelId: "m1",
      latencyMs: 55,
    });
    const r = await summarize("long text", "heavy");
    const req = summarizeRpc.mock.calls[0]![0];
    expect(req.level).toBe(SummarizeLevel.HEAVY);
    expect(req.timeoutSeconds).toBe(30);
    expect(r.ok).toBe(true);
    if (r.ok) {
      expect(r.data.text).toBe("short");
      expect(r.data.trace.providerTier).toBe("vrooli");
      expect(r.data.promptTokens).toBe(100);
    }
  });

  it("defaults to the moderate level", async () => {
    summarizeRpc.mockResolvedValue({
      text: "",
      promptTokens: 0,
      outputTokens: 0,
      providerTier: ProviderTier.LOCAL,
      providerId: "",
      modelId: "",
      latencyMs: 0,
    });
    await summarize("x");
    expect(summarizeRpc.mock.calls[0]![0].level).toBe(SummarizeLevel.MODERATE);
  });

  it("maps the light level and an unknown provider tier to an empty label", async () => {
    summarizeRpc.mockResolvedValue({
      text: "",
      promptTokens: 0,
      outputTokens: 0,
      providerTier: ProviderTier.UNSPECIFIED,
      providerId: "",
      modelId: "",
      latencyMs: 0,
    });
    const r = await summarize("x", "light");
    expect(summarizeRpc.mock.calls[0]![0].level).toBe(SummarizeLevel.LIGHT);
    if (r.ok) expect(r.data.trace.providerTier).toBe("");
  });

  it("returns a failure envelope on a connect error", async () => {
    summarizeRpc.mockRejectedValue({ code: "unimplemented", message: "no summarize" });
    const r = await summarize("x");
    expect(r.ok).toBe(false);
    if (!r.ok) {
      expect(r.error.code).toBe("unimplemented");
      expect(r.error.status).toBe(501);
    }
  });
});

const protoStep = {
  capability: Capability.STT,
  ok: true,
  errorCode: "",
  errorMessage: "",
  startedAtUnixMs: 1_000n,
  finishedAtUnixMs: 1_200n,
  providerTier: ProviderTier.LOCAL,
  providerId: "whisper",
  modelId: "v3",
  latencyMs: 200,
  details: { foo: "bar" },
};

describe("runSuite", () => {
  it("encodes the requested capabilities and shapes the run", async () => {
    runSuiteRpc.mockResolvedValue({
      run: {
        runId: "r1",
        startedAtUnixMs: 1_000n,
        finishedAtUnixMs: 2_000n,
        steps: [
          protoStep,
          { ...protoStep, capability: Capability.TTS },
          { ...protoStep, capability: Capability.SUMMARIZE },
          { ...protoStep, capability: Capability.TRANSCODE },
          { ...protoStep, capability: Capability.UNSPECIFIED },
        ],
        overall: {
          status: SuiteOverall_Status.PARTIAL,
          passCount: 3,
          failCount: 1,
          totalCount: 4,
        },
      },
    });
    const r = await runSuite(["stt", "tts", "summarize", "transcode"]);
    const req = runSuiteRpc.mock.calls[0]![0];
    expect(req.capabilities).toEqual([
      Capability.STT,
      Capability.TTS,
      Capability.SUMMARIZE,
      Capability.TRANSCODE,
    ]);
    expect(r.ok).toBe(true);
    if (r.ok) {
      expect(r.data.runId).toBe("r1");
      expect(r.data.overall).toBe("partial");
      expect(r.data.passCount).toBe(3);
      expect(r.data.steps.map((s) => s.capability)).toEqual([
        "stt",
        "tts",
        "summarize",
        "transcode",
        "unknown",
      ]);
      expect(r.data.steps[0]!.providerTier).toBe("local");
      expect(r.data.steps[0]!.details).toEqual({ foo: "bar" });
      expect(r.data.startedAtUnixMs).toBe(1_000);
    }
  });

  it("maps every overall status value", async () => {
    const statuses: Array<[SuiteOverall_Status, string]> = [
      [SuiteOverall_Status.NEVER, "never"],
      [SuiteOverall_Status.PASS, "pass"],
      [SuiteOverall_Status.PARTIAL, "partial"],
      [SuiteOverall_Status.FAIL, "fail"],
      [SuiteOverall_Status.UNSPECIFIED, "unknown"],
    ];
    for (const [status, label] of statuses) {
      runSuiteRpc.mockResolvedValue({
        run: {
          runId: "r",
          startedAtUnixMs: 0n,
          finishedAtUnixMs: 0n,
          steps: [],
          overall: { status, passCount: 0, failCount: 0, totalCount: 0 },
        },
      });
      const r = await runSuite();
      if (r.ok) expect(r.data.overall).toBe(label);
    }
  });

  it("returns a 'never' empty run when the response omits the run", async () => {
    runSuiteRpc.mockResolvedValue({});
    const r = await runSuite();
    expect(r.ok).toBe(true);
    if (r.ok) {
      expect(r.data.overall).toBe("never");
      expect(r.data.steps).toEqual([]);
      expect(r.data.runId).toBe("");
    }
  });

  it("defaults overall counts when the overall block is missing", async () => {
    runSuiteRpc.mockResolvedValue({
      run: { runId: "r", startedAtUnixMs: 0n, finishedAtUnixMs: 0n, steps: [] },
    });
    const r = await runSuite();
    if (r.ok) {
      expect(r.data.overall).toBe("unknown");
      expect(r.data.passCount).toBe(0);
    }
  });

  it("maps connect errors", async () => {
    runSuiteRpc.mockRejectedValue(new Error("suite blew up"));
    const r = await runSuite(["stt"]);
    expect(r.ok).toBe(false);
  });
});

describe("getLastSuiteRun", () => {
  it("shapes the last run", async () => {
    getLastRunRpc.mockResolvedValue({
      run: {
        runId: "last",
        startedAtUnixMs: 10n,
        finishedAtUnixMs: 20n,
        steps: [],
        overall: { status: SuiteOverall_Status.PASS, passCount: 1, failCount: 0, totalCount: 1 },
      },
    });
    const r = await getLastSuiteRun();
    if (r.ok) {
      expect(r.data.runId).toBe("last");
      expect(r.data.overall).toBe("pass");
    }
  });

  it("returns the empty run when there has never been one", async () => {
    getLastRunRpc.mockResolvedValue({});
    const r = await getLastSuiteRun();
    if (r.ok) expect(r.data.overall).toBe("never");
  });

  it("maps connect errors", async () => {
    getLastRunRpc.mockRejectedValue(new Error("x"));
    const r = await getLastSuiteRun();
    expect(r.ok).toBe(false);
  });
});

describe("transcribe", () => {
  function file(): File {
    return new File([new Uint8Array([1, 2, 3])], "clip.wav", { type: "audio/wav" });
  }

  it("uploads the audio and decodes the REST trace", async () => {
    uploadFileMock.mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          text: "hello",
          provider_tier: "byok",
          provider_id: "openai",
          model_id: "whisper-1",
          latency_ms: 99,
        }),
    });
    const r = await transcribe(file());
    expect(uploadFileMock).toHaveBeenCalledTimes(1);
    const [path, fd] = uploadFileMock.mock.calls[0]!;
    expect(path).toBe("/api/v1/voice/transcribe");
    expect(fd).toBeInstanceOf(FormData);
    expect(r.ok).toBe(true);
    if (r.ok) {
      expect(r.data.text).toBe("hello");
      expect(r.data.trace).toEqual({
        providerTier: "byok",
        providerId: "openai",
        modelId: "whisper-1",
        latencyMs: 99,
      });
    }
  });

  it("defaults missing JSON fields", async () => {
    uploadFileMock.mockResolvedValue({ ok: true, json: () => Promise.resolve({}) });
    const r = await transcribe(file());
    if (r.ok) {
      expect(r.data.text).toBe("");
      expect(r.data.trace.latencyMs).toBe(0);
    }
  });

  it("decodes the server error envelope on a non-2xx response", async () => {
    uploadFileMock.mockResolvedValue({
      ok: false,
      status: 502,
      json: () => Promise.resolve({ code: "unavailable", message: "stt down" }),
    });
    const r = await transcribe(file());
    expect(r.ok).toBe(false);
    if (!r.ok) {
      expect(r.error.code).toBe("unavailable");
      expect(r.error.status).toBe(502);
    }
  });
});
