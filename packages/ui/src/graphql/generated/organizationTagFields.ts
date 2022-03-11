/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: organizationTagFields
// ====================================================

export interface organizationTagFields_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface organizationTagFields {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: organizationTagFields_translations[];
}
