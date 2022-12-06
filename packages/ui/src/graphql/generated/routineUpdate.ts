/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RoutineUpdateInput, NodeType, ResourceListUsedFor, ResourceUsedFor, RunStatus, RunRoutineStepStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: routineUpdate
// ====================================================

export interface routineUpdate_routineUpdate_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface routineUpdate_routineUpdate_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface routineUpdate_routineUpdate_inputs_standard {
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
  translations: routineUpdate_routineUpdate_inputs_standard_translations[];
}

export interface routineUpdate_routineUpdate_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: routineUpdate_routineUpdate_inputs_translations[];
  standard: routineUpdate_routineUpdate_inputs_standard | null;
}

export interface routineUpdate_routineUpdate_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineUpdate_routineUpdate_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: routineUpdate_routineUpdate_nodeLinks_whens_translations[];
}

export interface routineUpdate_routineUpdate_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  operation: string | null;
  whens: routineUpdate_routineUpdate_nodeLinks_whens[];
}

export interface routineUpdate_routineUpdate_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard {
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
  translations: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_translations[];
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_inputs_translations[];
  standard: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard | null;
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard {
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
  translations: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_translations[];
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_outputs_translations[];
  standard: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard | null;
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization_translations[];
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_owner = routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization | routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_owner_User;

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources_translations[];
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_translations[];
  resources: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources[];
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_tags {
  __typename: "Tag";
  tag: string;
  translations: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_tags_translations[];
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion {
  __typename: "Routine";
  id: string;
  complexity: number;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  inputs: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_inputs[];
  nodesCount: number | null;
  outputs: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_outputs[];
  owner: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_owner | null;
  permissionsRoutine: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_permissionsRoutine;
  resourceLists: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists[];
  simplicity: number;
  tags: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_tags[];
  translations: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion_translations[];
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routineVersion: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_routineVersion;
  translations: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines_translations[];
}

export interface routineUpdate_routineUpdate_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: routineUpdate_routineUpdate_nodes_data_NodeRoutineList_routines[];
}

export type routineUpdate_routineUpdate_nodes_data = routineUpdate_routineUpdate_nodes_data_NodeEnd | routineUpdate_routineUpdate_nodes_data_NodeRoutineList;

export interface routineUpdate_routineUpdate_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineUpdate_routineUpdate_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: routineUpdate_routineUpdate_nodes_loop_whiles_translations[];
}

export interface routineUpdate_routineUpdate_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: routineUpdate_routineUpdate_nodes_loop_whiles[];
}

export interface routineUpdate_routineUpdate_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineUpdate_routineUpdate_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: routineUpdate_routineUpdate_nodes_data | null;
  loop: routineUpdate_routineUpdate_nodes_loop | null;
  translations: routineUpdate_routineUpdate_nodes_translations[];
}

export interface routineUpdate_routineUpdate_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface routineUpdate_routineUpdate_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface routineUpdate_routineUpdate_outputs_standard {
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
  translations: routineUpdate_routineUpdate_outputs_standard_translations[];
}

export interface routineUpdate_routineUpdate_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: routineUpdate_routineUpdate_outputs_translations[];
  standard: routineUpdate_routineUpdate_outputs_standard | null;
}

export interface routineUpdate_routineUpdate_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface routineUpdate_routineUpdate_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: routineUpdate_routineUpdate_owner_Organization_translations[];
}

export interface routineUpdate_routineUpdate_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type routineUpdate_routineUpdate_owner = routineUpdate_routineUpdate_owner_Organization | routineUpdate_routineUpdate_owner_User;

export interface routineUpdate_routineUpdate_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface routineUpdate_routineUpdate_parent {
  __typename: "Routine";
  id: string;
  translations: routineUpdate_routineUpdate_parent_translations[];
}

export interface routineUpdate_routineUpdate_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineUpdate_routineUpdate_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineUpdate_routineUpdate_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: routineUpdate_routineUpdate_resourceLists_resources_translations[];
}

export interface routineUpdate_routineUpdate_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: routineUpdate_routineUpdate_resourceLists_translations[];
  resources: routineUpdate_routineUpdate_resourceLists_resources[];
}

export interface routineUpdate_routineUpdate_runs_inputs_input {
  __typename: "InputItem";
  id: string;
}

export interface routineUpdate_routineUpdate_runs_inputs {
  __typename: "RunRoutineInput";
  id: string;
  data: string;
  input: routineUpdate_routineUpdate_runs_inputs_input;
}

export interface routineUpdate_routineUpdate_runs_steps_node {
  __typename: "Node";
  id: string;
}

export interface routineUpdate_routineUpdate_runs_steps {
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
  node: routineUpdate_routineUpdate_runs_steps_node | null;
}

export interface routineUpdate_routineUpdate_runs {
  __typename: "RunRoutine";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  inputs: routineUpdate_routineUpdate_runs_inputs[];
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  steps: routineUpdate_routineUpdate_runs_steps[];
}

export interface routineUpdate_routineUpdate_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface routineUpdate_routineUpdate_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineUpdate_routineUpdate_tags {
  __typename: "Tag";
  tag: string;
  translations: routineUpdate_routineUpdate_tags_translations[];
}

export interface routineUpdate_routineUpdate_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface routineUpdate_routineUpdate {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: routineUpdate_routineUpdate_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: routineUpdate_routineUpdate_nodeLinks[];
  nodes: routineUpdate_routineUpdate_nodes[];
  outputs: routineUpdate_routineUpdate_outputs[];
  owner: routineUpdate_routineUpdate_owner | null;
  parent: routineUpdate_routineUpdate_parent | null;
  reportsCount: number;
  resourceLists: routineUpdate_routineUpdate_resourceLists[];
  runs: routineUpdate_routineUpdate_runs[];
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: routineUpdate_routineUpdate_permissionsRoutine;
  tags: routineUpdate_routineUpdate_tags[];
  translations: routineUpdate_routineUpdate_translations[];
  updated_at: any;
}

export interface routineUpdate {
  routineUpdate: routineUpdate_routineUpdate;
}

export interface routineUpdateVariables {
  input: RoutineUpdateInput;
}
