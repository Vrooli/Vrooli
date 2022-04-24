/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: routineNodeLinkFields
// ====================================================

export interface routineNodeLinkFields_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineNodeLinkFields_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: routineNodeLinkFields_whens_translations[];
}

export interface routineNodeLinkFields {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  whens: routineNodeLinkFields_whens[];
}
