/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RoutineCancelInput, LogType } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: routineCancel
// ====================================================

export interface routineCancel_routineCancel {
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

export interface routineCancel {
  routineCancel: routineCancel_routineCancel;
}

export interface routineCancelVariables {
  input: RoutineCancelInput;
}
