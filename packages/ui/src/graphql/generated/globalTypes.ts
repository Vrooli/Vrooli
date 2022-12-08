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

export enum CommentSortBy {
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  StarsAsc = "StarsAsc",
  StarsDesc = "StarsDesc",
  VotesAsc = "VotesAsc",
  VotesDesc = "VotesDesc",
}

export enum CopyType {
  ApiVersion = "ApiVersion",
  NoteVersion = "NoteVersion",
  Organization = "Organization",
  ProjectVersion = "ProjectVersion",
  RoutineVersion = "RoutineVersion",
  SmartContractVersion = "SmartContractVersion",
  StandardVersion = "StandardVersion",
}

export enum DeleteType {
  Api = "Api",
  ApiVersion = "ApiVersion",
  Comment = "Comment",
  Email = "Email",
  Node = "Node",
  Organization = "Organization",
  Project = "Project",
  ProjectVersion = "ProjectVersion",
  PushDevice = "PushDevice",
  Reminder = "Reminder",
  ReminderList = "ReminderList",
  Report = "Report",
  Routine = "Routine",
  RoutineVersion = "RoutineVersion",
  Run = "Run",
  SmartContract = "SmartContract",
  SmartContractVersion = "SmartContractVersion",
  Standard = "Standard",
  StandardVersion = "StandardVersion",
  Wallet = "Wallet",
}

export enum NodeType {
  End = "End",
  Redirect = "Redirect",
  RoutineList = "RoutineList",
  Start = "Start",
}

export enum NotificationSortBy {
  CategoryAsc = "CategoryAsc",
  CategoryDesc = "CategoryDesc",
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  TitleAsc = "TitleAsc",
  TitleDesc = "TitleDesc",
}

export enum OrganizationSortBy {
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  StarsAsc = "StarsAsc",
  StarsDesc = "StarsDesc",
}

export enum ProjectOrOrganizationSortBy {
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  StarsAsc = "StarsAsc",
  StarsDesc = "StarsDesc",
}

export enum ProjectOrRoutineSortBy {
  DateCompletedAsc = "DateCompletedAsc",
  DateCompletedDesc = "DateCompletedDesc",
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  IssuesAsc = "IssuesAsc",
  IssuesDesc = "IssuesDesc",
  PullRequestsAsc = "PullRequestsAsc",
  PullRequestsDesc = "PullRequestsDesc",
  QuestionsAsc = "QuestionsAsc",
  QuestionsDesc = "QuestionsDesc",
  StarsAsc = "StarsAsc",
  StarsDesc = "StarsDesc",
  VersionsAsc = "VersionsAsc",
  VersionsDesc = "VersionsDesc",
  ViewsAsc = "ViewsAsc",
  ViewsDesc = "ViewsDesc",
  VotesAsc = "VotesAsc",
  VotesDesc = "VotesDesc",
}

export enum ProjectSortBy {
  DateCompletedAsc = "DateCompletedAsc",
  DateCompletedDesc = "DateCompletedDesc",
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  IssuesAsc = "IssuesAsc",
  IssuesDesc = "IssuesDesc",
  PullRequestsAsc = "PullRequestsAsc",
  PullRequestsDesc = "PullRequestsDesc",
  QuestionsAsc = "QuestionsAsc",
  QuestionsDesc = "QuestionsDesc",
  StarsAsc = "StarsAsc",
  StarsDesc = "StarsDesc",
  VersionsAsc = "VersionsAsc",
  VersionsDesc = "VersionsDesc",
  ViewsAsc = "ViewsAsc",
  ViewsDesc = "ViewsDesc",
  VotesAsc = "VotesAsc",
  VotesDesc = "VotesDesc",
}

export enum ReportFor {
  ApiVersion = "ApiVersion",
  Comment = "Comment",
  Issue = "Issue",
  NoteVersion = "NoteVersion",
  Organization = "Organization",
  Post = "Post",
  ProjectVersion = "ProjectVersion",
  RoutineVersion = "RoutineVersion",
  StandardVersion = "StandardVersion",
  Tag = "Tag",
  User = "User",
}

