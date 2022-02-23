/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: nodeLinkFields
// ====================================================

export interface nodeLinkFields_conditions_when {
  __typename: "NodeLinkConditionCase";
  id: string;
  condition: string;
}

export interface nodeLinkFields_conditions {
  __typename: "NodeLinkCondition";
  id: string;
  description: string | null;
  title: string;
  when: nodeLinkFields_conditions_when[];
}

export interface nodeLinkFields {
  __typename: "NodeLink";
  id: string;
  nextId: string;
  previousId: string;
  conditions: nodeLinkFields_conditions[];
}
