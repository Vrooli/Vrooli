/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole, NodeType } from "./globalTypes";

// ====================================================
// GraphQL fragment: deepRoutineFields
// ====================================================

export interface deepRoutineFields_inputs_standard_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface deepRoutineFields_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  description: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: deepRoutineFields_inputs_standard_tags[];
}

export interface deepRoutineFields_inputs {
  __typename: "InputItem";
  id: string;
  standard: deepRoutineFields_inputs_standard | null;
}

export interface deepRoutineFields_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface deepRoutineFields_nodes_data_NodeLoop {
  __typename: "NodeLoop";
  id: string;
}

export interface deepRoutineFields_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  isInternal: boolean | null;
  role: MemberRole | null;
  title: string | null;
}

export interface deepRoutineFields_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  title: string | null;
  description: string | null;
  isOptional: boolean;
  routine: deepRoutineFields_nodes_data_NodeRoutineList_routines_routine;
}

export interface deepRoutineFields_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: deepRoutineFields_nodes_data_NodeRoutineList_routines[];
}

export type deepRoutineFields_nodes_data = deepRoutineFields_nodes_data_NodeEnd | deepRoutineFields_nodes_data_NodeLoop | deepRoutineFields_nodes_data_NodeRoutineList;

export interface deepRoutineFields_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  description: string | null;
  rowIndex: number | null;
  title: string;
  type: NodeType;
  updated_at: any;
  data: deepRoutineFields_nodes_data | null;
}

export interface deepRoutineFields_nodeLinks_conditions_when {
  __typename: "NodeLinkConditionCase";
  id: string;
  condition: string;
}

export interface deepRoutineFields_nodeLinks_conditions {
  __typename: "NodeLinkCondition";
  id: string;
  description: string | null;
  title: string;
  when: deepRoutineFields_nodeLinks_conditions_when[];
}

export interface deepRoutineFields_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  conditions: deepRoutineFields_nodeLinks_conditions[];
}

export interface deepRoutineFields_owner_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface deepRoutineFields_owner_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type deepRoutineFields_owner = deepRoutineFields_owner_Organization | deepRoutineFields_owner_User;

export interface deepRoutineFields_outputs_standard_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface deepRoutineFields_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  description: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: deepRoutineFields_outputs_standard_tags[];
}

export interface deepRoutineFields_outputs {
  __typename: "OutputItem";
  id: string;
  standard: deepRoutineFields_outputs_standard | null;
}

export interface deepRoutineFields_parent {
  __typename: "Routine";
  id: string;
  title: string | null;
}

export interface deepRoutineFields_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  description: string | null;
  link: string;
  title: string | null;
  updated_at: any;
}

export interface deepRoutineFields_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface deepRoutineFields {
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
  inputs: deepRoutineFields_inputs[];
  nodes: deepRoutineFields_nodes[];
  nodeLinks: deepRoutineFields_nodeLinks[];
  owner: deepRoutineFields_owner | null;
  outputs: deepRoutineFields_outputs[];
  parent: deepRoutineFields_parent | null;
  resources: deepRoutineFields_resources[];
  tags: deepRoutineFields_tags[];
}
