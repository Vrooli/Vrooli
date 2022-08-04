/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { CopyInput, NodeType, ResourceListUsedFor, ResourceUsedFor, RunStatus, RunStepStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: copy
// ====================================================

export interface copy_copy_node_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface copy_copy_node_data_NodeRoutineList_routines_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface copy_copy_node_data_NodeRoutineList_routines_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface copy_copy_node_data_NodeRoutineList_routines_routine_tags {
  __typename: "Tag";
  tag: string;
  translations: copy_copy_node_data_NodeRoutineList_routines_routine_tags_translations[];
}

export interface copy_copy_node_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface copy_copy_node_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  version: string;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  simplicity: number;
  permissionsRoutine: copy_copy_node_data_NodeRoutineList_routines_routine_permissionsRoutine;
  tags: copy_copy_node_data_NodeRoutineList_routines_routine_tags[];
  translations: copy_copy_node_data_NodeRoutineList_routines_routine_translations[];
}

export interface copy_copy_node_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface copy_copy_node_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: copy_copy_node_data_NodeRoutineList_routines_routine;
  translations: copy_copy_node_data_NodeRoutineList_routines_translations[];
}

export interface copy_copy_node_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: copy_copy_node_data_NodeRoutineList_routines[];
}

export type copy_copy_node_data = copy_copy_node_data_NodeEnd | copy_copy_node_data_NodeRoutineList;

export interface copy_copy_node_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface copy_copy_node_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: copy_copy_node_loop_whiles_translations[];
}

export interface copy_copy_node_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: copy_copy_node_loop_whiles[];
}

export interface copy_copy_node_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface copy_copy_node {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: copy_copy_node_data | null;
  loop: copy_copy_node_loop | null;
  translations: copy_copy_node_translations[];
}

export interface copy_copy_organization_permissionsOrganization {
  __typename: "OrganizationPermission";
  canAddMembers: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  isMember: boolean;
}

export interface copy_copy_organization_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface copy_copy_organization_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface copy_copy_organization_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: copy_copy_organization_resourceLists_resources_translations[];
}

export interface copy_copy_organization_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: copy_copy_organization_resourceLists_translations[];
  resources: copy_copy_organization_resourceLists_resources[];
}

export interface copy_copy_organization_roles_translations {
  __typename: "RoleTranslation";
  id: string;
  language: string;
  description: string;
}

export interface copy_copy_organization_roles {
  __typename: "Role";
  id: string;
  created_at: any;
  updated_at: any;
  title: string;
  translations: copy_copy_organization_roles_translations[];
}

export interface copy_copy_organization_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface copy_copy_organization_tags {
  __typename: "Tag";
  tag: string;
  translations: copy_copy_organization_tags_translations[];
}

export interface copy_copy_organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  bio: string | null;
  name: string;
}

export interface copy_copy_organization {
  __typename: "Organization";
  id: string;
  created_at: any;
  handle: string | null;
  isOpenToNewMembers: boolean;
  isPrivate: boolean;
  isStarred: boolean;
  stars: number;
  permissionsOrganization: copy_copy_organization_permissionsOrganization | null;
  resourceLists: copy_copy_organization_resourceLists[];
  roles: copy_copy_organization_roles[] | null;
  tags: copy_copy_organization_tags[];
  translations: copy_copy_organization_translations[];
}

export interface copy_copy_project_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface copy_copy_project_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface copy_copy_project_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface copy_copy_project_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: copy_copy_project_resourceLists_resources_translations[];
}

export interface copy_copy_project_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: copy_copy_project_resourceLists_translations[];
  resources: copy_copy_project_resourceLists_resources[];
}

export interface copy_copy_project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface copy_copy_project_tags {
  __typename: "Tag";
  tag: string;
  translations: copy_copy_project_tags_translations[];
}

export interface copy_copy_project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface copy_copy_project_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface copy_copy_project_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: copy_copy_project_owner_Organization_translations[];
}

export interface copy_copy_project_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type copy_copy_project_owner = copy_copy_project_owner_Organization | copy_copy_project_owner_User;

export interface copy_copy_project {
  __typename: "Project";
  id: string;
  completedAt: any | null;
  created_at: any;
  handle: string | null;
  isComplete: boolean;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  score: number;
  stars: number;
  permissionsProject: copy_copy_project_permissionsProject;
  resourceLists: copy_copy_project_resourceLists[] | null;
  tags: copy_copy_project_tags[];
  translations: copy_copy_project_translations[];
  owner: copy_copy_project_owner | null;
}

export interface copy_copy_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface copy_copy_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface copy_copy_routine_inputs_standard {
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
  translations: copy_copy_routine_inputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface copy_copy_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: copy_copy_routine_inputs_translations[];
  standard: copy_copy_routine_inputs_standard | null;
}

export interface copy_copy_routine_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface copy_copy_routine_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: copy_copy_routine_nodeLinks_whens_translations[];
}

export interface copy_copy_routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  operation: string | null;
  whens: copy_copy_routine_nodeLinks_whens[];
}

