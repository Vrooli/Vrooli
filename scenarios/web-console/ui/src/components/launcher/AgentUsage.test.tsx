import { screen } from "@testing-library/react";
import { renderWithProviders as render } from "../../test-utils";
import { describe, expect, it } from "vitest";
import { AgentUsage } from "./AgentUsage";
import type { AgentUsage as Reading } from "./useAgentUsage";

const reading = (): Reading => ({
  targetId: "local", agentId: "codex", match: "likely", reason: "Account is estimated", source: "provider", observedAt: new Date().toISOString(),
  windows: [{ pool: "primary", windowMinutes: 10080, usedPercent: 42, resetsAt: new Date(Date.now() + 86_400_000).toISOString() }],
});

describe("AgentUsage", () => {
  it("shows real reported usage with estimated-account status and does not invent a missing window", () => {
    render(<AgentUsage usage={reading()} agent="Codex" />);
    expect(screen.getByText("launcher.usageEstimated")).toBeInTheDocument();
    expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "42");
    expect(screen.getByText("launcher.usageNoReading")).toBeInTheDocument();
  });

  it("does not present expired readings as current", () => {
    const stale = reading();
    stale.observedAt = new Date(Date.now() - 7 * 60 * 60_000).toISOString();
    render(<AgentUsage usage={stale} agent="Codex" />);
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
    expect(screen.getAllByText("launcher.usageStale").length).toBeGreaterThan(0);
  });

  it("does not combine separate pools and shows each separately in details", () => {
    const multiple = reading();
    multiple.windows.push({ pool: "other", windowMinutes: 10080, usedPercent: 8, resetsAt: "" });
    const { rerender } = render(<AgentUsage usage={multiple} agent="Codex" />);
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
    expect(screen.getByText("launcher.usageMultiplePools")).toBeInTheDocument();
    rerender(<AgentUsage usage={multiple} agent="Codex" detailed />);
    expect(screen.getAllByRole("progressbar").map((bar) => bar.getAttribute("aria-valuenow"))).toEqual(["8", "42"]);
    expect(screen.getByText("Account is estimated")).toBeInTheDocument();
  });
});
