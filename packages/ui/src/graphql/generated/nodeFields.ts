/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeType } from "./globalTypes";

// ====================================================
// GraphQL fragment: nodeFields
// ====================================================

export interface nodeFields_data_NodeCombine_from {
  __typename: "NodeCombineFrom";
  id: string;
}

export interface nodeFields_data_NodeCombine_to {
  __typename: "Node";
  id: string;
}

export interface nodeFields_data_NodeCombine {
  __typename: "NodeCombine";
  id: string;
  from: nodeFields_data_NodeCombine_from[];
  to: nodeFields_data_NodeCombine_to | null;
}

export interface nodeFields_data_NodeDecision_decisions_when {
  __typename: "NodeDecisionItemCase";
  id: string;
  condition: string;
}

export interface nodeFields_data_NodeDecision_decisions {
  __typename: "NodeDecisionItem";
  id: string;
  title: string;
  when: (nodeFields_data_NodeDecision_decisions_when | null)[];
}

export interface nodeFields_data_NodeDecision {
  __typename: "NodeDecision";
  id: string;
  decisions: nodeFields_data_NodeDecision_decisions[];
}

export interface nodeFields_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
}

export interface nodeFields_data_NodeLoop {
  __typename: "NodeLoop";
  id: string;
}

export interface nodeFields_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
}

export interface nodeFields_data_NodeRedirect {
  __typename: "NodeRedirect";
  id: string;
}

export interface nodeFields_data_NodeStart {
  __typename: "NodeStart";
  id: string;
}

export type nodeFields_data = nodeFields_data_NodeCombine | nodeFields_data_NodeDecision | nodeFields_data_NodeEnd | nodeFields_data_NodeLoop | nodeFields_data_NodeRoutineList | nodeFields_data_NodeRedirect | nodeFields_data_NodeStart;

export interface nodeFields_previous {
  __typename: "Node";
  id: string;
}

export interface nodeFields_next {
  __typename: "Node";
  id: string;
}

export interface nodeFields {
  __typename: "Node";
  id: string;
  created_at: any;
  updated_at: any;
  routineId: string;
  title: string;
  description: string | null;
  type: NodeType;
  data: nodeFields_data | null;
  previous: nodeFields_previous | null;
  next: nodeFields_next | null;
}
