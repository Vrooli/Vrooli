import { Code, ConnectError } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { InstallationSource, LightingRuntimeStatusSchema, LightingSelectionSchema } from "@vrooli/proto-types/lighting-control/v1/setup/setup_pb";
import { PolicyConfigSchema } from "@vrooli/proto-types/lighting-control/v1/policy/policy_pb";
import { lightingSetupApi } from "../features/lights/setup";
import { lightingApi } from "../features/lights/api";
import { strings } from "../consts/strings";
import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { i18n, setLocale } from "../i18n";
import { SettingsPage } from "./SettingsPage";

const copy = strings.lightDesk.settingsUi;
const selection = create(LightingSelectionSchema, { source: InstallationSource.MANAGED, version: "1.1", sha256: "a".repeat(64), devicePolicyId: "approved-policy" });
const disabled = () => create(LightingRuntimeStatusSchema, { phase: "disabled", state: "disabled", pinnedSelection: selection, resourceObserved: false });
const renderSettings = () => renderWithProviders(<MemoryRouter initialEntries={["/settings"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><SettingsPage /></MemoryRouter>, { withoutRouter: true });

describe("Settings native lighting owner actions", () => {
  beforeEach(() => {
    vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined, config: create(PolicyConfigSchema, {}) });
    vi.spyOn(lightingSetupApi, "status").mockResolvedValue(disabled());
  });
  afterEach(() => vi.restoreAllMocks());

  it("opens read-only and reports disabled resource observations as unknown", async () => {
    const setup = vi.spyOn(lightingSetupApi, "setup");
    const enable = vi.spyOn(lightingSetupApi, "enable");
    const disable = vi.spyOn(lightingSetupApi, "disable");
    renderSettings();
    expect(await screen.findAllByText(copy.unobserved)).toHaveLength(3);
    expect(screen.getByRole("button", { name: copy.enableLighting })).toBeDisabled();
    expect(screen.getByRole("button", { name: copy.disableLighting })).toBeDisabled();
    expect(setup).not.toHaveBeenCalled(); expect(enable).not.toHaveBeenCalled(); expect(disable).not.toHaveBeenCalled();
  });

  it("prepares exact source without enablement then shows native launch blocker truthfully", async () => {
    const ownerMessage = "Selected artifact verified";
    const prepared = create(LightingRuntimeStatusSchema, { ...disabled(), selection, setupVerified: true, installed: true, resourceObserved: false, message: ownerMessage });
    const setup = vi.spyOn(lightingSetupApi, "setup").mockResolvedValue(prepared);
    const enable = vi.spyOn(lightingSetupApi, "enable").mockResolvedValue(create(LightingRuntimeStatusSchema, { ...prepared, phase: "enabled", state: "degraded", degraded: true, launchBlocker: "Native confinement witness is pending", message: "Intent saved" }));
    const disable = vi.spyOn(lightingSetupApi, "disable").mockResolvedValue(disabled());
    renderSettings();
    const prepare = await screen.findByRole("button", { name: copy.prepareSetup });
    await waitFor(() => expect(prepare).toBeEnabled()); fireEvent.click(prepare);
    expect(await screen.findByText(ownerMessage)).toBeInTheDocument();
    expect(screen.getAllByText(copy.unobserved)).toHaveLength(2);
    expect(setup.mock.calls[0]?.[0]).toEqual(selection); expect(enable).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: copy.enableLighting }));
    expect(await screen.findByText(copy.launchBlocked)).toBeInTheDocument();
    expect(enable).toHaveBeenCalledWith(selection);
    expect(screen.queryByText(copy.physicalUnverified)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: copy.disableLighting }));
    await waitFor(() => expect(disable).toHaveBeenCalledOnce());
    expect(await screen.findAllByText(copy.unobserved)).toHaveLength(3);
  });

  it("requires a fresh setup after changing source and delegates compatible existing verification", async () => {
    const prepared = create(LightingRuntimeStatusSchema, { ...disabled(), selection, setupVerified: true });
    vi.spyOn(lightingSetupApi, "status").mockResolvedValue(prepared);
    const setup = vi.spyOn(lightingSetupApi, "setup").mockResolvedValue(prepared);
    renderSettings();
    await waitFor(() => expect(screen.getByRole("button", { name: copy.enableLighting })).toBeEnabled());
    fireEvent.change(screen.getByLabelText(copy.installationSource), { target: { value: String(InstallationSource.EXISTING) } });
    expect(screen.getByRole("button", { name: copy.enableLighting })).toBeDisabled();
    expect(screen.getByLabelText(copy.selectedHash)).toHaveValue("");
    fireEvent.click(screen.getByRole("button", { name: copy.prepareSetup }));
    await waitFor(() => expect(setup).toHaveBeenCalledOnce());
    expect(setup.mock.calls[0]?.[0]).toMatchObject({ source: InstallationSource.EXISTING, sha256: "", devicePolicyId: "approved-policy" });
  });

  it("fences mutation retries after an uncertain error until current status is read", async () => {
    const ownerError = "Owner response lost";
    const setup = vi.spyOn(lightingSetupApi, "setup").mockRejectedValue(new Error(ownerError));
    renderSettings();
    const prepare = await screen.findByRole("button", { name: copy.prepareSetup });
    await waitFor(() => expect(prepare).toBeEnabled()); fireEvent.click(prepare);
    expect(await screen.findByText(ownerError)).toBeInTheDocument();
    expect(prepare).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: copy.ownerRetry }));
    await waitFor(() => expect(prepare).toBeEnabled());
    expect(setup).toHaveBeenCalledOnce();
  });

  it("shows an initial read failure and retries without any mutation", async () => {
    vi.spyOn(lightingSetupApi, "status").mockRejectedValueOnce(new Error("Owner unavailable"));
    const setup = vi.spyOn(lightingSetupApi, "setup");
    renderSettings();
    expect(await screen.findByText(copy.ownerStatusFailed)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: copy.prepareSetup })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: copy.ownerRetry }));
    expect(await screen.findByRole("button", { name: copy.prepareSetup })).toBeInTheDocument();
    expect(setup).not.toHaveBeenCalled();
  });
  it("keeps the setup controls accessible with translated labels", async () => {
    await setLocale("en");
    const { container } = renderSettings();
    await screen.findByLabelText(i18n.t(copy.installationSource));
    await expectNoA11yViolations(container);
  });

  it("keeps theme changes usable and labels them ephemeral when storage is denied", async () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new DOMException("Storage denied", "SecurityError"); });
    renderSettings();
    const theme = await screen.findByRole("button", { name: strings.theme.switcherLabel });
    fireEvent.click(theme);
    fireEvent.click(await screen.findByRole("option", { name: strings.theme.choice.dark }));
    expect(await screen.findByText(copy.themeSessionOnly)).toBeInTheDocument();
    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
    expect(theme).toHaveTextContent(strings.theme.choice.dark);
  });

  it("permits installation-only setup while policy enrollment remains incomplete", async () => {
    const installation = create(LightingSelectionSchema, { ...selection, devicePolicyId: "" });
    vi.spyOn(lightingSetupApi, "status").mockResolvedValue(create(LightingRuntimeStatusSchema, { ...disabled(), pinnedSelection: installation }));
    const setup = vi.spyOn(lightingSetupApi, "setup").mockResolvedValue(create(LightingRuntimeStatusSchema, { ...disabled(), selection: installation, installed: true, setupVerified: false, launchBlocker: "Device policy enrollment is pending" }));
    renderSettings();
    const prepare = await screen.findByRole("button", { name: copy.prepareSetup });
    await waitFor(() => expect(prepare).toBeEnabled());
    fireEvent.click(prepare);
    expect(await screen.findByText(copy.launchBlocked)).toBeInTheDocument();
    expect(setup).toHaveBeenCalledWith(installation);
    expect(screen.getByRole("button", { name: copy.enableLighting })).toBeDisabled();
    expect(screen.getAllByText(copy.unobserved)).toHaveLength(2);
  });

});


