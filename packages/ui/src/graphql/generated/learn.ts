/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL query operation: learn
// ====================================================

export interface learn_learn_courses_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface learn_learn_courses_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface learn_learn_courses_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: learn_learn_courses_tags_translations[];
}

export interface learn_learn_courses_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface learn_learn_courses {
  __typename: "Project";
  id: string;
  commentsCount: number;
  handle: string | null;
  score: number;
  stars: number;
  isComplete: boolean;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  reportsCount: number;
  permissionsProject: learn_learn_courses_permissionsProject;
  tags: learn_learn_courses_tags[];
  translations: learn_learn_courses_translations[];
}

export interface learn_learn_tutorials_permissionsRoutine {
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

export interface learn_learn_tutorials_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface learn_learn_tutorials_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: learn_learn_tutorials_tags_translations[];
}

export interface learn_learn_tutorials_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface learn_learn_tutorials {
  __typename: "Routine";
  id: string;
  commentsCount: number;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isDeleted: boolean;
  isInternal: boolean | null;
  isComplete: boolean;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  reportsCount: number;
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: learn_learn_tutorials_permissionsRoutine;
  tags: learn_learn_tutorials_tags[];
  translations: learn_learn_tutorials_translations[];
}

export interface learn_learn {
  __typename: "LearnResult";
  courses: learn_learn_courses[];
  tutorials: learn_learn_tutorials[];
}

export interface learn {
  learn: learn_learn;
}
