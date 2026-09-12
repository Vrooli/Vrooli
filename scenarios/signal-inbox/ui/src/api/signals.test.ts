import { beforeEach, describe, expect, it, vi } from "vitest";

const client = vi.hoisted(() => ({
  uploadFile: vi.fn(),
  decodeApiError: vi.fn(),
  fromJson: vi.fn(),
}));

vi.mock("@connectrpc/connect", () => ({ createClient: vi.fn(() => ({})) }));
vi.mock("./client", () => ({
  transport: {},
  uploadFile: client.uploadFile,
  decodeApiError: client.decodeApiError,
  fromJson: client.fromJson,
  PROTO_READ_OPTIONS: {},
}));

import { uploadSignalImage } from "./signals";

describe("uploadSignalImage", () => {
  beforeEach(() => vi.clearAllMocks());

  it("returns the durable payload reference from a successful upload", async () => {
    client.uploadFile.mockResolvedValue(new Response(JSON.stringify({}), { status: 200 }));
    client.fromJson.mockReturnValue({ image: { payloadRef: "blobs/signal.png" } });
    await expect(uploadSignalImage(new File(["image"], "signal.png", { type: "image/png" }))).resolves.toBe("blobs/signal.png");
  });

  it("decodes server errors and rejects missing payload references", async () => {
    client.uploadFile.mockResolvedValueOnce(new Response("blocked", { status: 429 }));
    client.decodeApiError.mockResolvedValue(new Error("blocked"));
    await expect(uploadSignalImage(new File(["x"], "signal.png"))).rejects.toThrow("blocked");

    client.uploadFile.mockResolvedValueOnce(new Response(JSON.stringify({}), { status: 200 }));
    client.fromJson.mockReturnValue({ image: {} });
    await expect(uploadSignalImage(new File(["x"], "signal.png"))).rejects.toThrow("image upload returned no payload reference");
  });
});
