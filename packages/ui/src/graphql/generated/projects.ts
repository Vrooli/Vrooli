/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProjectSearchInput, MemberRole, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL query operation: projects
// ====================================================

export interface projects_projects_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface projects_projects_edges_node_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface projects_projects_edges_node_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface projects_projects_edges_node_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: projects_projects_edges_node_resourceLists_resources_translations[];
}

export interface projects_projects_edges_node_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: projects_projects_edges_node_resourceLists_translations[];
  resources: projects_projects_edges_node_resourceLists_resources[];
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
  tag: string;
  translations: projects_projects_edges_node_tags_translations[];
}

export interface projects_projects_edges_node_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface projects_projects_edges_node_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface projects_projects_edges_node_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: projects_projects_edges_node_owner_Organization_translations[];
}

export interface projects_projects_edges_node_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type projects_projects_edges_node_owner = projects_projects_edges_node_owner_Organization | projects_projects_edges_node_owner_User;

export interface projects_projects_edges_node {
  __typename: "Project";
  id: string;
  completedAt: any | null;
  created_at: any;
  handle: string | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  resourceLists: projects_projects_edges_node_resourceLists[] | null;
  tags: projects_projects_edges_node_tags[];
  translations: projects_projects_edges_node_translations[];
  owner: projects_projects_edges_node_owner | null;
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
