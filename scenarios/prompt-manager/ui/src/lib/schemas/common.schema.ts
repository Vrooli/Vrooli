/**
 * Common schemas shared across multiple domains.
 */

import { z } from 'zod'

/**
 * Folder type enum for skill organization.
 * - core: Important skills (git-tracked)
 * - local: Personal skills (gitignored)
 * - drafts: Work in progress skills
 * - scenario: Skills owned by a scenario and read from its own skills/ root
 * - vendor: Third-party skills imported from an external source (gitignored, read-only)
 */
export const FolderTypeSchema = z.enum(['core', 'local', 'drafts', 'scenario', 'vendor'])
export type FolderType = z.infer<typeof FolderTypeSchema>

/**
 * Hex color validation pattern.
 * Matches #RRGGBB format (case insensitive).
 */
export const HexColorSchema = z.string().regex(/^#[0-9A-Fa-f]{6}$/, {
  message: 'Must be a valid hex color (e.g., #FF5733)',
})

/**
 * Kebab-case identifier validation.
 * Matches lowercase alphanumeric strings with hyphens.
 */
export const KebabCaseIdSchema = z.string().regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/, {
  message: 'Must be kebab-case (e.g., my-skill-id)',
})

/**
 * ISO timestamp string schema.
 * Validates that the string is a valid date format.
 */
export const TimestampSchema = z.string().refine(
  (val) => !isNaN(Date.parse(val)),
  { message: 'Must be a valid ISO date string' }
)

/**
 * Proto JSON wire schema for 64-bit integers (int64/uint64).
 *
 * `toJson` from @bufbuild/protobuf encodes 64-bit integers as decimal strings
 * to preserve precision, so a Go `int64` field arrives as `"689"` rather than
 * `689`. Accept either the string wire form or a number and normalize to a
 * JavaScript number for domain consumers.
 */
export const ProtoInt64Schema = z
  .union([
    z.number(),
    z
      .string()
      .regex(/^-?\d+$/, { message: 'Must be an integer or a decimal integer string' }),
  ])
  .transform((value) => (typeof value === 'string' ? Number(value) : value))

/**
 * Proto JSON wire schema for a proto3 bool that defaults to `false`.
 *
 * Proto3 JSON omits fields set to their default value, so a `false` boolean
 * field such as `is_dir` is absent from the payload rather than sent as
 * `false`. Normalize the absence to `false`.
 */
export const ProtoBoolDefaultFalseSchema = z.boolean().optional().default(false)
