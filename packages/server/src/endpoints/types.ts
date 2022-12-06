export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: string;
  String: string;
  Boolean: boolean;
  Int: number;
  Float: number;
  Date: any;
  Upload: any;
};

export enum AccountStatus {
  Deleted = 'Deleted',
  HardLocked = 'HardLocked',
  SoftLocked = 'SoftLocked',
  Unlocked = 'Unlocked'
}

export type Api = {
  __typename?: 'Api';
  id: Scalars['ID'];
};

export type ApiCreateInput = {
  id: Scalars['ID'];
};

export type ApiEdge = {
  __typename?: 'ApiEdge';
  cursor: Scalars['String'];
  node: Api;
};

export type ApiKey = {
  __typename?: 'ApiKey';
  absoluteMax: Scalars['Int'];
  creditsUsed: Scalars['Int'];
  creditsUsedBeforeLimit: Scalars['Int'];
  id: Scalars['ID'];
  resetsAt: Scalars['Date'];
  stopAtLimit: Scalars['Boolean'];
};

export type ApiKeyCreateInput = {
  absoluteMax: Scalars['Int'];
  creditsUsedBeforeLimit: Scalars['Int'];
  stopAtLimit: Scalars['Boolean'];
};

export type ApiKeyDeleteOneInput = {
  id: Scalars['ID'];
};

export type ApiKeyUpdateInput = {
  absoluteMax?: InputMaybe<Scalars['Int']>;
  creditsUsedBeforeLimit?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  stopAtLimit?: InputMaybe<Scalars['Boolean']>;
};

export type ApiKeyValidateInput = {
  id: Scalars['ID'];
  secret: Scalars['String'];
};

