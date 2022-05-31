/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: stepInputData
// ====================================================

export interface stepInputData_stepInputData_inputs {
  __typename: "StepInputDataInput";
  id: string;
  inputId: string;
  standardId: string | null;
  name: string;
  value: string;
}

export interface stepInputData_stepInputData {
  __typename: "StepInputData";
  id: string;
  stepId: string;
  runId: string;
  nodeId: string;
  routineId: string;
  subroutineId: string | null;
  inputs: stepInputData_stepInputData_inputs[];
}

export interface stepInputData {
  stepInputData: stepInputData_stepInputData | null;
}

export interface stepInputDataVariables {
  input: FindByIdInput;
}
