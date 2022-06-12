/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { CommentSearchInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL query operation: comments
// ====================================================

export interface comments_comments_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface comments_comments_edges_node_commentedOn_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
}

export interface comments_comments_edges_node_commentedOn_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  translations: comments_comments_edges_node_commentedOn_Project_translations[];
}

export interface comments_comments_edges_node_commentedOn_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface comments_comments_edges_node_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  translations: comments_comments_edges_node_commentedOn_Routine_translations[];
}

export interface comments_comments_edges_node_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
}

export type comments_comments_edges_node_commentedOn = comments_comments_edges_node_commentedOn_Project | comments_comments_edges_node_commentedOn_Routine | comments_comments_edges_node_commentedOn_Standard;

export interface comments_comments_edges_node_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface comments_comments_edges_node_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: comments_comments_edges_node_creator_Organization_translations[];
}

export interface comments_comments_edges_node_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type comments_comments_edges_node_creator = comments_comments_edges_node_creator_Organization | comments_comments_edges_node_creator_User;

export interface comments_comments_edges_node_translations {
  __typename: "CommentTranslation";
  id: string;
  language: string;
  text: string;
}

export interface comments_comments_edges_node {
  __typename: "Comment";
  id: string;
  created_at: any;
  updated_at: any;
  score: number | null;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  isStarred: boolean;
  commentedOn: comments_comments_edges_node_commentedOn;
  creator: comments_comments_edges_node_creator | null;
  translations: comments_comments_edges_node_translations[];
}

export interface comments_comments_edges {
  __typename: "CommentEdge";
  cursor: string;
  node: comments_comments_edges_node;
}

export interface comments_comments {
  __typename: "CommentSearchResult";
  pageInfo: comments_comments_pageInfo;
  edges: comments_comments_edges[];
}

export interface comments {
  comments: comments_comments;
}

export interface commentsVariables {
  input: CommentSearchInput;
}
