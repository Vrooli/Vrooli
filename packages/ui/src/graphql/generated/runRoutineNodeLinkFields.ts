/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: runRoutineNodeLinkFields
// ====================================================

export interface runRoutineNodeLinkFields_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runRoutineNodeLinkFields_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: runRoutineNodeLinkFields_whens_translations[];
}

export interface runRoutineNodeLinkFields {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  operation: string | null;
  whens: runRoutineNodeLinkFields_whens[];
}
