import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RuleResultCard } from "./RuleResultCard";
import type { RuleResult, RuleWithState } from "../lib/api";

const noop = () => {};

function makeResult(overrides: Partial<RuleResult> = {}): RuleResult {
  return {
    rule_id: "TEST_RULE",
    passed: false,
    started_at: "2026-01-01T00:00:00Z",
    finished_at: "2026-01-01T00:00:01Z",
    error_count: 0,
    warn_count: 0,
    ...overrides,
  };
}

function makeRuleDef(overrides: Partial<RuleWithState> = {}): RuleWithState {
  return {
    id: "TEST_RULE",
    title: "Test Rule",
    summary: "A test rule",
    why_important: "Testing",
    category: "test",
    severity: "error",
    default_enabled: true,
    fixable: false,
    enabled: true,
    ...overrides,
  };
}

const defaultProps = {
  onToggleExpand: noop,
  onFix: noop,
  fixPending: false,
  dryRun: false,
  onToggleDryRun: noop,
};

describe("RuleResultCard", () => {
  describe("passing rule", () => {
    it("shows 'No findings' and check icon for passing rules", () => {
      render(
        <RuleResultCard
          result={makeResult({ passed: true })}
          ruleDefinition={makeRuleDef()}
          expanded={false}
          {...defaultProps}
        />
      );
      expect(screen.getByText("No findings")).toBeInTheDocument();
    });
  });

  describe("error/warn/info count display", () => {
    it("shows only errors when there are no warnings or info", () => {
      render(
        <RuleResultCard
          result={makeResult({
            error_count: 3,
            warn_count: 0,
            findings: [
              { level: "error", message: "e1" },
              { level: "error", message: "e2" },
              { level: "error", message: "e3" },
            ],
          })}
          ruleDefinition={makeRuleDef()}
          expanded={false}
          {...defaultProps}
        />
      );
      expect(screen.getByText("3 errors")).toBeInTheDocument();
      expect(screen.queryByText(/warning/)).not.toBeInTheDocument();
      expect(screen.queryByText(/info/)).not.toBeInTheDocument();
    });

    it("shows only warnings when there are no errors or info", () => {
      render(
        <RuleResultCard
          result={makeResult({
            error_count: 0,
            warn_count: 2,
            findings: [
              { level: "warn", message: "w1" },
              { level: "warn", message: "w2" },
            ],
          })}
          ruleDefinition={makeRuleDef()}
          expanded={false}
          {...defaultProps}
        />
      );
      expect(screen.getByText("2 warnings")).toBeInTheDocument();
      expect(screen.queryByText(/error/i)).not.toBeInTheDocument();
    });

    it("shows only info when there are no errors or warnings", () => {
      render(
        <RuleResultCard
          result={makeResult({
            error_count: 0,
            warn_count: 0,
            findings: [
              { level: "info", message: "i1" },
              { level: "info", message: "i2" },
              { level: "info", message: "i3" },
            ],
          })}
          ruleDefinition={makeRuleDef()}
          expanded={false}
          {...defaultProps}
        />
      );
      expect(screen.getByText("3 info")).toBeInTheDocument();
    });

    it("shows all three counts when mixed", () => {
      render(
        <RuleResultCard
          result={makeResult({
            error_count: 2,
            warn_count: 1,
            findings: [
              { level: "error", message: "e1" },
              { level: "error", message: "e2" },
              { level: "warn", message: "w1" },
              { level: "info", message: "i1" },
            ],
          })}
          ruleDefinition={makeRuleDef()}
          expanded={false}
          {...defaultProps}
        />
      );
      expect(screen.getByText("2 errors")).toBeInTheDocument();
      expect(screen.getByText("1 warning")).toBeInTheDocument();
      expect(screen.getByText("1 info")).toBeInTheDocument();
    });

    it("uses singular 'error' for count of 1", () => {
      render(
        <RuleResultCard
          result={makeResult({
            error_count: 1,
            warn_count: 0,
            findings: [{ level: "error", message: "e1" }],
          })}
          ruleDefinition={makeRuleDef()}
          expanded={false}
          {...defaultProps}
        />
      );
      expect(screen.getByText("1 error")).toBeInTheDocument();
    });

    it("uses singular 'warning' for count of 1", () => {
      render(
        <RuleResultCard
          result={makeResult({
            error_count: 0,
            warn_count: 1,
            findings: [{ level: "warn", message: "w1" }],
          })}
          ruleDefinition={makeRuleDef()}
          expanded={false}
          {...defaultProps}
        />
      );
      expect(screen.getByText("1 warning")).toBeInTheDocument();
    });

    it("shows 'Failed (no details available)' when failed with zero counts and no findings", () => {
      render(
        <RuleResultCard
          result={makeResult({ passed: false, error_count: 0, warn_count: 0, findings: [] })}
          ruleDefinition={makeRuleDef()}
          expanded={false}
          {...defaultProps}
        />
      );
      expect(screen.getByText("Failed (no details available)")).toBeInTheDocument();
    });
  });

  describe("color coding", () => {
    it("applies red color to error count", () => {
      const { container } = render(
        <RuleResultCard
          result={makeResult({
            error_count: 1,
            findings: [{ level: "error", message: "e1" }],
          })}
          ruleDefinition={makeRuleDef()}
          expanded={false}
          {...defaultProps}
        />
      );
      const errorSpans = container.querySelectorAll(".text-red-300");
      const errorCountSpan = Array.from(errorSpans).find((el) => el.textContent === "1 error");
      expect(errorCountSpan).toBeTruthy();
    });

    it("applies amber color to warning count", () => {
      const { container } = render(
        <RuleResultCard
          result={makeResult({
            warn_count: 2,
            findings: [
              { level: "warn", message: "w1" },
              { level: "warn", message: "w2" },
            ],
          })}
          ruleDefinition={makeRuleDef()}
          expanded={false}
          {...defaultProps}
        />
      );
      const warnSpan = container.querySelector(".text-amber-300");
      expect(warnSpan).toBeTruthy();
      expect(warnSpan?.textContent).toBe("2 warnings");
    });

    it("applies slate color to info count", () => {
      const { container } = render(
        <RuleResultCard
          result={makeResult({
            findings: [{ level: "info", message: "i1" }],
          })}
          ruleDefinition={makeRuleDef()}
          expanded={false}
          {...defaultProps}
        />
      );
      const infoSpan = container.querySelector(".text-slate-300");
      expect(infoSpan).toBeTruthy();
      expect(infoSpan?.textContent).toBe("1 info");
    });
  });

  describe("expand/collapse", () => {
    it("calls onToggleExpand when button is clicked", async () => {
      const user = userEvent.setup();
      const onToggle = vi.fn();
      render(
        <RuleResultCard
          result={makeResult({ passed: true })}
          ruleDefinition={makeRuleDef()}
          expanded={false}
          {...defaultProps}
          onToggleExpand={onToggle}
        />
      );
      await user.click(screen.getByRole("button"));
      expect(onToggle).toHaveBeenCalledOnce();
    });

    it("shows findings grouped by scenario when expanded", () => {
      render(
        <RuleResultCard
          result={makeResult({
            error_count: 2,
            findings: [
              { level: "error", message: "issue A", scenario_name: "foo" },
              { level: "error", message: "issue B", scenario_name: "bar" },
            ],
          })}
          ruleDefinition={makeRuleDef()}
          expanded={true}
          {...defaultProps}
        />
      );
      expect(screen.getByText("foo")).toBeInTheDocument();
      expect(screen.getByText("bar")).toBeInTheDocument();
      expect(screen.getByText("issue A")).toBeInTheDocument();
      expect(screen.getByText("issue B")).toBeInTheDocument();
    });

    it("does not render findings section when collapsed", () => {
      render(
        <RuleResultCard
          result={makeResult({
            error_count: 1,
            findings: [{ level: "error", message: "hidden issue", scenario_name: "baz" }],
          })}
          ruleDefinition={makeRuleDef()}
          expanded={false}
          {...defaultProps}
        />
      );
      expect(screen.queryByText("hidden issue")).not.toBeInTheDocument();
    });
  });

  describe("rule title display", () => {
    it("shows rule definition title when provided", () => {
      render(
        <RuleResultCard
          result={makeResult()}
          ruleDefinition={makeRuleDef({ title: "Custom Title" })}
          expanded={false}
          {...defaultProps}
        />
      );
      expect(screen.getByText("Custom Title")).toBeInTheDocument();
    });

    it("falls back to rule_id when no definition provided", () => {
      render(
        <RuleResultCard
          result={makeResult({ rule_id: "SOME_RULE_ID" })}
          expanded={false}
          {...defaultProps}
        />
      );
      expect(screen.getByText("SOME_RULE_ID")).toBeInTheDocument();
    });
  });

  describe("fix actions", () => {
    it("shows fix actions when rule is fixable and expanded with scenarios", () => {
      render(
        <RuleResultCard
          result={makeResult({
            error_count: 1,
            findings: [{ level: "error", message: "e1", scenario_name: "my-scenario" }],
          })}
          ruleDefinition={makeRuleDef({ fixable: true })}
          expanded={true}
          {...defaultProps}
        />
      );
      // The "Fix" button per-scenario and "Fix All" button should be present
      expect(screen.getByText("Fix")).toBeInTheDocument();
      expect(screen.getByText("Fix All (1)")).toBeInTheDocument();
    });

    it("does not show fix actions when rule is not fixable", () => {
      render(
        <RuleResultCard
          result={makeResult({
            error_count: 1,
            findings: [{ level: "error", message: "e1", scenario_name: "my-scenario" }],
          })}
          ruleDefinition={makeRuleDef({ fixable: false })}
          expanded={true}
          {...defaultProps}
        />
      );
      expect(screen.queryByText("Fix All (1)")).not.toBeInTheDocument();
    });
  });
});
