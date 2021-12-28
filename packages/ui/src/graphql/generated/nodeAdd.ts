/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeInput, NodeType } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: nodeAdd
// ====================================================

export interface nodeAdd_nodeAdd_data_NodeCombine_from {
  __typename: "NodeCombineFrom";
  id: string;
}

export interface nodeAdd_nodeAdd_data_NodeCombine_to {
  __typename: "Node";
  id: string;
}

export interface nodeAdd_nodeAdd_data_NodeCombine {
  __typename: "NodeCombine";
  id: string;
  from: nodeAdd_nodeAdd_data_NodeCombine_from[];
  to: nodeAdd_nodeAdd_data_NodeCombine_to | null;
}

export interface nodeAdd_nodeAdd_data_NodeDecision_decisions_when {
  __typename: "NodeDecisionItemCase";
  id: string;
  condition: string;
}

export interface nodeAdd_nodeAdd_data_NodeDecision_decisions {
  __typename: "NodeDecisionItem";
  id: string;
  title: string;
  when: (nodeAdd_nodeAdd_data_NodeDecision_decisions_when | null)[];
}

export interface nodeAdd_nodeAdd_data_NodeDecision {
  __typename: "NodeDecision";
  id: string;
  decisions: nodeAdd_nodeAdd_data_NodeDecision_decisions[];
}

export interface nodeAdd_nodeAdd_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
}

export interface nodeAdd_nodeAdd_data_NodeLoop {
  __typename: "NodeLoop";
  id: string;
}

export interface nodeAdd_nodeAdd_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
}

export interface nodeAdd_nodeAdd_data_NodeRedirect {
  __typename: "NodeRedirect";
  id: string;
}

export interface nodeAdd_nodeAdd_data_NodeStart {
  __typename: "NodeStart";
  id: string;
}

export type nodeAdd_nodeAdd_data = nodeAdd_nodeAdd_data_NodeCombine | nodeAdd_nodeAdd_data_NodeDecision | nodeAdd_nodeAdd_data_NodeEnd | nodeAdd_nodeAdd_data_NodeLoop | nodeAdd_nodeAdd_data_NodeRoutineList | nodeAdd_nodeAdd_data_NodeRedirect | nodeAdd_nodeAdd_data_NodeStart;

export interface nodeAdd_nodeAdd_previous {
  __typename: "Node";
  id: string;
}

export interface nodeAdd_nodeAdd_next {
  __typename: "Node";
  id: string;
}

export interface nodeAdd_nodeAdd {
  __typename: "Node";
  id: string;
  created_at: any;
  updated_at: any;
  routineId: string;
  title: string;
  description: string | null;
  type: NodeType;
  data: nodeAdd_nodeAdd_data | null;
  previous: nodeAdd_nodeAdd_previous | null;
  next: nodeAdd_nodeAdd_next | null;
}

export interface nodeAdd {
  nodeAdd: nodeAdd_nodeAdd;
}

export interface nodeAddVariables {
  input: NodeInput;
}
