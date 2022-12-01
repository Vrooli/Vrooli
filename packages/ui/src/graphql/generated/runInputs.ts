/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunInputSearchInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: runInputs
// ====================================================

export interface runInputs_runInputs_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface runInputs_runInputs_edges_node_input_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runInputs_runInputs_edges_node_input_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runInputs_runInputs_edges_node_input_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runInputs_runInputs_edges_node_input_standard_tags_translations[];
}

export interface runInputs_runInputs_edges_node_input_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runInputs_runInputs_edges_node_input_standard {
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
  tags: runInputs_runInputs_edges_node_input_standard_tags[];
  translations: runInputs_runInputs_edges_node_input_standard_translations[];
}

export interface runInputs_runInputs_edges_node_input {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runInputs_runInputs_edges_node_input_translations[];
  standard: runInputs_runInputs_edges_node_input_standard | null;
}

export interface runInputs_runInputs_edges_node {
  __typename: "RunInput";
  id: string;
  data: string;
  input: runInputs_runInputs_edges_node_input;
}

export interface runInputs_runInputs_edges {
  __typename: "RunInputEdge";
  cursor: string;
  node: runInputs_runInputs_edges_node;
}

export interface runInputs_runInputs {
  __typename: "RunInputSearchResult";
  pageInfo: runInputs_runInputs_pageInfo;
  edges: runInputs_runInputs_edges[];
}

export interface runInputs {
  runInputs: runInputs_runInputs;
}

export interface runInputsVariables {
  input: RunInputSearchInput;
}
