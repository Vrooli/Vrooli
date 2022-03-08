/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { UserSearchInput, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL query operation: users
// ====================================================

export interface users_users_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface users_users_edges_node_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface users_users_edges_node_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface users_users_edges_node_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: users_users_edges_node_resourceLists_resources_translations[];
}

export interface users_users_edges_node_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: users_users_edges_node_resourceLists_translations[];
  resources: users_users_edges_node_resourceLists_resources[];
}

export interface users_users_edges_node_translations {
  __typename: "UserTranslation";
  id: string;
  language: string;
  bio: string | null;
}

export interface users_users_edges_node {
  __typename: "User";
  id: string;
  username: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean;
  resourceLists: users_users_edges_node_resourceLists[];
  translations: users_users_edges_node_translations[];
}

export interface users_users_edges {
  __typename: "UserEdge";
  cursor: string;
  node: users_users_edges_node;
}

export interface users_users {
  __typename: "UserSearchResult";
  pageInfo: users_users_pageInfo;
  edges: users_users_edges[];
}

export interface users {
  users: users_users;
}

export interface usersVariables {
  input: UserSearchInput;
}
