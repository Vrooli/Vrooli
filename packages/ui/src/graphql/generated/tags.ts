/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { TagSearchInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: tags
// ====================================================

export interface tags_tags_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface tags_tags_edges_node_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface tags_tags_edges_node {
  __typename: "Tag";
  tag: string;
  created_at: any;
  stars: number;
  isStarred: boolean;
  translations: tags_tags_edges_node_translations[];
}

export interface tags_tags_edges {
  __typename: "TagEdge";
  cursor: string;
  node: tags_tags_edges_node;
}

export interface tags_tags {
  __typename: "TagSearchResult";
  pageInfo: tags_tags_pageInfo;
  edges: tags_tags_edges[];
}

export interface tags {
  tags: tags_tags;
}

export interface tagsVariables {
  input: TagSearchInput;
}