export enum ReportSortBy {
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  ResponsesAsc = "ResponsesAsc",
  ResponsesDesc = "ResponsesDesc",
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
  UsedForAsc = "UsedForAsc",
  UsedForDesc = "UsedForDesc",
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
  DateCompletedAsc = "DateCompletedAsc",
  DateCompletedDesc = "DateCompletedDesc",
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  IssuesAsc = "IssuesAsc",
  IssuesDesc = "IssuesDesc",
  PullRequestsAsc = "PullRequestsAsc",
  PullRequestsDesc = "PullRequestsDesc",
  QuestionsAsc = "QuestionsAsc",
  QuestionsDesc = "QuestionsDesc",
  QuizzesAsc = "QuizzesAsc",
  QuizzesDesc = "QuizzesDesc",
  StarsAsc = "StarsAsc",
  StarsDesc = "StarsDesc",
  VersionsAsc = "VersionsAsc",
  VersionsDesc = "VersionsDesc",
  ViewsAsc = "ViewsAsc",
  ViewsDesc = "ViewsDesc",
  VotesAsc = "VotesAsc",
  VotesDesc = "VotesDesc",
}

export enum RunRoutineSortBy {
  ContextSwitchesAsc = "ContextSwitchesAsc",
  ContextSwitchesDesc = "ContextSwitchesDesc",
  DateCompletedAsc = "DateCompletedAsc",
  DateCompletedDesc = "DateCompletedDesc",
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateStartedAsc = "DateStartedAsc",
  DateStartedDesc = "DateStartedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  RunRoutinesAsc = "RunRoutinesAsc",
  RunRoutinesDesc = "RunRoutinesDesc",
  StepsAsc = "StepsAsc",
  StepsDesc = "StepsDesc",
}

export enum RunRoutineStepStatus {
  Completed = "Completed",
  InProgress = "InProgress",
  Skipped = "Skipped",
}

export enum RunStatus {
  Cancelled = "Cancelled",
  Completed = "Completed",
  Failed = "Failed",
  InProgress = "InProgress",
  Scheduled = "Scheduled",
}

export enum StandardSortBy {
  DateCompletedAsc = "DateCompletedAsc",
  DateCompletedDesc = "DateCompletedDesc",
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  IssuesAsc = "IssuesAsc",
  IssuesDesc = "IssuesDesc",
  PullRequestsAsc = "PullRequestsAsc",
  PullRequestsDesc = "PullRequestsDesc",
  QuestionsAsc = "QuestionsAsc",
  QuestionsDesc = "QuestionsDesc",
  StarsAsc = "StarsAsc",
  StarsDesc = "StarsDesc",
  VersionsAsc = "VersionsAsc",
  VersionsDesc = "VersionsDesc",
  ViewsAsc = "ViewsAsc",
  ViewsDesc = "ViewsDesc",
  VotesAsc = "VotesAsc",
  VotesDesc = "VotesDesc",
}

export enum StarFor {
  Api = "Api",
  Comment = "Comment",
  Issue = "Issue",
  Note = "Note",
  Organization = "Organization",
  Post = "Post",
  Project = "Project",
  Question = "Question",
  QuestionAnswer = "QuestionAnswer",
  Quiz = "Quiz",
  Routine = "Routine",
  SmartContract = "SmartContract",
  Standard = "Standard",
  Tag = "Tag",
  User = "User",
}

