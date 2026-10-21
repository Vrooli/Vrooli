import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { i18n, setLocale } from "../../i18n";
import { strings } from "../../consts/strings";
import { MediaReadiness, SourceMediaKind } from "@vrooli/proto-types/device-studio/v1/projects/projects_pb";
import { projectsClient, uploadSourceMedia, getSourceMediaContent } from "../../api/projects";
import { SourceMediaPanel } from "./SourceMediaPanel";

vi.mock("../../api/projects", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/projects")>();
  return {
    ...actual,
    projectsClient: { listProjects: vi.fn(), createProject: vi.fn(), getProject: vi.fn(), renameProject: vi.fn(), listSourceMedia: vi.fn(), getSourceMedia: vi.fn(), ensureSourceMediaInspection: vi.fn(), retrySourceMediaInspection: vi.fn(), cancelSourceMediaInspection: vi.fn() },
    uploadSourceMedia: vi.fn(),
    getSourceMediaContent: vi.fn(),
  };
});

const record = {
  id: "m1", projectId: "p1", sha256: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
  sizeBytes: 3n, mediaType: "image/png", width: 1, height: 1, originalFilename: "capture.png",
  kind: SourceMediaKind.IMAGE, readiness: MediaReadiness.READY, inspectionIssueCode: "", inspectionIssueMessage: "",
  orientation: "identity", colorModel: "RGB", colorProfileStatus: "unspecified", importMethod: "local_upload", decoder: "png",
};

