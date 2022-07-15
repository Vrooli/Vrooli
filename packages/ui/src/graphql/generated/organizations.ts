/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { OrganizationSearchInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL query operation: organizations
// ====================================================

export interface organizations_organizations_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface organizations_organizations_edges_node_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface organizations_organizations_edges_node_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: organizations_organizations_edges_node_tags_translations[];
}

export interface organizations_organizations_edges_node_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface organizations_organizations_edges_node {
  __typename: "Organization";
  id: string;
  commentsCount: number;
  handle: string | null;
  stars: number;
  isOpenToNewMembers: boolean;
  isStarred: boolean;
  reportsCount: number;
  role: MemberRole | null;
  tags: organizations_organizations_edges_node_tags[];
  translations: organizations_organizations_edges_node_translations[];
}

export interface organizations_organizations_edges {
  __typename: "OrganizationEdge";
  cursor: string;
  node: organizations_organizations_edges_node;
}

export interface organizations_organizations {
  __typename: "OrganizationSearchResult";
  pageInfo: organizations_organizations_pageInfo;
  edges: organizations_organizations_edges[];
}

export interface organizations {
  organizations: organizations_organizations;
}

export interface organizationsVariables {
  input: OrganizationSearchInput;
}