export enum StarSortBy {
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
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

export enum ViewSortBy {
  LastViewedAsc = "LastViewedAsc",
  LastViewedDesc = "LastViewedDesc",
}

export enum VisibilityType {
  All = "All",
  Private = "Private",
  Public = "Public",
}

export enum VoteFor {
  Api = "Api",
  Comment = "Comment",
  Issue = "Issue",
  Note = "Note",
  Post = "Post",
  Project = "Project",
  Question = "Question",
  QuestionAnswer = "QuestionAnswer",
  Quiz = "Quiz",
  Routine = "Routine",
  SmartContract = "SmartContract",
  Standard = "Standard",
}

export interface CommentCreateInput {
  id: string;
  createdFor: CommentFor;
  forId: string;
  parentId?: string | null;
  translationsCreate?: CommentTranslationCreateInput[] | null;
}

export interface CommentSearchInput {
  after?: string | null;
  createdTimeFrame?: TimeFrame | null;
  languages?: string[] | null;
  minScore?: number | null;
  minStars?: number | null;
  organizationId?: string | null;
  projectId?: string | null;
  routineId?: string | null;
  searchString?: string | null;
  sortBy?: CommentSortBy | null;
  standardId?: string | null;
  take?: number | null;
  updatedTimeFrame?: TimeFrame | null;
  userId?: string | null;
}

export interface CommentTranslationCreateInput {
  id: string;
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

export interface CopyInput {
  id: string;
  intendToPullRequest: boolean;
  objectType: CopyType;
}

export interface DeleteManyInput {
  ids: string[];
  objectType: DeleteType;
}

export interface DeleteOneInput {
  id: string;
  objectType: DeleteType;
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

export interface FindByIdOrHandleInput {
  id?: string | null;
  handle?: string | null;
}

export interface FindByVersionInput {
  id?: string | null;
  versionGroupId?: string | null;
}

export interface FindHandlesInput {
  organizationId?: string | null;
}

export interface HistoryInput {
  searchString: string;
  take?: number | null;
}

export interface InputItemCreateInput {
  id: string;
  isRequired?: boolean | null;
  name?: string | null;
  standardVersionConnect?: string | null;
  standardCreate?: StandardCreateInput | null;
  translationsDelete?: string[] | null;
  translationsCreate?: InputItemTranslationCreateInput[] | null;
  translationsUpdate?: InputItemTranslationUpdateInput[] | null;
}

export interface InputItemTranslationCreateInput {
  id: string;
  language: string;
  description?: string | null;
  helpText?: string | null;
}

export interface InputItemTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
  helpText?: string | null;
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

export interface LogOutInput {
  id?: string | null;
}

export interface LoopCreateInput {
  id: string;
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
  id: string;
  translationsCreate?: LoopWhileTranslationCreateInput[] | null;
  condition: string;
  toId?: string | null;
}

export interface LoopWhileTranslationCreateInput {
  id: string;
  language: string;
  description?: string | null;
  name: string;
}

export interface LoopWhileTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
  name?: string | null;
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
  type: NodeType;
  loopCreate?: LoopCreateInput | null;
  nodeEndCreate?: NodeEndCreateInput | null;
  nodeRoutineListCreate?: NodeRoutineListCreateInput | null;
  routineVersionId: string;
  translationsCreate?: NodeTranslationCreateInput[] | null;
}

export interface NodeEndCreateInput {
  id: string;
  wasSuccessful?: boolean | null;
}

export interface NodeEndUpdateInput {
  id: string;
  wasSuccessful?: boolean | null;
}

export interface NodeLinkCreateInput {
  id: string;
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
  id: string;
  linkId?: string | null;
  translationsCreate?: NodeLinkWhenTranslationCreateInput[] | null;
  condition: string;
}

export interface NodeLinkWhenTranslationCreateInput {
  id: string;
  language: string;
  description?: string | null;
  name: string;
}

export interface NodeLinkWhenTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
  name?: string | null;
}

export interface NodeLinkWhenUpdateInput {
  id: string;
  linkId?: string | null;
  translationsDelete?: string[] | null;
  translationsCreate?: NodeLinkWhenTranslationCreateInput[] | null;
  translationsUpdate?: NodeLinkWhenTranslationUpdateInput[] | null;
  condition?: string | null;
}

export interface NodeRoutineListCreateInput {
  id: string;
  isOrdered?: boolean | null;
  isOptional?: boolean | null;
  routinesCreate?: NodeRoutineListItemCreateInput[] | null;
}

export interface NodeRoutineListItemCreateInput {
  id: string;
  index: number;
  isOptional?: boolean | null;
  routineVersionConnect: string;
  translationsCreate?: NodeRoutineListItemTranslationCreateInput[] | null;
}

export interface NodeRoutineListItemTranslationCreateInput {
  id: string;
  language: string;
  description?: string | null;
  name?: string | null;
}

export interface NodeRoutineListItemTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
  name?: string | null;
}

