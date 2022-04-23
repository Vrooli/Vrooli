/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole, ResourceListUsedFor, ResourceUsedFor, NodeType } from "./globalTypes";

// ====================================================
// GraphQL query operation: developPage
// ====================================================

export interface developPage_developPage_completed_Project_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface developPage_developPage_completed_Project_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface developPage_developPage_completed_Project_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: developPage_developPage_completed_Project_resourceLists_resources_translations[];
}

export interface developPage_developPage_completed_Project_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: developPage_developPage_completed_Project_resourceLists_translations[];
  resources: developPage_developPage_completed_Project_resourceLists_resources[];
}

export interface developPage_developPage_completed_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_completed_Project_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: developPage_developPage_completed_Project_tags_translations[];
}

export interface developPage_developPage_completed_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface developPage_developPage_completed_Project_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface developPage_developPage_completed_Project_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: developPage_developPage_completed_Project_owner_Organization_translations[];
}

export interface developPage_developPage_completed_Project_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type developPage_developPage_completed_Project_owner = developPage_developPage_completed_Project_owner_Organization | developPage_developPage_completed_Project_owner_User;

export interface developPage_developPage_completed_Project {
  __typename: "Project";
  id: string;
  completedAt: any | null;
  created_at: any;
  handle: string | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  resourceLists: developPage_developPage_completed_Project_resourceLists[] | null;
  tags: developPage_developPage_completed_Project_tags[];
  translations: developPage_developPage_completed_Project_translations[];
  owner: developPage_developPage_completed_Project_owner | null;
}

export interface developPage_developPage_completed_Routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_completed_Routine_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_completed_Routine_inputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: developPage_developPage_completed_Routine_inputs_standard_tags_translations[];
}

export interface developPage_developPage_completed_Routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_completed_Routine_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: developPage_developPage_completed_Routine_inputs_standard_tags[];
  translations: developPage_developPage_completed_Routine_inputs_standard_translations[];
}

export interface developPage_developPage_completed_Routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: developPage_developPage_completed_Routine_inputs_translations[];
  standard: developPage_developPage_completed_Routine_inputs_standard | null;
}

export interface developPage_developPage_completed_Routine_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface developPage_developPage_completed_Routine_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: developPage_developPage_completed_Routine_nodeLinks_whens_translations[];
}

export interface developPage_developPage_completed_Routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  whens: developPage_developPage_completed_Routine_nodeLinks_whens[];
}

export interface developPage_developPage_completed_Routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface developPage_developPage_completed_Routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface developPage_developPage_completed_Routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: developPage_developPage_completed_Routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface developPage_developPage_completed_Routine_nodes_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type developPage_developPage_completed_Routine_nodes_data_NodeRoutineList_routines_routine_owner = developPage_developPage_completed_Routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization | developPage_developPage_completed_Routine_nodes_data_NodeRoutineList_routines_routine_owner_User;

export interface developPage_developPage_completed_Routine_nodes_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface developPage_developPage_completed_Routine_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isInternal: boolean | null;
  nodesCount: number | null;
  role: MemberRole | null;
  owner: developPage_developPage_completed_Routine_nodes_data_NodeRoutineList_routines_routine_owner | null;
  simplicity: number;
  translations: developPage_developPage_completed_Routine_nodes_data_NodeRoutineList_routines_routine_translations[];
  version: string | null;
}

export interface developPage_developPage_completed_Routine_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface developPage_developPage_completed_Routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: developPage_developPage_completed_Routine_nodes_data_NodeRoutineList_routines_routine;
  translations: developPage_developPage_completed_Routine_nodes_data_NodeRoutineList_routines_translations[];
}

export interface developPage_developPage_completed_Routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: developPage_developPage_completed_Routine_nodes_data_NodeRoutineList_routines[];
}

export type developPage_developPage_completed_Routine_nodes_data = developPage_developPage_completed_Routine_nodes_data_NodeEnd | developPage_developPage_completed_Routine_nodes_data_NodeRoutineList;

export interface developPage_developPage_completed_Routine_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface developPage_developPage_completed_Routine_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: developPage_developPage_completed_Routine_nodes_loop_whiles_translations[];
}

export interface developPage_developPage_completed_Routine_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: developPage_developPage_completed_Routine_nodes_loop_whiles[];
}

