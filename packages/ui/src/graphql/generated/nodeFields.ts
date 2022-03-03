/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeType, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: nodeFields
// ====================================================

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
  description: string | null;
  tag: string;
}

export interface nodeFields_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  version: string | null;
  title: string | null;
  description: string | null;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  role: MemberRole | null;
  tags: nodeFields_data_NodeRoutineList_routines_routine_tags[];
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

export type nodeFields_data = nodeFields_data_NodeEnd | nodeFields_data_NodeLoop | nodeFields_data_NodeRoutineList;

export interface nodeFields {
  __typename: "Node";
  id: string;
  created_at: any;
  description: string | null;
  title: string;
  type: NodeType;
  updated_at: any;
  data: nodeFields_data | null;
}