export type ApiPermission = {
  __typename?: 'ApiPermission';
  canCopy: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type ApiSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ApiSortBy>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ApiSearchResult = {
  __typename?: 'ApiSearchResult';
  edges: Array<ApiEdge>;
  pageInfo: PageInfo;
};

export enum ApiSortBy {
  CalledByRoutinesAsc = 'CalledByRoutinesAsc',
  CalledByRoutinesDesc = 'CalledByRoutinesDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  IssuesAsc = 'IssuesAsc',
  IssuesDesc = 'IssuesDesc',
  PullRequestsAsc = 'PullRequestsAsc',
  PullRequestsDesc = 'PullRequestsDesc',
  QuestionsAsc = 'QuestionsAsc',
  QuestionsDesc = 'QuestionsDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc',
  VotesAsc = 'VotesAsc',
  VotesDesc = 'VotesDesc'
}

export type ApiUpdateInput = {
  id: Scalars['ID'];
};

export type ApiVersion = {
  __typename?: 'ApiVersion';
  id: Scalars['ID'];
};

export type ApiVersionCreateInput = {
  id: Scalars['ID'];
};

export type ApiVersionEdge = {
  __typename?: 'ApiVersionEdge';
  cursor: Scalars['String'];
  node: Api;
};

export type ApiVersionPermission = {
  __typename?: 'ApiVersionPermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type ApiVersionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ApiVersionSearchResult = {
  __typename?: 'ApiVersionSearchResult';
  edges: Array<ApiEdge>;
  pageInfo: PageInfo;
};

export enum ApiVersionSortBy {
  CommentsAsc = 'CommentsAsc',
  CommentsDesc = 'CommentsDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  DirectoryListingsAsc = 'DirectoryListingsAsc',
  DirectoryListingsDesc = 'DirectoryListingsDesc',
  ForksAsc = 'ForksAsc',
  ForksDesc = 'ForksDesc',
  ReportsAsc = 'ReportsAsc',
  ReportsDesc = 'ReportsDesc'
}

export type ApiVersionTranslation = {
  __typename?: 'ApiVersionTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type ApiVersionTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type ApiVersionTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type ApiVersionUpdateInput = {
  id: Scalars['ID'];
};

export type Award = {
  __typename?: 'Award';
  category: AwardCategory;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  progress: Scalars['Int'];
  timeCurrentTierCompleted?: Maybe<Scalars['Date']>;
  updated_at: Scalars['Date'];
};

export enum AwardCategory {
  AccountAnniversary = 'AccountAnniversary',
  AccountNew = 'AccountNew',
  ApiCreate = 'ApiCreate',
  CommentCreate = 'CommentCreate',
  IssueCreate = 'IssueCreate',
  NoteCreate = 'NoteCreate',
  ObjectStar = 'ObjectStar',
  ObjectVote = 'ObjectVote',
  OrganizationCreate = 'OrganizationCreate',
  OrganizationJoin = 'OrganizationJoin',
  PostCreate = 'PostCreate',
  ProjectCreate = 'ProjectCreate',
  PullRequestComplete = 'PullRequestComplete',
  PullRequestCreate = 'PullRequestCreate',
  QuestionAnswer = 'QuestionAnswer',
  QuestionCreate = 'QuestionCreate',
  QuizPass = 'QuizPass',
  ReportContribute = 'ReportContribute',
  ReportEnd = 'ReportEnd',
  Reputation = 'Reputation',
  RoutineCreate = 'RoutineCreate',
  RunProject = 'RunProject',
  RunRoutine = 'RunRoutine',
  SmartContractCreate = 'SmartContractCreate',
  StandardCreate = 'StandardCreate',
  Streak = 'Streak',
  UserInvite = 'UserInvite'
}

export type Comment = {
  __typename?: 'Comment';
  commentedOn: CommentedOn;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  id: Scalars['ID'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  permissionsComment?: Maybe<CommentPermission>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  translations: Array<CommentTranslation>;
  updated_at: Scalars['Date'];
};

export type CommentCreateInput = {
  createdFor: CommentFor;
  forId: Scalars['ID'];
  id: Scalars['ID'];
  parentId?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<CommentTranslationCreateInput>>;
};

export enum CommentFor {
  Project = 'Project',
  Routine = 'Routine',
  Standard = 'Standard'
}

export type CommentPermission = {
  __typename?: 'CommentPermission';
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReply: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type CommentSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<CommentSortBy>;
  standardId?: InputMaybe<Scalars['ID']>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type CommentSearchResult = {
  __typename?: 'CommentSearchResult';
  endCursor?: Maybe<Scalars['String']>;
  threads?: Maybe<Array<CommentThread>>;
  totalThreads?: Maybe<Scalars['Int']>;
};

export enum CommentSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VotesAsc = 'VotesAsc',
  VotesDesc = 'VotesDesc'
}

export type CommentThread = {
  __typename?: 'CommentThread';
  childThreads: Array<CommentThread>;
  comment: Comment;
  endCursor?: Maybe<Scalars['String']>;
  totalInThread?: Maybe<Scalars['Int']>;
};

export type CommentTranslation = {
  __typename?: 'CommentTranslation';
  id: Scalars['ID'];
  language: Scalars['String'];
  text: Scalars['String'];
};

export type CommentTranslationCreateInput = {
  id: Scalars['ID'];
  language: Scalars['String'];
  text: Scalars['String'];
};

export type CommentTranslationUpdateInput = {
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  text?: InputMaybe<Scalars['String']>;
};

export type CommentUpdateInput = {
  id: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<CommentTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<CommentTranslationUpdateInput>>;
};

export type CommentedOn = Project | Routine | Standard;

export type Contributor = Organization | User;

export type CopyInput = {
  id: Scalars['ID'];
  intendToPullRequest: Scalars['Boolean'];
  objectType: CopyType;
};

export type CopyResult = {
  __typename?: 'CopyResult';
  organization?: Maybe<Organization>;
  project?: Maybe<Project>;
  routine?: Maybe<Routine>;
  standard?: Maybe<Standard>;
};

export enum CopyType {
  ApiVersion = 'ApiVersion',
  NoteVersion = 'NoteVersion',
  Organization = 'Organization',
  ProjectVersion = 'ProjectVersion',
  RoutineVersion = 'RoutineVersion',
  SmartContractVersion = 'SmartContractVersion',
  StandardVersion = 'StandardVersion'
}

export type Count = {
  __typename?: 'Count';
  count: Scalars['Int'];
};

export type DeleteManyInput = {
  ids: Array<Scalars['ID']>;
  objectType: DeleteType;
};

export type DeleteOneInput = {
  id: Scalars['ID'];
  objectType: DeleteType;
};

export enum DeleteType {
  Api = 'Api',
  ApiVersion = 'ApiVersion',
  Comment = 'Comment',
  Email = 'Email',
  Node = 'Node',
  Organization = 'Organization',
  Project = 'Project',
  ProjectVersion = 'ProjectVersion',
  PushDevice = 'PushDevice',
  Reminder = 'Reminder',
  ReminderList = 'ReminderList',
  Report = 'Report',
  Routine = 'Routine',
  RoutineVersion = 'RoutineVersion',
  Run = 'Run',
  SmartContract = 'SmartContract',
  SmartContractVersion = 'SmartContractVersion',
  Standard = 'Standard',
  StandardVersion = 'StandardVersion',
  Wallet = 'Wallet'
}

export type DevelopResult = {
  __typename?: 'DevelopResult';
  completed: Array<ProjectOrRoutine>;
  inProgress: Array<ProjectOrRoutine>;
  recent: Array<ProjectOrRoutine>;
};

export type Email = {
  __typename?: 'Email';
  emailAddress: Scalars['String'];
  id: Scalars['ID'];
  receivesAccountUpdates: Scalars['Boolean'];
  receivesBusinessUpdates: Scalars['Boolean'];
  user?: Maybe<User>;
  userId?: Maybe<Scalars['ID']>;
  verified: Scalars['Boolean'];
};

export type EmailCreateInput = {
  emailAddress: Scalars['String'];
  receivesAccountUpdates?: InputMaybe<Scalars['Boolean']>;
  receivesBusinessUpdates?: InputMaybe<Scalars['Boolean']>;
};

export type EmailLogInInput = {
  email?: InputMaybe<Scalars['String']>;
  password?: InputMaybe<Scalars['String']>;
  verificationCode?: InputMaybe<Scalars['String']>;
};

export type EmailRequestPasswordChangeInput = {
  email: Scalars['String'];
};

export type EmailResetPasswordInput = {
  code: Scalars['String'];
  id: Scalars['ID'];
  newPassword: Scalars['String'];
};

export type EmailSignUpInput = {
  confirmPassword: Scalars['String'];
  email: Scalars['String'];
  marketingEmails: Scalars['Boolean'];
  name: Scalars['String'];
  password: Scalars['String'];
  theme: Scalars['String'];
};

export type EmailUpdateInput = {
  id: Scalars['ID'];
  receivesAccountUpdates?: InputMaybe<Scalars['Boolean']>;
  receivesBusinessUpdates?: InputMaybe<Scalars['Boolean']>;
};

export type FeedbackInput = {
  text: Scalars['String'];
  userId?: InputMaybe<Scalars['ID']>;
};

export type FindByIdInput = {
  id: Scalars['ID'];
};

export type FindByIdOrHandleInput = {
  handle?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
};

export type FindByVersionInput = {
  id?: InputMaybe<Scalars['ID']>;
  versionGroupId?: InputMaybe<Scalars['ID']>;
};

export type FindHandlesInput = {
  organizationId?: InputMaybe<Scalars['ID']>;
};

export type Handle = {
  __typename?: 'Handle';
  handle: Scalars['String'];
  id: Scalars['ID'];
  wallet: Wallet;
};

export type HistoryInput = {
  searchString: Scalars['String'];
  take?: InputMaybe<Scalars['Int']>;
};

export type HistoryResult = {
  __typename?: 'HistoryResult';
  activeRuns: Array<RunRoutine>;
  completedRuns: Array<RunRoutine>;
  recentlyStarred: Array<Star>;
  recentlyViewed: Array<View>;
};

export type InputItem = {
  __typename?: 'InputItem';
  id: Scalars['ID'];
  isRequired?: Maybe<Scalars['Boolean']>;
  name?: Maybe<Scalars['String']>;
  routine: Routine;
  routineVersion: RoutineVersion;
  standard?: Maybe<Standard>;
  standardVersion?: Maybe<StandardVersion>;
  translations: Array<InputItemTranslation>;
};

export type InputItemCreateInput = {
  id: Scalars['ID'];
  isRequired?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  standardCreate?: InputMaybe<StandardCreateInput>;
  standardVersionConnect?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<InputItemTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<InputItemTranslationUpdateInput>>;
};

export type InputItemTranslation = {
  __typename?: 'InputItemTranslation';
  description?: Maybe<Scalars['String']>;
  helpText?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type InputItemTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  helpText?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type InputItemTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  helpText?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
};

export type InputItemUpdateInput = {
  id: Scalars['ID'];
  isRequired?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  standardConnect?: InputMaybe<Scalars['ID']>;
  standardCreate?: InputMaybe<StandardCreateInput>;
  translationsCreate?: InputMaybe<Array<InputItemTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<InputItemTranslationUpdateInput>>;
};

export type Issue = {
  __typename?: 'Issue';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type IssueCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootId: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type IssueEdge = {
  __typename?: 'IssueEdge';
  cursor: Scalars['String'];
  node: Api;
};

export type IssuePermission = {
  __typename?: 'IssuePermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type IssueSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type IssueSearchResult = {
  __typename?: 'IssueSearchResult';
  edges: Array<ApiEdge>;
  pageInfo: PageInfo;
};

export enum IssueSortBy {
  CommentsAsc = 'CommentsAsc',
  CommentsDesc = 'CommentsDesc',
  DateCompletedAsc = 'DateCompletedAsc',
  DateCompletedDesc = 'DateCompletedDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  ForksAsc = 'ForksAsc',
  ForksDesc = 'ForksDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VotesAsc = 'VotesAsc',
  VotesDesc = 'VotesDesc'
}

export type IssueTranslation = {
  __typename?: 'IssueTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type IssueTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type IssueTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type IssueUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type Label = {
  __typename?: 'Label';
  apis?: Maybe<Array<Api>>;
  color?: Maybe<Scalars['String']>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  issues?: Maybe<Array<Issue>>;
  label: Scalars['String'];
  meetings?: Maybe<Array<Meeting>>;
  notes?: Maybe<Array<Note>>;
  projects?: Maybe<Array<Project>>;
  routines?: Maybe<Array<Routine>>;
  runProjectSchedules?: Maybe<Array<RunProjectSchedule>>;
  runRoutineSchedules?: Maybe<Array<RunRoutineSchedule>>;
  smartContracts?: Maybe<Array<SmartContract>>;
  standards?: Maybe<Array<Standard>>;
  translations: Array<LabelTranslation>;
  updated_at: Scalars['Date'];
  userSchedules?: Maybe<Array<UserSchedule>>;
};

export type LabelCreateInput = {
  apisConnect?: InputMaybe<Array<Scalars['ID']>>;
  color?: InputMaybe<Scalars['String']>;
  createdByOrganizationId?: InputMaybe<Scalars['ID']>;
  createdByUserId?: InputMaybe<Scalars['ID']>;
  id: Scalars['ID'];
  issuesConnect?: InputMaybe<Array<Scalars['ID']>>;
  label: Scalars['String'];
  meetingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  notesConnect?: InputMaybe<Array<Scalars['ID']>>;
  projectsConnect?: InputMaybe<Array<Scalars['ID']>>;
  routinesConnect?: InputMaybe<Array<Scalars['ID']>>;
  runProjectSchedulesConnect?: InputMaybe<Array<Scalars['ID']>>;
  runRoutineSchedulesConnect?: InputMaybe<Array<Scalars['ID']>>;
  smartContractsConnect?: InputMaybe<Array<Scalars['ID']>>;
  standardsConnect?: InputMaybe<Array<Scalars['ID']>>;
  translationsCreate?: InputMaybe<Array<LabelTranslationCreateInput>>;
  userSchedulesConnect?: InputMaybe<Array<Scalars['ID']>>;
};

export type LabelEdge = {
  __typename?: 'LabelEdge';
  cursor: Scalars['String'];
  node: Label;
};

export type LabelPermission = {
  __typename?: 'LabelPermission';
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
};

export type LabelSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdByOrganizationId?: InputMaybe<Scalars['ID']>;
  createdByUserId?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<LabelSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type LabelSearchResult = {
  __typename?: 'LabelSearchResult';
  edges: Array<LabelEdge>;
  pageInfo: PageInfo;
};

export enum LabelSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type LabelTranslation = {
  __typename?: 'LabelTranslation';
  description: Scalars['String'];
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type LabelTranslationCreateInput = {
  description: Scalars['String'];
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type LabelTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
};

export type LabelUpdateInput = {
  apisConnect?: InputMaybe<Array<Scalars['ID']>>;
  apisDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  color?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  issuesConnect?: InputMaybe<Array<Scalars['ID']>>;
  issuesDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  label?: InputMaybe<Scalars['String']>;
  meetingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  meetingsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  notesConnect?: InputMaybe<Array<Scalars['ID']>>;
  notesDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  projectsConnect?: InputMaybe<Array<Scalars['ID']>>;
  projectsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  routinesConnect?: InputMaybe<Array<Scalars['ID']>>;
  routinesDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  runProjectSchedulesConnect?: InputMaybe<Array<Scalars['ID']>>;
  runProjectSchedulesDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  runRoutineSchedulesConnect?: InputMaybe<Array<Scalars['ID']>>;
  runRoutineSchedulesDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  smartContractsConnect?: InputMaybe<Array<Scalars['ID']>>;
  smartContractsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  standardsConnect?: InputMaybe<Array<Scalars['ID']>>;
  standardsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userSchedulesConnect?: InputMaybe<Array<Scalars['ID']>>;
  userSchedulesDisconnect?: InputMaybe<Array<Scalars['ID']>>;
};

export type LearnResult = {
  __typename?: 'LearnResult';
  courses: Array<Project>;
  tutorials: Array<Routine>;
};

export type LogOutInput = {
  id?: InputMaybe<Scalars['ID']>;
};

export type Loop = {
  __typename?: 'Loop';
  id: Scalars['ID'];
  loops?: Maybe<Scalars['Int']>;
  maxLoops?: Maybe<Scalars['Int']>;
  operation?: Maybe<Scalars['String']>;
  whiles: Array<LoopWhile>;
};

export type LoopCreateInput = {
  id: Scalars['ID'];
  loops?: InputMaybe<Scalars['Int']>;
  maxLoops?: InputMaybe<Scalars['Int']>;
  operation?: InputMaybe<Scalars['String']>;
  whilesCreate: Array<LoopWhileCreateInput>;
};

export type LoopUpdateInput = {
  id: Scalars['ID'];
  loops?: InputMaybe<Scalars['Int']>;
  maxLoops?: InputMaybe<Scalars['Int']>;
  operation?: InputMaybe<Scalars['String']>;
  whilesCreate: Array<LoopWhileCreateInput>;
  whilesDelete?: InputMaybe<Array<Scalars['ID']>>;
  whilesUpdate: Array<LoopWhileUpdateInput>;
};

export type LoopWhile = {
  __typename?: 'LoopWhile';
  condition: Scalars['String'];
  id: Scalars['ID'];
  toId?: Maybe<Scalars['ID']>;
  translations: Array<LoopWhileTranslation>;
};

export type LoopWhileCreateInput = {
  condition: Scalars['String'];
  id: Scalars['ID'];
  toId?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<LoopWhileTranslationCreateInput>>;
};

export type LoopWhileTranslation = {
  __typename?: 'LoopWhileTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  title: Scalars['String'];
};

export type LoopWhileTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  title: Scalars['String'];
};

export type LoopWhileTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  title?: InputMaybe<Scalars['String']>;
};

export type LoopWhileUpdateInput = {
  condition?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  toId?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<LoopWhileTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<LoopWhileTranslationUpdateInput>>;
};

export type Meeting = {
  __typename?: 'Meeting';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type MeetingCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootId: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type MeetingEdge = {
  __typename?: 'MeetingEdge';
  cursor: Scalars['String'];
  node: Api;
};

export type MeetingInvite = {
  __typename?: 'MeetingInvite';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type MeetingInviteCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootId: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type MeetingInviteEdge = {
  __typename?: 'MeetingInviteEdge';
  cursor: Scalars['String'];
  node: Api;
};

export type MeetingInvitePermission = {
  __typename?: 'MeetingInvitePermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type MeetingInviteSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type MeetingInviteSearchResult = {
  __typename?: 'MeetingInviteSearchResult';
  edges: Array<ApiEdge>;
  pageInfo: PageInfo;
};

export enum MeetingInviteSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  MemberNameAsc = 'MemberNameAsc',
  MemberNameDesc = 'MemberNameDesc'
}

export type MeetingInviteTranslation = {
  __typename?: 'MeetingInviteTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type MeetingInviteTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type MeetingInviteTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type MeetingInviteUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type MeetingPermission = {
  __typename?: 'MeetingPermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type MeetingSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type MeetingSearchResult = {
  __typename?: 'MeetingSearchResult';
  edges: Array<ApiEdge>;
  pageInfo: PageInfo;
};

export enum MeetingSortBy {
  AttendeesAsc = 'AttendeesAsc',
  AttendeesDesc = 'AttendeesDesc',
  EventEndAsc = 'EventEndAsc',
  EventEndDesc = 'EventEndDesc',
  EventStartAsc = 'EventStartAsc',
  EventStartDesc = 'EventStartDesc',
  InvitesAsc = 'InvitesAsc',
  InvitesDesc = 'InvitesDesc',
  RecurrEndAsc = 'RecurrEndAsc',
  RecurrEndDesc = 'RecurrEndDesc',
  RecurrStartAsc = 'RecurrStartAsc',
  RecurrStartDesc = 'RecurrStartDesc'
}

export type MeetingTranslation = {
  __typename?: 'MeetingTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type MeetingTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type MeetingTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type MeetingUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type Member = {
  __typename?: 'Member';
  organization: Organization;
  user: User;
};

export type Mutation = {
  __typename?: 'Mutation';
  apiCreate: Api;
  apiKeyCreate: ApiKey;
  apiKeyDeleteOne: Success;
  apiKeyUpdate: ApiKey;
  apiKeyValidate: ApiKey;
  apiUpdate: Api;
  apiVersionCreate: ApiVersion;
  apiVersionUpdate: ApiVersion;
  commentCreate: Comment;
  commentUpdate: Comment;
  copy: CopyResult;
  deleteMany: Count;
  deleteOne: Success;
  emailCreate: Email;
  emailLogIn: Session;
  emailRequestPasswordChange: Success;
  emailResetPassword: Session;
  emailSignUp: Session;
  emailUpdate: Email;
  exportData: Scalars['String'];
  feedbackCreate: Success;
  guestLogIn: Session;
  issueCreate: Issue;
  issueUpdate: Issue;
  labelCreate: Label;
  labelUpdate: Label;
  logOut: Session;
  meetingCreate: Meeting;
  meetingInviteCreate: MeetingInvite;
  meetingInviteUpdate: MeetingInvite;
  meetingUpdate: Meeting;
  nodeCreate: Node;
  nodeUpdate: Node;
  noteCreate: Note;
  noteUpdate: Api;
  noteVersionCreate: NoteVersion;
  noteVersionUpdate: NoteVersion;
  notificationMarkAllAsRead: Count;
  notificationMarkAsRead: Success;
  notificationSubscriptionCreate: NotificationSubscription;
  notificationSubscriptionUpdate: NotificationSubscription;
  organizationCreate: Organization;
  organizationUpdate: Organization;
  phoneCreate: Phone;
  phoneUpdate: Phone;
  postCreate: Post;
  postUpdate: Post;
  profileEmailUpdate: User;
  profileUpdate: User;
  projectCreate: Project;
  projectUpdate: Project;
  projectVersionCreate: ProjectVersion;
  projectVersionUpdate: ProjectVersion;
  pullRequestAccept: PullRequest;
  pullRequestCreate: PullRequest;
  pullRequestReject: PullRequest;
  pullRequestUpdate: PullRequest;
  pushDeviceCreate: PushDevice;
  pushDeviceUpdate: PushDevice;
  questionAnswerCreate: QuestionAnswer;
  questionAnswerUpdate: QuestionAnswer;
  questionCreate: Question;
  questionUpdate: Question;
  quizCreate: Quiz;
  quizUpdate: Quiz;
  reminderCreate: Reminder;
  reminderListCreate: ReminderList;
  reminderListUpdate: ReminderList;
  reminderUpdate: Reminder;
  reportCreate: Report;
  reportResponseCreate: ReportResponse;
  reportResponseUpdate: ReportResponse;
  reportUpdate: Report;
  resourceCreate: Resource;
  resourceListCreate: ResourceList;
  resourceListUpdate: ResourceList;
  resourceUpdate: Resource;
  routineCreate: Routine;
  routineUpdate: Routine;
  routineVersionCreate: RoutineVersion;
  routineVersionUpdate: RoutineVersion;
  runProjectCreate: RunProject;
  runProjectStepCreate: RunProjectStep;
  runProjectStepUpdate: RunProjectStep;
  runProjectUpdate: RunProject;
  runRoutineCancel: RunRoutine;
  runRoutineComplete: RunRoutine;
  runRoutineCreate: RunRoutine;
  runRoutineDeleteAll: Count;
  runRoutineUpdate: RunRoutine;
  sendVerificationEmail: Success;
  sendVerificationText: Success;
  smartContractCreate: SmartContract;
  smartContractUpdate: Api;
  smartContractVersionCreate: SmartContractVersion;
  smartContractVersionUpdate: SmartContractVersion;
  standardCreate: Standard;
  standardUpdate: Standard;
  standardVersionCreate: StandardVersion;
  standardVersionUpdate: StandardVersion;
  star: Success;
  switchCurrentAccount: Session;
  tagCreate: Tag;
  tagUpdate: Tag;
  transferCreate: Transfer;
  transferUpdate: Transfer;
  userDeleteOne: Success;
  userScheduleCreate: UserSchedule;
  userScheduleUpdate: UserSchedule;
  validateSession: Session;
  vote: Success;
  walletComplete: WalletComplete;
  walletInit: Scalars['String'];
  walletUpdate: Wallet;
  writeAssets?: Maybe<Scalars['Boolean']>;
};


export type MutationApiCreateArgs = {
  input: ApiCreateInput;
};


export type MutationApiKeyCreateArgs = {
  input: ApiKeyCreateInput;
};


export type MutationApiKeyDeleteOneArgs = {
  input: ApiKeyDeleteOneInput;
};


export type MutationApiKeyUpdateArgs = {
  input: ApiKeyUpdateInput;
};


export type MutationApiKeyValidateArgs = {
  input: ApiKeyValidateInput;
};


export type MutationApiUpdateArgs = {
  input: ApiUpdateInput;
};


export type MutationApiVersionCreateArgs = {
  input: ApiVersionCreateInput;
};


export type MutationApiVersionUpdateArgs = {
  input: ApiVersionUpdateInput;
};


export type MutationCommentCreateArgs = {
  input: CommentCreateInput;
};


export type MutationCommentUpdateArgs = {
  input: CommentUpdateInput;
};


export type MutationCopyArgs = {
  input: CopyInput;
};


export type MutationDeleteManyArgs = {
  input: DeleteManyInput;
};


export type MutationDeleteOneArgs = {
  input: DeleteOneInput;
};


export type MutationEmailCreateArgs = {
  input: EmailCreateInput;
};


export type MutationEmailLogInArgs = {
  input: EmailLogInInput;
};


export type MutationEmailRequestPasswordChangeArgs = {
  input: EmailRequestPasswordChangeInput;
};


export type MutationEmailResetPasswordArgs = {
  input: EmailResetPasswordInput;
};


export type MutationEmailSignUpArgs = {
  input: EmailSignUpInput;
};


export type MutationEmailUpdateArgs = {
  input: EmailUpdateInput;
};


export type MutationFeedbackCreateArgs = {
  input: FeedbackInput;
};


export type MutationIssueCreateArgs = {
  input: IssueCreateInput;
};


export type MutationIssueUpdateArgs = {
  input: IssueUpdateInput;
};


export type MutationLabelCreateArgs = {
  input: LabelCreateInput;
};


export type MutationLabelUpdateArgs = {
  input: LabelUpdateInput;
};


export type MutationLogOutArgs = {
  input: LogOutInput;
};


export type MutationMeetingCreateArgs = {
  input: MeetingCreateInput;
};


export type MutationMeetingInviteCreateArgs = {
  input: MeetingInviteCreateInput;
};


export type MutationMeetingInviteUpdateArgs = {
  input: MeetingInviteUpdateInput;
};


export type MutationMeetingUpdateArgs = {
  input: MeetingUpdateInput;
};


export type MutationNodeCreateArgs = {
  input: NodeCreateInput;
};


export type MutationNodeUpdateArgs = {
  input: NodeUpdateInput;
};


export type MutationNoteCreateArgs = {
  input: NoteCreateInput;
};


export type MutationNoteUpdateArgs = {
  input: NoteUpdateInput;
};


export type MutationNoteVersionCreateArgs = {
  input: NoteVersionCreateInput;
};


export type MutationNoteVersionUpdateArgs = {
  input: NoteVersionUpdateInput;
};


export type MutationNotificationMarkAsReadArgs = {
  input: FindByIdInput;
};


export type MutationNotificationSubscriptionCreateArgs = {
  input: NotificationSubscriptionCreateInput;
};


export type MutationNotificationSubscriptionUpdateArgs = {
  input: NotificationSubscriptionUpdateInput;
};


export type MutationOrganizationCreateArgs = {
  input: OrganizationCreateInput;
};


export type MutationOrganizationUpdateArgs = {
  input: OrganizationUpdateInput;
};


export type MutationPhoneCreateArgs = {
  input: PhoneCreateInput;
};


export type MutationPhoneUpdateArgs = {
  input: PhoneUpdateInput;
};


export type MutationPostCreateArgs = {
  input: PostCreateInput;
};


export type MutationPostUpdateArgs = {
  input: PostUpdateInput;
};


export type MutationProfileEmailUpdateArgs = {
  input: ProfileEmailUpdateInput;
};


export type MutationProfileUpdateArgs = {
  input: ProfileUpdateInput;
};


export type MutationProjectCreateArgs = {
  input: ProjectCreateInput;
};


export type MutationProjectUpdateArgs = {
  input: ProjectUpdateInput;
};


export type MutationProjectVersionCreateArgs = {
  input: ProjectVersionCreateInput;
};


export type MutationProjectVersionUpdateArgs = {
  input: ProjectVersionUpdateInput;
};


export type MutationPullRequestAcceptArgs = {
  input: FindByIdInput;
};


export type MutationPullRequestCreateArgs = {
  input: PullRequestCreateInput;
};


export type MutationPullRequestRejectArgs = {
  input: FindByIdInput;
};


export type MutationPullRequestUpdateArgs = {
  input: PullRequestUpdateInput;
};


export type MutationPushDeviceCreateArgs = {
  input: PushDeviceCreateInput;
};


export type MutationPushDeviceUpdateArgs = {
  input: PushDeviceUpdateInput;
};


export type MutationQuestionAnswerCreateArgs = {
  input: QuestionAnswerCreateInput;
};


export type MutationQuestionAnswerUpdateArgs = {
  input: QuestionAnswerUpdateInput;
};


export type MutationQuestionCreateArgs = {
  input: QuestionCreateInput;
};


export type MutationQuestionUpdateArgs = {
  input: QuestionUpdateInput;
};


export type MutationQuizCreateArgs = {
  input: QuizCreateInput;
};


export type MutationQuizUpdateArgs = {
  input: QuizUpdateInput;
};


export type MutationReminderCreateArgs = {
  input: ReminderCreateInput;
};


export type MutationReminderListCreateArgs = {
  input: ReminderListCreateInput;
};


export type MutationReminderListUpdateArgs = {
  input: ReminderListUpdateInput;
};


export type MutationReminderUpdateArgs = {
  input: ReminderUpdateInput;
};


export type MutationReportCreateArgs = {
  input: ReportCreateInput;
};


export type MutationReportResponseCreateArgs = {
  input: ReportResponseCreateInput;
};


export type MutationReportResponseUpdateArgs = {
  input: ReportResponseUpdateInput;
};


export type MutationReportUpdateArgs = {
  input: ReportUpdateInput;
};


export type MutationResourceCreateArgs = {
  input: ResourceCreateInput;
};


export type MutationResourceListCreateArgs = {
  input: ResourceListCreateInput;
};


export type MutationResourceListUpdateArgs = {
  input: ResourceListUpdateInput;
};


export type MutationResourceUpdateArgs = {
  input: ResourceUpdateInput;
};


export type MutationRoutineCreateArgs = {
  input: RoutineCreateInput;
};


export type MutationRoutineUpdateArgs = {
  input: RoutineUpdateInput;
};


export type MutationRoutineVersionCreateArgs = {
  input: RoutineVersionCreateInput;
};


export type MutationRoutineVersionUpdateArgs = {
  input: RoutineVersionUpdateInput;
};


export type MutationRunProjectCreateArgs = {
  input: RunProjectCreateInput;
};


export type MutationRunProjectStepCreateArgs = {
  input: RunProjectStepCreateInput;
};


export type MutationRunProjectStepUpdateArgs = {
  input: RunProjectStepUpdateInput;
};


export type MutationRunProjectUpdateArgs = {
  input: RunProjectUpdateInput;
};


export type MutationRunRoutineCancelArgs = {
  input: RunRoutineCancelInput;
};


export type MutationRunRoutineCompleteArgs = {
  input: RunRoutineCompleteInput;
};


export type MutationRunRoutineCreateArgs = {
  input: RunRoutineCreateInput;
};


export type MutationRunRoutineUpdateArgs = {
  input: RunRoutineUpdateInput;
};


export type MutationSendVerificationEmailArgs = {
  input: SendVerificationEmailInput;
};


export type MutationSendVerificationTextArgs = {
  input: SendVerificationTextInput;
};


export type MutationSmartContractCreateArgs = {
  input: SmartContractCreateInput;
};


export type MutationSmartContractUpdateArgs = {
  input: SmartContractUpdateInput;
};


export type MutationSmartContractVersionCreateArgs = {
  input: SmartContractVersionCreateInput;
};


export type MutationSmartContractVersionUpdateArgs = {
  input: SmartContractVersionUpdateInput;
};


export type MutationStandardCreateArgs = {
  input: StandardCreateInput;
};


export type MutationStandardUpdateArgs = {
  input: StandardUpdateInput;
};


export type MutationStandardVersionCreateArgs = {
  input: StandardVersionCreateInput;
};


export type MutationStandardVersionUpdateArgs = {
  input: StandardVersionUpdateInput;
};


export type MutationStarArgs = {
  input: StarInput;
};


export type MutationSwitchCurrentAccountArgs = {
  input: SwitchCurrentAccountInput;
};


export type MutationTagCreateArgs = {
  input: TagCreateInput;
};


export type MutationTagUpdateArgs = {
  input: TagUpdateInput;
};


export type MutationTransferCreateArgs = {
  input: TransferCreateInput;
};


export type MutationTransferUpdateArgs = {
  input: TransferUpdateInput;
};


export type MutationUserDeleteOneArgs = {
  input: UserDeleteInput;
};


export type MutationUserScheduleCreateArgs = {
  input: UserScheduleCreateInput;
};


export type MutationUserScheduleUpdateArgs = {
  input: UserScheduleUpdateInput;
};


export type MutationValidateSessionArgs = {
  input: ValidateSessionInput;
};


export type MutationVoteArgs = {
  input: VoteInput;
};


export type MutationWalletCompleteArgs = {
  input: WalletCompleteInput;
};


export type MutationWalletInitArgs = {
  input: WalletInitInput;
};


export type MutationWalletUpdateArgs = {
  input: WalletUpdateInput;
};


export type MutationWriteAssetsArgs = {
  input: WriteAssetsInput;
};

export type Node = {
  __typename?: 'Node';
  columnIndex?: Maybe<Scalars['Int']>;
  created_at: Scalars['Date'];
  data?: Maybe<NodeData>;
  id: Scalars['ID'];
  loop?: Maybe<Loop>;
  routine: Routine;
  routineId: Scalars['ID'];
  rowIndex?: Maybe<Scalars['Int']>;
  translations: Array<NodeTranslation>;
  type: NodeType;
  updated_at: Scalars['Date'];
};

export type NodeCreateInput = {
  columnIndex?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  loopCreate?: InputMaybe<LoopCreateInput>;
  nodeEndCreate?: InputMaybe<NodeEndCreateInput>;
  nodeRoutineListCreate?: InputMaybe<NodeRoutineListCreateInput>;
  routineVersionId: Scalars['ID'];
  rowIndex?: InputMaybe<Scalars['Int']>;
  translationsCreate?: InputMaybe<Array<NodeTranslationCreateInput>>;
  type: NodeType;
};

export type NodeData = NodeEnd | NodeRoutineList;

export type NodeEnd = {
  __typename?: 'NodeEnd';
  id: Scalars['ID'];
  wasSuccessful: Scalars['Boolean'];
};

export type NodeEndCreateInput = {
  id: Scalars['ID'];
  wasSuccessful?: InputMaybe<Scalars['Boolean']>;
};

export type NodeEndUpdateInput = {
  id: Scalars['ID'];
  wasSuccessful?: InputMaybe<Scalars['Boolean']>;
};

export type NodeLink = {
  __typename?: 'NodeLink';
  fromId: Scalars['ID'];
  id: Scalars['ID'];
  operation?: Maybe<Scalars['String']>;
  toId: Scalars['ID'];
  whens: Array<NodeLinkWhen>;
};

export type NodeLinkCreateInput = {
  fromId: Scalars['ID'];
  id: Scalars['ID'];
  operation?: InputMaybe<Scalars['String']>;
  toId: Scalars['ID'];
  whens?: InputMaybe<Array<NodeLinkWhenCreateInput>>;
};

export type NodeLinkUpdateInput = {
  fromId?: InputMaybe<Scalars['ID']>;
  id: Scalars['ID'];
  operation?: InputMaybe<Scalars['String']>;
  toId?: InputMaybe<Scalars['ID']>;
  whensCreate?: InputMaybe<Array<NodeLinkWhenCreateInput>>;
  whensDelete?: InputMaybe<Array<Scalars['ID']>>;
  whensUpdate?: InputMaybe<Array<NodeLinkWhenUpdateInput>>;
};

export type NodeLinkWhen = {
  __typename?: 'NodeLinkWhen';
  condition: Scalars['String'];
  id: Scalars['ID'];
  translations: Array<NodeLinkWhenTranslation>;
};

export type NodeLinkWhenCreateInput = {
  condition: Scalars['String'];
  id: Scalars['ID'];
  linkId?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<NodeLinkWhenTranslationCreateInput>>;
};

export type NodeLinkWhenTranslation = {
  __typename?: 'NodeLinkWhenTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  title: Scalars['String'];
};

export type NodeLinkWhenTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  title: Scalars['String'];
};

export type NodeLinkWhenTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  title?: InputMaybe<Scalars['String']>;
};

export type NodeLinkWhenUpdateInput = {
  condition?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  linkId?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<NodeLinkWhenTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<NodeLinkWhenTranslationUpdateInput>>;
};

export type NodeRoutineList = {
  __typename?: 'NodeRoutineList';
  id: Scalars['ID'];
  isOptional: Scalars['Boolean'];
  isOrdered: Scalars['Boolean'];
  routines: Array<NodeRoutineListItem>;
};

export type NodeRoutineListCreateInput = {
  id: Scalars['ID'];
  isOptional?: InputMaybe<Scalars['Boolean']>;
  isOrdered?: InputMaybe<Scalars['Boolean']>;
  routinesCreate?: InputMaybe<Array<NodeRoutineListItemCreateInput>>;
};

export type NodeRoutineListItem = {
  __typename?: 'NodeRoutineListItem';
  id: Scalars['ID'];
  index: Scalars['Int'];
  isOptional: Scalars['Boolean'];
  routineVersion: Routine;
  translations: Array<NodeRoutineListItemTranslation>;
};

export type NodeRoutineListItemCreateInput = {
  id: Scalars['ID'];
  index: Scalars['Int'];
  isOptional?: InputMaybe<Scalars['Boolean']>;
  routineVersionConnect: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<NodeRoutineListItemTranslationCreateInput>>;
};

export type NodeRoutineListItemTranslation = {
  __typename?: 'NodeRoutineListItemTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  title?: Maybe<Scalars['String']>;
};

export type NodeRoutineListItemTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  title?: InputMaybe<Scalars['String']>;
};

