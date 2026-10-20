import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";
import { create, toJson } from "@bufbuild/protobuf";
import { PhotographicSceneSchema, ScreenPresentationSchema, ShotConflictSchema, ShotIssueCode, ShotRevisionSchema, ShotSchema, ShotSnapshotSchema } from "@vrooli/proto-types/device-studio/v1/shots/shots_pb";
import { SourceMediaSchema } from "@vrooli/proto-types/device-studio/v1/projects/projects_pb";
import { renderWithProviders } from "../../test-utils";
import { i18n, setLocale } from "../../i18n";
import { strings } from "../../consts/strings";
import { ShotWorkspace } from "./ShotWorkspace";

vi.mock("../../api/projects", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/projects")>();
  return { ...actual, projectsClient: { ...actual.projectsClient, listSourceMedia: vi.fn().mockResolvedValue({ sourceMedia: [] }), getSourceMedia: vi.fn() }, getSourceMediaContent: vi.fn() };
});
vi.mock("../../api/shots", () => ({
  shotsClient: {
    listShots: vi.fn(), createShot: vi.fn(), getShot: vi.fn(), getShotRevision: vi.fn(), listShotRevisions: vi.fn(),
    saveShot: vi.fn(), validateShot: vi.fn(),
  },
  shotConflict: (error: unknown) => error instanceof ConnectError ? error.findDetails(ShotConflictSchema)[0] : undefined,
}));
import { shotsClient } from "../../api/shots";
import { getSourceMediaContent, projectsClient } from "../../api/projects";

const snapshot = create(ShotSnapshotSchema, {});
const shot = (revision: bigint, name = "Daylight") => create(ShotSchema, { id: "shot-1", projectId: "project-1", name, headRevision: revision });
const revision = (value: bigint, name = "Daylight") => create(ShotRevisionSchema, { shotId: "shot-1", revision: value, name, snapshot, issues: [] });
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
  beforeEach(async () => { await setLocale("en"); });
  afterEach(() => { cleanup(); vi.clearAllMocks(); vi.unstubAllGlobals(); });

  it("creates and opens a shot using the typed incomplete draft contract", async () => {
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
    const source = create(SourceMediaSchema, { id: "opaque-media-id", projectId: "project-1", sha256: "8f8cbb7dcf46e0bc7d53265749a6c17d116093a6ba95e442764060c76fd4a86c", sizeBytes: 3n, mediaType: "image/png", width: 1200, height: 800, originalFilename: "travertine.png" });
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
    const source = create(SourceMediaSchema, { id: "older-media-id", projectId: "project-1", sha256: "8f8cbb7dcf46e0bc7d53265749a6c17d116093a6ba95e442764060c76fd4a86c", sizeBytes: 3n, mediaType: "image/png", width: 1200, height: 800, originalFilename: "older-plate.png" });
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
    const source = create(SourceMediaSchema, { id: "screen-1", projectId: "project-1", sha256: "8f8cbb7dcf46e0bc7d53265749a6c17d116093a6ba95e442764060c76fd4a86c", sizeBytes: 3n, mediaType: "image/png", width: 1200, height: 800, originalFilename: "raw-screen.png" });
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
    await user.click(screen.getByRole("button", { name: i18n.t(strings.shot.save) }));
    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenCalled());
    const saved = vi.mocked(shotsClient.saveShot).mock.calls[0]?.[0].snapshot?.presentation;
    expect(saved).toMatchObject({ sourceMediaId: "screen-1", captureArea: 1, kind: 3, viewportWidthPixels: 0, viewportHeightPixels: 0, browserDesktopChrome: { windowTitle: "Daylight in the field", addressText: "app.example.test" } });
  });
});
