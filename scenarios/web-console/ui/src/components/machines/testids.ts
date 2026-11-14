/**
 * Test ids for the machines surface.
 *
 * `slugify` strips the colon out of a catalog id, turning
 * `bridge-node:node-a` into `bridge-nodenode-a` — unreadable, and ambiguous
 * between two ids that differ only by that separator. Machine and request ids
 * are already opaque and safe, so they are used directly with the separator
 * normalized to a hyphen.
 */
export function machineTestID(id: string): string {
  return id.replace(/[^\w-]+/g, "-");
}
