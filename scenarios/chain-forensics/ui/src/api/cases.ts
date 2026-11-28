import { createClient } from "@connectrpc/connect";
import { CasesService } from "@vrooli/proto-types/chain-forensics/v1/cases/cases_pb";
import { transport } from "./client";

export const casesClient = createClient(CasesService, transport);
