import { createClient } from "@connectrpc/connect";
import {
  SettingsService,
  type ExpirationPolicy as ProtoExpirationPolicy,
  type SessionDefaults,
  type UpdateSessionDefaultsRequest as ProtoUpdateRequest,
} from "@vrooli/proto-types/web-console/v1/settings/settings_pb";

import { transport } from "./client";

// settingsClient is the Connect-Web client for SettingsService. UI code
// imports this directly; the legacy REST helpers in lib/api.ts are
// shims that delegate here.
export const settingsClient = createClient(SettingsService, transport);

export type { SessionDefaults, ProtoExpirationPolicy, ProtoUpdateRequest };
