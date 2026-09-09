import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";
import { Code, ConnectError } from "@connectrpc/connect";
import { renderWithProviders } from "../test-utils";
import { i18n } from "../i18n";
import { DesignPage } from "./DesignPage";

const api = vi.hoisted(() => ({
  acceptCandidate: vi.fn(),
  checkCandidateAcceptance: vi.fn(),
  listDesignPages: vi.fn(),
  listCandidates: vi.fn(),
  listCritiques: vi.fn(),
  getCritique: vi.fn(),
  getCritiqueRubric: vi.fn(),
  getCandidate: vi.fn(),
  renderSketch: vi.fn(),
  renderCandidate: vi.fn(),
  captureCandidate: vi.fn(),
  attachCapture: vi.fn(),
  cancelCapture: vi.fn(),
  retryCapture: vi.fn(),
  getCaptureScreenshot: vi.fn(),
  getCapture: vi.fn(),
  mapCandidateRegions: vi.fn(),
  proposeSketch: vi.fn(),
  putSketch: vi.fn(),
  saveCandidate: vi.fn(),
  importPage: vi.fn(),
  setTemplate: vi.fn(),
  searchDesignAssets: vi.fn(),
  getSketch: vi.fn(),
  getHistory: vi.fn(),
  addNote: vi.fn(),
  verifySketch: vi.fn(),
}));
vi.mock("../api/catalog", () => ({ searchDesignAssets: api.searchDesignAssets }));
vi.mock("../api/sketch", () => ({ sketchClient: api }));
const sketch = {
  regions: [{ id: "body", note: "Main task", elements: [] }],
  placements: [],
  notes: [],
  unplaced: [],
};
function renderPage(path = "/design/demo/page") {
  return renderWithProviders(
    <Routes>
      <Route path="/design" element={<DesignPage />} />
      <Route path="/design/:scenario/:page" element={<DesignPage />} />
    </Routes>,
    { routerEntries: [path] },
  );
}
beforeEach(async () => {
  sessionStorage.clear();
  await i18n.changeLanguage("en");
  api.listCandidates.mockResolvedValue({ candidates: [] });
  api.listCritiques.mockResolvedValue({ reviews: [], nextBeforeId: "" });
  api.getCritiqueRubric.mockResolvedValue({ calibrated: false });
});
afterEach(() => {
  cleanup();
  vi.resetAllMocks();
});
function setup() {
  api.getSketch.mockResolvedValue({ sketch, contentHash: "current-hash" });
  api.getHistory.mockResolvedValue({
    revisions: [{ contentHash: "old-hash", createdAt: "2026-01-01", sketch, current: false }],
  });
}
describe("Design workspace", () => {
  it("reopens an archived candidate without selecting it as the current page", async () => {
    setup();
    const candidate = { scenario: "demo", designId: "design-one", hash: "saved-revision" };
    api.listCandidates.mockResolvedValue({ candidates: [{ candidate, parentHash: "parent-revision", templateAsset: "templates.page", templateVersion: "1.0.0" }] });
    api.getCandidate.mockResolvedValue({ candidate, page: "page", baseHash: "older-base", parentHash: "parent-revision", sketch, declaredRegions: [], refinement: { round: 1, budget: 3, reason: "Clarify recovery", changedRegions: ["body"] } });
    renderPage();
    fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.openCandidate", { design: "design-one", revision: "saved-revision".slice(0, 12) }) }));
    await waitFor(() => expect(api.getCandidate).toHaveBeenCalledWith(candidate));
    await waitFor(() => expect(document.querySelector('[data-candidate-hash="saved-revision"]')).toBeTruthy());
    expect(await screen.findByText("Clarify recovery")).toBeInTheDocument();
    expect(screen.getByText(i18n.t("design.refinementRound", { round: 1, budget: 3 }))).toBeInTheDocument();
    expect(api.putSketch).not.toHaveBeenCalled();
    expect(api.saveCandidate).not.toHaveBeenCalled();
  });
  it("requires confirmation for displaced occupants after template preview", async () => {
    setup();
    api.searchDesignAssets.mockResolvedValue({
      results: [
        {
          catalogId: "templates.new",
          name: "New template",
          description: "A workspace",
          implemented: false,
          regions: ["main"],
        },
      ],
    });
    api.setTemplate.mockResolvedValue({
      changed: false,
      newlyUnplaced: [{ region: "body", was: "controls.button", reason: "No region mapping" }],
      requiresConfirmation: true,
    });
    renderPage();
    await screen.findByLabelText(i18n.t("design.note", { region: "body" }));
    const details = screen.getByText(i18n.t("design.changeTemplate")).closest("details")!;
    details.open = true;
    fireEvent(details, new Event("toggle"));
    await screen.findByRole("option", { name: /New template/ });
    fireEvent.change(screen.getByLabelText(i18n.t("design.template")), { target: { value: "templates.new" } });
    fireEvent.click(screen.getByRole("button", { name: i18n.t("design.previewRemap") }));
    const apply = await screen.findByRole("button", { name: i18n.t("design.applyTemplate") });
    expect(apply).toBeDisabled();
    fireEvent.click(screen.getByRole("checkbox", { name: /I confirm moving 1 placements/ }));
    expect(apply).toBeEnabled();
    fireEvent.click(apply);
    await waitFor(() =>
      expect(api.setTemplate).toHaveBeenLastCalledWith(
        expect.objectContaining({
          preview: false,
          confirmUnplaced: true,
          expectedContentHash: "current-hash",
        }),
      ),
    );
  });

  it("previews import before publishing against the current revision", async () => {
    setup();
    api.importPage.mockResolvedValue({
      sketch,
      contentHash: "current-hash",
      items: [],
      written: false,
    });
    renderPage();
    fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.previewImport") }));
    expect(await screen.findByRole("button", { name: i18n.t("design.applyImport") })).toBeEnabled();
    expect(api.importPage).toHaveBeenLastCalledWith({
      target: { scenario: "demo", page: "page" },
      write: false,
      expectedContentHash: "current-hash",
    });
    fireEvent.click(screen.getByRole("button", { name: i18n.t("design.applyImport") }));
    await waitFor(() =>
      expect(api.importPage).toHaveBeenLastCalledWith({
        target: { scenario: "demo", page: "page" },
        write: true,
        expectedContentHash: "current-hash",
      }),
    );
  });

  it("marks verification stale after a saved revision changes", async () => {
    setup();
    api.verifySketch.mockResolvedValue({
      passes: true,
      coverage: { built: 1, total: 1 },
      regions: [],
    });
    api.addNote.mockImplementation(() => {
      api.getSketch.mockResolvedValue({ sketch, contentHash: "next-hash" });
      return Promise.resolve({ sketch, contentHash: "next-hash" });
    });
    renderPage();
    await screen.findByLabelText(i18n.t("design.note", { region: "body" }));
    fireEvent.click(screen.getByRole("button", { name: i18n.t("design.verify") }));
    await screen.findByText(i18n.t("design.verificationPassed"));
    fireEvent.change(screen.getByLabelText(i18n.t("design.note", { region: "body" })), {
      target: { value: "Preserve task hierarchy" },
    });
    fireEvent.click(screen.getByRole("button", { name: i18n.t("design.saveNote") }));
    expect(
      await screen.findByText(/These findings belong to an earlier revision/),
    ).toBeInTheDocument();
    expect(screen.queryByText(i18n.t("design.verificationPassed"))).not.toBeInTheDocument();
  });

  it("shows authored semantic regions even before a sketch is composed", async () => {
    setup();
    api.getSketch.mockResolvedValue({
      sketch: { ...sketch, regions: [] },
      contentHash: "current-hash",
      declaredRegions: [{ id: "authored-body", note: "The primary task", elements: [] }],
    });
    renderPage();
    expect(await screen.findByRole("button", { name: "authored-body" })).toBeInTheDocument();
    expect(screen.getByText(/The primary task/)).toBeInTheDocument();
  });

  it("lists real scenarios as reachable design links", async () => {
    api.listDesignPages.mockResolvedValue({
      scenarios: [{ scenario: "demo", pageCount: 3 }],
      pages: [],
      issues: [],
    });
    renderPage("/design");
    expect(await screen.findByRole("link", { name: /demo/ })).toHaveAttribute(
      "href",
      "/design/demo",
    );
  });
  it("ranks declared work first and makes empty scenarios available on request", async () => {
    api.listDesignPages.mockResolvedValue({ scenarios: [
      { scenario: "aaa-empty", pageCount: 0 },
      { scenario: "small", pageCount: 1 },
      { scenario: "useful", pageCount: 8 },
    ], pages: [], issues: [] });
    renderPage("/design");
    await screen.findByRole("link", { name: /useful/ });
    expect(screen.queryByRole("link", { name: /aaa-empty/ })).not.toBeInTheDocument();
    const links = screen.getAllByRole("link").filter(link => link.getAttribute("href")?.startsWith("/design/"));
    expect(links.map(link => link.getAttribute("href"))).toEqual(["/design/useful", "/design/small"]);
    fireEvent.click(screen.getByRole("checkbox"));
    expect(screen.getByRole("link", { name: /aaa-empty/ })).toBeInTheDocument();
  });
  it("preserves a note after a conflict and sends the loaded revision", async () => {
    setup();
    api.addNote.mockRejectedValue(new ConnectError("conflict", Code.Aborted));
    renderPage();
    const note = await screen.findByLabelText(i18n.t("design.note", { region: "body" }));
    fireEvent.change(note, { target: { value: "Keep the primary task visible" } });
    fireEvent.click(screen.getByRole("button", { name: i18n.t("design.saveNote") }));
    await waitFor(() =>
      expect(api.addNote).toHaveBeenCalledWith(
        expect.objectContaining({
          expectedContentHash: "current-hash",
          text: "Keep the primary task visible",
          scope: "body",
        }),
      ),
    );
    expect(await screen.findByRole("alert")).toHaveTextContent("Your note is preserved");
    expect(note).toHaveValue("Keep the primary task visible");
  });
  it("keeps historical revisions read only", async () => {
    setup();
    renderPage();
    const picker = await screen.findByLabelText(i18n.t("design.revision"));
    await screen.findByRole("option", { name: /2026-01-01/ });
    fireEvent.change(picker, { target: { value: "old-hash" } });
    expect(screen.getByRole("button", { name: i18n.t("design.verify") })).toBeDisabled();
    expect(screen.getByLabelText(i18n.t("design.note", { region: "body" }))).toBeDisabled();
  });
  it("does not present source verification as visual acceptance", async () => {
    setup();
    api.verifySketch.mockResolvedValue({
      passes: false,
      coverage: { built: 0, total: 2, resolved: 1, resolvedLocal: 1, libraryBacked: 1, local: 1 },
      regions: [],
    });
    renderPage();
    fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.verify") }));
    expect(
      await screen.findByText(i18n.t("design.coverage", { built: 0, total: 2 })),
    ).toBeInTheDocument();
    expect(screen.getByText(i18n.t("design.verificationNeedsWork"))).toBeInTheDocument();
    expect(screen.getByText(i18n.t("design.sourceCoverage", { resolved: 1, total: 2, custom: 1 }))).toBeInTheDocument();
    expect(screen.getByText(i18n.t("design.declaredCoverage", { library: 1, local: 1 }))).toBeInTheDocument();
  });
});

