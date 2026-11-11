import { create } from "@bufbuild/protobuf";
import { EmptySchema } from "@bufbuild/protobuf/wkt";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  BackendState,
  BackendStatusSchema,
  ControllerSchema,
  ManualOverrideSchema,
  OperationState,
  SetManualOffResponseSchema,
  ZoneApplyResultSchema,
  ZoneCapabilitySchema,
  ZoneStateSchema,
  ZoneStatusSchema,
} from "@vrooli/proto-types/lighting-control/v1/control/control_pb";
import { EffectivePolicySchema, PolicyConfigSchema, PresetKind } from "@vrooli/proto-types/lighting-control/v1/policy/policy_pb";
import { EvidenceSourceSchema, EvidenceState } from "@vrooli/proto-types/lighting-control/v1/policy/policy_pb";

import { lightingApi } from "../features/lights/api";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import * as healthApi from "../api/health";
import { makeHealthResponse } from "../test-utils/factories";
import { DashboardPage } from "./DashboardPage";

const zoneState = create(ZoneStateSchema, {
  controllerId: "42",
  zoneId: "zone-main",
  desiredMode: { case: "off", value: create(EmptySchema, {}) },
});

const desk = {
  backend: create(BackendStatusSchema, { state: BackendState.READY, sdkProtocolVersion: 6 }),
  controllers: [create(ControllerSchema, {
    controllerId: "42", name: "Desk strip", vendor: "Example", enrolled: true, identitySource: "openrgb_sdk_serial",
    zones: [create(ZoneCapabilitySchema, { zoneId: "zone-main", name: "Main", ledCount: 12, supportsStaticColor: true, supportsBrightness: true })],
  })],
  zones: [create(ZoneStatusSchema, {
    desired: zoneState,
    lastApply: create(ZoneApplyResultSchema, { desired: zoneState, operation: OperationState.BACKEND_ACKNOWLEDGED, backendAcknowledged: true }),
  })],
  manualOverride: create(ManualOverrideSchema, {}),
  effective: create(EffectivePolicySchema, { preset: PresetKind.WORK, brightnessPercent: 35, reason: "Load is low", evidence: [] }),
  config: create(PolicyConfigSchema, { brightnessCapPercent: 100 }),
};

