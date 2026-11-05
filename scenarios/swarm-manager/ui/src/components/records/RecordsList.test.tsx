import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import { RecordsList } from "./RecordsList";
import { renderWithProviders } from "../../test-utils";
import type { RecordItem } from "../../types";

function make(overrides: Partial<RecordItem> = {}): RecordItem {
  return {
    id: overrides.id ?? "01HFAKEID",
    kind: overrides.kind ?? "fix",
    scenario: overrides.scenario ?? "audio-tools",
    trigger: overrides.trigger ?? "voice auto-stop race",
    approach: overrides.approach ?? "added silence_timed_out flag",
    ruledOut: overrides.ruledOut ?? [],
    filesChanged: overrides.filesChanged ?? [],
    outcome: overrides.outcome ?? "shipped",
    stub: overrides.stub ?? false,
    createdAt: overrides.createdAt ?? "2026-05-27T12:00:00Z",
    ...overrides,
  };
}

describe("RecordsList", () => {
  it("renders a record card per item by default", () => {
    const items = [make({ id: "a" }), make({ id: "b", scenario: "web-console", kind: "execute" })];
    renderWithProviders(<RecordsList records={items} />);
    expect(screen.getByTestId("record-card-a")).toBeInTheDocument();
    expect(screen.getByTestId("record-card-b")).toBeInTheDocument();
  });

  it("hides stub records by default and reveals when toggled", () => {
    const items = [make({ id: "filled" }), make({ id: "stub-1", stub: true })];
    const onIncludeStubs = vi.fn();
    const { rerender } = renderWithProviders(
      <RecordsList records={items} onIncludeStubsChange={onIncludeStubs} />,
    );
    expect(screen.getByTestId("record-card-filled")).toBeInTheDocument();
    expect(screen.queryByTestId("record-card-stub-1")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("records-filter-include-stubs"));
    expect(onIncludeStubs).toHaveBeenCalledWith(true);

    rerender(<RecordsList records={items} includeStubs onIncludeStubsChange={onIncludeStubs} />);
    expect(screen.getByTestId("record-card-stub-1")).toBeInTheDocument();
  });

  it("filters by kind", () => {
    const items = [make({ id: "fix-1", kind: "fix" }), make({ id: "exec-1", kind: "execute" })];
    renderWithProviders(<RecordsList records={items} kindFilter="fix" />);
    expect(screen.getByTestId("record-card-fix-1")).toBeInTheDocument();
    expect(screen.queryByTestId("record-card-exec-1")).not.toBeInTheDocument();
  });

  it("renders empty state when no matches", () => {
    renderWithProviders(<RecordsList records={[]} />);
    expect(screen.getByText(/No records match/i)).toBeInTheDocument();
  });
});
