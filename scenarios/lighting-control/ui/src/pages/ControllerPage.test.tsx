import { create } from "@bufbuild/protobuf";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BackendState, BackendStatusSchema, ControllerSchema, OperationState, ReportedLEDColorSchema, ZoneApplyResultSchema, ZoneCapabilitySchema, ZoneStateSchema, ZoneStatusSchema } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";
import { PolicyConfigSchema, PresetKind } from "@vrooli/proto-types/lighting-control/v1/policy/policy_pb";
import { lightingApi } from "../features/lights/api";
import { strings } from "../consts/strings";
import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { i18n, setLocale } from "../i18n";
import { ControllerPage } from "./ControllerPage";

describe("ControllerPage", () => {
  afterEach(() => vi.restoreAllMocks());

  it("applies a static color only to an enrolled zone on a ready backend", async () => {
    const controller = create(ControllerSchema, {
      controllerId: "42", name: "Desk strip", vendor: "Example", enrolled: true, identitySource: "openrgb_sdk_serial",
      zones: [create(ZoneCapabilitySchema, { zoneId: "zone-main", name: "Main", ledCount: 12, supportsStaticColor: true, supportsBrightness: true })],
    });
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({
      backend: create(BackendStatusSchema, { state: BackendState.READY, sdkProtocolVersion: 6 }),
      controllers: [controller], zones: [], manualOverride: undefined, effective: undefined,
      config: create(PolicyConfigSchema, { defaultPreset: PresetKind.OFF, brightnessCapPercent: 35 }),
    });
    const setStatic = vi.spyOn(lightingApi, "setStatic").mockResolvedValue({} as never);
    renderWithProviders(<MemoryRouter initialEntries={["/controllers/42"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><Routes><Route path="/controllers/:controllerId" element={<ControllerPage />}/></Routes></MemoryRouter>, { withoutRouter: true });

    fireEvent.click(await screen.findByRole("button", { name: /lightDesk\.controllerUi\.saveApply/ }));
    await waitFor(() => expect(setStatic).toHaveBeenCalledWith("42", "zone-main", { red: 87, green: 190, blue: 138 }, 20));
  });

  it("explains when a controller is absent from the current inventory", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined, config: undefined });
    renderWithProviders(<MemoryRouter initialEntries={["/controllers/missing"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><Routes><Route path="/controllers/:controllerId" element={<ControllerPage />}/></Routes></MemoryRouter>, { withoutRouter: true });
    expect(await screen.findByRole("heading", { name: strings.lightDesk.controllerUi.unavailableTitle })).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent(strings.lightDesk.controllerUi.unavailableHelp);
  });

  it("shows cached controller colors with their time and without claiming per-zone or physical verification", async () => {
    await setLocale("en");
    const observedAt = { seconds: BigInt(1_800_000_000), nanos: 0 };
    const attemptedAt = { seconds: BigInt(1_800_000_010), nanos: 0 };
    const payload = create(ZoneStateSchema, {
      controllerId: "42", zoneId: "zone-main",
      desiredMode: { case: "staticColor", value: { color: { red: 217, green: 154, blue: 36 }, brightnessPercent: 25 } },
    });
    const controller = create(ControllerSchema, {
      controllerId: "42", name: "GPU", vendor: "Example", enrolled: true, identitySource: "openrgb_linux_pci_i2c_location",
      zones: [create(ZoneCapabilitySchema, { zoneId: "zone-main", name: "Main", supportsStaticColor: true })],
      reportedColorSource: "openrgb_sdk_controller_data_cached_state",
      reportedColorsObservedAt: observedAt,
      reportedActiveMode: "Direct",
      reportedLedColors: [0, 1, 2, 3].map((index) => create(ReportedLEDColorSchema, { index, displayName: `LED ${index + 1}`, color: { red: 54, green: 39, blue: 9 } })),
    });
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({
      backend: create(BackendStatusSchema, { state: BackendState.READY }), controllers: [controller],
      zones: [create(ZoneStatusSchema, { desired: payload, lastApply: create(ZoneApplyResultSchema, { desired: payload, operation: OperationState.BACKEND_ACKNOWLEDGED, completedAt: attemptedAt, backendAcknowledged: true }) })],
      manualOverride: undefined, effective: undefined, config: create(PolicyConfigSchema, { brightnessCapPercent: 35 }),
    });
    const { container } = renderWithProviders(<MemoryRouter initialEntries={["/controllers/42"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><Routes><Route path="/controllers/:controllerId" element={<ControllerPage />}/></Routes></MemoryRouter>, { withoutRouter: true });

    expect(await screen.findByRole("heading", { name: i18n.t(strings.lightDesk.openRgbReportedState) })).toBeInTheDocument();
    const reportText = container.querySelector(".lightdesk-reported-colors")?.textContent ?? "";
    expect(reportText).toContain("#362709");
    expect(reportText).toContain(i18n.t(strings.lightDesk.reportedColorPredatesCommand));
    expect(reportText).toContain(i18n.t(strings.lightDesk.reportedColorScope));
    expect(reportText).toContain(i18n.t(strings.lightDesk.reportedColorSource, { source: i18n.t(strings.lightDesk.reportedColorSourceOpenRgbCache) }));
    expect(reportText).not.toContain("openrgb_sdk_controller_data_cached_state");
  });

  it("shows errors when durable Off cannot be saved", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({
      backend: create(BackendStatusSchema, { state: BackendState.READY }),
      controllers: [create(ControllerSchema, { controllerId: "42", name: "Desk strip", enrolled: true, identitySource: "openrgb_sdk_serial", zones: [] })],
      zones: [], manualOverride: undefined, effective: undefined,
      config: create(PolicyConfigSchema, { defaultPreset: PresetKind.OFF, brightnessCapPercent: 35 }),
    });
    const setOff = vi.spyOn(lightingApi, "setOff").mockRejectedValue(new Error("Local persistence failed"));
    renderWithProviders(<MemoryRouter initialEntries={["/controllers/42"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><Routes><Route path="/controllers/:controllerId" element={<ControllerPage />}/></Routes></MemoryRouter>, { withoutRouter: true });

    fireEvent.click(await screen.findByRole("button", { name: strings.lightDesk.controllerUi.keepOff }));
    await waitFor(() => expect(setOff).toHaveBeenCalledOnce());
    expect(await screen.findByRole("alert")).toHaveTextContent("Local persistence failed");
  });

  it("keeps writes disabled until the controller is enrolled and the backend is ready", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({
      backend: create(BackendStatusSchema, { state: BackendState.UNAVAILABLE }),
      controllers: [create(ControllerSchema, {
        controllerId: "42", name: "Desk strip", enrolled: false, identitySource: "openrgb_sdk_serial",
        zones: [create(ZoneCapabilitySchema, { zoneId: "zone-main", name: "Main", supportsStaticColor: true })],
      })], zones: [], manualOverride: undefined, effective: undefined,
      config: create(PolicyConfigSchema, { defaultPreset: PresetKind.OFF, brightnessCapPercent: 35 }),
    });
    const setStatic = vi.spyOn(lightingApi, "setStatic");
    const enroll = vi.spyOn(lightingApi, "enroll").mockResolvedValue({} as never);
    renderWithProviders(<MemoryRouter initialEntries={["/controllers/42"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><Routes><Route path="/controllers/:controllerId" element={<ControllerPage />}/></Routes></MemoryRouter>, { withoutRouter: true });

    expect(await screen.findByRole("heading", { name: strings.lightDesk.controllerUi.zoneControls })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /lightDesk\.controllerUi\.saveApply/ })).toBeDisabled();
    expect(screen.getByRole("button", { name: strings.lightDesk.controllerUi.enrollController })).toBeDisabled();
    expect(enroll).not.toHaveBeenCalled();
    expect(setStatic).not.toHaveBeenCalled();
  });

  it("shows successful static apply and keeps backend rejection visible", async () => {
    const controller = create(ControllerSchema, {
      controllerId: "42", name: "Desk strip", vendor: "Example", enrolled: true, identitySource: "openrgb_sdk_serial",
      zones: [create(ZoneCapabilitySchema, { zoneId: "zone-main", name: "Main", supportsStaticColor: true })],
    });
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({
      backend: create(BackendStatusSchema, { state: BackendState.READY }), controllers: [controller], zones: [],
      manualOverride: undefined, effective: undefined, config: create(PolicyConfigSchema, { defaultPreset: PresetKind.OFF, brightnessCapPercent: 35 }),
    });
    const setStatic = vi.spyOn(lightingApi, "setStatic").mockRejectedValueOnce(new Error("Controller disconnected"));
    const setOff = vi.spyOn(lightingApi, "setOff").mockResolvedValue({} as never);
    renderWithProviders(<MemoryRouter initialEntries={["/controllers/42"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><Routes><Route path="/controllers/:controllerId" element={<ControllerPage />}/></Routes></MemoryRouter>, { withoutRouter: true });

    fireEvent.click(await screen.findByRole("button", { name: /lightDesk\.controllerUi\.saveApply/ }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Controller disconnected");
    setStatic.mockResolvedValueOnce({} as never);
    fireEvent.click(screen.getByRole("button", { name: /lightDesk\.controllerUi\.saveApply/ }));
    expect(await screen.findByRole("status")).toHaveTextContent(strings.lightDesk.controllerUi.colorSaved);
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.controllerUi.keepOff }));
    await waitFor(() => expect(setOff).toHaveBeenCalledOnce());
    expect(await screen.findByRole("status")).toHaveTextContent(strings.lightDesk.controllerUi.offSaved);
  });

  it("renders each per-zone command outcome and never exposes writes for an unstable identity", async () => {
    const operations = [
      OperationState.BACKEND_ACKNOWLEDGED,
      OperationState.COMMAND_SENT,
      OperationState.FAILED,
      OperationState.UNSUPPORTED,
      OperationState.DEFERRED,
      OperationState.UNSPECIFIED,
    ];
    const zones = operations.map((_, index) => create(ZoneCapabilitySchema, {
      zoneId: `zone-${index}`, name: `Zone ${index}`, ledCount: 2, supportsStaticColor: true,
      supportedEffects: index === 0 ? ["Breathing"] : [],
    }));
    const controller = create(ControllerSchema, {
      controllerId: "unstable-42", name: "", vendor: "", enrolled: true,
      identitySource: "openrgb_sdk_transient_handle", zones,
    });
    const zoneStatuses = operations.map((operation, index) => {
      const desired = create(ZoneStateSchema, { controllerId: controller.controllerId, zoneId: `zone-${index}`, desiredMode: { case: "off", value: {} } });
      return create(ZoneStatusSchema, {
        desired,
        lastApply: create(ZoneApplyResultSchema, { desired, operation, reason: index === 2 ? "SDK rejected update; cached mode readback mismatched" : index === 0 ? "cached mode readback matches; physical output is unverified" : "", backendAcknowledged: operation === OperationState.BACKEND_ACKNOWLEDGED }),
      });
    });
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({
      backend: create(BackendStatusSchema, { state: BackendState.READY }), controllers: [controller], zones: zoneStatuses,
      manualOverride: undefined, effective: undefined, config: create(PolicyConfigSchema, { defaultPreset: PresetKind.OFF, brightnessCapPercent: 35 }),
    });
    const setStatic = vi.spyOn(lightingApi, "setStatic");
    const setOff = vi.spyOn(lightingApi, "setOff");
    const enroll = vi.spyOn(lightingApi, "enroll");
    renderWithProviders(<MemoryRouter initialEntries={["/controllers/unstable-42"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><Routes><Route path="/controllers/:controllerId" element={<ControllerPage />}/></Routes></MemoryRouter>, { withoutRouter: true });

    await screen.findByRole("heading", { name: strings.lightDesk.controllerUi.unknownVendor });
    expect(screen.getByText(strings.lightDesk.controllerUi.identityUnavailable)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: strings.lightDesk.controllerUi.keepOff })).toBeDisabled();
    expect(screen.getAllByRole("button", { name: /lightDesk\.controllerUi\.saveApply/ }).every((button) => (button as HTMLButtonElement).disabled)).toBe(true);
    const outcomes = [...screen.getByTestId("page-controller").querySelectorAll(".lightdesk-operation")].map((node) => node.textContent);
    expect(outcomes.some((text) => text.includes(strings.lightDesk.controllerUi.backendAccepted))).toBe(true);
    expect(outcomes.some((text) => text.includes("cached mode readback matches"))).toBe(true);
    expect(outcomes.some((text) => text.includes("cached mode readback mismatched"))).toBe(true);
    expect(outcomes.some((text) => text.includes(strings.lightDesk.controllerUi.commandSent))).toBe(true);
    expect(outcomes.some((text) => text.includes("SDK rejected update"))).toBe(true);
    expect(outcomes.some((text) => text.includes(strings.lightDesk.controllerUi.unsupported))).toBe(true);
    expect(outcomes.some((text) => text.includes(strings.lightDesk.controllerUi.deferred))).toBe(true);
    expect(outcomes.some((text) => text.includes(strings.lightDesk.controllerUi.noCommand))).toBe(true);
    expect(setStatic).not.toHaveBeenCalled();
    expect(setOff).not.toHaveBeenCalled();
    expect(enroll).not.toHaveBeenCalled();
  });

  it("retains a failed enrollment in the controller page and permits retry", async () => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({
      backend: create(BackendStatusSchema, { state: BackendState.READY }),
      controllers: [create(ControllerSchema, { controllerId: "42", name: "Desk strip", identitySource: "openrgb_sdk_serial" })],
      zones: [], manualOverride: undefined, effective: undefined, config: undefined,
    });
    const enroll = vi.spyOn(lightingApi, "enroll").mockRejectedValueOnce(new Error("Enrollment storage failed")).mockResolvedValueOnce({} as never);
    renderWithProviders(<MemoryRouter initialEntries={["/controllers/42"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><Routes><Route path="/controllers/:controllerId" element={<ControllerPage />}/></Routes></MemoryRouter>, { withoutRouter: true });
    fireEvent.click(await screen.findByRole("button", { name: strings.lightDesk.controllerUi.enrollController }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Enrollment storage failed");
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.controllerUi.enrollController }));
    expect(await screen.findByRole("status")).toHaveTextContent(strings.lightDesk.controllerEnrolled);
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(enroll).toHaveBeenLastCalledWith("42", true);
  });

  it("shows a query error separately from an absent controller and retries in place", async () => {
    const read = vi.spyOn(lightingApi, "readDesk").mockRejectedValueOnce(new Error("Controller inventory timed out")).mockResolvedValueOnce({
      backend: create(BackendStatusSchema, { state: BackendState.READY }),
      controllers: [create(ControllerSchema, { controllerId: "42", name: "Recovered controller", identitySource: "openrgb_sdk_serial" })],
      zones: [], manualOverride: undefined, effective: undefined, config: undefined,
    });
    renderWithProviders(<MemoryRouter initialEntries={["/controllers/42"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><Routes><Route path="/controllers/:controllerId" element={<ControllerPage />}/></Routes></MemoryRouter>, { withoutRouter: true });
    expect(await screen.findByRole("alert")).toHaveTextContent("Controller inventory timed out");
    fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.retry }));
    expect(await screen.findByRole("heading", { name: "Recovered controller" })).toBeInTheDocument();
    expect(read).toHaveBeenCalledTimes(2);
  });

  it("caps the preview brightness and addresses the chosen zone accessibly", async () => {
    const controller = create(ControllerSchema, {
      controllerId: "42", name: "Desk strip", enrolled: true, identitySource: "openrgb_sdk_serial",
      zones: ["Main", "Logo"].map((name) => create(ZoneCapabilitySchema, { zoneId: name.toLowerCase(), name, supportsStaticColor: true })),
    });
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ backend: create(BackendStatusSchema, { state: BackendState.READY }), controllers: [controller], zones: [], manualOverride: undefined, effective: undefined, config: create(PolicyConfigSchema, { brightnessCapPercent: 10 }) });
    const apply = vi.spyOn(lightingApi, "setStatic").mockResolvedValue({} as never);
    renderWithProviders(<MemoryRouter initialEntries={["/controllers/42"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><Routes><Route path="/controllers/:controllerId" element={<ControllerPage />}/></Routes></MemoryRouter>, { withoutRouter: true });
    const slider = await screen.findByRole("slider", { name: `${strings.lightDesk.brightness} Logo` });
    expect(slider).toHaveAttribute("max", "10");
    expect(slider).toHaveValue("10");
    const logoColorLabel = `${strings.lightDesk.color} Logo`;
    fireEvent.change(screen.getByLabelText(logoColorLabel), { target: { value: "#123456" } });
    expect(screen.getByText("#123456")).toBeInTheDocument();
    const logo = screen.getByRole("group", { name: "Logo" });
    fireEvent.click(within(logo).getByRole("button", { name: `${strings.lightDesk.controllerUi.saveApply} · Logo` }));
    await waitFor(() => expect(apply).toHaveBeenCalledWith("42", "logo", { red: 18, green: 52, blue: 86 }, 10));
  });

  it("exposes usable controls without accessibility violations", async () => {
    await setLocale("en");
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({
      backend: create(BackendStatusSchema, { state: BackendState.READY, detail: "Connected", checkedAt: "2026-11-10T21:00:00Z", sdkProtocolVersion: 6 }),
      controllers: [create(ControllerSchema, { controllerId: "42", name: "Desk strip", vendor: "Example", enrolled: true, identitySource: "openrgb_sdk_serial", zones: [create(ZoneCapabilitySchema, { zoneId: "main", name: "Main", supportsStaticColor: true })] })],
      zones: [], manualOverride: undefined, effective: undefined, config: create(PolicyConfigSchema, { brightnessCapPercent: 35 }),
    });
    const { container } = renderWithProviders(<MemoryRouter initialEntries={["/controllers/42"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><Routes><Route path="/controllers/:controllerId" element={<ControllerPage />}/></Routes></MemoryRouter>, { withoutRouter: true });
    await screen.findByRole("heading", { name: "Desk strip" });
    await expectNoA11yViolations(container);
  });
});

it("inspects saved offline wiring with hardware controls disabled while durable Off remains available", async () => {
  const controller = create(ControllerSchema, {
    controllerId: "saved", name: "Saved strip", enrolled: true, identitySource: "openrgb_sdk_serial",
    zones: [create(ZoneCapabilitySchema, { zoneId: "header", name: "Saved header", ledCount: 24, supportsStaticColor: true, supportsLedCountConfiguration: true, minLedCount: 0, maxLedCount: 500, configuredLedCount: 24 })],
  });
  vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ backend: create(BackendStatusSchema, { state: BackendState.UNAVAILABLE, detail: "SDK disconnected" }), controllers: [controller], zones: [], manualOverride: undefined, effective: undefined, config: create(PolicyConfigSchema, { brightnessCapPercent: 35 }) });
  const staticWrite = vi.spyOn(lightingApi, "setStatic");
  const resize = vi.spyOn(lightingApi, "configureZone");
  const off = vi.spyOn(lightingApi, "setOff").mockResolvedValue({} as never);
  renderWithProviders(<MemoryRouter initialEntries={["/controllers/saved"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><Routes><Route path="/controllers/:controllerId" element={<ControllerPage />}/></Routes></MemoryRouter>, { withoutRouter: true });
  expect(await screen.findByRole("heading", { name: "Saved strip" })).toBeInTheDocument();
  expect(screen.getByText(controller.zones[0]!.name)).toBeInTheDocument();
  expect(screen.getByText(strings.lightDesk.desiredStateRetained)).toBeInTheDocument();
  const apply = screen.getByRole("button", { name: /lightDesk\.controllerUi\.saveApply/ });
  const length = screen.getByRole("button", { name: strings.lightDesk.manualUi.saveLength });
  expect(apply).toBeDisabled(); expect(length).toBeDisabled();
  fireEvent.click(apply); fireEvent.click(length);
  expect(staticWrite).not.toHaveBeenCalled(); expect(resize).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.controllerUi.keepOff }));
  await waitFor(() => expect(off).toHaveBeenCalledOnce());
  expect(await screen.findByRole("status")).toHaveTextContent(strings.lightDesk.controllerUi.offSaved);
});
