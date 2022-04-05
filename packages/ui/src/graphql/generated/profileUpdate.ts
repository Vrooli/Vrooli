/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProfileUpdateInput, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: profileUpdate
// ====================================================

export interface profileUpdate_profileUpdate_emails {
  __typename: "Email";
  id: string;
  emailAddress: string;
  receivesAccountUpdates: boolean;
  receivesBusinessUpdates: boolean;
  verified: boolean;
}

export interface profileUpdate_profileUpdate_wallets {
  __typename: "Wallet";
  id: string;
  name: string | null;
  publicAddress: string;
  verified: boolean;
}

export interface profileUpdate_profileUpdate_translations {
  __typename: "UserTranslation";
  id: string;
  language: string;
  bio: string | null;
}

export interface profileUpdate_profileUpdate_starredTags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface profileUpdate_profileUpdate_starredTags {
  __typename: "Tag";
  id: string;
  tag: string;
  created_at: any;
  stars: number;
  isStarred: boolean;
  translations: profileUpdate_profileUpdate_starredTags_translations[];
}

export interface profileUpdate_profileUpdate_hiddenTags_tag_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface profileUpdate_profileUpdate_hiddenTags_tag {
  __typename: "Tag";
  id: string;
  tag: string;
  created_at: any;
  stars: number;
  isStarred: boolean;
  translations: profileUpdate_profileUpdate_hiddenTags_tag_translations[];
}

export interface profileUpdate_profileUpdate_hiddenTags {
  __typename: "TagHidden";
  isBlur: boolean;
  tag: profileUpdate_profileUpdate_hiddenTags_tag;
}

export interface profileUpdate_profileUpdate_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface profileUpdate_profileUpdate_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface profileUpdate_profileUpdate_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: profileUpdate_profileUpdate_resourceLists_resources_translations[];
}

export interface profileUpdate_profileUpdate_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: profileUpdate_profileUpdate_resourceLists_translations[];
  resources: profileUpdate_profileUpdate_resourceLists_resources[];
}

export interface profileUpdate_profileUpdate {
  __typename: "Profile";
  id: string;
  username: string | null;
  emails: profileUpdate_profileUpdate_emails[];
  wallets: profileUpdate_profileUpdate_wallets[];
  theme: string;
  translations: profileUpdate_profileUpdate_translations[];
  starredTags: profileUpdate_profileUpdate_starredTags[] | null;
  hiddenTags: profileUpdate_profileUpdate_hiddenTags[] | null;
  resourceLists: profileUpdate_profileUpdate_resourceLists[];
}

export interface profileUpdate {
  profileUpdate: profileUpdate_profileUpdate;
}

export interface profileUpdateVariables {
  input: ProfileUpdateInput;
}
