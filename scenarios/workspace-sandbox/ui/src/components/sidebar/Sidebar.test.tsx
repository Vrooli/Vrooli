/**
 * Sidebar render tests — exercises the live tab/search/filter behavior
 * that the reducer alone can't validate.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { Sidebar } from "./Sidebar";
import { sidebarReducer, createInitialState } from "./useSidebarState";
import type { Sandbox, DiffArchive } from "../../lib/api";

vi.mock("../../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../lib/api")>();
  return {
    ...actual,
    listHistory: vi.fn(),
  };
});

const apiModule = await import("../../lib/api");
const listHistoryMock = apiModule.listHistory as unknown as ReturnType<typeof vi.fn>;

function makeSandbox(overrides: Partial<Sandbox> = {}): Sandbox {
  return {
    id: "00000000-0000-0000-0000-000000000001",
    name: undefined,
    scopePath: "/repo/active-thing",
    reservedPath: "/repo/active-thing",
    reservedPaths: [],
    noLock: false,
    projectRoot: "/repo",
    owner: "agent-abc",
    ownerType: "agent",
    status: "active",
    createdAt: new Date().toISOString(),
    lastUsedAt: new Date().toISOString(),
    driverId: "overlayfs",
    driverVersion: "1.0",
    sizeBytes: 1024,
    fileCount: 5,
    activePids: [],
    sessionCount: 0,
    ...overrides,
  };
}

function makeArchive(overrides: Partial<DiffArchive> = {}): DiffArchive {
  return {
    sandboxId: "11111111-1111-1111-1111-111111111111",
    snapshotAt: new Date().toISOString(),
    archiveState: "complete",
    sandboxStatus: "approved",
    files: [],
    stats: {
      filesChanged: 0,
      filesAdded: 0,
      filesModified: 0,
      filesDeleted: 0,
      linesAdded: 0,
      linesRemoved: 0,
      totalBytes: 0,
    },
    totalBlobBytes: 2048,
    projectRoot: "/repo",
    owner: "agent-xyz",
    ...overrides,
  };
}

function Harness({
  sandboxes,
  initialTab = "active",
  onSelectArchive,
  onSelectActive,
}: {
  sandboxes: Sandbox[];
  initialTab?: "active" | "history";
  onSelectArchive?: (a: DiffArchive) => void;
  onSelectActive?: (s: Sandbox) => void;
}) {
  // Use a real reducer instance bound by `useReducer` inside a tiny
  // wrapper. We avoid the localStorage-restoring `useSidebarState` so
  // tests don't bleed into one another.
  const [state, dispatch] = useReducerLocal(initialTab);
  return (
    <Sidebar
      sandboxes={sandboxes}
      isLoading={false}
      onSelectActive={onSelectActive ?? (() => {})}
      onSelectHistory={(a) => onSelectArchive?.(a)}
      state={state}
      dispatch={dispatch}
    />
  );
}

import { useReducer } from "react";
function useReducerLocal(initialTab: "active" | "history") {
  const init = createInitialState();
  return useReducer(sidebarReducer, { ...init, activeTab: initialTab });
}

function renderWithQueryClient(ui: React.ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
}

describe("Sidebar", () => {
  beforeEach(() => {
    listHistoryMock.mockReset();
    listHistoryMock.mockResolvedValue({
      archives: [],
      totalCount: 0,
      limit: 100,
      offset: 0,
    });
  });

  it("renders the active list and tab counts", () => {
    const sandboxes = [
      makeSandbox({ id: "a1", status: "active", scopePath: "/repo/sb1" }),
      makeSandbox({ id: "a2", status: "stopped", scopePath: "/repo/sb2" }),
    ];
    renderWithQueryClient(<Harness sandboxes={sandboxes} />);
    expect(screen.getByTestId("sidebar-active-tab")).toBeInTheDocument();
    expect(screen.getByTestId("sidebar-tab-active-count")).toHaveTextContent("2");
    expect(screen.queryByTestId("sidebar-history-tab")).not.toBeInTheDocument();
  });

  it("does NOT call listHistory while on the Active tab", () => {
    renderWithQueryClient(<Harness sandboxes={[makeSandbox({ status: "active" })]} />);
    expect(listHistoryMock).not.toHaveBeenCalled();
  });

  it("calls listHistory when the user switches to the History tab", async () => {
    listHistoryMock.mockResolvedValueOnce({
      archives: [makeArchive()],
      totalCount: 1,
      limit: 100,
      offset: 0,
    });
    renderWithQueryClient(<Harness sandboxes={[]} />);
    await act(async () => {
      fireEvent.click(screen.getByTestId("sidebar-tab-history"));
    });
    await waitFor(() => expect(listHistoryMock).toHaveBeenCalled());
    await waitFor(() => expect(screen.queryByTestId("sidebar-history-list")).toBeInTheDocument());
  });

  it("History tab renders archives and surfaces the archive size", async () => {
    listHistoryMock.mockResolvedValueOnce({
      archives: [makeArchive({ totalBlobBytes: 4096, sandboxStatus: "approved" })],
      totalCount: 1,
      limit: 100,
      offset: 0,
    });
    renderWithQueryClient(<Harness sandboxes={[]} initialTab="history" />);
    await waitFor(() => expect(screen.queryByTestId("sidebar-history-list")).toBeInTheDocument());
    // Size formatting from formatBytes; expect KB unit.
    expect(screen.getByTestId("sidebar-history-list")).toHaveTextContent(/4(\.0)? KB/);
  });

  it("clicking a history row invokes onSelectHistory with the archive", async () => {
    const archive = makeArchive({ sandboxId: "deadbeef-dead-beef-dead-beef00000001" });
    listHistoryMock.mockResolvedValueOnce({
      archives: [archive],
      totalCount: 1,
      limit: 100,
      offset: 0,
    });
    const onSelectArchive = vi.fn();
    renderWithQueryClient(
      <Harness sandboxes={[]} initialTab="history" onSelectArchive={onSelectArchive} />,
    );
    await waitFor(() => expect(screen.queryByTestId("sidebar-history-list")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("sidebar-history-list").querySelector('[data-testid="sandbox-item"]')!);
    expect(onSelectArchive).toHaveBeenCalledWith(expect.objectContaining({ sandboxId: archive.sandboxId }));
  });

  it("filters Active tab client-side by free-text search", () => {
    const sandboxes = [
      makeSandbox({ id: "a1", status: "active", scopePath: "/repo/alpha" }),
      makeSandbox({ id: "a2", status: "active", scopePath: "/repo/beta" }),
    ];
    renderWithQueryClient(<Harness sandboxes={sandboxes} />);
    fireEvent.change(screen.getByTestId("sidebar-search"), { target: { value: "alpha" } });
    const list = screen.getByTestId("sidebar-active-list");
    const items = list.querySelectorAll('[data-testid="sandbox-item"]');
    expect(items).toHaveLength(1);
    expect(items[0]).toHaveAttribute("data-sandbox-id", "a1");
  });

  it("history-tab status filter forwards into the listHistory request", async () => {
    listHistoryMock.mockResolvedValue({
      archives: [],
      totalCount: 0,
      limit: 100,
      offset: 0,
    });
    renderWithQueryClient(<Harness sandboxes={[]} initialTab="history" />);
    await waitFor(() => expect(listHistoryMock).toHaveBeenCalledTimes(1));

    // Open filter bar, toggle the rejected status pill.
    fireEvent.click(screen.getByTestId("sidebar-filter-toggle"));
    fireEvent.click(screen.getByTestId("sidebar-filter-status-rejected"));

    await waitFor(() => {
      expect(listHistoryMock).toHaveBeenLastCalledWith(
        expect.objectContaining({ statuses: ["rejected"] }),
      );
    });
  });

  it("Active tab excludes terminal-status sandboxes (deleted is not rendered)", () => {
    const sandboxes = [
      makeSandbox({ id: "a1", status: "active" }),
      makeSandbox({ id: "d1", status: "deleted" }),
    ];
    renderWithQueryClient(<Harness sandboxes={sandboxes} />);
    const list = screen.getByTestId("sidebar-active-list");
    const items = list.querySelectorAll('[data-testid="sandbox-item"]');
    expect(items).toHaveLength(1);
    expect(items[0]).toHaveAttribute("data-sandbox-id", "a1");
  });

  it("error-status sandboxes render in the Active tab (operationally interesting)", () => {
    const sandboxes = [makeSandbox({ id: "e1", status: "error", errorMessage: "boom" })];
    renderWithQueryClient(<Harness sandboxes={sandboxes} />);
    expect(screen.getByTestId("sidebar-active-list")).toBeInTheDocument();
    const item = screen.getByTestId("sandbox-item");
    expect(item).toHaveAttribute("data-sandbox-status", "error");
  });
});
