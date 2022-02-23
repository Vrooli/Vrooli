/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: deepRoutineOutputFields
// ====================================================

export interface deepRoutineOutputFields_standard_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface deepRoutineOutputFields_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  description: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: deepRoutineOutputFields_standard_tags[];
}

export interface deepRoutineOutputFields {
  __typename: "OutputItem";
  id: string;
  standard: deepRoutineOutputFields_standard | null;
}
