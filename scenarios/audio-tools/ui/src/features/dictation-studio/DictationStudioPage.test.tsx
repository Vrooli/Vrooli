import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { expectNoA11yViolations } from "../../test-utils/a11y";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { DICTATION_SCRIPTS } from "./scripts";

// Stub the recorder so the page test never touches VoiceStreamProvider; the
// button hands the page a fixed captured clip.
vi.mock("./DictationRecorder", () => ({
  DictationRecorder: ({ onCaptured }: { onCaptured: (clip: unknown) => void }) => (
    <button
      type="button"
      data-testid="mock-capture"
      onClick={() =>
        onCaptured({
          audio: new Uint8Array([1, 2, 3, 4]),
          durationMs: 1_000,
          sampleRateHz: 16_000,
          transcript: "captured words",
        })
      }
    >
      capture
    </button>
  ),
}));

const createClip = vi.fn();
const listClips = vi.fn();
const deleteClip = vi.fn();
vi.mock("../../services/corpus", () => ({
  ClipSource: { UNSPECIFIED: 0, FREE_FORM: 1, SCRIPTED: 2 },
  createClip: (a: unknown) => createClip(a),
  listClips: () => listClips(),
  deleteClip: (id: string) => deleteClip(id),
}));
vi.mock("../../services/experiment", () => ({
  listExperiments: () => Promise.resolve([]),
  startExperiment: () => Promise.resolve({ id: "exp-1", name: "exp-1", status: "queued", recipe: { strategies: [] } }),
  waitExperiment: () => Promise.resolve({ experiment: null, runs: [] }),
  cancelExperiment: () => Promise.resolve(null),
  getExperimentReport: () => Promise.resolve({ experiment: null, report: { perStrategy: [], qualityMeasured: false, latencyMeasured: false, summary: null, warnings: [], normalizationPolicy: null, latencyHonesty: "" }, runs: [] }),
  compareExperiments: () => Promise.resolve([]),
}));

const pushToast = vi.fn();
vi.mock("../../components/ui/toast", () => ({
  pushToast: (...args: unknown[]) => pushToast(...args),
}));

import { DictationStudioPage } from "./DictationStudioPage";

beforeEach(() => {
  vi.clearAllMocks();
  createClip.mockResolvedValue({ id: "c1" });
  listClips.mockResolvedValue([]);
});
afterEach(cleanup);

describe("DictationStudioPage", () => {
  it("renders the three tabs and has no a11y violations", async () => {
    const { container } = renderWithProviders(
      <main>
        <DictationStudioPage />
      </main>,
    );
    expect(screen.getByRole("tab", { name: strings.dictationStudio.tabRecord })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: strings.dictationStudio.tabCorpus })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: strings.dictationStudio.tabLab })).toBeInTheDocument();
    await expectNoA11yViolations(container);
  });

  it("captures a turn, pre-fills the transcript, and saves a free-form clip", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DictationStudioPage />);

    // Save is disabled before any capture.
    expect(screen.getByTestId(selectors.dictationStudio.saveClip)).toBeDisabled();

    await user.click(screen.getByTestId("mock-capture"));
    await waitFor(() =>
      expect(screen.getByTestId(selectors.dictationStudio.transcriptEditor)).toHaveValue("captured words"),
    );

    await user.click(screen.getByTestId(selectors.dictationStudio.saveClip));
    await waitFor(() =>
      expect(createClip).toHaveBeenCalledWith(
        expect.objectContaining({
          referenceText: "captured words",
          format: "pcm_s16le",
          sampleRateHz: 16_000,
          source: 1,
        }),
      ),
    );
    expect(pushToast).toHaveBeenCalled();
  });

  it("marks scripted captures with the scripted source and prompt text", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DictationStudioPage />);

    await user.click(screen.getByRole("button", { name: strings.dictationStudio.modeScripted }));
    await user.type(screen.getByTestId(selectors.dictationStudio.promptInput), "read this aloud");
    await user.click(screen.getByTestId("mock-capture"));

    await waitFor(() =>
      expect(screen.getByTestId(selectors.dictationStudio.transcriptEditor)).toHaveValue("read this aloud"),
    );
    await user.click(screen.getByTestId(selectors.dictationStudio.saveClip));
    await waitFor(() =>
      expect(createClip).toHaveBeenCalledWith(
        expect.objectContaining({ referenceText: "read this aloud", source: 2 }),
      ),
    );
  });

  it("selects a built-in script, pre-fills prompt text, and applies recommended tags", async () => {
    const user = userEvent.setup();
    const script = DICTATION_SCRIPTS[0]!;
    renderWithProviders(<DictationStudioPage />);

    await user.click(screen.getByRole("button", { name: strings.dictationStudio.modeScripted }));
    await user.selectOptions(screen.getByTestId(selectors.dictationStudio.scriptPicker), script.id);

    expect(screen.getByTestId(selectors.dictationStudio.promptInput)).toHaveValue(script.text);
    expect(screen.getByTestId(selectors.dictationStudio.promptInput)).toHaveAttribute("readonly");
    expect(screen.getByTestId(selectors.dictationStudio.transcriptEditor)).toHaveValue(script.text);
    expect(screen.getByTestId(selectors.dictationStudio.transcriptEditor)).toHaveAttribute("readonly");
    for (const tag of script.tags) {
      expect(screen.getByText(tag)).toBeInTheDocument();
    }

    await user.click(screen.getByTestId("mock-capture"));
    await waitFor(() =>
      expect(screen.getByTestId(selectors.dictationStudio.transcriptEditor)).toHaveValue(script.text),
    );
    await user.click(screen.getByTestId(selectors.dictationStudio.saveClip));

    await waitFor(() =>
      expect(createClip).toHaveBeenCalledWith(
        expect.objectContaining({ referenceText: script.text, tags: script.tags, source: 2 }),
      ),
    );
  });

  it("switches to the Corpus tab and lists clips", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DictationStudioPage />);
    await user.click(screen.getByRole("tab", { name: strings.dictationStudio.tabCorpus }));
    await waitFor(() => expect(listClips).toHaveBeenCalled());
    expect(await screen.findByText(strings.dictationStudio.corpusEmpty)).toBeInTheDocument();
  });

  it("switches to the Experiment Lab tab", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DictationStudioPage />);
    await user.click(screen.getByRole("tab", { name: strings.dictationStudio.tabLab }));
    expect(await screen.findByTestId(selectors.dictationStudio.startExperiment)).toBeInTheDocument();
  });
});