describe("SourceMediaPanel", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.mocked(projectsClient.listSourceMedia).mockReset();
    vi.mocked(projectsClient.getSourceMedia).mockReset();
    vi.mocked(projectsClient.ensureSourceMediaInspection).mockReset();
    vi.mocked(projectsClient.retrySourceMediaInspection).mockReset();
    vi.mocked(projectsClient.cancelSourceMediaInspection).mockReset();
    vi.mocked(uploadSourceMedia).mockReset();
    vi.mocked(getSourceMediaContent).mockReset();
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [] } as never);
  });
  afterEach(() => { cleanup(); vi.clearAllMocks(); vi.unstubAllGlobals(); });

  it("keeps the chosen file available for retry after an import failure", async () => {
    const user = userEvent.setup();
    vi.mocked(uploadSourceMedia).mockRejectedValueOnce(new Error("unsupported image profile")).mockRejectedValueOnce(new Error("unsupported image profile"));
    renderWithProviders(<SourceMediaPanel projectId="p1" />);
    const file = new File(["abc"], "capture.png", { type: "image/png" });
    await user.upload(screen.getByTestId(selectors.media.fileInput), file);
    await user.click(screen.getByTestId(selectors.media.uploadButton));
    expect(await screen.findByTestId(selectors.media.uploadError)).toHaveTextContent("unsupported image profile");
    expect(screen.getByTestId(selectors.media.fileInput)).toHaveValue("C:\\fakepath\\capture.png");
    await user.click(screen.getByRole("button", { name: "Retry" }));
    await waitFor(() => expect(uploadSourceMedia).toHaveBeenCalledTimes(2));
    expect(uploadSourceMedia).toHaveBeenLastCalledWith("p1", file);
  });

  it("locks the picker while an import is pending and selects the returned saved record", async () => {
    const user = userEvent.setup();
    let finish!: (value: typeof record) => void;
    vi.mocked(uploadSourceMedia).mockReturnValueOnce(new Promise((resolve) => { finish = resolve; }) as never);
    vi.mocked(projectsClient.getSourceMedia).mockResolvedValue({ sourceMedia: record } as never);
    vi.mocked(getSourceMediaContent).mockResolvedValue(new Blob(["abc"], { type: "image/png" }));
    const testURL = class extends URL { static createObjectURL = vi.fn(() => "blob:returned-source"); static revokeObjectURL = vi.fn(); };
    vi.stubGlobal("URL", testURL);
    renderWithProviders(<SourceMediaPanel projectId="p1" />);
    const input = screen.getByTestId(selectors.media.fileInput);
    await user.upload(input, new File(["abc"], "capture.png", { type: "image/png" }));
    await user.click(screen.getByTestId(selectors.media.uploadButton));
    expect(input).toBeDisabled();
    finish(record);
    expect(await screen.findByTestId(selectors.media.image)).toHaveAttribute("alt", "Imported source image: capture.png");
    expect(await screen.findByRole("status")).toHaveTextContent("Imported capture.png and selected it as source media.");
    expect(input).not.toBeDisabled();
    expect(input).toHaveValue("");
    expect(projectsClient.getSourceMedia).toHaveBeenCalledWith({ projectId: "p1", id: record.id });
  });

  it("lists, selects and shows only content whose saved digest matches", async () => {
    const user = userEvent.setup();
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [record], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMedia).mockResolvedValue({ sourceMedia: record } as never);
    vi.mocked(getSourceMediaContent).mockResolvedValue(new Blob(["abc"], { type: "image/png" }));
    const testURL = class extends URL { static createObjectURL = vi.fn(() => "blob:test-source"); static revokeObjectURL = vi.fn(); };
    vi.stubGlobal("URL", testURL);
    renderWithProviders(<SourceMediaPanel projectId="p1" />);
    await user.click(await screen.findByTestId(selectors.media.selectButton({ id: "m1" })));
    expect(await screen.findByTestId(selectors.media.image)).toHaveAttribute("alt", "Imported source image: capture.png");
    await user.click(screen.getByText(i18n.t(strings.media.sourceDetails)));
    expect(document.querySelector("details p")).toHaveTextContent(`${i18n.t(strings.media.sha256)}: ${record.sha256}`);
    expect(screen.getByText(i18n.t(strings.media.sourcePreviewDescription))).toBeInTheDocument();
  });

  it("withholds the preview when the verified content endpoint rejects the bytes", async () => {
    const user = userEvent.setup();
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [record], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMedia).mockResolvedValue({ sourceMedia: record } as never);
    vi.mocked(getSourceMediaContent).mockRejectedValue(new Error("source digest verification failed"));
    renderWithProviders(<SourceMediaPanel projectId="p1" />);
    await user.click(await screen.findByTestId(selectors.media.selectButton({ id: "m1" })));
    expect(await screen.findByRole("alert")).toHaveTextContent("source digest verification failed");
    expect(screen.queryByTestId(selectors.media.image)).not.toBeInTheDocument();
  });

  it("lets a user retry metadata lookup for the selected image", async () => {
    const user = userEvent.setup();
    const selected = { ...record, id: "meta-retry", originalFilename: "metadata.png" };
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [selected], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMedia)
      .mockRejectedValueOnce(new Error("metadata temporarily unavailable"))
      .mockResolvedValueOnce({ sourceMedia: selected } as never);
    vi.mocked(getSourceMediaContent).mockResolvedValue(new Blob(["abc"], { type: "image/png" }));
    const testURL = class extends URL { static createObjectURL = vi.fn(() => "blob:retried-source"); static revokeObjectURL = vi.fn(); };
    vi.stubGlobal("URL", testURL);
    renderWithProviders(<SourceMediaPanel projectId="p1" />);
    await user.click(await screen.findByTestId(selectors.media.selectButton({ id: selected.id })));
    const failure = await screen.findByRole("alert");
    expect(failure).toHaveTextContent("metadata temporarily unavailable");
    await user.click(within(failure).getByRole("button", { name: "Retry" }));
    expect(await screen.findByTestId(selectors.media.image)).toBeInTheDocument();
  });

  it("retries verified content after a transient source read failure", async () => {
    const user = userEvent.setup();
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [record], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMedia).mockResolvedValue({ sourceMedia: record } as never);
    vi.mocked(getSourceMediaContent).mockRejectedValueOnce(new Error("temporary read failure")).mockResolvedValueOnce(new Blob(["abc"], { type: "image/png" }));
    const testURL = class extends URL { static createObjectURL = vi.fn(() => "blob:retried-content"); static revokeObjectURL = vi.fn(); };
    vi.stubGlobal("URL", testURL);
    renderWithProviders(<SourceMediaPanel projectId="p1" />);
    await user.click(await screen.findByTestId(selectors.media.selectButton({ id: "m1" })));
    const failure = await screen.findByRole("alert");
    expect(failure).toHaveTextContent("temporary read failure");
    await user.click(within(failure).getByRole("button", { name: "Retry" }));
    expect(await screen.findByTestId(selectors.media.image)).toHaveAttribute("src", "blob:retried-content");
    expect(getSourceMediaContent).toHaveBeenCalledTimes(2);
  });

  it("recovers a saved-media list and fetches the requested next page", async () => {
    const user = userEvent.setup();
    const fetchPage = vi.mocked(projectsClient.listSourceMedia);
    fetchPage.mockRejectedValueOnce(new Error("list unavailable"))
      .mockResolvedValueOnce({ sourceMedia: [record], nextPageToken: "page-2" } as never)
      .mockResolvedValueOnce({ sourceMedia: [{ ...record, id: "m2", originalFilename: "second.png" }], nextPageToken: "" } as never);
    renderWithProviders(<SourceMediaPanel projectId="p1" />);
    const failure = await screen.findByRole("alert");
    await user.click(within(failure).getByRole("button", { name: "Retry" }));
    expect(await screen.findByTestId(selectors.media.selectButton({ id: "m1" }))).toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.media.loadMore));
    expect(await screen.findByTestId(selectors.media.selectButton({ id: "m2" }))).toBeInTheDocument();
    expect(fetchPage).toHaveBeenLastCalledWith(expect.objectContaining({ projectId: "p1", pageToken: "page-2" }));
  });

  it("streams video from its routed URL without reading it into a JavaScript Blob", async () => {
    const user = userEvent.setup();
    const video = { ...record, id: "v1", kind: SourceMediaKind.VIDEO, readiness: MediaReadiness.PENDING, mediaType: "video/mp4", width: undefined, height: undefined, inspectionJobId: "job-1", inspectionAttemptId: "attempt-1", inspectionJob: { id: "job-1", state: "running", version: 4n, currentAttemptId: "attempt-1" } };
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [video], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMedia).mockResolvedValue({ sourceMedia: video } as never);
    const testURL = class extends URL { static createObjectURL = vi.fn(() => "blob:unexpected"); static revokeObjectURL = vi.fn(); };
    vi.stubGlobal("URL", testURL);
    renderWithProviders(<SourceMediaPanel projectId="p1" />);
    await user.click(await screen.findByTestId(selectors.media.selectButton({ id: "v1" })));
    const player = await screen.findByTestId(selectors.media.videoPreview);
    expect(player).toHaveAttribute("src", expect.stringMatching(/\/api\/v1\/projects\/p1\/media\/v1\/content$/));
    expect(player).toHaveProperty("muted", true);
    expect(player).toHaveAttribute("aria-label", "Original recording: capture.png");
    expect(getSourceMediaContent).not.toHaveBeenCalled();
    expect(screen.getByText((_, element) => element?.textContent === `${i18n.t(strings.media.inspectionStatus)}: ${i18n.t(strings.media.readinessPending)}`)).toBeInTheDocument();
  });

  it("cancels a current inspection using its expected version and keeps an uncertain response visible", async () => {
    const user = userEvent.setup();
    const video = { ...record, id: "v1", kind: SourceMediaKind.VIDEO, readiness: MediaReadiness.PENDING, mediaType: "video/mp4", inspectionJobId: "job-1", inspectionAttemptId: "attempt-1", inspectionJob: { id: "job-1", state: "running", version: 7n, currentAttemptId: "attempt-1" } };
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [video], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMedia).mockResolvedValue({ sourceMedia: video } as never);
    vi.mocked(projectsClient.cancelSourceMediaInspection).mockResolvedValue({ sourceMedia: { ...video, inspectionJob: { ...video.inspectionJob, state: "cancel_requested", version: 8n } } } as never);
    renderWithProviders(<SourceMediaPanel projectId="p1" />);
    await user.click(await screen.findByTestId(selectors.media.selectButton({ id: "v1" })));
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.media.cancelInspection) }));
    await waitFor(() => expect(projectsClient.cancelSourceMediaInspection).toHaveBeenCalledWith({ projectId: "p1", mediaId: "v1", expectedJobVersion: 7n }));
  });

  it("offers explicit retry only for settled execution failures and preserves a stale-version conflict", async () => {
    const user = userEvent.setup();
    const video = { ...record, id: "v1", kind: SourceMediaKind.VIDEO, readiness: MediaReadiness.REJECTED, mediaType: "video/mp4", inspectionJobId: "job-1", inspectionAttemptId: "attempt-1", inspectionIssueCode: "inspection_execution_failed", inspectionIssueMessage: "Inspection failed.", inspectionJob: { id: "job-1", state: "failed", version: 9n, currentAttemptId: "attempt-1" } };
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [video], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMedia).mockResolvedValue({ sourceMedia: video } as never);
    vi.mocked(projectsClient.retrySourceMediaInspection).mockRejectedValueOnce(new Error("revision conflict: expected 9, current 10"));
    renderWithProviders(<SourceMediaPanel projectId="p1" />);
    await user.click(await screen.findByTestId(selectors.media.selectButton({ id: "v1" })));
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.media.retryInspection) }));
    await waitFor(() => expect(projectsClient.retrySourceMediaInspection).toHaveBeenCalledWith({ projectId: "p1", mediaId: "v1", expectedJobVersion: 9n }));
    expect(await screen.findByRole("alert")).toHaveTextContent("revision conflict: expected 9, current 10");
    expect(projectsClient.retrySourceMediaInspection).toHaveBeenCalledTimes(1);
  });

  it("refreshes a stale inspection version after conflict and retries only by explicit action", async () => {
    const user = userEvent.setup();
    const failed = { ...record, id: "v1", kind: SourceMediaKind.VIDEO, readiness: MediaReadiness.REJECTED, mediaType: "video/mp4", inspectionIssueCode: "inspection_execution_failed", inspectionJob: { id: "job-1", state: "failed", version: 7n, currentAttemptId: "attempt-1" } };
    const refreshed = { ...failed, inspectionJob: { ...failed.inspectionJob, version: 8n } };
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [failed], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMedia).mockResolvedValueOnce({ sourceMedia: failed } as never).mockResolvedValue({ sourceMedia: refreshed } as never);
    vi.mocked(projectsClient.retrySourceMediaInspection).mockRejectedValueOnce(new Error("revision conflict: expected 7, current 8")).mockResolvedValueOnce({ sourceMedia: refreshed } as never);
    renderWithProviders(<SourceMediaPanel projectId="p1" />);
    await user.click(await screen.findByTestId(selectors.media.selectButton({ id: "v1" })));
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.media.retryInspection) }));
    expect(await screen.findByRole("alert")).toHaveTextContent("revision conflict: expected 7, current 8");
    await waitFor(() => expect(projectsClient.getSourceMedia).toHaveBeenCalledTimes(2));
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.media.retryInspection) }));
    await waitFor(() => expect(projectsClient.retrySourceMediaInspection).toHaveBeenCalledTimes(2));
    expect(vi.mocked(projectsClient.retrySourceMediaInspection).mock.calls.map(([request]) => request.expectedJobVersion)).toEqual([7n, 8n]);
  });

  it("does not show a pending mutation failure on a newly selected recording", async () => {
    const user = userEvent.setup();
    const first = { ...record, id: "v1", originalFilename: "first.mp4", kind: SourceMediaKind.VIDEO, readiness: MediaReadiness.REJECTED, mediaType: "video/mp4", inspectionIssueCode: "inspection_execution_failed", inspectionJob: { id: "job-1", state: "failed", version: 4n, currentAttemptId: "attempt-1" } };
    const second = { ...first, id: "v2", originalFilename: "second.mp4", inspectionJob: { ...first.inspectionJob, id: "job-2", version: 11n, currentAttemptId: "attempt-2" } };
    let rejectRetry!: (error: Error) => void;
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [first, second], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMedia).mockImplementation(({ id }) => Promise.resolve({ sourceMedia: id === first.id ? first : second }) as never);
    vi.mocked(projectsClient.retrySourceMediaInspection).mockReturnValueOnce(new Promise((_, reject) => { rejectRetry = reject; }) as never);
    renderWithProviders(<SourceMediaPanel projectId="p1" />);
    await user.click(await screen.findByTestId(selectors.media.selectButton({ id: first.id })));
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.media.retryInspection) }));
    await user.click(await screen.findByTestId(selectors.media.selectButton({ id: second.id })));
    rejectRetry(new Error("old media conflict"));
    await waitFor(() => expect(projectsClient.retrySourceMediaInspection).toHaveBeenCalledWith({ projectId: "p1", mediaId: first.id, expectedJobVersion: 4n }));
    await waitFor(() => expect(projectsClient.getSourceMedia).toHaveBeenCalledWith({ projectId: "p1", id: second.id }));
    expect(screen.queryAllByRole("alert")).toHaveLength(0);
    expect(screen.getByTestId(selectors.media.selectButton({ id: second.id }))).toHaveAttribute("aria-pressed", "true");
  });

  it("offers idempotent ensure for a pending inspection with an activation-pending job", async () => {
    const user = userEvent.setup();
    const video = { ...record, id: "v1", kind: SourceMediaKind.VIDEO, readiness: MediaReadiness.PENDING, mediaType: "video/mp4", inspectionJobId: "job-1", inspectionAttemptId: "attempt-1", inspectionJob: { id: "job-1", state: "activation_pending", version: 3n, currentAttemptId: "attempt-1" } };
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [video], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMedia).mockResolvedValue({ sourceMedia: video } as never);
    vi.mocked(projectsClient.ensureSourceMediaInspection).mockResolvedValue({ sourceMedia: video } as never);
    renderWithProviders(<SourceMediaPanel projectId="p1" />);
    await user.click(await screen.findByTestId(selectors.media.selectButton({ id: "v1" })));
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.media.checkInspection) }));
    expect(projectsClient.ensureSourceMediaInspection).toHaveBeenCalledWith({ projectId: "p1", mediaId: "v1" });
    expect(projectsClient.retrySourceMediaInspection).not.toHaveBeenCalled();
  });

  it("refreshes a terminal job until its pending media projection becomes ready", async () => {
    const user = userEvent.setup();
    const terminalPending = { ...record, id: "v1", kind: SourceMediaKind.VIDEO, readiness: MediaReadiness.PENDING, mediaType: "video/mp4", inspectionJobId: "job-1", inspectionJob: { id: "job-1", state: "succeeded", version: 6n, currentAttemptId: "attempt-1" } };
    const ready = { ...terminalPending, readiness: MediaReadiness.READY };
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [terminalPending], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMedia).mockResolvedValueOnce({ sourceMedia: terminalPending } as never).mockResolvedValue({ sourceMedia: ready } as never);
    const view = renderWithProviders(<SourceMediaPanel projectId="p1" />);
    await user.click(await screen.findByTestId(selectors.media.selectButton({ id: "v1" })));
    await waitFor(() => expect(projectsClient.getSourceMedia).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(projectsClient.getSourceMedia).toHaveBeenCalledTimes(2), { timeout: 3000, interval: 50 });
    expect(await screen.findByText((_, element) => element?.textContent === [i18n.t(strings.media.inspectionStatus), i18n.t(strings.media.readinessReady)].join(": "))).toBeInTheDocument();
    expect(projectsClient.ensureSourceMediaInspection).not.toHaveBeenCalled();
    view.unmount();
  });
});
