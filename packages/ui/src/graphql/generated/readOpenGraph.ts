/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL query operation: readOpenGraph
// ====================================================

export interface readOpenGraph_readOpenGraph {
  __typename: "OpenGraphResponse";
  site: string | null;
  title: string | null;
  description: string | null;
  imageUrl: string | null;
}

export interface readOpenGraph {
  readOpenGraph: readOpenGraph_readOpenGraph;
}

export interface readOpenGraphVariables {
  url: string;
}