describe("Light Desk", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("distinguishes backend acceptance from observed output and offers persistent manual Off", async () => {
    const read = vi.spyOn(lightingApi, "readDesk").mockResolvedValue(desk);
    const off = vi.spyOn(lightingApi, "setOff").mockResolvedValue(create(SetManualOffResponseSchema, {}));
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><DashboardPage /></MemoryRouter>, { withoutRouter: true });

    await waitFor(() => expect(screen.getByTestId(selectors.pages.dashboard).querySelector(".lightdesk-status")).not.toBeNull());
    const backend = screen.getByRole("heading", { name: strings.lightDesk.backend }).closest("section")!;
    await waitFor(() => expect(backend.querySelector(".lightdesk-status")).toHaveTextContent(`${strings.lightDesk.backendReady} · READY`));
    const evidence = screen.getByRole("heading", { name: strings.lightDesk.commandEvidence }).closest("section")!;
    expect(within(evidence).getByText(strings.lightDesk.acknowledgedNotObserved)).toBeInTheDocument();
    expect(within(evidence).getByText("0")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId(selectors.lightDesk.manualOff));
    await waitFor(() => expect(off).toHaveBeenCalledOnce());
    await waitFor(() => expect(read).toHaveBeenCalled());
  });

  it("shows empty and degraded states and exposes recovery actions", async () => {
    const degraded = {
      backend: undefined,
      controllers: [],
      zones: [],
      manualOverride: create(ManualOverrideSchema, { active: true, allEnrolled: true }),
      effective: create(EffectivePolicySchema, {
        preset: PresetKind.COMBINED,
        manualPriority: true,
        staticFallback: true,
        reason: "",
        evidence: [create(EvidenceSourceSchema, { source: "system-monitor", state: EvidenceState.DEGRADED, reason: "source timeout" })],
      }),
      config: undefined,
    };
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue(degraded);
    const clearOff = vi.spyOn(lightingApi, "clearOff").mockResolvedValue({} as never);
    const clearPreset = vi.spyOn(lightingApi, "clearPreset").mockResolvedValue({} as never);
    const off = vi.spyOn(lightingApi, "setOff").mockResolvedValue(create(SetManualOffResponseSchema, {}));
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><DashboardPage /></MemoryRouter>, { withoutRouter: true });

    expect(await screen.findByText(strings.lightDesk.noControllers)).toBeInTheDocument();
    expect(screen.getByText(strings.lightDesk.staleWarning)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.lightDesk.manualOff)).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.resumeLighting }));
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.resumeAutomatic }));
    await waitFor(() => expect(clearOff).toHaveBeenCalledOnce());
    await waitFor(() => expect(clearPreset).toHaveBeenCalledOnce());
    expect(off).not.toHaveBeenCalled();
  });

  it("explains when no policy has been evaluated yet", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined, config: undefined });
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><DashboardPage /></MemoryRouter>, { withoutRouter: true });
    expect(await screen.findByText(strings.lightDesk.policyUnavailable)).toBeInTheDocument();
  });

  it("checks backend state, applies an editable color, and chooses a policy preset", async () => {
    const read = vi.spyOn(lightingApi, "readDesk").mockResolvedValue(desk);
    const setStatic = vi.spyOn(lightingApi, "setStatic").mockResolvedValue({} as never);
    const choose = vi.spyOn(lightingApi, "choosePreset").mockResolvedValue({} as never);
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><DashboardPage /></MemoryRouter>, { withoutRouter: true });

    await screen.findByRole("heading", { name: strings.lightDesk.controllers });
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.checkBackend }));
    const colorLabel = `${strings.lightDesk.color} Main`;
    const brightnessLabel = `${strings.lightDesk.brightness} Main`;
    fireEvent.change(screen.getByLabelText(colorLabel), { target: { value: "#123456" } });
    fireEvent.change(screen.getByLabelText(brightnessLabel), { target: { value: "55" } });
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.applyStatic }));
    await waitFor(() => expect(setStatic).toHaveBeenCalledWith("42", "zone-main", { red: 18, green: 52, blue: 86 }, 55));
    fireEvent.change(screen.getByLabelText(strings.lightDesk.choosePreset), { target: { value: String(PresetKind.HEALTH) } });
    await waitFor(() => expect(choose).toHaveBeenCalledWith(PresetKind.HEALTH));
    expect(read.mock.calls.length).toBeGreaterThanOrEqual(2);
  });

  it("uses the phone composition and reports a non-Error preset failure", async () => {
    vi.spyOn(window, "matchMedia").mockImplementation((query) => ({
      matches: query === "(max-width: 720px)", media: query, onchange: null,
      addEventListener: () => undefined, removeEventListener: () => undefined,
      addListener: () => undefined, removeListener: () => undefined, dispatchEvent: () => false,
    } as MediaQueryList));
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue(desk);
    const choose = vi.spyOn(lightingApi, "choosePreset").mockRejectedValue("temporary policy error");
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><DashboardPage /></MemoryRouter>, { withoutRouter: true });

    await screen.findByRole("heading", { name: strings.lightDesk.controllers });
    expect(screen.getByTestId(selectors.pages.dashboard).querySelector(".lightdesk-mobile")).not.toBeNull();
    fireEvent.change(screen.getByLabelText(strings.lightDesk.choosePreset), { target: { value: String(PresetKind.WORK) } });
    expect(await screen.findByText(strings.lightDesk.actionFailed)).toBeInTheDocument();
    expect(choose).toHaveBeenCalledWith(PresetKind.WORK);
  });

  it("shows load errors and retries the desk query", async () => {
    const read = vi.spyOn(lightingApi, "readDesk").mockRejectedValueOnce(new Error("Dashboard unavailable")).mockResolvedValue(desk);
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><DashboardPage /></MemoryRouter>, { withoutRouter: true });
    expect(await screen.findByRole("alert")).toHaveTextContent("Dashboard unavailable");
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.retry }));
    await screen.findByRole("heading", { name: strings.lightDesk.controllers });
    expect(read).toHaveBeenCalledTimes(2);
  });

  it("explains all command outcomes and evidence freshness states", async () => {
    const operations = [OperationState.COMMAND_SENT, OperationState.DEFERRED, OperationState.UNSUPPORTED, OperationState.FAILED, OperationState.UNSPECIFIED];
    const controller = create(ControllerSchema, {
      controllerId: "durable-42", name: "Desk strip", vendor: "Example", enrolled: false, identitySource: "openrgb_sdk_serial",
      zones: operations.map((_, index) => create(ZoneCapabilitySchema, { zoneId: `zone-${index}`, name: `Zone ${index}`, ledCount: 1 })),
    });
    const zones = operations.map((operation, index) => {
      const desired = create(ZoneStateSchema, { controllerId: controller.controllerId, zoneId: `zone-${index}`, desiredMode: { case: "off", value: {} } });
      return create(ZoneStatusSchema, { desired, lastApply: create(ZoneApplyResultSchema, { desired, operation, reason: index === 3 ? "controller rejected request" : "" }) });
    });
    const timestamp = { seconds: BigInt(Math.floor(Date.now() / 1000) - 2), nanos: 0 };
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({
      ...desk, controllers: [controller], zones,
      effective: create(EffectivePolicySchema, {
        preset: PresetKind.COMBINED, reason: "Partial sources", evidence: [
          create(EvidenceSourceSchema, { source: "fresh", state: EvidenceState.FRESH, observedAt: timestamp }),
          create(EvidenceSourceSchema, { source: "stale", state: EvidenceState.STALE, observedAt: timestamp }),
          create(EvidenceSourceSchema, { source: "unavailable", state: EvidenceState.UNAVAILABLE }),
          create(EvidenceSourceSchema, { source: "degraded", state: EvidenceState.DEGRADED, reason: "partial" }),
          create(EvidenceSourceSchema, { source: "unknown", state: EvidenceState.UNSPECIFIED }),
        ],
      }),
    });
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><DashboardPage /></MemoryRouter>, { withoutRouter: true });

    await screen.findByText(strings.lightDesk.staleWarning);
    const outcomeText = [...screen.getByTestId(selectors.pages.dashboard).querySelectorAll(".lightdesk-operation")].map((node) => node.textContent).join(" ");
    expect(outcomeText).toContain(strings.lightDesk.commandSent);
    expect(outcomeText).toContain(strings.lightDesk.deferred);
    expect(outcomeText).toContain(strings.lightDesk.unsupported);
    expect(outcomeText).toContain(strings.lightDesk.failed);
    expect(outcomeText).toContain(strings.lightDesk.noCommand);
    const evidenceText = [...screen.getByTestId(selectors.pages.dashboard).querySelectorAll(".lightdesk-evidence-row")].map((node) => node.textContent).join(" ");
    expect(evidenceText).toContain("FRESH");
    expect(evidenceText).toContain("STALE");
    expect(evidenceText).toContain("UNAVAILABLE");
    expect(evidenceText).toContain("DEGRADED");
    expect(evidenceText).toContain("UNKNOWN");
  });

  it("shows operation failures after a control request", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ ...desk, manualOverride: create(ManualOverrideSchema, {}), config: undefined });
    const saveError = "Local save failed";
    vi.spyOn(lightingApi, "setOff").mockRejectedValue(new Error(saveError));
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><DashboardPage /></MemoryRouter>, { withoutRouter: true });
    fireEvent.click(await screen.findByTestId(selectors.lightDesk.manualOff));
    expect(await screen.findByText(saveError)).toBeInTheDocument();
    expect(screen.getByText(saveError)).toHaveAttribute("role", "alert");
  });

  it.each([BackendState.DISABLED, BackendState.STOPPED, BackendState.UNAVAILABLE, BackendState.PROTOCOL_MISMATCH])("keeps stored controllers visible without writing to a non-ready backend (%s)", async (state) => {
    const backendDetail = "Lighting backend requires operator setup";
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ ...desk, backend: create(BackendStatusSchema, { state, detail: backendDetail }) });
    const apply = vi.spyOn(lightingApi, "setStatic");
    const off = vi.spyOn(lightingApi, "setOff");
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><DashboardPage /></MemoryRouter>, { withoutRouter: true });
    expect(await screen.findByRole("link", { name: "Desk strip" })).toHaveAttribute("href", "/controllers/42");
    const button = screen.getByRole("button", { name: strings.lightDesk.applyStatic });
    expect(button).toBeDisabled();
    fireEvent.click(button);
    expect(apply).not.toHaveBeenCalled();
    expect(off).not.toHaveBeenCalled();
    expect(screen.getByText(backendDetail)).toBeInTheDocument();
    expect(screen.getByText(strings.lightDesk.desiredStateRetained)).toBeInTheDocument();
  });

  it("blocks a static command for an unstable saved identity", async () => {
    const controller = create(ControllerSchema, { ...desk.controllers[0]!, identitySource: "openrgb_sdk_transient_handle" });
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ ...desk, controllers: [controller] });
    const apply = vi.spyOn(lightingApi, "setStatic");
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><DashboardPage /></MemoryRouter>, { withoutRouter: true });
    expect(await screen.findByText(strings.lightDesk.controllerUi.identityUnavailable)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: strings.lightDesk.applyStatic })).toBeDisabled();
    expect(apply).not.toHaveBeenCalled();
  });

  it("preserves command and physical evidence in the sequential phone layout", async () => {
    vi.spyOn(window, "matchMedia").mockImplementation((query) => ({ matches: query === "(max-width: 720px)", media: query, onchange: null, addEventListener: () => undefined, removeEventListener: () => undefined, addListener: () => undefined, removeListener: () => undefined, dispatchEvent: () => false } as MediaQueryList));
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue(desk);
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><DashboardPage /></MemoryRouter>, { withoutRouter: true });
    const policy = await screen.findByRole("heading", { name: strings.lightDesk.policy });
    const controllers = screen.getByRole("heading", { name: strings.lightDesk.controllers });
    const evidence = screen.getByRole("heading", { name: strings.lightDesk.commandEvidence }).closest("section")!;
    expect(policy.compareDocumentPosition(controllers) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(within(evidence).getByText(strings.lightDesk.backendAccepted)).toBeInTheDocument();
    expect(within(evidence).getByText(strings.lightDesk.observedOutput)).toBeInTheDocument();
    expect(within(evidence).getByText(strings.lightDesk.acknowledgedNotObserved)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.lightDesk.manualOff)).toBeEnabled();
  });

  it("renders the loaded desktop workbench within one accessible main landmark", async () => {
    await setLocale("en");
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue(desk);
    vi.spyOn(healthApi, "fetchHealth").mockResolvedValue(makeHealthResponse());
    const { container } = renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><main><DashboardPage /></main></MemoryRouter>, { withoutRouter: true });
    await screen.findByRole("link", { name: "Desk strip" });
    await screen.findByTestId(selectors.health.statusValue);
    expect(screen.getAllByRole("main")).toHaveLength(1);
    await expectNoA11yViolations(container);
  });
  it("constrains Overview manual apply to the saved maximum brightness", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ ...desk, config: create(PolicyConfigSchema, { brightnessCapPercent: 10 }) });
    const apply = vi.spyOn(lightingApi, "setStatic").mockResolvedValue({} as never);
    renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><DashboardPage/></MemoryRouter>, { withoutRouter: true });
    expect(await screen.findByRole("slider")).toHaveAttribute("max", "10");
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.applyStatic }));
    await waitFor(() => expect(apply).toHaveBeenCalledWith("42", "zone-main", expect.anything(), 10));
  });

});
