/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { TagsQueryInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: tags
// ====================================================

export interface tags_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
}

export interface tags {
  tags: tags_tags[];
}

export interface tagsVariables {
  input: TagsQueryInput;
}
