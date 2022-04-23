/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole, ResourceListUsedFor, ResourceUsedFor, NodeType } from "./globalTypes";

// ====================================================
// GraphQL query operation: researchPage
// ====================================================

export interface researchPage_researchPage_processes_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface researchPage_researchPage_processes_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: researchPage_researchPage_processes_tags_translations[];
}

export interface researchPage_researchPage_processes_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface researchPage_researchPage_processes {
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
  tags: researchPage_researchPage_processes_tags[];
  translations: researchPage_researchPage_processes_translations[];
  version: string | null;
}

export interface researchPage_researchPage_newlyCompleted_Project_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface researchPage_researchPage_newlyCompleted_Project_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface researchPage_researchPage_newlyCompleted_Project_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: researchPage_researchPage_newlyCompleted_Project_resourceLists_resources_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Project_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: researchPage_researchPage_newlyCompleted_Project_resourceLists_translations[];
  resources: researchPage_researchPage_newlyCompleted_Project_resourceLists_resources[];
}

export interface researchPage_researchPage_newlyCompleted_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface researchPage_researchPage_newlyCompleted_Project_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: researchPage_researchPage_newlyCompleted_Project_tags_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface researchPage_researchPage_newlyCompleted_Project_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface researchPage_researchPage_newlyCompleted_Project_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: researchPage_researchPage_newlyCompleted_Project_owner_Organization_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Project_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type researchPage_researchPage_newlyCompleted_Project_owner = researchPage_researchPage_newlyCompleted_Project_owner_Organization | researchPage_researchPage_newlyCompleted_Project_owner_User;

export interface researchPage_researchPage_newlyCompleted_Project {
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
  resourceLists: researchPage_researchPage_newlyCompleted_Project_resourceLists[] | null;
  tags: researchPage_researchPage_newlyCompleted_Project_tags[];
  translations: researchPage_researchPage_newlyCompleted_Project_translations[];
  owner: researchPage_researchPage_newlyCompleted_Project_owner | null;
}

export interface researchPage_researchPage_newlyCompleted_Routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface researchPage_researchPage_newlyCompleted_Routine_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface researchPage_researchPage_newlyCompleted_Routine_inputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: researchPage_researchPage_newlyCompleted_Routine_inputs_standard_tags_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface researchPage_researchPage_newlyCompleted_Routine_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: researchPage_researchPage_newlyCompleted_Routine_inputs_standard_tags[];
  translations: researchPage_researchPage_newlyCompleted_Routine_inputs_standard_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: researchPage_researchPage_newlyCompleted_Routine_inputs_translations[];
  standard: researchPage_researchPage_newlyCompleted_Routine_inputs_standard | null;
}

export interface researchPage_researchPage_newlyCompleted_Routine_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface researchPage_researchPage_newlyCompleted_Routine_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: researchPage_researchPage_newlyCompleted_Routine_nodeLinks_whens_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  whens: researchPage_researchPage_newlyCompleted_Routine_nodeLinks_whens[];
}

export interface researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeRoutineList_routines_routine_owner = researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization | researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeRoutineList_routines_routine_owner_User;

export interface researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isInternal: boolean | null;
  nodesCount: number | null;
  role: MemberRole | null;
  owner: researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeRoutineList_routines_routine_owner | null;
  simplicity: number;
  translations: researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeRoutineList_routines_routine_translations[];
  version: string | null;
}

export interface researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeRoutineList_routines_routine;
  translations: researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeRoutineList_routines_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeRoutineList_routines[];
}

export type researchPage_researchPage_newlyCompleted_Routine_nodes_data = researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeEnd | researchPage_researchPage_newlyCompleted_Routine_nodes_data_NodeRoutineList;

export interface researchPage_researchPage_newlyCompleted_Routine_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface researchPage_researchPage_newlyCompleted_Routine_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: researchPage_researchPage_newlyCompleted_Routine_nodes_loop_whiles_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Routine_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: researchPage_researchPage_newlyCompleted_Routine_nodes_loop_whiles[];
}

export interface researchPage_researchPage_newlyCompleted_Routine_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface researchPage_researchPage_newlyCompleted_Routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: researchPage_researchPage_newlyCompleted_Routine_nodes_data | null;
  loop: researchPage_researchPage_newlyCompleted_Routine_nodes_loop | null;
  translations: researchPage_researchPage_newlyCompleted_Routine_nodes_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface researchPage_researchPage_newlyCompleted_Routine_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface researchPage_researchPage_newlyCompleted_Routine_outputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: researchPage_researchPage_newlyCompleted_Routine_outputs_standard_tags_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface researchPage_researchPage_newlyCompleted_Routine_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: researchPage_researchPage_newlyCompleted_Routine_outputs_standard_tags[];
  translations: researchPage_researchPage_newlyCompleted_Routine_outputs_standard_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: researchPage_researchPage_newlyCompleted_Routine_outputs_translations[];
  standard: researchPage_researchPage_newlyCompleted_Routine_outputs_standard | null;
}

