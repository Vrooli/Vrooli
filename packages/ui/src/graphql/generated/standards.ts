/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { StandardSearchInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL query operation: standards
// ====================================================

export interface standards_standards_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface standards_standards_edges_node_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface standards_standards_edges_node_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: standards_standards_edges_node_tags_translations[];
}

export interface standards_standards_edges_node_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface standards_standards_edges_node {
  __typename: "Standard";
  id: string;
  commentsCount: number;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  name: string;
  reportsCount: number;
  role: MemberRole | null;
  tags: standards_standards_edges_node_tags[];
  translations: standards_standards_edges_node_translations[];
  type: string;
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