it("renders the exact revision and hides stale appearance evidence", async () => {
  setup();
  api.getSketch.mockResolvedValue({ sketch: { ...sketch, render: { templateExport: "Page" } }, contentHash: "current-hash" });
  api.renderSketch.mockResolvedValue({ html: "<html><head></head><body>Fixture</body></html>", renderHash: "render-hash", bundle: { gaps: [] } });
  renderPage();
  fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.canvasRender") }));
  const frame = await screen.findByTitle(i18n.t("design.canvas"));
  expect(frame).toHaveAttribute("sandbox", "allow-scripts allow-forms");
  expect(frame).toHaveAttribute("data-render-hash", "render-hash");
  expect(screen.queryByText(i18n.t("design.canvasEvidence"))).not.toBeInTheDocument();
  fireEvent(window, new MessageEvent("message", { source: (frame as HTMLIFrameElement).contentWindow,
    data: { type: "preview-ready", sha256: "wrong-hash" } }));
  expect(screen.queryByText(i18n.t("design.canvasEvidence"))).not.toBeInTheDocument();
  fireEvent(window, new MessageEvent("message", { source: (frame as HTMLIFrameElement).contentWindow,
    data: { type: "preview-ready", sha256: "render-hash" } }));
  expect(await screen.findByText(i18n.t("design.canvasEvidence"))).toBeInTheDocument();
  expect(api.renderSketch).toHaveBeenCalledWith(expect.objectContaining({ expectedContentHash: "current-hash", theme: "light", direction: "ltr" }));
  fireEvent.change(screen.getByLabelText(i18n.t("design.canvasTheme")), { target: { value: "dark" } });
  expect(screen.queryByTitle(i18n.t("design.canvas"))).not.toBeInTheDocument();
  expect(screen.getByText(/revision or appearance changed/)).toBeInTheDocument();
});

