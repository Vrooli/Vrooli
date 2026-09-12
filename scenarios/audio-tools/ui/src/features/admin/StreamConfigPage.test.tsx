import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { renderWithProviders as render } from "@vrooli/api-base/testing";

const listEngines = vi.fn();
const getEngineSwitchImpact = vi.fn();
const setEngine = vi.fn();

vi.mock("../../services/sttEngines", () => ({
  listEngines: () => listEngines(),
  getEngineSwitchImpact: (id: string) => getEngineSwitchImpact(id),
  setEngine: (id: string) => setEngine(id),
}));

const pushToast = vi.fn();
vi.mock("../../components/ui/toast", () => ({
  pushToast: (...args: unknown[]) => pushToast(...args),
}));

// OverlapStallGuard (mounted by the page) reads/writes StreamConfig via the
// audio-integration voice API; stub those two calls so the engine-picker
// suite stays focused and offline.
const getVoiceStreamConfig = vi.fn();
const updateVoiceStreamConfig = vi.fn();
vi.mock("../../audio-integration", () => ({
  getVoiceStreamConfig: () => getVoiceStreamConfig(),
  updateVoiceStreamConfig: (patch: { overlapMaxStallRejects?: number }) => updateVoiceStreamConfig(patch),
}));

import { StreamConfigPage } from "./StreamConfigPage";

const ENGINES = [
  { id: "whisper-local", displayName: "Whisper (local)", kind: "local_resource", available: true, nativeStreaming: false, isActive: true },
  { id: "deepgram", displayName: "Deepgram", kind: "byok_api", available: true, nativeStreaming: true, isActive: false },
  { id: "vosk-local", displayName: "Vosk (local)", kind: "local_resource", available: false, nativeStreaming: false, isActive: false },
];

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<StreamConfigPage />, { queryClient: qc });
}

beforeEach(() => {
  vi.clearAllMocks();
  listEngines.mockResolvedValue(ENGINES);
  setEngine.mockResolvedValue(undefined);
  getVoiceStreamConfig.mockResolvedValue({ overlapMaxStallRejects: 3 });
  updateVoiceStreamConfig.mockResolvedValue({ overlapMaxStallRejects: 3 });
});

