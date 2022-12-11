/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ViewSearchInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: views
// ====================================================

export interface views_views_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface views_views_edges_node_to_Organization_permissionsOrganization {
  __typename: "OrganizationPermission";
  canAddMembers: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  isMember: boolean;
}

export interface views_views_edges_node_to_Organization_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface views_views_edges_node_to_Organization_tags {
  __typename: "Tag";
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: views_views_edges_node_to_Organization_tags_translations[];
}

export interface views_views_edges_node_to_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface views_views_edges_node_to_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  stars: number;
  isPrivate: boolean;
  isStarred: boolean;
  permissionsOrganization: views_views_edges_node_to_Organization_permissionsOrganization | null;
  tags: views_views_edges_node_to_Organization_tags[];
  translations: views_views_edges_node_to_Organization_translations[];
}

export interface views_views_edges_node_to_Project_permissionsProject {
  __typename: "ProjectPermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface views_views_edges_node_to_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface views_views_edges_node_to_Project_tags {
  __typename: "Tag";
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: views_views_edges_node_to_Project_tags_translations[];
}

export interface views_views_edges_node_to_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface views_views_edges_node_to_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  score: number;
  stars: number;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  permissionsProject: views_views_edges_node_to_Project_permissionsProject;
  tags: views_views_edges_node_to_Project_tags[];
  translations: views_views_edges_node_to_Project_translations[];
}

export interface views_views_edges_node_to_Routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface views_views_edges_node_to_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface views_views_edges_node_to_Routine_tags {
  __typename: "Tag";
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: views_views_edges_node_to_Routine_tags_translations[];
}

export interface views_views_edges_node_to_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface views_views_edges_node_to_Routine {
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
  permissionsRoutine: views_views_edges_node_to_Routine_permissionsRoutine;
  tags: views_views_edges_node_to_Routine_tags[];
  translations: views_views_edges_node_to_Routine_translations[];
}

export interface views_views_edges_node_to_Standard_permissionsStandard {
  __typename: "StandardPermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface views_views_edges_node_to_Standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface views_views_edges_node_to_Standard_tags {
  __typename: "Tag";
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: views_views_edges_node_to_Standard_tags_translations[];
}

export interface views_views_edges_node_to_Standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface views_views_edges_node_to_Standard {
  __typename: "Standard";
  id: string;
  score: number;
  stars: number;
  isDeleted: boolean;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  name: string;
  permissionsStandard: views_views_edges_node_to_Standard_permissionsStandard;
  tags: views_views_edges_node_to_Standard_tags[];
  translations: views_views_edges_node_to_Standard_translations[];
}

export interface views_views_edges_node_to_User {
  __typename: "User";
  id: string;
  handle: string | null;
  name: string;
  stars: number;
  isStarred: boolean;
}

export type views_views_edges_node_to = views_views_edges_node_to_Organization | views_views_edges_node_to_Project | views_views_edges_node_to_Routine | views_views_edges_node_to_Standard | views_views_edges_node_to_User;

export interface views_views_edges_node {
  __typename: "View";
  id: string;
  lastViewedAt: any;
  name: string;
  to: views_views_edges_node_to;
}

export interface views_views_edges {
  __typename: "ViewEdge";
  cursor: string;
  node: views_views_edges_node;
}

export interface views_views {
  __typename: "ViewSearchResult";
  pageInfo: views_views_pageInfo;
  edges: views_views_edges[];
}

export interface views {
  views: views_views;
}

export interface viewsVariables {
  input: ViewSearchInput;
}
