/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { StandardSearchInput, MemberRole, StandardType } from "./globalTypes";

// ====================================================
// GraphQL query operation: standards
// ====================================================

export interface standards_standards_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface standards_standards_edges_node_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface standards_standards_edges_node_creator_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface standards_standards_edges_node_creator_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type standards_standards_edges_node_creator = standards_standards_edges_node_creator_Organization | standards_standards_edges_node_creator_User;

export interface standards_standards_edges_node {
  __typename: "Standard";
  id: string;
  name: string;
  description: string | null;
  role: MemberRole | null;
  type: StandardType;
  schema: string;
  default: string | null;
  isFile: boolean;
  created_at: any;
  tags: standards_standards_edges_node_tags[];
  creator: standards_standards_edges_node_creator | null;
  stars: number;
  isStarred: boolean;
  score: number;
  isUpvoted: boolean | null;
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