export interface researchPage_researchPage_newlyCompleted_Routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface researchPage_researchPage_newlyCompleted_Routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: researchPage_researchPage_newlyCompleted_Routine_owner_Organization_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type researchPage_researchPage_newlyCompleted_Routine_owner = researchPage_researchPage_newlyCompleted_Routine_owner_Organization | researchPage_researchPage_newlyCompleted_Routine_owner_User;

export interface researchPage_researchPage_newlyCompleted_Routine_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface researchPage_researchPage_newlyCompleted_Routine_parent {
  __typename: "Routine";
  id: string;
  translations: researchPage_researchPage_newlyCompleted_Routine_parent_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface researchPage_researchPage_newlyCompleted_Routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface researchPage_researchPage_newlyCompleted_Routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: researchPage_researchPage_newlyCompleted_Routine_resourceLists_resources_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: researchPage_researchPage_newlyCompleted_Routine_resourceLists_translations[];
  resources: researchPage_researchPage_newlyCompleted_Routine_resourceLists_resources[];
}

export interface researchPage_researchPage_newlyCompleted_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface researchPage_researchPage_newlyCompleted_Routine_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: researchPage_researchPage_newlyCompleted_Routine_tags_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface researchPage_researchPage_newlyCompleted_Routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: researchPage_researchPage_newlyCompleted_Routine_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isInternal: boolean | null;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: researchPage_researchPage_newlyCompleted_Routine_nodeLinks[];
  nodes: researchPage_researchPage_newlyCompleted_Routine_nodes[];
  outputs: researchPage_researchPage_newlyCompleted_Routine_outputs[];
  owner: researchPage_researchPage_newlyCompleted_Routine_owner | null;
  parent: researchPage_researchPage_newlyCompleted_Routine_parent | null;
  resourceLists: researchPage_researchPage_newlyCompleted_Routine_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  role: MemberRole | null;
  tags: researchPage_researchPage_newlyCompleted_Routine_tags[];
  translations: researchPage_researchPage_newlyCompleted_Routine_translations[];
  updated_at: any;
  version: string | null;
}

export type researchPage_researchPage_newlyCompleted = researchPage_researchPage_newlyCompleted_Project | researchPage_researchPage_newlyCompleted_Routine;

export interface researchPage_researchPage_needVotes_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface researchPage_researchPage_needVotes_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface researchPage_researchPage_needVotes_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: researchPage_researchPage_needVotes_resourceLists_resources_translations[];
}

export interface researchPage_researchPage_needVotes_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: researchPage_researchPage_needVotes_resourceLists_translations[];
  resources: researchPage_researchPage_needVotes_resourceLists_resources[];
}

export interface researchPage_researchPage_needVotes_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface researchPage_researchPage_needVotes_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: researchPage_researchPage_needVotes_tags_translations[];
}

export interface researchPage_researchPage_needVotes_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface researchPage_researchPage_needVotes_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface researchPage_researchPage_needVotes_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: researchPage_researchPage_needVotes_owner_Organization_translations[];
}

export interface researchPage_researchPage_needVotes_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type researchPage_researchPage_needVotes_owner = researchPage_researchPage_needVotes_owner_Organization | researchPage_researchPage_needVotes_owner_User;

export interface researchPage_researchPage_needVotes {
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
  resourceLists: researchPage_researchPage_needVotes_resourceLists[] | null;
  tags: researchPage_researchPage_needVotes_tags[];
  translations: researchPage_researchPage_needVotes_translations[];
  owner: researchPage_researchPage_needVotes_owner | null;
}

export interface researchPage_researchPage_needInvestments {
  __typename: "Project";
}

export interface researchPage_researchPage_needMembers_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface researchPage_researchPage_needMembers_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface researchPage_researchPage_needMembers_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: researchPage_researchPage_needMembers_resourceLists_resources_translations[];
}

export interface researchPage_researchPage_needMembers_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: researchPage_researchPage_needMembers_resourceLists_translations[];
  resources: researchPage_researchPage_needMembers_resourceLists_resources[];
}

export interface researchPage_researchPage_needMembers_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface researchPage_researchPage_needMembers_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: researchPage_researchPage_needMembers_tags_translations[];
}

export interface researchPage_researchPage_needMembers_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  bio: string | null;
  name: string;
}

export interface researchPage_researchPage_needMembers {
  __typename: "Organization";
  id: string;
  created_at: any;
  handle: string | null;
  isOpenToNewMembers: boolean;
  isStarred: boolean;
  role: MemberRole | null;
  stars: number;
  resourceLists: researchPage_researchPage_needMembers_resourceLists[];
  tags: researchPage_researchPage_needMembers_tags[];
  translations: researchPage_researchPage_needMembers_translations[];
}

export interface researchPage_researchPage {
  __typename: "ResearchPageResult";
  processes: researchPage_researchPage_processes[];
  newlyCompleted: researchPage_researchPage_newlyCompleted[];
  needVotes: researchPage_researchPage_needVotes[];
  needInvestments: researchPage_researchPage_needInvestments[];
  needMembers: researchPage_researchPage_needMembers[];
}

export interface researchPage {
  researchPage: researchPage_researchPage;
}
