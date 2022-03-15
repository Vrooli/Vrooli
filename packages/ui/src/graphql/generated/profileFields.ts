/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL fragment: profileFields
// ====================================================

export interface profileFields_emails {
  __typename: "Email";
  id: string;
  emailAddress: string;
  receivesAccountUpdates: boolean;
  receivesBusinessUpdates: boolean;
}

export interface profileFields_wallets {
  __typename: "Wallet";
  name: string | null;
  publicAddress: string;
  verified: boolean;
}

export interface profileFields_translations {
  __typename: "UserTranslation";
  id: string;
  language: string;
  bio: string | null;
}

export interface profileFields_starredTags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface profileFields_starredTags {
  __typename: "Tag";
  id: string;
  tag: string;
  created_at: any;
  stars: number;
  isStarred: boolean;
  translations: profileFields_starredTags_translations[];
}

export interface profileFields_hiddenTags_tag_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface profileFields_hiddenTags_tag {
  __typename: "Tag";
  id: string;
  tag: string;
  created_at: any;
  stars: number;
  isStarred: boolean;
  translations: profileFields_hiddenTags_tag_translations[];
}

export interface profileFields_hiddenTags {
  __typename: "TagHidden";
  isBlur: boolean;
  tag: profileFields_hiddenTags_tag;
}

export interface profileFields_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface profileFields_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface profileFields_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: profileFields_resourceLists_resources_translations[];
}

export interface profileFields_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: profileFields_resourceLists_translations[];
  resources: profileFields_resourceLists_resources[];
}

export interface profileFields {
  __typename: "Profile";
  id: string;
  username: string | null;
  emails: profileFields_emails[];
  wallets: profileFields_wallets[];
  theme: string;
  translations: profileFields_translations[];
  starredTags: profileFields_starredTags[] | null;
  hiddenTags: profileFields_hiddenTags[] | null;
  resourceLists: profileFields_resourceLists[];
}
