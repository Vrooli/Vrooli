/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ForkInput, ResourceListUsedFor, ResourceUsedFor, NodeType, RunStatus, RunStepStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: fork
// ====================================================

export interface fork_fork_organization_permissionsOrganization {
  __typename: "OrganizationPermission";
  canAddMembers: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  isMember: boolean;
}

export interface fork_fork_organization_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface fork_fork_organization_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface fork_fork_organization_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: fork_fork_organization_resourceLists_resources_translations[];
}

export interface fork_fork_organization_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: fork_fork_organization_resourceLists_translations[];
  resources: fork_fork_organization_resourceLists_resources[];
}

export interface fork_fork_organization_roles_translations {
  __typename: "RoleTranslation";
  id: string;
  language: string;
  description: string;
}

export interface fork_fork_organization_roles {
  __typename: "Role";
  id: string;
  created_at: any;
  updated_at: any;
  title: string;
  translations: fork_fork_organization_roles_translations[];
}

export interface fork_fork_organization_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface fork_fork_organization_tags {
  __typename: "Tag";
  tag: string;
  translations: fork_fork_organization_tags_translations[];
}

export interface fork_fork_organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  bio: string | null;
  name: string;
}

export interface fork_fork_organization {
  __typename: "Organization";
  id: string;
  created_at: any;
  handle: string | null;
  isOpenToNewMembers: boolean;
  isPrivate: boolean;
  isStarred: boolean;
  stars: number;
  permissionsOrganization: fork_fork_organization_permissionsOrganization | null;
  resourceLists: fork_fork_organization_resourceLists[];
  roles: fork_fork_organization_roles[] | null;
  tags: fork_fork_organization_tags[];
  translations: fork_fork_organization_translations[];
}

export interface fork_fork_project_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface fork_fork_project_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface fork_fork_project_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface fork_fork_project_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: fork_fork_project_resourceLists_resources_translations[];
}

export interface fork_fork_project_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: fork_fork_project_resourceLists_translations[];
  resources: fork_fork_project_resourceLists_resources[];
}

export interface fork_fork_project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface fork_fork_project_tags {
  __typename: "Tag";
  tag: string;
  translations: fork_fork_project_tags_translations[];
}

export interface fork_fork_project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface fork_fork_project_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface fork_fork_project_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: fork_fork_project_owner_Organization_translations[];
}

export interface fork_fork_project_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type fork_fork_project_owner = fork_fork_project_owner_Organization | fork_fork_project_owner_User;

export interface fork_fork_project {
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
  permissionsProject: fork_fork_project_permissionsProject;
  resourceLists: fork_fork_project_resourceLists[] | null;
  tags: fork_fork_project_tags[];
  translations: fork_fork_project_translations[];
  owner: fork_fork_project_owner | null;
}

export interface fork_fork_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface fork_fork_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface fork_fork_routine_inputs_standard {
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
  translations: fork_fork_routine_inputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface fork_fork_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: fork_fork_routine_inputs_translations[];
  standard: fork_fork_routine_inputs_standard | null;
}

export interface fork_fork_routine_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface fork_fork_routine_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: fork_fork_routine_nodeLinks_whens_translations[];
}

export interface fork_fork_routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  operation: string | null;
  whens: fork_fork_routine_nodeLinks_whens[];
}

export interface fork_fork_routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard {
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
  translations: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_inputs_translations[];
  standard: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard | null;
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard {
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
  translations: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_outputs_translations[];
  standard: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard | null;
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_owner = fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization | fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_owner_User;

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources_translations[];
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_translations[];
  resources: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources[];
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_tags {
  __typename: "Tag";
  tag: string;
  translations: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_tags_translations[];
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  inputs: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_inputs[];
  nodesCount: number | null;
  outputs: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_outputs[];
  owner: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_owner | null;
  permissionsRoutine: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_permissionsRoutine;
  resourceLists: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists[];
  simplicity: number;
  tags: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_tags[];
  translations: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine_translations[];
  version: string;
  versionGroupId: string;
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface fork_fork_routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: fork_fork_routine_nodes_data_NodeRoutineList_routines_routine;
  translations: fork_fork_routine_nodes_data_NodeRoutineList_routines_translations[];
}

export interface fork_fork_routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: fork_fork_routine_nodes_data_NodeRoutineList_routines[];
}

export type fork_fork_routine_nodes_data = fork_fork_routine_nodes_data_NodeEnd | fork_fork_routine_nodes_data_NodeRoutineList;

export interface fork_fork_routine_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface fork_fork_routine_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: fork_fork_routine_nodes_loop_whiles_translations[];
}