it("proposes from typed intent and only writes after draft selection", async () => {
  setup();
  api.proposeSketch.mockResolvedValue({ contentHash: "current-hash", retrievalMode: "lexical", designSource: { path: "DESIGN.md", contentHash: "source-hash", content: "# Source requirements\nPreserve density. <script>unsafe()</script>" }, diagnostics: [], candidates: [{ title: "Collection", sketch: { ...sketch, template: { asset: "templates.collection-page", version: "1.0.7" } }, obligations: ["Configure ports"] }] });
  api.putSketch.mockResolvedValue({});
  api.saveCandidate.mockImplementation((request: { sketch: unknown; expectedContentHash: string }) => Promise.resolve({ sketch: request.sketch, baseHash: request.expectedContentHash }));
  renderPage();
  fireEvent.click(await screen.findByText(i18n.t("design.proposeTitle")));
  fireEvent.change(screen.getByLabelText(i18n.t("design.proposeIntent")), { target: { value: "Browse records" } });
  fireEvent.change(screen.getByLabelText(i18n.t("design.proposeUsers")), { target: { value: "Operator" } });
  fireEvent.change(screen.getByLabelText(i18n.t("design.proposeTasks")), { target: { value: "Find a record\nInspect context" } });
  fireEvent.click(screen.getByRole("button", { name: i18n.t("design.proposeAction") }));
  await screen.findByRole("button", { name: i18n.t("design.proposeChoose") });
  expect(api.proposeSketch).toHaveBeenCalledWith(expect.objectContaining({ expectedContentHash: "current-hash", candidateLimit: 3,
    intent: expect.objectContaining({ primaryTasks: ["Find a record", "Inspect context"], users: ["Operator"], constraints: { designSource: "", preserveRoutes: true, preserveBusinessBehavior: true } }) }));
  expect(api.putSketch).not.toHaveBeenCalled();
  expect(api.saveCandidate).not.toHaveBeenCalled();
  fireEvent.change(screen.getByLabelText(i18n.t("design.proposeDesignSource")), { target: { value: "DESIGN.md" } });
  fireEvent.click(screen.getByLabelText(i18n.t("design.proposePreserveRoutes")));
  expect(screen.queryByRole("button", { name: i18n.t("design.proposeChoose") })).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: i18n.t("design.proposeAction") }));
  const revisedChoose = await screen.findByRole("button", { name: i18n.t("design.proposeChoose") });
  expect(screen.getByText(i18n.t("design.proposeSourceSnapshot", { path: "DESIGN.md" }))).toBeInTheDocument();
  expect(screen.getByText(/Preserve density/).tagName).toBe("PRE");
  expect(screen.getByText(/Preserve density/).querySelector("script")).toBeNull();
  expect(api.proposeSketch).toHaveBeenLastCalledWith(expect.objectContaining({ intent: expect.objectContaining({
    constraints: { designSource: "DESIGN.md", preserveRoutes: false, preserveBusinessBehavior: true },
  }) }));
  fireEvent.click(revisedChoose);
  await waitFor(() => expect(api.saveCandidate).toHaveBeenCalledWith(expect.objectContaining({ designId: "page-intent", expectedContentHash: "current-hash" })));
  await waitFor(() => expect(api.putSketch).toHaveBeenCalledWith(expect.objectContaining({ expectedContentHash: "current-hash", sketch: expect.objectContaining({ template: { asset: "templates.collection-page", version: "1.0.7" } }) })));
});