export interface NodeRoutineListItemUpdateInput {
  id: string;
  index?: number | null;
  isOptional?: boolean | null;
  routineUpdate?: RoutineUpdateInput | null;
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
  id: string;
  language: string;
  name: string;
  description?: string | null;
}

export interface NodeTranslationUpdateInput {
  id: string;
  language?: string | null;
  name?: string | null;
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
  nodeEndCreate?: NodeEndCreateInput | null;
  nodeEndUpdate?: NodeEndUpdateInput | null;
  nodeRoutineListCreate?: NodeRoutineListCreateInput | null;
  nodeRoutineListUpdate?: NodeRoutineListUpdateInput | null;
  routineVersionId?: string | null;
  translationsDelete?: string[] | null;
  translationsCreate?: NodeTranslationCreateInput[] | null;
  translationsUpdate?: NodeTranslationUpdateInput[] | null;
}

export interface NotificationSearchInput {
  after?: string | null;
  createdTimeFrame?: TimeFrame | null;
  ids?: string[] | null;
  searchString?: string | null;
  sortBy?: NotificationSortBy | null;
  take?: number | null;
  visibility?: VisibilityType | null;
}

export interface NotificationSettingsCategoryUpdateInput {
  category: string;
  enabled?: boolean | null;
  dailyLimit?: number | null;
  toEmails?: boolean | null;
  toSms?: boolean | null;
  toPush?: boolean | null;
}

export interface NotificationSettingsUpdateInput {
  includedEmails?: string[] | null;
  includedSms?: string[] | null;
  includedPush?: string[] | null;
  toEmails?: boolean | null;
  toSms?: boolean | null;
  toPush?: boolean | null;
  dailyLimit?: number | null;
  enabled?: boolean | null;
  categories?: NotificationSettingsCategoryUpdateInput[] | null;
}

export interface OrganizationCreateInput {
  id: string;
  handle?: string | null;
  isOpenToNewMembers?: boolean | null;
  isPrivate?: boolean | null;
  resourceListsCreate?: ResourceListCreateInput[] | null;
  tagsConnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
  translationsCreate?: OrganizationTranslationCreateInput[] | null;
  roles?: RoleCreateInput[] | null;
}

export interface OrganizationSearchInput {
  after?: string | null;
  createdTimeFrame?: TimeFrame | null;
  ids?: string[] | null;
  isOpenToNewMembers?: boolean | null;
  languages?: string[] | null;
  minStars?: number | null;
  minViews?: number | null;
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
  visibility?: VisibilityType | null;
}

export interface OrganizationTranslationCreateInput {
  id: string;
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
  isPrivate?: boolean | null;
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
  rolesDelete?: string[] | null;
  rolesCreate?: RoleCreateInput[] | null;
  rolesUpdate?: RoleUpdateInput[] | null;
}

export interface OutputItemCreateInput {
  id: string;
  name?: string | null;
  standardVersionConnect?: string | null;
  standardCreate?: StandardCreateInput | null;
  translationsCreate?: OutputItemTranslationCreateInput[] | null;
}

export interface OutputItemTranslationCreateInput {
  id: string;
  language: string;
  description?: string | null;
  helpText?: string | null;
}

export interface OutputItemTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
  helpText?: string | null;
}

