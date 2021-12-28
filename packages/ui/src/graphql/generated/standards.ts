/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { StandardSearchInput, StandardType } from "./globalTypes";

// ====================================================
// GraphQL query operation: standards
// ====================================================

export interface standards_standards_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface standards_standards_edges_node {
  __typename: "Standard";
  id: string;
  name: string;
  description: string | null;
  type: StandardType;
  schema: string;
  default: string | null;
  isFile: boolean;
  created_at: any;
}

export interface standards_standards_edges {
  __typename: "StandardEdge";
  cursor: string;
  node: standards_standards_edges_node;
}

export interface standards_standards {
  __typename: "StandardSearchResult";
  pageInfo: standards_standards_pageInfo;
  edges: standards_standards_edges[];
}

export interface standards {
  standards: standards_standards;
}

export interface standardsVariables {
  input: StandardSearchInput;
}
