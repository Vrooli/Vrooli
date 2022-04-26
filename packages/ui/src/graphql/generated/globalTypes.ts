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

export enum LogSortBy {
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
}

export enum LogType {
  Create = "Create",
  Delete = "Delete",
  Downvote = "Downvote",
  OrganizationAddMember = "OrganizationAddMember",
  OrganizationJoin = "OrganizationJoin",
  OrganizationLeave = "OrganizationLeave",
  OrganizationRemoveMember = "OrganizationRemoveMember",
  OrganizationUpdateMember = "OrganizationUpdateMember",
  ProjectComplete = "ProjectComplete",
  RemoveStar = "RemoveStar",
  RemoveVote = "RemoveVote",
  RoutineCancel = "RoutineCancel",
  RoutineComplete = "RoutineComplete",
  RoutineStartCanceled = "RoutineStartCanceled",
  RoutineStartCompleted = "RoutineStartCompleted",
  RoutineStartIncomplete = "RoutineStartIncomplete",
  Star = "Star",
  Update = "Update",
  Upvote = "Upvote",
  View = "View",
}

export enum MemberRole {
  Admin = "Admin",
  Member = "Member",
  Owner = "Owner",
}

export enum NodeType {
  End = "End",
  Redirect = "Redirect",
  RoutineList = "RoutineList",
  Start = "Start",
}

export enum OrganizationSortBy {
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  StarsAsc = "StarsAsc",
  StarsDesc = "StarsDesc",
}

export enum ProjectSortBy {
  CommentsAsc = "CommentsAsc",
  CommentsDesc = "CommentsDesc",
  DateCompletedAsc = "DateCompletedAsc",
  DateCompletedDesc = "DateCompletedDesc",
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
  Routine = "Routine",
  User = "User",
}

export enum ResourceListUsedFor {
  Custom = "Custom",
  Develop = "Develop",
  Display = "Display",
  Learn = "Learn",
  Research = "Research",
}

export enum ResourceSortBy {
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  IndexAsc = "IndexAsc",
  IndexDesc = "IndexDesc",
}

export enum ResourceUsedFor {
  Community = "Community",
  Context = "Context",
  Developer = "Developer",
  Donation = "Donation",
  ExternalService = "ExternalService",
  Feed = "Feed",
  Install = "Install",
  Learning = "Learning",
  Notes = "Notes",
  OfficialWebsite = "OfficialWebsite",
  Proposal = "Proposal",
  Related = "Related",
  Researching = "Researching",
  Scheduling = "Scheduling",
  Social = "Social",
  Tutorial = "Tutorial",
}

export enum RoutineSortBy {
  CommentsAsc = "CommentsAsc",
  CommentsDesc = "CommentsDesc",
  DateCompletedAsc = "DateCompletedAsc",
  DateCompletedDesc = "DateCompletedDesc",
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
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  StarsAsc = "StarsAsc",
  StarsDesc = "StarsDesc",
}

