import { createClient } from "@connectrpc/connect";
import {
  ReportService,
  type TupleVerdict,
  type GoldenSummary,
  type GetGoldenSummaryResponse,
  type TupleHistory,
  type GetTupleHistoryResponse,
  type CoverageRow,
  type Coverage,
  type GetCoverageResponse,
} from "@vrooli/proto-types/development-toolchain-validator/v1/report/report_pb";

import { transport } from "./client";

export const reportClient = createClient(ReportService, transport);

export type {
  TupleVerdict,
  GoldenSummary,
  GetGoldenSummaryResponse,
  TupleHistory,
  GetTupleHistoryResponse,
  CoverageRow,
  Coverage,
  GetCoverageResponse,
};
