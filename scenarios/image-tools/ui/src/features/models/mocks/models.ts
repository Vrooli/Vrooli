/**
 * Mock builders for `api/models` — the UI ↔ API models boundary.
 * Co-located with the models feature; deleting `features/models/` takes
 * these with it.
 *
 * See `test-utils/mocks/api.ts` for the full builder/hoisting rationale.
 * Canonical usage:
 *
 *   import { makeModelsMocks } from "./mocks/models";
 *
 *   vi.mock("../../api/models", async (importOriginal) => {
 *     const actual = await importOriginal<typeof import("../../api/models")>();
 *     return { ...actual, ...makeModelsMocks() };
 *   });
 *
 * Default behaviors:
 *   - `modelsClient.listModels` resolves to an empty list
 *   - `modelsClient.listOperations` resolves to an empty list
 *   - `modelsClient.setModelEnabled({ id, enabled })` echoes the change back
 */
import { vi } from "vitest";

import {
  makeListModelsResponse,
  makeListOperationsResponse,
  makeModel,
  makeSetModelEnabledResponse,
} from "./factories";

export interface ModelsMocks {
  modelsClient: {
    listModels: ReturnType<typeof vi.fn>;
    listOperations: ReturnType<typeof vi.fn>;
    setModelEnabled: ReturnType<typeof vi.fn>;
  };
}

export const makeModelsMocks = (): ModelsMocks => ({
  modelsClient: {
    listModels: vi.fn().mockResolvedValue(makeListModelsResponse()),
    listOperations: vi.fn().mockResolvedValue(makeListOperationsResponse()),
    setModelEnabled: vi
      .fn()
      .mockImplementation((input: { id: string; enabled: boolean }) =>
        Promise.resolve(
          makeSetModelEnabledResponse({
            model: makeModel({ id: input.id, enabled: input.enabled }),
          }),
        ),
      ),
  },
});
