// API client for the inventory domain. Thin wrapper over the generated
// InventoryService Connect client. ScanScenario walks a single scenario's
// UI tree and returns SurfaceRecord[], ComponentProvenance[], and
// WidgetDeclaration[]. The backend does not expose a global "list all
// surfaces" RPC, so the UI is always scenario-scoped.
import { createClient } from "@connectrpc/connect";

import {
  InventoryService,
  type ScanScenarioResponse as ProtoResponse,
  type SurfaceRecord as ProtoSurfaceRecord,
} from "@vrooli/proto-types/ui-health/v1/inventory/inventory_pb";
import { Provenance as ProtoProvenance } from "@vrooli/proto-types/ui-health/v1/contracts/provenance/provenance_pb";
import type { ComponentProvenance as ProtoComponentProvenance } from "@vrooli/proto-types/ui-health/v1/contracts/provenance/provenance_pb";
import type { WidgetDeclaration as ProtoWidgetDeclaration } from "@vrooli/proto-types/ui-health/v1/contracts/widget/widget_pb";

import { transport } from "./client";
import {
  surfaceKindFromProto,
  SURFACE_KIND_FILTERS,
  type ProvenanceTag,
  type SurfaceKind,
  type SurfaceKindFilter,
} from "./search";

export const inventoryClient = createClient(InventoryService, transport);

export type SurfaceRecord = {
  scenario: string;
  slot: string;
  kind: SurfaceKind;
  displayName: string;
  description: string;
  filePath: string;
};

export type ProvenanceRecord = {
  provenance: ProvenanceTag;
  library: string;
  libraryVersion: string;
  componentName: string;
  adoptionId: string;
};

export type WidgetRecord = {
  widgetId: string;
  componentName: string;
  propsSchemaJson: string;
};

export type InventoryScan = {
  scenario: string;
  surfaces: SurfaceRecord[];
  provenance: ProvenanceRecord[];
  widgets: WidgetRecord[];
  scannedAt: string;
};

export async function scanScenario(scenario: string): Promise<InventoryScan> {
  const resp = await inventoryClient.scanScenario({ scenario });
  return scanFromProto(resp);
}

function provenanceTagFromProto(p: ProtoProvenance | undefined): ProvenanceTag {
  switch (p) {
    case ProtoProvenance.CUSTOM:
      return "custom";
    case ProtoProvenance.ADOPTED_UNMODIFIED:
      return "adopted-unmodified";
    case ProtoProvenance.ADOPTED_MODIFIED:
      return "adopted-modified";
    case ProtoProvenance.UNKNOWN:
      return "unknown";
    default:
      return "unspecified";
  }
}

function surfaceFromProto(s: ProtoSurfaceRecord): SurfaceRecord {
  return {
    scenario: s.scenario,
    slot: s.slot,
    kind: surfaceKindFromProto(s.kind),
    displayName: s.displayName,
    description: s.description,
    filePath: s.filePath,
  };
}

function provenanceFromProto(p: ProtoComponentProvenance): ProvenanceRecord {
  return {
    provenance: provenanceTagFromProto(p.provenance),
    library: p.library,
    libraryVersion: p.libraryVersion,
    componentName: p.componentName,
    adoptionId: p.adoptionId,
  };
}

function widgetFromProto(w: ProtoWidgetDeclaration): WidgetRecord {
  return {
    widgetId: w.widgetId,
    componentName: w.componentName,
    propsSchemaJson: w.propsSchemaJson,
  };
}

function scanFromProto(p: ProtoResponse): InventoryScan {
  return {
    scenario: p.scenario,
    surfaces: p.surfaces.map(surfaceFromProto),
    provenance: p.provenance.map(provenanceFromProto),
    widgets: p.widgets.map(widgetFromProto),
    scannedAt: new Date().toISOString(),
  };
}

export function filterSurfaces(
  surfaces: SurfaceRecord[],
  kind: SurfaceKindFilter,
): SurfaceRecord[] {
  if (kind === "all") return surfaces;
  return surfaces.filter((s) => s.kind === kind);
}

export function countSurfacesByKind(
  surfaces: SurfaceRecord[],
): Record<SurfaceKindFilter, number> {
  const counts: Record<SurfaceKindFilter, number> = {
    all: surfaces.length,
    component: 0,
    page: 0,
    feature: 0,
    hook: 0,
    layout: 0,
    other: 0,
  };
  for (const s of surfaces) {
    if (s.kind === "unspecified") continue;
    counts[s.kind] += 1;
  }
  return counts;
}

export { SURFACE_KIND_FILTERS };
export type { SurfaceKind, SurfaceKindFilter, ProvenanceTag };

export function encodeSurfaceId(scenario: string, slot: string): string {
  return `${scenario}__${slot || "_"}`;
}

export function decodeSurfaceId(
  id: string,
): { scenario: string; slot: string } | null {
  const idx = id.indexOf("__");
  if (idx < 0) return null;
  const scenario = id.slice(0, idx);
  const slotRaw = id.slice(idx + 2);
  if (!scenario) return null;
  return { scenario, slot: slotRaw === "_" ? "" : slotRaw };
}
