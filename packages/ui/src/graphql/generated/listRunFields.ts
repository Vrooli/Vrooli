/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunStatus, NodeType, MemberRole, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL fragment: listRunFields
// ====================================================

export interface listRunFields_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listRunFields_routine_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listRunFields_routine_inputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: listRunFields_routine_inputs_standard_tags_translations[];
}

export interface listRunFields_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listRunFields_routine_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: listRunFields_routine_inputs_standard_tags[];
  translations: listRunFields_routine_inputs_standard_translations[];
}

export interface listRunFields_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: listRunFields_routine_inputs_translations[];
  standard: listRunFields_routine_inputs_standard | null;
}

export interface listRunFields_routine_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface listRunFields_routine_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: listRunFields_routine_nodeLinks_whens_translations[];
}

export interface listRunFields_routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  whens: listRunFields_routine_nodeLinks_whens[];
}

export interface listRunFields_routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface listRunFields_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface listRunFields_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: listRunFields_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface listRunFields_routine_nodes_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type listRunFields_routine_nodes_data_NodeRoutineList_routines_routine_owner = listRunFields_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization | listRunFields_routine_nodes_data_NodeRoutineList_routines_routine_owner_User;

export interface listRunFields_routine_nodes_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface listRunFields_routine_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isInternal: boolean | null;
  nodesCount: number | null;
  role: MemberRole | null;
  owner: listRunFields_routine_nodes_data_NodeRoutineList_routines_routine_owner | null;
  simplicity: number;
  translations: listRunFields_routine_nodes_data_NodeRoutineList_routines_routine_translations[];
  version: string | null;
}

export interface listRunFields_routine_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface listRunFields_routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: listRunFields_routine_nodes_data_NodeRoutineList_routines_routine;
  translations: listRunFields_routine_nodes_data_NodeRoutineList_routines_translations[];
}

export interface listRunFields_routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: listRunFields_routine_nodes_data_NodeRoutineList_routines[];
}

export type listRunFields_routine_nodes_data = listRunFields_routine_nodes_data_NodeEnd | listRunFields_routine_nodes_data_NodeRoutineList;

export interface listRunFields_routine_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface listRunFields_routine_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: listRunFields_routine_nodes_loop_whiles_translations[];
}

export interface listRunFields_routine_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: listRunFields_routine_nodes_loop_whiles[];
}

export interface listRunFields_routine_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface listRunFields_routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: listRunFields_routine_nodes_data | null;
  loop: listRunFields_routine_nodes_loop | null;
  translations: listRunFields_routine_nodes_translations[];
}

export interface listRunFields_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listRunFields_routine_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listRunFields_routine_outputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: listRunFields_routine_outputs_standard_tags_translations[];
}

export interface listRunFields_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listRunFields_routine_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: listRunFields_routine_outputs_standard_tags[];
  translations: listRunFields_routine_outputs_standard_translations[];
}

export interface listRunFields_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: listRunFields_routine_outputs_translations[];
  standard: listRunFields_routine_outputs_standard | null;
}

export interface listRunFields_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface listRunFields_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: listRunFields_routine_owner_Organization_translations[];
}

export interface listRunFields_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type listRunFields_routine_owner = listRunFields_routine_owner_Organization | listRunFields_routine_owner_User;

export interface listRunFields_routine_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface listRunFields_routine_parent {
  __typename: "Routine";
  id: string;
  translations: listRunFields_routine_parent_translations[];
}

export interface listRunFields_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface listRunFields_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface listRunFields_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: listRunFields_routine_resourceLists_resources_translations[];
}

export interface listRunFields_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: listRunFields_routine_resourceLists_translations[];
  resources: listRunFields_routine_resourceLists_resources[];
}

export interface listRunFields_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listRunFields_routine_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: listRunFields_routine_tags_translations[];
}

export interface listRunFields_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface listRunFields_routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: listRunFields_routine_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isInternal: boolean | null;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: listRunFields_routine_nodeLinks[];
  nodes: listRunFields_routine_nodes[];
  outputs: listRunFields_routine_outputs[];
  owner: listRunFields_routine_owner | null;
  parent: listRunFields_routine_parent | null;
  resourceLists: listRunFields_routine_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  role: MemberRole | null;
  tags: listRunFields_routine_tags[];
  translations: listRunFields_routine_translations[];
  updated_at: any;
  version: string | null;
}

export interface listRunFields {
  __typename: "Run";
  id: string;
  completedComplexity: number;
  pickups: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  routine: listRunFields_routine | null;
}
