import { describe, it, expect, vi, beforeEach } from "vitest";
import { apiBaseMock } from "../test-utils";

vi.mock("@vrooli/api-base", () => apiBaseMock());

const generateMock = vi.fn();
vi.mock("../api/ai", async () => {
  const actual = await vi.importActual<typeof import("../api/ai")>("../api/ai");
  return {
    ...actual,
    aiClient: { generate: generateMock },
    generateAICommand: async (prompt: string, context?: string) => {
      const resp = await generateMock({ prompt, context: context ?? "" });
      return { command: resp.command, provider: resp.provider };
    },
  };
});

// [REQ:P0-005a] AI Command Generation API - client tests
describe("generateAICommand", () => {
  beforeEach(() => {
    generateMock.mockReset();
  });

  it("returns command and provider from aiClient.generate", async () => {
    generateMock.mockResolvedValue({ command: "find . -name '*.go'", provider: "ollama" });

    const { generateAICommand } = await import("../api/ai");
    const result = await generateAICommand("find all Go files");

    expect(result).toEqual({ command: "find . -name '*.go'", provider: "ollama" });
    expect(generateMock).toHaveBeenCalledWith({ prompt: "find all Go files", context: "" });
  });

  it("includes context when provided", async () => {
    generateMock.mockResolvedValue({ command: "ls /tmp", provider: "ollama" });

    const { generateAICommand } = await import("../api/ai");
    await generateAICommand("list temp files", "cwd: /home/user");

    expect(generateMock).toHaveBeenCalledWith({ prompt: "list temp files", context: "cwd: /home/user" });
  });

  it("propagates errors from the Connect client", async () => {
    generateMock.mockRejectedValue(new Error("AI unavailable"));

    const { generateAICommand } = await import("../api/ai");
    await expect(generateAICommand("test prompt")).rejects.toThrow("AI unavailable");
  });

  it("sends empty context when not provided", async () => {
    generateMock.mockResolvedValue({ command: "pwd", provider: "openrouter" });

    const { generateAICommand } = await import("../api/ai");
    await generateAICommand("current directory");

    expect(generateMock).toHaveBeenCalledWith({ prompt: "current directory", context: "" });
  });
});
