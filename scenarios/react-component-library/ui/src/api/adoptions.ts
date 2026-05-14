import { createClient } from "@connectrpc/connect";
import {
  AdoptionsService,
  LibraryVersionStatus,
  LocalStatus,
  type Adoption,
  type ListAdoptionsResponse,
  type RefreshAdoptionsResponse,
} from "@vrooli/proto-types/react-component-library/v1/adoptions/adoptions_pb";

import { transport } from "./client";

export const adoptionsClient = createClient(AdoptionsService, transport);

export { LibraryVersionStatus, LocalStatus };
export type { Adoption, ListAdoptionsResponse, RefreshAdoptionsResponse };
