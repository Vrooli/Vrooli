import { create } from "@bufbuild/protobuf";
import { createClient } from "@connectrpc/connect";
import { SetupService, GetSetupStatusRequestSchema, SetupLightingRequestSchema, EnableLightingRequestSchema, DisableLightingRequestSchema, GetSetupDevicesRequestSchema, PlanLightingDevicesRequestSchema, ApplyLightingDevicesRequestSchema, RollbackLightingDevicesRequestSchema, type LightingSelection } from "@vrooli/proto-types/lighting-control/v1/setup/setup_pb";
import { transport } from "../../api/client";

const client = createClient(SetupService, transport);

export const lightingSetupApi = {
  devices() {
    return client.getSetupDevices(create(GetSetupDevicesRequestSchema, {}));
  },
  planDevices(deviceIds: string[]) {
    return client.planLightingDevices(create(PlanLightingDevicesRequestSchema, { deviceIds }));
  },
  applyDevices(planId: string) {
    return client.applyLightingDevices(create(ApplyLightingDevicesRequestSchema, { planId }));
  },
  rollbackDevices(planId: string) {
    return client.rollbackLightingDevices(create(RollbackLightingDevicesRequestSchema, { planId }));
  },
  status() {
    return client.getSetupStatus(create(GetSetupStatusRequestSchema, {}));
  },
  setup(selection: LightingSelection) {
    return client.setupLighting(create(SetupLightingRequestSchema, { selection }));
  },
  enable(selection: LightingSelection) {
    return client.enableLighting(create(EnableLightingRequestSchema, { selection }));
  },
  disable() {
    return client.disableLighting(create(DisableLightingRequestSchema, {}));
  },
};
