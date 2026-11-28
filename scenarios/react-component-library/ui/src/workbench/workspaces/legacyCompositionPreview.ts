import { createClient } from "@connectrpc/connect";
import {
  PreviewService,
  RenderCompositionRequestSchema,
  type RenderCompositionRequest,
} from "@vrooli/proto-types/react-component-library/v1/preview/preview_pb";
import { fromJson, type JsonValue, transport } from "../../api/client";

type RecordValue = Record<string, unknown>;

function record(value: unknown): RecordValue | null {
  return value && typeof value === "object" && !Array.isArray(value)
    ? (value as RecordValue)
    : null;
}

function stringValue(value: unknown): string {
  return typeof value === "string" ? value : "";
}

/**
 * Build a read-only PreviewService request from the state adapter. This keeps
 * the legacy renderer as a compatibility preview while the editable canvas
 * remains a native Workspace TemplateNode tree.
 */
export function legacyCompositionPreviewRequest(argsJson: string): RenderCompositionRequest | null {
  let args: unknown;
  try {
    args = JSON.parse(argsJson);
  } catch {
    return null;
  }
  const envelope = record(record(args)?.legacyExperienceComposition);
  if (envelope?.kind !== "legacy-render-adapter" || envelope.version !== 1) return null;
  if (typeof envelope.sourceRevision !== "string" || envelope.sourceRevision === "") return null;
  return sourceCompositionPreviewRequest(envelope.sourceSketch, envelope.sourceRevision);
}

/** Translate the source contract shape to the shared renderer input. */
export function sourceCompositionPreviewRequest(
  sourceSketch: unknown,
  sourceRevision: string,
): RenderCompositionRequest | null {
  const sketch = record(sourceSketch);
  const template = record(sketch?.template);
  const render = record(sketch?.render);
  if (
    !sketch ||
    typeof template?.asset !== "string" ||
    typeof render?.templateExport !== "string" ||
    !Array.isArray(render.regions)
  ) {
    return null;
  }

  const placements = new Map<string, RecordValue>();
  for (const raw of Array.isArray(sketch.placements) ? sketch.placements : []) {
    const placement = record(raw);
    const fills = record(placement?.fills);
    if (typeof placement?.region === "string" && fills) {
      if (placements.has(placement.region)) return null;
      placements.set(placement.region, fills);
    }
  }
  const regions = render.regions.map((raw) => {
    const region = record(raw);
    if (
      !region ||
      typeof region.id !== "string" ||
      !Array.isArray(region.slot) ||
      !region.slot.every((part) => typeof part === "string")
    ) {
      throw new Error("legacy composition has an invalid render region");
    }
    const placement = placements.get(region.id);
    const asset =
      placement && typeof placement.asset === "string"
        ? {
            catalogId: placement.asset,
            version: typeof placement.version === "string" ? placement.version : "",
            export: typeof region.export === "string" ? region.export : "",
          }
        : undefined;
    return {
      id: region.id,
      parent: typeof region.parent === "string" ? region.parent : "",
      slot: region.slot,
      required: region.optional !== true,
      ...(asset ? { asset } : {}),
    };
  });
  const fixtures = (Array.isArray(render.fixtures) ? render.fixtures : []).map((raw) => {
    const fixture = record(raw);
    if (!fixture) throw new Error("legacy composition has an invalid fixture binding");
    return {
      target: stringValue(fixture.target),
      asset: stringValue(fixture.asset),
      version: stringValue(fixture.version),
      state: stringValue(fixture.state),
      field: stringValue(fixture.field),
      prop: Array.isArray(fixture.prop) ? fixture.prop.map(stringValue) : [],
    };
  });

  return fromJson(RenderCompositionRequestSchema, {
    composition: {
      revision: sourceRevision,
      template: {
        catalogId: template.asset,
        version: typeof template.version === "string" ? template.version : "",
        export: render.templateExport,
      },
      regions,
    },
    bindings: record(render.bindings) ?? {},
    fixtures,
    kit: "vrooli-default",
    theme: "light",
    direction: "ltr",
  } as JsonValue);
}

export function hasLegacyCompositionAdapter(argsJson: string): boolean {
  try {
    const args = record(JSON.parse(argsJson));
    const adapter = record(args?.legacyExperienceComposition);
    return adapter?.kind === "legacy-render-adapter" && adapter.version === 1;
  } catch {
    return false;
  }
}

const previewClient = createClient(PreviewService, transport);

export async function renderLegacyComposition(argsJson: string): Promise<string> {
  const request = legacyCompositionPreviewRequest(argsJson);
  if (!request) throw new Error("This workspace state has no supported legacy composition adapter");
  const response = await previewClient.renderComposition(request);
  return response.html;
}
