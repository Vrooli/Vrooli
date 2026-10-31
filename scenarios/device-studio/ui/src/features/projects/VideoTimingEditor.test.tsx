import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import userEvent from "@testing-library/user-event";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import { AudioInspectionFactsSchema, AudioStreamFactSchema, GetSourceMediaClockPageResponseSchema, MediaInspectionReceiptSchema, SourceClockPointSchema, SourceMediaKind, MediaReadiness, SourceMediaSchema } from "@vrooli/proto-types/device-studio/v1/projects/projects_pb";
import { VideoAudioPolicy, VideoEOFPolicy, VideoSampleBindingSchema, VideoTimingSchema } from "@vrooli/proto-types/device-studio/v1/shots/shots_pb";
import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { i18n, setLocale } from "../../i18n";
import { projectsClient } from "../../api/projects";
import { VideoTimingEditor } from "./VideoTimingEditor";

vi.mock("../../api/projects", () => ({ projectsClient: { getSourceMediaClockPage: vi.fn() } }));

const source = create(SourceMediaSchema, { id: "video-1", projectId: "project-1", kind: SourceMediaKind.VIDEO, readiness: MediaReadiness.READY });
const timing = create(VideoTimingSchema, {
  sourceMediaId: "video-1", sourceSha256: "a".repeat(64), sourceSizeBytes: 500n,
  clockSha256: "b".repeat(64), clockSizeBytes: 48n, timeBaseNum: 1n, timeBaseDen: 24n,
  streamStartPts: 0n, sourceEndPts: 120n, trimStartPts: 0n, trimEndPts: 120n,
  lastFrameDurationTicks: 1n, playbackRateNum: 1, playbackRateDen: 1, outputRateNum: 24, outputRateDen: 1,
  eofPolicy: VideoEOFPolicy.VIDEO_EOF_POLICY_REJECT,
});
const page = (points: Array<{ ordinal: number; pts: bigint }>, nextPageToken = "") => create(GetSourceMediaClockPageResponseSchema, {
  clockSha256: timing.clockSha256, timeBaseNumerator: 1n, timeBaseDenominator: 24n, points: points.map((point) => create(SourceClockPointSchema, point)), nextPageToken,
});

