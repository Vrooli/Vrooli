/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { TagCreateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: tagCreate
// ====================================================

export interface tagCreate_tagCreate {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean;
}

export interface tagCreate {
  tagCreate: tagCreate_tagCreate;
}

export interface tagCreateVariables {
  input: TagCreateInput;
}