it("guides a signed-out setup reader to the existing account without enabling protected actions", async () => {
  await setLocale("en");
  vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined, config: create(PolicyConfigSchema, {}) });
  vi.spyOn(lightingSetupApi, "status").mockRejectedValue(new ConnectError("permission_denied", Code.PermissionDenied));
  const setup = vi.spyOn(lightingSetupApi, "setup");
  const enable = vi.spyOn(lightingSetupApi, "enable");
  renderSettings();
  const link = await screen.findByRole("link", { name: i18n.t(strings.lightDesk.account.signIn) });
  expect(link).toHaveAttribute("href", "/settings#account-heading");
  const panel = screen.getByRole("heading", { name: i18n.t(copy.setupTitle) }).closest("section")!;
  expect(within(panel).getByText(i18n.t(strings.lightDesk.account.intro))).toBeInTheDocument();
  expect(within(panel).queryByText(/permission_denied/)).not.toBeInTheDocument();
  expect(within(panel).queryByRole("button", { name: i18n.t(copy.prepareSetup) })).not.toBeInTheDocument();
  expect(setup).not.toHaveBeenCalled();
  expect(enable).not.toHaveBeenCalled();
  vi.restoreAllMocks();
});


it("keeps setup Account navigation within its route and query under a router basename and root document base", async () => {
  await setLocale("en");
  vi.spyOn(lightingApi, "readDesk").mockResolvedValue({ backend: undefined, controllers: [], zones: [], manualOverride: undefined, effective: undefined, config: create(PolicyConfigSchema, {}) });
  vi.spyOn(lightingSetupApi, "status").mockRejectedValue(new ConnectError("permission_denied", Code.PermissionDenied));
  const setup = vi.spyOn(lightingSetupApi, "setup");
  const enable = vi.spyOn(lightingSetupApi, "enable");
  const base = document.createElement("base");
  base.href = "/";
  document.head.append(base);
  function LocationProof() {
    const { pathname, search, hash } = useLocation();
    return <output aria-label="Current route">{pathname}{search}{hash}</output>;
  }
  try {
    renderWithProviders(<MemoryRouter basename="/apps/lighting-control" initialEntries={["/apps/lighting-control/preferences/settings?panel=setup"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><Routes><Route path="/preferences/settings" element={<><SettingsPage/><LocationProof/></>}/><Route path="/" element={<h1>Home destination</h1>}/></Routes></MemoryRouter>, { withoutRouter: true });
    const link = await screen.findByRole("link", { name: i18n.t(strings.lightDesk.account.signIn) });
    expect(link).toHaveAttribute("href", "/apps/lighting-control/preferences/settings?panel=setup#account-heading");
    const account = screen.getByRole("heading", { name: i18n.t(strings.lightDesk.account.title) });
    const scroll = vi.fn();
    Object.defineProperty(account, "scrollIntoView", { configurable: true, value: scroll });
    for (const modified of [{ metaKey: true }, { ctrlKey: true }, { shiftKey: true }, { altKey: true }, { button: 1 }]) {
      link.focus();
      let preventedBeforeNativeDefault = true;
      const suppressNativeNavigation = (event: globalThis.MouseEvent) => {
        preventedBeforeNativeDefault = event.defaultPrevented;
        event.preventDefault(); // Keep jsdom from attempting a new tab/navigation after React's handlers.
      };
      document.addEventListener("click", suppressNativeNavigation, { once: true });
      fireEvent.click(link, { button: 0, ...modified });
      expect(preventedBeforeNativeDefault).toBe(false);
      expect(link).toHaveFocus();
      expect(scroll).not.toHaveBeenCalled();
      expect(screen.getByRole("status", { name: "Current route" })).toHaveTextContent("/preferences/settings?panel=setup");
      expect(screen.getByRole("status", { name: "Current route" })).not.toHaveTextContent("#account-heading");
    }
    fireEvent.click(link);
    expect(account).toHaveFocus();
    expect(scroll).toHaveBeenCalledWith({ block: "start" });
    expect(screen.getByRole("status", { name: "Current route" })).toHaveTextContent("/preferences/settings?panel=setup#account-heading");
    expect(screen.queryByRole("heading", { name: "Home destination" })).not.toBeInTheDocument();
    expect(setup).not.toHaveBeenCalled();
    expect(enable).not.toHaveBeenCalled();
  } finally { base.remove(); vi.restoreAllMocks(); }
});
