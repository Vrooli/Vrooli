import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@/test-utils";
import { CapturesDrawer } from "./CapturesDrawer";

// Mock the Drawer to render children directly (avoids portal/animation issues)
vi.mock("../ui/drawer", () => ({
  Drawer: ({ open, children }: { open: boolean; children: React.ReactNode }) =>
    open ? <div data-testid="drawer">{children}</div> : null,
  DrawerBody: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  DrawerHeader: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}));

vi.mock("../../lib/api/captures", () => ({
  buildCaptureFileUrl: (scenario: string, id: string) =>
    `/mock/${scenario}/${id}`,
}));

const mockState = {
  isOpen: true,
  close: vi.fn(),
  scenarioName: "my-app",
  captures: [] as Array<{
    captureId: string;
    scenarioName: string;
    kind: "screenshot" | "recording";
    filename: string;
    fileSizeBytes: bigint;
    sourceSessionId: string;
    createdAt: { seconds: bigint; nanos: number };
  }>,
  summary: { count: 0, totalBytes: 0n },
  selectedIds: new Set<string>(),
  loading: false,
  error: null as string | null,
  toggleSelect: vi.fn(),
  selectAll: vi.fn(),
  deselectAll: vi.fn(),
  deleteCapture: vi.fn(),
  deleteAll: vi.fn(),
  downloadSelected: vi.fn(),
};

vi.mock("../../store/capturesStore", () => ({
  useCapturesStore: (selector: (s: typeof mockState) => unknown) =>
    selector(mockState),
}));

const sampleCapture = (
  id: string,
  type: "screenshot" | "recording" = "screenshot",
) => ({
  captureId: id,
  scenarioName: "my-app",
  kind: type,
  filename: `${type}-123.${type === "screenshot" ? "png" : "mp4"}`,
  fileSizeBytes: 1024n,
  sourceSessionId: "session-1",
  createdAt: { seconds: BigInt(Math.floor(Date.now() / 1_000)), nanos: 0 },
});

beforeEach(() => {
  mockState.isOpen = true;
  mockState.captures = [];
  mockState.selectedIds = new Set();
  mockState.loading = false;
  mockState.error = null;
  mockState.summary = { count: 0, totalBytes: 0n };
  vi.clearAllMocks();
});

describe("CapturesDrawer", () => {
  it("shows empty state when no captures", () => {
    render(<CapturesDrawer />);
    expect(screen.getByText("No captures yet")).toBeInTheDocument();
  });

  it("renders capture cards", () => {
    mockState.captures = [sampleCapture("cap-1"), sampleCapture("cap-2")];
    mockState.summary = { count: 2, totalBytes: 2048n };
    render(<CapturesDrawer />);
    const sizeLabels = screen.getAllByText("1 KB");
    expect(sizeLabels.length).toBe(2);
  });

  it("shows loading state", () => {
    mockState.loading = true;
    const { container } = render(<CapturesDrawer />);
    expect(container.querySelector(".animate-spin")).toBeTruthy();
  });

  it("shows error state", () => {
    mockState.error = "Network error";
    render(<CapturesDrawer />);
    expect(screen.getByText("Network error")).toBeInTheDocument();
  });

  it("calls delete on individual capture", () => {
    mockState.captures = [sampleCapture("cap-1")];
    render(<CapturesDrawer />);
    // The delete button has a "Delete" title attribute
    const deleteBtn = screen.getByTitle("Delete");
    fireEvent.click(deleteBtn);
    expect(mockState.deleteCapture).toHaveBeenCalledWith("cap-1");
  });

  it("shows download button with count when items selected", () => {
    mockState.captures = [sampleCapture("cap-1"), sampleCapture("cap-2")];
    mockState.selectedIds = new Set(["cap-1", "cap-2"]);
    render(<CapturesDrawer />);
    expect(screen.getByText("Download (2)")).toBeInTheDocument();
  });

  it("calls deleteAll with confirmation", () => {
    mockState.captures = [sampleCapture("cap-1")];
    window.confirm = vi.fn(() => true);
    render(<CapturesDrawer />);
    const cleanBtn = screen.getByText("Clean Up All");
    fireEvent.click(cleanBtn);
    expect(window.confirm).toHaveBeenCalled();
    expect(mockState.deleteAll).toHaveBeenCalled();
  });

  it("does not call deleteAll when confirmation cancelled", () => {
    mockState.captures = [sampleCapture("cap-1")];
    window.confirm = vi.fn(() => false);
    render(<CapturesDrawer />);
    const cleanBtn = screen.getByText("Clean Up All");
    fireEvent.click(cleanBtn);
    expect(mockState.deleteAll).not.toHaveBeenCalled();
  });

  it("renders video element for recordings", () => {
    mockState.captures = [sampleCapture("cap-1", "recording")];
    const { container } = render(<CapturesDrawer />);
    expect(container.querySelector("video")).toBeTruthy();
  });

  it("renders img element for screenshots", () => {
    mockState.captures = [sampleCapture("cap-1", "screenshot")];
    const { container } = render(<CapturesDrawer />);
    expect(container.querySelector("img")).toBeTruthy();
  });

  it("does not render when closed", () => {
    mockState.isOpen = false;
    const { container } = render(<CapturesDrawer />);
    expect(container.innerHTML).toBe("");
  });
});
