/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { StepInputDataSearchInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: stepInputDatas
// ====================================================

export interface stepInputDatas_stepInputDatas_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface stepInputDatas_stepInputDatas_edges_node_inputs {
  __typename: "StepInputDataInput";
  id: string;
  inputId: string;
  standardId: string | null;
  name: string;
  value: string;
}

export interface stepInputDatas_stepInputDatas_edges_node {
  __typename: "StepInputData";
  id: string;
  stepId: string;
  runId: string;
  nodeId: string;
  routineId: string;
  subroutineId: string | null;
  inputs: stepInputDatas_stepInputDatas_edges_node_inputs[];
}

export interface stepInputDatas_stepInputDatas_edges {
  __typename: "StepInputDataEdge";
  cursor: string;
  node: stepInputDatas_stepInputDatas_edges_node;
}

export interface stepInputDatas_stepInputDatas {
  __typename: "StepInputDataSearchResult";
  pageInfo: stepInputDatas_stepInputDatas_pageInfo;
  edges: stepInputDatas_stepInputDatas_edges[];
}

export interface stepInputDatas {
  stepInputDatas: stepInputDatas_stepInputDatas;
}

export interface stepInputDatasVariables {
  input: StepInputDataSearchInput;
}
