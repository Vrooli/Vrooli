import { describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";

import {
  Mode,
  SearchResponseSchema,
  StatusResponseSchema,
  SurfaceKind,
} from "@vrooli/proto-types/ui-health/v1/search/search_pb";
import { Provenance } from "@vrooli/proto-types/ui-health/v1/contracts/provenance/provenance_pb";

import { searchClient, searchStatus, searchSurfaces, surfaceKindFromProto } from "./search";

describe("searchSurfaces (FromProto)", () => {
  it("converts every kind/provenance combination", async () => {
    const proto = create(SearchResponseSchema, {
      modeUsed: Mode.AI,
      results: [
        {
          scenario: "ui-health",
          slot: "Dash",
          kind: SurfaceKind.PAGE,
          displayName: "Dash",
          description: "",
          filePath: "",
          score: 0.9,
          provenance: {
            provenance: Provenance.CUSTOM,
            library: "lib",
            libraryVersion: "1",
            componentName: "Card",
            adoptionId: "a",
          },
        },
        {
          scenario: "ui-health",
          slot: "H",
          kind: SurfaceKind.HOOK,
          displayName: "H",
          description: "",
          filePath: "",
          score: 0.1,
        },
      ],
    });
    vi.spyOn(searchClient, "search").mockResolvedValueOnce(proto);
    const out = await searchSurfaces("hi");
    expect(out.modeUsed).toBe("ai");
    expect(out.hits.map((h) => h.kind)).toEqual(["page", "hook"]);
    expect(out.hits[0]?.provenance).toBe("custom");
    expect(out.hits[1]?.provenance).toBe("unspecified");
  });
});

describe("surfaceKindFromProto", () => {
  it.each([
    [SurfaceKind.COMPONENT, "component"],
    [SurfaceKind.PAGE, "page"],
    [SurfaceKind.FEATURE, "feature"],
    [SurfaceKind.HOOK, "hook"],
    [SurfaceKind.LAYOUT, "layout"],
    [SurfaceKind.OTHER, "other"],
  ] as const)("maps %s → %s", (input, expected) => {
    expect(surfaceKindFromProto(input)).toBe(expected);
  });
});

describe("searchStatus (FromProto)", () => {
  it("forwards status fields verbatim", async () => {
    const proto = create(StatusResponseSchema, {
      available: true,
      ollama: false,
      qdrant: true,
      indexedCount: 17,
      lastReconcileAt: "2026-05-20T10:00:00Z",
      lastReconcileOutcome: "succeeded",
    });
    vi.spyOn(searchClient, "status").mockResolvedValueOnce(proto);
    const out = await searchStatus();
    expect(out).toEqual({
      available: true,
      ollama: false,
      qdrant: true,
      indexedCount: 17,
      lastReconcileAt: "2026-05-20T10:00:00Z",
      lastReconcileOutcome: "succeeded",
    });
  });
});
