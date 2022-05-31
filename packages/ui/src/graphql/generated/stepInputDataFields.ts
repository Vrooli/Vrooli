/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: stepInputDataFields
// ====================================================

export interface stepInputDataFields_inputs {
  __typename: "StepInputDataInput";
  id: string;
  inputId: string;
  standardId: string | null;
  name: string;
  value: string;
}

export interface stepInputDataFields {
  __typename: "StepInputData";
  id: string;
  stepId: string;
  runId: string;
  nodeId: string;
  routineId: string;
  subroutineId: string | null;
  inputs: stepInputDataFields_inputs[];
}
