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
  tag: string;
  translations: organizations_organizations_edges_node_tags_translations[];
}

export interface organizations_organizations_edges_node_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  bio: string | null;
  name: string;
}

export interface organizations_organizations_edges_node {
  __typename: "Organization";
  id: string;
  created_at: any;
  isOpenToNewMembers: boolean;
  isStarred: boolean;
  role: MemberRole | null;
  stars: number;
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
