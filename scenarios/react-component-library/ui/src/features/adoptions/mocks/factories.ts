import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  AdoptionSchema,
  LibraryVersionStatus,
  LocalStatus,
  ListAdoptionsResponseSchema,
  RefreshAdoptionsResponseSchema,
  type Adoption,
  type ListAdoptionsResponse,
  type RefreshAdoptionsResponse,
} from "@vrooli/proto-types/react-component-library/v1/adoptions/adoptions_pb";

export type { Adoption, ListAdoptionsResponse, RefreshAdoptionsResponse };
export { LibraryVersionStatus, LocalStatus };

export const makeAdoption = (
  overrides: MessageInitShape<typeof AdoptionSchema> = {},
): Adoption =>
  create(AdoptionSchema, {
    id: "ad-1",
    componentId: "cmp-btn",
    libraryId: "react-component-library:Button",
    scenario: "swarm-manager",
    adoptedPath: "ui/src/components/Button.tsx",
    adoptedVersion: "1.0.0",
    libraryVersionStatus: LibraryVersionStatus.CURRENT,
    localStatus: LocalStatus.CLEAN,
    statusDetail: "",
    createdAt: timestampFromDate(new Date("2026-05-12T00:00:00.000Z")),
    refreshedAt: timestampFromDate(new Date("2026-05-12T01:00:00.000Z")),
    ...overrides,
  });

export const makeListAdoptionsResponse = (
  overrides: MessageInitShape<typeof ListAdoptionsResponseSchema> = {},
): ListAdoptionsResponse =>
  create(ListAdoptionsResponseSchema, {
    adoptions: [],
    ...overrides,
  });

export const makeRefreshAdoptionsResponse = (
  overrides: MessageInitShape<typeof RefreshAdoptionsResponseSchema> = {},
): RefreshAdoptionsResponse =>
  create(RefreshAdoptionsResponseSchema, {
    adoptions: [],
    libraryCurrent: 0,
    libraryBehind: 0,
    libraryDeprecated: 0,
    libraryMissing: 0,
    libraryUnknown: 0,
    localClean: 0,
    localModified: 0,
    localMissing: 0,
    localUnknown: 0,
    ...overrides,
  });
