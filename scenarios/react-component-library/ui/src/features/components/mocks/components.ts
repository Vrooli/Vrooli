import { vi } from "vitest";

import {
  makeGetComponentContentResponse,
  makeGetComponentVersionContentResponse,
  makeIndexComponentsResponse,
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
    getComponentVersionContent: ReturnType<typeof vi.fn>;
    listComponentVersions: ReturnType<typeof vi.fn>;
    updateComponentContent: ReturnType<typeof vi.fn>;
  };
	listComponentStories: ReturnType<typeof vi.fn>;
}

export const makeComponentsMocks = (): ComponentsMocks => ({
  componentsClient: {
    listComponents: vi.fn().mockResolvedValue(makeListComponentsResponse()),
    getComponent: vi.fn().mockResolvedValue({ component: undefined }),
    getComponentByLibraryId: vi.fn().mockResolvedValue({ component: undefined }),
    indexComponents: vi.fn().mockResolvedValue(makeIndexComponentsResponse()),
    getComponentContent: vi.fn().mockResolvedValue(makeGetComponentContentResponse()),
    getComponentVersionContent: vi.fn().mockResolvedValue(makeGetComponentVersionContentResponse()),
    listComponentVersions: vi.fn().mockResolvedValue({ versions: [{ version: "1.0.0", files: [] }] }),
    updateComponentContent: vi.fn().mockResolvedValue(makeUpdateComponentContentResponse()),
  },
	listComponentStories: vi.fn().mockResolvedValue({ stories: [] }),
});
