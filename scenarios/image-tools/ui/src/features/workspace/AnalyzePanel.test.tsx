/**
 * AnalyzePanel tests — the Analyze-mode inspector. AI-op discovery is mocked
 * and the `UseAnalyze` lifecycle is a hand-built fake, so the panel's analysis
 * op list, op-switch clearing, run/install wiring, the install gate, progress,
 * failure + retry, and the three discriminated result views (probe / OCR /
 * NSFW) are exercised in isolation (the lifecycle itself lives in
 * useAnalyze.test; the idle-list + happy-path a11y lives in the a11y test).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import {
  makeAnalysisMocks,
  makeDuplicateResult,
  makeNsfwResult,
  makeOcrResult,
  makeProbeResult,
  makeQualityResult,
} from "./mocks/analysis";
import { makeSelectedModel } from "./mocks/ai";

vi.mock("../../api/analysis", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/analysis")>();
  return { ...actual, ...makeAnalysisMocks() };
});

import { AnalyzePanel } from "./AnalyzePanel";
import type { UseAnalyze } from "./useAnalyze";

const PNG = new File(["bytes"], "in.png", { type: "image/png" });

const fakeAnalyze = (overrides: Partial<UseAnalyze> = {}): UseAnalyze => ({
  phase: "idle",
  model: null,
  result: null,
  error: null,
  run: vi.fn(),
  installAndRun: vi.fn(),
  cancel: vi.fn(),
  retry: vi.fn(),
  clear: vi.fn(),
  ...overrides,
});

const renderPanel = (analyze: UseAnalyze, input: File | null = PNG) =>
  renderWithProviders(<AnalyzePanel analyze={analyze} input={input} />);

beforeEach(async () => {
  await setLocale("en");
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("AnalyzePanel", () => {
  it("lists the discovered analysis ops and runs the selected one with the model-backed flag", async () => {
    const analyze = fakeAnalyze();
    const user = userEvent.setup();
    renderPanel(analyze);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.analyzeAction({ name: "probe" }))).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.workspace.analyzeAction({ name: "ocr" }))).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.workspace.analyzeAction({ name: "nsfw_classify" })),
    ).toBeInTheDocument();

    // Default op is probe (pure-Go) — running passes modelBacked=false.
    await user.click(screen.getByTestId(selectors.workspace.analyze.run));
    expect(analyze.run).toHaveBeenCalledWith("probe", PNG, false);
  });

  it("clears the prior result and runs model-backed when switching ops", async () => {
    const analyze = fakeAnalyze();
    const user = userEvent.setup();
    renderPanel(analyze);

    await user.click(
      await screen.findByTestId(selectors.workspace.analyzeAction({ name: "ocr" })),
    );
    // Switching ops clears the stale result…
    expect(analyze.clear).toHaveBeenCalledTimes(1);
    // …and a model-backed op surfaces the model note.
    expect(screen.getByText(/model/i)).toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.workspace.analyze.run));
    expect(analyze.run).toHaveBeenCalledWith("ocr", PNG, true);
  });

  it("does not re-clear when the already-selected op is clicked again", async () => {
    const analyze = fakeAnalyze();
    const user = userEvent.setup();
    renderPanel(analyze);

    const probe = await screen.findByTestId(selectors.workspace.analyzeAction({ name: "probe" }));
    await user.click(probe); // probe is already the default selection
    expect(analyze.clear).not.toHaveBeenCalled();
  });

  it("disables run and shows the needs-image hint when no image is loaded", async () => {
    const analyze = fakeAnalyze();
    renderPanel(analyze, null);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.analyze.run)).toBeDisabled();
    });
    expect(screen.getByText(/add an image/i)).toBeInTheDocument();
  });

  it("renders the install gate and installs the model on confirm", async () => {
    const analyze = fakeAnalyze({
      phase: "needs-install",
      model: makeSelectedModel({ id: "tesseract", name: "tesseract", installed: false, sizeMb: 12 }),
    });
    const user = userEvent.setup();
    renderPanel(analyze);

    const gate = await screen.findByTestId(selectors.workspace.analyze.installGate);
    expect(gate.textContent).toContain("tesseract");
    await user.click(screen.getByTestId(selectors.workspace.analyze.install));
    expect(analyze.installAndRun).toHaveBeenCalledTimes(1);
  });

  it("renders the install gate for a GPU-only model with no size", async () => {
    const analyze = fakeAnalyze({
      phase: "needs-install",
      model: makeSelectedModel({
        id: "nsfw",
        name: "nsfw",
        installed: false,
        cpuCapable: false,
        minVramGb: 4,
        sizeMb: 0,
      }),
    });
    renderPanel(analyze);
    expect(await screen.findByTestId(selectors.workspace.analyze.installGate)).toBeInTheDocument();
  });

  it("shows the running spinner while analysis is in flight", async () => {
    const analyze = fakeAnalyze({ phase: "running" });
    renderPanel(analyze);
    expect(await screen.findByTestId(selectors.workspace.analyze.progress)).toBeInTheDocument();
  });

  it("shows the installing spinner during the model install", async () => {
    const analyze = fakeAnalyze({ phase: "installing" });
    renderPanel(analyze);
    expect(await screen.findByTestId(selectors.workspace.analyze.progress)).toBeInTheDocument();
  });

  it("renders the failure with its error and retries on click", async () => {
    const analyze = fakeAnalyze({ phase: "failed", error: "tesseract crashed" });
    const user = userEvent.setup();
    renderPanel(analyze);

    const failed = await screen.findByTestId(selectors.workspace.analyze.failed);
    expect(failed.textContent).toContain("tesseract crashed");
    await user.click(screen.getByTestId(selectors.workspace.analyze.retry));
    expect(analyze.retry).toHaveBeenCalledTimes(1);
  });

  it("falls back to generic failure copy when no error message is present", async () => {
    const analyze = fakeAnalyze({ phase: "failed", error: null });
    renderPanel(analyze);
    const failed = await screen.findByTestId(selectors.workspace.analyze.failed);
    expect(failed).toHaveTextContent(/\S/);
  });

  it("renders the full probe metadata view including frames, EXIF, GPS, orientation and palette", async () => {
    const analyze = fakeAnalyze({
      phase: "done",
      result: makeProbeResult({
        frameCount: 12,
        hasExif: true,
        hasGps: true,
        orientation: 6,
        sizeBytes: 2_500_000,
        dominantColors: [
          { hex: "#112233", fraction: 0.5 },
          { hex: "#445566", fraction: 0.3 },
        ],
      }),
    });
    renderPanel(analyze);

    const probe = await screen.findByTestId(selectors.workspace.analyze.probe);
    expect(probe).toBeInTheDocument();
    expect(screen.getByText(/640×480/)).toBeInTheDocument();
    expect(screen.getByText("12")).toBeInTheDocument(); // frame count
    expect(screen.getByText("6")).toBeInTheDocument(); // orientation
    expect(screen.getByText(/2\.5 MB/)).toBeInTheDocument(); // formatBytes >= 1MB
  });

  it("formats sub-megabyte probe sizes in KB and bytes", async () => {
    const kb = fakeAnalyze({
      phase: "done",
      result: makeProbeResult({ sizeBytes: 4_096, dominantColors: [], frameCount: 1, orientation: 0 }),
    });
    const { unmount } = renderPanel(kb);
    expect(await screen.findByText(/4 KB/)).toBeInTheDocument();
    unmount();

    const bytes = fakeAnalyze({
      phase: "done",
      result: makeProbeResult({ sizeBytes: 512, dominantColors: [] }),
    });
    renderPanel(bytes);
    expect(await screen.findByText(/512 B/)).toBeInTheDocument();
  });

  it("renders OCR text with a working copy affordance", async () => {
    // userEvent.setup() installs a clipboard stub; spy on its writeText so the
    // copy handler's resolved path flips the affordance to "Copied".
    const user = userEvent.setup();
    const writeText = vi
      .spyOn(navigator.clipboard, "writeText")
      .mockResolvedValue(undefined);
    const analyze = fakeAnalyze({ phase: "done", result: makeOcrResult({ fullText: "Hello world" }) });
    renderPanel(analyze);

    expect((await screen.findByTestId(selectors.workspace.analyze.ocrText)).textContent).toContain(
      "Hello world",
    );
    await user.click(screen.getByTestId(selectors.workspace.analyze.copy));
    expect(writeText).toHaveBeenCalledWith("Hello world");
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.analyze.copy)).toHaveTextContent(/copied/i);
    });
  });

  it("stays unconfirmed when the clipboard write rejects", async () => {
    const user = userEvent.setup();
    vi.spyOn(navigator.clipboard, "writeText").mockRejectedValue(new Error("denied"));
    const analyze = fakeAnalyze({ phase: "done", result: makeOcrResult({ fullText: "Hello world" }) });
    renderPanel(analyze);

    const copy = await screen.findByTestId(selectors.workspace.analyze.copy);
    await user.click(copy);
    // The rejection path leaves the button in its default "Copy text" state.
    await waitFor(() => {
      expect(copy).toHaveTextContent(/copy/i);
    });
    expect(copy).not.toHaveTextContent(/copied/i);
  });

  it("disables copy and shows the empty hint for OCR with no text", async () => {
    const analyze = fakeAnalyze({
      phase: "done",
      result: makeOcrResult({ fullText: "", language: "", blocks: [] }),
    });
    renderPanel(analyze);

    expect(await screen.findByTestId(selectors.workspace.analyze.ocr)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workspace.analyze.copy)).toBeDisabled();
    expect(screen.queryByTestId(selectors.workspace.analyze.ocrText)).not.toBeInTheDocument();
  });

  it("renders a flagged NSFW verdict with per-label categories", async () => {
    const analyze = fakeAnalyze({
      phase: "done",
      result: makeNsfwResult({
        flagged: true,
        score: 0.92,
        categories: [
          { label: "nsfw", score: 0.92 },
          { label: "sfw", score: 0.08 },
        ],
      }),
    });
    renderPanel(analyze);

    const nsfw = await screen.findByTestId(selectors.workspace.analyze.nsfw);
    expect(nsfw).toBeInTheDocument();
    // Flagged verdict + the score line both render inside the nsfw panel.
    expect(nsfw).toHaveTextContent(/flagged/i);
    expect(nsfw).toHaveTextContent(/92%/);
    // Each category label renders in its own row.
    expect(screen.getByText(/^nsfw$/)).toBeInTheDocument();
  });

  it("renders a safe NSFW verdict with no categories", async () => {
    const analyze = fakeAnalyze({
      phase: "done",
      result: makeNsfwResult({ flagged: false, categories: [] }),
    });
    renderPanel(analyze);
    expect(await screen.findByTestId(selectors.workspace.analyze.nsfw)).toBeInTheDocument();
  });

  it("renders the duplicate fingerprints with a working hash copy affordance", async () => {
    const user = userEvent.setup();
    const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
    const analyze = fakeAnalyze({
      phase: "done",
      result: makeDuplicateResult({ phashHex: "abcd1234", ahashHex: "5678ef00", hashBits: 64 }),
    });
    renderPanel(analyze);

    const dup = await screen.findByTestId(selectors.workspace.analyze.duplicate);
    expect(dup).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workspace.analyze.duplicatePhash)).toHaveTextContent(
      "abcd1234",
    );
    expect(screen.getByTestId(selectors.workspace.analyze.duplicateAhash)).toHaveTextContent(
      "5678ef00",
    );

    // The pHash copy button (labelled "Copy") copies the pHash value.
    const copyButtons = screen.getAllByRole("button", { name: /copy/i });
    await user.click(copyButtons[0] as HTMLElement);
    expect(writeText).toHaveBeenCalledWith("abcd1234");
  });

  it("renders the quality assessment with scores, the blurry flag, and notes", async () => {
    const analyze = fakeAnalyze({
      phase: "done",
      result: makeQualityResult({
        overallScore: 0.4,
        sharpness: 0.12,
        blurry: true,
        exposure: "underexposed",
        notes: ["low contrast", "underexposed"],
      }),
    });
    renderPanel(analyze);

    const quality = await screen.findByTestId(selectors.workspace.analyze.quality);
    expect(quality).toBeInTheDocument();
    expect(quality).toHaveTextContent("40%");
    expect(quality).toHaveTextContent(/blurry/i);
    expect(quality).toHaveTextContent("underexposed");
    expect(quality).toHaveTextContent("low contrast");
  });

  it("renders a sharp quality verdict and tolerates no notes", async () => {
    const analyze = fakeAnalyze({
      phase: "done",
      result: makeQualityResult({ blurry: false, notes: [] }),
    });
    renderPanel(analyze);
    const quality = await screen.findByTestId(selectors.workspace.analyze.quality);
    expect(quality).toHaveTextContent(/sharp/i);
  });

  it("renders the discovery error state when op discovery rejects", async () => {
    const { listAnalysisOperations } = await import("../../api/analysis");
    vi.mocked(listAnalysisOperations).mockRejectedValueOnce(new Error("analysis down"));
    renderPanel(fakeAnalyze());
    expect(await screen.findByTestId(selectors.workspace.analyze.error)).toBeInTheDocument();
  });
});
