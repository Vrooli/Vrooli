/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeType, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: deepRoutineNodeFields
// ====================================================

export interface deepRoutineNodeFields_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface deepRoutineNodeFields_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface deepRoutineNodeFields_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  translations: deepRoutineNodeFields_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface deepRoutineNodeFields_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type deepRoutineNodeFields_data_NodeRoutineList_routines_routine_owner = deepRoutineNodeFields_data_NodeRoutineList_routines_routine_owner_Organization | deepRoutineNodeFields_data_NodeRoutineList_routines_routine_owner_User;

export interface deepRoutineNodeFields_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface deepRoutineNodeFields_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isInternal: boolean | null;
  nodesCount: number | null;
  role: MemberRole | null;
  owner: deepRoutineNodeFields_data_NodeRoutineList_routines_routine_owner | null;
  simplicity: number;
  translations: deepRoutineNodeFields_data_NodeRoutineList_routines_routine_translations[];
  version: string | null;
}

export interface deepRoutineNodeFields_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface deepRoutineNodeFields_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  isOptional: boolean;
  routine: deepRoutineNodeFields_data_NodeRoutineList_routines_routine;
  translations: deepRoutineNodeFields_data_NodeRoutineList_routines_translations[];
}

export interface deepRoutineNodeFields_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: deepRoutineNodeFields_data_NodeRoutineList_routines[];
}

export type deepRoutineNodeFields_data = deepRoutineNodeFields_data_NodeEnd | deepRoutineNodeFields_data_NodeRoutineList;

export interface deepRoutineNodeFields_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface deepRoutineNodeFields_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: deepRoutineNodeFields_loop_whiles_translations[];
}

export interface deepRoutineNodeFields_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: deepRoutineNodeFields_loop_whiles[];
}

export interface deepRoutineNodeFields_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface deepRoutineNodeFields {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: deepRoutineNodeFields_data | null;
  loop: deepRoutineNodeFields_loop | null;
  translations: deepRoutineNodeFields_translations[];
}
