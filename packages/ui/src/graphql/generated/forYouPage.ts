/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ForYouPageInput, RunStatus, NodeType, MemberRole, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL query operation: forYouPage
// ====================================================

export interface forYouPage_forYouPage_activeRuns_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_activeRuns_routine_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_activeRuns_routine_inputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: forYouPage_forYouPage_activeRuns_routine_inputs_standard_tags_translations[];
}

export interface forYouPage_forYouPage_activeRuns_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_activeRuns_routine_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: forYouPage_forYouPage_activeRuns_routine_inputs_standard_tags[];
  translations: forYouPage_forYouPage_activeRuns_routine_inputs_standard_translations[];
}

export interface forYouPage_forYouPage_activeRuns_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: forYouPage_forYouPage_activeRuns_routine_inputs_translations[];
  standard: forYouPage_forYouPage_activeRuns_routine_inputs_standard | null;
}

export interface forYouPage_forYouPage_activeRuns_routine_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface forYouPage_forYouPage_activeRuns_routine_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: forYouPage_forYouPage_activeRuns_routine_nodeLinks_whens_translations[];
}

export interface forYouPage_forYouPage_activeRuns_routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  whens: forYouPage_forYouPage_activeRuns_routine_nodeLinks_whens[];
}

export interface forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeRoutineList_routines_routine_owner = forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization | forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeRoutineList_routines_routine_owner_User;

export interface forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isInternal: boolean | null;
  nodesCount: number | null;
  role: MemberRole | null;
  owner: forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeRoutineList_routines_routine_owner | null;
  simplicity: number;
  translations: forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeRoutineList_routines_routine_translations[];
  version: string | null;
}

export interface forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeRoutineList_routines_routine;
  translations: forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeRoutineList_routines_translations[];
}

export interface forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeRoutineList_routines[];
}

export type forYouPage_forYouPage_activeRuns_routine_nodes_data = forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeEnd | forYouPage_forYouPage_activeRuns_routine_nodes_data_NodeRoutineList;

export interface forYouPage_forYouPage_activeRuns_routine_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface forYouPage_forYouPage_activeRuns_routine_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: forYouPage_forYouPage_activeRuns_routine_nodes_loop_whiles_translations[];
}

export interface forYouPage_forYouPage_activeRuns_routine_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: forYouPage_forYouPage_activeRuns_routine_nodes_loop_whiles[];
}

export interface forYouPage_forYouPage_activeRuns_routine_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface forYouPage_forYouPage_activeRuns_routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: forYouPage_forYouPage_activeRuns_routine_nodes_data | null;
  loop: forYouPage_forYouPage_activeRuns_routine_nodes_loop | null;
  translations: forYouPage_forYouPage_activeRuns_routine_nodes_translations[];
}

export interface forYouPage_forYouPage_activeRuns_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_activeRuns_routine_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_activeRuns_routine_outputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: forYouPage_forYouPage_activeRuns_routine_outputs_standard_tags_translations[];
}

export interface forYouPage_forYouPage_activeRuns_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_activeRuns_routine_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: forYouPage_forYouPage_activeRuns_routine_outputs_standard_tags[];
  translations: forYouPage_forYouPage_activeRuns_routine_outputs_standard_translations[];
}

export interface forYouPage_forYouPage_activeRuns_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: forYouPage_forYouPage_activeRuns_routine_outputs_translations[];
  standard: forYouPage_forYouPage_activeRuns_routine_outputs_standard | null;
}

export interface forYouPage_forYouPage_activeRuns_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface forYouPage_forYouPage_activeRuns_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: forYouPage_forYouPage_activeRuns_routine_owner_Organization_translations[];
}

export interface forYouPage_forYouPage_activeRuns_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type forYouPage_forYouPage_activeRuns_routine_owner = forYouPage_forYouPage_activeRuns_routine_owner_Organization | forYouPage_forYouPage_activeRuns_routine_owner_User;

export interface forYouPage_forYouPage_activeRuns_routine_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface forYouPage_forYouPage_activeRuns_routine_parent {
  __typename: "Routine";
  id: string;
  translations: forYouPage_forYouPage_activeRuns_routine_parent_translations[];
}

export interface forYouPage_forYouPage_activeRuns_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface forYouPage_forYouPage_activeRuns_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface forYouPage_forYouPage_activeRuns_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: forYouPage_forYouPage_activeRuns_routine_resourceLists_resources_translations[];
}

export interface forYouPage_forYouPage_activeRuns_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: forYouPage_forYouPage_activeRuns_routine_resourceLists_translations[];
  resources: forYouPage_forYouPage_activeRuns_routine_resourceLists_resources[];
}

