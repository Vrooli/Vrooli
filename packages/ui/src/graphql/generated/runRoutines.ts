/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunRoutineSearchInput, RunStatus } from "./globalTypes";

// ====================================================
// GraphQL query operation: runRoutines
// ====================================================

export interface runRoutines_runRoutines_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface runRoutines_runRoutines_edges_node_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface runRoutines_runRoutines_edges_node_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutines_runRoutines_edges_node_routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: runRoutines_runRoutines_edges_node_routine_tags_translations[];
}

export interface runRoutines_runRoutines_edges_node_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runRoutines_runRoutines_edges_node_routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: runRoutines_runRoutines_edges_node_routine_permissionsRoutine;
  tags: runRoutines_runRoutines_edges_node_routine_tags[];
  translations: runRoutines_runRoutines_edges_node_routine_translations[];
}

export interface runRoutines_runRoutines_edges_node {
  __typename: "RunRoutine";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  isPrivate: boolean;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  routine: runRoutines_runRoutines_edges_node_routine | null;
}

export interface runRoutines_runRoutines_edges {
  __typename: "RunRoutineEdge";
  cursor: string;
  node: runRoutines_runRoutines_edges_node;
}

export interface runRoutines_runRoutines {
  __typename: "RunRoutineSearchResult";
  pageInfo: runRoutines_runRoutines_pageInfo;
  edges: runRoutines_runRoutines_edges[];
}

export interface runRoutines {
  runRoutines: runRoutines_runRoutines;
}

export interface runRoutinesVariables {
  input: RunRoutineSearchInput;
}
