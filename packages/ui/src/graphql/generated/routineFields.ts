/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: routineFields
// ====================================================

export interface routineFields_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface routineFields {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  created_at: any;
  description: string | null;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  tags: routineFields_tags[];
  title: string | null;
  version: string | null;
}
