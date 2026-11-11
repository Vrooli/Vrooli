import { create } from "@bufbuild/protobuf";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { ZoneCapabilitySchema, ZoneStateSchema } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";
import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { lightingApi } from "./api";
import { ManualZoneEditor } from "./ManualZoneEditor";

const zone = create(ZoneCapabilitySchema, { zoneId: "main", name: "Main", supportsStaticColor: true, supportedEffects: ["Breathing", "Rainbow"] });
const act = async (action: () => Promise<unknown>) => { await action(); };
afterEach(() => vi.restoreAllMocks());

it("sends only the selected zone's supported effect and caps a saved draft after the maximum changes", async () => {
  const apply = vi.spyOn(lightingApi, "setEffect").mockResolvedValue({} as never);
  const desired = create(ZoneStateSchema, { controllerId: "gpu", zoneId: "main", desiredMode: { case: "effect", value: { effectId: "Breathing", color: { red: 12, green: 34, blue: 56 }, brightnessPercent: 80 } } });
  const view = renderWithProviders(<ManualZoneEditor controllerId="gpu" zone={zone} desired={desired} maxBrightness={90} disabled={false} act={act}/>);
  expect(screen.getByRole("combobox")).toHaveValue("effect:Breathing");
  view.rerender(<ManualZoneEditor controllerId="gpu" zone={zone} desired={desired} maxBrightness={10} disabled={false} act={act}/>);
  expect(screen.getByRole("slider")).toHaveAttribute("max", "10");
  fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.controllerUi.saveApply }));
  await waitFor(() => expect(apply).toHaveBeenCalledWith("gpu", "main", "Breathing", { red: 12, green: 34, blue: 56 }, 10));
});

it("supports an effect-only zone and retains unsupported saved effect feedback without sending it", () => {
  const apply = vi.spyOn(lightingApi, "setEffect");
  const effectOnly = create(ZoneCapabilitySchema, { zoneId: "main", name: "Main", supportedEffects: ["Breathing"] });
  const view = renderWithProviders(<ManualZoneEditor controllerId="gpu" zone={effectOnly} maxBrightness={35} disabled={false} act={act}/>);
  expect(screen.getByRole("combobox")).toHaveValue("effect:Breathing");
  expect(screen.queryByRole("option", { name: strings.lightDesk.staticColor })).not.toBeInTheDocument();
  view.rerender(<ManualZoneEditor controllerId="gpu" zone={create(ZoneCapabilitySchema, { ...effectOnly, supportedEffects: [] })} maxBrightness={35} disabled={false} act={act}/>);
  expect(screen.getByText(strings.lightDesk.manualUi.unsupported)).toBeInTheDocument();
  expect(screen.getByRole("button")).toBeDisabled();
  fireEvent.click(screen.getByRole("button"));
  expect(apply).not.toHaveBeenCalled();
});

it("keeps separate drafts for different zones and leaves edits unapplied until saved", async () => {
  const apply = vi.spyOn(lightingApi, "setStatic").mockResolvedValue({} as never);
  renderWithProviders(<><div role="group" aria-label="Main"><ManualZoneEditor controllerId="gpu" zone={zone} maxBrightness={35} disabled={false} act={act}/></div><div role="group" aria-label="Logo"><ManualZoneEditor controllerId="gpu" zone={create(ZoneCapabilitySchema, { ...zone, zoneId: "logo", name: "Logo" })} maxBrightness={35} disabled={false} act={act}/></div></>);
  const mainColorLabel = `${strings.lightDesk.color} Main`;
  fireEvent.change(screen.getByLabelText(mainColorLabel), { target: { value: "#123456" } });
  expect(apply).not.toHaveBeenCalled();
  fireEvent.click(within(screen.getByRole("group", { name: "Logo" })).getByRole("button"));
  await waitFor(() => expect(apply).toHaveBeenCalledWith("gpu", "logo", { red: 87, green: 190, blue: 138 }, 20));
});

it("cancels edits back to the saved zone draft without sending a command", () => {
  const apply = vi.spyOn(lightingApi, "setEffect");
  const desired = create(ZoneStateSchema, { controllerId: "gpu", zoneId: "main", desiredMode: { case: "effect", value: { effectId: "Breathing", color: { red: 12, green: 34, blue: 56 }, brightnessPercent: 8 } } });
  renderWithProviders(<ManualZoneEditor controllerId="gpu" zone={zone} desired={desired} maxBrightness={35} disabled={false} detail act={act}/>);
  fireEvent.change(screen.getByRole("combobox"), { target: { value: "effect:Rainbow" } });
  fireEvent.change(screen.getByRole("slider"), { target: { value: "30" } });
  fireEvent.click(screen.getByRole("button", { name: strings.lightDesk.presetsUi.cancelEdits }));
  expect(screen.getByRole("combobox")).toHaveValue("effect:Breathing");
  expect(screen.getByRole("slider")).toHaveValue("8");
  expect(apply).not.toHaveBeenCalled();
});

it("keeps manual commands disabled when the authoritative maximum is unavailable", () => {
  const apply = vi.spyOn(lightingApi, "setStatic");
  renderWithProviders(<ManualZoneEditor controllerId="gpu" zone={zone} disabled={false} act={act}/>);
  expect(screen.getByText(strings.lightDesk.policyUnavailable)).toBeInTheDocument();
  expect(screen.getByRole("button")).toBeDisabled();
  fireEvent.click(screen.getByRole("button"));
  expect(apply).not.toHaveBeenCalled();
});
