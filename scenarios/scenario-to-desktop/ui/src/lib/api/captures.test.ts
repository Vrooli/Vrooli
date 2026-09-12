import { describe, expect, it, vi } from "vitest";

const client = vi.hoisted(() => ({
  listEvidenceCaptures: vi.fn(),
  getEvidenceCapturesSummary: vi.fn(),
  deleteEvidenceCapture: vi.fn(),
  deleteAllEvidenceCaptures: vi.fn(),
}));

vi.mock("./connect", () => ({ evidenceConnectClient: client }));
vi.mock("./client", () => ({
  buildUrl: (path: string) => `http://api.test${path}`,
}));

import {
  buildCaptureFileUrl,
  buildCapturesDownloadUrl,
  deleteAllCaptures,
  deleteCapture,
  getCapturesSummary,
  listCaptures,
} from "./captures";

describe("evidence capture Connect client", () => {
  it("returns generated capture messages without a display-model adapter", async () => {
    client.listEvidenceCaptures.mockResolvedValue({
      captures: [
        {
          captureId: "cap-1",
          scenarioName: "calculator",
          kind: "recording",
          filename: "run.webm",
          fileSizeBytes: 42n,
          width: 1280,
          height: 720,
          durationMs: 5000n,
          sourceSessionId: "desktop-1",
          createdAt: { seconds: 1n, nanos: 0 },
        },
      ],
    });
    client.getEvidenceCapturesSummary.mockResolvedValue({
      count: 1,
      totalBytes: 42n,
    });

    await expect(listCaptures("calculator")).resolves.toEqual([
      expect.objectContaining({
        captureId: "cap-1",
        kind: "recording",
        fileSizeBytes: 42n,
        durationMs: 5000n,
        createdAt: { seconds: 1n, nanos: 0 },
      }),
    ]);
    await expect(getCapturesSummary("calculator")).resolves.toEqual({
      count: 1,
      totalBytes: 42n,
    });
    expect(client.listEvidenceCaptures).toHaveBeenCalledWith({
      scenarioName: "calculator",
    });
  });

  it("uses typed deletion requests and safely encodes binary download URLs", async () => {
    await deleteCapture("my app", "capture/a");
    await deleteAllCaptures("my app");

    expect(client.deleteEvidenceCapture).toHaveBeenCalledWith({
      scenarioName: "my app",
      captureId: "capture/a",
    });
    expect(client.deleteAllEvidenceCaptures).toHaveBeenCalledWith({
      scenarioName: "my app",
    });
    expect(buildCaptureFileUrl("my app", "capture/a")).toBe(
      "http://api.test/captures/my%20app/capture%2Fa/file",
    );
    expect(buildCapturesDownloadUrl("my app", ["a/b", "one two"])).toBe(
      "http://api.test/captures/my%20app/download?ids=a%2Fb,one%20two",
    );
  });
});