export interface developPage_developPage_completed_Routine_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface developPage_developPage_completed_Routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: developPage_developPage_completed_Routine_nodes_data | null;
  loop: developPage_developPage_completed_Routine_nodes_loop | null;
  translations: developPage_developPage_completed_Routine_nodes_translations[];
}

export interface developPage_developPage_completed_Routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_completed_Routine_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_completed_Routine_outputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: developPage_developPage_completed_Routine_outputs_standard_tags_translations[];
}

export interface developPage_developPage_completed_Routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_completed_Routine_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: developPage_developPage_completed_Routine_outputs_standard_tags[];
  translations: developPage_developPage_completed_Routine_outputs_standard_translations[];
}

export interface developPage_developPage_completed_Routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: developPage_developPage_completed_Routine_outputs_translations[];
  standard: developPage_developPage_completed_Routine_outputs_standard | null;
}

export interface developPage_developPage_completed_Routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface developPage_developPage_completed_Routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: developPage_developPage_completed_Routine_owner_Organization_translations[];
}

export interface developPage_developPage_completed_Routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type developPage_developPage_completed_Routine_owner = developPage_developPage_completed_Routine_owner_Organization | developPage_developPage_completed_Routine_owner_User;

export interface developPage_developPage_completed_Routine_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface developPage_developPage_completed_Routine_parent {
  __typename: "Routine";
  id: string;
  translations: developPage_developPage_completed_Routine_parent_translations[];
}

export interface developPage_developPage_completed_Routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface developPage_developPage_completed_Routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface developPage_developPage_completed_Routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: developPage_developPage_completed_Routine_resourceLists_resources_translations[];
}

export interface developPage_developPage_completed_Routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: developPage_developPage_completed_Routine_resourceLists_translations[];
  resources: developPage_developPage_completed_Routine_resourceLists_resources[];
}

export interface developPage_developPage_completed_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_completed_Routine_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: developPage_developPage_completed_Routine_tags_translations[];
}

export interface developPage_developPage_completed_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface developPage_developPage_completed_Routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: developPage_developPage_completed_Routine_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isInternal: boolean | null;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: developPage_developPage_completed_Routine_nodeLinks[];
  nodes: developPage_developPage_completed_Routine_nodes[];
  outputs: developPage_developPage_completed_Routine_outputs[];
  owner: developPage_developPage_completed_Routine_owner | null;
  parent: developPage_developPage_completed_Routine_parent | null;
  resourceLists: developPage_developPage_completed_Routine_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  role: MemberRole | null;
  tags: developPage_developPage_completed_Routine_tags[];
  translations: developPage_developPage_completed_Routine_translations[];
  updated_at: any;
  version: string | null;
}

export type developPage_developPage_completed = developPage_developPage_completed_Project | developPage_developPage_completed_Routine;

export interface developPage_developPage_inProgress_Project_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface developPage_developPage_inProgress_Project_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface developPage_developPage_inProgress_Project_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: developPage_developPage_inProgress_Project_resourceLists_resources_translations[];
}

export interface developPage_developPage_inProgress_Project_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: developPage_developPage_inProgress_Project_resourceLists_translations[];
  resources: developPage_developPage_inProgress_Project_resourceLists_resources[];
}

export interface developPage_developPage_inProgress_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_inProgress_Project_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: developPage_developPage_inProgress_Project_tags_translations[];
}

export interface developPage_developPage_inProgress_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface developPage_developPage_inProgress_Project_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface developPage_developPage_inProgress_Project_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: developPage_developPage_inProgress_Project_owner_Organization_translations[];
}

export interface developPage_developPage_inProgress_Project_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type developPage_developPage_inProgress_Project_owner = developPage_developPage_inProgress_Project_owner_Organization | developPage_developPage_inProgress_Project_owner_User;

export interface developPage_developPage_inProgress_Project {
  __typename: "Project";
  id: string;
  completedAt: any | null;
  created_at: any;
  handle: string | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  resourceLists: developPage_developPage_inProgress_Project_resourceLists[] | null;
  tags: developPage_developPage_inProgress_Project_tags[];
  translations: developPage_developPage_inProgress_Project_translations[];
  owner: developPage_developPage_inProgress_Project_owner | null;
}

export interface developPage_developPage_inProgress_Routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_inProgress_Routine_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_inProgress_Routine_inputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: developPage_developPage_inProgress_Routine_inputs_standard_tags_translations[];
}

