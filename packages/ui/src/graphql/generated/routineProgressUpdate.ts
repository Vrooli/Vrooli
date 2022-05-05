/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RoutineProgressUpdateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: routineProgressUpdate
// ====================================================

export interface routineProgressUpdate_routineProgressUpdate {
  __typename: "Success";
  success: boolean | null;
}

export interface routineProgressUpdate {
  routineProgressUpdate: routineProgressUpdate_routineProgressUpdate;
}

export interface routineProgressUpdateVariables {
  input: RoutineProgressUpdateInput;
}
