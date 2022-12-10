/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunRoutineUpdateInput, RunStatus, NodeType, ResourceListUsedFor, ResourceUsedFor, RunRoutineStepStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: runRoutineUpdate
// ====================================================

export interface runRoutineUpdate_runRoutineUpdate_inputs_input_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_inputs_input_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_inputs_input_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineUpdate_runRoutineUpdate_inputs_input_standard_tags_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_inputs_input_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_inputs_input_standard {
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
  tags: runRoutineUpdate_runRoutineUpdate_inputs_input_standard_tags[];
  translations: runRoutineUpdate_runRoutineUpdate_inputs_input_standard_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_inputs_input {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineUpdate_runRoutineUpdate_inputs_input_translations[];
  standard: runRoutineUpdate_runRoutineUpdate_inputs_input_standard | null;
}

export interface runRoutineUpdate_runRoutineUpdate_inputs {
  __typename: "RunRoutineInput";
  id: string;
  data: string;
  input: runRoutineUpdate_runRoutineUpdate_inputs_input;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_inputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineUpdate_runRoutineUpdate_routine_inputs_standard_tags_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_inputs_standard {
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
  tags: runRoutineUpdate_runRoutineUpdate_routine_inputs_standard_tags[];
  translations: runRoutineUpdate_runRoutineUpdate_routine_inputs_standard_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineUpdate_runRoutineUpdate_routine_inputs_translations[];
  standard: runRoutineUpdate_runRoutineUpdate_routine_inputs_standard | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: runRoutineUpdate_runRoutineUpdate_routine_nodeLinks_whens_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  operation: string | null;
  whens: runRoutineUpdate_runRoutineUpdate_routine_nodeLinks_whens[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard {
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
  tags: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags[];
  translations: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_translations[];
  standard: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard {
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
  tags: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags[];
  translations: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_translations[];
  standard: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner = runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization | runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_User;

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_translations[];
  resources: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_tags_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
  instructions: string;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion {
  __typename: "Routine";
  id: string;
  complexity: number;
  inputs: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs[];
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  nodesCount: number | null;
  outputs: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs[];
  owner: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner | null;
  permissionsRoutine: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_permissionsRoutine;
  resourceLists: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists[];
  simplicity: number;
  tags: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_tags[];
  translations: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routineVersion: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_routineVersion;
  translations: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList_routines[];
}

export type runRoutineUpdate_runRoutineUpdate_routine_nodes_data = runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeEnd | runRoutineUpdate_runRoutineUpdate_routine_nodes_data_NodeRoutineList;

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: runRoutineUpdate_runRoutineUpdate_routine_nodes_loop_whiles_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: runRoutineUpdate_runRoutineUpdate_routine_nodes_loop_whiles[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: runRoutineUpdate_runRoutineUpdate_routine_nodes_data | null;
  loop: runRoutineUpdate_runRoutineUpdate_routine_nodes_loop | null;
  translations: runRoutineUpdate_runRoutineUpdate_routine_nodes_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_outputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineUpdate_runRoutineUpdate_routine_outputs_standard_tags_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_outputs_standard {
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
  tags: runRoutineUpdate_runRoutineUpdate_routine_outputs_standard_tags[];
  translations: runRoutineUpdate_runRoutineUpdate_routine_outputs_standard_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runRoutineUpdate_runRoutineUpdate_routine_outputs_translations[];
  standard: runRoutineUpdate_runRoutineUpdate_routine_outputs_standard | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runRoutineUpdate_runRoutineUpdate_routine_owner_Organization_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runRoutineUpdate_runRoutineUpdate_routine_owner = runRoutineUpdate_runRoutineUpdate_routine_owner_Organization | runRoutineUpdate_runRoutineUpdate_routine_owner_User;

export interface runRoutineUpdate_runRoutineUpdate_routine_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_parent {
  __typename: "Routine";
  id: string;
  translations: runRoutineUpdate_runRoutineUpdate_routine_parent_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runRoutineUpdate_runRoutineUpdate_routine_resourceLists_resources_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runRoutineUpdate_runRoutineUpdate_routine_resourceLists_translations[];
  resources: runRoutineUpdate_runRoutineUpdate_routine_resourceLists_resources[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineUpdate_runRoutineUpdate_routine_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineUpdate_runRoutineUpdate_routine_tags_translations[];
}

export interface runRoutineUpdate_runRoutineUpdate_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  name: string;
}

export interface runRoutineUpdate_runRoutineUpdate_routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: runRoutineUpdate_runRoutineUpdate_routine_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: runRoutineUpdate_runRoutineUpdate_routine_nodeLinks[];
  nodes: runRoutineUpdate_runRoutineUpdate_routine_nodes[];
  outputs: runRoutineUpdate_runRoutineUpdate_routine_outputs[];
  owner: runRoutineUpdate_runRoutineUpdate_routine_owner | null;
  parent: runRoutineUpdate_runRoutineUpdate_routine_parent | null;
  resourceLists: runRoutineUpdate_runRoutineUpdate_routine_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: runRoutineUpdate_runRoutineUpdate_routine_permissionsRoutine;
  tags: runRoutineUpdate_runRoutineUpdate_routine_tags[];
  translations: runRoutineUpdate_runRoutineUpdate_routine_translations[];
  updated_at: any;
}

export interface runRoutineUpdate_runRoutineUpdate_steps_node {
  __typename: "Node";
  id: string;
}

export interface runRoutineUpdate_runRoutineUpdate_steps {
  __typename: "RunRoutineStep";
  id: string;
  order: number;
  contextSwitches: number;
  startedAt: any | null;
  timeElapsed: number | null;
  completedAt: any | null;
  name: string;
  status: RunRoutineStepStatus;
  step: number[];
  node: runRoutineUpdate_runRoutineUpdate_steps_node | null;
}

export interface runRoutineUpdate_runRoutineUpdate {
  __typename: "RunRoutine";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  isPrivate: boolean;
  startedAt: any | null;
  timeElapsed: number | null;
  completedAt: any | null;
  name: string;
  status: RunStatus;
  inputs: runRoutineUpdate_runRoutineUpdate_inputs[];
  routine: runRoutineUpdate_runRoutineUpdate_routine | null;
  steps: runRoutineUpdate_runRoutineUpdate_steps[];
}

export interface runRoutineUpdate {
  runRoutineUpdate: runRoutineUpdate_runRoutineUpdate;
}

export interface runRoutineUpdateVariables {
  input: RunRoutineUpdateInput;
}
