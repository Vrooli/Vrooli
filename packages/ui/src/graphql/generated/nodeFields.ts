/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole, NodeType } from "./globalTypes";

// ====================================================
// GraphQL fragment: nodeFields
// ====================================================

export interface nodeFields_data_NodeCombine {
  __typename: "NodeCombine";
  id: string;
  from: string[];
}

export interface nodeFields_data_NodeDecision_decisions_when {
  __typename: "NodeDecisionItemWhen";
  id: string;
  condition: string;
}

export interface nodeFields_data_NodeDecision_decisions {
  __typename: "NodeDecisionItem";
  id: string;
  description: string | null;
  title: string;
  toId: string | null;
  when: nodeFields_data_NodeDecision_decisions_when[];
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
  isStarred: boolean;
}

export interface nodeFields_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  version: string | null;
  title: string | null;
  description: string | null;
  created_at: any;
  isAutomatable: boolean | null;
  role: MemberRole | null;
  tags: nodeFields_data_NodeRoutineList_routines_routine_tags[];
  stars: number;
  isStarred: boolean;
  score: number;
  isUpvoted: boolean | null;
}

export interface nodeFields_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  title: string | null;
  description: string | null;
  isOptional: boolean;
  routine: nodeFields_data_NodeRoutineList_routines_routine;
}

export interface nodeFields_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: nodeFields_data_NodeRoutineList_routines[];
}

export type nodeFields_data = nodeFields_data_NodeCombine | nodeFields_data_NodeDecision | nodeFields_data_NodeEnd | nodeFields_data_NodeLoop | nodeFields_data_NodeRoutineList;

export interface nodeFields {
  __typename: "Node";
  id: string;
  created_at: any;
  description: string | null;
  role: MemberRole | null;
  next: string | null;
  previous: string | null;
  title: string;
  type: NodeType;
  updated_at: any;
  data: nodeFields_data | null;
}
