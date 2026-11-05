import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";
import { create, toJson } from "@bufbuild/protobuf";
import { CaptureArea, CoordinateConvention, MotionMode, PhotographicCameraMotionSchema, PhotographicSceneSchema, PhoneSystemChromeSchema, PresentationAppearance, PresentationKind, RenderJobSchema, RenderJobStatus, RenderMode, RenderOutputSchema, ScreenFitMode, ScreenPresentationSchema, ShotConflictSchema, ShotIssueCode, ShotRevisionSchema, ShotSchema, ShotSnapshotSchema, VideoEOFPolicy, VideoSampleBindingSchema, VideoTimingSchema, type ShotSnapshot } from "@vrooli/proto-types/device-studio/v1/shots/shots_pb";
import { GetSourceMediaClockPageResponseSchema, MediaInspectionReceiptSchema, MediaReadiness, SourceClockPointSchema, SourceMediaKind, SourceMediaSchema } from "@vrooli/proto-types/device-studio/v1/projects/projects_pb";
import { renderWithProviders } from "../../test-utils";
import { i18n, setLocale } from "../../i18n";
import { strings } from "../../consts/strings";
import { ShotWorkspace } from "./ShotWorkspace";

vi.mock("../../api/projects", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/projects")>();
  return { ...actual, projectsClient: { ...actual.projectsClient, listSourceMedia: vi.fn().mockResolvedValue({ sourceMedia: [] }), getSourceMedia: vi.fn(), getSourceMediaClockPage: vi.fn() }, getSourceMediaContent: vi.fn() };
});
vi.mock("../../api/renders", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/renders")>();
  return { ...actual, rendersClient: { ...actual.rendersClient, listRenders: vi.fn().mockResolvedValue({ jobs: [], nextPageToken: "" }) }, getRenderOutputContent: vi.fn(), getVideoSampleBinding: vi.fn(), verifyRenderOutputContent: vi.fn((blob: Blob) => Promise.resolve(blob)) };
});
vi.mock("../../api/shots", () => ({
  shotsClient: {
    listShots: vi.fn(), createShot: vi.fn(), getShot: vi.fn(), getShotRevision: vi.fn(), listShotRevisions: vi.fn(),
    saveShot: vi.fn(), validateShot: vi.fn(),
  },
  shotConflict: (error: unknown) => error instanceof ConnectError ? error.findDetails(ShotConflictSchema)[0] : undefined,
}));
vi.mock("../../api/physical", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/physical")>();
  return { ...actual, getReviewedPhysicalStarter: vi.fn().mockRejectedValue(new Error("starter unavailable in this test")) };
});
vi.mock("../../api/photographic-scene", () => ({ cancelPhotographicOperation: vi.fn(), getPhotographicPNG: vi.fn(), readPhotographicOperation: vi.fn().mockRejectedValue(new Error("no saved operation")), submitPhotographicScene: vi.fn(), waitPhotographicOperation: vi.fn() }));
import { shotsClient } from "../../api/shots";
import { getSourceMediaContent, projectsClient, sourceMediaContentURL } from "../../api/projects";
import { getRenderOutputContent, getVideoSampleBinding, rendersClient } from "../../api/renders";
import { submitPhotographicScene } from "../../api/photographic-scene";
import { PhotographicScenePanel } from "./PhotographicScenePanel";

const snapshot = create(ShotSnapshotSchema, {});
const shot = (revision: bigint, name = "Daylight") => create(ShotSchema, { id: "shot-1", projectId: "project-1", name, headRevision: revision });
const revision = (value: bigint, name = "Daylight", savedSnapshot = snapshot) => create(ShotRevisionSchema, { shotId: "shot-1", revision: value, name, snapshot: savedSnapshot, issues: [] });
const list = () => ({ shots: [shot(1n)], nextPageToken: "" });
async function openShotManagement(user: ReturnType<typeof userEvent.setup>) {
  const summary = screen.getByText(new RegExp(`^${i18n.t(strings.shot.savedShots)}`));
  if (!summary.closest("details")?.open) await user.click(summary);
}
async function openShotHistory(user: ReturnType<typeof userEvent.setup>) {
  const summary = screen.getByText(new RegExp(`^${i18n.t(strings.shot.pinnedRevision)}`), { selector: "summary" });
  if (!summary.closest("details")?.open) await user.click(summary);
}

