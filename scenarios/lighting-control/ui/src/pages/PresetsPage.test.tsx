import { create } from "@bufbuild/protobuf";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ActivityEffectsSchema, PolicyConfigSchema, PresetKind } from "@vrooli/proto-types/lighting-control/v1/policy/policy_pb";
import { ColorSchema, ControllerSchema, ManualOverrideSchema } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";
import { lightingApi } from "../features/lights/api";
import { strings } from "../consts/strings";
import { selectors } from "../consts/selectors";
import { renderWithProviders } from "../test-utils";
import { PresetsPage } from "./PresetsPage";

const config = create(PolicyConfigSchema, {
  defaultPreset: PresetKind.OFF,
  presets: [
    { kind: PresetKind.OFF, label: "Off", enabled: true },
    { kind: PresetKind.LOAD, label: "Load", enabled: true, color: create(ColorSchema, { red: 255, green: 112, blue: 48 }), brightnessCapPercent: 30 },
    { kind: PresetKind.WORK, label: "Work", enabled: true, color: create(ColorSchema, { red: 48, green: 128, blue: 224 }), brightnessCapPercent: 25 },
    { kind: PresetKind.HEALTH, label: "Health", enabled: true, color: create(ColorSchema, { red: 217, green: 77, blue: 77 }), brightnessCapPercent: 35 },
    { kind: PresetKind.COMBINED, label: "Combined", enabled: true, color: create(ColorSchema, { red: 40, green: 176, blue: 120 }), brightnessCapPercent: 25, effectId: "Breathing" },
  ],
  sourceMaxAgeSeconds: 60,
  loadEnterPercent: 80,
  loadExitPercent: 60,
  debounceSeconds: 5,
  brightnessCapPercent: 35,
  staleFallbackColor: create(ColorSchema, { red: 32, green: 48, blue: 64 }),
  staleFallbackBrightnessPercent: 20,
  healthPalette: {
    healthy: create(ColorSchema, { red: 87, green: 190, blue: 138 }),
    attention: create(ColorSchema, { red: 217, green: 154, blue: 36 }),
    failure: create(ColorSchema, { red: 217, green: 77, blue: 77 }),
  },
  activityEffects: { activeEffectId: "Breathing", busyEffectId: "Breathing", busyAgentThreshold: 5 },
});