export interface OutputItemUpdateInput {
  id: string;
  name?: string | null;
  standardVersionConnect?: string | null;
  standardCreate?: StandardCreateInput | null;
  translationsDelete?: string[] | null;
  translationsCreate?: OutputItemTranslationCreateInput[] | null;
  translationsUpdate?: OutputItemTranslationUpdateInput[] | null;
}

export interface PopularInput {
  searchString: string;
  take?: number | null;
}

export interface ProfileEmailUpdateInput {
  emailsCreate?: EmailCreateInput[] | null;
  emailsUpdate?: EmailUpdateInput[] | null;
  emailsDelete?: string[] | null;
  currentPassword: string;
  newPassword?: string | null;
}

export interface ProfileUpdateInput {
  handle?: string | null;
  name?: string | null;
  theme?: string | null;
  hiddenTagsDelete?: string[] | null;
  hiddenTagsCreate?: TagHiddenCreateInput[] | null;
  hiddenTagsUpdate?: TagHiddenUpdateInput[] | null;
  resourceListsDelete?: string[] | null;
  resourceListsCreate?: ResourceListCreateInput[] | null;
  resourceListsUpdate?: ResourceListUpdateInput[] | null;
  starredTagsConnect?: string[] | null;
  starredTagsDisconnect?: string[] | null;
  starredTagsCreate?: TagCreateInput[] | null;
  translationsDelete?: string[] | null;
  translationsCreate?: UserTranslationCreateInput[] | null;
  translationsUpdate?: UserTranslationUpdateInput[] | null;
}

export interface ProjectCreateInput {
  id: string;
  createdByOrganizationId?: string | null;
  createdByUserId?: string | null;
  handle?: string | null;
  isComplete?: boolean | null;
  isPrivate?: boolean | null;
  parentId?: string | null;
  resourceListsCreate?: ResourceListCreateInput[] | null;
  tagsConnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
  translationsCreate?: ProjectTranslationCreateInput[] | null;
}

export interface ProjectOrOrganizationSearchInput {
  createdTimeFrame?: TimeFrame | null;
  excludeIds?: string[] | null;
  ids?: string[] | null;
  languages?: string[] | null;
  minStars?: number | null;
  minViews?: number | null;
  objectType?: string | null;
  organizationAfter?: string | null;
  organizationIsOpenToNewMembers?: boolean | null;
  organizationProjectId?: string | null;
  organizationRoutineId?: string | null;
  projectAfter?: string | null;
  projectIsComplete?: boolean | null;
  projectIsCompleteExceptions?: SearchException[] | null;
  projectMinScore?: number | null;
  projectOrganizationId?: string | null;
  projectParentId?: string | null;
  reportId?: string | null;
  resourceLists?: string[] | null;
  resourceTypes?: ResourceUsedFor[] | null;
  searchString?: string | null;
  sortBy?: ProjectOrOrganizationSortBy | null;
  tags?: string[] | null;
  take?: number | null;
  updatedTimeFrame?: TimeFrame | null;
  userId?: string | null;
  visibility?: VisibilityType | null;
}

export interface ProjectOrRoutineSearchInput {
  createdTimeFrame?: TimeFrame | null;
  excludeIds?: string[] | null;
  ids?: string[] | null;
  isComplete?: boolean | null;
  isCompleteExceptions?: SearchException[] | null;
  languages?: string[] | null;
  minScore?: number | null;
  minStars?: number | null;
  minViews?: number | null;
  objectType?: string | null;
  organizationId?: string | null;
  parentId?: string | null;
  projectAfter?: string | null;
  reportId?: string | null;
  resourceLists?: string[] | null;
  resourceTypes?: ResourceUsedFor[] | null;
  routineAfter?: string | null;
  routineIsInternal?: boolean | null;
  routineMinComplexity?: number | null;
  routineMaxComplexity?: number | null;
  routineMinSimplicity?: number | null;
  routineMaxSimplicity?: number | null;
  routineMaxTimesCompleted?: number | null;
  routineMinTimesCompleted?: number | null;
  routineProjectId?: string | null;
  searchString?: string | null;
  sortBy?: ProjectOrRoutineSortBy | null;
  tags?: string[] | null;
  take?: number | null;
  updatedTimeFrame?: TimeFrame | null;
  userId?: string | null;
  visibility?: VisibilityType | null;
}