it("previews an immutable candidate without selecting the current draft", async () => {
  setup();
  const candidate = { scenario: "demo", designId: "page-intent", hash: "candidate-hash" };
  const candidateSketch = { ...sketch, render: { templateExport: "Page" } };
  api.proposeSketch.mockResolvedValue({ contentHash: "current-hash", diagnostics: [], candidates: [{ title: "Page", sketch: candidateSketch, obligations: [] }] });
  api.saveCandidate.mockResolvedValue({ candidate, sketch: candidateSketch, declaredRegions: [], baseHash: "current-hash" });
  api.renderCandidate.mockResolvedValue({ html: "<html><body>Candidate</body></html>", renderHash: "candidate-render", target: { renderHash: "candidate-render" }, bundle: { gaps: [] } });
  renderPage();
  fireEvent.click(await screen.findByText(i18n.t("design.proposeTitle")));
  for (const [key, value] of [["design.proposeIntent", "Browse records"], ["design.proposeUsers", "Operator"], ["design.proposeTasks", "Inspect a record"]] as const) {
    fireEvent.change(screen.getByLabelText(i18n.t(key)), { target: { value } });
  }
  fireEvent.click(screen.getByRole("button", { name: i18n.t("design.proposeAction") }));
  fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.proposePreview") }));
  await waitFor(() => expect(api.saveCandidate).toHaveBeenCalled());
  const renderButtons = await screen.findAllByRole("button", { name: i18n.t("design.canvasRender") });
  fireEvent.click(renderButtons.find((button) => !(button as HTMLButtonElement).disabled)!);
  await screen.findByTitle(i18n.t("design.canvas"));
  expect(api.renderCandidate).toHaveBeenCalledWith(expect.objectContaining({ candidate }));
  expect(api.putSketch).not.toHaveBeenCalled();
  expect(api.renderSketch).not.toHaveBeenCalled();
  const operation = { id: "capture-one", state: "running", producerId: "bas-one", width: 1440, height: 900, artifacts: [] };
  api.captureCandidate.mockRejectedValueOnce(new ConnectError("attachment timed out", Code.DeadlineExceeded)).mockResolvedValue(operation);
  api.getCapture.mockResolvedValue(operation);
  api.attachCapture.mockResolvedValue({ ...operation, state: "completed", artifacts: [{ kind: "screenshot", reference: "bas:one:screenshot" }] });
  fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.captureStart") }));
  await screen.findByText(/attachment timed out/);
  fireEvent.click(screen.getByRole("button", { name: i18n.t("design.captureStart") }));
  await screen.findByText(i18n.t("design.captureRunning"));
  expect(api.captureCandidate).toHaveBeenCalledTimes(2);
  expect(api.captureCandidate.mock.calls[0]?.[0].idempotencyKey).toBe(api.captureCandidate.mock.calls[1]?.[0].idempotencyKey);
  expect(api.captureCandidate).toHaveBeenLastCalledWith(expect.objectContaining({ expectedRenderHash: "candidate-render", width: 1440, height: 900, render: expect.objectContaining({ candidate }) }));
  api.cancelCapture.mockResolvedValue({ ...operation, state: "cancel_requested" });
  fireEvent.click(screen.getByRole("button", { name: i18n.t("design.captureCancel") }));
  await screen.findByText(i18n.t("design.captureCancelRequested"));
  expect(api.cancelCapture).toHaveBeenCalledWith({ id: "capture-one" });
  expect(screen.queryByText(i18n.t("design.captureCancelled"))).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: i18n.t("design.captureAttach") }));
  await screen.findByText(i18n.t("design.captureCompleted"));
  expect(api.attachCapture).toHaveBeenCalledWith({ id: "capture-one" });
  expect(api.getCaptureScreenshot).not.toHaveBeenCalled();
  api.getCaptureScreenshot.mockResolvedValue({ reference: "bas:one:screenshot", url: "http://bas.local/captured.png", width: 2880, height: 1800 });
  fireEvent.click(screen.getByRole("button", { name: i18n.t("design.captureViewScreenshot") }));
  const image = await screen.findByAltText(i18n.t("design.captureScreenshotAlt"));
  expect(api.getCaptureScreenshot).toHaveBeenCalledWith({ id: "capture-one", reference: "bas:one:screenshot" });
  expect(image).toHaveAttribute("src", "http://bas.local/captured.png");
  expect(image).toHaveAttribute("width", "1440");
  expect(image).toHaveAttribute("height", "900");
  fireEvent.error(image);
  await screen.findByText(i18n.t("design.captureImageUnavailable"));

  expect(screen.queryByRole("button", { name: i18n.t("design.captureStart") })).not.toBeInTheDocument();
  expect(api.putSketch).not.toHaveBeenCalled();
});