export interface copy_copy_routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard {
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
  translations: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_inputs_translations[];
  standard: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard | null;
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard {
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
  translations: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_outputs_translations[];
  standard: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard | null;
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_owner = copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization | copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_owner_User;

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources_translations[];
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_translations[];
  resources: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources[];
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_tags {
  __typename: "Tag";
  tag: string;
  translations: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_tags_translations[];
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  inputs: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_inputs[];
  nodesCount: number | null;
  outputs: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_outputs[];
  owner: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_owner | null;
  permissionsRoutine: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_permissionsRoutine;
  resourceLists: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists[];
  simplicity: number;
  tags: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_tags[];
  translations: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine_translations[];
  version: string;
  versionGroupId: string;
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface copy_copy_routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: copy_copy_routine_nodes_data_NodeRoutineList_routines_routine;
  translations: copy_copy_routine_nodes_data_NodeRoutineList_routines_translations[];
}

export interface copy_copy_routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: copy_copy_routine_nodes_data_NodeRoutineList_routines[];
}

export type copy_copy_routine_nodes_data = copy_copy_routine_nodes_data_NodeEnd | copy_copy_routine_nodes_data_NodeRoutineList;

export interface copy_copy_routine_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface copy_copy_routine_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: copy_copy_routine_nodes_loop_whiles_translations[];
}

export interface copy_copy_routine_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: copy_copy_routine_nodes_loop_whiles[];
}

export interface copy_copy_routine_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface copy_copy_routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: copy_copy_routine_nodes_data | null;
  loop: copy_copy_routine_nodes_loop | null;
  translations: copy_copy_routine_nodes_translations[];
}

export interface copy_copy_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface copy_copy_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface copy_copy_routine_outputs_standard {
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
  translations: copy_copy_routine_outputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface copy_copy_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: copy_copy_routine_outputs_translations[];
  standard: copy_copy_routine_outputs_standard | null;
}

export interface copy_copy_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface copy_copy_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: copy_copy_routine_owner_Organization_translations[];
}

export interface copy_copy_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type copy_copy_routine_owner = copy_copy_routine_owner_Organization | copy_copy_routine_owner_User;

export interface copy_copy_routine_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface copy_copy_routine_parent {
  __typename: "Routine";
  id: string;
  translations: copy_copy_routine_parent_translations[];
}

export interface copy_copy_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface copy_copy_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface copy_copy_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: copy_copy_routine_resourceLists_resources_translations[];
}

export interface copy_copy_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: copy_copy_routine_resourceLists_translations[];
  resources: copy_copy_routine_resourceLists_resources[];
}

export interface copy_copy_routine_runs_inputs_input {
  __typename: "InputItem";
  id: string;
}

export interface copy_copy_routine_runs_inputs {
  __typename: "RunInput";
  id: string;
  data: string;
  input: copy_copy_routine_runs_inputs_input;
}

export interface copy_copy_routine_runs_steps_node {
  __typename: "Node";
  id: string;
}

export interface copy_copy_routine_runs_steps {
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
  node: copy_copy_routine_runs_steps_node | null;
}

export interface copy_copy_routine_runs {
  __typename: "Run";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  inputs: copy_copy_routine_runs_inputs[];
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  steps: copy_copy_routine_runs_steps[];
}

export interface copy_copy_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface copy_copy_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface copy_copy_routine_tags {
  __typename: "Tag";
  tag: string;
  translations: copy_copy_routine_tags_translations[];
}

export interface copy_copy_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface copy_copy_routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: copy_copy_routine_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: copy_copy_routine_nodeLinks[];
  nodes: copy_copy_routine_nodes[];
  outputs: copy_copy_routine_outputs[];
  owner: copy_copy_routine_owner | null;
  parent: copy_copy_routine_parent | null;
  resourceLists: copy_copy_routine_resourceLists[];
  runs: copy_copy_routine_runs[];
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: copy_copy_routine_permissionsRoutine;
  tags: copy_copy_routine_tags[];
  translations: copy_copy_routine_translations[];
  updated_at: any;
  version: string;
  versionGroupId: string;
}

export interface copy_copy_standard_permissionsStandard {
  __typename: "StandardPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface copy_copy_standard_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface copy_copy_standard_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface copy_copy_standard_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: copy_copy_standard_resourceLists_resources_translations[];
}

export interface copy_copy_standard_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: copy_copy_standard_resourceLists_translations[];
  resources: copy_copy_standard_resourceLists_resources[];
}

export interface copy_copy_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface copy_copy_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: copy_copy_standard_tags_translations[];
}

export interface copy_copy_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface copy_copy_standard_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface copy_copy_standard_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: copy_copy_standard_creator_Organization_translations[];
}

export interface copy_copy_standard_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type copy_copy_standard_creator = copy_copy_standard_creator_Organization | copy_copy_standard_creator_User;

export interface copy_copy_standard {
  __typename: "Standard";
  id: string;
  isDeleted: boolean;
  isInternal: boolean;
  isPrivate: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  default: string | null;
  created_at: any;
  permissionsStandard: copy_copy_standard_permissionsStandard;
  resourceLists: copy_copy_standard_resourceLists[];
  tags: copy_copy_standard_tags[];
  translations: copy_copy_standard_translations[];
  creator: copy_copy_standard_creator | null;
  stars: number;
  isStarred: boolean;
  score: number;
  isUpvoted: boolean | null;
  version: string;
  versionGroupId: string;
}

export interface copy_copy {
  __typename: "CopyResult";
  node: copy_copy_node | null;
  organization: copy_copy_organization | null;
  project: copy_copy_project | null;
  routine: copy_copy_routine | null;
  standard: copy_copy_standard | null;
}

export interface copy {
  copy: copy_copy;
}

export interface copyVariables {
  input: CopyInput;
}
