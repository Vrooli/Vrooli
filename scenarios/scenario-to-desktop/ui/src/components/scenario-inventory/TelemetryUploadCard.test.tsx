import { fireEvent, render, screen, waitFor } from "@/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { TelemetryUploadCard } from "./TelemetryUploadCard";

const mocks = vi.hoisted(() => ({
  uploadTelemetry: vi.fn(),
  deleteTelemetry: vi.fn(),
  fetchTelemetryInsights: vi.fn(),
  fetchTelemetrySummary: vi.fn(),
  fetchTelemetryTail: vi.fn(),
  getTelemetryDownloadUrl: vi.fn(),
  readFileAsText: vi.fn(),
  writeToClipboard: vi.fn(),
}));

vi.mock("../../lib/api", () => ({
  uploadTelemetry: mocks.uploadTelemetry,
  deleteTelemetry: mocks.deleteTelemetry,
  fetchTelemetryInsights: mocks.fetchTelemetryInsights,
  fetchTelemetrySummary: mocks.fetchTelemetrySummary,
  fetchTelemetryTail: mocks.fetchTelemetryTail,
  getTelemetryDownloadUrl: mocks.getTelemetryDownloadUrl,
}));

vi.mock("../../lib/browser", () => ({
  readFileAsText: mocks.readFileAsText,
  writeToClipboard: mocks.writeToClipboard,
}));

const emptyInsights = { scenario_name: "desktop-demo", exists: false };
const emptySummary = { scenario_name: "desktop-demo", exists: false };

function renderCard() {
  return render(
    <TelemetryUploadCard
      scenarioName="desktop-demo"
      appDisplayName="Desktop Demo"
    />,
  );
}

async function expandCard() {
  fireEvent.click(
    screen.getByRole("button", { name: /uploading telemetry/i }),
  );
  await screen.findByText("Upload telemetry");
}

