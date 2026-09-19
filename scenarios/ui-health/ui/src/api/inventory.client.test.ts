// Exercises the FromProto conversion chain by stubbing the underlying
// Connect client. Without this, the helpers stay uncovered because every
// page-level test mocks scanScenario directly.
import { describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";

import { ScanScenarioResponseSchema } from "@vrooli/proto-types/ui-health/v1/inventory/inventory_pb";
import { SurfaceKind } from "@vrooli/proto-types/ui-health/v1/search/search_pb";
import { Provenance } from "@vrooli/proto-types/ui-health/v1/contracts/provenance/provenance_pb";

import { inventoryClient, scanScenario } from "./inventory";

describe("scanScenario (FromProto)", () => {
  it("converts a full proto response into plain TS shapes", async () => {
    const protoResp = create(ScanScenarioResponseSchema, {
      scenario: "ui-health",
      surfaces: [
        {
          scenario: "ui-health",
          slot: "DashboardPage",
          kind: SurfaceKind.PAGE,
          displayName: "DashboardPage",
          description: "main",
          filePath: "ui/src/pages/DashboardPage.tsx",
        },
        {
          scenario: "ui-health",
          slot: "Header",
          kind: SurfaceKind.COMPONENT,
          displayName: "Header",
          description: "",
          filePath: "",
        },
      ],
      provenance: [
        {
          provenance: Provenance.ADOPTED_MODIFIED,
          library: "react-component-library",
          libraryVersion: "1.2.3",
          componentName: "Card",
          adoptionId: "a1",
        },
        {
          provenance: Provenance.CUSTOM,
          library: "",
          libraryVersion: "",
          componentName: "Local",
          adoptionId: "",
        },
        {
          provenance: Provenance.UNKNOWN,
          library: "",
          libraryVersion: "",
          componentName: "",
          adoptionId: "",
        },
      ],
      widgets: [
        {
          widgetId: "ui-health.widget.dashboard-card",
          componentName: "DashboardCard",
          propsSchemaJson: '{"a":1}',
        },
      ],
    });
    vi.spyOn(inventoryClient, "scanScenario").mockResolvedValueOnce(protoResp);
    const out = await scanScenario("ui-health");
    expect(out.scenario).toBe("ui-health");
    expect(out.surfaces).toHaveLength(2);
    expect(out.surfaces[0]?.kind).toBe("page");
    expect(out.surfaces[1]?.kind).toBe("component");
    expect(out.provenance.map((p) => p.provenance)).toEqual([
      "adopted-modified",
      "custom",
      "unknown",
    ]);
    expect(out.widgets[0]?.widgetId).toBe("ui-health.widget.dashboard-card");
  });
});
