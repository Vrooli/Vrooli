/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeType, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL fragment: runRoutineFields
// ====================================================

export interface runRoutineFields_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineFields_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineFields_inputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineFields_inputs_standard_tags_translations[];
}

export interface runRoutineFields_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineFields_inputs_standard {
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
  tags: runRoutineFields_inputs_standard_tags[];
  translations: runRoutineFields_inputs_standard_translations[];
}

export interface runRoutineFields_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineFields_inputs_translations[];
  standard: runRoutineFields_inputs_standard | null;
}

export interface runRoutineFields_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runRoutineFields_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: runRoutineFields_nodeLinks_whens_translations[];
}

export interface runRoutineFields_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  operation: string | null;
  whens: runRoutineFields_nodeLinks_whens[];
}

export interface runRoutineFields_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags_translations[];
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard {
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
  tags: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags[];
  translations: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_translations[];
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_translations[];
  standard: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard | null;
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags_translations[];
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard {
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
  tags: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags[];
  translations: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_translations[];
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_translations[];
  standard: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard | null;
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization_translations[];
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_owner = runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization | runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_owner_User;

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources_translations[];
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_translations[];
  resources: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources[];
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_tags_translations[];
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion {
  __typename: "Routine";
  id: string;
  complexity: number;
  inputs: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_inputs[];
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  nodesCount: number | null;
  outputs: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_outputs[];
  owner: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_owner | null;
  permissionsRoutine: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_permissionsRoutine;
  resourceLists: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists[];
  simplicity: number;
  tags: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_tags[];
  translations: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion_translations[];
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineFields_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routineVersion: runRoutineFields_nodes_data_NodeRoutineList_routines_routineVersion;
  translations: runRoutineFields_nodes_data_NodeRoutineList_routines_translations[];
}

export interface runRoutineFields_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: runRoutineFields_nodes_data_NodeRoutineList_routines[];
}

export type runRoutineFields_nodes_data = runRoutineFields_nodes_data_NodeEnd | runRoutineFields_nodes_data_NodeRoutineList;

export interface runRoutineFields_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runRoutineFields_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: runRoutineFields_nodes_loop_whiles_translations[];
}

export interface runRoutineFields_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: runRoutineFields_nodes_loop_whiles[];
}

export interface runRoutineFields_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runRoutineFields_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: runRoutineFields_nodes_data | null;
  loop: runRoutineFields_nodes_loop | null;
  translations: runRoutineFields_nodes_translations[];
}

export interface runRoutineFields_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineFields_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineFields_outputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineFields_outputs_standard_tags_translations[];
}

export interface runRoutineFields_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineFields_outputs_standard {
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
  tags: runRoutineFields_outputs_standard_tags[];
  translations: runRoutineFields_outputs_standard_translations[];
}

export interface runRoutineFields_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runRoutineFields_outputs_translations[];
  standard: runRoutineFields_outputs_standard | null;
}

export interface runRoutineFields_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runRoutineFields_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runRoutineFields_owner_Organization_translations[];
}

export interface runRoutineFields_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runRoutineFields_owner = runRoutineFields_owner_Organization | runRoutineFields_owner_User;

export interface runRoutineFields_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface runRoutineFields_parent {
  __typename: "Routine";
  id: string;
  translations: runRoutineFields_parent_translations[];
}

export interface runRoutineFields_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineFields_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineFields_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runRoutineFields_resourceLists_resources_translations[];
}

export interface runRoutineFields_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runRoutineFields_resourceLists_translations[];
  resources: runRoutineFields_resourceLists_resources[];
}

export interface runRoutineFields_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface runRoutineFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineFields_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineFields_tags_translations[];
}

export interface runRoutineFields_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface runRoutineFields {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: runRoutineFields_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: runRoutineFields_nodeLinks[];
  nodes: runRoutineFields_nodes[];
  outputs: runRoutineFields_outputs[];
  owner: runRoutineFields_owner | null;
  parent: runRoutineFields_parent | null;
  resourceLists: runRoutineFields_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: runRoutineFields_permissionsRoutine;
  tags: runRoutineFields_tags[];
  translations: runRoutineFields_translations[];
  updated_at: any;
}
