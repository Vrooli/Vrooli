import { describe, it, expect, vi, beforeEach } from "vitest";
import { createElement } from "react";
import { render, screen, fireEvent, act, cleanup } from "@testing-library/react";
import { apiBaseMock } from "../test-utils";
import type { AIGenerateResponse } from "../api/ai";

vi.mock("@vrooli/api-base", () => apiBaseMock());

const mockStoreState: Record<string, unknown> = {
  aiModalOpen: true,
  setAiModalOpen: vi.fn(),
  activePane: "sess-1",
};

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector: (state: Record<string, unknown>) => unknown) =>
    selector(mockStoreState),
}));

type GenerateRequest = { prompt: string; context: string };

const generateMock = vi.fn<(request: GenerateRequest) => Promise<AIGenerateResponse>>();
vi.mock("../api/ai", () => ({
  aiClient: { generate: generateMock },
  generateAICommand: async (prompt: string, context?: string) => {
    const resp = await generateMock({ prompt, context: context ?? "" });
    return { command: resp.command, provider: resp.provider };
  },
}));

// [REQ:P0-005b] AI Input UI Component - component tests
describe("AiInput component", () => {
  beforeEach(() => {
    generateMock.mockReset();
  });

  it("component module exports default function", async () => {
    const mod = await import("../components/AiInput");
    expect(typeof mod.default).toBe("function");
  });

  it("generateAICommand is importable from api module", async () => {
    const { generateAICommand } = await import("../api/ai");
    expect(typeof generateAICommand).toBe("function");
  });

  it("AIGenerateResponse shape exposes command and provider", async () => {
    generateMock.mockResolvedValue({ command: "ls -la", provider: "ollama" });

    const { generateAICommand } = await import("../api/ai");
    const result = await generateAICommand("list files");

    expect(result).toHaveProperty("command");
    expect(result).toHaveProperty("provider");
    expect(typeof result.command).toBe("string");
    expect(typeof result.provider).toBe("string");
  });

  it("generateAICommand passes prompt and context to aiClient.generate", async () => {
    generateMock.mockResolvedValue({ command: "pwd", provider: "ollama" });

    const { generateAICommand } = await import("../api/ai");
    await generateAICommand("show current dir", "cwd: /home");

    expect(generateMock).toHaveBeenCalledWith({ prompt: "show current dir", context: "cwd: /home" });
  });

  it("generateAICommand propagates errors from the Connect client", async () => {
    generateMock.mockRejectedValue(new Error("AI unavailable"));

    const { generateAICommand } = await import("../api/ai");
    await expect(generateAICommand("test")).rejects.toThrow("AI unavailable");
  });
});

describe("AiInput rendering (DrawerShell compact)", () => {
  beforeEach(() => {
    cleanup();
    mockStoreState.aiModalOpen = true;
    mockStoreState.setAiModalOpen = vi.fn();
  });

  it("renders dialog semantics on the compact drawer and auto-focuses the prompt", async () => {
    const { default: AiInput } = await import("../components/AiInput");
    render(createElement(AiInput, { onExecute: vi.fn() }));
    const panel = screen.getByTestId("ai-input");
    expect(panel.getAttribute("role")).toBe("dialog");
    expect(panel.getAttribute("aria-modal")).toBe("true");
    expect(panel.className).toContain("md:max-w-md");
    // The prompt input auto-focuses shortly after open (50ms defer) and must
    // be the initially focused element inside the trap.
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 80));
    });
    expect(document.activeElement).toBe(screen.getByTestId("ai-input-prompt"));
  });

  it("closes on Escape", async () => {
    const { default: AiInput } = await import("../components/AiInput");
    render(createElement(AiInput, { onExecute: vi.fn() }));
    fireEvent.keyDown(window, { key: "Escape" });
    expect(mockStoreState.setAiModalOpen).toHaveBeenCalledWith(false);
  });

  it("renders nothing when the modal flag is off", async () => {
    mockStoreState.aiModalOpen = false;
    const { default: AiInput } = await import("../components/AiInput");
    render(createElement(AiInput, { onExecute: vi.fn() }));
    expect(screen.queryByTestId("ai-input")).toBeNull();
  });
});
