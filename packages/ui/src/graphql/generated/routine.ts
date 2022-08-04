/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdInput, NodeType, ResourceListUsedFor, ResourceUsedFor, RunStatus, RunStepStatus } from "./globalTypes";

// ====================================================
// GraphQL query operation: routine
// ====================================================

export interface routine_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routine_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface routine_routine_inputs_standard {
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
  translations: routine_routine_inputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface routine_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: routine_routine_inputs_translations[];
  standard: routine_routine_inputs_standard | null;
}

export interface routine_routine_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routine_routine_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: routine_routine_nodeLinks_whens_translations[];
}

export interface routine_routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  operation: string | null;
  whens: routine_routine_nodeLinks_whens[];
}

export interface routine_routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard {
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
  translations: routine_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: routine_routine_nodes_data_NodeRoutineList_routines_routine_inputs_translations[];
  standard: routine_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard | null;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard {
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
  translations: routine_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: routine_routine_nodes_data_NodeRoutineList_routines_routine_outputs_translations[];
  standard: routine_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard | null;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: routine_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type routine_routine_nodes_data_NodeRoutineList_routines_routine_owner = routine_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization | routine_routine_nodes_data_NodeRoutineList_routines_routine_owner_User;

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: routine_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources_translations[];
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: routine_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_translations[];
  resources: routine_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources[];
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_tags {
  __typename: "Tag";
  tag: string;
  translations: routine_routine_nodes_data_NodeRoutineList_routines_routine_tags_translations[];
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  inputs: routine_routine_nodes_data_NodeRoutineList_routines_routine_inputs[];
  nodesCount: number | null;
  outputs: routine_routine_nodes_data_NodeRoutineList_routines_routine_outputs[];
  owner: routine_routine_nodes_data_NodeRoutineList_routines_routine_owner | null;
  permissionsRoutine: routine_routine_nodes_data_NodeRoutineList_routines_routine_permissionsRoutine;
  resourceLists: routine_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists[];
  simplicity: number;
  tags: routine_routine_nodes_data_NodeRoutineList_routines_routine_tags[];
  translations: routine_routine_nodes_data_NodeRoutineList_routines_routine_translations[];
  version: string;
  versionGroupId: string;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routine_routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: routine_routine_nodes_data_NodeRoutineList_routines_routine;
  translations: routine_routine_nodes_data_NodeRoutineList_routines_translations[];
}

export interface routine_routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: routine_routine_nodes_data_NodeRoutineList_routines[];
}

export type routine_routine_nodes_data = routine_routine_nodes_data_NodeEnd | routine_routine_nodes_data_NodeRoutineList;

export interface routine_routine_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routine_routine_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: routine_routine_nodes_loop_whiles_translations[];
}

export interface routine_routine_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: routine_routine_nodes_loop_whiles[];
}

export interface routine_routine_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routine_routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: routine_routine_nodes_data | null;
  loop: routine_routine_nodes_loop | null;
  translations: routine_routine_nodes_translations[];
}

export interface routine_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routine_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface routine_routine_outputs_standard {
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
  translations: routine_routine_outputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface routine_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: routine_routine_outputs_translations[];
  standard: routine_routine_outputs_standard | null;
}

export interface routine_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface routine_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: routine_routine_owner_Organization_translations[];
}

export interface routine_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type routine_routine_owner = routine_routine_owner_Organization | routine_routine_owner_User;

export interface routine_routine_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface routine_routine_parent {
  __typename: "Routine";
  id: string;
  translations: routine_routine_parent_translations[];
}

export interface routine_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routine_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routine_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: routine_routine_resourceLists_resources_translations[];
}

export interface routine_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: routine_routine_resourceLists_translations[];
  resources: routine_routine_resourceLists_resources[];
}

export interface routine_routine_runs_inputs_input {
  __typename: "InputItem";
  id: string;
}

export interface routine_routine_runs_inputs {
  __typename: "RunInput";
  id: string;
  data: string;
  input: routine_routine_runs_inputs_input;
}

export interface routine_routine_runs_steps_node {
  __typename: "Node";
  id: string;
}

export interface routine_routine_runs_steps {
  __typename: "RunStep";
  id: string;
  order: number;
  contextSwitches: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStepStatus;
  step: number[];
  node: routine_routine_runs_steps_node | null;
}

export interface routine_routine_runs {
  __typename: "Run";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  inputs: routine_routine_runs_inputs[];
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  steps: routine_routine_runs_steps[];
}

export interface routine_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface routine_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routine_routine_tags {
  __typename: "Tag";
  tag: string;
  translations: routine_routine_tags_translations[];
}

export interface routine_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface routine_routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: routine_routine_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: routine_routine_nodeLinks[];
  nodes: routine_routine_nodes[];
  outputs: routine_routine_outputs[];
  owner: routine_routine_owner | null;
  parent: routine_routine_parent | null;
  resourceLists: routine_routine_resourceLists[];
  runs: routine_routine_runs[];
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: routine_routine_permissionsRoutine;
  tags: routine_routine_tags[];
  translations: routine_routine_translations[];
  updated_at: any;
  version: string;
  versionGroupId: string;
}

export interface routine {
  routine: routine_routine | null;
}

export interface routineVariables {
  input: FindByIdInput;
}