it("restores a capture receipt when reopening the same rendered candidate", async () => {
  setup();
  const candidate = { scenario: "demo", designId: "saved-design", hash: "saved-hash" };
  const candidateSketch = { ...sketch, render: { templateExport: "Page" } };
  api.listCandidates.mockResolvedValue({ candidates: [{ candidate }] });
  api.getCandidate.mockResolvedValue({ candidate, sketch: candidateSketch, declaredRegions: [], baseHash: "current-hash" });
  api.renderCandidate.mockResolvedValue({ html: "<html><body>Candidate</body></html>", renderHash: "saved-render", target: { renderHash: "saved-render" }, bundle: { gaps: [] } });
  sessionStorage.setItem("rcl:capture:saved-render:1440x900", JSON.stringify({ key: "saved-intent", id: "capture-saved" }));
  api.getCapture.mockResolvedValue({ id: "capture-saved", state: "completed", producerId: "bas-saved", artifacts: [] });
  renderPage();
  fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.openCandidate", { design: "saved-design", revision: "saved-hash" }) }));
  await waitFor(() => expect(document.querySelector('[data-candidate-hash="saved-hash"]')).toBeTruthy());
  const buttons = await screen.findAllByRole("button", { name: i18n.t("design.canvasRender") });
  const enabled = buttons.find((button) => !(button as HTMLButtonElement).disabled);
  expect(enabled).toBeTruthy();
  if (enabled) fireEvent.click(enabled);
  await screen.findByText(i18n.t("design.captureCompleted"));
  expect(api.getCapture).toHaveBeenCalledWith({ id: "capture-saved" });
  expect(api.captureCandidate).not.toHaveBeenCalled();
  expect(api.attachCapture).not.toHaveBeenCalled();
});


