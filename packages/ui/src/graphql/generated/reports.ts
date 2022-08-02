/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ReportSearchInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: reports
// ====================================================

export interface reports_reports_pageInfo {
  __typename: "PageInfo";
  hasNextPage: boolean;
  endCursor: string | null;
}

export interface reports_reports_edges_node {
  __typename: "Report";
  id: string | null;
  language: string;
  reason: string;
  details: string | null;
  isOwn: boolean;
}

export interface reports_reports_edges {
  __typename: "ReportEdge";
  cursor: string;
  node: reports_reports_edges_node;
}

export interface reports_reports {
  __typename: "ReportSearchResult";
  pageInfo: reports_reports_pageInfo;
  edges: reports_reports_edges[];
}

export interface reports {
  reports: reports_reports;
}

export interface reportsVariables {
  input: ReportSearchInput;
}
