/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: deepRoutineNodeLinkFields
// ====================================================

export interface deepRoutineNodeLinkFields_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface deepRoutineNodeLinkFields_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: deepRoutineNodeLinkFields_whens_translations[];
}

export interface deepRoutineNodeLinkFields {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  whens: deepRoutineNodeLinkFields_whens[];
}
