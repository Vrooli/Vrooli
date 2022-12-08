/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunRoutineStepStatus } from "./globalTypes";

// ====================================================
// GraphQL fragment: runRoutineStepFields
// ====================================================

export interface runRoutineStepFields_node {
  __typename: "Node";
  id: string;
}

export interface runRoutineStepFields {
  __typename: "RunRoutineStep";
  id: string;
  order: number;
  contextSwitches: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  name: string;
  status: RunRoutineStepStatus;
  step: number[];
  node: runRoutineStepFields_node | null;
}
