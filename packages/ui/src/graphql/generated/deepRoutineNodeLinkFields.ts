/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: deepRoutineNodeLinkFields
// ====================================================

export interface deepRoutineNodeLinkFields_conditions_when {
  __typename: "NodeLinkConditionCase";
  id: string;
  condition: string;
}

export interface deepRoutineNodeLinkFields_conditions {
  __typename: "NodeLinkCondition";
  id: string;
  description: string | null;
  title: string;
  when: deepRoutineNodeLinkFields_conditions_when[];
}

export interface deepRoutineNodeLinkFields {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  conditions: deepRoutineNodeLinkFields_conditions[];
}
