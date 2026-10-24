import { z } from 'zod'

const digest = z.string().regex(/^[0-9a-f]{64}$/)
export const BuildProvenanceSchema = z.object({
  algorithm: z.literal('sha256'),
  sourceScope: z.literal('ui/src runtime files, vite.config.ts, scripts/world-build-provenance.ts'),
  sourceSha256: digest,
  dependencyScope: z.literal('ui/package.json and ui/pnpm-lock.yaml'),
  dependencySha256: digest,
  assetSha256: digest,
  assets: z.array(z.object({
    path: z.string().max(512).regex(/^[a-zA-Z0-9_-]+(?:\/[a-zA-Z0-9_.-]+)*\.[a-zA-Z0-9]+$/),
    bytes: z.number().int().nonnegative(), sha256: digest,
  }).strict()).max(10000),
}).strict()

declare const __WORLD_BUILD_PROVENANCE__: unknown

/** Test-only/non-Vite consumers report unavailable provenance explicitly. */
export function readBuildProvenance() {
  return typeof __WORLD_BUILD_PROVENANCE__ === 'undefined' ? null : BuildProvenanceSchema.parse(__WORLD_BUILD_PROVENANCE__)
}