export enum UserSortBy {
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

export interface CommentCreateInput {
  createdFor: CommentFor;
  forId: string;
  translationsCreate?: CommentTranslationCreateInput[] | null;
}

export interface CommentTranslationCreateInput {
  language: string;
  text: string;
}

export interface CommentTranslationUpdateInput {
  id: string;
  language?: string | null;
  text?: string | null;
}

export interface CommentUpdateInput {
  id: string;
  translationsDelete?: string[] | null;
  translationsCreate?: CommentTranslationCreateInput[] | null;
  translationsUpdate?: CommentTranslationUpdateInput[] | null;
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
  name: string;
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

export interface FindHandlesInput {
  organizationId?: string | null;
}

export interface ForYouPageInput {
  searchString: string;
  take?: number | null;
}

export interface HomePageInput {
  searchString: string;
  take?: number | null;
}

export interface InputItemCreateInput {
  isRequired?: boolean | null;
  name?: string | null;
  standardConnect?: string | null;
  standardCreate?: StandardCreateInput | null;
  translationsDelete?: string[] | null;
  translationsCreate?: InputItemTranslationCreateInput[] | null;
  translationsUpdate?: InputItemTranslationUpdateInput[] | null;
}

export interface InputItemTranslationCreateInput {
  language: string;
  description?: string | null;
}

export interface InputItemTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
}

export interface InputItemUpdateInput {
  id: string;
  isRequired?: boolean | null;
  name?: string | null;
  standardConnect?: string | null;
  standardCreate?: StandardCreateInput | null;
  translationsDelete?: string[] | null;
  translationsCreate?: InputItemTranslationCreateInput[] | null;
  translationsUpdate?: InputItemTranslationUpdateInput[] | null;
}

export interface LogSearchInput {
  actions?: string[] | null;
  after?: string | null;
  createdTimeFrame?: TimeFrame | null;
  data?: string | null;
  ids?: string[] | null;
  sortBy?: LogSortBy | null;
  object1Type?: string | null;
  object1Id?: string | null;
  object2Type?: string | null;
  object2Id?: string | null;
  take?: number | null;
}

export interface LoopCreateInput {
  loops?: number | null;
  maxLoops?: number | null;
  operation?: string | null;
  whilesCreate: LoopWhileCreateInput[];
}

export interface LoopUpdateInput {
  id: string;
  loops?: number | null;
  maxLoops?: number | null;
  operation?: string | null;
  whilesCreate: LoopWhileCreateInput[];
  whilesUpdate: LoopWhileUpdateInput[];
  whilesDelete?: string[] | null;
}

export interface LoopWhileCreateInput {
  translationsCreate?: LoopWhileTranslationCreateInput[] | null;
  condition: string;
  toId?: string | null;
}

export interface LoopWhileTranslationCreateInput {
  language: string;
  description?: string | null;
  title: string;
}

export interface LoopWhileTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
  title?: string | null;
}

export interface LoopWhileUpdateInput {
  id: string;
  toId?: string | null;
  translationsDelete?: string[] | null;
  translationsCreate?: LoopWhileTranslationCreateInput[] | null;
  translationsUpdate?: LoopWhileTranslationUpdateInput[] | null;
  condition?: string | null;
}

export interface NodeCreateInput {
  id: string;
  columnIndex?: number | null;
  rowIndex?: number | null;
  type?: NodeType | null;
  loopCreate?: LoopCreateInput | null;
  nodeEndCreate?: NodeEndCreateInput | null;
  nodeRoutineListCreate?: NodeRoutineListCreateInput | null;
  routineId?: string | null;
  translationsCreate?: NodeTranslationCreateInput[] | null;
}

export interface NodeEndCreateInput {
  wasSuccessful?: boolean | null;
}

export interface NodeEndUpdateInput {
  id: string;
  wasSuccessful?: boolean | null;
}

export interface NodeLinkCreateInput {
  whens?: NodeLinkWhenCreateInput[] | null;
  operation?: string | null;
  fromId: string;
  toId: string;
}

export interface NodeLinkUpdateInput {
  id: string;
  whensCreate?: NodeLinkWhenCreateInput[] | null;
  whensUpdate?: NodeLinkWhenUpdateInput[] | null;
  whensDelete?: string[] | null;
  operation?: string | null;
  fromId?: string | null;
  toId?: string | null;
}

export interface NodeLinkWhenCreateInput {
  toId?: string | null;
  translationsCreate?: NodeLinkWhenTranslationCreateInput[] | null;
  condition: string;
}

export interface NodeLinkWhenTranslationCreateInput {
  language: string;
  description?: string | null;
  title: string;
}

export interface NodeLinkWhenTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
  title?: string | null;
}

export interface NodeLinkWhenUpdateInput {
  id: string;
  toId?: string | null;
  translationsDelete?: string[] | null;
  translationsCreate?: NodeLinkWhenTranslationCreateInput[] | null;
  translationsUpdate?: NodeLinkWhenTranslationUpdateInput[] | null;
  condition?: string | null;
}