export type NodeRoutineListItemTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  title?: InputMaybe<Scalars['String']>;
};

export type NodeRoutineListItemUpdateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  isOptional?: InputMaybe<Scalars['Boolean']>;
  routineUpdate?: InputMaybe<RoutineUpdateInput>;
  translationsCreate?: InputMaybe<Array<NodeRoutineListItemTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<NodeRoutineListItemTranslationUpdateInput>>;
};

export type NodeRoutineListUpdateInput = {
  id: Scalars['ID'];
  isOptional?: InputMaybe<Scalars['Boolean']>;
  isOrdered?: InputMaybe<Scalars['Boolean']>;
  routinesCreate?: InputMaybe<Array<NodeRoutineListItemCreateInput>>;
  routinesDelete?: InputMaybe<Array<Scalars['ID']>>;
  routinesUpdate?: InputMaybe<Array<NodeRoutineListItemUpdateInput>>;
};

export type NodeTranslation = {
  __typename?: 'NodeTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  title: Scalars['String'];
};

export type NodeTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  title: Scalars['String'];
};

export type NodeTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  title?: InputMaybe<Scalars['String']>;
};

export enum NodeType {
  End = 'End',
  Redirect = 'Redirect',
  RoutineList = 'RoutineList',
  Start = 'Start'
}

export type NodeUpdateInput = {
  columnIndex?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  loopCreate?: InputMaybe<LoopCreateInput>;
  loopDelete?: InputMaybe<Scalars['ID']>;
  loopUpdate?: InputMaybe<LoopUpdateInput>;
  nodeEndCreate?: InputMaybe<NodeEndCreateInput>;
  nodeEndUpdate?: InputMaybe<NodeEndUpdateInput>;
  nodeRoutineListCreate?: InputMaybe<NodeRoutineListCreateInput>;
  nodeRoutineListUpdate?: InputMaybe<NodeRoutineListUpdateInput>;
  routineVersionId?: InputMaybe<Scalars['ID']>;
  rowIndex?: InputMaybe<Scalars['Int']>;
  translationsCreate?: InputMaybe<Array<NodeTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<NodeTranslationUpdateInput>>;
  type?: InputMaybe<NodeType>;
};

export type Note = {
  __typename?: 'Note';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type NoteCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootId: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type NoteEdge = {
  __typename?: 'NoteEdge';
  cursor: Scalars['String'];
  node: Note;
};

export type NotePermission = {
  __typename?: 'NotePermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type NoteSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type NoteSearchResult = {
  __typename?: 'NoteSearchResult';
  edges: Array<NoteEdge>;
  pageInfo: PageInfo;
};

export enum NoteSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  IssuesAsc = 'IssuesAsc',
  IssuesDesc = 'IssuesDesc',
  PullRequestsAsc = 'PullRequestsAsc',
  PullRequestsDesc = 'PullRequestsDesc',
  QuestionsAsc = 'QuestionsAsc',
  QuestionsDesc = 'QuestionsDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc',
  VotesAsc = 'VotesAsc',
  VotesDesc = 'VotesDesc'
}

export type NoteTranslation = {
  __typename?: 'NoteTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type NoteTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type NoteTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type NoteUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type NoteVersion = {
  __typename?: 'NoteVersion';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type NoteVersionCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootId: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type NoteVersionEdge = {
  __typename?: 'NoteVersionEdge';
  cursor: Scalars['String'];
  node: SmartContract;
};

export type NoteVersionPermission = {
  __typename?: 'NoteVersionPermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type NoteVersionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type NoteVersionSearchResult = {
  __typename?: 'NoteVersionSearchResult';
  edges: Array<SmartContractEdge>;
  pageInfo: PageInfo;
};

export enum NoteVersionSortBy {
  CommentsAsc = 'CommentsAsc',
  CommentsDesc = 'CommentsDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  DirectoryListingsAsc = 'DirectoryListingsAsc',
  DirectoryListingsDesc = 'DirectoryListingsDesc',
  ForksAsc = 'ForksAsc',
  ForksDesc = 'ForksDesc',
  ReportsAsc = 'ReportsAsc',
  ReportsDesc = 'ReportsDesc'
}

export type NoteVersionTranslation = {
  __typename?: 'NoteVersionTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type NoteVersionTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type NoteVersionTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type NoteVersionUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type Notification = {
  __typename?: 'Notification';
  category: Scalars['String'];
  created_at: Scalars['Date'];
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  imgLink?: Maybe<Scalars['String']>;
  isRead: Scalars['Boolean'];
  link?: Maybe<Scalars['String']>;
  title: Scalars['String'];
};

export type NotificationEdge = {
  __typename?: 'NotificationEdge';
  cursor: Scalars['String'];
  node: Notification;
};

export type NotificationSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<NotificationSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type NotificationSearchResult = {
  __typename?: 'NotificationSearchResult';
  edges: Array<NotificationEdge>;
  pageInfo: PageInfo;
};

export enum NotificationSortBy {
  CategoryAsc = 'CategoryAsc',
  CategoryDesc = 'CategoryDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  TitleAsc = 'TitleAsc',
  TitleDesc = 'TitleDesc'
}

export type NotificationSubscription = {
  __typename?: 'NotificationSubscription';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type NotificationSubscriptionCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootId: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type NotificationSubscriptionEdge = {
  __typename?: 'NotificationSubscriptionEdge';
  cursor: Scalars['String'];
  node: Api;
};

export type NotificationSubscriptionPermission = {
  __typename?: 'NotificationSubscriptionPermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type NotificationSubscriptionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type NotificationSubscriptionSearchResult = {
  __typename?: 'NotificationSubscriptionSearchResult';
  edges: Array<ApiEdge>;
  pageInfo: PageInfo;
};

export enum NotificationSubscriptionSortBy {
  ObjectTypeAsc = 'ObjectTypeAsc',
  ObjectTypeDesc = 'ObjectTypeDesc'
}

export type NotificationSubscriptionTranslation = {
  __typename?: 'NotificationSubscriptionTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type NotificationSubscriptionTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type NotificationSubscriptionTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type NotificationSubscriptionUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type Organization = {
  __typename?: 'Organization';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  created_at: Scalars['Date'];
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isOpenToNewMembers: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isViewed: Scalars['Boolean'];
  members: Array<Member>;
  membersCount: Scalars['Int'];
  permissionsOrganization?: Maybe<OrganizationPermission>;
  projects: Array<Project>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists: Array<ResourceList>;
  roles?: Maybe<Array<Role>>;
  routines: Array<Routine>;
  routinesCreated: Array<Routine>;
  starredBy: Array<User>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<OrganizationTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets: Array<Wallet>;
};

export type OrganizationCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isOpenToNewMembers?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  roles?: InputMaybe<Array<RoleCreateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<OrganizationTranslationCreateInput>>;
};

export type OrganizationEdge = {
  __typename?: 'OrganizationEdge';
  cursor: Scalars['String'];
  node: Organization;
};

export type OrganizationPermission = {
  __typename?: 'OrganizationPermission';
  canAddMembers: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  isMember: Scalars['Boolean'];
};

export type OrganizationSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isOpenToNewMembers?: InputMaybe<Scalars['Boolean']>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  projectId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<OrganizationSortBy>;
  standardId?: InputMaybe<Scalars['ID']>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type OrganizationSearchResult = {
  __typename?: 'OrganizationSearchResult';
  edges: Array<OrganizationEdge>;
  pageInfo: PageInfo;
};

export enum OrganizationSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc'
}

