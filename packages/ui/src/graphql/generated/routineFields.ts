/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: routineFields
// ====================================================

export interface routineFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineFields_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: routineFields_tags_translations[];
}

export interface routineFields_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineFields {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  tags: routineFields_tags[];
  translations: routineFields_translations[];
  version: string | null;
}
