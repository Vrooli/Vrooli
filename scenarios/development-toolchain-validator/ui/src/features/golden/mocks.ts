import { vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  GoldenSchema,
  ListGoldensResponseSchema,
  GetGoldenResponseSchema,
  RegisterGoldenResponseSchema,
  UpdateGoldenResponseSchema,
  DeleteGoldenResponseSchema,
  RegenerateGoldenResponseSchema,
  type Golden,
} from "@vrooli/proto-types/development-toolchain-validator/v1/golden/golden_pb";

const baseTimestamp = timestampFromDate(new Date("2026-01-01T00:00:00Z"));

export const makeGolden = (overrides: Partial<Golden> = {}): Golden =>
  create(GoldenSchema, {
    id: "id-alpha",
    slug: "alpha",
    templateId: "react-vite",
    templateVersionPinned: "1.0.1",
    path: "scenarios/alpha",
    createdAt: baseTimestamp,
    lastRegeneratedAt: baseTimestamp,
    ...overrides,
  });

export interface GoldenMocks {
  goldenClient: {
    listGoldens: ReturnType<typeof vi.fn>;
    getGolden: ReturnType<typeof vi.fn>;
    registerGolden: ReturnType<typeof vi.fn>;
    updateGolden: ReturnType<typeof vi.fn>;
    deleteGolden: ReturnType<typeof vi.fn>;
    regenerateGolden: ReturnType<typeof vi.fn>;
  };
}

export const makeGoldenMocks = (): GoldenMocks => ({
  goldenClient: {
    listGoldens: vi.fn().mockResolvedValue(create(ListGoldensResponseSchema, {})),
    getGolden: vi.fn().mockImplementation((input: { slug: string }) =>
      Promise.resolve(create(GetGoldenResponseSchema, { golden: makeGolden({ slug: input.slug }) })),
    ),
    registerGolden: vi.fn().mockImplementation((input: { slug: string; templateId: string; templateVersion: string; path: string }) =>
      Promise.resolve(
        create(RegisterGoldenResponseSchema, {
          golden: makeGolden({
            slug: input.slug,
            templateId: input.templateId,
            templateVersionPinned: input.templateVersion,
            path: input.path,
          }),
        }),
      ),
    ),
    updateGolden: vi.fn().mockImplementation((input: { slug: string }) =>
      Promise.resolve(create(UpdateGoldenResponseSchema, { golden: makeGolden({ slug: input.slug }) })),
    ),
    deleteGolden: vi.fn().mockResolvedValue(create(DeleteGoldenResponseSchema, {})),
    regenerateGolden: vi.fn().mockImplementation((input: { slug: string }) =>
      Promise.resolve(create(RegenerateGoldenResponseSchema, { golden: makeGolden({ slug: input.slug }) })),
    ),
  },
});
