/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeType, MemberRole, RunStatus, RunStepStatus } from "./globalTypes";

// ====================================================
// GraphQL fragment: routineRunFields
// ====================================================

export interface routineRunFields_endNode_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface routineRunFields_endNode_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface routineRunFields_endNode_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: routineRunFields_endNode_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface routineRunFields_endNode_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type routineRunFields_endNode_data_NodeRoutineList_routines_routine_owner = routineRunFields_endNode_data_NodeRoutineList_routines_routine_owner_Organization | routineRunFields_endNode_data_NodeRoutineList_routines_routine_owner_User;

export interface routineRunFields_endNode_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface routineRunFields_endNode_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isInternal: boolean | null;
  nodesCount: number | null;
  role: MemberRole | null;
  owner: routineRunFields_endNode_data_NodeRoutineList_routines_routine_owner | null;
  simplicity: number;
  translations: routineRunFields_endNode_data_NodeRoutineList_routines_routine_translations[];
  version: string | null;
}

export interface routineRunFields_endNode_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineRunFields_endNode_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: routineRunFields_endNode_data_NodeRoutineList_routines_routine;
  translations: routineRunFields_endNode_data_NodeRoutineList_routines_translations[];
}

export interface routineRunFields_endNode_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: routineRunFields_endNode_data_NodeRoutineList_routines[];
}

export type routineRunFields_endNode_data = routineRunFields_endNode_data_NodeEnd | routineRunFields_endNode_data_NodeRoutineList;

export interface routineRunFields_endNode_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineRunFields_endNode_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: routineRunFields_endNode_loop_whiles_translations[];
}

export interface routineRunFields_endNode_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: routineRunFields_endNode_loop_whiles[];
}

export interface routineRunFields_endNode_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineRunFields_endNode {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: routineRunFields_endNode_data | null;
  loop: routineRunFields_endNode_loop | null;
  translations: routineRunFields_endNode_translations[];
}

export interface routineRunFields_steps_node {
  __typename: "Node";
  id: string;
}

export interface routineRunFields_steps {
  __typename: "RunStep";
  id: string;
  order: number;
  pickups: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStepStatus;
  node: routineRunFields_steps_node | null;
}

export interface routineRunFields {
  __typename: "Run";
  id: string;
  completedComplexity: number;
  endNode: routineRunFields_endNode | null;
  pickups: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  steps: routineRunFields_steps[];
}
