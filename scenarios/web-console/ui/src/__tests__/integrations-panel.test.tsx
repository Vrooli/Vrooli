import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders } from "../test-utils/render";
import IntegrationsPanel from "../components/IntegrationsPanel";
import type { CapabilitiesResponse } from "../api/capabilities";

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:4200/api/v1",
  buildApiUrl: (path: string) => `http://localhost:4200/api/v1${path}`,
  buildWsUrl: (path: string) => `ws://localhost:4200/api/v1${path}`,
  createScenarioConnectTransport: () => ({}),
}));

const { mockFetchCapabilities } = vi.hoisted(() => ({
  mockFetchCapabilities: vi.fn(),
}));

vi.mock("../api/capabilities", () => ({
  fetchCapabilities: mockFetchCapabilities,
}));

const mockCapabilities: CapabilitiesResponse = {
  capabilities: [
    {
      id: "whisper-stt",
      name: "Whisper STT",
      description: "Speech-to-text transcription via Whisper",
      dependencyKind: "resource",
      dependencySlug: "whisper",
      features: ["voice-input", "voice-streaming"],
      status: "available",
      message: "resource is healthy and transcription verified",
    },
    {
      id: "kokoro-tts",
      name: "Kokoro TTS",
      description: "Text-to-speech synthesis via Kokoro",
      dependencyKind: "resource",
      dependencySlug: "kokoro",
      features: ["voice-output"],
      status: "unavailable",
      message: "resource is not responding",
    },
    {
      id: "ollama",
      name: "Ollama",
      description: "Local LLM inference for AI command generation",
      dependencyKind: "resource",
      dependencySlug: "ollama",
      features: ["ai-command-generation"],
      status: "available",
    },
    {
      id: "openrouter",
      name: "OpenRouter",
      description: "Cloud LLM API for AI command generation",
      dependencyKind: "resource",
      dependencySlug: "openrouter",
      features: ["ai-command-generation"],
      status: "unavailable",
      message: "OPENROUTER_API_KEY not configured",
    },
  ],
  timestamp: "2026-03-17T00:00:00Z",
};

beforeEach(() => {
  mockFetchCapabilities.mockReset();
});

describe("IntegrationsPanel", () => {
  it("shows loading state while fetching", () => {
    mockFetchCapabilities.mockReturnValue(new Promise(() => {}));
    renderWithProviders(<IntegrationsPanel open={true} />);
    expect(screen.getByText("Checking integrations...")).toBeTruthy();
  });

  it("shows error state when fetch fails", async () => {
    mockFetchCapabilities.mockRejectedValue(new Error("Server error"));
    renderWithProviders(<IntegrationsPanel open={true} />);

    await waitFor(() => {
      expect(screen.getByText(/Failed to load integrations/)).toBeTruthy();
    });
  });

  it("renders all capability cards", async () => {
    mockFetchCapabilities.mockResolvedValue(mockCapabilities);
    renderWithProviders(<IntegrationsPanel open={true} />);

    await waitFor(() => {
      expect(screen.getByTestId("cap-card-whisper-stt")).toBeTruthy();
      expect(screen.getByTestId("cap-card-kokoro-tts")).toBeTruthy();
      expect(screen.getByTestId("cap-card-ollama")).toBeTruthy();
      expect(screen.getByTestId("cap-card-openrouter")).toBeTruthy();
    });
  });

  it("shows correct active count", async () => {
    mockFetchCapabilities.mockResolvedValue(mockCapabilities);
    renderWithProviders(<IntegrationsPanel open={true} />);

    await waitFor(() => {
      expect(screen.getByText("2/4 active")).toBeTruthy();
    });
  });

  it("shows diagnostic messages", async () => {
    mockFetchCapabilities.mockResolvedValue(mockCapabilities);
    renderWithProviders(<IntegrationsPanel open={true} />);

    await waitFor(() => {
      expect(screen.getByTestId("cap-message-kokoro-tts")).toBeTruthy();
      expect(screen.getByText("resource is not responding")).toBeTruthy();
      expect(screen.getByText("OPENROUTER_API_KEY not configured")).toBeTruthy();
    });
  });

  it("shows dependency kind badges", async () => {
    mockFetchCapabilities.mockResolvedValue(mockCapabilities);
    renderWithProviders(<IntegrationsPanel open={true} />);

    await waitFor(() => {
      const badges = screen.getAllByText("resource");
      expect(badges.length).toBe(4);
    });
  });

  it("crosses out features for unavailable capabilities", async () => {
    mockFetchCapabilities.mockResolvedValue(mockCapabilities);
    renderWithProviders(<IntegrationsPanel open={true} />);

    await waitFor(() => {
      const voiceOutput = screen.getByText("voice-output");
      expect(voiceOutput.className).toContain("line-through");
    });
  });

  it("does not cross out features for available capabilities", async () => {
    mockFetchCapabilities.mockResolvedValue(mockCapabilities);
    renderWithProviders(<IntegrationsPanel open={true} />);

    await waitFor(() => {
      const voiceInput = screen.getByText("voice-input");
      expect(voiceInput.className).not.toContain("line-through");
    });
  });

  it("does not fetch when open is false", () => {
    renderWithProviders(<IntegrationsPanel open={false} />);
    expect(mockFetchCapabilities).not.toHaveBeenCalled();
  });
});
