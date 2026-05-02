import { describe, it, expect, vi } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ArrowUpRight } from "lucide-react";
import { useLocation } from "react-router-dom";
import { DependencyChipList } from "./dependency-chip-list";
import type { ResolvedDependency } from "../../lib/backlog-queue-utils";
import type { BacklogStatus } from "../../types";
import { renderWithProviders } from "../../test-utils";

function makeDep(overrides?: Partial<ResolvedDependency>): ResolvedDependency {
  return {
    kind: "idea",
    name: "dep-item",
    title: "Dep Item",
    status: "ready" as BacklogStatus,
    ...overrides,
  };
}

function renderChips(
  items: ResolvedDependency[],
  label = "Depends on",
  onStatusChange?: (dep: ResolvedDependency, newStatus: BacklogStatus) => void,
) {
  return renderWithProviders(
    <DependencyChipList label={label} items={items} icon={ArrowUpRight} onStatusChange={onStatusChange} />,
  );
}

function LocationProbe() {
  const location = useLocation();
  return <span data-testid="location-path">{location.pathname}</span>;
}

describe("DependencyChipList", () => {
  it("renders nothing when items array is empty", () => {
    const { container } = renderChips([]);
    expect(container.innerHTML).toBe("");
  });

  it("renders the label", () => {
    renderChips([makeDep()], "Depends on");
    expect(screen.getByText("Depends on")).toBeInTheDocument();
  });

  it("renders correct number of rows with titles", () => {
    const items = [
      makeDep({ name: "a", title: "Alpha" }),
      makeDep({ name: "b", title: "Beta" }),
      makeDep({ name: "c", title: "Gamma" }),
    ];
    renderChips(items);
    expect(screen.getByText("Alpha")).toBeInTheDocument();
    expect(screen.getByText("Beta")).toBeInTheDocument();
    expect(screen.getByText("Gamma")).toBeInTheDocument();
  });

  it("navigates to backlog detail when title is clicked", () => {
    renderWithProviders(
      <>
        <DependencyChipList
          label="Depends on"
          items={[makeDep({ kind: "fix", name: "broken-thing", title: "Broken" })]}
          icon={ArrowUpRight}
        />
        <LocationProbe />
      </>,
    );
    fireEvent.click(screen.getByText("Broken"));
    expect(screen.getByTestId("location-path")).toHaveTextContent("/backlog/fix/broken-thing");
  });

  it("renders a labeled status chip with formatted status text", () => {
    renderChips([makeDep({ status: "completed" as BacklogStatus, title: "Done" })]);
    const chip = screen.getByTestId("dep-status-chip-idea-dep-item");
    expect(chip).toHaveTextContent("Completed");
  });

  it("applies status-based color classes on the status chip", () => {
    renderChips([makeDep({ status: "completed" as BacklogStatus, title: "Done" })]);
    const chip = screen.getByTestId("dep-status-chip-idea-dep-item");
    expect(chip.className).toContain("bg-emerald");
  });

  it("uses formatted status as the chip tooltip", () => {
    renderChips([makeDep({ status: "in_progress" as BacklogStatus, title: "Working" })]);
    const chip = screen.getByTestId("dep-status-chip-idea-dep-item");
    expect(chip).toHaveAttribute("title", "In progress");
  });

  it("renders a static status chip (no popover trigger) when onStatusChange is not provided", () => {
    renderChips([makeDep()]);
    expect(screen.queryByTestId("dep-status-dot-idea-dep-item")).not.toBeInTheDocument();
    expect(screen.getByTestId("dep-status-chip-idea-dep-item")).toBeInTheDocument();
  });

  it("renders a clickable status chip trigger when onStatusChange is provided", () => {
    renderChips([makeDep()], "Depends on", vi.fn());
    expect(screen.getByTestId("dep-status-dot-idea-dep-item")).toBeInTheDocument();
  });

  it("opens popover on status chip click and calls onStatusChange", async () => {
    const user = userEvent.setup();
    const onStatusChange = vi.fn();
    const dep = makeDep({ status: "ready" as BacklogStatus });
    renderChips([dep], "Depends on", onStatusChange);

    await user.click(screen.getByTestId("dep-status-dot-idea-dep-item"));
    expect(screen.getByTestId("dep-status-popover-idea-dep-item")).toBeInTheDocument();

    await user.click(screen.getByTestId("dep-status-option-backlog"));
    expect(onStatusChange).toHaveBeenCalledWith(dep, "backlog");
  });

  describe("activity chip", () => {
    it("shows a purpose-specific chip with pulse when an agent is running", () => {
      const dep = makeDep({
        status: "ready" as BacklogStatus,
        activity: { purpose: "workshop", status: "running" },
      });
      renderChips([dep]);
      const chip = screen.getByTestId("dep-activity-chip-idea-dep-item");
      expect(chip).toHaveTextContent("Workshopping");
      expect(chip.querySelector(".animate-ping")).toBeTruthy();
    });

    it("shows a 'needs review' chip without pulse and in cyan tone", () => {
      const dep = makeDep({
        status: "researching" as BacklogStatus,
        activity: { purpose: "review", status: "needs_review" },
      });
      renderChips([dep]);
      const chip = screen.getByTestId("dep-activity-chip-idea-dep-item");
      expect(chip).toHaveTextContent("Reviewing");
      expect(chip.className).toContain("text-cyan");
      expect(chip.querySelector(".animate-ping")).toBeNull();
    });

    it("suppresses the activity chip when status=in_progress and purpose=process (redundant)", () => {
      const dep = makeDep({
        status: "in_progress" as BacklogStatus,
        activity: { purpose: "process", status: "running" },
      });
      renderChips([dep]);
      expect(screen.queryByTestId("dep-activity-chip-idea-dep-item")).not.toBeInTheDocument();
    });

    it("still shows the activity chip when status=in_progress but purpose differs (e.g., fixup)", () => {
      const dep = makeDep({
        status: "in_progress" as BacklogStatus,
        activity: { purpose: "fixup", status: "running" },
      });
      renderChips([dep]);
      expect(screen.getByTestId("dep-activity-chip-idea-dep-item")).toHaveTextContent("Fixing up");
    });
  });

  describe("attention badge", () => {
    it("shows pending-decisions badge when no agent is active", () => {
      const dep = makeDep({
        status: "researching" as BacklogStatus,
        attentionReasons: [{ kind: "pending-decisions", count: 3 }],
      });
      renderChips([dep]);
      expect(screen.getByText("3 decisions")).toBeInTheDocument();
    });

    it("suppresses the attention badge when an activity chip is rendered", () => {
      const dep = makeDep({
        status: "researching" as BacklogStatus,
        activity: { purpose: "workshop", status: "running" },
        attentionReasons: [{ kind: "pending-decisions", count: 3 }],
      });
      renderChips([dep]);
      expect(screen.queryByText("3 decisions")).not.toBeInTheDocument();
      expect(screen.getByTestId("dep-activity-chip-idea-dep-item")).toBeInTheDocument();
    });

    it("renders plan-ready and review-ready badges when applicable", () => {
      const dep1 = makeDep({
        name: "p",
        status: "ready" as BacklogStatus,
        attentionReasons: [{ kind: "plan-ready" }],
      });
      const dep2 = makeDep({
        name: "r",
        status: "researching" as BacklogStatus,
        attentionReasons: [{ kind: "research-complete" }],
      });
      renderChips([dep1, dep2]);
      expect(screen.getByText("Plan ready")).toBeInTheDocument();
      expect(screen.getByText("Review ready")).toBeInTheDocument();
    });
  });
});
