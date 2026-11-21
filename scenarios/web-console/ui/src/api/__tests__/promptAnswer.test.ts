import { beforeEach, describe, expect, it, vi } from "vitest";
import { Code, ConnectError } from "@connectrpc/connect";

const { answerPromptRpc } = vi.hoisted(() => ({ answerPromptRpc: vi.fn() }));

vi.mock("@connectrpc/connect", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@connectrpc/connect")>();
  return { ...actual, createClient: () => ({ answerPrompt: answerPromptRpc }) };
});

vi.mock("../client", () => ({ transport: {} }));

import { answerPrompt } from "../promptAnswer";

describe("answerPrompt", () => {
  beforeEach(() => {
    answerPromptRpc.mockReset();
  });

  it("[REQ:P0-017i] sends the option and the prompt hash, and returns the delivery and the chosen label", async () => {
    answerPromptRpc.mockResolvedValue({ delivery: "keystrokes", answer: "Blue" });
    await expect(answerPrompt("s1", { optionKey: "2", promptHash: "h1", cancel: false })).resolves.toEqual({ delivery: "keystrokes", answer: "Blue" });
    expect(answerPromptRpc).toHaveBeenCalledWith({ sessionId: "s1", optionKey: "2", promptHash: "h1", cancel: false });
  });

  it("[REQ:P0-017i] a cancel sends no option", async () => {
    answerPromptRpc.mockResolvedValue({ delivery: "keystrokes", answer: "" });
    await answerPrompt("s1", { promptHash: "h1", cancel: true });
    expect(answerPromptRpc).toHaveBeenCalledWith({ sessionId: "s1", optionKey: "", promptHash: "h1", cancel: true });
  });

  it("[REQ:P0-017i] a refusal rejects with the server's reason, not the transport code", async () => {
    answerPromptRpc.mockRejectedValue(new ConnectError("the prompt changed since it was shown", Code.FailedPrecondition));
    await expect(answerPrompt("s1", { optionKey: "1", promptHash: "stale", cancel: false })).rejects.toThrow(/^the prompt changed since it was shown$/);
  });
});
