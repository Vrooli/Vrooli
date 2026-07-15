import { vi } from "vitest";

import { makeListAdoptionsResponse, makeRefreshAdoptionsResponse, makeSuggestAdoptionsResponse } from "./factories";

export interface AdoptionsMocks {
  adoptionsClient: {
    listAdoptions: ReturnType<typeof vi.fn>;
    applyAdoption: ReturnType<typeof vi.fn>;
    reapplyAdoption: ReturnType<typeof vi.fn>;
    deleteAdoption: ReturnType<typeof vi.fn>;
    refreshAdoptions: ReturnType<typeof vi.fn>;
    resolveAdoptionPath: ReturnType<typeof vi.fn>;
    suggestAdoptions: ReturnType<typeof vi.fn>;
  };
}

export const makeAdoptionsMocks = (): AdoptionsMocks => ({
  adoptionsClient: {
    listAdoptions: vi.fn().mockResolvedValue(makeListAdoptionsResponse()),
    applyAdoption: vi.fn().mockResolvedValue({ adoption: undefined, writtenPath: "" }),
    reapplyAdoption: vi.fn().mockResolvedValue({ adoption: undefined, writtenPath: "" }),
    deleteAdoption: vi.fn().mockResolvedValue({}),
    refreshAdoptions: vi.fn().mockResolvedValue(makeRefreshAdoptionsResponse()),
    resolveAdoptionPath: vi.fn().mockResolvedValue({
      path: "ui/src/components/Button.tsx",
      source: 2,
      slot: "ui-primitive",
      warnings: [],
    }),
    suggestAdoptions: vi.fn().mockResolvedValue(makeSuggestAdoptionsResponse()),
  },
});
