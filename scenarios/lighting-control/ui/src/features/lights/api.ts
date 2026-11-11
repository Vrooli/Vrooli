import { createClient } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";

import { ControlService, GetBackendStatusRequestSchema, GetDesiredStateRequestSchema, ListControllersRequestSchema, SetManualOffRequestSchema, ClearManualOverrideRequestSchema, ReconcileRequestSchema, EnrollControllerRequestSchema, SetZoneStateRequestSchema, ZoneStateSchema, ZoneStatusSchema, StaticZoneStateSchema, EffectZoneStateSchema, ColorSchema } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";
import { PolicyService, GetPolicyRequestSchema, SetPolicyRequestSchema, PolicyConfigSchema, EvaluatePolicyRequestSchema, SetManualPresetRequestSchema, ClearManualPresetRequestSchema, PresetKind, type PolicyConfig } from "@vrooli/proto-types/lighting-control/v1/policy/policy_pb";

import { transport } from "../../api/client";

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
      const coversController = override?.active && (override.allEnrolled || override.controllerIds.includes(zone.desired?.controllerId ?? ""));
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
      backend: backend.backend,
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