export interface developPage_developPage_inProgress_Routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_inProgress_Routine_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: developPage_developPage_inProgress_Routine_inputs_standard_tags[];
  translations: developPage_developPage_inProgress_Routine_inputs_standard_translations[];
}

export interface developPage_developPage_inProgress_Routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: developPage_developPage_inProgress_Routine_inputs_translations[];
  standard: developPage_developPage_inProgress_Routine_inputs_standard | null;
}

export interface developPage_developPage_inProgress_Routine_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface developPage_developPage_inProgress_Routine_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: developPage_developPage_inProgress_Routine_nodeLinks_whens_translations[];
}

export interface developPage_developPage_inProgress_Routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  whens: developPage_developPage_inProgress_Routine_nodeLinks_whens[];
}

export interface developPage_developPage_inProgress_Routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface developPage_developPage_inProgress_Routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface developPage_developPage_inProgress_Routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: developPage_developPage_inProgress_Routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface developPage_developPage_inProgress_Routine_nodes_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type developPage_developPage_inProgress_Routine_nodes_data_NodeRoutineList_routines_routine_owner = developPage_developPage_inProgress_Routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization | developPage_developPage_inProgress_Routine_nodes_data_NodeRoutineList_routines_routine_owner_User;

export interface developPage_developPage_inProgress_Routine_nodes_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface developPage_developPage_inProgress_Routine_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isInternal: boolean | null;
  nodesCount: number | null;
  role: MemberRole | null;
  owner: developPage_developPage_inProgress_Routine_nodes_data_NodeRoutineList_routines_routine_owner | null;
  simplicity: number;
  translations: developPage_developPage_inProgress_Routine_nodes_data_NodeRoutineList_routines_routine_translations[];
  version: string | null;
}

export interface developPage_developPage_inProgress_Routine_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface developPage_developPage_inProgress_Routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: developPage_developPage_inProgress_Routine_nodes_data_NodeRoutineList_routines_routine;
  translations: developPage_developPage_inProgress_Routine_nodes_data_NodeRoutineList_routines_translations[];
}

export interface developPage_developPage_inProgress_Routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: developPage_developPage_inProgress_Routine_nodes_data_NodeRoutineList_routines[];
}

export type developPage_developPage_inProgress_Routine_nodes_data = developPage_developPage_inProgress_Routine_nodes_data_NodeEnd | developPage_developPage_inProgress_Routine_nodes_data_NodeRoutineList;

export interface developPage_developPage_inProgress_Routine_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface developPage_developPage_inProgress_Routine_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: developPage_developPage_inProgress_Routine_nodes_loop_whiles_translations[];
}

export interface developPage_developPage_inProgress_Routine_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: developPage_developPage_inProgress_Routine_nodes_loop_whiles[];
}

export interface developPage_developPage_inProgress_Routine_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface developPage_developPage_inProgress_Routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: developPage_developPage_inProgress_Routine_nodes_data | null;
  loop: developPage_developPage_inProgress_Routine_nodes_loop | null;
  translations: developPage_developPage_inProgress_Routine_nodes_translations[];
}

export interface developPage_developPage_inProgress_Routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_inProgress_Routine_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_inProgress_Routine_outputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: developPage_developPage_inProgress_Routine_outputs_standard_tags_translations[];
}

export interface developPage_developPage_inProgress_Routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_inProgress_Routine_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: developPage_developPage_inProgress_Routine_outputs_standard_tags[];
  translations: developPage_developPage_inProgress_Routine_outputs_standard_translations[];
}

export interface developPage_developPage_inProgress_Routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: developPage_developPage_inProgress_Routine_outputs_translations[];
  standard: developPage_developPage_inProgress_Routine_outputs_standard | null;
}

export interface developPage_developPage_inProgress_Routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface developPage_developPage_inProgress_Routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: developPage_developPage_inProgress_Routine_owner_Organization_translations[];
}

export interface developPage_developPage_inProgress_Routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type developPage_developPage_inProgress_Routine_owner = developPage_developPage_inProgress_Routine_owner_Organization | developPage_developPage_inProgress_Routine_owner_User;

export interface developPage_developPage_inProgress_Routine_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface developPage_developPage_inProgress_Routine_parent {
  __typename: "Routine";
  id: string;
  translations: developPage_developPage_inProgress_Routine_parent_translations[];
}

