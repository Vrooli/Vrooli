/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NotificationSearchInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: notifications
// ====================================================

export interface notifications_notifications_pageInfo {
  __typename: "PageInfo";
  endCursor: string | null;
  hasNextPage: boolean;
}

export interface notifications_notifications_edges_node {
  __typename: "Notification";
  id: string;
  created_at: any;
  category: string;
  isRead: boolean;
  name: string;
  description: string | null;
  link: string | null;
  imgLink: string | null;
}

export interface notifications_notifications_edges {
  __typename: "NotificationEdge";
  cursor: string;
  node: notifications_notifications_edges_node;
}

export interface notifications_notifications {
  __typename: "NotificationSearchResult";
  pageInfo: notifications_notifications_pageInfo;
  edges: notifications_notifications_edges[];
}

export interface notifications {
  notifications: notifications_notifications;
}

export interface notificationsVariables {
  input: NotificationSearchInput;
}
