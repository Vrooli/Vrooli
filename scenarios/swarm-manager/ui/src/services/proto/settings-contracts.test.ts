import { describe, expect, it } from "vitest";
import { DeleteConfirmLevel } from "@vrooli/proto-types/swarm-manager/v1/domain/settings_pb";
import { mapProtoSettings } from "./settings-contracts";

describe("settings proto contracts", () => {
  it("normalizes unspecified delete confirmation levels to simple", () => {
    const result = mapProtoSettings({
      theme: "dark",
      defaultMode: "manual",
      deleteConfirmation: {
        backlog: DeleteConfirmLevel.UNSPECIFIED,
        initiative: DeleteConfirmLevel.SIMPLE,
        capture: DeleteConfirmLevel.NONE,
      },
    } as Parameters<typeof mapProtoSettings>[0]);

    expect(result.deleteConfirmation).toEqual({
      backlog: "simple",
      initiative: "simple",
      capture: "none",
    });
  });
});