export interface developPage_developPage_inProgress_Routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface developPage_developPage_inProgress_Routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface developPage_developPage_inProgress_Routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: developPage_developPage_inProgress_Routine_resourceLists_resources_translations[];
}

export interface developPage_developPage_inProgress_Routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: developPage_developPage_inProgress_Routine_resourceLists_translations[];
  resources: developPage_developPage_inProgress_Routine_resourceLists_resources[];
}

export interface developPage_developPage_inProgress_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_inProgress_Routine_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: developPage_developPage_inProgress_Routine_tags_translations[];
}

export interface developPage_developPage_inProgress_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface developPage_developPage_inProgress_Routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: developPage_developPage_inProgress_Routine_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isInternal: boolean | null;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: developPage_developPage_inProgress_Routine_nodeLinks[];
  nodes: developPage_developPage_inProgress_Routine_nodes[];
  outputs: developPage_developPage_inProgress_Routine_outputs[];
  owner: developPage_developPage_inProgress_Routine_owner | null;
  parent: developPage_developPage_inProgress_Routine_parent | null;
  resourceLists: developPage_developPage_inProgress_Routine_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  role: MemberRole | null;
  tags: developPage_developPage_inProgress_Routine_tags[];
  translations: developPage_developPage_inProgress_Routine_translations[];
  updated_at: any;
  version: string | null;
}

export type developPage_developPage_inProgress = developPage_developPage_inProgress_Project | developPage_developPage_inProgress_Routine;

export interface developPage_developPage_recent_Project_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface developPage_developPage_recent_Project_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface developPage_developPage_recent_Project_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: developPage_developPage_recent_Project_resourceLists_resources_translations[];
}

export interface developPage_developPage_recent_Project_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: developPage_developPage_recent_Project_resourceLists_translations[];
  resources: developPage_developPage_recent_Project_resourceLists_resources[];
}

export interface developPage_developPage_recent_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_recent_Project_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: developPage_developPage_recent_Project_tags_translations[];
}

export interface developPage_developPage_recent_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface developPage_developPage_recent_Project_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface developPage_developPage_recent_Project_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: developPage_developPage_recent_Project_owner_Organization_translations[];
}

export interface developPage_developPage_recent_Project_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type developPage_developPage_recent_Project_owner = developPage_developPage_recent_Project_owner_Organization | developPage_developPage_recent_Project_owner_User;

export interface developPage_developPage_recent_Project {
  __typename: "Project";
  id: string;
  completedAt: any | null;
  created_at: any;
  handle: string | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  resourceLists: developPage_developPage_recent_Project_resourceLists[] | null;
  tags: developPage_developPage_recent_Project_tags[];
  translations: developPage_developPage_recent_Project_translations[];
  owner: developPage_developPage_recent_Project_owner | null;
}

export interface developPage_developPage_recent_Routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_recent_Routine_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_recent_Routine_inputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: developPage_developPage_recent_Routine_inputs_standard_tags_translations[];
}

export interface developPage_developPage_recent_Routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_recent_Routine_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: developPage_developPage_recent_Routine_inputs_standard_tags[];
  translations: developPage_developPage_recent_Routine_inputs_standard_translations[];
}

export interface developPage_developPage_recent_Routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: developPage_developPage_recent_Routine_inputs_translations[];
  standard: developPage_developPage_recent_Routine_inputs_standard | null;
}

export interface developPage_developPage_recent_Routine_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface developPage_developPage_recent_Routine_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: developPage_developPage_recent_Routine_nodeLinks_whens_translations[];
}

export interface developPage_developPage_recent_Routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  whens: developPage_developPage_recent_Routine_nodeLinks_whens[];
}

export interface developPage_developPage_recent_Routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface developPage_developPage_recent_Routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface developPage_developPage_recent_Routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: developPage_developPage_recent_Routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface developPage_developPage_recent_Routine_nodes_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type developPage_developPage_recent_Routine_nodes_data_NodeRoutineList_routines_routine_owner = developPage_developPage_recent_Routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization | developPage_developPage_recent_Routine_nodes_data_NodeRoutineList_routines_routine_owner_User;