describe("ShotWorkspace", () => {
  beforeEach(async () => { await setLocale("en"); vi.mocked(rendersClient.listRenders).mockReset().mockResolvedValue({ jobs: [], nextPageToken: "" } as never); });
  afterEach(() => { cleanup(); vi.clearAllMocks(); vi.unstubAllGlobals(); });

  it("[REQ:DS-EDIT-GUIDED][REQ:DS-CONTRACT-SHOT] creates and opens a shot using the typed incomplete draft contract", async () => {
    const user = userEvent.setup();
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.createShot).mockResolvedValue({ shot: shot(1n), revision: revision(1n) } as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head: revision(1n) } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [revision(1n)], nextPageToken: "" } as never);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.type(await screen.findByTestId("new-shot-name"), "Daylight");
    await user.click(screen.getByRole("button", { name: "New shot" }));
    await screen.findByTestId("shot-editor");
    const request = vi.mocked(shotsClient.createShot).mock.calls[0]?.[0];
    if (!request?.snapshot) throw new Error("create request snapshot missing");
    expect(request.projectId).toBe("project-1");
    expect(request.name).toBe("Daylight");
    const createdScene = request.snapshot.scene;
    const createdPresentation = request.snapshot.presentation;
    if (!createdScene || !createdPresentation) throw new Error("created snapshot sections missing");
    expect(createdPresentation.sourceMediaId).toBe("");
    expect(createdScene.plateQuad).toBeUndefined();
    expect(JSON.stringify(toJson(ShotSnapshotSchema, create(ShotSnapshotSchema, request.snapshot)))).not.toContain("plateQuad");
  });

  it("opens the physical still setup from the existing shot workspace", async () => {
    const user = userEvent.setup();
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.newButton) }));
    expect(await screen.findByRole("heading", { name: i18n.t(strings.physical.newTitle) })).toBeInTheDocument();
    expect(screen.getByTestId("physical-shot-create")).toBeInTheDocument();
  });

  it("puts a verified final PNG in the main stage and returns explicitly to calibration", async () => {
    const user = userEvent.setup();
    const output = create(RenderOutputSchema, { mediaType: "image/png", sizeBytes: 3n, widthPixels: 1206, heightPixels: 2622, sha256: "d".repeat(64), contentUrl: "/api/v1/projects/project-1/renders/render-1/content" });
    const render = create(RenderJobSchema, { id: "render-1", projectId: "project-1", shotId: "shot-1", shotRevision: 1n, status: RenderJobStatus.SUCCEEDED, version: 2n, output });
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head: revision(1n) } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [revision(1n)], nextPageToken: "" } as never);
    vi.mocked(rendersClient.listRenders).mockResolvedValue({ jobs: [render], nextPageToken: "" } as never);
    vi.mocked(getRenderOutputContent).mockResolvedValue(new Blob(["png"], { type: "image/png" }));
    vi.stubGlobal("URL", { ...URL, createObjectURL: vi.fn(() => "blob:final-stage"), revokeObjectURL: vi.fn() });
    const view = renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await screen.findByTestId("shot-editor");
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.shot.renderViewOutput) }));
    const finalStage = await screen.findByTestId("shot-video-final-stage");
    expect(screen.getByTestId("shot-editor").querySelector(".shot-stage-focus")).toHaveFocus();
    expect(finalStage.querySelector("img")).toHaveAttribute("src", "blob:final-stage");
    const finalStageCard = finalStage.closest<HTMLElement>(".shot-plate-stage");
    if (!finalStageCard) throw new Error("final stage card missing");
    expect(within(finalStageCard).getByText((_, element) => element?.textContent === `${output.widthPixels} × ${output.heightPixels} px`)).toBeInTheDocument();
    expect(within(finalStageCard).queryByText((_, element) => /image\/png|3 bytes/.test(element?.textContent ?? ""))).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: i18n.t(strings.shot.renderFinalTitle, { revision: "1" }) })).toBeInTheDocument();
    await user.click(within(screen.getByTestId("shot-editor").querySelector(".shot-plate-stage") as HTMLElement).getByRole("button", { name: i18n.t(strings.shot.renderHideOutput) }));
    await waitFor(() => expect(screen.queryByTestId("shot-video-final-stage")).not.toBeInTheDocument());
    await view.queryClient.invalidateQueries({ queryKey: ["renders", "project-1"] });
    await waitFor(() => expect(rendersClient.listRenders).toHaveBeenCalledTimes(2));
    expect(screen.queryByTestId("shot-video-final-stage")).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: i18n.t(strings.shot.scenePlate) })).toBeInTheDocument();
  });

  it("saves an incomplete shot without serializing a phantom plate quad", async () => {
    const user = userEvent.setup();
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head: revision(1n) } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [revision(1n)], nextPageToken: "" } as never);
    vi.mocked(shotsClient.saveShot).mockResolvedValue({ shot: shot(2n), revision: revision(2n) } as never);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.click(await screen.findByRole("button", { name: /Daylight/ }));
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.shot.save) }));
    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenCalled());
    const sent = vi.mocked(shotsClient.saveShot).mock.calls[0]?.[0].snapshot;
    if (!sent) throw new Error("save request snapshot missing");
    const savedScene = sent.scene;
    if (!savedScene) throw new Error("saved scene missing");
    expect(savedScene.coordinateConvention).toBe(0);
    expect(savedScene.plateQuad).toBeUndefined();
    expect(JSON.stringify(toJson(ShotSnapshotSchema, create(ShotSnapshotSchema, sent)))).not.toContain("plateQuad");
  });

  it.each([
    ["absent", undefined, "84"],
    ["explicit zero", 0, "0"],
  ])("reopens optional phone battery values (%s) with a usable default", async (_label, battery, expected) => {
    const user = userEvent.setup();
    const phone = create(ShotSnapshotSchema, { presentation: create(ScreenPresentationSchema, {
      captureArea: CaptureArea.APP_ONLY, kind: PresentationKind.PHONE_SYSTEM, fitMode: ScreenFitMode.CONTAIN,
      phoneSystemChrome: create(PhoneSystemChromeSchema, { batteryPercent: battery }),
    }) });
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head: revision(1n, "Daylight", phone) } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [revision(1n, "Daylight", phone)], nextPageToken: "" } as never);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.click(await screen.findByRole("button", { name: /Daylight/ }));
    await user.click(screen.getByText(i18n.t(strings.shot.screenPresentation)));
    await user.click(screen.getByText(i18n.t(strings.shot.phoneChrome)));
    expect(screen.getByLabelText(i18n.t(strings.shot.battery))).toHaveValue(Number(expected));
  });

  it("replaces submitted presentation draft with the server's canonical defaults after save", async () => {
    const user = userEvent.setup();
    const canonical = create(ShotSnapshotSchema, {
      motionMode: MotionMode.FIXED,
      scene: create(PhotographicSceneSchema, { coordinateConvention: CoordinateConvention.PLATE_RASTER_EDGE_PIXELS_V1 }),
      presentation: create(ScreenPresentationSchema, {
        captureArea: CaptureArea.APP_ONLY, kind: PresentationKind.PHONE_SYSTEM, fitMode: ScreenFitMode.CONTAIN,
        phoneSystemChrome: create(PhoneSystemChromeSchema, { carrierText: "Meridian", clockText: "9:41", batteryPercent: 84, showCellularIndicator: true, showWifiIndicator: true, showHomeIndicator: true, showStatus: true, showBatteryPercent: false, appearance: PresentationAppearance.LIGHT }),
      }),
    });
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head: revision(1n) } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [revision(1n)], nextPageToken: "" } as never);
    vi.mocked(shotsClient.saveShot).mockResolvedValue({ shot: shot(2n), revision: revision(2n, "Daylight", canonical) } as never);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.click(await screen.findByRole("button", { name: /Daylight/ }));
    await user.click(screen.getByText(i18n.t(strings.shot.screenPresentation)));
    const captureAreas = screen.getAllByLabelText(i18n.t(strings.shot.captureArea));
    await user.selectOptions(captureAreas[1]!, String(PresentationKind.PHONE_SYSTEM));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.save) }));
    await waitFor(() => expect(screen.getByLabelText(i18n.t(strings.shot.carrier))).toHaveValue("Meridian"));
    expect(screen.getByText(i18n.t(strings.shot.renderSaved, { revision: "2" }))).toBeInTheDocument();
  });

  it("reconciles server defaults after save but preserves edits made while save was pending", async () => {
    const user = userEvent.setup();
    const canonical = create(ShotSnapshotSchema, {
      motionMode: MotionMode.FIXED,
      scene: create(PhotographicSceneSchema, { coordinateConvention: CoordinateConvention.PLATE_RASTER_EDGE_PIXELS_V1 }),
      presentation: create(ScreenPresentationSchema, {
        captureArea: CaptureArea.APP_ONLY, kind: PresentationKind.PHONE_SYSTEM, fitMode: ScreenFitMode.CONTAIN,
        phoneSystemChrome: create(PhoneSystemChromeSchema, { carrierText: "Meridian", clockText: "9:41", batteryPercent: 84, showCellularIndicator: true, showWifiIndicator: true, showHomeIndicator: true, showStatus: true, showBatteryPercent: false, appearance: PresentationAppearance.LIGHT }),
      }),
    });
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head: revision(1n) } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [revision(1n)], nextPageToken: "" } as never);
    let resolveSave: ((result: unknown) => void) | undefined;
    vi.mocked(shotsClient.saveShot).mockImplementation(() => new Promise((resolve) => { resolveSave = resolve; }) as never);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.click(await screen.findByRole("button", { name: /Daylight/ }));
    await user.click(screen.getByText(i18n.t(strings.shot.screenPresentation)));
    const captureAreas = screen.getAllByLabelText(i18n.t(strings.shot.captureArea));
    await user.selectOptions(captureAreas[1]!, String(PresentationKind.PHONE_SYSTEM));
    const name = screen.getByTestId("shot-name-input");
    await user.clear(name);
    await user.type(name, "Submitted name");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.save) }));
    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenCalledTimes(1));
    await user.clear(name);
    await user.type(name, "Later edit");
    resolveSave?.({ shot: shot(2n, "Submitted name"), revision: revision(2n, "Submitted name", canonical) });
    await waitFor(() => expect(screen.getByTestId("shot-name-input")).toHaveValue("Later edit"));
    expect(screen.getByText(i18n.t(strings.shot.renderUnsaved, { revision: "2" }))).toBeInTheDocument();
    expect(screen.getByLabelText(i18n.t(strings.shot.carrier))).toHaveValue("");
  });

  it("puts the selected shot stage before its collapsed management controls", async () => {
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head: revision(1n) } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [revision(1n)], nextPageToken: "" } as never);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    const stage = await screen.findByRole("heading", { name: i18n.t(strings.shot.scenePlate) });
    const management = screen.getByText(new RegExp(`^${i18n.t(strings.shot.savedShots)}`), { selector: "summary" });
    expect(management.closest("details")).not.toHaveAttribute("open");
    expect(stage.compareDocumentPosition(management) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("preserves the local draft through a typed revision conflict and explicit head reload", async () => {
    const user = userEvent.setup();
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.getShot)
      .mockResolvedValueOnce({ shot: shot(1n), head: revision(1n) } as never)
      .mockResolvedValueOnce({ shot: shot(2n, "Remote"), head: revision(2n, "Remote") } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [revision(1n)], nextPageToken: "" } as never);
    const conflict = new ConnectError("shot changed", Code.Aborted, undefined, [{ desc: ShotConflictSchema, value: { shotId: "shot-1", expectedRevision: 1n, currentRevision: 2n } }]);
    vi.mocked(shotsClient.saveShot).mockRejectedValueOnce(conflict).mockResolvedValueOnce({ shot: shot(3n, "My draft"), revision: revision(3n, "My draft") } as never);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.click(await screen.findByRole("button", { name: /Daylight/ }));
    const name = await screen.findByTestId("shot-name-input");
    await user.clear(name);
    await user.type(name, "My draft");
    await user.click(screen.getByRole("button", { name: "Save shot" }));
    expect(await screen.findByText(/changed elsewhere/)).toBeInTheDocument();
    expect(name).toHaveValue("My draft");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.reloadKeepDraft) }));
    await waitFor(() => expect(name).toHaveValue("My draft"));
    await user.click(screen.getByRole("button", { name: "Save shot" }));
    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenLastCalledWith(expect.objectContaining({ shotId: "shot-1", expectedRevision: 2n, name: "My draft" })));
    expect(name).toHaveValue("My draft");
  });

  it("shows typed validation issues without discarding draft edits", async () => {
    const user = userEvent.setup();
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head: revision(1n) } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [revision(1n)], nextPageToken: "" } as never);
    vi.mocked(shotsClient.validateShot).mockResolvedValue({ renderEligible: false, issues: [{ code: ShotIssueCode.DRAFT_INCOMPLETE, fieldPath: "scene.plate_media_id", summary: i18n.t(strings.shot.choosePlate), repair: "Import or select an image.", blocksRender: true }] } as never);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.click(await screen.findByRole("button", { name: /Daylight/ }));
    const name = await screen.findByTestId("shot-name-input");
    await user.clear(name);
    await user.type(name, "Unsaved study");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.checkDraft) }));
    expect(await screen.findByText(i18n.t(strings.shot.choosePlate), { selector: ".shot-issues span" })).toBeInTheDocument();
    expect(name).toHaveValue("Unsaved study");
  });

  it("keeps a draft pinned to its original base revision when navigating between saved shots", async () => {
    const user = userEvent.setup();
    const second = create(ShotSchema, { id: "shot-2", projectId: "project-1", name: "Evening", headRevision: 1n });
    vi.mocked(shotsClient.listShots).mockResolvedValue({ shots: [shot(1n), second], nextPageToken: "" } as never);
    let daylightReads = 0;
    vi.mocked(shotsClient.getShot).mockImplementation(({ id }) => Promise.resolve(id === "shot-2" ? { shot: second, head: create(ShotRevisionSchema, { shotId: "shot-2", revision: 1n, name: "Evening", snapshot, issues: [] }) } as never : daylightReads++ === 0 ? { shot: shot(1n), head: revision(1n) } as never : { shot: shot(2n, "Remote daylight"), head: revision(2n, "Remote daylight") } as never));
    vi.mocked(shotsClient.getShotRevision).mockResolvedValue({ revision: revision(1n) } as never);
    vi.mocked(shotsClient.listShotRevisions).mockImplementation(({ shotId }) => Promise.resolve({ revisions: [shotId === "shot-2" ? create(ShotRevisionSchema, { shotId: "shot-2", revision: 1n, name: "Evening", snapshot, issues: [] }) : revision(1n)], nextPageToken: "" }) as never);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.click(await screen.findByRole("button", { name: /Daylight/ }));
    const name = await screen.findByTestId("shot-name-input");
    await user.clear(name);
    await user.type(name, "Unfinished daylight");
    await user.click(screen.getByRole("button", { name: /Evening/ }));
    await waitFor(() => expect(screen.getByTestId("shot-name-input")).toHaveValue("Evening"));
    await user.click(screen.getByRole("button", { name: /Daylight/ }));
    await waitFor(() => expect(screen.getByTestId("shot-name-input")).toHaveValue("Unfinished daylight"));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.save) }));
    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenCalledWith(expect.objectContaining({ expectedRevision: 1n, name: "Unfinished daylight" })));
  });

  it("surfaces retry for a saved-revision fetch failure without rebasing to a newer head", async () => {
    const initial = revision(1n);
    const newer = revision(2n, "Remote daylight");
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head: initial } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [initial], nextPageToken: "" } as never);
    vi.mocked(shotsClient.getShotRevision).mockRejectedValueOnce(new Error("pinned read unavailable")).mockResolvedValueOnce({ revision: initial } as never);
    const view = renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await screen.findByTestId("shot-name-input");
    view.queryClient.setQueryData(["shot", "shot-1"], { shot: shot(2n, "Remote daylight"), head: newer });
    await screen.findByText(/pinned read unavailable/);
    await userEvent.setup().click(screen.getByRole("button", { name: i18n.t(strings.media.retry) }));
    expect(await screen.findByText(i18n.t(strings.shot.renderSaved, { revision: "1" }))).toBeInTheDocument();
    expect(shotsClient.getShotRevision).toHaveBeenLastCalledWith({ shotId: "shot-1", revision: 1n });
  });

  it("shows historical revisions read-only without replacing the current edit buffer", async () => {
    const user = userEvent.setup();
    const head = revision(2n, "Current title");
    const old = revision(1n, "Historical title");
    vi.mocked(shotsClient.listShots).mockResolvedValue({ shots: [shot(2n, "Current title")], nextPageToken: "" } as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(2n, "Current title"), head } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [head, old], nextPageToken: "" } as never);
    vi.mocked(shotsClient.getShotRevision).mockResolvedValue({ revision: old } as never);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.click(await screen.findByRole("button", { name: /Current title/ }));
    const name = await screen.findByTestId("shot-name-input");
    await user.clear(name);
    await user.type(name, "Unsaved current edit");
    await openShotHistory(user);
    const pin = screen.getByLabelText(i18n.t(strings.shot.pinnedRevision));
    await user.selectOptions(pin, "1");
    await waitFor(() => expect(screen.getByTestId("shot-name-input")).toHaveValue("Historical title"));
    const historicalName = screen.getByTestId("shot-name-input");
    expect(historicalName).toBeDisabled();
    expect(await screen.findByText(i18n.t(strings.shot.historicalReadonly))).toBeInTheDocument();
    const currentPin = screen.getByLabelText(i18n.t(strings.shot.pinnedRevision));
    await user.selectOptions(currentPin, "2");
    await waitFor(() => expect(screen.getByLabelText(i18n.t(strings.shot.pinnedRevision))).toHaveValue("2"));
    await waitFor(() => expect(screen.getByTestId("shot-name-input")).toHaveValue("Unsaved current edit"));
    const currentName = screen.getByTestId("shot-name-input");
    expect(currentName).not.toBeDisabled();
    expect(shotsClient.saveShot).not.toHaveBeenCalled();
  });

  it("shows camera duration, matte, and clip-limit state from the selected historical revision", async () => {
    const user = userEvent.setup();
    const timing = (end: bigint, rate: number) => create(VideoTimingSchema, {
      sourceMediaId: "clip", timeBaseNum: 1n, timeBaseDen: BigInt(rate),
      streamStartPts: 0n, sourceEndPts: end, lastFrameDurationTicks: 1n,
      trimStartPts: 0n, trimEndPts: end, playbackRateNum: 1, playbackRateDen: 1,
      outputRateNum: rate, outputRateDen: 1,
    });
    const oldSnapshot = create(ShotSnapshotSchema, {
      motionMode: MotionMode.CROP_PUSH, videoTiming: timing(48n, 24),
      cameraMotion: create(PhotographicCameraMotionSchema, { profileId: "screen-entry-projective-contain-v1", profileVersion: 1, durationOutputFrames: 12, matteRgb: { red: 18, green: 52, blue: 86 }, targetRect: {x:4000,y:1000,width:5500,height:8000} }),
    });
    const currentSnapshot = create(ShotSnapshotSchema, {
      motionMode: MotionMode.CROP_PUSH, videoTiming: timing(400n, 30),
      cameraMotion: create(PhotographicCameraMotionSchema, { profileId: "screen-entry-projective-contain-v1", profileVersion: 1, durationOutputFrames: 75, matteRgb: { red: 171, green: 205, blue: 239 } }),
    });
    const head = revision(2n, "Current title", currentSnapshot);
    const old = revision(1n, "Historical title", oldSnapshot);
    vi.mocked(shotsClient.listShots).mockResolvedValue({ shots: [shot(2n, "Current title")], nextPageToken: "" } as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(2n, "Current title"), head } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [head, old], nextPageToken: "" } as never);
    vi.mocked(shotsClient.getShotRevision).mockResolvedValue({ revision: old } as never);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.click(await screen.findByRole("button", { name: /Current title/ }));
    expect(await screen.findByText(i18n.t(strings.shot.cameraDurationSummary, { frames: "75", seconds: "2.50", rate: "30" }))).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.shot.cameraClipTooLong))).toBeInTheDocument();
    expect(screen.getByLabelText(i18n.t(strings.shot.cameraDurationFrames))).toHaveAttribute("max", "360");
    await openShotHistory(user);
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.shot.pinnedRevision)), "1");
    expect(await screen.findByText(i18n.t(strings.shot.cameraDurationSummary, { frames: "12", seconds: "0.50", rate: "24" }))).toBeInTheDocument();
    expect(screen.queryByText(i18n.t(strings.shot.cameraClipTooLong))).not.toBeInTheDocument();
    expect(screen.getByLabelText(i18n.t(strings.shot.cameraDurationFrames))).toHaveValue(12);
    expect(screen.getByLabelText(i18n.t(strings.shot.cameraMatte))).toHaveValue("#123456");
    expect(screen.getByLabelText(i18n.t(strings.shot.cameraTargetX))).toHaveValue(40);
    expect(screen.getByLabelText(i18n.t(strings.shot.cameraTargetX))).toBeDisabled();
  });

  it("lets a user retry a failed historical revision read or return to the current edit", async () => {
    const user = userEvent.setup();
    const head = revision(2n, "Current title");
    const old = revision(1n, "Historical title");
    vi.mocked(shotsClient.listShots).mockResolvedValue({ shots: [shot(2n, "Current title")], nextPageToken: "" } as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(2n, "Current title"), head } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [head, old], nextPageToken: "" } as never);
    vi.mocked(shotsClient.getShotRevision).mockRejectedValueOnce(new Error("revision temporarily unavailable")).mockResolvedValueOnce({ revision: old } as never);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.click(await screen.findByRole("button", { name: /Current title/ }));
    await openShotHistory(user);
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.shot.pinnedRevision)), "1");
    const failure = await screen.findByRole("alert");
    expect(failure).toHaveTextContent("revision temporarily unavailable");
    await user.click(within(failure).getByRole("button", { name: i18n.t(strings.media.retry) }));
    await waitFor(() => expect(screen.getByTestId("shot-name-input")).toHaveValue("Historical title"));
    expect(screen.getByTestId("shot-name-input")).toBeDisabled();
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.shot.pinnedRevision)), "2");
    await waitFor(() => expect(screen.getByTestId("shot-name-input")).toHaveValue("Current title"));
    expect(screen.getByTestId("shot-name-input")).not.toBeDisabled();
  });

  it("labels media choices by filename and overlays the quad in plate raster pixels", async () => {
    const user = userEvent.setup();
    const source = create(SourceMediaSchema, { id: "opaque-media-id", projectId: "project-1", sha256: "8f8cbb7dcf46e0bc7d53265749a6c17d116093a6ba95e442764060c76fd4a86c", sizeBytes: 3n, mediaType: "image/png", width: 1200, height: 800, kind: SourceMediaKind.IMAGE, readiness: MediaReadiness.READY, originalFilename: "travertine.png" });
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head: revision(1n) } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [revision(1n)], nextPageToken: "" } as never);
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [source], nextPageToken: "" } as never);
    vi.mocked(getSourceMediaContent).mockResolvedValue(new Blob(["png"], { type: "image/png" }));
    const testURL = class extends URL { static createObjectURL = vi.fn(() => "blob:scene-plate"); static revokeObjectURL = vi.fn(); };
    vi.stubGlobal("URL", testURL);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.click(await screen.findByRole("button", { name: /Daylight/ }));
    const plateSelect = await screen.findByLabelText(i18n.t(strings.shot.scenePlate));
    await user.selectOptions(plateSelect, "opaque-media-id");
    expect(await screen.findByRole("img", { name: i18n.t(strings.shot.plateAlt, { name: "travertine.png" }) })).toHaveAttribute("src", "blob:scene-plate");
    expect(plateSelect).toHaveTextContent("travertine.png");
    expect(plateSelect.querySelector('option[value="opaque-media-id"]')).toHaveTextContent("travertine.png");
    expect(plateSelect).not.toHaveTextContent("opaque-media-id");
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.shot.coordinateConvention)), "1");
    await user.type(screen.getByTestId("plate-x-topLeft"), "0");
    await user.type(screen.getByTestId("plate-y-topLeft"), "0");
    await user.type(screen.getByTestId("plate-x-topRight"), "1080");
    await user.type(screen.getByTestId("plate-y-topRight"), "80");
    await user.type(screen.getByTestId("plate-x-bottomRight"), "1080");
    await user.type(screen.getByTestId("plate-y-bottomRight"), "720");
    await user.type(screen.getByTestId("plate-x-bottomLeft"), "120");
    await user.type(screen.getByTestId("plate-y-bottomLeft"), "720");
    const stage = screen.getByTestId("shot-plate-preview");
    expect(stage.querySelector("svg")).toHaveAttribute("viewBox", "0 0 1200 800");
    expect(stage.querySelector("polygon")).toHaveAttribute("points", "0,0 1080,80 1080,720 120,720");
    fireEvent.change(screen.getByTestId("plate-x-topLeft"), { target: { value: "1e300" } });
    expect(screen.getByRole("button", { name: i18n.t(strings.shot.save) })).toBeDisabled();
  });

  it("resolves and previews saved media even when it is outside the first media page", async () => {
    const user = userEvent.setup();
    const source = create(SourceMediaSchema, { id: "older-media-id", projectId: "project-1", sha256: "8f8cbb7dcf46e0bc7d53265749a6c17d116093a6ba95e442764060c76fd4a86c", sizeBytes: 3n, mediaType: "image/png", width: 1200, height: 800, kind: SourceMediaKind.IMAGE, readiness: MediaReadiness.READY, originalFilename: "older-plate.png" });
    const savedSnapshot = create(ShotSnapshotSchema, { scene: create(PhotographicSceneSchema, { plateMediaId: source.id }), presentation: create(ScreenPresentationSchema, { sourceMediaId: source.id }) });
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head: create(ShotRevisionSchema, { shotId: "shot-1", revision: 1n, name: "Daylight", snapshot: savedSnapshot, issues: [] }) } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [revision(1n)], nextPageToken: "" } as never);
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMedia).mockResolvedValue({ sourceMedia: source } as never);
    vi.mocked(getSourceMediaContent).mockResolvedValue(new Blob(["png"], { type: "image/png" }));
    const testURL = class extends URL { static createObjectURL = vi.fn(() => "blob:older-source"); static revokeObjectURL = vi.fn(); };
    vi.stubGlobal("URL", testURL);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.click(await screen.findByRole("button", { name: /Daylight/ }));
    expect(await screen.findByRole("img", { name: i18n.t(strings.shot.plateAlt, { name: "older-plate.png" }) })).toHaveAttribute("src", "blob:older-source");
    expect(await screen.findByTestId("shot-screen-preview")).toHaveAttribute("alt", i18n.t(strings.shot.screenSourceAlt, { name: "older-plate.png" }));
    expect(projectsClient.getSourceMedia).toHaveBeenCalledWith({ projectId: "project-1", id: source.id });
  });

  it("preserves the selected saved composition and render action when photo generation is unavailable", async () => {
    const user = userEvent.setup();
    const source = create(SourceMediaSchema, { id: "older-media-id", projectId: "project-1", sha256: "8f8cbb7dcf46e0bc7d53265749a6c17d116093a6ba95e442764060c76fd4a86c", sizeBytes: 3n, mediaType: "image/png", width: 1200, height: 800, kind: SourceMediaKind.IMAGE, readiness: MediaReadiness.READY, originalFilename: "older-plate.png" });
    const savedSnapshot = create(ShotSnapshotSchema, { scene: create(PhotographicSceneSchema, { plateMediaId: source.id }), presentation: create(ScreenPresentationSchema, { sourceMediaId: source.id }) });
    const savedRevision = create(ShotRevisionSchema, { shotId: "shot-1", revision: 1n, name: "Daylight", snapshot: savedSnapshot, renderEligible: true, issues: [] });
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head: savedRevision } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [savedRevision], nextPageToken: "" } as never);
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMedia).mockResolvedValue({ sourceMedia: source } as never);
    vi.mocked(getSourceMediaContent).mockResolvedValue(new Blob(["png"], { type: "image/png" }));
    vi.mocked(submitPhotographicScene).mockRejectedValueOnce(new Error("Image Tools unavailable"));
    vi.stubGlobal("URL", { ...URL, createObjectURL: vi.fn(() => "blob:older-source"), revokeObjectURL: vi.fn() });

    renderWithProviders(<><ShotWorkspace projectId="project-1" /><PhotographicScenePanel projectId="project-1" /></>);
    await openShotManagement(user);
    await user.click(await screen.findByRole("button", { name: /Daylight/ }));
    const plateSelect = await screen.findByLabelText(i18n.t(strings.shot.scenePlate));
    expect(plateSelect).toHaveValue(source.id);
    expect(await screen.findByRole("img", { name: i18n.t(strings.shot.plateAlt, { name: "older-plate.png" }) })).toHaveAttribute("src", "blob:older-source");
    expect(await screen.findByTestId("shot-screen-preview")).toHaveAttribute("alt", i18n.t(strings.shot.screenSourceAlt, { name: "older-plate.png" }));

    await user.type(await screen.findByLabelText("Describe the scene"), "A sunlit reading room");
    await user.click(screen.getByRole("button", { name: "Generate one local image" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("submit outcome is unknown");
    expect(plateSelect).toHaveValue(source.id);
    expect(screen.getByRole("button", { name: i18n.t(strings.shot.renderRevisionAction, { revision: "1" }) })).toBeEnabled();
    expect(shotsClient.saveShot).not.toHaveBeenCalled();
    expect(projectsClient.getSourceMedia).toHaveBeenCalledTimes(1);
  });

  it("does not offer pending video or dimensionless media to image-only plate and source previews", async () => {
    const user = userEvent.setup();
    const video = create(SourceMediaSchema, { id: "pending-video", projectId: "project-1", sha256: "8f8cbb7dcf46e0bc7d53265749a6c17d116093a6ba95e442764060c76fd4a86c", sizeBytes: 512n << 20n, mediaType: "video/mp4", kind: SourceMediaKind.VIDEO, readiness: MediaReadiness.PENDING, originalFilename: "large-recording.mp4" });
    const incompleteVideo = create(SourceMediaSchema, { id: "incomplete-video", projectId: "project-1", sha256: "7f8cbb7dcf46e0bc7d53265749a6c17d116093a6ba95e442764060c76fd4a86c", sizeBytes: 100n, mediaType: "video/mp4", kind: SourceMediaKind.VIDEO, readiness: MediaReadiness.READY, originalFilename: "incomplete.mp4", inspectionReceipt: create(MediaInspectionReceiptSchema, { classification: "accepted", clockSha256: "b".repeat(64) }) });
    const unknownImage = create(SourceMediaSchema, { id: "unknown-raster", projectId: "project-1", sha256: "9f8cbb7dcf46e0bc7d53265749a6c17d116093a6ba95e442764060c76fd4a86c", sizeBytes: 3n, mediaType: "image/png", kind: SourceMediaKind.IMAGE, readiness: MediaReadiness.READY, originalFilename: "unknown-raster.png" });
    const savedSnapshot = create(ShotSnapshotSchema, { scene: create(PhotographicSceneSchema, { plateMediaId: video.id }), presentation: create(ScreenPresentationSchema, { sourceMediaId: video.id }) });
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head: create(ShotRevisionSchema, { shotId: "shot-1", revision: 1n, name: "Daylight", snapshot: savedSnapshot, issues: [] }) } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [revision(1n)], nextPageToken: "" } as never);
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [video, incompleteVideo, unknownImage], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMedia).mockResolvedValue({ sourceMedia: video } as never);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.click(await screen.findByRole("button", { name: /Daylight/ }));
    const plateSelect = screen.getByLabelText(i18n.t(strings.shot.scenePlate));
    expect(plateSelect).not.toHaveTextContent("large-recording.mp4");
    expect(plateSelect).not.toHaveTextContent("incomplete.mp4");
    expect(plateSelect).not.toHaveTextContent("unknown-raster.png");
    const screenSelect = screen.getByLabelText(i18n.t(strings.shot.screenSource));
    expect(screenSelect).not.toHaveTextContent("large-recording.mp4");
    expect(screenSelect).not.toHaveTextContent("incomplete.mp4");
    expect(getSourceMediaContent).not.toHaveBeenCalled();
    expect(screen.queryByTestId("shot-plate-preview")).not.toBeInTheDocument();
  });

  it("opens only accepted recording previews as routed streams and pins trim controls to the inspected clock", async () => {
    const user = userEvent.setup();
    const video = create(SourceMediaSchema, {
      id: "accepted-video", projectId: "project-1", sha256: "a".repeat(64), sizeBytes: 512n << 20n,
      mediaType: "video/mp4", kind: SourceMediaKind.VIDEO, readiness: MediaReadiness.READY, originalFilename: "field-recording.mp4",
      inspectionReceipt: create(MediaInspectionReceiptSchema, {
        id: "receipt-1", sha256: "c".repeat(64), classification: "accepted", clockSha256: "b".repeat(64), clockArtifactRef: "clock-1", clockArtifactSizeBytes: 32n,
        sourceSha256: "a".repeat(64), sourceSizeBytes: 512n << 20n, jobId: "inspect-job", attemptId: "inspect-attempt", videoStreamIndex: 0,
        timeBaseNumerator: 1n, timeBaseDenominator: 24n, streamStartPts: 0n, sourceEndPts: 48n, lastFrameDurationTicks: 1n,
      }),
    });
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head: revision(1n) } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [revision(1n)], nextPageToken: "" } as never);
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [video], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMediaClockPage).mockResolvedValue(create(GetSourceMediaClockPageResponseSchema, {
      clockSha256: "b".repeat(64), timeBaseNumerator: 1n, timeBaseDenominator: 24n,
      points: [create(SourceClockPointSchema, { ordinal: 0, pts: 0n }), create(SourceClockPointSchema, { ordinal: 1, pts: 24n })],
    }));
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.click(await screen.findByRole("button", { name: /Daylight/ }));
    await user.selectOptions(await screen.findByLabelText(i18n.t(strings.shot.screenSource)), "accepted-video");
    expect(await screen.findByTestId("shot-video-timing")).toBeInTheDocument();
    expect(screen.getByTestId("shot-video-source-preview")).toHaveAttribute("src", sourceMediaContentURL("project-1", "accepted-video"));
    expect(screen.getByTestId("shot-video-source-preview")).toHaveAttribute("preload", "metadata");
    fireEvent.error(screen.getByTestId("shot-video-source-preview"));
    expect(await screen.findByText(i18n.t(strings.media.videoPreviewError))).toBeInTheDocument();
    expect(projectsClient.getSourceMediaClockPage).toHaveBeenCalledWith(expect.objectContaining({ mediaId: "accepted-video", expectedClockSha256: "b".repeat(64), pageSize: 256 }));
    expect(getSourceMediaContent).not.toHaveBeenCalled();
  });

  it("saves screen entry frames and matte separately from source timing and invalidates the prior sample", async () => {
    const user = userEvent.setup();
    const timing = create(VideoTimingSchema, {
      sourceMediaId: "accepted-video", sourceSha256: "a".repeat(64), sourceSizeBytes: 500n,
      inspectionJobId: "inspect-job", inspectionAttemptId: "inspect-attempt", inspectionReceiptId: "receipt-1", inspectionReceiptSha256: "c".repeat(64),
      clockArtifactId: "clock-1", clockSha256: "b".repeat(64), clockSizeBytes: 32n, selectedVideoStream: 0,
      timeBaseNum: 1n, timeBaseDen: 24n, streamStartPts: 0n, sourceEndPts: 312n, lastFrameDurationTicks: 1n,
      trimStartPts: 0n, trimEndPts: 312n, playbackRateNum: 1, playbackRateDen: 1, outputRateNum: 24, outputRateDen: 1,
      eofPolicy: VideoEOFPolicy.VIDEO_EOF_POLICY_REJECT, videoCompositionProfileId: "device-studio-fixed-plate-video-v1", videoCompositionProfileVersion: 1,
      sampleBaseRecipeVersion: 1,
      sampleBinding: create(VideoSampleBindingSchema, { jobId: "old-sample", sourceSha256: "a".repeat(64), clockSha256: "b".repeat(64), trimStartPts: 0n, trimEndPts: 312n, baseShotRevision: 1n, baseRecipeVersion: "device-studio-video-recipe-v1" }),
    });
    const savedSnapshot = create(ShotSnapshotSchema, {
      motionMode: MotionMode.FIXED, videoTiming: timing,
      presentation: create(ScreenPresentationSchema, { sourceMediaId: "accepted-video", kind: PresentationKind.PLAIN }),
    });
    const head = revision(1n, "Daylight", savedSnapshot);
    const video = create(SourceMediaSchema, {
      id: "accepted-video", projectId: "project-1", sha256: "a".repeat(64), sizeBytes: 500n, mediaType: "video/mp4", kind: SourceMediaKind.VIDEO, readiness: MediaReadiness.READY, originalFilename: "field-recording.mp4",
      inspectionReceipt: create(MediaInspectionReceiptSchema, { id: "receipt-1", sha256: "c".repeat(64), classification: "accepted", clockSha256: "b".repeat(64), clockArtifactRef: "clock-1", clockArtifactSizeBytes: 32n, sourceSha256: "a".repeat(64), sourceSizeBytes: 500n, jobId: "inspect-job", attemptId: "inspect-attempt", videoStreamIndex: 0, timeBaseNumerator: 1n, timeBaseDenominator: 24n, streamStartPts: 0n, sourceEndPts: 312n, lastFrameDurationTicks: 1n }),
    });
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [head], nextPageToken: "" } as never);
    vi.mocked(shotsClient.saveShot).mockImplementation((request) => {
      const current = vi.mocked(shotsClient.saveShot).mock.calls.length;
      const savedRevision = create(ShotRevisionSchema, {
        shotId: "shot-1", revision: BigInt(current + 1), name: request.name, snapshot: request.snapshot,
        renderEligible: current > 1, issues: [],
      });
      return Promise.resolve({ shot: shot(savedRevision.revision, request.name), revision: savedRevision } as never);
    });
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [video], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMediaClockPage).mockResolvedValue(create(GetSourceMediaClockPageResponseSchema, {
      clockSha256: "b".repeat(64), timeBaseNumerator: 1n, timeBaseDenominator: 24n,
      points: [create(SourceClockPointSchema, { ordinal: 0, pts: 0n }), create(SourceClockPointSchema, { ordinal: 1, pts: 48n }), create(SourceClockPointSchema, { ordinal: 2, pts: 312n })],
    }));
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await screen.findByTestId("shot-editor");
    const mode = screen.getByLabelText(i18n.t(strings.shot.motionMode));
    await user.selectOptions(mode, String(MotionMode.CROP_PUSH));
    expect(screen.getByText(i18n.t(strings.shot.cameraClipTooLong))).toBeInTheDocument();
    expect(screen.getByRole("button", { name: i18n.t(strings.shot.save) })).toBeEnabled();
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.save) }));
    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenCalledTimes(1));
    expect(screen.getByText(i18n.t(strings.shot.renderBlocked))).toBeInTheDocument();
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.shot.videoTrimEnd)), "48");
    expect(screen.queryByText(i18n.t(strings.shot.cameraClipTooLong))).not.toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.shot.cameraDurationSummary, { frames: "24", seconds: "1.00", rate: "24" }))).toHaveAttribute("role", "status");
    const durationInput = screen.getByLabelText(i18n.t(strings.shot.cameraDurationFrames));
    await user.clear(durationInput);
    await user.type(durationInput, "49");
    expect(screen.getByText(i18n.t(strings.shot.cameraDurationInvalid))).toBeInTheDocument();
    expect(screen.getByRole("button", { name: i18n.t(strings.shot.save) })).toBeDisabled();
    await user.clear(durationInput);
    await user.type(durationInput, "12");
    const matteInput = screen.getByLabelText(i18n.t(strings.shot.cameraMatte));
    await user.clear(matteInput);
    expect(screen.getByText(i18n.t(strings.shot.cameraMatteInvalid))).toBeInTheDocument();
    expect(screen.getByRole("button", { name: i18n.t(strings.shot.save) })).toBeDisabled();
    await user.type(matteInput, "#123456");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.save) }));
    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenCalledTimes(2));
    const saved = vi.mocked(shotsClient.saveShot).mock.calls[1]?.[0].snapshot;
    expect(saved?.motionMode).toBe(MotionMode.CROP_PUSH);
    expect(saved?.cameraMotion).toEqual(create(PhotographicCameraMotionSchema, { profileId: "screen-entry-stationary-scene-v1", profileVersion: 1, durationOutputFrames: 12, matteRgb: { red: 18, green: 52, blue: 86 } }));
    expect(saved?.videoTiming?.trimStartPts).toBe(0n);
    expect(saved?.videoTiming?.trimEndPts).toBe(48n);
    expect(saved?.videoTiming?.sampleBinding).toBeUndefined();
    await user.click(screen.getByLabelText(i18n.t(strings.shot.cameraTargetEnabled)));
    const targetX = screen.getByLabelText(i18n.t(strings.shot.cameraTargetX));
    await user.clear(targetX); await user.type(targetX, "40");
    expect(screen.getByRole("button", { name: i18n.t(strings.shot.save) })).toBeDisabled();
    const targetWidth = screen.getByLabelText(i18n.t(strings.shot.cameraTargetWidth));
    await user.clear(targetWidth); await user.type(targetWidth, "55");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.save) }));
    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenCalledTimes(3));
    expect(vi.mocked(shotsClient.saveShot).mock.calls[2]?.[0].snapshot?.cameraMotion?.targetRect).toMatchObject({x:4000,y:0,width:5500,height:10000});
    expect(screen.getByLabelText(i18n.t(strings.shot.cameraTargetX))).toHaveValue(40);
    expect(screen.getByLabelText(i18n.t(strings.shot.cameraTargetWidth))).toHaveValue(55);
    await user.click(screen.getByLabelText(i18n.t(strings.shot.cameraTargetEnabled)));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.save) }));
    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenCalledTimes(4));
    expect(vi.mocked(shotsClient.saveShot).mock.calls[3]?.[0].snapshot?.cameraMotion?.targetRect).toBeUndefined();
    await user.click(screen.getByLabelText(i18n.t(strings.shot.cameraSceneEnabled)));
    const sceneX=screen.getByLabelText(i18n.t(strings.shot.cameraSceneX));
    await user.clear(sceneX); await user.type(sceneX,"35");
    expect(screen.getByRole("button",{name:i18n.t(strings.shot.save)})).toBeDisabled();
    const sceneWidth=screen.getByLabelText(i18n.t(strings.shot.cameraSceneWidth));
    await user.clear(sceneWidth); await user.type(sceneWidth,"60");
    await user.click(screen.getByRole("button",{name:i18n.t(strings.shot.save)}));
    await waitFor(()=>expect(shotsClient.saveShot).toHaveBeenCalledTimes(5));
    expect(vi.mocked(shotsClient.saveShot).mock.calls[4]?.[0].snapshot?.cameraMotion?.sceneRect).toMatchObject({x:3500,y:0,width:6000,height:10000});
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.shot.cameraProfile)),"screen-entry-projective-contain-v1");
    await user.click(screen.getByRole("button",{name:i18n.t(strings.shot.save)}));
    await waitFor(()=>expect(shotsClient.saveShot).toHaveBeenCalledTimes(6));
    expect(vi.mocked(shotsClient.saveShot).mock.calls[5]?.[0].snapshot?.cameraMotion?.sceneRect).toBeUndefined();

  });

  it("omits camera configuration when a shot is fixed", async () => {
    const user = userEvent.setup();
    vi.mocked(shotsClient.listShots).mockResolvedValue({ shots: [], nextPageToken: "" } as never);
    vi.mocked(shotsClient.createShot).mockResolvedValue({ shot: shot(1n, "Fixed"), revision: revision(1n, "Fixed") } as never);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.type(await screen.findByTestId("new-shot-name"), "Fixed");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.create) }));
    await screen.findByTestId("shot-editor");
    expect(vi.mocked(shotsClient.createShot).mock.calls[0]?.[0].snapshot?.cameraMotion).toBeUndefined();
  });

  it("accepts a verified AUTO-phone sample only by saving a new immutable shot revision", async () => {
    const user = userEvent.setup();
    const timing = create(VideoTimingSchema, {
      sourceMediaId: "accepted-video", sourceSha256: "a".repeat(64), sourceSizeBytes: 500n,
      inspectionJobId: "inspect-job", inspectionAttemptId: "inspect-attempt", inspectionReceiptId: "receipt-1", inspectionReceiptSha256: "c".repeat(64),
      clockArtifactId: "clock-1", clockSha256: "b".repeat(64), clockSizeBytes: 32n, selectedVideoStream: 0,
      timeBaseNum: 1n, timeBaseDen: 24n, streamStartPts: 0n, sourceEndPts: 48n, lastFrameDurationTicks: 1n,
      trimStartPts: 0n, trimEndPts: 48n, playbackRateNum: 1, playbackRateDen: 1, outputRateNum: 24, outputRateDen: 1,
      eofPolicy: VideoEOFPolicy.VIDEO_EOF_POLICY_REJECT, videoCompositionProfileId: "device-studio-fixed-plate-video-v1", videoCompositionProfileVersion: 1,
    });
    const savedSnapshot = create(ShotSnapshotSchema, {
      presentation: create(ScreenPresentationSchema, { sourceMediaId: "accepted-video", kind: PresentationKind.PHONE_SYSTEM, phoneSystemChrome: create(PhoneSystemChromeSchema, { appearance: PresentationAppearance.AUTO }) }),
      videoTiming: timing,
    });
    const head = create(ShotRevisionSchema, { shotId: "shot-1", revision: 1n, name: "Daylight", snapshot: savedSnapshot, renderEligible: true, issues: [] });
    const video = create(SourceMediaSchema, {
      id: "accepted-video", projectId: "project-1", sha256: "a".repeat(64), sizeBytes: 500n, mediaType: "video/mp4", kind: SourceMediaKind.VIDEO, readiness: MediaReadiness.READY, originalFilename: "field-recording.mp4",
      inspectionReceipt: create(MediaInspectionReceiptSchema, { id: "receipt-1", sha256: "c".repeat(64), classification: "accepted", clockSha256: "b".repeat(64), clockArtifactRef: "clock-1", clockArtifactSizeBytes: 32n, sourceSha256: "a".repeat(64), sourceSizeBytes: 500n, jobId: "inspect-job", attemptId: "inspect-attempt", videoStreamIndex: 0, timeBaseNumerator: 1n, timeBaseDenominator: 24n, streamStartPts: 0n, sourceEndPts: 48n, lastFrameDurationTicks: 1n }),
    });
    const sampleOutput = create(RenderOutputSchema, { mediaType: "image/png", sizeBytes: 3n, widthPixels: 16, heightPixels: 9, sha256: "d".repeat(64), contentUrl: "/api/v1/projects/project-1/renders/sample/content" });
    const sampleJob = create(RenderJobSchema, { id: "sample", projectId: "project-1", shotId: "shot-1", shotRevision: 1n, status: RenderJobStatus.SUCCEEDED, version: 2n, renderMode: RenderMode.VIDEO_SAMPLE_PNG, output: sampleOutput });
    const binding = create(VideoSampleBindingSchema, { jobId: "sample", sourceSha256: "a".repeat(64), clockSha256: "b".repeat(64), trimStartPts: 0n, trimEndPts: 48n, baseShotRevision: 1n, baseRecipeVersion: "device-studio-video-recipe-v1", topFillSourceResolved: true, resolvedTopFillRed: 24, resolvedTopFillGreen: 40, resolvedTopFillBlue: 56, bottomFillSourceResolved: true, resolvedBottomFillRed: 70, resolvedBottomFillGreen: 80, resolvedBottomFillBlue: 90 });
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [head], nextPageToken: "" } as never);
    vi.mocked(shotsClient.saveShot).mockReset().mockImplementation((request) => Promise.resolve({ shot: shot(2n), revision: create(ShotRevisionSchema, { ...head, revision: 2n, snapshot: request.snapshot ? create(ShotSnapshotSchema, request.snapshot) : undefined }) }) as never);
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [video], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMediaClockPage).mockResolvedValue(create(GetSourceMediaClockPageResponseSchema, { clockSha256: "b".repeat(64), timeBaseNumerator: 1n, timeBaseDenominator: 24n }));
    vi.mocked(rendersClient.listRenders).mockResolvedValue({ jobs: [sampleJob], nextPageToken: "" } as never);
    vi.mocked(getVideoSampleBinding).mockResolvedValue(binding);
    vi.mocked(getRenderOutputContent).mockResolvedValue(new Blob(["png"], { type: "image/png" }));
    vi.stubGlobal("URL", { ...URL, createObjectURL: vi.fn(() => "blob:sample"), revokeObjectURL: vi.fn() });
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await screen.findByTestId("shot-editor");
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.shot.renderViewSample) }));
    await user.click(await screen.findByTestId("shot-video-sample-accept"));
    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenCalled());
    expect(getVideoSampleBinding).toHaveBeenCalledWith("project-1", "shot-1", "sample");
    const saveRequest = vi.mocked(shotsClient.saveShot).mock.calls[0]?.[0];
    expect(saveRequest).toMatchObject({ shotId: "shot-1", expectedRevision: 1n });
    expect(saveRequest?.snapshot?.videoTiming?.sampleBinding).toMatchObject({ jobId: "sample", sourceSha256: "a".repeat(64), clockSha256: "b".repeat(64) });
    expect(saveRequest?.snapshot?.presentation?.phoneSystemChrome?.topFill).toMatchObject({ red: 24, green: 40, blue: 56 });
    const topFill = await screen.findByLabelText(i18n.t(strings.shot.topFill));
    await user.clear(topFill);
    await user.type(topFill, "#123456");
    expect(screen.getByLabelText(i18n.t(strings.shot.bottomFill))).toHaveValue("");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.save) }));
    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenCalledTimes(2));
    const edited = vi.mocked(shotsClient.saveShot).mock.calls[1]?.[0].snapshot?.presentation?.phoneSystemChrome;
    expect(edited?.topFill).toMatchObject({ red: 18, green: 52, blue: 86 });
    expect(edited?.bottomFill).toBeUndefined();
  });

  it("loads the next saved-shot page without replacing the selected project", async () => {
    const user = userEvent.setup();
    const second = create(ShotSchema, { id: "shot-2", projectId: "project-1", name: "Evening", headRevision: 1n });
    vi.mocked(shotsClient.listShots).mockResolvedValueOnce({ shots: [shot(1n)], nextPageToken: "shots-2" } as never).mockResolvedValueOnce({ shots: [second], nextPageToken: "" } as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head: revision(1n) } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [revision(1n)], nextPageToken: "" } as never);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.click(await screen.findByRole("button", { name: /Daylight/ }));
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.projects.loadMore) }));
    expect(await screen.findByRole("button", { name: /Evening/ })).toBeInTheDocument();
    expect(shotsClient.listShots).toHaveBeenLastCalledWith({ projectId: "project-1", pageSize: 25, pageToken: "shots-2" });
    expect(screen.getByTestId("shot-editor")).toBeInTheDocument();
  });

  it("saves browser presentation choices as typed snapshot fields", async () => {
    const user = userEvent.setup();
    const source = create(SourceMediaSchema, { id: "screen-1", projectId: "project-1", sha256: "8f8cbb7dcf46e0bc7d53265749a6c17d116093a6ba95e442764060c76fd4a86c", sizeBytes: 3n, mediaType: "image/png", width: 1200, height: 800, kind: SourceMediaKind.IMAGE, readiness: MediaReadiness.READY, originalFilename: "raw-screen.png" });
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head: revision(1n) } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [revision(1n)], nextPageToken: "" } as never);
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [source], nextPageToken: "" } as never);
    vi.mocked(getSourceMediaContent).mockResolvedValue(new Blob(["png"], { type: "image/png" }));
    const testURL = class extends URL { static createObjectURL = vi.fn(() => "blob:screen-source"); static revokeObjectURL = vi.fn(); };
    vi.stubGlobal("URL", testURL);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.click(await screen.findByRole("button", { name: /Daylight/ }));
    await user.click(screen.getByText(i18n.t(strings.shot.screenPresentation)));
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.shot.screenSource)), "screen-1");
    const captureControls = screen.getAllByLabelText(i18n.t(strings.shot.captureArea));
    await user.selectOptions(captureControls[0]!, "1");
    await user.selectOptions(captureControls[1]!, "3");
    await user.click(screen.getByText(i18n.t(strings.shot.browserChrome)));
    await user.type(screen.getByLabelText(i18n.t(strings.shot.windowTitle)), "Daylight in the field");
    await user.type(screen.getByLabelText(i18n.t(strings.shot.addressText)), "app.example.test");
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.shot.appearance)), String(PresentationAppearance.DARK));
    await user.type(screen.getByLabelText(i18n.t(strings.shot.windowX)), "500");
    await user.type(screen.getByLabelText(i18n.t(strings.shot.windowY)), "500");
    await user.type(screen.getByLabelText(i18n.t(strings.shot.windowWidth)), "8000");
    await user.type(screen.getByLabelText(i18n.t(strings.shot.windowHeight)), "7000");
    await user.type(screen.getByLabelText(i18n.t(strings.shot.windowColor)), "#112233");
    await user.click(screen.getByLabelText(i18n.t(strings.shot.menuBar)));
    await user.click(screen.getByLabelText(i18n.t(strings.shot.dock)));
    const toolbar = screen.getByLabelText(i18n.t(strings.shot.toolbar));
    await user.click(toolbar);
    expect(screen.getByText(i18n.t(strings.shot.browserControlsRequireToolbar), { selector: '[role="alert"]' })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: i18n.t(strings.shot.save) })).toBeDisabled();
    await user.click(toolbar);
    await user.click(screen.getByText(i18n.t(strings.shot.advancedPresentation)));
    await user.type(screen.getByLabelText(i18n.t(strings.shot.matteColor)), "#f0e8dd");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.save) }));
    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenCalled());
    const saved = vi.mocked(shotsClient.saveShot).mock.calls[0]?.[0].snapshot?.presentation;
    expect(saved).toMatchObject({ sourceMediaId: "screen-1", captureArea: 1, kind: 3, viewportWidthPixels: 0, viewportHeightPixels: 0, matte: { red: 240, green: 232, blue: 221 }, browserDesktopChrome: { windowTitle: "Daylight in the field", addressText: "app.example.test", appearance: PresentationAppearance.DARK, window: { x: 500, y: 500, width: 8000, height: 7000 }, showMenuBar: true, showDock: true, windowColor: { red: 17, green: 34, blue: 51 } } });
  });

  it("persists phone appearance, status visibility and explicit fills in the saved revision", async () => {
    const user = userEvent.setup();
    vi.mocked(shotsClient.listShots).mockResolvedValue(list() as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: shot(1n), head: revision(1n) } as never);
    vi.mocked(shotsClient.listShotRevisions).mockResolvedValue({ revisions: [revision(1n)], nextPageToken: "" } as never);
    vi.mocked(shotsClient.saveShot).mockResolvedValue({ shot: shot(2n), revision: revision(2n) } as never);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.click(await screen.findByRole("button", { name: /Daylight/ }));
    await user.click(screen.getByText(i18n.t(strings.shot.screenPresentation)));
    const captureControls = screen.getAllByLabelText(i18n.t(strings.shot.captureArea));
    await user.selectOptions(captureControls[0]!, "1");
    await user.selectOptions(captureControls[1]!, "2");
    await user.click(screen.getByText(i18n.t(strings.shot.phoneChrome)));
    const appearance = screen.getByLabelText(i18n.t(strings.shot.appearance));
    expect(appearance).toHaveValue(String(PresentationAppearance.AUTO));
    await user.selectOptions(appearance, String(PresentationAppearance.DARK));
    await user.click(screen.getByLabelText(i18n.t(strings.shot.statusBar)));
    await user.click(screen.getByLabelText(i18n.t(strings.shot.batteryLabel)));
    await user.type(screen.getByLabelText(i18n.t(strings.shot.topFill)), "#112233");
    await user.type(screen.getByLabelText(i18n.t(strings.shot.bottomFill)), "#445566");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.save) }));
    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenCalled());
    const saved = vi.mocked(shotsClient.saveShot).mock.calls[0]?.[0].snapshot?.presentation;
    expect(saved?.phoneSystemChrome).toMatchObject({ appearance: PresentationAppearance.DARK, showStatus: false, showBatteryPercent: true, batteryPercent: 84, topFill: { red: 17, green: 34, blue: 51 }, bottomFill: { red: 68, green: 85, blue: 102 } });
  });

  it("persists Auto contrast for a new phone presentation", async () => {
    const user = userEvent.setup();
    vi.mocked(shotsClient.listShots).mockResolvedValue({ shots: [], nextPageToken: "" } as never);
    let createdSnapshot: ShotSnapshot = snapshot;
    vi.mocked(shotsClient.createShot).mockImplementation((request) => {
      createdSnapshot = create(ShotSnapshotSchema, request.snapshot ?? {});
      return Promise.resolve({ shot: shot(1n, request.name), revision: revision(1n, request.name, createdSnapshot) } as never);
    });
    vi.mocked(shotsClient.getShot).mockImplementation(() => Promise.resolve({ shot: shot(1n, "New phone"), head: revision(1n, "New phone", createdSnapshot) } as never));
    vi.mocked(shotsClient.listShotRevisions).mockImplementation(() => Promise.resolve({ revisions: [revision(1n, "New phone", createdSnapshot)], nextPageToken: "" } as never));
    vi.mocked(shotsClient.saveShot).mockResolvedValue({ shot: shot(2n), revision: revision(2n) } as never);
    renderWithProviders(<ShotWorkspace projectId="project-1" />);
    await openShotManagement(user);
    await user.type(screen.getByTestId("new-shot-name"), "New phone");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.create) }));
    await screen.findByTestId("shot-editor");
    await user.click(screen.getByText(i18n.t(strings.shot.screenPresentation)));
    const [captureArea, presentationKind] = screen.getAllByLabelText(i18n.t(strings.shot.captureArea));
    await user.selectOptions(captureArea!, String(CaptureArea.APP_ONLY));
    await user.selectOptions(presentationKind!, String(PresentationKind.PHONE_SYSTEM));
    await user.click(screen.getByText(i18n.t(strings.shot.phoneChrome)));
    expect(screen.getByLabelText(i18n.t(strings.shot.appearance))).toHaveValue(String(PresentationAppearance.AUTO));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.save) }));
    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenCalled());
    expect(vi.mocked(shotsClient.saveShot).mock.calls[0]?.[0].snapshot?.presentation?.phoneSystemChrome?.appearance).toBe(PresentationAppearance.AUTO);
  });
});
