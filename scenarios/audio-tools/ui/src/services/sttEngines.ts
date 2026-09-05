// STT-engine client: lists selectable STT engines, reports the
// shared-resource impact of switching away from one, and persists the
// active engine through audio-tools' own STTService/STTAdminService over
// the same-origin Connect transport. No cross-scenario calls — audio-tools
// owns this surface and serves it to its own UI.

import { create } from "@bufbuild/protobuf";
import { createClient as createConnectClient } from "@connectrpc/connect";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";

import { STTService } from "@vrooli/proto-types/audio-tools/v1/stt/stt_pb";
import { STTAdminService } from "@vrooli/proto-types/audio-tools/v1/stt/stt_admin_pb";

import { transport } from "../api/client";

const sttClient = createConnectClient(STTService, transport);
const adminClient = createConnectClient(STTAdminService, transport);

export interface Engine {
  id: string;
  displayName: string;
  /** "local_resource" | "byok_api" | "vrooli_hosted" */
  kind: string;
  available: boolean;
  nativeStreaming: boolean;
  isActive: boolean;
}

export interface EngineConsumer {
  scenario: string;
  displayName: string;
  required: boolean;
}

export interface EngineSwitchImpact {
  /** Backing resource of the from-engine; empty for non-local engines. */
  resource: string;
  consumers: EngineConsumer[];
  /** True when no other scenario consumes the resource. */
  safeToStop: boolean;
  /** Exact command to stop the resource. The UI NEVER runs this. */
  stopCommand: string;
  /** False when other consumers couldn't be enumerated. */
  consumersKnown: boolean;
}

export async function listEngines(): Promise<Engine[]> {
  const resp = await sttClient.listEngines({});
  return resp.engines.map((e) => ({
    id: e.id,
    displayName: e.displayName,
    kind: e.kind,
    available: e.available,
    nativeStreaming: e.nativeStreaming,
    isActive: e.isActive,
  }));
}

export async function getEngineSwitchImpact(fromEngineId: string): Promise<EngineSwitchImpact> {
  const resp = await adminClient.getEngineSwitchImpact({ fromEngineId });
  return {
    resource: resp.resource,
    consumers: resp.consumers.map((c) => ({
      scenario: c.scenario,
      displayName: c.displayName,
      required: c.required,
    })),
    safeToStop: resp.safeToStop,
    stopCommand: resp.stopCommand,
    consumersKnown: resp.consumersKnown,
  };
}

export async function setEngine(engineId: string): Promise<void> {
  await adminClient.updateStreamConfig({
    updateMask: create(FieldMaskSchema, { paths: ["engine_id"] }),
    config: { engineId },
  });
}
