/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: runNodeLinkFields
// ====================================================

export interface runNodeLinkFields_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runNodeLinkFields_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: runNodeLinkFields_whens_translations[];
}

export interface runNodeLinkFields {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  operation: string | null;
  whens: runNodeLinkFields_whens[];
}
