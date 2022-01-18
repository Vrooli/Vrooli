/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeInput, NodeType } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: nodeAdd
// ====================================================

export interface nodeAdd_nodeAdd_data_NodeCombine {
  __typename: "NodeCombine";
  id: string;
  from: string[];
  to: string;
}

export interface nodeAdd_nodeAdd_data_NodeDecision_decisions_when {
  __typename: "NodeDecisionItemCase";
  id: string;
  condition: string;
}

export interface nodeAdd_nodeAdd_data_NodeDecision_decisions {
  __typename: "NodeDecisionItem";
  id: string;
  description: string | null;
  title: string;
  toId: string | null;
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
  wasSuccessful: boolean;
}

export interface nodeAdd_nodeAdd_data_NodeLoop {
  __typename: "NodeLoop";
  id: string;
}

export interface nodeAdd_nodeAdd_data_NodeRoutineList_routines_routine_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  score: number;
  isUpvoted: boolean;
}

export interface nodeAdd_nodeAdd_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  version: string | null;
  title: string | null;
  description: string | null;
  created_at: any;
  isAutomatable: boolean | null;
  tags: nodeAdd_nodeAdd_data_NodeRoutineList_routines_routine_tags[];
  stars: number;
  score: number;
  isUpvoted: boolean;
}

export interface nodeAdd_nodeAdd_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  title: string;
  description: string | null;
  isOptional: boolean;
  routine: nodeAdd_nodeAdd_data_NodeRoutineList_routines_routine | null;
}

export interface nodeAdd_nodeAdd_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: nodeAdd_nodeAdd_data_NodeRoutineList_routines[];
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

export interface nodeAdd_nodeAdd {
  __typename: "Node";
  id: string;
  created_at: any;
  description: string | null;
  next: string | null;
  previous: string | null;
  title: string;
  type: NodeType;
  updated_at: any;
  data: nodeAdd_nodeAdd_data | null;
}

export interface nodeAdd {
  nodeAdd: nodeAdd_nodeAdd;
}

export interface nodeAddVariables {
  input: NodeInput;
}
