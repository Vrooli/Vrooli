/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

//==============================================================
// START Enums and Input Objects
//==============================================================

export enum CommentFor {
  Project = "Project",
  Routine = "Routine",
  Standard = "Standard",
}

export enum MemberRole {
  Admin = "Admin",
  Member = "Member",
  Owner = "Owner",
}

export enum NodeType {
  End = "End",
  Loop = "Loop",
  Redirect = "Redirect",
  RoutineList = "RoutineList",
  Start = "Start",
}

export enum OrganizationSortBy {
  AlphabeticalAsc = "AlphabeticalAsc",
  AlphabeticalDesc = "AlphabeticalDesc",
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  StarsAsc = "StarsAsc",
  StarsDesc = "StarsDesc",
}

export enum ProjectSortBy {
  AlphabeticalAsc = "AlphabeticalAsc",
  AlphabeticalDesc = "AlphabeticalDesc",
  CommentsAsc = "CommentsAsc",
  CommentsDesc = "CommentsDesc",
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  ForksAsc = "ForksAsc",
  ForksDesc = "ForksDesc",
  StarsAsc = "StarsAsc",
  StarsDesc = "StarsDesc",
  VotesAsc = "VotesAsc",
  VotesDesc = "VotesDesc",
}

export enum ReportFor {
  Comment = "Comment",
  Organization = "Organization",
  Project = "Project",
  Routine = "Routine",
  Standard = "Standard",
  Tag = "Tag",
  User = "User",
}

export enum ResourceFor {
  Organization = "Organization",
  Project = "Project",
  RoutineContextual = "RoutineContextual",
  RoutineExternal = "RoutineExternal",
  User = "User",
}

export enum ResourceSortBy {
  AlphabeticalAsc = "AlphabeticalAsc",
  AlphabeticalDesc = "AlphabeticalDesc",
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
}

export enum ResourceUsedFor {
  Community = "Community",
  Context = "Context",
  Donation = "Donation",
  Learning = "Learning",
  OfficialWebsite = "OfficialWebsite",
  Proposal = "Proposal",
  Related = "Related",
  Social = "Social",
  Tutorial = "Tutorial",
}

export enum RoutineSortBy {
  AlphabeticalAsc = "AlphabeticalAsc",
  AlphabeticalDesc = "AlphabeticalDesc",
  CommentsAsc = "CommentsAsc",
  CommentsDesc = "CommentsDesc",
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  ForksAsc = "ForksAsc",
  ForksDesc = "ForksDesc",
  StarsAsc = "StarsAsc",
  StarsDesc = "StarsDesc",
  VotesAsc = "VotesAsc",
  VotesDesc = "VotesDesc",
}

export enum StandardSortBy {
  AlphabeticalAsc = "AlphabeticalAsc",
  AlphabeticalDesc = "AlphabeticalDesc",
  CommentsAsc = "CommentsAsc",
  CommentsDesc = "CommentsDesc",
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  StarsAsc = "StarsAsc",
  StarsDesc = "StarsDesc",
  VotesAsc = "VotesAsc",
  VotesDesc = "VotesDesc",
}

export enum StandardType {
  Array = "Array",
  Boolean = "Boolean",
  File = "File",
  Number = "Number",
  Object = "Object",
  String = "String",
  Url = "Url",
}

export enum StarFor {
  Comment = "Comment",
  Organization = "Organization",
  Project = "Project",
  Routine = "Routine",
  Standard = "Standard",
  Tag = "Tag",
  User = "User",
}

export enum TagSortBy {
  AlphabeticalAsc = "AlphabeticalAsc",
  AlphabeticalDesc = "AlphabeticalDesc",
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  StarsAsc = "StarsAsc",
  StarsDesc = "StarsDesc",
}

export enum UserSortBy {
  AlphabeticalAsc = "AlphabeticalAsc",
  AlphabeticalDesc = "AlphabeticalDesc",
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  StarsAsc = "StarsAsc",
  StarsDesc = "StarsDesc",
}

export enum VoteFor {
  Comment = "Comment",
  Project = "Project",
  Routine = "Routine",
  Standard = "Standard",
  Tag = "Tag",
}

export interface AutocompleteInput {
  searchString: string;
  take?: number | null;
}

export interface CommentCreateInput {
  text: string;
  createdFor: CommentFor;
  forId: string;
}

export interface CommentUpdateInput {
  id: string;
  text?: string | null;
}

export interface DeleteManyInput {
  ids: string[];
}

export interface DeleteOneInput {
  id: string;
}

export interface EmailCreateInput {
  emailAddress: string;
  receivesAccountUpdates?: boolean | null;
  receivesBusinessUpdates?: boolean | null;
}

export interface EmailLogInInput {
  email?: string | null;
  password?: string | null;
  verificationCode?: string | null;
}

