import { createClient } from "@connectrpc/connect";
import { SettingsService } from "@vrooli/proto-types/web-console/v1/settings/settings_pb";

import type { ExpirationPolicy, PolicyMode } from "./sessions";
import { transport } from "./client";

export const settingsClient = createClient(SettingsService, transport);

export interface SessionDefaultsResponse {
  default_backend: string;
  default_policy: ExpirationPolicy;
}

function decodePolicy(p: { mode?: string; duration?: string } | undefined): ExpirationPolicy {
  return {
    mode: ((p?.mode as PolicyMode) || "never"),
    duration: p?.duration || "",
  };
}

export async function getSessionDefaults(): Promise<SessionDefaultsResponse> {
  const resp = await settingsClient.getSessionDefaults({});
  const d = resp.defaults;
  return {
    default_backend: d?.defaultBackend ?? "",
    default_policy: decodePolicy(d?.defaultPolicy),
  };
}

export async function updateSessionDefaults(update: {
  default_backend?: string;
  default_policy?: ExpirationPolicy;
}): Promise<SessionDefaultsResponse> {
  const req: {
    defaultBackend?: string;
    defaultPolicy?: { mode: string; duration: string };
  } = {};
  if (update.default_backend !== undefined) {
    req.defaultBackend = update.default_backend;
  }
  if (update.default_policy !== undefined) {
    req.defaultPolicy = {
      mode: update.default_policy.mode,
      duration: update.default_policy.duration ?? "",
    };
  }
  const resp = await settingsClient.updateSessionDefaults(req);
  const d = resp.defaults;
  return {
    default_backend: d?.defaultBackend ?? "",
    default_policy: decodePolicy(d?.defaultPolicy),
  };
}
