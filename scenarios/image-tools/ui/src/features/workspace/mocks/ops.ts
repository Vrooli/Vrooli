/**
 * Mock builders for `api/ops` — the UI ↔ API ops boundary. Co-located
 * with the workspace feature; deleting `features/workspace/` takes these with it.
 *
 * Canonical usage (vitest hoists `vi.mock`, so the builder is called from
 * inside the factory closure):
 *
 *   import { makeOpsMocks } from "./mocks/ops";
 *
 *   vi.mock("../../api/ops", async (importOriginal) => {
 *     const actual = await importOriginal<typeof import("../../api/ops")>();
 *     return { ...actual, ...makeOpsMocks() };
 *   });
 *
 * The `...actual` spread keeps re-exported types + the `OP_SPECS`-driven
 * surface intact — only the network-touching functions are substituted.
 *
 * Default behaviors:
 *   - `listOperations` resolves to the canonical operation list
 *   - `runOp` resolves to an image result
 */
import { vi } from "vitest";

import { makeListOperationsResponse, makeRunOpImageResult } from "./factories";

export interface OpsMocks {
  listOperations: ReturnType<typeof vi.fn>;
  runOp: ReturnType<typeof vi.fn>;
}

export const makeOpsMocks = (): OpsMocks => ({
  listOperations: vi.fn().mockResolvedValue(makeListOperationsResponse()),
  runOp: vi.fn().mockResolvedValue(makeRunOpImageResult()),
});
