import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { GitHistory } from "./GitHistory";

vi.mock("@tanstack/react-virtual", () => ({
  useVirtualizer: ({ count }: { count: number }) => ({
    getTotalSize: () => count * 80,
    getVirtualItems: () =>
      Array.from({ length: count }, (_, index) => ({
        index,
        start: index * 80,
        size: 80,
        key: index
      })),
    measureElement: vi.fn()
  })
}));

// Minimal history lines with pN pattern commits and a non-pN commit
const HISTORY_LINES = [
  "* abc1234 web-console TTS p10",
  "* def5678 web-console TTS p9",
  "* fff9999 fix typo in readme",
];

const ENTRIES = [
  { hash: "abc1234", subject: "web-console TTS p10", files: ["src/a.ts"] },
  { hash: "def5678", subject: "web-console TTS p9", files: ["src/b.ts"] },
  { hash: "fff9999", subject: "fix typo in readme", files: ["README.md"] },
];

function renderHistory(overrides: Partial<React.ComponentProps<typeof GitHistory>> = {}) {
  return render(
    <GitHistory
      lines={HISTORY_LINES}
      entries={ENTRIES}
      isLoading={false}
      {...overrides}
    />
  );
}

describe("GitHistory continue action", () => {
  it("renders continue button only on pN-matching entries", () => {
    const onContinue = vi.fn();
    renderHistory({ onContinueCommit: onContinue });

    const buttons = screen.getAllByTestId("history-continue-btn");
    // Two entries match pN, one doesn't
    expect(buttons).toHaveLength(2);
  });

  it("does not render continue buttons when callback is not provided", () => {
    renderHistory();
    expect(screen.queryAllByTestId("history-continue-btn")).toHaveLength(0);
  });

  it("calls onContinueCommit with incremented message", () => {
    const onContinue = vi.fn();
    renderHistory({ onContinueCommit: onContinue });

    const buttons = screen.getAllByTestId("history-continue-btn");
    fireEvent.click(buttons[0] as HTMLElement);

    expect(onContinue).toHaveBeenCalledWith("web-console TTS p11");
  });

  it("increments correctly for second entry", () => {
    const onContinue = vi.fn();
    renderHistory({ onContinueCommit: onContinue });

    const buttons = screen.getAllByTestId("history-continue-btn");
    fireEvent.click(buttons[1] as HTMLElement);

    expect(onContinue).toHaveBeenCalledWith("web-console TTS p10");
  });
});

describe("GitHistory filter group action", () => {
  it("renders filter group button only on pN-matching entries", () => {
    const onFilter = vi.fn();
    renderHistory({ onFilterGroup: onFilter });

    const buttons = screen.getAllByTestId("history-filter-group-btn");
    expect(buttons).toHaveLength(2);
  });

  it("does not render filter buttons when callback is not provided", () => {
    renderHistory();
    expect(screen.queryAllByTestId("history-filter-group-btn")).toHaveLength(0);
  });

  it("calls onFilterGroup with the correct prefix", () => {
    const onFilter = vi.fn();
    renderHistory({ onFilterGroup: onFilter });

    const buttons = screen.getAllByTestId("history-filter-group-btn");
    fireEvent.click(buttons[0] as HTMLElement);

    expect(onFilter).toHaveBeenCalledWith("web-console TTS");
  });
});

describe("GitHistory group filter banner", () => {
  it("renders banner when activeGroupFilter is set", () => {
    const onClear = vi.fn();
    renderHistory({
      activeGroupFilter: { prefix: "web-console TTS", count: 10 },
      onClearGroupFilter: onClear,
    });

    expect(screen.getByText("web-console TTS (10 commits)")).toBeInTheDocument();
  });

  it("does not render banner when activeGroupFilter is null", () => {
    renderHistory({ activeGroupFilter: null });
    expect(screen.queryByTestId("clear-group-filter-btn")).not.toBeInTheDocument();
  });

  it("calls onClearGroupFilter when clear button is clicked", () => {
    const onClear = vi.fn();
    renderHistory({
      activeGroupFilter: { prefix: "web-console TTS", count: 5 },
      onClearGroupFilter: onClear,
    });

    fireEvent.click(screen.getByTestId("clear-group-filter-btn"));
    expect(onClear).toHaveBeenCalledOnce();
  });
});

describe("GitHistory commit check badges", () => {
  it("renders precommit status badge for entries with recorded checks", () => {
    renderHistory({
      entries: [
        {
          hash: "abc1234",
          subject: "web-console TTS p10",
          files: ["src/a.ts"],
          checks: [{
            kind: "precommit",
            status: "passed",
            command: "custom check",
            exit_code: 0,
            summary: "checks passed",
            duration_ms: 12,
            timestamp: "2026-05-09T12:00:00Z"
          }]
        },
        ...ENTRIES.slice(1)
      ]
    });

    expect(screen.getByTestId("history-precommit-badge")).toHaveTextContent("precommit passed");
  });
});