export interface ProjectSearchInput {
  after?: string | null;
  createdTimeFrame?: TimeFrame | null;
  ids?: string[] | null;
  isComplete?: boolean | null;
  isCompleteExceptions?: SearchException[] | null;
  languages?: string[] | null;
  minScore?: number | null;
  minStars?: number | null;
  minViews?: number | null;
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
  visibility?: VisibilityType | null;
}

export interface ProjectTranslationCreateInput {
  id: string;
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
  isPrivate?: boolean | null;
  organizationId?: string | null;
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

export interface PushDeviceCreateInput {
  endpoint: string;
  expires?: any | null;
  keys: PushDeviceKeysInput;
  name?: string | null;
}

export interface PushDeviceKeysInput {
  auth: string;
  p256dh: string;
}

export interface PushDeviceUpdateInput {
  id: string;
  name?: string | null;
}

export interface ReadAssetsInput {
  files: string[];
}

export interface ReportCreateInput {
  id: string;
  createdFor: ReportFor;
  createdForId: string;
  details?: string | null;
  language: string;
  reason: string;
}

export interface ReportSearchInput {
  userId?: string | null;
  organizationId?: string | null;
  projectId?: string | null;
  routineId?: string | null;
  standardId?: string | null;
  tagId?: string | null;
  ids?: string[] | null;
  languages?: string[] | null;
  sortBy?: ReportSortBy | null;
  createdTimeFrame?: TimeFrame | null;
  updatedTimeFrame?: TimeFrame | null;
  searchString?: string | null;
  after?: string | null;
  take?: number | null;
}

export interface ReportUpdateInput {
  id: string;
  details?: string | null;
  language?: string | null;
  reason?: string | null;
}

export interface ResourceCreateInput {
  id: string;
  listId: string;
  index?: number | null;
  link: string;
  translationsCreate?: ResourceTranslationCreateInput[] | null;
  usedFor: ResourceUsedFor;
}

export interface ResourceListCreateInput {
  id: string;
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
  id: string;
  language: string;
  description?: string | null;
  name?: string | null;
}

export interface ResourceListTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
  name?: string | null;
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
  id: string;
  language: string;
  description?: string | null;
  name?: string | null;
}

export interface ResourceTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
  name?: string | null;
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

export interface RoleCreateInput {
  id: string;
  name: string;
  translationsCreate?: RoleTranslationCreateInput[] | null;
}

export interface RoleTranslationCreateInput {
  id: string;
  language: string;
  description: string;
}

export interface RoleTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
}

export interface RoleUpdateInput {
  id: string;
  translationsDelete?: string[] | null;
  translationsCreate?: RoleTranslationCreateInput[] | null;
  translationsUpdate?: RoleTranslationUpdateInput[] | null;
}

export interface RoutineCreateInput {
  id: string;
  isAutomatable?: boolean | null;
  isComplete?: boolean | null;
  isInternal?: boolean | null;
  isPrivate?: boolean | null;
  versionLabel?: string | null;
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
  isCompleteExceptions?: SearchException[] | null;
  isInternal?: boolean | null;
  isInternalExceptions?: SearchException[] | null;
  languages?: string[] | null;
  minComplexity?: number | null;
  maxComplexity?: number | null;
  minSimplicity?: number | null;
  maxSimplicity?: number | null;
  maxTimesCompleted?: number | null;
  minScore?: number | null;
  minStars?: number | null;
  minTimesCompleted?: number | null;
  minViews?: number | null;
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
  visibility?: VisibilityType | null;
}

