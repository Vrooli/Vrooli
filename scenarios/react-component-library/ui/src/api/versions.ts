import { createClient } from "@connectrpc/connect";
import {
  VersionsService,
  DiffOp,
  type Version,
  type ListVersionsResponse,
  type GetVersionResponse,
  type DiffVersionsResponse,
  type DiffRow,
  type DiffCell,
} from "@vrooli/proto-types/react-component-library/v1/versions/versions_pb";

import { transport } from "./client";

export const versionsClient = createClient(VersionsService, transport);

export { DiffOp };
export type {
  Version,
  ListVersionsResponse,
  GetVersionResponse,
  DiffVersionsResponse,
  DiffRow,
  DiffCell,
};
