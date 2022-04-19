/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { LogSearchInput, LogType } from "./globalTypes";

// ====================================================
// GraphQL query operation: logs
// ====================================================

export interface logs_logs {
  __typename: "LogSearchResult";
}

export interface logs {
  logs: logs_logs;
}

export interface logsVariables {
  input: LogSearchInput;
}
