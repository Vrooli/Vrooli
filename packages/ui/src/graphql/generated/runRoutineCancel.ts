/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunRoutineCancelInput, RunStatus, NodeType, ResourceListUsedFor, ResourceUsedFor, RunRoutineStepStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: runRoutineCancel
// ====================================================

export interface runRoutineCancel_runRoutineCancel_inputs_input_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineCancel_runRoutineCancel_inputs_input_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCancel_runRoutineCancel_inputs_input_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineCancel_runRoutineCancel_inputs_input_standard_tags_translations[];
}

export interface runRoutineCancel_runRoutineCancel_inputs_input_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCancel_runRoutineCancel_inputs_input_standard {
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
  tags: runRoutineCancel_runRoutineCancel_inputs_input_standard_tags[];
  translations: runRoutineCancel_runRoutineCancel_inputs_input_standard_translations[];
}

export interface runRoutineCancel_runRoutineCancel_inputs_input {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineCancel_runRoutineCancel_inputs_input_translations[];
  standard: runRoutineCancel_runRoutineCancel_inputs_input_standard | null;
}

export interface runRoutineCancel_runRoutineCancel_inputs {
  __typename: "RunRoutineInput";
  id: string;
  data: string;
  input: runRoutineCancel_runRoutineCancel_inputs_input;
}

export interface runRoutineCancel_runRoutineCancel_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_inputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineCancel_runRoutineCancel_routine_inputs_standard_tags_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_inputs_standard {
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
  tags: runRoutineCancel_runRoutineCancel_routine_inputs_standard_tags[];
  translations: runRoutineCancel_runRoutineCancel_routine_inputs_standard_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineCancel_runRoutineCancel_routine_inputs_translations[];
  standard: runRoutineCancel_runRoutineCancel_routine_inputs_standard | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: runRoutineCancel_runRoutineCancel_routine_nodeLinks_whens_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  operation: string | null;
  whens: runRoutineCancel_runRoutineCancel_routine_nodeLinks_whens[];
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard {
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
  tags: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags[];
  translations: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_translations[];
  standard: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard {
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
  tags: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags[];
  translations: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_translations[];
  standard: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner = runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization | runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner_User;

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_translations[];
  resources: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources[];
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_tags_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion {
  __typename: "Routine";
  id: string;
  complexity: number;
  inputs: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_inputs[];
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  nodesCount: number | null;
  outputs: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_outputs[];
  owner: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_owner | null;
  permissionsRoutine: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_permissionsRoutine;
  resourceLists: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists[];
  simplicity: number;
  tags: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_tags[];
  translations: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routineVersion: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_routineVersion;
  translations: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList_routines[];
}

export type runRoutineCancel_runRoutineCancel_routine_nodes_data = runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeEnd | runRoutineCancel_runRoutineCancel_routine_nodes_data_NodeRoutineList;

export interface runRoutineCancel_runRoutineCancel_routine_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: runRoutineCancel_runRoutineCancel_routine_nodes_loop_whiles_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: runRoutineCancel_runRoutineCancel_routine_nodes_loop_whiles[];
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runRoutineCancel_runRoutineCancel_routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: runRoutineCancel_runRoutineCancel_routine_nodes_data | null;
  loop: runRoutineCancel_runRoutineCancel_routine_nodes_loop | null;
  translations: runRoutineCancel_runRoutineCancel_routine_nodes_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_outputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineCancel_runRoutineCancel_routine_outputs_standard_tags_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_outputs_standard {
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
  tags: runRoutineCancel_runRoutineCancel_routine_outputs_standard_tags[];
  translations: runRoutineCancel_runRoutineCancel_routine_outputs_standard_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runRoutineCancel_runRoutineCancel_routine_outputs_translations[];
  standard: runRoutineCancel_runRoutineCancel_routine_outputs_standard | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runRoutineCancel_runRoutineCancel_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runRoutineCancel_runRoutineCancel_routine_owner_Organization_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runRoutineCancel_runRoutineCancel_routine_owner = runRoutineCancel_runRoutineCancel_routine_owner_Organization | runRoutineCancel_runRoutineCancel_routine_owner_User;

export interface runRoutineCancel_runRoutineCancel_routine_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface runRoutineCancel_runRoutineCancel_routine_parent {
  __typename: "Routine";
  id: string;
  translations: runRoutineCancel_runRoutineCancel_routine_parent_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runRoutineCancel_runRoutineCancel_routine_resourceLists_resources_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runRoutineCancel_runRoutineCancel_routine_resourceLists_translations[];
  resources: runRoutineCancel_runRoutineCancel_routine_resourceLists_resources[];
}

export interface runRoutineCancel_runRoutineCancel_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface runRoutineCancel_runRoutineCancel_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineCancel_runRoutineCancel_routine_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineCancel_runRoutineCancel_routine_tags_translations[];
}

export interface runRoutineCancel_runRoutineCancel_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface runRoutineCancel_runRoutineCancel_routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: runRoutineCancel_runRoutineCancel_routine_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: runRoutineCancel_runRoutineCancel_routine_nodeLinks[];
  nodes: runRoutineCancel_runRoutineCancel_routine_nodes[];
  outputs: runRoutineCancel_runRoutineCancel_routine_outputs[];
  owner: runRoutineCancel_runRoutineCancel_routine_owner | null;
  parent: runRoutineCancel_runRoutineCancel_routine_parent | null;
  resourceLists: runRoutineCancel_runRoutineCancel_routine_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: runRoutineCancel_runRoutineCancel_routine_permissionsRoutine;
  tags: runRoutineCancel_runRoutineCancel_routine_tags[];
  translations: runRoutineCancel_runRoutineCancel_routine_translations[];
  updated_at: any;
}

export interface runRoutineCancel_runRoutineCancel_steps_node {
  __typename: "Node";
  id: string;
}

export interface runRoutineCancel_runRoutineCancel_steps {
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
  node: runRoutineCancel_runRoutineCancel_steps_node | null;
}

export interface runRoutineCancel_runRoutineCancel {
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
  inputs: runRoutineCancel_runRoutineCancel_inputs[];
  routine: runRoutineCancel_runRoutineCancel_routine | null;
  steps: runRoutineCancel_runRoutineCancel_steps[];
}

export interface runRoutineCancel {
  runRoutineCancel: runRoutineCancel_runRoutineCancel;
}

export interface runRoutineCancelVariables {
  input: RunRoutineCancelInput;
}
