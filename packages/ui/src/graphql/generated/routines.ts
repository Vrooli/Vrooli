/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RoutineSearchInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL query operation: routines
// ====================================================

export interface routines_routines_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface routines_routines_edges_node_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routines_routines_edges_node_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: routines_routines_edges_node_tags_translations[];
}

export interface routines_routines_edges_node_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routines_routines_edges_node {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  tags: routines_routines_edges_node_tags[];
  translations: routines_routines_edges_node_translations[];
  version: string | null;
}

export interface routines_routines_edges {
  __typename: "RoutineEdge";
  cursor: string;
  node: routines_routines_edges_node;
}

export interface routines_routines {
  __typename: "RoutineSearchResult";
  pageInfo: routines_routines_pageInfo;
  edges: routines_routines_edges[];
}

export interface routines {
  routines: routines_routines;
}

export interface routinesVariables {
  input: RoutineSearchInput;
}
