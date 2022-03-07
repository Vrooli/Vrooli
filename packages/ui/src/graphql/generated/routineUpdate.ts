/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RoutineUpdateInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: routineUpdate
// ====================================================

export interface routineUpdate_routineUpdate_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineUpdate_routineUpdate_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: routineUpdate_routineUpdate_tags_translations[];
}

export interface routineUpdate_routineUpdate_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineUpdate_routineUpdate {
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
  tags: routineUpdate_routineUpdate_tags[];
  translations: routineUpdate_routineUpdate_translations[];
  version: string | null;
}

export interface routineUpdate {
  routineUpdate: routineUpdate_routineUpdate;
}

export interface routineUpdateVariables {
  input: RoutineUpdateInput;
}
