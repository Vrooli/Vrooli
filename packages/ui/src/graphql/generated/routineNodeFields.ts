/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeType, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: routineNodeFields
// ====================================================

export interface routineNodeFields_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: routineNodeFields_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type routineNodeFields_data_NodeRoutineList_routines_routine_owner = routineNodeFields_data_NodeRoutineList_routines_routine_owner_Organization | routineNodeFields_data_NodeRoutineList_routines_routine_owner_User;

export interface routineNodeFields_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isInternal: boolean | null;
  nodesCount: number | null;
  role: MemberRole | null;
  owner: routineNodeFields_data_NodeRoutineList_routines_routine_owner | null;
  simplicity: number;
  translations: routineNodeFields_data_NodeRoutineList_routines_routine_translations[];
  version: string | null;
}

export interface routineNodeFields_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineNodeFields_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: routineNodeFields_data_NodeRoutineList_routines_routine;
  translations: routineNodeFields_data_NodeRoutineList_routines_translations[];
}

export interface routineNodeFields_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: routineNodeFields_data_NodeRoutineList_routines[];
}

export type routineNodeFields_data = routineNodeFields_data_NodeEnd | routineNodeFields_data_NodeRoutineList;

export interface routineNodeFields_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineNodeFields_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: routineNodeFields_loop_whiles_translations[];
}

export interface routineNodeFields_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: routineNodeFields_loop_whiles[];
}

export interface routineNodeFields_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineNodeFields {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: routineNodeFields_data | null;
  loop: routineNodeFields_loop | null;
  translations: routineNodeFields_translations[];
}
