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
 * The `...actual` spread keeps the re-exported proto types + enums (e.g.
 * CommercialUse) intact — only the network-touching client methods are
 * substituted.
 */
import { vi } from "vitest";

import {
  makeAddCustomModelResponse,
  makeDoctorBackendsResponse,
  makeEnsureBackendResponse,
  makeImportModelResponse,
  makeInspectModelSourceResponse,
  makeInstallModelResponse,
  makeListBlocklistResponse,
  makeListDefaultsResponse,
  makeListModelsResponse,
  makeHostSummary,
  makeListOperationModelsResponse,
  makeListOperationsResponse,
  makeModel,
  makeRemoveModelResponse,
  makeExplainResolutionResponse,
  makeSelectModelResponse,
  makeSetDefaultModelResponse,
  makeSetModelEnabledResponse,
} from "./factories";

export interface ModelsMocks {
  modelsClient: {
    listModels: ReturnType<typeof vi.fn>;
    listOperations: ReturnType<typeof vi.fn>;
    setModelEnabled: ReturnType<typeof vi.fn>;
    installModel: ReturnType<typeof vi.fn>;
    removeModel: ReturnType<typeof vi.fn>;
    addCustomModel: ReturnType<typeof vi.fn>;
    inspectModelSource: ReturnType<typeof vi.fn>;
    importModel: ReturnType<typeof vi.fn>;
    setDefaultModel: ReturnType<typeof vi.fn>;
    listDefaults: ReturnType<typeof vi.fn>;
    listBlocklist: ReturnType<typeof vi.fn>;
    doctorBackends: ReturnType<typeof vi.fn>;
    ensureBackend: ReturnType<typeof vi.fn>;
    selectModel: ReturnType<typeof vi.fn>;
    explainResolution: ReturnType<typeof vi.fn>;
    listOperationModels: ReturnType<typeof vi.fn>;
    getHostSummary: ReturnType<typeof vi.fn>;
  };
  /** Standalone `listOperationModels(operation)` helper re-exported by api/models. */
  listOperationModels: ReturnType<typeof vi.fn>;
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
    installModel: vi.fn().mockResolvedValue(makeInstallModelResponse()),
    removeModel: vi.fn().mockResolvedValue(makeRemoveModelResponse()),
    addCustomModel: vi.fn().mockResolvedValue(makeAddCustomModelResponse()),
    inspectModelSource: vi.fn().mockResolvedValue(makeInspectModelSourceResponse()),
    importModel: vi.fn().mockResolvedValue(makeImportModelResponse()),
    setDefaultModel: vi
      .fn()
      .mockImplementation((input: { operation: string; modelId: string }) =>
        Promise.resolve(makeSetDefaultModelResponse(input)),
      ),
    listDefaults: vi.fn().mockResolvedValue(makeListDefaultsResponse()),
    listBlocklist: vi.fn().mockResolvedValue(makeListBlocklistResponse()),
    doctorBackends: vi.fn().mockResolvedValue(makeDoctorBackendsResponse()),
    ensureBackend: vi
      .fn()
      .mockImplementation((input: { tool: string }) =>
        Promise.resolve(makeEnsureBackendResponse({ tool: input.tool })),
      ),
    selectModel: vi.fn().mockResolvedValue(makeSelectModelResponse()),
    explainResolution: vi.fn().mockResolvedValue(makeExplainResolutionResponse()),
    listOperationModels: vi.fn().mockResolvedValue(makeListOperationModelsResponse()),
    getHostSummary: vi.fn().mockResolvedValue({ host: makeHostSummary() }),
  },
  listOperationModels: vi.fn().mockResolvedValue(makeListOperationModelsResponse()),
});
