import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ArrowUpRight } from "lucide-react";
import { DependencyChipList } from "./dependency-chip-list";
import type { ResolvedDependency } from "../../lib/backlog-queue-utils";
import type { BacklogStatus } from "../../types";

function makeDep(overrides?: Partial<ResolvedDependency>): ResolvedDependency {
  return {
    kind: "idea",
    name: "dep-item",
    title: "Dep Item",
    status: "ready" as BacklogStatus,
    ...overrides,
  };
}

function renderChips(items: ResolvedDependency[], label = "Depends on") {
  return render(
    <MemoryRouter>
      <DependencyChipList label={label} items={items} icon={ArrowUpRight} />
    </MemoryRouter>,
  );
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

  it("renders correct number of chips", () => {
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

  it("links to the correct backlog details URL", () => {
    renderChips([makeDep({ kind: "fix", name: "broken-thing", title: "Broken" })]);
    const link = screen.getByText("Broken").closest("a");
    expect(link).toHaveAttribute("href", "/backlog/fix/broken-thing");
  });

  it("applies status-based color classes", () => {
    renderChips([makeDep({ status: "completed" as BacklogStatus, title: "Done" })]);
    const chip = screen.getByText("Done");
    expect(chip.className).toContain("bg-emerald");
  });

  it("shows status in title tooltip", () => {
    renderChips([makeDep({ status: "in_progress" as BacklogStatus, title: "Working" })]);
    const chip = screen.getByText("Working");
    expect(chip).toHaveAttribute("title", "In progress");
  });
});