export interface forYouPage_forYouPage_activeRuns_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_activeRuns_routine_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: forYouPage_forYouPage_activeRuns_routine_tags_translations[];
}

export interface forYouPage_forYouPage_activeRuns_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface forYouPage_forYouPage_activeRuns_routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: forYouPage_forYouPage_activeRuns_routine_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isInternal: boolean | null;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: forYouPage_forYouPage_activeRuns_routine_nodeLinks[];
  nodes: forYouPage_forYouPage_activeRuns_routine_nodes[];
  outputs: forYouPage_forYouPage_activeRuns_routine_outputs[];
  owner: forYouPage_forYouPage_activeRuns_routine_owner | null;
  parent: forYouPage_forYouPage_activeRuns_routine_parent | null;
  resourceLists: forYouPage_forYouPage_activeRuns_routine_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  role: MemberRole | null;
  tags: forYouPage_forYouPage_activeRuns_routine_tags[];
  translations: forYouPage_forYouPage_activeRuns_routine_translations[];
  updated_at: any;
  version: string | null;
}

export interface forYouPage_forYouPage_activeRuns {
  __typename: "Run";
  id: string;
  completedComplexity: number;
  pickups: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  routine: forYouPage_forYouPage_activeRuns_routine | null;
}

export interface forYouPage_forYouPage_completedRuns_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_completedRuns_routine_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_completedRuns_routine_inputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: forYouPage_forYouPage_completedRuns_routine_inputs_standard_tags_translations[];
}

export interface forYouPage_forYouPage_completedRuns_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_completedRuns_routine_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: forYouPage_forYouPage_completedRuns_routine_inputs_standard_tags[];
  translations: forYouPage_forYouPage_completedRuns_routine_inputs_standard_translations[];
}

export interface forYouPage_forYouPage_completedRuns_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: forYouPage_forYouPage_completedRuns_routine_inputs_translations[];
  standard: forYouPage_forYouPage_completedRuns_routine_inputs_standard | null;
}

export interface forYouPage_forYouPage_completedRuns_routine_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface forYouPage_forYouPage_completedRuns_routine_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: forYouPage_forYouPage_completedRuns_routine_nodeLinks_whens_translations[];
}

export interface forYouPage_forYouPage_completedRuns_routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  whens: forYouPage_forYouPage_completedRuns_routine_nodeLinks_whens[];
}

export interface forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeRoutineList_routines_routine_owner = forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization | forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeRoutineList_routines_routine_owner_User;

export interface forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isInternal: boolean | null;
  nodesCount: number | null;
  role: MemberRole | null;
  owner: forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeRoutineList_routines_routine_owner | null;
  simplicity: number;
  translations: forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeRoutineList_routines_routine_translations[];
  version: string | null;
}

export interface forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeRoutineList_routines_routine;
  translations: forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeRoutineList_routines_translations[];
}

export interface forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeRoutineList_routines[];
}

export type forYouPage_forYouPage_completedRuns_routine_nodes_data = forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeEnd | forYouPage_forYouPage_completedRuns_routine_nodes_data_NodeRoutineList;

export interface forYouPage_forYouPage_completedRuns_routine_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface forYouPage_forYouPage_completedRuns_routine_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: forYouPage_forYouPage_completedRuns_routine_nodes_loop_whiles_translations[];
}

export interface forYouPage_forYouPage_completedRuns_routine_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: forYouPage_forYouPage_completedRuns_routine_nodes_loop_whiles[];
}

export interface forYouPage_forYouPage_completedRuns_routine_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface forYouPage_forYouPage_completedRuns_routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: forYouPage_forYouPage_completedRuns_routine_nodes_data | null;
  loop: forYouPage_forYouPage_completedRuns_routine_nodes_loop | null;
  translations: forYouPage_forYouPage_completedRuns_routine_nodes_translations[];
}

export interface forYouPage_forYouPage_completedRuns_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_completedRuns_routine_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_completedRuns_routine_outputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: forYouPage_forYouPage_completedRuns_routine_outputs_standard_tags_translations[];
}

export interface forYouPage_forYouPage_completedRuns_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_completedRuns_routine_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: forYouPage_forYouPage_completedRuns_routine_outputs_standard_tags[];
  translations: forYouPage_forYouPage_completedRuns_routine_outputs_standard_translations[];
}

export interface forYouPage_forYouPage_completedRuns_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: forYouPage_forYouPage_completedRuns_routine_outputs_translations[];
  standard: forYouPage_forYouPage_completedRuns_routine_outputs_standard | null;
}

