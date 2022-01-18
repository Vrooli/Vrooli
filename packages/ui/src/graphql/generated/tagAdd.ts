/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { TagInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: tagAdd
// ====================================================

export interface tagAdd_tagAdd {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  score: number;
  isUpvoted: boolean;
}

export interface tagAdd {
  tagAdd: tagAdd_tagAdd;
}

export interface tagAddVariables {
  input: TagInput;
}
