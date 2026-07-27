import { fireEvent, render, screen, waitFor } from "@/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DesktopRecordItemView } from "../records/recordPresentation";
import { RecordsManager } from "./RecordsManager";

const mocks = vi.hoisted(() => ({
  deleteDesktopBuild: vi.fn(),
  fetchDesktopRecords: vi.fn(),
  moveDesktopRecord: vi.fn(),
  presentDesktopRecords: vi.fn(),
}));

vi.mock("../../lib/api", () => ({
  deleteDesktopBuild: mocks.deleteDesktopBuild,
  fetchDesktopRecords: mocks.fetchDesktopRecords,
  moveDesktopRecord: mocks.moveDesktopRecord,
}));
vi.mock("../records/recordPresentation", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../records/recordPresentation")>()),
  presentDesktopRecords: mocks.presentDesktopRecords,
}));
vi.mock("../records/AppCard", () => ({
  AppCard: ({ item, onClick }: { item: DesktopRecordItemView; onClick: () => void }) => (
    <button type="button" onClick={onClick}>Open {item.record.scenario_name}</button>
  ),
}));
vi.mock("../records/AppDetailDrawer", () => ({
  AppDetailDrawer: ({
    item,
    open,
    onClose,
    onMove,
    onDelete,
    onSwitchTemplate,
    onEditSigning,
    onRebuildWithSigning,
  }: {
    item: DesktopRecordItemView | null;
    open: boolean;
    onClose: () => void;
    onMove: (recordId: string, target: "destination" | "custom", path?: string) => void;
    onDelete: (scenarioName: string) => void;
    onSwitchTemplate?: (scenarioName: string, templateType?: string) => void;
    onEditSigning?: (scenarioName: string) => void;
    onRebuildWithSigning?: (scenarioName: string) => void;
  }) => {
    if (!open || !item) return null;
    const { record } = item;
    return (
    <section aria-label="Selected desktop app">
      <p>{record.scenario_name}</p>
      <button type="button" onClick={() => { onMove(record.id, "destination"); }}>Move selected</button>
      <button type="button" onClick={() => { onDelete(record.scenario_name); }}>Delete selected</button>
      <button type="button" onClick={() => { onSwitchTemplate?.(record.scenario_name, "electron"); }}>Switch template</button>
      <button type="button" onClick={() => { onEditSigning?.(record.scenario_name); }}>Edit signing</button>
      <button type="button" onClick={() => { onRebuildWithSigning?.(record.scenario_name); }}>Rebuild signing</button>
      <button type="button" onClick={onClose}>Close drawer</button>
    </section>
    );
  },
}));

const item: DesktopRecordItemView = {
  record: {
    id: "record-1",
    build_id: "build-1",
    scenario_name: "canvas-lab",
    output_path: "/tmp/canvas-lab",
  },
  has_build: true,
  build_state: "ready",
};

describe("RecordsManager", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.fetchDesktopRecords.mockResolvedValue({});
    mocks.presentDesktopRecords.mockReturnValue([]);
    mocks.moveDesktopRecord.mockResolvedValue({});
    mocks.deleteDesktopBuild.mockResolvedValue({});
  });

  it("shows loading and empty-record states", async () => {
    let resolveRecords: (value: unknown) => void = () => undefined;
    mocks.fetchDesktopRecords.mockImplementation(() => new Promise((resolve) => {
      resolveRecords = resolve;
    }));
    render(<RecordsManager />);
    expect(screen.getByText("Loading generated apps…")).toBeInTheDocument();
    resolveRecords({});
    expect(await screen.findByText("No desktop apps yet")).toBeInTheDocument();
  });

  it("shows a records retrieval failure", async () => {
    mocks.fetchDesktopRecords.mockRejectedValue(new Error("records unavailable"));
    render(<RecordsManager />);
    expect(await screen.findByText("records unavailable")).toBeInTheDocument();
  });

  it("opens a selected record and executes move, delete, and parent actions", async () => {
    mocks.presentDesktopRecords.mockReturnValue([item]);
    const onSwitchTemplate = vi.fn();
    const onEditSigning = vi.fn();
    const onRebuildWithSigning = vi.fn();
    render(
      <RecordsManager
        onSwitchTemplate={onSwitchTemplate}
        onEditSigning={onEditSigning}
        onRebuildWithSigning={onRebuildWithSigning}
      />,
    );

    await screen.findAllByRole("button", { name: "Open canvas-lab" });
    expect(screen.getByText("1")).toBeInTheDocument();
    const [firstCard] = screen.getAllByRole("button", { name: "Open canvas-lab" });
    if (!firstCard) throw new Error("Expected a desktop record card");
    fireEvent.click(firstCard);
    expect(screen.getByRole("region", { name: "Selected desktop app" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Move selected" }));
    await waitFor(() => {
      expect(mocks.moveDesktopRecord).toHaveBeenCalledWith("record-1", {
        target: "destination",
        destination_path: undefined,
      });
    });
    expect(await screen.findByText("Move updated.")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Switch template" }));
    fireEvent.click(screen.getByRole("button", { name: "Edit signing" }));
    fireEvent.click(screen.getByRole("button", { name: "Rebuild signing" }));
    expect(onSwitchTemplate).toHaveBeenCalledWith("canvas-lab", "electron");
    expect(onEditSigning).toHaveBeenCalledWith("canvas-lab");
    expect(onRebuildWithSigning).toHaveBeenCalledWith("canvas-lab");

    fireEvent.click(screen.getByRole("button", { name: "Delete selected" }));
    await waitFor(() => {
      expect(mocks.deleteDesktopBuild).toHaveBeenCalledWith("canvas-lab");
    });
    await waitFor(() => {
      expect(screen.queryByRole("region", { name: "Selected desktop app" })).not.toBeInTheDocument();
    });
  });

  it("refreshes records and surfaces a failed move", async () => {
    mocks.presentDesktopRecords.mockReturnValue([item]);
    mocks.moveDesktopRecord.mockRejectedValue(new Error("move unavailable"));
    render(<RecordsManager />);
    await screen.findAllByRole("button", { name: "Open canvas-lab" });

    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    await waitFor(() => {
      expect(mocks.fetchDesktopRecords).toHaveBeenCalledTimes(2);
    });
    const [firstCard] = screen.getAllByRole("button", { name: "Open canvas-lab" });
    if (!firstCard) throw new Error("Expected a desktop record card");
    fireEvent.click(firstCard);
    fireEvent.click(screen.getByRole("button", { name: "Move selected" }));
    expect(await screen.findByText("move unavailable")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Close drawer" }));
    expect(screen.queryByRole("region", { name: "Selected desktop app" })).not.toBeInTheDocument();
  });
});
