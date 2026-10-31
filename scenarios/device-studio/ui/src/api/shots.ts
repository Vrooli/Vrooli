import { ConnectError } from "@connectrpc/connect";
import { createClient } from "@connectrpc/connect";
import { ShotConflictSchema, ShotsService } from "@vrooli/proto-types/device-studio/v1/shots/shots_pb";
import { transport } from "./client";

export const shotsClient = createClient(ShotsService, transport);

export function shotConflict(error: unknown) {
  if (!(error instanceof ConnectError)) return undefined;
  return error.findDetails(ShotConflictSchema)[0];
}
