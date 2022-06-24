/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunStatus, RunStepStatus } from "./globalTypes";

// ====================================================
// GraphQL fragment: routineRunFields
// ====================================================

export interface routineRunFields_steps_node {
  __typename: "Node";
  id: string;
}

export interface routineRunFields_steps {
  __typename: "RunStep";
  id: string;
  order: number;
  pickups: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStepStatus;
  step: number[];
  node: routineRunFields_steps_node | null;
}

export interface routineRunFields {
  __typename: "Run";
  id: string;
  completedComplexity: number;
  pickups: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  steps: routineRunFields_steps[];
}
