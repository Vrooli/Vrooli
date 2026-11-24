import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { makeApiError } from "../../api/client";
import type { ModelRow } from "../../services/models";
import { renderWithProviders as render } from "@vrooli/api-base/testing";

const listEngines = vi.fn();
const getEngineSwitchImpact = vi.fn();
const setEngine = vi.fn();
const listModels = vi.fn();

vi.mock("../../services/sttEngines", () => ({
  listEngines: () => listEngines(),
  getEngineSwitchImpact: (id: string) => getEngineSwitchImpact(id),
  setEngine: (id: string) => setEngine(id),
}));

vi.mock("../../services/models", () => ({
  listModels: (filters: unknown) => listModels(filters),
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

function makeModel(
  overrides: Partial<ModelRow> & Pick<ModelRow, "id" | "name" | "engineIds">,
): ModelRow {
  return {
    providerIds: [],
    engineDefaults: [],
    capability: "stt",
    runtime: "faster-whisper",
    languages: ["en"],
    sizeMbApprox: 0,
    params: "",
    license: "MIT",
    commercialUse: "allowed",
    licenseNotes: "",
    sourceUrl: "",
    sourceRepo: "",
    readiness: "available",
    roles: [],
    defaultEligible: true,
    isDefault: false,
    allowDefault: true,
    hardware: {
      cpuCapable: true,
      gpuRequired: false,
      minVramGb: 0,
      minRamGb: 0,
      osArch: [],
    },
    benchmarks: { wer: 0, sim: 0, rtf: 0, notes: "" },
    notes: "",
    ...overrides,
  };
}

const MODELS: ModelRow[] = [
  makeModel({
    id: "whisper-large-v3",
    name: "Whisper Large v3",
    engineIds: ["whisper-local"],
    runtime: "faster-whisper",
    sizeMbApprox: 3090,
    languages: ["en", "ja", "es", "fr"],
    isDefault: true,
  }),
  makeModel({
    id: "whisper-small",
    name: "Whisper Small",
    engineIds: ["whisper-local"],
    runtime: "faster-whisper",
    sizeMbApprox: 484,
    languages: ["en", "ja"],
    readiness: "candidate",
    commercialUse: "unknown",
  }),
  // Two registry records claim the default for this engine so the "mark only
  // the first" rule is exercised.
  makeModel({
    id: "deepgram-nova",
    name: "Deepgram Nova",
    engineIds: ["deepgram"],
    runtime: "deepgram",
    languages: ["en"],
    isDefault: true,
  }),
  makeModel({
    id: "deepgram-flux",
    name: "Deepgram Flux",
    engineIds: ["deepgram"],
    runtime: "deepgram",
    languages: ["en"],
    readiness: "experimental",
    commercialUse: "unknown",
    isDefault: true,
  }),
];

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<StreamConfigPage />, { queryClient: qc });
}

beforeEach(() => {
  vi.clearAllMocks();
  listEngines.mockResolvedValue(ENGINES);
  listModels.mockResolvedValue({ ok: true, data: MODELS });
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

describe("StreamConfigPage model registry facts", () => {
  it("renders registry facts for the models declared for an engine", async () => {
    renderPage();
    const row = await screen.findByTestId(
      selectors.streamConfig.engineRow({ id: "whisper-local" }),
    );

    expect(await within(row).findByText(/Whisper Large v3/)).toBeInTheDocument();
    expect(within(row).getByText(strings.models.readiness.available)).toBeInTheDocument();
    expect(within(row).getByText(strings.models.commercial.allowed)).toBeInTheDocument();
    expect(within(row).getByText(strings.streamConfigAdmin.modelsTitle)).toBeInTheDocument();

    const large = within(row).getByText(/Whisper Large v3/).closest("li");
    expect(large).not.toBeNull();
    if (!large) return;

    expect(within(large).getByText(/whisper-large-v3/)).toBeInTheDocument();
    // runtime is raw data, so it renders verbatim in cimode
    expect(within(large).getByText(/faster-whisper/)).toBeInTheDocument();
    expect(
      within(large).getByText(strings.streamConfigAdmin.modelSizeLabel, { exact: false }),
    ).toBeInTheDocument();
    expect(
      within(large).getByText(strings.streamConfigAdmin.modelLanguagesLabel, { exact: false }),
    ).toBeInTheDocument();
    expect(within(large).getByText(/en, ja, es/)).toBeInTheDocument();
    expect(within(large).getByText(strings.models.languageOverflow)).toBeInTheDocument();

    // one query for the whole page, scoped to the STT capability
    expect(listModels).toHaveBeenCalledTimes(1);
    expect(listModels).toHaveBeenCalledWith({ capability: "stt" });
  });

  it("shows an inline loading line per engine without blocking the picker", async () => {
    listModels.mockReturnValue(new Promise(() => {}));
    renderPage();

    const row = await screen.findByTestId(
      selectors.streamConfig.engineRow({ id: "whisper-local" }),
    );
    expect(
      within(row).getByText(strings.streamConfigAdmin.modelsLoading),
    ).toBeInTheDocument();
    // the engine list and its actions are usable while the registry is pending
    expect(screen.getByTestId(selectors.streamConfig.engineSelect({ id: "deepgram" }))).toBeEnabled();
  });

  it("marks only the first default when several records claim the default", async () => {
    renderPage();
    const row = await screen.findByTestId(
      selectors.streamConfig.engineRow({ id: "deepgram" }),
    );

    const first = (await within(row).findByText(/Deepgram Nova/)).closest("li");
    const second = within(row).getByText(/Deepgram Flux/).closest("li");
    expect(first).not.toBeNull();
    expect(second).not.toBeNull();
    if (!first || !second) return;

    expect(within(first).getByText(strings.models.defaultBadge)).toBeInTheDocument();
    // both records are listed, but the badge marks only the first
    expect(within(row).getAllByText(strings.models.defaultBadge)).toHaveLength(1);
    expect(within(second).queryByText(strings.models.defaultBadge)).not.toBeInTheDocument();
  });

  it("keeps a non-default candidate model visible with its readiness badge", async () => {
    renderPage();
    const row = await screen.findByTestId(
      selectors.streamConfig.engineRow({ id: "whisper-local" }),
    );

    const small = (await within(row).findByText(/Whisper Small/)).closest("li");
    expect(small).not.toBeNull();
    if (!small) return;

    expect(within(small).getByText(strings.models.readiness.candidate)).toBeInTheDocument();
    expect(within(small).getByText(strings.models.commercial.unknown)).toBeInTheDocument();
    expect(within(small).queryByText(strings.models.defaultBadge)).not.toBeInTheDocument();
  });

  it("shows an explicit empty state for an engine with no registered models", async () => {
    renderPage();
    const row = await screen.findByTestId(
      selectors.streamConfig.engineRow({ id: "vosk-local" }),
    );

    expect(
      await within(row).findByText(strings.streamConfigAdmin.modelsEmpty),
    ).toBeInTheDocument();
    expect(within(row).queryByText(strings.models.defaultBadge)).not.toBeInTheDocument();
    // the row itself still renders normally
    expect(within(row).getByText(strings.streamConfigAdmin.engineUnavailable)).toBeInTheDocument();
  });

  it("shows an inline registry error with retry and still allows switching engines", async () => {
    listModels.mockResolvedValue({
      ok: false,
      error: makeApiError("unavailable", "registry down", 503),
    });
    getEngineSwitchImpact.mockResolvedValue({
      resource: "whisper",
      consumers: [],
      safeToStop: true,
      stopCommand: "vrooli resource stop whisper",
      consumersKnown: true,
    });

    renderPage();
    const row = await screen.findByTestId(
      selectors.streamConfig.engineRow({ id: "whisper-local" }),
    );

    expect(
      await within(row).findByText(strings.streamConfigAdmin.modelsLoadError),
    ).toBeInTheDocument();
    expect(within(row).queryByText(strings.streamConfigAdmin.modelsEmpty)).not.toBeInTheDocument();

    await userEvent.click(
      within(row).getByRole("button", { name: strings.streamConfigAdmin.modelsRetry }),
    );
    await waitFor(() => expect(listModels).toHaveBeenCalledTimes(2));

    // engine selection is unaffected by the registry failure
    await userEvent.click(screen.getByTestId(selectors.streamConfig.engineSelect({ id: "deepgram" })));
    await waitFor(() => expect(getEngineSwitchImpact).toHaveBeenCalledWith("whisper-local"));
    expect(await screen.findByTestId(selectors.streamConfig.switchPrompt)).toBeInTheDocument();
  });
});