export interface EmailRequestPasswordChangeInput {
  email: string;
}

export interface EmailResetPasswordInput {
  id: string;
  code: string;
  newPassword: string;
}

export interface EmailSignUpInput {
  username: string;
  email: string;
  theme: string;
  marketingEmails: boolean;
  password: string;
  confirmPassword: string;
}

export interface EmailUpdateInput {
  id: string;
  receivesAccountUpdates?: boolean | null;
  receivesBusinessUpdates?: boolean | null;
}

export interface FeedbackInput {
  text: string;
  userId?: string | null;
}

export interface FindByIdInput {
  id: string;
}

export interface InputItemCreateInput {
  description?: string | null;
  isRequired?: boolean | null;
  name?: string | null;
  standardConnect?: string | null;
  standardCreate?: StandardCreateInput | null;
}

export interface InputItemUpdateInput {
  id: string;
  description?: string | null;
  isRequired?: boolean | null;
  name?: string | null;
  standardConnect?: string | null;
  standardCreate?: StandardCreateInput | null;
}

export interface NodeCreateInput {
  description?: string | null;
  title?: string | null;
  type?: NodeType | null;
  nodeEndCreate?: NodeEndCreateInput | null;
  nodeLoopCreate?: NodeLoopCreateInput | null;
  nodeRoutineListCreate?: NodeRoutineListCreateInput | null;
  routineId: string;
}

export interface NodeEndCreateInput {
  wasSuccessful?: boolean | null;
}

export interface NodeEndUpdateInput {
  id: string;
  wasSuccessful?: boolean | null;
}

export interface NodeLinkConditionCaseCreateInput {
  condition: string;
}

export interface NodeLinkConditionCaseUpdateInput {
  id: string;
  condition?: string | null;
}

export interface NodeLinkConditionCreateInput {
  description?: string | null;
  title: string;
  whenCreate: NodeLinkConditionCaseCreateInput[];
  toId?: string | null;
}

export interface NodeLinkConditionUpdateInput {
  id: string;
  description?: string | null;
  title?: string | null;
  whenCreate?: NodeLinkConditionCaseCreateInput[] | null;
  whenUpdate?: NodeLinkConditionCaseUpdateInput[] | null;
  whenDelete?: string[] | null;
  toId?: string | null;
}

export interface NodeLinkCreateInput {
  conditions: NodeLinkConditionCreateInput[];
  fromId: string;
  toId: string;
}

export interface NodeLinkUpdateInput {
  id: string;
  conditionsCreate?: NodeLinkConditionCreateInput[] | null;
  conditionsUpdate?: NodeLinkConditionUpdateInput[] | null;
  conditionsDelete?: string[] | null;
  fromId?: string | null;
  toId?: string | null;
}

export interface NodeLoopCreateInput {
  loops?: number | null;
  maxLoops?: number | null;
  whilesCreate: NodeLoopWhileCreateInput[];
}

export interface NodeLoopUpdateInput {
  id: string;
  loops?: number | null;
  maxLoops?: number | null;
  whilesCreate: NodeLoopWhileCreateInput[];
  whilesUpdate: NodeLoopWhileUpdateInput[];
  whilesDelete?: string[] | null;
}

export interface NodeLoopWhileCaseCreateInput {
  condition: string;
}

export interface NodeLoopWhileCaseUpdateInput {
  id: string;
  condition?: string | null;
}

export interface NodeLoopWhileCreateInput {
  description?: string | null;
  title: string;
  whenCreate: NodeLoopWhileCaseCreateInput[];
  toId?: string | null;
}

export interface NodeLoopWhileUpdateInput {
  id: string;
  description?: string | null;
  title?: string | null;
  whenCreate?: NodeLoopWhileCaseCreateInput[] | null;
  whenUpdate?: NodeLoopWhileCaseUpdateInput[] | null;
  whenDelete?: string[] | null;
  toId?: string | null;
}

export interface NodeRoutineListCreateInput {
  isOrdered?: boolean | null;
  isOptional?: boolean | null;
  routinesConnect?: string[] | null;
  routinesCreate?: NodeRoutineListItemCreateInput[] | null;
}

export interface NodeRoutineListItemCreateInput {
  description?: string | null;
  isOptional?: boolean | null;
  title?: string | null;
  routineConnect: string;
}

export interface NodeRoutineListItemUpdateInput {
  id: string;
  description?: string | null;
  isOptional?: boolean | null;
  title?: string | null;
  routineConnect?: string | null;
}

export interface NodeRoutineListUpdateInput {
  id: string;
  isOrdered?: boolean | null;
  isOptional?: boolean | null;
  routinesConnect?: string[] | null;
  routinesDisconnect?: string[] | null;
  routinesDelete?: string[] | null;
  routinesCreate?: NodeRoutineListItemCreateInput[] | null;
  routinesUpdate?: NodeRoutineListItemUpdateInput[] | null;
}

