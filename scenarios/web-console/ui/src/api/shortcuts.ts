import { createClient } from "@connectrpc/connect";
import { ShortcutsService } from "@vrooli/proto-types/web-console/v1/shortcuts/shortcuts_pb";

import { transport } from "./client";

// shortcutsClient is the Connect-Web client for ShortcutsService.
// UI code imports this directly; the legacy fetch helpers in lib/api.ts
// are shims that delegate here and normalize the camelCase proto shape
// to the existing snake_case wire types.
export const shortcutsClient = createClient(ShortcutsService, transport);
