import { createClient } from "@connectrpc/connect";
import { SettingsService } from "@vrooli/proto-types/web-console/v1/settings/settings_pb";
import { transport } from "./client";
export const settingsClient = createClient(SettingsService, transport);
function decodePolicy(p) {
    return {
        mode: (p?.mode || "never"),
        duration: p?.duration || "",
    };
}
export async function getSessionDefaults() {
    const resp = await settingsClient.getSessionDefaults({});
    const d = resp.defaults;
    return {
        default_backend: d?.defaultBackend ?? "",
        default_policy: decodePolicy(d?.defaultPolicy),
    };
}
export async function updateSessionDefaults(update) {
    const req = {};
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
