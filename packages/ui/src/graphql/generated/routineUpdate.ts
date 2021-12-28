/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RoutineInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: routineUpdate
// ====================================================

export interface routineUpdate_routineUpdate {
  __typename: "Routine";
  id: string;
  version: string | null;
  title: string | null;
  description: string | null;
  created_at: any;
  isAutomatable: boolean | null;
}

export interface routineUpdate {
  routineUpdate: routineUpdate_routineUpdate;
}

export interface routineUpdateVariables {
  input: RoutineInput;
}
