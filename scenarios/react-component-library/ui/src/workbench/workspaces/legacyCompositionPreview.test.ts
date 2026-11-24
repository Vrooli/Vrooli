import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { createHash } from "node:crypto";
import { toJson } from "@bufbuild/protobuf";
import { RenderCompositionRequestSchema } from "@vrooli/proto-types/react-component-library/v1/preview/preview_pb";
import { describe, expect, it } from "vitest";
import {
  hasLegacyCompositionAdapter,
  legacyCompositionPreviewRequest,
  sourceCompositionPreviewRequest,
} from "./legacyCompositionPreview";

describe("legacy workspace composition preview adapter", () => {
  it.each(["dashboard", "conversations"])(
    "builds the same read-only PreviewService input from the %s native workspace state",
    (page) => {
      const pagePath = resolve(
        process.cwd(),
        `../api/internal/experienceimport/testdata/switchboard/experience/pages/${page}.json`,
      );
      const sourcePage = JSON.parse(readFileSync(pagePath, "utf8")) as {
        sketch: Record<string, unknown>;
      };
      const sourceRevision = createHash("sha256")
        .update(JSON.stringify(canonical(sourcePage)))
        .digest("hex");
      const workspaceArgs = JSON.stringify({
        legacyExperienceComposition: {
          kind: "legacy-render-adapter",
          version: 1,
          sourceRevision,
          sourceSketch: sourcePage.sketch,
        },
      });

      expect(hasLegacyCompositionAdapter(workspaceArgs)).toBe(true);
      const request = legacyCompositionPreviewRequest(workspaceArgs);
      const sourceRequest = sourceCompositionPreviewRequest(sourcePage.sketch, sourceRevision);
      expect(request).not.toBeNull();
      expect(sourceRequest).not.toBeNull();
      expect(toJson(RenderCompositionRequestSchema, request!)).toEqual(
        toJson(RenderCompositionRequestSchema, sourceRequest!),
      );
      expect(request?.composition?.template).toMatchObject({
        catalogId: "templates.collection-page",
        version: "1.7.1",
        export: "CollectionPage",
      });
      expect(request?.composition?.regions).toHaveLength(
        (sourcePage.sketch.render as { regions: unknown[] }).regions.length,
      );
      expect(
        request?.composition?.regions.map((region) => ({ id: region.id, slot: region.slot })),
      ).toEqual(
        (sourcePage.sketch.render as { regions: Array<{ id: string; slot: string[] }> }).regions.map(
          ({ id, slot }) => ({ id, slot }),
        ),
      );
      expect(request?.bindings).toEqual(
        (sourcePage.sketch.render as { bindings: unknown }).bindings,
      );
    },
  );

  it("keeps the unsupported agent-detail composition out of the supported cutover set", () => {
    const pagePath = resolve(
      process.cwd(),
      "../api/internal/experienceimport/testdata/switchboard/experience/pages/agent-detail.json",
    );
    const sourcePage = JSON.parse(readFileSync(pagePath, "utf8")) as {
      sketch: Record<string, unknown>;
    };
    const sourceRevision = createHash("sha256")
      .update(JSON.stringify(canonical(sourcePage)))
      .digest("hex");
    expect(sourceCompositionPreviewRequest(sourcePage.sketch, sourceRevision)).toBeNull();
  });

  it("does not mistake ordinary workspace state for a legacy adapter", () => {
    expect(hasLegacyCompositionAdapter("{}")).toBe(false);
    expect(legacyCompositionPreviewRequest("{}")).toBeNull();
  });
});

function canonical(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(canonical);
  if (value && typeof value === "object") {
    return Object.fromEntries(
      Object.entries(value)
        .sort(([left], [right]) => (left < right ? -1 : left > right ? 1 : 0))
        .map(([key, entry]) => [key, canonical(entry)]),
    );
  }
  return value;
}