export interface NodeRoutineListCreateInput {
  isOrdered?: boolean | null;
  isOptional?: boolean | null;
  routinesCreate?: NodeRoutineListItemCreateInput[] | null;
}

export interface NodeRoutineListItemCreateInput {
  id?: string | null;
  index?: number | null;
  isOptional?: boolean | null;
  routineConnect: string;
  translationsCreate?: NodeRoutineListItemTranslationCreateInput[] | null;
}

export interface NodeRoutineListItemTranslationCreateInput {
  language: string;
  description?: string | null;
  title?: string | null;
}

export interface NodeRoutineListItemTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
  title?: string | null;
}

export interface NodeRoutineListItemUpdateInput {
  id: string;
  index?: number | null;
  isOptional?: boolean | null;
  translationsDelete?: string[] | null;
  translationsCreate?: NodeRoutineListItemTranslationCreateInput[] | null;
  translationsUpdate?: NodeRoutineListItemTranslationUpdateInput[] | null;
}

export interface NodeRoutineListUpdateInput {
  id: string;
  isOrdered?: boolean | null;
  isOptional?: boolean | null;
  routinesDelete?: string[] | null;
  routinesCreate?: NodeRoutineListItemCreateInput[] | null;
  routinesUpdate?: NodeRoutineListItemUpdateInput[] | null;
}

export interface NodeTranslationCreateInput {
  language: string;
  title: string;
  description?: string | null;
}

export interface NodeTranslationUpdateInput {
  id: string;
  language?: string | null;
  title?: string | null;
  description?: string | null;
}

export interface NodeUpdateInput {
  id: string;
  columnIndex?: number | null;
  rowIndex?: number | null;
  type?: NodeType | null;
  loopDelete?: string | null;
  loopCreate?: LoopCreateInput | null;
  loopUpdate?: LoopUpdateInput | null;
  nodeEndUpdate?: NodeEndUpdateInput | null;
  nodeRoutineListUpdate?: NodeRoutineListUpdateInput | null;
  routineId?: string | null;
  translationsDelete?: string[] | null;
  translationsCreate?: NodeTranslationCreateInput[] | null;
  translationsUpdate?: NodeTranslationUpdateInput[] | null;
}

export interface OrganizationCountInput {
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface OrganizationCreateInput {
  isOpenToNewMembers?: boolean | null;
  resourceListsCreate?: ResourceListCreateInput[] | null;
  tagsConnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
  translationsCreate?: OrganizationTranslationCreateInput[] | null;
}

export interface OrganizationSearchInput {
  after?: string | null;
  createdTimeFrame?: TimeFrame | null;
  ids?: string[] | null;
  isOpenToNewMembers?: boolean | null;
  languages?: string[] | null;
  minStars?: number | null;
  projectId?: string | null;
  reportId?: string | null;
  resourceLists?: string[] | null;
  resourceTypes?: ResourceUsedFor[] | null;
  routineId?: string | null;
  searchString?: string | null;
  standardId?: string | null;
  sortBy?: OrganizationSortBy | null;
  tags?: string[] | null;
  take?: number | null;
  updatedTimeFrame?: TimeFrame | null;
  userId?: string | null;
}

export interface OrganizationTranslationCreateInput {
  language: string;
  bio?: string | null;
  name: string;
}

export interface OrganizationTranslationUpdateInput {
  id: string;
  language?: string | null;
  bio?: string | null;
  name?: string | null;
}

export interface OrganizationUpdateInput {
  id: string;
  handle?: string | null;
  isOpenToNewMembers?: boolean | null;
  membersConnect?: string[] | null;
  membersDisconnect?: string[] | null;
  resourceListsDelete?: string[] | null;
  resourceListsCreate?: ResourceListCreateInput[] | null;
  resourceListsUpdate?: ResourceListUpdateInput[] | null;
  tagsConnect?: string[] | null;
  tagsDisconnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
  translationsDelete?: string[] | null;
  translationsCreate?: OrganizationTranslationCreateInput[] | null;
  translationsUpdate?: OrganizationTranslationUpdateInput[] | null;
}

export interface OutputItemCreateInput {
  name?: string | null;
  standardConnect?: string | null;
  standardCreate?: StandardCreateInput | null;
  translationsCreate?: OutputItemTranslationCreateInput[] | null;
}

export interface OutputItemTranslationCreateInput {
  language: string;
  description?: string | null;
}

export interface OutputItemTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
}

