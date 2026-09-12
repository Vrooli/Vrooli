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
 */
export const FolderTypeSchema = z.enum(['core', 'local', 'drafts', 'scenario'])
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
