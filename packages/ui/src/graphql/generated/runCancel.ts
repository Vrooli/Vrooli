/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunCancelInput, NodeType, MemberRole, RunStatus, ResourceListUsedFor, ResourceUsedFor, RunStepStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: runCancel
// ====================================================

export interface runCancel_runCancel_endNode_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface runCancel_runCancel_endNode_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runCancel_runCancel_endNode_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runCancel_runCancel_endNode_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface runCancel_runCancel_endNode_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runCancel_runCancel_endNode_data_NodeRoutineList_routines_routine_owner = runCancel_runCancel_endNode_data_NodeRoutineList_routines_routine_owner_Organization | runCancel_runCancel_endNode_data_NodeRoutineList_routines_routine_owner_User;

export interface runCancel_runCancel_endNode_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface runCancel_runCancel_endNode_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isInternal: boolean | null;
  nodesCount: number | null;
  role: MemberRole | null;
  owner: runCancel_runCancel_endNode_data_NodeRoutineList_routines_routine_owner | null;
  simplicity: number;
  translations: runCancel_runCancel_endNode_data_NodeRoutineList_routines_routine_translations[];
  version: string | null;
}

export interface runCancel_runCancel_endNode_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runCancel_runCancel_endNode_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: runCancel_runCancel_endNode_data_NodeRoutineList_routines_routine;
  translations: runCancel_runCancel_endNode_data_NodeRoutineList_routines_translations[];
}

export interface runCancel_runCancel_endNode_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: runCancel_runCancel_endNode_data_NodeRoutineList_routines[];
}

export type runCancel_runCancel_endNode_data = runCancel_runCancel_endNode_data_NodeEnd | runCancel_runCancel_endNode_data_NodeRoutineList;

export interface runCancel_runCancel_endNode_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runCancel_runCancel_endNode_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: runCancel_runCancel_endNode_loop_whiles_translations[];
}

export interface runCancel_runCancel_endNode_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: runCancel_runCancel_endNode_loop_whiles[];
}

export interface runCancel_runCancel_endNode_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runCancel_runCancel_endNode {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: runCancel_runCancel_endNode_data | null;
  loop: runCancel_runCancel_endNode_loop | null;
  translations: runCancel_runCancel_endNode_translations[];
}

export interface runCancel_runCancel_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCancel_runCancel_routine_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCancel_runCancel_routine_inputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: runCancel_runCancel_routine_inputs_standard_tags_translations[];
}

export interface runCancel_runCancel_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCancel_runCancel_routine_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: runCancel_runCancel_routine_inputs_standard_tags[];
  translations: runCancel_runCancel_routine_inputs_standard_translations[];
}

export interface runCancel_runCancel_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runCancel_runCancel_routine_inputs_translations[];
  standard: runCancel_runCancel_routine_inputs_standard | null;
}

export interface runCancel_runCancel_routine_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runCancel_runCancel_routine_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: runCancel_runCancel_routine_nodeLinks_whens_translations[];
}

export interface runCancel_runCancel_routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  whens: runCancel_runCancel_routine_nodeLinks_whens[];
}

export interface runCancel_runCancel_routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface runCancel_runCancel_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runCancel_runCancel_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runCancel_runCancel_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface runCancel_runCancel_routine_nodes_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runCancel_runCancel_routine_nodes_data_NodeRoutineList_routines_routine_owner = runCancel_runCancel_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization | runCancel_runCancel_routine_nodes_data_NodeRoutineList_routines_routine_owner_User;

export interface runCancel_runCancel_routine_nodes_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface runCancel_runCancel_routine_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isInternal: boolean | null;
  nodesCount: number | null;
  role: MemberRole | null;
  owner: runCancel_runCancel_routine_nodes_data_NodeRoutineList_routines_routine_owner | null;
  simplicity: number;
  translations: runCancel_runCancel_routine_nodes_data_NodeRoutineList_routines_routine_translations[];
  version: string | null;
}