export interface OutputItemUpdateInput {
  id: string;
  name?: string | null;
  standardConnect?: string | null;
  standardCreate?: StandardCreateInput | null;
  translationsDelete?: string[] | null;
  translationsCreate?: OutputItemTranslationCreateInput[] | null;
  translationsUpdate?: OutputItemTranslationUpdateInput[] | null;
}

export interface ProfileUpdateInput {
  handle?: string | null;
  name?: string | null;
  theme?: string | null;
  hiddenTagsConnect?: string[] | null;
  hiddenTagsDisconnect?: string[] | null;
  hiddenTagsCreate?: TagCreateInput[] | null;
  resourceListsDelete?: string[] | null;
  resourceListsCreate?: ResourceCreateInput[] | null;
  resourceListsUpdate?: ResourceUpdateInput[] | null;
  starredTagsConnect?: string[] | null;
  starredTagsDisconnect?: string[] | null;
  starredTagsCreate?: TagCreateInput[] | null;
  translationsDelete?: string[] | null;
  translationsCreate?: UserTranslationCreateInput[] | null;
  translationsUpdate?: UserTranslationUpdateInput[] | null;
}

export interface ProjectCountInput {
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface ProjectCreateInput {
  createdByOrganizationId?: string | null;
  createdByUserId?: string | null;
  isComplete?: boolean | null;
  parentId?: string | null;
  resourceListsCreate?: ResourceListCreateInput[] | null;
  tagsConnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
  translationsCreate?: ProjectTranslationCreateInput[] | null;
}

export interface ProjectSearchInput {
  after?: string | null;
  createdTimeFrame?: TimeFrame | null;
  ids?: string[] | null;
  isComplete?: boolean | null;
  languages?: string[] | null;
  minScore?: number | null;
  minStars?: number | null;
  organizationId?: string | null;
  parentId?: string | null;
  reportId?: string | null;
  resourceLists?: string[] | null;
  resourceTypes?: ResourceUsedFor[] | null;
  searchString?: string | null;
  sortBy?: ProjectSortBy | null;
  tags?: string[] | null;
  take?: number | null;
  updatedTimeFrame?: TimeFrame | null;
  userId?: string | null;
}

export interface ProjectTranslationCreateInput {
  language: string;
  description?: string | null;
  name: string;
}

export interface ProjectTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
  name?: string | null;
}

export interface ProjectUpdateInput {
  id: string;
  handle?: string | null;
  isComplete?: boolean | null;
  organizationId?: string | null;
  parentId?: string | null;
  userId?: string | null;
  resourceListsDelete?: string[] | null;
  resourceListsCreate?: ResourceListCreateInput[] | null;
  resourceListsUpdate?: ResourceListUpdateInput[] | null;
  tagsConnect?: string[] | null;
  tagsDisconnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
  translationsDelete?: string[] | null;
  translationsCreate?: ProjectTranslationCreateInput[] | null;
  translationsUpdate?: ProjectTranslationUpdateInput[] | null;
}

export interface ReadAssetsInput {
  files: string[];
}

export interface ReportCreateInput {
  createdFor: ReportFor;
  createdForId: string;
  details?: string | null;
  language: string;
  reason: string;
}

export interface ReportUpdateInput {
  id: string;
  details?: string | null;
  language?: string | null;
  reason?: string | null;
}

