import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { StalePlanPanel } from "./stale-plan-panel";
import { extractMissingPaths } from "./stale-plan-utils";

vi.mock("../../services", () => ({
  backlogService: {
    reWorkshop: vi.fn().mockResolvedValue({
      deletedRounds: 3,
      statusReverted: true,
    }),
  },
}));

import { backlogService } from "../../services";

describe("StalePlanPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders the missing paths and the re-workshop button", () => {
    render(
      <StalePlanPanel
        kind="research"
        name="agent-sandbox-auditability-contract"
        missingPaths={[
          {
            glob: "scripts/lib/scenario/**",
            resolved: "scripts/lib/scenario",
            reason: "path does not exist under project root",
          },
          {
            glob: "cli/commands/scenario/**",
            resolved: "cli/commands",
            reason: "path does not exist under project root",
          },
        ]}
      />,
    );

    expect(
      screen.getByText(/this plan references paths that no longer exist/i),
    ).toBeInTheDocument();
    expect(screen.getByText("scripts/lib/scenario/**")).toBeInTheDocument();
    expect(screen.getByText("cli/commands/scenario/**")).toBeInTheDocument();
    expect(
      screen.getByTestId("stale-plan-reworkshop-button"),
    ).toBeInTheDocument();
  });

  it("calls backlogService.reWorkshop and onReWorkshopped when the button is clicked", async () => {
    const onReWorkshopped = vi.fn();
    render(
      <StalePlanPanel
        kind="research"
        name="agent-sandbox-auditability-contract"
        missingPaths={[{ glob: "x/**" }]}
        onReWorkshopped={onReWorkshopped}
      />,
    );

    fireEvent.click(screen.getByTestId("stale-plan-reworkshop-button"));

    await waitFor(() => {
      expect(backlogService.reWorkshop).toHaveBeenCalledWith(
        "research",
        "agent-sandbox-auditability-contract",
      );
      expect(onReWorkshopped).toHaveBeenCalled();
    });
  });

  it("calls onCancel when the cancel button is clicked", () => {
    const onCancel = vi.fn();
    render(
      <StalePlanPanel
        kind="research"
        name="x"
        missingPaths={[]}
        onCancel={onCancel}
      />,
    );
    fireEvent.click(screen.getByText("Cancel"));
    expect(onCancel).toHaveBeenCalled();
  });
});

describe("extractMissingPaths", () => {
  it("parses snake_case and camelCase keys", () => {
    const result = extractMissingPaths({
      missingPaths: [
        { glob: "a/**", resolved: "a", reason: "missing" },
        { Glob: "b/**", Resolved: "b", Reason: "missing too" },
      ],
    });
    expect(result).toEqual([
      { glob: "a/**", resolved: "a", reason: "missing" },
      { glob: "b/**", resolved: "b", reason: "missing too" },
    ]);
  });

  it("returns [] for malformed input", () => {
    expect(extractMissingPaths(null)).toEqual([]);
    expect(extractMissingPaths({})).toEqual([]);
    expect(extractMissingPaths({ missingPaths: "no" })).toEqual([]);
  });
});
