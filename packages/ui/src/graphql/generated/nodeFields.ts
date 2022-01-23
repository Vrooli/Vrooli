/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeType } from "./globalTypes";

// ====================================================
// GraphQL fragment: nodeFields
// ====================================================

export interface nodeFields_data_NodeCombine {
  __typename: "NodeCombine";
  id: string;
  from: string[];
  to: string;
}

export interface nodeFields_data_NodeDecision_decisions_when {
  __typename: "NodeDecisionItemCase";
  id: string;
  condition: string;
}

export interface nodeFields_data_NodeDecision_decisions {
  __typename: "NodeDecisionItem";
  id: string;
  description: string | null;
  title: string;
  toId: string | null;
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
  wasSuccessful: boolean;
}

export interface nodeFields_data_NodeLoop {
  __typename: "NodeLoop";
  id: string;
}

export interface nodeFields_data_NodeRoutineList_routines_routine_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean | null;
  score: number;
  isUpvoted: boolean | null;
}

export interface nodeFields_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  version: string | null;
  title: string | null;
  description: string | null;
  created_at: any;
  isAutomatable: boolean | null;
  tags: nodeFields_data_NodeRoutineList_routines_routine_tags[];
  stars: number;
  isStarred: boolean | null;
  score: number;
  isUpvoted: boolean | null;
}

export interface nodeFields_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  title: string;
  description: string | null;
  isOptional: boolean;
  routine: nodeFields_data_NodeRoutineList_routines_routine | null;
}

export interface nodeFields_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: nodeFields_data_NodeRoutineList_routines[];
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

export interface nodeFields {
  __typename: "Node";
  id: string;
  created_at: any;
  description: string | null;
  next: string | null;
  previous: string | null;
  title: string;
  type: NodeType;
  updated_at: any;
  data: nodeFields_data | null;
}
