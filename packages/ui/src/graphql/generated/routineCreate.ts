/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RoutineCreateInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: routineCreate
// ====================================================

export interface routineCreate_routineCreate_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineCreate_routineCreate_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: routineCreate_routineCreate_tags_translations[];
}

export interface routineCreate_routineCreate_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineCreate_routineCreate {
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
  tags: routineCreate_routineCreate_tags[];
  translations: routineCreate_routineCreate_translations[];
  version: string | null;
}

export interface routineCreate {
  routineCreate: routineCreate_routineCreate;
}

export interface routineCreateVariables {
  input: RoutineCreateInput;
}
