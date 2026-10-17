import { createClient } from "@connectrpc/connect";
import { ClaimsService } from "@vrooli/proto-types/content-desk/v1/claims/claims_pb";
import { transport } from "./client";

export const claimsClient = createClient(ClaimsService, transport);
