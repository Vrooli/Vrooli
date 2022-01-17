/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProjectSearchInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: projects
// ====================================================

export interface projects_projects_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface projects_projects_edges_node_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  votes: number;
  isUpvoted: boolean;
}

export interface projects_projects_edges_node_owner_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface projects_projects_edges_node_owner_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type projects_projects_edges_node_owner = projects_projects_edges_node_owner_Organization | projects_projects_edges_node_owner_User;

export interface projects_projects_edges_node {
  __typename: "Project";
  id: string;
  name: string;
  description: string | null;
  created_at: any;
  tags: projects_projects_edges_node_tags[];
  owner: projects_projects_edges_node_owner | null;
  stars: number;
  votes: number;
  isUpvoted: boolean;
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
