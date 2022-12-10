/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunRoutineCompleteInput, RunStatus, NodeType, ResourceListUsedFor, ResourceUsedFor, RunRoutineStepStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: runRoutineComplete
// ====================================================

export interface runRoutineComplete_runRoutineComplete_inputs_input_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineComplete_runRoutineComplete_inputs_input_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineComplete_runRoutineComplete_inputs_input_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineComplete_runRoutineComplete_inputs_input_standard_tags_translations[];
}

export interface runRoutineComplete_runRoutineComplete_inputs_input_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineComplete_runRoutineComplete_inputs_input_standard {
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
  tags: runRoutineComplete_runRoutineComplete_inputs_input_standard_tags[];
  translations: runRoutineComplete_runRoutineComplete_inputs_input_standard_translations[];
}

export interface runRoutineComplete_runRoutineComplete_inputs_input {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineComplete_runRoutineComplete_inputs_input_translations[];
  standard: runRoutineComplete_runRoutineComplete_inputs_input_standard | null;
}

export interface runRoutineComplete_runRoutineComplete_inputs {
  __typename: "RunRoutineInput";
  id: string;
  data: string;
  input: runRoutineComplete_runRoutineComplete_inputs_input;
}

export interface runRoutineComplete_runRoutineComplete_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_inputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineComplete_runRoutineComplete_routine_inputs_standard_tags_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_inputs_standard {
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
  tags: runRoutineComplete_runRoutineComplete_routine_inputs_standard_tags[];
  translations: runRoutineComplete_runRoutineComplete_routine_inputs_standard_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineComplete_runRoutineComplete_routine_inputs_translations[];
  standard: runRoutineComplete_runRoutineComplete_routine_inputs_standard | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: runRoutineComplete_runRoutineComplete_routine_nodeLinks_whens_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  operation: string | null;
  whens: runRoutineComplete_runRoutineComplete_routine_nodeLinks_whens[];
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard {
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
  tags: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags[];
  translations: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_translations[];
  standard: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard {
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
  tags: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags[];
  translations: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_translations[];
  standard: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner = runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization | runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_User;

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_translations[];
  resources: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources[];
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_tags_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
  instructions: string;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion {
  __typename: "Routine";
  id: string;
  complexity: number;
  inputs: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs[];
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  nodesCount: number | null;
  outputs: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs[];
  owner: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner | null;
  permissionsRoutine: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_permissionsRoutine;
  resourceLists: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists[];
  simplicity: number;
  tags: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_tags[];
  translations: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routineVersion: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_routineVersion;
  translations: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList_routines[];
}

export type runRoutineComplete_runRoutineComplete_routine_nodes_data = runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeEnd | runRoutineComplete_runRoutineComplete_routine_nodes_data_NodeRoutineList;

export interface runRoutineComplete_runRoutineComplete_routine_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: runRoutineComplete_runRoutineComplete_routine_nodes_loop_whiles_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: runRoutineComplete_runRoutineComplete_routine_nodes_loop_whiles[];
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface runRoutineComplete_runRoutineComplete_routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: runRoutineComplete_runRoutineComplete_routine_nodes_data | null;
  loop: runRoutineComplete_runRoutineComplete_routine_nodes_loop | null;
  translations: runRoutineComplete_runRoutineComplete_routine_nodes_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_outputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineComplete_runRoutineComplete_routine_outputs_standard_tags_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_outputs_standard {
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
  tags: runRoutineComplete_runRoutineComplete_routine_outputs_standard_tags[];
  translations: runRoutineComplete_runRoutineComplete_routine_outputs_standard_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runRoutineComplete_runRoutineComplete_routine_outputs_translations[];
  standard: runRoutineComplete_runRoutineComplete_routine_outputs_standard | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runRoutineComplete_runRoutineComplete_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runRoutineComplete_runRoutineComplete_routine_owner_Organization_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runRoutineComplete_runRoutineComplete_routine_owner = runRoutineComplete_runRoutineComplete_routine_owner_Organization | runRoutineComplete_runRoutineComplete_routine_owner_User;

export interface runRoutineComplete_runRoutineComplete_routine_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runRoutineComplete_runRoutineComplete_routine_parent {
  __typename: "Routine";
  id: string;
  translations: runRoutineComplete_runRoutineComplete_routine_parent_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runRoutineComplete_runRoutineComplete_routine_resourceLists_resources_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runRoutineComplete_runRoutineComplete_routine_resourceLists_translations[];
  resources: runRoutineComplete_runRoutineComplete_routine_resourceLists_resources[];
}

export interface runRoutineComplete_runRoutineComplete_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface runRoutineComplete_runRoutineComplete_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineComplete_runRoutineComplete_routine_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineComplete_runRoutineComplete_routine_tags_translations[];
}

export interface runRoutineComplete_runRoutineComplete_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  name: string;
}

export interface runRoutineComplete_runRoutineComplete_routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: runRoutineComplete_runRoutineComplete_routine_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: runRoutineComplete_runRoutineComplete_routine_nodeLinks[];
  nodes: runRoutineComplete_runRoutineComplete_routine_nodes[];
  outputs: runRoutineComplete_runRoutineComplete_routine_outputs[];
  owner: runRoutineComplete_runRoutineComplete_routine_owner | null;
  parent: runRoutineComplete_runRoutineComplete_routine_parent | null;
  resourceLists: runRoutineComplete_runRoutineComplete_routine_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: runRoutineComplete_runRoutineComplete_routine_permissionsRoutine;
  tags: runRoutineComplete_runRoutineComplete_routine_tags[];
  translations: runRoutineComplete_runRoutineComplete_routine_translations[];
  updated_at: any;
}

export interface runRoutineComplete_runRoutineComplete_steps_node {
  __typename: "Node";
  id: string;
}

export interface runRoutineComplete_runRoutineComplete_steps {
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
  node: runRoutineComplete_runRoutineComplete_steps_node | null;
}

export interface runRoutineComplete_runRoutineComplete {
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
  inputs: runRoutineComplete_runRoutineComplete_inputs[];
  routine: runRoutineComplete_runRoutineComplete_routine | null;
  steps: runRoutineComplete_runRoutineComplete_steps[];
}

export interface runRoutineComplete {
  runRoutineComplete: runRoutineComplete_runRoutineComplete;
}

export interface runRoutineCompleteVariables {
  input: RunRoutineCompleteInput;
}
