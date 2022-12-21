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
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  issues: Array<Issue>;
  labels: Array<Label>;
  owner?: Maybe<Owner>;
  parent?: Maybe<Api>;
  permissionsRoot: RootPermission;
  pullRequests: Array<PullRequest>;
  pullRequestsCount: Scalars['Int'];
  questions: Array<Question>;
  questionsCount: Scalars['Int'];
  score: Scalars['Int'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  stats: Array<StatsApi>;
  tags: Array<Tag>;
  transfers: Array<Transfer>;
  transfersCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  versions: Array<ApiVersion>;
  versionsCount: Scalars['Int'];
  views: Scalars['Int'];
};

export type ApiCreateInput = {
  id: Scalars['ID'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  organizationConnect?: InputMaybe<Scalars['ID']>;
  parentConnect?: InputMaybe<Scalars['ID']>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  userConnect?: InputMaybe<Scalars['ID']>;
  versionsCreate?: InputMaybe<Array<ApiVersionCreateInput>>;
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

export type ApiSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  maxScore?: InputMaybe<Scalars['Int']>;
  maxStars?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  ownedByOrganizationId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ApiSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ApiSearchResult = {
  __typename?: 'ApiSearchResult';
  edges: Array<ApiEdge>;
  pageInfo: PageInfo;
};

export enum ApiSortBy {
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
  ScoreAsc = 'ScoreAsc',
  ScoreDesc = 'ScoreDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc'
}

export type ApiUpdateInput = {
  id: Scalars['ID'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  labelsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  organizationConnect?: InputMaybe<Scalars['ID']>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  userConnect?: InputMaybe<Scalars['ID']>;
  versionsCreate?: InputMaybe<Array<ApiVersionCreateInput>>;
  versionsDelete?: InputMaybe<Array<Scalars['ID']>>;
  versionsUpdate?: InputMaybe<Array<ApiVersionUpdateInput>>;
};

export type ApiVersion = {
  __typename?: 'ApiVersion';
  callLink: Scalars['String'];
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  created_at: Scalars['Date'];
  directoryListings: Array<ProjectVersionDirectory>;
  directoryListingsCount: Scalars['Int'];
  documentationLink?: Maybe<Scalars['String']>;
  forks: Array<Api>;
  forksCount: Scalars['Int'];
  id: Scalars['ID'];
  isLatest: Scalars['Boolean'];
  permissionsVersion: VersionPermission;
  pullRequest?: Maybe<PullRequest>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceList?: Maybe<ResourceList>;
  root: Api;
  translations: Array<ApiVersionTranslation>;
  updated_at: Scalars['Date'];
  versionIndex: Scalars['Int'];
  versionLabel: Scalars['String'];
  versionNotes?: Maybe<Scalars['String']>;
};

export type ApiVersionCreateInput = {
  callLink: Scalars['String'];
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  documentationLink?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isLatest?: InputMaybe<Scalars['Boolean']>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  rootConnect?: InputMaybe<Scalars['ID']>;
  rootCreate?: InputMaybe<ApiCreateInput>;
  translationsCreate?: InputMaybe<Array<ApiVersionTranslationCreateInput>>;
  versionIndex: Scalars['Int'];
  versionLabel: Scalars['String'];
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type ApiVersionEdge = {
  __typename?: 'ApiVersionEdge';
  cursor: Scalars['String'];
  node: ApiVersion;
};

export type ApiVersionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  ownedByOrganizationId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ApiVersionSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ApiVersionSearchResult = {
  __typename?: 'ApiVersionSearchResult';
  edges: Array<ApiVersionEdge>;
  pageInfo: PageInfo;
};

export enum ApiVersionSortBy {
  CalledByRoutinesAsc = 'CalledByRoutinesAsc',
  CalledByRoutinesDesc = 'CalledByRoutinesDesc',
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
  details?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  summary?: Maybe<Scalars['String']>;
};

export type ApiVersionTranslationCreateInput = {
  details: Scalars['String'];
  id: Scalars['ID'];
  language: Scalars['String'];
  summary?: InputMaybe<Scalars['String']>;
};

export type ApiVersionTranslationUpdateInput = {
  details?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  summary?: InputMaybe<Scalars['String']>;
};

export type ApiVersionUpdateInput = {
  callLink?: InputMaybe<Scalars['String']>;
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  directoryListingsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  documentationLink?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isLatest?: InputMaybe<Scalars['Boolean']>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
  translationsCreate?: InputMaybe<Array<ApiVersionTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ApiVersionTranslationUpdateInput>>;
  versionIndex?: InputMaybe<Scalars['Int']>;
  versionLabel?: InputMaybe<Scalars['String']>;
  versionNotes?: InputMaybe<Scalars['String']>;
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
  id: Scalars['ID'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  owner?: Maybe<Owner>;
  permissionsComment?: Maybe<CommentPermission>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  translations: Array<CommentTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
};

export type CommentCreateInput = {
  createdFor: CommentFor;
  forConnect: Scalars['ID'];
  id: Scalars['ID'];
  parentConnect?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<CommentTranslationCreateInput>>;
};

export enum CommentFor {
  ApiVersion = 'ApiVersion',
  Issue = 'Issue',
  NoteVersion = 'NoteVersion',
  Post = 'Post',
  ProjectVersion = 'ProjectVersion',
  PullRequest = 'PullRequest',
  Question = 'Question',
  QuestionAnswer = 'QuestionAnswer',
  RoutineVersion = 'RoutineVersion',
  SmartContractVersion = 'SmartContractVersion',
  StandardVersion = 'StandardVersion'
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
  apiVersionId?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  issueId?: InputMaybe<Scalars['ID']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  noteVersionId?: InputMaybe<Scalars['ID']>;
  ownedByOrganizationId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  postId?: InputMaybe<Scalars['ID']>;
  projectVersionId?: InputMaybe<Scalars['ID']>;
  pullRequestId?: InputMaybe<Scalars['ID']>;
  questionAnswerId?: InputMaybe<Scalars['ID']>;
  questionId?: InputMaybe<Scalars['ID']>;
  routineVersionId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  smartContractVersionId?: InputMaybe<Scalars['ID']>;
  sortBy?: InputMaybe<CommentSortBy>;
  standardVersionId?: InputMaybe<Scalars['ID']>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
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
  ScoreAsc = 'ScoreAsc',
  ScoreDesc = 'ScoreDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc'
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
  Issue = 'Issue',
  Meeting = 'Meeting',
  MeetingInvite = 'MeetingInvite',
  Node = 'Node',
  Organization = 'Organization',
  Post = 'Post',
  Project = 'Project',
  ProjectVersion = 'ProjectVersion',
  PullRequest = 'PullRequest',
  PushDevice = 'PushDevice',
  Question = 'Question',
  QuestionAnswer = 'QuestionAnswer',
  Quiz = 'Quiz',
  Reminder = 'Reminder',
  ReminderList = 'ReminderList',
  Report = 'Report',
  Routine = 'Routine',
  RoutineVersion = 'RoutineVersion',
  RunProject = 'RunProject',
  RunRoutine = 'RunRoutine',
  SmartContract = 'SmartContract',
  SmartContractVersion = 'SmartContractVersion',
  Standard = 'Standard',
  StandardVersion = 'StandardVersion',
  Transfer = 'Transfer',
  UserSchedule = 'UserSchedule',
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
  verified: Scalars['Boolean'];
};

export type EmailCreateInput = {
  emailAddress: Scalars['String'];
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

export type Issue = {
  __typename?: 'Issue';
  closedAt?: Maybe<Scalars['Date']>;
  closedBy?: Maybe<User>;
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  labels: Array<Label>;
  labelsCount: Scalars['Int'];
  permissionsIssue: IssuePermission;
  referencedVersionConnect?: Maybe<Scalars['ID']>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  score: Scalars['Int'];
  starredBy?: Maybe<Array<Star>>;
  stars: Scalars['Int'];
  status: IssueStatus;
  to: IssueTo;
  translations: Array<IssueTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
};

export type IssueCloseInput = {
  id: Scalars['ID'];
  status: IssueStatus;
};

export type IssueCreateInput = {
  forConnect: Scalars['ID'];
  id: Scalars['ID'];
  issueFor: IssueFor;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  translationsCreate?: InputMaybe<Array<IssueTranslationCreateInput>>;
};

export type IssueEdge = {
  __typename?: 'IssueEdge';
  cursor: Scalars['String'];
  node: Issue;
};

export enum IssueFor {
  Api = 'Api',
  Note = 'Note',
  Organization = 'Organization',
  Project = 'Project',
  Routine = 'Routine',
  SmartContract = 'SmartContract',
  Standard = 'Standard'
}

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
  apiId?: InputMaybe<Scalars['ID']>;
  closedById?: InputMaybe<Scalars['ID']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  noteId?: InputMaybe<Scalars['ID']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  referencedVersionId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  smartContractId?: InputMaybe<Scalars['ID']>;
  sortBy?: InputMaybe<IssueSortBy>;
  standardId?: InputMaybe<Scalars['ID']>;
  status?: InputMaybe<IssueStatus>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type IssueSearchResult = {
  __typename?: 'IssueSearchResult';
  edges: Array<IssueEdge>;
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
  ScoreAsc = 'ScoreAsc',
  ScoreDesc = 'ScoreDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc'
}

export enum IssueStatus {
  CloseUnresolved = 'CloseUnresolved',
  ClosedResolved = 'ClosedResolved',
  Open = 'Open',
  Rejected = 'Rejected'
}

export type IssueTo = Api | Note | Organization | Project | Routine | SmartContract | Standard;

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
  id: Scalars['ID'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  labelsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  translationsCreate?: InputMaybe<Array<IssueTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<IssueTranslationUpdateInput>>;
};

export type Label = {
  __typename?: 'Label';
  apis?: Maybe<Array<Api>>;
  apisCount: Scalars['Int'];
  color?: Maybe<Scalars['String']>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  issues?: Maybe<Array<Issue>>;
  issuesCount: Scalars['Int'];
  label: Scalars['String'];
  meetings?: Maybe<Array<Meeting>>;
  meetingsCount: Scalars['Int'];
  notes?: Maybe<Array<Note>>;
  notesCount: Scalars['Int'];
  permissionsLabel: LabelPermission;
  projects?: Maybe<Array<Project>>;
  projectsCount: Scalars['Int'];
  routines?: Maybe<Array<Routine>>;
  routinesCount: Scalars['Int'];
  runProjectSchedules?: Maybe<Array<RunProjectSchedule>>;
  runProjectSchedulesCount: Scalars['Int'];
  runRoutineSchedules?: Maybe<Array<RunRoutineSchedule>>;
  runRoutineSchedulesCount: Scalars['Int'];
  smartContracts?: Maybe<Array<SmartContract>>;
  smartContractsCount: Scalars['Int'];
  standards?: Maybe<Array<Standard>>;
  standardsCount: Scalars['Int'];
  translations: Array<LabelTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  userSchedules?: Maybe<Array<UserSchedule>>;
  userSchedulesCount: Scalars['Int'];
};

export type LabelCreateInput = {
  apisConnect?: InputMaybe<Array<Scalars['ID']>>;
  color?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  issuesConnect?: InputMaybe<Array<Scalars['ID']>>;
  label: Scalars['String'];
  meetingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  notesConnect?: InputMaybe<Array<Scalars['ID']>>;
  organizationConnect?: InputMaybe<Scalars['ID']>;
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
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  label?: InputMaybe<Scalars['String']>;
  ownedByOrganizationId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<LabelSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
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
  translationsCreate?: InputMaybe<Array<LabelTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<LabelTranslationUpdateInput>>;
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

export type Meeting = {
  __typename?: 'Meeting';
  attendees: Array<User>;
  attendeesCount: Scalars['Int'];
  eventEnd?: Maybe<Scalars['Date']>;
  eventStart?: Maybe<Scalars['Date']>;
  id: Scalars['ID'];
  invites: Array<MeetingInvite>;
  invitesCount: Scalars['Int'];
  labels: Array<Label>;
  labelsCount: Scalars['Int'];
  openToAnyoneWithInvite: Scalars['Boolean'];
  organization: Organization;
  permissionsMeeting: MeetingPermission;
  recurrEnd?: Maybe<Scalars['Date']>;
  recurrStart?: Maybe<Scalars['Date']>;
  recurring: Scalars['Boolean'];
  restrictedToRoles: Array<Role>;
  showOnOrganizationProfile: Scalars['Boolean'];
  timeZone?: Maybe<Scalars['String']>;
  translations: Array<MeetingTranslation>;
  translationsCount: Scalars['Int'];
};

export type MeetingCreateInput = {
  eventEnd?: InputMaybe<Scalars['Date']>;
  eventStart?: InputMaybe<Scalars['Date']>;
  id: Scalars['ID'];
  invitesCreate?: InputMaybe<Array<MeetingInviteCreateInput>>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  openToAnyoneWithInvite?: InputMaybe<Scalars['Boolean']>;
  organizationConnect: Scalars['ID'];
  recurrEnd?: InputMaybe<Scalars['Date']>;
  recurrStart?: InputMaybe<Scalars['Date']>;
  recurring?: InputMaybe<Scalars['Boolean']>;
  restrictedToRolesConnect?: InputMaybe<Array<Scalars['ID']>>;
  showOnOrganizationProfile?: InputMaybe<Scalars['Boolean']>;
  timeZone?: InputMaybe<Scalars['String']>;
  translationsCreate?: InputMaybe<Array<MeetingTranslationCreateInput>>;
};

export type MeetingEdge = {
  __typename?: 'MeetingEdge';
  cursor: Scalars['String'];
  node: Meeting;
};

export type MeetingInvite = {
  __typename?: 'MeetingInvite';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  meeting: Meeting;
  message?: Maybe<Scalars['String']>;
  permissionsMeetingInvite: MeetingInvitePermission;
  status: MeetingInviteStatus;
  updated_at: Scalars['Date'];
  user: User;
};

export type MeetingInviteCreateInput = {
  id: Scalars['ID'];
  meetingConnect: Scalars['ID'];
  message?: InputMaybe<Scalars['String']>;
  userConnect: Scalars['ID'];
};

export type MeetingInviteEdge = {
  __typename?: 'MeetingInviteEdge';
  cursor: Scalars['String'];
  node: MeetingInvite;
};

export type MeetingInvitePermission = {
  __typename?: 'MeetingInvitePermission';
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
};

export type MeetingInviteSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  meetingId?: InputMaybe<Scalars['ID']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<MeetingInviteSortBy>;
  status?: InputMaybe<MeetingInviteStatus>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type MeetingInviteSearchResult = {
  __typename?: 'MeetingInviteSearchResult';
  edges: Array<MeetingInviteEdge>;
  pageInfo: PageInfo;
};

export enum MeetingInviteSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  MemberNameAsc = 'MemberNameAsc',
  MemberNameDesc = 'MemberNameDesc',
  StatusAsc = 'StatusAsc',
  StatusDesc = 'StatusDesc'
}

export enum MeetingInviteStatus {
  Accepted = 'Accepted',
  Declined = 'Declined',
  Pending = 'Pending'
}

export type MeetingInviteUpdateInput = {
  id: Scalars['ID'];
  message?: InputMaybe<Scalars['String']>;
};

export type MeetingPermission = {
  __typename?: 'MeetingPermission';
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canInvite: Scalars['Boolean'];
};

export type MeetingSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  labelsId?: InputMaybe<Array<Scalars['ID']>>;
  maxEventEnd?: InputMaybe<Scalars['Date']>;
  maxEventStart?: InputMaybe<Scalars['Date']>;
  maxRecurrEnd?: InputMaybe<Scalars['Date']>;
  maxRecurrStart?: InputMaybe<Scalars['Date']>;
  minEventEnd?: InputMaybe<Scalars['Date']>;
  minEventStart?: InputMaybe<Scalars['Date']>;
  minRecurrEnd?: InputMaybe<Scalars['Date']>;
  minRecurrStart?: InputMaybe<Scalars['Date']>;
  openToAnyoneWithInvite?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  showOnOrganizationProfile?: InputMaybe<Scalars['Boolean']>;
  sortBy?: InputMaybe<MeetingSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type MeetingSearchResult = {
  __typename?: 'MeetingSearchResult';
  edges: Array<MeetingEdge>;
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
  link?: Maybe<Scalars['String']>;
  name?: Maybe<Scalars['String']>;
};

export type MeetingTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  link?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type MeetingTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  link?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type MeetingUpdateInput = {
  eventEnd?: InputMaybe<Scalars['Date']>;
  eventStart?: InputMaybe<Scalars['Date']>;
  id: Scalars['ID'];
  invitesCreate?: InputMaybe<Array<MeetingInviteCreateInput>>;
  invitesDelete?: InputMaybe<Array<Scalars['ID']>>;
  invitesUpdate?: InputMaybe<Array<MeetingInviteUpdateInput>>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  openToAnyoneWithInvite?: InputMaybe<Scalars['Boolean']>;
  recurrEnd?: InputMaybe<Scalars['Date']>;
  recurrStart?: InputMaybe<Scalars['Date']>;
  recurring?: InputMaybe<Scalars['Boolean']>;
  restrictedToRolesConnect?: InputMaybe<Array<Scalars['ID']>>;
  restrictedToRolesDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  showOnOrganizationProfile?: InputMaybe<Scalars['Boolean']>;
  timeZone?: InputMaybe<Scalars['String']>;
  translationsCreate?: InputMaybe<Array<MeetingTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<MeetingTranslationUpdateInput>>;
};

export type Member = {
  __typename?: 'Member';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isAdmin: Scalars['Boolean'];
  organization: Organization;
  permissions: Scalars['String'];
  updated_at: Scalars['Date'];
  user: User;
};

export type MemberEdge = {
  __typename?: 'MemberEdge';
  cursor: Scalars['String'];
  node: Member;
};

export type MemberInvite = {
  __typename?: 'MemberInvite';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  message?: Maybe<Scalars['String']>;
  organization: Organization;
  permissionsMemberInvite: MemberInvitePermission;
  status: MemberInviteStatus;
  updated_at: Scalars['Date'];
  user: User;
  willBeAdmin: Scalars['Boolean'];
  willHavePermissions?: Maybe<Scalars['String']>;
};

export type MemberInviteCreateInput = {
  id: Scalars['ID'];
  message?: InputMaybe<Scalars['String']>;
  organizationConnect: Scalars['ID'];
  userConnect: Scalars['ID'];
  willBeAdmin?: InputMaybe<Scalars['Boolean']>;
  willHavePermissions?: InputMaybe<Scalars['String']>;
};

export type MemberInviteEdge = {
  __typename?: 'MemberInviteEdge';
  cursor: Scalars['String'];
  node: MemberInvite;
};

export type MemberInvitePermission = {
  __typename?: 'MemberInvitePermission';
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
};

export type MemberInviteSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  organizationId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<MemberInviteSortBy>;
  status?: InputMaybe<MemberInviteStatus>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type MemberInviteSearchResult = {
  __typename?: 'MemberInviteSearchResult';
  edges: Array<MemberInviteEdge>;
  pageInfo: PageInfo;
};

export enum MemberInviteSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  MemberNameAsc = 'MemberNameAsc',
  MemberNameDesc = 'MemberNameDesc',
  StatusAsc = 'StatusAsc',
  StatusDesc = 'StatusDesc'
}

export enum MemberInviteStatus {
  Accepted = 'Accepted',
  Declined = 'Declined',
  Pending = 'Pending'
}

export type MemberInviteUpdateInput = {
  id: Scalars['ID'];
  message?: InputMaybe<Scalars['String']>;
  willBeAdmin?: InputMaybe<Scalars['Boolean']>;
  willHavePermissions?: InputMaybe<Scalars['String']>;
};

export type MemberSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isAdmin?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  roles?: InputMaybe<Array<Scalars['String']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<MemberSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type MemberSearchResult = {
  __typename?: 'MemberSearchResult';
  edges: Array<MemberEdge>;
  pageInfo: PageInfo;
};

export enum MemberSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type MemberUpdateInput = {
  id: Scalars['ID'];
  isAdmin?: InputMaybe<Scalars['Boolean']>;
  permissions?: InputMaybe<Scalars['String']>;
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
  exportData: Scalars['String'];
  guestLogIn: Session;
  issueClose: Issue;
  issueCreate: Issue;
  issueUpdate: Issue;
  labelCreate: Label;
  labelUpdate: Label;
  logOut: Session;
  meetingCreate: Meeting;
  meetingInviteAccept: MeetingInvite;
  meetingInviteCreate: MeetingInvite;
  meetingInviteDecline: MeetingInvite;
  meetingInviteUpdate: MeetingInvite;
  meetingUpdate: Meeting;
  memberInviteCreate: MemberInvite;
  memberInviteUpdate: MemberInvite;
  memberUpdate: Member;
  nodeCreate: Node;
  nodeUpdate: Node;
  noteCreate: Note;
  noteUpdate: Note;
  noteVersionCreate: NoteVersion;
  noteVersionUpdate: NoteVersion;
  notificationMarkAllAsRead: Count;
  notificationMarkAsRead: Success;
  notificationSettingsUpdate: NotificationSettings;
  notificationSubscriptionCreate: NotificationSubscription;
  notificationSubscriptionUpdate: NotificationSubscription;
  organizationCreate: Organization;
  organizationUpdate: Organization;
  phoneCreate: Phone;
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
  questionAnswerMarkAsAccepted: QuestionAnswer;
  questionAnswerUpdate: QuestionAnswer;
  questionCreate: Question;
  questionUpdate: Question;
  quizAttemptCreate: QuizAttempt;
  quizAttemptUpdate: QuizAttempt;
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
  roleCreate: Role;
  roleUpdate: Role;
  routineCreate: Routine;
  routineUpdate: Routine;
  routineVersionCreate: RoutineVersion;
  routineVersionUpdate: RoutineVersion;
  runProjectCancel: RunProject;
  runProjectComplete: RunProject;
  runProjectCreate: RunProject;
  runProjectDeleteAll: Count;
  runProjectScheduleCreate: RunProjectSchedule;
  runProjectScheduleUpdate: RunProjectSchedule;
  runProjectUpdate: RunProject;
  runRoutineCancel: RunRoutine;
  runRoutineComplete: RunRoutine;
  runRoutineCreate: RunRoutine;
  runRoutineDeleteAll: Count;
  runRoutineScheduleCreate: RunRoutineSchedule;
  runRoutineScheduleUpdate: RunRoutineSchedule;
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
  transferAccept: Transfer;
  transferCancel: Transfer;
  transferDeny: Transfer;
  transferRequestReceive: Transfer;
  transferRequestSend: Transfer;
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


export type MutationIssueCloseArgs = {
  input: IssueCloseInput;
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


export type MutationMeetingInviteAcceptArgs = {
  input: FindByIdInput;
};


export type MutationMeetingInviteCreateArgs = {
  input: MeetingInviteCreateInput;
};


export type MutationMeetingInviteDeclineArgs = {
  input: FindByIdInput;
};


export type MutationMeetingInviteUpdateArgs = {
  input: MeetingInviteUpdateInput;
};


export type MutationMeetingUpdateArgs = {
  input: MeetingUpdateInput;
};


export type MutationMemberInviteCreateArgs = {
  input: MemberInviteCreateInput;
};


export type MutationMemberInviteUpdateArgs = {
  input: MemberInviteUpdateInput;
};


export type MutationMemberUpdateArgs = {
  input: MemberUpdateInput;
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


export type MutationNotificationSettingsUpdateArgs = {
  input: NotificationSettingsUpdateInput;
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


export type MutationQuestionAnswerMarkAsAcceptedArgs = {
  input: FindByIdInput;
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


export type MutationQuizAttemptCreateArgs = {
  input: QuizAttemptCreateInput;
};


export type MutationQuizAttemptUpdateArgs = {
  input: QuizAttemptUpdateInput;
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


export type MutationRoleCreateArgs = {
  input: RoleCreateInput;
};


export type MutationRoleUpdateArgs = {
  input: RoleUpdateInput;
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


export type MutationRunProjectCancelArgs = {
  input: RunProjectCancelInput;
};


export type MutationRunProjectCompleteArgs = {
  input: RunProjectCompleteInput;
};


export type MutationRunProjectCreateArgs = {
  input: RunProjectCreateInput;
};


export type MutationRunProjectScheduleCreateArgs = {
  input: RunProjectScheduleCreateInput;
};


export type MutationRunProjectScheduleUpdateArgs = {
  input: RunProjectScheduleUpdateInput;
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


export type MutationRunRoutineScheduleCreateArgs = {
  input: RunRoutineScheduleCreateInput;
};


export type MutationRunRoutineScheduleUpdateArgs = {
  input: RunRoutineScheduleUpdateInput;
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


export type MutationTransferAcceptArgs = {
  input: FindByIdInput;
};


export type MutationTransferCancelArgs = {
  input: FindByIdInput;
};


export type MutationTransferDenyArgs = {
  input: TransferDenyInput;
};


export type MutationTransferRequestReceiveArgs = {
  input: TransferRequestReceiveInput;
};


export type MutationTransferRequestSendArgs = {
  input: TransferRequestSendInput;
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
  loop?: Maybe<NodeLoop>;
  routineVersion: RoutineVersion;
  routineVersionId: Scalars['ID'];
  rowIndex?: Maybe<Scalars['Int']>;
  translations: Array<NodeTranslation>;
  type: NodeType;
  updated_at: Scalars['Date'];
};

export type NodeCreateInput = {
  columnIndex?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  loopCreate?: InputMaybe<NodeLoopCreateInput>;
  nodeEndCreate?: InputMaybe<NodeEndCreateInput>;
  nodeRoutineListCreate?: InputMaybe<NodeRoutineListCreateInput>;
  routineVersionConnect: Scalars['ID'];
  rowIndex?: InputMaybe<Scalars['Int']>;
  translationsCreate?: InputMaybe<Array<NodeTranslationCreateInput>>;
  type: NodeType;
};

export type NodeData = NodeEnd | NodeRoutineList;

export type NodeEnd = {
  __typename?: 'NodeEnd';
  id: Scalars['ID'];
  nodeConnect: any;
  suggestedNextRoutineVersion?: Maybe<RoutineVersion>;
  wasSuccessful: Scalars['Boolean'];
};

export type NodeEndCreateInput = {
  id: Scalars['ID'];
  nodeConnect: any;
  suggestedNextRoutineVersionsConnect?: string[];
  wasSuccessful?: InputMaybe<Scalars['Boolean']>;
};

export type NodeEndUpdateInput = {
  id: Scalars['ID'];
  suggestedNextRoutineVersionsConnect?: string[];
  suggestedNextRoutineVersionsDisconnect?: string[];
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
  fromConnect: Scalars['ID'];
  id: Scalars['ID'];
  operation?: InputMaybe<Scalars['String']>;
  routineVersionConnect: Scalars['ID'];
  toConnect: Scalars['ID'];
  whensCreate?: InputMaybe<Array<NodeLinkWhenCreateInput>>;
};

export type NodeLinkUpdateInput = {
  fromConnect?: InputMaybe<Scalars['ID']>;
  id: Scalars['ID'];
  operation?: InputMaybe<Scalars['String']>;
  toConnect?: InputMaybe<Scalars['ID']>;
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
  linkConnect: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<NodeLinkWhenTranslationCreateInput>>;
};

export type NodeLinkWhenTranslation = {
  __typename?: 'NodeLinkWhenTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type NodeLinkWhenTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type NodeLinkWhenTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type NodeLinkWhenUpdateInput = {
  condition?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  linkConnect?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<NodeLinkWhenTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<NodeLinkWhenTranslationUpdateInput>>;
};

export type NodeLoop = {
  __typename?: 'NodeLoop';
  id: Scalars['ID'];
  loops?: Maybe<Scalars['Int']>;
  maxLoops?: Maybe<Scalars['Int']>;
  operation?: Maybe<Scalars['String']>;
  whiles: Array<NodeLoopWhile>;
};

export type NodeLoopCreateInput = {
  id: Scalars['ID'];
  loops?: InputMaybe<Scalars['Int']>;
  maxLoops?: InputMaybe<Scalars['Int']>;
  operation?: InputMaybe<Scalars['String']>;
  whilesCreate: Array<NodeLoopWhileCreateInput>;
};

export type NodeLoopUpdateInput = {
  id: Scalars['ID'];
  loops?: InputMaybe<Scalars['Int']>;
  maxLoops?: InputMaybe<Scalars['Int']>;
  operation?: InputMaybe<Scalars['String']>;
  whilesCreate: Array<NodeLoopWhileCreateInput>;
  whilesDelete?: InputMaybe<Array<Scalars['ID']>>;
  whilesUpdate: Array<NodeLoopWhileUpdateInput>;
};

export type NodeLoopWhile = {
  __typename?: 'NodeLoopWhile';
  condition: Scalars['String'];
  id: Scalars['ID'];
  toId?: Maybe<Scalars['ID']>;
  translations: Array<NodeLoopWhileTranslation>;
};

export type NodeLoopWhileCreateInput = {
  condition: Scalars['String'];
  id: Scalars['ID'];
  toConnect?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<NodeLoopWhileTranslationCreateInput>>;
};

export type NodeLoopWhileTranslation = {
  __typename?: 'NodeLoopWhileTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type NodeLoopWhileTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type NodeLoopWhileTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type NodeLoopWhileUpdateInput = {
  condition?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  toConnect?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<NodeLoopWhileTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<NodeLoopWhileTranslationUpdateInput>>;
};

export type NodeRoutineList = {
  __typename?: 'NodeRoutineList';
  id: Scalars['ID'];
  isOptional: Scalars['Boolean'];
  isOrdered: Scalars['Boolean'];
  items: Array<NodeRoutineListItem>;
};

export type NodeRoutineListCreateInput = {
  id: Scalars['ID'];
  isOptional?: InputMaybe<Scalars['Boolean']>;
  isOrdered?: InputMaybe<Scalars['Boolean']>;
  itemsCreate?: InputMaybe<Array<NodeRoutineListItemCreateInput>>;
  nodeConnect: Scalars['ID'];
};

export type NodeRoutineListItem = {
  __typename?: 'NodeRoutineListItem';
  id: Scalars['ID'];
  index: Scalars['Int'];
  isOptional: Scalars['Boolean'];
  routineVersion: RoutineVersion;
  translations: Array<NodeRoutineListItemTranslation>;
};

export type NodeRoutineListItemCreateInput = {
  id: Scalars['ID'];
  index: Scalars['Int'];
  isOptional?: InputMaybe<Scalars['Boolean']>;
  listConnect: Scalars['ID'];
  routineVersionConnect: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<NodeRoutineListItemTranslationCreateInput>>;
};

export type NodeRoutineListItemTranslation = {
  __typename?: 'NodeRoutineListItemTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name?: Maybe<Scalars['String']>;
};

export type NodeRoutineListItemTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
};

export type NodeRoutineListItemTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type NodeRoutineListItemUpdateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  isOptional?: InputMaybe<Scalars['Boolean']>;
  routineVersionUpdate?: InputMaybe<RoutineVersionUpdateInput>;
  translationsCreate?: InputMaybe<Array<NodeRoutineListItemTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<NodeRoutineListItemTranslationUpdateInput>>;
};

export type NodeRoutineListUpdateInput = {
  id: Scalars['ID'];
  isOptional?: InputMaybe<Scalars['Boolean']>;
  isOrdered?: InputMaybe<Scalars['Boolean']>;
  itemsCreate?: InputMaybe<Array<NodeRoutineListItemCreateInput>>;
  itemsDelete?: InputMaybe<Array<Scalars['ID']>>;
  itemsUpdate?: InputMaybe<Array<NodeRoutineListItemUpdateInput>>;
};

export type NodeTranslation = {
  __typename?: 'NodeTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type NodeTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type NodeTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
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
  loopCreate?: InputMaybe<NodeLoopCreateInput>;
  loopDelete?: InputMaybe<Scalars['ID']>;
  loopUpdate?: InputMaybe<NodeLoopUpdateInput>;
  nodeEndUpdate?: InputMaybe<NodeEndUpdateInput>;
  nodeRoutineListUpdate?: InputMaybe<NodeRoutineListUpdateInput>;
  routineVersionConnect?: InputMaybe<Scalars['ID']>;
  rowIndex?: InputMaybe<Scalars['Int']>;
  translationsCreate?: InputMaybe<Array<NodeTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<NodeTranslationUpdateInput>>;
  type?: InputMaybe<NodeType>;
};

export type Note = {
  __typename?: 'Note';
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  issues: Array<Issue>;
  issuesCount: Scalars['Int'];
  labels: Array<Label>;
  labelsCount: Scalars['Int'];
  owner?: Maybe<Owner>;
  parent?: Maybe<Note>;
  permissionsRoot: RootPermission;
  pullRequests: Array<PullRequest>;
  pullRequestsCount: Scalars['Int'];
  questions: Array<Question>;
  questionsCount: Scalars['Int'];
  score: Scalars['Int'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  stats: Array<StatsNote>;
  tags: Array<Tag>;
  transfers: Array<Transfer>;
  transfersCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  versions: Array<NoteVersion>;
  versionsCount: Scalars['Int'];
  views: Scalars['Int'];
};

export type NoteCreateInput = {
  id: Scalars['ID'];
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  organizationConnect?: InputMaybe<Scalars['ID']>;
  parentConnect?: InputMaybe<Scalars['ID']>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  userConnect?: InputMaybe<Scalars['ID']>;
  versionsCreate?: InputMaybe<Array<NoteVersionCreateInput>>;
};

export type NoteEdge = {
  __typename?: 'NoteEdge';
  cursor: Scalars['String'];
  node: Note;
};

export type NoteSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  maxScore?: InputMaybe<Scalars['Int']>;
  maxStars?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  ownedByOrganizationId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<NoteSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
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
  ScoreAsc = 'ScoreAsc',
  ScoreDesc = 'ScoreDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc'
}

export type NoteUpdateInput = {
  id: Scalars['ID'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  labelsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  organizationConnect?: InputMaybe<Scalars['ID']>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  userConnect?: InputMaybe<Scalars['ID']>;
  versionsCreate?: InputMaybe<Array<ApiVersionCreateInput>>;
  versionsDelete?: InputMaybe<Array<Scalars['ID']>>;
  versionsUpdate?: InputMaybe<Array<ApiVersionUpdateInput>>;
};

export type NoteVersion = {
  __typename?: 'NoteVersion';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  created_at: Scalars['Date'];
  directoryListings: Array<ProjectVersionDirectory>;
  directoryListingsCount: Scalars['Int'];
  forks: Array<Note>;
  forksCount: Scalars['Int'];
  id: Scalars['ID'];
  isLatest: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  permissionsVersion: VersionPermission;
  pullRequest?: Maybe<PullRequest>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  root: Note;
  translations: Array<NoteVersionTranslation>;
  updated_at: Scalars['Date'];
  versionIndex: Scalars['Int'];
  versionLabel: Scalars['String'];
  versionNotes?: Maybe<Scalars['String']>;
};

export type NoteVersionCreateInput = {
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  isLatest?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  rootConnect?: InputMaybe<Scalars['ID']>;
  rootCreate?: InputMaybe<NoteCreateInput>;
  translationsCreate?: InputMaybe<Array<NoteVersionTranslationCreateInput>>;
  versionIndex: Scalars['Int'];
  versionLabel: Scalars['String'];
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type NoteVersionEdge = {
  __typename?: 'NoteVersionEdge';
  cursor: Scalars['String'];
  node: NoteVersion;
};

export type NoteVersionPermission = {
  __typename?: 'NoteVersionPermission';
  canComment: Scalars['Boolean'];
  canCopy: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type NoteVersionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  ownedByOrganizationId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<NoteVersionSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type NoteVersionSearchResult = {
  __typename?: 'NoteVersionSearchResult';
  edges: Array<NoteVersionEdge>;
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
  text: Scalars['String'];
};

export type NoteVersionTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  text: Scalars['String'];
};

export type NoteVersionTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  text?: InputMaybe<Scalars['String']>;
};

export type NoteVersionUpdateInput = {
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  directoryListingsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  isLatest?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  translationsCreate?: InputMaybe<Array<NoteVersionTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<NoteVersionTranslationUpdateInput>>;
  versionIndex?: InputMaybe<Scalars['Int']>;
  versionLabel?: InputMaybe<Scalars['String']>;
  versionNotes?: InputMaybe<Scalars['String']>;
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

export type NotificationSettings = {
  __typename?: 'NotificationSettings';
  categories?: Maybe<Array<NotificationSettingsCategory>>;
  dailyLimit?: Maybe<Scalars['Int']>;
  enabled: Scalars['Boolean'];
  includedEmails?: Maybe<Array<Scalars['ID']>>;
  includedPush?: Maybe<Array<Scalars['ID']>>;
  includedSms?: Maybe<Array<Scalars['ID']>>;
  toEmails?: Maybe<Scalars['Boolean']>;
  toPush?: Maybe<Scalars['Boolean']>;
  toSms?: Maybe<Scalars['Boolean']>;
};

export type NotificationSettingsCategory = {
  __typename?: 'NotificationSettingsCategory';
  category: Scalars['String'];
  dailyLimit?: Maybe<Scalars['Int']>;
  enabled: Scalars['Boolean'];
  toEmails?: Maybe<Scalars['Boolean']>;
  toPush?: Maybe<Scalars['Boolean']>;
  toSms?: Maybe<Scalars['Boolean']>;
};

export type NotificationSettingsCategoryUpdateInput = {
  category: Scalars['String'];
  dailyLimit?: InputMaybe<Scalars['Int']>;
  enabled?: InputMaybe<Scalars['Boolean']>;
  toEmails?: InputMaybe<Scalars['Boolean']>;
  toPush?: InputMaybe<Scalars['Boolean']>;
  toSms?: InputMaybe<Scalars['Boolean']>;
};

export type NotificationSettingsUpdateInput = {
  categories?: InputMaybe<Array<NotificationSettingsCategoryUpdateInput>>;
  dailyLimit?: InputMaybe<Scalars['Int']>;
  enabled?: InputMaybe<Scalars['Boolean']>;
  includedEmails?: InputMaybe<Array<Scalars['ID']>>;
  includedPush?: InputMaybe<Array<Scalars['ID']>>;
  includedSms?: InputMaybe<Array<Scalars['ID']>>;
  toEmails?: InputMaybe<Scalars['Boolean']>;
  toPush?: InputMaybe<Scalars['Boolean']>;
  toSms?: InputMaybe<Scalars['Boolean']>;
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
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  object: SubscribedObject;
  silent: Scalars['Boolean'];
};

export type NotificationSubscriptionCreateInput = {
  id: Scalars['ID'];
  objectConnect: Scalars['ID'];
  objectType: SubscribableObject;
  silent?: InputMaybe<Scalars['Boolean']>;
};

export type NotificationSubscriptionEdge = {
  __typename?: 'NotificationSubscriptionEdge';
  cursor: Scalars['String'];
  node: NotificationSubscription;
};

export type NotificationSubscriptionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  objectId?: InputMaybe<Scalars['ID']>;
  objectType?: InputMaybe<SubscribableObject>;
  silent?: InputMaybe<Scalars['Boolean']>;
  sortBy?: InputMaybe<NotificationSubscriptionSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type NotificationSubscriptionSearchResult = {
  __typename?: 'NotificationSubscriptionSearchResult';
  edges: Array<NotificationSubscriptionEdge>;
  pageInfo: PageInfo;
};

export enum NotificationSubscriptionSortBy {
  ObjectTypeAsc = 'ObjectTypeAsc',
  ObjectTypeDesc = 'ObjectTypeDesc'
}

export type NotificationSubscriptionUpdateInput = {
  id: Scalars['ID'];
  silent?: InputMaybe<Scalars['Boolean']>;
};

export type Organization = {
  __typename?: 'Organization';
  apis: Array<Api>;
  apisCount: Scalars['Int'];
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  created_at: Scalars['Date'];
  directoryListings: Array<ProjectVersionDirectory>;
  forks: Array<Organization>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isOpenToNewMembers: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isViewed: Scalars['Boolean'];
  issues: Array<Issue>;
  issuesCount: Scalars['Int'];
  labels: Array<Label>;
  labelsCount: Scalars['Int'];
  meetings: Array<Meeting>;
  meetingsCount: Scalars['Int'];
  members: Array<Member>;
  membersCount: Scalars['Int'];
  notes: Array<Note>;
  notesCount: Scalars['Int'];
  parent?: Maybe<Organization>;
  paymentHistory: Array<Payment>;
  permissionsOrganization?: Maybe<OrganizationPermission>;
  posts: Array<Post>;
  postsCount: Scalars['Int'];
  premium?: Maybe<Premium>;
  projects: Array<Project>;
  projectsCount: Scalars['Int'];
  questions: Array<Question>;
  questionsCount: Scalars['Int'];
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceList?: Maybe<ResourceList>;
  roles?: Maybe<Array<Role>>;
  rolesCount: Scalars['Int'];
  routines: Array<Routine>;
  routinesCount: Scalars['Int'];
  smartContracts: Array<SmartContract>;
  smartContractsCount: Scalars['Int'];
  standards: Array<Standard>;
  standardsCount: Scalars['Int'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  stats: Array<StatsOrganization>;
  tags: Array<Tag>;
  transfersIncoming: Array<Transfer>;
  transfersOutgoing: Array<Transfer>;
  translations: Array<OrganizationTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets: Array<Wallet>;
};

export type OrganizationCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isOpenToNewMembers?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  memberInvitesCreate?: InputMaybe<Array<MemberInviteCreateInput>>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  rolesCreate?: InputMaybe<Array<RoleCreateInput>>;
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
  maxStars?: InputMaybe<Scalars['Int']>;
  maxViews?: InputMaybe<Scalars['Int']>;
  memberUserIds?: InputMaybe<Array<Scalars['ID']>>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  projectId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<OrganizationSortBy>;
  standardId?: InputMaybe<Scalars['ID']>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
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
  memberInvitesCreate?: InputMaybe<Array<MemberInviteCreateInput>>;
  memberInvitesDelete?: InputMaybe<Array<Scalars['ID']>>;
  membersDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
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

export type Owner = Organization | User;

export type PageInfo = {
  __typename?: 'PageInfo';
  endCursor?: Maybe<Scalars['String']>;
  hasNextPage: Scalars['Boolean'];
};

export type Payment = {
  __typename?: 'Payment';
  amount: Scalars['Int'];
  cardExpDate?: Maybe<Scalars['String']>;
  cardLast4?: Maybe<Scalars['String']>;
  cardType?: Maybe<Scalars['String']>;
  created_at: Scalars['Date'];
  currency: Scalars['String'];
  description: Scalars['String'];
  id: Scalars['ID'];
  organization: Organization;
  paymentMethod: Scalars['String'];
  status: PaymentStatus;
  updated_at: Scalars['Date'];
  user: User;
};

export enum PaymentStatus {
  Failed = 'Failed',
  Paid = 'Paid',
  Pending = 'Pending'
}

export type Phone = {
  __typename?: 'Phone';
  id: Scalars['ID'];
  phoneNumber: Scalars['String'];
  verified: Scalars['Boolean'];
};

export type PhoneCreateInput = {
  phoneNumber: Scalars['String'];
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
  comments: Array<Comment>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  owner: Owner;
  reports: Array<Report>;
  repostedFrom?: Maybe<Post>;
  reposts: Array<Post>;
  score: Scalars['Int'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<PostTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
};

export type PostCreateInput = {
  id: Scalars['ID'];
  isPinned?: InputMaybe<Scalars['Boolean']>;
  isPublic?: InputMaybe<Scalars['Boolean']>;
  organizationConnect?: InputMaybe<Scalars['ID']>;
  repostedFromConnect?: InputMaybe<Scalars['ID']>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
};

export type PostEdge = {
  __typename?: 'PostEdge';
  cursor: Scalars['String'];
  node: Post;
};

export type PostSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isPinned?: InputMaybe<Scalars['Boolean']>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  repostedFromIds?: InputMaybe<Array<Scalars['ID']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<PostSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
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
  ScoreAsc = 'ScoreAsc',
  ScoreDesc = 'ScoreDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc'
}

export type PostTranslation = {
  __typename?: 'PostTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type PostTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type PostTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type PostUpdateInput = {
  id: Scalars['ID'];
  isPinned?: InputMaybe<Scalars['Boolean']>;
  isPublic?: InputMaybe<Scalars['Boolean']>;
  resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
};

export type Premium = {
  __typename?: 'Premium';
  customPlan?: Maybe<Scalars['String']>;
  enabledAt?: Maybe<Scalars['Date']>;
  expiresAt?: Maybe<Scalars['Date']>;
  id: Scalars['ID'];
  isActive: Scalars['Boolean'];
};

export type ProfileEmailUpdateInput = {
  currentPassword: Scalars['String'];
  emailsCreate?: InputMaybe<Array<EmailCreateInput>>;
  emailsDelete?: InputMaybe<Array<Scalars['ID']>>;
  newPassword?: InputMaybe<Scalars['String']>;
};

export type ProfileUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  isPrivateApis?: InputMaybe<Scalars['Boolean']>;
  isPrivateApisCreated?: InputMaybe<Scalars['Boolean']>;
  isPrivateMemberships?: InputMaybe<Scalars['Boolean']>;
  isPrivateOrganizationsCreated?: InputMaybe<Scalars['Boolean']>;
  isPrivateProjects?: InputMaybe<Scalars['Boolean']>;
  isPrivateProjectsCreated?: InputMaybe<Scalars['Boolean']>;
  isPrivatePullRequests?: InputMaybe<Scalars['Boolean']>;
  isPrivateQuestionsAnswered?: InputMaybe<Scalars['Boolean']>;
  isPrivateQuestionsAsked?: InputMaybe<Scalars['Boolean']>;
  isPrivateQuizzesCreated?: InputMaybe<Scalars['Boolean']>;
  isPrivateRoles?: InputMaybe<Scalars['Boolean']>;
  isPrivateRoutines?: InputMaybe<Scalars['Boolean']>;
  isPrivateRoutinesCreated?: InputMaybe<Scalars['Boolean']>;
  isPrivateSmartContracts?: InputMaybe<Scalars['Boolean']>;
  isPrivateStandards?: InputMaybe<Scalars['Boolean']>;
  isPrivateStandardsCreated?: InputMaybe<Scalars['Boolean']>;
  isPrivateStars?: InputMaybe<Scalars['Boolean']>;
  isPrivateVotes?: InputMaybe<Scalars['Boolean']>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  name?: InputMaybe<Scalars['String']>;
  notificationSettings?: InputMaybe<Scalars['String']>;
  schedulesCreate?: InputMaybe<Array<UserScheduleCreateInput>>;
  schedulesDelete?: InputMaybe<Array<Scalars['ID']>>;
  schedulesUpdate?: InputMaybe<Array<UserScheduleUpdateInput>>;
  theme?: InputMaybe<Scalars['String']>;
  translationsCreate?: InputMaybe<Array<UserTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<UserTranslationUpdateInput>>;
};

export type Project = {
  __typename?: 'Project';
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  handle?: Maybe<Scalars['String']>;
  hasCompleteVersion: Scalars['Boolean'];
  id: Scalars['ID'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  issues: Array<Issue>;
  issuesCount: Scalars['Int'];
  labels: Array<Label>;
  labelsCount: Scalars['Int'];
  owner?: Maybe<Owner>;
  parent?: Maybe<Project>;
  permissions?: Maybe<Scalars['String']>;
  permissionsRoot: RootPermission;
  pullRequests: Array<PullRequest>;
  pullRequestsCount: Scalars['Int'];
  questions: Array<Question>;
  questionsCount: Scalars['Int'];
  quizzes: Array<Quiz>;
  quizzesCount: Scalars['Int'];
  score: Scalars['Int'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  stats: Array<StatsProject>;
  tags: Array<Tag>;
  transfers: Array<Transfer>;
  updated_at: Scalars['Date'];
  versions: Array<ProjectVersion>;
  versionsCount: Scalars['Int'];
  views: Scalars['Int'];
};

export type ProjectCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  organizationConnect?: InputMaybe<Scalars['ID']>;
  parentConnect?: InputMaybe<Scalars['ID']>;
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  userConnect?: InputMaybe<Scalars['ID']>;
  versionsCreate?: InputMaybe<Array<ProjectVersionCreateInput>>;
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
  ScoreAsc = 'ScoreAsc',
  ScoreDesc = 'ScoreDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc'
}

export type ProjectSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  hasCompleteVersion?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  labelsId?: InputMaybe<Scalars['ID']>;
  maxScore?: InputMaybe<Scalars['Int']>;
  maxStars?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  ownedByOrganizationId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  translationLanguagesLatestVersion?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
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
  ScoreAsc = 'ScoreAsc',
  ScoreDesc = 'ScoreDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc'
}

export type ProjectUpdateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  labelsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  organizationConnect?: InputMaybe<Scalars['ID']>;
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  userConnect?: InputMaybe<Scalars['ID']>;
  versionsCreate?: InputMaybe<Array<ProjectVersionCreateInput>>;
  versionsDelete?: InputMaybe<Array<Scalars['ID']>>;
  versionsUpdate?: InputMaybe<Array<ProjectVersionUpdateInput>>;
};

export type ProjectVersion = {
  __typename?: 'ProjectVersion';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  created_at: Scalars['Date'];
  directories: Array<ProjectVersionDirectory>;
  directoriesCount: Scalars['Int'];
  directoryListings: Array<ProjectVersionDirectory>;
  directoryListingsCount: Scalars['Int'];
  forks: Array<Project>;
  forksCount: Scalars['Int'];
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isLatest: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  permissionsVersion: VersionPermission;
  pullRequest?: Maybe<PullRequest>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  root: Project;
  runs: Array<RunProject>;
  runsCount: Scalars['Int'];
  suggestedNextByProject: Array<Project>;
  translations: Array<ProjectVersionTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  versionIndex: Scalars['Int'];
  versionLabel: Scalars['String'];
  versionNotes?: Maybe<Scalars['String']>;
};

export type ProjectVersionCreateInput = {
  directoryListingsCreate?: InputMaybe<Array<ProjectVersionDirectoryCreateInput>>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isLatest?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  rootConnect?: InputMaybe<Scalars['ID']>;
  rootCreate?: InputMaybe<ProjectCreateInput>;
  suggestedNextByProjectConnect?: InputMaybe<Array<Scalars['ID']>>;
  translationsCreate?: InputMaybe<Array<ProjectVersionTranslationCreateInput>>;
  versionIndex: Scalars['Int'];
  versionLabel: Scalars['String'];
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type ProjectVersionDirectory = {
  __typename?: 'ProjectVersionDirectory';
  childApiVersions: Array<ApiVersion>;
  childNoteVersions: Array<NoteVersion>;
  childOrder?: Maybe<Scalars['String']>;
  childOrganizations: Array<Organization>;
  childProjectVersions: Array<ProjectVersion>;
  childRoutineVersions: Array<RoutineVersion>;
  childSmartContractVersions: Array<SmartContractVersion>;
  childStandardVersions: Array<StandardVersion>;
  children: Array<ProjectVersionDirectory>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isRoot: Scalars['Boolean'];
  parentDirectory?: Maybe<ProjectVersionDirectory>;
  projectVersion?: Maybe<ProjectVersion>;
  runProjectSteps: Array<RunProjectStep>;
  translations: Array<ProjectVersionDirectoryTranslation>;
  updated_at: Scalars['Date'];
};

export type ProjectVersionDirectoryCreateInput = {
  childOrder?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isRoot?: InputMaybe<Scalars['Boolean']>;
  parentDirectoryConnect?: InputMaybe<Scalars['ID']>;
  projectVersionConnect: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<ProjectVersionDirectoryTranslationCreateInput>>;
};

export type ProjectVersionDirectoryTranslation = {
  __typename?: 'ProjectVersionDirectoryTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name?: Maybe<Scalars['String']>;
};

export type ProjectVersionDirectoryTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
};

export type ProjectVersionDirectoryTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type ProjectVersionDirectoryUpdateInput = {
  childOrder?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isRoot?: InputMaybe<Scalars['Boolean']>;
  parentDirectoryConnect?: InputMaybe<Scalars['ID']>;
  parentDirectoryDisconnect?: InputMaybe<Scalars['ID']>;
  projectVersionConnect?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<ProjectVersionDirectoryTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectVersionDirectoryTranslationUpdateInput>>;
};

export type ProjectVersionEdge = {
  __typename?: 'ProjectVersionEdge';
  cursor: Scalars['String'];
  node: ProjectVersion;
};

export type ProjectVersionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  directoryListingsId?: InputMaybe<Scalars['ID']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  minScoreRoot?: InputMaybe<Scalars['Int']>;
  minStarsRoot?: InputMaybe<Scalars['Int']>;
  minViewsRoot?: InputMaybe<Scalars['Int']>;
  ownedByOrganizationId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  rootId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectVersionSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
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
  directoryListingsCreate?: InputMaybe<Array<ProjectVersionDirectoryCreateInput>>;
  directoryListingsDelete?: InputMaybe<Array<Scalars['ID']>>;
  directoryListingsUpdate?: InputMaybe<Array<ProjectVersionDirectoryUpdateInput>>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isLatest?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  suggestedNextByProjectConnect?: InputMaybe<Array<Scalars['ID']>>;
  suggestedNextByProjectDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  translationsCreate?: InputMaybe<Array<ProjectVersionTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectVersionTranslationUpdateInput>>;
  versionIndex?: InputMaybe<Scalars['Int']>;
  versionLabel?: InputMaybe<Scalars['String']>;
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type PullRequest = {
  __typename?: 'PullRequest';
  comments: Array<Comment>;
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  from: PullRequestFrom;
  id: Scalars['ID'];
  mergedOrRejectedAt?: Maybe<Scalars['Date']>;
  permissionsPullRequest: PullRequestPermission;
  status: PullRequestStatus;
  to: PullRequestTo;
  updated_at: Scalars['Date'];
};

export type PullRequestCreateInput = {
  fromConnect: Scalars['ID'];
  id: Scalars['ID'];
  toConnect: Scalars['ID'];
  toObjectType: PullRequestToObjectType;
};

export type PullRequestEdge = {
  __typename?: 'PullRequestEdge';
  cursor: Scalars['String'];
  node: PullRequest;
};

export type PullRequestFrom = ApiVersion | NoteVersion | ProjectVersion | RoutineVersion | SmartContractVersion | StandardVersion;

export type PullRequestPermission = {
  __typename?: 'PullRequestPermission';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
};

export type PullRequestSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isMergedOrRejected?: InputMaybe<Scalars['Boolean']>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<PullRequestSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  toId?: InputMaybe<Scalars['ID']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type PullRequestSearchResult = {
  __typename?: 'PullRequestSearchResult';
  edges: Array<PullRequestEdge>;
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

export enum PullRequestStatus {
  Merged = 'Merged',
  Open = 'Open',
  Rejected = 'Rejected'
}

export type PullRequestTo = Api | Note | Project | Routine | SmartContract | Standard;

export enum PullRequestToObjectType {
  Api = 'Api',
  Note = 'Note',
  Project = 'Project',
  Routine = 'Routine',
  SmartContract = 'SmartContract',
  Standard = 'Standard'
}

export type PullRequestUpdateInput = {
  id: Scalars['ID'];
  status?: InputMaybe<PullRequestStatus>;
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
  member?: Maybe<Member>;
  memberInvite?: Maybe<MemberInvite>;
  memberInvites: MemberInviteSearchResult;
  members: MemberSearchResult;
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
  quizAttempt?: Maybe<QuizAttempt>;
  quizAttempts: QuizAttemptSearchResult;
  quizQuestion?: Maybe<QuizQuestion>;
  quizQuestionResponse?: Maybe<QuizQuestionResponse>;
  quizQuestionResponses: QuizQuestionResponseSearchResult;
  quizQuestions: QuizQuestionSearchResult;
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
  role?: Maybe<Role>;
  roles: RoleSearchResult;
  routine?: Maybe<Routine>;
  routineVersion?: Maybe<RoutineVersion>;
  routineVersions: RoutineVersionSearchResult;
  routines: RoutineSearchResult;
  runProject?: Maybe<RunProject>;
  runProjectSchedule?: Maybe<RunProjectSchedule>;
  runProjectSchedules: RunProjectScheduleSearchResult;
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
  votes: VoteSearchResult;
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


export type QueryMemberArgs = {
  input: FindByIdInput;
};


export type QueryMemberInviteArgs = {
  input: FindByIdInput;
};


export type QueryMemberInvitesArgs = {
  input: MemberInviteSearchInput;
};


export type QueryMembersArgs = {
  input: MemberSearchInput;
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


export type QueryQuizAttemptArgs = {
  input: FindByIdInput;
};


export type QueryQuizAttemptsArgs = {
  input: QuizAttemptSearchInput;
};


export type QueryQuizQuestionArgs = {
  input: FindByIdInput;
};


export type QueryQuizQuestionResponseArgs = {
  input: FindByIdInput;
};


export type QueryQuizQuestionResponsesArgs = {
  input: QuizQuestionResponseSearchInput;
};


export type QueryQuizQuestionsArgs = {
  input: QuizQuestionSearchInput;
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


export type QueryRoleArgs = {
  input: FindByIdInput;
};


export type QueryRolesArgs = {
  input: RoleSearchInput;
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


export type QueryVotesArgs = {
  input: VoteSearchInput;
};

export type Question = {
  __typename?: 'Question';
  answers: Array<QuestionAnswer>;
  comments: Array<Comment>;
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  forObject: QuestionFor;
  hasAcceptedAnswer: Scalars['Boolean'];
  id: Scalars['ID'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  score: Scalars['Int'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  translations: Array<QuestionTranslation>;
  updated_at: Scalars['Date'];
};

export type QuestionAnswer = {
  __typename?: 'QuestionAnswer';
  comments: Array<Comment>;
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isAccepted: Scalars['Boolean'];
  question: Question;
  score: Scalars['Int'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  translations: Array<QuestionAnswerTranslation>;
  updated_at: Scalars['Date'];
};

export type QuestionAnswerCreateInput = {
  id: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<QuestionAnswerTranslationCreateInput>>;
};

export type QuestionAnswerEdge = {
  __typename?: 'QuestionAnswerEdge';
  cursor: Scalars['String'];
  node: QuestionAnswer;
};

export type QuestionAnswerSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
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
  ScoreAsc = 'ScoreAsc',
  ScoreDesc = 'ScoreDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc'
}

export type QuestionAnswerTranslation = {
  __typename?: 'QuestionAnswerTranslation';
  description: Scalars['String'];
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type QuestionAnswerTranslationCreateInput = {
  description: Scalars['String'];
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type QuestionAnswerTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
};

export type QuestionAnswerUpdateInput = {
  id: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<QuestionAnswerTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<QuestionAnswerTranslationUpdateInput>>;
};

export type QuestionCreateInput = {
  forConnect: Scalars['ID'];
  forType: QuestionForType;
  id: Scalars['ID'];
  referencing?: InputMaybe<Scalars['String']>;
  translationsCreate?: InputMaybe<Array<QuestionTranslationCreateInput>>;
};

export type QuestionEdge = {
  __typename?: 'QuestionEdge';
  cursor: Scalars['String'];
  node: Question;
};

export type QuestionFor = Api | Note | Organization | Project | Routine | SmartContract | Standard;

export enum QuestionForType {
  Api = 'Api',
  Note = 'Note',
  Organization = 'Organization',
  Project = 'Project',
  Routine = 'Routine',
  SmartContract = 'SmartContract',
  Standard = 'Standard'
}

export type QuestionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  apiId?: InputMaybe<Scalars['ID']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  hasAcceptedAnswer?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  maxScore?: InputMaybe<Scalars['Int']>;
  maxStars?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  noteId?: InputMaybe<Scalars['ID']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  smartContractId?: InputMaybe<Scalars['ID']>;
  sortBy?: InputMaybe<QuestionSortBy>;
  standardId?: InputMaybe<Scalars['ID']>;
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
  ScoreAsc = 'ScoreAsc',
  ScoreDesc = 'ScoreDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc'
}

export type QuestionTranslation = {
  __typename?: 'QuestionTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type QuestionTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type QuestionTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type QuestionUpdateInput = {
  acceptedAnswerConnect?: InputMaybe<Scalars['ID']>;
  id: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<QuestionTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<QuestionTranslationUpdateInput>>;
};

export type Quiz = {
  __typename?: 'Quiz';
  attempts: Array<QuizAttempt>;
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isCompleted: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  project?: Maybe<Project>;
  quizQuestions: Array<QuizQuestion>;
  routine?: Maybe<Routine>;
  score: Scalars['Int'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  stats: Array<StatsQuiz>;
  translations: Array<QuizTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
};

export type QuizAttempt = {
  __typename?: 'QuizAttempt';
  contextSwitches: Scalars['Int'];
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  pointsEarned: Scalars['Int'];
  quiz: Quiz;
  responses: Array<QuizQuestionResponse>;
  status: QuizAttemptStatus;
  timeTaken?: Maybe<Scalars['Int']>;
  updated_at: Scalars['Date'];
};

export type QuizAttemptCreateInput = {
  contextSwitches?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  responsesCreate?: InputMaybe<Array<QuizQuestionResponseCreateInput>>;
  timeTaken?: InputMaybe<Scalars['Int']>;
};

export type QuizAttemptEdge = {
  __typename?: 'QuizAttemptEdge';
  cursor: Scalars['String'];
  node: QuizAttempt;
};

export type QuizAttemptPermission = {
  __typename?: 'QuizAttemptPermission';
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
};

export type QuizAttemptSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languageIn?: InputMaybe<Array<Scalars['String']>>;
  quizId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<QuizAttemptSortBy>;
  status?: InputMaybe<QuizAttemptStatus>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type QuizAttemptSearchResult = {
  __typename?: 'QuizAttemptSearchResult';
  edges: Array<QuizAttemptEdge>;
  pageInfo: PageInfo;
};

export enum QuizAttemptSortBy {
  ContextSwitchesAsc = 'ContextSwitchesAsc',
  ContextSwitchesDesc = 'ContextSwitchesDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  PointsEarnedAsc = 'PointsEarnedAsc',
  PointsEarnedDesc = 'PointsEarnedDesc',
  TimeTakenAsc = 'TimeTakenAsc',
  TimeTakenDesc = 'TimeTakenDesc'
}

export enum QuizAttemptStatus {
  Failed = 'Failed',
  InProgress = 'InProgress',
  NotStarted = 'NotStarted',
  Passed = 'Passed'
}

export type QuizAttemptUpdateInput = {
  contextSwitches?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  responsesCreate?: InputMaybe<Array<QuizQuestionResponseCreateInput>>;
  responsesDelete?: InputMaybe<Array<Scalars['ID']>>;
  responsesUpdate?: InputMaybe<Array<QuizQuestionResponseUpdateInput>>;
  timeTaken?: InputMaybe<Scalars['Int']>;
};

export type QuizCreateInput = {
  id: Scalars['ID'];
  maxAttempts?: InputMaybe<Scalars['Int']>;
  pointsToPass?: InputMaybe<Scalars['Int']>;
  projectConnect?: InputMaybe<Scalars['ID']>;
  quizQuestionsCreate?: InputMaybe<Array<QuizQuestionCreateInput>>;
  randomizeQuestionORder?: InputMaybe<Scalars['Boolean']>;
  revealCorrectAnswers?: InputMaybe<Scalars['Boolean']>;
  routineConnect?: InputMaybe<Scalars['ID']>;
  timeLimit?: InputMaybe<Scalars['Int']>;
  translationsCreate?: InputMaybe<Array<QuizTranslationCreateInput>>;
};

export type QuizEdge = {
  __typename?: 'QuizEdge';
  cursor: Scalars['String'];
  node: Quiz;
};

export type QuizPermission = {
  __typename?: 'QuizPermission';
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type QuizQuestion = {
  __typename?: 'QuizQuestion';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  order?: Maybe<Scalars['Int']>;
  points: Scalars['Int'];
  quiz: Quiz;
  responses?: Maybe<Array<QuizQuestionResponse>>;
  standard?: Maybe<Standard>;
  translations?: Maybe<Array<QuizQuestionTranslation>>;
  updated_at: Scalars['Date'];
};

export type QuizQuestionCreateInput = {
  id: Scalars['ID'];
  order?: InputMaybe<Scalars['Int']>;
  points?: InputMaybe<Scalars['Int']>;
  quizConnect?: InputMaybe<Scalars['ID']>;
  standardConnect?: InputMaybe<Scalars['ID']>;
  standardCreate?: InputMaybe<StandardCreateInput>;
  translationsCreate?: InputMaybe<Array<QuizQuestionTranslationCreateInput>>;
};

export type QuizQuestionEdge = {
  __typename?: 'QuizQuestionEdge';
  cursor: Scalars['String'];
  node: QuizQuestion;
};

export type QuizQuestionPermission = {
  __typename?: 'QuizQuestionPermission';
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
};

export type QuizQuestionResponse = {
  __typename?: 'QuizQuestionResponse';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  permissionsQuizQuestionResponse: QuizQuestionResponsePermission;
  quizAttempt: QuizAttempt;
  quizQuestion: QuizQuestion;
  response?: Maybe<Scalars['String']>;
  updated_at: Scalars['Date'];
};

export type QuizQuestionResponseCreateInput = {
  id: Scalars['ID'];
  quizQuestionConnect: Scalars['ID'];
  response?: InputMaybe<Scalars['String']>;
  translationsCreate?: InputMaybe<Array<QuizQuestionResponseTranslationCreateInput>>;
};

export type QuizQuestionResponseEdge = {
  __typename?: 'QuizQuestionResponseEdge';
  cursor: Scalars['String'];
  node: QuizQuestionResponse;
};

export type QuizQuestionResponsePermission = {
  __typename?: 'QuizQuestionResponsePermission';
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
};

export type QuizQuestionResponseSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  quizAttemptId?: InputMaybe<Scalars['ID']>;
  quizQuestionId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<QuizQuestionResponseSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type QuizQuestionResponseSearchResult = {
  __typename?: 'QuizQuestionResponseSearchResult';
  edges: Array<QuizQuestionResponseEdge>;
  pageInfo: PageInfo;
};

export enum QuizQuestionResponseSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type QuizQuestionResponseTranslation = {
  __typename?: 'QuizQuestionResponseTranslation';
  id: Scalars['ID'];
  language: Scalars['String'];
  response: Scalars['String'];
};

export type QuizQuestionResponseTranslationCreateInput = {
  id: Scalars['ID'];
  language: Scalars['String'];
  response: Scalars['String'];
};

export type QuizQuestionResponseTranslationUpdateInput = {
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  response?: InputMaybe<Scalars['String']>;
};

export type QuizQuestionResponseUpdateInput = {
  id: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<QuizQuestionResponseTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<QuizQuestionResponseTranslationUpdateInput>>;
};

export type QuizQuestionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  quizId?: InputMaybe<Scalars['ID']>;
  responseId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<QuizQuestionSortBy>;
  standardId?: InputMaybe<Scalars['ID']>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type QuizQuestionSearchResult = {
  __typename?: 'QuizQuestionSearchResult';
  edges: Array<QuizQuestionEdge>;
  pageInfo: PageInfo;
};

export enum QuizQuestionSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  OrderAsc = 'OrderAsc',
  OrderDesc = 'OrderDesc'
}

export type QuizQuestionTranslation = {
  __typename?: 'QuizQuestionTranslation';
  helpText?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  questionText: Scalars['String'];
};

export type QuizQuestionTranslationCreateInput = {
  helpText?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  questionText: Scalars['String'];
};

export type QuizQuestionTranslationUpdateInput = {
  helpText?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  questionText?: InputMaybe<Scalars['String']>;
};

export type QuizQuestionUpdateInput = {
  id: Scalars['ID'];
  order?: InputMaybe<Scalars['Int']>;
  points?: InputMaybe<Scalars['Int']>;
  standardConnect?: InputMaybe<Scalars['ID']>;
  standardCreate?: InputMaybe<StandardCreateInput>;
  standardDisconnect?: InputMaybe<Scalars['ID']>;
  standardUpdate?: InputMaybe<StandardUpdateInput>;
  translationsCreate?: InputMaybe<Array<QuizQuestionTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<QuizQuestionTranslationUpdateInput>>;
};

export type QuizSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  projectId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<QuizSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type QuizSearchResult = {
  __typename?: 'QuizSearchResult';
  edges: Array<QuizEdge>;
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
  ScoreAsc = 'ScoreAsc',
  ScoreDesc = 'ScoreDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc'
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
  id: Scalars['ID'];
  maxAttempts?: InputMaybe<Scalars['Int']>;
  pointsToPass?: InputMaybe<Scalars['Int']>;
  projectConnect?: InputMaybe<Scalars['ID']>;
  projectDisconnect?: InputMaybe<Scalars['ID']>;
  quizQuestionsCreate?: InputMaybe<Array<QuizQuestionCreateInput>>;
  quizQuestionsDelete?: InputMaybe<Array<Scalars['ID']>>;
  quizQuestionsUpdate?: InputMaybe<Array<QuizQuestionUpdateInput>>;
  randomizeQuestionORder?: InputMaybe<Scalars['Boolean']>;
  revealCorrectAnswers?: InputMaybe<Scalars['Boolean']>;
  routineConnect?: InputMaybe<Scalars['ID']>;
  routineDisconnect?: InputMaybe<Scalars['ID']>;
  timeLimit?: InputMaybe<Scalars['Int']>;
  translationsCreate?: InputMaybe<Array<QuizTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<QuizTranslationUpdateInput>>;
};

export type ReadAssetsInput = {
  files: Array<Scalars['String']>;
};

export type Reminder = {
  __typename?: 'Reminder';
  completed: Scalars['Boolean'];
  created_at: Scalars['Date'];
  description?: Maybe<Scalars['String']>;
  dueDate?: Maybe<Scalars['Date']>;
  id: Scalars['ID'];
  index: Scalars['Int'];
  name: Scalars['String'];
  reminderItems: Array<ReminderItem>;
  reminderList: ReminderList;
  updated_at: Scalars['Date'];
};

export type ReminderCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  dueDate?: InputMaybe<Scalars['Date']>;
  id: Scalars['ID'];
  index: Scalars['Int'];
  name: Scalars['String'];
  reminderItemsCreate?: InputMaybe<Array<ReminderItemCreateInput>>;
  reminderListConnect: Scalars['ID'];
};

export type ReminderEdge = {
  __typename?: 'ReminderEdge';
  cursor: Scalars['String'];
  node: Reminder;
};

export type ReminderItem = {
  __typename?: 'ReminderItem';
  completed: Scalars['Boolean'];
  created_at: Scalars['Date'];
  description?: Maybe<Scalars['String']>;
  dueDate?: Maybe<Scalars['Date']>;
  id: Scalars['ID'];
  index: Scalars['Int'];
  name: Scalars['String'];
  reminder: Reminder;
  updated_at: Scalars['Date'];
};

export type ReminderItemCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  dueDate?: InputMaybe<Scalars['Date']>;
  id: Scalars['ID'];
  index: Scalars['Int'];
  name: Scalars['String'];
};

export type ReminderItemUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  dueDate?: InputMaybe<Scalars['Date']>;
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  name?: InputMaybe<Scalars['String']>;
};

export type ReminderList = {
  __typename?: 'ReminderList';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  reminders: Array<Reminder>;
  updated_at: Scalars['Date'];
  userSchedule?: Maybe<UserSchedule>;
};

export type ReminderListCreateInput = {
  id: Scalars['ID'];
  remindersCreate?: InputMaybe<Array<ReminderCreateInput>>;
  userScheduleConnect?: InputMaybe<Scalars['ID']>;
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
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ReminderListSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userSchedule?: InputMaybe<Scalars['ID']>;
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

export type ReminderListUpdateInput = {
  id: Scalars['ID'];
  remindersCreate?: InputMaybe<Array<ReminderCreateInput>>;
  remindersDelete?: InputMaybe<Array<Scalars['ID']>>;
  remindersUpdate?: InputMaybe<Array<ReminderUpdateInput>>;
  userScheduleConnect?: InputMaybe<Scalars['ID']>;
};

export type ReminderSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  reminderListId?: InputMaybe<Scalars['ID']>;
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

export type ReminderUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  dueDate?: InputMaybe<Scalars['Date']>;
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  name?: InputMaybe<Scalars['String']>;
  reminderItemsCreate?: InputMaybe<Array<ReminderItemCreateInput>>;
  reminderItemsDelete?: InputMaybe<Array<Scalars['ID']>>;
  reminderItemsUpdate?: InputMaybe<Array<ReminderItemUpdateInput>>;
};

export type Report = {
  __typename?: 'Report';
  details?: Maybe<Scalars['String']>;
  id?: Maybe<Scalars['ID']>;
  isOwn: Scalars['Boolean'];
  language: Scalars['String'];
  reason: Scalars['String'];
  responses: Array<ReportResponse>;
};

export type ReportCreateInput = {
  createdFor: ReportFor;
  createdForConnect: Scalars['ID'];
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
  actionSuggested: ReportSuggestedAction;
  created_at: Scalars['Date'];
  details?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: Maybe<Scalars['String']>;
  permissionsReportResponse: ReportResponsePermission;
  report: Report;
  updated_at: Scalars['Date'];
};

export type ReportResponseCreateInput = {
  actionSuggested: ReportSuggestedAction;
  details?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  reportConnect: Scalars['ID'];
};

export type ReportResponseEdge = {
  __typename?: 'ReportResponseEdge';
  cursor: Scalars['String'];
  node: ReportResponse;
};

export type ReportResponsePermission = {
  __typename?: 'ReportResponsePermission';
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
};

export type ReportResponseSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languageIn?: InputMaybe<Array<Scalars['String']>>;
  reportId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ReportResponseSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ReportResponseSearchResult = {
  __typename?: 'ReportResponseSearchResult';
  edges: Array<ReportResponseEdge>;
  pageInfo: PageInfo;
};

export enum ReportResponseSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc'
}

export type ReportResponseUpdateInput = {
  actionSuggested?: InputMaybe<ReportSuggestedAction>;
  details?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
};

export type ReportSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  apiVersionId?: InputMaybe<Scalars['ID']>;
  commentId?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  fromId?: InputMaybe<Scalars['ID']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  issueId?: InputMaybe<Scalars['ID']>;
  languageIn?: InputMaybe<Array<Scalars['String']>>;
  noteVersionId?: InputMaybe<Scalars['ID']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  postId?: InputMaybe<Scalars['ID']>;
  projectVersionId?: InputMaybe<Scalars['ID']>;
  routineVersionId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  smartContractVersionId?: InputMaybe<Scalars['ID']>;
  sortBy?: InputMaybe<ReportSortBy>;
  standardVersionId?: InputMaybe<Scalars['ID']>;
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

export enum ReportSuggestedAction {
  Delete = 'Delete',
  FalseReport = 'FalseReport',
  HideUntilFixed = 'HideUntilFixed',
  NonIssue = 'NonIssue',
  SuspendUser = 'SuspendUser'
}

export type ReportUpdateInput = {
  details?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  reason?: InputMaybe<Scalars['String']>;
};

export type ReputationHistory = {
  __typename?: 'ReputationHistory';
  amound: Scalars['Int'];
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  updated_at: Scalars['Date'];
};

export type ReputationHistoryEdge = {
  __typename?: 'ReputationHistoryEdge';
  cursor: Scalars['String'];
  node: ReputationHistory;
};

export type ReputationHistorySearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ReputationHistorySortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ReputationHistorySearchResult = {
  __typename?: 'ReputationHistorySearchResult';
  edges: Array<ReputationHistoryEdge>;
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
  listConnect: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<ResourceTranslationCreateInput>>;
  usedFor: ResourceUsedFor;
};

export type ResourceEdge = {
  __typename?: 'ResourceEdge';
  cursor: Scalars['String'];
  node: Resource;
};

export type ResourceList = {
  __typename?: 'ResourceList';
  apiVersion?: Maybe<ApiVersion>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  organization?: Maybe<Organization>;
  post?: Maybe<Post>;
  projectVersion?: Maybe<ProjectVersion>;
  resources: Array<Resource>;
  routineVersion?: Maybe<RoutineVersion>;
  smartContractVersion?: Maybe<SmartContractVersion>;
  standardVersion?: Maybe<StandardVersion>;
  translations: Array<ResourceListTranslation>;
  updated_at: Scalars['Date'];
  userSchedule?: Maybe<UserSchedule>;
};

export type ResourceListCreateInput = {
  apiVersionConnect?: InputMaybe<Scalars['ID']>;
  id: Scalars['ID'];
  organizationConnect?: InputMaybe<Scalars['ID']>;
  postConnect?: InputMaybe<Scalars['ID']>;
  projectVersionConnect?: InputMaybe<Scalars['ID']>;
  resourcesCreate?: InputMaybe<Array<ResourceCreateInput>>;
  routineVersionConnect?: InputMaybe<Scalars['ID']>;
  smartContractVersionConnect?: InputMaybe<Scalars['ID']>;
  standardVersionConnect?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<ResourceListTranslationCreateInput>>;
  userScheduleConnect?: InputMaybe<Scalars['ID']>;
};

export type ResourceListEdge = {
  __typename?: 'ResourceListEdge';
  cursor: Scalars['String'];
  node: ResourceList;
};

export type ResourceListSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  apiVersionId?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  organizationId?: InputMaybe<Scalars['ID']>;
  postId?: InputMaybe<Scalars['ID']>;
  projectVersionId?: InputMaybe<Scalars['ID']>;
  routineVersionId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  smartContractVersionId?: InputMaybe<Scalars['ID']>;
  sortBy?: InputMaybe<ResourceListSortBy>;
  standardVersionId?: InputMaybe<Scalars['ID']>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userScheduleId?: InputMaybe<Scalars['ID']>;
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
  name?: Maybe<Scalars['String']>;
};

export type ResourceListTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
};

export type ResourceListTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type ResourceListUpdateInput = {
  id: Scalars['ID'];
  resourcesCreate?: InputMaybe<Array<ResourceCreateInput>>;
  resourcesDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourcesUpdate?: InputMaybe<Array<ResourceUpdateInput>>;
  translationsCreate?: InputMaybe<Array<ResourceListTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ResourceListTranslationUpdateInput>>;
};

export type ResourceSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  resourceListId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ResourceSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
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
  name?: Maybe<Scalars['String']>;
};

export type ResourceTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
};

export type ResourceTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type ResourceUpdateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  link?: InputMaybe<Scalars['String']>;
  listConnect?: InputMaybe<Scalars['ID']>;
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
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  members?: Maybe<Array<Member>>;
  name: Scalars['String'];
  organization: Organization;
  permissions: Scalars['String'];
  translations: Array<RoleTranslation>;
  updated_at: Scalars['Date'];
};

export type RoleCreateInput = {
  id: Scalars['ID'];
  membersConnect?: InputMaybe<Array<Scalars['ID']>>;
  name: Scalars['String'];
  organizationConnect: Scalars['ID'];
  permissions: Scalars['String'];
  translationsCreate?: InputMaybe<Array<RoleTranslationCreateInput>>;
};

export type RoleEdge = {
  __typename?: 'RoleEdge';
  cursor: Scalars['String'];
  node: Role;
};

export type RoleSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  organizationId: Scalars['ID'];
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RoleSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type RoleSearchResult = {
  __typename?: 'RoleSearchResult';
  edges: Array<RoleEdge>;
  pageInfo: PageInfo;
};

export enum RoleSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  MembersAsc = 'MembersAsc',
  MembersDesc = 'MembersDesc'
}

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
  membersConnect?: InputMaybe<Array<Scalars['ID']>>;
  membersDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  name?: InputMaybe<Scalars['String']>;
  permissions?: InputMaybe<Scalars['String']>;
  translationsCreate?: InputMaybe<Array<RoleTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<RoleTranslationUpdateInput>>;
};

export type RootPermission = {
  __typename?: 'RootPermission';
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canStar: Scalars['Boolean'];
  canTransfer: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type Routine = {
  __typename?: 'Routine';
  completedAt?: Maybe<Scalars['Date']>;
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  forks: Array<Routine>;
  forksCount: Scalars['Int'];
  hasCompletedVersion: Scalars['Boolean'];
  id: Scalars['ID'];
  issues: any
        issuesCount: any
  isDeleted: Scalars['Boolean'];
  isInternal?: Maybe<Scalars['Boolean']>;
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  labels: Array<Label>;
  owner?: Maybe<Owner>;
  parent?: Maybe<Routine>;
  permissionsRoutine: RoutinePermission;
  score: Scalars['Int'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  updated_at: Scalars['Date'];
  versions: Array<RoutineVersion>;
  versionsCount?: Maybe<Scalars['Int']>;
  views: Scalars['Int'];
};

export type RoutineCreateInput = {
  id: Scalars['ID'];
  isInternal?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  organizationConnect?: InputMaybe<Scalars['ID']>;
  parentConnect?: InputMaybe<Scalars['ID']>;
  permissions: Scalars['String'];
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  userConnect?: InputMaybe<Scalars['ID']>;
  versionsCreate?: InputMaybe<Array<RoutineVersionCreateInput>>;
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
  canStar: Scalars['Boolean'];
  canView: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
};

export type RoutineSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  hasCompleteVersion?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isInternal?: InputMaybe<Scalars['Boolean']>;
  labelsId?: InputMaybe<Scalars['ID']>;
  maxScore?: InputMaybe<Scalars['Int']>;
  maxStars?: InputMaybe<Scalars['Int']>;
  maxViews?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  ownedByOrganizationId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RoutineSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguagesLatestVersion?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
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
  ScoreAsc = 'ScoreAsc',
  ScoreDesc = 'ScoreDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc'
}

export type RoutineUpdateInput = {
  isInternal?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  labelsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  organizationConnect?: InputMaybe<Scalars['ID']>;
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  userConnect?: InputMaybe<Scalars['ID']>;
  versionsCreate?: InputMaybe<Array<RoutineVersionCreateInput>>;
  versionsDelete?: InputMaybe<Array<Scalars['ID']>>;
  versionsUpdate?: InputMaybe<Array<RoutineVersionUpdateInput>>;
};

export type RoutineVersion = {
  __typename?: 'RoutineVersion';
  directoryListings?: any;
  api?: Maybe<Api>;
  apiCallData?: Maybe<Scalars['String']>;
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  complexity: Scalars['Int'];
  created_at: Scalars['Date'];
  forks: Array<Routine>;
  forksCount: Scalars['Int'];
  id: Scalars['ID'];
  inputs: Array<RoutineVersionInput>;
  inputsCount: Scalars['Int'];
  isAutomatable?: Maybe<Scalars['Boolean']>;
  isComplete: Scalars['Boolean'];
  isDeleted: Scalars['Boolean'];
  isInternal?: Maybe<Scalars['Boolean']>;
  isLatest: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  nodeLinks: Array<NodeLink>;
  nodeLinksCount: Scalars['Int'];
  nodes: Array<Node>;
  nodesCount: Scalars['Int'];
  outputs: Array<RoutineVersionOutput>;
  outputsCount: Scalars['Int'];
  permissions: RoutineVersionPermission;
  pullRequest?: Maybe<PullRequest>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceList?: Maybe<ResourceList>;
  root: Routine;
  runs: Array<RunRoutine>;
  simplicity: Scalars['Int'];
  smartContract?: Maybe<SmartContract>;
  smartContractCallData?: Maybe<Scalars['String']>;
  suggestedNextByRoutineVersion: Array<RoutineVersion>;
  suggestedNextByRoutineVersionCount: Scalars['Int'];
  timesCompleted: Scalars['Int'];
  timesStarted: Scalars['Int'];
  translations: Array<RoutineVersionTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
};

export type RoutineVersionCreateInput = {
  apiCallData?: InputMaybe<Scalars['String']>;
  apiVersionConnect?: InputMaybe<Scalars['ID']>;
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  inputsCreate?: InputMaybe<Array<RoutineVersionInputCreateInput>>;
  isAutomatable?: InputMaybe<Scalars['Boolean']>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isInternal?: InputMaybe<Scalars['Boolean']>;
  isLatest?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  nodeLinksCreate?: InputMaybe<Array<NodeLinkCreateInput>>;
  nodesCreate?: InputMaybe<Array<NodeCreateInput>>;
  outputsCreate?: InputMaybe<Array<RoutineVersionOutputCreateInput>>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  rootConnect: Scalars['ID'];
  rootCreate?: InputMaybe<RoutineCreateInput>;
  smartContractCallData?: InputMaybe<Scalars['String']>;
  smartContractVersionConnect?: InputMaybe<Scalars['ID']>;
  suggestedNextByRoutineVersionConnect?: InputMaybe<Array<Scalars['ID']>>;
  versionIndex: Scalars['Int'];
  versionLabel: Scalars['String'];
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type RoutineVersionEdge = {
  __typename?: 'RoutineVersionEdge';
  cursor: Scalars['String'];
  node: RoutineVersion;
};

export type RoutineVersionInput = {
  __typename?: 'RoutineVersionInput';
  id: Scalars['ID'];
  index?: Maybe<Scalars['Int']>;
  isRequired?: Maybe<Scalars['Boolean']>;
  name?: Maybe<Scalars['String']>;
  routineVersion: RoutineVersion;
  standardVersion?: Maybe<StandardVersion>;
  translations: Array<RoutineVersionInputTranslation>;
};

export type RoutineVersionInputCreateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  isRequired?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  routineVersionConnect: Scalars['ID'];
  standardVersionConnect?: InputMaybe<Scalars['ID']>;
  standardVersionCreate?: InputMaybe<StandardVersionCreateInput>;
  translationsCreate?: InputMaybe<Array<RoutineVersionInputTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<RoutineVersionInputTranslationUpdateInput>>;
};

export type RoutineVersionInputTranslation = {
  __typename?: 'RoutineVersionInputTranslation';
  description?: Maybe<Scalars['String']>;
  helpText?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type RoutineVersionInputTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  helpText?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type RoutineVersionInputTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  helpText?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
};

export type RoutineVersionInputUpdateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  isRequired?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  standardVersionConnect?: InputMaybe<Scalars['ID']>;
  standardVersionCreate?: InputMaybe<StandardVersionCreateInput>;
  standardVersionDisconnect?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<RoutineVersionInputTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<RoutineVersionInputTranslationUpdateInput>>;
};

export type RoutineVersionOutput = {
  __typename?: 'RoutineVersionOutput';
  id: Scalars['ID'];
  index?: Maybe<Scalars['Int']>;
  isRequired?: Maybe<Scalars['Boolean']>;
  name?: Maybe<Scalars['String']>;
  routineVersion: RoutineVersion;
  standardVersion?: Maybe<StandardVersion>;
  translations: Array<RoutineVersionOutputTranslation>;
};

export type RoutineVersionOutputCreateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  isRequired?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  routineVersionConnect: Scalars['ID'];
  standardVersionConnect?: InputMaybe<Scalars['ID']>;
  standardVersionCreate?: InputMaybe<StandardVersionCreateInput>;
  translationsCreate?: InputMaybe<Array<RoutineVersionOutputTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<RoutineVersionOutputTranslationUpdateInput>>;
};

export type RoutineVersionOutputTranslation = {
  __typename?: 'RoutineVersionOutputTranslation';
  description?: Maybe<Scalars['String']>;
  helpText?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type RoutineVersionOutputTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  helpText?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type RoutineVersionOutputTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  helpText?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
};

export type RoutineVersionOutputUpdateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  isRequired?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  standardVersionConnect?: InputMaybe<Scalars['ID']>;
  standardVersionCreate?: InputMaybe<StandardVersionCreateInput>;
  standardVersionDisconnect?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<RoutineVersionOutputTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<RoutineVersionOutputTranslationUpdateInput>>;
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
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  directoryListingsId?: InputMaybe<Scalars['ID']>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isCompleteWithRoot?: InputMaybe<Scalars['Boolean']>;
  isCompleteWithRootExcludeOwnedByOrganizationId?: InputMaybe<Scalars['ID']>;
  isCompleteWithRootExcludeOwnedByUserId?: InputMaybe<Scalars['ID']>;
  isInternalWithRoot?: InputMaybe<Scalars['Boolean']>;
  isInternalWithRootExcludeOwnedByOrganizationId?: InputMaybe<Scalars['ID']>;
  isInternalWithRootExcludeOwnedByUserId?: InputMaybe<Scalars['ID']>;
  maxComplexity?: InputMaybe<Scalars['Int']>;
  maxSimplicity?: InputMaybe<Scalars['Int']>;
  maxTimesCompleted?: InputMaybe<Scalars['Int']>;
  minComplexity?: InputMaybe<Scalars['Int']>;
  minScoreRoot?: InputMaybe<Scalars['Int']>;
  minSimplicity?: InputMaybe<Scalars['Int']>;
  minStarsRoot?: InputMaybe<Scalars['Int']>;
  minTimesCompleted?: InputMaybe<Scalars['Int']>;
  minViewsRoot?: InputMaybe<Scalars['Int']>;
  ownedByOrganizationId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  rootId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RoutineVersionSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type RoutineVersionSearchResult = {
  __typename?: 'RoutineVersionSearchResult';
  edges: Array<RoutineVersionEdge>;
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
  name: Scalars['String'];
};

export type RoutineVersionTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  instructions: Scalars['String'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type RoutineVersionTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  instructions?: InputMaybe<Scalars['String']>;
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type RoutineVersionUpdateInput = {
  apiCallData?: InputMaybe<Scalars['String']>;
  apiVersionConnect?: InputMaybe<Scalars['ID']>;
  apiVersionDisconnect?: InputMaybe<Scalars['ID']>;
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  directoryListingsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  inputsCreate?: InputMaybe<Array<RoutineVersionInputCreateInput>>;
  inputsDelete?: InputMaybe<Array<Scalars['ID']>>;
  inputsUpdate?: InputMaybe<Array<RoutineVersionInputUpdateInput>>;
  isAutomatable?: InputMaybe<Scalars['Boolean']>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isInternal?: InputMaybe<Scalars['Boolean']>;
  isLatest?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  nodeLinksCreate?: InputMaybe<Array<NodeLinkCreateInput>>;
  nodeLinksDelete?: InputMaybe<Array<Scalars['ID']>>;
  nodeLinksUpdate?: InputMaybe<Array<NodeLinkUpdateInput>>;
  nodesCreate?: InputMaybe<Array<NodeCreateInput>>;
  nodesDelete?: InputMaybe<Array<Scalars['ID']>>;
  nodesUpdate?: InputMaybe<Array<NodeUpdateInput>>;
  outputsCreate?: InputMaybe<Array<RoutineVersionOutputCreateInput>>;
  outputsDelete?: InputMaybe<Array<Scalars['ID']>>;
  outputsUpdate?: InputMaybe<Array<RoutineVersionOutputUpdateInput>>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
  smartContractCallData?: InputMaybe<Scalars['String']>;
  smartContractVersionConnect?: InputMaybe<Scalars['ID']>;
  smartContractVersionDisconnect?: InputMaybe<Scalars['ID']>;
  suggestedNextByRoutineVersionConnect?: InputMaybe<Array<Scalars['ID']>>;
  suggestedNextByRoutineVersionDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  versionIndex: Scalars['Int'];
  versionLabel?: InputMaybe<Scalars['String']>;
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type RunProject = {
  __typename?: 'RunProject';
  completedAt?: Maybe<Scalars['Date']>;
  completedComplexity: Scalars['Int'];
  contextSwitches: Scalars['Int'];
  id: Scalars['ID'];
  isPrivate: Scalars['Boolean'];
  name: Scalars['String'];
  organization?: Maybe<Organization>;
  permissionsRun: RunProjectPermission;
  projectVersion?: Maybe<ProjectVersion>;
  runProjectSchedule?: Maybe<RunProjectSchedule>;
  startedAt?: Maybe<Scalars['Date']>;
  status: RunStatus;
  steps: Array<RunProjectStep>;
  timeElapsed?: Maybe<Scalars['Int']>;
  user?: Maybe<User>;
  wasRunAutomaticaly: Scalars['Boolean'];
};

export type RunProjectCancelInput = {
  id: Scalars['ID'];
};

export type RunProjectCompleteInput = {
  completedComplexity?: InputMaybe<Scalars['Int']>;
  exists?: InputMaybe<Scalars['Boolean']>;
  finalStepCreate?: InputMaybe<RunProjectStepCreateInput>;
  finalStepUpdate?: InputMaybe<RunProjectStepUpdateInput>;
  id: Scalars['ID'];
  name?: InputMaybe<Scalars['String']>;
  wasSuccessful?: InputMaybe<Scalars['Boolean']>;
};

export type RunProjectCreateInput = {
  completedComplexity?: InputMaybe<Scalars['Int']>;
  contextSwitches?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  name: Scalars['String'];
  organizationId?: InputMaybe<Scalars['ID']>;
  projectVersionConnect: Scalars['ID'];
  runProjectScheduleConnect?: InputMaybe<Scalars['ID']>;
  runProjectScheduleCreate?: InputMaybe<RunProjectScheduleCreateInput>;
  status: RunStatus;
  stepsCreate?: InputMaybe<Array<RunProjectStepCreateInput>>;
};

export type RunProjectEdge = {
  __typename?: 'RunProjectEdge';
  cursor: Scalars['String'];
  node: RunProject;
};

export type RunProjectPermission = {
  __typename?: 'RunProjectPermission';
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canView: Scalars['Boolean'];
};

export type RunProjectSchedule = {
  __typename?: 'RunProjectSchedule';
  id: Scalars['ID'];
  labels: Array<Label>;
  recurrEnd?: Maybe<Scalars['Date']>;
  recurrStart?: Maybe<Scalars['Date']>;
  recurring: Scalars['Boolean'];
  runProject: RunProject;
  timeZone?: Maybe<Scalars['String']>;
  translations: Array<RunProjectScheduleTranslation>;
  windowEnd?: Maybe<Scalars['Date']>;
  windowStart?: Maybe<Scalars['Date']>;
};

export type RunProjectScheduleCreateInput = {
  id: Scalars['ID'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  recurrEnd?: InputMaybe<Scalars['Date']>;
  recurrStart?: InputMaybe<Scalars['Date']>;
  recurring?: InputMaybe<Scalars['Boolean']>;
  runProjectConnect: Scalars['ID'];
  timeZone?: InputMaybe<Scalars['String']>;
  translationsCreate?: InputMaybe<Array<RunProjectScheduleTranslationCreateInput>>;
  windowEnd?: InputMaybe<Scalars['Date']>;
  windowStart?: InputMaybe<Scalars['Date']>;
};

export type RunProjectScheduleEdge = {
  __typename?: 'RunProjectScheduleEdge';
  cursor: Scalars['String'];
  node: RunProjectSchedule;
};

export type RunProjectScheduleSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  labelsId?: InputMaybe<Array<Scalars['ID']>>;
  maxEventEnd?: InputMaybe<Scalars['Date']>;
  maxEventStart?: InputMaybe<Scalars['Date']>;
  maxRecurrEnd?: InputMaybe<Scalars['Date']>;
  maxRecurrStart?: InputMaybe<Scalars['Date']>;
  minEventEnd?: InputMaybe<Scalars['Date']>;
  minEventStart?: InputMaybe<Scalars['Date']>;
  minRecurrEnd?: InputMaybe<Scalars['Date']>;
  minRecurrStart?: InputMaybe<Scalars['Date']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RunProjectScheduleSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type RunProjectScheduleSearchResult = {
  __typename?: 'RunProjectScheduleSearchResult';
  edges: Array<RunProjectScheduleEdge>;
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
  id: Scalars['ID'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  labelsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  recurrEnd?: InputMaybe<Scalars['Date']>;
  recurrStart?: InputMaybe<Scalars['Date']>;
  recurring?: InputMaybe<Scalars['Boolean']>;
  runProjectConnect: Scalars['ID'];
  timeZone?: InputMaybe<Scalars['String']>;
  translationsCreate?: InputMaybe<Array<RunProjectScheduleTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<RunProjectScheduleTranslationUpdateInput>>;
  windowEnd?: InputMaybe<Scalars['Date']>;
  windowStart?: InputMaybe<Scalars['Date']>;
};

export type RunProjectSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  completedTimeFrame?: InputMaybe<TimeFrame>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RunProjectSortBy>;
  startedTimeFrame?: InputMaybe<TimeFrame>;
  status?: InputMaybe<RunStatus>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type RunProjectSearchResult = {
  __typename?: 'RunProjectSearchResult';
  edges: Array<RunProjectEdge>;
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
  completedAt?: Maybe<Scalars['Date']>;
  contextSwitches: Scalars['Int'];
  directory?: Maybe<ProjectVersionDirectory>;
  id: Scalars['ID'];
  name: Scalars['String'];
  node?: Maybe<Node>;
  order: Scalars['Int'];
  run: RunProject;
  startedAt?: Maybe<Scalars['Date']>;
  status: RunProjectStepStatus;
  step: Array<Scalars['Int']>;
  timeElapsed?: Maybe<Scalars['Int']>;
};

export type RunProjectStepCreateInput = {
  contextSwitches?: InputMaybe<Scalars['Int']>;
  directoryConnect?: InputMaybe<Scalars['ID']>;
  id: Scalars['ID'];
  name: Scalars['String'];
  nodeConnect?: InputMaybe<Scalars['ID']>;
  order: Scalars['Int'];
  step: Array<Scalars['Int']>;
  timeElapsed?: InputMaybe<Scalars['Int']>;
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

export enum RunProjectStepStatus {
  Completed = 'Completed',
  InProgress = 'InProgress',
  Skipped = 'Skipped'
}

export type RunProjectStepUpdateInput = {
  contextSwitches?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  status?: InputMaybe<RunProjectStepStatus>;
  timeElapsed?: InputMaybe<Scalars['Int']>;
};

export type RunProjectUpdateInput = {
  completedComplexity?: InputMaybe<Scalars['Int']>;
  contextSwitches?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  isStarted?: InputMaybe<Scalars['Boolean']>;
  runProjectScheduleConnect?: InputMaybe<Scalars['ID']>;
  runProjectScheduleCreate?: InputMaybe<RunProjectScheduleCreateInput>;
  stepsCreate?: InputMaybe<Array<RunProjectStepCreateInput>>;
  stepsDelete?: InputMaybe<Array<Scalars['ID']>>;
  stepsUpdate?: InputMaybe<Array<RunProjectStepUpdateInput>>;
  timeElapsed?: InputMaybe<Scalars['Int']>;
};

export type RunRoutine = {
  __typename?: 'RunRoutine';
  completedAt?: Maybe<Scalars['Date']>;
  completedComplexity: Scalars['Int'];
  contextSwitches: Scalars['Int'];
  id: Scalars['ID'];
  inputs: Array<RunRoutineInput>;
  isPrivate: Scalars['Boolean'];
  name: Scalars['String'];
  organization?: Maybe<Organization>;
  permissionsRun: RunRoutinePermission;
  routineVersion?: Maybe<RoutineVersion>;
  runProject?: Maybe<RunProject>;
  runRoutineSchedule?: Maybe<RunRoutineSchedule>;
  startedAt?: Maybe<Scalars['Date']>;
  status: RunStatus;
  steps: Array<RunRoutineStep>;
  timeElapsed?: Maybe<Scalars['Int']>;
  user?: Maybe<User>;
  wasRunAutomaticaly: Scalars['Boolean'];
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
  name?: InputMaybe<Scalars['String']>;
  wasSuccessful?: InputMaybe<Scalars['Boolean']>;
};

export type RunRoutineCreateInput = {
  completedComplexity?: InputMaybe<Scalars['Int']>;
  contextSwitches?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  inputsCreate?: InputMaybe<Array<RunRoutineInputCreateInput>>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  name: Scalars['String'];
  organizationId?: InputMaybe<Scalars['ID']>;
  routineVersionConnect: Scalars['ID'];
  runProjectConnect?: InputMaybe<Scalars['ID']>;
  runRoutineScheduleConnect?: InputMaybe<Scalars['ID']>;
  runRoutineScheduleCreate?: InputMaybe<RunProjectScheduleCreateInput>;
  status: RunStatus;
  stepsCreate?: InputMaybe<Array<RunRoutineStepCreateInput>>;
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
  input: RoutineVersionInput;
};

export type RunRoutineInputCreateInput = {
  data: Scalars['String'];
  id: Scalars['ID'];
  inputConnect: Scalars['ID'];
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
  id: Scalars['ID'];
  labels: Array<Label>;
  recurrEnd?: Maybe<Scalars['Date']>;
  recurrStart?: Maybe<Scalars['Date']>;
  recurring: Scalars['Boolean'];
  runRoutine: RunRoutine;
  timeZone?: Maybe<Scalars['String']>;
  translations: Array<RunRoutineScheduleTranslation>;
  windowEnd?: Maybe<Scalars['Date']>;
  windowStart?: Maybe<Scalars['Date']>;
};

export type RunRoutineScheduleCreateInput = {
  id: Scalars['ID'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  recurrEnd?: InputMaybe<Scalars['Date']>;
  recurrStart?: InputMaybe<Scalars['Date']>;
  recurring?: InputMaybe<Scalars['Boolean']>;
  runRoutineConnect: Scalars['ID'];
  timeZone?: InputMaybe<Scalars['String']>;
  translationsCreate?: InputMaybe<Array<RunRoutineScheduleTranslationCreateInput>>;
  windowEnd?: InputMaybe<Scalars['Date']>;
  windowStart?: InputMaybe<Scalars['Date']>;
};

export type RunRoutineScheduleEdge = {
  __typename?: 'RunRoutineScheduleEdge';
  cursor: Scalars['String'];
  node: RunRoutineSchedule;
};

export type RunRoutineScheduleSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  labelsId?: InputMaybe<Array<Scalars['ID']>>;
  maxEventEnd?: InputMaybe<Scalars['Date']>;
  maxEventStart?: InputMaybe<Scalars['Date']>;
  maxRecurrEnd?: InputMaybe<Scalars['Date']>;
  maxRecurrStart?: InputMaybe<Scalars['Date']>;
  minEventEnd?: InputMaybe<Scalars['Date']>;
  minEventStart?: InputMaybe<Scalars['Date']>;
  minRecurrEnd?: InputMaybe<Scalars['Date']>;
  minRecurrStart?: InputMaybe<Scalars['Date']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RunRoutineScheduleSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type RunRoutineScheduleSearchResult = {
  __typename?: 'RunRoutineScheduleSearchResult';
  edges: Array<RunRoutineScheduleEdge>;
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
  id: Scalars['ID'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  labelsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  recurrEnd?: InputMaybe<Scalars['Date']>;
  recurrStart?: InputMaybe<Scalars['Date']>;
  recurring?: InputMaybe<Scalars['Boolean']>;
  runRoutineConnect: Scalars['ID'];
  timeZone?: InputMaybe<Scalars['String']>;
  translationsCreate?: InputMaybe<Array<RunRoutineScheduleTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<RunRoutineScheduleTranslationUpdateInput>>;
  windowEnd?: InputMaybe<Scalars['Date']>;
  windowStart?: InputMaybe<Scalars['Date']>;
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
  completedAt?: Maybe<Scalars['Date']>;
  contextSwitches: Scalars['Int'];
  id: Scalars['ID'];
  name: Scalars['String'];
  node?: Maybe<Node>;
  order: Scalars['Int'];
  run: RunRoutine;
  startedAt?: Maybe<Scalars['Date']>;
  status: RunRoutineStepStatus;
  step: Array<Scalars['Int']>;
  subroutine?: Maybe<Routine>;
  timeElapsed?: Maybe<Scalars['Int']>;
};

export type RunRoutineStepCreateInput = {
  contextSwitches?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  name: Scalars['String'];
  nodeConnect?: InputMaybe<Scalars['ID']>;
  order: Scalars['Int'];
  step: Array<Scalars['Int']>;
  subroutineVersionConnect?: InputMaybe<Scalars['ID']>;
  timeElapsed?: InputMaybe<Scalars['Int']>;
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
  isStarted?: InputMaybe<Scalars['Boolean']>;
  runRoutineScheduleConnect?: InputMaybe<Scalars['ID']>;
  runRoutineScheduleCreate?: InputMaybe<RunProjectScheduleCreateInput>;
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
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Owner>;
  parent?: Maybe<Project>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type SmartContractCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentConnect?: InputMaybe<Scalars['ID']>;
  resourceListCreate: ResourceListCreateInput;
  rootConnect: Scalars['ID'];
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
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
  ScoreAsc = 'ScoreAsc',
  ScoreDesc = 'ScoreDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc'
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
  organizationConnect?: InputMaybe<Scalars['ID']>;
  resourceListCreate: ResourceListCreateInput;
  resourceListDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListUpdate: ResourceListUpdateInput;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  userConnect?: InputMaybe<Scalars['ID']>;
};

export type SmartContractVersion = {
  __typename?: 'SmartContractVersion';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  forks: Array<Project>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Owner>;
  parent?: Maybe<Project>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  root: SmartContract;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets?: Maybe<Array<Wallet>>;
};

export type SmartContractVersionCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  parentConnect?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootConnect?: InputMaybe<Scalars['ID']>;
  rootCreate?: InputMaybe<SmartContractCreateInput>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
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
  CalledByRoutinesAsc = 'CalledByRoutinesAsc',
  CalledByRoutinesDesc = 'CalledByRoutinesDesc',
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
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
};

export type Standard = {
  __typename?: 'Standard';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  default?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isDeleted: Scalars['Boolean'];
  isInternal: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  name: Scalars['String'];
  owner?: Maybe<Owner>;
  permissionsRoot: RootPermission;
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
  default?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isInternal?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  organizationConnect?: InputMaybe<Scalars['ID']>;
  props: Scalars['String'];
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<StandardTranslationCreateInput>>;
  type: Scalars['String'];
  userConnect?: InputMaybe<Scalars['ID']>;
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
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  issuesId?: InputMaybe<Scalars['ID']>;
  labelsId?: InputMaybe<Scalars['ID']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  ownedByOrganizationId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  pullRequestsId?: InputMaybe<Scalars['ID']>;
  questionsId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StandardSortBy>;
  standardTypeLatestVersion?: InputMaybe<Scalars['String']>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  transfersId?: InputMaybe<Scalars['ID']>;
  translationLanguagesLatestVersion?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
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
  ScoreAsc = 'ScoreAsc',
  ScoreDesc = 'ScoreDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc',
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc'
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
  organizationConnect?: InputMaybe<Scalars['ID']>;
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
  userConnect?: InputMaybe<Scalars['ID']>;
  versionConnect?: InputMaybe<Scalars['ID']>;
  versionLabel?: InputMaybe<Scalars['String']>;
  yup?: InputMaybe<Scalars['String']>;
};

export type StandardVersion = {
  __typename?: 'StandardVersion';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  created_at: Scalars['Date'];
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
  root: Standard;
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
} & { [x: string]: any };

export type StandardVersionCreateInput = {
  default?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isInternal?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  props: Scalars['String'];
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  rootConnect?: InputMaybe<Scalars['ID']>;
  rootCreate?: InputMaybe<StandardCreateInput>;
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
  reportId?: InputMaybe<Scalars['ID']>;
  rootId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StandardVersionSortBy>;
  standardType?: InputMaybe<Scalars['String']>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
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
  versionConnect?: InputMaybe<Scalars['ID']>;
  versionLabel?: InputMaybe<Scalars['String']>;
  yup?: InputMaybe<Scalars['String']>;
};

export type Star = {
  __typename?: 'Star';
  by: User;
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
  forConnect: Scalars['ID'];
  isStar: Scalars['Boolean'];
  starFor: StarFor;
};

export type StarSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  excludeLinkedToTag?: InputMaybe<Scalars['Boolean']>;
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

export type StatsNote = {
  __typename?: 'StatsNote';
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

export enum SubscribableObject {
  Api = 'Api',
  Comment = 'Comment',
  Issue = 'Issue',
  Meeting = 'Meeting',
  Note = 'Note',
  Organization = 'Organization',
  Project = 'Project',
  PullRequest = 'PullRequest',
  Question = 'Question',
  Quiz = 'Quiz',
  Report = 'Report',
  Routine = 'Routine',
  SmartContract = 'SmartContract',
  Standard = 'Standard'
}

export type SubscribedObject = Api | Comment | Issue | Meeting | Note | Organization | Project | PullRequest | Question | Quiz | Report | Routine | SmartContract | Standard;

export type Success = {
  __typename?: 'Success';
  success: Scalars['Boolean'];
};

export type SwitchCurrentAccountInput = {
  id: Scalars['ID'];
};

export type Tag = {
  __typename?: 'Tag';
  apis: Array<Api>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isOwn: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  notes: Array<Note>;
  organizations: Array<Organization>;
  posts: Array<Post>;
  projects: Array<Project>;
  reports: Array<Report>;
  routines: Array<Routine>;
  smartContracts: Array<SmartContract>;
  standards: Array<Standard>;
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

export type TagSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  maxStars?: InputMaybe<Scalars['Int']>;
  minStars?: InputMaybe<Scalars['Int']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<TagSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
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
  created_at: Scalars['Date'];
  fromOwner?: Maybe<Owner>;
  id: Scalars['ID'];
  mergedOrRejectedAt?: Maybe<Scalars['Date']>;
  object: TransferObject;
  status: TransferStatus;
  toOwner?: Maybe<Owner>;
  updated_at: Scalars['Date'];
};

export type TransferDenyInput = {
  id: Scalars['ID'];
  reason?: InputMaybe<Scalars['String']>;
};

export type TransferEdge = {
  __typename?: 'TransferEdge';
  cursor: Scalars['String'];
  node: Transfer;
};

export type TransferObject = Api | Note | Project | Routine | SmartContract | Standard;

export enum TransferObjectType {
  Api = 'Api',
  Note = 'Note',
  Project = 'Project',
  Routine = 'Routine',
  SmartContract = 'SmartContract',
  Standard = 'Standard'
}

export type TransferPermission = {
  __typename?: 'TransferPermission';
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
};

export type TransferRequestReceiveInput = {
  id: Scalars['ID'];
  message?: InputMaybe<Scalars['String']>;
  objectConnect: Scalars['ID'];
  objectType: TransferObjectType;
  toOrganizationConnect?: InputMaybe<Scalars['ID']>;
};

export type TransferRequestSendInput = {
  id: Scalars['ID'];
  message?: InputMaybe<Scalars['String']>;
  objectConnect: Scalars['ID'];
  objectType: TransferObjectType;
  toOrganizationConnect?: InputMaybe<Scalars['ID']>;
  toUserConnect?: InputMaybe<Scalars['ID']>;
};

export type TransferSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  apiId?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  fromOrganizationId?: InputMaybe<Scalars['ID']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  noteId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  smartContractId?: InputMaybe<Scalars['ID']>;
  sortBy?: InputMaybe<TransferSortBy>;
  standardId?: InputMaybe<Scalars['ID']>;
  status?: InputMaybe<TransferStatus>;
  take?: InputMaybe<Scalars['Int']>;
  toOrganizationId?: InputMaybe<Scalars['ID']>;
  toUserId?: InputMaybe<Scalars['ID']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type TransferSearchResult = {
  __typename?: 'TransferSearchResult';
  edges: Array<TransferEdge>;
  pageInfo: PageInfo;
};

export enum TransferSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export enum TransferStatus {
  Accepted = 'Accepted',
  Denied = 'Denied',
  Pending = 'Pending'
}

export type TransferUpdateInput = {
  id: Scalars['ID'];
  message?: InputMaybe<Scalars['String']>;
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
  membershipsInvited?: Maybe<Array<MemberInvite>>;
  name: Scalars['String'];
  notes?: Maybe<Array<Note>>;
  notesCreated?: Maybe<Array<Note>>;
  notificationSettings?: Maybe<Scalars['String']>;
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

export type UserSchedule = {
  __typename?: 'UserSchedule';
  created_at: Scalars['Date'];
  description?: Maybe<Scalars['String']>;
  eventEnd?: Maybe<Scalars['Date']>;
  eventStart?: Maybe<Scalars['Date']>;
  filters: Array<UserScheduleFilter>;
  id: Scalars['ID'];
  labels: Array<Label>;
  name: Scalars['String'];
  recurrEnd?: Maybe<Scalars['Date']>;
  recurrStart?: Maybe<Scalars['Date']>;
  recurring: Scalars['Boolean'];
  reminderList?: Maybe<ReminderList>;
  timeZone?: Maybe<Scalars['String']>;
  updated_at: Scalars['Date'];
};

export type UserScheduleCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  eventEnd?: InputMaybe<Scalars['Date']>;
  eventStart?: InputMaybe<Scalars['Date']>;
  filtersCreate?: InputMaybe<Array<UserScheduleFilterCreateInput>>;
  id: Scalars['ID'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  name: Scalars['String'];
  recurrEnd?: InputMaybe<Scalars['Date']>;
  recurrStart?: InputMaybe<Scalars['Date']>;
  recurring?: InputMaybe<Scalars['Boolean']>;
  reminderListConnect?: InputMaybe<Scalars['ID']>;
  reminderListCreate?: InputMaybe<ReminderListCreateInput>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  timeZone?: InputMaybe<Scalars['String']>;
};

export type UserScheduleEdge = {
  __typename?: 'UserScheduleEdge';
  cursor: Scalars['String'];
  node: UserSchedule;
};

export type UserScheduleFilter = {
  __typename?: 'UserScheduleFilter';
  filterType: UserScheduleFilterType;
  id: Scalars['ID'];
  tag?: Maybe<Tag>;
  userSchedule: UserSchedule;
};

export type UserScheduleFilterCreateInput = {
  filterType: UserScheduleFilterType;
  id: Scalars['ID'];
  tagConnect?: InputMaybe<Scalars['ID']>;
  tagCreate?: InputMaybe<TagCreateInput>;
  userScheduleConnect: Scalars['ID'];
};

export enum UserScheduleFilterType {
  Blur = 'Blur',
  Hide = 'Hide',
  ShowMore = 'ShowMore'
}

export type UserScheduleSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  eventEndTimeFrame?: InputMaybe<TimeFrame>;
  eventStartTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  recurrEndTimeFrame?: InputMaybe<TimeFrame>;
  recurrStartTimeFrame?: InputMaybe<TimeFrame>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<UserScheduleSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  timeZone?: InputMaybe<Scalars['String']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type UserScheduleSearchResult = {
  __typename?: 'UserScheduleSearchResult';
  edges: Array<UserScheduleEdge>;
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

export type UserScheduleUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  eventEnd?: InputMaybe<Scalars['Date']>;
  eventStart?: InputMaybe<Scalars['Date']>;
  filtersCreate?: InputMaybe<Array<UserScheduleFilterCreateInput>>;
  filtersDelete?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  labelsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  name?: InputMaybe<Scalars['String']>;
  recurrEnd?: InputMaybe<Scalars['Date']>;
  recurrStart?: InputMaybe<Scalars['Date']>;
  recurring?: InputMaybe<Scalars['Boolean']>;
  reminderListConnect?: InputMaybe<Scalars['ID']>;
  reminderListCreate?: InputMaybe<ReminderListCreateInput>;
  reminderListDisconnect?: InputMaybe<Scalars['ID']>;
  reminderListUpdate?: InputMaybe<ReminderListUpdateInput>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
  timeZone?: InputMaybe<Scalars['String']>;
};

export type UserSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  minStars?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<UserSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
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

export type VersionPermission = {
  __typename?: 'VersionPermission';
  canComment: Scalars['Boolean'];
  canCopy: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canEdit: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canUse: Scalars['Boolean'];
  canView: Scalars['Boolean'];
};

export type View = {
  __typename?: 'View';
  by: User;
  id: Scalars['ID'];
  lastViewedAt: Scalars['Date'];
  name: Scalars['String'];
  to: ViewTo;
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

export type ViewTo = Api | Issue | Note | Organization | Post | Project | Question | Routine | SmartContract | Standard | User;

export enum VisibilityType {
  All = 'All',
  Private = 'Private',
  Public = 'Public'
}

export type Vote = {
  __typename?: 'Vote';
  by: User;
  isUpvote?: Maybe<Scalars['Boolean']>;
  to: VoteTo;
};

export type VoteEdge = {
  __typename?: 'VoteEdge';
  cursor: Scalars['String'];
  node: Vote;
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
  forConnect: Scalars['ID'];
  isUpvote?: InputMaybe<Scalars['Boolean']>;
  voteFor: VoteFor;
};

export type VoteSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  excludeLinkedToTag?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<VoteSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type VoteSearchResult = {
  __typename?: 'VoteSearchResult';
  edges: Array<VoteEdge>;
  pageInfo: PageInfo;
};

export enum VoteSortBy {
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

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
