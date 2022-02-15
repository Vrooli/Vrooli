/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ResourceSearchInput, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL query operation: resources
// ====================================================

export interface resources_resources_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface resources_resources_edges_node {
  __typename: "Resource";
  id: string;
  title: string;
  description: string | null;
  link: string;
  usedFor: ResourceUsedFor | null;
}

export interface resources_resources_edges {
  __typename: "ResourceEdge";
  cursor: string;
  node: resources_resources_edges_node;
}

export interface resources_resources {
  __typename: "ResourceSearchResult";
  pageInfo: resources_resources_pageInfo;
  edges: resources_resources_edges[];
}

export interface resources {
  resources: resources_resources;
}

export interface resourcesVariables {
  input: ResourceSearchInput;
}
