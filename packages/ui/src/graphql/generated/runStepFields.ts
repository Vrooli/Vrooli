/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunStepStatus } from "./globalTypes";

// ====================================================
// GraphQL fragment: runStepFields
// ====================================================

export interface runStepFields_node {
  __typename: "Node";
  id: string;
}

export interface runStepFields {
  __typename: "RunStep";
  id: string;
  order: number;
  pickups: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStepStatus;
  node: runStepFields_node | null;
}
