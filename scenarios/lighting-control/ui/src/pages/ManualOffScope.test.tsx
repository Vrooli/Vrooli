import { create } from "@bufbuild/protobuf";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { BackendState, BackendStatusSchema, ControllerSchema, ManualOverrideSchema } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";
import { EffectivePolicySchema, PolicyConfigSchema, PresetKind } from "@vrooli/proto-types/lighting-control/v1/policy/policy_pb";
import { lightingApi } from "../features/lights/api";
import { strings } from "../consts/strings";
import { selectors } from "../consts/selectors";
import { i18n, setLocale } from "../i18n";
import { renderWithProviders } from "../test-utils";
import { DashboardPage } from "./DashboardPage";
import { ControllerPage } from "./ControllerPage";
import { PresetsPage } from "./PresetsPage";

const controllers = ["selected", "other"].map((controllerId) => create(ControllerSchema, {
  controllerId, name: controllerId, enrolled: true, identitySource: "openrgb_sdk_serial",
}));
const cases = [
  { name: "another controller scoped Off", active: true, allEnrolled: false, controllerIds: ["selected"], disabled: false },
  { name: "displayed controller scoped Off", active: true, allEnrolled: false, controllerIds: ["other"], disabled: false },
  { name: "all current IDs scoped Off", active: true, allEnrolled: false, controllerIds: ["selected", "other"], disabled: false },
  { name: "unknown active scope", active: true, allEnrolled: false, controllerIds: [], disabled: false },
  { name: "inactive remembered global scope", active: false, allEnrolled: true, controllerIds: [], disabled: false },
  { name: "explicit all-enrolled scope", active: true, allEnrolled: true, controllerIds: [], disabled: true },
];
beforeEach(async () => { await setLocale("en"); });
afterEach(() => { vi.restoreAllMocks(); localStorage.clear(); });

for (const page of ["dashboard", "controller", "presets"] as const) {
  for (const phone of [false, true]) {
    it.each(cases)(`${page} ${phone ? "phone" : "desktop"}: global Off availability under $name`, async (scope) => {
      vi.spyOn(window, "matchMedia").mockImplementation((query) => ({ matches: phone && query === "(max-width: 720px)", media: query, onchange: null, addEventListener: () => undefined, removeEventListener: () => undefined, addListener: () => undefined, removeListener: () => undefined, dispatchEvent: () => false } as MediaQueryList));
      vi.spyOn(lightingApi, "readDesk").mockResolvedValue({
        backend: create(BackendStatusSchema, { state: BackendState.UNAVAILABLE }), controllers, zones: [],
        manualOverride: create(ManualOverrideSchema, scope),
        effective: create(EffectivePolicySchema, { preset: PresetKind.WORK }),
        config: create(PolicyConfigSchema, { defaultPreset: PresetKind.WORK, brightnessCapPercent: 35, presets: [{ kind: PresetKind.WORK, label: "Work", enabled: true }] }),
      });
      const off = vi.spyOn(lightingApi, "setOff").mockResolvedValue({} as never);
      const clear = vi.spyOn(lightingApi, "clearOff").mockResolvedValue({} as never);
      const content = page === "dashboard" ? <DashboardPage /> : page === "presets" ? <PresetsPage /> : <Routes><Route path="/controllers/:controllerId" element={<ControllerPage />} /></Routes>;
      renderWithProviders(<MemoryRouter initialEntries={["/controllers/other"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>{content}</MemoryRouter>, { withoutRouter: true });
      const button = page === "controller" ? await screen.findByRole("button", { name: i18n.t(strings.lightDesk.controllerUi.keepOff) }) : await screen.findByTestId(selectors.lightDesk.manualOff);
      if (scope.disabled) expect(button).toBeDisabled();
      else {
        expect(button).toBeEnabled();
        fireEvent.click(button);
        await waitFor(() => expect(off).toHaveBeenCalledOnce());
      }
      expect(clear).not.toHaveBeenCalled();
      if (scope.active && !scope.allEnrolled) {
        const description = scope.controllerIds.length ? i18n.t(strings.lightDesk.manualOffScope, { controllers: scope.controllerIds.map((id) => `${id} (${id})`).join(", ") }) : i18n.t(strings.lightDesk.manualOffScopeUnknown);
        expect(screen.getAllByText(description).length).toBeGreaterThan(0);
        expect(screen.queryByText(i18n.t(strings.lightDesk.manualOffActive))).not.toBeInTheDocument();
      }
      if (page === "controller") {
        const label = screen.getByText(i18n.t(strings.lightDesk.controllerUi.desiredPolicy)).parentElement!;
        const covered = scope.active && (scope.allEnrolled || scope.controllerIds.includes("other"));
        expect(within(label).getByText(i18n.t(covered ? strings.lightDesk.presets.off : strings.lightDesk.presets.work))).toBeInTheDocument();
      }
    });
  }
}