describe("PresetsPage", () => {
  afterEach(() => { vi.restoreAllMocks(); localStorage.clear(); });

  it("stores a draft locally without changing the active policy or sending a command", async () => {
    const save = vi.spyOn(lightingApi, "savePolicy").mockResolvedValue({ config } as never);
    const choose = vi.spyOn(lightingApi, "choosePreset").mockResolvedValue({} as never);
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ config, backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined });
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });

    fireEvent.click(await screen.findByRole("button", { name: /Health/ }));
    const [healthyColorInput] = await screen.findAllByLabelText(strings.lightDesk.presetsUi.colorAria);
    if (!healthyColorInput) throw new Error("Healthy color control was not rendered");
    fireEvent.change(healthyColorInput, { target: { value: "#40a0d0" } });
    fireEvent.change(screen.getByLabelText(strings.lightDesk.presetsUi.presetName), { target: { value: "Careful health" } });
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.presetsUi.saveDraft }));

    await waitFor(() => expect(localStorage.getItem("lighting-control.preset-drafts.v1")).not.toBeNull());
    expect(save).not.toHaveBeenCalled();
    expect(choose).not.toHaveBeenCalled();
    const persisted = JSON.parse(localStorage.getItem("lighting-control.preset-drafts.v1") ?? "{}") as Record<string, { preset: { label: string }; health: { healthy: { red: number; green: number; blue: number } } }>;
    expect(persisted["4"]?.preset.label).toBe("Careful health");
    expect(persisted["4"]?.health.healthy).toMatchObject({ red: 64, green: 160, blue: 208 });
  });

  it("restores saved drafts when returning to the preset editor", async () => {
    localStorage.setItem("lighting-control.preset-drafts.v1", JSON.stringify({
      4: { preset: { ...config.presets[3], label: "Saved health draft" }, health: config.healthPalette, activity: config.activityEffects },
    }));
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ config, backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined });
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });
    fireEvent.click(await screen.findByRole("button", { name: /Health/ }));
    expect(screen.getByLabelText(strings.lightDesk.presetsUi.presetName)).toHaveValue("Saved health draft");
  });

  it("persists and selects a preset only after Save and use", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ config, backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined });
    const save = vi.spyOn(lightingApi, "savePolicy").mockResolvedValue({ config } as never);
    const choose = vi.spyOn(lightingApi, "choosePreset").mockResolvedValue({} as never);
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });
    fireEvent.click(await screen.findByRole("button", { name: /Health/ }));
    fireEvent.change(screen.getByLabelText(strings.lightDesk.presetsUi.presetName), { target: { value: "Careful health" } });
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.presetsUi.saveUse }));
    await waitFor(() => expect(save).toHaveBeenCalledOnce());
    expect(choose).toHaveBeenCalledWith(PresetKind.HEALTH);
    expect(save.mock.calls[0]?.[0].presets.find((preset) => preset.kind === PresetKind.HEALTH)?.label).toBe("Careful health");
    expect(config.presets.find((preset) => preset.kind === PresetKind.HEALTH)?.label).toBe("Health");
  });

  it("saves the configurable activity threshold as a local draft", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ config, backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined });
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });
    fireEvent.click(await screen.findByRole("button", { name: /Work/ }));
    fireEvent.change(screen.getByLabelText(strings.lightDesk.presetsUi.activityThreshold), { target: { value: "12" } });
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.presetsUi.saveDraft }));
    await waitFor(() => {
      const saved = JSON.parse(localStorage.getItem("lighting-control.preset-drafts.v1") ?? "{}") as Record<string, { activity: { busyAgentThreshold: number } }>;
      expect(saved[String(PresetKind.WORK)]?.activity.busyAgentThreshold).toBe(12);
    });
  });

  it("reports saved configuration separately from an unconfirmed selection", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ config, backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined });
    const save = vi.spyOn(lightingApi, "savePolicy").mockRejectedValueOnce(new Error("Policy storage unavailable"));
    const choose = vi.spyOn(lightingApi, "choosePreset").mockRejectedValueOnce(new Error("Policy evaluation unavailable"));
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });
    fireEvent.click(await screen.findByRole("button", { name: /health/i }));
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.presetsUi.saveUse }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Policy storage unavailable");
    expect(choose).not.toHaveBeenCalled();

    save.mockResolvedValueOnce({ config } as never);
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.presetsUi.saveUse }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Policy evaluation unavailable");
    expect(screen.getByRole("status")).toHaveTextContent(strings.lightDesk.presetsUi.savedButNotApplied);
    expect(save).toHaveBeenCalledTimes(2);
    expect(choose).toHaveBeenCalledWith(PresetKind.HEALTH);
  });

  it("shows loading and unavailable policy states", async () => {
    const pending = vi.spyOn(lightingApi, "readDesk").mockReturnValue(new Promise(() => undefined));
    const loading = renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });
    expect(await screen.findByRole("status")).toHaveTextContent(strings.lightDesk.presetsUi.loading);
    loading.unmount();
    pending.mockRejectedValue(new Error("Saved policy is unavailable"));
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });
    expect(await screen.findByRole("alert")).toHaveTextContent("Saved policy is unavailable");
  });

  it("keeps health and activity drafts with their selected preset", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ config, backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined });
    const save = vi.spyOn(lightingApi, "savePolicy").mockResolvedValue({ config } as never);
    vi.spyOn(lightingApi, "choosePreset").mockResolvedValue({} as never);
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });
    fireEvent.click(await screen.findByRole("button", { name: /Health/ }));
    fireEvent.change(screen.getAllByLabelText(strings.lightDesk.presetsUi.colorAria)[0]!, { target: { value: "#123456" } });
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.presetsUi.saveDraft }));
    fireEvent.click(screen.getByRole("button", { name: /Work/ }));
    fireEvent.change(screen.getByLabelText(strings.lightDesk.presetsUi.activityThreshold), { target: { value: "12" } });
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.presetsUi.saveDraft }));
    fireEvent.click(screen.getByRole("button", { name: /Combined/ }));
    expect(screen.getAllByLabelText(strings.lightDesk.presetsUi.colorAria)[0]).toHaveValue("#57be8a");
    expect(screen.getByLabelText(strings.lightDesk.presetsUi.activityThreshold)).toHaveValue("5");
    fireEvent.click(screen.getByRole("button", { name: /Health/ }));
    expect(screen.getAllByLabelText(strings.lightDesk.presetsUi.colorAria)[0]).toHaveValue("#123456");
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.presetsUi.saveUse }));
    await waitFor(() => expect(save).toHaveBeenCalledOnce());
    expect(save.mock.calls[0]![0].healthPalette?.healthy).toMatchObject({ red: 18, green: 52, blue: 86 });
    expect(save.mock.calls[0]![0].activityEffects?.busyAgentThreshold).toBe(5);
  });

  it("preserves an unsaved draft when its already-selected card is pressed", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ config, backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined });
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });
    fireEvent.change(await screen.findByLabelText(strings.lightDesk.presetsUi.presetName), { target: { value: "Quiet workbench" } });
    fireEvent.click(screen.getByRole("button", { name: /Combined/ }));
    expect(screen.getByLabelText(strings.lightDesk.presetsUi.presetName)).toHaveValue("Quiet workbench");
  });

  it("keeps manual Off active while previewing or saving a draft", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ config, backend: undefined, controllers: [], zones: [], manualOverride: create(ManualOverrideSchema, { active: true, allEnrolled: true }), effective: undefined });
    const save = vi.spyOn(lightingApi, "savePolicy");
    const choose = vi.spyOn(lightingApi, "choosePreset");
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });
    const off = await screen.findByRole("button", { name: /^Off / });
    expect(within(off).getByText(strings.lightDesk.presetsUi.active)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Health/ }));
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.presetsUi.saveDraft }));
    expect(screen.getByText(strings.lightDesk.presetsUi.manualOff)).toBeInTheDocument();
    expect(within(off).getByText(strings.lightDesk.presetsUi.active)).toBeInTheDocument();
    expect(save).not.toHaveBeenCalled();
    expect(choose).not.toHaveBeenCalled();
  });

  it("reports Off persistence errors and supports a successful retry", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ config, backend: undefined, controllers: [create(ControllerSchema, { enrolled: true })], zones: [], manualOverride: undefined, effective: undefined });
    const off = vi.spyOn(lightingApi, "setOff").mockRejectedValueOnce(new Error("Off could not be persisted")).mockResolvedValueOnce({} as never);
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });
    fireEvent.click(await screen.findByRole("button", { name: strings.lightDesk.presetsUi.keepOff }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Off could not be persisted");
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.presetsUi.keepOff }));
    expect(await screen.findByRole("status")).toHaveTextContent(strings.lightDesk.controllerUi.offSaved);
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(off).toHaveBeenCalledTimes(2);
  });

  it("reports remote success even when browser draft cleanup fails", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ config, backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined });
    const save = vi.spyOn(lightingApi, "savePolicy").mockResolvedValue({ config } as never);
    const choose = vi.spyOn(lightingApi, "choosePreset").mockResolvedValue({} as never);
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new Error("Browser storage full"); });
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });
    fireEvent.click(await screen.findByRole("button", { name: strings.lightDesk.presetsUi.saveUse }));
    expect(await screen.findByRole("status")).toHaveTextContent(strings.lightDesk.presetsUi.savedUsed);
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(save).toHaveBeenCalledOnce();
    expect(choose).toHaveBeenCalledOnce();
  });

  it("allows editing with blocked storage and reports local draft failures", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ config, backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined });
    const getItem = Storage.prototype.getItem.bind(localStorage);
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(function (this: Storage, key) {
      if (key === "lighting-control.preset-drafts.v1") throw new Error("Browser storage blocked");
      return getItem(key);
    });
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });
    fireEvent.change(await screen.findByLabelText(strings.lightDesk.presetsUi.presetName), { target: { value: "Private workbench" } });
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.presetsUi.saveDraft }));
    expect(await screen.findByRole("alert")).toHaveTextContent(strings.lightDesk.presetsUi.draftStorageFailed);
    expect(screen.getByLabelText(strings.lightDesk.presetsUi.presetName)).toHaveValue("Private workbench");
  });

  it("replaces corrupt browser draft data through a normal Save draft action", async () => {
    localStorage.setItem("lighting-control.preset-drafts.v1", "{corrupt");
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ config, backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined });
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });
    fireEvent.change(await screen.findByLabelText(strings.lightDesk.presetsUi.presetName), { target: { value: "Recovered draft" } });
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.presetsUi.saveDraft }));
    expect(await screen.findByRole("status")).toHaveTextContent(strings.lightDesk.presetsUi.savedDraft);
    const saved = JSON.parse(localStorage.getItem("lighting-control.preset-drafts.v1") ?? "{}") as Record<string, { preset: { label: string } }>;
    expect(saved[String(PresetKind.COMBINED)]?.preset.label).toBe("Recovered draft");
  });

  it("keeps the selected draft and all commands locked through save and selection", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ config, backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined });
    vi.spyOn(lightingApi, "savePolicy").mockResolvedValue({ config } as never);
    let finishSelection: () => void = () => undefined;
    const choose = vi.spyOn(lightingApi, "choosePreset").mockImplementation(() => new Promise((resolve) => { finishSelection = () => resolve({} as never); }));
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });
    fireEvent.click(await screen.findByRole("button", { name: strings.lightDesk.presetsUi.saveUse }));
    await waitFor(() => expect(choose).toHaveBeenCalledWith(PresetKind.COMBINED));
    expect(screen.getByRole("button", { name: strings.lightDesk.presetsUi.saveUse })).toBeDisabled();
    expect(screen.getByRole("button", { name: /Health/ })).toBeDisabled();
    finishSelection();
    expect(await screen.findByRole("status")).toHaveTextContent(strings.lightDesk.presetsUi.savedUsed);
    expect(screen.getByRole("button", { name: /Health/ })).toBeEnabled();
  });

  it("composes a single phone action footer at page level outside the editor", async () => {
    vi.spyOn(window, "matchMedia").mockImplementation((query) => ({ matches: query === "(max-width: 720px)", media: query, onchange: null, addEventListener: () => undefined, removeEventListener: () => undefined, addListener: () => undefined, removeListener: () => undefined, dispatchEvent: () => false } as MediaQueryList));
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ config, backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined });
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });
    const actions = await screen.findByTestId(selectors.lightDesk.presetActions);
    const footer = actions.closest(".lightdesk-mobile-footer");
    expect(footer?.parentElement).toBe(screen.getByTestId(selectors.pages.presets));
    expect(within(footer as HTMLElement).getByTestId(selectors.lightDesk.manualOff)).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: strings.lightDesk.presetsUi.saveUse })).toHaveLength(1);
    expect(screen.getAllByRole("button", { name: strings.lightDesk.presetsUi.saveDraft })).toHaveLength(1);
  });
  it("keeps physical band choices independent of reduced-motion preview and saves them only explicitly", async () => {
    vi.spyOn(window, "matchMedia").mockImplementation((query) => ({ matches: query === "(prefers-reduced-motion: reduce)", media: query, onchange: null, addEventListener: () => undefined, removeEventListener: () => undefined, addListener: () => undefined, removeListener: () => undefined, dispatchEvent: () => false } as MediaQueryList));
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ config, backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined });
    const save = vi.spyOn(lightingApi, "savePolicy").mockResolvedValue({ config } as never);
    const choose = vi.spyOn(lightingApi, "choosePreset").mockResolvedValue({} as never);
    const view = renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });
    const active = await screen.findByLabelText(strings.lightDesk.presetsUi.activeEffect);
    expect(active).toHaveValue("Breathing");
    expect(view.container.querySelector(".lightdesk-preview-dot--pulse")).not.toBeInTheDocument();
    fireEvent.click(screen.getByLabelText(strings.lightDesk.presetsUi.respectPreviewMotion));
    expect(active).toHaveValue("Breathing");
    expect(view.container.querySelector(".lightdesk-preview-dot--pulse")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText(strings.lightDesk.presetsUi.busyEffect), { target: { value: "" } });
    fireEvent.change(screen.getByLabelText(strings.lightDesk.presetsUi.staleAfter), { target: { value: "120" } });
    fireEvent.change(screen.getByLabelText(strings.lightDesk.presetsUi.fallbackColor), { target: { value: "#102030" } });
    fireEvent.change(screen.getByLabelText(strings.lightDesk.presetsUi.fallbackBrightness), { target: { value: "0" } });
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.presetsUi.saveDraft }));
    expect(save).not.toHaveBeenCalled();
    expect(choose).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: /Health/ }));
    expect(screen.getByLabelText(strings.lightDesk.presetsUi.staleAfter)).toHaveValue(60);
    fireEvent.click(screen.getByRole("button", { name: /Combined/ }));
    expect(screen.getByLabelText(strings.lightDesk.presetsUi.staleAfter)).toHaveValue(120);
    expect(screen.getByLabelText(strings.lightDesk.presetsUi.busyEffect)).toHaveValue("");
    expect(screen.getByLabelText(strings.lightDesk.presetsUi.respectPreviewMotion)).not.toBeChecked();
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.presetsUi.saveUse }));
    await waitFor(() => expect(save).toHaveBeenCalledOnce());
    const sent = save.mock.calls[0]![0];
    expect(sent.activityEffects).toMatchObject({ activeEffectId: "Breathing", busyEffectId: "" });
    expect(sent).toMatchObject({ sourceMaxAgeSeconds: 120, staleFallbackBrightnessPercent: 0, staleFallbackColor: { red: 16, green: 32, blue: 48 } });
    expect(config.sourceMaxAgeSeconds).toBe(60);
    expect(config.activityEffects?.busyEffectId).toBe("Breathing");
  });

  it("validates load thresholds and cancel restores saved values without policy writes", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ config, backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined });
    const save = vi.spyOn(lightingApi, "savePolicy").mockResolvedValue({ config } as never);
    vi.spyOn(lightingApi, "choosePreset").mockResolvedValue({} as never);
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });
    fireEvent.click(await screen.findByRole("button", { name: /Load/ }));
    fireEvent.change(screen.getByLabelText(strings.lightDesk.presetsUi.loadEnter), { target: { value: "40" } });
    expect(screen.getByLabelText(strings.lightDesk.presetsUi.loadExit)).toHaveValue(40);
    fireEvent.change(screen.getByLabelText(strings.lightDesk.presetsUi.loadExit), { target: { value: "90" } });
    expect(screen.getByLabelText(strings.lightDesk.presetsUi.loadExit)).toHaveValue(40);
    fireEvent.change(screen.getByLabelText(strings.lightDesk.presetsUi.debounceControl), { target: { value: "9000" } });
    expect(screen.getByLabelText(strings.lightDesk.presetsUi.debounceControl)).toHaveValue(3600);
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.presetsUi.cancelEdits }));
    expect(screen.getByLabelText(strings.lightDesk.presetsUi.loadEnter)).toHaveValue(80);
    expect(screen.getByLabelText(strings.lightDesk.presetsUi.loadExit)).toHaveValue(60);
    expect(save).not.toHaveBeenCalled();
    fireEvent.change(screen.getByLabelText(strings.lightDesk.presetsUi.loadEnter), { target: { value: "75" } });
    fireEvent.change(screen.getByLabelText(strings.lightDesk.presetsUi.loadExit), { target: { value: "55" } });
    fireEvent.change(screen.getByLabelText(strings.lightDesk.presetsUi.debounceControl), { target: { value: "12" } });
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.presetsUi.saveUse }));
    await waitFor(() => expect(save).toHaveBeenCalledOnce());
    expect(save.mock.calls[0]![0]).toMatchObject({ loadEnterPercent: 75, loadExitPercent: 55, debounceSeconds: 12 });
    expect(config.loadEnterPercent).toBe(80);
  });

  it("preserves unavailable saved effects while exposing advertised zone choices with truthful fallback", async () => {
    const saved = create(PolicyConfigSchema, { ...config, activityEffects: create(ActivityEffectsSchema, { activeEffectId: "Saved custom", busyEffectId: "Breathing", busyAgentThreshold: 5 }) });
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ config: saved, backend: undefined, controllers: [create(ControllerSchema, { zones: [{ supportedEffects: ["breathing", "  Spectrum  "] }] })], zones: [], manualOverride: undefined, effective: undefined });
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><PresetsPage /></MemoryRouter>, { withoutRouter: true });
    const active = await screen.findByLabelText(strings.lightDesk.presetsUi.activeEffect);
    expect(active).toHaveValue("Saved custom");
    expect(within(active).getByRole("option", { name: "Spectrum" })).toBeInTheDocument();
    expect(within(active).getAllByRole("option")).toHaveLength(4);
    expect(screen.getByText(strings.lightDesk.presetsUi.effectFallback)).toBeInTheDocument();
    expect(screen.getByLabelText(strings.lightDesk.presetsUi.activityThreshold)).toHaveAttribute("min", "2");
  });

});
