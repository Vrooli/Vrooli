/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: ioFields
// ====================================================

export interface ioFields_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
}

export interface ioFields_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  description: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: ioFields_standard_tags[];
}

export interface ioFields {
  __typename: "RoutineInputItem";
  id: string;
  standard: ioFields_standard;
}
