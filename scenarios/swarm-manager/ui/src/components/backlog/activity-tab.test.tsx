import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ActivityTab } from "./activity-tab";
import { selectors } from "../../consts/selectors";
import type { ActivityTabProps } from "./activity-tab";

vi.mock("../detail/ActivityTimeline", () => ({
  ActivityTimeline: (props: Record<string, unknown>) => (
    <div
      data-testid="mock-activity-timeline"
      data-is-loading={String(props.isLoading)}
      data-agent-run-is-active={String(props.agentRunIsActive)}
      data-has-error={String(props.error != null)}
      data-entry-count={String((props.entries as unknown[])?.length ?? 0)}
      data-agent-manager-ui-url={String(props.agentManagerUiUrl ?? "")}
    />
  ),
}));

const defaultProps: ActivityTabProps = {
  timeline: {
    entries: [],
    isLoading: false,
    error: null,
  },
  agentRunIsBlocking: false,
  latestAgentActivity: null,
  agentManagerUiUrl: null,
  onStopRun: vi.fn(),
  onFollowUp: vi.fn(),
  onViewExecution: vi.fn(),
};

describe("ActivityTab", () => {
  it("renders with correct data-testid", () => {
    render(<ActivityTab {...defaultProps} />);
    expect(screen.getByTestId(selectors.backlogDetails.activityTab)).toBeInTheDocument();
  });

  it("renders ActivityTimeline with correct props when idle", () => {
    render(<ActivityTab {...defaultProps} />);
    const timeline = screen.getByTestId("mock-activity-timeline");
    expect(timeline).toBeInTheDocument();
    expect(timeline).toHaveAttribute("data-is-loading", "false");
    expect(timeline).toHaveAttribute("data-agent-run-is-active", "false");
    expect(timeline).toHaveAttribute("data-has-error", "false");
    expect(timeline).toHaveAttribute("data-entry-count", "0");
  });

  it("passes loading state to ActivityTimeline", () => {
    render(
      <ActivityTab
        {...defaultProps}
        timeline={{ entries: [], isLoading: true, error: null }}
      />,
    );
    const timeline = screen.getByTestId("mock-activity-timeline");
    expect(timeline).toHaveAttribute("data-is-loading", "true");
  });

  it("passes error to ActivityTimeline", () => {
    render(
      <ActivityTab
        {...defaultProps}
        timeline={{ entries: [], isLoading: false, error: new Error("fail") }}
      />,
    );
    const timeline = screen.getByTestId("mock-activity-timeline");
    expect(timeline).toHaveAttribute("data-has-error", "true");
  });

  it("passes entries to ActivityTimeline", () => {
    const entries = [
      { id: "e1", type: "execution" as const, timestamp: "2026-01-01T00:00:00Z" },
      { id: "e2", type: "execution" as const, timestamp: "2026-01-02T00:00:00Z" },
    ];
    render(
      <ActivityTab
        {...defaultProps}
        timeline={{ entries: entries as ActivityTabProps["timeline"]["entries"], isLoading: false, error: null }}
      />,
    );
    const timeline = screen.getByTestId("mock-activity-timeline");
    expect(timeline).toHaveAttribute("data-entry-count", "2");
  });

  it("passes agentRunIsBlocking through to ActivityTimeline", () => {
    render(<ActivityTab {...defaultProps} agentRunIsBlocking />);
    const timeline = screen.getByTestId("mock-activity-timeline");
    expect(timeline).toHaveAttribute("data-agent-run-is-active", "true");
  });

  it("passes agentManagerUiUrl when provided", () => {
    render(<ActivityTab {...defaultProps} agentManagerUiUrl="http://localhost:3000" />);
    const timeline = screen.getByTestId("mock-activity-timeline");
    expect(timeline).toHaveAttribute("data-agent-manager-ui-url", "http://localhost:3000");
  });

  it("converts null agentManagerUiUrl to undefined for ActivityTimeline", () => {
    render(<ActivityTab {...defaultProps} agentManagerUiUrl={null} />);
    const timeline = screen.getByTestId("mock-activity-timeline");
    // When null, the component passes undefined, which our mock renders as ""
    expect(timeline).toHaveAttribute("data-agent-manager-ui-url", "");
  });
});