export type OrganizationTranslation = {
  __typename?: 'OrganizationTranslation';
  bio?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type OrganizationTranslationCreateInput = {
  bio?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type OrganizationTranslationUpdateInput = {
  bio?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type OrganizationUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isOpenToNewMembers?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  membersConnect?: InputMaybe<Array<Scalars['ID']>>;
  membersDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  rolesCreate?: InputMaybe<Array<RoleCreateInput>>;
  rolesDelete?: InputMaybe<Array<Scalars['ID']>>;
  rolesUpdate?: InputMaybe<Array<RoleUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<OrganizationTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<OrganizationTranslationUpdateInput>>;
};

export type OutputItem = {
  __typename?: 'OutputItem';
  id: Scalars['ID'];
  name?: Maybe<Scalars['String']>;
  routine: Routine;
  routineVersion: RoutineVersion;
  standard?: Maybe<Standard>;
  standardVersion?: Maybe<StandardVersion>;
  translations: Array<OutputItemTranslation>;
};

export type OutputItemCreateInput = {
  id: Scalars['ID'];
  name?: InputMaybe<Scalars['String']>;
  standardCreate?: InputMaybe<StandardCreateInput>;
  standardVersionConnect?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<OutputItemTranslationCreateInput>>;
};

export type OutputItemTranslation = {
  __typename?: 'OutputItemTranslation';
  description?: Maybe<Scalars['String']>;
  helpText?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type OutputItemTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  helpText?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type OutputItemTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  helpText?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
};

export type OutputItemUpdateInput = {
  id: Scalars['ID'];
  name?: InputMaybe<Scalars['String']>;
  standardCreate?: InputMaybe<StandardCreateInput>;
  standardVersionConnect?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<OutputItemTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<OutputItemTranslationUpdateInput>>;
};

export type PageInfo = {
  __typename?: 'PageInfo';
  endCursor?: Maybe<Scalars['String']>;
  hasNextPage: Scalars['Boolean'];
};

export type Payment = {
  __typename?: 'Payment';
  id: Scalars['ID'];
};

export type Phone = {
  __typename?: 'Phone';
  emailAddress: Scalars['String'];
  id: Scalars['ID'];
  receivesAccountUpdates: Scalars['Boolean'];
  receivesBusinessUpdates: Scalars['Boolean'];
  user?: Maybe<User>;
  userId?: Maybe<Scalars['ID']>;
  verified: Scalars['Boolean'];
};

export type PhoneCreateInput = {
  emailAddress: Scalars['String'];
  receivesAccountUpdates?: InputMaybe<Scalars['Boolean']>;
  receivesBusinessUpdates?: InputMaybe<Scalars['Boolean']>;
};

export type PhoneUpdateInput = {
  id: Scalars['ID'];
  receivesAccountUpdates?: InputMaybe<Scalars['Boolean']>;
  receivesBusinessUpdates?: InputMaybe<Scalars['Boolean']>;
};

export type PopularInput = {
  searchString: Scalars['String'];
  take?: InputMaybe<Scalars['Int']>;
};

export type PopularResult = {
  __typename?: 'PopularResult';
  organizations: Array<Organization>;
  projects: Array<Project>;
  routines: Array<Routine>;
  standards: Array<Standard>;
  users: Array<User>;
};

export type Post = {
  __typename?: 'Post';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isOwn: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  tag: Scalars['String'];
  translations: Array<PostTranslation>;
  updated_at: Scalars['Date'];
};

export type PostCreateInput = {
  anonymous?: InputMaybe<Scalars['Boolean']>;
  tag: Scalars['String'];
  translationsCreate?: InputMaybe<Array<PostTranslationCreateInput>>;
};

export type PostEdge = {
  __typename?: 'PostEdge';
  cursor: Scalars['String'];
  node: Post;
};

export type PostHidden = {
  __typename?: 'PostHidden';
  id: Scalars['ID'];
  isBlur: Scalars['Boolean'];
  tag: Post;
};

export type PostHiddenCreateInput = {
  id: Scalars['ID'];
  isBlur?: InputMaybe<Scalars['Boolean']>;
  tagConnect?: InputMaybe<Scalars['ID']>;
  tagCreate?: InputMaybe<PostCreateInput>;
};

export type PostHiddenUpdateInput = {
  id: Scalars['ID'];
  isBlur?: InputMaybe<Scalars['Boolean']>;
};

export type PostSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  hidden?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minStars?: InputMaybe<Scalars['Int']>;
  myPosts?: InputMaybe<Scalars['Boolean']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<PostSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type PostSearchResult = {
  __typename?: 'PostSearchResult';
  edges: Array<PostEdge>;
  pageInfo: PageInfo;
};

export enum PostSortBy {
  CommentsAsc = 'CommentsAsc',
  CommentsDesc = 'CommentsDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  ReportsAsc = 'ReportsAsc',
  ReportsDesc = 'ReportsDesc',
  RepostsAsc = 'RepostsAsc',
  RepostsDesc = 'RepostsDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc',
  VotesAsc = 'VotesAsc',
  VotesDesc = 'VotesDesc'
}

export type PostTranslation = {
  __typename?: 'PostTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type PostTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type PostTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
};

export type PostUpdateInput = {
  anonymous?: InputMaybe<Scalars['Boolean']>;
  tag: Scalars['String'];
  translationsCreate?: InputMaybe<Array<PostTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<PostTranslationUpdateInput>>;
};

export type Premium = {
  __typename?: 'Premium';
  id: Scalars['ID'];
};

export type ProfileEmailUpdateInput = {
  currentPassword: Scalars['String'];
  emailsCreate?: InputMaybe<Array<EmailCreateInput>>;
  emailsDelete?: InputMaybe<Array<Scalars['ID']>>;
  emailsUpdate?: InputMaybe<Array<EmailUpdateInput>>;
  newPassword?: InputMaybe<Scalars['String']>;
};

export type ProfileUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  hiddenTagsCreate?: InputMaybe<Array<TagHiddenCreateInput>>;
  hiddenTagsDelete?: InputMaybe<Array<Scalars['ID']>>;
  hiddenTagsUpdate?: InputMaybe<Array<TagHiddenUpdateInput>>;
  name?: InputMaybe<Scalars['String']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  starredTagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  starredTagsCreate?: InputMaybe<Array<TagCreateInput>>;
  starredTagsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  theme?: InputMaybe<Scalars['String']>;
  translationsCreate?: InputMaybe<Array<UserTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<UserTranslationUpdateInput>>;
};

export type Project = {
  __typename?: 'Project';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type ProjectCreateInput = {
  createdByOrganizationId?: InputMaybe<Scalars['ID']>;
  createdByUserId?: InputMaybe<Scalars['ID']>;
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type ProjectEdge = {
  __typename?: 'ProjectEdge';
  cursor: Scalars['String'];
  node: Project;
};

export type ProjectOrOrganization = Organization | Project;

export type ProjectOrOrganizationEdge = {
  __typename?: 'ProjectOrOrganizationEdge';
  cursor: Scalars['String'];
  node: ProjectOrOrganization;
};

export type ProjectOrOrganizationOrRoutineOrStandardOrUser = Organization | Project | Routine | Standard | User;

export type ProjectOrOrganizationPageInfo = {
  __typename?: 'ProjectOrOrganizationPageInfo';
  endCursorOrganization?: Maybe<Scalars['String']>;
  endCursorProject?: Maybe<Scalars['String']>;
  hasNextPage: Scalars['Boolean'];
};

export type ProjectOrOrganizationSearchInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  objectType?: InputMaybe<Scalars['String']>;
  organizationAfter?: InputMaybe<Scalars['String']>;
  organizationIsOpenToNewMembers?: InputMaybe<Scalars['Boolean']>;
  organizationProjectId?: InputMaybe<Scalars['ID']>;
  organizationRoutineId?: InputMaybe<Scalars['ID']>;
  projectAfter?: InputMaybe<Scalars['String']>;
  projectIsComplete?: InputMaybe<Scalars['Boolean']>;
  projectIsCompleteExceptions?: InputMaybe<Array<SearchException>>;
  projectMinScore?: InputMaybe<Scalars['Int']>;
  projectOrganizationId?: InputMaybe<Scalars['ID']>;
  projectParentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectOrOrganizationSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ProjectOrOrganizationSearchResult = {
  __typename?: 'ProjectOrOrganizationSearchResult';
  edges: Array<ProjectOrOrganizationEdge>;
  pageInfo: ProjectOrOrganizationPageInfo;
};

export enum ProjectOrOrganizationSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc'
}

export type ProjectOrRoutine = Project | Routine;

export type ProjectOrRoutineEdge = {
  __typename?: 'ProjectOrRoutineEdge';
  cursor: Scalars['String'];
  node: ProjectOrRoutine;
};

export type ProjectOrRoutinePageInfo = {
  __typename?: 'ProjectOrRoutinePageInfo';
  endCursorProject?: Maybe<Scalars['String']>;
  endCursorRoutine?: Maybe<Scalars['String']>;
  hasNextPage: Scalars['Boolean'];
};

export type ProjectOrRoutineSearchInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  objectType?: InputMaybe<Scalars['String']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  projectAfter?: InputMaybe<Scalars['String']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  routineAfter?: InputMaybe<Scalars['String']>;
  routineIsInternal?: InputMaybe<Scalars['Boolean']>;
  routineMaxComplexity?: InputMaybe<Scalars['Int']>;
  routineMaxSimplicity?: InputMaybe<Scalars['Int']>;
  routineMaxTimesCompleted?: InputMaybe<Scalars['Int']>;
  routineMinComplexity?: InputMaybe<Scalars['Int']>;
  routineMinSimplicity?: InputMaybe<Scalars['Int']>;
  routineMinTimesCompleted?: InputMaybe<Scalars['Int']>;
  routineProjectId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectOrRoutineSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ProjectOrRoutineSearchResult = {
  __typename?: 'ProjectOrRoutineSearchResult';
  edges: Array<ProjectOrRoutineEdge>;
  pageInfo: ProjectOrRoutinePageInfo;
};

export enum ProjectOrRoutineSortBy {
  DateCompletedAsc = 'DateCompletedAsc',
  DateCompletedDesc = 'DateCompletedDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  IssuesAsc = 'IssuesAsc',
  IssuesDesc = 'IssuesDesc',
  PullRequestsAsc = 'PullRequestsAsc',
  PullRequestsDesc = 'PullRequestsDesc',
  QuestionsAsc = 'QuestionsAsc',
  QuestionsDesc = 'QuestionsDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc',
  VotesAsc = 'VotesAsc',
  VotesDesc = 'VotesDesc'
}

export type ProjectPermission = {
  __typename?: 'ProjectPermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type ProjectSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ProjectSearchResult = {
  __typename?: 'ProjectSearchResult';
  edges: Array<ProjectEdge>;
  pageInfo: PageInfo;
};

export enum ProjectSortBy {
  DateCompletedAsc = 'DateCompletedAsc',
  DateCompletedDesc = 'DateCompletedDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  IssuesAsc = 'IssuesAsc',
  IssuesDesc = 'IssuesDesc',
  PullRequestsAsc = 'PullRequestsAsc',
  PullRequestsDesc = 'PullRequestsDesc',
  QuestionsAsc = 'QuestionsAsc',
  QuestionsDesc = 'QuestionsDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc',
  VotesAsc = 'VotesAsc',
  VotesDesc = 'VotesDesc'
}

export type ProjectTranslation = {
  __typename?: 'ProjectTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type ProjectTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type ProjectTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type ProjectUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type ProjectVersion = {
  __typename?: 'ProjectVersion';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type ProjectVersionCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootId: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type ProjectVersionEdge = {
  __typename?: 'ProjectVersionEdge';
  cursor: Scalars['String'];
  node: ProjectVersion;
};

export type ProjectVersionPermission = {
  __typename?: 'ProjectVersionPermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type ProjectVersionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ProjectVersionSearchResult = {
  __typename?: 'ProjectVersionSearchResult';
  edges: Array<ProjectVersionEdge>;
  pageInfo: PageInfo;
};

export enum ProjectVersionSortBy {
  CommentsAsc = 'CommentsAsc',
  CommentsDesc = 'CommentsDesc',
  ComplexityAsc = 'ComplexityAsc',
  ComplexityDesc = 'ComplexityDesc',
  DateCompletedAsc = 'DateCompletedAsc',
  DateCompletedDesc = 'DateCompletedDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  DirectoryListingsAsc = 'DirectoryListingsAsc',
  DirectoryListingsDesc = 'DirectoryListingsDesc',
  ForksAsc = 'ForksAsc',
  ForksDesc = 'ForksDesc',
  RunProjectsAsc = 'RunProjectsAsc',
  RunProjectsDesc = 'RunProjectsDesc',
  SimplicityAsc = 'SimplicityAsc',
  SimplicityDesc = 'SimplicityDesc'
}

export type ProjectVersionTranslation = {
  __typename?: 'ProjectVersionTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type ProjectVersionTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type ProjectVersionTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type ProjectVersionUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type PullRequest = {
  __typename?: 'PullRequest';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type PullRequestCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootId: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type PullRequestEdge = {
  __typename?: 'PullRequestEdge';
  cursor: Scalars['String'];
  node: Api;
};

export type PullRequestPermission = {
  __typename?: 'PullRequestPermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type PullRequestSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type PullRequestSearchResult = {
  __typename?: 'PullRequestSearchResult';
  edges: Array<ApiEdge>;
  pageInfo: PageInfo;
};

export enum PullRequestSortBy {
  CommentsAsc = 'CommentsAsc',
  CommentsDesc = 'CommentsDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type PullRequestTranslation = {
  __typename?: 'PullRequestTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type PullRequestTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type PullRequestTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type PullRequestUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type PushDevice = {
  __typename?: 'PushDevice';
  expires?: Maybe<Scalars['Date']>;
  id: Scalars['ID'];
  name?: Maybe<Scalars['String']>;
};

export type PushDeviceCreateInput = {
  endpoint: Scalars['String'];
  expires?: InputMaybe<Scalars['Date']>;
  keys: PushDeviceKeysInput;
  name?: InputMaybe<Scalars['String']>;
};

export type PushDeviceKeysInput = {
  auth: Scalars['String'];
  p256dh: Scalars['String'];
};

export type PushDeviceUpdateInput = {
  id: Scalars['ID'];
  name?: InputMaybe<Scalars['String']>;
};

export type Query = {
  __typename?: 'Query';
  api?: Maybe<Api>;
  apiVersion?: Maybe<ApiVersion>;
  apiVersions: ApiVersionSearchResult;
  apis: ApiSearchResult;
  comment?: Maybe<Comment>;
  comments: CommentSearchResult;
  develop: DevelopResult;
  findHandles: Array<Scalars['String']>;
  history: HistoryResult;
  issue?: Maybe<Issue>;
  issues: IssueSearchResult;
  label?: Maybe<Label>;
  labels: LabelSearchResult;
  learn: LearnResult;
  meeting?: Maybe<Meeting>;
  meetingInvite?: Maybe<MeetingInvite>;
  meetingInvites: MeetingInviteSearchResult;
  meetings: MeetingSearchResult;
  note?: Maybe<Note>;
  noteVersion?: Maybe<NoteVersion>;
  noteVersions: NoteVersionSearchResult;
  notes: NoteSearchResult;
  notification?: Maybe<Notification>;
  notificationSubscription?: Maybe<NotificationSubscription>;
  notificationSubscriptions: NotificationSubscriptionSearchResult;
  notifications: NotificationSearchResult;
  organization?: Maybe<Organization>;
  organizations: OrganizationSearchResult;
  popular: PopularResult;
  post?: Maybe<Post>;
  posts: PostSearchResult;
  profile: User;
  project?: Maybe<Project>;
  projectOrOrganizations: ProjectOrOrganizationSearchResult;
  projectOrRoutines: ProjectOrRoutineSearchResult;
  projectVersion?: Maybe<ProjectVersion>;
  projectVersions: ProjectVersionSearchResult;
  projects: ProjectSearchResult;
  pullRequest?: Maybe<PullRequest>;
  pullRequests: PullRequestSearchResult;
  pushDevices: Array<PushDevice>;
  question?: Maybe<Question>;
  questionAnswer?: Maybe<QuestionAnswer>;
  questionAnswers: QuestionAnswerSearchResult;
  questions: QuestionSearchResult;
  quiz?: Maybe<Quiz>;
  quizzes: QuizSearchResult;
  readAssets: Array<Maybe<Scalars['String']>>;
  reminder?: Maybe<Reminder>;
  reminderList?: Maybe<ReminderList>;
  reminderLists: ReminderListSearchResult;
  reminders: ReminderSearchResult;
  report?: Maybe<Report>;
  reportResponse?: Maybe<ReportResponse>;
  reportResponses: ReportResponseSearchResult;
  reports: ReportSearchResult;
  reputationHistories: ReputationHistorySearchResult;
  reputationHistory?: Maybe<ReputationHistory>;
  research: ResearchResult;
  resource?: Maybe<Resource>;
  resourceList?: Maybe<Resource>;
  resourceLists: ResourceListSearchResult;
  resources: ResourceSearchResult;
  routine?: Maybe<Routine>;
  routineVersion?: Maybe<RoutineVersion>;
  routineVersions: RoutineVersionSearchResult;
  routines: RoutineSearchResult;
  runProject?: Maybe<RunProject>;
  runProjectSchedule?: Maybe<RunProjectSchedule>;
  runProjectSchedules: RunProjectScheduleSearchResult;
  runProjectStep?: Maybe<RunProjectStep>;
  runProjectSteps: RunProjectStepSearchResult;
  runProjects: RunProjectSearchResult;
  runRoutine?: Maybe<RunRoutine>;
  runRoutineInputs: RunRoutineInputSearchResult;
  runRoutineSchedule?: Maybe<RunRoutineSchedule>;
  runRoutineSchedules: RunRoutineScheduleSearchResult;
  runRoutines: RunRoutineSearchResult;
  smartContract?: Maybe<SmartContract>;
  smartContractVersion?: Maybe<SmartContractVersion>;
  smartContractVersions: SmartContractVersionSearchResult;
  smartContracts: SmartContractSearchResult;
  standard?: Maybe<Standard>;
  standardVersion?: Maybe<StandardVersion>;
  standardVersions: StandardVersionSearchResult;
  standards: StandardSearchResult;
  stars: StarSearchResult;
  statsSite: StatsSiteSearchResult;
  tag?: Maybe<Tag>;
  tags: TagSearchResult;
  transfer?: Maybe<Transfer>;
  transfers: TransferSearchResult;
  translate: Translate;
  user?: Maybe<User>;
  userSchedule?: Maybe<UserSchedule>;
  userSchedules: UserScheduleSearchResult;
  users: UserSearchResult;
  views: ViewSearchResult;
};


export type QueryApiArgs = {
  input: FindByIdInput;
};


export type QueryApiVersionArgs = {
  input: FindByIdInput;
};


export type QueryApiVersionsArgs = {
  input: ApiVersionSearchInput;
};


export type QueryApisArgs = {
  input: ApiSearchInput;
};


export type QueryCommentArgs = {
  input: FindByIdInput;
};


export type QueryCommentsArgs = {
  input: CommentSearchInput;
};


export type QueryFindHandlesArgs = {
  input: FindHandlesInput;
};


export type QueryHistoryArgs = {
  input: HistoryInput;
};


export type QueryIssueArgs = {
  input: FindByIdInput;
};


export type QueryIssuesArgs = {
  input: IssueSearchInput;
};


export type QueryLabelArgs = {
  input: FindByIdInput;
};


export type QueryLabelsArgs = {
  input: LabelSearchInput;
};


export type QueryMeetingArgs = {
  input: FindByIdInput;
};


export type QueryMeetingInviteArgs = {
  input: FindByIdInput;
};


export type QueryMeetingInvitesArgs = {
  input: MeetingInviteSearchInput;
};


export type QueryMeetingsArgs = {
  input: MeetingSearchInput;
};


export type QueryNoteArgs = {
  input: FindByIdInput;
};


export type QueryNoteVersionArgs = {
  input: FindByIdInput;
};


export type QueryNoteVersionsArgs = {
  input: NoteVersionSearchInput;
};


export type QueryNotesArgs = {
  input: NoteSearchInput;
};


export type QueryNotificationArgs = {
  input: FindByIdInput;
};


export type QueryNotificationSubscriptionArgs = {
  input: FindByIdInput;
};


export type QueryNotificationSubscriptionsArgs = {
  input: NotificationSubscriptionSearchInput;
};


export type QueryNotificationsArgs = {
  input: NotificationSearchInput;
};


export type QueryOrganizationArgs = {
  input: FindByIdOrHandleInput;
};


export type QueryOrganizationsArgs = {
  input: OrganizationSearchInput;
};


export type QueryPopularArgs = {
  input: PopularInput;
};


export type QueryPostArgs = {
  input: FindByIdInput;
};


export type QueryPostsArgs = {
  input: PostSearchInput;
};


export type QueryProjectArgs = {
  input: FindByIdOrHandleInput;
};


export type QueryProjectOrOrganizationsArgs = {
  input: ProjectOrOrganizationSearchInput;
};


export type QueryProjectOrRoutinesArgs = {
  input: ProjectOrRoutineSearchInput;
};


export type QueryProjectVersionArgs = {
  input: FindByIdInput;
};


export type QueryProjectVersionsArgs = {
  input: ProjectVersionSearchInput;
};


export type QueryProjectsArgs = {
  input: ProjectSearchInput;
};


export type QueryPullRequestArgs = {
  input: FindByIdInput;
};


export type QueryPullRequestsArgs = {
  input: PullRequestSearchInput;
};


export type QueryQuestionArgs = {
  input: FindByIdInput;
};


export type QueryQuestionAnswerArgs = {
  input: FindByIdInput;
};


export type QueryQuestionAnswersArgs = {
  input: QuestionAnswerSearchInput;
};


export type QueryQuestionsArgs = {
  input: QuestionSearchInput;
};


export type QueryQuizArgs = {
  input: FindByIdInput;
};


export type QueryQuizzesArgs = {
  input: QuizSearchInput;
};


export type QueryReadAssetsArgs = {
  input: ReadAssetsInput;
};


export type QueryReminderArgs = {
  input: FindByIdInput;
};


export type QueryReminderListArgs = {
  input: FindByIdInput;
};


export type QueryReminderListsArgs = {
  input: ReminderListSearchInput;
};


export type QueryRemindersArgs = {
  input: ReminderSearchInput;
};


export type QueryReportArgs = {
  input: FindByIdInput;
};


export type QueryReportResponseArgs = {
  input: FindByIdInput;
};


export type QueryReportResponsesArgs = {
  input: ReportResponseSearchInput;
};


export type QueryReportsArgs = {
  input: ReportSearchInput;
};


export type QueryReputationHistoriesArgs = {
  input: ReputationHistorySearchInput;
};


export type QueryReputationHistoryArgs = {
  input: FindByIdInput;
};


export type QueryResourceArgs = {
  input: FindByIdInput;
};


export type QueryResourceListArgs = {
  input: FindByIdInput;
};


export type QueryResourceListsArgs = {
  input: ResourceListSearchInput;
};


export type QueryResourcesArgs = {
  input: ResourceSearchInput;
};


export type QueryRoutineArgs = {
  input: FindByVersionInput;
};


export type QueryRoutineVersionArgs = {
  input: FindByIdInput;
};


export type QueryRoutineVersionsArgs = {
  input: RoutineVersionSearchInput;
};


export type QueryRoutinesArgs = {
  input: RoutineSearchInput;
};


export type QueryRunProjectArgs = {
  input: FindByIdInput;
};


export type QueryRunProjectScheduleArgs = {
  input: FindByIdInput;
};


export type QueryRunProjectSchedulesArgs = {
  input: RunProjectScheduleSearchInput;
};


export type QueryRunProjectStepArgs = {
  input: FindByIdInput;
};


export type QueryRunProjectStepsArgs = {
  input: RunProjectStepSearchInput;
};


export type QueryRunProjectsArgs = {
  input: RunProjectSearchInput;
};


export type QueryRunRoutineArgs = {
  input: FindByIdInput;
};


export type QueryRunRoutineInputsArgs = {
  input: RunRoutineInputSearchInput;
};


export type QueryRunRoutineScheduleArgs = {
  input: FindByIdInput;
};


export type QueryRunRoutineSchedulesArgs = {
  input: RunRoutineScheduleSearchInput;
};


export type QueryRunRoutinesArgs = {
  input: RunRoutineSearchInput;
};


export type QuerySmartContractArgs = {
  input: FindByIdInput;
};


export type QuerySmartContractVersionArgs = {
  input: FindByIdInput;
};


export type QuerySmartContractVersionsArgs = {
  input: SmartContractVersionSearchInput;
};


export type QuerySmartContractsArgs = {
  input: SmartContractSearchInput;
};


export type QueryStandardArgs = {
  input: FindByVersionInput;
};


export type QueryStandardVersionArgs = {
  input: FindByIdInput;
};


export type QueryStandardVersionsArgs = {
  input: StandardVersionSearchInput;
};


export type QueryStandardsArgs = {
  input: StandardSearchInput;
};


export type QueryStarsArgs = {
  input: StarSearchInput;
};


export type QueryStatsSiteArgs = {
  input: StatsSiteSearchInput;
};


export type QueryTagArgs = {
  input: FindByIdInput;
};


export type QueryTagsArgs = {
  input: TagSearchInput;
};


export type QueryTransferArgs = {
  input: FindByIdInput;
};


export type QueryTransfersArgs = {
  input: TransferSearchInput;
};


export type QueryTranslateArgs = {
  input: TranslateInput;
};


export type QueryUserArgs = {
  input: FindByIdOrHandleInput;
};


export type QueryUserScheduleArgs = {
  input: FindByIdInput;
};


export type QueryUserSchedulesArgs = {
  input: UserScheduleSearchInput;
};


export type QueryUsersArgs = {
  input: UserSearchInput;
};


export type QueryViewsArgs = {
  input: ViewSearchInput;
};

export type Question = {
  __typename?: 'Question';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isOwn: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  tag: Scalars['String'];
  translations: Array<QuestionTranslation>;
  updated_at: Scalars['Date'];
};

export type QuestionAnswer = {
  __typename?: 'QuestionAnswer';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isOwn: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  tag: Scalars['String'];
  translations: Array<QuestionAnswerTranslation>;
  updated_at: Scalars['Date'];
};

export type QuestionAnswerCreateInput = {
  anonymous?: InputMaybe<Scalars['Boolean']>;
  tag: Scalars['String'];
  translationsCreate?: InputMaybe<Array<QuestionAnswerTranslationCreateInput>>;
};

export type QuestionAnswerEdge = {
  __typename?: 'QuestionAnswerEdge';
  cursor: Scalars['String'];
  node: QuestionAnswer;
};

export type QuestionAnswerHidden = {
  __typename?: 'QuestionAnswerHidden';
  id: Scalars['ID'];
  isBlur: Scalars['Boolean'];
  tag: QuestionAnswer;
};

export type QuestionAnswerHiddenCreateInput = {
  id: Scalars['ID'];
  isBlur?: InputMaybe<Scalars['Boolean']>;
  tagConnect?: InputMaybe<Scalars['ID']>;
  tagCreate?: InputMaybe<QuestionAnswerCreateInput>;
};

export type QuestionAnswerHiddenUpdateInput = {
  id: Scalars['ID'];
  isBlur?: InputMaybe<Scalars['Boolean']>;
};

export type QuestionAnswerSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  hidden?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minStars?: InputMaybe<Scalars['Int']>;
  myQuestionAnswers?: InputMaybe<Scalars['Boolean']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<QuestionAnswerSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type QuestionAnswerSearchResult = {
  __typename?: 'QuestionAnswerSearchResult';
  edges: Array<QuestionAnswerEdge>;
  pageInfo: PageInfo;
};

export enum QuestionAnswerSortBy {
  CommentsAsc = 'CommentsAsc',
  CommentsDesc = 'CommentsDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VotesAsc = 'VotesAsc',
  VotesDesc = 'VotesDesc'
}

export type QuestionAnswerTranslation = {
  __typename?: 'QuestionAnswerTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type QuestionAnswerTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type QuestionAnswerTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
};

export type QuestionAnswerUpdateInput = {
  anonymous?: InputMaybe<Scalars['Boolean']>;
  tag: Scalars['String'];
  translationsCreate?: InputMaybe<Array<QuestionAnswerTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<QuestionAnswerTranslationUpdateInput>>;
};

export type QuestionCreateInput = {
  anonymous?: InputMaybe<Scalars['Boolean']>;
  tag: Scalars['String'];
  translationsCreate?: InputMaybe<Array<QuestionTranslationCreateInput>>;
};

export type QuestionEdge = {
  __typename?: 'QuestionEdge';
  cursor: Scalars['String'];
  node: Question;
};

export type QuestionHidden = {
  __typename?: 'QuestionHidden';
  id: Scalars['ID'];
  isBlur: Scalars['Boolean'];
  tag: Question;
};

export type QuestionHiddenCreateInput = {
  id: Scalars['ID'];
  isBlur?: InputMaybe<Scalars['Boolean']>;
  tagConnect?: InputMaybe<Scalars['ID']>;
  tagCreate?: InputMaybe<QuestionCreateInput>;
};

export type QuestionHiddenUpdateInput = {
  id: Scalars['ID'];
  isBlur?: InputMaybe<Scalars['Boolean']>;
};

export type QuestionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  hidden?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minStars?: InputMaybe<Scalars['Int']>;
  myQuestions?: InputMaybe<Scalars['Boolean']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<QuestionSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type QuestionSearchResult = {
  __typename?: 'QuestionSearchResult';
  edges: Array<QuestionEdge>;
  pageInfo: PageInfo;
};

export enum QuestionSortBy {
  AnswersAsc = 'AnswersAsc',
  AnswersDesc = 'AnswersDesc',
  CommentsAsc = 'CommentsAsc',
  CommentsDesc = 'CommentsDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VotesAsc = 'VotesAsc',
  VotesDesc = 'VotesDesc'
}

export type QuestionTranslation = {
  __typename?: 'QuestionTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type QuestionTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type QuestionTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
};

export type QuestionUpdateInput = {
  anonymous?: InputMaybe<Scalars['Boolean']>;
  tag: Scalars['String'];
  translationsCreate?: InputMaybe<Array<QuestionTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<QuestionTranslationUpdateInput>>;
};

export type Quiz = {
  __typename?: 'Quiz';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type QuizCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootId: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type QuizEdge = {
  __typename?: 'QuizEdge';
  cursor: Scalars['String'];
  node: Api;
};

export type QuizPermission = {
  __typename?: 'QuizPermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type QuizSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type QuizSearchResult = {
  __typename?: 'QuizSearchResult';
  edges: Array<ApiEdge>;
  pageInfo: PageInfo;
};

export enum QuizSortBy {
  AttemptsAsc = 'AttemptsAsc',
  AttemptsDesc = 'AttemptsDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  QuestionsAsc = 'QuestionsAsc',
  QuestionsDesc = 'QuestionsDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VotesAsc = 'VotesAsc',
  VotesDesc = 'VotesDesc'
}

export type QuizTranslation = {
  __typename?: 'QuizTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type QuizTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type QuizTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type QuizUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type ReadAssetsInput = {
  files: Array<Scalars['String']>;
};

export type Reminder = {
  __typename?: 'Reminder';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  index?: Maybe<Scalars['Int']>;
  link: Scalars['String'];
  listId: Scalars['ID'];
  translations: Array<ReminderTranslation>;
  updated_at: Scalars['Date'];
};

export type ReminderCreateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  link: Scalars['String'];
  listId: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<ReminderTranslationCreateInput>>;
};

export type ReminderEdge = {
  __typename?: 'ReminderEdge';
  cursor: Scalars['String'];
  node: Reminder;
};

export type ReminderList = {
  __typename?: 'ReminderList';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  index?: Maybe<Scalars['Int']>;
  organization?: Maybe<Organization>;
  project?: Maybe<Project>;
  resources: Array<Resource>;
  routine?: Maybe<Routine>;
  standard?: Maybe<Standard>;
  translations: Array<ReminderListTranslation>;
  updated_at: Scalars['Date'];
};

export type ReminderListCreateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  resourcesCreate?: InputMaybe<Array<ResourceCreateInput>>;
  routineId?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<ReminderListTranslationCreateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type ReminderListEdge = {
  __typename?: 'ReminderListEdge';
  cursor: Scalars['String'];
  node: ReminderList;
};

export type ReminderListSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  organizationId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ReminderListSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type ReminderListSearchResult = {
  __typename?: 'ReminderListSearchResult';
  edges: Array<ReminderListEdge>;
  pageInfo: PageInfo;
};

export enum ReminderListSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type ReminderListTranslation = {
  __typename?: 'ReminderListTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  title?: Maybe<Scalars['String']>;
};

export type ReminderListTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  title?: InputMaybe<Scalars['String']>;
};

export type ReminderListTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  title?: InputMaybe<Scalars['String']>;
};

export type ReminderListUpdateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  resourcesCreate?: InputMaybe<Array<ResourceCreateInput>>;
  resourcesDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourcesUpdate?: InputMaybe<Array<ResourceUpdateInput>>;
  routineId?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<ReminderListTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ReminderListTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type ReminderSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  forId?: InputMaybe<Scalars['ID']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ReminderSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type ReminderSearchResult = {
  __typename?: 'ReminderSearchResult';
  edges: Array<ReminderEdge>;
  pageInfo: PageInfo;
};

export enum ReminderSortBy {
  DueDateAsc = 'DueDateAsc',
  DueDateDesc = 'DueDateDesc',
  NameAsc = 'NameAsc',
  NameDesc = 'NameDesc'
}

export type ReminderTranslation = {
  __typename?: 'ReminderTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  title?: Maybe<Scalars['String']>;
};

export type ReminderTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  title?: InputMaybe<Scalars['String']>;
};

export type ReminderTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  title?: InputMaybe<Scalars['String']>;
};

