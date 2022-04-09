/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeType, MemberRole, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL fragment: deepRoutineFields
// ====================================================

export interface deepRoutineFields_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface deepRoutineFields_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface deepRoutineFields_inputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: deepRoutineFields_inputs_standard_tags_translations[];
}

export interface deepRoutineFields_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface deepRoutineFields_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: deepRoutineFields_inputs_standard_tags[];
  translations: deepRoutineFields_inputs_standard_translations[];
}

export interface deepRoutineFields_inputs {
  __typename: "InputItem";
  id: string;
  name: string | null;
  translations: deepRoutineFields_inputs_translations[];
  standard: deepRoutineFields_inputs_standard | null;
}

export interface deepRoutineFields_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface deepRoutineFields_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: deepRoutineFields_nodeLinks_whens_translations[];
}

export interface deepRoutineFields_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  whens: deepRoutineFields_nodeLinks_whens[];
}

export interface deepRoutineFields_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface deepRoutineFields_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface deepRoutineFields_nodes_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: deepRoutineFields_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface deepRoutineFields_nodes_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type deepRoutineFields_nodes_data_NodeRoutineList_routines_routine_owner = deepRoutineFields_nodes_data_NodeRoutineList_routines_routine_owner_Organization | deepRoutineFields_nodes_data_NodeRoutineList_routines_routine_owner_User;

export interface deepRoutineFields_nodes_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface deepRoutineFields_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isInternal: boolean | null;
  nodesCount: number | null;
  role: MemberRole | null;
  owner: deepRoutineFields_nodes_data_NodeRoutineList_routines_routine_owner | null;
  simplicity: number;
  translations: deepRoutineFields_nodes_data_NodeRoutineList_routines_routine_translations[];
  version: string | null;
}

export interface deepRoutineFields_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface deepRoutineFields_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: deepRoutineFields_nodes_data_NodeRoutineList_routines_routine;
  translations: deepRoutineFields_nodes_data_NodeRoutineList_routines_translations[];
}

export interface deepRoutineFields_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: deepRoutineFields_nodes_data_NodeRoutineList_routines[];
}

export type deepRoutineFields_nodes_data = deepRoutineFields_nodes_data_NodeEnd | deepRoutineFields_nodes_data_NodeRoutineList;

export interface deepRoutineFields_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface deepRoutineFields_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: deepRoutineFields_nodes_loop_whiles_translations[];
}

export interface deepRoutineFields_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: deepRoutineFields_nodes_loop_whiles[];
}

export interface deepRoutineFields_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface deepRoutineFields_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: deepRoutineFields_nodes_data | null;
  loop: deepRoutineFields_nodes_loop | null;
  translations: deepRoutineFields_nodes_translations[];
}

export interface deepRoutineFields_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface deepRoutineFields_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface deepRoutineFields_outputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: deepRoutineFields_outputs_standard_tags_translations[];
}

export interface deepRoutineFields_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface deepRoutineFields_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: deepRoutineFields_outputs_standard_tags[];
  translations: deepRoutineFields_outputs_standard_translations[];
}

export interface deepRoutineFields_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: deepRoutineFields_outputs_translations[];
  standard: deepRoutineFields_outputs_standard | null;
}

export interface deepRoutineFields_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface deepRoutineFields_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: deepRoutineFields_owner_Organization_translations[];
}

export interface deepRoutineFields_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type deepRoutineFields_owner = deepRoutineFields_owner_Organization | deepRoutineFields_owner_User;

export interface deepRoutineFields_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface deepRoutineFields_parent {
  __typename: "Routine";
  id: string;
  translations: deepRoutineFields_parent_translations[];
}

export interface deepRoutineFields_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface deepRoutineFields_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface deepRoutineFields_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: deepRoutineFields_resourceLists_resources_translations[];
}

export interface deepRoutineFields_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: deepRoutineFields_resourceLists_translations[];
  resources: deepRoutineFields_resourceLists_resources[];
}

export interface deepRoutineFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface deepRoutineFields_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: deepRoutineFields_tags_translations[];
}

export interface deepRoutineFields_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface deepRoutineFields {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: deepRoutineFields_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isInternal: boolean | null;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: deepRoutineFields_nodeLinks[];
  nodes: deepRoutineFields_nodes[];
  outputs: deepRoutineFields_outputs[];
  owner: deepRoutineFields_owner | null;
  parent: deepRoutineFields_parent | null;
  resourceLists: deepRoutineFields_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  role: MemberRole | null;
  tags: deepRoutineFields_tags[];
  translations: deepRoutineFields_translations[];
  updated_at: any;
  version: string | null;
}
