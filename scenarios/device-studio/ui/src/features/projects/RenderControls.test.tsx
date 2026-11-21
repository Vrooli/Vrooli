import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useState, type ComponentProps } from "react";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import { PhysicalMotionOutputBindingSchema, PhysicalRenderBackend, PhysicalShotBindingSchema, PresentationKind, ReadabilityProofSchema, ReadabilityVerdict, RenderJobSchema, RenderJobStatus, RenderMode, RenderOutputProfile, RenderOutputSchema, ScreenPresentationSchema, ShotRevisionSchema, ShotSnapshotSchema, VideoTimingSchema } from "@vrooli/proto-types/device-studio/v1/shots/shots_pb";
import { renderWithProviders } from "../../test-utils";
import { i18n } from "../../i18n";
import { strings } from "../../consts/strings";
import { RenderControls } from "./RenderControls";

vi.mock("../../api/renders", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/renders")>();
  return {
    ...actual,
    rendersClient: { listRenders: vi.fn(), submitRender: vi.fn(), cancelRender: vi.fn(), getRender: vi.fn() },
    getRenderOutputContent: vi.fn(),
    verifyRenderOutputContent: vi.fn(),
  };
});
vi.mock("../../api/shots", () => ({ shotsClient: { getOutputReadabilityReview: vi.fn(), reviewOutputReadability: vi.fn() } }));

import { getRenderOutputContent, renderOutputContentURL, rendersClient, verifyRenderOutputContent } from "../../api/renders";
import { shotsClient } from "../../api/shots";

function job(id: string, revision: bigint, status: RenderJobStatus, overrides: Record<string, unknown> = {}) {
  return create(RenderJobSchema, { id, projectId: "project-1", shotId: "shot-1", shotRevision: revision, status, version: 4n, summary: "", ...overrides });
}

const savedRevision = (revision = 7n) => create(ShotRevisionSchema, { shotId: "shot-1", revision, name: "Daylight", renderEligible: true, issues: [] });
const videoRevision = (revision = 7n) => create(ShotRevisionSchema, {
  ...savedRevision(revision),
  snapshot: create(ShotSnapshotSchema, {
    presentation: create(ScreenPresentationSchema, { kind: PresentationKind.PLAIN }),
    videoTiming: create(VideoTimingSchema, { sourceMediaId: "video-source", sourceSha256: "a".repeat(64), sourceSizeBytes: 50n, clockSha256: "b".repeat(64), clockSizeBytes: 25n, timeBaseNum: 1n, timeBaseDen: 24n, streamStartPts: 0n, sourceEndPts: 48n, trimStartPts: 0n, trimEndPts: 48n, lastFrameDurationTicks: 1n, playbackRateNum: 1, playbackRateDen: 1, outputRateNum: 24, outputRateDen: 1 }),
  }),
});
const physicalMotionRevision = (revision = 7n) => create(ShotRevisionSchema, {
  ...savedRevision(revision),
  snapshot: create(ShotSnapshotSchema, {
    physicalBinding: create(PhysicalShotBindingSchema, {
      backend: PhysicalRenderBackend.OPTIX,
      motionOutput: create(PhysicalMotionOutputBindingSchema, { profileId: "device-studio-physical-motion-1080p-v1", profileVersion: 1, width: 1920, height: 1080, outputRateNumerator: 24, outputRateDenominator: 1, frameCount: 48 }),
    }),
  }),
});
const physicalStillRevision = (revision = 7n) => create(ShotRevisionSchema, {
  ...savedRevision(revision),
  snapshot: create(ShotSnapshotSchema, { physicalBinding: create(PhysicalShotBindingSchema, { backend: PhysicalRenderBackend.OPTIX }) }),
});
type HarnessProps = Omit<ComponentProps<typeof RenderControls>, "showOutput" | "onShowOutputChange">;
function RenderControlsHarness(props: HarnessProps) {
  const [showOutput, setShowOutput] = useState(false);
  return <RenderControls {...props} showOutput={showOutput} onShowOutputChange={setShowOutput} />;
}

