import { fireEvent, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../test-utils";
import AgentGridSection from "./AgentGridSection";

vi.mock("./useAgentUpdates", () => ({ useAgentUpdates: () => ({ updates: {}, loading: false }) }));
vi.mock("./useAgentUsage", () => ({ useAgentUsage: () => ({ usages: { codex: {
  agentId: "codex", targetId: "local", match: "likely", reason: "The launch may use a different account", source: "Codex", observedAt: new Date().toISOString(),
  windows: [{ pool: "primary", windowMinutes: 10080, usedPercent: 42, resetsAt: new Date(Date.now() + 86_400_000).toISOString() }],
} }, loading: false, refresh: vi.fn() }) }));

describe("agent usage in launcher", () => {
  it("shows account uncertainty and a provider reading in the card and its details in management", () => {
    render(<AgentGridSection shortcuts={[{ agentId: "codex", label: "Codex", command: "codex --yolo" }]}
      target={{ id: "local", kind: "local", label: "This machine", available: true, state: "dispatchable", readiness: [{ key: "capability:codex", label: "Codex", passed: true, state: "ready", detail: "" }] }}
      disabled={false} onLaunch={vi.fn()} />);
    const grid = screen.getByTestId("launcher-agent-grid");
    expect(within(grid).getByText("launcher.usageEstimated")).toBeInTheDocument();
    expect(within(grid).getByRole("progressbar")).toHaveAttribute("aria-valuenow", "42");
    fireEvent.click(screen.getByTestId("launcher-manage-agents"));
    const management = screen.getByTestId("launcher-agent-management");
    expect(within(management).getByText("The launch may use a different account")).toBeInTheDocument();
    expect(within(management).getByRole("progressbar")).toHaveAttribute("aria-valuenow", "42");
  });
});
