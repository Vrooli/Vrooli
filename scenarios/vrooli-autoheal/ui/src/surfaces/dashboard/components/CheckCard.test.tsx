import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils";
import { CheckCard } from "./CheckCard";

const baseCheck = {
  checkId: "infra-dns",
  status: "warning" as const,
  message: "DNS is degraded",
  timestamp: new Date().toISOString(),
  duration: 1200000,
  title: "DNS Resolution",
  description: "Checks name resolution",
  importance: "Required for service discovery",
  category: "infrastructure" as const,
  intervalSeconds: 3600,
  autoHealIssue: {
    id: 1,
    checkId: "infra-dns",
    actionId: "restart",
    success: false,
    message: "exit status 1",
    timestamp: new Date().toISOString(),
    durationMs: 100,
  },
  metrics: {
    score: 60,
    subChecks: [
      { name: "System resolver", passed: true, detail: "available" },
      { name: "External resolver", passed: false, detail: "timed out" },
    ],
  },
};

describe("CheckCard", () => {
  it("renders metadata, score bands, sub-checks, and details action", () => {
    const onInfo = vi.fn();
    const { rerender } = renderWithProviders(<CheckCard check={baseCheck} onInfoClick={onInfo} mobileListItem />);
    expect(screen.getByText("DNS Resolution")).toBeInTheDocument();
    expect(screen.getByText("Score 60%")).toBeInTheDocument();
    expect(screen.getByText(/external resolver/i)).toBeInTheDocument();
    expect(screen.getByText("Auto-heal failed")).toBeInTheDocument();
    expect(screen.getByText("exit status 1")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "View check details" }));
    expect(onInfo).toHaveBeenCalledWith("infra-dns");

    rerender(<CheckCard check={{ ...baseCheck, status: "ok", metrics: { score: 90, subChecks: [] } }} />);
    expect(screen.getByText("Score 90%")).toBeInTheDocument();
    rerender(<CheckCard check={{ ...baseCheck, status: "critical", metrics: { score: 20, subChecks: [] }, title: undefined }} />);
    expect(screen.getByText("infra-dns")).toBeInTheDocument();
  });

  it("labels policy skips separately from failed recovery", () => {
    renderWithProviders(
      <CheckCard
        check={{
          ...baseCheck,
          autoHealIssue: {
            ...baseCheck.autoHealIssue,
            actionId: "autoheal-skip",
            message: "in cooldown (300s remaining)",
          },
        }}
      />,
    );

    expect(screen.getByText("Auto-heal skipped")).toBeInTheDocument();
    expect(screen.getByText("in cooldown (300s remaining)")).toBeInTheDocument();
  });
});
