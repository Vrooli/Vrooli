/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: tagFields
// ====================================================

export interface tagFields_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface tagFields {
  __typename: "Tag";
  id: string;
  tag: string;
  created_at: any;
  stars: number;
  isStarred: boolean;
  translations: tagFields_translations[];
}
