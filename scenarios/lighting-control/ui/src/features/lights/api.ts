import { createClient } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";

import { BackendState, ControlService, ConfigureZoneRequestSchema, GetBackendStatusRequestSchema, GetDesiredStateRequestSchema, ListControllersRequestSchema, SetManualOffRequestSchema, ClearManualOverrideRequestSchema, ReconcileRequestSchema, EnrollControllerRequestSchema, SetZoneStateRequestSchema, ZoneStateSchema, ZoneStatusSchema, StaticZoneStateSchema, EffectZoneStateSchema, ColorSchema } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";
import { PolicyService, GetPolicyRequestSchema, SetPolicyRequestSchema, PolicyConfigSchema, EvaluatePolicyRequestSchema, SetManualPresetRequestSchema, ClearManualPresetRequestSchema, PresetKind, type PolicyConfig } from "@vrooli/proto-types/lighting-control/v1/policy/policy_pb";

import { transport } from "../../api/client";

import { manualOffCoversController } from "./manualOff";

const control = createClient(ControlService, transport);
const policy = createClient(PolicyService, transport);

export const lightingApi = {
  async readDesk() {
    const [backend, inventory, desired, currentPolicy] = await Promise.all([
      control.getBackendStatus(create(GetBackendStatusRequestSchema, {})),
      control.listControllers(create(ListControllersRequestSchema, { includeUnavailable: true })),
      control.getDesiredState(create(GetDesiredStateRequestSchema, {})),
      policy.getPolicy(create(GetPolicyRequestSchema, {})),
    ]);
    const override = desired.manualOverride;
    const zones = desired.zones.map((zone) => {
      const coversController = manualOffCoversController(override, zone.desired?.controllerId ?? "");
      if (!coversController || !zone.desired) return zone;
      return create(ZoneStatusSchema, {
        ...zone,
        desired: create(ZoneStateSchema, {
          controllerId: zone.desired.controllerId,
          zoneId: zone.desired.zoneId,
          desiredMode: { case: "off", value: {} },
        }),
      });
    });
    return {
      // A protocol probe cannot override failed discovery, and a later
      // unavailable probe cannot be upgraded by an earlier inventory.
      backend: backend.backend && backend.backend.state !== BackendState.READY
        ? backend.backend : inventory.backend ?? backend.backend,
      controllers: inventory.controllers,
      zones,
      manualOverride: desired.manualOverride,
      effective: currentPolicy.effective,
      config: currentPolicy.config,
    };
  },
  enroll(controllerId: string, enrolled: boolean) {
    return control.enrollController(create(EnrollControllerRequestSchema, { controllerId, enrolled }));
  },
  configureZone(controllerId: string, zoneId: string, ledCount: number) {
    return control.configureZone(create(ConfigureZoneRequestSchema, { controllerId, zoneId, ledCount }));
  },
  setOff() {
    return control.setManualOff(create(SetManualOffRequestSchema, { allEnrolled: true }));
  },
  clearOff() {
    return control.clearManualOverride(create(ClearManualOverrideRequestSchema, {}));
  },
  reconcile() {
    return control.reconcile(create(ReconcileRequestSchema, { trigger: "operator_ui" }));
  },
  choosePreset(preset: PresetKind) {
    return policy.setManualPreset(create(SetManualPresetRequestSchema, { preset }));
  },
  clearPreset() {
    return policy.clearManualPreset(create(ClearManualPresetRequestSchema, {}));
  },
  savePolicy(config: PolicyConfig) {
    return policy.setPolicy(create(SetPolicyRequestSchema, { config: create(PolicyConfigSchema, config) }));
  },
  evaluatePolicy() {
    return policy.evaluatePolicy(create(EvaluatePolicyRequestSchema, { trigger: "operator_settings" }));
  },
  setEffect(controllerId: string, zoneId: string, effectId: string, color: { red: number; green: number; blue: number }, brightnessPercent: number) {
    return control.setZoneState(create(SetZoneStateRequestSchema, {
      desired: create(ZoneStateSchema, {
        controllerId, zoneId,
        desiredMode: { case: "effect", value: create(EffectZoneStateSchema, { effectId, color: create(ColorSchema, color), brightnessPercent }) },
      }),
    }));
  },
  setStatic(controllerId: string, zoneId: string, color: { red: number; green: number; blue: number }, brightnessPercent: number) {
    const desired = create(ZoneStateSchema, {
      controllerId,
      zoneId,
      desiredMode: {
        case: "staticColor",
        value: create(StaticZoneStateSchema, {
          color: create(ColorSchema, color),
          brightnessPercent,
        }),
      },
    });
    return control.setZoneState(create(SetZoneStateRequestSchema, { desired }));
  },
};

export { PresetKind };