export interface forYouPage_forYouPage_completedRuns_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface forYouPage_forYouPage_completedRuns_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: forYouPage_forYouPage_completedRuns_routine_owner_Organization_translations[];
}

export interface forYouPage_forYouPage_completedRuns_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type forYouPage_forYouPage_completedRuns_routine_owner = forYouPage_forYouPage_completedRuns_routine_owner_Organization | forYouPage_forYouPage_completedRuns_routine_owner_User;

export interface forYouPage_forYouPage_completedRuns_routine_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface forYouPage_forYouPage_completedRuns_routine_parent {
  __typename: "Routine";
  id: string;
  translations: forYouPage_forYouPage_completedRuns_routine_parent_translations[];
}

export interface forYouPage_forYouPage_completedRuns_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface forYouPage_forYouPage_completedRuns_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface forYouPage_forYouPage_completedRuns_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: forYouPage_forYouPage_completedRuns_routine_resourceLists_resources_translations[];
}

export interface forYouPage_forYouPage_completedRuns_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: forYouPage_forYouPage_completedRuns_routine_resourceLists_translations[];
  resources: forYouPage_forYouPage_completedRuns_routine_resourceLists_resources[];
}

export interface forYouPage_forYouPage_completedRuns_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_completedRuns_routine_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: forYouPage_forYouPage_completedRuns_routine_tags_translations[];
}

export interface forYouPage_forYouPage_completedRuns_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface forYouPage_forYouPage_completedRuns_routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: forYouPage_forYouPage_completedRuns_routine_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isInternal: boolean | null;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: forYouPage_forYouPage_completedRuns_routine_nodeLinks[];
  nodes: forYouPage_forYouPage_completedRuns_routine_nodes[];
  outputs: forYouPage_forYouPage_completedRuns_routine_outputs[];
  owner: forYouPage_forYouPage_completedRuns_routine_owner | null;
  parent: forYouPage_forYouPage_completedRuns_routine_parent | null;
  resourceLists: forYouPage_forYouPage_completedRuns_routine_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  role: MemberRole | null;
  tags: forYouPage_forYouPage_completedRuns_routine_tags[];
  translations: forYouPage_forYouPage_completedRuns_routine_translations[];
  updated_at: any;
  version: string | null;
}

export interface forYouPage_forYouPage_completedRuns {
  __typename: "Run";
  id: string;
  completedComplexity: number;
  pickups: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  routine: forYouPage_forYouPage_completedRuns_routine | null;
}

export interface forYouPage_forYouPage_recentlyViewed_to_Organization_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_recentlyViewed_to_Organization_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: forYouPage_forYouPage_recentlyViewed_to_Organization_tags_translations[];
}

export interface forYouPage_forYouPage_recentlyViewed_to_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface forYouPage_forYouPage_recentlyViewed_to_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  stars: number;
  isStarred: boolean;
  role: MemberRole | null;
  tags: forYouPage_forYouPage_recentlyViewed_to_Organization_tags[];
  translations: forYouPage_forYouPage_recentlyViewed_to_Organization_translations[];
}

export interface forYouPage_forYouPage_recentlyViewed_to_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_recentlyViewed_to_Project_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: forYouPage_forYouPage_recentlyViewed_to_Project_tags_translations[];
}

export interface forYouPage_forYouPage_recentlyViewed_to_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface forYouPage_forYouPage_recentlyViewed_to_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  tags: forYouPage_forYouPage_recentlyViewed_to_Project_tags[];
  translations: forYouPage_forYouPage_recentlyViewed_to_Project_translations[];
}

export interface forYouPage_forYouPage_recentlyViewed_to_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_recentlyViewed_to_Routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: forYouPage_forYouPage_recentlyViewed_to_Routine_tags_translations[];
}

export interface forYouPage_forYouPage_recentlyViewed_to_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface forYouPage_forYouPage_recentlyViewed_to_Routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  score: number;
  simplicity: number;
  stars: number;
  tags: forYouPage_forYouPage_recentlyViewed_to_Routine_tags[];
  translations: forYouPage_forYouPage_recentlyViewed_to_Routine_translations[];
  version: string | null;
}

export interface forYouPage_forYouPage_recentlyViewed_to_Standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_recentlyViewed_to_Standard_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: forYouPage_forYouPage_recentlyViewed_to_Standard_tags_translations[];
}

export interface forYouPage_forYouPage_recentlyViewed_to_Standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_recentlyViewed_to_Standard {
  __typename: "Standard";
  id: string;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  name: string;
  role: MemberRole | null;
  tags: forYouPage_forYouPage_recentlyViewed_to_Standard_tags[];
  translations: forYouPage_forYouPage_recentlyViewed_to_Standard_translations[];
}

