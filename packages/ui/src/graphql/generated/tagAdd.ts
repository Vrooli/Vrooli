/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { TagAddInput } from "./globalTypes";

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
  isStarred: boolean | null;
}

export interface tagAdd {
  tagAdd: tagAdd_tagAdd;
}

export interface tagAddVariables {
  input: TagAddInput;
}
