import { create } from "@bufbuild/protobuf";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BackendState, BackendStatusSchema } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";
import { PolicyConfigSchema, PresetKind } from "@vrooli/proto-types/lighting-control/v1/policy/policy_pb";
import { LightingRuntimeStatusSchema, LightingSelectionSchema, InstallationSource } from "@vrooli/proto-types/lighting-control/v1/setup/setup_pb";
import { sessionApi } from "../api/session";
import { lightingSetupApi } from "../features/lights/setup";
import { lightingApi } from "../features/lights/api";
import { strings } from "../consts/strings";
import { selectors } from "../consts/selectors";
import { renderWithProviders } from "../test-utils";
import { SettingsPage } from "./SettingsPage";

vi.mock("../api/session", () => ({ sessionApi: { status: vi.fn().mockResolvedValue({ authenticated: false }) } }));

const config = create(PolicyConfigSchema, {
  defaultPreset: PresetKind.OFF, sourceMaxAgeSeconds: 60, loadEnterPercent: 80, loadExitPercent: 60,
  debounceSeconds: 5, brightnessCapPercent: 35, staleFallbackBrightnessPercent: 20,
});

describe("SettingsPage", () => {
  beforeEach(() => {
    vi.mocked(sessionApi.status).mockResolvedValue({ authenticated: false });
    vi.spyOn(lightingSetupApi, "status").mockResolvedValue(create(LightingRuntimeStatusSchema, { phase: "disabled", state: "disabled", pinnedSelection: create(LightingSelectionSchema, { source: InstallationSource.MANAGED, version: "1.1", sha256: "a".repeat(64), devicePolicyId: "approved-policy" }) }));
  });
  afterEach(() => vi.restoreAllMocks());

  it("reports backend readiness and saves guardrail changes before evaluation", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ backend: create(BackendStatusSchema, { state: BackendState.READY, sdkProtocolVersion: 6 }), controllers: [], zones: [], manualOverride: undefined, effective: undefined, config });
    const savePolicy = vi.spyOn(lightingApi, "savePolicy").mockResolvedValue({ config } as never);
    const evaluatePolicy = vi.spyOn(lightingApi, "evaluatePolicy").mockResolvedValue({} as never);
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><SettingsPage /></MemoryRouter>, { withoutRouter: true });

    expect(await screen.findByText(strings.lightDesk.settingsUi.setupTitle)).toBeInTheDocument();
    const [stale] = await screen.findAllByRole("slider");
    if (!stale) throw new Error("Staleness control was not rendered");
    fireEvent.change(stale, { target: { value: "120" } });
    fireEvent.click(screen.getByRole("button", { name: "lightDesk.settingsUi.savePolicy" }));
    await waitFor(() => expect(savePolicy).toHaveBeenCalledOnce());
    expect(savePolicy.mock.calls[0]?.[0].sourceMaxAgeSeconds).toBe(120);
    expect(evaluatePolicy).toHaveBeenCalledOnce();
  });

  it("offers explicit setup actions and reconciles saved lighting", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ backend: create(BackendStatusSchema, { state: BackendState.READY }), controllers: [], zones: [], manualOverride: undefined, effective: undefined, config });
    const reconcile = vi.spyOn(lightingApi, "reconcile").mockResolvedValue({} as never);
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><SettingsPage /></MemoryRouter>, { withoutRouter: true });
    expect(await screen.findByRole("button", { name: strings.lightDesk.settingsUi.prepareSetup })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.settingsUi.reconcile }));
    await waitFor(() => expect(reconcile).toHaveBeenCalledOnce());
    fireEvent.click(screen.getByTestId(selectors.settingsPage.themeSelect));
    fireEvent.click(await screen.findByRole("option", { name: strings.theme.choice.dark }));
  });

  it("shows backend diagnostics, edits bounded policy controls, and reports evaluation failures", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({
      backend: create(BackendStatusSchema, { state: BackendState.UNAVAILABLE, detail: "SDK socket is closed" }),
      controllers: [], zones: [], manualOverride: undefined, effective: undefined, config,
    });
    const savePolicy = vi.spyOn(lightingApi, "savePolicy").mockResolvedValue({ config } as never);
    vi.spyOn(lightingApi, "evaluatePolicy").mockRejectedValue(new Error("Policy evaluation failed"));
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><SettingsPage /></MemoryRouter>, { withoutRouter: true });

    expect(await screen.findByText(/SDK socket is closed/)).toBeInTheDocument();
    const sliders = await screen.findAllByRole("slider");
    expect(sliders).toHaveLength(6);
    fireEvent.change(sliders[1]!, { target: { value: "20" } });
    fireEvent.change(sliders[2]!, { target: { value: "50" } });
    fireEvent.change(sliders[3]!, { target: { value: "30" } });
    fireEvent.change(sliders[4]!, { target: { value: "10" } });
    fireEvent.change(screen.getByLabelText(strings.lightDesk.settingsUi.fallbackColor), { target: { value: "#112233" } });
    fireEvent.change(sliders[5]!, { target: { value: "10" } });
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.settingsUi.savePolicy }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Policy evaluation failed");
    expect(savePolicy).toHaveBeenCalledOnce();
    expect(savePolicy.mock.calls[0]?.[0]).toMatchObject({
      brightnessCapPercent: 20, loadEnterPercent: 50, loadExitPercent: 30,
      debounceSeconds: 10, staleFallbackBrightnessPercent: 10,
      staleFallbackColor: { red: 17, green: 34, blue: 51 },
    });
  });

  it("surfaces read errors and keeps setup actions safe when state is unknown", async () => {
    const refresh = vi.spyOn(lightingApi, "readDesk").mockRejectedValue(new Error("Backend status unavailable"));
    const reconcile = vi.spyOn(lightingApi, "reconcile");
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><SettingsPage /></MemoryRouter>, { withoutRouter: true });

    expect(await screen.findByRole("alert")).toHaveTextContent(strings.lightDesk.settingsUi.statusUnavailable);
    expect(screen.getByRole("button", { name: strings.lightDesk.settingsUi.reconcile })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.settingsUi.retry }));
    await waitFor(() => expect(refresh).toHaveBeenCalledTimes(2));
    expect(reconcile).not.toHaveBeenCalled();
    expect(screen.getByText(strings.lightDesk.settingsUi.notAvailable)).toBeInTheDocument();
  });

  it("labels an unreported backend without inventing readiness details", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined, config });
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><SettingsPage /></MemoryRouter>, { withoutRouter: true });
    expect(await screen.findByText(/lightDesk\.settingsUi\.noDetail/)).toBeInTheDocument();
    expect(screen.getByText(strings.lightDesk.settingsUi.notChecked)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: strings.lightDesk.settingsUi.reconcile })).toBeDisabled();
  });
});