export interface runCancel_runCancel_routine_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runCancel_runCancel_routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: runCancel_runCancel_routine_nodes_data_NodeRoutineList_routines_routine;
  translations: runCancel_runCancel_routine_nodes_data_NodeRoutineList_routines_translations[];
}

export interface runCancel_runCancel_routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: runCancel_runCancel_routine_nodes_data_NodeRoutineList_routines[];
}

export type runCancel_runCancel_routine_nodes_data = runCancel_runCancel_routine_nodes_data_NodeEnd | runCancel_runCancel_routine_nodes_data_NodeRoutineList;

export interface runCancel_runCancel_routine_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runCancel_runCancel_routine_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: runCancel_runCancel_routine_nodes_loop_whiles_translations[];
}

export interface runCancel_runCancel_routine_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: runCancel_runCancel_routine_nodes_loop_whiles[];
}

export interface runCancel_runCancel_routine_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runCancel_runCancel_routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: runCancel_runCancel_routine_nodes_data | null;
  loop: runCancel_runCancel_routine_nodes_loop | null;
  translations: runCancel_runCancel_routine_nodes_translations[];
}

export interface runCancel_runCancel_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCancel_runCancel_routine_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCancel_runCancel_routine_outputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: runCancel_runCancel_routine_outputs_standard_tags_translations[];
}

export interface runCancel_runCancel_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCancel_runCancel_routine_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: runCancel_runCancel_routine_outputs_standard_tags[];
  translations: runCancel_runCancel_routine_outputs_standard_translations[];
}

export interface runCancel_runCancel_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runCancel_runCancel_routine_outputs_translations[];
  standard: runCancel_runCancel_routine_outputs_standard | null;
}

export interface runCancel_runCancel_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runCancel_runCancel_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runCancel_runCancel_routine_owner_Organization_translations[];
}

export interface runCancel_runCancel_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runCancel_runCancel_routine_owner = runCancel_runCancel_routine_owner_Organization | runCancel_runCancel_routine_owner_User;

export interface runCancel_runCancel_routine_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface runCancel_runCancel_routine_parent {
  __typename: "Routine";
  id: string;
  translations: runCancel_runCancel_routine_parent_translations[];
}

export interface runCancel_runCancel_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runCancel_runCancel_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runCancel_runCancel_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runCancel_runCancel_routine_resourceLists_resources_translations[];
}

export interface runCancel_runCancel_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runCancel_runCancel_routine_resourceLists_translations[];
  resources: runCancel_runCancel_routine_resourceLists_resources[];
}

export interface runCancel_runCancel_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCancel_runCancel_routine_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: runCancel_runCancel_routine_tags_translations[];
}

export interface runCancel_runCancel_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface runCancel_runCancel_routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: runCancel_runCancel_routine_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isInternal: boolean | null;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: runCancel_runCancel_routine_nodeLinks[];
  nodes: runCancel_runCancel_routine_nodes[];
  outputs: runCancel_runCancel_routine_outputs[];
  owner: runCancel_runCancel_routine_owner | null;
  parent: runCancel_runCancel_routine_parent | null;
  resourceLists: runCancel_runCancel_routine_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  role: MemberRole | null;
  tags: runCancel_runCancel_routine_tags[];
  translations: runCancel_runCancel_routine_translations[];
  updated_at: any;
  version: string | null;
}

export interface runCancel_runCancel_steps_node {
  __typename: "Node";
  id: string;
}

export interface runCancel_runCancel_steps {
  __typename: "RunStep";
  id: string;
  order: number;
  pickups: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStepStatus;
  node: runCancel_runCancel_steps_node | null;
}

export interface runCancel_runCancel {
  __typename: "Run";
  id: string;
  completedComplexity: number;
  endNode: runCancel_runCancel_endNode | null;
  pickups: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  routine: runCancel_runCancel_routine | null;
  steps: runCancel_runCancel_steps[];
}

export interface runCancel {
  runCancel: runCancel_runCancel;
}

export interface runCancelVariables {
  input: RunCancelInput;
}
