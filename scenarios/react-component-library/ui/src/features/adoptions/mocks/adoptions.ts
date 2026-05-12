import { vi } from "vitest";

import { makeListAdoptionsResponse, makeRefreshAdoptionsResponse } from "./factories";

export interface AdoptionsMocks {
  adoptionsClient: {
    listAdoptions: ReturnType<typeof vi.fn>;
    createAdoption: ReturnType<typeof vi.fn>;
    deleteAdoption: ReturnType<typeof vi.fn>;
    refreshAdoptions: ReturnType<typeof vi.fn>;
  };
}

export const makeAdoptionsMocks = (): AdoptionsMocks => ({
  adoptionsClient: {
    listAdoptions: vi.fn().mockResolvedValue(makeListAdoptionsResponse()),
    createAdoption: vi.fn().mockResolvedValue({ adoption: undefined }),
    deleteAdoption: vi.fn().mockResolvedValue({}),
    refreshAdoptions: vi.fn().mockResolvedValue(makeRefreshAdoptionsResponse()),
  },
});