export interface NodeUpdateInput {
  id: string;
  description?: string | null;
  title?: string | null;
  type?: NodeType | null;
  nodeEndCreate?: NodeEndCreateInput | null;
  nodeEndUpdate?: NodeEndUpdateInput | null;
  nodeLoopCreate?: NodeLoopCreateInput | null;
  nodeLoopUpdate?: NodeLoopUpdateInput | null;
  nodeRoutineListCreate?: NodeRoutineListCreateInput | null;
  nodeRoutineListUpdate?: NodeRoutineListUpdateInput | null;
  routineId?: string | null;
}

export interface OrganizationCountInput {
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface OrganizationCreateInput {
  bio?: string | null;
  name: string;
  resourcesCreate?: ResourceCreateInput[] | null;
  tagsConnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
}

export interface OrganizationSearchInput {
  userId?: string | null;
  projectId?: string | null;
  routineId?: string | null;
  standardId?: string | null;
  reportId?: string | null;
  ids?: string[] | null;
  sortBy?: OrganizationSortBy | null;
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
  searchString?: string | null;
  after?: string | null;
  take?: number | null;
}

export interface OrganizationUpdateInput {
  id: string;
  bio?: string | null;
  name?: string | null;
  membersConnect?: string[] | null;
  membersDisconnect?: string[] | null;
  resourcesDelete?: string[] | null;
  resourcesCreate?: ResourceCreateInput[] | null;
  resourcesUpdate?: ResourceUpdateInput[] | null;
  tagsConnect?: string[] | null;
  tagsDisconnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
}

export interface OutputItemCreateInput {
  description?: string | null;
  name?: string | null;
  standardConnect?: string | null;
  standardCreate?: StandardCreateInput | null;
}

export interface OutputItemUpdateInput {
  id: string;
  description?: string | null;
  name?: string | null;
  standardConnect?: string | null;
  standardCreate?: StandardCreateInput | null;
}

export interface ProfileUpdateInput {
  username?: string | null;
  bio?: string | null;
  theme?: string | null;
  starredTagsConnect?: string[] | null;
  starredTagsDisconnect?: string[] | null;
  starredTagsCreate?: TagCreateInput[] | null;
  hiddenTagsConnect?: string[] | null;
  hiddenTagsDisconnect?: string[] | null;
  hiddenTagsCreate?: TagCreateInput[] | null;
}

export interface ProjectCountInput {
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface ProjectCreateInput {
  description?: string | null;
  name: string;
  parentId?: string | null;
  createdByUserId?: string | null;
  createdByOrganizationId?: string | null;
  resourcesCreate?: ResourceCreateInput[] | null;
  tagsConnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
}

export interface ProjectSearchInput {
  userId?: string | null;
  organizationId?: string | null;
  parentId?: string | null;
  reportId?: string | null;
  ids?: string[] | null;
  sortBy?: ProjectSortBy | null;
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
  searchString?: string | null;
  after?: string | null;
  take?: number | null;
}

export interface ProjectUpdateInput {
  id: string;
  description?: string | null;
  name?: string | null;
  parentId?: string | null;
  userId?: string | null;
  organizationId?: string | null;
  resourcesDelete?: string[] | null;
  resourcesCreate?: ResourceCreateInput[] | null;
  resourcesUpdate?: ResourceUpdateInput[] | null;
  tagsConnect?: string[] | null;
  tagsDisconnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
}

export interface ReadAssetsInput {
  files: string[];
}

export interface ReadOpenGraphInput {
  url: string;
}

export interface ReportCreateInput {
  createdFor: ReportFor;
  createdForId: string;
  details?: string | null;
  reason: string;
}

export interface ReportUpdateInput {
  id: string;
  details?: string | null;
  reason?: string | null;
}

export interface ResourceCountInput {
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface ResourceCreateInput {
  createdFor: ResourceFor;
  createdForId: string;
  description?: string | null;
  link: string;
  title?: string | null;
  usedFor: ResourceUsedFor;
}

export interface ResourceSearchInput {
  forId?: string | null;
  forType?: ResourceFor | null;
  ids?: string[] | null;
  sortBy?: ResourceSortBy | null;
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
  searchString?: string | null;
  after?: string | null;
  take?: number | null;
}

export interface ResourceUpdateInput {
  id: string;
  createdFor?: ResourceFor | null;
  createdForId?: string | null;
  description?: string | null;
  link?: string | null;
  title?: string | null;
  usedFor?: ResourceUsedFor | null;
}

export interface RoutineCountInput {
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface RoutineCreateInput {
  description?: string | null;
  instructions?: string | null;
  isAutomatable?: boolean | null;
  isInternal?: boolean | null;
  title: string;
  version?: string | null;
  parentId?: string | null;
  projectId?: string | null;
  createdByUserId?: string | null;
  createdByOrganizationId?: string | null;
  nodesCreate?: NodeCreateInput[] | null;
  nodeLinksCreate?: NodeLinkCreateInput[] | null;
  inputsCreate?: InputItemCreateInput[] | null;
  outputsCreate?: OutputItemCreateInput[] | null;
  resourcesContextualCreate?: ResourceCreateInput[] | null;
  resourcesExternalCreate?: ResourceCreateInput[] | null;
  tagsConnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
}

export interface RoutineSearchInput {
  userId?: string | null;
  organizationId?: string | null;
  projectId?: string | null;
  parentId?: string | null;
  reportId?: string | null;
  ids?: string[] | null;
  sortBy?: RoutineSortBy | null;
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
  searchString?: string | null;
  after?: string | null;
  take?: number | null;
}

export interface RoutineUpdateInput {
  id: string;
  description?: string | null;
  instructions?: string | null;
  isAutomatable?: boolean | null;
  isInternal?: boolean | null;
  title?: string | null;
  version?: string | null;
  parentId?: string | null;
  userId?: string | null;
  organizationId?: string | null;
  nodesDelete?: string[] | null;
  nodesCreate?: NodeCreateInput[] | null;
  nodesUpdate?: NodeUpdateInput[] | null;
  nodeLinksDelete?: string[] | null;
  nodeLinksCreate?: NodeLinkCreateInput[] | null;
  nodeLinksUpdate?: NodeLinkUpdateInput[] | null;
  inputsCreate?: InputItemCreateInput[] | null;
  inputsUpdate?: InputItemUpdateInput[] | null;
  inputsDelete?: string[] | null;
  outputsCreate?: OutputItemCreateInput[] | null;
  outputsUpdate?: OutputItemUpdateInput[] | null;
  outputsDelete?: string[] | null;
  resourcesContextualDelete?: string[] | null;
  resourcesContextualCreate?: ResourceCreateInput[] | null;
  resourcesContextualUpdate?: ResourceUpdateInput[] | null;
  resourcesExternalDelete?: string[] | null;
  resourcesExternalCreate?: ResourceCreateInput[] | null;
  resourcesExternalUpdate?: ResourceUpdateInput[] | null;
  tagsConnect?: string[] | null;
  tagsDisconnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
}

export interface StandardCountInput {
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface StandardCreateInput {
  default?: string | null;
  description?: string | null;
  isFile?: boolean | null;
  name: string;
  schema?: string | null;
  type?: StandardType | null;
  version?: string | null;
  createdByUserId?: string | null;
  createdByOrganizationId?: string | null;
  tagsConnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
}

export interface StandardSearchInput {
  userId?: string | null;
  organizationId?: string | null;
  projectId?: string | null;
  routineId?: string | null;
  reportId?: string | null;
  ids?: string[] | null;
  sortBy?: StandardSortBy | null;
  searchString?: string | null;
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
  after?: string | null;
  take?: number | null;
}

export interface StandardUpdateInput {
  id: string;
  description?: string | null;
  makeAnonymous?: boolean | null;
  tagsConnect?: string[] | null;
  tagsDisconnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
}

export interface StarInput {
  isStar: boolean;
  starFor: StarFor;
  forId: string;
}

export interface TagCountInput {
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface TagCreateInput {
  anonymous?: boolean | null;
  description?: string | null;
  tag: string;
}

export interface TagSearchInput {
  myTags?: boolean | null;
  hidden?: boolean | null;
  ids?: string[] | null;
  sortBy?: TagSortBy | null;
  searchString?: string | null;
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
  after?: string | null;
  take?: number | null;
}

export interface TagUpdateInput {
  id: string;
  anonymous?: boolean | null;
  description?: string | null;
  tag?: string | null;
}

export interface TimeFrame {
  after?: any | null;
  before?: any | null;
}

export interface UserCountInput {
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface UserDeleteInput {
  password: string;
}

export interface UserSearchInput {
  organizationId?: string | null;
  projectId?: string | null;
  routineId?: string | null;
  reportId?: string | null;
  standardId?: string | null;
  ids?: string[] | null;
  sortBy?: UserSortBy | null;
  searchString?: string | null;
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
  after?: string | null;
  take?: number | null;
}

export interface VoteInput {
  isUpvote?: boolean | null;
  voteFor: VoteFor;
  forId: string;
}

export interface WalletCompleteInput {
  publicAddress: string;
  signedPayload: string;
}

export interface WalletInitInput {
  publicAddress: string;
  nonceDescription?: string | null;
}

export interface WriteAssetsInput {
  files: any[];
}

//==============================================================
// END Enums and Input Objects
//==============================================================
