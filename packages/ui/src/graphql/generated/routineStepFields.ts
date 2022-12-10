/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunRoutineStepStatus } from "./globalTypes";

// ====================================================
// GraphQL fragment: routineStepFields
// ====================================================

export interface routineStepFields_node {
  __typename: "Node";
  id: string;
}

export interface routineStepFields {
  __typename: "RunRoutineStep";
  id: string;
  order: number;
  contextSwitches: number;
  startedAt: any | null;
  timeElapsed: number | null;
  completedAt: any | null;
  name: string;
  status: RunRoutineStepStatus;
  step: number[];
  node: routineStepFields_node | null;
}