export interface ResourceCountInput {
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface ResourceCreateInput {
  listId: string;
  index?: number | null;
  link: string;
  translationsCreate?: ResourceTranslationCreateInput[] | null;
  usedFor: ResourceUsedFor;
}

export interface ResourceListCreateInput {
  id?: string | null;
  index?: number | null;
  usedFor: ResourceListUsedFor;
  organizationId?: string | null;
  projectId?: string | null;
  routineId?: string | null;
  userId?: string | null;
  translationsCreate?: ResourceListTranslationCreateInput[] | null;
  resourcesCreate?: ResourceCreateInput[] | null;
}

export interface ResourceListTranslationCreateInput {
  language: string;
  description?: string | null;
  title?: string | null;
}

export interface ResourceListTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
  title?: string | null;
}

export interface ResourceListUpdateInput {
  id: string;
  index?: number | null;
  usedFor?: ResourceListUsedFor | null;
  organizationId?: string | null;
  projectId?: string | null;
  routineId?: string | null;
  userId?: string | null;
  translationsDelete?: string[] | null;
  translationsCreate?: ResourceListTranslationCreateInput[] | null;
  translationsUpdate?: ResourceListTranslationUpdateInput[] | null;
  resourcesDelete?: string[] | null;
  resourcesCreate?: ResourceCreateInput[] | null;
  resourcesUpdate?: ResourceUpdateInput[] | null;
}

export interface ResourceSearchInput {
  forId?: string | null;
  forType?: ResourceFor | null;
  ids?: string[] | null;
  languages?: string[] | null;
  sortBy?: ResourceSortBy | null;
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
  searchString?: string | null;
  after?: string | null;
  take?: number | null;
}

export interface ResourceTranslationCreateInput {
  language: string;
  description?: string | null;
  title?: string | null;
}

export interface ResourceTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
  title?: string | null;
}

export interface ResourceUpdateInput {
  id: string;
  listId?: string | null;
  index?: number | null;
  link?: string | null;
  translationsDelete?: string[] | null;
  translationsCreate?: ResourceTranslationCreateInput[] | null;
  translationsUpdate?: ResourceTranslationUpdateInput[] | null;
  usedFor?: ResourceUsedFor | null;
}

export interface RoutineCancelInput {
  id: string;
}

export interface RoutineCompleteInput {
  id: string;
}

export interface RoutineCountInput {
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface RoutineCreateInput {
  isAutomatable?: boolean | null;
  isComplete?: boolean | null;
  isInternal?: boolean | null;
  version?: string | null;
  parentId?: string | null;
  projectId?: string | null;
  createdByUserId?: string | null;
  createdByOrganizationId?: string | null;
  nodesCreate?: NodeCreateInput[] | null;
  nodeLinksCreate?: NodeLinkCreateInput[] | null;
  inputsCreate?: InputItemCreateInput[] | null;
  outputsCreate?: OutputItemCreateInput[] | null;
  resourceListsCreate?: ResourceListCreateInput[] | null;
  tagsConnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
  translationsCreate?: RoutineTranslationCreateInput[] | null;
}

export interface RoutineSearchInput {
  after?: string | null;
  createdTimeFrame?: TimeFrame | null;
  excludeIds?: string[] | null;
  ids?: string[] | null;
  isComplete?: boolean | null;
  languages?: string[] | null;
  minComplexity?: number | null;
  maxComplexity?: number | null;
  minSimplicity?: number | null;
  maxSimplicity?: number | null;
  minScore?: number | null;
  minStars?: number | null;
  organizationId?: string | null;
  projectId?: string | null;
  parentId?: string | null;
  reportId?: string | null;
  resourceLists?: string[] | null;
  resourceTypes?: ResourceUsedFor[] | null;
  searchString?: string | null;
  sortBy?: RoutineSortBy | null;
  tags?: string[] | null;
  take?: number | null;
  updatedTimeFrame?: TimeFrame | null;
  userId?: string | null;
}

export interface RoutineStartInput {
  id: string;
}

export interface RoutineTranslationCreateInput {
  language: string;
  description?: string | null;
  instructions: string;
  title: string;
}

export interface RoutineTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
  instructions?: string | null;
  title?: string | null;
}

