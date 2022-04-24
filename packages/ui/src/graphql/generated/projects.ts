/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProjectSearchInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL query operation: projects
// ====================================================

export interface projects_projects_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface projects_projects_edges_node_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface projects_projects_edges_node_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: projects_projects_edges_node_tags_translations[];
}

export interface projects_projects_edges_node_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface projects_projects_edges_node {
  __typename: "Project";
  id: string;
  handle: string | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  tags: projects_projects_edges_node_tags[];
  translations: projects_projects_edges_node_translations[];
}

export interface projects_projects_edges {
  __typename: "ProjectEdge";
  cursor: string;
  node: projects_projects_edges_node;
}

export interface projects_projects {
  __typename: "ProjectSearchResult";
  pageInfo: projects_projects_pageInfo;
  edges: projects_projects_edges[];
}

export interface projects {
  projects: projects_projects;
}

export interface projectsVariables {
  input: ProjectSearchInput;
}
