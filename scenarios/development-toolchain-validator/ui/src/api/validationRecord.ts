import { createClient } from "@connectrpc/connect";
import {
  ValidationRecordService,
  TupleKind,
  Verdict,
  type ValidationRecord,
  type ListRecordsResponse,
  type GetRecordResponse,
} from "@vrooli/proto-types/development-toolchain-validator/v1/validation_record/validation_record_pb";

import { transport } from "./client";

export const validationRecordClient = createClient(ValidationRecordService, transport);

export { TupleKind, Verdict };
export type { ValidationRecord, ListRecordsResponse, GetRecordResponse };
