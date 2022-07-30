/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunCreateInput, RunStatus, NodeType, ResourceListUsedFor, ResourceUsedFor, RunStepStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: runCreate
// ====================================================

export interface runCreate_runCreate_inputs {
  __typename: "RunInput";
  id: string;
  data: string;
}

export interface runCreate_runCreate_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCreate_runCreate_routine_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCreate_runCreate_routine_inputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runCreate_runCreate_routine_inputs_standard_tags_translations[];
}

export interface runCreate_runCreate_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCreate_runCreate_routine_inputs_standard {
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
  tags: runCreate_runCreate_routine_inputs_standard_tags[];
  translations: runCreate_runCreate_routine_inputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface runCreate_runCreate_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runCreate_runCreate_routine_inputs_translations[];
  standard: runCreate_runCreate_routine_inputs_standard | null;
}

export interface runCreate_runCreate_routine_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runCreate_runCreate_routine_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: runCreate_runCreate_routine_nodeLinks_whens_translations[];
}

export interface runCreate_runCreate_routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  operation: string | null;
  whens: runCreate_runCreate_routine_nodeLinks_whens[];
}

export interface runCreate_runCreate_routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_tags_translations[];
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard {
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
  tags: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_tags[];
  translations: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_translations[];
  standard: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard | null;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_tags_translations[];
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard {
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
  tags: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_tags[];
  translations: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_translations[];
  standard: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard | null;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_owner = runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization | runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_owner_User;

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources_translations[];
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_translations[];
  resources: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources[];
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_tags {
  __typename: "Tag";
  tag: string;
  translations: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_tags_translations[];
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  inputs: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_inputs[];
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  nodesCount: number | null;
  outputs: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_outputs[];
  owner: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_owner | null;
  permissionsRoutine: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_permissionsRoutine;
  resourceLists: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists[];
  simplicity: number;
  tags: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_tags[];
  translations: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine_translations[];
  version: string;
  versionGroupId: string;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_routine;
  translations: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines_translations[];
}

export interface runCreate_runCreate_routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: runCreate_runCreate_routine_nodes_data_NodeRoutineList_routines[];
}

export type runCreate_runCreate_routine_nodes_data = runCreate_runCreate_routine_nodes_data_NodeEnd | runCreate_runCreate_routine_nodes_data_NodeRoutineList;

export interface runCreate_runCreate_routine_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runCreate_runCreate_routine_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: runCreate_runCreate_routine_nodes_loop_whiles_translations[];
}

export interface runCreate_runCreate_routine_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: runCreate_runCreate_routine_nodes_loop_whiles[];
}

export interface runCreate_runCreate_routine_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runCreate_runCreate_routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: runCreate_runCreate_routine_nodes_data | null;
  loop: runCreate_runCreate_routine_nodes_loop | null;
  translations: runCreate_runCreate_routine_nodes_translations[];
}

export interface runCreate_runCreate_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCreate_runCreate_routine_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCreate_runCreate_routine_outputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runCreate_runCreate_routine_outputs_standard_tags_translations[];
}

export interface runCreate_runCreate_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCreate_runCreate_routine_outputs_standard {
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
  tags: runCreate_runCreate_routine_outputs_standard_tags[];
  translations: runCreate_runCreate_routine_outputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface runCreate_runCreate_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runCreate_runCreate_routine_outputs_translations[];
  standard: runCreate_runCreate_routine_outputs_standard | null;
}

export interface runCreate_runCreate_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runCreate_runCreate_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runCreate_runCreate_routine_owner_Organization_translations[];
}

export interface runCreate_runCreate_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runCreate_runCreate_routine_owner = runCreate_runCreate_routine_owner_Organization | runCreate_runCreate_routine_owner_User;

export interface runCreate_runCreate_routine_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface runCreate_runCreate_routine_parent {
  __typename: "Routine";
  id: string;
  translations: runCreate_runCreate_routine_parent_translations[];
}

export interface runCreate_runCreate_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runCreate_runCreate_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runCreate_runCreate_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runCreate_runCreate_routine_resourceLists_resources_translations[];
}

export interface runCreate_runCreate_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runCreate_runCreate_routine_resourceLists_translations[];
  resources: runCreate_runCreate_routine_resourceLists_resources[];
}

export interface runCreate_runCreate_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface runCreate_runCreate_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runCreate_runCreate_routine_tags {
  __typename: "Tag";
  tag: string;
  translations: runCreate_runCreate_routine_tags_translations[];
}

export interface runCreate_runCreate_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface runCreate_runCreate_routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: runCreate_runCreate_routine_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: runCreate_runCreate_routine_nodeLinks[];
  nodes: runCreate_runCreate_routine_nodes[];
  outputs: runCreate_runCreate_routine_outputs[];
  owner: runCreate_runCreate_routine_owner | null;
  parent: runCreate_runCreate_routine_parent | null;
  resourceLists: runCreate_runCreate_routine_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: runCreate_runCreate_routine_permissionsRoutine;
  tags: runCreate_runCreate_routine_tags[];
  translations: runCreate_runCreate_routine_translations[];
  updated_at: any;
  version: string;
  versionGroupId: string;
}

export interface runCreate_runCreate_steps_node {
  __typename: "Node";
  id: string;
}

export interface runCreate_runCreate_steps {
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
  node: runCreate_runCreate_steps_node | null;
}

export interface runCreate_runCreate {
  __typename: "Run";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  isPrivate: boolean;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  inputs: runCreate_runCreate_inputs[];
  routine: runCreate_runCreate_routine | null;
  steps: runCreate_runCreate_steps[];
}

export interface runCreate {
  runCreate: runCreate_runCreate;
}

export interface runCreateVariables {
  input: RunCreateInput;
}
