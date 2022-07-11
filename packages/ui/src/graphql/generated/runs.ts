/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunSearchInput, RunStatus, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL query operation: runs
// ====================================================

export interface runs_runs_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface runs_runs_edges_node_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runs_runs_edges_node_routine_tags {
  __typename: "Tag";
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: runs_runs_edges_node_routine_tags_translations[];
}

export interface runs_runs_edges_node_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runs_runs_edges_node_routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  score: number;
  simplicity: number;
  stars: number;
  tags: runs_runs_edges_node_routine_tags[];
  translations: runs_runs_edges_node_routine_translations[];
  version: string | null;
}

export interface runs_runs_edges_node {
  __typename: "Run";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  routine: runs_runs_edges_node_routine | null;
}

export interface runs_runs_edges {
  __typename: "RunEdge";
  cursor: string;
  node: runs_runs_edges_node;
}

export interface runs_runs {
  __typename: "RunSearchResult";
  pageInfo: runs_runs_pageInfo;
  edges: runs_runs_edges[];
}

export interface runs {
  runs: runs_runs;
}

export interface runsVariables {
  input: RunSearchInput;
}
