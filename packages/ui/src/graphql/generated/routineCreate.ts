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
  version: string | null;
  title: string | null;
  description: string | null;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  role: MemberRole | null;
  tags: routineCreate_routineCreate_tags[];
  stars: number;
  isStarred: boolean;
  score: number;
  isUpvoted: boolean | null;
}

export interface routineCreate {
  routineCreate: routineCreate_routineCreate;
}

export interface routineCreateVariables {
  input: RoutineCreateInput;
}
