import { describe, expect, it } from "vitest";
import { DeleteConfirmLevel } from "@vrooli/proto-types/swarm-manager/v1/domain/settings_pb";
import { mapProtoSettings } from "./settings-contracts";
import { defaultDeleteConfirmationLevels } from "../../lib/deletable-entities";

describe("settings proto contracts", () => {
  it("normalizes provided levels and fills missing entities from defaults", () => {
    const result = mapProtoSettings({
      theme: "dark",
      defaultMode: "manual",
      deleteConfirmationLevels: {
        // backlog provided as UNSPECIFIED → coerced to simple
        backlog: DeleteConfirmLevel.UNSPECIFIED,
        // explicit override
        session: DeleteConfirmLevel.STRONG,
        capture: DeleteConfirmLevel.NONE,
        // unknown forward-compat key must survive
        futureThing: DeleteConfirmLevel.STRONG,
      },
    } as unknown as Parameters<typeof mapProtoSettings>[0]);

    // Provided values win where valid.
    expect(result.deleteConfirmation.session).toBe("strong");
    expect(result.deleteConfirmation.capture).toBe("none");
    // UNSPECIFIED maps to simple.
    expect(result.deleteConfirmation.backlog).toBe("simple");
    // Missing entities fall back to registry defaults.
    expect(result.deleteConfirmation.scenario).toBe(
      defaultDeleteConfirmationLevels().scenario,
    );
    // Unknown key preserved.
    expect((result.deleteConfirmation as Record<string, string>).futureThing).toBe("strong");
  });

  it("falls back to all registry defaults when the map is absent", () => {
    const result = mapProtoSettings({
      theme: "dark",
      defaultMode: "manual",
    } as unknown as Parameters<typeof mapProtoSettings>[0]);

    expect(result.deleteConfirmation).toEqual(defaultDeleteConfirmationLevels());
  });
});
