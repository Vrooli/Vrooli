/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProjectOrOrganizationSearchInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: projectOrOrganizations
// ====================================================

export interface projectOrOrganizations_projectOrOrganizations_pageInfo {
  __typename: "ProjectOrOrganizationPageInfo";
  endCursorProject: string | null;
  endCursorOrganization: string | null;
  hasNextPage: boolean;
}

export interface projectOrOrganizations_projectOrOrganizations_edges_node_Project_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface projectOrOrganizations_projectOrOrganizations_edges_node_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface projectOrOrganizations_projectOrOrganizations_edges_node_Project_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: projectOrOrganizations_projectOrOrganizations_edges_node_Project_tags_translations[];
}

export interface projectOrOrganizations_projectOrOrganizations_edges_node_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface projectOrOrganizations_projectOrOrganizations_edges_node_Project {
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
  permissionsProject: projectOrOrganizations_projectOrOrganizations_edges_node_Project_permissionsProject;
  tags: projectOrOrganizations_projectOrOrganizations_edges_node_Project_tags[];
  translations: projectOrOrganizations_projectOrOrganizations_edges_node_Project_translations[];
}

export interface projectOrOrganizations_projectOrOrganizations_edges_node_Organization_permissionsOrganization {
  __typename: "OrganizationPermission";
  canAddMembers: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  isMember: boolean;
}

export interface projectOrOrganizations_projectOrOrganizations_edges_node_Organization_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface projectOrOrganizations_projectOrOrganizations_edges_node_Organization_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: projectOrOrganizations_projectOrOrganizations_edges_node_Organization_tags_translations[];
}

export interface projectOrOrganizations_projectOrOrganizations_edges_node_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface projectOrOrganizations_projectOrOrganizations_edges_node_Organization {
  __typename: "Organization";
  id: string;
  commentsCount: number;
  handle: string | null;
  stars: number;
  isOpenToNewMembers: boolean;
  isPrivate: boolean;
  isStarred: boolean;
  membersCount: number;
  reportsCount: number;
  permissionsOrganization: projectOrOrganizations_projectOrOrganizations_edges_node_Organization_permissionsOrganization | null;
  tags: projectOrOrganizations_projectOrOrganizations_edges_node_Organization_tags[];
  translations: projectOrOrganizations_projectOrOrganizations_edges_node_Organization_translations[];
}

export type projectOrOrganizations_projectOrOrganizations_edges_node = projectOrOrganizations_projectOrOrganizations_edges_node_Project | projectOrOrganizations_projectOrOrganizations_edges_node_Organization;

export interface projectOrOrganizations_projectOrOrganizations_edges {
  __typename: "ProjectOrOrganizationEdge";
  cursor: string;
  node: projectOrOrganizations_projectOrOrganizations_edges_node;
}

export interface projectOrOrganizations_projectOrOrganizations {
  __typename: "ProjectOrOrganizationSearchResult";
  pageInfo: projectOrOrganizations_projectOrOrganizations_pageInfo;
  edges: projectOrOrganizations_projectOrOrganizations_edges[];
}

export interface projectOrOrganizations {
  projectOrOrganizations: projectOrOrganizations_projectOrOrganizations;
}

export interface projectOrOrganizationsVariables {
  input: ProjectOrOrganizationSearchInput;
}
