/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { UserSearchInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: users
// ====================================================

export interface users_users_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
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
