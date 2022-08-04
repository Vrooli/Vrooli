/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunUpdateInput, RunStatus, NodeType, ResourceListUsedFor, ResourceUsedFor, RunStepStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: runUpdate
// ====================================================

export interface runUpdate_runUpdate_inputs_input_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runUpdate_runUpdate_inputs_input_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runUpdate_runUpdate_inputs_input_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runUpdate_runUpdate_inputs_input_standard_tags_translations[];
}

export interface runUpdate_runUpdate_inputs_input_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runUpdate_runUpdate_inputs_input_standard {
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
  tags: runUpdate_runUpdate_inputs_input_standard_tags[];
  translations: runUpdate_runUpdate_inputs_input_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface runUpdate_runUpdate_inputs_input {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runUpdate_runUpdate_inputs_input_translations[];
  standard: runUpdate_runUpdate_inputs_input_standard | null;
}

export interface runUpdate_runUpdate_inputs {
  __typename: "RunInput";
  id: string;
  data: string;
  input: runUpdate_runUpdate_inputs_input;
}

export interface runUpdate_runUpdate_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runUpdate_runUpdate_routine_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runUpdate_runUpdate_routine_inputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runUpdate_runUpdate_routine_inputs_standard_tags_translations[];
}

export interface runUpdate_runUpdate_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runUpdate_runUpdate_routine_inputs_standard {
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
  tags: runUpdate_runUpdate_routine_inputs_standard_tags[];
  translations: runUpdate_runUpdate_routine_inputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface runUpdate_runUpdate_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runUpdate_runUpdate_routine_inputs_translations[];
  standard: runUpdate_runUpdate_routine_inputs_standard | null;
}

export interface runUpdate_runUpdate_routine_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runUpdate_runUpdate_routine_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: runUpdate_runUpdate_routine_nodeLinks_whens_translations[];
}

export interface runUpdate_runUpdate_routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  operation: string | null;
  whens: runUpdate_runUpdate_routine_nodeLinks_whens[];
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_tags_translations[];
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard {
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
  tags: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_tags[];
  translations: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_translations[];
  standard: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard | null;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_tags_translations[];
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard {
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
  tags: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_tags[];
  translations: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_translations[];
  standard: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard | null;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_owner = runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization | runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_owner_User;

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources_translations[];
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_translations[];
  resources: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources[];
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_tags {
  __typename: "Tag";
  tag: string;
  translations: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_tags_translations[];
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  inputs: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_inputs[];
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  nodesCount: number | null;
  outputs: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_outputs[];
  owner: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_owner | null;
  permissionsRoutine: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_permissionsRoutine;
  resourceLists: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists[];
  simplicity: number;
  tags: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_tags[];
  translations: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine_translations[];
  version: string;
  versionGroupId: string;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_routine;
  translations: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines_translations[];
}

export interface runUpdate_runUpdate_routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: runUpdate_runUpdate_routine_nodes_data_NodeRoutineList_routines[];
}

export type runUpdate_runUpdate_routine_nodes_data = runUpdate_runUpdate_routine_nodes_data_NodeEnd | runUpdate_runUpdate_routine_nodes_data_NodeRoutineList;

export interface runUpdate_runUpdate_routine_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runUpdate_runUpdate_routine_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: runUpdate_runUpdate_routine_nodes_loop_whiles_translations[];
}

export interface runUpdate_runUpdate_routine_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: runUpdate_runUpdate_routine_nodes_loop_whiles[];
}

export interface runUpdate_runUpdate_routine_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runUpdate_runUpdate_routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: runUpdate_runUpdate_routine_nodes_data | null;
  loop: runUpdate_runUpdate_routine_nodes_loop | null;
  translations: runUpdate_runUpdate_routine_nodes_translations[];
}

export interface runUpdate_runUpdate_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runUpdate_runUpdate_routine_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runUpdate_runUpdate_routine_outputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runUpdate_runUpdate_routine_outputs_standard_tags_translations[];
}

export interface runUpdate_runUpdate_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runUpdate_runUpdate_routine_outputs_standard {
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
  tags: runUpdate_runUpdate_routine_outputs_standard_tags[];
  translations: runUpdate_runUpdate_routine_outputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface runUpdate_runUpdate_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runUpdate_runUpdate_routine_outputs_translations[];
  standard: runUpdate_runUpdate_routine_outputs_standard | null;
}

export interface runUpdate_runUpdate_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runUpdate_runUpdate_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runUpdate_runUpdate_routine_owner_Organization_translations[];
}

export interface runUpdate_runUpdate_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runUpdate_runUpdate_routine_owner = runUpdate_runUpdate_routine_owner_Organization | runUpdate_runUpdate_routine_owner_User;

export interface runUpdate_runUpdate_routine_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface runUpdate_runUpdate_routine_parent {
  __typename: "Routine";
  id: string;
  translations: runUpdate_runUpdate_routine_parent_translations[];
}

export interface runUpdate_runUpdate_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runUpdate_runUpdate_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runUpdate_runUpdate_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runUpdate_runUpdate_routine_resourceLists_resources_translations[];
}

export interface runUpdate_runUpdate_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runUpdate_runUpdate_routine_resourceLists_translations[];
  resources: runUpdate_runUpdate_routine_resourceLists_resources[];
}

export interface runUpdate_runUpdate_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface runUpdate_runUpdate_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runUpdate_runUpdate_routine_tags {
  __typename: "Tag";
  tag: string;
  translations: runUpdate_runUpdate_routine_tags_translations[];
}

export interface runUpdate_runUpdate_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface runUpdate_runUpdate_routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: runUpdate_runUpdate_routine_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: runUpdate_runUpdate_routine_nodeLinks[];
  nodes: runUpdate_runUpdate_routine_nodes[];
  outputs: runUpdate_runUpdate_routine_outputs[];
  owner: runUpdate_runUpdate_routine_owner | null;
  parent: runUpdate_runUpdate_routine_parent | null;
  resourceLists: runUpdate_runUpdate_routine_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: runUpdate_runUpdate_routine_permissionsRoutine;
  tags: runUpdate_runUpdate_routine_tags[];
  translations: runUpdate_runUpdate_routine_translations[];
  updated_at: any;
  version: string;
  versionGroupId: string;
}

export interface runUpdate_runUpdate_steps_node {
  __typename: "Node";
  id: string;
}

export interface runUpdate_runUpdate_steps {
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
  node: runUpdate_runUpdate_steps_node | null;
}

export interface runUpdate_runUpdate {
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
  inputs: runUpdate_runUpdate_inputs[];
  routine: runUpdate_runUpdate_routine | null;
  steps: runUpdate_runUpdate_steps[];
}

export interface runUpdate {
  runUpdate: runUpdate_runUpdate;
}

export interface runUpdateVariables {
  input: RunUpdateInput;
}