export interface forYouPage_forYouPage_recentlyViewed_to_User {
  __typename: "User";
  id: string;
  handle: string | null;
  name: string;
  stars: number;
  isStarred: boolean;
}

export type forYouPage_forYouPage_recentlyViewed_to = forYouPage_forYouPage_recentlyViewed_to_Organization | forYouPage_forYouPage_recentlyViewed_to_Project | forYouPage_forYouPage_recentlyViewed_to_Routine | forYouPage_forYouPage_recentlyViewed_to_Standard | forYouPage_forYouPage_recentlyViewed_to_User;

export interface forYouPage_forYouPage_recentlyViewed {
  __typename: "View";
  id: string;
  lastViewed: any;
  title: string;
  to: forYouPage_forYouPage_recentlyViewed_to;
}

export interface forYouPage_forYouPage_recentlyStarred_to_Comment {
  __typename: "Comment" | "Tag";
}

export interface forYouPage_forYouPage_recentlyStarred_to_Organization_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_recentlyStarred_to_Organization_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: forYouPage_forYouPage_recentlyStarred_to_Organization_tags_translations[];
}

export interface forYouPage_forYouPage_recentlyStarred_to_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface forYouPage_forYouPage_recentlyStarred_to_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  stars: number;
  isStarred: boolean;
  role: MemberRole | null;
  tags: forYouPage_forYouPage_recentlyStarred_to_Organization_tags[];
  translations: forYouPage_forYouPage_recentlyStarred_to_Organization_translations[];
}

export interface forYouPage_forYouPage_recentlyStarred_to_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_recentlyStarred_to_Project_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: forYouPage_forYouPage_recentlyStarred_to_Project_tags_translations[];
}

export interface forYouPage_forYouPage_recentlyStarred_to_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface forYouPage_forYouPage_recentlyStarred_to_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  tags: forYouPage_forYouPage_recentlyStarred_to_Project_tags[];
  translations: forYouPage_forYouPage_recentlyStarred_to_Project_translations[];
}

export interface forYouPage_forYouPage_recentlyStarred_to_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_recentlyStarred_to_Routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: forYouPage_forYouPage_recentlyStarred_to_Routine_tags_translations[];
}

export interface forYouPage_forYouPage_recentlyStarred_to_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface forYouPage_forYouPage_recentlyStarred_to_Routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  score: number;
  simplicity: number;
  stars: number;
  tags: forYouPage_forYouPage_recentlyStarred_to_Routine_tags[];
  translations: forYouPage_forYouPage_recentlyStarred_to_Routine_translations[];
  version: string | null;
}

export interface forYouPage_forYouPage_recentlyStarred_to_Standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_recentlyStarred_to_Standard_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: forYouPage_forYouPage_recentlyStarred_to_Standard_tags_translations[];
}

export interface forYouPage_forYouPage_recentlyStarred_to_Standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_recentlyStarred_to_Standard {
  __typename: "Standard";
  id: string;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  name: string;
  role: MemberRole | null;
  tags: forYouPage_forYouPage_recentlyStarred_to_Standard_tags[];
  translations: forYouPage_forYouPage_recentlyStarred_to_Standard_translations[];
}

export interface forYouPage_forYouPage_recentlyStarred_to_User {
  __typename: "User";
  id: string;
  handle: string | null;
  name: string;
  stars: number;
  isStarred: boolean;
}

export type forYouPage_forYouPage_recentlyStarred_to = forYouPage_forYouPage_recentlyStarred_to_Comment | forYouPage_forYouPage_recentlyStarred_to_Organization | forYouPage_forYouPage_recentlyStarred_to_Project | forYouPage_forYouPage_recentlyStarred_to_Routine | forYouPage_forYouPage_recentlyStarred_to_Standard | forYouPage_forYouPage_recentlyStarred_to_User;

export interface forYouPage_forYouPage_recentlyStarred {
  __typename: "Star";
  id: string;
  to: forYouPage_forYouPage_recentlyStarred_to;
}

export interface forYouPage_forYouPage {
  __typename: "ForYouPageResult";
  activeRuns: forYouPage_forYouPage_activeRuns[];
  completedRuns: forYouPage_forYouPage_completedRuns[];
  recentlyViewed: forYouPage_forYouPage_recentlyViewed[];
  recentlyStarred: forYouPage_forYouPage_recentlyStarred[];
}

export interface forYouPage {
  forYouPage: forYouPage_forYouPage;
}

export interface forYouPageVariables {
  input: ForYouPageInput;
}
