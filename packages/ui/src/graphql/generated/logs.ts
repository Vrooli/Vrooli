/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { LogSearchInput, LogType } from "./globalTypes";

// ====================================================
// GraphQL query operation: logs
// ====================================================

export interface logs_logs_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface logs_logs_edges_node {
  __typename: "Log";
  id: string;
  timestamp: any;
  action: LogType;
  object1Type: string | null;
  object1Id: string | null;
  object2Type: string | null;
  object2Id: string | null;
  data: string | null;
}

export interface logs_logs_edges {
  __typename: "LogEdge";
  cursor: string;
  node: logs_logs_edges_node;
}

export interface logs_logs {
  __typename: "LogSearchResult";
  pageInfo: logs_logs_pageInfo;
  edges: logs_logs_edges[];
}

export interface logs {
  logs: logs_logs;
}

export interface logsVariables {
  input: LogSearchInput;
}
