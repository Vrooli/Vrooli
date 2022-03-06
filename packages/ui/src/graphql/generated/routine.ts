/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdInput, MemberRole, NodeType } from "./globalTypes";

// ====================================================
// GraphQL query operation: routine
// ====================================================

export interface routine_routine_inputs_standard_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface routine_routine_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  description: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: routine_routine_inputs_standard_tags[];
}

export interface routine_routine_inputs {
  __typename: "InputItem";
  id: string;
  standard: routine_routine_inputs_standard | null;
}

export interface routine_routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface routine_routine_nodes_data_NodeLoop {
  __typename: "NodeLoop";
  id: string;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  isInternal: boolean | null;
  role: MemberRole | null;
  title: string | null;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  title: string | null;
  description: string | null;
  isOptional: boolean;
  routine: routine_routine_nodes_data_NodeRoutineList_routines_routine;
}

export interface routine_routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: routine_routine_nodes_data_NodeRoutineList_routines[];
}

export type routine_routine_nodes_data = routine_routine_nodes_data_NodeEnd | routine_routine_nodes_data_NodeLoop | routine_routine_nodes_data_NodeRoutineList;

export interface routine_routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  description: string | null;
  rowIndex: number | null;
  title: string;
  type: NodeType;
  updated_at: any;
  data: routine_routine_nodes_data | null;
}

export interface routine_routine_nodeLinks_conditions_when {
  __typename: "NodeLinkConditionCase";
  id: string;
  condition: string;
}

export interface routine_routine_nodeLinks_conditions {
  __typename: "NodeLinkCondition";
  id: string;
  description: string | null;
  title: string;
  when: routine_routine_nodeLinks_conditions_when[];
}

export interface routine_routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  conditions: routine_routine_nodeLinks_conditions[];
}

export interface routine_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface routine_routine_owner_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type routine_routine_owner = routine_routine_owner_Organization | routine_routine_owner_User;

export interface routine_routine_outputs_standard_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface routine_routine_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  description: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: routine_routine_outputs_standard_tags[];
}

export interface routine_routine_outputs {
  __typename: "OutputItem";
  id: string;
  standard: routine_routine_outputs_standard | null;
}

export interface routine_routine_parent {
  __typename: "Routine";
  id: string;
  title: string | null;
}

export interface routine_routine_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  description: string | null;
  link: string;
  title: string | null;
  updated_at: any;
}

export interface routine_routine_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface routine_routine {
  __typename: "Routine";
  id: string;
  created_at: any;
  instructions: string | null;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  title: string | null;
  description: string | null;
  updated_at: any;
  version: string | null;
  stars: number;
  score: number;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  isStarred: boolean;
  inputs: routine_routine_inputs[];
  nodes: routine_routine_nodes[];
  nodeLinks: routine_routine_nodeLinks[];
  owner: routine_routine_owner | null;
  outputs: routine_routine_outputs[];
  parent: routine_routine_parent | null;
  resources: routine_routine_resources[];
  tags: routine_routine_tags[];
}

export interface routine {
  routine: routine_routine | null;
}

export interface routineVariables {
  input: FindByIdInput;
}
