/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RoutineStartInput, LogType } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: routineStart
// ====================================================

export interface routineStart_routineStart {
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

export interface routineStart {
  routineStart: routineStart_routineStart;
}

export interface routineStartVariables {
  input: RoutineStartInput;
}
