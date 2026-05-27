import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../../consts/selectors";
import { AudioFormat } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";

const getWakeWordConfig = vi.fn();
const saveWakeWordTemplate = vi.fn();
const deleteWakeWordTemplate = vi.fn();

vi.mock("../../services/wakeWord", () => ({
  getWakeWordConfig: () => getWakeWordConfig(),
  saveWakeWordTemplate: (tpl: unknown) => saveWakeWordTemplate(tpl),
  deleteWakeWordTemplate: () => deleteWakeWordTemplate(),
}));

const recordWakeWordSample = vi.fn();
vi.mock("./wakeWordRecorder", () => ({
  recordWakeWordSample: () => recordWakeWordSample(),
}));

vi.mock("../../components/ui/toast", () => ({ pushToast: vi.fn() }));

import { WakeWordPage } from "./WakeWordPage";

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <WakeWordPage />
    </QueryClientProvider>,
  );
}

// A recorder handle whose `done` resolves immediately with a fixed sample.
function fakeRecorder() {
  return Promise.resolve({
    done: Promise.resolve({
      audio: new Uint8Array([1, 2, 3]),
      format: AudioFormat.WEBM,
      sampleRateHz: 48000,
    }),
    stop: vi.fn(),
  });
}

async function recordN(startCount: number, n: number) {
  for (let i = 0; i < n; i++) {
    await userEvent.click(screen.getByTestId(selectors.wakeWord.record));
    // wait until the machine-readable sample count reflects the new sample
    // (cimode renders the i18n *key*, so we read the data-count attribute).
    const expected = startCount + i + 1;
    await waitFor(() =>
      expect(screen.getByTestId(selectors.wakeWord.sampleCount)).toHaveAttribute(
        "data-count",
        String(expected),
      ),
    );
  }
}

beforeEach(() => {
  vi.clearAllMocks();
  getWakeWordConfig.mockResolvedValue({ configured: false });
  saveWakeWordTemplate.mockResolvedValue({ configured: true });
  deleteWakeWordTemplate.mockResolvedValue({ configured: false });
  recordWakeWordSample.mockImplementation(fakeRecorder);
});

describe("WakeWordPage enrollment", () => {
  it("enforces a minimum of 3 samples before save is enabled", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByTestId(selectors.wakeWord.record)).toBeInTheDocument());

    await userEvent.type(screen.getByTestId(selectors.wakeWord.label), "hey vrooli");

    // 2 samples: save still disabled
    await recordN(0, 2);
    expect(screen.getByTestId(selectors.wakeWord.save)).toBeDisabled();

    // 3rd sample: save enabled
    await recordN(2, 1);
    expect(screen.getByTestId(selectors.wakeWord.save)).toBeEnabled();
  });

  it("caps recording at 5 samples (record disabled at max)", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByTestId(selectors.wakeWord.record)).toBeInTheDocument());

    await recordN(0, 5);
    expect(screen.getByTestId(selectors.wakeWord.record)).toBeDisabled();
    // a 6th attempt is impossible — the button is disabled and the count is capped
    expect(screen.getByTestId(selectors.wakeWord.sampleCount)).toHaveAttribute("data-count", "5");
  });

  it("threshold slider is bounded to 0.1–0.95 with 0.05 steps", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByTestId(selectors.wakeWord.threshold)).toBeInTheDocument());

    const slider = screen.getByTestId(selectors.wakeWord.threshold);
    expect(slider).toHaveAttribute("min", "0.1");
    expect(slider).toHaveAttribute("max", "0.95");
    expect(slider).toHaveAttribute("step", "0.05");
  });

  it("save sends label, threshold, and the recorded samples", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByTestId(selectors.wakeWord.record)).toBeInTheDocument());

    await userEvent.type(screen.getByTestId(selectors.wakeWord.label), "hey vrooli");
    await recordN(0, 3);
    await userEvent.click(screen.getByTestId(selectors.wakeWord.save));

    await waitFor(() => expect(saveWakeWordTemplate).toHaveBeenCalledTimes(1));
    const arg = saveWakeWordTemplate.mock.calls[0]?.[0] as {
      label: string;
      threshold: number;
      samples: unknown[];
    };
    expect(arg.label).toBe("hey vrooli");
    expect(arg.threshold).toBeGreaterThanOrEqual(0.1);
    expect(arg.threshold).toBeLessThanOrEqual(0.95);
    expect(arg.samples).toHaveLength(3);
  });

  it("delete calls deleteWakeWordTemplate when a template is configured", async () => {
    getWakeWordConfig.mockResolvedValue({
      configured: true,
      template: { label: "hey vrooli", threshold: 0.6, samples: [{ audio: new Uint8Array(), format: AudioFormat.WEBM, sampleRateHz: 48000 }] },
    });
    renderPage();
    await waitFor(() => expect(screen.getByTestId(selectors.wakeWord.delete)).toBeEnabled());

    await userEvent.click(screen.getByTestId(selectors.wakeWord.delete));
    await waitFor(() => expect(deleteWakeWordTemplate).toHaveBeenCalledTimes(1));
  });
});
