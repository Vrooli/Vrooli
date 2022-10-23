/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: listStarFields
// ====================================================

export interface listStarFields_to_Comment_commentedOn_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
}

export interface listStarFields_to_Comment_commentedOn_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  translations: listStarFields_to_Comment_commentedOn_Project_translations[];
}

export interface listStarFields_to_Comment_commentedOn_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface listStarFields_to_Comment_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  translations: listStarFields_to_Comment_commentedOn_Routine_translations[];
}

export interface listStarFields_to_Comment_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
}

export type listStarFields_to_Comment_commentedOn = listStarFields_to_Comment_commentedOn_Project | listStarFields_to_Comment_commentedOn_Routine | listStarFields_to_Comment_commentedOn_Standard;

export interface listStarFields_to_Comment_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface listStarFields_to_Comment_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: listStarFields_to_Comment_creator_Organization_translations[];
}

export interface listStarFields_to_Comment_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type listStarFields_to_Comment_creator = listStarFields_to_Comment_creator_Organization | listStarFields_to_Comment_creator_User;

export interface listStarFields_to_Comment_permissionsComment {
  __typename: "CommentPermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReply: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface listStarFields_to_Comment_translations {
  __typename: "CommentTranslation";
  id: string;
  language: string;
  text: string;
}

export interface listStarFields_to_Comment {
  __typename: "Comment";
  id: string;
  created_at: any;
  updated_at: any;
  score: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  commentedOn: listStarFields_to_Comment_commentedOn;
  creator: listStarFields_to_Comment_creator | null;
  permissionsComment: listStarFields_to_Comment_permissionsComment | null;
  translations: listStarFields_to_Comment_translations[];
}

export interface listStarFields_to_Organization_permissionsOrganization {
  __typename: "OrganizationPermission";
  canAddMembers: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  isMember: boolean;
}

export interface listStarFields_to_Organization_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listStarFields_to_Organization_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listStarFields_to_Organization_tags_translations[];
}

export interface listStarFields_to_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface listStarFields_to_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  stars: number;
  isPrivate: boolean;
  isStarred: boolean;
  permissionsOrganization: listStarFields_to_Organization_permissionsOrganization | null;
  tags: listStarFields_to_Organization_tags[];
  translations: listStarFields_to_Organization_translations[];
}

export interface listStarFields_to_Project_permissionsProject {
  __typename: "ProjectPermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface listStarFields_to_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listStarFields_to_Project_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listStarFields_to_Project_tags_translations[];
}

export interface listStarFields_to_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface listStarFields_to_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  score: number;
  stars: number;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  permissionsProject: listStarFields_to_Project_permissionsProject;
  tags: listStarFields_to_Project_tags[];
  translations: listStarFields_to_Project_translations[];
}

export interface listStarFields_to_Routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface listStarFields_to_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listStarFields_to_Routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listStarFields_to_Routine_tags_translations[];
}

export interface listStarFields_to_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface listStarFields_to_Routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: listStarFields_to_Routine_permissionsRoutine;
  tags: listStarFields_to_Routine_tags[];
  translations: listStarFields_to_Routine_translations[];
  version: string;
  versionGroupId: string;
}

export interface listStarFields_to_Standard_permissionsStandard {
  __typename: "StandardPermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface listStarFields_to_Standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listStarFields_to_Standard_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listStarFields_to_Standard_tags_translations[];
}

export interface listStarFields_to_Standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listStarFields_to_Standard {
  __typename: "Standard";
  id: string;
  score: number;
  stars: number;
  isDeleted: boolean;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  name: string;
  permissionsStandard: listStarFields_to_Standard_permissionsStandard;
  tags: listStarFields_to_Standard_tags[];
  translations: listStarFields_to_Standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface listStarFields_to_User {
  __typename: "User";
  id: string;
  handle: string | null;
  name: string;
  stars: number;
  isStarred: boolean;
}

export interface listStarFields_to_Tag_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listStarFields_to_Tag {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listStarFields_to_Tag_translations[];
}

export type listStarFields_to = listStarFields_to_Comment | listStarFields_to_Organization | listStarFields_to_Project | listStarFields_to_Routine | listStarFields_to_Standard | listStarFields_to_User | listStarFields_to_Tag;

export interface listStarFields {
  __typename: "Star";
  id: string;
  to: listStarFields_to;
}
