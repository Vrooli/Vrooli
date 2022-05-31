/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { StepInputDataCreateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: stepInputDataCreate
// ====================================================

export interface stepInputDataCreate_stepInputDataCreate_inputs {
  __typename: "StepInputDataInput";
  id: string;
  inputId: string;
  standardId: string | null;
  name: string;
  value: string;
}

export interface stepInputDataCreate_stepInputDataCreate {
  __typename: "StepInputData";
  id: string;
  stepId: string;
  runId: string;
  nodeId: string;
  routineId: string;
  subroutineId: string | null;
  inputs: stepInputDataCreate_stepInputDataCreate_inputs[];
}

export interface stepInputDataCreate {
  stepInputDataCreate: stepInputDataCreate_stepInputDataCreate;
}

export interface stepInputDataCreateVariables {
  input: StepInputDataCreateInput;
}
