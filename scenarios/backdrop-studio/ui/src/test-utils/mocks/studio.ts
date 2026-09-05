import { vi } from "vitest";

import { QualityTier } from "@vrooli/proto-types/backdrop-studio/v1/shared/shared_pb";

import type { Candidate, Style, Surface } from "../../api/studio";

/**
 * Mock builders for `./api/studio` — the UI ↔ catalog/render boundary.
 *
 * Same hoisting constraint as the health mocks: call `makeStudioMocks()` from
 * *inside* a `vi.mock` factory closure, never at module top level.
 *
 *   vi.mock("../api/studio", async (importOriginal) => {
 *     const actual = await importOriginal<typeof import("../api/studio")>();
 *     return { ...actual, ...makeStudioMocks() };
 *   });
 *
 * The `...actual` spread matters more here than usual: the module exports pure
 * helpers (`permittedSurfaces`, `parseQuality`, `isModelBacked`) alongside its
 * network calls, and stubbing those would test the stub rather than the rule.
 */

/** A procedural style with one overlay region — the common case. */
export function makeStyle(overrides: Partial<Style> = {}): Style {
  return {
    $typeName: "vrooli.backdrop_studio.v1.shared.Style",
    id: "cyanotype-arcade",
    name: "Cyanotype Arcade",
    version: 1,
    role: "ambient",
    subject: "statuary_architecture",
    treatments: ["duotone", "halftone"],
    lineage: "cyanotype",
    placements: ["full_bleed", "split_panel"],
    strategy: "procedural-treated",
    regions: [],
    contrastThreshold: 4.5,
    treatmentParams: { halftone: '{"lpi":72}' },
    inks: { "$brand.primary": "#1b3fbf" },
    parentId: "",
    // Procedural is the honest default for a mock of the common case: it is
    // what every style seeded before the tier existed actually was.
    qualityTier: QualityTier.PROCEDURAL,
    // No plate spec: the common case is a style that ships one plate carrying
    // the whole picture, which is what every style drew before the plate model
    // existed and what forty of the forty-four still draw.
    plateSpec: [],
    ...overrides,
  };
}

export function makeSurface(overrides: Partial<Surface> = {}): Surface {
  return {
    $typeName: "vrooli.backdrop_studio.v1.surfaces.Surface",
    id: "web.hero",
    name: "Landing page hero",
    kind: "product",
    width: 1440,
    height: 720,
    placements: ["full_bleed", "split_panel"],
    authority: "Backdrop Studio product geometry",
    confirmedOn: "2026-08-12",
    ...overrides,
  };
}

export function makeCandidate(overrides: Partial<Candidate> = {}): Candidate {
  return {
    $typeName: "vrooli.backdrop_studio.v1.render.Candidate",
    id: "candidate-1",
    jobId: "job-1",
    // A one-pixel PNG: the tests assert that an image element appears with the
    // right alt text, never what the pixels are.
    imagePng: new Uint8Array([137, 80, 78, 71, 13, 10, 26, 10]),
    width: 1440,
    height: 720,
    seed: 7n,
    strategy: "procedural-treated",
    executionPath: "procedural",
    treatmentApplied: true,
    qualityJson: JSON.stringify({
      passed: true,
      metrics: { subject_survival: 0.91, tonal_occupancy: 0.72 },
      thresholds: { subject_survival: 0.6, tonal_occupancy: 0.4 },
    }),
    provenanceJson: JSON.stringify({ style_id: "cyanotype-arcade", seed: 7 }),
    ...overrides,
  } as Candidate;
}

export interface StudioMocks {
  listStyles: ReturnType<typeof vi.fn>;
  listSurfaces: ReturnType<typeof vi.fn>;
  submitRender: ReturnType<typeof vi.fn>;
  createStyle: ReturnType<typeof vi.fn>;
  candidateImageURL: ReturnType<typeof vi.fn>;
}

export function makeStudioMocks(): StudioMocks {
  return {
    listStyles: vi.fn(() => Promise.resolve([makeStyle()])),
    listSurfaces: vi.fn(() => Promise.resolve([makeSurface()])),
    submitRender: vi.fn(() => Promise.resolve({
      $typeName: "vrooli.backdrop_studio.v1.render.RenderJob",
      id: "job-1",
      styleId: "cyanotype-arcade",
      status: "completed",
      seed: 7n,
      executionPath: "procedural",
      candidates: [makeCandidate()],
      selectedCandidateId: "",
      selectedBy: "",
      surfaceId: "web.hero",
    })),
    createStyle: vi.fn((style: Style) => Promise.resolve(style)),
    // jsdom has no createObjectURL; the pages only ever hand the result to an
    // <img src>, so a stable stub is enough and keeps the assertions about
    // structure rather than about blob plumbing.
    candidateImageURL: vi.fn(() => "blob:candidate"),
  };
}
