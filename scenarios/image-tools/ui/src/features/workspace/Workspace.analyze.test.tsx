/**
 * Workspace × Analyze integration. Drives the real <Workspace /> with an
 * injected AnalyzeClient: switch to Analyze mode, load an image, run the
 * pure-Go probe (metadata panel) and the OCR op (copyable text), and assert the
 * structured results render — the Stage-5 wiring end to end, without a network.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { makeOpsMocks } from "./mocks/ops";
import { makeAnalysisMocks, makeAnalyzeClient, makeOcrResult, makeProbeResult } from "./mocks/analysis";

vi.mock("../../api/ops", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/ops")>();
  return { ...actual, ...makeOpsMocks() };
});

vi.mock("../../api/analysis", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/analysis")>();
  return { ...actual, ...makeAnalysisMocks() };
});

import { Workspace } from "./Workspace";

const PNG = new File(["bytes"], "in.png", { type: "image/png" });

describe("Workspace analyze flow", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.stubGlobal("URL", {
      ...URL,
      createObjectURL: vi.fn(() => "blob:fake"),
      revokeObjectURL: vi.fn(),
    });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
    vi.unstubAllGlobals();
  });

  it("runs probe (metadata panel) and OCR (copyable text) on the loaded image", async () => {
    const user = userEvent.setup();
    // The injected client returns a result keyed by op; both models are installed.
    const analyzeClient = makeAnalyzeClient({
      analyze: vi.fn((op: string) =>
        Promise.resolve(op === "ocr" ? makeOcrResult() : makeProbeResult()),
      ),
    });
    renderWithProviders(<Workspace analyzeClient={analyzeClient} />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.modeSwitcher)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.workspace.modeOption({ mode: "analyze" })));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.analyze.panel)).toBeInTheDocument();
    });

    await user.upload(screen.getByTestId(selectors.workspace.fileInput), PNG);

    // Default op is probe (pure-Go, no model gate) → metadata panel.
    await user.click(screen.getByTestId(selectors.workspace.analyze.run));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.analyze.probe)).toBeInTheDocument();
    });
    expect(screen.getByText(/640×480/)).toBeInTheDocument();

    // Switch to OCR and run → copyable text panel.
    await user.click(screen.getByTestId(selectors.workspace.analyzeAction({ name: "ocr" })));
    await user.click(screen.getByTestId(selectors.workspace.analyze.run));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.analyze.ocr)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.workspace.analyze.ocrText).textContent).toContain(
      "Hello world",
    );
    expect(screen.getByTestId(selectors.workspace.analyze.copy)).toBeInTheDocument();
  });
});