describe("TelemetryUploadCard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.fetchTelemetryInsights.mockResolvedValue(emptyInsights);
    mocks.fetchTelemetrySummary.mockResolvedValue(emptySummary);
    mocks.fetchTelemetryTail.mockResolvedValue({
      scenario_name: "desktop-demo",
      exists: false,
      limit: 200,
      entries: [],
    });
    mocks.getTelemetryDownloadUrl.mockReturnValue(
      "/api/v1/telemetry/desktop-demo/download",
    );
    mocks.readFileAsText.mockResolvedValue({ success: true, content: "" });
    mocks.writeToClipboard.mockResolvedValue({ success: true });
    vi.spyOn(window, "confirm").mockReturnValue(true);
  });

  it("expands only on request and loads the telemetry summaries for its scenario", async () => {
    renderCard();

    expect(screen.queryByText("Upload telemetry")).not.toBeInTheDocument();
    await expandCard();

    await waitFor(() => {
      expect(mocks.fetchTelemetryInsights).toHaveBeenCalledWith("desktop-demo");
      expect(mocks.fetchTelemetrySummary).toHaveBeenCalledWith("desktop-demo");
    });
    expect(screen.getByText("No telemetry insights yet.")).toBeInTheDocument();
    expect(screen.getByText("No uploaded telemetry yet.")).toBeInTheDocument();
  });

  it("shows insight, summary, and download details for uploaded telemetry", async () => {
    mocks.fetchTelemetryInsights.mockResolvedValue({
      scenario_name: "desktop-demo",
      exists: true,
      last_session: { session_id: "session-1", status: "passed" },
      last_error: { event: "startup_error", timestamp: "invalid-date", message: "No display" },
    });
    mocks.fetchTelemetrySummary.mockResolvedValue({
      scenario_name: "desktop-demo",
      exists: true,
      event_count: 3,
      file_size_bytes: 2048,
      file_path: "/tmp/telemetry.jsonl",
    });
    renderCard();
    await expandCard();

    expect(await screen.findByText("Session ID:")).toBeInTheDocument();
    expect(screen.getByText(/Last error: startup_error - No display/)).toBeInTheDocument();
    expect(screen.getByText("2 KB")).toBeInTheDocument();
    expect(screen.getByText("/tmp/telemetry.jsonl")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Download JSONL" })).toHaveAttribute(
      "href",
      "/api/v1/telemetry/desktop-demo/download",
    );
  });

  it("uploads a valid selected telemetry file and refreshes the summary", async () => {
    mocks.readFileAsText.mockResolvedValue({
      success: true,
      content: '{"event":"startup_complete"}',
    });
    mocks.uploadTelemetry.mockResolvedValue({ output_path: "/tmp/uploaded.jsonl" });
    renderCard();
    await expandCard();

    const file = new File(['{"event":"startup_complete"}'], "telemetry.jsonl", {
      type: "application/jsonl",
    });
    fireEvent.change(screen.getByLabelText("Telemetry file"), {
      target: { files: [file] },
    });
    fireEvent.click(screen.getByRole("button", { name: "Upload" }));

    await waitFor(() => {
      expect(mocks.uploadTelemetry).toHaveBeenCalledWith({
        scenario_name: "desktop-demo",
        events: [expect.objectContaining({ event: "startup_complete" })],
      });
    });
    expect(await screen.findByText("Telemetry uploaded successfully")).toBeInTheDocument();
  });

  it("surfaces file read failures without calling the upload endpoint", async () => {
    mocks.readFileAsText.mockResolvedValue({ success: false, error: "Unreadable file" });
    renderCard();
    await expandCard();

    const file = new File(["bad"], "telemetry.jsonl");
    fireEvent.change(screen.getByLabelText("Telemetry file"), {
      target: { files: [file] },
    });
    fireEvent.click(screen.getByRole("button", { name: "Upload" }));

    expect(await screen.findByText("Unreadable file")).toBeInTheDocument();
    expect(mocks.uploadTelemetry).not.toHaveBeenCalled();
  });

  it("confirms deletion and clears the tail view after a successful delete", async () => {
    mocks.fetchTelemetrySummary.mockResolvedValue({
      scenario_name: "desktop-demo",
      exists: true,
      event_count: 1,
    });
    mocks.deleteTelemetry.mockResolvedValue(undefined);
    renderCard();
    await expandCard();

    await screen.findByRole("button", { name: /view last 200 events/i });
    fireEvent.click(screen.getByRole("button", { name: /view last 200 events/i }));
    await waitFor(() => {
      expect(mocks.fetchTelemetryTail).toHaveBeenCalled();
    });
    fireEvent.click(screen.getByRole("button", { name: "Delete telemetry" }));

    await waitFor(() => {
      expect(mocks.deleteTelemetry).toHaveBeenCalledWith("desktop-demo");
    });
    expect(await screen.findByText("Telemetry deleted.")).toBeInTheDocument();
    expect(screen.queryByText(/showing last 200 events/i)).not.toBeInTheDocument();
  });

  it("filters the tail to error events and exposes malformed entries", async () => {
    mocks.fetchTelemetrySummary.mockResolvedValue({
      scenario_name: "desktop-demo",
      exists: true,
      event_count: 3,
    });
    mocks.fetchTelemetryTail.mockResolvedValue({
      scenario_name: "desktop-demo",
      exists: true,
      limit: 200,
      total_lines: 3,
      entries: [
        { raw: "ok", event: { event: "startup_complete", level: "info" } },
        { raw: "error", event: { event: "runtime_error", level: "error" } },
        { raw: "malformed", error: "invalid JSON" },
      ],
    });
    renderCard();
    await expandCard();

    await screen.findByRole("button", { name: /view last 200 events/i });
    fireEvent.click(screen.getByRole("button", { name: /view last 200 events/i }));
    expect(await screen.findByText("Unparsed telemetry line")).toBeInTheDocument();
    expect(screen.getByText("Parse error: invalid JSON")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Show errors only" }));

    expect(screen.queryByText("Unparsed telemetry line")).not.toBeInTheDocument();
  });

  it("shows platform file paths and copies the selected path", async () => {
    renderCard();
    await expandCard();

    fireEvent.click(screen.getByRole("button", { name: "Where is the file?" }));
    const copyButtons = screen.getAllByRole("button", {
      name: /Copy .* telemetry path/,
    });
    expect(copyButtons.length).toBeGreaterThan(0);
    const firstCopyButton = copyButtons[0];
    if (!firstCopyButton) {
      throw new Error("expected a telemetry path copy button");
    }
    fireEvent.click(firstCopyButton);

    await waitFor(() => {
      expect(mocks.writeToClipboard).toHaveBeenCalledTimes(1);
    });
    expect(
      screen.getByRole("button", { name: "Copy Windows telemetry path" }),
    ).toBeInTheDocument();
  });
});
