/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: tag
// ====================================================

export interface tag_tag_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface tag_tag {
  __typename: "Tag";
  tag: string;
  created_at: any;
  stars: number;
  isStarred: boolean;
  translations: tag_tag_translations[];
}

export interface tag {
  tag: tag_tag | null;
}

export interface tagVariables {
  input: FindByIdInput;
}
