/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL query operation: profile
// ====================================================

export interface profile_profile_emails {
  __typename: "Email";
  id: string;
  emailAddress: string;
  receivesAccountUpdates: boolean;
  receivesBusinessUpdates: boolean;
}

export interface profile_profile_wallets {
  __typename: "Wallet";
  publicAddress: string;
  verified: boolean;
}

export interface profile_profile_translations {
  __typename: "UserTranslation";
  id: string;
  language: string;
  bio: string | null;
}

export interface profile_profile_starredTags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface profile_profile_starredTags {
  __typename: "Tag";
  id: string;
  tag: string;
  created_at: any;
  stars: number;
  isStarred: boolean;
  translations: profile_profile_starredTags_translations[];
}

export interface profile_profile_hiddenTags_tag_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface profile_profile_hiddenTags_tag {
  __typename: "Tag";
  id: string;
  tag: string;
  created_at: any;
  stars: number;
  isStarred: boolean;
  translations: profile_profile_hiddenTags_tag_translations[];
}

export interface profile_profile_hiddenTags {
  __typename: "TagHidden";
  isBlur: boolean;
  tag: profile_profile_hiddenTags_tag;
}

export interface profile_profile_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface profile_profile_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: profile_profile_resources_translations[];
}

export interface profile_profile_resourcesLearning_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface profile_profile_resourcesLearning {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: profile_profile_resourcesLearning_translations[];
}

export interface profile_profile {
  __typename: "Profile";
  id: string;
  username: string | null;
  emails: profile_profile_emails[];
  wallets: profile_profile_wallets[];
  theme: string;
  translations: profile_profile_translations[];
  starredTags: profile_profile_starredTags[] | null;
  hiddenTags: profile_profile_hiddenTags[] | null;
  resources: profile_profile_resources[];
  resourcesLearning: profile_profile_resourcesLearning[];
}

export interface profile {
  profile: profile_profile;
}
