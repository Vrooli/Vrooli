import { vi } from "vitest";

import {
  makeGetComponentContentResponse,
  makeIndexComponentsResponse,
  makeListComponentExamplesResponse,
  makeListComponentsResponse,
  makeUpdateComponentContentResponse,
} from "./factories";

export interface ComponentsMocks {
  componentsClient: {
    listComponents: ReturnType<typeof vi.fn>;
    getComponent: ReturnType<typeof vi.fn>;
    getComponentByLibraryId: ReturnType<typeof vi.fn>;
    indexComponents: ReturnType<typeof vi.fn>;
    getComponentContent: ReturnType<typeof vi.fn>;
    updateComponentContent: ReturnType<typeof vi.fn>;
  };
  listComponentExamples: ReturnType<typeof vi.fn>;
}

export const makeComponentsMocks = (): ComponentsMocks => ({
  componentsClient: {
    listComponents: vi.fn().mockResolvedValue(makeListComponentsResponse()),
    getComponent: vi.fn().mockResolvedValue({ component: undefined }),
    getComponentByLibraryId: vi.fn().mockResolvedValue({ component: undefined }),
    indexComponents: vi.fn().mockResolvedValue(makeIndexComponentsResponse()),
    getComponentContent: vi.fn().mockResolvedValue(makeGetComponentContentResponse()),
    updateComponentContent: vi.fn().mockResolvedValue(makeUpdateComponentContentResponse()),
  },
  listComponentExamples: vi.fn().mockResolvedValue(makeListComponentExamplesResponse()),
});