export interface RoutineUpdateInput {
  id: string;
  isAutomatable?: boolean | null;
  isComplete?: boolean | null;
  isInternal?: boolean | null;
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
  resourceListsDelete?: string[] | null;
  resourceListsCreate?: ResourceListCreateInput[] | null;
  resourceListsUpdate?: ResourceListUpdateInput[] | null;
  tagsConnect?: string[] | null;
  tagsDisconnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
  translationsDelete?: string[] | null;
  translationsCreate?: RoutineTranslationCreateInput[] | null;
  translationsUpdate?: RoutineTranslationUpdateInput[] | null;
}

export interface SendVerificationEmailInput {
  emailAddress: string;
}

export interface StandardCountInput {
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface StandardCreateInput {
  default?: string | null;
  isFile?: boolean | null;
  name: string;
  schema?: string | null;
  type?: StandardType | null;
  version?: string | null;
  createdByUserId?: string | null;
  createdByOrganizationId?: string | null;
  tagsConnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
  translationsCreate?: StandardTranslationCreateInput[] | null;
}

export interface StandardSearchInput {
  after?: string | null;
  createdTimeFrame?: TimeFrame | null;
  ids?: string[] | null;
  languages?: string[] | null;
  minScore?: number | null;
  minStars?: number | null;
  organizationId?: string | null;
  projectId?: string | null;
  reportId?: string | null;
  routineId?: string | null;
  searchString?: string | null;
  sortBy?: StandardSortBy | null;
  tags?: string[] | null;
  take?: number | null;
  updatedTimeFrame?: TimeFrame | null;
  userId?: string | null;
}

export interface StandardTranslationCreateInput {
  language: string;
  description?: string | null;
}

export interface StandardTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
}

export interface StandardUpdateInput {
  id: string;
  makeAnonymous?: boolean | null;
  tagsConnect?: string[] | null;
  tagsDisconnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
  translationsDelete?: string[] | null;
  translationsCreate?: StandardTranslationCreateInput[] | null;
  translationsUpdate?: StandardTranslationUpdateInput[] | null;
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
  tag: string;
  translationsCreate?: TagTranslationCreateInput[] | null;
}

export interface TagSearchInput {
  after?: string | null;
  createdTimeFrame?: TimeFrame | null;
  hidden?: boolean | null;
  ids?: string[] | null;
  languages?: string[] | null;
  minStars?: number | null;
  myTags?: boolean | null;
  searchString?: string | null;
  sortBy?: TagSortBy | null;
  take?: number | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface TagTranslationCreateInput {
  language: string;
  description?: string | null;
}

export interface TagTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
}

export interface TagUpdateInput {
  id: string;
  anonymous?: boolean | null;
  tag?: string | null;
  translationsDelete?: string[] | null;
  translationsCreate?: TagTranslationCreateInput[] | null;
  translationsUpdate?: TagTranslationUpdateInput[] | null;
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
  languages?: string[] | null;
  minStars?: number | null;
  organizationId?: string | null;
  projectId?: string | null;
  routineId?: string | null;
  reportId?: string | null;
  standardId?: string | null;
  ids?: string[] | null;
  resourceLists?: string[] | null;
  resourceTypes?: ResourceUsedFor[] | null;
  sortBy?: UserSortBy | null;
  searchString?: string | null;
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
  after?: string | null;
  take?: number | null;
}

export interface UserTranslationCreateInput {
  language: string;
  bio?: string | null;
}

export interface UserTranslationUpdateInput {
  id: string;
  language: string;
  bio?: string | null;
}

export interface VoteInput {
  isUpvote?: boolean | null;
  voteFor: VoteFor;
  forId: string;
}

export interface WalletCompleteInput {
  stakingAddress: string;
  signedPayload: string;
}

export interface WalletInitInput {
  stakingAddress: string;
  nonceDescription?: string | null;
}

export interface WalletUpdateInput {
  id: string;
  name?: string | null;
}

export interface WriteAssetsInput {
  files: any[];
}

//==============================================================
// END Enums and Input Objects
//==============================================================
