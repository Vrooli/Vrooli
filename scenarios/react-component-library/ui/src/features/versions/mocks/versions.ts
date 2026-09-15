import { vi } from "vitest";

import { makeDiffVersionsResponse, makeListVersionsResponse } from "./factories";

export interface VersionsMocks {
  versionsClient: {
    listVersions: ReturnType<typeof vi.fn>;
    getVersion: ReturnType<typeof vi.fn>;
    diffVersions: ReturnType<typeof vi.fn>;
  };
}

export const makeVersionsMocks = (): VersionsMocks => ({
  versionsClient: {
    listVersions: vi.fn().mockResolvedValue(makeListVersionsResponse()),
    getVersion: vi.fn().mockResolvedValue({ version: undefined, content: "" }),
    diffVersions: vi.fn().mockResolvedValue(makeDiffVersionsResponse()),
  },
});