export interface fork_fork_routine_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: fork_fork_routine_nodes_loop_whiles[];
}

export interface fork_fork_routine_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface fork_fork_routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: fork_fork_routine_nodes_data | null;
  loop: fork_fork_routine_nodes_loop | null;
  translations: fork_fork_routine_nodes_translations[];
}

export interface fork_fork_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface fork_fork_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface fork_fork_routine_outputs_standard {
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
  translations: fork_fork_routine_outputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface fork_fork_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: fork_fork_routine_outputs_translations[];
  standard: fork_fork_routine_outputs_standard | null;
}

export interface fork_fork_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface fork_fork_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: fork_fork_routine_owner_Organization_translations[];
}

export interface fork_fork_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type fork_fork_routine_owner = fork_fork_routine_owner_Organization | fork_fork_routine_owner_User;

export interface fork_fork_routine_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface fork_fork_routine_parent {
  __typename: "Routine";
  id: string;
  translations: fork_fork_routine_parent_translations[];
}

export interface fork_fork_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface fork_fork_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface fork_fork_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: fork_fork_routine_resourceLists_resources_translations[];
}

export interface fork_fork_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: fork_fork_routine_resourceLists_translations[];
  resources: fork_fork_routine_resourceLists_resources[];
}

export interface fork_fork_routine_runs_steps_node {
  __typename: "Node";
  id: string;
}

export interface fork_fork_routine_runs_steps {
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
  node: fork_fork_routine_runs_steps_node | null;
}

export interface fork_fork_routine_runs {
  __typename: "Run";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  steps: fork_fork_routine_runs_steps[];
}

export interface fork_fork_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface fork_fork_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface fork_fork_routine_tags {
  __typename: "Tag";
  tag: string;
  translations: fork_fork_routine_tags_translations[];
}

export interface fork_fork_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface fork_fork_routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: fork_fork_routine_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: fork_fork_routine_nodeLinks[];
  nodes: fork_fork_routine_nodes[];
  outputs: fork_fork_routine_outputs[];
  owner: fork_fork_routine_owner | null;
  parent: fork_fork_routine_parent | null;
  resourceLists: fork_fork_routine_resourceLists[];
  runs: fork_fork_routine_runs[];
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: fork_fork_routine_permissionsRoutine;
  tags: fork_fork_routine_tags[];
  translations: fork_fork_routine_translations[];
  updated_at: any;
  version: string;
  versionGroupId: string;
}

export interface fork_fork_standard_permissionsStandard {
  __typename: "StandardPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface fork_fork_standard_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface fork_fork_standard_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface fork_fork_standard_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: fork_fork_standard_resourceLists_resources_translations[];
}

export interface fork_fork_standard_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: fork_fork_standard_resourceLists_translations[];
  resources: fork_fork_standard_resourceLists_resources[];
}

export interface fork_fork_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface fork_fork_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: fork_fork_standard_tags_translations[];
}

export interface fork_fork_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface fork_fork_standard_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface fork_fork_standard_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: fork_fork_standard_creator_Organization_translations[];
}

export interface fork_fork_standard_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type fork_fork_standard_creator = fork_fork_standard_creator_Organization | fork_fork_standard_creator_User;

export interface fork_fork_standard {
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
  permissionsStandard: fork_fork_standard_permissionsStandard;
  resourceLists: fork_fork_standard_resourceLists[];
  tags: fork_fork_standard_tags[];
  translations: fork_fork_standard_translations[];
  creator: fork_fork_standard_creator | null;
  stars: number;
  isStarred: boolean;
  score: number;
  isUpvoted: boolean | null;
  version: string;
  versionGroupId: string;
}

export interface fork_fork {
  __typename: "ForkResult";
  organization: fork_fork_organization | null;
  project: fork_fork_project | null;
  routine: fork_fork_routine | null;
  standard: fork_fork_standard | null;
}

export interface fork {
  fork: fork_fork;
}

export interface forkVariables {
  input: ForkInput;
}