it("creates a new linked capture intent and reuses it after a lost acknowledgement", async () => {
  setup();
  const candidate = { scenario: "demo", designId: "saved-design", hash: "saved-hash" };
  const candidateSketch = { ...sketch, render: { templateExport: "Page" } };
  api.listCandidates.mockResolvedValue({ candidates: [{ candidate }] });
  api.getCandidate.mockResolvedValue({ candidate, sketch: candidateSketch, declaredRegions: [], baseHash: "current-hash" });
  api.renderCandidate.mockResolvedValue({ html: "<html><body>Candidate</body></html>", renderHash: "saved-render", target: { renderHash: "saved-render" }, bundle: { gaps: [] } });
  sessionStorage.setItem("rcl:capture:saved-render:1440x900", JSON.stringify({ key: "original-intent", id: "capture-original" }));
  api.getCapture.mockImplementation(({ id }: { id: string }) => Promise.resolve({ id, state: id === "capture-original" ? "cancelled" : "running", producerId: "bas-" + id, artifacts: [] }));
  api.retryCapture.mockRejectedValueOnce(new ConnectError("retry acknowledgement lost", Code.DeadlineExceeded))
    .mockResolvedValue({ id: "capture-new", previousId: "capture-original", state: "running", producerId: "bas-new", artifacts: [] });
  renderPage();
  fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.openCandidate", { design: "saved-design", revision: "saved-hash" }) }));
  await waitFor(() => expect(document.querySelector('[data-candidate-hash="saved-hash"]')).toBeTruthy());
  const buttons = await screen.findAllByRole("button", { name: i18n.t("design.canvasRender") });
  fireEvent.click(buttons.find((button) => !(button as HTMLButtonElement).disabled)!);
  fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.captureAgain") }));
  await screen.findByText(/retry acknowledgement lost/);
  fireEvent.click(screen.getByRole("button", { name: i18n.t("design.captureStart") }));
  await screen.findByText(i18n.t("design.captureRunning"));
  expect(api.retryCapture).toHaveBeenCalledTimes(2);
  const intent = api.retryCapture.mock.calls[0]?.[0];
  expect(intent.id).toBe("capture-original");
  expect(intent.idempotencyKey).not.toBe("original-intent");
  expect(api.retryCapture.mock.calls[1]?.[0]).toEqual(intent);
  expect(api.captureCandidate).not.toHaveBeenCalled();
  expect(screen.queryByRole("button", { name: i18n.t("design.captureAgain") })).not.toBeInTheDocument();
  fireEvent.click(screen.getByText(i18n.t("design.capturePrevious", { id: "capture-original" })));
  fireEvent.click(screen.getByRole("button", { name: i18n.t("design.capturePreviousRead") }));
  await waitFor(() => expect(api.getCapture).toHaveBeenLastCalledWith({ id: "capture-original" }));
});