describe("StreamConfigPage engine picker", () => {
  it("renders each engine with its kind, streaming badge, and availability", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId(selectors.streamConfig.engineRow({ id: "whisper-local" }))).toBeInTheDocument(),
    );

    // native-streaming badge for the Deepgram row (cimode renders the typed key path)
    const deepgramRow = screen.getByTestId(selectors.streamConfig.engineRow({ id: "deepgram" }));
    expect(within(deepgramRow).getByText(strings.streamConfigAdmin.badgeNativeStreaming)).toBeInTheDocument();
    // batch badge for the non-streaming Whisper row
    const whisperRow = screen.getByTestId(selectors.streamConfig.engineRow({ id: "whisper-local" }));
    expect(within(whisperRow).getByText(strings.streamConfigAdmin.badgeBatch)).toBeInTheDocument();
    // active engine is flagged (badge + disabled button both read "Active")
    const activeRow = screen.getByTestId(selectors.streamConfig.engineRow({ id: "whisper-local" }));
    expect(within(activeRow).getAllByText(strings.streamConfigAdmin.engineActive).length).toBeGreaterThan(0);
    expect(screen.getByTestId(selectors.streamConfig.engineSelect({ id: "whisper-local" }))).toBeDisabled();
    // unavailable engine shows the resource-unavailable subtext and a disabled button
    const voskRow = screen.getByTestId(selectors.streamConfig.engineRow({ id: "vosk-local" }));
    expect(within(voskRow).getByText(strings.streamConfigAdmin.engineUnavailable)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.streamConfig.engineSelect({ id: "vosk-local" }))).toBeDisabled();
  });

  it("switching a non-active engine triggers the impact prompt before committing", async () => {
    getEngineSwitchImpact.mockResolvedValue({
      resource: "whisper",
      consumers: [],
      safeToStop: true,
      stopCommand: "vrooli resource stop whisper",
      consumersKnown: true,
    });
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId(selectors.streamConfig.engineRow({ id: "deepgram" }))).toBeInTheDocument(),
    );

    await userEvent.click(screen.getByTestId(selectors.streamConfig.engineSelect({ id: "deepgram" })));

    // impact is fetched against the CURRENTLY-ACTIVE engine, not the target
    await waitFor(() => expect(getEngineSwitchImpact).toHaveBeenCalledWith("whisper-local"));
    // prompt appears; the resource is not committed yet
    expect(await screen.findByTestId(selectors.streamConfig.switchPrompt)).toBeInTheDocument();
    expect(screen.getByText(/vrooli resource stop whisper/)).toBeInTheDocument();
    expect(setEngine).not.toHaveBeenCalled();
  });

  it("confirming the prompt calls setEngine with the target id and toasts", async () => {
    getEngineSwitchImpact.mockResolvedValue({
      resource: "whisper",
      consumers: [],
      safeToStop: true,
      stopCommand: "vrooli resource stop whisper",
      consumersKnown: true,
    });
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId(selectors.streamConfig.engineRow({ id: "deepgram" }))).toBeInTheDocument(),
    );

    await userEvent.click(screen.getByTestId(selectors.streamConfig.engineSelect({ id: "deepgram" })));
    await screen.findByTestId(selectors.streamConfig.switchPrompt);

    await userEvent.click(screen.getByTestId(selectors.streamConfig.confirmSwitch));

    await waitFor(() => expect(setEngine).toHaveBeenCalledWith("deepgram"));
    await waitFor(() => expect(pushToast).toHaveBeenCalled());
    // prompt closes after a successful switch
    await waitFor(() =>
      expect(screen.queryByTestId(selectors.streamConfig.switchPrompt)).not.toBeInTheDocument(),
    );
  });

  it("lists other consumers and recommends leaving the resource running", async () => {
    getEngineSwitchImpact.mockResolvedValue({
      resource: "whisper",
      consumers: [
        { scenario: "voice-notes", displayName: "Voice Notes", required: true },
        { scenario: "meeting-bot", displayName: "Meeting Bot", required: false },
      ],
      safeToStop: false,
      stopCommand: "vrooli resource stop whisper",
      consumersKnown: true,
    });
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId(selectors.streamConfig.engineRow({ id: "deepgram" }))).toBeInTheDocument(),
    );

    await userEvent.click(screen.getByTestId(selectors.streamConfig.engineSelect({ id: "deepgram" })));
    const prompt = await screen.findByTestId(selectors.streamConfig.switchPrompt);

    expect(within(prompt).getByText(/Voice Notes/)).toBeInTheDocument();
    expect(within(prompt).getByText(/Meeting Bot/)).toBeInTheDocument();
    // the required consumer is flagged (cimode renders the typed key path)
    expect(within(prompt).getByText(strings.streamConfigAdmin.consumerRequired)).toBeInTheDocument();
  });

  it("cancelling the prompt does not switch the engine", async () => {
    getEngineSwitchImpact.mockResolvedValue({
      resource: "",
      consumers: [],
      safeToStop: true,
      stopCommand: "",
      consumersKnown: true,
    });
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId(selectors.streamConfig.engineRow({ id: "deepgram" }))).toBeInTheDocument(),
    );

    await userEvent.click(screen.getByTestId(selectors.streamConfig.engineSelect({ id: "deepgram" })));
    await screen.findByTestId(selectors.streamConfig.switchPrompt);

    await userEvent.click(screen.getByTestId(selectors.streamConfig.cancelSwitch));

    await waitFor(() =>
      expect(screen.queryByTestId(selectors.streamConfig.switchPrompt)).not.toBeInTheDocument(),
    );
    expect(setEngine).not.toHaveBeenCalled();
  });
});
