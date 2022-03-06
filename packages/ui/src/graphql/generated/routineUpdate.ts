/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RoutineUpdateInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: routineUpdate
// ====================================================

export interface routineUpdate_routineUpdate_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface routineUpdate_routineUpdate {
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
  tags: routineUpdate_routineUpdate_tags[];
  title: string | null;
  version: string | null;
}

export interface routineUpdate {
  routineUpdate: routineUpdate_routineUpdate;
}

export interface routineUpdateVariables {
  input: RoutineUpdateInput;
}
