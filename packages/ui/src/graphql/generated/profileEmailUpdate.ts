/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProfileEmailUpdateInput, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: profileEmailUpdate
// ====================================================

export interface profileEmailUpdate_profileEmailUpdate_emails {
  __typename: "Email";
  id: string;
  emailAddress: string;
  receivesAccountUpdates: boolean;
  receivesBusinessUpdates: boolean;
  verified: boolean;
}

export interface profileEmailUpdate_profileEmailUpdate_wallets_handles {
  __typename: "Handle";
  id: string;
  handle: string;
}

export interface profileEmailUpdate_profileEmailUpdate_wallets {
  __typename: "Wallet";
  id: string;
  name: string | null;
  publicAddress: string | null;
  stakingAddress: string;
  handles: profileEmailUpdate_profileEmailUpdate_wallets_handles[];
  verified: boolean;
}

export interface profileEmailUpdate_profileEmailUpdate_translations {
  __typename: "UserTranslation";
  id: string;
  language: string;
  bio: string | null;
}

export interface profileEmailUpdate_profileEmailUpdate_starredTags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface profileEmailUpdate_profileEmailUpdate_starredTags {
  __typename: "Tag";
  tag: string;
  created_at: any;
  stars: number;
  isStarred: boolean;
  translations: profileEmailUpdate_profileEmailUpdate_starredTags_translations[];
}

export interface profileEmailUpdate_profileEmailUpdate_hiddenTags_tag_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface profileEmailUpdate_profileEmailUpdate_hiddenTags_tag {
  __typename: "Tag";
  tag: string;
  created_at: any;
  stars: number;
  isStarred: boolean;
  translations: profileEmailUpdate_profileEmailUpdate_hiddenTags_tag_translations[];
}

export interface profileEmailUpdate_profileEmailUpdate_hiddenTags {
  __typename: "TagHidden";
  id: string;
  isBlur: boolean;
  tag: profileEmailUpdate_profileEmailUpdate_hiddenTags_tag;
}

export interface profileEmailUpdate_profileEmailUpdate_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface profileEmailUpdate_profileEmailUpdate_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface profileEmailUpdate_profileEmailUpdate_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: profileEmailUpdate_profileEmailUpdate_resourceLists_resources_translations[];
}

export interface profileEmailUpdate_profileEmailUpdate_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: profileEmailUpdate_profileEmailUpdate_resourceLists_translations[];
  resources: profileEmailUpdate_profileEmailUpdate_resourceLists_resources[];
}

export interface profileEmailUpdate_profileEmailUpdate {
  __typename: "Profile";
  id: string;
  handle: string | null;
  name: string;
  emails: profileEmailUpdate_profileEmailUpdate_emails[];
  wallets: profileEmailUpdate_profileEmailUpdate_wallets[];
  theme: string;
  translations: profileEmailUpdate_profileEmailUpdate_translations[];
  starredTags: profileEmailUpdate_profileEmailUpdate_starredTags[] | null;
  hiddenTags: profileEmailUpdate_profileEmailUpdate_hiddenTags[] | null;
  resourceLists: profileEmailUpdate_profileEmailUpdate_resourceLists[];
}

export interface profileEmailUpdate {
  profileEmailUpdate: profileEmailUpdate_profileEmailUpdate;
}

export interface profileEmailUpdateVariables {
  input: ProfileEmailUpdateInput;
}
