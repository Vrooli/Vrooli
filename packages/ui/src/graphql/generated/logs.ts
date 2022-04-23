/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { LogSearchInput, MemberRole, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

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
