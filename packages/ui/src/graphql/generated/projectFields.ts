/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: projectFields
// ====================================================

export interface projectFields_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
}

export interface projectFields {
  __typename: "Project";
  id: string;
  name: string;
  description: string | null;
  created_at: any;
  tags: projectFields_tags[];
}
