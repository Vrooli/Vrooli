/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { TagCreateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: tagCreate
// ====================================================

export interface tagCreate_tagCreate_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface tagCreate_tagCreate {
  __typename: "Tag";
  tag: string;
  created_at: any;
  stars: number;
  isStarred: boolean;
  translations: tagCreate_tagCreate_translations[];
}

export interface tagCreate {
  tagCreate: tagCreate_tagCreate;
}

export interface tagCreateVariables {
  input: TagCreateInput;
}