export interface RoutineTranslationCreateInput {
  id: string;
  language: string;
  description?: string | null;
  instructions: string;
  name: string;
}

export interface RoutineTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
  instructions?: string | null;
  name?: string | null;
}

export interface RoutineUpdateInput {
  isAutomatable?: boolean | null;
  isComplete?: boolean | null;
  isInternal?: boolean | null;
  isPrivate?: boolean | null;
  versionId?: string | null;
  versionLabel?: string | null;
  userId?: string | null;
  organizationId?: string | null;
  projectId?: string | null;
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

export interface RunRoutineCancelInput {
  id: string;
}

export interface RunRoutineCompleteInput {
  id: string;
  completedComplexity?: number | null;
  exists?: boolean | null;
  name: string;
  finalStepCreate?: RunRoutineStepCreateInput | null;
  finalStepUpdate?: RunRoutineStepUpdateInput | null;
  inputsDelete?: string[] | null;
  inputsCreate?: RunRoutineInputCreateInput[] | null;
  inputsUpdate?: RunRoutineInputUpdateInput[] | null;
  wasSuccessful?: boolean | null;
}

export interface RunRoutineCreateInput {
  id: string;
  isPrivate?: boolean | null;
  routineVersionId: string;
  name: string;
  stepsCreate?: RunRoutineStepCreateInput[] | null;
  inputsCreate?: RunRoutineInputCreateInput[] | null;
}

export interface RunRoutineInputCreateInput {
  id: string;
  data: string;
  inputId: string;
}

export interface RunRoutineInputSearchInput {
  after?: string | null;
  createdTimeFrame?: TimeFrame | null;
  excludeIds?: string[] | null;
  ids?: string[] | null;
  routineIds?: string[] | null;
  standardIds?: string[] | null;
  take?: number | null;
  updatedTimeFrame?: TimeFrame | null;
}

export interface RunRoutineInputUpdateInput {
  id: string;
  data: string;
}

export interface RunRoutineSearchInput {
  after?: string | null;
  createdTimeFrame?: TimeFrame | null;
  startedTimeFrame?: TimeFrame | null;
  completedTimeFrame?: TimeFrame | null;
  excludeIds?: string[] | null;
  ids?: string[] | null;
  status?: RunStatus | null;
  routineId?: string | null;
  searchString?: string | null;
  sortBy?: RunRoutineSortBy | null;
  take?: number | null;
  updatedTimeFrame?: TimeFrame | null;
  visibility?: VisibilityType | null;
}

export interface RunRoutineStepCreateInput {
  id: string;
  nodeId?: string | null;
  contextSwitches?: number | null;
  subroutineVersionId?: string | null;
  order: number;
  step: number[];
  timeElapsed?: number | null;
  name: string;
}

export interface RunRoutineStepUpdateInput {
  id: string;
  contextSwitches?: number | null;
  status?: RunRoutineStepStatus | null;
  timeElapsed?: number | null;
}

export interface RunRoutineUpdateInput {
  id: string;
  completedComplexity?: number | null;
  contextSwitches?: number | null;
  isPrivate?: boolean | null;
  timeElapsed?: number | null;
  stepsDelete?: string[] | null;
  stepsCreate?: RunRoutineStepCreateInput[] | null;
  stepsUpdate?: RunRoutineStepUpdateInput[] | null;
  inputsDelete?: string[] | null;
  inputsCreate?: RunRoutineInputCreateInput[] | null;
  inputsUpdate?: RunRoutineInputUpdateInput[] | null;
}

export interface SearchException {
  field: string;
  value: string;
}

export interface SendVerificationEmailInput {
  emailAddress: string;
}

export interface StandardCreateInput {
  id: string;
  default?: string | null;
  isInternal?: boolean | null;
  isPrivate?: boolean | null;
  name?: string | null;
  type: string;
  props: string;
  yup?: string | null;
  versionLabel?: string | null;
  createdByUserId?: string | null;
  createdByOrganizationId?: string | null;
  resourceListsCreate?: ResourceListCreateInput[] | null;
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
  minViews?: number | null;
  organizationId?: string | null;
  projectId?: string | null;
  reportId?: string | null;
  routineId?: string | null;
  searchString?: string | null;
  sortBy?: StandardSortBy | null;
  tags?: string[] | null;
  take?: number | null;
  type?: string | null;
  updatedTimeFrame?: TimeFrame | null;
  userId?: string | null;
  visibility?: VisibilityType | null;
}

export interface StandardTranslationCreateInput {
  id: string;
  language: string;
  description?: string | null;
  jsonVariable?: string | null;
}

export interface StandardTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
  jsonVariable?: string | null;
}

