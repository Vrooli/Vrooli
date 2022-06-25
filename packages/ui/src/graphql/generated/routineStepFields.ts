/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunStepStatus } from "./globalTypes";

// ====================================================
// GraphQL fragment: routineStepFields
// ====================================================

export interface routineStepFields_node {
  __typename: "Node";
  id: string;
}

export interface routineStepFields {
  __typename: "RunStep";
  id: string;
  order: number;
  contextSwitches: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStepStatus;
  step: number[];
  node: routineStepFields_node | null;
}
