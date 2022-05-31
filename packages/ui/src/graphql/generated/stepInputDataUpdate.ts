/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { StepInputDataUpdateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: stepInputDataUpdate
// ====================================================

export interface stepInputDataUpdate_stepInputDataUpdate_inputs {
  __typename: "StepInputDataInput";
  id: string;
  inputId: string;
  standardId: string | null;
  name: string;
  value: string;
}

export interface stepInputDataUpdate_stepInputDataUpdate {
  __typename: "StepInputData";
  id: string;
  stepId: string;
  runId: string;
  nodeId: string;
  routineId: string;
  subroutineId: string | null;
  inputs: stepInputDataUpdate_stepInputDataUpdate_inputs[];
}

export interface stepInputDataUpdate {
  stepInputDataUpdate: stepInputDataUpdate_stepInputDataUpdate;
}

export interface stepInputDataUpdateVariables {
  input: StepInputDataUpdateInput;
}
