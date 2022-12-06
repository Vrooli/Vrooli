/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunRoutineInputSearchInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: runRoutineInputs
// ====================================================

export interface runRoutineInputs_runRoutineInputs_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface runRoutineInputs_runRoutineInputs_edges_node_input_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineInputs_runRoutineInputs_edges_node_input_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineInputs_runRoutineInputs_edges_node_input_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineInputs_runRoutineInputs_edges_node_input_standard_tags_translations[];
}

export interface runRoutineInputs_runRoutineInputs_edges_node_input_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineInputs_runRoutineInputs_edges_node_input_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isDeleted: boolean;
  isInternal: boolean;
  isPrivate: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  tags: runRoutineInputs_runRoutineInputs_edges_node_input_standard_tags[];
  translations: runRoutineInputs_runRoutineInputs_edges_node_input_standard_translations[];
}

export interface runRoutineInputs_runRoutineInputs_edges_node_input {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineInputs_runRoutineInputs_edges_node_input_translations[];
  standard: runRoutineInputs_runRoutineInputs_edges_node_input_standard | null;
}

export interface runRoutineInputs_runRoutineInputs_edges_node {
  __typename: "RunRoutineInput";
  id: string;
  data: string;
  input: runRoutineInputs_runRoutineInputs_edges_node_input;
}

export interface runRoutineInputs_runRoutineInputs_edges {
  __typename: "RunRoutineInputEdge";
  cursor: string;
  node: runRoutineInputs_runRoutineInputs_edges_node;
}

export interface runRoutineInputs_runRoutineInputs {
  __typename: "RunRoutineInputSearchResult";
  pageInfo: runRoutineInputs_runRoutineInputs_pageInfo;
  edges: runRoutineInputs_runRoutineInputs_edges[];
}

export interface runRoutineInputs {
  runRoutineInputs: runRoutineInputs_runRoutineInputs;
}

export interface runRoutineInputsVariables {
  input: RunRoutineInputSearchInput;
}
