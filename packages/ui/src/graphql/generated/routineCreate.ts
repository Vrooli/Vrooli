/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RoutineCreateInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: routineCreate
// ====================================================

export interface routineCreate_routineCreate_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface routineCreate_routineCreate {
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
  tags: routineCreate_routineCreate_tags[];
  title: string | null;
  version: string | null;
}

export interface routineCreate {
  routineCreate: routineCreate_routineCreate;
}

export interface routineCreateVariables {
  input: RoutineCreateInput;
}
