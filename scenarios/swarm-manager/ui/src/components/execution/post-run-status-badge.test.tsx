import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { PostRunStatusBadge } from "./post-run-status-badge";
import type { ExecutionRecord, Finalization, FinalizationWarning } from "../../types";

const makeExecution = (overrides?: Partial<ExecutionRecord>): ExecutionRecord => ({
  executionId: "exec-1",
  backlogKind: "execute",
  backlogName: "test-feature",
  status: "completed",
  mode: "manual",
  createdAt: "2026-03-20T00:00:00Z",
  updatedAt: "2026-03-20T01:00:00Z",
  ...overrides,
});

const makeFinalization = (overrides?: Partial<Finalization>): Finalization => ({
  eligible: true,
  status: "completed",
  phase: "completed",
  scopeSource: "sandbox_diff",
  warnings: [],
  affectedScenarios: ["swarm-manager"],
  aggregateClassification: "ready",
  aggregateSummary: "",
  scenarios: [],
  ...overrides,
});

const makeWarning = (overrides?: Partial<FinalizationWarning>): FinalizationWarning => ({
  code: "test_warning",
  message: "test warning message",
  retryable: false,
  createdAt: "2026-03-20T01:00:00Z",
  ...overrides,
});

describe("PostRunStatusBadge", () => {
  it("renders nothing when no finalization", () => {
    const { container } = render(
      <PostRunStatusBadge execution={makeExecution()} />,
    );
    expect(container.firstChild).toBeNull();
  });

  it("renders classification label for completed finalization", () => {
    render(
      <PostRunStatusBadge
        execution={makeExecution({ finalization: makeFinalization() })}
      />,
    );
    expect(screen.getByText("Post-run checks passed")).toBeInTheDocument();
  });

  it("shows evidence-skip warning inline without expanding", () => {
    const finalization = makeFinalization({
      warnings: [
        makeWarning({
          code: "evidence_skipped_disabled",
          message: "Review agent is disabled in settings. Enable it to gather evidence automatically.",
        }),
      ],
    });
    render(
      <PostRunStatusBadge execution={makeExecution({ finalization })} />,
    );
    // Should be visible without clicking expand
    expect(screen.getByTestId("evidence-skip-warning")).toBeInTheDocument();
    expect(
      screen.getByText(/Review agent is disabled in settings/),
    ).toBeInTheDocument();
  });

  it("shows policy-error evidence-skip warning inline", () => {
    const finalization = makeFinalization({
      warnings: [
        makeWarning({
          code: "evidence_skipped_policy_error",
          message: "Could not load settings to check review agent policy.",
        }),
      ],
    });
    render(
      <PostRunStatusBadge execution={makeExecution({ finalization })} />,
    );
    expect(screen.getByTestId("evidence-skip-warning")).toBeInTheDocument();
    expect(
      screen.getByText(/Could not load settings/),
    ).toBeInTheDocument();
  });

  it("shows other warnings only when expanded", () => {
    const finalization = makeFinalization({
      aggregateSummary: "Needs attention",
      warnings: [
        makeWarning({ code: "restart_retry", message: "Restarted twice" }),
      ],
    });
    render(
      <PostRunStatusBadge execution={makeExecution({ finalization })} />,
    );
    // Not visible before expand
    expect(screen.queryByText("Restarted twice")).not.toBeInTheDocument();
    // Click to expand
    fireEvent.click(screen.getByTestId("post-run-status-badge").querySelector("button")!);
    expect(screen.getByText("Restarted twice")).toBeInTheDocument();
  });

  it("separates evidence-skip and other warnings correctly", () => {
    const finalization = makeFinalization({
      aggregateSummary: "Summary",
      warnings: [
        makeWarning({
          code: "evidence_skipped_disabled",
          message: "Agent disabled",
        }),
        makeWarning({
          code: "restart_retry",
          message: "Restart warning",
        }),
      ],
    });
    render(
      <PostRunStatusBadge execution={makeExecution({ finalization })} />,
    );
    // Evidence-skip visible immediately
    expect(screen.getByText("Agent disabled")).toBeInTheDocument();
    // Other warning hidden
    expect(screen.queryByText("Restart warning")).not.toBeInTheDocument();
    // Expand to see other warning
    fireEvent.click(screen.getByTestId("post-run-status-badge").querySelector("button")!);
    expect(screen.getByText("Restart warning")).toBeInTheDocument();
  });

  it("shows progress stepper when validating", () => {
    const finalization = makeFinalization({
      status: "running",
      phase: "restarting",
      startedAt: "2026-03-20T00:00:00Z",
    });
    render(
      <PostRunStatusBadge
        execution={makeExecution({ status: "validating", finalization })}
      />,
    );
    expect(screen.getByTestId("post-run-validating-indicator")).toBeInTheDocument();
  });
});
