/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RoutineCompleteInput, LogType } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: routineComplete
// ====================================================

export interface routineComplete_routineComplete {
  __typename: "Log";
  id: string;
  timestamp: any;
  action: LogType;
  object1Type: string | null;
  object1Id: string | null;
  object2Type: string | null;
  object2Id: string | null;
  data: string | null;
}

export interface routineComplete {
  routineComplete: routineComplete_routineComplete;
}

export interface routineCompleteVariables {
  input: RoutineCompleteInput;
}