it("lists exact-render reviews and rejects mismatched review detail", async () => {
  setup();
  const candidate = { scenario: "demo", designId: "saved-design", hash: "saved-hash" };
  const target = { scenario: "demo", designId: "saved-design", revision: "saved-hash", renderHash: "saved-render" };
  const candidateSketch = { ...sketch, render: { templateExport: "Page" } };
  api.listCandidates.mockResolvedValue({ candidates: [{ candidate }] });
  api.getCandidate.mockResolvedValue({ candidate, sketch: candidateSketch, declaredRegions: [], baseHash: "current-hash" });
  api.renderCandidate.mockResolvedValue({ html: "<html><body>Candidate</body></html>", renderHash: "saved-render", target: { renderHash: "saved-render" }, bundle: { gaps: [] } });
  const summary = { id: "review-one", hash: "review-hash", critic: { id: "critic-one" }, recordedAt: "2026-09-05", assessment: { visualFloorMet: false } };
  api.listCritiques.mockImplementation(({ beforeId }: { beforeId: string }) => Promise.resolve(beforeId ? { reviews: [{ ...summary, id: "review-two", critic: { id: "critic-two" } }], nextBeforeId: "" } : { reviews: [summary], nextBeforeId: "review-one" }));
  const rationale = "No recovery action is visible.";
  const review = { target, rubricVersion: "visual-design/1", policyVersion: "visual-floor/1", critic: { id: "critic-one", version: "1", model: "fixture", profile: "test" },
    ratings: [{ dimension: "state_recovery", score: 2, rationale, evidence: [{ captureId: "alternate-capture", artifact: "bas:alternate:image", region: "$page", state: "detail", width: 390, height: 844, renderHash: "alternate-render" }] }],
    findings: [{ dimension: "state_recovery", severity: "major", rationale: "The user cannot retry.", correction: "Add an explicit retry action." }] };
  api.getCritique.mockResolvedValueOnce({ id: "review-one", hash: "review-hash", review, assessment: { minimumScore: 2 } })
    .mockResolvedValueOnce({ id: "review-two", hash: "review-hash", review: { ...review, target: { ...target, revision: "other-revision" } } });
  renderPage();
  fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.openCandidate", { design: "saved-design", revision: "saved-hash" }) }));
  await waitFor(() => expect(document.querySelector('[data-candidate-hash="saved-hash"]')).toBeTruthy());
  const buttons = await screen.findAllByRole("button", { name: i18n.t("design.canvasRender") });
  fireEvent.click(buttons.find((button) => !(button as HTMLButtonElement).disabled)!);
  fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.reviewOpen", { critic: "critic-one", date: "2026-09-05" }) }));
  await screen.findByText(rationale);
  expect(screen.getByText("alternate-render")).toBeInTheDocument();
  api.checkCandidateAcceptance.mockResolvedValue({ ready: false, acceptanceEstablished: false, requirements: [{ code: "rubric_calibration", status: "unavailable", detail: "Reviewed calibration examples are required." }] });
  fireEvent.click(screen.getByRole("button", { name: i18n.t("design.acceptanceCheck") }));
  await screen.findByText("Reviewed calibration examples are required.");
  expect(api.checkCandidateAcceptance).toHaveBeenCalledWith({ render: expect.objectContaining({ candidate }), expectedRenderHash: "saved-render", critiqueIds: ["review-one"] });
  expect(screen.getByText(i18n.t("design.acceptanceNotAccepted"))).toBeInTheDocument();
  api.acceptCandidate.mockRejectedValueOnce(new Error("response lost")).mockResolvedValue({ id: "acceptance-one", hash: "decision-hash", state: "needs_evidence", intent: { candidateHash: candidate.hash, expectedRenderHash: "saved-render" } });
  fireEvent.change(screen.getByLabelText(i18n.t("design.acceptanceActor")), { target: { value: "test-agent" } });
  fireEvent.click(screen.getByRole("button", { name: i18n.t("design.acceptanceRecord") }));
  fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.acceptanceRetry") }));
  await screen.findByText(i18n.t("design.acceptanceDecision_needs_evidence"));
  expect(api.acceptCandidate.mock.calls[0]?.[0]).toEqual(api.acceptCandidate.mock.calls[1]?.[0]);
  fireEvent.click(screen.getByRole("button", { name: i18n.t("design.acceptanceNewAttempt") }));
  await waitFor(() => expect(api.acceptCandidate).toHaveBeenCalledTimes(3));
  expect(api.acceptCandidate.mock.calls[2]?.[0].idempotencyKey).not.toBe(api.acceptCandidate.mock.calls[1]?.[0].idempotencyKey);


  expect(api.listCritiques).toHaveBeenCalledWith({ target, beforeId: "" });
  expect(screen.getByText(i18n.t("design.reviewMinimum", { score: 2 }))).toBeInTheDocument();
  expect(screen.getByText(i18n.t("design.reviewNotAcceptance"))).toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: i18n.t("design.reviewMore") }));
  fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.reviewOpen", { critic: "critic-two", date: "2026-09-05" }) }));
  await screen.findByText(i18n.t("design.reviewMismatch"));
  expect(screen.queryByText(rationale)).not.toBeInTheDocument();
  expect(screen.queryByText("Reviewed calibration examples are required.")).not.toBeInTheDocument();
  expect(api.captureCandidate).not.toHaveBeenCalled();
});

it("renders and captures the selected declared preview state", async () => {
  setup();
  const candidate = { scenario: "demo", designId: "states", hash: "state-hash" };
  api.listCandidates.mockResolvedValue({ candidates: [{ candidate }] });
  api.getCandidate.mockResolvedValue({ candidate, sketch: { ...sketch, render: {
    templateExport: "Page", bindings: { $preview: { initial: "list", states: { list: {}, detail: {} } } },
  } }, declaredRegions: [], baseHash: "current-hash" });
  api.renderCandidate.mockResolvedValue({ html: "<html><body>Detail</body></html>", renderHash: "detail-render", target: { renderHash: "detail-render" }, bundle: { gaps: [] } });
  api.captureCandidate.mockResolvedValue({ id: "detail-capture", state: "running", artifacts: [] });
  renderPage();
  fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.openCandidate", { design: "states", revision: "state-hash" }) }));
  fireEvent.change(await screen.findByLabelText(i18n.t("design.canvasState")), { target: { value: "detail" } });
  const buttons = await screen.findAllByRole("button", { name: i18n.t("design.canvasRender") });
  const enabled = buttons.find((button) => !(button as HTMLButtonElement).disabled)!;
  fireEvent.click(enabled);
  fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.captureStart") }));
  await waitFor(() => expect(api.captureCandidate).toHaveBeenCalledWith(expect.objectContaining({
    expectedRenderHash: "detail-render", render: expect.objectContaining({ candidate, previewState: "detail" }),
  })));
  expect(api.renderCandidate).toHaveBeenCalledWith(expect.objectContaining({ candidate, previewState: "detail" }));
  fireEvent.change(screen.getByLabelText(i18n.t("design.canvasState")), { target: { value: "list" } });
  expect(await screen.findByText(i18n.t("design.canvasStale"))).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: i18n.t("design.captureStart") })).not.toBeInTheDocument();
});

