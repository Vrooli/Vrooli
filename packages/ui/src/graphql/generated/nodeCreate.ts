/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeCreateInput, MemberRole, NodeType } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: nodeCreate
// ====================================================

export interface nodeCreate_nodeCreate_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface nodeCreate_nodeCreate_data_NodeLoop {
  __typename: "NodeLoop";
  id: string;
}

export interface nodeCreate_nodeCreate_data_NodeRoutineList_routines_routine_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean;
}

export interface nodeCreate_nodeCreate_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  version: string | null;
  title: string | null;
  description: string | null;
  created_at: any;
  isAutomatable: boolean | null;
  role: MemberRole | null;
  tags: nodeCreate_nodeCreate_data_NodeRoutineList_routines_routine_tags[];
  stars: number;
  isStarred: boolean;
  score: number;
  isUpvoted: boolean | null;
}

export interface nodeCreate_nodeCreate_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  title: string | null;
  description: string | null;
  isOptional: boolean;
  routine: nodeCreate_nodeCreate_data_NodeRoutineList_routines_routine;
}

export interface nodeCreate_nodeCreate_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: nodeCreate_nodeCreate_data_NodeRoutineList_routines[];
}

export type nodeCreate_nodeCreate_data = nodeCreate_nodeCreate_data_NodeEnd | nodeCreate_nodeCreate_data_NodeLoop | nodeCreate_nodeCreate_data_NodeRoutineList;

export interface nodeCreate_nodeCreate {
  __typename: "Node";
  id: string;
  created_at: any;
  description: string | null;
  role: MemberRole | null;
  title: string;
  type: NodeType;
  updated_at: any;
  data: nodeCreate_nodeCreate_data | null;
}

export interface nodeCreate {
  nodeCreate: nodeCreate_nodeCreate;
}

export interface nodeCreateVariables {
  input: NodeCreateInput;
}
