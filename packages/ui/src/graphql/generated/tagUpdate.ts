/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { TagInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: tagUpdate
// ====================================================

export interface tagUpdate_tagUpdate {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  score: number;
  isUpvoted: boolean;
}

export interface tagUpdate {
  tagUpdate: tagUpdate_tagUpdate;
}

export interface tagUpdateVariables {
  input: TagInput;
}
