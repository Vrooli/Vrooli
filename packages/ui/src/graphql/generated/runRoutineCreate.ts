/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunRoutineCreateInput, RunStatus, NodeType, ResourceListUsedFor, ResourceUsedFor, RunRoutineStepStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: runRoutineCreate
// ====================================================

export interface runRoutineCreate_runRoutineCreate_inputs_input_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineCreate_runRoutineCreate_inputs_input_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCreate_runRoutineCreate_inputs_input_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineCreate_runRoutineCreate_inputs_input_standard_tags_translations[];
}

export interface runRoutineCreate_runRoutineCreate_inputs_input_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCreate_runRoutineCreate_inputs_input_standard {
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
  tags: runRoutineCreate_runRoutineCreate_inputs_input_standard_tags[];
  translations: runRoutineCreate_runRoutineCreate_inputs_input_standard_translations[];
}

export interface runRoutineCreate_runRoutineCreate_inputs_input {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineCreate_runRoutineCreate_inputs_input_translations[];
  standard: runRoutineCreate_runRoutineCreate_inputs_input_standard | null;
}

export interface runRoutineCreate_runRoutineCreate_inputs {
  __typename: "RunRoutineInput";
  id: string;
  data: string;
  input: runRoutineCreate_runRoutineCreate_inputs_input;
}

export interface runRoutineCreate_runRoutineCreate_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_inputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineCreate_runRoutineCreate_routine_inputs_standard_tags_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_inputs_standard {
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
  tags: runRoutineCreate_runRoutineCreate_routine_inputs_standard_tags[];
  translations: runRoutineCreate_runRoutineCreate_routine_inputs_standard_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineCreate_runRoutineCreate_routine_inputs_translations[];
  standard: runRoutineCreate_runRoutineCreate_routine_inputs_standard | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: runRoutineCreate_runRoutineCreate_routine_nodeLinks_whens_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  operation: string | null;
  whens: runRoutineCreate_runRoutineCreate_routine_nodeLinks_whens[];
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard {
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
  tags: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags[];
  translations: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_translations[];
  standard: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard {
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
  tags: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags[];
  translations: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_translations[];
  standard: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner = runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization | runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_User;

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_translations[];
  resources: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources[];
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_tags_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion {
  __typename: "Routine";
  id: string;
  complexity: number;
  inputs: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs[];
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  nodesCount: number | null;
  outputs: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs[];
  owner: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner | null;
  permissionsRoutine: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_permissionsRoutine;
  resourceLists: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists[];
  simplicity: number;
  tags: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_tags[];
  translations: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routineVersion: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_routineVersion;
  translations: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList_routines[];
}

export type runRoutineCreate_runRoutineCreate_routine_nodes_data = runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeEnd | runRoutineCreate_runRoutineCreate_routine_nodes_data_NodeRoutineList;

export interface runRoutineCreate_runRoutineCreate_routine_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: runRoutineCreate_runRoutineCreate_routine_nodes_loop_whiles_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: runRoutineCreate_runRoutineCreate_routine_nodes_loop_whiles[];
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runRoutineCreate_runRoutineCreate_routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: runRoutineCreate_runRoutineCreate_routine_nodes_data | null;
  loop: runRoutineCreate_runRoutineCreate_routine_nodes_loop | null;
  translations: runRoutineCreate_runRoutineCreate_routine_nodes_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_outputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineCreate_runRoutineCreate_routine_outputs_standard_tags_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_outputs_standard {
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
  tags: runRoutineCreate_runRoutineCreate_routine_outputs_standard_tags[];
  translations: runRoutineCreate_runRoutineCreate_routine_outputs_standard_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runRoutineCreate_runRoutineCreate_routine_outputs_translations[];
  standard: runRoutineCreate_runRoutineCreate_routine_outputs_standard | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runRoutineCreate_runRoutineCreate_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runRoutineCreate_runRoutineCreate_routine_owner_Organization_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runRoutineCreate_runRoutineCreate_routine_owner = runRoutineCreate_runRoutineCreate_routine_owner_Organization | runRoutineCreate_runRoutineCreate_routine_owner_User;

export interface runRoutineCreate_runRoutineCreate_routine_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface runRoutineCreate_runRoutineCreate_routine_parent {
  __typename: "Routine";
  id: string;
  translations: runRoutineCreate_runRoutineCreate_routine_parent_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runRoutineCreate_runRoutineCreate_routine_resourceLists_resources_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runRoutineCreate_runRoutineCreate_routine_resourceLists_translations[];
  resources: runRoutineCreate_runRoutineCreate_routine_resourceLists_resources[];
}

export interface runRoutineCreate_runRoutineCreate_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface runRoutineCreate_runRoutineCreate_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCreate_runRoutineCreate_routine_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineCreate_runRoutineCreate_routine_tags_translations[];
}

export interface runRoutineCreate_runRoutineCreate_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface runRoutineCreate_runRoutineCreate_routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: runRoutineCreate_runRoutineCreate_routine_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: runRoutineCreate_runRoutineCreate_routine_nodeLinks[];
  nodes: runRoutineCreate_runRoutineCreate_routine_nodes[];
  outputs: runRoutineCreate_runRoutineCreate_routine_outputs[];
  owner: runRoutineCreate_runRoutineCreate_routine_owner | null;
  parent: runRoutineCreate_runRoutineCreate_routine_parent | null;
  resourceLists: runRoutineCreate_runRoutineCreate_routine_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: runRoutineCreate_runRoutineCreate_routine_permissionsRoutine;
  tags: runRoutineCreate_runRoutineCreate_routine_tags[];
  translations: runRoutineCreate_runRoutineCreate_routine_translations[];
  updated_at: any;
}

export interface runRoutineCreate_runRoutineCreate_steps_node {
  __typename: "Node";
  id: string;
}

export interface runRoutineCreate_runRoutineCreate_steps {
  __typename: "RunRoutineStep";
  id: string;
  order: number;
  contextSwitches: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunRoutineStepStatus;
  step: number[];
  node: runRoutineCreate_runRoutineCreate_steps_node | null;
}

export interface runRoutineCreate_runRoutineCreate {
  __typename: "RunRoutine";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  isPrivate: boolean;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  inputs: runRoutineCreate_runRoutineCreate_inputs[];
  routine: runRoutineCreate_runRoutineCreate_routine | null;
  steps: runRoutineCreate_runRoutineCreate_steps[];
}

export interface runRoutineCreate {
  runRoutineCreate: runRoutineCreate_runRoutineCreate;
}

export interface runRoutineCreateVariables {
  input: RunRoutineCreateInput;
}
