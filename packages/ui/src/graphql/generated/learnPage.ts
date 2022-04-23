/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole, ResourceListUsedFor, ResourceUsedFor, NodeType } from "./globalTypes";

// ====================================================
// GraphQL query operation: learnPage
// ====================================================

export interface learnPage_learnPage_courses_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface learnPage_learnPage_courses_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface learnPage_learnPage_courses_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: learnPage_learnPage_courses_resourceLists_resources_translations[];
}

export interface learnPage_learnPage_courses_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: learnPage_learnPage_courses_resourceLists_translations[];
  resources: learnPage_learnPage_courses_resourceLists_resources[];
}

export interface learnPage_learnPage_courses_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface learnPage_learnPage_courses_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: learnPage_learnPage_courses_tags_translations[];
}

export interface learnPage_learnPage_courses_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface learnPage_learnPage_courses_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface learnPage_learnPage_courses_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: learnPage_learnPage_courses_owner_Organization_translations[];
}

export interface learnPage_learnPage_courses_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type learnPage_learnPage_courses_owner = learnPage_learnPage_courses_owner_Organization | learnPage_learnPage_courses_owner_User;

export interface learnPage_learnPage_courses {
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
  resourceLists: learnPage_learnPage_courses_resourceLists[] | null;
  tags: learnPage_learnPage_courses_tags[];
  translations: learnPage_learnPage_courses_translations[];
  owner: learnPage_learnPage_courses_owner | null;
}

export interface learnPage_learnPage_tutorials_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface learnPage_learnPage_tutorials_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface learnPage_learnPage_tutorials_inputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: learnPage_learnPage_tutorials_inputs_standard_tags_translations[];
}

export interface learnPage_learnPage_tutorials_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface learnPage_learnPage_tutorials_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: learnPage_learnPage_tutorials_inputs_standard_tags[];
  translations: learnPage_learnPage_tutorials_inputs_standard_translations[];
}

export interface learnPage_learnPage_tutorials_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: learnPage_learnPage_tutorials_inputs_translations[];
  standard: learnPage_learnPage_tutorials_inputs_standard | null;
}

export interface learnPage_learnPage_tutorials_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface learnPage_learnPage_tutorials_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: learnPage_learnPage_tutorials_nodeLinks_whens_translations[];
}

export interface learnPage_learnPage_tutorials_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  whens: learnPage_learnPage_tutorials_nodeLinks_whens[];
}

export interface learnPage_learnPage_tutorials_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface learnPage_learnPage_tutorials_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface learnPage_learnPage_tutorials_nodes_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: learnPage_learnPage_tutorials_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface learnPage_learnPage_tutorials_nodes_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type learnPage_learnPage_tutorials_nodes_data_NodeRoutineList_routines_routine_owner = learnPage_learnPage_tutorials_nodes_data_NodeRoutineList_routines_routine_owner_Organization | learnPage_learnPage_tutorials_nodes_data_NodeRoutineList_routines_routine_owner_User;

export interface learnPage_learnPage_tutorials_nodes_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface learnPage_learnPage_tutorials_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isInternal: boolean | null;
  nodesCount: number | null;
  role: MemberRole | null;
  owner: learnPage_learnPage_tutorials_nodes_data_NodeRoutineList_routines_routine_owner | null;
  simplicity: number;
  translations: learnPage_learnPage_tutorials_nodes_data_NodeRoutineList_routines_routine_translations[];
  version: string | null;
}

export interface learnPage_learnPage_tutorials_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface learnPage_learnPage_tutorials_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: learnPage_learnPage_tutorials_nodes_data_NodeRoutineList_routines_routine;
  translations: learnPage_learnPage_tutorials_nodes_data_NodeRoutineList_routines_translations[];
}

export interface learnPage_learnPage_tutorials_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: learnPage_learnPage_tutorials_nodes_data_NodeRoutineList_routines[];
}

export type learnPage_learnPage_tutorials_nodes_data = learnPage_learnPage_tutorials_nodes_data_NodeEnd | learnPage_learnPage_tutorials_nodes_data_NodeRoutineList;

export interface learnPage_learnPage_tutorials_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface learnPage_learnPage_tutorials_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: learnPage_learnPage_tutorials_nodes_loop_whiles_translations[];
}

export interface learnPage_learnPage_tutorials_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: learnPage_learnPage_tutorials_nodes_loop_whiles[];
}

export interface learnPage_learnPage_tutorials_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface learnPage_learnPage_tutorials_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: learnPage_learnPage_tutorials_nodes_data | null;
  loop: learnPage_learnPage_tutorials_nodes_loop | null;
  translations: learnPage_learnPage_tutorials_nodes_translations[];
}

export interface learnPage_learnPage_tutorials_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface learnPage_learnPage_tutorials_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface learnPage_learnPage_tutorials_outputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: learnPage_learnPage_tutorials_outputs_standard_tags_translations[];
}

export interface learnPage_learnPage_tutorials_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface learnPage_learnPage_tutorials_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: learnPage_learnPage_tutorials_outputs_standard_tags[];
  translations: learnPage_learnPage_tutorials_outputs_standard_translations[];
}

export interface learnPage_learnPage_tutorials_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: learnPage_learnPage_tutorials_outputs_translations[];
  standard: learnPage_learnPage_tutorials_outputs_standard | null;
}

export interface learnPage_learnPage_tutorials_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface learnPage_learnPage_tutorials_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: learnPage_learnPage_tutorials_owner_Organization_translations[];
}

export interface learnPage_learnPage_tutorials_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type learnPage_learnPage_tutorials_owner = learnPage_learnPage_tutorials_owner_Organization | learnPage_learnPage_tutorials_owner_User;

export interface learnPage_learnPage_tutorials_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface learnPage_learnPage_tutorials_parent {
  __typename: "Routine";
  id: string;
  translations: learnPage_learnPage_tutorials_parent_translations[];
}

export interface learnPage_learnPage_tutorials_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface learnPage_learnPage_tutorials_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface learnPage_learnPage_tutorials_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: learnPage_learnPage_tutorials_resourceLists_resources_translations[];
}

export interface learnPage_learnPage_tutorials_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: learnPage_learnPage_tutorials_resourceLists_translations[];
  resources: learnPage_learnPage_tutorials_resourceLists_resources[];
}

export interface learnPage_learnPage_tutorials_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface learnPage_learnPage_tutorials_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: learnPage_learnPage_tutorials_tags_translations[];
}

export interface learnPage_learnPage_tutorials_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface learnPage_learnPage_tutorials {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: learnPage_learnPage_tutorials_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isInternal: boolean | null;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: learnPage_learnPage_tutorials_nodeLinks[];
  nodes: learnPage_learnPage_tutorials_nodes[];
  outputs: learnPage_learnPage_tutorials_outputs[];
  owner: learnPage_learnPage_tutorials_owner | null;
  parent: learnPage_learnPage_tutorials_parent | null;
  resourceLists: learnPage_learnPage_tutorials_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  role: MemberRole | null;
  tags: learnPage_learnPage_tutorials_tags[];
  translations: learnPage_learnPage_tutorials_translations[];
  updated_at: any;
  version: string | null;
}

export interface learnPage_learnPage {
  __typename: "LearnPageResult";
  courses: learnPage_learnPage_courses[];
  tutorials: learnPage_learnPage_tutorials[];
}

export interface learnPage {
  learnPage: learnPage_learnPage;
}