export type ReminderUpdateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  link?: InputMaybe<Scalars['String']>;
  listId?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<ReminderTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ReminderTranslationUpdateInput>>;
};

export type Report = {
  __typename?: 'Report';
  details?: Maybe<Scalars['String']>;
  id?: Maybe<Scalars['ID']>;
  isOwn: Scalars['Boolean'];
  language: Scalars['String'];
  reason: Scalars['String'];
};

export type ReportCreateInput = {
  createdFor: ReportFor;
  createdForId: Scalars['ID'];
  details?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  reason: Scalars['String'];
};

export type ReportEdge = {
  __typename?: 'ReportEdge';
  cursor: Scalars['String'];
  node: Report;
};

export enum ReportFor {
  ApiVersion = 'ApiVersion',
  Comment = 'Comment',
  Issue = 'Issue',
  NoteVersion = 'NoteVersion',
  Organization = 'Organization',
  Post = 'Post',
  ProjectVersion = 'ProjectVersion',
  RoutineVersion = 'RoutineVersion',
  StandardVersion = 'StandardVersion',
  Tag = 'Tag',
  User = 'User'
}

export type ReportResponse = {
  __typename?: 'ReportResponse';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type ReportResponseCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootId: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type ReportResponseEdge = {
  __typename?: 'ReportResponseEdge';
  cursor: Scalars['String'];
  node: Api;
};

export type ReportResponsePermission = {
  __typename?: 'ReportResponsePermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type ReportResponseSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ReportResponseSearchResult = {
  __typename?: 'ReportResponseSearchResult';
  edges: Array<ApiEdge>;
  pageInfo: PageInfo;
};

export enum ReportResponseSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc'
}

export type ReportResponseTranslation = {
  __typename?: 'ReportResponseTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type ReportResponseTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type ReportResponseTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type ReportResponseUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type ReportSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  organizationId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ReportSortBy>;
  standardId?: InputMaybe<Scalars['ID']>;
  tagId?: InputMaybe<Scalars['ID']>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type ReportSearchResult = {
  __typename?: 'ReportSearchResult';
  edges: Array<ReportEdge>;
  pageInfo: PageInfo;
};

export enum ReportSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  ResponsesAsc = 'ResponsesAsc',
  ResponsesDesc = 'ResponsesDesc'
}

export type ReportUpdateInput = {
  details?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  reason?: InputMaybe<Scalars['String']>;
};

export type ReputationHistory = {
  __typename?: 'ReputationHistory';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type ReputationHistoryEdge = {
  __typename?: 'ReputationHistoryEdge';
  cursor: Scalars['String'];
  node: Api;
};

export type ReputationHistorySearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ReputationHistorySearchResult = {
  __typename?: 'ReputationHistorySearchResult';
  edges: Array<ApiEdge>;
  pageInfo: PageInfo;
};

export enum ReputationHistorySortBy {
  AmountAsc = 'AmountAsc',
  AmountDesc = 'AmountDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc'
}

export type ResearchResult = {
  __typename?: 'ResearchResult';
  needInvestments: Array<Project>;
  needMembers: Array<Organization>;
  needVotes: Array<Project>;
  newlyCompleted: Array<ProjectOrRoutine>;
  processes: Array<Routine>;
};

export type Resource = {
  __typename?: 'Resource';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  index?: Maybe<Scalars['Int']>;
  link: Scalars['String'];
  listId: Scalars['ID'];
  translations: Array<ResourceTranslation>;
  updated_at: Scalars['Date'];
  usedFor?: Maybe<ResourceUsedFor>;
};

export type ResourceCreateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  link: Scalars['String'];
  listId: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<ResourceTranslationCreateInput>>;
  usedFor: ResourceUsedFor;
};

