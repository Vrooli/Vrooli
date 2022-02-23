/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeUpdateInput, NodeType, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: nodeUpdate
// ====================================================

export interface nodeUpdate_nodeUpdate_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface nodeUpdate_nodeUpdate_data_NodeLoop {
  __typename: "NodeLoop";
  id: string;
}

export interface nodeUpdate_nodeUpdate_data_NodeRoutineList_routines_routine_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface nodeUpdate_nodeUpdate_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  version: string | null;
  title: string | null;
  description: string | null;
  created_at: any;
  isAutomatable: boolean | null;
  role: MemberRole | null;
  tags: nodeUpdate_nodeUpdate_data_NodeRoutineList_routines_routine_tags[];
}

export interface nodeUpdate_nodeUpdate_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  title: string | null;
  description: string | null;
  isOptional: boolean;
  routine: nodeUpdate_nodeUpdate_data_NodeRoutineList_routines_routine;
}

export interface nodeUpdate_nodeUpdate_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: nodeUpdate_nodeUpdate_data_NodeRoutineList_routines[];
}

export type nodeUpdate_nodeUpdate_data = nodeUpdate_nodeUpdate_data_NodeEnd | nodeUpdate_nodeUpdate_data_NodeLoop | nodeUpdate_nodeUpdate_data_NodeRoutineList;

export interface nodeUpdate_nodeUpdate {
  __typename: "Node";
  id: string;
  created_at: any;
  description: string | null;
  title: string;
  type: NodeType;
  updated_at: any;
  data: nodeUpdate_nodeUpdate_data | null;
}

export interface nodeUpdate {
  nodeUpdate: nodeUpdate_nodeUpdate;
}

export interface nodeUpdateVariables {
  input: NodeUpdateInput;
}
