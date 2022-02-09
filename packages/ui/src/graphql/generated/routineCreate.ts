/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RoutineCreateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: routineCreate
// ====================================================

export interface routineCreate_routineCreate_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean | null;
}

export interface routineCreate_routineCreate {
  __typename: "Routine";
  id: string;
  version: string | null;
  title: string | null;
  description: string | null;
  created_at: any;
  isAutomatable: boolean | null;
  tags: routineCreate_routineCreate_tags[];
  stars: number;
  isStarred: boolean | null;
  score: number;
  isUpvoted: boolean | null;
}

export interface routineCreate {
  routineCreate: routineCreate_routineCreate;
}

export interface routineCreateVariables {
  input: RoutineCreateInput;
}
