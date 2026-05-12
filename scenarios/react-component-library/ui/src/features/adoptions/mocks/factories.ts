import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  AdoptionSchema,
  AdoptionStatus,
  ListAdoptionsResponseSchema,
  RefreshAdoptionsResponseSchema,
  type Adoption,
  type ListAdoptionsResponse,
  type RefreshAdoptionsResponse,
} from "@vrooli/proto-types/react-component-library/v1/adoptions/adoptions_pb";

export type { Adoption, ListAdoptionsResponse, RefreshAdoptionsResponse };
export { AdoptionStatus };

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
    status: AdoptionStatus.CURRENT,
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
    current: 0,
    behind: 0,
    modified: 0,
    unknown: 0,
    ...overrides,
  });