export type ResourceEdge = {
  __typename?: 'ResourceEdge';
  cursor: Scalars['String'];
  node: Resource;
};

export enum ResourceFor {
  Organization = 'Organization',
  Project = 'Project',
  Routine = 'Routine',
  User = 'User'
}

export type ResourceList = {
  __typename?: 'ResourceList';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  index?: Maybe<Scalars['Int']>;
  organization?: Maybe<Organization>;
  project?: Maybe<Project>;
  resources: Array<Resource>;
  routine?: Maybe<Routine>;
  standard?: Maybe<Standard>;
  translations: Array<ResourceListTranslation>;
  updated_at: Scalars['Date'];
  usedFor?: Maybe<ResourceListUsedFor>;
};

export type ResourceListCreateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  resourcesCreate?: InputMaybe<Array<ResourceCreateInput>>;
  routineId?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<ResourceListTranslationCreateInput>>;
  usedFor: ResourceListUsedFor;
  userId?: InputMaybe<Scalars['ID']>;
};

export type ResourceListEdge = {
  __typename?: 'ResourceListEdge';
  cursor: Scalars['String'];
  node: ResourceList;
};

export type ResourceListSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  organizationId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ResourceListSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type ResourceListSearchResult = {
  __typename?: 'ResourceListSearchResult';
  edges: Array<ResourceListEdge>;
  pageInfo: PageInfo;
};

export enum ResourceListSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  IndexAsc = 'IndexAsc',
  IndexDesc = 'IndexDesc'
}

export type ResourceListTranslation = {
  __typename?: 'ResourceListTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  title?: Maybe<Scalars['String']>;
};

export type ResourceListTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  title?: InputMaybe<Scalars['String']>;
};

export type ResourceListTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  title?: InputMaybe<Scalars['String']>;
};

export type ResourceListUpdateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  resourcesCreate?: InputMaybe<Array<ResourceCreateInput>>;
  resourcesDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourcesUpdate?: InputMaybe<Array<ResourceUpdateInput>>;
  routineId?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<ResourceListTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ResourceListTranslationUpdateInput>>;
  usedFor?: InputMaybe<ResourceListUsedFor>;
  userId?: InputMaybe<Scalars['ID']>;
};

export enum ResourceListUsedFor {
  Custom = 'Custom',
  Develop = 'Develop',
  Display = 'Display',
  Learn = 'Learn',
  Research = 'Research'
}

export type ResourceSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  forId?: InputMaybe<Scalars['ID']>;
  forType?: InputMaybe<ResourceFor>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ResourceSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type ResourceSearchResult = {
  __typename?: 'ResourceSearchResult';
  edges: Array<ResourceEdge>;
  pageInfo: PageInfo;
};

export enum ResourceSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  IndexAsc = 'IndexAsc',
  IndexDesc = 'IndexDesc',
  UsedForAsc = 'UsedForAsc',
  UsedForDesc = 'UsedForDesc'
}

export type ResourceTranslation = {
  __typename?: 'ResourceTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  title?: Maybe<Scalars['String']>;
};

export type ResourceTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  title?: InputMaybe<Scalars['String']>;
};

export type ResourceTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  title?: InputMaybe<Scalars['String']>;
};

export type ResourceUpdateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  link?: InputMaybe<Scalars['String']>;
  listId?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<ResourceTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ResourceTranslationUpdateInput>>;
  usedFor?: InputMaybe<ResourceUsedFor>;
};

export enum ResourceUsedFor {
  Community = 'Community',
  Context = 'Context',
  Developer = 'Developer',
  Donation = 'Donation',
  ExternalService = 'ExternalService',
  Feed = 'Feed',
  Install = 'Install',
  Learning = 'Learning',
  Notes = 'Notes',
  OfficialWebsite = 'OfficialWebsite',
  Proposal = 'Proposal',
  Related = 'Related',
  Researching = 'Researching',
  Scheduling = 'Scheduling',
  Social = 'Social',
  Tutorial = 'Tutorial'
}

export type Response = {
  __typename?: 'Response';
  code?: Maybe<Scalars['Int']>;
  message: Scalars['String'];
};

export type Role = {
  __typename?: 'Role';
  assignees?: Maybe<Array<UserRole>>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  organization: Organization;
  title: Scalars['String'];
  translations: Array<RoleTranslation>;
  updated_at: Scalars['Date'];
};

export type RoleCreateInput = {
  id: Scalars['ID'];
  title: Scalars['String'];
  translationsCreate?: InputMaybe<Array<RoleTranslationCreateInput>>;
};

