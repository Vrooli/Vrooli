import { createClient } from "@connectrpc/connect";
import { CatalogService } from "@vrooli/proto-types/backdrop-studio/v1/catalog/catalog_pb";
import { RenderService } from "@vrooli/proto-types/backdrop-studio/v1/render/render_pb";
import { SurfacesService } from "@vrooli/proto-types/backdrop-studio/v1/surfaces/surfaces_pb";
import type { Style } from "@vrooli/proto-types/backdrop-studio/v1/shared/shared_pb";
import type { Candidate, RenderJob } from "@vrooli/proto-types/backdrop-studio/v1/render/render_pb";
import type { Surface } from "@vrooli/proto-types/backdrop-studio/v1/surfaces/surfaces_pb";

import { transport } from "./client";

/**
 * The studio's data layer.
 *
 * Every page below this line reads the real catalog. That is the point of the
 * file: the workbench it replaces held four hardcoded style rows and rendered
 * its "specimens" as CSS gradients, so the one question the studio exists to
 * answer — *what does this style actually look like?* — could not be asked of
 * it. A gradient is not evidence.
 *
 * The clients are Connect clients over the generated service descriptors, so
 * the wire shape is the proto and nothing here can drift from it silently.
 */
const catalog = createClient(CatalogService, transport);
const render = createClient(RenderService, transport);
const surfaces = createClient(SurfacesService, transport);

export type { Style, Candidate, RenderJob, Surface };

/** Axis filters the catalog service accepts. Every field is optional. */
export type StyleFilter = {
  role?: string;
  subject?: string;
  treatment?: string;
  lineage?: string;
  placement?: string;
};

export async function listStyles(filter: StyleFilter = {}): Promise<Style[]> {
  const res = await catalog.listStyles({
    role: filter.role ?? "",
    subject: filter.subject ?? "",
    treatment: filter.treatment ?? "",
    lineage: filter.lineage ?? "",
    placement: filter.placement ?? "",
  });
  return res.styles;
}

/**
 * createStyle writes an operator-authored style.
 *
 * The catalog validates it at write time — axis values, the treatment chain
 * against image-tools' wire contract, ink slots, region geometry — so a style
 * the engine would reject is refused here rather than at its first render. The
 * error message is the server's own, because that message is the finding.
 */
export async function createStyle(style: Style): Promise<Style> {
  return catalog.createStyle({ style });
}

export async function listSurfaces(): Promise<Surface[]> {
  const res = await surfaces.listSurfaces({});
  return res.surfaces;
}

/** One render submission. Seed is a string on the wire because it is int64. */
export type RenderRequest = {
  styleId: string;
  surfaceId: string;
  placement?: string;
  seed: bigint;
  candidateCount?: number;
  brandTokens?: Record<string, string>;
};

export async function submitRender(req: RenderRequest): Promise<RenderJob> {
  const job = await render.submit({
    style: { id: req.styleId },
    surfaceId: req.surfaceId,
    placement: req.placement ?? "",
    seed: req.seed,
    candidateCount: req.candidateCount ?? 1,
    brandTokens: req.brandTokens ?? {},
  });
  if (!job.id) {
    throw new Error("render returned no job");
  }
  return job;
}

/**
 * candidateImageURL turns a candidate's PNG bytes into something an <img> can
 * display.
 *
 * Object URLs rather than data URIs: a delivery-resolution PNG is a megabyte or
 * more, and base64 inflates it by a third before it ever reaches the DOM. The
 * caller owns revocation — see `useObjectURL`.
 */
export function candidateImageURL(candidate: Candidate): string {
  const blob = new Blob([candidate.imagePng as unknown as BlobPart], { type: "image/png" });
  return URL.createObjectURL(blob);
}

/**
 * The perceptual verdict a candidate carries, as the render path recorded it.
 *
 * It is parsed rather than typed because the render service serialises it as a
 * JSON string: the metric set is expected to grow, and a proto message per
 * metric would make every new measurement a wire change. The shape is asserted
 * here so a page can rely on it.
 */
export type QualityVerdict = {
  passed: boolean;
  metrics: Record<string, number>;
  thresholds: Record<string, number>;
  failures?: string[];
};

export function parseQuality(candidate: Candidate): QualityVerdict | null {
  if (!candidate.qualityJson) {
    return null;
  }
  try {
    const parsed = JSON.parse(candidate.qualityJson) as Partial<QualityVerdict>;
    return {
      passed: Boolean(parsed.passed),
      metrics: parsed.metrics ?? {},
      thresholds: parsed.thresholds ?? {},
      failures: parsed.failures ?? [],
    };
  } catch {
    return null;
  }
}

/** The provenance record a candidate carries, parsed for display. */
export function parseProvenance(candidate: Candidate): Record<string, unknown> | null {
  if (!candidate.provenanceJson) {
    return null;
  }
  try {
    return JSON.parse(candidate.provenanceJson) as Record<string, unknown>;
  } catch {
    return null;
  }
}

/**
 * Whether a style needs an image model to render at all.
 *
 * The distinction is shown wherever a style is: on a local-first product the
 * unavailable case is common rather than exceptional, and an operator who picks
 * a metered style on a host with no model should learn that from the tile
 * rather than from a failed render.
 */
export function isModelBacked(style: Style): boolean {
  return style.strategy === "guided" || style.strategy === "synthesized";
}

/** Surfaces whose permitted placements intersect the style's. */
export function permittedSurfaces(style: Style, all: Surface[]): Surface[] {
  return all
    .filter((surface) => surface.placements.some((p) => style.placements.includes(p)))
    .sort((a, b) => a.width * a.height - b.width * b.height);
}

/**
 * The surface a style is best shown at: the landing-page hero when it permits
 * it, the largest permitted surface otherwise.
 *
 * Without the preference every hero style previews in 4:5 portrait, because a
 * social post card is taller than a hero is wide — so a colonnade meant to run
 * across a header gets judged cropped to a phone.
 */
export function representativeSurface(style: Style, all: Surface[]): Surface | undefined {
  const permitted = permittedSurfaces(style, all);
  return permitted.find((s) => s.id === "web.hero") ?? permitted[permitted.length - 1];
}