describe("saved shot rendering controls", () => {
  beforeEach(() => {
    sessionStorage.clear();
    vi.mocked(rendersClient.listRenders).mockResolvedValue({ jobs: [], nextPageToken: "" } as never);
    vi.mocked(rendersClient.cancelRender).mockReset();
    vi.mocked(rendersClient.submitRender).mockReset();
    vi.mocked(shotsClient.getOutputReadabilityReview).mockRejectedValue(new ConnectError("not found", Code.NotFound));
    vi.mocked(shotsClient.reviewOutputReadability).mockReset();
  });
  afterEach(() => { cleanup(); vi.clearAllMocks(); vi.unstubAllGlobals(); });

  it("submits only the saved revision and reuses an uncertain request identity", async () => {
    const user = userEvent.setup();
    vi.mocked(rendersClient.submitRender)
      .mockRejectedValueOnce(new Error("connection interrupted"))
      .mockResolvedValueOnce({ job: job("render-1", 7n, RenderJobStatus.QUEUED), reusedExisting: true } as never);
    renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={savedRevision()} historical={false} hasUnsavedChanges />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.shot.renderRevisionAction, { revision: "7" }) }));
    expect(await screen.findByRole("button", { name: i18n.t(strings.shot.renderRetrySubmit) })).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.shot.renderUnsaved, { revision: "7" }))).toBeInTheDocument();
    const first = vi.mocked(rendersClient.submitRender).mock.calls[0]?.[0];
    expect(first).toMatchObject({ projectId: "project-1", shotId: "shot-1", revision: 7n });
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.renderRetrySubmit) }));
    await screen.findByText(i18n.t(strings.shot.renderQueued));
    expect(vi.mocked(rendersClient.submitRender).mock.calls[1]?.[0].idempotencyKey).toBe(first?.idempotencyKey);
  });

  it("requires a viewed exact-clock sample before the manual video final and pins each mode to its output profile", async () => {
    const user = userEvent.setup();
    const sampleOutput = create(RenderOutputSchema, { mediaType: "image/png", sizeBytes: 3n, widthPixels: 16, heightPixels: 9, sha256: "a".repeat(64), contentUrl: "/api/v1/projects/project-1/renders/sample/content" });
    const sample = job("sample", 7n, RenderJobStatus.SUCCEEDED, { renderMode: RenderMode.VIDEO_SAMPLE_PNG, output: sampleOutput });
    vi.mocked(rendersClient.listRenders).mockResolvedValue({ jobs: [sample], nextPageToken: "" } as never);
    const blob = new Blob(["png"], { type: "image/png" });
    vi.mocked(getRenderOutputContent).mockResolvedValue(blob);
    vi.mocked(verifyRenderOutputContent).mockResolvedValue(blob);
    vi.stubGlobal("URL", { ...URL, createObjectURL: vi.fn(() => "blob:sample"), revokeObjectURL: vi.fn() });
    renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={videoRevision()} historical={false} hasUnsavedChanges={false} />);
    const finalButton = await screen.findByRole("button", { name: i18n.t(strings.shot.renderVideoFinal, { revision: "7" }) });
    expect(finalButton).toBeDisabled();
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.shot.renderViewSample) }));
    await waitFor(() => expect(finalButton).toBeEnabled());
    await user.click(finalButton);
    expect(rendersClient.submitRender).toHaveBeenCalledWith(expect.objectContaining({ renderMode: RenderMode.VIDEO_FINAL_MP4, outputProfile: RenderOutputProfile.MP4_H264_SDR_V1, revision: 7n }));
  });

  it("reopens a saved physical-motion binding as one explicit MP4 render action", async () => {
    const user = userEvent.setup();
    vi.mocked(rendersClient.submitRender).mockResolvedValue({ job: job("physical-motion", 7n, RenderJobStatus.QUEUED), reusedExisting: false } as never);
    renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={physicalMotionRevision()} historical={false} hasUnsavedChanges={false} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.renderMotion) }));
    expect(vi.mocked(rendersClient.submitRender)).toHaveBeenCalledWith(expect.objectContaining({
      projectId: "project-1", shotId: "shot-1", revision: 7n,
      renderMode: RenderMode.PHYSICAL_VIDEO_FINAL_MP4,
      outputProfile: RenderOutputProfile.MP4_H264_SDR_V1,
    }));
    expect(screen.queryByRole("button", { name: i18n.t(strings.shot.renderVideoSample) })).not.toBeInTheDocument();
  });

  it("offers the physical MP4 action without a PNG preview for recorded physical shots", async () => {
    const user = userEvent.setup();
    const revision = create(ShotRevisionSchema, {
      ...physicalMotionRevision(),
      snapshot: create(ShotSnapshotSchema, {
        physicalBinding: create(PhysicalShotBindingSchema, {
          backend: PhysicalRenderBackend.OPTIX,
          motionOutput: create(PhysicalMotionOutputBindingSchema, { profileId: "device-studio-physical-motion-1080p-v1", profileVersion: 1, width: 1920, height: 1080, outputRateNumerator: 24, outputRateDenominator: 1, frameCount: 24 }),
        }),
        videoTiming: create(VideoTimingSchema, { sourceMediaId: "video-source", sourceSha256: "a".repeat(64), sourceSizeBytes: 900n, clockSha256: "b".repeat(64), clockSizeBytes: 256n, timeBaseNum: 1n, timeBaseDen: 24n, streamStartPts: 0n, sourceEndPts: 24n, trimStartPts: 0n, trimEndPts: 24n, lastFrameDurationTicks: 1n, playbackRateNum: 1, playbackRateDen: 1, outputRateNum: 24, outputRateDen: 1 }),
      }),
    });
    vi.mocked(rendersClient.submitRender).mockResolvedValue({ job: job("physical-recording", 7n, RenderJobStatus.QUEUED), reusedExisting: false } as never);
    renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={revision} historical={false} hasUnsavedChanges={false} />);
    expect(screen.queryByRole("button", { name: i18n.t(strings.physical.renderPreview) })).not.toBeInTheDocument();
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.renderMotion) }));
    await waitFor(() => expect(rendersClient.submitRender).toHaveBeenCalledWith(expect.objectContaining({
      renderMode: RenderMode.PHYSICAL_VIDEO_FINAL_MP4,
      outputProfile: RenderOutputProfile.MP4_H264_SDR_V1,
    })));
  });

  it("submits a distinct 960×540 preview for physical still and motion shots", async () => {
    const user = userEvent.setup();
    expect(RenderMode.PHYSICAL_PREVIEW_PNG).toBe(6);
    expect(RenderOutputProfile.PHYSICAL_PREVIEW_PNG_V1).toBe(3);
    vi.mocked(rendersClient.submitRender).mockResolvedValue({ job: job("preview", 7n, RenderJobStatus.QUEUED, { renderMode: RenderMode.PHYSICAL_PREVIEW_PNG }), reusedExisting: false } as never);
    const { rerender } = renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={physicalStillRevision()} historical={false} hasUnsavedChanges={false} />);
    const previewLabel = i18n.t(strings.physical.renderPreview);
    await user.click(await screen.findByRole("button", { name: previewLabel }));
    expect(rendersClient.submitRender).toHaveBeenLastCalledWith(expect.objectContaining({
      renderMode: RenderMode.PHYSICAL_PREVIEW_PNG,
      outputProfile: RenderOutputProfile.PHYSICAL_PREVIEW_PNG_V1,
      revision: 7n,
    }));
    expect(screen.getByText(i18n.t(strings.physical.previewDraft))).toBeInTheDocument();
    vi.mocked(rendersClient.submitRender).mockClear();
    rerender(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={physicalMotionRevision()} historical={false} hasUnsavedChanges={false} />);
    await user.click(await screen.findByRole("button", { name: previewLabel }));
    expect(rendersClient.submitRender).toHaveBeenLastCalledWith(expect.objectContaining({
      renderMode: RenderMode.PHYSICAL_PREVIEW_PNG,
      outputProfile: RenderOutputProfile.PHYSICAL_PREVIEW_PNG_V1,
    }));
  });

  it("automatically displays a verified preview completed for the current saved revision", async () => {
    const user = userEvent.setup();
    const output = create(RenderOutputSchema, { mediaType: "image/png", sizeBytes: 3n, widthPixels: 960, heightPixels: 540, sha256: "e".repeat(64), contentUrl: "/api/v1/projects/project-1/renders/preview-ready/content" });
    const completed = job("preview-ready", 7n, RenderJobStatus.SUCCEEDED, { renderMode: RenderMode.PHYSICAL_PREVIEW_PNG, output });
    const blob = new Blob(["preview"], { type: "image/png" });
    vi.mocked(rendersClient.submitRender).mockResolvedValue({ job: completed, reusedExisting: false } as never);
    vi.mocked(getRenderOutputContent).mockResolvedValue(blob);
    vi.mocked(verifyRenderOutputContent).mockResolvedValue(blob);
    vi.stubGlobal("URL", { ...URL, createObjectURL: vi.fn(() => "blob:physical-preview"), revokeObjectURL: vi.fn() });
    renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={physicalStillRevision()} historical={false} hasUnsavedChanges={false} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.renderPreview) }));
    expect(await screen.findByRole("link", { name: i18n.t(strings.shot.renderDownload) })).toHaveAttribute("href", "blob:physical-preview");
    expect(verifyRenderOutputContent).toHaveBeenCalledWith(blob, output);
  });

  it.each([RenderJobStatus.FAILED, RenderJobStatus.SUCCEEDED])("does not display a preview without usable output metadata (status %s)", async (status) => {
    const user = userEvent.setup();
    vi.mocked(rendersClient.submitRender).mockResolvedValue({ job: job("preview-no-output", 7n, status, { renderMode: RenderMode.PHYSICAL_PREVIEW_PNG }), reusedExisting: false } as never);
    renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={physicalStillRevision()} historical={false} hasUnsavedChanges={false} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.renderPreview) }));
    await screen.findByText(i18n.t(status === RenderJobStatus.FAILED ? strings.shot.renderFailed : strings.shot.renderSucceeded));
    expect(screen.queryByRole("link", { name: i18n.t(strings.shot.renderDownload) })).not.toBeInTheDocument();
    expect(getRenderOutputContent).not.toHaveBeenCalled();
  });

  it("keeps an explicitly hidden preview hidden", async () => {
    const user = userEvent.setup();
    const output = create(RenderOutputSchema, { mediaType: "image/png", sizeBytes: 3n, widthPixels: 960, heightPixels: 540, sha256: "f".repeat(64), contentUrl: "/api/v1/projects/project-1/renders/preview-hidden/content" });
    const completed = job("preview-hidden", 7n, RenderJobStatus.SUCCEEDED, { renderMode: RenderMode.PHYSICAL_PREVIEW_PNG, output });
    const blob = new Blob(["preview"], { type: "image/png" });
    vi.mocked(rendersClient.submitRender).mockResolvedValue({ job: completed, reusedExisting: false } as never);
    vi.mocked(getRenderOutputContent).mockResolvedValue(blob);
    vi.mocked(verifyRenderOutputContent).mockResolvedValue(blob);
    vi.stubGlobal("URL", { ...URL, createObjectURL: vi.fn(() => "blob:physical-preview-hidden"), revokeObjectURL: vi.fn() });
    const view = renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={physicalStillRevision()} historical={false} hasUnsavedChanges={false} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.renderPreview) }));
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.shot.renderHideOutput) }));
    expect(screen.queryByRole("link", { name: i18n.t(strings.shot.renderDownload) })).not.toBeInTheDocument();
    view.rerender(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={physicalStillRevision()} historical={false} hasUnsavedChanges />);
    expect(screen.queryByRole("link", { name: i18n.t(strings.shot.renderDownload) })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: i18n.t(strings.shot.renderHideOutput) })).not.toBeInTheDocument();
  });

  it("does not auto-display a preview that completes after the shot becomes unsaved", async () => {
    const user = userEvent.setup();
    let finish: ((value: unknown) => void) | undefined;
    vi.mocked(rendersClient.submitRender).mockImplementation(() => new Promise((resolve) => { finish = resolve; }) as never);
    const view = renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={physicalStillRevision()} historical={false} hasUnsavedChanges={false} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.renderPreview) }));
    view.rerender(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={physicalStillRevision()} historical={false} hasUnsavedChanges />);
    const output = create(RenderOutputSchema, { mediaType: "image/png", sizeBytes: 3n, widthPixels: 960, heightPixels: 540, sha256: "1".repeat(64), contentUrl: "/api/v1/projects/project-1/renders/preview-stale/content" });
    finish?.({ job: job("preview-stale", 7n, RenderJobStatus.SUCCEEDED, { renderMode: RenderMode.PHYSICAL_PREVIEW_PNG, output }), reusedExisting: false });
    await waitFor(() => expect(rendersClient.submitRender).toHaveBeenCalledTimes(1));
    expect(screen.queryByRole("link", { name: i18n.t(strings.shot.renderDownload) })).not.toBeInTheDocument();
    expect(getRenderOutputContent).not.toHaveBeenCalled();
  });

  it("does not auto-display an older revision's preview after the saved head changes", async () => {
    const user = userEvent.setup();
    let finish: ((value: unknown) => void) | undefined;
    vi.mocked(rendersClient.submitRender).mockImplementation(() => new Promise((resolve) => { finish = resolve; }) as never);
    const view = renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={physicalStillRevision(7n)} historical={false} hasUnsavedChanges={false} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.renderPreview) }));
    view.rerender(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={physicalStillRevision(8n)} historical={false} hasUnsavedChanges={false} />);
    const output = create(RenderOutputSchema, { mediaType: "image/png", sizeBytes: 3n, widthPixels: 960, heightPixels: 540, sha256: "2".repeat(64), contentUrl: "/api/v1/projects/project-1/renders/preview-old-head/content" });
    finish?.({ job: job("preview-old-head", 7n, RenderJobStatus.SUCCEEDED, { renderMode: RenderMode.PHYSICAL_PREVIEW_PNG, output }), reusedExisting: false });
    await waitFor(() => expect(rendersClient.submitRender).toHaveBeenCalledTimes(1));
    expect(screen.queryByRole("link", { name: i18n.t(strings.shot.renderDownload) })).not.toBeInTheDocument();
    expect(getRenderOutputContent).not.toHaveBeenCalled();
  });

  it.each([RenderJobStatus.SUCCEEDED, RenderJobStatus.FAILED])("stops fast polling when the server reports a terminal preview (status %s)", async (terminalStatus) => {
    const user = userEvent.setup();
    const queued = job("preview-poll-stop", 7n, RenderJobStatus.QUEUED, { renderMode: RenderMode.PHYSICAL_PREVIEW_PNG });
    const output = create(RenderOutputSchema, { mediaType: "image/png", sizeBytes: 3n, widthPixels: 960, heightPixels: 540, sha256: "3".repeat(64), contentUrl: "/api/v1/projects/project-1/renders/preview-poll-stop/content" });
    const terminal = job("preview-poll-stop", 7n, terminalStatus, { renderMode: RenderMode.PHYSICAL_PREVIEW_PNG, ...(terminalStatus === RenderJobStatus.SUCCEEDED ? { output } : {}) });
    let listCalls = 0;
    vi.mocked(rendersClient.listRenders).mockImplementation(async () => {
      listCalls += 1;
      if (listCalls === 1) return { jobs: [], nextPageToken: "" } as never;
      return { jobs: [listCalls < 3 ? queued : terminal], nextPageToken: "" } as never;
    });
    vi.mocked(rendersClient.submitRender).mockResolvedValue({ job: queued, reusedExisting: false } as never);
    if (terminalStatus === RenderJobStatus.SUCCEEDED) {
      const blob = new Blob(["preview"], { type: "image/png" });
      vi.mocked(getRenderOutputContent).mockResolvedValue(blob);
      vi.mocked(verifyRenderOutputContent).mockResolvedValue(blob);
      vi.stubGlobal("URL", { ...URL, createObjectURL: vi.fn(() => "blob:poll-stop"), revokeObjectURL: vi.fn() });
    }
    renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={physicalStillRevision()} historical={false} hasUnsavedChanges={false} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.renderPreview) }));
    if (terminalStatus === RenderJobStatus.SUCCEEDED) {
      await screen.findByRole("link", { name: i18n.t(strings.shot.renderDownload) });
      await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.renderHideOutput) }));
    } else {
      await screen.findByText(i18n.t(strings.shot.renderFailed));
    }
    await waitFor(() => expect(listCalls).toBeGreaterThanOrEqual(3));
    const callsAtTerminal = listCalls;
    await new Promise((resolve) => setTimeout(resolve, 700));
    expect(listCalls).toBe(callsAtTerminal);
  });

  it("streams a selected MP4 output by its scoped URL without creating a full-file Blob", async () => {
    const user = userEvent.setup();
    const output = create(RenderOutputSchema, { mediaType: "video/mp4", sizeBytes: 9n, widthPixels: 1920, heightPixels: 1080, sha256: "d".repeat(64), contentUrl: "/api/v1/projects/project-1/renders/final-video/content" });
    vi.mocked(rendersClient.listRenders).mockResolvedValue({ jobs: [job("final-video", 7n, RenderJobStatus.SUCCEEDED, { renderMode: RenderMode.VIDEO_FINAL_MP4, output })], nextPageToken: "" } as never);
    const onFinalOutputChange = vi.fn();
    renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={videoRevision()} historical={false} hasUnsavedChanges={false} onFinalOutputChange={onFinalOutputChange} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.shot.renderViewFinalMP4) }));
    const expectedURL = renderOutputContentURL("project-1", "final-video", output.contentUrl);
    await waitFor(() => expect(onFinalOutputChange).toHaveBeenCalledWith(expect.objectContaining({ renderId: "final-video", url: expectedURL, isVideo: true })));
    expect(getRenderOutputContent).not.toHaveBeenCalled();
    expect(screen.getByRole("link", { name: i18n.t(strings.shot.renderDownloadMP4) })).toHaveAttribute("href", expectedURL);
    expect(screen.getByRole("link", { name: i18n.t(strings.shot.renderDownloadMP4) })).toHaveAttribute("download", "device-studio-final-video.mp4");
  });

  it("surfaces an invalid MP4 content binding without crashing the editor", async () => {
    const user = userEvent.setup();
    const output = create(RenderOutputSchema, { mediaType: "video/mp4", sizeBytes: 9n, widthPixels: 1920, heightPixels: 1080, sha256: "d".repeat(64), contentUrl: "https://example.invalid/private.mp4" });
    vi.mocked(rendersClient.listRenders).mockResolvedValue({ jobs: [job("bad-video", 7n, RenderJobStatus.SUCCEEDED, { renderMode: RenderMode.VIDEO_FINAL_MP4, output })], nextPageToken: "" } as never);
    renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={videoRevision()} historical={false} hasUnsavedChanges={false} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.shot.renderViewFinalMP4) }));
    expect(await screen.findByRole("alert")).toHaveTextContent(/project-scoped content path/);
    expect(screen.queryByRole("link", { name: i18n.t(strings.shot.renderDownloadMP4) })).not.toBeInTheDocument();
  });

  it("keeps an in-flight submit bound to its original shot and clears only that retry key", async () => {
    const user = userEvent.setup();
    const oldKey = "device-studio:render:project-1:shot-1:7";
    sessionStorage.setItem(oldKey, "retry-original");
    let resolveSubmit: ((value: unknown) => void) | undefined;
    vi.mocked(rendersClient.submitRender).mockImplementation(() => new Promise((resolve) => { resolveSubmit = resolve; }) as never);
    const view = renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={savedRevision(7n)} historical={false} hasUnsavedChanges={false} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.shot.renderRevisionAction) }));
    const originalInput = vi.mocked(rendersClient.submitRender).mock.calls[0]?.[0];
    expect(originalInput?.idempotencyKey).toBe("retry-original");

    view.rerender(<RenderControlsHarness projectId="project-1" shotId="shot-2" revision={create(ShotRevisionSchema, { ...savedRevision(8n), shotId: "shot-2" })} historical={false} hasUnsavedChanges={false} />);
    resolveSubmit?.({ job: job("old-shot-render", 7n, RenderJobStatus.QUEUED), reusedExisting: false });
    await waitFor(() => expect(sessionStorage.getItem(oldKey)).toBeNull());
    expect(screen.queryByText(i18n.t(strings.shot.renderQueued))).not.toBeInTheDocument();

    await user.click(await screen.findByRole("button", { name: i18n.t(strings.shot.renderRevisionAction) }));
    expect(vi.mocked(rendersClient.submitRender).mock.calls[1]?.[0]).toMatchObject({ shotId: "shot-2", revision: 8n });
    expect(vi.mocked(rendersClient.submitRender).mock.calls[1]?.[0].idempotencyKey).not.toBe("retry-original");
  });

  it("[REQ:DS-JOB-DURABILITY][REQ:DS-INTEGRATION-OPERATIONS] reopens project jobs and keeps each verified PNG attached only to its selected job", async () => {
    const user = userEvent.setup();
    const output = (sha256: string) => create(RenderOutputSchema, { mediaType: "image/png", sizeBytes: 3n, widthPixels: 12, heightPixels: 8, sha256, contentUrl: "/api/v1/projects/project-1/renders/job/content" });
    const first = job("render-a", 2n, RenderJobStatus.SUCCEEDED, { output: output("a".repeat(64)) });
    const second = job("render-b", 3n, RenderJobStatus.SUCCEEDED, { output: output("b".repeat(64)) });
    vi.mocked(rendersClient.listRenders)
      .mockResolvedValueOnce({ jobs: [first, second], nextPageToken: "" } as never)
      .mockResolvedValue({ jobs: [job("render-a", 2n, RenderJobStatus.SUCCEEDED, { output: output("a".repeat(64)) }), job("render-b", 3n, RenderJobStatus.SUCCEEDED, { output: output("b".repeat(64)) })], nextPageToken: "" } as never);
    const blobA = new Blob(["first"], { type: "image/png" });
    const blobB = new Blob(["second"], { type: "image/png" });
    const onFinalOutputChange = vi.fn();
    vi.mocked(getRenderOutputContent).mockResolvedValueOnce(blobA).mockResolvedValueOnce(blobB);
    vi.mocked(verifyRenderOutputContent).mockImplementation((blob) => Promise.resolve(blob));
    const createObjectURL = vi.fn((blob: Blob) => blob === blobA ? "blob:render-a" : "blob:render-b");
    vi.stubGlobal("URL", { ...URL, createObjectURL, revokeObjectURL: vi.fn() });
    const view = renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={savedRevision()} historical={false} hasUnsavedChanges={false} onFinalOutputChange={onFinalOutputChange} />);
    await screen.findAllByText(i18n.t(strings.shot.renderSucceeded));
    const renderButtons = [...document.querySelectorAll<HTMLButtonElement>(".shot-render-job > button")];
    const renderA = renderButtons[0];
    const renderB = renderButtons[1];
    if (!renderA || !renderB) throw new Error("expected two durable render rows");
    const rowA = renderA.closest<HTMLElement>(".shot-render-job");
    if (!rowA) throw new Error("first render row missing");
    await user.click(within(rowA).getByRole("button", { name: new RegExp(i18n.t(strings.shot.revision)) }));
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.shot.renderViewOutput) }));
    await waitFor(() => expect(onFinalOutputChange).toHaveBeenCalledWith(expect.objectContaining({ renderId: "render-a", url: "blob:render-a", revision: 2n })));
    expect(screen.getByRole("link", { name: i18n.t(strings.shot.renderDownload) })).toHaveAttribute("href", "blob:render-a");
    await view.queryClient.invalidateQueries({ queryKey: ["renders", "project-1"] });
    await waitFor(() => expect(rendersClient.listRenders).toHaveBeenCalledTimes(2));
    expect(createObjectURL).toHaveBeenCalledTimes(1);

    await user.click(renderB);
    expect(screen.queryByRole("link", { name: i18n.t(strings.shot.renderDownload) })).not.toBeInTheDocument();
    const rowB = renderB.closest<HTMLElement>(".shot-render-job");
    if (!rowB) throw new Error("second render row missing");
    await user.click(within(rowB).getByRole("button", { name: new RegExp(i18n.t(strings.shot.revision)) }));
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.shot.renderViewOutput) }));
    await waitFor(() => expect(onFinalOutputChange).toHaveBeenCalledWith(expect.objectContaining({ renderId: "render-b", url: "blob:render-b", revision: 3n })));
  });

  it("does not reveal output when a pending verification finishes after the viewer is hidden", async () => {
    const user = userEvent.setup();
    const output = create(RenderOutputSchema, { mediaType: "image/png", sizeBytes: 3n, widthPixels: 12, heightPixels: 8, sha256: "c".repeat(64), contentUrl: "/api/v1/projects/project-1/renders/render-late/content" });
    vi.mocked(rendersClient.listRenders).mockResolvedValue({ jobs: [job("render-late", 2n, RenderJobStatus.SUCCEEDED, { output })], nextPageToken: "" } as never);
    let resolveOutput: ((blob: Blob) => void) | undefined;
    vi.mocked(getRenderOutputContent).mockImplementation(() => new Promise((resolve) => { resolveOutput = resolve; }));
    vi.mocked(verifyRenderOutputContent).mockImplementation((blob) => Promise.resolve(blob));
    const onFinalOutputChange = vi.fn();
    vi.stubGlobal("URL", { ...URL, createObjectURL: vi.fn(() => "blob:late"), revokeObjectURL: vi.fn() });
    renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={savedRevision()} historical={false} hasUnsavedChanges={false} onFinalOutputChange={onFinalOutputChange} />);
    await screen.findAllByText(i18n.t(strings.shot.renderSucceeded));
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.shot.renderViewOutput) }));
    await waitFor(() => expect(getRenderOutputContent).toHaveBeenCalledTimes(1));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.renderHideOutput) }));
    resolveOutput?.(new Blob(["png"], { type: "image/png" }));
    await waitFor(() => expect(verifyRenderOutputContent).toHaveBeenCalled());
    expect(screen.queryByRole("link", { name: i18n.t(strings.shot.renderDownload) })).not.toBeInTheDocument();
    expect(onFinalOutputChange).not.toHaveBeenCalledWith(expect.objectContaining({ renderId: "render-late" }));
  });

  it("loads the next durable project-job page without losing the first page", async () => {
    const user = userEvent.setup();
    const first = job("render-page-1", 1n, RenderJobStatus.FAILED);
    const next = job("render-page-2", 2n, RenderJobStatus.CANCELLED);
    vi.mocked(rendersClient.listRenders)
      .mockResolvedValueOnce({ jobs: [first], nextPageToken: "cursor-2" } as never)
      .mockResolvedValueOnce({ jobs: [next], nextPageToken: "" } as never);
    renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={savedRevision()} historical={false} hasUnsavedChanges={false} />);
    await screen.findAllByText(i18n.t(strings.shot.renderFailed));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.renderLoadMore) }));
    await screen.findAllByText(i18n.t(strings.shot.renderCancelled));
    expect(rendersClient.listRenders).toHaveBeenNthCalledWith(2, { projectId: "project-1", pageSize: 50, pageToken: "cursor-2" });
    expect(document.querySelectorAll(".shot-render-job")).toHaveLength(2);
  });

  it("[REQ:DS-JOB-CANCEL] uses the job version for cancellation and refreshes after a stale cancel conflict", async () => {
    const user = userEvent.setup();
    const queued = job("render-queued", 7n, RenderJobStatus.QUEUED, { version: 9n });
    vi.mocked(rendersClient.listRenders).mockResolvedValue({ jobs: [queued], nextPageToken: "" } as never);
    vi.mocked(rendersClient.cancelRender).mockRejectedValue(new ConnectError("changed", Code.Aborted));
    renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={savedRevision()} historical={false} hasUnsavedChanges={false} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.shot.renderCancel) }));
    await screen.findByText(i18n.t(strings.shot.renderCancelConflict));
    expect(rendersClient.cancelRender).toHaveBeenCalledWith({ projectId: "project-1", renderId: "render-queued", expectedVersion: 9n });
    expect(rendersClient.listRenders).toHaveBeenCalledTimes(2);
  });

  it("binds an attended readability acceptance to the displayed revision and exact final output", async () => {
    const user = userEvent.setup();
    const proof = create(ReadabilityProofSchema, { regionName: "Browser address", requiredContent: "example.test", criteria: "Text is legible at final size", regionX: 0.1, regionY: 0.1, regionWidth: 0.8, regionHeight: 0.2, minimumOutputWidthPixels: 1000, minimumOutputHeightPixels: 700, minimumRegionWidthPixels: 500, minimumRegionHeightPixels: 100 });
    const revision = create(ShotRevisionSchema, { ...savedRevision(), snapshot: create(ShotSnapshotSchema, { readabilityProof: proof }) });
    const output = create(RenderOutputSchema, { mediaType: "image/png", sizeBytes: 3n, widthPixels: 1600, heightPixels: 900, sha256: "a".repeat(64), contentUrl: "/api/v1/projects/project-1/renders/final-readable/content" });
    vi.mocked(rendersClient.listRenders).mockResolvedValue({ jobs: [job("final-readable", 7n, RenderJobStatus.SUCCEEDED, { output, renderMode: RenderMode.STILL })], nextPageToken: "" } as never);
    vi.mocked(getRenderOutputContent).mockResolvedValue(new Blob(["png"], { type: "image/png" }));
    vi.mocked(verifyRenderOutputContent).mockResolvedValue(new Blob(["png"], { type: "image/png" }));
    vi.stubGlobal("URL", { ...URL, createObjectURL: vi.fn(() => "blob:readability"), revokeObjectURL: vi.fn() });
    vi.mocked(shotsClient.reviewOutputReadability).mockResolvedValue({ review: { verdict: ReadabilityVerdict.ACCEPTED } } as never);
    renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={revision} historical={false} hasUnsavedChanges={false} />);
    await screen.findByText(i18n.t(strings.shot.readabilityUnreviewed));
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.shot.renderViewOutput) }));
    await screen.findByRole("img", { name: i18n.t(strings.shot.readabilityReviewImage) });
    const nativeView = screen.getByTestId("readability-native-size-view");
    expect(nativeView.className).toContain("overflow-auto");
    expect(screen.getByRole("img", { name: i18n.t(strings.shot.readabilityReviewImage) })).toHaveStyle({ width: "1600px", height: "900px", maxWidth: "none" });
    const inspected = screen.getByRole("checkbox", { name: i18n.t(strings.shot.readabilityAttestation) });
    expect(screen.getByRole("button", { name: i18n.t(strings.shot.readabilityAccept) })).toBeDisabled();
    await user.click(inspected);
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.readabilityAccept) }));
    await waitFor(() => expect(shotsClient.reviewOutputReadability).toHaveBeenCalledWith(expect.objectContaining({
      projectId: "project-1", shotId: "shot-1", shotRevision: 7n, renderId: "final-readable", outputSha256: "a".repeat(64), verdict: ReadabilityVerdict.ACCEPTED, explicitlyInspectedFinalSize: true,
    })));
  });

  it("does not allow an attended acceptance when the final image is below its declared pixel minimum", async () => {
    const proof = create(ReadabilityProofSchema, { regionName: "Browser address", requiredContent: "example.test", criteria: "Text is legible at final size", regionX: 0.1, regionY: 0.1, regionWidth: 0.5, regionHeight: 0.1, minimumOutputWidthPixels: 1000, minimumOutputHeightPixels: 700, minimumRegionWidthPixels: 500, minimumRegionHeightPixels: 100 });
    const revision = create(ShotRevisionSchema, { ...savedRevision(), snapshot: create(ShotSnapshotSchema, { readabilityProof: proof }) });
    const output = create(RenderOutputSchema, { mediaType: "image/png", sizeBytes: 3n, widthPixels: 800, heightPixels: 600, sha256: "b".repeat(64), contentUrl: "/api/v1/projects/project-1/renders/too-small/content" });
    vi.mocked(rendersClient.listRenders).mockResolvedValue({ jobs: [job("too-small", 7n, RenderJobStatus.SUCCEEDED, { output, renderMode: RenderMode.STILL })], nextPageToken: "" } as never);
    const blob = new Blob(["png"], { type: "image/png" });
    vi.mocked(getRenderOutputContent).mockResolvedValue(blob);
    vi.mocked(verifyRenderOutputContent).mockResolvedValue(blob);
    vi.stubGlobal("URL", { ...URL, createObjectURL: vi.fn(() => "blob:too-small"), revokeObjectURL: vi.fn() });
    const user = userEvent.setup();
    renderWithProviders(<RenderControlsHarness projectId="project-1" shotId="shot-1" revision={revision} historical={false} hasUnsavedChanges={false} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.shot.renderViewOutput) }));
    await user.click(await screen.findByRole("checkbox", { name: i18n.t(strings.shot.readabilityAttestation) }));
    expect(await screen.findByRole("button", { name: i18n.t(strings.shot.readabilityAccept) })).toBeDisabled();
    expect(shotsClient.reviewOutputReadability).not.toHaveBeenCalled();
  });
});
