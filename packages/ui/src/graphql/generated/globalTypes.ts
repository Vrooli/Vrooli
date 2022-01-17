/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

//==============================================================
// START Enums and Input Objects
//==============================================================

export enum AccountStatus {
  Deleted = "Deleted",
  HardLocked = "HardLocked",
  SoftLocked = "SoftLocked",
  Unlocked = "Unlocked",
}

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

export interface CommentInput {
  id?: string | null;
  text?: string | null;
  createdFor: CommentFor;
  forId: string;
}

export interface DeleteCommentInput {
  id: string;
  createdFor: CommentFor;
  forId: string;
}

export interface DeleteManyInput {
  ids: string[];
}

export interface DeleteOneInput {
  id: string;
}

export interface EmailInput {
  id?: string | null;
  emailAddress: string;
  receivesAccountUpdates?: boolean | null;
  receivesBusinessUpdates?: boolean | null;
  userId?: string | null;
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
}

export interface FeedbackInput {
  text: string;
  userId?: string | null;
}

export interface FindByIdInput {
  id: string;
}

export interface NodeCombineInput {
  id?: string | null;
  from?: string[] | null;
  to?: string | null;
}

export interface NodeDecisionInput {
  id?: string | null;
  decisions: NodeDecisionItemInput[];
}

export interface NodeDecisionItemCaseInput {
  id?: string | null;
  condition?: string | null;
}

export interface NodeDecisionItemInput {
  id?: string | null;
  title?: string | null;
  description?: string | null;
  toId?: string | null;
  when?: (NodeDecisionItemCaseInput | null)[] | null;
}

export interface NodeEndInput {
  id?: string | null;
  wasSuccessful?: boolean | null;
}

export interface NodeInput {
  id?: string | null;
  routineId?: string | null;
  title?: string | null;
  description?: string | null;
  type?: NodeType | null;
  combineData?: NodeCombineInput | null;
  decisionData?: NodeDecisionInput | null;
  endData?: NodeEndInput | null;
  loopData?: NodeLoopInput | null;
  routineListData?: NodeRoutineListInput | null;
  redirectData?: NodeRedirectInput | null;
  startData?: NodeStartInput | null;
}

export interface NodeLoopInput {
  id?: string | null;
}

export interface NodeRedirectInput {
  id?: string | null;
}

export interface NodeRoutineListInput {
  id?: string | null;
  isOrdered?: boolean | null;
  isOptional?: boolean | null;
  routines: NodeRoutineListItemInput[];
}

export interface NodeRoutineListItemInput {
  id?: string | null;
  title?: string | null;
  description?: string | null;
  isOptional?: boolean | null;
  listId?: string | null;
  routineId?: string | null;
}

export interface NodeStartInput {
  id?: string | null;
}

export interface OrganizationCountInput {
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface OrganizationInput {
  id?: string | null;
  name: string;
  bio?: string | null;
  resources?: ResourceInput[] | null;
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

export interface ProfileUpdateInput {
  data: UserInput;
  currentPassword: string;
  newPassword?: string | null;
}

export interface ProjectCountInput {
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface ProjectInput {
  id?: string | null;
  name: string;
  description?: string | null;
  organizations?: OrganizationInput[] | null;
  users?: UserInput[] | null;
  resources?: ResourceInput[] | null;
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

export interface ReadAssetsInput {
  files: string[];
}

export interface ReadOpenGraphInput {
  url: string;
}

export interface ReportInput {
  id?: string | null;
  reason: string;
  details?: string | null;
  createdFor: ReportFor;
  forId: string;
}

export interface ResourceCountInput {
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface ResourceInput {
  id?: string | null;
  title: string;
  description?: string | null;
  link: string;
  displayUrl?: string | null;
  usedFor?: ResourceUsedFor | null;
  createdFor: ResourceFor;
  forId: string;
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

export interface RoutineCountInput {
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface RoutineInput {
  id?: string | null;
  version?: string | null;
  title?: string | null;
  description?: string | null;
  instructions?: string | null;
  isAutomatable?: boolean | null;
  inputs?: RoutineInputItemInput[] | null;
  outputs?: RoutineOutputItemInput[] | null;
}

export interface RoutineInputItemInput {
  id?: string | null;
  routineId: string;
  standardId?: string | null;
}

export interface RoutineOutputItemInput {
  id?: string | null;
  routineId: string;
  standardId?: string | null;
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

export interface StandardCountInput {
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface StandardInput {
  id?: string | null;
  name?: string | null;
  description?: string | null;
  type?: StandardType | null;
  schema?: string | null;
  default?: string | null;
  isFile?: boolean | null;
  tags?: TagInput[] | null;
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

export interface TagCountInput {
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface TagInput {
  id?: string | null;
}

export interface TagSearchInput {
  userId?: string | null;
  ids?: string[] | null;
  sortBy?: TagSortBy | null;
  searchString?: string | null;
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
  after?: string | null;
  take?: number | null;
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
  id: string;
  password: string;
}

export interface UserInput {
  id?: string | null;
  username?: string | null;
  bio?: string | null;
  emails?: EmailInput[] | null;
  theme?: string | null;
  status?: AccountStatus | null;
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
  isUpvote: boolean;
  voteFor: VoteFor;
  forId: string;
}

export interface VoteRemoveInput {
  voteFor: VoteFor;
  forId: string;
}

export interface WalletCompleteInput {
  publicAddress: string;
  signedMessage: string;
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