export interface developPage_developPage_recent_Routine_nodes_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface developPage_developPage_recent_Routine_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isInternal: boolean | null;
  nodesCount: number | null;
  role: MemberRole | null;
  owner: developPage_developPage_recent_Routine_nodes_data_NodeRoutineList_routines_routine_owner | null;
  simplicity: number;
  translations: developPage_developPage_recent_Routine_nodes_data_NodeRoutineList_routines_routine_translations[];
  version: string | null;
}

export interface developPage_developPage_recent_Routine_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface developPage_developPage_recent_Routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: developPage_developPage_recent_Routine_nodes_data_NodeRoutineList_routines_routine;
  translations: developPage_developPage_recent_Routine_nodes_data_NodeRoutineList_routines_translations[];
}

export interface developPage_developPage_recent_Routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: developPage_developPage_recent_Routine_nodes_data_NodeRoutineList_routines[];
}

export type developPage_developPage_recent_Routine_nodes_data = developPage_developPage_recent_Routine_nodes_data_NodeEnd | developPage_developPage_recent_Routine_nodes_data_NodeRoutineList;

export interface developPage_developPage_recent_Routine_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface developPage_developPage_recent_Routine_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: developPage_developPage_recent_Routine_nodes_loop_whiles_translations[];
}

export interface developPage_developPage_recent_Routine_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: developPage_developPage_recent_Routine_nodes_loop_whiles[];
}

export interface developPage_developPage_recent_Routine_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface developPage_developPage_recent_Routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: developPage_developPage_recent_Routine_nodes_data | null;
  loop: developPage_developPage_recent_Routine_nodes_loop | null;
  translations: developPage_developPage_recent_Routine_nodes_translations[];
}

export interface developPage_developPage_recent_Routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_recent_Routine_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_recent_Routine_outputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: developPage_developPage_recent_Routine_outputs_standard_tags_translations[];
}

export interface developPage_developPage_recent_Routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_recent_Routine_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: developPage_developPage_recent_Routine_outputs_standard_tags[];
  translations: developPage_developPage_recent_Routine_outputs_standard_translations[];
}

export interface developPage_developPage_recent_Routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: developPage_developPage_recent_Routine_outputs_translations[];
  standard: developPage_developPage_recent_Routine_outputs_standard | null;
}

export interface developPage_developPage_recent_Routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface developPage_developPage_recent_Routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: developPage_developPage_recent_Routine_owner_Organization_translations[];
}

export interface developPage_developPage_recent_Routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type developPage_developPage_recent_Routine_owner = developPage_developPage_recent_Routine_owner_Organization | developPage_developPage_recent_Routine_owner_User;

export interface developPage_developPage_recent_Routine_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface developPage_developPage_recent_Routine_parent {
  __typename: "Routine";
  id: string;
  translations: developPage_developPage_recent_Routine_parent_translations[];
}

export interface developPage_developPage_recent_Routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface developPage_developPage_recent_Routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface developPage_developPage_recent_Routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: developPage_developPage_recent_Routine_resourceLists_resources_translations[];
}

export interface developPage_developPage_recent_Routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: developPage_developPage_recent_Routine_resourceLists_translations[];
  resources: developPage_developPage_recent_Routine_resourceLists_resources[];
}

export interface developPage_developPage_recent_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_recent_Routine_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: developPage_developPage_recent_Routine_tags_translations[];
}

export interface developPage_developPage_recent_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface developPage_developPage_recent_Routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: developPage_developPage_recent_Routine_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isInternal: boolean | null;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: developPage_developPage_recent_Routine_nodeLinks[];
  nodes: developPage_developPage_recent_Routine_nodes[];
  outputs: developPage_developPage_recent_Routine_outputs[];
  owner: developPage_developPage_recent_Routine_owner | null;
  parent: developPage_developPage_recent_Routine_parent | null;
  resourceLists: developPage_developPage_recent_Routine_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  role: MemberRole | null;
  tags: developPage_developPage_recent_Routine_tags[];
  translations: developPage_developPage_recent_Routine_translations[];
  updated_at: any;
  version: string | null;
}

export type developPage_developPage_recent = developPage_developPage_recent_Project | developPage_developPage_recent_Routine;

export interface developPage_developPage {
  __typename: "DevelopPageResult";
  completed: developPage_developPage_completed[];
  inProgress: developPage_developPage_inProgress[];
  recent: developPage_developPage_recent[];
}

export interface developPage {
  developPage: developPage_developPage;
}