export interface StandardUpdateInput {
  id: string;
  makeAnonymous?: boolean | null;
  isPrivate?: boolean | null;
  userId?: string | null;
  default?: string | null;
  type: string;
  props: string;
  yup?: string | null;
  organizationId?: string | null;
  resourceListsDelete?: string[] | null;
  resourceListsCreate?: ResourceListCreateInput[] | null;
  resourceListsUpdate?: ResourceListUpdateInput[] | null;
  tagsConnect?: string[] | null;
  tagsDisconnect?: string[] | null;
  tagsCreate?: TagCreateInput[] | null;
  translationsDelete?: string[] | null;
  translationsCreate?: StandardTranslationCreateInput[] | null;
  translationsUpdate?: StandardTranslationUpdateInput[] | null;
  versionId?: string | null;
  versionLabel?: string | null;
}

export interface StarInput {
  isStar: boolean;
  starFor: StarFor;
  forId: string;
}

export interface StarSearchInput {
  after?: string | null;
  excludeTags?: boolean | null;
  ids?: string[] | null;
  searchString?: string | null;
  sortBy?: StarSortBy | null;
  take?: number | null;
}

export interface SwitchCurrentAccountInput {
  id: string;
}

export interface TagCreateInput {
  anonymous?: boolean | null;
  tag: string;
  translationsCreate?: TagTranslationCreateInput[] | null;
}

export interface TagHiddenCreateInput {
  id: string;
  isBlur?: boolean | null;
  tagCreate?: TagCreateInput | null;
  tagConnect?: string | null;
}

export interface TagHiddenUpdateInput {
  id: string;
  isBlur?: boolean | null;
}

export interface TagSearchInput {
  after?: string | null;
  createdTimeFrame?: TimeFrame | null;
  excludeIds?: string[] | null;
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
  id: string;
  language: string;
  description?: string | null;
}

export interface TagTranslationUpdateInput {
  id: string;
  language?: string | null;
  description?: string | null;
}

export interface TagUpdateInput {
  anonymous?: boolean | null;
  tag: string;
  translationsDelete?: string[] | null;
  translationsCreate?: TagTranslationCreateInput[] | null;
  translationsUpdate?: TagTranslationUpdateInput[] | null;
}

export interface TimeFrame {
  after?: any | null;
  before?: any | null;
}

export interface TranslateInput {
  fields: string;
  languageSource: string;
  languageTarget: string;
}

export interface UserDeleteInput {
  password: string;
  deletePublicData: boolean;
}

export interface UserSearchInput {
  languages?: string[] | null;
  minStars?: number | null;
  minViews?: number | null;
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
  id: string;
  language: string;
  bio?: string | null;
}

export interface UserTranslationUpdateInput {
  id: string;
  language?: string | null;
  bio?: string | null;
}

export interface ValidateSessionInput {
  timeZone: string;
}

export interface ViewSearchInput {
  after?: string | null;
  lastViewedTimeFrame?: TimeFrame | null;
  ids?: string[] | null;
  searchString?: string | null;
  sortBy?: ViewSortBy | null;
  take?: number | null;
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
