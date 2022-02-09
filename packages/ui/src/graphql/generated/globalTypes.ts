/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

//==============================================================
// START Enums and Input Objects
//==============================================================

export enum CommentFor {
  Organization = "Organization",
  Project = "Project",
  Routine = "Routine",
  Standard = "Standard",
  User = "User",
}

export enum NodeType {
  Combine = "Combine",
  Decision = "Decision",
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

export interface NodeCombineCreateInput {
  from: string[];
  toId?: string | null;
}

export interface NodeCombineUpdateInput {
  id: string;
  from?: string[] | null;
  toId?: string | null;
}

export interface NodeCreateInput {
  description?: string | null;
  title?: string | null;
  type?: NodeType | null;
  nodeCombineCreate?: NodeCombineCreateInput | null;
  nodeDecisionCreate?: NodeDecisionCreateInput | null;
  nodeEndCreate?: NodeEndCreateInput | null;
  nodeLoopCreate?: NodeLoopCreateInput | null;
  nodeRedirectCreate?: NodeRoutineListCreateInput | null;
  nodeRoutineListCreate?: NodeRedirectCreateInput | null;
  nodeStartCreate?: NodeStartCreateInput | null;
  previousId?: string | null;
  nextId?: string | null;
  routineId: string;
}

export interface NodeDecisionCreateInput {
  decisionsCreate: NodeDecisionItemCreateInput[];
}

export interface NodeDecisionItemCreateInput {
  description?: string | null;
  title: string;
  whenCreate: NodeDecisionItemWhenCreateInput[];
  toId?: string | null;
}

export interface NodeDecisionItemUpdateInput {
  id: string;
  description?: string | null;
  title?: string | null;
  whenCreate?: NodeDecisionItemWhenCreateInput[] | null;
  whenUpdate?: NodeDecisionItemWhenUpdateInput[] | null;
  whenDelete?: string[] | null;
  toId?: string | null;
}

export interface NodeDecisionItemWhenCreateInput {
  condition: string;
}

export interface NodeDecisionItemWhenUpdateInput {
  id: string;
  condition?: string | null;
}

export interface NodeDecisionUpdateInput {
  id: string;
  decisionsCreate?: NodeDecisionItemCreateInput[] | null;
  decisionsUpdate?: NodeDecisionItemUpdateInput[] | null;
  decisionsDelete?: string[] | null;
}

export interface NodeEndCreateInput {
  wasSuccessful?: boolean | null;
}

export interface NodeEndUpdateInput {
  id: string;
  wasSuccessful?: boolean | null;
}

export interface NodeLoopCreateInput {
  id?: string | null;
}

export interface NodeLoopUpdateInput {
  id?: string | null;
}

export interface NodeRedirectCreateInput {
  id?: string | null;
}

export interface NodeRedirectUpdateInput {
  id?: string | null;
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

export interface NodeStartCreateInput {
  _blank?: string | null;
}

export interface NodeStartUpdateInput {
  id: string;
}

export interface NodeUpdateInput {
  id: string;
  description?: string | null;
  title?: string | null;
  type?: NodeType | null;
  nodeCombineCreate?: NodeCombineCreateInput | null;
  nodeCombineUpdate?: NodeCombineUpdateInput | null;
  nodeDecisionCreate?: NodeDecisionCreateInput | null;
  nodeDecisionUpdate?: NodeDecisionUpdateInput | null;
  nodeEndCreate?: NodeEndCreateInput | null;
  nodeEndUpdate?: NodeEndUpdateInput | null;
  nodeLoopCreate?: NodeLoopCreateInput | null;
  nodeLoopUpdate?: NodeLoopUpdateInput | null;
  nodeRedirectCreate?: NodeRoutineListCreateInput | null;
  nodeRedirectUpdate?: NodeRoutineListUpdateInput | null;
  nodeRoutineListCreate?: NodeRedirectCreateInput | null;
  nodeRoutineListUpdate?: NodeRedirectUpdateInput | null;
  nodeStartCreate?: NodeStartCreateInput | null;
  nodeStartUpdate?: NodeStartUpdateInput | null;
  previousId?: string | null;
  nextId?: string | null;
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
  emails?: EmailUpdateInput[] | null;
  theme?: string | null;
  currentPassword: string;
  newPassword?: string | null;
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
  usedFor?: ResourceUsedFor | null;
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
  title: string;
  version?: string | null;
  parentId?: string | null;
  createdByUserId?: string | null;
  createdByOrganizationId?: string | null;
  nodesConnect?: string[] | null;
  nodesCreate?: NodeCreateInput[] | null;
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
  title?: string | null;
  version?: string | null;
  parentId?: string | null;
  userId?: string | null;
  organizationId?: string | null;
  nodesConnect?: string[] | null;
  nodesDisconnect?: string[] | null;
  nodesDelete?: string[] | null;
  nodesCreate?: NodeCreateInput[] | null;
  nodesUpdate?: NodeUpdateInput[] | null;
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
  id?: string | null;
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
