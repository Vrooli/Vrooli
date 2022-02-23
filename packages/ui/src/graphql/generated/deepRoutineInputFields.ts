/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: deepRoutineInputFields
// ====================================================

export interface deepRoutineInputFields_standard_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface deepRoutineInputFields_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  description: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: deepRoutineInputFields_standard_tags[];
}

export interface deepRoutineInputFields {
  __typename: "InputItem";
  id: string;
  standard: deepRoutineInputFields_standard | null;
}
