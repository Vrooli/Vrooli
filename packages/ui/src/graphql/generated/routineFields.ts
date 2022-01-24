/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: routineFields
// ====================================================

export interface routineFields_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean | null;
}

export interface routineFields {
  __typename: "Routine";
  id: string;
  version: string | null;
  title: string | null;
  description: string | null;
  created_at: any;
  isAutomatable: boolean | null;
  tags: routineFields_tags[];
  stars: number;
  isStarred: boolean | null;
  score: number;
  isUpvoted: boolean | null;
}
