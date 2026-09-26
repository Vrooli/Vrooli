// Wake-word client: reads and persists the wake-word enrollment template
// through audio-tools' own STTAdminService over the same-origin Connect
// transport. No cross-scenario calls — audio-tools owns this surface.

import { createClient as createConnectClient } from "@connectrpc/connect";

import { STTAdminService } from "@vrooli/proto-types/audio-tools/v1/stt/stt_admin_pb";
import { AudioFormat } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";

import { transport } from "../api/client";

const client = createConnectClient(STTAdminService, transport);

export interface WakeWordSample {
  audio: Uint8Array;
  /** Container/codec of `audio` (mirrors common.AudioFormat). */
  format: AudioFormat;
  sampleRateHz: number;
}

export interface WakeWordTemplate {
  label: string;
  threshold: number;
  samples: WakeWordSample[];
}

export interface WakeWordConfig {
  configured: boolean;
  template?: WakeWordTemplate;
}

function decodeConfig(c: {
  configured?: boolean;
  template?: {
    label?: string;
    threshold?: number;
    samples?: { audio?: Uint8Array; format?: AudioFormat; sampleRateHz?: number }[];
  };
} | undefined): WakeWordConfig {
  if (!c) return { configured: false };
  const tpl = c.template;
  return {
    configured: c.configured ?? false,
    template: tpl
      ? {
          label: tpl.label ?? "",
          threshold: tpl.threshold ?? 0,
          samples: (tpl.samples ?? []).map((s) => ({
            audio: s.audio ?? new Uint8Array(),
            format: s.format ?? AudioFormat.UNSPECIFIED,
            sampleRateHz: s.sampleRateHz ?? 0,
          })),
        }
      : undefined,
  };
}

export async function getWakeWordConfig(): Promise<WakeWordConfig> {
  const resp = await client.getWakeWordConfig({});
  return decodeConfig(resp.config);
}

export async function saveWakeWordTemplate(template: WakeWordTemplate): Promise<WakeWordConfig> {
  const resp = await client.updateWakeWordTemplate({
    template: {
      label: template.label,
      threshold: template.threshold,
      samples: template.samples.map((s) => ({
        audio: s.audio,
        format: s.format,
        sampleRateHz: s.sampleRateHz,
      })),
    },
  });
  return decodeConfig(resp.config);
}

export async function deleteWakeWordTemplate(): Promise<WakeWordConfig> {
  const resp = await client.deleteWakeWordTemplate({});
  return decodeConfig(resp.config);
}
