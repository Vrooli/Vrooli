import { createClient } from "@connectrpc/connect";
import {
  DepsService,
  IssueKind,
  VerdictKind,
  type DepDeclaration,
  type DepIssue,
  type ListDeclarationsResponse,
  type ValidateAdoptionResponse,
} from "@vrooli/proto-types/react-component-library/v1/deps/deps_pb";

import { transport } from "./client";

export const depsClient = createClient(DepsService, transport);

export { VerdictKind, IssueKind };
export type { DepDeclaration, DepIssue, ListDeclarationsResponse, ValidateAdoptionResponse };