export type RoleTranslation = {
  __typename?: 'RoleTranslation';
  description: Scalars['String'];
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type RoleTranslationCreateInput = {
  description: Scalars['String'];
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type RoleTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
};

export type RoleUpdateInput = {
  id: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<RoleTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<RoleTranslationUpdateInput>>;
};

export type Routine = {
  __typename?: 'Routine';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  complexity: Scalars['Int'];
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Routine>;
  id: Scalars['ID'];
  inputs: Array<InputItem>;
  isAutomatable?: Maybe<Scalars['Boolean']>;
  isComplete: Scalars['Boolean'];
  isDeleted: Scalars['Boolean'];
  isInternal?: Maybe<Scalars['Boolean']>;
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  nodeLinks: Array<NodeLink>;
  nodeLists: Array<NodeRoutineList>;
  nodes: Array<Node>;
  nodesCount?: Maybe<Scalars['Int']>;
  outputs: Array<OutputItem>;
  owner?: Maybe<Contributor>;
  parent?: Maybe<Routine>;
  permissionsRoutine: RoutinePermission;
  project?: Maybe<Project>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists: Array<ResourceList>;
  rootId: Scalars['ID'];
  runs: Array<RunRoutine>;
  score: Scalars['Int'];
  simplicity: Scalars['Int'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<RoutineTranslation>;
  updated_at: Scalars['Date'];
  versionLabel: Scalars['String'];
  views: Scalars['Int'];
};

export type RoutineCreateInput = {
  createdByOrganizationId?: InputMaybe<Scalars['ID']>;
  createdByUserId?: InputMaybe<Scalars['ID']>;
  id: Scalars['ID'];
  inputsCreate?: InputMaybe<Array<InputItemCreateInput>>;
  isAutomatable?: InputMaybe<Scalars['Boolean']>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isInternal?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  nodeLinksCreate?: InputMaybe<Array<NodeLinkCreateInput>>;
  nodesCreate?: InputMaybe<Array<NodeCreateInput>>;
  outputsCreate?: InputMaybe<Array<OutputItemCreateInput>>;
  parentId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<RoutineTranslationCreateInput>>;
  versionLabel?: InputMaybe<Scalars['String']>;
};

export type RoutineEdge = {
  __typename?: 'RoutineEdge';
  cursor: Scalars['String'];
  node: Routine;
};

export type RoutinePermission = {
  __typename?: 'RoutinePermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canFork: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canRun: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type RoutineSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  isInternal?: InputMaybe<Scalars['Boolean']>;
  isInternalExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  maxComplexity?: InputMaybe<Scalars['Int']>;
  maxSimplicity?: InputMaybe<Scalars['Int']>;
  maxTimesCompleted?: InputMaybe<Scalars['Int']>;
  minComplexity?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minSimplicity?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minTimesCompleted?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RoutineSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type RoutineSearchResult = {
  __typename?: 'RoutineSearchResult';
  edges: Array<RoutineEdge>;
  pageInfo: PageInfo;
};

export enum RoutineSortBy {
  DateCompletedAsc = 'DateCompletedAsc',
  DateCompletedDesc = 'DateCompletedDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  IssuesAsc = 'IssuesAsc',
  IssuesDesc = 'IssuesDesc',
  PullRequestsAsc = 'PullRequestsAsc',
  PullRequestsDesc = 'PullRequestsDesc',
  QuestionsAsc = 'QuestionsAsc',
  QuestionsDesc = 'QuestionsDesc',
  QuizzesAsc = 'QuizzesAsc',
  QuizzesDesc = 'QuizzesDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc',
  VotesAsc = 'VotesAsc',
  VotesDesc = 'VotesDesc'
}

export type RoutineTranslation = {
  __typename?: 'RoutineTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  instructions: Scalars['String'];
  language: Scalars['String'];
  title: Scalars['String'];
};

export type RoutineTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  instructions: Scalars['String'];
  language: Scalars['String'];
  title: Scalars['String'];
};

export type RoutineTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  instructions?: InputMaybe<Scalars['String']>;
  language?: InputMaybe<Scalars['String']>;
  title?: InputMaybe<Scalars['String']>;
};

export type RoutineUpdateInput = {
  inputsCreate?: InputMaybe<Array<InputItemCreateInput>>;
  inputsDelete?: InputMaybe<Array<Scalars['ID']>>;
  inputsUpdate?: InputMaybe<Array<InputItemUpdateInput>>;
  isAutomatable?: InputMaybe<Scalars['Boolean']>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isInternal?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  nodeLinksCreate?: InputMaybe<Array<NodeLinkCreateInput>>;
  nodeLinksDelete?: InputMaybe<Array<Scalars['ID']>>;
  nodeLinksUpdate?: InputMaybe<Array<NodeLinkUpdateInput>>;
  nodesCreate?: InputMaybe<Array<NodeCreateInput>>;
  nodesDelete?: InputMaybe<Array<Scalars['ID']>>;
  nodesUpdate?: InputMaybe<Array<NodeUpdateInput>>;
  organizationId?: InputMaybe<Scalars['ID']>;
  outputsCreate?: InputMaybe<Array<OutputItemCreateInput>>;
  outputsDelete?: InputMaybe<Array<Scalars['ID']>>;
  outputsUpdate?: InputMaybe<Array<OutputItemUpdateInput>>;
  projectId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  translationsCreate?: InputMaybe<Array<RoutineTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<RoutineTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
  versionId?: InputMaybe<Scalars['ID']>;
  versionLabel?: InputMaybe<Scalars['String']>;
};

export type RoutineVersion = {
  __typename?: 'RoutineVersion';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  complexity: Scalars['Int'];
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Routine>;
  id: Scalars['ID'];
  inputs: Array<InputItem>;
  isAutomatable?: Maybe<Scalars['Boolean']>;
  isComplete: Scalars['Boolean'];
  isDeleted: Scalars['Boolean'];
  isInternal?: Maybe<Scalars['Boolean']>;
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  nodeLinks: Array<NodeLink>;
  nodeLists: Array<NodeRoutineList>;
  nodes: Array<Node>;
  nodesCount?: Maybe<Scalars['Int']>;
  outputs: Array<OutputItem>;
  owner?: Maybe<Contributor>;
  parent?: Maybe<Routine>;
  permissionsRoutine: RoutinePermission;
  project?: Maybe<Project>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists: Array<ResourceList>;
  rootId: Scalars['ID'];
  runs: Array<RunRoutine>;
  score: Scalars['Int'];
  simplicity: Scalars['Int'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<RoutineTranslation>;
  updated_at: Scalars['Date'];
  versionLabel: Scalars['String'];
  views: Scalars['Int'];
};

export type RoutineVersionCreateInput = {
  createdByOrganizationId?: InputMaybe<Scalars['ID']>;
  createdByUserId?: InputMaybe<Scalars['ID']>;
  id: Scalars['ID'];
  inputsCreate?: InputMaybe<Array<InputItemCreateInput>>;
  isAutomatable?: InputMaybe<Scalars['Boolean']>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isInternal?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  nodeLinksCreate?: InputMaybe<Array<NodeLinkCreateInput>>;
  nodesCreate?: InputMaybe<Array<NodeCreateInput>>;
  outputsCreate?: InputMaybe<Array<OutputItemCreateInput>>;
  parentId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<RoutineTranslationCreateInput>>;
  versionLabel?: InputMaybe<Scalars['String']>;
};

export type RoutineVersionEdge = {
  __typename?: 'RoutineVersionEdge';
  cursor: Scalars['String'];
  node: RoutineVersion;
};

export type RoutineVersionPermission = {
  __typename?: 'RoutineVersionPermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canFork: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canRun: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type RoutineVersionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  isInternal?: InputMaybe<Scalars['Boolean']>;
  isInternalExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  maxComplexity?: InputMaybe<Scalars['Int']>;
  maxSimplicity?: InputMaybe<Scalars['Int']>;
  maxTimesCompleted?: InputMaybe<Scalars['Int']>;
  minComplexity?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minSimplicity?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minTimesCompleted?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RoutineSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type RoutineVersionSearchResult = {
  __typename?: 'RoutineVersionSearchResult';
  edges: Array<RoutineEdge>;
  pageInfo: PageInfo;
};

export enum RoutineVersionSortBy {
  CommentsAsc = 'CommentsAsc',
  CommentsDesc = 'CommentsDesc',
  ComplexityAsc = 'ComplexityAsc',
  ComplexityDesc = 'ComplexityDesc',
  DateCompletedAsc = 'DateCompletedAsc',
  DateCompletedDesc = 'DateCompletedDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  DirectoryListingsAsc = 'DirectoryListingsAsc',
  DirectoryListingsDesc = 'DirectoryListingsDesc',
  ForksAsc = 'ForksAsc',
  ForksDesc = 'ForksDesc',
  RunRoutinesAsc = 'RunRoutinesAsc',
  RunRoutinesDesc = 'RunRoutinesDesc',
  SimplicityAsc = 'SimplicityAsc',
  SimplicityDesc = 'SimplicityDesc'
}

export type RoutineVersionTranslation = {
  __typename?: 'RoutineVersionTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  instructions: Scalars['String'];
  language: Scalars['String'];
  title: Scalars['String'];
};

export type RoutineVersionTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  instructions: Scalars['String'];
  language: Scalars['String'];
  title: Scalars['String'];
};

export type RoutineVersionTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  instructions?: InputMaybe<Scalars['String']>;
  language?: InputMaybe<Scalars['String']>;
  title?: InputMaybe<Scalars['String']>;
};

export type RoutineVersionUpdateInput = {
  inputsCreate?: InputMaybe<Array<InputItemCreateInput>>;
  inputsDelete?: InputMaybe<Array<Scalars['ID']>>;
  inputsUpdate?: InputMaybe<Array<InputItemUpdateInput>>;
  isAutomatable?: InputMaybe<Scalars['Boolean']>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isInternal?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  nodeLinksCreate?: InputMaybe<Array<NodeLinkCreateInput>>;
  nodeLinksDelete?: InputMaybe<Array<Scalars['ID']>>;
  nodeLinksUpdate?: InputMaybe<Array<NodeLinkUpdateInput>>;
  nodesCreate?: InputMaybe<Array<NodeCreateInput>>;
  nodesDelete?: InputMaybe<Array<Scalars['ID']>>;
  nodesUpdate?: InputMaybe<Array<NodeUpdateInput>>;
  organizationId?: InputMaybe<Scalars['ID']>;
  outputsCreate?: InputMaybe<Array<OutputItemCreateInput>>;
  outputsDelete?: InputMaybe<Array<Scalars['ID']>>;
  outputsUpdate?: InputMaybe<Array<OutputItemUpdateInput>>;
  projectId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  translationsCreate?: InputMaybe<Array<RoutineTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<RoutineTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
  versionId?: InputMaybe<Scalars['ID']>;
  versionLabel?: InputMaybe<Scalars['String']>;
};

export type RunProject = {
  __typename?: 'RunProject';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type RunProjectCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootId: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type RunProjectEdge = {
  __typename?: 'RunProjectEdge';
  cursor: Scalars['String'];
  node: Api;
};

export type RunProjectPermission = {
  __typename?: 'RunProjectPermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type RunProjectSchedule = {
  __typename?: 'RunProjectSchedule';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type RunProjectScheduleCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootId: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type RunProjectScheduleEdge = {
  __typename?: 'RunProjectScheduleEdge';
  cursor: Scalars['String'];
  node: Api;
};

export type RunProjectSchedulePermission = {
  __typename?: 'RunProjectSchedulePermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type RunProjectScheduleSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type RunProjectScheduleSearchResult = {
  __typename?: 'RunProjectScheduleSearchResult';
  edges: Array<ApiEdge>;
  pageInfo: PageInfo;
};

export enum RunProjectScheduleSortBy {
  RecurrEndAsc = 'RecurrEndAsc',
  RecurrEndDesc = 'RecurrEndDesc',
  RecurrStartAsc = 'RecurrStartAsc',
  RecurrStartDesc = 'RecurrStartDesc',
  WindowEndAsc = 'WindowEndAsc',
  WindowEndDesc = 'WindowEndDesc',
  WindowStartAsc = 'WindowStartAsc',
  WindowStartDesc = 'WindowStartDesc'
}

export type RunProjectScheduleTranslation = {
  __typename?: 'RunProjectScheduleTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type RunProjectScheduleTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type RunProjectScheduleTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type RunProjectScheduleUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type RunProjectSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type RunProjectSearchResult = {
  __typename?: 'RunProjectSearchResult';
  edges: Array<ApiEdge>;
  pageInfo: PageInfo;
};

export enum RunProjectSortBy {
  ContextSwitchesAsc = 'ContextSwitchesAsc',
  ContextSwitchesDesc = 'ContextSwitchesDesc',
  DateCompletedAsc = 'DateCompletedAsc',
  DateCompletedDesc = 'DateCompletedDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateStartedAsc = 'DateStartedAsc',
  DateStartedDesc = 'DateStartedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  RunRoutinesAsc = 'RunRoutinesAsc',
  RunRoutinesDesc = 'RunRoutinesDesc',
  StepsAsc = 'StepsAsc',
  StepsDesc = 'StepsDesc'
}

export type RunProjectStep = {
  __typename?: 'RunProjectStep';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type RunProjectStepCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootId: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type RunProjectStepEdge = {
  __typename?: 'RunProjectStepEdge';
  cursor: Scalars['String'];
  node: Api;
};

export type RunProjectStepPermission = {
  __typename?: 'RunProjectStepPermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type RunProjectStepSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type RunProjectStepSearchResult = {
  __typename?: 'RunProjectStepSearchResult';
  edges: Array<ApiEdge>;
  pageInfo: PageInfo;
};

export enum RunProjectStepSortBy {
  ContextSwitchesAsc = 'ContextSwitchesAsc',
  ContextSwitchesDesc = 'ContextSwitchesDesc',
  OrderAsc = 'OrderAsc',
  OrderDesc = 'OrderDesc',
  TimeCompletedAsc = 'TimeCompletedAsc',
  TimeCompletedDesc = 'TimeCompletedDesc',
  TimeElapsedAsc = 'TimeElapsedAsc',
  TimeElapsedDesc = 'TimeElapsedDesc',
  TimeStartedAsc = 'TimeStartedAsc',
  TimeStartedDesc = 'TimeStartedDesc'
}

export type RunProjectStepTranslation = {
  __typename?: 'RunProjectStepTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type RunProjectStepTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type RunProjectStepTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type RunProjectStepUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type RunProjectTranslation = {
  __typename?: 'RunProjectTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type RunProjectTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type RunProjectTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type RunProjectUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type RunRoutine = {
  __typename?: 'RunRoutine';
  completedComplexity: Scalars['Int'];
  contextSwitches: Scalars['Int'];
  id: Scalars['ID'];
  inputs: Array<RunRoutineInput>;
  isPrivate: Scalars['Boolean'];
  permissionsRun: RunRoutinePermission;
  routine?: Maybe<Routine>;
  status: RunStatus;
  steps: Array<RunRoutineStep>;
  timeCompleted?: Maybe<Scalars['Date']>;
  timeElapsed?: Maybe<Scalars['Int']>;
  timeStarted?: Maybe<Scalars['Date']>;
  title: Scalars['String'];
  user: User;
};

export type RunRoutineCancelInput = {
  id: Scalars['ID'];
};

export type RunRoutineCompleteInput = {
  completedComplexity?: InputMaybe<Scalars['Int']>;
  exists?: InputMaybe<Scalars['Boolean']>;
  finalStepCreate?: InputMaybe<RunRoutineStepCreateInput>;
  finalStepUpdate?: InputMaybe<RunRoutineStepUpdateInput>;
  id: Scalars['ID'];
  inputsCreate?: InputMaybe<Array<RunRoutineInputCreateInput>>;
  inputsDelete?: InputMaybe<Array<Scalars['ID']>>;
  inputsUpdate?: InputMaybe<Array<RunRoutineInputUpdateInput>>;
  title: Scalars['String'];
  wasSuccessful?: InputMaybe<Scalars['Boolean']>;
};

export type RunRoutineCreateInput = {
  id: Scalars['ID'];
  inputsCreate?: InputMaybe<Array<RunRoutineInputCreateInput>>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  routineVersionId: Scalars['ID'];
  stepsCreate?: InputMaybe<Array<RunRoutineStepCreateInput>>;
  title: Scalars['String'];
};

export type RunRoutineEdge = {
  __typename?: 'RunRoutineEdge';
  cursor: Scalars['String'];
  node: RunRoutine;
};

export type RunRoutineInput = {
  __typename?: 'RunRoutineInput';
  data: Scalars['String'];
  id: Scalars['ID'];
  input: InputItem;
};

export type RunRoutineInputCreateInput = {
  data: Scalars['String'];
  id: Scalars['ID'];
  inputId: Scalars['ID'];
};

export type RunRoutineInputEdge = {
  __typename?: 'RunRoutineInputEdge';
  cursor: Scalars['String'];
  node: RunRoutineInput;
};

export type RunRoutineInputSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  routineIds?: InputMaybe<Array<Scalars['ID']>>;
  standardIds?: InputMaybe<Array<Scalars['ID']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type RunRoutineInputSearchResult = {
  __typename?: 'RunRoutineInputSearchResult';
  edges: Array<RunRoutineInputEdge>;
  pageInfo: PageInfo;
};

export enum RunRoutineInputSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type RunRoutineInputUpdateInput = {
  data: Scalars['String'];
  id: Scalars['ID'];
};

export type RunRoutinePermission = {
  __typename?: 'RunRoutinePermission';
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canView: Scalars['Boolean'];
};

export type RunRoutineSchedule = {
  __typename?: 'RunRoutineSchedule';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type RunRoutineScheduleCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootId: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type RunRoutineScheduleEdge = {
  __typename?: 'RunRoutineScheduleEdge';
  cursor: Scalars['String'];
  node: Api;
};

export type RunRoutineSchedulePermission = {
  __typename?: 'RunRoutineSchedulePermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type RunRoutineScheduleSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type RunRoutineScheduleSearchResult = {
  __typename?: 'RunRoutineScheduleSearchResult';
  edges: Array<ApiEdge>;
  pageInfo: PageInfo;
};

export enum RunRoutineScheduleSortBy {
  RecurrEndAsc = 'RecurrEndAsc',
  RecurrEndDesc = 'RecurrEndDesc',
  RecurrStartAsc = 'RecurrStartAsc',
  RecurrStartDesc = 'RecurrStartDesc',
  WindowEndAsc = 'WindowEndAsc',
  WindowEndDesc = 'WindowEndDesc',
  WindowStartAsc = 'WindowStartAsc',
  WindowStartDesc = 'WindowStartDesc'
}

export type RunRoutineScheduleTranslation = {
  __typename?: 'RunRoutineScheduleTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type RunRoutineScheduleTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type RunRoutineScheduleTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type RunRoutineScheduleUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type RunRoutineSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  completedTimeFrame?: InputMaybe<TimeFrame>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RunRoutineSortBy>;
  startedTimeFrame?: InputMaybe<TimeFrame>;
  status?: InputMaybe<RunStatus>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type RunRoutineSearchResult = {
  __typename?: 'RunRoutineSearchResult';
  edges: Array<RunRoutineEdge>;
  pageInfo: PageInfo;
};

export enum RunRoutineSortBy {
  ContextSwitchesAsc = 'ContextSwitchesAsc',
  ContextSwitchesDesc = 'ContextSwitchesDesc',
  DateCompletedAsc = 'DateCompletedAsc',
  DateCompletedDesc = 'DateCompletedDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateStartedAsc = 'DateStartedAsc',
  DateStartedDesc = 'DateStartedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  RunRoutinesAsc = 'RunRoutinesAsc',
  RunRoutinesDesc = 'RunRoutinesDesc',
  StepsAsc = 'StepsAsc',
  StepsDesc = 'StepsDesc'
}

export type RunRoutineStep = {
  __typename?: 'RunRoutineStep';
  contextSwitches: Scalars['Int'];
  id: Scalars['ID'];
  node?: Maybe<Node>;
  order: Scalars['Int'];
  run: RunRoutine;
  status: RunRoutineStepStatus;
  step: Array<Scalars['Int']>;
  subroutine?: Maybe<Routine>;
  timeCompleted?: Maybe<Scalars['Date']>;
  timeElapsed?: Maybe<Scalars['Int']>;
  timeStarted?: Maybe<Scalars['Date']>;
  title: Scalars['String'];
};

export type RunRoutineStepCreateInput = {
  contextSwitches?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  nodeId?: InputMaybe<Scalars['ID']>;
  order: Scalars['Int'];
  step: Array<Scalars['Int']>;
  subroutineVersionId?: InputMaybe<Scalars['ID']>;
  timeElapsed?: InputMaybe<Scalars['Int']>;
  title: Scalars['String'];
};

export enum RunRoutineStepSortBy {
  ContextSwitchesAsc = 'ContextSwitchesAsc',
  ContextSwitchesDesc = 'ContextSwitchesDesc',
  OrderAsc = 'OrderAsc',
  OrderDesc = 'OrderDesc',
  TimeCompletedAsc = 'TimeCompletedAsc',
  TimeCompletedDesc = 'TimeCompletedDesc',
  TimeElapsedAsc = 'TimeElapsedAsc',
  TimeElapsedDesc = 'TimeElapsedDesc',
  TimeStartedAsc = 'TimeStartedAsc',
  TimeStartedDesc = 'TimeStartedDesc'
}

export enum RunRoutineStepStatus {
  Completed = 'Completed',
  InProgress = 'InProgress',
  Skipped = 'Skipped'
}

export type RunRoutineStepUpdateInput = {
  contextSwitches?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  status?: InputMaybe<RunRoutineStepStatus>;
  timeElapsed?: InputMaybe<Scalars['Int']>;
};

export type RunRoutineUpdateInput = {
  completedComplexity?: InputMaybe<Scalars['Int']>;
  contextSwitches?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  inputsCreate?: InputMaybe<Array<RunRoutineInputCreateInput>>;
  inputsDelete?: InputMaybe<Array<Scalars['ID']>>;
  inputsUpdate?: InputMaybe<Array<RunRoutineInputUpdateInput>>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  stepsCreate?: InputMaybe<Array<RunRoutineStepCreateInput>>;
  stepsDelete?: InputMaybe<Array<Scalars['ID']>>;
  stepsUpdate?: InputMaybe<Array<RunRoutineStepUpdateInput>>;
  timeElapsed?: InputMaybe<Scalars['Int']>;
};

export enum RunStatus {
  Cancelled = 'Cancelled',
  Completed = 'Completed',
  Failed = 'Failed',
  InProgress = 'InProgress',
  Scheduled = 'Scheduled'
}

export type SearchException = {
  field: Scalars['String'];
  value: Scalars['String'];
};

export type SendVerificationEmailInput = {
  emailAddress: Scalars['String'];
};

export type SendVerificationTextInput = {
  phoneNumber: Scalars['String'];
};

export type Session = {
  __typename?: 'Session';
  isLoggedIn: Scalars['Boolean'];
  timeZone?: Maybe<Scalars['String']>;
  users?: Maybe<Array<SessionUser>>;
};

export type SessionUser = {
  __typename?: 'SessionUser';
  handle?: Maybe<Scalars['String']>;
  hasPremium: Scalars['Boolean'];
  id: Scalars['String'];
  languages: Array<Scalars['String']>;
  name?: Maybe<Scalars['String']>;
  theme?: Maybe<Scalars['String']>;
};

export type SmartContract = {
  __typename?: 'SmartContract';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type SmartContractCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootId: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type SmartContractEdge = {
  __typename?: 'SmartContractEdge';
  cursor: Scalars['String'];
  node: SmartContract;
};

export type SmartContractPermission = {
  __typename?: 'SmartContractPermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type SmartContractSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type SmartContractSearchResult = {
  __typename?: 'SmartContractSearchResult';
  edges: Array<SmartContractEdge>;
  pageInfo: PageInfo;
};

export enum SmartContractSortBy {
  DateCompletedAsc = 'DateCompletedAsc',
  DateCompletedDesc = 'DateCompletedDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  IssuesAsc = 'IssuesAsc',
  IssuesDesc = 'IssuesDesc',
  PullRequestsAsc = 'PullRequestsAsc',
  PullRequestsDesc = 'PullRequestsDesc',
  QuestionsAsc = 'QuestionsAsc',
  QuestionsDesc = 'QuestionsDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc',
  VotesAsc = 'VotesAsc',
  VotesDesc = 'VotesDesc'
}

export type SmartContractTranslation = {
  __typename?: 'SmartContractTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type SmartContractTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type SmartContractTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type SmartContractUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type SmartContractVersion = {
  __typename?: 'SmartContractVersion';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type SmartContractVersionCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootId: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type SmartContractVersionEdge = {
  __typename?: 'SmartContractVersionEdge';
  cursor: Scalars['String'];
  node: SmartContract;
};

export type SmartContractVersionPermission = {
  __typename?: 'SmartContractVersionPermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type SmartContractVersionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type SmartContractVersionSearchResult = {
  __typename?: 'SmartContractVersionSearchResult';
  edges: Array<SmartContractEdge>;
  pageInfo: PageInfo;
};

export enum SmartContractVersionSortBy {
  CommentsAsc = 'CommentsAsc',
  CommentsDesc = 'CommentsDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  DirectoryListingsAsc = 'DirectoryListingsAsc',
  DirectoryListingsDesc = 'DirectoryListingsDesc',
  ForksAsc = 'ForksAsc',
  ForksDesc = 'ForksDesc',
  ReportsAsc = 'ReportsAsc',
  ReportsDesc = 'ReportsDesc'
}

export type SmartContractVersionTranslation = {
  __typename?: 'SmartContractVersionTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type SmartContractVersionTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type SmartContractVersionTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type SmartContractVersionUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type Standard = {
  __typename?: 'Standard';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  default?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isDeleted: Scalars['Boolean'];
  isInternal: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  name: Scalars['String'];
  permissionsStandard: StandardPermission;
  props: Scalars['String'];
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists: Array<ResourceList>;
  rootId: Scalars['ID'];
  routineInputs: Array<Routine>;
  routineOutputs: Array<Routine>;
  score: Scalars['Int'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<StandardTranslation>;
  type: Scalars['String'];
  updated_at: Scalars['Date'];
  versionLabel: Scalars['String'];
  views: Scalars['Int'];
  yup?: Maybe<Scalars['String']>;
};

export type StandardCreateInput = {
  createdByOrganizationId?: InputMaybe<Scalars['ID']>;
  createdByUserId?: InputMaybe<Scalars['ID']>;
  default?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isInternal?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  props: Scalars['String'];
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<StandardTranslationCreateInput>>;
  type: Scalars['String'];
  versionLabel?: InputMaybe<Scalars['String']>;
  yup?: InputMaybe<Scalars['String']>;
};

export type StandardEdge = {
  __typename?: 'StandardEdge';
  cursor: Scalars['String'];
  node: Standard;
};

export type StandardPermission = {
  __typename?: 'StandardPermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type StandardSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StandardSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  type?: InputMaybe<Scalars['String']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type StandardSearchResult = {
  __typename?: 'StandardSearchResult';
  edges: Array<StandardEdge>;
  pageInfo: PageInfo;
};

export enum StandardSortBy {
  DateCompletedAsc = 'DateCompletedAsc',
  DateCompletedDesc = 'DateCompletedDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  IssuesAsc = 'IssuesAsc',
  IssuesDesc = 'IssuesDesc',
  PullRequestsAsc = 'PullRequestsAsc',
  PullRequestsDesc = 'PullRequestsDesc',
  QuestionsAsc = 'QuestionsAsc',
  QuestionsDesc = 'QuestionsDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc',
  VotesAsc = 'VotesAsc',
  VotesDesc = 'VotesDesc'
}

export type StandardTranslation = {
  __typename?: 'StandardTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  jsonVariable?: Maybe<Scalars['String']>;
  language: Scalars['String'];
};

export type StandardTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  jsonVariable?: InputMaybe<Scalars['String']>;
  language: Scalars['String'];
};

export type StandardTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  jsonVariable?: InputMaybe<Scalars['String']>;
  language?: InputMaybe<Scalars['String']>;
};

export type StandardUpdateInput = {
  default?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  makeAnonymous?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  props: Scalars['String'];
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<StandardTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<StandardTranslationUpdateInput>>;
  type: Scalars['String'];
  userId?: InputMaybe<Scalars['ID']>;
  versionId?: InputMaybe<Scalars['ID']>;
  versionLabel?: InputMaybe<Scalars['String']>;
  yup?: InputMaybe<Scalars['String']>;
};

export type StandardVersion = {
  __typename?: 'StandardVersion';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  default?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isDeleted: Scalars['Boolean'];
  isInternal: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  name: Scalars['String'];
  permissionsStandard: StandardPermission;
  props: Scalars['String'];
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists: Array<ResourceList>;
  rootId: Scalars['ID'];
  routineInputs: Array<Routine>;
  routineOutputs: Array<Routine>;
  score: Scalars['Int'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<StandardTranslation>;
  type: Scalars['String'];
  updated_at: Scalars['Date'];
  versionLabel: Scalars['String'];
  views: Scalars['Int'];
  yup?: Maybe<Scalars['String']>;
};

export type StandardVersionCreateInput = {
  createdByOrganizationId?: InputMaybe<Scalars['ID']>;
  createdByUserId?: InputMaybe<Scalars['ID']>;
  default?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isInternal?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  props: Scalars['String'];
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<StandardTranslationCreateInput>>;
  type: Scalars['String'];
  versionLabel?: InputMaybe<Scalars['String']>;
  yup?: InputMaybe<Scalars['String']>;
};

export type StandardVersionEdge = {
  __typename?: 'StandardVersionEdge';
  cursor: Scalars['String'];
  node: StandardVersion;
};

export type StandardVersionPermission = {
  __typename?: 'StandardVersionPermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type StandardVersionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StandardVersionSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  type?: InputMaybe<Scalars['String']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type StandardVersionSearchResult = {
  __typename?: 'StandardVersionSearchResult';
  edges: Array<StandardEdge>;
  pageInfo: PageInfo;
};

export enum StandardVersionSortBy {
  CommentsAsc = 'CommentsAsc',
  CommentsDesc = 'CommentsDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  DirectoryListingsAsc = 'DirectoryListingsAsc',
  DirectoryListingsDesc = 'DirectoryListingsDesc',
  ForksAsc = 'ForksAsc',
  ForksDesc = 'ForksDesc',
  ReportsAsc = 'ReportsAsc',
  ReportsDesc = 'ReportsDesc'
}

export type StandardVersionTranslation = {
  __typename?: 'StandardVersionTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  jsonVariable?: Maybe<Scalars['String']>;
  language: Scalars['String'];
};

export type StandardVersionTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  jsonVariable?: InputMaybe<Scalars['String']>;
  language: Scalars['String'];
};

export type StandardVersionTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  jsonVariable?: InputMaybe<Scalars['String']>;
  language?: InputMaybe<Scalars['String']>;
};

export type StandardVersionUpdateInput = {
  default?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  makeAnonymous?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  props: Scalars['String'];
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<StandardTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<StandardTranslationUpdateInput>>;
  type: Scalars['String'];
  userId?: InputMaybe<Scalars['ID']>;
  versionId?: InputMaybe<Scalars['ID']>;
  versionLabel?: InputMaybe<Scalars['String']>;
  yup?: InputMaybe<Scalars['String']>;
};

export type Star = {
  __typename?: 'Star';
  from: User;
  id: Scalars['ID'];
  to: StarTo;
};

export type StarEdge = {
  __typename?: 'StarEdge';
  cursor: Scalars['String'];
  node: Star;
};

export enum StarFor {
  Api = 'Api',
  Comment = 'Comment',
  Issue = 'Issue',
  Note = 'Note',
  Organization = 'Organization',
  Post = 'Post',
  Project = 'Project',
  Question = 'Question',
  QuestionAnswer = 'QuestionAnswer',
  Quiz = 'Quiz',
  Routine = 'Routine',
  SmartContract = 'SmartContract',
  Standard = 'Standard',
  Tag = 'Tag',
  User = 'User'
}

export type StarInput = {
  forId: Scalars['ID'];
  isStar: Scalars['Boolean'];
  starFor: StarFor;
};

export type StarSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  excludeTags?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StarSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type StarSearchResult = {
  __typename?: 'StarSearchResult';
  edges: Array<StarEdge>;
  pageInfo: PageInfo;
};

export enum StarSortBy {
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type StarTo = Api | Comment | Issue | Note | Organization | Post | Project | Question | QuestionAnswer | Quiz | Routine | SmartContract | Standard | Tag | User;

export type StatisticsTimeFrame = {
  __typename?: 'StatisticsTimeFrame';
  organizations: Array<Scalars['Int']>;
  projects: Array<Scalars['Int']>;
  routines: Array<Scalars['Int']>;
  standards: Array<Scalars['Int']>;
  users: Array<Scalars['Int']>;
};

export type StatsApi = {
  __typename?: 'StatsApi';
  id: Scalars['ID'];
};

export type StatsOrganization = {
  __typename?: 'StatsOrganization';
  id: Scalars['ID'];
};

export type StatsProject = {
  __typename?: 'StatsProject';
  id: Scalars['ID'];
};

export type StatsQuiz = {
  __typename?: 'StatsQuiz';
  id: Scalars['ID'];
};

export type StatsRoutine = {
  __typename?: 'StatsRoutine';
  id: Scalars['ID'];
};

export type StatsSiteSearchInput = {
  searchString: Scalars['String'];
  take?: InputMaybe<Scalars['Int']>;
};

export type StatsSiteSearchResult = {
  __typename?: 'StatsSiteSearchResult';
  allTime: StatisticsTimeFrame;
  daily: StatisticsTimeFrame;
  monthly: StatisticsTimeFrame;
  weekly: StatisticsTimeFrame;
  yearly: StatisticsTimeFrame;
};

export type StatsSmartContract = {
  __typename?: 'StatsSmartContract';
  id: Scalars['ID'];
};

export type StatsStandard = {
  __typename?: 'StatsStandard';
  id: Scalars['ID'];
};

export type StatsUser = {
  __typename?: 'StatsUser';
  id: Scalars['ID'];
};

export type Success = {
  __typename?: 'Success';
  success: Scalars['Boolean'];
};

export type SwitchCurrentAccountInput = {
  id: Scalars['ID'];
};

export type Tag = {
  __typename?: 'Tag';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isOwn: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  tag: Scalars['String'];
  translations: Array<TagTranslation>;
  updated_at: Scalars['Date'];
};

export type TagCreateInput = {
  anonymous?: InputMaybe<Scalars['Boolean']>;
  tag: Scalars['String'];
  translationsCreate?: InputMaybe<Array<TagTranslationCreateInput>>;
};

export type TagEdge = {
  __typename?: 'TagEdge';
  cursor: Scalars['String'];
  node: Tag;
};

export type TagHidden = {
  __typename?: 'TagHidden';
  id: Scalars['ID'];
  isBlur: Scalars['Boolean'];
  tag: Tag;
};

export type TagHiddenCreateInput = {
  id: Scalars['ID'];
  isBlur?: InputMaybe<Scalars['Boolean']>;
  tagConnect?: InputMaybe<Scalars['ID']>;
  tagCreate?: InputMaybe<TagCreateInput>;
};

export type TagHiddenUpdateInput = {
  id: Scalars['ID'];
  isBlur?: InputMaybe<Scalars['Boolean']>;
};

export type TagSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  hidden?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minStars?: InputMaybe<Scalars['Int']>;
  myTags?: InputMaybe<Scalars['Boolean']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<TagSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type TagSearchResult = {
  __typename?: 'TagSearchResult';
  edges: Array<TagEdge>;
  pageInfo: PageInfo;
};

export enum TagSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc'
}

export type TagTranslation = {
  __typename?: 'TagTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type TagTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type TagTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
};

export type TagUpdateInput = {
  anonymous?: InputMaybe<Scalars['Boolean']>;
  tag: Scalars['String'];
  translationsCreate?: InputMaybe<Array<TagTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<TagTranslationUpdateInput>>;
};

export type TimeFrame = {
  after?: InputMaybe<Scalars['Date']>;
  before?: InputMaybe<Scalars['Date']>;
};

export type Transfer = {
  __typename?: 'Transfer';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type TransferCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootId: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type TransferEdge = {
  __typename?: 'TransferEdge';
  cursor: Scalars['String'];
  node: Api;
};

export type TransferPermission = {
  __typename?: 'TransferPermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type TransferSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type TransferSearchResult = {
  __typename?: 'TransferSearchResult';
  edges: Array<ApiEdge>;
  pageInfo: PageInfo;
};

export enum TransferSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type TransferTranslation = {
  __typename?: 'TransferTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type TransferTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type TransferTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type TransferUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type Translate = {
  __typename?: 'Translate';
  fields: Scalars['String'];
  language: Scalars['String'];
};

export type TranslateInput = {
  fields: Scalars['String'];
  languageSource: Scalars['String'];
  languageTarget: Scalars['String'];
};

export type User = {
  __typename?: 'User';
  apiKeys?: Maybe<Array<ApiKey>>;
  apis: Array<Api>;
  apisCreated?: Maybe<Array<Api>>;
  awards?: Maybe<Array<Award>>;
  comments?: Maybe<Array<Comment>>;
  created_at: Scalars['Date'];
  emails?: Maybe<Array<Email>>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  invitedByUser?: Maybe<User>;
  invitedUsers?: Maybe<Array<User>>;
  isPrivate: Scalars['Boolean'];
  isPrivateApis: Scalars['Boolean'];
  isPrivateApisCreated: Scalars['Boolean'];
  isPrivateMemberships: Scalars['Boolean'];
  isPrivateOrganizationsCreated: Scalars['Boolean'];
  isPrivateProjects: Scalars['Boolean'];
  isPrivateProjectsCreated: Scalars['Boolean'];
  isPrivatePullRequests: Scalars['Boolean'];
  isPrivateQuestionsAnswered: Scalars['Boolean'];
  isPrivateQuestionsAsked: Scalars['Boolean'];
  isPrivateQuizzesCreated: Scalars['Boolean'];
  isPrivateRoles: Scalars['Boolean'];
  isPrivateRoutines: Scalars['Boolean'];
  isPrivateRoutinesCreated: Scalars['Boolean'];
  isPrivateSmartContracts: Scalars['Boolean'];
  isPrivateStandards: Scalars['Boolean'];
  isPrivateStandardsCreated: Scalars['Boolean'];
  isPrivateStars: Scalars['Boolean'];
  isPrivateVotes: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isViewed: Scalars['Boolean'];
  issuesClosed?: Maybe<Array<Issue>>;
  issuesCreated?: Maybe<Array<Issue>>;
  labels?: Maybe<Array<Label>>;
  languages?: Maybe<Array<Scalars['String']>>;
  meetingsAttending?: Maybe<Array<Meeting>>;
  meetingsInvited?: Maybe<Array<MeetingInvite>>;
  memberships?: Maybe<Array<Member>>;
  name: Scalars['String'];
  notes?: Maybe<Array<Note>>;
  notesCreated?: Maybe<Array<Note>>;
  notificationSubscriptions?: Maybe<Array<NotificationSubscription>>;
  notifications?: Maybe<Array<Notification>>;
  organizationsCreate?: Maybe<Array<Organization>>;
  paymentHistory?: Maybe<Array<Payment>>;
  premium?: Maybe<Premium>;
  projects?: Maybe<Array<Project>>;
  projectsCreated?: Maybe<Array<Project>>;
  pullRequests?: Maybe<Array<PullRequest>>;
  pushDevices?: Maybe<Array<PushDevice>>;
  questionsAnswered?: Maybe<Array<QuestionAnswer>>;
  questionsAsked?: Maybe<Array<Question>>;
  quizzesCreated?: Maybe<Array<Quiz>>;
  quizzesTaken?: Maybe<Array<Quiz>>;
  reportResponses?: Maybe<Array<ReportResponse>>;
  reportsCount: Scalars['Int'];
  reportsCreated?: Maybe<Array<Report>>;
  reportsReceived: Array<Report>;
  reputationHistory?: Maybe<Array<ReputationHistory>>;
  roles?: Maybe<Array<Role>>;
  routines?: Maybe<Array<Routine>>;
  routinesCreated?: Maybe<Array<Routine>>;
  runProjects?: Maybe<Array<RunProject>>;
  runRoutines?: Maybe<Array<RunRoutine>>;
  schedules?: Maybe<Array<UserSchedule>>;
  sentReports?: Maybe<Array<Report>>;
  smartContracts?: Maybe<Array<SmartContract>>;
  smartContractsCreated?: Maybe<Array<SmartContract>>;
  standards?: Maybe<Array<Standard>>;
  standardsCreated?: Maybe<Array<Standard>>;
  starred?: Maybe<Array<Star>>;
  starredBy: Array<User>;
  stars: Scalars['Int'];
  stats?: Maybe<StatsUser>;
  status?: Maybe<AccountStatus>;
  tags?: Maybe<Array<Tag>>;
  theme?: Maybe<Scalars['String']>;
  transfersIncoming?: Maybe<Array<Transfer>>;
  transfersOutgoing?: Maybe<Array<Transfer>>;
  translations: Array<UserTranslation>;
  updated_at?: Maybe<Scalars['Date']>;
  viewed?: Maybe<Array<View>>;
  viewedBy?: Maybe<Array<View>>;
  views: Scalars['Int'];
  voted?: Maybe<Array<Vote>>;
  wallets?: Maybe<Array<Wallet>>;
};

export type UserDeleteInput = {
  deletePublicData: Scalars['Boolean'];
  password: Scalars['String'];
};

export type UserEdge = {
  __typename?: 'UserEdge';
  cursor: Scalars['String'];
  node: User;
};

export type UserRole = {
  __typename?: 'UserRole';
  role: Role;
  user: User;
};

export type UserSchedule = {
  __typename?: 'UserSchedule';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  permissionsProject: ProjectPermission;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<ProjectTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type UserScheduleCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootId: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type UserScheduleEdge = {
  __typename?: 'UserScheduleEdge';
  cursor: Scalars['String'];
  node: Api;
};

export type UserSchedulePermission = {
  __typename?: 'UserSchedulePermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type UserScheduleSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<SearchException>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type UserScheduleSearchResult = {
  __typename?: 'UserScheduleSearchResult';
  edges: Array<ApiEdge>;
  pageInfo: PageInfo;
};

export enum UserScheduleSortBy {
  EventEndAsc = 'EventEndAsc',
  EventEndDesc = 'EventEndDesc',
  EventStartAsc = 'EventStartAsc',
  EventStartDesc = 'EventStartDesc',
  RecurrEndAsc = 'RecurrEndAsc',
  RecurrEndDesc = 'RecurrEndDesc',
  RecurrStartAsc = 'RecurrStartAsc',
  RecurrStartDesc = 'RecurrStartDesc',
  TitleAsc = 'TitleAsc',
  TitleDesc = 'TitleDesc'
}

export type UserScheduleTranslation = {
  __typename?: 'UserScheduleTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type UserScheduleTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type UserScheduleTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type UserScheduleUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type UserSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceLists?: InputMaybe<Array<Scalars['String']>>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<UserSortBy>;
  standardId?: InputMaybe<Scalars['ID']>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type UserSearchResult = {
  __typename?: 'UserSearchResult';
  edges: Array<UserEdge>;
  pageInfo: PageInfo;
};

export enum UserSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc'
}

export type UserTranslation = {
  __typename?: 'UserTranslation';
  bio?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type UserTranslationCreateInput = {
  bio?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type UserTranslationUpdateInput = {
  bio?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
};

export type ValidateSessionInput = {
  timeZone: Scalars['String'];
};

export type View = {
  __typename?: 'View';
  from: User;
  id: Scalars['ID'];
  lastViewed: Scalars['Date'];
  title: Scalars['String'];
  to: ProjectOrOrganizationOrRoutineOrStandardOrUser;
};

export type ViewEdge = {
  __typename?: 'ViewEdge';
  cursor: Scalars['String'];
  node: View;
};

export type ViewSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  lastViewedTimeFrame?: InputMaybe<TimeFrame>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ViewSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type ViewSearchResult = {
  __typename?: 'ViewSearchResult';
  edges: Array<ViewEdge>;
  pageInfo: PageInfo;
};

export enum ViewSortBy {
  LastViewedAsc = 'LastViewedAsc',
  LastViewedDesc = 'LastViewedDesc'
}

export enum VisibilityType {
  All = 'All',
  Private = 'Private',
  Public = 'Public'
}

export type Vote = {
  __typename?: 'Vote';
  from: User;
  isUpvote?: Maybe<Scalars['Boolean']>;
  to: VoteTo;
};

export enum VoteFor {
  Api = 'Api',
  Comment = 'Comment',
  Issue = 'Issue',
  Note = 'Note',
  Post = 'Post',
  Project = 'Project',
  Question = 'Question',
  QuestionAnswer = 'QuestionAnswer',
  Quiz = 'Quiz',
  Routine = 'Routine',
  SmartContract = 'SmartContract',
  Standard = 'Standard'
}

export type VoteInput = {
  forId: Scalars['ID'];
  isUpvote?: InputMaybe<Scalars['Boolean']>;
  voteFor: VoteFor;
};

export type VoteTo = Api | Comment | Issue | Note | Post | Project | Question | QuestionAnswer | Quiz | Routine | SmartContract | Standard;

export type Wallet = {
  __typename?: 'Wallet';
  handles: Array<Handle>;
  id: Scalars['ID'];
  name?: Maybe<Scalars['String']>;
  organization?: Maybe<Organization>;
  publicAddress?: Maybe<Scalars['String']>;
  stakingAddress: Scalars['String'];
  user?: Maybe<User>;
  verified: Scalars['Boolean'];
};

export type WalletComplete = {
  __typename?: 'WalletComplete';
  firstLogIn: Scalars['Boolean'];
  session?: Maybe<Session>;
  wallet?: Maybe<Wallet>;
};

export type WalletCompleteInput = {
  signedPayload: Scalars['String'];
  stakingAddress: Scalars['String'];
};

export type WalletInitInput = {
  nonceDescription?: InputMaybe<Scalars['String']>;
  stakingAddress: Scalars['String'];
};

export type WalletUpdateInput = {
  id: Scalars['ID'];
  name?: InputMaybe<Scalars['String']>;
};

export type WriteAssetsInput = {
  files: Array<Scalars['Upload']>;
};
