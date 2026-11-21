import { createClient } from "@connectrpc/connect";
import { LedgerService } from "@vrooli/proto-types/content-desk/v1/ledger/ledger_pb";
import { transport } from "./client";

export const ledgerClient = createClient(LedgerService, transport);
