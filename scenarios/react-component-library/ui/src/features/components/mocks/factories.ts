import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  ComponentSchema,
  ListComponentsResponseSchema,
  IndexComponentsResponseSchema,
  GetComponentContentResponseSchema,
  GetComponentVersionContentResponseSchema,
  UpdateComponentContentResponseSchema,
  type Component,
  type ListComponentsResponse,
  type IndexComponentsResponse,
  type GetComponentContentResponse,
  type GetComponentVersionContentResponse,
  type UpdateComponentContentResponse,
} from "@vrooli/proto-types/react-component-library/v1/components/components_pb";

export type {
  Component,
  ListComponentsResponse,
  IndexComponentsResponse,
  GetComponentContentResponse,
  GetComponentVersionContentResponse,
  UpdateComponentContentResponse,
};

export const makeComponent = (
  overrides: MessageInitShape<typeof ComponentSchema> = {},
): Component => {
  const component = create(ComponentSchema, {
    id: "cmp-1",
    libraryId: "react-component-library:Button",
    displayName: "Button",
    description: "Primary CTA.",
    slot: "ui-primitive",
    sourcePath: "components/Button.tsx",
    version: "1.0.0",
    tags: ["form"],
    indexedAt: timestampFromDate(new Date("2026-05-12T00:00:00.000Z")),
    updatedAt: timestampFromDate(new Date("2026-05-12T00:00:00.000Z")),
  });
  return Object.assign(component, overrides);
};

export const makeListComponentsResponse = (
  overrides: MessageInitShape<typeof ListComponentsResponseSchema> = {},
): ListComponentsResponse =>
  create(ListComponentsResponseSchema, {
    components: [],
    ...overrides,
  });

export const makeGetComponentContentResponse = (
  overrides: MessageInitShape<typeof GetComponentContentResponseSchema> = {},
): GetComponentContentResponse =>
  create(GetComponentContentResponseSchema, {
    content: "export const Button = () => null;\n",
    sourcePath: "components/Button.tsx",
    sha256: "abc123def456",
    ...overrides,
  });

export const makeGetComponentVersionContentResponse = (
  overrides: MessageInitShape<typeof GetComponentVersionContentResponseSchema> = {},
): GetComponentVersionContentResponse =>
  create(GetComponentVersionContentResponseSchema, {
    content: "export const ButtonV1 = () => null;\n",
    ...overrides,
  });

export const makeUpdateComponentContentResponse = (
  overrides: MessageInitShape<typeof UpdateComponentContentResponseSchema> = {},
): UpdateComponentContentResponse =>
  create(UpdateComponentContentResponseSchema, {
    sha256: "newsha789",
    sourcePath: "components/Button.tsx",
    ...overrides,
  });

export const makeIndexComponentsResponse = (
  overrides: MessageInitShape<typeof IndexComponentsResponseSchema> = {},
): IndexComponentsResponse =>
  create(IndexComponentsResponseSchema, {
    scanned: 0,
    indexed: 0,
    skipped: 0,
    deleted: 0,
    libraryIds: [],
    errors: [],
    ...overrides,
  });
