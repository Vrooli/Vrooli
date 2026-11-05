import { describe, expect, it } from "vitest";
import { DeleteConfirmLevel } from "@vrooli/proto-types/swarm-manager/v1/domain/settings_pb";
import { SettingsFieldRole } from "@vrooli/proto-types/swarm-manager/v1/api/settings_pb";
import { mapProtoPolicyProjection, mapProtoSettings } from "./settings-contracts";
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

  it("maps the policy projection: effective controls, roles, and control paths", () => {
    const result = mapProtoPolicyProjection({
      effectiveControls: {
        defaultMode: "yolo",
        autoFixup: true,
        maxFixupAttempts: 3,
        reviewAgentEnabled: true,
        reviewCodeQualityMinScore: 60,
        reviewTestMinPassRate: 1,
        reviewMaxBlockingViolations: 0,
        reviewMaxWarnings: -1,
        reviewRequireScreenshots: true,
        reviewRequireTests: true,
        agentMaxTurns: 600,
        agentTimeoutSeconds: 3600,
      },
      classifications: [
        {
          field: "max_fixup_attempts",
          role: SettingsFieldRole.POLICY_CONTROL,
          control: "retry.max_fixup_attempts",
          note: "Retained user preference.",
        },
        {
          field: "agent_timeout_seconds",
          role: SettingsFieldRole.DORMANT,
          control: "budgets.timeout_seconds",
          note: "No runtime reader.",
        },
      ],
    } as unknown as Parameters<typeof mapProtoPolicyProjection>[0]);

    expect(result).not.toBeNull();
    expect(result?.effectiveControls.defaultMode).toBe("yolo");
    expect(result?.effectiveControls.reviewMaxWarnings).toBe(-1);
    expect(result?.classifications).toEqual([
      {
        field: "max_fixup_attempts",
        role: "policy_control",
        control: "retry.max_fixup_attempts",
        note: "Retained user preference.",
      },
      {
        field: "agent_timeout_seconds",
        role: "dormant",
        control: "budgets.timeout_seconds",
        note: "No runtime reader.",
      },
    ]);
  });

  it("returns null for a missing projection (older API)", () => {
    expect(mapProtoPolicyProjection(undefined)).toBeNull();
    expect(
      mapProtoPolicyProjection({} as unknown as Parameters<typeof mapProtoPolicyProjection>[0]),
    ).toBeNull();
  });
});
