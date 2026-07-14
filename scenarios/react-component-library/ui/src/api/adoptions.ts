import { createClient } from "@connectrpc/connect";
import {
  AdoptionsService,
  LibraryVersionStatus,
  LocalStatus,
  ResolveSource,
  type Adoption,
  type AdoptionSuggestion,
  type ListAdoptionsResponse,
  type RefreshAdoptionsResponse,
  type ResolveAdoptionPathRequest,
  type ResolveAdoptionPathResponse,
} from "@vrooli/proto-types/react-component-library/v1/adoptions/adoptions_pb";

import { transport } from "./client";

export const adoptionsClient = createClient(AdoptionsService, transport);

export { LibraryVersionStatus, LocalStatus, ResolveSource };
export type {
  Adoption,
  AdoptionSuggestion,
  ListAdoptionsResponse,
  RefreshAdoptionsResponse,
  ResolveAdoptionPathRequest,
  ResolveAdoptionPathResponse,
};
