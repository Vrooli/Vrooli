import { describe, expect, it } from "vitest";
import { Code, ConnectError } from "@connectrpc/connect";
import { ShotConflictSchema } from "@vrooli/proto-types/device-studio/v1/shots/shots_pb";
import { shotConflict } from "./shots";

describe("shot conflict details", () => {
  it("returns only the typed conflict detail from an aborted save", () => {
    const error = new ConnectError("conflict", Code.Aborted, undefined, [{ desc: ShotConflictSchema, value: { shotId: "shot-1", expectedRevision: 2n, currentRevision: 3n } }]);
    expect(shotConflict(error)).toMatchObject({ shotId: "shot-1", expectedRevision: 2n, currentRevision: 3n });
    expect(shotConflict(new Error("unrelated"))).toBeUndefined();
  });
});
