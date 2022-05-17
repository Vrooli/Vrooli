/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeType, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: runNodeFields
// ====================================================

export interface runNodeFields_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface runNodeFields_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runNodeFields_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runNodeFields_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface runNodeFields_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runNodeFields_data_NodeRoutineList_routines_routine_owner = runNodeFields_data_NodeRoutineList_routines_routine_owner_Organization | runNodeFields_data_NodeRoutineList_routines_routine_owner_User;

export interface runNodeFields_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface runNodeFields_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isInternal: boolean | null;
  nodesCount: number | null;
  role: MemberRole | null;
  owner: runNodeFields_data_NodeRoutineList_routines_routine_owner | null;
  simplicity: number;
  translations: runNodeFields_data_NodeRoutineList_routines_routine_translations[];
  version: string | null;
}

export interface runNodeFields_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runNodeFields_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: runNodeFields_data_NodeRoutineList_routines_routine;
  translations: runNodeFields_data_NodeRoutineList_routines_translations[];
}

export interface runNodeFields_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: runNodeFields_data_NodeRoutineList_routines[];
}

export type runNodeFields_data = runNodeFields_data_NodeEnd | runNodeFields_data_NodeRoutineList;

export interface runNodeFields_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runNodeFields_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: runNodeFields_loop_whiles_translations[];
}

export interface runNodeFields_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: runNodeFields_loop_whiles[];
}

export interface runNodeFields_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runNodeFields {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: runNodeFields_data | null;
  loop: runNodeFields_loop | null;
  translations: runNodeFields_translations[];
}
