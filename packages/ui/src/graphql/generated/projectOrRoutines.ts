/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProjectOrRoutineSearchInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: projectOrRoutines
// ====================================================

export interface projectOrRoutines_projectOrRoutines_pageInfo {
  __typename: "ProjectOrRoutinePageInfo";
  endCursorProject: string | null;
  endCursorRoutine: string | null;
  hasNextPage: boolean;
}

export interface projectOrRoutines_projectOrRoutines_edges_node_Project_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface projectOrRoutines_projectOrRoutines_edges_node_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface projectOrRoutines_projectOrRoutines_edges_node_Project_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: projectOrRoutines_projectOrRoutines_edges_node_Project_tags_translations[];
}

export interface projectOrRoutines_projectOrRoutines_edges_node_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface projectOrRoutines_projectOrRoutines_edges_node_Project {
  __typename: "Project";
  id: string;
  commentsCount: number;
  handle: string | null;
  score: number;
  stars: number;
  isComplete: boolean;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  reportsCount: number;
  permissionsProject: projectOrRoutines_projectOrRoutines_edges_node_Project_permissionsProject;
  tags: projectOrRoutines_projectOrRoutines_edges_node_Project_tags[];
  translations: projectOrRoutines_projectOrRoutines_edges_node_Project_translations[];
}

export interface projectOrRoutines_projectOrRoutines_edges_node_Routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface projectOrRoutines_projectOrRoutines_edges_node_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface projectOrRoutines_projectOrRoutines_edges_node_Routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: projectOrRoutines_projectOrRoutines_edges_node_Routine_tags_translations[];
}

export interface projectOrRoutines_projectOrRoutines_edges_node_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface projectOrRoutines_projectOrRoutines_edges_node_Routine {
  __typename: "Routine";
  id: string;
  commentsCount: number;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isDeleted: boolean;
  isInternal: boolean | null;
  isComplete: boolean;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  reportsCount: number;
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: projectOrRoutines_projectOrRoutines_edges_node_Routine_permissionsRoutine;
  tags: projectOrRoutines_projectOrRoutines_edges_node_Routine_tags[];
  translations: projectOrRoutines_projectOrRoutines_edges_node_Routine_translations[];
  version: string;
  versionGroupId: string;
}

export type projectOrRoutines_projectOrRoutines_edges_node = projectOrRoutines_projectOrRoutines_edges_node_Project | projectOrRoutines_projectOrRoutines_edges_node_Routine;

export interface projectOrRoutines_projectOrRoutines_edges {
  __typename: "ProjectOrRoutineEdge";
  cursor: string;
  node: projectOrRoutines_projectOrRoutines_edges_node;
}

export interface projectOrRoutines_projectOrRoutines {
  __typename: "ProjectOrRoutineSearchResult";
  pageInfo: projectOrRoutines_projectOrRoutines_pageInfo;
  edges: projectOrRoutines_projectOrRoutines_edges[];
}

export interface projectOrRoutines {
  projectOrRoutines: projectOrRoutines_projectOrRoutines;
}

export interface projectOrRoutinesVariables {
  input: ProjectOrRoutineSearchInput;
}