describe("VideoTimingEditor", () => {
  beforeEach(async () => { await setLocale("en"); vi.mocked(projectsClient.getSourceMediaClockPage).mockReset(); });
  afterEach(() => { cleanup(); vi.clearAllMocks(); });

  it("pins frame choices to the expected clock and preserves exact PTS while paging", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    vi.mocked(projectsClient.getSourceMediaClockPage)
      .mockResolvedValueOnce(page([{ ordinal: 0, pts: 0n }, { ordinal: 1, pts: 40n }, { ordinal: 2, pts: 80n }], "next"))
      .mockResolvedValueOnce(page([{ ordinal: 3, pts: 120n }], ""));
    renderWithProviders(<VideoTimingEditor projectId="project-1" source={source} timing={timing} disabled={false} onChange={onChange} />);
    const start = await screen.findByLabelText(i18n.t(strings.shot.videoTrimStart));
    await screen.findByRole("option", { name: /Frame 2 · 1\.666s/ });
    await user.selectOptions(start, "40");
    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ trimStartPts: 40n, clockSha256: timing.clockSha256 }));
    expect(projectsClient.getSourceMediaClockPage).toHaveBeenCalledWith({ projectId: "project-1", mediaId: "video-1", expectedClockSha256: timing.clockSha256, pageSize: 256, pageToken: "" });
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.videoLoadClock) }));
    await waitFor(() => expect(projectsClient.getSourceMediaClockPage).toHaveBeenLastCalledWith(expect.objectContaining({ pageToken: "next", expectedClockSha256: timing.clockSha256 })));
    expect(screen.getByLabelText(i18n.t(strings.shot.videoTrimEnd))).toHaveTextContent("5.000s");
  });

  it("refuses pages whose digest or timebase differs from the saved timing", async () => {
    vi.mocked(projectsClient.getSourceMediaClockPage).mockResolvedValue(create(GetSourceMediaClockPageResponseSchema, {
      ...page([{ ordinal: 0, pts: 0n }]), clockSha256: "c".repeat(64), timeBaseDenominator: 30n,
    }));
    renderWithProviders(<VideoTimingEditor projectId="project-1" source={source} timing={timing} disabled={false} onChange={vi.fn()} />);
    expect(await screen.findByRole("alert")).toHaveTextContent(i18n.t(strings.shot.videoClockMismatch));
    expect(screen.getByLabelText(i18n.t(strings.shot.videoTrimStart))).toBeDisabled();
  });

  it("shows the ten-second limit even when the source-rate frame count is below 300", async () => {
    const longClip = create(VideoTimingSchema, { ...timing, sourceEndPts: 264n, trimEndPts: 264n });
    vi.mocked(projectsClient.getSourceMediaClockPage).mockResolvedValue(page([]));
    renderWithProviders(<VideoTimingEditor projectId="project-1" source={source} timing={longClip} disabled={false} onChange={vi.fn()} />);
    expect(await screen.findByRole("alert")).toHaveTextContent(i18n.t(strings.shot.videoClipTooLong));
  });

  it("keeps a saved rational playback rate visible instead of showing an empty selector", async () => {
    const customRate = create(VideoTimingSchema, { ...timing, playbackRateNum: 3, playbackRateDen: 2 });
    vi.mocked(projectsClient.getSourceMediaClockPage).mockResolvedValue(page([]));
    renderWithProviders(<VideoTimingEditor projectId="project-1" source={source} timing={customRate} disabled={false} onChange={vi.fn()} />);
    const speed = await screen.findByLabelText(i18n.t(strings.shot.videoPlaybackRate));
    expect(speed).toHaveValue("3/2");
    expect(speed).toHaveTextContent(i18n.t(strings.shot.videoSavedRate, { rate: "3/2" }));
  });

  it.each([
    ["unreported", undefined, strings.shot.videoAudioUnreported],
    ["observed zero", create(AudioInspectionFactsSchema, { version: 1, streams: [] }), strings.shot.videoAudioNone],
    ["detected", create(AudioInspectionFactsSchema, { version: 1, streams: [create(AudioStreamFactSchema, { originalStreamIndex: 1, codec: "pcm_s16le" })] }), strings.shot.videoAudioPresent],
  ])("explains silent output for %s source audio facts", async (_state, audio, message) => {
    vi.mocked(projectsClient.getSourceMediaClockPage).mockResolvedValue(page([]));
    const sourceWithReceipt = create(SourceMediaSchema, {
      ...source,
      inspectionReceipt: create(MediaInspectionReceiptSchema, { ...(audio ? { audio } : {}) }),
    });
    renderWithProviders(<VideoTimingEditor projectId="project-1" source={sourceWithReceipt} timing={timing} disabled={false} onChange={vi.fn()} />);
    expect(await screen.findByRole("status")).toHaveTextContent(i18n.t(message));
    expect(screen.queryByText(/passthrough/i)).not.toBeInTheDocument();
  });

  it("offers the documented silent repair only for saved passthrough policy and invalidates its sample", async () => {
    const user = userEvent.setup();
    vi.mocked(projectsClient.getSourceMediaClockPage).mockResolvedValue(page([]));
    const onChange = vi.fn();
    const blockedTiming = create(VideoTimingSchema, {
      ...timing,
      audioPolicy: VideoAudioPolicy.PASSTHROUGH,
      sampleBinding: create(VideoSampleBindingSchema, {
        jobId: "sample-1", sourceSha256: "a".repeat(64), clockSha256: timing.clockSha256,
        trimStartPts: 0n, trimEndPts: 48n, baseShotRevision: 1n, baseRecipeVersion: "device-studio-video-recipe-v1",
      }),
      sampleBaseRecipeVersion: 1,
    });
    renderWithProviders(<VideoTimingEditor projectId="project-1" source={source} timing={blockedTiming} disabled={false} onChange={onChange} />);
    const repair = await screen.findByRole("button", { name: i18n.t(strings.shot.videoUseSilentOutput) });
    await user.click(repair);
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({
      audioPolicy: VideoAudioPolicy.SILENT,
      sampleBinding: undefined,
      sampleBaseRecipeVersion: 0,
    }));

    cleanup();
    renderWithProviders(<VideoTimingEditor projectId="project-1" source={source} timing={timing} disabled={false} onChange={vi.fn()} />);
    expect(screen.queryByRole("button", { name: i18n.t(strings.shot.videoUseSilentOutput) })).not.toBeInTheDocument();
  });
});
