import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithProviders } from "@vrooli/api-base/testing";
import IntegrationsPanel from "../components/IntegrationsPanel";
import { strings } from "../consts/strings";
import { i18n } from "../i18n";
import type { CapabilitiesResponse } from "../api/capabilities";

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:4200/api/v1",
  buildApiUrl: (path: string) => `http://localhost:4200/api/v1${path}`,
  buildWsUrl: (path: string) => `ws://localhost:4200/api/v1${path}`,
  createScenarioConnectTransport: () => ({}),
}));

const { mockFetchCapabilities, mockRunCapabilityAction } = vi.hoisted(() => ({
  mockFetchCapabilities: vi.fn(),
  mockRunCapabilityAction: vi.fn(),
}));

vi.mock("../api/capabilities", () => ({
  fetchCapabilities: mockFetchCapabilities,
  runCapabilityAction: mockRunCapabilityAction,
}));

const mockCapabilities: CapabilitiesResponse = {
  capabilities: [
    {
      id: "audio-tools",
      name: "Audio Tools",
      description: "Shared audio capability scenario",
      dependencyKind: "scenario",
      dependencySlug: "audio-tools",
      features: ["voice-input", "voice-streaming"],
      status: "available",
      message: "scenario is healthy",
      checkedAt: "2026-03-17T00:00:00Z",
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
    {
      id: "session-backend-standard",
      name: "Standard Terminal Sessions",
      description: "Local PTY terminal sessions",
      dependencyKind: "resource",
      dependencySlug: "session-backend-standard",
      features: [],
      status: "available",
    },
    {
      id: "session-backend-persistent",
      name: "Persistent Terminal Sessions",
      description: "tmux-backed terminal sessions",
      dependencyKind: "resource",
      dependencySlug: "session-backend-persistent",
      features: [],
      status: "available",
    },
    {
      id: "vrooli-bridge",
      name: "Remote Terminals",
      description: "Bridged terminal sessions on registered nodes",
      dependencyKind: "scenario",
      dependencySlug: "vrooli-bridge",
      features: [],
      status: "unavailable",
      message: "Bridge URL is not configured",
      reasonCode: "bridge_url_missing",
      actionKind: "scenario_start",
      actionLabel: "Start Bridge",
      operatorCommand: "vrooli scenario start vrooli-bridge --json",
    },
  ],
  timestamp: "2026-03-17T00:00:00Z",
};

beforeEach(() => {
  mockFetchCapabilities.mockReset();
  mockRunCapabilityAction.mockReset();
});

describe("IntegrationsPanel", () => {
  it("shows loading state while fetching", () => {
    mockFetchCapabilities.mockReturnValue(new Promise(() => {}));
    renderWithProviders(<IntegrationsPanel open={true} />);
    expect(screen.getByText(strings.integrationsPanel.checking)).toBeTruthy();
  });

  it("shows error state when fetch fails", async () => {
    await i18n.changeLanguage("en");
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
      for (const capability of mockCapabilities.capabilities) {
        expect(screen.getByTestId(`cap-card-${capability.id}`)).toBeTruthy();
      }
    });
  });

  it("shows correct active count", async () => {
    await i18n.changeLanguage("en");
    mockFetchCapabilities.mockResolvedValue(mockCapabilities);
    renderWithProviders(<IntegrationsPanel open={true} />);

    await waitFor(() => {
      expect(screen.getByText("4/6 active")).toBeTruthy();
    });
  });

  it("shows diagnostic messages", async () => {
    mockFetchCapabilities.mockResolvedValue(mockCapabilities);
    renderWithProviders(<IntegrationsPanel open={true} />);

    await waitFor(() => {
      expect(screen.getByText("OPENROUTER_API_KEY not configured")).toBeTruthy();
      expect(screen.getByText("Bridge URL is not configured")).toBeTruthy();
    });
  });

  it("shows typed reason and safe operator command metadata", async () => {
    mockFetchCapabilities.mockResolvedValue({
      ...mockCapabilities,
      capabilities: [
        {
          id: "audio-tools",
          name: "Audio Tools",
          description: "Shared audio capability scenario",
          dependencyKind: "scenario",
          dependencySlug: "audio-tools",
          features: ["voice-input"],
          status: "unavailable",
          message: "scenario is installed but stopped",
          reasonCode: "scenario_stopped",
          actionKind: "scenario_start",
          actionLabel: "Start scenario",
          operatorCommand: "vrooli scenario start audio-tools --json",
        },
      ],
    });
    renderWithProviders(<IntegrationsPanel open={true} />);

    await waitFor(() => {
      expect(screen.getByTestId("cap-reason-audio-tools").textContent).toContain("scenario_stopped");
      expect(screen.getByTestId("cap-action-audio-tools").textContent).toContain("Start scenario");
      expect(screen.getByTestId("cap-command-audio-tools").textContent).toContain(
        "vrooli scenario start audio-tools --json",
      );
    });
  });

  it("runs a supported scenario action and refreshes capabilities", async () => {
    await i18n.changeLanguage("en");
    const stopped: CapabilitiesResponse = {
      capabilities: [
        {
          id: "audio-tools",
          name: "Audio Tools",
          description: "Shared audio capability scenario",
          dependencyKind: "scenario",
          dependencySlug: "audio-tools",
          features: ["voice-input"],
          status: "unavailable",
          message: "scenario is installed but stopped",
          reasonCode: "scenario_stopped",
          actionKind: "scenario_start",
          actionLabel: "Start scenario",
          operatorCommand: "vrooli scenario start audio-tools --json",
        },
      ],
      timestamp: "2026-03-17T00:00:00Z",
    };
    const stoppedCap = stopped.capabilities[0];
    if (!stoppedCap) throw new Error("test fixture missing capability");
    const refreshed: CapabilitiesResponse = {
      capabilities: [
        {
          ...stoppedCap,
          status: "available",
          message: "scenario is healthy",
          reasonCode: undefined,
          actionKind: undefined,
          actionLabel: undefined,
          operatorCommand: undefined,
        },
      ],
      timestamp: "2026-03-17T00:01:00Z",
    };
    mockFetchCapabilities.mockResolvedValue(refreshed);
    mockFetchCapabilities.mockResolvedValueOnce(stopped);
    mockRunCapabilityAction.mockResolvedValue({
      success: true,
      status: "healthy",
      message: "lifecycle action completed",
      capabilityId: "audio-tools",
      actionKind: "scenario_start",
      capabilities: refreshed.capabilities,
      timestamp: refreshed.timestamp,
    });
    renderWithProviders(<IntegrationsPanel open={true} />);

    const button = await screen.findByTestId("cap-run-action-audio-tools");
    fireEvent.click(button);

    await waitFor(() => {
      expect(mockRunCapabilityAction).toHaveBeenCalledWith("audio-tools", "scenario_start");
      expect(screen.getByTestId("cap-action-result-audio-tools").textContent).toContain(
        "lifecycle action completed",
      );
      expect(screen.getByText("scenario is healthy")).toBeTruthy();
    });
  });

  it("shows action failure when the delegated lifecycle call fails", async () => {
    await i18n.changeLanguage("en");
    mockFetchCapabilities.mockResolvedValue({
      capabilities: [
        {
          id: "audio-tools",
          name: "Audio Tools",
          description: "Shared audio capability scenario",
          dependencyKind: "scenario",
          dependencySlug: "audio-tools",
          features: ["voice-input"],
          status: "unavailable",
          message: "scenario start failed",
          reasonCode: "scenario_start_failed",
          actionKind: "scenario_restart",
          actionLabel: "Restart scenario",
          operatorCommand: "vrooli scenario restart audio-tools --json",
        },
      ],
      timestamp: "2026-03-17T00:00:00Z",
    });
    mockRunCapabilityAction.mockRejectedValue(new Error("wait timed out"));
    renderWithProviders(<IntegrationsPanel open={true} />);

    const button = await screen.findByTestId("cap-run-action-audio-tools");
    fireEvent.click(button);

    await waitFor(() => {
      expect(screen.getByTestId("cap-action-error-audio-tools").textContent).toContain("wait timed out");
    });
  });

  it("normalizes a non-Error action rejection for the operator", async () => {
    await i18n.changeLanguage("en");
    mockFetchCapabilities.mockResolvedValue({
      capabilities: [{
        id: "audio-tools",
        name: "Audio Tools",
        description: "Shared audio capability scenario",
        dependencyKind: "scenario",
        dependencySlug: "audio-tools",
        features: [],
        status: "unavailable",
        actionKind: "scenario_start",
        actionLabel: "Start scenario",
      }],
      timestamp: "2026-03-17T00:00:00Z",
    });
    mockRunCapabilityAction.mockRejectedValue("bridge returned a non-error failure");
    renderWithProviders(<IntegrationsPanel open />);

    fireEvent.click(await screen.findByTestId("cap-run-action-audio-tools"));
    await waitFor(() => {
      expect(screen.getByTestId("cap-action-error-audio-tools")).toHaveTextContent("bridge returned a non-error failure");
    });
  });

  it("shows unsuccessful lifecycle results returned by the backend", async () => {
    await i18n.changeLanguage("en");
    const stopped: CapabilitiesResponse = {
      capabilities: [
        {
          id: "audio-tools",
          name: "Audio Tools",
          description: "Shared audio capability scenario",
          dependencyKind: "scenario",
          dependencySlug: "audio-tools",
          features: ["voice-input"],
          status: "unavailable",
          message: "scenario is installed but stopped",
          reasonCode: "scenario_stopped",
          actionKind: "scenario_start",
          actionLabel: "Start scenario",
          operatorCommand: "vrooli scenario start audio-tools --json",
        },
      ],
      timestamp: "2026-03-17T00:00:00Z",
    };
    mockFetchCapabilities.mockResolvedValue(stopped);
    mockRunCapabilityAction.mockResolvedValue({
      success: false,
      status: "failed",
      message: "lifecycle wait failed: timeout",
      capabilityId: "audio-tools",
      actionKind: "scenario_start",
      capabilities: stopped.capabilities,
      timestamp: "2026-03-17T00:01:00Z",
    });
    renderWithProviders(<IntegrationsPanel open={true} />);

    const button = await screen.findByTestId("cap-run-action-audio-tools");
    fireEvent.click(button);

    await waitFor(() => {
      const result = screen.getByTestId("cap-action-result-audio-tools");
      expect(result.textContent).toContain("lifecycle wait failed: timeout");
      expect(result.className).toContain("text-red-400");
    });
  });

  it("does not show a backend action button for operator-only guidance", async () => {
    mockFetchCapabilities.mockResolvedValue({
      capabilities: [
        {
          id: "audio-tools",
          name: "Audio Tools",
          description: "Shared audio capability scenario",
          dependencyKind: "scenario",
          dependencySlug: "audio-tools",
          features: ["voice-input"],
          status: "unavailable",
          actionKind: "operator_command",
          actionLabel: "Wait for scenario",
          operatorCommand: "vrooli scenario wait audio-tools --json",
        },
      ],
      timestamp: "2026-03-17T00:00:00Z",
    });
    renderWithProviders(<IntegrationsPanel open={true} />);

    await waitFor(() => {
      expect(screen.getByTestId("cap-action-audio-tools").textContent).toContain("Wait for scenario");
      expect(screen.queryByTestId("cap-run-action-audio-tools")).toBeNull();
    });
  });

  it("shows dependency kind badges", async () => {
    mockFetchCapabilities.mockResolvedValue(mockCapabilities);
    renderWithProviders(<IntegrationsPanel open={true} />);

    await waitFor(() => {
      // Four resources (AI providers and session backends) + two scenarios.
      expect(screen.getAllByText("resource").length).toBe(4);
      expect(screen.getAllByText("scenario").length).toBe(2);
    });
  });

  it("crosses out features for unavailable capabilities", async () => {
    mockFetchCapabilities.mockResolvedValue(mockCapabilities);
    renderWithProviders(<IntegrationsPanel open={true} />);

    await waitFor(() => {
      const remoteReason = screen.getByText("Bridge URL is not configured");
      expect(remoteReason).toBeTruthy();
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

  it("renders an unknown capability with neutral status styling", async () => {
    mockFetchCapabilities.mockResolvedValue({
      capabilities: [{
        id: "future-capability",
        name: "Future capability",
        description: "Status not understood by this client yet",
        dependencyKind: "resource",
        dependencySlug: "future-capability",
        features: [],
        status: "unknown",
      }],
      timestamp: "2026-03-17T00:00:00Z",
    });
    renderWithProviders(<IntegrationsPanel open />);

    await waitFor(() => {
      const card = screen.getByTestId("cap-card-future-capability");
      expect(card.className).toContain("border-l-wc-default");
      expect(screen.getByText(strings.integrationsPanel.activeCount)).toBeInTheDocument();
    });
  });

  it("reports an empty capability catalogue", async () => {
    mockFetchCapabilities.mockResolvedValue({ capabilities: [], timestamp: "2026-03-17T00:00:00Z" });
    renderWithProviders(<IntegrationsPanel open />);

    await waitFor(() => {
      expect(screen.getByText(strings.integrationsPanel.noneConfigured)).toBeInTheDocument();
    });
  });

  it("falls back to the lifecycle status when an action has no message", async () => {
    const capability = mockCapabilities.capabilities[0];
    if (!capability) throw new Error("test fixture missing capability");
    mockFetchCapabilities.mockResolvedValue({ capabilities: [{
      ...capability,
      status: "unavailable",
      reasonCode: "scenario_stopped",
      actionKind: "scenario_start",
      actionLabel: "Start scenario",
    }], timestamp: "2026-03-17T00:00:00Z" });
    mockRunCapabilityAction.mockResolvedValue({
      success: true,
      status: "healthy",
      capabilityId: capability.id,
      actionKind: "scenario_start",
      capabilities: [{ ...capability, status: "available" }],
      timestamp: "2026-03-17T00:01:00Z",
    });
    renderWithProviders(<IntegrationsPanel open />);

    fireEvent.click(await screen.findByTestId("cap-run-action-audio-tools"));
    await waitFor(() => {
      expect(screen.getByTestId("cap-action-result-audio-tools")).toHaveTextContent("healthy");
    });
  });
});
