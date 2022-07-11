/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { TagUpdateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: tagUpdate
// ====================================================

export interface tagUpdate_tagUpdate_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface tagUpdate_tagUpdate {
  __typename: "Tag";
  tag: string;
  created_at: any;
  stars: number;
  isStarred: boolean;
  translations: tagUpdate_tagUpdate_translations[];
}

export interface tagUpdate {
  tagUpdate: tagUpdate_tagUpdate;
}

export interface tagUpdateVariables {
  input: TagUpdateInput;
}
