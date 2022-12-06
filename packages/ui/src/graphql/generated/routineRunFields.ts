/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunStatus, RunRoutineStepStatus } from "./globalTypes";

// ====================================================
// GraphQL fragment: routineRunFields
// ====================================================

export interface routineRunFields_inputs_input {
  __typename: "InputItem";
  id: string;
}

export interface routineRunFields_inputs {
  __typename: "RunRoutineInput";
  id: string;
  data: string;
  input: routineRunFields_inputs_input;
}

export interface routineRunFields_steps_node {
  __typename: "Node";
  id: string;
}

export interface routineRunFields_steps {
  __typename: "RunRoutineStep";
  id: string;
  order: number;
  contextSwitches: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunRoutineStepStatus;
  step: number[];
  node: routineRunFields_steps_node | null;
}

export interface routineRunFields {
  __typename: "RunRoutine";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  inputs: routineRunFields_inputs[];
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  steps: routineRunFields_steps[];
}
