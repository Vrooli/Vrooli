/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RoutineInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: routineAdd
// ====================================================

export interface routineAdd_routineAdd_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  score: number;
  isUpvoted: boolean;
}

export interface routineAdd_routineAdd {
  __typename: "Routine";
  id: string;
  version: string | null;
  title: string | null;
  description: string | null;
  created_at: any;
  isAutomatable: boolean | null;
  tags: routineAdd_routineAdd_tags[];
  stars: number;
  score: number;
  isUpvoted: boolean;
}

export interface routineAdd {
  routineAdd: routineAdd_routineAdd;
}

export interface routineAddVariables {
  input: RoutineInput;
}
