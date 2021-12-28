/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeInput, NodeType } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: nodeUpdate
// ====================================================

export interface nodeUpdate_nodeUpdate_data_NodeCombine_from {
  __typename: "NodeCombineFrom";
  id: string;
}

export interface nodeUpdate_nodeUpdate_data_NodeCombine_to {
  __typename: "Node";
  id: string;
}

export interface nodeUpdate_nodeUpdate_data_NodeCombine {
  __typename: "NodeCombine";
  id: string;
  from: nodeUpdate_nodeUpdate_data_NodeCombine_from[];
  to: nodeUpdate_nodeUpdate_data_NodeCombine_to | null;
}

export interface nodeUpdate_nodeUpdate_data_NodeDecision_decisions_when {
  __typename: "NodeDecisionItemCase";
  id: string;
  condition: string;
}

export interface nodeUpdate_nodeUpdate_data_NodeDecision_decisions {
  __typename: "NodeDecisionItem";
  id: string;
  title: string;
  when: (nodeUpdate_nodeUpdate_data_NodeDecision_decisions_when | null)[];
}

export interface nodeUpdate_nodeUpdate_data_NodeDecision {
  __typename: "NodeDecision";
  id: string;
  decisions: nodeUpdate_nodeUpdate_data_NodeDecision_decisions[];
}

export interface nodeUpdate_nodeUpdate_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
}

export interface nodeUpdate_nodeUpdate_data_NodeLoop {
  __typename: "NodeLoop";
  id: string;
}

export interface nodeUpdate_nodeUpdate_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
}

export interface nodeUpdate_nodeUpdate_data_NodeRedirect {
  __typename: "NodeRedirect";
  id: string;
}

export interface nodeUpdate_nodeUpdate_data_NodeStart {
  __typename: "NodeStart";
  id: string;
}

export type nodeUpdate_nodeUpdate_data = nodeUpdate_nodeUpdate_data_NodeCombine | nodeUpdate_nodeUpdate_data_NodeDecision | nodeUpdate_nodeUpdate_data_NodeEnd | nodeUpdate_nodeUpdate_data_NodeLoop | nodeUpdate_nodeUpdate_data_NodeRoutineList | nodeUpdate_nodeUpdate_data_NodeRedirect | nodeUpdate_nodeUpdate_data_NodeStart;

export interface nodeUpdate_nodeUpdate_previous {
  __typename: "Node";
  id: string;
}

export interface nodeUpdate_nodeUpdate_next {
  __typename: "Node";
  id: string;
}

export interface nodeUpdate_nodeUpdate {
  __typename: "Node";
  id: string;
  created_at: any;
  updated_at: any;
  routineId: string;
  title: string;
  description: string | null;
  type: NodeType;
  data: nodeUpdate_nodeUpdate_data | null;
  previous: nodeUpdate_nodeUpdate_previous | null;
  next: nodeUpdate_nodeUpdate_next | null;
}

export interface nodeUpdate {
  nodeUpdate: nodeUpdate_nodeUpdate;
}

export interface nodeUpdateVariables {
  input: NodeInput;
}
