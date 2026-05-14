import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  DiffCellSchema,
  DiffOp,
  DiffRowSchema,
  DiffVersionsResponseSchema,
  ListVersionsResponseSchema,
  VersionSchema,
  type DiffCell,
  type DiffRow,
  type DiffVersionsResponse,
  type ListVersionsResponse,
  type Version,
} from "@vrooli/proto-types/react-component-library/v1/versions/versions_pb";

export type { DiffCell, DiffRow, DiffVersionsResponse, ListVersionsResponse, Version };
export { DiffOp };

export const makeVersion = (
  overrides: MessageInitShape<typeof VersionSchema> = {},
): Version =>
  create(VersionSchema, {
    id: "ver-1",
    componentId: "cmp-btn",
    version: "1.0.0",
    contentSha256: "abc123def456",
    changelogMd: "auto-recorded on save",
    recordedAt: timestampFromDate(new Date("2026-05-13T00:00:00.000Z")),
    ...overrides,
  });

export const makeListVersionsResponse = (
  overrides: MessageInitShape<typeof ListVersionsResponseSchema> = {},
): ListVersionsResponse =>
  create(ListVersionsResponseSchema, {
    versions: [],
    ...overrides,
  });

export const makeDiffCell = (
  overrides: MessageInitShape<typeof DiffCellSchema> = {},
): DiffCell => create(DiffCellSchema, { lineNumber: 0, text: "", op: DiffOp.EMPTY, ...overrides });

export const makeDiffRow = (
  left: MessageInitShape<typeof DiffCellSchema>,
  right: MessageInitShape<typeof DiffCellSchema>,
): DiffRow =>
  create(DiffRowSchema, {
    left: makeDiffCell(left),
    right: makeDiffCell(right),
  });

export const makeDiffVersionsResponse = (
  overrides: MessageInitShape<typeof DiffVersionsResponseSchema> = {},
): DiffVersionsResponse =>
  create(DiffVersionsResponseSchema, {
    rows: [],
    additions: 0,
    removals: 0,
    fromLabel: "",
    toLabel: "",
    ...overrides,
  });