it("locks and separately unlocks a region against the current revision", async () => {
  setup();
  const locked = { ...sketch, regions: sketch.regions.map((region) => ({ ...region, locked: true })) };
  api.getSketch.mockResolvedValueOnce({ sketch, contentHash: "current-hash" })
    .mockResolvedValue({ sketch: locked, contentHash: "locked-hash" });
  api.putSketch.mockResolvedValue({});
  renderPage();
  fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.lockRegion") }));
  await waitFor(() => expect(api.putSketch).toHaveBeenCalledWith(expect.objectContaining({
    expectedContentHash: "current-hash", sketch: expect.objectContaining({
      regions: [expect.objectContaining({ id: "body", locked: true, note: "Main task" })],
    }),
  })));
  fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.unlockRegion") }));
  await waitFor(() => expect(api.putSketch).toHaveBeenLastCalledWith(expect.objectContaining({
    expectedContentHash: "locked-hash", sketch: expect.objectContaining({
      regions: [expect.objectContaining({ id: "body", locked: false, note: "Main task" })],
    }),
  })));
});

it("compares two exact candidates without changing the open candidate or page", async () => {
  setup();
  const refs = ["a", "b", "c"].map((hash) => ({ scenario: "demo", designId: "comparison", hash }));
  api.listCandidates.mockResolvedValue({ candidates: refs.map((candidate) => ({ candidate })) });
  api.getCandidate.mockImplementation(async (candidate) => ({ candidate, page: "page", baseHash: "current-hash",
    sketch: { ...sketch, render: { templateExport: "Page" } }, declaredRegions: [] }));
  api.renderCandidate.mockImplementation(async ({ candidate }) => ({ html: "<html><body>" + candidate.hash + "</body></html>", renderHash: candidate.hash, bundle: { gaps: [] } }));
  renderPage();
  fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.openCandidate", { design: "comparison", revision: "a" }) }));
  await waitFor(() => expect(document.querySelector('[data-candidate-hash="a"]')).toBeTruthy());
  fireEvent.click(screen.getByRole("button", { name: i18n.t("design.compareCandidate", { revision: "a" }) }));
  await waitFor(() => expect(document.querySelector('[data-comparison-candidate="a"]')).toBeTruthy());
  fireEvent.click(screen.getByRole("button", { name: i18n.t("design.compareCandidate", { revision: "b" }) }));
  await waitFor(() => expect(document.querySelector('[data-comparison-candidate="b"]')).toBeTruthy());
  expect(screen.getByRole("button", { name: i18n.t("design.compareCandidate", { revision: "c" }) })).toBeDisabled();
  const left = document.querySelector('[data-comparison-candidate="a"]')!;
  const right = document.querySelector('[data-comparison-candidate="b"]')!;
  fireEvent.click(left.querySelector('[data-action="render-design"]')!);
  fireEvent.click(right.querySelector('[data-action="render-design"]')!);
  await waitFor(() => expect(right.querySelector('iframe[data-render-hash="b"]')).toBeTruthy());
  const frame = right.querySelector("iframe")!;
  expect(frame.style.width).toBe("1440px");
  expect(frame.style.height).toBe("900px");
  fireEvent.change(right.querySelector("select")!, { target: { value: "390px" } });
  expect(frame.style.width).toBe("390px");
  expect(frame.style.height).toBe("844px");
  fireEvent.click(screen.getByRole("button", { name: i18n.t("design.compareCandidate", { revision: "a" }) }));
  expect(document.querySelector('[data-comparison-candidate="a"]')).toBeNull();
  expect(right.querySelector("iframe")).toBe(frame);
  expect(document.querySelector('[data-candidate-hash="a"]')).toBeTruthy();
  expect(api.putSketch).not.toHaveBeenCalled();
  expect(api.saveCandidate).not.toHaveBeenCalled();
});

it("rejects a comparison response for another candidate", async () => {
  setup();
  const candidate = { scenario: "demo", designId: "comparison", hash: "a" };
  api.listCandidates.mockResolvedValue({ candidates: [{ candidate }] });
  api.getCandidate.mockResolvedValue({ candidate: { ...candidate, hash: "other" }, page: "page", sketch });
  renderPage();
  fireEvent.click(await screen.findByRole("button", { name: i18n.t("design.compareCandidate", { revision: "a" }) }));
  expect(await screen.findByText(i18n.t("design.compareMismatch"))).toBeInTheDocument();
  expect(document.querySelector('[data-comparison-candidate]')).toBeNull();
});
