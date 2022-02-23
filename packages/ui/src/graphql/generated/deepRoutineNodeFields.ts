/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeType } from "./globalTypes";

// ====================================================
// GraphQL fragment: deepRoutineNodeFields
// ====================================================

export interface deepRoutineNodeFields_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface deepRoutineNodeFields_data_NodeLoop {
  __typename: "NodeLoop";
  id: string;
}

export interface deepRoutineNodeFields_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  title: string | null;
}

export interface deepRoutineNodeFields_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  title: string | null;
  description: string | null;
  isOptional: boolean;
  routine: deepRoutineNodeFields_data_NodeRoutineList_routines_routine;
}

export interface deepRoutineNodeFields_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: deepRoutineNodeFields_data_NodeRoutineList_routines[];
}

export type deepRoutineNodeFields_data = deepRoutineNodeFields_data_NodeEnd | deepRoutineNodeFields_data_NodeLoop | deepRoutineNodeFields_data_NodeRoutineList;

export interface deepRoutineNodeFields {
  __typename: "Node";
  id: string;
  created_at: any;
  description: string | null;
  title: string;
  type: NodeType;
  updated_at: any;
  data: deepRoutineNodeFields_data | null;
}
