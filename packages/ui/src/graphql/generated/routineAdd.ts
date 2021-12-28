/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RoutineInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: routineAdd
// ====================================================

export interface routineAdd_routineAdd {
  __typename: "Routine";
  id: string;
  version: string | null;
  title: string | null;
  description: string | null;
  created_at: any;
  isAutomatable: boolean | null;
}

export interface routineAdd {
  routineAdd: routineAdd_routineAdd;
}

export interface routineAddVariables {
  input: RoutineInput;
}
