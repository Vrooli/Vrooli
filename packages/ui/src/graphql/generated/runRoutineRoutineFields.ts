/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeType, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL fragment: runRoutineRoutineFields
// ====================================================

export interface runRoutineRoutineFields_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineRoutineFields_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineRoutineFields_inputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineRoutineFields_inputs_standard_tags_translations[];
}

export interface runRoutineRoutineFields_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineRoutineFields_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isDeleted: boolean;
  isInternal: boolean;
  isPrivate: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  tags: runRoutineRoutineFields_inputs_standard_tags[];
  translations: runRoutineRoutineFields_inputs_standard_translations[];
}

export interface runRoutineRoutineFields_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineRoutineFields_inputs_translations[];
  standard: runRoutineRoutineFields_inputs_standard | null;
}

export interface runRoutineRoutineFields_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runRoutineRoutineFields_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: runRoutineRoutineFields_nodeLinks_whens_translations[];
}

export interface runRoutineRoutineFields_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  operation: string | null;
  whens: runRoutineRoutineFields_nodeLinks_whens[];
}

export interface runRoutineRoutineFields_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags_translations[];
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isDeleted: boolean;
  isInternal: boolean;
  isPrivate: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  tags: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags[];
  translations: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_translations[];
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_translations[];
  standard: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard | null;
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags_translations[];
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isDeleted: boolean;
  isInternal: boolean;
  isPrivate: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  tags: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags[];
  translations: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_translations[];
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_translations[];
  standard: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard | null;
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization_translations[];
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_owner = runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization | runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_owner_User;

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources_translations[];
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_translations[];
  resources: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources[];
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_tags_translations[];
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion {
  __typename: "Routine";
  id: string;
  complexity: number;
  inputs: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs[];
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  nodesCount: number | null;
  outputs: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs[];
  owner: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_owner | null;
  permissionsRoutine: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_permissionsRoutine;
  resourceLists: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists[];
  simplicity: number;
  tags: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_tags[];
  translations: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_translations[];
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routineVersion: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion;
  translations: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines_translations[];
}

export interface runRoutineRoutineFields_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: runRoutineRoutineFields_nodes_data_NodeRoutineList_routines[];
}

export type runRoutineRoutineFields_nodes_data = runRoutineRoutineFields_nodes_data_NodeEnd | runRoutineRoutineFields_nodes_data_NodeRoutineList;

export interface runRoutineRoutineFields_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runRoutineRoutineFields_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: runRoutineRoutineFields_nodes_loop_whiles_translations[];
}

export interface runRoutineRoutineFields_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: runRoutineRoutineFields_nodes_loop_whiles[];
}

export interface runRoutineRoutineFields_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runRoutineRoutineFields_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: runRoutineRoutineFields_nodes_data | null;
  loop: runRoutineRoutineFields_nodes_loop | null;
  translations: runRoutineRoutineFields_nodes_translations[];
}

export interface runRoutineRoutineFields_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineRoutineFields_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineRoutineFields_outputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineRoutineFields_outputs_standard_tags_translations[];
}

export interface runRoutineRoutineFields_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineRoutineFields_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isDeleted: boolean;
  isInternal: boolean;
  isPrivate: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  tags: runRoutineRoutineFields_outputs_standard_tags[];
  translations: runRoutineRoutineFields_outputs_standard_translations[];
}

export interface runRoutineRoutineFields_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runRoutineRoutineFields_outputs_translations[];
  standard: runRoutineRoutineFields_outputs_standard | null;
}

export interface runRoutineRoutineFields_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runRoutineRoutineFields_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runRoutineRoutineFields_owner_Organization_translations[];
}

export interface runRoutineRoutineFields_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runRoutineRoutineFields_owner = runRoutineRoutineFields_owner_Organization | runRoutineRoutineFields_owner_User;

export interface runRoutineRoutineFields_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface runRoutineRoutineFields_parent {
  __typename: "Routine";
  id: string;
  translations: runRoutineRoutineFields_parent_translations[];
}

export interface runRoutineRoutineFields_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineRoutineFields_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineRoutineFields_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runRoutineRoutineFields_resourceLists_resources_translations[];
}

export interface runRoutineRoutineFields_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runRoutineRoutineFields_resourceLists_translations[];
  resources: runRoutineRoutineFields_resourceLists_resources[];
}

export interface runRoutineRoutineFields_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface runRoutineRoutineFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineRoutineFields_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineRoutineFields_tags_translations[];
}

export interface runRoutineRoutineFields_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface runRoutineRoutineFields {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: runRoutineRoutineFields_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: runRoutineRoutineFields_nodeLinks[];
  nodes: runRoutineRoutineFields_nodes[];
  outputs: runRoutineRoutineFields_outputs[];
  owner: runRoutineRoutineFields_owner | null;
  parent: runRoutineRoutineFields_parent | null;
  resourceLists: runRoutineRoutineFields_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: runRoutineRoutineFields_permissionsRoutine;
  tags: runRoutineRoutineFields_tags[];
  translations: runRoutineRoutineFields_translations[];
  updated_at: any;
}
