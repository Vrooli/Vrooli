import { GraphQLResolveInfo, GraphQLScalarType, GraphQLScalarTypeConfig } from 'graphql';
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
export type Omit<T, K extends keyof T> = Pick<T, Exclude<keyof T, K>>;
export type RequireFields<T, K extends keyof T> = Omit<T, K> & { [P in K]-?: NonNullable<T[P]> };
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

export type AnyRun = RunProject | RunRoutine;

export type Api = {
  __typename: 'Api';
  completedAt?: Maybe<Scalars['Date']>;
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  hasCompleteVersion: Scalars['Boolean'];
  id: Scalars['ID'];
  isDeleted: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  issues: Array<Issue>;
  issuesCount: Scalars['Int'];
  labels: Array<Label>;
  owner?: Maybe<Owner>;
  parent?: Maybe<ApiVersion>;
  permissions: Scalars['String'];
  pullRequests: Array<PullRequest>;
  pullRequestsCount: Scalars['Int'];
  questions: Array<Question>;
  questionsCount: Scalars['Int'];
  score: Scalars['Int'];
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
  stats: Array<StatsApi>;
  tags: Array<Tag>;
  transfers: Array<Transfer>;
  transfersCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  versions: Array<ApiVersion>;
  versionsCount: Scalars['Int'];
  views: Scalars['Int'];
  you: ApiYou;
};

export type ApiCreateInput = {
  id: Scalars['ID'];
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  ownedByOrganizationConnect?: InputMaybe<Scalars['ID']>;
  ownedByUserConnect?: InputMaybe<Scalars['ID']>;
  parentConnect?: InputMaybe<Scalars['ID']>;
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  versionsCreate?: InputMaybe<Array<ApiVersionCreateInput>>;
};

export type ApiEdge = {
  __typename: 'ApiEdge';
  cursor: Scalars['String'];
  node: Api;
};

export type ApiKey = {
  __typename: 'ApiKey';
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
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  hasCompleteVersion?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  issuesId?: InputMaybe<Scalars['ID']>;
  labelsIds?: InputMaybe<Array<Scalars['ID']>>;
  maxScore?: InputMaybe<Scalars['Int']>;
  maxSaves?: InputMaybe<Scalars['Int']>;
  maxViews?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  ownedByOrganizationId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  pullRequestsId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ApiSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguagesLatestVersion?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ApiSearchResult = {
  __typename: 'ApiSearchResult';
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc'
}

export type ApiUpdateInput = {
  id: Scalars['ID'];
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  labelsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  ownedByOrganizationConnect?: InputMaybe<Scalars['ID']>;
  ownedByUserConnect?: InputMaybe<Scalars['ID']>;
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  versionsCreate?: InputMaybe<Array<ApiVersionCreateInput>>;
  versionsDelete?: InputMaybe<Array<Scalars['ID']>>;
  versionsUpdate?: InputMaybe<Array<ApiVersionUpdateInput>>;
};

export type ApiVersion = {
  __typename: 'ApiVersion';
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
  isPrivate: Scalars['Boolean'];
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
  you: VersionYou;
};

export type ApiVersionCreateInput = {
  callLink: Scalars['String'];
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  documentationLink?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isLatest?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  rootConnect?: InputMaybe<Scalars['ID']>;
  rootCreate?: InputMaybe<ApiCreateInput>;
  translationsCreate?: InputMaybe<Array<ApiVersionTranslationCreateInput>>;
  versionLabel: Scalars['String'];
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type ApiVersionEdge = {
  __typename: 'ApiVersionEdge';
  cursor: Scalars['String'];
  node: ApiVersion;
};

export type ApiVersionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
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
  __typename: 'ApiVersionSearchResult';
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
  __typename: 'ApiVersionTranslation';
  details?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
  summary?: Maybe<Scalars['String']>;
};

export type ApiVersionTranslationCreateInput = {
  details?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
  summary?: InputMaybe<Scalars['String']>;
};

export type ApiVersionTranslationUpdateInput = {
  details?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
  summary?: InputMaybe<Scalars['String']>;
};

export type ApiVersionUpdateInput = {
  callLink?: InputMaybe<Scalars['String']>;
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  directoryListingsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  documentationLink?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isLatest?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
  rootUpdate?: InputMaybe<ApiUpdateInput>;
  translationsCreate?: InputMaybe<Array<ApiVersionTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ApiVersionTranslationUpdateInput>>;
  versionLabel?: InputMaybe<Scalars['String']>;
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type ApiYou = {
  __typename: 'ApiYou';
  canDelete: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canBookmark: Scalars['Boolean'];
  canTransfer: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
};

export type Award = {
  __typename: 'Award';
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
  __typename: 'Comment';
  commentedOn: CommentedOn;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  owner?: Maybe<Owner>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  score: Scalars['Int'];
  bookmarkedBy?: Maybe<Array<User>>;
  bookmarks: Scalars['Int'];
  translations: Array<CommentTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  you: CommentYou;
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

export type CommentSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  apiVersionId?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  issueId?: InputMaybe<Scalars['ID']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
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
  __typename: 'CommentSearchResult';
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc'
}

export type CommentThread = {
  __typename: 'CommentThread';
  childThreads: Array<CommentThread>;
  comment: Comment;
  endCursor?: Maybe<Scalars['String']>;
  totalInThread?: Maybe<Scalars['Int']>;
};

export type CommentTranslation = {
  __typename: 'CommentTranslation';
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

export type CommentYou = {
  __typename: 'CommentYou';
  canDelete: Scalars['Boolean'];
  canReply: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canBookmark: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
};

export type CommentedOn = ApiVersion | Issue | NoteVersion | Post | ProjectVersion | PullRequest | Question | QuestionAnswer | RoutineVersion | SmartContractVersion | StandardVersion;

export type CopyInput = {
  id: Scalars['ID'];
  intendToPullRequest: Scalars['Boolean'];
  objectType: CopyType;
};

export type CopyResult = {
  __typename: 'CopyResult';
  apiVersion?: Maybe<ApiVersion>;
  noteVersion?: Maybe<NoteVersion>;
  organization?: Maybe<Organization>;
  projectVersion?: Maybe<ProjectVersion>;
  routineVersion?: Maybe<RoutineVersion>;
  smartContractVersion?: Maybe<SmartContractVersion>;
  standardVersion?: Maybe<StandardVersion>;
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
  __typename: 'Count';
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
  __typename: 'DevelopResult';
  completed: Array<ProjectOrRoutine>;
  inProgress: Array<ProjectOrRoutine>;
  recent: Array<ProjectOrRoutine>;
};

export type Email = {
  __typename: 'Email';
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

export type FindHandlesInput = {
  organizationId?: InputMaybe<Scalars['ID']>;
};

export type FindVersionInput = {
  handleRoot?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
  idRoot?: InputMaybe<Scalars['ID']>;
};

export enum GqlModelType {
  Api = 'Api',
  ApiKey = 'ApiKey',
  ApiVersion = 'ApiVersion',
  Comment = 'Comment',
  Copy = 'Copy',
  DevelopResult = 'DevelopResult',
  Email = 'Email',
  Fork = 'Fork',
  Handle = 'Handle',
  HistoryResult = 'HistoryResult',
  Issue = 'Issue',
  Label = 'Label',
  LearnResult = 'LearnResult',
  Meeting = 'Meeting',
  MeetingInvite = 'MeetingInvite',
  Member = 'Member',
  MemberInvite = 'MemberInvite',
  Node = 'Node',
  NodeEnd = 'NodeEnd',
  NodeLink = 'NodeLink',
  NodeLinkWhen = 'NodeLinkWhen',
  NodeLoop = 'NodeLoop',
  NodeLoopWhile = 'NodeLoopWhile',
  NodeRoutineList = 'NodeRoutineList',
  NodeRoutineListItem = 'NodeRoutineListItem',
  Note = 'Note',
  NoteVersion = 'NoteVersion',
  Notification = 'Notification',
  NotificationSubscription = 'NotificationSubscription',
  Organization = 'Organization',
  Payment = 'Payment',
  Phone = 'Phone',
  PopularResult = 'PopularResult',
  Post = 'Post',
  Premium = 'Premium',
  Project = 'Project',
  ProjectOrOrganizationSearchResult = 'ProjectOrOrganizationSearchResult',
  ProjectOrRoutineSearchResult = 'ProjectOrRoutineSearchResult',
  ProjectVersion = 'ProjectVersion',
  ProjectVersionDirectory = 'ProjectVersionDirectory',
  PullRequest = 'PullRequest',
  PushDevice = 'PushDevice',
  Question = 'Question',
  QuestionAnswer = 'QuestionAnswer',
  Quiz = 'Quiz',
  QuizAttempt = 'QuizAttempt',
  QuizQuestion = 'QuizQuestion',
  QuizQuestionResponse = 'QuizQuestionResponse',
  Reminder = 'Reminder',
  ReminderItem = 'ReminderItem',
  ReminderList = 'ReminderList',
  Report = 'Report',
  ReportResponse = 'ReportResponse',
  ReputationHistory = 'ReputationHistory',
  ResearchResult = 'ResearchResult',
  Resource = 'Resource',
  ResourceList = 'ResourceList',
  Role = 'Role',
  Routine = 'Routine',
  RoutineVersion = 'RoutineVersion',
  RoutineVersionInput = 'RoutineVersionInput',
  RoutineVersionOutput = 'RoutineVersionOutput',
  RunProject = 'RunProject',
  RunProjectOrRunRoutineSearchResult = 'RunProjectOrRunRoutineSearchResult',
  RunProjectSchedule = 'RunProjectSchedule',
  RunProjectStep = 'RunProjectStep',
  RunRoutine = 'RunRoutine',
  RunRoutineInput = 'RunRoutineInput',
  RunRoutineSchedule = 'RunRoutineSchedule',
  RunRoutineStep = 'RunRoutineStep',
  Session = 'Session',
  SessionUser = 'SessionUser',
  SmartContract = 'SmartContract',
  SmartContractVersion = 'SmartContractVersion',
  Standard = 'Standard',
  StandardVersion = 'StandardVersion',
  Star = 'Bookmark',
  StatsApi = 'StatsApi',
  StatsOrganization = 'StatsOrganization',
  StatsProject = 'StatsProject',
  StatsQuiz = 'StatsQuiz',
  StatsRoutine = 'StatsRoutine',
  StatsSite = 'StatsSite',
  StatsSmartContract = 'StatsSmartContract',
  StatsStandard = 'StatsStandard',
  StatsUser = 'StatsUser',
  Tag = 'Tag',
  Transfer = 'Transfer',
  User = 'User',
  UserSchedule = 'UserSchedule',
  UserScheduleFilter = 'UserScheduleFilter',
  View = 'View',
  Vote = 'Vote',
  Wallet = 'Wallet'
}

export type Handle = {
  __typename: 'Handle';
  handle: Scalars['String'];
  id: Scalars['ID'];
  wallet: Wallet;
};

export type HistoryInput = {
  searchString: Scalars['String'];
  take?: InputMaybe<Scalars['Int']>;
};

export type HistoryResult = {
  __typename: 'HistoryResult';
  activeRuns: Array<AnyRun>;
  completedRuns: Array<AnyRun>;
  recentlyBookmarked: Array<Bookmark>;
  recentlyViewed: Array<View>;
};

export type Issue = {
  __typename: 'Issue';
  closedAt?: Maybe<Scalars['Date']>;
  closedBy?: Maybe<User>;
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  labels: Array<Label>;
  labelsCount: Scalars['Int'];
  referencedVersionId?: Maybe<Scalars['String']>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  score: Scalars['Int'];
  bookmarkedBy?: Maybe<Array<Bookmark>>;
  bookmarks: Scalars['Int'];
  status: IssueStatus;
  to: IssueTo;
  translations: Array<IssueTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  you: IssueYou;
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
  referencedVersionIdConnect?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<IssueTranslationCreateInput>>;
};

export type IssueEdge = {
  __typename: 'IssueEdge';
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

export type IssueSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  apiId?: InputMaybe<Scalars['ID']>;
  closedById?: InputMaybe<Scalars['ID']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
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
  __typename: 'IssueSearchResult';
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc'
}

export enum IssueStatus {
  CloseUnresolved = 'CloseUnresolved',
  ClosedResolved = 'ClosedResolved',
  Open = 'Open',
  Rejected = 'Rejected'
}

export type IssueTo = Api | Note | Organization | Project | Routine | SmartContract | Standard;

export type IssueTranslation = {
  __typename: 'IssueTranslation';
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

export type IssueYou = {
  __typename: 'IssueYou';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canBookmark: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
};

export type Label = {
  __typename: 'Label';
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
  owner: Owner;
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
  you: LabelYou;
};

export type LabelCreateInput = {
  color?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  label: Scalars['String'];
  organizationConnect?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<LabelTranslationCreateInput>>;
};

export type LabelEdge = {
  __typename: 'LabelEdge';
  cursor: Scalars['String'];
  node: Label;
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
  __typename: 'LabelSearchResult';
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
  __typename: 'LabelTranslation';
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

export type LabelYou = {
  __typename: 'LabelYou';
  canDelete: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
};

export type LearnResult = {
  __typename: 'LearnResult';
  courses: Array<Project>;
  tutorials: Array<Routine>;
};

export type LogOutInput = {
  id?: InputMaybe<Scalars['ID']>;
};

export type Meeting = {
  __typename: 'Meeting';
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
  recurrEnd?: Maybe<Scalars['Date']>;
  recurrStart?: Maybe<Scalars['Date']>;
  recurring: Scalars['Boolean'];
  restrictedToRoles: Array<Role>;
  showOnOrganizationProfile: Scalars['Boolean'];
  timeZone?: Maybe<Scalars['String']>;
  translations: Array<MeetingTranslation>;
  translationsCount: Scalars['Int'];
  you: MeetingYou;
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
  __typename: 'MeetingEdge';
  cursor: Scalars['String'];
  node: Meeting;
};

export type MeetingInvite = {
  __typename: 'MeetingInvite';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  meeting: Meeting;
  message?: Maybe<Scalars['String']>;
  status: MeetingInviteStatus;
  updated_at: Scalars['Date'];
  user: User;
  you: MeetingInviteYou;
};

export type MeetingInviteCreateInput = {
  id: Scalars['ID'];
  meetingConnect: Scalars['ID'];
  message?: InputMaybe<Scalars['String']>;
  userConnect: Scalars['ID'];
};

export type MeetingInviteEdge = {
  __typename: 'MeetingInviteEdge';
  cursor: Scalars['String'];
  node: MeetingInvite;
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
  __typename: 'MeetingInviteSearchResult';
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

export type MeetingInviteYou = {
  __typename: 'MeetingInviteYou';
  canDelete: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
};

export type MeetingSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  labelsIds?: InputMaybe<Array<Scalars['ID']>>;
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
  __typename: 'MeetingSearchResult';
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
  __typename: 'MeetingTranslation';
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

export type MeetingYou = {
  __typename: 'MeetingYou';
  canDelete: Scalars['Boolean'];
  canInvite: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
};

export type Member = {
  __typename: 'Member';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isAdmin: Scalars['Boolean'];
  organization: Organization;
  permissions: Scalars['String'];
  updated_at: Scalars['Date'];
  user: User;
};

export type MemberEdge = {
  __typename: 'MemberEdge';
  cursor: Scalars['String'];
  node: Member;
};

export type MemberInvite = {
  __typename: 'MemberInvite';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  message?: Maybe<Scalars['String']>;
  organization: Organization;
  status: MemberInviteStatus;
  updated_at: Scalars['Date'];
  user: User;
  willBeAdmin: Scalars['Boolean'];
  willHavePermissions?: Maybe<Scalars['String']>;
  you: MemberInviteYou;
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
  __typename: 'MemberInviteEdge';
  cursor: Scalars['String'];
  node: MemberInvite;
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
  __typename: 'MemberInviteSearchResult';
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

export type MemberInviteYou = {
  __typename: 'MemberInviteYou';
  canDelete: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
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
  __typename: 'MemberSearchResult';
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
  __typename: 'Mutation';
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
  memberInviteAccept: MemberInvite;
  memberInviteCreate: MemberInvite;
  memberInviteDecline: MemberInvite;
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


export type MutationMemberInviteAcceptArgs = {
  input: FindByIdInput;
};


export type MutationMemberInviteCreateArgs = {
  input: MemberInviteCreateInput;
};


export type MutationMemberInviteDeclineArgs = {
  input: FindByIdInput;
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


export type MutationBookmarkArgs = {
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
  __typename: 'Node';
  columnIndex?: Maybe<Scalars['Int']>;
  created_at: Scalars['Date'];
  end?: Maybe<NodeEnd>;
  id: Scalars['ID'];
  loop?: Maybe<NodeLoop>;
  nodeType: NodeType;
  routineList?: Maybe<NodeRoutineList>;
  routineVersion: RoutineVersion;
  rowIndex?: Maybe<Scalars['Int']>;
  translations: Array<NodeTranslation>;
  updated_at: Scalars['Date'];
};

export type NodeCreateInput = {
  columnIndex?: InputMaybe<Scalars['Int']>;
  endCreate?: InputMaybe<NodeEndCreateInput>;
  id: Scalars['ID'];
  loopCreate?: InputMaybe<NodeLoopCreateInput>;
  nodeType: NodeType;
  routineListCreate?: InputMaybe<NodeRoutineListCreateInput>;
  routineVersionConnect: Scalars['ID'];
  rowIndex?: InputMaybe<Scalars['Int']>;
  translationsCreate?: InputMaybe<Array<NodeTranslationCreateInput>>;
};

export type NodeEnd = {
  __typename: 'NodeEnd';
  id: Scalars['ID'];
  node: Node;
  suggestedNextRoutineVersions?: Maybe<Array<RoutineVersion>>;
  wasSuccessful: Scalars['Boolean'];
};

export type NodeEndCreateInput = {
  id: Scalars['ID'];
  nodeConnect: Scalars['ID'];
  suggestedNextRoutineVersionsConnect?: InputMaybe<Array<Scalars['ID']>>;
  wasSuccessful?: InputMaybe<Scalars['Boolean']>;
};

export type NodeEndUpdateInput = {
  id: Scalars['ID'];
  suggestedNextRoutineVersionsConnect?: InputMaybe<Array<Scalars['ID']>>;
  suggestedNextRoutineVersionsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  wasSuccessful?: InputMaybe<Scalars['Boolean']>;
};

export type NodeLink = {
  __typename: 'NodeLink';
  from: Node;
  id: Scalars['ID'];
  operation?: Maybe<Scalars['String']>;
  routineVersion: RoutineVersion;
  to: Node;
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
  fromDisconnect?: InputMaybe<Scalars['Boolean']>;
  id: Scalars['ID'];
  operation?: InputMaybe<Scalars['String']>;
  toConnect?: InputMaybe<Scalars['ID']>;
  toDisconnect?: InputMaybe<Scalars['Boolean']>;
  whensCreate?: InputMaybe<Array<NodeLinkWhenCreateInput>>;
  whensDelete?: InputMaybe<Array<Scalars['ID']>>;
  whensUpdate?: InputMaybe<Array<NodeLinkWhenUpdateInput>>;
};

export type NodeLinkWhen = {
  __typename: 'NodeLinkWhen';
  condition: Scalars['String'];
  id: Scalars['ID'];
  link: NodeLink;
  translations: Array<NodeLinkWhenTranslation>;
};

export type NodeLinkWhenCreateInput = {
  condition: Scalars['String'];
  id: Scalars['ID'];
  linkConnect: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<NodeLinkWhenTranslationCreateInput>>;
};

export type NodeLinkWhenTranslation = {
  __typename: 'NodeLinkWhenTranslation';
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
  __typename: 'NodeLoop';
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
  __typename: 'NodeLoopWhile';
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
  __typename: 'NodeLoopWhileTranslation';
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
  __typename: 'NodeRoutineList';
  id: Scalars['ID'];
  isOptional: Scalars['Boolean'];
  isOrdered: Scalars['Boolean'];
  items: Array<NodeRoutineListItem>;
  node: Node;
};

export type NodeRoutineListCreateInput = {
  id: Scalars['ID'];
  isOptional?: InputMaybe<Scalars['Boolean']>;
  isOrdered?: InputMaybe<Scalars['Boolean']>;
  itemsCreate?: InputMaybe<Array<NodeRoutineListItemCreateInput>>;
  nodeConnect: Scalars['ID'];
};

export type NodeRoutineListItem = {
  __typename: 'NodeRoutineListItem';
  id: Scalars['ID'];
  index: Scalars['Int'];
  isOptional: Scalars['Boolean'];
  list: NodeRoutineList;
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
  __typename: 'NodeRoutineListItemTranslation';
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
  __typename: 'NodeTranslation';
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
  endUpdate?: InputMaybe<NodeEndUpdateInput>;
  id: Scalars['ID'];
  loopCreate?: InputMaybe<NodeLoopCreateInput>;
  loopDelete?: InputMaybe<Scalars['Boolean']>;
  loopUpdate?: InputMaybe<NodeLoopUpdateInput>;
  nodeType?: InputMaybe<NodeType>;
  routineListUpdate?: InputMaybe<NodeRoutineListUpdateInput>;
  routineVersionConnect?: InputMaybe<Scalars['ID']>;
  rowIndex?: InputMaybe<Scalars['Int']>;
  translationsCreate?: InputMaybe<Array<NodeTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<NodeTranslationUpdateInput>>;
};

export type Note = {
  __typename: 'Note';
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isPrivate: Scalars['Boolean'];
  issues: Array<Issue>;
  issuesCount: Scalars['Int'];
  labels: Array<Label>;
  labelsCount: Scalars['Int'];
  owner?: Maybe<Owner>;
  parent?: Maybe<NoteVersion>;
  permissions: Scalars['String'];
  pullRequests: Array<PullRequest>;
  pullRequestsCount: Scalars['Int'];
  questions: Array<Question>;
  questionsCount: Scalars['Int'];
  score: Scalars['Int'];
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
  tags: Array<Tag>;
  transfers: Array<Transfer>;
  transfersCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  versions: Array<NoteVersion>;
  versionsCount: Scalars['Int'];
  views: Scalars['Int'];
  you: NoteYou;
};

export type NoteCreateInput = {
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
  versionsCreate?: InputMaybe<Array<NoteVersionCreateInput>>;
};

export type NoteEdge = {
  __typename: 'NoteEdge';
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
  maxSaves?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
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
  __typename: 'NoteSearchResult';
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
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
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  userConnect?: InputMaybe<Scalars['ID']>;
  versionsCreate?: InputMaybe<Array<ApiVersionCreateInput>>;
  versionsDelete?: InputMaybe<Array<Scalars['ID']>>;
  versionsUpdate?: InputMaybe<Array<ApiVersionUpdateInput>>;
};

export type NoteVersion = {
  __typename: 'NoteVersion';
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
  pullRequest?: Maybe<PullRequest>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  root: Note;
  translations: Array<NoteVersionTranslation>;
  updated_at: Scalars['Date'];
  versionIndex: Scalars['Int'];
  versionLabel: Scalars['String'];
  versionNotes?: Maybe<Scalars['String']>;
  you: VersionYou;
};

export type NoteVersionCreateInput = {
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  isLatest?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  rootConnect?: InputMaybe<Scalars['ID']>;
  rootCreate?: InputMaybe<NoteCreateInput>;
  translationsCreate?: InputMaybe<Array<NoteVersionTranslationCreateInput>>;
  versionLabel: Scalars['String'];
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type NoteVersionEdge = {
  __typename: 'NoteVersionEdge';
  cursor: Scalars['String'];
  node: NoteVersion;
};

export type NoteVersionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
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
  __typename: 'NoteVersionSearchResult';
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
  __typename: 'NoteVersionTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
  text: Scalars['String'];
};

export type NoteVersionTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
  text: Scalars['String'];
};

export type NoteVersionTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
  text?: InputMaybe<Scalars['String']>;
};

export type NoteVersionUpdateInput = {
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  directoryListingsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  isLatest?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  rootUpdate?: InputMaybe<NoteUpdateInput>;
  translationsCreate?: InputMaybe<Array<NoteVersionTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<NoteVersionTranslationUpdateInput>>;
  versionLabel?: InputMaybe<Scalars['String']>;
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type NoteYou = {
  __typename: 'NoteYou';
  canDelete: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canBookmark: Scalars['Boolean'];
  canTransfer: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
};

export type Notification = {
  __typename: 'Notification';
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
  __typename: 'NotificationEdge';
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
  __typename: 'NotificationSearchResult';
  edges: Array<NotificationEdge>;
  pageInfo: PageInfo;
};

export type NotificationSettings = {
  __typename: 'NotificationSettings';
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
  __typename: 'NotificationSettingsCategory';
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
  __typename: 'NotificationSubscription';
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
  __typename: 'NotificationSubscriptionEdge';
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
  __typename: 'NotificationSubscriptionSearchResult';
  edges: Array<NotificationSubscriptionEdge>;
  pageInfo: PageInfo;
};

export enum NotificationSubscriptionSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  ObjectTypeAsc = 'ObjectTypeAsc',
  ObjectTypeDesc = 'ObjectTypeDesc'
}

export type NotificationSubscriptionUpdateInput = {
  id: Scalars['ID'];
  silent?: InputMaybe<Scalars['Boolean']>;
};

export type Organization = {
  __typename: 'Organization';
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
  permissions: Scalars['String'];
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
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
  stats: Array<StatsOrganization>;
  tags: Array<Tag>;
  transfersIncoming: Array<Transfer>;
  transfersOutgoing: Array<Transfer>;
  translatedName: Scalars['String'];
  translations: Array<OrganizationTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets: Array<Wallet>;
  you: OrganizationYou;
};

export type OrganizationCreateInput = {
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isOpenToNewMembers?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  memberInvitesCreate?: InputMaybe<Array<MemberInviteCreateInput>>;
  permissions?: InputMaybe<Scalars['String']>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  rolesCreate?: InputMaybe<Array<RoleCreateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<OrganizationTranslationCreateInput>>;
};

export type OrganizationEdge = {
  __typename: 'OrganizationEdge';
  cursor: Scalars['String'];
  node: Organization;
};

export type OrganizationSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isOpenToNewMembers?: InputMaybe<Scalars['Boolean']>;
  maxSaves?: InputMaybe<Scalars['Int']>;
  maxViews?: InputMaybe<Scalars['Int']>;
  memberUserIds?: InputMaybe<Array<Scalars['ID']>>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
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
  __typename: 'OrganizationSearchResult';
  edges: Array<OrganizationEdge>;
  pageInfo: PageInfo;
};

export enum OrganizationSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc'
}

export type OrganizationTranslation = {
  __typename: 'OrganizationTranslation';
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
  membersDelete?: InputMaybe<Array<Scalars['ID']>>;
  permissions?: InputMaybe<Scalars['String']>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
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

export type OrganizationYou = {
  __typename: 'OrganizationYou';
  canAddMembers: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canBookmark: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isViewed: Scalars['Boolean'];
  yourMembership?: Maybe<Member>;
};

export type Owner = Organization | User;

export type PageInfo = {
  __typename: 'PageInfo';
  endCursor?: Maybe<Scalars['String']>;
  hasNextPage: Scalars['Boolean'];
};

export type Payment = {
  __typename: 'Payment';
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
  __typename: 'Phone';
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
  __typename: 'PopularResult';
  organizations: Array<Organization>;
  projects: Array<Project>;
  routines: Array<Routine>;
  standards: Array<Standard>;
  users: Array<User>;
};

export type Post = {
  __typename: 'Post';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  owner: Owner;
  reports: Array<Report>;
  repostedFrom?: Maybe<Post>;
  reposts: Array<Post>;
  repostsCount: Scalars['Int'];
  resourceList: ResourceList;
  score: Scalars['Int'];
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<PostTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
};

export type PostCreateInput = {
  id: Scalars['ID'];
  isPinned?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  organizationConnect?: InputMaybe<Scalars['ID']>;
  repostedFromConnect?: InputMaybe<Scalars['ID']>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  userConnect?: InputMaybe<Scalars['ID']>;
};

export type PostEdge = {
  __typename: 'PostEdge';
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
  minBookmarks?: InputMaybe<Scalars['Int']>;
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
  __typename: 'PostSearchResult';
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc'
}

export type PostTranslation = {
  __typename: 'PostTranslation';
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
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
};

export type Premium = {
  __typename: 'Premium';
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
  isPrivateBookmarks?: InputMaybe<Scalars['Boolean']>;
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
  __typename: 'Project';
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  handle?: Maybe<Scalars['String']>;
  hasCompleteVersion: Scalars['Boolean'];
  id: Scalars['ID'];
  isPrivate: Scalars['Boolean'];
  issues: Array<Issue>;
  issuesCount: Scalars['Int'];
  labels: Array<Label>;
  labelsCount: Scalars['Int'];
  owner?: Maybe<Owner>;
  parent?: Maybe<ProjectVersion>;
  permissions: Scalars['String'];
  pullRequests: Array<PullRequest>;
  pullRequestsCount: Scalars['Int'];
  questions: Array<Question>;
  questionsCount: Scalars['Int'];
  quizzes: Array<Quiz>;
  quizzesCount: Scalars['Int'];
  score: Scalars['Int'];
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
  stats: Array<StatsProject>;
  tags: Array<Tag>;
  transfers: Array<Transfer>;
  transfersCount: Scalars['Int'];
  translatedName: Scalars['String'];
  updated_at: Scalars['Date'];
  versions: Array<ProjectVersion>;
  versionsCount: Scalars['Int'];
  views: Scalars['Int'];
  you: ProjectYou;
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
  __typename: 'ProjectEdge';
  cursor: Scalars['String'];
  node: Project;
};

export type ProjectOrOrganization = Organization | Project;

export type ProjectOrOrganizationEdge = {
  __typename: 'ProjectOrOrganizationEdge';
  cursor: Scalars['String'];
  node: ProjectOrOrganization;
};

export type ProjectOrOrganizationPageInfo = {
  __typename: 'ProjectOrOrganizationPageInfo';
  endCursorOrganization?: Maybe<Scalars['String']>;
  endCursorProject?: Maybe<Scalars['String']>;
  hasNextPage: Scalars['Boolean'];
};

export type ProjectOrOrganizationSearchInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
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
  __typename: 'ProjectOrOrganizationSearchResult';
  edges: Array<ProjectOrOrganizationEdge>;
  pageInfo: ProjectOrOrganizationPageInfo;
};

export enum ProjectOrOrganizationSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc'
}

export type ProjectOrRoutine = Project | Routine;

export type ProjectOrRoutineEdge = {
  __typename: 'ProjectOrRoutineEdge';
  cursor: Scalars['String'];
  node: ProjectOrRoutine;
};

export type ProjectOrRoutinePageInfo = {
  __typename: 'ProjectOrRoutinePageInfo';
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
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  objectType?: InputMaybe<Scalars['String']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  projectAfter?: InputMaybe<Scalars['String']>;
  reportId?: InputMaybe<Scalars['ID']>;
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
  __typename: 'ProjectOrRoutineSearchResult';
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc'
}

export type ProjectSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  hasCompleteVersion?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  issuesId?: InputMaybe<Scalars['ID']>;
  labelsIds?: InputMaybe<Array<Scalars['ID']>>;
  maxScore?: InputMaybe<Scalars['Int']>;
  maxSaves?: InputMaybe<Scalars['Int']>;
  maxViews?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  ownedByOrganizationId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  pullRequestsId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguagesLatestVersion?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ProjectSearchResult = {
  __typename: 'ProjectSearchResult';
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
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
  __typename: 'ProjectVersion';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  complexity: Scalars['Int'];
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
  pullRequest?: Maybe<PullRequest>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  root: Project;
  runProjectsCount: Scalars['Int'];
  simplicity: Scalars['Int'];
  suggestedNextByProject: Array<Project>;
  timesCompleted: Scalars['Int'];
  timesStarted: Scalars['Int'];
  translations: Array<ProjectVersionTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  versionIndex: Scalars['Int'];
  versionLabel: Scalars['String'];
  versionNotes?: Maybe<Scalars['String']>;
  you: ProjectVersionYou;
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
  versionLabel: Scalars['String'];
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type ProjectVersionDirectory = {
  __typename: 'ProjectVersionDirectory';
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
  __typename: 'ProjectVersionDirectoryTranslation';
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
  __typename: 'ProjectVersionEdge';
  cursor: Scalars['String'];
  node: ProjectVersion;
};

export type ProjectVersionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  directoryListingsId?: InputMaybe<Scalars['ID']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isCompleteWithRoot?: InputMaybe<Scalars['Boolean']>;
  isCompleteWithRootExcludeOwnedByOrganizationId?: InputMaybe<Scalars['ID']>;
  isCompleteWithRootExcludeOwnedByUserId?: InputMaybe<Scalars['ID']>;
  maxComplexity?: InputMaybe<Scalars['Int']>;
  maxSimplicity?: InputMaybe<Scalars['Int']>;
  maxTimesCompleted?: InputMaybe<Scalars['Int']>;
  minComplexity?: InputMaybe<Scalars['Int']>;
  minScoreRoot?: InputMaybe<Scalars['Int']>;
  minSimplicity?: InputMaybe<Scalars['Int']>;
  minBookmarksRoot?: InputMaybe<Scalars['Int']>;
  minTimesCompleted?: InputMaybe<Scalars['Int']>;
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
  __typename: 'ProjectVersionSearchResult';
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
  __typename: 'ProjectVersionTranslation';
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
  rootUpdate?: InputMaybe<ProjectUpdateInput>;
  suggestedNextByProjectConnect?: InputMaybe<Array<Scalars['ID']>>;
  suggestedNextByProjectDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  translationsCreate?: InputMaybe<Array<ProjectVersionTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectVersionTranslationUpdateInput>>;
  versionLabel?: InputMaybe<Scalars['String']>;
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type ProjectVersionYou = {
  __typename: 'ProjectVersionYou';
  canComment: Scalars['Boolean'];
  canCopy: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  canUse: Scalars['Boolean'];
  runs: Array<RunProject>;
};

export type ProjectYou = {
  __typename: 'ProjectYou';
  canDelete: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canBookmark: Scalars['Boolean'];
  canTransfer: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
};

export type PullRequest = {
  __typename: 'PullRequest';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  from: PullRequestFrom;
  id: Scalars['ID'];
  mergedOrRejectedAt?: Maybe<Scalars['Date']>;
  status: PullRequestStatus;
  to: PullRequestTo;
  updated_at: Scalars['Date'];
  you: PullRequestYou;
};

export type PullRequestCreateInput = {
  fromConnect: Scalars['ID'];
  id: Scalars['ID'];
  toConnect: Scalars['ID'];
  toObjectType: PullRequestToObjectType;
};

export type PullRequestEdge = {
  __typename: 'PullRequestEdge';
  cursor: Scalars['String'];
  node: PullRequest;
};

export type PullRequestFrom = ApiVersion | NoteVersion | ProjectVersion | RoutineVersion | SmartContractVersion | StandardVersion;

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
  __typename: 'PullRequestSearchResult';
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

export type PullRequestYou = {
  __typename: 'PullRequestYou';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
};

export type PushDevice = {
  __typename: 'PushDevice';
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
  __typename: 'Query';
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
  runProjectOrRunRoutines: RunProjectOrRunRoutineSearchResult;
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
  bookmarks: BookmarkSearchResult;
  statsApi: StatsApiSearchResult;
  statsOrganization: StatsOrganizationSearchResult;
  statsProject: StatsProjectSearchResult;
  statsQuiz: StatsQuizSearchResult;
  statsRoutine: StatsRoutineSearchResult;
  statsSite: StatsSiteSearchResult;
  statsSmartContract: StatsSmartContractSearchResult;
  statsStandard: StatsStandardSearchResult;
  statsUser: StatsUserSearchResult;
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
  input: FindVersionInput;
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
  input: FindVersionInput;
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
  input: FindVersionInput;
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
  input: FindByIdInput;
};


export type QueryRoutineVersionArgs = {
  input: FindVersionInput;
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


export type QueryRunProjectOrRunRoutinesArgs = {
  input: RunProjectOrRunRoutineSearchInput;
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
  input: FindVersionInput;
};


export type QuerySmartContractVersionsArgs = {
  input: SmartContractVersionSearchInput;
};


export type QuerySmartContractsArgs = {
  input: SmartContractSearchInput;
};


export type QueryStandardArgs = {
  input: FindByIdInput;
};


export type QueryStandardVersionArgs = {
  input: FindVersionInput;
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


export type QueryStatsApiArgs = {
  input: StatsApiSearchInput;
};


export type QueryStatsOrganizationArgs = {
  input: StatsOrganizationSearchInput;
};


export type QueryStatsProjectArgs = {
  input: StatsProjectSearchInput;
};


export type QueryStatsQuizArgs = {
  input: StatsQuizSearchInput;
};


export type QueryStatsRoutineArgs = {
  input: StatsRoutineSearchInput;
};


export type QueryStatsSiteArgs = {
  input: StatsSiteSearchInput;
};


export type QueryStatsSmartContractArgs = {
  input: StatsSmartContractSearchInput;
};


export type QueryStatsStandardArgs = {
  input: StatsStandardSearchInput;
};


export type QueryStatsUserArgs = {
  input: StatsUserSearchInput;
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
  __typename: 'Question';
  answers: Array<QuestionAnswer>;
  answersCount: Scalars['Int'];
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  forObject: QuestionFor;
  hasAcceptedAnswer: Scalars['Boolean'];
  id: Scalars['ID'];
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  score: Scalars['Int'];
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<QuestionTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  you: QuestionYou;
};

export type QuestionAnswer = {
  __typename: 'QuestionAnswer';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isAccepted: Scalars['Boolean'];
  question: Question;
  score: Scalars['Int'];
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
  translations: Array<QuestionAnswerTranslation>;
  updated_at: Scalars['Date'];
};

export type QuestionAnswerCreateInput = {
  id: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<QuestionAnswerTranslationCreateInput>>;
};

export type QuestionAnswerEdge = {
  __typename: 'QuestionAnswerEdge';
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
  minBookmarks?: InputMaybe<Scalars['Int']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<QuestionAnswerSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type QuestionAnswerSearchResult = {
  __typename: 'QuestionAnswerSearchResult';
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc'
}

export type QuestionAnswerTranslation = {
  __typename: 'QuestionAnswerTranslation';
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
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<QuestionTranslationCreateInput>>;
};

export type QuestionEdge = {
  __typename: 'QuestionEdge';
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
  maxSaves?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
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
  __typename: 'QuestionSearchResult';
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc'
}

export type QuestionTranslation = {
  __typename: 'QuestionTranslation';
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
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<QuestionTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<QuestionTranslationUpdateInput>>;
};

export type QuestionYou = {
  __typename: 'QuestionYou';
  canDelete: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canBookmark: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
};

export type Quiz = {
  __typename: 'Quiz';
  attempts: Array<QuizAttempt>;
  attemptsCount: Scalars['Int'];
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isPrivate: Scalars['Boolean'];
  project?: Maybe<Project>;
  quizQuestions: Array<QuizQuestion>;
  quizQuestionsCount: Scalars['Int'];
  randomizeQuestionOrder: Scalars['Boolean'];
  routine?: Maybe<Routine>;
  score: Scalars['Int'];
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
  stats: Array<StatsQuiz>;
  translations: Array<QuizTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  you: QuizYou;
};

export type QuizAttempt = {
  __typename: 'QuizAttempt';
  contextSwitches: Scalars['Int'];
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  pointsEarned: Scalars['Int'];
  quiz: Quiz;
  responses: Array<QuizQuestionResponse>;
  responsesCount: Scalars['Int'];
  status: QuizAttemptStatus;
  timeTaken?: Maybe<Scalars['Int']>;
  updated_at: Scalars['Date'];
  user: User;
  you: QuizAttemptYou;
};

export type QuizAttemptCreateInput = {
  contextSwitches?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  responsesCreate?: InputMaybe<Array<QuizQuestionResponseCreateInput>>;
  timeTaken?: InputMaybe<Scalars['Int']>;
};

export type QuizAttemptEdge = {
  __typename: 'QuizAttemptEdge';
  cursor: Scalars['String'];
  node: QuizAttempt;
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
  __typename: 'QuizAttemptSearchResult';
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

export type QuizAttemptYou = {
  __typename: 'QuizAttemptYou';
  canDelete: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
};

export type QuizCreateInput = {
  id: Scalars['ID'];
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  maxAttempts?: InputMaybe<Scalars['Int']>;
  pointsToPass?: InputMaybe<Scalars['Int']>;
  projectConnect?: InputMaybe<Scalars['ID']>;
  quizQuestionsCreate?: InputMaybe<Array<QuizQuestionCreateInput>>;
  randomizeQuestionOder?: InputMaybe<Scalars['Boolean']>;
  revealCorrectAnswers?: InputMaybe<Scalars['Boolean']>;
  routineConnect?: InputMaybe<Scalars['ID']>;
  timeLimit?: InputMaybe<Scalars['Int']>;
  translationsCreate?: InputMaybe<Array<QuizTranslationCreateInput>>;
};

export type QuizEdge = {
  __typename: 'QuizEdge';
  cursor: Scalars['String'];
  node: Quiz;
};

export type QuizQuestion = {
  __typename: 'QuizQuestion';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  order?: Maybe<Scalars['Int']>;
  points: Scalars['Int'];
  quiz: Quiz;
  responses?: Maybe<Array<QuizQuestionResponse>>;
  responsesCount: Scalars['Int'];
  standardVersion?: Maybe<StandardVersion>;
  translations?: Maybe<Array<QuizQuestionTranslation>>;
  updated_at: Scalars['Date'];
  you: QuizQuestionYou;
};

export type QuizQuestionCreateInput = {
  id: Scalars['ID'];
  order?: InputMaybe<Scalars['Int']>;
  points?: InputMaybe<Scalars['Int']>;
  quizConnect?: InputMaybe<Scalars['ID']>;
  standardVersionConnect?: InputMaybe<Scalars['ID']>;
  standardVersionCreate?: InputMaybe<StandardVersionCreateInput>;
  translationsCreate?: InputMaybe<Array<QuizQuestionTranslationCreateInput>>;
};

export type QuizQuestionEdge = {
  __typename: 'QuizQuestionEdge';
  cursor: Scalars['String'];
  node: QuizQuestion;
};

export type QuizQuestionResponse = {
  __typename: 'QuizQuestionResponse';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  quizAttempt: QuizAttempt;
  quizQuestion: QuizQuestion;
  response?: Maybe<Scalars['String']>;
  translations: Array<QuizQuestionResponseTranslation>;
  updated_at: Scalars['Date'];
  you: QuizQuestionResponseYou;
};

export type QuizQuestionResponseCreateInput = {
  id: Scalars['ID'];
  quizAttemptConnect: Scalars['ID'];
  quizQuestionConnect: Scalars['ID'];
  response?: InputMaybe<Scalars['String']>;
  translationsCreate?: InputMaybe<Array<QuizQuestionResponseTranslationCreateInput>>;
};

export type QuizQuestionResponseEdge = {
  __typename: 'QuizQuestionResponseEdge';
  cursor: Scalars['String'];
  node: QuizQuestionResponse;
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
  __typename: 'QuizQuestionResponseSearchResult';
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
  __typename: 'QuizQuestionResponseTranslation';
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

export type QuizQuestionResponseYou = {
  __typename: 'QuizQuestionResponseYou';
  canDelete: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
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
  __typename: 'QuizQuestionSearchResult';
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
  __typename: 'QuizQuestionTranslation';
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
  standardVersionConnect?: InputMaybe<Scalars['ID']>;
  standardVersionCreate?: InputMaybe<StandardVersionCreateInput>;
  standardVersionUpdate?: InputMaybe<StandardVersionUpdateInput>;
  translationsCreate?: InputMaybe<Array<QuizQuestionTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<QuizQuestionTranslationUpdateInput>>;
};

export type QuizQuestionYou = {
  __typename: 'QuizQuestionYou';
  canDelete: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
};

export type QuizSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  minScore?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
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
  __typename: 'QuizSearchResult';
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc'
}

export type QuizTranslation = {
  __typename: 'QuizTranslation';
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
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  maxAttempts?: InputMaybe<Scalars['Int']>;
  pointsToPass?: InputMaybe<Scalars['Int']>;
  projectConnect?: InputMaybe<Scalars['ID']>;
  projectDisconnect?: InputMaybe<Scalars['ID']>;
  quizQuestionsCreate?: InputMaybe<Array<QuizQuestionCreateInput>>;
  quizQuestionsDelete?: InputMaybe<Array<Scalars['ID']>>;
  quizQuestionsUpdate?: InputMaybe<Array<QuizQuestionUpdateInput>>;
  randomizeQuestionOrder?: InputMaybe<Scalars['Boolean']>;
  revealCorrectAnswers?: InputMaybe<Scalars['Boolean']>;
  routineConnect?: InputMaybe<Scalars['ID']>;
  routineDisconnect?: InputMaybe<Scalars['ID']>;
  timeLimit?: InputMaybe<Scalars['Int']>;
  translationsCreate?: InputMaybe<Array<QuizTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<QuizTranslationUpdateInput>>;
};

export type QuizYou = {
  __typename: 'QuizYou';
  canDelete: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canBookmark: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
  hasCompleted: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
};

export type ReadAssetsInput = {
  files: Array<Scalars['String']>;
};

export type Reminder = {
  __typename: 'Reminder';
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
  __typename: 'ReminderEdge';
  cursor: Scalars['String'];
  node: Reminder;
};

export type ReminderItem = {
  __typename: 'ReminderItem';
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
  __typename: 'ReminderList';
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
  __typename: 'ReminderListEdge';
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
  __typename: 'ReminderListSearchResult';
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
  __typename: 'ReminderSearchResult';
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
  __typename: 'Report';
  created_at: Scalars['Date'];
  details?: Maybe<Scalars['String']>;
  id?: Maybe<Scalars['ID']>;
  language: Scalars['String'];
  reason: Scalars['String'];
  responses: Array<ReportResponse>;
  responsesCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  you: ReportYou;
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
  __typename: 'ReportEdge';
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
  __typename: 'ReportResponse';
  actionSuggested: ReportSuggestedAction;
  created_at: Scalars['Date'];
  details?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: Maybe<Scalars['String']>;
  report: Report;
  updated_at: Scalars['Date'];
  you: ReportResponseYou;
};

export type ReportResponseCreateInput = {
  actionSuggested: ReportSuggestedAction;
  details?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  reportConnect: Scalars['ID'];
};

export type ReportResponseEdge = {
  __typename: 'ReportResponseEdge';
  cursor: Scalars['String'];
  node: ReportResponse;
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
  __typename: 'ReportResponseSearchResult';
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

export type ReportResponseYou = {
  __typename: 'ReportResponseYou';
  canDelete: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
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
  __typename: 'ReportSearchResult';
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

export type ReportYou = {
  __typename: 'ReportYou';
  canDelete: Scalars['Boolean'];
  canRespond: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
};

export type ReputationHistory = {
  __typename: 'ReputationHistory';
  amount: Scalars['Int'];
  created_at: Scalars['Date'];
  event: Scalars['String'];
  id: Scalars['ID'];
  objectId1?: Maybe<Scalars['ID']>;
  objectId2?: Maybe<Scalars['ID']>;
  updated_at: Scalars['Date'];
};

export type ReputationHistoryEdge = {
  __typename: 'ReputationHistoryEdge';
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
  __typename: 'ReputationHistorySearchResult';
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
  __typename: 'ResearchResult';
  needInvestments: Array<Project>;
  needMembers: Array<Organization>;
  needVotes: Array<Project>;
  newlyCompleted: Array<ProjectOrRoutine>;
  processes: Array<Routine>;
};

export type Resource = {
  __typename: 'Resource';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  index?: Maybe<Scalars['Int']>;
  link: Scalars['String'];
  list: ResourceList;
  translations: Array<ResourceTranslation>;
  updated_at: Scalars['Date'];
  usedFor: ResourceUsedFor;
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
  __typename: 'ResourceEdge';
  cursor: Scalars['String'];
  node: Resource;
};

export type ResourceList = {
  __typename: 'ResourceList';
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
  __typename: 'ResourceListEdge';
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
  __typename: 'ResourceListSearchResult';
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
  __typename: 'ResourceListTranslation';
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
  __typename: 'ResourceSearchResult';
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
  __typename: 'ResourceTranslation';
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
  __typename: 'Response';
  code?: Maybe<Scalars['Int']>;
  message: Scalars['String'];
};

export type Role = {
  __typename: 'Role';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  members?: Maybe<Array<Member>>;
  membersCount: Scalars['Int'];
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
  __typename: 'RoleEdge';
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
  __typename: 'RoleSearchResult';
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
  __typename: 'RoleTranslation';
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

export type Routine = {
  __typename: 'Routine';
  completedAt?: Maybe<Scalars['Date']>;
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  forks: Array<Routine>;
  forksCount: Scalars['Int'];
  hasCompleteVersion: Scalars['Boolean'];
  id: Scalars['ID'];
  isDeleted: Scalars['Boolean'];
  isInternal?: Maybe<Scalars['Boolean']>;
  isPrivate: Scalars['Boolean'];
  issues: Array<Issue>;
  issuesCount: Scalars['Int'];
  labels: Array<Label>;
  owner?: Maybe<Owner>;
  parent?: Maybe<RoutineVersion>;
  permissions: Scalars['String'];
  pullRequests: Array<PullRequest>;
  pullRequestsCount: Scalars['Int'];
  questions: Array<Question>;
  questionsCount: Scalars['Int'];
  quizzes: Array<Quiz>;
  quizzesCount: Scalars['Int'];
  score: Scalars['Int'];
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
  stats: Array<StatsRoutine>;
  tags: Array<Tag>;
  transfers: Array<Transfer>;
  transfersCount: Scalars['Int'];
  translatedName: Scalars['String'];
  updated_at: Scalars['Date'];
  versions: Array<RoutineVersion>;
  versionsCount?: Maybe<Scalars['Int']>;
  views: Scalars['Int'];
  you: RoutineYou;
};

export type RoutineCreateInput = {
  id: Scalars['ID'];
  isInternal?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  organizationConnect?: InputMaybe<Scalars['ID']>;
  parentConnect?: InputMaybe<Scalars['ID']>;
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  userConnect?: InputMaybe<Scalars['ID']>;
  versionsCreate?: InputMaybe<Array<RoutineVersionCreateInput>>;
};

export type RoutineEdge = {
  __typename: 'RoutineEdge';
  cursor: Scalars['String'];
  node: Routine;
};

export type RoutineSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  hasCompleteVersion?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isInternal?: InputMaybe<Scalars['Boolean']>;
  issuesId?: InputMaybe<Scalars['ID']>;
  labelsIds?: InputMaybe<Array<Scalars['ID']>>;
  maxScore?: InputMaybe<Scalars['Int']>;
  maxSaves?: InputMaybe<Scalars['Int']>;
  maxViews?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  ownedByOrganizationId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  pullRequestsId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RoutineSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguagesLatestVersion?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type RoutineSearchResult = {
  __typename: 'RoutineSearchResult';
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
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
  __typename: 'RoutineVersion';
  apiCallData?: Maybe<Scalars['String']>;
  apiVersion?: Maybe<ApiVersion>;
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  complexity: Scalars['Int'];
  created_at: Scalars['Date'];
  directoryListings: Array<ProjectVersionDirectory>;
  directoryListingsCount: Scalars['Int'];
  forks: Array<Routine>;
  forksCount: Scalars['Int'];
  id: Scalars['ID'];
  inputs: Array<RoutineVersionInput>;
  inputsCount: Scalars['Int'];
  isAutomatable?: Maybe<Scalars['Boolean']>;
  isComplete: Scalars['Boolean'];
  isDeleted: Scalars['Boolean'];
  isLatest: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  nodeLinks: Array<NodeLink>;
  nodeLinksCount: Scalars['Int'];
  nodes: Array<Node>;
  nodesCount: Scalars['Int'];
  outputs: Array<RoutineVersionOutput>;
  outputsCount: Scalars['Int'];
  pullRequest?: Maybe<PullRequest>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceList?: Maybe<ResourceList>;
  root: Routine;
  simplicity: Scalars['Int'];
  smartContractCallData?: Maybe<Scalars['String']>;
  smartContractVersion?: Maybe<SmartContractVersion>;
  suggestedNextByRoutineVersion: Array<RoutineVersion>;
  suggestedNextByRoutineVersionCount: Scalars['Int'];
  timesCompleted: Scalars['Int'];
  timesStarted: Scalars['Int'];
  translations: Array<RoutineVersionTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  versionIndex: Scalars['Int'];
  versionLabel: Scalars['String'];
  versionNotes?: Maybe<Scalars['String']>;
  you: RoutineVersionYou;
};

export type RoutineVersionCreateInput = {
  apiCallData?: InputMaybe<Scalars['String']>;
  apiVersionConnect?: InputMaybe<Scalars['ID']>;
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  inputsCreate?: InputMaybe<Array<RoutineVersionInputCreateInput>>;
  isAutomatable?: InputMaybe<Scalars['Boolean']>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isLatest?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  nodeLinksCreate?: InputMaybe<Array<NodeLinkCreateInput>>;
  nodesCreate?: InputMaybe<Array<NodeCreateInput>>;
  outputsCreate?: InputMaybe<Array<RoutineVersionOutputCreateInput>>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  rootConnect?: InputMaybe<Scalars['ID']>;
  rootCreate?: InputMaybe<RoutineCreateInput>;
  smartContractCallData?: InputMaybe<Scalars['String']>;
  smartContractVersionConnect?: InputMaybe<Scalars['ID']>;
  suggestedNextByRoutineVersionConnect?: InputMaybe<Array<Scalars['ID']>>;
  translationsCreate?: InputMaybe<Array<RoutineVersionTranslationCreateInput>>;
  versionLabel: Scalars['String'];
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type RoutineVersionEdge = {
  __typename: 'RoutineVersionEdge';
  cursor: Scalars['String'];
  node: RoutineVersion;
};

export type RoutineVersionInput = {
  __typename: 'RoutineVersionInput';
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
  __typename: 'RoutineVersionInputTranslation';
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
  __typename: 'RoutineVersionOutput';
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
  __typename: 'RoutineVersionOutputTranslation';
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
  minBookmarksRoot?: InputMaybe<Scalars['Int']>;
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
  __typename: 'RoutineVersionSearchResult';
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
  __typename: 'RoutineVersionTranslation';
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
  apiVersionDisconnect?: InputMaybe<Scalars['Boolean']>;
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  directoryListingsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  inputsCreate?: InputMaybe<Array<RoutineVersionInputCreateInput>>;
  inputsDelete?: InputMaybe<Array<Scalars['ID']>>;
  inputsUpdate?: InputMaybe<Array<RoutineVersionInputUpdateInput>>;
  isAutomatable?: InputMaybe<Scalars['Boolean']>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
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
  rootUpdate?: InputMaybe<RoutineUpdateInput>;
  smartContractCallData?: InputMaybe<Scalars['String']>;
  smartContractVersionConnect?: InputMaybe<Scalars['ID']>;
  smartContractVersionDisconnect?: InputMaybe<Scalars['Boolean']>;
  suggestedNextByRoutineVersionConnect?: InputMaybe<Array<Scalars['ID']>>;
  suggestedNextByRoutineVersionDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  translationsCreate?: InputMaybe<Array<RoutineVersionTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<RoutineVersionTranslationUpdateInput>>;
  versionLabel?: InputMaybe<Scalars['String']>;
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type RoutineVersionYou = {
  __typename: 'RoutineVersionYou';
  canComment: Scalars['Boolean'];
  canCopy: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canRun: Scalars['Boolean'];
  canBookmark: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
  runs: Array<RunRoutine>;
};

export type RoutineYou = {
  __typename: 'RoutineYou';
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canBookmark: Scalars['Boolean'];
  canTransfer: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
};

export type RunProject = {
  __typename: 'RunProject';
  completedAt?: Maybe<Scalars['Date']>;
  completedComplexity: Scalars['Int'];
  contextSwitches: Scalars['Int'];
  id: Scalars['ID'];
  isPrivate: Scalars['Boolean'];
  name: Scalars['String'];
  organization?: Maybe<Organization>;
  projectVersion?: Maybe<ProjectVersion>;
  runProjectSchedule?: Maybe<RunProjectSchedule>;
  startedAt?: Maybe<Scalars['Date']>;
  status: RunStatus;
  steps: Array<RunProjectStep>;
  stepsCount: Scalars['Int'];
  timeElapsed?: Maybe<Scalars['Int']>;
  user?: Maybe<User>;
  wasRunAutomaticaly: Scalars['Boolean'];
  you: RunProjectYou;
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
  __typename: 'RunProjectEdge';
  cursor: Scalars['String'];
  node: RunProject;
};

export type RunProjectOrRunRoutine = RunProject | RunRoutine;

export type RunProjectOrRunRoutineEdge = {
  __typename: 'RunProjectOrRunRoutineEdge';
  cursor: Scalars['String'];
  node: RunProjectOrRunRoutine;
};

export type RunProjectOrRunRoutinePageInfo = {
  __typename: 'RunProjectOrRunRoutinePageInfo';
  endCursorProject?: Maybe<Scalars['String']>;
  endCursorRoutine?: Maybe<Scalars['String']>;
  hasNextPage: Scalars['Boolean'];
};

export type RunProjectOrRunRoutineSearchInput = {
  completedTimeFrame?: InputMaybe<TimeFrame>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  objectType?: InputMaybe<Scalars['String']>;
  projectVersionId?: InputMaybe<Scalars['ID']>;
  routineVersionId?: InputMaybe<Scalars['ID']>;
  runProjectAfter?: InputMaybe<Scalars['String']>;
  runRoutineAfter?: InputMaybe<Scalars['String']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RunProjectOrRunRoutineSortBy>;
  startedTimeFrame?: InputMaybe<TimeFrame>;
  status?: InputMaybe<RunStatus>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type RunProjectOrRunRoutineSearchResult = {
  __typename: 'RunProjectOrRunRoutineSearchResult';
  edges: Array<RunProjectOrRunRoutineEdge>;
  pageInfo: RunProjectOrRunRoutinePageInfo;
};

export enum RunProjectOrRunRoutineSortBy {
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
  StepsAsc = 'StepsAsc',
  StepsDesc = 'StepsDesc'
}

export type RunProjectSchedule = {
  __typename: 'RunProjectSchedule';
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
  __typename: 'RunProjectScheduleEdge';
  cursor: Scalars['String'];
  node: RunProjectSchedule;
};

export type RunProjectScheduleSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  labelIds?: InputMaybe<Array<Scalars['ID']>>;
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
  __typename: 'RunProjectScheduleSearchResult';
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
  __typename: 'RunProjectScheduleTranslation';
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
  runProjectConnect?: InputMaybe<Scalars['ID']>;
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
  projectVersionId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RunProjectSortBy>;
  startedTimeFrame?: InputMaybe<TimeFrame>;
  status?: InputMaybe<RunStatus>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type RunProjectSearchResult = {
  __typename: 'RunProjectSearchResult';
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
  __typename: 'RunProjectStep';
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

export type RunProjectYou = {
  __typename: 'RunProjectYou';
  canDelete: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
};

export type RunRoutine = {
  __typename: 'RunRoutine';
  completedAt?: Maybe<Scalars['Date']>;
  completedComplexity: Scalars['Int'];
  contextSwitches: Scalars['Int'];
  id: Scalars['ID'];
  inputs: Array<RunRoutineInput>;
  inputsCount: Scalars['Int'];
  isPrivate: Scalars['Boolean'];
  name: Scalars['String'];
  organization?: Maybe<Organization>;
  routineVersion?: Maybe<RoutineVersion>;
  runProject?: Maybe<RunProject>;
  runRoutineSchedule?: Maybe<RunRoutineSchedule>;
  startedAt?: Maybe<Scalars['Date']>;
  status: RunStatus;
  steps: Array<RunRoutineStep>;
  stepsCount: Scalars['Int'];
  timeElapsed?: Maybe<Scalars['Int']>;
  user?: Maybe<User>;
  wasRunAutomaticaly: Scalars['Boolean'];
  you: RunRoutineYou;
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
  routineVersionConnect?: InputMaybe<Scalars['ID']>;
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
  __typename: 'RunRoutineEdge';
  cursor: Scalars['String'];
  node: RunRoutine;
};

export type RunRoutineInput = {
  __typename: 'RunRoutineInput';
  data: Scalars['String'];
  id: Scalars['ID'];
  input: RoutineVersionInput;
  runRoutine: RunRoutine;
};

export type RunRoutineInputCreateInput = {
  data: Scalars['String'];
  id: Scalars['ID'];
  inputConnect: Scalars['ID'];
  runRoutineConnect: Scalars['ID'];
};

export type RunRoutineInputEdge = {
  __typename: 'RunRoutineInputEdge';
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
  __typename: 'RunRoutineInputSearchResult';
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

export type RunRoutineSchedule = {
  __typename: 'RunRoutineSchedule';
  attemptAutomatic: Scalars['Boolean'];
  id: Scalars['ID'];
  labels: Array<Label>;
  maxAutomaticAttempts: Scalars['Int'];
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
  attemptAutomatic?: InputMaybe<Scalars['Boolean']>;
  id: Scalars['ID'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  maxAutomaticAttempts?: InputMaybe<Scalars['Int']>;
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
  __typename: 'RunRoutineScheduleEdge';
  cursor: Scalars['String'];
  node: RunRoutineSchedule;
};

export type RunRoutineScheduleSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  labelId?: InputMaybe<Array<Scalars['ID']>>;
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
  __typename: 'RunRoutineScheduleSearchResult';
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
  __typename: 'RunRoutineScheduleTranslation';
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
  attemptAutomatic?: InputMaybe<Scalars['Boolean']>;
  id: Scalars['ID'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  labelsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  maxAutomaticAttempts?: InputMaybe<Scalars['Int']>;
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
  routineVersionId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RunRoutineSortBy>;
  startedTimeFrame?: InputMaybe<TimeFrame>;
  status?: InputMaybe<RunStatus>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type RunRoutineSearchResult = {
  __typename: 'RunRoutineSearchResult';
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
  StepsAsc = 'StepsAsc',
  StepsDesc = 'StepsDesc'
}

export type RunRoutineStep = {
  __typename: 'RunRoutineStep';
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
  subroutine?: Maybe<RoutineVersion>;
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

export type RunRoutineYou = {
  __typename: 'RunRoutineYou';
  canDelete: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
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
  __typename: 'Session';
  isLoggedIn: Scalars['Boolean'];
  timeZone?: Maybe<Scalars['String']>;
  users?: Maybe<Array<SessionUser>>;
};

export type SessionUser = {
  __typename: 'SessionUser';
  handle?: Maybe<Scalars['String']>;
  hasPremium: Scalars['Boolean'];
  id: Scalars['String'];
  languages: Array<Scalars['String']>;
  name?: Maybe<Scalars['String']>;
  schedules: Array<UserSchedule>;
  theme?: Maybe<Scalars['String']>;
};

export type SmartContract = {
  __typename: 'SmartContract';
  completedAt?: Maybe<Scalars['Date']>;
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  hasCompleteVersion: Scalars['Boolean'];
  id: Scalars['ID'];
  isDeleted: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  issues: Array<Issue>;
  issuesCount: Scalars['Int'];
  labels: Array<Label>;
  owner?: Maybe<Owner>;
  parent?: Maybe<SmartContractVersion>;
  permissions: Scalars['String'];
  pullRequests: Array<PullRequest>;
  pullRequestsCount: Scalars['Int'];
  questions: Array<Question>;
  questionsCount: Scalars['Int'];
  score: Scalars['Int'];
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
  stats: Array<StatsSmartContract>;
  tags: Array<Tag>;
  transfers: Array<Transfer>;
  transfersCount: Scalars['Int'];
  translatedName: Scalars['String'];
  updated_at: Scalars['Date'];
  versions: Array<SmartContractVersion>;
  versionsCount?: Maybe<Scalars['Int']>;
  views: Scalars['Int'];
  you: SmartContractYou;
};

export type SmartContractCreateInput = {
  id: Scalars['ID'];
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  organizationConnect?: InputMaybe<Scalars['ID']>;
  parentConnect?: InputMaybe<Scalars['ID']>;
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  userConnect?: InputMaybe<Scalars['ID']>;
  versionsCreate?: InputMaybe<Array<SmartContractVersionCreateInput>>;
};

export type SmartContractEdge = {
  __typename: 'SmartContractEdge';
  cursor: Scalars['String'];
  node: SmartContract;
};

export type SmartContractSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  hasCompleteVersion?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  issuesId?: InputMaybe<Scalars['ID']>;
  labelsIds?: InputMaybe<Array<Scalars['ID']>>;
  maxScore?: InputMaybe<Scalars['Int']>;
  maxSaves?: InputMaybe<Scalars['Int']>;
  maxViews?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  ownedByOrganizationId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  pullRequestsId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<SmartContractSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguagesLatestVersion?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type SmartContractSearchResult = {
  __typename: 'SmartContractSearchResult';
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc'
}

export type SmartContractUpdateInput = {
  id: Scalars['ID'];
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
  versionsCreate?: InputMaybe<Array<SmartContractVersionCreateInput>>;
  versionsDelete?: InputMaybe<Array<Scalars['ID']>>;
  versionsUpdate?: InputMaybe<Array<SmartContractVersionUpdateInput>>;
};

export type SmartContractVersion = {
  __typename: 'SmartContractVersion';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  content: Scalars['String'];
  contractType: Scalars['String'];
  created_at: Scalars['Date'];
  default?: Maybe<Scalars['String']>;
  directoryListings: Array<ProjectVersionDirectory>;
  directoryListingsCount: Scalars['Int'];
  forks: Array<SmartContract>;
  forksCount: Scalars['Int'];
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isDeleted: Scalars['Boolean'];
  isLatest: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  pullRequest?: Maybe<PullRequest>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceList?: Maybe<ResourceList>;
  root: SmartContract;
  translations: Array<SmartContractVersionTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  versionIndex: Scalars['Int'];
  versionLabel: Scalars['String'];
  versionNotes?: Maybe<Scalars['String']>;
  you: VersionYou;
};

export type SmartContractVersionCreateInput = {
  content: Scalars['String'];
  contractType: Scalars['String'];
  default?: InputMaybe<Scalars['String']>;
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isLatest?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  rootConnect: Scalars['ID'];
  rootCreate?: InputMaybe<SmartContractCreateInput>;
  translationsCreate?: InputMaybe<Array<SmartContractVersionTranslationCreateInput>>;
  versionLabel: Scalars['String'];
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type SmartContractVersionEdge = {
  __typename: 'SmartContractVersionEdge';
  cursor: Scalars['String'];
  node: SmartContractVersion;
};

export type SmartContractVersionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  completedTimeFrame?: InputMaybe<TimeFrame>;
  contractType?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  reportId?: InputMaybe<Scalars['ID']>;
  rootId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<SmartContractVersionSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type SmartContractVersionSearchResult = {
  __typename: 'SmartContractVersionSearchResult';
  edges: Array<SmartContractVersionEdge>;
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
  __typename: 'SmartContractVersionTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  jsonVariable?: Maybe<Scalars['String']>;
  language: Scalars['String'];
  name: Scalars['String'];
};

export type SmartContractVersionTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  jsonVariable?: InputMaybe<Scalars['String']>;
  language: Scalars['String'];
  name: Scalars['String'];
};

export type SmartContractVersionTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  jsonVariable?: InputMaybe<Scalars['String']>;
  language?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type SmartContractVersionUpdateInput = {
  content?: InputMaybe<Scalars['String']>;
  contractType?: InputMaybe<Scalars['String']>;
  default?: InputMaybe<Scalars['String']>;
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  directoryListingsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isLatest?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
  rootUpdate?: InputMaybe<SmartContractUpdateInput>;
  translationsCreate?: InputMaybe<Array<SmartContractVersionTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<SmartContractVersionTranslationUpdateInput>>;
  versionLabel?: InputMaybe<Scalars['String']>;
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type SmartContractYou = {
  __typename: 'SmartContractYou';
  canDelete: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canBookmark: Scalars['Boolean'];
  canTransfer: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
};

export type Standard = {
  __typename: 'Standard';
  completedAt?: Maybe<Scalars['Date']>;
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  forks: Array<Standard>;
  forksCount: Scalars['Int'];
  hasCompleteVersion: Scalars['Boolean'];
  id: Scalars['ID'];
  isDeleted: Scalars['Boolean'];
  isInternal: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  issues: Array<Issue>;
  issuesCount: Scalars['Int'];
  labels: Array<Label>;
  name: Scalars['String'];
  owner?: Maybe<Owner>;
  parent?: Maybe<StandardVersion>;
  permissions: Scalars['String'];
  pullRequests: Array<PullRequest>;
  pullRequestsCount: Scalars['Int'];
  questions: Array<Question>;
  questionsCount: Scalars['Int'];
  score: Scalars['Int'];
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
  stats: Array<StatsStandard>;
  tags: Array<Tag>;
  transfers: Array<Transfer>;
  transfersCount: Scalars['Int'];
  translatedName: Scalars['String'];
  updated_at: Scalars['Date'];
  versions: Array<StandardVersion>;
  versionsCount?: Maybe<Scalars['Int']>;
  views: Scalars['Int'];
  you: StandardYou;
};

export type StandardCreateInput = {
  id: Scalars['ID'];
  isInternal?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  name: Scalars['String'];
  organizationConnect?: InputMaybe<Scalars['ID']>;
  parentConnect?: InputMaybe<Scalars['ID']>;
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  userConnect?: InputMaybe<Scalars['ID']>;
  versionsCreate?: InputMaybe<Array<StandardVersionCreateInput>>;
};

export type StandardEdge = {
  __typename: 'StandardEdge';
  cursor: Scalars['String'];
  node: Standard;
};

export type StandardSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  hasCompleteVersion?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isInternal?: InputMaybe<Scalars['Boolean']>;
  issuesId?: InputMaybe<Scalars['ID']>;
  labelsIds?: InputMaybe<Array<Scalars['ID']>>;
  maxScore?: InputMaybe<Scalars['Int']>;
  maxSaves?: InputMaybe<Scalars['Int']>;
  maxViews?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  ownedByOrganizationId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  pullRequestsId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StandardSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguagesLatestVersion?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type StandardSearchResult = {
  __typename: 'StandardSearchResult';
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc'
}

export type StandardUpdateInput = {
  id: Scalars['ID'];
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
  versionsCreate?: InputMaybe<Array<StandardVersionCreateInput>>;
  versionsDelete?: InputMaybe<Array<Scalars['ID']>>;
  versionsUpdate?: InputMaybe<Array<StandardVersionUpdateInput>>;
};

export type StandardVersion = {
  __typename: 'StandardVersion';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  created_at: Scalars['Date'];
  default?: Maybe<Scalars['String']>;
  directoryListings: Array<ProjectVersionDirectory>;
  directoryListingsCount: Scalars['Int'];
  forks: Array<Standard>;
  forksCount: Scalars['Int'];
  id: Scalars['ID'];
  isComplete: Scalars['Boolean'];
  isDeleted: Scalars['Boolean'];
  isFile?: Maybe<Scalars['Boolean']>;
  isLatest: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  props: Scalars['String'];
  pullRequest?: Maybe<PullRequest>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceList?: Maybe<ResourceList>;
  root: Standard;
  standardType: Scalars['String'];
  translations: Array<StandardVersionTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  versionIndex: Scalars['Int'];
  versionLabel: Scalars['String'];
  versionNotes?: Maybe<Scalars['String']>;
  you: VersionYou;
  yup?: Maybe<Scalars['String']>;
};

export type StandardVersionCreateInput = {
  default?: InputMaybe<Scalars['String']>;
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isFile?: InputMaybe<Scalars['Boolean']>;
  isLatest?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  props: Scalars['String'];
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  rootConnect: Scalars['ID'];
  rootCreate?: InputMaybe<StandardCreateInput>;
  standardType: Scalars['String'];
  translationsCreate?: InputMaybe<Array<StandardVersionTranslationCreateInput>>;
  versionLabel: Scalars['String'];
  versionNotes?: InputMaybe<Scalars['String']>;
  yup?: InputMaybe<Scalars['String']>;
};

export type StandardVersionEdge = {
  __typename: 'StandardVersionEdge';
  cursor: Scalars['String'];
  node: StandardVersion;
};

export type StandardVersionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  completedTimeFrame?: InputMaybe<TimeFrame>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
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
  __typename: 'StandardVersionSearchResult';
  edges: Array<StandardVersionEdge>;
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
  __typename: 'StandardVersionTranslation';
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
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  directoryListingsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isFile?: InputMaybe<Scalars['Boolean']>;
  isLatest?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  props?: InputMaybe<Scalars['String']>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
  rootUpdate?: InputMaybe<StandardUpdateInput>;
  standardType?: InputMaybe<Scalars['String']>;
  translationsCreate?: InputMaybe<Array<StandardVersionTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<StandardVersionTranslationUpdateInput>>;
  versionLabel?: InputMaybe<Scalars['String']>;
  versionNotes?: InputMaybe<Scalars['String']>;
  yup?: InputMaybe<Scalars['String']>;
};

export type StandardYou = {
  __typename: 'StandardYou';
  canDelete: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canBookmark: Scalars['Boolean'];
  canTransfer: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  canVote: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
};

export type Bookmark = {
  __typename: 'Bookmark';
  by: User;
  id: Scalars['ID'];
  to: BookmarkTo;
};

export type BookmarkEdge = {
  __typename: 'Bookmark';
  cursor: Scalars['String'];
  node: Bookmark;
};

export enum BookmarkFor {
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
  starFor: BookmarkFor;
};

export type StarSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  excludeLinkedToTag?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<BookmarkSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type BookmarkSearchResult = {
  __typename: 'BookmarkSearchResult';
  edges: Array<BookmarkEdge>;
  pageInfo: PageInfo;
};

export enum BookmarkSortBy {
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type BookmarkTo = Api | Comment | Issue | Note | Organization | Post | Project | Question | QuestionAnswer | Quiz | Routine | SmartContract | Standard | Tag | User;

export enum StatPeriodType {
  Daily = 'Daily',
  Monthly = 'Monthly',
  Weekly = 'Weekly',
  Yearly = 'Yearly'
}

export type StatsApi = {
  __typename: 'StatsApi';
  calls: Scalars['Int'];
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  periodEnd: Scalars['Date'];
  periodStart: Scalars['Date'];
  periodType: StatPeriodType;
  routineVersions: Scalars['Int'];
};

export type StatsApiEdge = {
  __typename: 'StatsApiEdge';
  cursor: Scalars['String'];
  node: StatsApi;
};

export type StatsApiSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  periodTimeFrame?: InputMaybe<TimeFrame>;
  periodType: StatPeriodType;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StatsApiSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type StatsApiSearchResult = {
  __typename: 'StatsApiSearchResult';
  edges: Array<StatsApiEdge>;
  pageInfo: PageInfo;
};

export enum StatsApiSortBy {
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type StatsOrganization = {
  __typename: 'StatsOrganization';
  apis: Scalars['Int'];
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  members: Scalars['Int'];
  notes: Scalars['Int'];
  periodEnd: Scalars['Date'];
  periodStart: Scalars['Date'];
  periodType: StatPeriodType;
  projects: Scalars['Int'];
  routines: Scalars['Int'];
  runRoutineCompletionTimeAverage: Scalars['Float'];
  runRoutineContextSwitchesAverage: Scalars['Float'];
  runRoutinesCompleted: Scalars['Int'];
  runRoutinesStarted: Scalars['Int'];
  smartContracts: Scalars['Int'];
  standards: Scalars['Int'];
};

export type StatsOrganizationEdge = {
  __typename: 'StatsOrganizationEdge';
  cursor: Scalars['String'];
  node: StatsOrganization;
};

export type StatsOrganizationSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  periodTimeFrame?: InputMaybe<TimeFrame>;
  periodType: StatPeriodType;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StatsOrganizationSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type StatsOrganizationSearchResult = {
  __typename: 'StatsOrganizationSearchResult';
  edges: Array<StatsOrganizationEdge>;
  pageInfo: PageInfo;
};

export enum StatsOrganizationSortBy {
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type StatsProject = {
  __typename: 'StatsProject';
  apis: Scalars['Int'];
  created_at: Scalars['Date'];
  directories: Scalars['Int'];
  id: Scalars['ID'];
  notes: Scalars['Int'];
  organizations: Scalars['Int'];
  periodEnd: Scalars['Date'];
  periodStart: Scalars['Date'];
  periodType: StatPeriodType;
  projects: Scalars['Int'];
  routines: Scalars['Int'];
  runCompletionTimeAverage: Scalars['Float'];
  runContextSwitchesAverage: Scalars['Float'];
  runsCompleted: Scalars['Int'];
  runsStarted: Scalars['Int'];
  smartContracts: Scalars['Int'];
  standards: Scalars['Int'];
};

export type StatsProjectEdge = {
  __typename: 'StatsProjectEdge';
  cursor: Scalars['String'];
  node: StatsProject;
};

export type StatsProjectSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  periodTimeFrame?: InputMaybe<TimeFrame>;
  periodType: StatPeriodType;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StatsProjectSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type StatsProjectSearchResult = {
  __typename: 'StatsProjectSearchResult';
  edges: Array<StatsProjectEdge>;
  pageInfo: PageInfo;
};

export enum StatsProjectSortBy {
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type StatsQuiz = {
  __typename: 'StatsQuiz';
  completionTimeAverage: Scalars['Float'];
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  periodEnd: Scalars['Date'];
  periodStart: Scalars['Date'];
  periodType: StatPeriodType;
  scoreAverage: Scalars['Float'];
  timesFailed: Scalars['Int'];
  timesPassed: Scalars['Int'];
  timesStarted: Scalars['Int'];
};

export type StatsQuizEdge = {
  __typename: 'StatsQuizEdge';
  cursor: Scalars['String'];
  node: StatsQuiz;
};

export type StatsQuizSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  periodTimeFrame?: InputMaybe<TimeFrame>;
  periodType: StatPeriodType;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StatsQuizSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type StatsQuizSearchResult = {
  __typename: 'StatsQuizSearchResult';
  edges: Array<StatsQuizEdge>;
  pageInfo: PageInfo;
};

export enum StatsQuizSortBy {
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type StatsRoutine = {
  __typename: 'StatsRoutine';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  periodEnd: Scalars['Date'];
  periodStart: Scalars['Date'];
  periodType: StatPeriodType;
  runCompletionTimeAverage: Scalars['Float'];
  runContextSwitchesAverage: Scalars['Float'];
  runsCompleted: Scalars['Int'];
  runsStarted: Scalars['Int'];
};

export type StatsRoutineEdge = {
  __typename: 'StatsRoutineEdge';
  cursor: Scalars['String'];
  node: StatsRoutine;
};

export type StatsRoutineSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  periodTimeFrame?: InputMaybe<TimeFrame>;
  periodType: StatPeriodType;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StatsRoutineSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type StatsRoutineSearchResult = {
  __typename: 'StatsRoutineSearchResult';
  edges: Array<StatsRoutineEdge>;
  pageInfo: PageInfo;
};

export enum StatsRoutineSortBy {
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type StatsSite = {
  __typename: 'StatsSite';
  activeUsers: Scalars['Int'];
  apiCalls: Scalars['Int'];
  apisCreated: Scalars['Int'];
  id: Scalars['ID'];
  organizationsCreated: Scalars['Int'];
  periodEnd: Scalars['Date'];
  periodStart: Scalars['Date'];
  periodType: StatPeriodType;
  projectCompletionTimeAverage: Scalars['Float'];
  projectsCompleted: Scalars['Int'];
  projectsCreated: Scalars['Int'];
  quizzesCompleted: Scalars['Int'];
  quizzesCreated: Scalars['Int'];
  routineCompletionTimeAverage: Scalars['Float'];
  routineComplexityAverage: Scalars['Float'];
  routineSimplicityAverage: Scalars['Float'];
  routinesCompleted: Scalars['Int'];
  routinesCreated: Scalars['Int'];
  runProjectCompletionTimeAverage: Scalars['Float'];
  runProjectContextSwitchesAverage: Scalars['Float'];
  runProjectsCompleted: Scalars['Int'];
  runProjectsStarted: Scalars['Int'];
  runRoutineCompletionTimeAverage: Scalars['Float'];
  runRoutineContextSwitchesAverage: Scalars['Float'];
  runRoutinesCompleted: Scalars['Int'];
  runRoutinesStarted: Scalars['Int'];
  smartContractCalls: Scalars['Int'];
  smartContractCompletionTimeAverage: Scalars['Float'];
  smartContractsCompleted: Scalars['Int'];
  smartContractsCreated: Scalars['Int'];
  standardCompletionTimeAverage: Scalars['Float'];
  standardsCompleted: Scalars['Int'];
  standardsCreated: Scalars['Int'];
  verifiedEmailsCreated: Scalars['Int'];
  verifiedWalletsCreated: Scalars['Int'];
};

export type StatsSiteEdge = {
  __typename: 'StatsSiteEdge';
  cursor: Scalars['String'];
  node: StatsSite;
};

export type StatsSiteSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  periodTimeFrame?: InputMaybe<TimeFrame>;
  periodType: StatPeriodType;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StatsSiteSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type StatsSiteSearchResult = {
  __typename: 'StatsSiteSearchResult';
  edges: Array<StatsSiteEdge>;
  pageInfo: PageInfo;
};

export enum StatsSiteSortBy {
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type StatsSmartContract = {
  __typename: 'StatsSmartContract';
  calls: Scalars['Int'];
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  periodEnd: Scalars['Date'];
  periodStart: Scalars['Date'];
  periodType: StatPeriodType;
  routineVersions: Scalars['Int'];
};

export type StatsSmartContractEdge = {
  __typename: 'StatsSmartContractEdge';
  cursor: Scalars['String'];
  node: StatsSmartContract;
};

export type StatsSmartContractSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  periodTimeFrame?: InputMaybe<TimeFrame>;
  periodType: StatPeriodType;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StatsSmartContractSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type StatsSmartContractSearchResult = {
  __typename: 'StatsSmartContractSearchResult';
  edges: Array<StatsSmartContractEdge>;
  pageInfo: PageInfo;
};

export enum StatsSmartContractSortBy {
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type StatsStandard = {
  __typename: 'StatsStandard';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  linksToInputs: Scalars['Int'];
  linksToOutputs: Scalars['Int'];
  periodEnd: Scalars['Date'];
  periodStart: Scalars['Date'];
  periodType: StatPeriodType;
};

export type StatsStandardEdge = {
  __typename: 'StatsStandardEdge';
  cursor: Scalars['String'];
  node: StatsStandard;
};

export type StatsStandardSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  periodTimeFrame?: InputMaybe<TimeFrame>;
  periodType: StatPeriodType;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StatsStandardSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type StatsStandardSearchResult = {
  __typename: 'StatsStandardSearchResult';
  edges: Array<StatsStandardEdge>;
  pageInfo: PageInfo;
};

export enum StatsStandardSortBy {
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type StatsUser = {
  __typename: 'StatsUser';
  apisCreated: Scalars['Int'];
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  organizationsCreated: Scalars['Int'];
  periodEnd: Scalars['Date'];
  periodStart: Scalars['Date'];
  periodType: StatPeriodType;
  projectCompletionTimeAverage: Scalars['Float'];
  projectsCompleted: Scalars['Int'];
  projectsCreated: Scalars['Int'];
  quizzesFailed: Scalars['Int'];
  quizzesPassed: Scalars['Int'];
  routineCompletionTimeAverage: Scalars['Float'];
  routinesCompleted: Scalars['Int'];
  routinesCreated: Scalars['Int'];
  runProjectCompletionTimeAverage: Scalars['Float'];
  runProjectContextSwitchesAverage: Scalars['Float'];
  runProjectsCompleted: Scalars['Int'];
  runProjectsStarted: Scalars['Int'];
  runRoutineCompletionTimeAverage: Scalars['Float'];
  runRoutineContextSwitchesAverage: Scalars['Float'];
  runRoutinesCompleted: Scalars['Int'];
  runRoutinesStarted: Scalars['Int'];
  smartContractCompletionTimeAverage: Scalars['Float'];
  smartContractsCompleted: Scalars['Int'];
  smartContractsCreated: Scalars['Int'];
  standardCompletionTimeAverage: Scalars['Float'];
  standardsCompleted: Scalars['Int'];
  standardsCreated: Scalars['Int'];
};

export type StatsUserEdge = {
  __typename: 'StatsUserEdge';
  cursor: Scalars['String'];
  node: StatsUser;
};

export type StatsUserSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  periodTimeFrame?: InputMaybe<TimeFrame>;
  periodType: StatPeriodType;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StatsUserSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type StatsUserSearchResult = {
  __typename: 'StatsUserSearchResult';
  edges: Array<StatsUserEdge>;
  pageInfo: PageInfo;
};

export enum StatsUserSortBy {
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

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
  __typename: 'Success';
  success: Scalars['Boolean'];
};

export type SwitchCurrentAccountInput = {
  id: Scalars['ID'];
};

export type Tag = {
  __typename: 'Tag';
  apis: Array<Api>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  notes: Array<Note>;
  organizations: Array<Organization>;
  posts: Array<Post>;
  projects: Array<Project>;
  reports: Array<Report>;
  routines: Array<Routine>;
  smartContracts: Array<SmartContract>;
  standards: Array<Standard>;
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
  tag: Scalars['String'];
  translations: Array<TagTranslation>;
  updated_at: Scalars['Date'];
  you: TagYou;
};

export type TagCreateInput = {
  anonymous?: InputMaybe<Scalars['Boolean']>;
  tag: Scalars['String'];
  translationsCreate?: InputMaybe<Array<TagTranslationCreateInput>>;
};

export type TagEdge = {
  __typename: 'TagEdge';
  cursor: Scalars['String'];
  node: Tag;
};

export type TagSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  maxSaves?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<TagSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type TagSearchResult = {
  __typename: 'TagSearchResult';
  edges: Array<TagEdge>;
  pageInfo: PageInfo;
};

export enum TagSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc'
}

export type TagTranslation = {
  __typename: 'TagTranslation';
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

export type TagYou = {
  __typename: 'TagYou';
  isOwn: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
};

export type TimeFrame = {
  after?: InputMaybe<Scalars['Date']>;
  before?: InputMaybe<Scalars['Date']>;
};

export type Transfer = {
  __typename: 'Transfer';
  created_at: Scalars['Date'];
  fromOwner?: Maybe<Owner>;
  id: Scalars['ID'];
  mergedOrRejectedAt?: Maybe<Scalars['Date']>;
  object: TransferObject;
  status: TransferStatus;
  toOwner?: Maybe<Owner>;
  updated_at: Scalars['Date'];
  you: TransferYou;
};

export type TransferDenyInput = {
  id: Scalars['ID'];
  reason?: InputMaybe<Scalars['String']>;
};

export type TransferEdge = {
  __typename: 'TransferEdge';
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
  __typename: 'TransferSearchResult';
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

export type TransferYou = {
  __typename: 'TransferYou';
  canDelete: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
};

export type Translate = {
  __typename: 'Translate';
  fields: Scalars['String'];
  language: Scalars['String'];
};

export type TranslateInput = {
  fields: Scalars['String'];
  languageSource: Scalars['String'];
  languageTarget: Scalars['String'];
};

export type User = {
  __typename: 'User';
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
  isPrivateBookmarks: Scalars['Boolean'];
  isPrivateVotes: Scalars['Boolean'];
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
  reportsCreated?: Maybe<Array<Report>>;
  reportsReceived: Array<Report>;
  reportsReceivedCount: Scalars['Int'];
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
  starred?: Maybe<Array<Bookmark>>;
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
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
  you: UserYou;
};

export type UserDeleteInput = {
  deletePublicData: Scalars['Boolean'];
  password: Scalars['String'];
};

export type UserEdge = {
  __typename: 'UserEdge';
  cursor: Scalars['String'];
  node: User;
};

export type UserSchedule = {
  __typename: 'UserSchedule';
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
  __typename: 'UserScheduleEdge';
  cursor: Scalars['String'];
  node: UserSchedule;
};

export type UserScheduleFilter = {
  __typename: 'UserScheduleFilter';
  filterType: UserScheduleFilterType;
  id: Scalars['ID'];
  tag: Tag;
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
  __typename: 'UserScheduleSearchResult';
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
  reminderListDisconnect?: InputMaybe<Scalars['Boolean']>;
  reminderListUpdate?: InputMaybe<ReminderListUpdateInput>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
  timeZone?: InputMaybe<Scalars['String']>;
};

export type UserSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<UserSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type UserSearchResult = {
  __typename: 'UserSearchResult';
  edges: Array<UserEdge>;
  pageInfo: PageInfo;
};

export enum UserSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc'
}

export type UserTranslation = {
  __typename: 'UserTranslation';
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

export type UserYou = {
  __typename: 'UserYou';
  canDelete: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isViewed: Scalars['Boolean'];
};

export type ValidateSessionInput = {
  timeZone: Scalars['String'];
};

export type VersionYou = {
  __typename: 'VersionYou';
  canComment: Scalars['Boolean'];
  canCopy: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  canUse: Scalars['Boolean'];
};

export type View = {
  __typename: 'View';
  by: User;
  id: Scalars['ID'];
  lastViewedAt: Scalars['Date'];
  name: Scalars['String'];
  to: ViewTo;
};

export type ViewEdge = {
  __typename: 'ViewEdge';
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
  __typename: 'ViewSearchResult';
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
  __typename: 'Vote';
  by: User;
  id: Scalars['ID'];
  isUpvote?: Maybe<Scalars['Boolean']>;
  to: VoteTo;
};

export type VoteEdge = {
  __typename: 'VoteEdge';
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
  __typename: 'VoteSearchResult';
  edges: Array<VoteEdge>;
  pageInfo: PageInfo;
};

export enum VoteSortBy {
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type VoteTo = Api | Comment | Issue | Note | Post | Project | Question | QuestionAnswer | Quiz | Routine | SmartContract | Standard;

export type Wallet = {
  __typename: 'Wallet';
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
  __typename: 'WalletComplete';
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



export type ResolverTypeWrapper<T> = Promise<T> | T;


export type ResolverWithResolve<TResult, TParent, TContext, TArgs> = {
  resolve: ResolverFn<TResult, TParent, TContext, TArgs>;
};
export type Resolver<TResult, TParent = {}, TContext = {}, TArgs = {}> = ResolverFn<TResult, TParent, TContext, TArgs> | ResolverWithResolve<TResult, TParent, TContext, TArgs>;

export type ResolverFn<TResult, TParent, TContext, TArgs> = (
  parent: TParent,
  args: TArgs,
  context: TContext,
  info: GraphQLResolveInfo
) => Promise<TResult> | TResult;

export type SubscriptionSubscribeFn<TResult, TParent, TContext, TArgs> = (
  parent: TParent,
  args: TArgs,
  context: TContext,
  info: GraphQLResolveInfo
) => AsyncIterable<TResult> | Promise<AsyncIterable<TResult>>;

export type SubscriptionResolveFn<TResult, TParent, TContext, TArgs> = (
  parent: TParent,
  args: TArgs,
  context: TContext,
  info: GraphQLResolveInfo
) => TResult | Promise<TResult>;

export interface SubscriptionSubscriberObject<TResult, TKey extends string, TParent, TContext, TArgs> {
  subscribe: SubscriptionSubscribeFn<{ [key in TKey]: TResult }, TParent, TContext, TArgs>;
  resolve?: SubscriptionResolveFn<TResult, { [key in TKey]: TResult }, TContext, TArgs>;
}

export interface SubscriptionResolverObject<TResult, TParent, TContext, TArgs> {
  subscribe: SubscriptionSubscribeFn<any, TParent, TContext, TArgs>;
  resolve: SubscriptionResolveFn<TResult, any, TContext, TArgs>;
}

export type SubscriptionObject<TResult, TKey extends string, TParent, TContext, TArgs> =
  | SubscriptionSubscriberObject<TResult, TKey, TParent, TContext, TArgs>
  | SubscriptionResolverObject<TResult, TParent, TContext, TArgs>;

export type SubscriptionResolver<TResult, TKey extends string, TParent = {}, TContext = {}, TArgs = {}> =
  | ((...args: any[]) => SubscriptionObject<TResult, TKey, TParent, TContext, TArgs>)
  | SubscriptionObject<TResult, TKey, TParent, TContext, TArgs>;

export type TypeResolveFn<TTypes, TParent = {}, TContext = {}> = (
  parent: TParent,
  context: TContext,
  info: GraphQLResolveInfo
) => Maybe<TTypes> | Promise<Maybe<TTypes>>;

export type IsTypeOfResolverFn<T = {}, TContext = {}> = (obj: T, context: TContext, info: GraphQLResolveInfo) => boolean | Promise<boolean>;

export type NextResolverFn<T> = () => Promise<T>;

export type DirectiveResolverFn<TResult = {}, TParent = {}, TContext = {}, TArgs = {}> = (
  next: NextResolverFn<TResult>,
  parent: TParent,
  args: TArgs,
  context: TContext,
  info: GraphQLResolveInfo
) => TResult | Promise<TResult>;

/** Mapping between all available schema types and the resolvers types */
export type ResolversTypes = {
  AccountStatus: AccountStatus;
  AnyRun: ResolversTypes['RunProject'] | ResolversTypes['RunRoutine'];
  Api: ResolverTypeWrapper<Omit<Api, 'owner'> & { owner?: Maybe<ResolversTypes['Owner']> }>;
  ApiCreateInput: ApiCreateInput;
  ApiEdge: ResolverTypeWrapper<ApiEdge>;
  ApiKey: ResolverTypeWrapper<ApiKey>;
  ApiKeyCreateInput: ApiKeyCreateInput;
  ApiKeyDeleteOneInput: ApiKeyDeleteOneInput;
  ApiKeyUpdateInput: ApiKeyUpdateInput;
  ApiKeyValidateInput: ApiKeyValidateInput;
  ApiSearchInput: ApiSearchInput;
  ApiSearchResult: ResolverTypeWrapper<ApiSearchResult>;
  ApiSortBy: ApiSortBy;
  ApiUpdateInput: ApiUpdateInput;
  ApiVersion: ResolverTypeWrapper<ApiVersion>;
  ApiVersionCreateInput: ApiVersionCreateInput;
  ApiVersionEdge: ResolverTypeWrapper<ApiVersionEdge>;
  ApiVersionSearchInput: ApiVersionSearchInput;
  ApiVersionSearchResult: ResolverTypeWrapper<ApiVersionSearchResult>;
  ApiVersionSortBy: ApiVersionSortBy;
  ApiVersionTranslation: ResolverTypeWrapper<ApiVersionTranslation>;
  ApiVersionTranslationCreateInput: ApiVersionTranslationCreateInput;
  ApiVersionTranslationUpdateInput: ApiVersionTranslationUpdateInput;
  ApiVersionUpdateInput: ApiVersionUpdateInput;
  ApiYou: ResolverTypeWrapper<ApiYou>;
  Award: ResolverTypeWrapper<Award>;
  AwardCategory: AwardCategory;
  Boolean: ResolverTypeWrapper<Scalars['Boolean']>;
  Comment: ResolverTypeWrapper<Omit<Comment, 'commentedOn' | 'owner'> & { commentedOn: ResolversTypes['CommentedOn'], owner?: Maybe<ResolversTypes['Owner']> }>;
  CommentCreateInput: CommentCreateInput;
  CommentFor: CommentFor;
  CommentSearchInput: CommentSearchInput;
  CommentSearchResult: ResolverTypeWrapper<CommentSearchResult>;
  CommentSortBy: CommentSortBy;
  CommentThread: ResolverTypeWrapper<CommentThread>;
  CommentTranslation: ResolverTypeWrapper<CommentTranslation>;
  CommentTranslationCreateInput: CommentTranslationCreateInput;
  CommentTranslationUpdateInput: CommentTranslationUpdateInput;
  CommentUpdateInput: CommentUpdateInput;
  CommentYou: ResolverTypeWrapper<CommentYou>;
  CommentedOn: ResolversTypes['ApiVersion'] | ResolversTypes['Issue'] | ResolversTypes['NoteVersion'] | ResolversTypes['Post'] | ResolversTypes['ProjectVersion'] | ResolversTypes['PullRequest'] | ResolversTypes['Question'] | ResolversTypes['QuestionAnswer'] | ResolversTypes['RoutineVersion'] | ResolversTypes['SmartContractVersion'] | ResolversTypes['StandardVersion'];
  CopyInput: CopyInput;
  CopyResult: ResolverTypeWrapper<CopyResult>;
  CopyType: CopyType;
  Count: ResolverTypeWrapper<Count>;
  Date: ResolverTypeWrapper<Scalars['Date']>;
  DeleteManyInput: DeleteManyInput;
  DeleteOneInput: DeleteOneInput;
  DeleteType: DeleteType;
  DevelopResult: ResolverTypeWrapper<Omit<DevelopResult, 'completed' | 'inProgress' | 'recent'> & { completed: Array<ResolversTypes['ProjectOrRoutine']>, inProgress: Array<ResolversTypes['ProjectOrRoutine']>, recent: Array<ResolversTypes['ProjectOrRoutine']> }>;
  Email: ResolverTypeWrapper<Email>;
  EmailCreateInput: EmailCreateInput;
  EmailLogInInput: EmailLogInInput;
  EmailRequestPasswordChangeInput: EmailRequestPasswordChangeInput;
  EmailResetPasswordInput: EmailResetPasswordInput;
  EmailSignUpInput: EmailSignUpInput;
  FindByIdInput: FindByIdInput;
  FindByIdOrHandleInput: FindByIdOrHandleInput;
  FindHandlesInput: FindHandlesInput;
  FindVersionInput: FindVersionInput;
  Float: ResolverTypeWrapper<Scalars['Float']>;
  GqlModelType: GqlModelType;
  Handle: ResolverTypeWrapper<Handle>;
  HistoryInput: HistoryInput;
  HistoryResult: ResolverTypeWrapper<Omit<HistoryResult, 'activeRuns' | 'completedRuns'> & { activeRuns: Array<ResolversTypes['AnyRun']>, completedRuns: Array<ResolversTypes['AnyRun']> }>;
  ID: ResolverTypeWrapper<Scalars['ID']>;
  Int: ResolverTypeWrapper<Scalars['Int']>;
  Issue: ResolverTypeWrapper<Omit<Issue, 'to'> & { to: ResolversTypes['IssueTo'] }>;
  IssueCloseInput: IssueCloseInput;
  IssueCreateInput: IssueCreateInput;
  IssueEdge: ResolverTypeWrapper<IssueEdge>;
  IssueFor: IssueFor;
  IssueSearchInput: IssueSearchInput;
  IssueSearchResult: ResolverTypeWrapper<IssueSearchResult>;
  IssueSortBy: IssueSortBy;
  IssueStatus: IssueStatus;
  IssueTo: ResolversTypes['Api'] | ResolversTypes['Note'] | ResolversTypes['Organization'] | ResolversTypes['Project'] | ResolversTypes['Routine'] | ResolversTypes['SmartContract'] | ResolversTypes['Standard'];
  IssueTranslation: ResolverTypeWrapper<IssueTranslation>;
  IssueTranslationCreateInput: IssueTranslationCreateInput;
  IssueTranslationUpdateInput: IssueTranslationUpdateInput;
  IssueUpdateInput: IssueUpdateInput;
  IssueYou: ResolverTypeWrapper<IssueYou>;
  Label: ResolverTypeWrapper<Omit<Label, 'owner'> & { owner: ResolversTypes['Owner'] }>;
  LabelCreateInput: LabelCreateInput;
  LabelEdge: ResolverTypeWrapper<LabelEdge>;
  LabelSearchInput: LabelSearchInput;
  LabelSearchResult: ResolverTypeWrapper<LabelSearchResult>;
  LabelSortBy: LabelSortBy;
  LabelTranslation: ResolverTypeWrapper<LabelTranslation>;
  LabelTranslationCreateInput: LabelTranslationCreateInput;
  LabelTranslationUpdateInput: LabelTranslationUpdateInput;
  LabelUpdateInput: LabelUpdateInput;
  LabelYou: ResolverTypeWrapper<LabelYou>;
  LearnResult: ResolverTypeWrapper<LearnResult>;
  LogOutInput: LogOutInput;
  Meeting: ResolverTypeWrapper<Meeting>;
  MeetingCreateInput: MeetingCreateInput;
  MeetingEdge: ResolverTypeWrapper<MeetingEdge>;
  MeetingInvite: ResolverTypeWrapper<MeetingInvite>;
  MeetingInviteCreateInput: MeetingInviteCreateInput;
  MeetingInviteEdge: ResolverTypeWrapper<MeetingInviteEdge>;
  MeetingInviteSearchInput: MeetingInviteSearchInput;
  MeetingInviteSearchResult: ResolverTypeWrapper<MeetingInviteSearchResult>;
  MeetingInviteSortBy: MeetingInviteSortBy;
  MeetingInviteStatus: MeetingInviteStatus;
  MeetingInviteUpdateInput: MeetingInviteUpdateInput;
  MeetingInviteYou: ResolverTypeWrapper<MeetingInviteYou>;
  MeetingSearchInput: MeetingSearchInput;
  MeetingSearchResult: ResolverTypeWrapper<MeetingSearchResult>;
  MeetingSortBy: MeetingSortBy;
  MeetingTranslation: ResolverTypeWrapper<MeetingTranslation>;
  MeetingTranslationCreateInput: MeetingTranslationCreateInput;
  MeetingTranslationUpdateInput: MeetingTranslationUpdateInput;
  MeetingUpdateInput: MeetingUpdateInput;
  MeetingYou: ResolverTypeWrapper<MeetingYou>;
  Member: ResolverTypeWrapper<Member>;
  MemberEdge: ResolverTypeWrapper<MemberEdge>;
  MemberInvite: ResolverTypeWrapper<MemberInvite>;
  MemberInviteCreateInput: MemberInviteCreateInput;
  MemberInviteEdge: ResolverTypeWrapper<MemberInviteEdge>;
  MemberInviteSearchInput: MemberInviteSearchInput;
  MemberInviteSearchResult: ResolverTypeWrapper<MemberInviteSearchResult>;
  MemberInviteSortBy: MemberInviteSortBy;
  MemberInviteStatus: MemberInviteStatus;
  MemberInviteUpdateInput: MemberInviteUpdateInput;
  MemberInviteYou: ResolverTypeWrapper<MemberInviteYou>;
  MemberSearchInput: MemberSearchInput;
  MemberSearchResult: ResolverTypeWrapper<MemberSearchResult>;
  MemberSortBy: MemberSortBy;
  MemberUpdateInput: MemberUpdateInput;
  Mutation: ResolverTypeWrapper<{}>;
  Node: ResolverTypeWrapper<Node>;
  NodeCreateInput: NodeCreateInput;
  NodeEnd: ResolverTypeWrapper<NodeEnd>;
  NodeEndCreateInput: NodeEndCreateInput;
  NodeEndUpdateInput: NodeEndUpdateInput;
  NodeLink: ResolverTypeWrapper<NodeLink>;
  NodeLinkCreateInput: NodeLinkCreateInput;
  NodeLinkUpdateInput: NodeLinkUpdateInput;
  NodeLinkWhen: ResolverTypeWrapper<NodeLinkWhen>;
  NodeLinkWhenCreateInput: NodeLinkWhenCreateInput;
  NodeLinkWhenTranslation: ResolverTypeWrapper<NodeLinkWhenTranslation>;
  NodeLinkWhenTranslationCreateInput: NodeLinkWhenTranslationCreateInput;
  NodeLinkWhenTranslationUpdateInput: NodeLinkWhenTranslationUpdateInput;
  NodeLinkWhenUpdateInput: NodeLinkWhenUpdateInput;
  NodeLoop: ResolverTypeWrapper<NodeLoop>;
  NodeLoopCreateInput: NodeLoopCreateInput;
  NodeLoopUpdateInput: NodeLoopUpdateInput;
  NodeLoopWhile: ResolverTypeWrapper<NodeLoopWhile>;
  NodeLoopWhileCreateInput: NodeLoopWhileCreateInput;
  NodeLoopWhileTranslation: ResolverTypeWrapper<NodeLoopWhileTranslation>;
  NodeLoopWhileTranslationCreateInput: NodeLoopWhileTranslationCreateInput;
  NodeLoopWhileTranslationUpdateInput: NodeLoopWhileTranslationUpdateInput;
  NodeLoopWhileUpdateInput: NodeLoopWhileUpdateInput;
  NodeRoutineList: ResolverTypeWrapper<NodeRoutineList>;
  NodeRoutineListCreateInput: NodeRoutineListCreateInput;
  NodeRoutineListItem: ResolverTypeWrapper<NodeRoutineListItem>;
  NodeRoutineListItemCreateInput: NodeRoutineListItemCreateInput;
  NodeRoutineListItemTranslation: ResolverTypeWrapper<NodeRoutineListItemTranslation>;
  NodeRoutineListItemTranslationCreateInput: NodeRoutineListItemTranslationCreateInput;
  NodeRoutineListItemTranslationUpdateInput: NodeRoutineListItemTranslationUpdateInput;
  NodeRoutineListItemUpdateInput: NodeRoutineListItemUpdateInput;
  NodeRoutineListUpdateInput: NodeRoutineListUpdateInput;
  NodeTranslation: ResolverTypeWrapper<NodeTranslation>;
  NodeTranslationCreateInput: NodeTranslationCreateInput;
  NodeTranslationUpdateInput: NodeTranslationUpdateInput;
  NodeType: NodeType;
  NodeUpdateInput: NodeUpdateInput;
  Note: ResolverTypeWrapper<Omit<Note, 'owner'> & { owner?: Maybe<ResolversTypes['Owner']> }>;
  NoteCreateInput: NoteCreateInput;
  NoteEdge: ResolverTypeWrapper<NoteEdge>;
  NoteSearchInput: NoteSearchInput;
  NoteSearchResult: ResolverTypeWrapper<NoteSearchResult>;
  NoteSortBy: NoteSortBy;
  NoteUpdateInput: NoteUpdateInput;
  NoteVersion: ResolverTypeWrapper<NoteVersion>;
  NoteVersionCreateInput: NoteVersionCreateInput;
  NoteVersionEdge: ResolverTypeWrapper<NoteVersionEdge>;
  NoteVersionSearchInput: NoteVersionSearchInput;
  NoteVersionSearchResult: ResolverTypeWrapper<NoteVersionSearchResult>;
  NoteVersionSortBy: NoteVersionSortBy;
  NoteVersionTranslation: ResolverTypeWrapper<NoteVersionTranslation>;
  NoteVersionTranslationCreateInput: NoteVersionTranslationCreateInput;
  NoteVersionTranslationUpdateInput: NoteVersionTranslationUpdateInput;
  NoteVersionUpdateInput: NoteVersionUpdateInput;
  NoteYou: ResolverTypeWrapper<NoteYou>;
  Notification: ResolverTypeWrapper<Notification>;
  NotificationEdge: ResolverTypeWrapper<NotificationEdge>;
  NotificationSearchInput: NotificationSearchInput;
  NotificationSearchResult: ResolverTypeWrapper<NotificationSearchResult>;
  NotificationSettings: ResolverTypeWrapper<NotificationSettings>;
  NotificationSettingsCategory: ResolverTypeWrapper<NotificationSettingsCategory>;
  NotificationSettingsCategoryUpdateInput: NotificationSettingsCategoryUpdateInput;
  NotificationSettingsUpdateInput: NotificationSettingsUpdateInput;
  NotificationSortBy: NotificationSortBy;
  NotificationSubscription: ResolverTypeWrapper<Omit<NotificationSubscription, 'object'> & { object: ResolversTypes['SubscribedObject'] }>;
  NotificationSubscriptionCreateInput: NotificationSubscriptionCreateInput;
  NotificationSubscriptionEdge: ResolverTypeWrapper<NotificationSubscriptionEdge>;
  NotificationSubscriptionSearchInput: NotificationSubscriptionSearchInput;
  NotificationSubscriptionSearchResult: ResolverTypeWrapper<NotificationSubscriptionSearchResult>;
  NotificationSubscriptionSortBy: NotificationSubscriptionSortBy;
  NotificationSubscriptionUpdateInput: NotificationSubscriptionUpdateInput;
  Organization: ResolverTypeWrapper<Organization>;
  OrganizationCreateInput: OrganizationCreateInput;
  OrganizationEdge: ResolverTypeWrapper<OrganizationEdge>;
  OrganizationSearchInput: OrganizationSearchInput;
  OrganizationSearchResult: ResolverTypeWrapper<OrganizationSearchResult>;
  OrganizationSortBy: OrganizationSortBy;
  OrganizationTranslation: ResolverTypeWrapper<OrganizationTranslation>;
  OrganizationTranslationCreateInput: OrganizationTranslationCreateInput;
  OrganizationTranslationUpdateInput: OrganizationTranslationUpdateInput;
  OrganizationUpdateInput: OrganizationUpdateInput;
  OrganizationYou: ResolverTypeWrapper<OrganizationYou>;
  Owner: ResolversTypes['Organization'] | ResolversTypes['User'];
  PageInfo: ResolverTypeWrapper<PageInfo>;
  Payment: ResolverTypeWrapper<Payment>;
  PaymentStatus: PaymentStatus;
  Phone: ResolverTypeWrapper<Phone>;
  PhoneCreateInput: PhoneCreateInput;
  PopularInput: PopularInput;
  PopularResult: ResolverTypeWrapper<PopularResult>;
  Post: ResolverTypeWrapper<Omit<Post, 'owner'> & { owner: ResolversTypes['Owner'] }>;
  PostCreateInput: PostCreateInput;
  PostEdge: ResolverTypeWrapper<PostEdge>;
  PostSearchInput: PostSearchInput;
  PostSearchResult: ResolverTypeWrapper<PostSearchResult>;
  PostSortBy: PostSortBy;
  PostTranslation: ResolverTypeWrapper<PostTranslation>;
  PostTranslationCreateInput: PostTranslationCreateInput;
  PostTranslationUpdateInput: PostTranslationUpdateInput;
  PostUpdateInput: PostUpdateInput;
  Premium: ResolverTypeWrapper<Premium>;
  ProfileEmailUpdateInput: ProfileEmailUpdateInput;
  ProfileUpdateInput: ProfileUpdateInput;
  Project: ResolverTypeWrapper<Omit<Project, 'owner'> & { owner?: Maybe<ResolversTypes['Owner']> }>;
  ProjectCreateInput: ProjectCreateInput;
  ProjectEdge: ResolverTypeWrapper<ProjectEdge>;
  ProjectOrOrganization: ResolversTypes['Organization'] | ResolversTypes['Project'];
  ProjectOrOrganizationEdge: ResolverTypeWrapper<Omit<ProjectOrOrganizationEdge, 'node'> & { node: ResolversTypes['ProjectOrOrganization'] }>;
  ProjectOrOrganizationPageInfo: ResolverTypeWrapper<ProjectOrOrganizationPageInfo>;
  ProjectOrOrganizationSearchInput: ProjectOrOrganizationSearchInput;
  ProjectOrOrganizationSearchResult: ResolverTypeWrapper<ProjectOrOrganizationSearchResult>;
  ProjectOrOrganizationSortBy: ProjectOrOrganizationSortBy;
  ProjectOrRoutine: ResolversTypes['Project'] | ResolversTypes['Routine'];
  ProjectOrRoutineEdge: ResolverTypeWrapper<Omit<ProjectOrRoutineEdge, 'node'> & { node: ResolversTypes['ProjectOrRoutine'] }>;
  ProjectOrRoutinePageInfo: ResolverTypeWrapper<ProjectOrRoutinePageInfo>;
  ProjectOrRoutineSearchInput: ProjectOrRoutineSearchInput;
  ProjectOrRoutineSearchResult: ResolverTypeWrapper<ProjectOrRoutineSearchResult>;
  ProjectOrRoutineSortBy: ProjectOrRoutineSortBy;
  ProjectSearchInput: ProjectSearchInput;
  ProjectSearchResult: ResolverTypeWrapper<ProjectSearchResult>;
  ProjectSortBy: ProjectSortBy;
  ProjectUpdateInput: ProjectUpdateInput;
  ProjectVersion: ResolverTypeWrapper<ProjectVersion>;
  ProjectVersionCreateInput: ProjectVersionCreateInput;
  ProjectVersionDirectory: ResolverTypeWrapper<ProjectVersionDirectory>;
  ProjectVersionDirectoryCreateInput: ProjectVersionDirectoryCreateInput;
  ProjectVersionDirectoryTranslation: ResolverTypeWrapper<ProjectVersionDirectoryTranslation>;
  ProjectVersionDirectoryTranslationCreateInput: ProjectVersionDirectoryTranslationCreateInput;
  ProjectVersionDirectoryTranslationUpdateInput: ProjectVersionDirectoryTranslationUpdateInput;
  ProjectVersionDirectoryUpdateInput: ProjectVersionDirectoryUpdateInput;
  ProjectVersionEdge: ResolverTypeWrapper<ProjectVersionEdge>;
  ProjectVersionSearchInput: ProjectVersionSearchInput;
  ProjectVersionSearchResult: ResolverTypeWrapper<ProjectVersionSearchResult>;
  ProjectVersionSortBy: ProjectVersionSortBy;
  ProjectVersionTranslation: ResolverTypeWrapper<ProjectVersionTranslation>;
  ProjectVersionTranslationCreateInput: ProjectVersionTranslationCreateInput;
  ProjectVersionTranslationUpdateInput: ProjectVersionTranslationUpdateInput;
  ProjectVersionUpdateInput: ProjectVersionUpdateInput;
  ProjectVersionYou: ResolverTypeWrapper<ProjectVersionYou>;
  ProjectYou: ResolverTypeWrapper<ProjectYou>;
  PullRequest: ResolverTypeWrapper<Omit<PullRequest, 'from' | 'to'> & { from: ResolversTypes['PullRequestFrom'], to: ResolversTypes['PullRequestTo'] }>;
  PullRequestCreateInput: PullRequestCreateInput;
  PullRequestEdge: ResolverTypeWrapper<PullRequestEdge>;
  PullRequestFrom: ResolversTypes['ApiVersion'] | ResolversTypes['NoteVersion'] | ResolversTypes['ProjectVersion'] | ResolversTypes['RoutineVersion'] | ResolversTypes['SmartContractVersion'] | ResolversTypes['StandardVersion'];
  PullRequestSearchInput: PullRequestSearchInput;
  PullRequestSearchResult: ResolverTypeWrapper<PullRequestSearchResult>;
  PullRequestSortBy: PullRequestSortBy;
  PullRequestStatus: PullRequestStatus;
  PullRequestTo: ResolversTypes['Api'] | ResolversTypes['Note'] | ResolversTypes['Project'] | ResolversTypes['Routine'] | ResolversTypes['SmartContract'] | ResolversTypes['Standard'];
  PullRequestToObjectType: PullRequestToObjectType;
  PullRequestUpdateInput: PullRequestUpdateInput;
  PullRequestYou: ResolverTypeWrapper<PullRequestYou>;
  PushDevice: ResolverTypeWrapper<PushDevice>;
  PushDeviceCreateInput: PushDeviceCreateInput;
  PushDeviceKeysInput: PushDeviceKeysInput;
  PushDeviceUpdateInput: PushDeviceUpdateInput;
  Query: ResolverTypeWrapper<{}>;
  Question: ResolverTypeWrapper<Omit<Question, 'forObject'> & { forObject: ResolversTypes['QuestionFor'] }>;
  QuestionAnswer: ResolverTypeWrapper<QuestionAnswer>;
  QuestionAnswerCreateInput: QuestionAnswerCreateInput;
  QuestionAnswerEdge: ResolverTypeWrapper<QuestionAnswerEdge>;
  QuestionAnswerSearchInput: QuestionAnswerSearchInput;
  QuestionAnswerSearchResult: ResolverTypeWrapper<QuestionAnswerSearchResult>;
  QuestionAnswerSortBy: QuestionAnswerSortBy;
  QuestionAnswerTranslation: ResolverTypeWrapper<QuestionAnswerTranslation>;
  QuestionAnswerTranslationCreateInput: QuestionAnswerTranslationCreateInput;
  QuestionAnswerTranslationUpdateInput: QuestionAnswerTranslationUpdateInput;
  QuestionAnswerUpdateInput: QuestionAnswerUpdateInput;
  QuestionCreateInput: QuestionCreateInput;
  QuestionEdge: ResolverTypeWrapper<QuestionEdge>;
  QuestionFor: ResolversTypes['Api'] | ResolversTypes['Note'] | ResolversTypes['Organization'] | ResolversTypes['Project'] | ResolversTypes['Routine'] | ResolversTypes['SmartContract'] | ResolversTypes['Standard'];
  QuestionForType: QuestionForType;
  QuestionSearchInput: QuestionSearchInput;
  QuestionSearchResult: ResolverTypeWrapper<QuestionSearchResult>;
  QuestionSortBy: QuestionSortBy;
  QuestionTranslation: ResolverTypeWrapper<QuestionTranslation>;
  QuestionTranslationCreateInput: QuestionTranslationCreateInput;
  QuestionTranslationUpdateInput: QuestionTranslationUpdateInput;
  QuestionUpdateInput: QuestionUpdateInput;
  QuestionYou: ResolverTypeWrapper<QuestionYou>;
  Quiz: ResolverTypeWrapper<Quiz>;
  QuizAttempt: ResolverTypeWrapper<QuizAttempt>;
  QuizAttemptCreateInput: QuizAttemptCreateInput;
  QuizAttemptEdge: ResolverTypeWrapper<QuizAttemptEdge>;
  QuizAttemptSearchInput: QuizAttemptSearchInput;
  QuizAttemptSearchResult: ResolverTypeWrapper<QuizAttemptSearchResult>;
  QuizAttemptSortBy: QuizAttemptSortBy;
  QuizAttemptStatus: QuizAttemptStatus;
  QuizAttemptUpdateInput: QuizAttemptUpdateInput;
  QuizAttemptYou: ResolverTypeWrapper<QuizAttemptYou>;
  QuizCreateInput: QuizCreateInput;
  QuizEdge: ResolverTypeWrapper<QuizEdge>;
  QuizQuestion: ResolverTypeWrapper<QuizQuestion>;
  QuizQuestionCreateInput: QuizQuestionCreateInput;
  QuizQuestionEdge: ResolverTypeWrapper<QuizQuestionEdge>;
  QuizQuestionResponse: ResolverTypeWrapper<QuizQuestionResponse>;
  QuizQuestionResponseCreateInput: QuizQuestionResponseCreateInput;
  QuizQuestionResponseEdge: ResolverTypeWrapper<QuizQuestionResponseEdge>;
  QuizQuestionResponseSearchInput: QuizQuestionResponseSearchInput;
  QuizQuestionResponseSearchResult: ResolverTypeWrapper<QuizQuestionResponseSearchResult>;
  QuizQuestionResponseSortBy: QuizQuestionResponseSortBy;
  QuizQuestionResponseTranslation: ResolverTypeWrapper<QuizQuestionResponseTranslation>;
  QuizQuestionResponseTranslationCreateInput: QuizQuestionResponseTranslationCreateInput;
  QuizQuestionResponseTranslationUpdateInput: QuizQuestionResponseTranslationUpdateInput;
  QuizQuestionResponseUpdateInput: QuizQuestionResponseUpdateInput;
  QuizQuestionResponseYou: ResolverTypeWrapper<QuizQuestionResponseYou>;
  QuizQuestionSearchInput: QuizQuestionSearchInput;
  QuizQuestionSearchResult: ResolverTypeWrapper<QuizQuestionSearchResult>;
  QuizQuestionSortBy: QuizQuestionSortBy;
  QuizQuestionTranslation: ResolverTypeWrapper<QuizQuestionTranslation>;
  QuizQuestionTranslationCreateInput: QuizQuestionTranslationCreateInput;
  QuizQuestionTranslationUpdateInput: QuizQuestionTranslationUpdateInput;
  QuizQuestionUpdateInput: QuizQuestionUpdateInput;
  QuizQuestionYou: ResolverTypeWrapper<QuizQuestionYou>;
  QuizSearchInput: QuizSearchInput;
  QuizSearchResult: ResolverTypeWrapper<QuizSearchResult>;
  QuizSortBy: QuizSortBy;
  QuizTranslation: ResolverTypeWrapper<QuizTranslation>;
  QuizTranslationCreateInput: QuizTranslationCreateInput;
  QuizTranslationUpdateInput: QuizTranslationUpdateInput;
  QuizUpdateInput: QuizUpdateInput;
  QuizYou: ResolverTypeWrapper<QuizYou>;
  ReadAssetsInput: ReadAssetsInput;
  Reminder: ResolverTypeWrapper<Reminder>;
  ReminderCreateInput: ReminderCreateInput;
  ReminderEdge: ResolverTypeWrapper<ReminderEdge>;
  ReminderItem: ResolverTypeWrapper<ReminderItem>;
  ReminderItemCreateInput: ReminderItemCreateInput;
  ReminderItemUpdateInput: ReminderItemUpdateInput;
  ReminderList: ResolverTypeWrapper<ReminderList>;
  ReminderListCreateInput: ReminderListCreateInput;
  ReminderListEdge: ResolverTypeWrapper<ReminderListEdge>;
  ReminderListSearchInput: ReminderListSearchInput;
  ReminderListSearchResult: ResolverTypeWrapper<ReminderListSearchResult>;
  ReminderListSortBy: ReminderListSortBy;
  ReminderListUpdateInput: ReminderListUpdateInput;
  ReminderSearchInput: ReminderSearchInput;
  ReminderSearchResult: ResolverTypeWrapper<ReminderSearchResult>;
  ReminderSortBy: ReminderSortBy;
  ReminderUpdateInput: ReminderUpdateInput;
  Report: ResolverTypeWrapper<Report>;
  ReportCreateInput: ReportCreateInput;
  ReportEdge: ResolverTypeWrapper<ReportEdge>;
  ReportFor: ReportFor;
  ReportResponse: ResolverTypeWrapper<ReportResponse>;
  ReportResponseCreateInput: ReportResponseCreateInput;
  ReportResponseEdge: ResolverTypeWrapper<ReportResponseEdge>;
  ReportResponseSearchInput: ReportResponseSearchInput;
  ReportResponseSearchResult: ResolverTypeWrapper<ReportResponseSearchResult>;
  ReportResponseSortBy: ReportResponseSortBy;
  ReportResponseUpdateInput: ReportResponseUpdateInput;
  ReportResponseYou: ResolverTypeWrapper<ReportResponseYou>;
  ReportSearchInput: ReportSearchInput;
  ReportSearchResult: ResolverTypeWrapper<ReportSearchResult>;
  ReportSortBy: ReportSortBy;
  ReportSuggestedAction: ReportSuggestedAction;
  ReportUpdateInput: ReportUpdateInput;
  ReportYou: ResolverTypeWrapper<ReportYou>;
  ReputationHistory: ResolverTypeWrapper<ReputationHistory>;
  ReputationHistoryEdge: ResolverTypeWrapper<ReputationHistoryEdge>;
  ReputationHistorySearchInput: ReputationHistorySearchInput;
  ReputationHistorySearchResult: ResolverTypeWrapper<ReputationHistorySearchResult>;
  ReputationHistorySortBy: ReputationHistorySortBy;
  ResearchResult: ResolverTypeWrapper<Omit<ResearchResult, 'newlyCompleted'> & { newlyCompleted: Array<ResolversTypes['ProjectOrRoutine']> }>;
  Resource: ResolverTypeWrapper<Resource>;
  ResourceCreateInput: ResourceCreateInput;
  ResourceEdge: ResolverTypeWrapper<ResourceEdge>;
  ResourceList: ResolverTypeWrapper<ResourceList>;
  ResourceListCreateInput: ResourceListCreateInput;
  ResourceListEdge: ResolverTypeWrapper<ResourceListEdge>;
  ResourceListSearchInput: ResourceListSearchInput;
  ResourceListSearchResult: ResolverTypeWrapper<ResourceListSearchResult>;
  ResourceListSortBy: ResourceListSortBy;
  ResourceListTranslation: ResolverTypeWrapper<ResourceListTranslation>;
  ResourceListTranslationCreateInput: ResourceListTranslationCreateInput;
  ResourceListTranslationUpdateInput: ResourceListTranslationUpdateInput;
  ResourceListUpdateInput: ResourceListUpdateInput;
  ResourceSearchInput: ResourceSearchInput;
  ResourceSearchResult: ResolverTypeWrapper<ResourceSearchResult>;
  ResourceSortBy: ResourceSortBy;
  ResourceTranslation: ResolverTypeWrapper<ResourceTranslation>;
  ResourceTranslationCreateInput: ResourceTranslationCreateInput;
  ResourceTranslationUpdateInput: ResourceTranslationUpdateInput;
  ResourceUpdateInput: ResourceUpdateInput;
  ResourceUsedFor: ResourceUsedFor;
  Response: ResolverTypeWrapper<Response>;
  Role: ResolverTypeWrapper<Role>;
  RoleCreateInput: RoleCreateInput;
  RoleEdge: ResolverTypeWrapper<RoleEdge>;
  RoleSearchInput: RoleSearchInput;
  RoleSearchResult: ResolverTypeWrapper<RoleSearchResult>;
  RoleSortBy: RoleSortBy;
  RoleTranslation: ResolverTypeWrapper<RoleTranslation>;
  RoleTranslationCreateInput: RoleTranslationCreateInput;
  RoleTranslationUpdateInput: RoleTranslationUpdateInput;
  RoleUpdateInput: RoleUpdateInput;
  Routine: ResolverTypeWrapper<Omit<Routine, 'owner'> & { owner?: Maybe<ResolversTypes['Owner']> }>;
  RoutineCreateInput: RoutineCreateInput;
  RoutineEdge: ResolverTypeWrapper<RoutineEdge>;
  RoutineSearchInput: RoutineSearchInput;
  RoutineSearchResult: ResolverTypeWrapper<RoutineSearchResult>;
  RoutineSortBy: RoutineSortBy;
  RoutineUpdateInput: RoutineUpdateInput;
  RoutineVersion: ResolverTypeWrapper<RoutineVersion>;
  RoutineVersionCreateInput: RoutineVersionCreateInput;
  RoutineVersionEdge: ResolverTypeWrapper<RoutineVersionEdge>;
  RoutineVersionInput: ResolverTypeWrapper<RoutineVersionInput>;
  RoutineVersionInputCreateInput: RoutineVersionInputCreateInput;
  RoutineVersionInputTranslation: ResolverTypeWrapper<RoutineVersionInputTranslation>;
  RoutineVersionInputTranslationCreateInput: RoutineVersionInputTranslationCreateInput;
  RoutineVersionInputTranslationUpdateInput: RoutineVersionInputTranslationUpdateInput;
  RoutineVersionInputUpdateInput: RoutineVersionInputUpdateInput;
  RoutineVersionOutput: ResolverTypeWrapper<RoutineVersionOutput>;
  RoutineVersionOutputCreateInput: RoutineVersionOutputCreateInput;
  RoutineVersionOutputTranslation: ResolverTypeWrapper<RoutineVersionOutputTranslation>;
  RoutineVersionOutputTranslationCreateInput: RoutineVersionOutputTranslationCreateInput;
  RoutineVersionOutputTranslationUpdateInput: RoutineVersionOutputTranslationUpdateInput;
  RoutineVersionOutputUpdateInput: RoutineVersionOutputUpdateInput;
  RoutineVersionSearchInput: RoutineVersionSearchInput;
  RoutineVersionSearchResult: ResolverTypeWrapper<RoutineVersionSearchResult>;
  RoutineVersionSortBy: RoutineVersionSortBy;
  RoutineVersionTranslation: ResolverTypeWrapper<RoutineVersionTranslation>;
  RoutineVersionTranslationCreateInput: RoutineVersionTranslationCreateInput;
  RoutineVersionTranslationUpdateInput: RoutineVersionTranslationUpdateInput;
  RoutineVersionUpdateInput: RoutineVersionUpdateInput;
  RoutineVersionYou: ResolverTypeWrapper<RoutineVersionYou>;
  RoutineYou: ResolverTypeWrapper<RoutineYou>;
  RunProject: ResolverTypeWrapper<RunProject>;
  RunProjectCancelInput: RunProjectCancelInput;
  RunProjectCompleteInput: RunProjectCompleteInput;
  RunProjectCreateInput: RunProjectCreateInput;
  RunProjectEdge: ResolverTypeWrapper<RunProjectEdge>;
  RunProjectOrRunRoutine: ResolversTypes['RunProject'] | ResolversTypes['RunRoutine'];
  RunProjectOrRunRoutineEdge: ResolverTypeWrapper<Omit<RunProjectOrRunRoutineEdge, 'node'> & { node: ResolversTypes['RunProjectOrRunRoutine'] }>;
  RunProjectOrRunRoutinePageInfo: ResolverTypeWrapper<RunProjectOrRunRoutinePageInfo>;
  RunProjectOrRunRoutineSearchInput: RunProjectOrRunRoutineSearchInput;
  RunProjectOrRunRoutineSearchResult: ResolverTypeWrapper<RunProjectOrRunRoutineSearchResult>;
  RunProjectOrRunRoutineSortBy: RunProjectOrRunRoutineSortBy;
  RunProjectSchedule: ResolverTypeWrapper<RunProjectSchedule>;
  RunProjectScheduleCreateInput: RunProjectScheduleCreateInput;
  RunProjectScheduleEdge: ResolverTypeWrapper<RunProjectScheduleEdge>;
  RunProjectScheduleSearchInput: RunProjectScheduleSearchInput;
  RunProjectScheduleSearchResult: ResolverTypeWrapper<RunProjectScheduleSearchResult>;
  RunProjectScheduleSortBy: RunProjectScheduleSortBy;
  RunProjectScheduleTranslation: ResolverTypeWrapper<RunProjectScheduleTranslation>;
  RunProjectScheduleTranslationCreateInput: RunProjectScheduleTranslationCreateInput;
  RunProjectScheduleTranslationUpdateInput: RunProjectScheduleTranslationUpdateInput;
  RunProjectScheduleUpdateInput: RunProjectScheduleUpdateInput;
  RunProjectSearchInput: RunProjectSearchInput;
  RunProjectSearchResult: ResolverTypeWrapper<RunProjectSearchResult>;
  RunProjectSortBy: RunProjectSortBy;
  RunProjectStep: ResolverTypeWrapper<RunProjectStep>;
  RunProjectStepCreateInput: RunProjectStepCreateInput;
  RunProjectStepStatus: RunProjectStepStatus;
  RunProjectStepUpdateInput: RunProjectStepUpdateInput;
  RunProjectUpdateInput: RunProjectUpdateInput;
  RunProjectYou: ResolverTypeWrapper<RunProjectYou>;
  RunRoutine: ResolverTypeWrapper<RunRoutine>;
  RunRoutineCancelInput: RunRoutineCancelInput;
  RunRoutineCompleteInput: RunRoutineCompleteInput;
  RunRoutineCreateInput: RunRoutineCreateInput;
  RunRoutineEdge: ResolverTypeWrapper<RunRoutineEdge>;
  RunRoutineInput: ResolverTypeWrapper<RunRoutineInput>;
  RunRoutineInputCreateInput: RunRoutineInputCreateInput;
  RunRoutineInputEdge: ResolverTypeWrapper<RunRoutineInputEdge>;
  RunRoutineInputSearchInput: RunRoutineInputSearchInput;
  RunRoutineInputSearchResult: ResolverTypeWrapper<RunRoutineInputSearchResult>;
  RunRoutineInputSortBy: RunRoutineInputSortBy;
  RunRoutineInputUpdateInput: RunRoutineInputUpdateInput;
  RunRoutineSchedule: ResolverTypeWrapper<RunRoutineSchedule>;
  RunRoutineScheduleCreateInput: RunRoutineScheduleCreateInput;
  RunRoutineScheduleEdge: ResolverTypeWrapper<RunRoutineScheduleEdge>;
  RunRoutineScheduleSearchInput: RunRoutineScheduleSearchInput;
  RunRoutineScheduleSearchResult: ResolverTypeWrapper<RunRoutineScheduleSearchResult>;
  RunRoutineScheduleSortBy: RunRoutineScheduleSortBy;
  RunRoutineScheduleTranslation: ResolverTypeWrapper<RunRoutineScheduleTranslation>;
  RunRoutineScheduleTranslationCreateInput: RunRoutineScheduleTranslationCreateInput;
  RunRoutineScheduleTranslationUpdateInput: RunRoutineScheduleTranslationUpdateInput;
  RunRoutineScheduleUpdateInput: RunRoutineScheduleUpdateInput;
  RunRoutineSearchInput: RunRoutineSearchInput;
  RunRoutineSearchResult: ResolverTypeWrapper<RunRoutineSearchResult>;
  RunRoutineSortBy: RunRoutineSortBy;
  RunRoutineStep: ResolverTypeWrapper<RunRoutineStep>;
  RunRoutineStepCreateInput: RunRoutineStepCreateInput;
  RunRoutineStepSortBy: RunRoutineStepSortBy;
  RunRoutineStepStatus: RunRoutineStepStatus;
  RunRoutineStepUpdateInput: RunRoutineStepUpdateInput;
  RunRoutineUpdateInput: RunRoutineUpdateInput;
  RunRoutineYou: ResolverTypeWrapper<RunRoutineYou>;
  RunStatus: RunStatus;
  SearchException: SearchException;
  SendVerificationEmailInput: SendVerificationEmailInput;
  SendVerificationTextInput: SendVerificationTextInput;
  Session: ResolverTypeWrapper<Session>;
  SessionUser: ResolverTypeWrapper<SessionUser>;
  SmartContract: ResolverTypeWrapper<Omit<SmartContract, 'owner'> & { owner?: Maybe<ResolversTypes['Owner']> }>;
  SmartContractCreateInput: SmartContractCreateInput;
  SmartContractEdge: ResolverTypeWrapper<SmartContractEdge>;
  SmartContractSearchInput: SmartContractSearchInput;
  SmartContractSearchResult: ResolverTypeWrapper<SmartContractSearchResult>;
  SmartContractSortBy: SmartContractSortBy;
  SmartContractUpdateInput: SmartContractUpdateInput;
  SmartContractVersion: ResolverTypeWrapper<SmartContractVersion>;
  SmartContractVersionCreateInput: SmartContractVersionCreateInput;
  SmartContractVersionEdge: ResolverTypeWrapper<SmartContractVersionEdge>;
  SmartContractVersionSearchInput: SmartContractVersionSearchInput;
  SmartContractVersionSearchResult: ResolverTypeWrapper<SmartContractVersionSearchResult>;
  SmartContractVersionSortBy: SmartContractVersionSortBy;
  SmartContractVersionTranslation: ResolverTypeWrapper<SmartContractVersionTranslation>;
  SmartContractVersionTranslationCreateInput: SmartContractVersionTranslationCreateInput;
  SmartContractVersionTranslationUpdateInput: SmartContractVersionTranslationUpdateInput;
  SmartContractVersionUpdateInput: SmartContractVersionUpdateInput;
  SmartContractYou: ResolverTypeWrapper<SmartContractYou>;
  Standard: ResolverTypeWrapper<Omit<Standard, 'owner'> & { owner?: Maybe<ResolversTypes['Owner']> }>;
  StandardCreateInput: StandardCreateInput;
  StandardEdge: ResolverTypeWrapper<StandardEdge>;
  StandardSearchInput: StandardSearchInput;
  StandardSearchResult: ResolverTypeWrapper<StandardSearchResult>;
  StandardSortBy: StandardSortBy;
  StandardUpdateInput: StandardUpdateInput;
  StandardVersion: ResolverTypeWrapper<StandardVersion>;
  StandardVersionCreateInput: StandardVersionCreateInput;
  StandardVersionEdge: ResolverTypeWrapper<StandardVersionEdge>;
  StandardVersionSearchInput: StandardVersionSearchInput;
  StandardVersionSearchResult: ResolverTypeWrapper<StandardVersionSearchResult>;
  StandardVersionSortBy: StandardVersionSortBy;
  StandardVersionTranslation: ResolverTypeWrapper<StandardVersionTranslation>;
  StandardVersionTranslationCreateInput: StandardVersionTranslationCreateInput;
  StandardVersionTranslationUpdateInput: StandardVersionTranslationUpdateInput;
  StandardVersionUpdateInput: StandardVersionUpdateInput;
  StandardYou: ResolverTypeWrapper<StandardYou>;
  Star: ResolverTypeWrapper<Omit<Bookmark, 'to'> & { to: ResolversTypes['BookmarkTo'] }>;
  Bookmark: ResolverTypeWrapper<BookmarkEdge>;
  StarFor: BookmarkFor;
  StarInput: StarInput;
  StarSearchInput: StarSearchInput;
  BookmarkSearchResult: ResolverTypeWrapper<BookmarkSearchResult>;
  BookmarkSortBy: BookmarkSortBy;
  BookmarkTo: ResolversTypes['Api'] | ResolversTypes['Comment'] | ResolversTypes['Issue'] | ResolversTypes['Note'] | ResolversTypes['Organization'] | ResolversTypes['Post'] | ResolversTypes['Project'] | ResolversTypes['Question'] | ResolversTypes['QuestionAnswer'] | ResolversTypes['Quiz'] | ResolversTypes['Routine'] | ResolversTypes['SmartContract'] | ResolversTypes['Standard'] | ResolversTypes['Tag'] | ResolversTypes['User'];
  StatPeriodType: StatPeriodType;
  StatsApi: ResolverTypeWrapper<StatsApi>;
  StatsApiEdge: ResolverTypeWrapper<StatsApiEdge>;
  StatsApiSearchInput: StatsApiSearchInput;
  StatsApiSearchResult: ResolverTypeWrapper<StatsApiSearchResult>;
  StatsApiSortBy: StatsApiSortBy;
  StatsOrganization: ResolverTypeWrapper<StatsOrganization>;
  StatsOrganizationEdge: ResolverTypeWrapper<StatsOrganizationEdge>;
  StatsOrganizationSearchInput: StatsOrganizationSearchInput;
  StatsOrganizationSearchResult: ResolverTypeWrapper<StatsOrganizationSearchResult>;
  StatsOrganizationSortBy: StatsOrganizationSortBy;
  StatsProject: ResolverTypeWrapper<StatsProject>;
  StatsProjectEdge: ResolverTypeWrapper<StatsProjectEdge>;
  StatsProjectSearchInput: StatsProjectSearchInput;
  StatsProjectSearchResult: ResolverTypeWrapper<StatsProjectSearchResult>;
  StatsProjectSortBy: StatsProjectSortBy;
  StatsQuiz: ResolverTypeWrapper<StatsQuiz>;
  StatsQuizEdge: ResolverTypeWrapper<StatsQuizEdge>;
  StatsQuizSearchInput: StatsQuizSearchInput;
  StatsQuizSearchResult: ResolverTypeWrapper<StatsQuizSearchResult>;
  StatsQuizSortBy: StatsQuizSortBy;
  StatsRoutine: ResolverTypeWrapper<StatsRoutine>;
  StatsRoutineEdge: ResolverTypeWrapper<StatsRoutineEdge>;
  StatsRoutineSearchInput: StatsRoutineSearchInput;
  StatsRoutineSearchResult: ResolverTypeWrapper<StatsRoutineSearchResult>;
  StatsRoutineSortBy: StatsRoutineSortBy;
  StatsSite: ResolverTypeWrapper<StatsSite>;
  StatsSiteEdge: ResolverTypeWrapper<StatsSiteEdge>;
  StatsSiteSearchInput: StatsSiteSearchInput;
  StatsSiteSearchResult: ResolverTypeWrapper<StatsSiteSearchResult>;
  StatsSiteSortBy: StatsSiteSortBy;
  StatsSmartContract: ResolverTypeWrapper<StatsSmartContract>;
  StatsSmartContractEdge: ResolverTypeWrapper<StatsSmartContractEdge>;
  StatsSmartContractSearchInput: StatsSmartContractSearchInput;
  StatsSmartContractSearchResult: ResolverTypeWrapper<StatsSmartContractSearchResult>;
  StatsSmartContractSortBy: StatsSmartContractSortBy;
  StatsStandard: ResolverTypeWrapper<StatsStandard>;
  StatsStandardEdge: ResolverTypeWrapper<StatsStandardEdge>;
  StatsStandardSearchInput: StatsStandardSearchInput;
  StatsStandardSearchResult: ResolverTypeWrapper<StatsStandardSearchResult>;
  StatsStandardSortBy: StatsStandardSortBy;
  StatsUser: ResolverTypeWrapper<StatsUser>;
  StatsUserEdge: ResolverTypeWrapper<StatsUserEdge>;
  StatsUserSearchInput: StatsUserSearchInput;
  StatsUserSearchResult: ResolverTypeWrapper<StatsUserSearchResult>;
  StatsUserSortBy: StatsUserSortBy;
  String: ResolverTypeWrapper<Scalars['String']>;
  SubscribableObject: SubscribableObject;
  SubscribedObject: ResolversTypes['Api'] | ResolversTypes['Comment'] | ResolversTypes['Issue'] | ResolversTypes['Meeting'] | ResolversTypes['Note'] | ResolversTypes['Organization'] | ResolversTypes['Project'] | ResolversTypes['PullRequest'] | ResolversTypes['Question'] | ResolversTypes['Quiz'] | ResolversTypes['Report'] | ResolversTypes['Routine'] | ResolversTypes['SmartContract'] | ResolversTypes['Standard'];
  Success: ResolverTypeWrapper<Success>;
  SwitchCurrentAccountInput: SwitchCurrentAccountInput;
  Tag: ResolverTypeWrapper<Tag>;
  TagCreateInput: TagCreateInput;
  TagEdge: ResolverTypeWrapper<TagEdge>;
  TagSearchInput: TagSearchInput;
  TagSearchResult: ResolverTypeWrapper<TagSearchResult>;
  TagSortBy: TagSortBy;
  TagTranslation: ResolverTypeWrapper<TagTranslation>;
  TagTranslationCreateInput: TagTranslationCreateInput;
  TagTranslationUpdateInput: TagTranslationUpdateInput;
  TagUpdateInput: TagUpdateInput;
  TagYou: ResolverTypeWrapper<TagYou>;
  TimeFrame: TimeFrame;
  Transfer: ResolverTypeWrapper<Omit<Transfer, 'fromOwner' | 'object' | 'toOwner'> & { fromOwner?: Maybe<ResolversTypes['Owner']>, object: ResolversTypes['TransferObject'], toOwner?: Maybe<ResolversTypes['Owner']> }>;
  TransferDenyInput: TransferDenyInput;
  TransferEdge: ResolverTypeWrapper<TransferEdge>;
  TransferObject: ResolversTypes['Api'] | ResolversTypes['Note'] | ResolversTypes['Project'] | ResolversTypes['Routine'] | ResolversTypes['SmartContract'] | ResolversTypes['Standard'];
  TransferObjectType: TransferObjectType;
  TransferRequestReceiveInput: TransferRequestReceiveInput;
  TransferRequestSendInput: TransferRequestSendInput;
  TransferSearchInput: TransferSearchInput;
  TransferSearchResult: ResolverTypeWrapper<TransferSearchResult>;
  TransferSortBy: TransferSortBy;
  TransferStatus: TransferStatus;
  TransferUpdateInput: TransferUpdateInput;
  TransferYou: ResolverTypeWrapper<TransferYou>;
  Translate: ResolverTypeWrapper<Translate>;
  TranslateInput: TranslateInput;
  Upload: ResolverTypeWrapper<Scalars['Upload']>;
  User: ResolverTypeWrapper<User>;
  UserDeleteInput: UserDeleteInput;
  UserEdge: ResolverTypeWrapper<UserEdge>;
  UserSchedule: ResolverTypeWrapper<UserSchedule>;
  UserScheduleCreateInput: UserScheduleCreateInput;
  UserScheduleEdge: ResolverTypeWrapper<UserScheduleEdge>;
  UserScheduleFilter: ResolverTypeWrapper<UserScheduleFilter>;
  UserScheduleFilterCreateInput: UserScheduleFilterCreateInput;
  UserScheduleFilterType: UserScheduleFilterType;
  UserScheduleSearchInput: UserScheduleSearchInput;
  UserScheduleSearchResult: ResolverTypeWrapper<UserScheduleSearchResult>;
  UserScheduleSortBy: UserScheduleSortBy;
  UserScheduleUpdateInput: UserScheduleUpdateInput;
  UserSearchInput: UserSearchInput;
  UserSearchResult: ResolverTypeWrapper<UserSearchResult>;
  UserSortBy: UserSortBy;
  UserTranslation: ResolverTypeWrapper<UserTranslation>;
  UserTranslationCreateInput: UserTranslationCreateInput;
  UserTranslationUpdateInput: UserTranslationUpdateInput;
  UserYou: ResolverTypeWrapper<UserYou>;
  ValidateSessionInput: ValidateSessionInput;
  VersionYou: ResolverTypeWrapper<VersionYou>;
  View: ResolverTypeWrapper<Omit<View, 'to'> & { to: ResolversTypes['ViewTo'] }>;
  ViewEdge: ResolverTypeWrapper<ViewEdge>;
  ViewSearchInput: ViewSearchInput;
  ViewSearchResult: ResolverTypeWrapper<ViewSearchResult>;
  ViewSortBy: ViewSortBy;
  ViewTo: ResolversTypes['Api'] | ResolversTypes['Issue'] | ResolversTypes['Note'] | ResolversTypes['Organization'] | ResolversTypes['Post'] | ResolversTypes['Project'] | ResolversTypes['Question'] | ResolversTypes['Routine'] | ResolversTypes['SmartContract'] | ResolversTypes['Standard'] | ResolversTypes['User'];
  VisibilityType: VisibilityType;
  Vote: ResolverTypeWrapper<Omit<Vote, 'to'> & { to: ResolversTypes['VoteTo'] }>;
  VoteEdge: ResolverTypeWrapper<VoteEdge>;
  VoteFor: VoteFor;
  VoteInput: VoteInput;
  VoteSearchInput: VoteSearchInput;
  VoteSearchResult: ResolverTypeWrapper<VoteSearchResult>;
  VoteSortBy: VoteSortBy;
  VoteTo: ResolversTypes['Api'] | ResolversTypes['Comment'] | ResolversTypes['Issue'] | ResolversTypes['Note'] | ResolversTypes['Post'] | ResolversTypes['Project'] | ResolversTypes['Question'] | ResolversTypes['QuestionAnswer'] | ResolversTypes['Quiz'] | ResolversTypes['Routine'] | ResolversTypes['SmartContract'] | ResolversTypes['Standard'];
  Wallet: ResolverTypeWrapper<Wallet>;
  WalletComplete: ResolverTypeWrapper<WalletComplete>;
  WalletCompleteInput: WalletCompleteInput;
  WalletInitInput: WalletInitInput;
  WalletUpdateInput: WalletUpdateInput;
  WriteAssetsInput: WriteAssetsInput;
};

/** Mapping between all available schema types and the resolvers parents */
export type ResolversParentTypes = {
  AnyRun: ResolversParentTypes['RunProject'] | ResolversParentTypes['RunRoutine'];
  Api: Omit<Api, 'owner'> & { owner?: Maybe<ResolversParentTypes['Owner']> };
  ApiCreateInput: ApiCreateInput;
  ApiEdge: ApiEdge;
  ApiKey: ApiKey;
  ApiKeyCreateInput: ApiKeyCreateInput;
  ApiKeyDeleteOneInput: ApiKeyDeleteOneInput;
  ApiKeyUpdateInput: ApiKeyUpdateInput;
  ApiKeyValidateInput: ApiKeyValidateInput;
  ApiSearchInput: ApiSearchInput;
  ApiSearchResult: ApiSearchResult;
  ApiUpdateInput: ApiUpdateInput;
  ApiVersion: ApiVersion;
  ApiVersionCreateInput: ApiVersionCreateInput;
  ApiVersionEdge: ApiVersionEdge;
  ApiVersionSearchInput: ApiVersionSearchInput;
  ApiVersionSearchResult: ApiVersionSearchResult;
  ApiVersionTranslation: ApiVersionTranslation;
  ApiVersionTranslationCreateInput: ApiVersionTranslationCreateInput;
  ApiVersionTranslationUpdateInput: ApiVersionTranslationUpdateInput;
  ApiVersionUpdateInput: ApiVersionUpdateInput;
  ApiYou: ApiYou;
  Award: Award;
  Boolean: Scalars['Boolean'];
  Comment: Omit<Comment, 'commentedOn' | 'owner'> & { commentedOn: ResolversParentTypes['CommentedOn'], owner?: Maybe<ResolversParentTypes['Owner']> };
  CommentCreateInput: CommentCreateInput;
  CommentSearchInput: CommentSearchInput;
  CommentSearchResult: CommentSearchResult;
  CommentThread: CommentThread;
  CommentTranslation: CommentTranslation;
  CommentTranslationCreateInput: CommentTranslationCreateInput;
  CommentTranslationUpdateInput: CommentTranslationUpdateInput;
  CommentUpdateInput: CommentUpdateInput;
  CommentYou: CommentYou;
  CommentedOn: ResolversParentTypes['ApiVersion'] | ResolversParentTypes['Issue'] | ResolversParentTypes['NoteVersion'] | ResolversParentTypes['Post'] | ResolversParentTypes['ProjectVersion'] | ResolversParentTypes['PullRequest'] | ResolversParentTypes['Question'] | ResolversParentTypes['QuestionAnswer'] | ResolversParentTypes['RoutineVersion'] | ResolversParentTypes['SmartContractVersion'] | ResolversParentTypes['StandardVersion'];
  CopyInput: CopyInput;
  CopyResult: CopyResult;
  Count: Count;
  Date: Scalars['Date'];
  DeleteManyInput: DeleteManyInput;
  DeleteOneInput: DeleteOneInput;
  DevelopResult: Omit<DevelopResult, 'completed' | 'inProgress' | 'recent'> & { completed: Array<ResolversParentTypes['ProjectOrRoutine']>, inProgress: Array<ResolversParentTypes['ProjectOrRoutine']>, recent: Array<ResolversParentTypes['ProjectOrRoutine']> };
  Email: Email;
  EmailCreateInput: EmailCreateInput;
  EmailLogInInput: EmailLogInInput;
  EmailRequestPasswordChangeInput: EmailRequestPasswordChangeInput;
  EmailResetPasswordInput: EmailResetPasswordInput;
  EmailSignUpInput: EmailSignUpInput;
  FindByIdInput: FindByIdInput;
  FindByIdOrHandleInput: FindByIdOrHandleInput;
  FindHandlesInput: FindHandlesInput;
  FindVersionInput: FindVersionInput;
  Float: Scalars['Float'];
  Handle: Handle;
  HistoryInput: HistoryInput;
  HistoryResult: Omit<HistoryResult, 'activeRuns' | 'completedRuns'> & { activeRuns: Array<ResolversParentTypes['AnyRun']>, completedRuns: Array<ResolversParentTypes['AnyRun']> };
  ID: Scalars['ID'];
  Int: Scalars['Int'];
  Issue: Omit<Issue, 'to'> & { to: ResolversParentTypes['IssueTo'] };
  IssueCloseInput: IssueCloseInput;
  IssueCreateInput: IssueCreateInput;
  IssueEdge: IssueEdge;
  IssueSearchInput: IssueSearchInput;
  IssueSearchResult: IssueSearchResult;
  IssueTo: ResolversParentTypes['Api'] | ResolversParentTypes['Note'] | ResolversParentTypes['Organization'] | ResolversParentTypes['Project'] | ResolversParentTypes['Routine'] | ResolversParentTypes['SmartContract'] | ResolversParentTypes['Standard'];
  IssueTranslation: IssueTranslation;
  IssueTranslationCreateInput: IssueTranslationCreateInput;
  IssueTranslationUpdateInput: IssueTranslationUpdateInput;
  IssueUpdateInput: IssueUpdateInput;
  IssueYou: IssueYou;
  Label: Omit<Label, 'owner'> & { owner: ResolversParentTypes['Owner'] };
  LabelCreateInput: LabelCreateInput;
  LabelEdge: LabelEdge;
  LabelSearchInput: LabelSearchInput;
  LabelSearchResult: LabelSearchResult;
  LabelTranslation: LabelTranslation;
  LabelTranslationCreateInput: LabelTranslationCreateInput;
  LabelTranslationUpdateInput: LabelTranslationUpdateInput;
  LabelUpdateInput: LabelUpdateInput;
  LabelYou: LabelYou;
  LearnResult: LearnResult;
  LogOutInput: LogOutInput;
  Meeting: Meeting;
  MeetingCreateInput: MeetingCreateInput;
  MeetingEdge: MeetingEdge;
  MeetingInvite: MeetingInvite;
  MeetingInviteCreateInput: MeetingInviteCreateInput;
  MeetingInviteEdge: MeetingInviteEdge;
  MeetingInviteSearchInput: MeetingInviteSearchInput;
  MeetingInviteSearchResult: MeetingInviteSearchResult;
  MeetingInviteUpdateInput: MeetingInviteUpdateInput;
  MeetingInviteYou: MeetingInviteYou;
  MeetingSearchInput: MeetingSearchInput;
  MeetingSearchResult: MeetingSearchResult;
  MeetingTranslation: MeetingTranslation;
  MeetingTranslationCreateInput: MeetingTranslationCreateInput;
  MeetingTranslationUpdateInput: MeetingTranslationUpdateInput;
  MeetingUpdateInput: MeetingUpdateInput;
  MeetingYou: MeetingYou;
  Member: Member;
  MemberEdge: MemberEdge;
  MemberInvite: MemberInvite;
  MemberInviteCreateInput: MemberInviteCreateInput;
  MemberInviteEdge: MemberInviteEdge;
  MemberInviteSearchInput: MemberInviteSearchInput;
  MemberInviteSearchResult: MemberInviteSearchResult;
  MemberInviteUpdateInput: MemberInviteUpdateInput;
  MemberInviteYou: MemberInviteYou;
  MemberSearchInput: MemberSearchInput;
  MemberSearchResult: MemberSearchResult;
  MemberUpdateInput: MemberUpdateInput;
  Mutation: {};
  Node: Node;
  NodeCreateInput: NodeCreateInput;
  NodeEnd: NodeEnd;
  NodeEndCreateInput: NodeEndCreateInput;
  NodeEndUpdateInput: NodeEndUpdateInput;
  NodeLink: NodeLink;
  NodeLinkCreateInput: NodeLinkCreateInput;
  NodeLinkUpdateInput: NodeLinkUpdateInput;
  NodeLinkWhen: NodeLinkWhen;
  NodeLinkWhenCreateInput: NodeLinkWhenCreateInput;
  NodeLinkWhenTranslation: NodeLinkWhenTranslation;
  NodeLinkWhenTranslationCreateInput: NodeLinkWhenTranslationCreateInput;
  NodeLinkWhenTranslationUpdateInput: NodeLinkWhenTranslationUpdateInput;
  NodeLinkWhenUpdateInput: NodeLinkWhenUpdateInput;
  NodeLoop: NodeLoop;
  NodeLoopCreateInput: NodeLoopCreateInput;
  NodeLoopUpdateInput: NodeLoopUpdateInput;
  NodeLoopWhile: NodeLoopWhile;
  NodeLoopWhileCreateInput: NodeLoopWhileCreateInput;
  NodeLoopWhileTranslation: NodeLoopWhileTranslation;
  NodeLoopWhileTranslationCreateInput: NodeLoopWhileTranslationCreateInput;
  NodeLoopWhileTranslationUpdateInput: NodeLoopWhileTranslationUpdateInput;
  NodeLoopWhileUpdateInput: NodeLoopWhileUpdateInput;
  NodeRoutineList: NodeRoutineList;
  NodeRoutineListCreateInput: NodeRoutineListCreateInput;
  NodeRoutineListItem: NodeRoutineListItem;
  NodeRoutineListItemCreateInput: NodeRoutineListItemCreateInput;
  NodeRoutineListItemTranslation: NodeRoutineListItemTranslation;
  NodeRoutineListItemTranslationCreateInput: NodeRoutineListItemTranslationCreateInput;
  NodeRoutineListItemTranslationUpdateInput: NodeRoutineListItemTranslationUpdateInput;
  NodeRoutineListItemUpdateInput: NodeRoutineListItemUpdateInput;
  NodeRoutineListUpdateInput: NodeRoutineListUpdateInput;
  NodeTranslation: NodeTranslation;
  NodeTranslationCreateInput: NodeTranslationCreateInput;
  NodeTranslationUpdateInput: NodeTranslationUpdateInput;
  NodeUpdateInput: NodeUpdateInput;
  Note: Omit<Note, 'owner'> & { owner?: Maybe<ResolversParentTypes['Owner']> };
  NoteCreateInput: NoteCreateInput;
  NoteEdge: NoteEdge;
  NoteSearchInput: NoteSearchInput;
  NoteSearchResult: NoteSearchResult;
  NoteUpdateInput: NoteUpdateInput;
  NoteVersion: NoteVersion;
  NoteVersionCreateInput: NoteVersionCreateInput;
  NoteVersionEdge: NoteVersionEdge;
  NoteVersionSearchInput: NoteVersionSearchInput;
  NoteVersionSearchResult: NoteVersionSearchResult;
  NoteVersionTranslation: NoteVersionTranslation;
  NoteVersionTranslationCreateInput: NoteVersionTranslationCreateInput;
  NoteVersionTranslationUpdateInput: NoteVersionTranslationUpdateInput;
  NoteVersionUpdateInput: NoteVersionUpdateInput;
  NoteYou: NoteYou;
  Notification: Notification;
  NotificationEdge: NotificationEdge;
  NotificationSearchInput: NotificationSearchInput;
  NotificationSearchResult: NotificationSearchResult;
  NotificationSettings: NotificationSettings;
  NotificationSettingsCategory: NotificationSettingsCategory;
  NotificationSettingsCategoryUpdateInput: NotificationSettingsCategoryUpdateInput;
  NotificationSettingsUpdateInput: NotificationSettingsUpdateInput;
  NotificationSubscription: Omit<NotificationSubscription, 'object'> & { object: ResolversParentTypes['SubscribedObject'] };
  NotificationSubscriptionCreateInput: NotificationSubscriptionCreateInput;
  NotificationSubscriptionEdge: NotificationSubscriptionEdge;
  NotificationSubscriptionSearchInput: NotificationSubscriptionSearchInput;
  NotificationSubscriptionSearchResult: NotificationSubscriptionSearchResult;
  NotificationSubscriptionUpdateInput: NotificationSubscriptionUpdateInput;
  Organization: Organization;
  OrganizationCreateInput: OrganizationCreateInput;
  OrganizationEdge: OrganizationEdge;
  OrganizationSearchInput: OrganizationSearchInput;
  OrganizationSearchResult: OrganizationSearchResult;
  OrganizationTranslation: OrganizationTranslation;
  OrganizationTranslationCreateInput: OrganizationTranslationCreateInput;
  OrganizationTranslationUpdateInput: OrganizationTranslationUpdateInput;
  OrganizationUpdateInput: OrganizationUpdateInput;
  OrganizationYou: OrganizationYou;
  Owner: ResolversParentTypes['Organization'] | ResolversParentTypes['User'];
  PageInfo: PageInfo;
  Payment: Payment;
  Phone: Phone;
  PhoneCreateInput: PhoneCreateInput;
  PopularInput: PopularInput;
  PopularResult: PopularResult;
  Post: Omit<Post, 'owner'> & { owner: ResolversParentTypes['Owner'] };
  PostCreateInput: PostCreateInput;
  PostEdge: PostEdge;
  PostSearchInput: PostSearchInput;
  PostSearchResult: PostSearchResult;
  PostTranslation: PostTranslation;
  PostTranslationCreateInput: PostTranslationCreateInput;
  PostTranslationUpdateInput: PostTranslationUpdateInput;
  PostUpdateInput: PostUpdateInput;
  Premium: Premium;
  ProfileEmailUpdateInput: ProfileEmailUpdateInput;
  ProfileUpdateInput: ProfileUpdateInput;
  Project: Omit<Project, 'owner'> & { owner?: Maybe<ResolversParentTypes['Owner']> };
  ProjectCreateInput: ProjectCreateInput;
  ProjectEdge: ProjectEdge;
  ProjectOrOrganization: ResolversParentTypes['Organization'] | ResolversParentTypes['Project'];
  ProjectOrOrganizationEdge: Omit<ProjectOrOrganizationEdge, 'node'> & { node: ResolversParentTypes['ProjectOrOrganization'] };
  ProjectOrOrganizationPageInfo: ProjectOrOrganizationPageInfo;
  ProjectOrOrganizationSearchInput: ProjectOrOrganizationSearchInput;
  ProjectOrOrganizationSearchResult: ProjectOrOrganizationSearchResult;
  ProjectOrRoutine: ResolversParentTypes['Project'] | ResolversParentTypes['Routine'];
  ProjectOrRoutineEdge: Omit<ProjectOrRoutineEdge, 'node'> & { node: ResolversParentTypes['ProjectOrRoutine'] };
  ProjectOrRoutinePageInfo: ProjectOrRoutinePageInfo;
  ProjectOrRoutineSearchInput: ProjectOrRoutineSearchInput;
  ProjectOrRoutineSearchResult: ProjectOrRoutineSearchResult;
  ProjectSearchInput: ProjectSearchInput;
  ProjectSearchResult: ProjectSearchResult;
  ProjectUpdateInput: ProjectUpdateInput;
  ProjectVersion: ProjectVersion;
  ProjectVersionCreateInput: ProjectVersionCreateInput;
  ProjectVersionDirectory: ProjectVersionDirectory;
  ProjectVersionDirectoryCreateInput: ProjectVersionDirectoryCreateInput;
  ProjectVersionDirectoryTranslation: ProjectVersionDirectoryTranslation;
  ProjectVersionDirectoryTranslationCreateInput: ProjectVersionDirectoryTranslationCreateInput;
  ProjectVersionDirectoryTranslationUpdateInput: ProjectVersionDirectoryTranslationUpdateInput;
  ProjectVersionDirectoryUpdateInput: ProjectVersionDirectoryUpdateInput;
  ProjectVersionEdge: ProjectVersionEdge;
  ProjectVersionSearchInput: ProjectVersionSearchInput;
  ProjectVersionSearchResult: ProjectVersionSearchResult;
  ProjectVersionTranslation: ProjectVersionTranslation;
  ProjectVersionTranslationCreateInput: ProjectVersionTranslationCreateInput;
  ProjectVersionTranslationUpdateInput: ProjectVersionTranslationUpdateInput;
  ProjectVersionUpdateInput: ProjectVersionUpdateInput;
  ProjectVersionYou: ProjectVersionYou;
  ProjectYou: ProjectYou;
  PullRequest: Omit<PullRequest, 'from' | 'to'> & { from: ResolversParentTypes['PullRequestFrom'], to: ResolversParentTypes['PullRequestTo'] };
  PullRequestCreateInput: PullRequestCreateInput;
  PullRequestEdge: PullRequestEdge;
  PullRequestFrom: ResolversParentTypes['ApiVersion'] | ResolversParentTypes['NoteVersion'] | ResolversParentTypes['ProjectVersion'] | ResolversParentTypes['RoutineVersion'] | ResolversParentTypes['SmartContractVersion'] | ResolversParentTypes['StandardVersion'];
  PullRequestSearchInput: PullRequestSearchInput;
  PullRequestSearchResult: PullRequestSearchResult;
  PullRequestTo: ResolversParentTypes['Api'] | ResolversParentTypes['Note'] | ResolversParentTypes['Project'] | ResolversParentTypes['Routine'] | ResolversParentTypes['SmartContract'] | ResolversParentTypes['Standard'];
  PullRequestUpdateInput: PullRequestUpdateInput;
  PullRequestYou: PullRequestYou;
  PushDevice: PushDevice;
  PushDeviceCreateInput: PushDeviceCreateInput;
  PushDeviceKeysInput: PushDeviceKeysInput;
  PushDeviceUpdateInput: PushDeviceUpdateInput;
  Query: {};
  Question: Omit<Question, 'forObject'> & { forObject: ResolversParentTypes['QuestionFor'] };
  QuestionAnswer: QuestionAnswer;
  QuestionAnswerCreateInput: QuestionAnswerCreateInput;
  QuestionAnswerEdge: QuestionAnswerEdge;
  QuestionAnswerSearchInput: QuestionAnswerSearchInput;
  QuestionAnswerSearchResult: QuestionAnswerSearchResult;
  QuestionAnswerTranslation: QuestionAnswerTranslation;
  QuestionAnswerTranslationCreateInput: QuestionAnswerTranslationCreateInput;
  QuestionAnswerTranslationUpdateInput: QuestionAnswerTranslationUpdateInput;
  QuestionAnswerUpdateInput: QuestionAnswerUpdateInput;
  QuestionCreateInput: QuestionCreateInput;
  QuestionEdge: QuestionEdge;
  QuestionFor: ResolversParentTypes['Api'] | ResolversParentTypes['Note'] | ResolversParentTypes['Organization'] | ResolversParentTypes['Project'] | ResolversParentTypes['Routine'] | ResolversParentTypes['SmartContract'] | ResolversParentTypes['Standard'];
  QuestionSearchInput: QuestionSearchInput;
  QuestionSearchResult: QuestionSearchResult;
  QuestionTranslation: QuestionTranslation;
  QuestionTranslationCreateInput: QuestionTranslationCreateInput;
  QuestionTranslationUpdateInput: QuestionTranslationUpdateInput;
  QuestionUpdateInput: QuestionUpdateInput;
  QuestionYou: QuestionYou;
  Quiz: Quiz;
  QuizAttempt: QuizAttempt;
  QuizAttemptCreateInput: QuizAttemptCreateInput;
  QuizAttemptEdge: QuizAttemptEdge;
  QuizAttemptSearchInput: QuizAttemptSearchInput;
  QuizAttemptSearchResult: QuizAttemptSearchResult;
  QuizAttemptUpdateInput: QuizAttemptUpdateInput;
  QuizAttemptYou: QuizAttemptYou;
  QuizCreateInput: QuizCreateInput;
  QuizEdge: QuizEdge;
  QuizQuestion: QuizQuestion;
  QuizQuestionCreateInput: QuizQuestionCreateInput;
  QuizQuestionEdge: QuizQuestionEdge;
  QuizQuestionResponse: QuizQuestionResponse;
  QuizQuestionResponseCreateInput: QuizQuestionResponseCreateInput;
  QuizQuestionResponseEdge: QuizQuestionResponseEdge;
  QuizQuestionResponseSearchInput: QuizQuestionResponseSearchInput;
  QuizQuestionResponseSearchResult: QuizQuestionResponseSearchResult;
  QuizQuestionResponseTranslation: QuizQuestionResponseTranslation;
  QuizQuestionResponseTranslationCreateInput: QuizQuestionResponseTranslationCreateInput;
  QuizQuestionResponseTranslationUpdateInput: QuizQuestionResponseTranslationUpdateInput;
  QuizQuestionResponseUpdateInput: QuizQuestionResponseUpdateInput;
  QuizQuestionResponseYou: QuizQuestionResponseYou;
  QuizQuestionSearchInput: QuizQuestionSearchInput;
  QuizQuestionSearchResult: QuizQuestionSearchResult;
  QuizQuestionTranslation: QuizQuestionTranslation;
  QuizQuestionTranslationCreateInput: QuizQuestionTranslationCreateInput;
  QuizQuestionTranslationUpdateInput: QuizQuestionTranslationUpdateInput;
  QuizQuestionUpdateInput: QuizQuestionUpdateInput;
  QuizQuestionYou: QuizQuestionYou;
  QuizSearchInput: QuizSearchInput;
  QuizSearchResult: QuizSearchResult;
  QuizTranslation: QuizTranslation;
  QuizTranslationCreateInput: QuizTranslationCreateInput;
  QuizTranslationUpdateInput: QuizTranslationUpdateInput;
  QuizUpdateInput: QuizUpdateInput;
  QuizYou: QuizYou;
  ReadAssetsInput: ReadAssetsInput;
  Reminder: Reminder;
  ReminderCreateInput: ReminderCreateInput;
  ReminderEdge: ReminderEdge;
  ReminderItem: ReminderItem;
  ReminderItemCreateInput: ReminderItemCreateInput;
  ReminderItemUpdateInput: ReminderItemUpdateInput;
  ReminderList: ReminderList;
  ReminderListCreateInput: ReminderListCreateInput;
  ReminderListEdge: ReminderListEdge;
  ReminderListSearchInput: ReminderListSearchInput;
  ReminderListSearchResult: ReminderListSearchResult;
  ReminderListUpdateInput: ReminderListUpdateInput;
  ReminderSearchInput: ReminderSearchInput;
  ReminderSearchResult: ReminderSearchResult;
  ReminderUpdateInput: ReminderUpdateInput;
  Report: Report;
  ReportCreateInput: ReportCreateInput;
  ReportEdge: ReportEdge;
  ReportResponse: ReportResponse;
  ReportResponseCreateInput: ReportResponseCreateInput;
  ReportResponseEdge: ReportResponseEdge;
  ReportResponseSearchInput: ReportResponseSearchInput;
  ReportResponseSearchResult: ReportResponseSearchResult;
  ReportResponseUpdateInput: ReportResponseUpdateInput;
  ReportResponseYou: ReportResponseYou;
  ReportSearchInput: ReportSearchInput;
  ReportSearchResult: ReportSearchResult;
  ReportUpdateInput: ReportUpdateInput;
  ReportYou: ReportYou;
  ReputationHistory: ReputationHistory;
  ReputationHistoryEdge: ReputationHistoryEdge;
  ReputationHistorySearchInput: ReputationHistorySearchInput;
  ReputationHistorySearchResult: ReputationHistorySearchResult;
  ResearchResult: Omit<ResearchResult, 'newlyCompleted'> & { newlyCompleted: Array<ResolversParentTypes['ProjectOrRoutine']> };
  Resource: Resource;
  ResourceCreateInput: ResourceCreateInput;
  ResourceEdge: ResourceEdge;
  ResourceList: ResourceList;
  ResourceListCreateInput: ResourceListCreateInput;
  ResourceListEdge: ResourceListEdge;
  ResourceListSearchInput: ResourceListSearchInput;
  ResourceListSearchResult: ResourceListSearchResult;
  ResourceListTranslation: ResourceListTranslation;
  ResourceListTranslationCreateInput: ResourceListTranslationCreateInput;
  ResourceListTranslationUpdateInput: ResourceListTranslationUpdateInput;
  ResourceListUpdateInput: ResourceListUpdateInput;
  ResourceSearchInput: ResourceSearchInput;
  ResourceSearchResult: ResourceSearchResult;
  ResourceTranslation: ResourceTranslation;
  ResourceTranslationCreateInput: ResourceTranslationCreateInput;
  ResourceTranslationUpdateInput: ResourceTranslationUpdateInput;
  ResourceUpdateInput: ResourceUpdateInput;
  Response: Response;
  Role: Role;
  RoleCreateInput: RoleCreateInput;
  RoleEdge: RoleEdge;
  RoleSearchInput: RoleSearchInput;
  RoleSearchResult: RoleSearchResult;
  RoleTranslation: RoleTranslation;
  RoleTranslationCreateInput: RoleTranslationCreateInput;
  RoleTranslationUpdateInput: RoleTranslationUpdateInput;
  RoleUpdateInput: RoleUpdateInput;
  Routine: Omit<Routine, 'owner'> & { owner?: Maybe<ResolversParentTypes['Owner']> };
  RoutineCreateInput: RoutineCreateInput;
  RoutineEdge: RoutineEdge;
  RoutineSearchInput: RoutineSearchInput;
  RoutineSearchResult: RoutineSearchResult;
  RoutineUpdateInput: RoutineUpdateInput;
  RoutineVersion: RoutineVersion;
  RoutineVersionCreateInput: RoutineVersionCreateInput;
  RoutineVersionEdge: RoutineVersionEdge;
  RoutineVersionInput: RoutineVersionInput;
  RoutineVersionInputCreateInput: RoutineVersionInputCreateInput;
  RoutineVersionInputTranslation: RoutineVersionInputTranslation;
  RoutineVersionInputTranslationCreateInput: RoutineVersionInputTranslationCreateInput;
  RoutineVersionInputTranslationUpdateInput: RoutineVersionInputTranslationUpdateInput;
  RoutineVersionInputUpdateInput: RoutineVersionInputUpdateInput;
  RoutineVersionOutput: RoutineVersionOutput;
  RoutineVersionOutputCreateInput: RoutineVersionOutputCreateInput;
  RoutineVersionOutputTranslation: RoutineVersionOutputTranslation;
  RoutineVersionOutputTranslationCreateInput: RoutineVersionOutputTranslationCreateInput;
  RoutineVersionOutputTranslationUpdateInput: RoutineVersionOutputTranslationUpdateInput;
  RoutineVersionOutputUpdateInput: RoutineVersionOutputUpdateInput;
  RoutineVersionSearchInput: RoutineVersionSearchInput;
  RoutineVersionSearchResult: RoutineVersionSearchResult;
  RoutineVersionTranslation: RoutineVersionTranslation;
  RoutineVersionTranslationCreateInput: RoutineVersionTranslationCreateInput;
  RoutineVersionTranslationUpdateInput: RoutineVersionTranslationUpdateInput;
  RoutineVersionUpdateInput: RoutineVersionUpdateInput;
  RoutineVersionYou: RoutineVersionYou;
  RoutineYou: RoutineYou;
  RunProject: RunProject;
  RunProjectCancelInput: RunProjectCancelInput;
  RunProjectCompleteInput: RunProjectCompleteInput;
  RunProjectCreateInput: RunProjectCreateInput;
  RunProjectEdge: RunProjectEdge;
  RunProjectOrRunRoutine: ResolversParentTypes['RunProject'] | ResolversParentTypes['RunRoutine'];
  RunProjectOrRunRoutineEdge: Omit<RunProjectOrRunRoutineEdge, 'node'> & { node: ResolversParentTypes['RunProjectOrRunRoutine'] };
  RunProjectOrRunRoutinePageInfo: RunProjectOrRunRoutinePageInfo;
  RunProjectOrRunRoutineSearchInput: RunProjectOrRunRoutineSearchInput;
  RunProjectOrRunRoutineSearchResult: RunProjectOrRunRoutineSearchResult;
  RunProjectSchedule: RunProjectSchedule;
  RunProjectScheduleCreateInput: RunProjectScheduleCreateInput;
  RunProjectScheduleEdge: RunProjectScheduleEdge;
  RunProjectScheduleSearchInput: RunProjectScheduleSearchInput;
  RunProjectScheduleSearchResult: RunProjectScheduleSearchResult;
  RunProjectScheduleTranslation: RunProjectScheduleTranslation;
  RunProjectScheduleTranslationCreateInput: RunProjectScheduleTranslationCreateInput;
  RunProjectScheduleTranslationUpdateInput: RunProjectScheduleTranslationUpdateInput;
  RunProjectScheduleUpdateInput: RunProjectScheduleUpdateInput;
  RunProjectSearchInput: RunProjectSearchInput;
  RunProjectSearchResult: RunProjectSearchResult;
  RunProjectStep: RunProjectStep;
  RunProjectStepCreateInput: RunProjectStepCreateInput;
  RunProjectStepUpdateInput: RunProjectStepUpdateInput;
  RunProjectUpdateInput: RunProjectUpdateInput;
  RunProjectYou: RunProjectYou;
  RunRoutine: RunRoutine;
  RunRoutineCancelInput: RunRoutineCancelInput;
  RunRoutineCompleteInput: RunRoutineCompleteInput;
  RunRoutineCreateInput: RunRoutineCreateInput;
  RunRoutineEdge: RunRoutineEdge;
  RunRoutineInput: RunRoutineInput;
  RunRoutineInputCreateInput: RunRoutineInputCreateInput;
  RunRoutineInputEdge: RunRoutineInputEdge;
  RunRoutineInputSearchInput: RunRoutineInputSearchInput;
  RunRoutineInputSearchResult: RunRoutineInputSearchResult;
  RunRoutineInputUpdateInput: RunRoutineInputUpdateInput;
  RunRoutineSchedule: RunRoutineSchedule;
  RunRoutineScheduleCreateInput: RunRoutineScheduleCreateInput;
  RunRoutineScheduleEdge: RunRoutineScheduleEdge;
  RunRoutineScheduleSearchInput: RunRoutineScheduleSearchInput;
  RunRoutineScheduleSearchResult: RunRoutineScheduleSearchResult;
  RunRoutineScheduleTranslation: RunRoutineScheduleTranslation;
  RunRoutineScheduleTranslationCreateInput: RunRoutineScheduleTranslationCreateInput;
  RunRoutineScheduleTranslationUpdateInput: RunRoutineScheduleTranslationUpdateInput;
  RunRoutineScheduleUpdateInput: RunRoutineScheduleUpdateInput;
  RunRoutineSearchInput: RunRoutineSearchInput;
  RunRoutineSearchResult: RunRoutineSearchResult;
  RunRoutineStep: RunRoutineStep;
  RunRoutineStepCreateInput: RunRoutineStepCreateInput;
  RunRoutineStepUpdateInput: RunRoutineStepUpdateInput;
  RunRoutineUpdateInput: RunRoutineUpdateInput;
  RunRoutineYou: RunRoutineYou;
  SearchException: SearchException;
  SendVerificationEmailInput: SendVerificationEmailInput;
  SendVerificationTextInput: SendVerificationTextInput;
  Session: Session;
  SessionUser: SessionUser;
  SmartContract: Omit<SmartContract, 'owner'> & { owner?: Maybe<ResolversParentTypes['Owner']> };
  SmartContractCreateInput: SmartContractCreateInput;
  SmartContractEdge: SmartContractEdge;
  SmartContractSearchInput: SmartContractSearchInput;
  SmartContractSearchResult: SmartContractSearchResult;
  SmartContractUpdateInput: SmartContractUpdateInput;
  SmartContractVersion: SmartContractVersion;
  SmartContractVersionCreateInput: SmartContractVersionCreateInput;
  SmartContractVersionEdge: SmartContractVersionEdge;
  SmartContractVersionSearchInput: SmartContractVersionSearchInput;
  SmartContractVersionSearchResult: SmartContractVersionSearchResult;
  SmartContractVersionTranslation: SmartContractVersionTranslation;
  SmartContractVersionTranslationCreateInput: SmartContractVersionTranslationCreateInput;
  SmartContractVersionTranslationUpdateInput: SmartContractVersionTranslationUpdateInput;
  SmartContractVersionUpdateInput: SmartContractVersionUpdateInput;
  SmartContractYou: SmartContractYou;
  Standard: Omit<Standard, 'owner'> & { owner?: Maybe<ResolversParentTypes['Owner']> };
  StandardCreateInput: StandardCreateInput;
  StandardEdge: StandardEdge;
  StandardSearchInput: StandardSearchInput;
  StandardSearchResult: StandardSearchResult;
  StandardUpdateInput: StandardUpdateInput;
  StandardVersion: StandardVersion;
  StandardVersionCreateInput: StandardVersionCreateInput;
  StandardVersionEdge: StandardVersionEdge;
  StandardVersionSearchInput: StandardVersionSearchInput;
  StandardVersionSearchResult: StandardVersionSearchResult;
  StandardVersionTranslation: StandardVersionTranslation;
  StandardVersionTranslationCreateInput: StandardVersionTranslationCreateInput;
  StandardVersionTranslationUpdateInput: StandardVersionTranslationUpdateInput;
  StandardVersionUpdateInput: StandardVersionUpdateInput;
  StandardYou: StandardYou;
  Star: Omit<Bookmark, 'to'> & { to: ResolversParentTypes['BookmarkTo'] };
  Bookmark: BookmarkEdge;
  StarInput: StarInput;
  StarSearchInput: StarSearchInput;
  BookmarkSearchResult: BookmarkSearchResult;
  BookmarkTo: ResolversParentTypes['Api'] | ResolversParentTypes['Comment'] | ResolversParentTypes['Issue'] | ResolversParentTypes['Note'] | ResolversParentTypes['Organization'] | ResolversParentTypes['Post'] | ResolversParentTypes['Project'] | ResolversParentTypes['Question'] | ResolversParentTypes['QuestionAnswer'] | ResolversParentTypes['Quiz'] | ResolversParentTypes['Routine'] | ResolversParentTypes['SmartContract'] | ResolversParentTypes['Standard'] | ResolversParentTypes['Tag'] | ResolversParentTypes['User'];
  StatsApi: StatsApi;
  StatsApiEdge: StatsApiEdge;
  StatsApiSearchInput: StatsApiSearchInput;
  StatsApiSearchResult: StatsApiSearchResult;
  StatsOrganization: StatsOrganization;
  StatsOrganizationEdge: StatsOrganizationEdge;
  StatsOrganizationSearchInput: StatsOrganizationSearchInput;
  StatsOrganizationSearchResult: StatsOrganizationSearchResult;
  StatsProject: StatsProject;
  StatsProjectEdge: StatsProjectEdge;
  StatsProjectSearchInput: StatsProjectSearchInput;
  StatsProjectSearchResult: StatsProjectSearchResult;
  StatsQuiz: StatsQuiz;
  StatsQuizEdge: StatsQuizEdge;
  StatsQuizSearchInput: StatsQuizSearchInput;
  StatsQuizSearchResult: StatsQuizSearchResult;
  StatsRoutine: StatsRoutine;
  StatsRoutineEdge: StatsRoutineEdge;
  StatsRoutineSearchInput: StatsRoutineSearchInput;
  StatsRoutineSearchResult: StatsRoutineSearchResult;
  StatsSite: StatsSite;
  StatsSiteEdge: StatsSiteEdge;
  StatsSiteSearchInput: StatsSiteSearchInput;
  StatsSiteSearchResult: StatsSiteSearchResult;
  StatsSmartContract: StatsSmartContract;
  StatsSmartContractEdge: StatsSmartContractEdge;
  StatsSmartContractSearchInput: StatsSmartContractSearchInput;
  StatsSmartContractSearchResult: StatsSmartContractSearchResult;
  StatsStandard: StatsStandard;
  StatsStandardEdge: StatsStandardEdge;
  StatsStandardSearchInput: StatsStandardSearchInput;
  StatsStandardSearchResult: StatsStandardSearchResult;
  StatsUser: StatsUser;
  StatsUserEdge: StatsUserEdge;
  StatsUserSearchInput: StatsUserSearchInput;
  StatsUserSearchResult: StatsUserSearchResult;
  String: Scalars['String'];
  SubscribedObject: ResolversParentTypes['Api'] | ResolversParentTypes['Comment'] | ResolversParentTypes['Issue'] | ResolversParentTypes['Meeting'] | ResolversParentTypes['Note'] | ResolversParentTypes['Organization'] | ResolversParentTypes['Project'] | ResolversParentTypes['PullRequest'] | ResolversParentTypes['Question'] | ResolversParentTypes['Quiz'] | ResolversParentTypes['Report'] | ResolversParentTypes['Routine'] | ResolversParentTypes['SmartContract'] | ResolversParentTypes['Standard'];
  Success: Success;
  SwitchCurrentAccountInput: SwitchCurrentAccountInput;
  Tag: Tag;
  TagCreateInput: TagCreateInput;
  TagEdge: TagEdge;
  TagSearchInput: TagSearchInput;
  TagSearchResult: TagSearchResult;
  TagTranslation: TagTranslation;
  TagTranslationCreateInput: TagTranslationCreateInput;
  TagTranslationUpdateInput: TagTranslationUpdateInput;
  TagUpdateInput: TagUpdateInput;
  TagYou: TagYou;
  TimeFrame: TimeFrame;
  Transfer: Omit<Transfer, 'fromOwner' | 'object' | 'toOwner'> & { fromOwner?: Maybe<ResolversParentTypes['Owner']>, object: ResolversParentTypes['TransferObject'], toOwner?: Maybe<ResolversParentTypes['Owner']> };
  TransferDenyInput: TransferDenyInput;
  TransferEdge: TransferEdge;
  TransferObject: ResolversParentTypes['Api'] | ResolversParentTypes['Note'] | ResolversParentTypes['Project'] | ResolversParentTypes['Routine'] | ResolversParentTypes['SmartContract'] | ResolversParentTypes['Standard'];
  TransferRequestReceiveInput: TransferRequestReceiveInput;
  TransferRequestSendInput: TransferRequestSendInput;
  TransferSearchInput: TransferSearchInput;
  TransferSearchResult: TransferSearchResult;
  TransferUpdateInput: TransferUpdateInput;
  TransferYou: TransferYou;
  Translate: Translate;
  TranslateInput: TranslateInput;
  Upload: Scalars['Upload'];
  User: User;
  UserDeleteInput: UserDeleteInput;
  UserEdge: UserEdge;
  UserSchedule: UserSchedule;
  UserScheduleCreateInput: UserScheduleCreateInput;
  UserScheduleEdge: UserScheduleEdge;
  UserScheduleFilter: UserScheduleFilter;
  UserScheduleFilterCreateInput: UserScheduleFilterCreateInput;
  UserScheduleSearchInput: UserScheduleSearchInput;
  UserScheduleSearchResult: UserScheduleSearchResult;
  UserScheduleUpdateInput: UserScheduleUpdateInput;
  UserSearchInput: UserSearchInput;
  UserSearchResult: UserSearchResult;
  UserTranslation: UserTranslation;
  UserTranslationCreateInput: UserTranslationCreateInput;
  UserTranslationUpdateInput: UserTranslationUpdateInput;
  UserYou: UserYou;
  ValidateSessionInput: ValidateSessionInput;
  VersionYou: VersionYou;
  View: Omit<View, 'to'> & { to: ResolversParentTypes['ViewTo'] };
  ViewEdge: ViewEdge;
  ViewSearchInput: ViewSearchInput;
  ViewSearchResult: ViewSearchResult;
  ViewTo: ResolversParentTypes['Api'] | ResolversParentTypes['Issue'] | ResolversParentTypes['Note'] | ResolversParentTypes['Organization'] | ResolversParentTypes['Post'] | ResolversParentTypes['Project'] | ResolversParentTypes['Question'] | ResolversParentTypes['Routine'] | ResolversParentTypes['SmartContract'] | ResolversParentTypes['Standard'] | ResolversParentTypes['User'];
  Vote: Omit<Vote, 'to'> & { to: ResolversParentTypes['VoteTo'] };
  VoteEdge: VoteEdge;
  VoteInput: VoteInput;
  VoteSearchInput: VoteSearchInput;
  VoteSearchResult: VoteSearchResult;
  VoteTo: ResolversParentTypes['Api'] | ResolversParentTypes['Comment'] | ResolversParentTypes['Issue'] | ResolversParentTypes['Note'] | ResolversParentTypes['Post'] | ResolversParentTypes['Project'] | ResolversParentTypes['Question'] | ResolversParentTypes['QuestionAnswer'] | ResolversParentTypes['Quiz'] | ResolversParentTypes['Routine'] | ResolversParentTypes['SmartContract'] | ResolversParentTypes['Standard'];
  Wallet: Wallet;
  WalletComplete: WalletComplete;
  WalletCompleteInput: WalletCompleteInput;
  WalletInitInput: WalletInitInput;
  WalletUpdateInput: WalletUpdateInput;
  WriteAssetsInput: WriteAssetsInput;
};

export type AnyRunResolvers<ContextType = any, ParentType extends ResolversParentTypes['AnyRun'] = ResolversParentTypes['AnyRun']> = {
  __resolveType: TypeResolveFn<'RunProject' | 'RunRoutine', ParentType, ContextType>;
};

export type ApiResolvers<ContextType = any, ParentType extends ResolversParentTypes['Api'] = ResolversParentTypes['Api']> = {
  completedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  createdBy?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  hasCompleteVersion?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isDeleted?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  issues?: Resolver<Array<ResolversTypes['Issue']>, ParentType, ContextType>;
  issuesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  labels?: Resolver<Array<ResolversTypes['Label']>, ParentType, ContextType>;
  owner?: Resolver<Maybe<ResolversTypes['Owner']>, ParentType, ContextType>;
  parent?: Resolver<Maybe<ResolversTypes['ApiVersion']>, ParentType, ContextType>;
  permissions?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  pullRequests?: Resolver<Array<ResolversTypes['PullRequest']>, ParentType, ContextType>;
  pullRequestsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  questions?: Resolver<Array<ResolversTypes['Question']>, ParentType, ContextType>;
  questionsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  stats?: Resolver<Array<ResolversTypes['StatsApi']>, ParentType, ContextType>;
  tags?: Resolver<Array<ResolversTypes['Tag']>, ParentType, ContextType>;
  transfers?: Resolver<Array<ResolversTypes['Transfer']>, ParentType, ContextType>;
  transfersCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  versions?: Resolver<Array<ResolversTypes['ApiVersion']>, ParentType, ContextType>;
  versionsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  views?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['ApiYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ApiEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ApiEdge'] = ResolversParentTypes['ApiEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Api'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ApiKeyResolvers<ContextType = any, ParentType extends ResolversParentTypes['ApiKey'] = ResolversParentTypes['ApiKey']> = {
  absoluteMax?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  creditsUsed?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  creditsUsedBeforeLimit?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  resetsAt?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  stopAtLimit?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ApiSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ApiSearchResult'] = ResolversParentTypes['ApiSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ApiEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ApiVersionResolvers<ContextType = any, ParentType extends ResolversParentTypes['ApiVersion'] = ResolversParentTypes['ApiVersion']> = {
  callLink?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  comments?: Resolver<Array<ResolversTypes['Comment']>, ParentType, ContextType>;
  commentsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  directoryListings?: Resolver<Array<ResolversTypes['ProjectVersionDirectory']>, ParentType, ContextType>;
  directoryListingsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  documentationLink?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  forks?: Resolver<Array<ResolversTypes['Api']>, ParentType, ContextType>;
  forksCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isLatest?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  pullRequest?: Resolver<Maybe<ResolversTypes['PullRequest']>, ParentType, ContextType>;
  reports?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  reportsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  resourceList?: Resolver<Maybe<ResolversTypes['ResourceList']>, ParentType, ContextType>;
  root?: Resolver<ResolversTypes['Api'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['ApiVersionTranslation']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  versionIndex?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  versionLabel?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  versionNotes?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  you?: Resolver<ResolversTypes['VersionYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ApiVersionEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ApiVersionEdge'] = ResolversParentTypes['ApiVersionEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['ApiVersion'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ApiVersionSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ApiVersionSearchResult'] = ResolversParentTypes['ApiVersionSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ApiVersionEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ApiVersionTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['ApiVersionTranslation'] = ResolversParentTypes['ApiVersionTranslation']> = {
  details?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  summary?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ApiYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['ApiYou'] = ResolversParentTypes['ApiYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canTransfer?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canVote?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isUpvoted?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  isViewed?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type AwardResolvers<ContextType = any, ParentType extends ResolversParentTypes['Award'] = ResolversParentTypes['Award']> = {
  category?: Resolver<ResolversTypes['AwardCategory'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  progress?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  timeCurrentTierCompleted?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CommentResolvers<ContextType = any, ParentType extends ResolversParentTypes['Comment'] = ResolversParentTypes['Comment']> = {
  commentedOn?: Resolver<ResolversTypes['CommentedOn'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  owner?: Resolver<Maybe<ResolversTypes['Owner']>, ParentType, ContextType>;
  reports?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  reportsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  bookmarkedBy?: Resolver<Maybe<Array<ResolversTypes['User']>>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['CommentTranslation']>, ParentType, ContextType>;
  translationsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['CommentYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CommentSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['CommentSearchResult'] = ResolversParentTypes['CommentSearchResult']> = {
  endCursor?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  threads?: Resolver<Maybe<Array<ResolversTypes['CommentThread']>>, ParentType, ContextType>;
  totalThreads?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CommentThreadResolvers<ContextType = any, ParentType extends ResolversParentTypes['CommentThread'] = ResolversParentTypes['CommentThread']> = {
  childThreads?: Resolver<Array<ResolversTypes['CommentThread']>, ParentType, ContextType>;
  comment?: Resolver<ResolversTypes['Comment'], ParentType, ContextType>;
  endCursor?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  totalInThread?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CommentTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['CommentTranslation'] = ResolversParentTypes['CommentTranslation']> = {
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  text?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CommentYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['CommentYou'] = ResolversParentTypes['CommentYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReply?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReport?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canVote?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isUpvoted?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CommentedOnResolvers<ContextType = any, ParentType extends ResolversParentTypes['CommentedOn'] = ResolversParentTypes['CommentedOn']> = {
  __resolveType: TypeResolveFn<'ApiVersion' | 'Issue' | 'NoteVersion' | 'Post' | 'ProjectVersion' | 'PullRequest' | 'Question' | 'QuestionAnswer' | 'RoutineVersion' | 'SmartContractVersion' | 'StandardVersion', ParentType, ContextType>;
};

export type CopyResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['CopyResult'] = ResolversParentTypes['CopyResult']> = {
  apiVersion?: Resolver<Maybe<ResolversTypes['ApiVersion']>, ParentType, ContextType>;
  noteVersion?: Resolver<Maybe<ResolversTypes['NoteVersion']>, ParentType, ContextType>;
  organization?: Resolver<Maybe<ResolversTypes['Organization']>, ParentType, ContextType>;
  projectVersion?: Resolver<Maybe<ResolversTypes['ProjectVersion']>, ParentType, ContextType>;
  routineVersion?: Resolver<Maybe<ResolversTypes['RoutineVersion']>, ParentType, ContextType>;
  smartContractVersion?: Resolver<Maybe<ResolversTypes['SmartContractVersion']>, ParentType, ContextType>;
  standardVersion?: Resolver<Maybe<ResolversTypes['StandardVersion']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CountResolvers<ContextType = any, ParentType extends ResolversParentTypes['Count'] = ResolversParentTypes['Count']> = {
  count?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export interface DateScalarConfig extends GraphQLScalarTypeConfig<ResolversTypes['Date'], any> {
  name: 'Date';
}

export type DevelopResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['DevelopResult'] = ResolversParentTypes['DevelopResult']> = {
  completed?: Resolver<Array<ResolversTypes['ProjectOrRoutine']>, ParentType, ContextType>;
  inProgress?: Resolver<Array<ResolversTypes['ProjectOrRoutine']>, ParentType, ContextType>;
  recent?: Resolver<Array<ResolversTypes['ProjectOrRoutine']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type EmailResolvers<ContextType = any, ParentType extends ResolversParentTypes['Email'] = ResolversParentTypes['Email']> = {
  emailAddress?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  verified?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type HandleResolvers<ContextType = any, ParentType extends ResolversParentTypes['Handle'] = ResolversParentTypes['Handle']> = {
  handle?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  wallet?: Resolver<ResolversTypes['Wallet'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type HistoryResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['HistoryResult'] = ResolversParentTypes['HistoryResult']> = {
  activeRuns?: Resolver<Array<ResolversTypes['AnyRun']>, ParentType, ContextType>;
  completedRuns?: Resolver<Array<ResolversTypes['AnyRun']>, ParentType, ContextType>;
  recentlyBookmarked?: Resolver<Array<ResolversTypes['Bookmark']>, ParentType, ContextType>;
  recentlyViewed?: Resolver<Array<ResolversTypes['View']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type IssueResolvers<ContextType = any, ParentType extends ResolversParentTypes['Issue'] = ResolversParentTypes['Issue']> = {
  closedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  closedBy?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  comments?: Resolver<Array<ResolversTypes['Comment']>, ParentType, ContextType>;
  commentsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  createdBy?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  labels?: Resolver<Array<ResolversTypes['Label']>, ParentType, ContextType>;
  labelsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  referencedVersionId?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  reports?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  reportsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  bookmarkedBy?: Resolver<Maybe<Array<ResolversTypes['Bookmark']>>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  status?: Resolver<ResolversTypes['IssueStatus'], ParentType, ContextType>;
  to?: Resolver<ResolversTypes['IssueTo'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['IssueTranslation']>, ParentType, ContextType>;
  translationsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  views?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['IssueYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type IssueEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['IssueEdge'] = ResolversParentTypes['IssueEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Issue'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type IssueSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['IssueSearchResult'] = ResolversParentTypes['IssueSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['IssueEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type IssueToResolvers<ContextType = any, ParentType extends ResolversParentTypes['IssueTo'] = ResolversParentTypes['IssueTo']> = {
  __resolveType: TypeResolveFn<'Api' | 'Note' | 'Organization' | 'Project' | 'Routine' | 'SmartContract' | 'Standard', ParentType, ContextType>;
};

export type IssueTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['IssueTranslation'] = ResolversParentTypes['IssueTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type IssueYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['IssueYou'] = ResolversParentTypes['IssueYou']> = {
  canComment?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReport?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canVote?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isUpvoted?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type LabelResolvers<ContextType = any, ParentType extends ResolversParentTypes['Label'] = ResolversParentTypes['Label']> = {
  apis?: Resolver<Maybe<Array<ResolversTypes['Api']>>, ParentType, ContextType>;
  apisCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  color?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  issues?: Resolver<Maybe<Array<ResolversTypes['Issue']>>, ParentType, ContextType>;
  issuesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  label?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  meetings?: Resolver<Maybe<Array<ResolversTypes['Meeting']>>, ParentType, ContextType>;
  meetingsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  notes?: Resolver<Maybe<Array<ResolversTypes['Note']>>, ParentType, ContextType>;
  notesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  owner?: Resolver<ResolversTypes['Owner'], ParentType, ContextType>;
  projects?: Resolver<Maybe<Array<ResolversTypes['Project']>>, ParentType, ContextType>;
  projectsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  routines?: Resolver<Maybe<Array<ResolversTypes['Routine']>>, ParentType, ContextType>;
  routinesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  runProjectSchedules?: Resolver<Maybe<Array<ResolversTypes['RunProjectSchedule']>>, ParentType, ContextType>;
  runProjectSchedulesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  runRoutineSchedules?: Resolver<Maybe<Array<ResolversTypes['RunRoutineSchedule']>>, ParentType, ContextType>;
  runRoutineSchedulesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  smartContracts?: Resolver<Maybe<Array<ResolversTypes['SmartContract']>>, ParentType, ContextType>;
  smartContractsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  standards?: Resolver<Maybe<Array<ResolversTypes['Standard']>>, ParentType, ContextType>;
  standardsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['LabelTranslation']>, ParentType, ContextType>;
  translationsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  userSchedules?: Resolver<Maybe<Array<ResolversTypes['UserSchedule']>>, ParentType, ContextType>;
  userSchedulesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['LabelYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type LabelEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['LabelEdge'] = ResolversParentTypes['LabelEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Label'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type LabelSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['LabelSearchResult'] = ResolversParentTypes['LabelSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['LabelEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type LabelTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['LabelTranslation'] = ResolversParentTypes['LabelTranslation']> = {
  description?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type LabelYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['LabelYou'] = ResolversParentTypes['LabelYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type LearnResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['LearnResult'] = ResolversParentTypes['LearnResult']> = {
  courses?: Resolver<Array<ResolversTypes['Project']>, ParentType, ContextType>;
  tutorials?: Resolver<Array<ResolversTypes['Routine']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MeetingResolvers<ContextType = any, ParentType extends ResolversParentTypes['Meeting'] = ResolversParentTypes['Meeting']> = {
  attendees?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  attendeesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  eventEnd?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  eventStart?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  invites?: Resolver<Array<ResolversTypes['MeetingInvite']>, ParentType, ContextType>;
  invitesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  labels?: Resolver<Array<ResolversTypes['Label']>, ParentType, ContextType>;
  labelsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  openToAnyoneWithInvite?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  organization?: Resolver<ResolversTypes['Organization'], ParentType, ContextType>;
  recurrEnd?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  recurrStart?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  recurring?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  restrictedToRoles?: Resolver<Array<ResolversTypes['Role']>, ParentType, ContextType>;
  showOnOrganizationProfile?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  timeZone?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['MeetingTranslation']>, ParentType, ContextType>;
  translationsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['MeetingYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MeetingEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['MeetingEdge'] = ResolversParentTypes['MeetingEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Meeting'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MeetingInviteResolvers<ContextType = any, ParentType extends ResolversParentTypes['MeetingInvite'] = ResolversParentTypes['MeetingInvite']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  meeting?: Resolver<ResolversTypes['Meeting'], ParentType, ContextType>;
  message?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  status?: Resolver<ResolversTypes['MeetingInviteStatus'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  user?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['MeetingInviteYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MeetingInviteEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['MeetingInviteEdge'] = ResolversParentTypes['MeetingInviteEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['MeetingInvite'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MeetingInviteSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['MeetingInviteSearchResult'] = ResolversParentTypes['MeetingInviteSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['MeetingInviteEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MeetingInviteYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['MeetingInviteYou'] = ResolversParentTypes['MeetingInviteYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MeetingSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['MeetingSearchResult'] = ResolversParentTypes['MeetingSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['MeetingEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MeetingTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['MeetingTranslation'] = ResolversParentTypes['MeetingTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  link?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  name?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MeetingYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['MeetingYou'] = ResolversParentTypes['MeetingYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canInvite?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MemberResolvers<ContextType = any, ParentType extends ResolversParentTypes['Member'] = ResolversParentTypes['Member']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isAdmin?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  organization?: Resolver<ResolversTypes['Organization'], ParentType, ContextType>;
  permissions?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  user?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MemberEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['MemberEdge'] = ResolversParentTypes['MemberEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Member'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MemberInviteResolvers<ContextType = any, ParentType extends ResolversParentTypes['MemberInvite'] = ResolversParentTypes['MemberInvite']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  message?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  organization?: Resolver<ResolversTypes['Organization'], ParentType, ContextType>;
  status?: Resolver<ResolversTypes['MemberInviteStatus'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  user?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
  willBeAdmin?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  willHavePermissions?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  you?: Resolver<ResolversTypes['MemberInviteYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MemberInviteEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['MemberInviteEdge'] = ResolversParentTypes['MemberInviteEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['MemberInvite'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MemberInviteSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['MemberInviteSearchResult'] = ResolversParentTypes['MemberInviteSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['MemberInviteEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MemberInviteYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['MemberInviteYou'] = ResolversParentTypes['MemberInviteYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MemberSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['MemberSearchResult'] = ResolversParentTypes['MemberSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['MemberEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MutationResolvers<ContextType = any, ParentType extends ResolversParentTypes['Mutation'] = ResolversParentTypes['Mutation']> = {
  apiCreate?: Resolver<ResolversTypes['Api'], ParentType, ContextType, RequireFields<MutationApiCreateArgs, 'input'>>;
  apiKeyCreate?: Resolver<ResolversTypes['ApiKey'], ParentType, ContextType, RequireFields<MutationApiKeyCreateArgs, 'input'>>;
  apiKeyDeleteOne?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationApiKeyDeleteOneArgs, 'input'>>;
  apiKeyUpdate?: Resolver<ResolversTypes['ApiKey'], ParentType, ContextType, RequireFields<MutationApiKeyUpdateArgs, 'input'>>;
  apiKeyValidate?: Resolver<ResolversTypes['ApiKey'], ParentType, ContextType, RequireFields<MutationApiKeyValidateArgs, 'input'>>;
  apiUpdate?: Resolver<ResolversTypes['Api'], ParentType, ContextType, RequireFields<MutationApiUpdateArgs, 'input'>>;
  apiVersionCreate?: Resolver<ResolversTypes['ApiVersion'], ParentType, ContextType, RequireFields<MutationApiVersionCreateArgs, 'input'>>;
  apiVersionUpdate?: Resolver<ResolversTypes['ApiVersion'], ParentType, ContextType, RequireFields<MutationApiVersionUpdateArgs, 'input'>>;
  commentCreate?: Resolver<ResolversTypes['Comment'], ParentType, ContextType, RequireFields<MutationCommentCreateArgs, 'input'>>;
  commentUpdate?: Resolver<ResolversTypes['Comment'], ParentType, ContextType, RequireFields<MutationCommentUpdateArgs, 'input'>>;
  copy?: Resolver<ResolversTypes['CopyResult'], ParentType, ContextType, RequireFields<MutationCopyArgs, 'input'>>;
  deleteMany?: Resolver<ResolversTypes['Count'], ParentType, ContextType, RequireFields<MutationDeleteManyArgs, 'input'>>;
  deleteOne?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationDeleteOneArgs, 'input'>>;
  emailCreate?: Resolver<ResolversTypes['Email'], ParentType, ContextType, RequireFields<MutationEmailCreateArgs, 'input'>>;
  emailLogIn?: Resolver<ResolversTypes['Session'], ParentType, ContextType, RequireFields<MutationEmailLogInArgs, 'input'>>;
  emailRequestPasswordChange?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationEmailRequestPasswordChangeArgs, 'input'>>;
  emailResetPassword?: Resolver<ResolversTypes['Session'], ParentType, ContextType, RequireFields<MutationEmailResetPasswordArgs, 'input'>>;
  emailSignUp?: Resolver<ResolversTypes['Session'], ParentType, ContextType, RequireFields<MutationEmailSignUpArgs, 'input'>>;
  exportData?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  guestLogIn?: Resolver<ResolversTypes['Session'], ParentType, ContextType>;
  issueClose?: Resolver<ResolversTypes['Issue'], ParentType, ContextType, RequireFields<MutationIssueCloseArgs, 'input'>>;
  issueCreate?: Resolver<ResolversTypes['Issue'], ParentType, ContextType, RequireFields<MutationIssueCreateArgs, 'input'>>;
  issueUpdate?: Resolver<ResolversTypes['Issue'], ParentType, ContextType, RequireFields<MutationIssueUpdateArgs, 'input'>>;
  labelCreate?: Resolver<ResolversTypes['Label'], ParentType, ContextType, RequireFields<MutationLabelCreateArgs, 'input'>>;
  labelUpdate?: Resolver<ResolversTypes['Label'], ParentType, ContextType, RequireFields<MutationLabelUpdateArgs, 'input'>>;
  logOut?: Resolver<ResolversTypes['Session'], ParentType, ContextType, RequireFields<MutationLogOutArgs, 'input'>>;
  meetingCreate?: Resolver<ResolversTypes['Meeting'], ParentType, ContextType, RequireFields<MutationMeetingCreateArgs, 'input'>>;
  meetingInviteAccept?: Resolver<ResolversTypes['MeetingInvite'], ParentType, ContextType, RequireFields<MutationMeetingInviteAcceptArgs, 'input'>>;
  meetingInviteCreate?: Resolver<ResolversTypes['MeetingInvite'], ParentType, ContextType, RequireFields<MutationMeetingInviteCreateArgs, 'input'>>;
  meetingInviteDecline?: Resolver<ResolversTypes['MeetingInvite'], ParentType, ContextType, RequireFields<MutationMeetingInviteDeclineArgs, 'input'>>;
  meetingInviteUpdate?: Resolver<ResolversTypes['MeetingInvite'], ParentType, ContextType, RequireFields<MutationMeetingInviteUpdateArgs, 'input'>>;
  meetingUpdate?: Resolver<ResolversTypes['Meeting'], ParentType, ContextType, RequireFields<MutationMeetingUpdateArgs, 'input'>>;
  memberInviteAccept?: Resolver<ResolversTypes['MemberInvite'], ParentType, ContextType, RequireFields<MutationMemberInviteAcceptArgs, 'input'>>;
  memberInviteCreate?: Resolver<ResolversTypes['MemberInvite'], ParentType, ContextType, RequireFields<MutationMemberInviteCreateArgs, 'input'>>;
  memberInviteDecline?: Resolver<ResolversTypes['MemberInvite'], ParentType, ContextType, RequireFields<MutationMemberInviteDeclineArgs, 'input'>>;
  memberInviteUpdate?: Resolver<ResolversTypes['MemberInvite'], ParentType, ContextType, RequireFields<MutationMemberInviteUpdateArgs, 'input'>>;
  memberUpdate?: Resolver<ResolversTypes['Member'], ParentType, ContextType, RequireFields<MutationMemberUpdateArgs, 'input'>>;
  nodeCreate?: Resolver<ResolversTypes['Node'], ParentType, ContextType, RequireFields<MutationNodeCreateArgs, 'input'>>;
  nodeUpdate?: Resolver<ResolversTypes['Node'], ParentType, ContextType, RequireFields<MutationNodeUpdateArgs, 'input'>>;
  noteCreate?: Resolver<ResolversTypes['Note'], ParentType, ContextType, RequireFields<MutationNoteCreateArgs, 'input'>>;
  noteUpdate?: Resolver<ResolversTypes['Note'], ParentType, ContextType, RequireFields<MutationNoteUpdateArgs, 'input'>>;
  noteVersionCreate?: Resolver<ResolversTypes['NoteVersion'], ParentType, ContextType, RequireFields<MutationNoteVersionCreateArgs, 'input'>>;
  noteVersionUpdate?: Resolver<ResolversTypes['NoteVersion'], ParentType, ContextType, RequireFields<MutationNoteVersionUpdateArgs, 'input'>>;
  notificationMarkAllAsRead?: Resolver<ResolversTypes['Count'], ParentType, ContextType>;
  notificationMarkAsRead?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationNotificationMarkAsReadArgs, 'input'>>;
  notificationSettingsUpdate?: Resolver<ResolversTypes['NotificationSettings'], ParentType, ContextType, RequireFields<MutationNotificationSettingsUpdateArgs, 'input'>>;
  notificationSubscriptionCreate?: Resolver<ResolversTypes['NotificationSubscription'], ParentType, ContextType, RequireFields<MutationNotificationSubscriptionCreateArgs, 'input'>>;
  notificationSubscriptionUpdate?: Resolver<ResolversTypes['NotificationSubscription'], ParentType, ContextType, RequireFields<MutationNotificationSubscriptionUpdateArgs, 'input'>>;
  organizationCreate?: Resolver<ResolversTypes['Organization'], ParentType, ContextType, RequireFields<MutationOrganizationCreateArgs, 'input'>>;
  organizationUpdate?: Resolver<ResolversTypes['Organization'], ParentType, ContextType, RequireFields<MutationOrganizationUpdateArgs, 'input'>>;
  phoneCreate?: Resolver<ResolversTypes['Phone'], ParentType, ContextType, RequireFields<MutationPhoneCreateArgs, 'input'>>;
  postCreate?: Resolver<ResolversTypes['Post'], ParentType, ContextType, RequireFields<MutationPostCreateArgs, 'input'>>;
  postUpdate?: Resolver<ResolversTypes['Post'], ParentType, ContextType, RequireFields<MutationPostUpdateArgs, 'input'>>;
  profileEmailUpdate?: Resolver<ResolversTypes['User'], ParentType, ContextType, RequireFields<MutationProfileEmailUpdateArgs, 'input'>>;
  profileUpdate?: Resolver<ResolversTypes['User'], ParentType, ContextType, RequireFields<MutationProfileUpdateArgs, 'input'>>;
  projectCreate?: Resolver<ResolversTypes['Project'], ParentType, ContextType, RequireFields<MutationProjectCreateArgs, 'input'>>;
  projectUpdate?: Resolver<ResolversTypes['Project'], ParentType, ContextType, RequireFields<MutationProjectUpdateArgs, 'input'>>;
  projectVersionCreate?: Resolver<ResolversTypes['ProjectVersion'], ParentType, ContextType, RequireFields<MutationProjectVersionCreateArgs, 'input'>>;
  projectVersionUpdate?: Resolver<ResolversTypes['ProjectVersion'], ParentType, ContextType, RequireFields<MutationProjectVersionUpdateArgs, 'input'>>;
  pullRequestAccept?: Resolver<ResolversTypes['PullRequest'], ParentType, ContextType, RequireFields<MutationPullRequestAcceptArgs, 'input'>>;
  pullRequestCreate?: Resolver<ResolversTypes['PullRequest'], ParentType, ContextType, RequireFields<MutationPullRequestCreateArgs, 'input'>>;
  pullRequestReject?: Resolver<ResolversTypes['PullRequest'], ParentType, ContextType, RequireFields<MutationPullRequestRejectArgs, 'input'>>;
  pullRequestUpdate?: Resolver<ResolversTypes['PullRequest'], ParentType, ContextType, RequireFields<MutationPullRequestUpdateArgs, 'input'>>;
  pushDeviceCreate?: Resolver<ResolversTypes['PushDevice'], ParentType, ContextType, RequireFields<MutationPushDeviceCreateArgs, 'input'>>;
  pushDeviceUpdate?: Resolver<ResolversTypes['PushDevice'], ParentType, ContextType, RequireFields<MutationPushDeviceUpdateArgs, 'input'>>;
  questionAnswerCreate?: Resolver<ResolversTypes['QuestionAnswer'], ParentType, ContextType, RequireFields<MutationQuestionAnswerCreateArgs, 'input'>>;
  questionAnswerMarkAsAccepted?: Resolver<ResolversTypes['QuestionAnswer'], ParentType, ContextType, RequireFields<MutationQuestionAnswerMarkAsAcceptedArgs, 'input'>>;
  questionAnswerUpdate?: Resolver<ResolversTypes['QuestionAnswer'], ParentType, ContextType, RequireFields<MutationQuestionAnswerUpdateArgs, 'input'>>;
  questionCreate?: Resolver<ResolversTypes['Question'], ParentType, ContextType, RequireFields<MutationQuestionCreateArgs, 'input'>>;
  questionUpdate?: Resolver<ResolversTypes['Question'], ParentType, ContextType, RequireFields<MutationQuestionUpdateArgs, 'input'>>;
  quizAttemptCreate?: Resolver<ResolversTypes['QuizAttempt'], ParentType, ContextType, RequireFields<MutationQuizAttemptCreateArgs, 'input'>>;
  quizAttemptUpdate?: Resolver<ResolversTypes['QuizAttempt'], ParentType, ContextType, RequireFields<MutationQuizAttemptUpdateArgs, 'input'>>;
  quizCreate?: Resolver<ResolversTypes['Quiz'], ParentType, ContextType, RequireFields<MutationQuizCreateArgs, 'input'>>;
  quizUpdate?: Resolver<ResolversTypes['Quiz'], ParentType, ContextType, RequireFields<MutationQuizUpdateArgs, 'input'>>;
  reminderCreate?: Resolver<ResolversTypes['Reminder'], ParentType, ContextType, RequireFields<MutationReminderCreateArgs, 'input'>>;
  reminderListCreate?: Resolver<ResolversTypes['ReminderList'], ParentType, ContextType, RequireFields<MutationReminderListCreateArgs, 'input'>>;
  reminderListUpdate?: Resolver<ResolversTypes['ReminderList'], ParentType, ContextType, RequireFields<MutationReminderListUpdateArgs, 'input'>>;
  reminderUpdate?: Resolver<ResolversTypes['Reminder'], ParentType, ContextType, RequireFields<MutationReminderUpdateArgs, 'input'>>;
  reportCreate?: Resolver<ResolversTypes['Report'], ParentType, ContextType, RequireFields<MutationReportCreateArgs, 'input'>>;
  reportResponseCreate?: Resolver<ResolversTypes['ReportResponse'], ParentType, ContextType, RequireFields<MutationReportResponseCreateArgs, 'input'>>;
  reportResponseUpdate?: Resolver<ResolversTypes['ReportResponse'], ParentType, ContextType, RequireFields<MutationReportResponseUpdateArgs, 'input'>>;
  reportUpdate?: Resolver<ResolversTypes['Report'], ParentType, ContextType, RequireFields<MutationReportUpdateArgs, 'input'>>;
  resourceCreate?: Resolver<ResolversTypes['Resource'], ParentType, ContextType, RequireFields<MutationResourceCreateArgs, 'input'>>;
  resourceListCreate?: Resolver<ResolversTypes['ResourceList'], ParentType, ContextType, RequireFields<MutationResourceListCreateArgs, 'input'>>;
  resourceListUpdate?: Resolver<ResolversTypes['ResourceList'], ParentType, ContextType, RequireFields<MutationResourceListUpdateArgs, 'input'>>;
  resourceUpdate?: Resolver<ResolversTypes['Resource'], ParentType, ContextType, RequireFields<MutationResourceUpdateArgs, 'input'>>;
  roleCreate?: Resolver<ResolversTypes['Role'], ParentType, ContextType, RequireFields<MutationRoleCreateArgs, 'input'>>;
  roleUpdate?: Resolver<ResolversTypes['Role'], ParentType, ContextType, RequireFields<MutationRoleUpdateArgs, 'input'>>;
  routineCreate?: Resolver<ResolversTypes['Routine'], ParentType, ContextType, RequireFields<MutationRoutineCreateArgs, 'input'>>;
  routineUpdate?: Resolver<ResolversTypes['Routine'], ParentType, ContextType, RequireFields<MutationRoutineUpdateArgs, 'input'>>;
  routineVersionCreate?: Resolver<ResolversTypes['RoutineVersion'], ParentType, ContextType, RequireFields<MutationRoutineVersionCreateArgs, 'input'>>;
  routineVersionUpdate?: Resolver<ResolversTypes['RoutineVersion'], ParentType, ContextType, RequireFields<MutationRoutineVersionUpdateArgs, 'input'>>;
  runProjectCancel?: Resolver<ResolversTypes['RunProject'], ParentType, ContextType, RequireFields<MutationRunProjectCancelArgs, 'input'>>;
  runProjectComplete?: Resolver<ResolversTypes['RunProject'], ParentType, ContextType, RequireFields<MutationRunProjectCompleteArgs, 'input'>>;
  runProjectCreate?: Resolver<ResolversTypes['RunProject'], ParentType, ContextType, RequireFields<MutationRunProjectCreateArgs, 'input'>>;
  runProjectDeleteAll?: Resolver<ResolversTypes['Count'], ParentType, ContextType>;
  runProjectScheduleCreate?: Resolver<ResolversTypes['RunProjectSchedule'], ParentType, ContextType, RequireFields<MutationRunProjectScheduleCreateArgs, 'input'>>;
  runProjectScheduleUpdate?: Resolver<ResolversTypes['RunProjectSchedule'], ParentType, ContextType, RequireFields<MutationRunProjectScheduleUpdateArgs, 'input'>>;
  runProjectUpdate?: Resolver<ResolversTypes['RunProject'], ParentType, ContextType, RequireFields<MutationRunProjectUpdateArgs, 'input'>>;
  runRoutineCancel?: Resolver<ResolversTypes['RunRoutine'], ParentType, ContextType, RequireFields<MutationRunRoutineCancelArgs, 'input'>>;
  runRoutineComplete?: Resolver<ResolversTypes['RunRoutine'], ParentType, ContextType, RequireFields<MutationRunRoutineCompleteArgs, 'input'>>;
  runRoutineCreate?: Resolver<ResolversTypes['RunRoutine'], ParentType, ContextType, RequireFields<MutationRunRoutineCreateArgs, 'input'>>;
  runRoutineDeleteAll?: Resolver<ResolversTypes['Count'], ParentType, ContextType>;
  runRoutineScheduleCreate?: Resolver<ResolversTypes['RunRoutineSchedule'], ParentType, ContextType, RequireFields<MutationRunRoutineScheduleCreateArgs, 'input'>>;
  runRoutineScheduleUpdate?: Resolver<ResolversTypes['RunRoutineSchedule'], ParentType, ContextType, RequireFields<MutationRunRoutineScheduleUpdateArgs, 'input'>>;
  runRoutineUpdate?: Resolver<ResolversTypes['RunRoutine'], ParentType, ContextType, RequireFields<MutationRunRoutineUpdateArgs, 'input'>>;
  sendVerificationEmail?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationSendVerificationEmailArgs, 'input'>>;
  sendVerificationText?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationSendVerificationTextArgs, 'input'>>;
  smartContractCreate?: Resolver<ResolversTypes['SmartContract'], ParentType, ContextType, RequireFields<MutationSmartContractCreateArgs, 'input'>>;
  smartContractUpdate?: Resolver<ResolversTypes['Api'], ParentType, ContextType, RequireFields<MutationSmartContractUpdateArgs, 'input'>>;
  smartContractVersionCreate?: Resolver<ResolversTypes['SmartContractVersion'], ParentType, ContextType, RequireFields<MutationSmartContractVersionCreateArgs, 'input'>>;
  smartContractVersionUpdate?: Resolver<ResolversTypes['SmartContractVersion'], ParentType, ContextType, RequireFields<MutationSmartContractVersionUpdateArgs, 'input'>>;
  standardCreate?: Resolver<ResolversTypes['Standard'], ParentType, ContextType, RequireFields<MutationStandardCreateArgs, 'input'>>;
  standardUpdate?: Resolver<ResolversTypes['Standard'], ParentType, ContextType, RequireFields<MutationStandardUpdateArgs, 'input'>>;
  standardVersionCreate?: Resolver<ResolversTypes['StandardVersion'], ParentType, ContextType, RequireFields<MutationStandardVersionCreateArgs, 'input'>>;
  standardVersionUpdate?: Resolver<ResolversTypes['StandardVersion'], ParentType, ContextType, RequireFields<MutationStandardVersionUpdateArgs, 'input'>>;
  star?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationBookmarkArgs, 'input'>>;
  switchCurrentAccount?: Resolver<ResolversTypes['Session'], ParentType, ContextType, RequireFields<MutationSwitchCurrentAccountArgs, 'input'>>;
  tagCreate?: Resolver<ResolversTypes['Tag'], ParentType, ContextType, RequireFields<MutationTagCreateArgs, 'input'>>;
  tagUpdate?: Resolver<ResolversTypes['Tag'], ParentType, ContextType, RequireFields<MutationTagUpdateArgs, 'input'>>;
  transferAccept?: Resolver<ResolversTypes['Transfer'], ParentType, ContextType, RequireFields<MutationTransferAcceptArgs, 'input'>>;
  transferCancel?: Resolver<ResolversTypes['Transfer'], ParentType, ContextType, RequireFields<MutationTransferCancelArgs, 'input'>>;
  transferDeny?: Resolver<ResolversTypes['Transfer'], ParentType, ContextType, RequireFields<MutationTransferDenyArgs, 'input'>>;
  transferRequestReceive?: Resolver<ResolversTypes['Transfer'], ParentType, ContextType, RequireFields<MutationTransferRequestReceiveArgs, 'input'>>;
  transferRequestSend?: Resolver<ResolversTypes['Transfer'], ParentType, ContextType, RequireFields<MutationTransferRequestSendArgs, 'input'>>;
  transferUpdate?: Resolver<ResolversTypes['Transfer'], ParentType, ContextType, RequireFields<MutationTransferUpdateArgs, 'input'>>;
  userDeleteOne?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationUserDeleteOneArgs, 'input'>>;
  userScheduleCreate?: Resolver<ResolversTypes['UserSchedule'], ParentType, ContextType, RequireFields<MutationUserScheduleCreateArgs, 'input'>>;
  userScheduleUpdate?: Resolver<ResolversTypes['UserSchedule'], ParentType, ContextType, RequireFields<MutationUserScheduleUpdateArgs, 'input'>>;
  validateSession?: Resolver<ResolversTypes['Session'], ParentType, ContextType, RequireFields<MutationValidateSessionArgs, 'input'>>;
  vote?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationVoteArgs, 'input'>>;
  walletComplete?: Resolver<ResolversTypes['WalletComplete'], ParentType, ContextType, RequireFields<MutationWalletCompleteArgs, 'input'>>;
  walletInit?: Resolver<ResolversTypes['String'], ParentType, ContextType, RequireFields<MutationWalletInitArgs, 'input'>>;
  walletUpdate?: Resolver<ResolversTypes['Wallet'], ParentType, ContextType, RequireFields<MutationWalletUpdateArgs, 'input'>>;
  writeAssets?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType, RequireFields<MutationWriteAssetsArgs, 'input'>>;
};

export type NodeResolvers<ContextType = any, ParentType extends ResolversParentTypes['Node'] = ResolversParentTypes['Node']> = {
  columnIndex?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  end?: Resolver<Maybe<ResolversTypes['NodeEnd']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  loop?: Resolver<Maybe<ResolversTypes['NodeLoop']>, ParentType, ContextType>;
  nodeType?: Resolver<ResolversTypes['NodeType'], ParentType, ContextType>;
  routineList?: Resolver<Maybe<ResolversTypes['NodeRoutineList']>, ParentType, ContextType>;
  routineVersion?: Resolver<ResolversTypes['RoutineVersion'], ParentType, ContextType>;
  rowIndex?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['NodeTranslation']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NodeEndResolvers<ContextType = any, ParentType extends ResolversParentTypes['NodeEnd'] = ResolversParentTypes['NodeEnd']> = {
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Node'], ParentType, ContextType>;
  suggestedNextRoutineVersions?: Resolver<Maybe<Array<ResolversTypes['RoutineVersion']>>, ParentType, ContextType>;
  wasSuccessful?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NodeLinkResolvers<ContextType = any, ParentType extends ResolversParentTypes['NodeLink'] = ResolversParentTypes['NodeLink']> = {
  from?: Resolver<ResolversTypes['Node'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  operation?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  routineVersion?: Resolver<ResolversTypes['RoutineVersion'], ParentType, ContextType>;
  to?: Resolver<ResolversTypes['Node'], ParentType, ContextType>;
  whens?: Resolver<Array<ResolversTypes['NodeLinkWhen']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NodeLinkWhenResolvers<ContextType = any, ParentType extends ResolversParentTypes['NodeLinkWhen'] = ResolversParentTypes['NodeLinkWhen']> = {
  condition?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  link?: Resolver<ResolversTypes['NodeLink'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['NodeLinkWhenTranslation']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NodeLinkWhenTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['NodeLinkWhenTranslation'] = ResolversParentTypes['NodeLinkWhenTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NodeLoopResolvers<ContextType = any, ParentType extends ResolversParentTypes['NodeLoop'] = ResolversParentTypes['NodeLoop']> = {
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  loops?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  maxLoops?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  operation?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  whiles?: Resolver<Array<ResolversTypes['NodeLoopWhile']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NodeLoopWhileResolvers<ContextType = any, ParentType extends ResolversParentTypes['NodeLoopWhile'] = ResolversParentTypes['NodeLoopWhile']> = {
  condition?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  toId?: Resolver<Maybe<ResolversTypes['ID']>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['NodeLoopWhileTranslation']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NodeLoopWhileTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['NodeLoopWhileTranslation'] = ResolversParentTypes['NodeLoopWhileTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NodeRoutineListResolvers<ContextType = any, ParentType extends ResolversParentTypes['NodeRoutineList'] = ResolversParentTypes['NodeRoutineList']> = {
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isOptional?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isOrdered?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  items?: Resolver<Array<ResolversTypes['NodeRoutineListItem']>, ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Node'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NodeRoutineListItemResolvers<ContextType = any, ParentType extends ResolversParentTypes['NodeRoutineListItem'] = ResolversParentTypes['NodeRoutineListItem']> = {
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  index?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  isOptional?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  list?: Resolver<ResolversTypes['NodeRoutineList'], ParentType, ContextType>;
  routineVersion?: Resolver<ResolversTypes['RoutineVersion'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['NodeRoutineListItemTranslation']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NodeRoutineListItemTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['NodeRoutineListItemTranslation'] = ResolversParentTypes['NodeRoutineListItemTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NodeTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['NodeTranslation'] = ResolversParentTypes['NodeTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NoteResolvers<ContextType = any, ParentType extends ResolversParentTypes['Note'] = ResolversParentTypes['Note']> = {
  createdBy?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  issues?: Resolver<Array<ResolversTypes['Issue']>, ParentType, ContextType>;
  issuesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  labels?: Resolver<Array<ResolversTypes['Label']>, ParentType, ContextType>;
  labelsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  owner?: Resolver<Maybe<ResolversTypes['Owner']>, ParentType, ContextType>;
  parent?: Resolver<Maybe<ResolversTypes['NoteVersion']>, ParentType, ContextType>;
  permissions?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  pullRequests?: Resolver<Array<ResolversTypes['PullRequest']>, ParentType, ContextType>;
  pullRequestsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  questions?: Resolver<Array<ResolversTypes['Question']>, ParentType, ContextType>;
  questionsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  tags?: Resolver<Array<ResolversTypes['Tag']>, ParentType, ContextType>;
  transfers?: Resolver<Array<ResolversTypes['Transfer']>, ParentType, ContextType>;
  transfersCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  versions?: Resolver<Array<ResolversTypes['NoteVersion']>, ParentType, ContextType>;
  versionsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  views?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['NoteYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NoteEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['NoteEdge'] = ResolversParentTypes['NoteEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Note'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NoteSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['NoteSearchResult'] = ResolversParentTypes['NoteSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['NoteEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NoteVersionResolvers<ContextType = any, ParentType extends ResolversParentTypes['NoteVersion'] = ResolversParentTypes['NoteVersion']> = {
  comments?: Resolver<Array<ResolversTypes['Comment']>, ParentType, ContextType>;
  commentsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  directoryListings?: Resolver<Array<ResolversTypes['ProjectVersionDirectory']>, ParentType, ContextType>;
  directoryListingsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  forks?: Resolver<Array<ResolversTypes['Note']>, ParentType, ContextType>;
  forksCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isLatest?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  pullRequest?: Resolver<Maybe<ResolversTypes['PullRequest']>, ParentType, ContextType>;
  reports?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  reportsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  root?: Resolver<ResolversTypes['Note'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['NoteVersionTranslation']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  versionIndex?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  versionLabel?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  versionNotes?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  you?: Resolver<ResolversTypes['VersionYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NoteVersionEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['NoteVersionEdge'] = ResolversParentTypes['NoteVersionEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['NoteVersion'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NoteVersionSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['NoteVersionSearchResult'] = ResolversParentTypes['NoteVersionSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['NoteVersionEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NoteVersionTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['NoteVersionTranslation'] = ResolversParentTypes['NoteVersionTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  text?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NoteYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['NoteYou'] = ResolversParentTypes['NoteYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canTransfer?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canVote?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isUpvoted?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  isViewed?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NotificationResolvers<ContextType = any, ParentType extends ResolversParentTypes['Notification'] = ResolversParentTypes['Notification']> = {
  category?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  imgLink?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  isRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  link?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  title?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NotificationEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['NotificationEdge'] = ResolversParentTypes['NotificationEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Notification'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NotificationSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['NotificationSearchResult'] = ResolversParentTypes['NotificationSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['NotificationEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NotificationSettingsResolvers<ContextType = any, ParentType extends ResolversParentTypes['NotificationSettings'] = ResolversParentTypes['NotificationSettings']> = {
  categories?: Resolver<Maybe<Array<ResolversTypes['NotificationSettingsCategory']>>, ParentType, ContextType>;
  dailyLimit?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  enabled?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  includedEmails?: Resolver<Maybe<Array<ResolversTypes['ID']>>, ParentType, ContextType>;
  includedPush?: Resolver<Maybe<Array<ResolversTypes['ID']>>, ParentType, ContextType>;
  includedSms?: Resolver<Maybe<Array<ResolversTypes['ID']>>, ParentType, ContextType>;
  toEmails?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  toPush?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  toSms?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NotificationSettingsCategoryResolvers<ContextType = any, ParentType extends ResolversParentTypes['NotificationSettingsCategory'] = ResolversParentTypes['NotificationSettingsCategory']> = {
  category?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  dailyLimit?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  enabled?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  toEmails?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  toPush?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  toSms?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NotificationSubscriptionResolvers<ContextType = any, ParentType extends ResolversParentTypes['NotificationSubscription'] = ResolversParentTypes['NotificationSubscription']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  object?: Resolver<ResolversTypes['SubscribedObject'], ParentType, ContextType>;
  silent?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NotificationSubscriptionEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['NotificationSubscriptionEdge'] = ResolversParentTypes['NotificationSubscriptionEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['NotificationSubscription'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NotificationSubscriptionSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['NotificationSubscriptionSearchResult'] = ResolversParentTypes['NotificationSubscriptionSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['NotificationSubscriptionEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type OrganizationResolvers<ContextType = any, ParentType extends ResolversParentTypes['Organization'] = ResolversParentTypes['Organization']> = {
  apis?: Resolver<Array<ResolversTypes['Api']>, ParentType, ContextType>;
  apisCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  comments?: Resolver<Array<ResolversTypes['Comment']>, ParentType, ContextType>;
  commentsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  directoryListings?: Resolver<Array<ResolversTypes['ProjectVersionDirectory']>, ParentType, ContextType>;
  forks?: Resolver<Array<ResolversTypes['Organization']>, ParentType, ContextType>;
  handle?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isOpenToNewMembers?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  issues?: Resolver<Array<ResolversTypes['Issue']>, ParentType, ContextType>;
  issuesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  labels?: Resolver<Array<ResolversTypes['Label']>, ParentType, ContextType>;
  labelsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  meetings?: Resolver<Array<ResolversTypes['Meeting']>, ParentType, ContextType>;
  meetingsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  members?: Resolver<Array<ResolversTypes['Member']>, ParentType, ContextType>;
  membersCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  notes?: Resolver<Array<ResolversTypes['Note']>, ParentType, ContextType>;
  notesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  parent?: Resolver<Maybe<ResolversTypes['Organization']>, ParentType, ContextType>;
  paymentHistory?: Resolver<Array<ResolversTypes['Payment']>, ParentType, ContextType>;
  permissions?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  posts?: Resolver<Array<ResolversTypes['Post']>, ParentType, ContextType>;
  postsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  premium?: Resolver<Maybe<ResolversTypes['Premium']>, ParentType, ContextType>;
  projects?: Resolver<Array<ResolversTypes['Project']>, ParentType, ContextType>;
  projectsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  questions?: Resolver<Array<ResolversTypes['Question']>, ParentType, ContextType>;
  questionsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  reports?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  reportsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  resourceList?: Resolver<Maybe<ResolversTypes['ResourceList']>, ParentType, ContextType>;
  roles?: Resolver<Maybe<Array<ResolversTypes['Role']>>, ParentType, ContextType>;
  rolesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  routines?: Resolver<Array<ResolversTypes['Routine']>, ParentType, ContextType>;
  routinesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  smartContracts?: Resolver<Array<ResolversTypes['SmartContract']>, ParentType, ContextType>;
  smartContractsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  standards?: Resolver<Array<ResolversTypes['Standard']>, ParentType, ContextType>;
  standardsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  stats?: Resolver<Array<ResolversTypes['StatsOrganization']>, ParentType, ContextType>;
  tags?: Resolver<Array<ResolversTypes['Tag']>, ParentType, ContextType>;
  transfersIncoming?: Resolver<Array<ResolversTypes['Transfer']>, ParentType, ContextType>;
  transfersOutgoing?: Resolver<Array<ResolversTypes['Transfer']>, ParentType, ContextType>;
  translatedName?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['OrganizationTranslation']>, ParentType, ContextType>;
  translationsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  views?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  wallets?: Resolver<Array<ResolversTypes['Wallet']>, ParentType, ContextType>;
  you?: Resolver<ResolversTypes['OrganizationYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type OrganizationEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['OrganizationEdge'] = ResolversParentTypes['OrganizationEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Organization'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type OrganizationSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['OrganizationSearchResult'] = ResolversParentTypes['OrganizationSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['OrganizationEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type OrganizationTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['OrganizationTranslation'] = ResolversParentTypes['OrganizationTranslation']> = {
  bio?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type OrganizationYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['OrganizationYou'] = ResolversParentTypes['OrganizationYou']> = {
  canAddMembers?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReport?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isViewed?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  yourMembership?: Resolver<Maybe<ResolversTypes['Member']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type OwnerResolvers<ContextType = any, ParentType extends ResolversParentTypes['Owner'] = ResolversParentTypes['Owner']> = {
  __resolveType: TypeResolveFn<'Organization' | 'User', ParentType, ContextType>;
};

export type PageInfoResolvers<ContextType = any, ParentType extends ResolversParentTypes['PageInfo'] = ResolversParentTypes['PageInfo']> = {
  endCursor?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  hasNextPage?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PaymentResolvers<ContextType = any, ParentType extends ResolversParentTypes['Payment'] = ResolversParentTypes['Payment']> = {
  amount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  cardExpDate?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  cardLast4?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  cardType?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  currency?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  description?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  organization?: Resolver<ResolversTypes['Organization'], ParentType, ContextType>;
  paymentMethod?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  status?: Resolver<ResolversTypes['PaymentStatus'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  user?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PhoneResolvers<ContextType = any, ParentType extends ResolversParentTypes['Phone'] = ResolversParentTypes['Phone']> = {
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  phoneNumber?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  verified?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PopularResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['PopularResult'] = ResolversParentTypes['PopularResult']> = {
  organizations?: Resolver<Array<ResolversTypes['Organization']>, ParentType, ContextType>;
  projects?: Resolver<Array<ResolversTypes['Project']>, ParentType, ContextType>;
  routines?: Resolver<Array<ResolversTypes['Routine']>, ParentType, ContextType>;
  standards?: Resolver<Array<ResolversTypes['Standard']>, ParentType, ContextType>;
  users?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PostResolvers<ContextType = any, ParentType extends ResolversParentTypes['Post'] = ResolversParentTypes['Post']> = {
  comments?: Resolver<Array<ResolversTypes['Comment']>, ParentType, ContextType>;
  commentsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  owner?: Resolver<ResolversTypes['Owner'], ParentType, ContextType>;
  reports?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  repostedFrom?: Resolver<Maybe<ResolversTypes['Post']>, ParentType, ContextType>;
  reposts?: Resolver<Array<ResolversTypes['Post']>, ParentType, ContextType>;
  repostsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  resourceList?: Resolver<ResolversTypes['ResourceList'], ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  tags?: Resolver<Array<ResolversTypes['Tag']>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['PostTranslation']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  views?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PostEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['PostEdge'] = ResolversParentTypes['PostEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Post'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PostSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['PostSearchResult'] = ResolversParentTypes['PostSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['PostEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PostTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['PostTranslation'] = ResolversParentTypes['PostTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PremiumResolvers<ContextType = any, ParentType extends ResolversParentTypes['Premium'] = ResolversParentTypes['Premium']> = {
  customPlan?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  enabledAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  expiresAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isActive?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectResolvers<ContextType = any, ParentType extends ResolversParentTypes['Project'] = ResolversParentTypes['Project']> = {
  createdBy?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  handle?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  hasCompleteVersion?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  issues?: Resolver<Array<ResolversTypes['Issue']>, ParentType, ContextType>;
  issuesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  labels?: Resolver<Array<ResolversTypes['Label']>, ParentType, ContextType>;
  labelsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  owner?: Resolver<Maybe<ResolversTypes['Owner']>, ParentType, ContextType>;
  parent?: Resolver<Maybe<ResolversTypes['ProjectVersion']>, ParentType, ContextType>;
  permissions?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  pullRequests?: Resolver<Array<ResolversTypes['PullRequest']>, ParentType, ContextType>;
  pullRequestsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  questions?: Resolver<Array<ResolversTypes['Question']>, ParentType, ContextType>;
  questionsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  quizzes?: Resolver<Array<ResolversTypes['Quiz']>, ParentType, ContextType>;
  quizzesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  stats?: Resolver<Array<ResolversTypes['StatsProject']>, ParentType, ContextType>;
  tags?: Resolver<Array<ResolversTypes['Tag']>, ParentType, ContextType>;
  transfers?: Resolver<Array<ResolversTypes['Transfer']>, ParentType, ContextType>;
  transfersCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  translatedName?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  versions?: Resolver<Array<ResolversTypes['ProjectVersion']>, ParentType, ContextType>;
  versionsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  views?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['ProjectYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectEdge'] = ResolversParentTypes['ProjectEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Project'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectOrOrganizationResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectOrOrganization'] = ResolversParentTypes['ProjectOrOrganization']> = {
  __resolveType: TypeResolveFn<'Organization' | 'Project', ParentType, ContextType>;
};

export type ProjectOrOrganizationEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectOrOrganizationEdge'] = ResolversParentTypes['ProjectOrOrganizationEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['ProjectOrOrganization'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectOrOrganizationPageInfoResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectOrOrganizationPageInfo'] = ResolversParentTypes['ProjectOrOrganizationPageInfo']> = {
  endCursorOrganization?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  endCursorProject?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  hasNextPage?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectOrOrganizationSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectOrOrganizationSearchResult'] = ResolversParentTypes['ProjectOrOrganizationSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ProjectOrOrganizationEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['ProjectOrOrganizationPageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectOrRoutineResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectOrRoutine'] = ResolversParentTypes['ProjectOrRoutine']> = {
  __resolveType: TypeResolveFn<'Project' | 'Routine', ParentType, ContextType>;
};

export type ProjectOrRoutineEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectOrRoutineEdge'] = ResolversParentTypes['ProjectOrRoutineEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['ProjectOrRoutine'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectOrRoutinePageInfoResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectOrRoutinePageInfo'] = ResolversParentTypes['ProjectOrRoutinePageInfo']> = {
  endCursorProject?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  endCursorRoutine?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  hasNextPage?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectOrRoutineSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectOrRoutineSearchResult'] = ResolversParentTypes['ProjectOrRoutineSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ProjectOrRoutineEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['ProjectOrRoutinePageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectSearchResult'] = ResolversParentTypes['ProjectSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ProjectEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectVersionResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectVersion'] = ResolversParentTypes['ProjectVersion']> = {
  comments?: Resolver<Array<ResolversTypes['Comment']>, ParentType, ContextType>;
  commentsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  completedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  complexity?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  directories?: Resolver<Array<ResolversTypes['ProjectVersionDirectory']>, ParentType, ContextType>;
  directoriesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  directoryListings?: Resolver<Array<ResolversTypes['ProjectVersionDirectory']>, ParentType, ContextType>;
  directoryListingsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  forks?: Resolver<Array<ResolversTypes['Project']>, ParentType, ContextType>;
  forksCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isComplete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isLatest?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  pullRequest?: Resolver<Maybe<ResolversTypes['PullRequest']>, ParentType, ContextType>;
  reports?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  reportsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  root?: Resolver<ResolversTypes['Project'], ParentType, ContextType>;
  runProjectsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  simplicity?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  suggestedNextByProject?: Resolver<Array<ResolversTypes['Project']>, ParentType, ContextType>;
  timesCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  timesStarted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['ProjectVersionTranslation']>, ParentType, ContextType>;
  translationsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  versionIndex?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  versionLabel?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  versionNotes?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  you?: Resolver<ResolversTypes['ProjectVersionYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectVersionDirectoryResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectVersionDirectory'] = ResolversParentTypes['ProjectVersionDirectory']> = {
  childApiVersions?: Resolver<Array<ResolversTypes['ApiVersion']>, ParentType, ContextType>;
  childNoteVersions?: Resolver<Array<ResolversTypes['NoteVersion']>, ParentType, ContextType>;
  childOrder?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  childOrganizations?: Resolver<Array<ResolversTypes['Organization']>, ParentType, ContextType>;
  childProjectVersions?: Resolver<Array<ResolversTypes['ProjectVersion']>, ParentType, ContextType>;
  childRoutineVersions?: Resolver<Array<ResolversTypes['RoutineVersion']>, ParentType, ContextType>;
  childSmartContractVersions?: Resolver<Array<ResolversTypes['SmartContractVersion']>, ParentType, ContextType>;
  childStandardVersions?: Resolver<Array<ResolversTypes['StandardVersion']>, ParentType, ContextType>;
  children?: Resolver<Array<ResolversTypes['ProjectVersionDirectory']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isRoot?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  parentDirectory?: Resolver<Maybe<ResolversTypes['ProjectVersionDirectory']>, ParentType, ContextType>;
  projectVersion?: Resolver<Maybe<ResolversTypes['ProjectVersion']>, ParentType, ContextType>;
  runProjectSteps?: Resolver<Array<ResolversTypes['RunProjectStep']>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['ProjectVersionDirectoryTranslation']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectVersionDirectoryTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectVersionDirectoryTranslation'] = ResolversParentTypes['ProjectVersionDirectoryTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectVersionEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectVersionEdge'] = ResolversParentTypes['ProjectVersionEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['ProjectVersion'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectVersionSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectVersionSearchResult'] = ResolversParentTypes['ProjectVersionSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ProjectVersionEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectVersionTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectVersionTranslation'] = ResolversParentTypes['ProjectVersionTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectVersionYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectVersionYou'] = ResolversParentTypes['ProjectVersionYou']> = {
  canComment?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canCopy?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReport?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUse?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  runs?: Resolver<Array<ResolversTypes['RunProject']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectYou'] = ResolversParentTypes['ProjectYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canTransfer?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canVote?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isUpvoted?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  isViewed?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PullRequestResolvers<ContextType = any, ParentType extends ResolversParentTypes['PullRequest'] = ResolversParentTypes['PullRequest']> = {
  comments?: Resolver<Array<ResolversTypes['Comment']>, ParentType, ContextType>;
  commentsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  createdBy?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  from?: Resolver<ResolversTypes['PullRequestFrom'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  mergedOrRejectedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  status?: Resolver<ResolversTypes['PullRequestStatus'], ParentType, ContextType>;
  to?: Resolver<ResolversTypes['PullRequestTo'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['PullRequestYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PullRequestEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['PullRequestEdge'] = ResolversParentTypes['PullRequestEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['PullRequest'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PullRequestFromResolvers<ContextType = any, ParentType extends ResolversParentTypes['PullRequestFrom'] = ResolversParentTypes['PullRequestFrom']> = {
  __resolveType: TypeResolveFn<'ApiVersion' | 'NoteVersion' | 'ProjectVersion' | 'RoutineVersion' | 'SmartContractVersion' | 'StandardVersion', ParentType, ContextType>;
};

export type PullRequestSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['PullRequestSearchResult'] = ResolversParentTypes['PullRequestSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['PullRequestEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PullRequestToResolvers<ContextType = any, ParentType extends ResolversParentTypes['PullRequestTo'] = ResolversParentTypes['PullRequestTo']> = {
  __resolveType: TypeResolveFn<'Api' | 'Note' | 'Project' | 'Routine' | 'SmartContract' | 'Standard', ParentType, ContextType>;
};

export type PullRequestYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['PullRequestYou'] = ResolversParentTypes['PullRequestYou']> = {
  canComment?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReport?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PushDeviceResolvers<ContextType = any, ParentType extends ResolversParentTypes['PushDevice'] = ResolversParentTypes['PushDevice']> = {
  expires?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  name?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QueryResolvers<ContextType = any, ParentType extends ResolversParentTypes['Query'] = ResolversParentTypes['Query']> = {
  api?: Resolver<Maybe<ResolversTypes['Api']>, ParentType, ContextType, RequireFields<QueryApiArgs, 'input'>>;
  apiVersion?: Resolver<Maybe<ResolversTypes['ApiVersion']>, ParentType, ContextType, RequireFields<QueryApiVersionArgs, 'input'>>;
  apiVersions?: Resolver<ResolversTypes['ApiVersionSearchResult'], ParentType, ContextType, RequireFields<QueryApiVersionsArgs, 'input'>>;
  apis?: Resolver<ResolversTypes['ApiSearchResult'], ParentType, ContextType, RequireFields<QueryApisArgs, 'input'>>;
  comment?: Resolver<Maybe<ResolversTypes['Comment']>, ParentType, ContextType, RequireFields<QueryCommentArgs, 'input'>>;
  comments?: Resolver<ResolversTypes['CommentSearchResult'], ParentType, ContextType, RequireFields<QueryCommentsArgs, 'input'>>;
  develop?: Resolver<ResolversTypes['DevelopResult'], ParentType, ContextType>;
  findHandles?: Resolver<Array<ResolversTypes['String']>, ParentType, ContextType, RequireFields<QueryFindHandlesArgs, 'input'>>;
  history?: Resolver<ResolversTypes['HistoryResult'], ParentType, ContextType, RequireFields<QueryHistoryArgs, 'input'>>;
  issue?: Resolver<Maybe<ResolversTypes['Issue']>, ParentType, ContextType, RequireFields<QueryIssueArgs, 'input'>>;
  issues?: Resolver<ResolversTypes['IssueSearchResult'], ParentType, ContextType, RequireFields<QueryIssuesArgs, 'input'>>;
  label?: Resolver<Maybe<ResolversTypes['Label']>, ParentType, ContextType, RequireFields<QueryLabelArgs, 'input'>>;
  labels?: Resolver<ResolversTypes['LabelSearchResult'], ParentType, ContextType, RequireFields<QueryLabelsArgs, 'input'>>;
  learn?: Resolver<ResolversTypes['LearnResult'], ParentType, ContextType>;
  meeting?: Resolver<Maybe<ResolversTypes['Meeting']>, ParentType, ContextType, RequireFields<QueryMeetingArgs, 'input'>>;
  meetingInvite?: Resolver<Maybe<ResolversTypes['MeetingInvite']>, ParentType, ContextType, RequireFields<QueryMeetingInviteArgs, 'input'>>;
  meetingInvites?: Resolver<ResolversTypes['MeetingInviteSearchResult'], ParentType, ContextType, RequireFields<QueryMeetingInvitesArgs, 'input'>>;
  meetings?: Resolver<ResolversTypes['MeetingSearchResult'], ParentType, ContextType, RequireFields<QueryMeetingsArgs, 'input'>>;
  member?: Resolver<Maybe<ResolversTypes['Member']>, ParentType, ContextType, RequireFields<QueryMemberArgs, 'input'>>;
  memberInvite?: Resolver<Maybe<ResolversTypes['MemberInvite']>, ParentType, ContextType, RequireFields<QueryMemberInviteArgs, 'input'>>;
  memberInvites?: Resolver<ResolversTypes['MemberInviteSearchResult'], ParentType, ContextType, RequireFields<QueryMemberInvitesArgs, 'input'>>;
  members?: Resolver<ResolversTypes['MemberSearchResult'], ParentType, ContextType, RequireFields<QueryMembersArgs, 'input'>>;
  note?: Resolver<Maybe<ResolversTypes['Note']>, ParentType, ContextType, RequireFields<QueryNoteArgs, 'input'>>;
  noteVersion?: Resolver<Maybe<ResolversTypes['NoteVersion']>, ParentType, ContextType, RequireFields<QueryNoteVersionArgs, 'input'>>;
  noteVersions?: Resolver<ResolversTypes['NoteVersionSearchResult'], ParentType, ContextType, RequireFields<QueryNoteVersionsArgs, 'input'>>;
  notes?: Resolver<ResolversTypes['NoteSearchResult'], ParentType, ContextType, RequireFields<QueryNotesArgs, 'input'>>;
  notification?: Resolver<Maybe<ResolversTypes['Notification']>, ParentType, ContextType, RequireFields<QueryNotificationArgs, 'input'>>;
  notificationSubscription?: Resolver<Maybe<ResolversTypes['NotificationSubscription']>, ParentType, ContextType, RequireFields<QueryNotificationSubscriptionArgs, 'input'>>;
  notificationSubscriptions?: Resolver<ResolversTypes['NotificationSubscriptionSearchResult'], ParentType, ContextType, RequireFields<QueryNotificationSubscriptionsArgs, 'input'>>;
  notifications?: Resolver<ResolversTypes['NotificationSearchResult'], ParentType, ContextType, RequireFields<QueryNotificationsArgs, 'input'>>;
  organization?: Resolver<Maybe<ResolversTypes['Organization']>, ParentType, ContextType, RequireFields<QueryOrganizationArgs, 'input'>>;
  organizations?: Resolver<ResolversTypes['OrganizationSearchResult'], ParentType, ContextType, RequireFields<QueryOrganizationsArgs, 'input'>>;
  popular?: Resolver<ResolversTypes['PopularResult'], ParentType, ContextType, RequireFields<QueryPopularArgs, 'input'>>;
  post?: Resolver<Maybe<ResolversTypes['Post']>, ParentType, ContextType, RequireFields<QueryPostArgs, 'input'>>;
  posts?: Resolver<ResolversTypes['PostSearchResult'], ParentType, ContextType, RequireFields<QueryPostsArgs, 'input'>>;
  profile?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
  project?: Resolver<Maybe<ResolversTypes['Project']>, ParentType, ContextType, RequireFields<QueryProjectArgs, 'input'>>;
  projectOrOrganizations?: Resolver<ResolversTypes['ProjectOrOrganizationSearchResult'], ParentType, ContextType, RequireFields<QueryProjectOrOrganizationsArgs, 'input'>>;
  projectOrRoutines?: Resolver<ResolversTypes['ProjectOrRoutineSearchResult'], ParentType, ContextType, RequireFields<QueryProjectOrRoutinesArgs, 'input'>>;
  projectVersion?: Resolver<Maybe<ResolversTypes['ProjectVersion']>, ParentType, ContextType, RequireFields<QueryProjectVersionArgs, 'input'>>;
  projectVersions?: Resolver<ResolversTypes['ProjectVersionSearchResult'], ParentType, ContextType, RequireFields<QueryProjectVersionsArgs, 'input'>>;
  projects?: Resolver<ResolversTypes['ProjectSearchResult'], ParentType, ContextType, RequireFields<QueryProjectsArgs, 'input'>>;
  pullRequest?: Resolver<Maybe<ResolversTypes['PullRequest']>, ParentType, ContextType, RequireFields<QueryPullRequestArgs, 'input'>>;
  pullRequests?: Resolver<ResolversTypes['PullRequestSearchResult'], ParentType, ContextType, RequireFields<QueryPullRequestsArgs, 'input'>>;
  pushDevices?: Resolver<Array<ResolversTypes['PushDevice']>, ParentType, ContextType>;
  question?: Resolver<Maybe<ResolversTypes['Question']>, ParentType, ContextType, RequireFields<QueryQuestionArgs, 'input'>>;
  questionAnswer?: Resolver<Maybe<ResolversTypes['QuestionAnswer']>, ParentType, ContextType, RequireFields<QueryQuestionAnswerArgs, 'input'>>;
  questionAnswers?: Resolver<ResolversTypes['QuestionAnswerSearchResult'], ParentType, ContextType, RequireFields<QueryQuestionAnswersArgs, 'input'>>;
  questions?: Resolver<ResolversTypes['QuestionSearchResult'], ParentType, ContextType, RequireFields<QueryQuestionsArgs, 'input'>>;
  quiz?: Resolver<Maybe<ResolversTypes['Quiz']>, ParentType, ContextType, RequireFields<QueryQuizArgs, 'input'>>;
  quizAttempt?: Resolver<Maybe<ResolversTypes['QuizAttempt']>, ParentType, ContextType, RequireFields<QueryQuizAttemptArgs, 'input'>>;
  quizAttempts?: Resolver<ResolversTypes['QuizAttemptSearchResult'], ParentType, ContextType, RequireFields<QueryQuizAttemptsArgs, 'input'>>;
  quizQuestion?: Resolver<Maybe<ResolversTypes['QuizQuestion']>, ParentType, ContextType, RequireFields<QueryQuizQuestionArgs, 'input'>>;
  quizQuestionResponse?: Resolver<Maybe<ResolversTypes['QuizQuestionResponse']>, ParentType, ContextType, RequireFields<QueryQuizQuestionResponseArgs, 'input'>>;
  quizQuestionResponses?: Resolver<ResolversTypes['QuizQuestionResponseSearchResult'], ParentType, ContextType, RequireFields<QueryQuizQuestionResponsesArgs, 'input'>>;
  quizQuestions?: Resolver<ResolversTypes['QuizQuestionSearchResult'], ParentType, ContextType, RequireFields<QueryQuizQuestionsArgs, 'input'>>;
  quizzes?: Resolver<ResolversTypes['QuizSearchResult'], ParentType, ContextType, RequireFields<QueryQuizzesArgs, 'input'>>;
  readAssets?: Resolver<Array<Maybe<ResolversTypes['String']>>, ParentType, ContextType, RequireFields<QueryReadAssetsArgs, 'input'>>;
  reminder?: Resolver<Maybe<ResolversTypes['Reminder']>, ParentType, ContextType, RequireFields<QueryReminderArgs, 'input'>>;
  reminderList?: Resolver<Maybe<ResolversTypes['ReminderList']>, ParentType, ContextType, RequireFields<QueryReminderListArgs, 'input'>>;
  reminderLists?: Resolver<ResolversTypes['ReminderListSearchResult'], ParentType, ContextType, RequireFields<QueryReminderListsArgs, 'input'>>;
  reminders?: Resolver<ResolversTypes['ReminderSearchResult'], ParentType, ContextType, RequireFields<QueryRemindersArgs, 'input'>>;
  report?: Resolver<Maybe<ResolversTypes['Report']>, ParentType, ContextType, RequireFields<QueryReportArgs, 'input'>>;
  reportResponse?: Resolver<Maybe<ResolversTypes['ReportResponse']>, ParentType, ContextType, RequireFields<QueryReportResponseArgs, 'input'>>;
  reportResponses?: Resolver<ResolversTypes['ReportResponseSearchResult'], ParentType, ContextType, RequireFields<QueryReportResponsesArgs, 'input'>>;
  reports?: Resolver<ResolversTypes['ReportSearchResult'], ParentType, ContextType, RequireFields<QueryReportsArgs, 'input'>>;
  reputationHistories?: Resolver<ResolversTypes['ReputationHistorySearchResult'], ParentType, ContextType, RequireFields<QueryReputationHistoriesArgs, 'input'>>;
  reputationHistory?: Resolver<Maybe<ResolversTypes['ReputationHistory']>, ParentType, ContextType, RequireFields<QueryReputationHistoryArgs, 'input'>>;
  research?: Resolver<ResolversTypes['ResearchResult'], ParentType, ContextType>;
  resource?: Resolver<Maybe<ResolversTypes['Resource']>, ParentType, ContextType, RequireFields<QueryResourceArgs, 'input'>>;
  resourceList?: Resolver<Maybe<ResolversTypes['Resource']>, ParentType, ContextType, RequireFields<QueryResourceListArgs, 'input'>>;
  resourceLists?: Resolver<ResolversTypes['ResourceListSearchResult'], ParentType, ContextType, RequireFields<QueryResourceListsArgs, 'input'>>;
  resources?: Resolver<ResolversTypes['ResourceSearchResult'], ParentType, ContextType, RequireFields<QueryResourcesArgs, 'input'>>;
  role?: Resolver<Maybe<ResolversTypes['Role']>, ParentType, ContextType, RequireFields<QueryRoleArgs, 'input'>>;
  roles?: Resolver<ResolversTypes['RoleSearchResult'], ParentType, ContextType, RequireFields<QueryRolesArgs, 'input'>>;
  routine?: Resolver<Maybe<ResolversTypes['Routine']>, ParentType, ContextType, RequireFields<QueryRoutineArgs, 'input'>>;
  routineVersion?: Resolver<Maybe<ResolversTypes['RoutineVersion']>, ParentType, ContextType, RequireFields<QueryRoutineVersionArgs, 'input'>>;
  routineVersions?: Resolver<ResolversTypes['RoutineVersionSearchResult'], ParentType, ContextType, RequireFields<QueryRoutineVersionsArgs, 'input'>>;
  routines?: Resolver<ResolversTypes['RoutineSearchResult'], ParentType, ContextType, RequireFields<QueryRoutinesArgs, 'input'>>;
  runProject?: Resolver<Maybe<ResolversTypes['RunProject']>, ParentType, ContextType, RequireFields<QueryRunProjectArgs, 'input'>>;
  runProjectOrRunRoutines?: Resolver<ResolversTypes['RunProjectOrRunRoutineSearchResult'], ParentType, ContextType, RequireFields<QueryRunProjectOrRunRoutinesArgs, 'input'>>;
  runProjectSchedule?: Resolver<Maybe<ResolversTypes['RunProjectSchedule']>, ParentType, ContextType, RequireFields<QueryRunProjectScheduleArgs, 'input'>>;
  runProjectSchedules?: Resolver<ResolversTypes['RunProjectScheduleSearchResult'], ParentType, ContextType, RequireFields<QueryRunProjectSchedulesArgs, 'input'>>;
  runProjects?: Resolver<ResolversTypes['RunProjectSearchResult'], ParentType, ContextType, RequireFields<QueryRunProjectsArgs, 'input'>>;
  runRoutine?: Resolver<Maybe<ResolversTypes['RunRoutine']>, ParentType, ContextType, RequireFields<QueryRunRoutineArgs, 'input'>>;
  runRoutineInputs?: Resolver<ResolversTypes['RunRoutineInputSearchResult'], ParentType, ContextType, RequireFields<QueryRunRoutineInputsArgs, 'input'>>;
  runRoutineSchedule?: Resolver<Maybe<ResolversTypes['RunRoutineSchedule']>, ParentType, ContextType, RequireFields<QueryRunRoutineScheduleArgs, 'input'>>;
  runRoutineSchedules?: Resolver<ResolversTypes['RunRoutineScheduleSearchResult'], ParentType, ContextType, RequireFields<QueryRunRoutineSchedulesArgs, 'input'>>;
  runRoutines?: Resolver<ResolversTypes['RunRoutineSearchResult'], ParentType, ContextType, RequireFields<QueryRunRoutinesArgs, 'input'>>;
  smartContract?: Resolver<Maybe<ResolversTypes['SmartContract']>, ParentType, ContextType, RequireFields<QuerySmartContractArgs, 'input'>>;
  smartContractVersion?: Resolver<Maybe<ResolversTypes['SmartContractVersion']>, ParentType, ContextType, RequireFields<QuerySmartContractVersionArgs, 'input'>>;
  smartContractVersions?: Resolver<ResolversTypes['SmartContractVersionSearchResult'], ParentType, ContextType, RequireFields<QuerySmartContractVersionsArgs, 'input'>>;
  smartContracts?: Resolver<ResolversTypes['SmartContractSearchResult'], ParentType, ContextType, RequireFields<QuerySmartContractsArgs, 'input'>>;
  standard?: Resolver<Maybe<ResolversTypes['Standard']>, ParentType, ContextType, RequireFields<QueryStandardArgs, 'input'>>;
  standardVersion?: Resolver<Maybe<ResolversTypes['StandardVersion']>, ParentType, ContextType, RequireFields<QueryStandardVersionArgs, 'input'>>;
  standardVersions?: Resolver<ResolversTypes['StandardVersionSearchResult'], ParentType, ContextType, RequireFields<QueryStandardVersionsArgs, 'input'>>;
  standards?: Resolver<ResolversTypes['StandardSearchResult'], ParentType, ContextType, RequireFields<QueryStandardsArgs, 'input'>>;
  bookmarks?: Resolver<ResolversTypes['BookmarkSearchResult'], ParentType, ContextType, RequireFields<QueryStarsArgs, 'input'>>;
  statsApi?: Resolver<ResolversTypes['StatsApiSearchResult'], ParentType, ContextType, RequireFields<QueryStatsApiArgs, 'input'>>;
  statsOrganization?: Resolver<ResolversTypes['StatsOrganizationSearchResult'], ParentType, ContextType, RequireFields<QueryStatsOrganizationArgs, 'input'>>;
  statsProject?: Resolver<ResolversTypes['StatsProjectSearchResult'], ParentType, ContextType, RequireFields<QueryStatsProjectArgs, 'input'>>;
  statsQuiz?: Resolver<ResolversTypes['StatsQuizSearchResult'], ParentType, ContextType, RequireFields<QueryStatsQuizArgs, 'input'>>;
  statsRoutine?: Resolver<ResolversTypes['StatsRoutineSearchResult'], ParentType, ContextType, RequireFields<QueryStatsRoutineArgs, 'input'>>;
  statsSite?: Resolver<ResolversTypes['StatsSiteSearchResult'], ParentType, ContextType, RequireFields<QueryStatsSiteArgs, 'input'>>;
  statsSmartContract?: Resolver<ResolversTypes['StatsSmartContractSearchResult'], ParentType, ContextType, RequireFields<QueryStatsSmartContractArgs, 'input'>>;
  statsStandard?: Resolver<ResolversTypes['StatsStandardSearchResult'], ParentType, ContextType, RequireFields<QueryStatsStandardArgs, 'input'>>;
  statsUser?: Resolver<ResolversTypes['StatsUserSearchResult'], ParentType, ContextType, RequireFields<QueryStatsUserArgs, 'input'>>;
  tag?: Resolver<Maybe<ResolversTypes['Tag']>, ParentType, ContextType, RequireFields<QueryTagArgs, 'input'>>;
  tags?: Resolver<ResolversTypes['TagSearchResult'], ParentType, ContextType, RequireFields<QueryTagsArgs, 'input'>>;
  transfer?: Resolver<Maybe<ResolversTypes['Transfer']>, ParentType, ContextType, RequireFields<QueryTransferArgs, 'input'>>;
  transfers?: Resolver<ResolversTypes['TransferSearchResult'], ParentType, ContextType, RequireFields<QueryTransfersArgs, 'input'>>;
  translate?: Resolver<ResolversTypes['Translate'], ParentType, ContextType, RequireFields<QueryTranslateArgs, 'input'>>;
  user?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType, RequireFields<QueryUserArgs, 'input'>>;
  userSchedule?: Resolver<Maybe<ResolversTypes['UserSchedule']>, ParentType, ContextType, RequireFields<QueryUserScheduleArgs, 'input'>>;
  userSchedules?: Resolver<ResolversTypes['UserScheduleSearchResult'], ParentType, ContextType, RequireFields<QueryUserSchedulesArgs, 'input'>>;
  users?: Resolver<ResolversTypes['UserSearchResult'], ParentType, ContextType, RequireFields<QueryUsersArgs, 'input'>>;
  views?: Resolver<ResolversTypes['ViewSearchResult'], ParentType, ContextType, RequireFields<QueryViewsArgs, 'input'>>;
  votes?: Resolver<ResolversTypes['VoteSearchResult'], ParentType, ContextType, RequireFields<QueryVotesArgs, 'input'>>;
};

export type QuestionResolvers<ContextType = any, ParentType extends ResolversParentTypes['Question'] = ResolversParentTypes['Question']> = {
  answers?: Resolver<Array<ResolversTypes['QuestionAnswer']>, ParentType, ContextType>;
  answersCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  comments?: Resolver<Array<ResolversTypes['Comment']>, ParentType, ContextType>;
  commentsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  createdBy?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  forObject?: Resolver<ResolversTypes['QuestionFor'], ParentType, ContextType>;
  hasAcceptedAnswer?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  reports?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  reportsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  tags?: Resolver<Array<ResolversTypes['Tag']>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['QuestionTranslation']>, ParentType, ContextType>;
  translationsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['QuestionYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuestionAnswerResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuestionAnswer'] = ResolversParentTypes['QuestionAnswer']> = {
  comments?: Resolver<Array<ResolversTypes['Comment']>, ParentType, ContextType>;
  commentsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  createdBy?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isAccepted?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  question?: Resolver<ResolversTypes['Question'], ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['QuestionAnswerTranslation']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuestionAnswerEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuestionAnswerEdge'] = ResolversParentTypes['QuestionAnswerEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['QuestionAnswer'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuestionAnswerSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuestionAnswerSearchResult'] = ResolversParentTypes['QuestionAnswerSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['QuestionAnswerEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuestionAnswerTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuestionAnswerTranslation'] = ResolversParentTypes['QuestionAnswerTranslation']> = {
  description?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuestionEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuestionEdge'] = ResolversParentTypes['QuestionEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Question'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuestionForResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuestionFor'] = ResolversParentTypes['QuestionFor']> = {
  __resolveType: TypeResolveFn<'Api' | 'Note' | 'Organization' | 'Project' | 'Routine' | 'SmartContract' | 'Standard', ParentType, ContextType>;
};

export type QuestionSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuestionSearchResult'] = ResolversParentTypes['QuestionSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['QuestionEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuestionTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuestionTranslation'] = ResolversParentTypes['QuestionTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuestionYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuestionYou'] = ResolversParentTypes['QuestionYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canVote?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isUpvoted?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizResolvers<ContextType = any, ParentType extends ResolversParentTypes['Quiz'] = ResolversParentTypes['Quiz']> = {
  attempts?: Resolver<Array<ResolversTypes['QuizAttempt']>, ParentType, ContextType>;
  attemptsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  createdBy?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  project?: Resolver<Maybe<ResolversTypes['Project']>, ParentType, ContextType>;
  quizQuestions?: Resolver<Array<ResolversTypes['QuizQuestion']>, ParentType, ContextType>;
  quizQuestionsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  randomizeQuestionOrder?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  routine?: Resolver<Maybe<ResolversTypes['Routine']>, ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  stats?: Resolver<Array<ResolversTypes['StatsQuiz']>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['QuizTranslation']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  views?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['QuizYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizAttemptResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuizAttempt'] = ResolversParentTypes['QuizAttempt']> = {
  contextSwitches?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  pointsEarned?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  quiz?: Resolver<ResolversTypes['Quiz'], ParentType, ContextType>;
  responses?: Resolver<Array<ResolversTypes['QuizQuestionResponse']>, ParentType, ContextType>;
  responsesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  status?: Resolver<ResolversTypes['QuizAttemptStatus'], ParentType, ContextType>;
  timeTaken?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  user?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['QuizAttemptYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizAttemptEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuizAttemptEdge'] = ResolversParentTypes['QuizAttemptEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['QuizAttempt'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizAttemptSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuizAttemptSearchResult'] = ResolversParentTypes['QuizAttemptSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['QuizAttemptEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizAttemptYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuizAttemptYou'] = ResolversParentTypes['QuizAttemptYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuizEdge'] = ResolversParentTypes['QuizEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Quiz'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizQuestionResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuizQuestion'] = ResolversParentTypes['QuizQuestion']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  order?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  points?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  quiz?: Resolver<ResolversTypes['Quiz'], ParentType, ContextType>;
  responses?: Resolver<Maybe<Array<ResolversTypes['QuizQuestionResponse']>>, ParentType, ContextType>;
  responsesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  standardVersion?: Resolver<Maybe<ResolversTypes['StandardVersion']>, ParentType, ContextType>;
  translations?: Resolver<Maybe<Array<ResolversTypes['QuizQuestionTranslation']>>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['QuizQuestionYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizQuestionEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuizQuestionEdge'] = ResolversParentTypes['QuizQuestionEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['QuizQuestion'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizQuestionResponseResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuizQuestionResponse'] = ResolversParentTypes['QuizQuestionResponse']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  quizAttempt?: Resolver<ResolversTypes['QuizAttempt'], ParentType, ContextType>;
  quizQuestion?: Resolver<ResolversTypes['QuizQuestion'], ParentType, ContextType>;
  response?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['QuizQuestionResponseTranslation']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['QuizQuestionResponseYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizQuestionResponseEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuizQuestionResponseEdge'] = ResolversParentTypes['QuizQuestionResponseEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['QuizQuestionResponse'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizQuestionResponseSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuizQuestionResponseSearchResult'] = ResolversParentTypes['QuizQuestionResponseSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['QuizQuestionResponseEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizQuestionResponseTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuizQuestionResponseTranslation'] = ResolversParentTypes['QuizQuestionResponseTranslation']> = {
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  response?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizQuestionResponseYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuizQuestionResponseYou'] = ResolversParentTypes['QuizQuestionResponseYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizQuestionSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuizQuestionSearchResult'] = ResolversParentTypes['QuizQuestionSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['QuizQuestionEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizQuestionTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuizQuestionTranslation'] = ResolversParentTypes['QuizQuestionTranslation']> = {
  helpText?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  questionText?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizQuestionYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuizQuestionYou'] = ResolversParentTypes['QuizQuestionYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuizSearchResult'] = ResolversParentTypes['QuizSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['QuizEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuizTranslation'] = ResolversParentTypes['QuizTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuizYou'] = ResolversParentTypes['QuizYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canVote?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  hasCompleted?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isUpvoted?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReminderResolvers<ContextType = any, ParentType extends ResolversParentTypes['Reminder'] = ResolversParentTypes['Reminder']> = {
  completed?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  dueDate?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  index?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  reminderItems?: Resolver<Array<ResolversTypes['ReminderItem']>, ParentType, ContextType>;
  reminderList?: Resolver<ResolversTypes['ReminderList'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReminderEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReminderEdge'] = ResolversParentTypes['ReminderEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Reminder'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReminderItemResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReminderItem'] = ResolversParentTypes['ReminderItem']> = {
  completed?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  dueDate?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  index?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  reminder?: Resolver<ResolversTypes['Reminder'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReminderListResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReminderList'] = ResolversParentTypes['ReminderList']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  reminders?: Resolver<Array<ResolversTypes['Reminder']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  userSchedule?: Resolver<Maybe<ResolversTypes['UserSchedule']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReminderListEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReminderListEdge'] = ResolversParentTypes['ReminderListEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['ReminderList'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReminderListSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReminderListSearchResult'] = ResolversParentTypes['ReminderListSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ReminderListEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReminderSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReminderSearchResult'] = ResolversParentTypes['ReminderSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ReminderEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReportResolvers<ContextType = any, ParentType extends ResolversParentTypes['Report'] = ResolversParentTypes['Report']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  details?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<Maybe<ResolversTypes['ID']>, ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  reason?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  responses?: Resolver<Array<ResolversTypes['ReportResponse']>, ParentType, ContextType>;
  responsesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['ReportYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReportEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReportEdge'] = ResolversParentTypes['ReportEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Report'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReportResponseResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReportResponse'] = ResolversParentTypes['ReportResponse']> = {
  actionSuggested?: Resolver<ResolversTypes['ReportSuggestedAction'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  details?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  report?: Resolver<ResolversTypes['Report'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['ReportResponseYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReportResponseEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReportResponseEdge'] = ResolversParentTypes['ReportResponseEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['ReportResponse'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReportResponseSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReportResponseSearchResult'] = ResolversParentTypes['ReportResponseSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ReportResponseEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReportResponseYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReportResponseYou'] = ResolversParentTypes['ReportResponseYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReportSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReportSearchResult'] = ResolversParentTypes['ReportSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ReportEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReportYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReportYou'] = ResolversParentTypes['ReportYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRespond?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReputationHistoryResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReputationHistory'] = ResolversParentTypes['ReputationHistory']> = {
  amount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  event?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  objectId1?: Resolver<Maybe<ResolversTypes['ID']>, ParentType, ContextType>;
  objectId2?: Resolver<Maybe<ResolversTypes['ID']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReputationHistoryEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReputationHistoryEdge'] = ResolversParentTypes['ReputationHistoryEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['ReputationHistory'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReputationHistorySearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReputationHistorySearchResult'] = ResolversParentTypes['ReputationHistorySearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ReputationHistoryEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ResearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ResearchResult'] = ResolversParentTypes['ResearchResult']> = {
  needInvestments?: Resolver<Array<ResolversTypes['Project']>, ParentType, ContextType>;
  needMembers?: Resolver<Array<ResolversTypes['Organization']>, ParentType, ContextType>;
  needVotes?: Resolver<Array<ResolversTypes['Project']>, ParentType, ContextType>;
  newlyCompleted?: Resolver<Array<ResolversTypes['ProjectOrRoutine']>, ParentType, ContextType>;
  processes?: Resolver<Array<ResolversTypes['Routine']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ResourceResolvers<ContextType = any, ParentType extends ResolversParentTypes['Resource'] = ResolversParentTypes['Resource']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  index?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  link?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  list?: Resolver<ResolversTypes['ResourceList'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['ResourceTranslation']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  usedFor?: Resolver<ResolversTypes['ResourceUsedFor'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ResourceEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ResourceEdge'] = ResolversParentTypes['ResourceEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Resource'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ResourceListResolvers<ContextType = any, ParentType extends ResolversParentTypes['ResourceList'] = ResolversParentTypes['ResourceList']> = {
  apiVersion?: Resolver<Maybe<ResolversTypes['ApiVersion']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  organization?: Resolver<Maybe<ResolversTypes['Organization']>, ParentType, ContextType>;
  post?: Resolver<Maybe<ResolversTypes['Post']>, ParentType, ContextType>;
  projectVersion?: Resolver<Maybe<ResolversTypes['ProjectVersion']>, ParentType, ContextType>;
  resources?: Resolver<Array<ResolversTypes['Resource']>, ParentType, ContextType>;
  routineVersion?: Resolver<Maybe<ResolversTypes['RoutineVersion']>, ParentType, ContextType>;
  smartContractVersion?: Resolver<Maybe<ResolversTypes['SmartContractVersion']>, ParentType, ContextType>;
  standardVersion?: Resolver<Maybe<ResolversTypes['StandardVersion']>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['ResourceListTranslation']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  userSchedule?: Resolver<Maybe<ResolversTypes['UserSchedule']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ResourceListEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ResourceListEdge'] = ResolversParentTypes['ResourceListEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['ResourceList'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ResourceListSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ResourceListSearchResult'] = ResolversParentTypes['ResourceListSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ResourceListEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ResourceListTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['ResourceListTranslation'] = ResolversParentTypes['ResourceListTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ResourceSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ResourceSearchResult'] = ResolversParentTypes['ResourceSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ResourceEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ResourceTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['ResourceTranslation'] = ResolversParentTypes['ResourceTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ResponseResolvers<ContextType = any, ParentType extends ResolversParentTypes['Response'] = ResolversParentTypes['Response']> = {
  code?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  message?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoleResolvers<ContextType = any, ParentType extends ResolversParentTypes['Role'] = ResolversParentTypes['Role']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  members?: Resolver<Maybe<Array<ResolversTypes['Member']>>, ParentType, ContextType>;
  membersCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  organization?: Resolver<ResolversTypes['Organization'], ParentType, ContextType>;
  permissions?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['RoleTranslation']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoleEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['RoleEdge'] = ResolversParentTypes['RoleEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Role'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoleSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['RoleSearchResult'] = ResolversParentTypes['RoleSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['RoleEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoleTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['RoleTranslation'] = ResolversParentTypes['RoleTranslation']> = {
  description?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoutineResolvers<ContextType = any, ParentType extends ResolversParentTypes['Routine'] = ResolversParentTypes['Routine']> = {
  completedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  createdBy?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  forks?: Resolver<Array<ResolversTypes['Routine']>, ParentType, ContextType>;
  forksCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  hasCompleteVersion?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isDeleted?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isInternal?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  issues?: Resolver<Array<ResolversTypes['Issue']>, ParentType, ContextType>;
  issuesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  labels?: Resolver<Array<ResolversTypes['Label']>, ParentType, ContextType>;
  owner?: Resolver<Maybe<ResolversTypes['Owner']>, ParentType, ContextType>;
  parent?: Resolver<Maybe<ResolversTypes['RoutineVersion']>, ParentType, ContextType>;
  permissions?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  pullRequests?: Resolver<Array<ResolversTypes['PullRequest']>, ParentType, ContextType>;
  pullRequestsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  questions?: Resolver<Array<ResolversTypes['Question']>, ParentType, ContextType>;
  questionsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  quizzes?: Resolver<Array<ResolversTypes['Quiz']>, ParentType, ContextType>;
  quizzesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  stats?: Resolver<Array<ResolversTypes['StatsRoutine']>, ParentType, ContextType>;
  tags?: Resolver<Array<ResolversTypes['Tag']>, ParentType, ContextType>;
  transfers?: Resolver<Array<ResolversTypes['Transfer']>, ParentType, ContextType>;
  transfersCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  translatedName?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  versions?: Resolver<Array<ResolversTypes['RoutineVersion']>, ParentType, ContextType>;
  versionsCount?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  views?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['RoutineYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoutineEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['RoutineEdge'] = ResolversParentTypes['RoutineEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Routine'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoutineSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['RoutineSearchResult'] = ResolversParentTypes['RoutineSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['RoutineEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoutineVersionResolvers<ContextType = any, ParentType extends ResolversParentTypes['RoutineVersion'] = ResolversParentTypes['RoutineVersion']> = {
  apiCallData?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  apiVersion?: Resolver<Maybe<ResolversTypes['ApiVersion']>, ParentType, ContextType>;
  comments?: Resolver<Array<ResolversTypes['Comment']>, ParentType, ContextType>;
  commentsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  completedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  complexity?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  directoryListings?: Resolver<Array<ResolversTypes['ProjectVersionDirectory']>, ParentType, ContextType>;
  directoryListingsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  forks?: Resolver<Array<ResolversTypes['Routine']>, ParentType, ContextType>;
  forksCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  inputs?: Resolver<Array<ResolversTypes['RoutineVersionInput']>, ParentType, ContextType>;
  inputsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  isAutomatable?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  isComplete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isDeleted?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isLatest?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  nodeLinks?: Resolver<Array<ResolversTypes['NodeLink']>, ParentType, ContextType>;
  nodeLinksCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  nodes?: Resolver<Array<ResolversTypes['Node']>, ParentType, ContextType>;
  nodesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  outputs?: Resolver<Array<ResolversTypes['RoutineVersionOutput']>, ParentType, ContextType>;
  outputsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  pullRequest?: Resolver<Maybe<ResolversTypes['PullRequest']>, ParentType, ContextType>;
  reports?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  reportsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  resourceList?: Resolver<Maybe<ResolversTypes['ResourceList']>, ParentType, ContextType>;
  root?: Resolver<ResolversTypes['Routine'], ParentType, ContextType>;
  simplicity?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  smartContractCallData?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  smartContractVersion?: Resolver<Maybe<ResolversTypes['SmartContractVersion']>, ParentType, ContextType>;
  suggestedNextByRoutineVersion?: Resolver<Array<ResolversTypes['RoutineVersion']>, ParentType, ContextType>;
  suggestedNextByRoutineVersionCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  timesCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  timesStarted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['RoutineVersionTranslation']>, ParentType, ContextType>;
  translationsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  versionIndex?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  versionLabel?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  versionNotes?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  you?: Resolver<ResolversTypes['RoutineVersionYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoutineVersionEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['RoutineVersionEdge'] = ResolversParentTypes['RoutineVersionEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['RoutineVersion'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoutineVersionInputResolvers<ContextType = any, ParentType extends ResolversParentTypes['RoutineVersionInput'] = ResolversParentTypes['RoutineVersionInput']> = {
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  index?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  isRequired?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  name?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  routineVersion?: Resolver<ResolversTypes['RoutineVersion'], ParentType, ContextType>;
  standardVersion?: Resolver<Maybe<ResolversTypes['StandardVersion']>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['RoutineVersionInputTranslation']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoutineVersionInputTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['RoutineVersionInputTranslation'] = ResolversParentTypes['RoutineVersionInputTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  helpText?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoutineVersionOutputResolvers<ContextType = any, ParentType extends ResolversParentTypes['RoutineVersionOutput'] = ResolversParentTypes['RoutineVersionOutput']> = {
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  index?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  isRequired?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  name?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  routineVersion?: Resolver<ResolversTypes['RoutineVersion'], ParentType, ContextType>;
  standardVersion?: Resolver<Maybe<ResolversTypes['StandardVersion']>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['RoutineVersionOutputTranslation']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoutineVersionOutputTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['RoutineVersionOutputTranslation'] = ResolversParentTypes['RoutineVersionOutputTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  helpText?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoutineVersionSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['RoutineVersionSearchResult'] = ResolversParentTypes['RoutineVersionSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['RoutineVersionEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoutineVersionTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['RoutineVersionTranslation'] = ResolversParentTypes['RoutineVersionTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  instructions?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoutineVersionYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['RoutineVersionYou'] = ResolversParentTypes['RoutineVersionYou']> = {
  canComment?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canCopy?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReport?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRun?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canVote?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  runs?: Resolver<Array<ResolversTypes['RunRoutine']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoutineYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['RoutineYou'] = ResolversParentTypes['RoutineYou']> = {
  canComment?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canTransfer?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canVote?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isUpvoted?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  isViewed?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunProjectResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunProject'] = ResolversParentTypes['RunProject']> = {
  completedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  completedComplexity?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  contextSwitches?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  organization?: Resolver<Maybe<ResolversTypes['Organization']>, ParentType, ContextType>;
  projectVersion?: Resolver<Maybe<ResolversTypes['ProjectVersion']>, ParentType, ContextType>;
  runProjectSchedule?: Resolver<Maybe<ResolversTypes['RunProjectSchedule']>, ParentType, ContextType>;
  startedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  status?: Resolver<ResolversTypes['RunStatus'], ParentType, ContextType>;
  steps?: Resolver<Array<ResolversTypes['RunProjectStep']>, ParentType, ContextType>;
  stepsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  timeElapsed?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  user?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  wasRunAutomaticaly?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['RunProjectYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunProjectEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunProjectEdge'] = ResolversParentTypes['RunProjectEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['RunProject'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunProjectOrRunRoutineResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunProjectOrRunRoutine'] = ResolversParentTypes['RunProjectOrRunRoutine']> = {
  __resolveType: TypeResolveFn<'RunProject' | 'RunRoutine', ParentType, ContextType>;
};

export type RunProjectOrRunRoutineEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunProjectOrRunRoutineEdge'] = ResolversParentTypes['RunProjectOrRunRoutineEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['RunProjectOrRunRoutine'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunProjectOrRunRoutinePageInfoResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunProjectOrRunRoutinePageInfo'] = ResolversParentTypes['RunProjectOrRunRoutinePageInfo']> = {
  endCursorProject?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  endCursorRoutine?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  hasNextPage?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunProjectOrRunRoutineSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunProjectOrRunRoutineSearchResult'] = ResolversParentTypes['RunProjectOrRunRoutineSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['RunProjectOrRunRoutineEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['RunProjectOrRunRoutinePageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunProjectScheduleResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunProjectSchedule'] = ResolversParentTypes['RunProjectSchedule']> = {
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  labels?: Resolver<Array<ResolversTypes['Label']>, ParentType, ContextType>;
  recurrEnd?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  recurrStart?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  recurring?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  runProject?: Resolver<ResolversTypes['RunProject'], ParentType, ContextType>;
  timeZone?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['RunProjectScheduleTranslation']>, ParentType, ContextType>;
  windowEnd?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  windowStart?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunProjectScheduleEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunProjectScheduleEdge'] = ResolversParentTypes['RunProjectScheduleEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['RunProjectSchedule'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunProjectScheduleSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunProjectScheduleSearchResult'] = ResolversParentTypes['RunProjectScheduleSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['RunProjectScheduleEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunProjectScheduleTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunProjectScheduleTranslation'] = ResolversParentTypes['RunProjectScheduleTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunProjectSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunProjectSearchResult'] = ResolversParentTypes['RunProjectSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['RunProjectEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunProjectStepResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunProjectStep'] = ResolversParentTypes['RunProjectStep']> = {
  completedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  contextSwitches?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  directory?: Resolver<Maybe<ResolversTypes['ProjectVersionDirectory']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<Maybe<ResolversTypes['Node']>, ParentType, ContextType>;
  order?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  run?: Resolver<ResolversTypes['RunProject'], ParentType, ContextType>;
  startedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  status?: Resolver<ResolversTypes['RunProjectStepStatus'], ParentType, ContextType>;
  step?: Resolver<Array<ResolversTypes['Int']>, ParentType, ContextType>;
  timeElapsed?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunProjectYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunProjectYou'] = ResolversParentTypes['RunProjectYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunRoutineResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunRoutine'] = ResolversParentTypes['RunRoutine']> = {
  completedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  completedComplexity?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  contextSwitches?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  inputs?: Resolver<Array<ResolversTypes['RunRoutineInput']>, ParentType, ContextType>;
  inputsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  organization?: Resolver<Maybe<ResolversTypes['Organization']>, ParentType, ContextType>;
  routineVersion?: Resolver<Maybe<ResolversTypes['RoutineVersion']>, ParentType, ContextType>;
  runProject?: Resolver<Maybe<ResolversTypes['RunProject']>, ParentType, ContextType>;
  runRoutineSchedule?: Resolver<Maybe<ResolversTypes['RunRoutineSchedule']>, ParentType, ContextType>;
  startedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  status?: Resolver<ResolversTypes['RunStatus'], ParentType, ContextType>;
  steps?: Resolver<Array<ResolversTypes['RunRoutineStep']>, ParentType, ContextType>;
  stepsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  timeElapsed?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  user?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  wasRunAutomaticaly?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['RunRoutineYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunRoutineEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunRoutineEdge'] = ResolversParentTypes['RunRoutineEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['RunRoutine'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunRoutineInputResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunRoutineInput'] = ResolversParentTypes['RunRoutineInput']> = {
  data?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  input?: Resolver<ResolversTypes['RoutineVersionInput'], ParentType, ContextType>;
  runRoutine?: Resolver<ResolversTypes['RunRoutine'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunRoutineInputEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunRoutineInputEdge'] = ResolversParentTypes['RunRoutineInputEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['RunRoutineInput'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunRoutineInputSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunRoutineInputSearchResult'] = ResolversParentTypes['RunRoutineInputSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['RunRoutineInputEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunRoutineScheduleResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunRoutineSchedule'] = ResolversParentTypes['RunRoutineSchedule']> = {
  attemptAutomatic?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  labels?: Resolver<Array<ResolversTypes['Label']>, ParentType, ContextType>;
  maxAutomaticAttempts?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  recurrEnd?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  recurrStart?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  recurring?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  runRoutine?: Resolver<ResolversTypes['RunRoutine'], ParentType, ContextType>;
  timeZone?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['RunRoutineScheduleTranslation']>, ParentType, ContextType>;
  windowEnd?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  windowStart?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunRoutineScheduleEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunRoutineScheduleEdge'] = ResolversParentTypes['RunRoutineScheduleEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['RunRoutineSchedule'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunRoutineScheduleSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunRoutineScheduleSearchResult'] = ResolversParentTypes['RunRoutineScheduleSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['RunRoutineScheduleEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunRoutineScheduleTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunRoutineScheduleTranslation'] = ResolversParentTypes['RunRoutineScheduleTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunRoutineSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunRoutineSearchResult'] = ResolversParentTypes['RunRoutineSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['RunRoutineEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunRoutineStepResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunRoutineStep'] = ResolversParentTypes['RunRoutineStep']> = {
  completedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  contextSwitches?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<Maybe<ResolversTypes['Node']>, ParentType, ContextType>;
  order?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  run?: Resolver<ResolversTypes['RunRoutine'], ParentType, ContextType>;
  startedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  status?: Resolver<ResolversTypes['RunRoutineStepStatus'], ParentType, ContextType>;
  step?: Resolver<Array<ResolversTypes['Int']>, ParentType, ContextType>;
  subroutine?: Resolver<Maybe<ResolversTypes['RoutineVersion']>, ParentType, ContextType>;
  timeElapsed?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunRoutineYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunRoutineYou'] = ResolversParentTypes['RunRoutineYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type SessionResolvers<ContextType = any, ParentType extends ResolversParentTypes['Session'] = ResolversParentTypes['Session']> = {
  isLoggedIn?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  timeZone?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  users?: Resolver<Maybe<Array<ResolversTypes['SessionUser']>>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type SessionUserResolvers<ContextType = any, ParentType extends ResolversParentTypes['SessionUser'] = ResolversParentTypes['SessionUser']> = {
  handle?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  hasPremium?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  languages?: Resolver<Array<ResolversTypes['String']>, ParentType, ContextType>;
  name?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  schedules?: Resolver<Array<ResolversTypes['UserSchedule']>, ParentType, ContextType>;
  theme?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type SmartContractResolvers<ContextType = any, ParentType extends ResolversParentTypes['SmartContract'] = ResolversParentTypes['SmartContract']> = {
  completedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  createdBy?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  hasCompleteVersion?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isDeleted?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  issues?: Resolver<Array<ResolversTypes['Issue']>, ParentType, ContextType>;
  issuesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  labels?: Resolver<Array<ResolversTypes['Label']>, ParentType, ContextType>;
  owner?: Resolver<Maybe<ResolversTypes['Owner']>, ParentType, ContextType>;
  parent?: Resolver<Maybe<ResolversTypes['SmartContractVersion']>, ParentType, ContextType>;
  permissions?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  pullRequests?: Resolver<Array<ResolversTypes['PullRequest']>, ParentType, ContextType>;
  pullRequestsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  questions?: Resolver<Array<ResolversTypes['Question']>, ParentType, ContextType>;
  questionsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  stats?: Resolver<Array<ResolversTypes['StatsSmartContract']>, ParentType, ContextType>;
  tags?: Resolver<Array<ResolversTypes['Tag']>, ParentType, ContextType>;
  transfers?: Resolver<Array<ResolversTypes['Transfer']>, ParentType, ContextType>;
  transfersCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  translatedName?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  versions?: Resolver<Array<ResolversTypes['SmartContractVersion']>, ParentType, ContextType>;
  versionsCount?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  views?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['SmartContractYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type SmartContractEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['SmartContractEdge'] = ResolversParentTypes['SmartContractEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['SmartContract'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type SmartContractSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['SmartContractSearchResult'] = ResolversParentTypes['SmartContractSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['SmartContractEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type SmartContractVersionResolvers<ContextType = any, ParentType extends ResolversParentTypes['SmartContractVersion'] = ResolversParentTypes['SmartContractVersion']> = {
  comments?: Resolver<Array<ResolversTypes['Comment']>, ParentType, ContextType>;
  commentsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  completedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  content?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  contractType?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  default?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  directoryListings?: Resolver<Array<ResolversTypes['ProjectVersionDirectory']>, ParentType, ContextType>;
  directoryListingsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  forks?: Resolver<Array<ResolversTypes['SmartContract']>, ParentType, ContextType>;
  forksCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isComplete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isDeleted?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isLatest?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  pullRequest?: Resolver<Maybe<ResolversTypes['PullRequest']>, ParentType, ContextType>;
  reports?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  reportsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  resourceList?: Resolver<Maybe<ResolversTypes['ResourceList']>, ParentType, ContextType>;
  root?: Resolver<ResolversTypes['SmartContract'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['SmartContractVersionTranslation']>, ParentType, ContextType>;
  translationsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  versionIndex?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  versionLabel?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  versionNotes?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  you?: Resolver<ResolversTypes['VersionYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type SmartContractVersionEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['SmartContractVersionEdge'] = ResolversParentTypes['SmartContractVersionEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['SmartContractVersion'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type SmartContractVersionSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['SmartContractVersionSearchResult'] = ResolversParentTypes['SmartContractVersionSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['SmartContractVersionEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type SmartContractVersionTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['SmartContractVersionTranslation'] = ResolversParentTypes['SmartContractVersionTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  jsonVariable?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type SmartContractYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['SmartContractYou'] = ResolversParentTypes['SmartContractYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canTransfer?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canVote?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isUpvoted?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  isViewed?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StandardResolvers<ContextType = any, ParentType extends ResolversParentTypes['Standard'] = ResolversParentTypes['Standard']> = {
  completedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  createdBy?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  forks?: Resolver<Array<ResolversTypes['Standard']>, ParentType, ContextType>;
  forksCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  hasCompleteVersion?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isDeleted?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isInternal?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  issues?: Resolver<Array<ResolversTypes['Issue']>, ParentType, ContextType>;
  issuesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  labels?: Resolver<Array<ResolversTypes['Label']>, ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  owner?: Resolver<Maybe<ResolversTypes['Owner']>, ParentType, ContextType>;
  parent?: Resolver<Maybe<ResolversTypes['StandardVersion']>, ParentType, ContextType>;
  permissions?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  pullRequests?: Resolver<Array<ResolversTypes['PullRequest']>, ParentType, ContextType>;
  pullRequestsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  questions?: Resolver<Array<ResolversTypes['Question']>, ParentType, ContextType>;
  questionsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  stats?: Resolver<Array<ResolversTypes['StatsStandard']>, ParentType, ContextType>;
  tags?: Resolver<Array<ResolversTypes['Tag']>, ParentType, ContextType>;
  transfers?: Resolver<Array<ResolversTypes['Transfer']>, ParentType, ContextType>;
  transfersCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  translatedName?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  versions?: Resolver<Array<ResolversTypes['StandardVersion']>, ParentType, ContextType>;
  versionsCount?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  views?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['StandardYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StandardEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['StandardEdge'] = ResolversParentTypes['StandardEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Standard'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StandardSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['StandardSearchResult'] = ResolversParentTypes['StandardSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['StandardEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StandardVersionResolvers<ContextType = any, ParentType extends ResolversParentTypes['StandardVersion'] = ResolversParentTypes['StandardVersion']> = {
  comments?: Resolver<Array<ResolversTypes['Comment']>, ParentType, ContextType>;
  commentsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  completedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  default?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  directoryListings?: Resolver<Array<ResolversTypes['ProjectVersionDirectory']>, ParentType, ContextType>;
  directoryListingsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  forks?: Resolver<Array<ResolversTypes['Standard']>, ParentType, ContextType>;
  forksCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isComplete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isDeleted?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isFile?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  isLatest?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  props?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  pullRequest?: Resolver<Maybe<ResolversTypes['PullRequest']>, ParentType, ContextType>;
  reports?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  reportsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  resourceList?: Resolver<Maybe<ResolversTypes['ResourceList']>, ParentType, ContextType>;
  root?: Resolver<ResolversTypes['Standard'], ParentType, ContextType>;
  standardType?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['StandardVersionTranslation']>, ParentType, ContextType>;
  translationsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  versionIndex?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  versionLabel?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  versionNotes?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  you?: Resolver<ResolversTypes['VersionYou'], ParentType, ContextType>;
  yup?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StandardVersionEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['StandardVersionEdge'] = ResolversParentTypes['StandardVersionEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['StandardVersion'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StandardVersionSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['StandardVersionSearchResult'] = ResolversParentTypes['StandardVersionSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['StandardVersionEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StandardVersionTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['StandardVersionTranslation'] = ResolversParentTypes['StandardVersionTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  jsonVariable?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StandardYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['StandardYou'] = ResolversParentTypes['StandardYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canTransfer?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canVote?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isUpvoted?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  isViewed?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StarResolvers<ContextType = any, ParentType extends ResolversParentTypes['Bookmark'] = ResolversParentTypes['Bookmark']> = {
  by?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  to?: Resolver<ResolversTypes['BookmarkTo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type BookmarkResolvers<ContextType = any, ParentType extends ResolversParentTypes['Bookmark'] = ResolversParentTypes['Bookmark']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Bookmark'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type BookmarkSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['BookmarkSearchResult'] = ResolversParentTypes['BookmarkSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['Bookmark']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type BookmarkToResolvers<ContextType = any, ParentType extends ResolversParentTypes['BookmarkTo'] = ResolversParentTypes['BookmarkTo']> = {
  __resolveType: TypeResolveFn<'Api' | 'Comment' | 'Issue' | 'Note' | 'Organization' | 'Post' | 'Project' | 'Question' | 'QuestionAnswer' | 'Quiz' | 'Routine' | 'SmartContract' | 'Standard' | 'Tag' | 'User', ParentType, ContextType>;
};

export type StatsApiResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsApi'] = ResolversParentTypes['StatsApi']> = {
  calls?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  periodEnd?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodStart?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodType?: Resolver<ResolversTypes['StatPeriodType'], ParentType, ContextType>;
  routineVersions?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsApiEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsApiEdge'] = ResolversParentTypes['StatsApiEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['StatsApi'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsApiSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsApiSearchResult'] = ResolversParentTypes['StatsApiSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['StatsApiEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsOrganizationResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsOrganization'] = ResolversParentTypes['StatsOrganization']> = {
  apis?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  members?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  notes?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  periodEnd?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodStart?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodType?: Resolver<ResolversTypes['StatPeriodType'], ParentType, ContextType>;
  projects?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  routines?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  runRoutineCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  runRoutineContextSwitchesAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  runRoutinesCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  runRoutinesStarted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  smartContracts?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  standards?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsOrganizationEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsOrganizationEdge'] = ResolversParentTypes['StatsOrganizationEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['StatsOrganization'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsOrganizationSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsOrganizationSearchResult'] = ResolversParentTypes['StatsOrganizationSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['StatsOrganizationEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsProjectResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsProject'] = ResolversParentTypes['StatsProject']> = {
  apis?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  directories?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  notes?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  organizations?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  periodEnd?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodStart?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodType?: Resolver<ResolversTypes['StatPeriodType'], ParentType, ContextType>;
  projects?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  routines?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  runCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  runContextSwitchesAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  runsCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  runsStarted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  smartContracts?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  standards?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsProjectEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsProjectEdge'] = ResolversParentTypes['StatsProjectEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['StatsProject'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsProjectSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsProjectSearchResult'] = ResolversParentTypes['StatsProjectSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['StatsProjectEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsQuizResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsQuiz'] = ResolversParentTypes['StatsQuiz']> = {
  completionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  periodEnd?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodStart?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodType?: Resolver<ResolversTypes['StatPeriodType'], ParentType, ContextType>;
  scoreAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  timesFailed?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  timesPassed?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  timesStarted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsQuizEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsQuizEdge'] = ResolversParentTypes['StatsQuizEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['StatsQuiz'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsQuizSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsQuizSearchResult'] = ResolversParentTypes['StatsQuizSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['StatsQuizEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsRoutineResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsRoutine'] = ResolversParentTypes['StatsRoutine']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  periodEnd?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodStart?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodType?: Resolver<ResolversTypes['StatPeriodType'], ParentType, ContextType>;
  runCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  runContextSwitchesAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  runsCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  runsStarted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsRoutineEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsRoutineEdge'] = ResolversParentTypes['StatsRoutineEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['StatsRoutine'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsRoutineSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsRoutineSearchResult'] = ResolversParentTypes['StatsRoutineSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['StatsRoutineEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsSiteResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsSite'] = ResolversParentTypes['StatsSite']> = {
  activeUsers?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  apiCalls?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  apisCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  organizationsCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  periodEnd?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodStart?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodType?: Resolver<ResolversTypes['StatPeriodType'], ParentType, ContextType>;
  projectCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  projectsCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  projectsCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  quizzesCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  quizzesCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  routineCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  routineComplexityAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  routineSimplicityAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  routinesCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  routinesCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  runProjectCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  runProjectContextSwitchesAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  runProjectsCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  runProjectsStarted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  runRoutineCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  runRoutineContextSwitchesAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  runRoutinesCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  runRoutinesStarted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  smartContractCalls?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  smartContractCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  smartContractsCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  smartContractsCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  standardCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  standardsCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  standardsCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  verifiedEmailsCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  verifiedWalletsCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsSiteEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsSiteEdge'] = ResolversParentTypes['StatsSiteEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['StatsSite'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsSiteSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsSiteSearchResult'] = ResolversParentTypes['StatsSiteSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['StatsSiteEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsSmartContractResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsSmartContract'] = ResolversParentTypes['StatsSmartContract']> = {
  calls?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  periodEnd?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodStart?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodType?: Resolver<ResolversTypes['StatPeriodType'], ParentType, ContextType>;
  routineVersions?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsSmartContractEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsSmartContractEdge'] = ResolversParentTypes['StatsSmartContractEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['StatsSmartContract'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsSmartContractSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsSmartContractSearchResult'] = ResolversParentTypes['StatsSmartContractSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['StatsSmartContractEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsStandardResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsStandard'] = ResolversParentTypes['StatsStandard']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  linksToInputs?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  linksToOutputs?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  periodEnd?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodStart?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodType?: Resolver<ResolversTypes['StatPeriodType'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsStandardEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsStandardEdge'] = ResolversParentTypes['StatsStandardEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['StatsStandard'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsStandardSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsStandardSearchResult'] = ResolversParentTypes['StatsStandardSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['StatsStandardEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsUserResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsUser'] = ResolversParentTypes['StatsUser']> = {
  apisCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  organizationsCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  periodEnd?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodStart?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodType?: Resolver<ResolversTypes['StatPeriodType'], ParentType, ContextType>;
  projectCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  projectsCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  projectsCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  quizzesFailed?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  quizzesPassed?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  routineCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  routinesCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  routinesCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  runProjectCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  runProjectContextSwitchesAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  runProjectsCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  runProjectsStarted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  runRoutineCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  runRoutineContextSwitchesAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  runRoutinesCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  runRoutinesStarted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  smartContractCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  smartContractsCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  smartContractsCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  standardCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  standardsCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  standardsCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsUserEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsUserEdge'] = ResolversParentTypes['StatsUserEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['StatsUser'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsUserSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsUserSearchResult'] = ResolversParentTypes['StatsUserSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['StatsUserEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type SubscribedObjectResolvers<ContextType = any, ParentType extends ResolversParentTypes['SubscribedObject'] = ResolversParentTypes['SubscribedObject']> = {
  __resolveType: TypeResolveFn<'Api' | 'Comment' | 'Issue' | 'Meeting' | 'Note' | 'Organization' | 'Project' | 'PullRequest' | 'Question' | 'Quiz' | 'Report' | 'Routine' | 'SmartContract' | 'Standard', ParentType, ContextType>;
};

export type SuccessResolvers<ContextType = any, ParentType extends ResolversParentTypes['Success'] = ResolversParentTypes['Success']> = {
  success?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type TagResolvers<ContextType = any, ParentType extends ResolversParentTypes['Tag'] = ResolversParentTypes['Tag']> = {
  apis?: Resolver<Array<ResolversTypes['Api']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  notes?: Resolver<Array<ResolversTypes['Note']>, ParentType, ContextType>;
  organizations?: Resolver<Array<ResolversTypes['Organization']>, ParentType, ContextType>;
  posts?: Resolver<Array<ResolversTypes['Post']>, ParentType, ContextType>;
  projects?: Resolver<Array<ResolversTypes['Project']>, ParentType, ContextType>;
  reports?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  routines?: Resolver<Array<ResolversTypes['Routine']>, ParentType, ContextType>;
  smartContracts?: Resolver<Array<ResolversTypes['SmartContract']>, ParentType, ContextType>;
  standards?: Resolver<Array<ResolversTypes['Standard']>, ParentType, ContextType>;
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  tag?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['TagTranslation']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['TagYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type TagEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['TagEdge'] = ResolversParentTypes['TagEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Tag'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type TagSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['TagSearchResult'] = ResolversParentTypes['TagSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['TagEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type TagTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['TagTranslation'] = ResolversParentTypes['TagTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type TagYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['TagYou'] = ResolversParentTypes['TagYou']> = {
  isOwn?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type TransferResolvers<ContextType = any, ParentType extends ResolversParentTypes['Transfer'] = ResolversParentTypes['Transfer']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  fromOwner?: Resolver<Maybe<ResolversTypes['Owner']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  mergedOrRejectedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  object?: Resolver<ResolversTypes['TransferObject'], ParentType, ContextType>;
  status?: Resolver<ResolversTypes['TransferStatus'], ParentType, ContextType>;
  toOwner?: Resolver<Maybe<ResolversTypes['Owner']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['TransferYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type TransferEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['TransferEdge'] = ResolversParentTypes['TransferEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Transfer'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type TransferObjectResolvers<ContextType = any, ParentType extends ResolversParentTypes['TransferObject'] = ResolversParentTypes['TransferObject']> = {
  __resolveType: TypeResolveFn<'Api' | 'Note' | 'Project' | 'Routine' | 'SmartContract' | 'Standard', ParentType, ContextType>;
};

export type TransferSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['TransferSearchResult'] = ResolversParentTypes['TransferSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['TransferEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type TransferYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['TransferYou'] = ResolversParentTypes['TransferYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type TranslateResolvers<ContextType = any, ParentType extends ResolversParentTypes['Translate'] = ResolversParentTypes['Translate']> = {
  fields?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export interface UploadScalarConfig extends GraphQLScalarTypeConfig<ResolversTypes['Upload'], any> {
  name: 'Upload';
}

export type UserResolvers<ContextType = any, ParentType extends ResolversParentTypes['User'] = ResolversParentTypes['User']> = {
  apiKeys?: Resolver<Maybe<Array<ResolversTypes['ApiKey']>>, ParentType, ContextType>;
  apis?: Resolver<Array<ResolversTypes['Api']>, ParentType, ContextType>;
  apisCreated?: Resolver<Maybe<Array<ResolversTypes['Api']>>, ParentType, ContextType>;
  awards?: Resolver<Maybe<Array<ResolversTypes['Award']>>, ParentType, ContextType>;
  comments?: Resolver<Maybe<Array<ResolversTypes['Comment']>>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  emails?: Resolver<Maybe<Array<ResolversTypes['Email']>>, ParentType, ContextType>;
  handle?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  invitedByUser?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  invitedUsers?: Resolver<Maybe<Array<ResolversTypes['User']>>, ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateApis?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateApisCreated?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateMemberships?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateOrganizationsCreated?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateProjects?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateProjectsCreated?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivatePullRequests?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateQuestionsAnswered?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateQuestionsAsked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateQuizzesCreated?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateRoles?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateRoutines?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateRoutinesCreated?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateSmartContracts?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateStandards?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateStandardsCreated?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateBookmarks?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateVotes?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  issuesClosed?: Resolver<Maybe<Array<ResolversTypes['Issue']>>, ParentType, ContextType>;
  issuesCreated?: Resolver<Maybe<Array<ResolversTypes['Issue']>>, ParentType, ContextType>;
  labels?: Resolver<Maybe<Array<ResolversTypes['Label']>>, ParentType, ContextType>;
  languages?: Resolver<Maybe<Array<ResolversTypes['String']>>, ParentType, ContextType>;
  meetingsAttending?: Resolver<Maybe<Array<ResolversTypes['Meeting']>>, ParentType, ContextType>;
  meetingsInvited?: Resolver<Maybe<Array<ResolversTypes['MeetingInvite']>>, ParentType, ContextType>;
  memberships?: Resolver<Maybe<Array<ResolversTypes['Member']>>, ParentType, ContextType>;
  membershipsInvited?: Resolver<Maybe<Array<ResolversTypes['MemberInvite']>>, ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  notes?: Resolver<Maybe<Array<ResolversTypes['Note']>>, ParentType, ContextType>;
  notesCreated?: Resolver<Maybe<Array<ResolversTypes['Note']>>, ParentType, ContextType>;
  notificationSettings?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  notificationSubscriptions?: Resolver<Maybe<Array<ResolversTypes['NotificationSubscription']>>, ParentType, ContextType>;
  notifications?: Resolver<Maybe<Array<ResolversTypes['Notification']>>, ParentType, ContextType>;
  organizationsCreate?: Resolver<Maybe<Array<ResolversTypes['Organization']>>, ParentType, ContextType>;
  paymentHistory?: Resolver<Maybe<Array<ResolversTypes['Payment']>>, ParentType, ContextType>;
  premium?: Resolver<Maybe<ResolversTypes['Premium']>, ParentType, ContextType>;
  projects?: Resolver<Maybe<Array<ResolversTypes['Project']>>, ParentType, ContextType>;
  projectsCreated?: Resolver<Maybe<Array<ResolversTypes['Project']>>, ParentType, ContextType>;
  pullRequests?: Resolver<Maybe<Array<ResolversTypes['PullRequest']>>, ParentType, ContextType>;
  pushDevices?: Resolver<Maybe<Array<ResolversTypes['PushDevice']>>, ParentType, ContextType>;
  questionsAnswered?: Resolver<Maybe<Array<ResolversTypes['QuestionAnswer']>>, ParentType, ContextType>;
  questionsAsked?: Resolver<Maybe<Array<ResolversTypes['Question']>>, ParentType, ContextType>;
  quizzesCreated?: Resolver<Maybe<Array<ResolversTypes['Quiz']>>, ParentType, ContextType>;
  quizzesTaken?: Resolver<Maybe<Array<ResolversTypes['Quiz']>>, ParentType, ContextType>;
  reportResponses?: Resolver<Maybe<Array<ResolversTypes['ReportResponse']>>, ParentType, ContextType>;
  reportsCreated?: Resolver<Maybe<Array<ResolversTypes['Report']>>, ParentType, ContextType>;
  reportsReceived?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  reportsReceivedCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  reputationHistory?: Resolver<Maybe<Array<ResolversTypes['ReputationHistory']>>, ParentType, ContextType>;
  roles?: Resolver<Maybe<Array<ResolversTypes['Role']>>, ParentType, ContextType>;
  routines?: Resolver<Maybe<Array<ResolversTypes['Routine']>>, ParentType, ContextType>;
  routinesCreated?: Resolver<Maybe<Array<ResolversTypes['Routine']>>, ParentType, ContextType>;
  runProjects?: Resolver<Maybe<Array<ResolversTypes['RunProject']>>, ParentType, ContextType>;
  runRoutines?: Resolver<Maybe<Array<ResolversTypes['RunRoutine']>>, ParentType, ContextType>;
  schedules?: Resolver<Maybe<Array<ResolversTypes['UserSchedule']>>, ParentType, ContextType>;
  sentReports?: Resolver<Maybe<Array<ResolversTypes['Report']>>, ParentType, ContextType>;
  smartContracts?: Resolver<Maybe<Array<ResolversTypes['SmartContract']>>, ParentType, ContextType>;
  smartContractsCreated?: Resolver<Maybe<Array<ResolversTypes['SmartContract']>>, ParentType, ContextType>;
  standards?: Resolver<Maybe<Array<ResolversTypes['Standard']>>, ParentType, ContextType>;
  standardsCreated?: Resolver<Maybe<Array<ResolversTypes['Standard']>>, ParentType, ContextType>;
  starred?: Resolver<Maybe<Array<ResolversTypes['Bookmark']>>, ParentType, ContextType>;
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  stats?: Resolver<Maybe<ResolversTypes['StatsUser']>, ParentType, ContextType>;
  status?: Resolver<Maybe<ResolversTypes['AccountStatus']>, ParentType, ContextType>;
  tags?: Resolver<Maybe<Array<ResolversTypes['Tag']>>, ParentType, ContextType>;
  theme?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  transfersIncoming?: Resolver<Maybe<Array<ResolversTypes['Transfer']>>, ParentType, ContextType>;
  transfersOutgoing?: Resolver<Maybe<Array<ResolversTypes['Transfer']>>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['UserTranslation']>, ParentType, ContextType>;
  updated_at?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  viewed?: Resolver<Maybe<Array<ResolversTypes['View']>>, ParentType, ContextType>;
  viewedBy?: Resolver<Maybe<Array<ResolversTypes['View']>>, ParentType, ContextType>;
  views?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  voted?: Resolver<Maybe<Array<ResolversTypes['Vote']>>, ParentType, ContextType>;
  wallets?: Resolver<Maybe<Array<ResolversTypes['Wallet']>>, ParentType, ContextType>;
  you?: Resolver<ResolversTypes['UserYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type UserEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['UserEdge'] = ResolversParentTypes['UserEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type UserScheduleResolvers<ContextType = any, ParentType extends ResolversParentTypes['UserSchedule'] = ResolversParentTypes['UserSchedule']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  eventEnd?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  eventStart?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  filters?: Resolver<Array<ResolversTypes['UserScheduleFilter']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  labels?: Resolver<Array<ResolversTypes['Label']>, ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  recurrEnd?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  recurrStart?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  recurring?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  reminderList?: Resolver<Maybe<ResolversTypes['ReminderList']>, ParentType, ContextType>;
  timeZone?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type UserScheduleEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['UserScheduleEdge'] = ResolversParentTypes['UserScheduleEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['UserSchedule'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type UserScheduleFilterResolvers<ContextType = any, ParentType extends ResolversParentTypes['UserScheduleFilter'] = ResolversParentTypes['UserScheduleFilter']> = {
  filterType?: Resolver<ResolversTypes['UserScheduleFilterType'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  tag?: Resolver<ResolversTypes['Tag'], ParentType, ContextType>;
  userSchedule?: Resolver<ResolversTypes['UserSchedule'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type UserScheduleSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['UserScheduleSearchResult'] = ResolversParentTypes['UserScheduleSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['UserScheduleEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type UserSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['UserSearchResult'] = ResolversParentTypes['UserSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['UserEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type UserTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['UserTranslation'] = ResolversParentTypes['UserTranslation']> = {
  bio?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type UserYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['UserYou'] = ResolversParentTypes['UserYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReport?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isViewed?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type VersionYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['VersionYou'] = ResolversParentTypes['VersionYou']> = {
  canComment?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canCopy?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReport?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUse?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ViewResolvers<ContextType = any, ParentType extends ResolversParentTypes['View'] = ResolversParentTypes['View']> = {
  by?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  lastViewedAt?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  to?: Resolver<ResolversTypes['ViewTo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ViewEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ViewEdge'] = ResolversParentTypes['ViewEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['View'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ViewSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ViewSearchResult'] = ResolversParentTypes['ViewSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ViewEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ViewToResolvers<ContextType = any, ParentType extends ResolversParentTypes['ViewTo'] = ResolversParentTypes['ViewTo']> = {
  __resolveType: TypeResolveFn<'Api' | 'Issue' | 'Note' | 'Organization' | 'Post' | 'Project' | 'Question' | 'Routine' | 'SmartContract' | 'Standard' | 'User', ParentType, ContextType>;
};

export type VoteResolvers<ContextType = any, ParentType extends ResolversParentTypes['Vote'] = ResolversParentTypes['Vote']> = {
  by?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isUpvote?: Resolver<Maybe<ResolversTypes['Boolean']>, ParentType, ContextType>;
  to?: Resolver<ResolversTypes['VoteTo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type VoteEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['VoteEdge'] = ResolversParentTypes['VoteEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Vote'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type VoteSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['VoteSearchResult'] = ResolversParentTypes['VoteSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['VoteEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type VoteToResolvers<ContextType = any, ParentType extends ResolversParentTypes['VoteTo'] = ResolversParentTypes['VoteTo']> = {
  __resolveType: TypeResolveFn<'Api' | 'Comment' | 'Issue' | 'Note' | 'Post' | 'Project' | 'Question' | 'QuestionAnswer' | 'Quiz' | 'Routine' | 'SmartContract' | 'Standard', ParentType, ContextType>;
};

export type WalletResolvers<ContextType = any, ParentType extends ResolversParentTypes['Wallet'] = ResolversParentTypes['Wallet']> = {
  handles?: Resolver<Array<ResolversTypes['Handle']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  name?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  organization?: Resolver<Maybe<ResolversTypes['Organization']>, ParentType, ContextType>;
  publicAddress?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  stakingAddress?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  user?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  verified?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type WalletCompleteResolvers<ContextType = any, ParentType extends ResolversParentTypes['WalletComplete'] = ResolversParentTypes['WalletComplete']> = {
  firstLogIn?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  session?: Resolver<Maybe<ResolversTypes['Session']>, ParentType, ContextType>;
  wallet?: Resolver<Maybe<ResolversTypes['Wallet']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type Resolvers<ContextType = any> = {
  AnyRun?: AnyRunResolvers<ContextType>;
  Api?: ApiResolvers<ContextType>;
  ApiEdge?: ApiEdgeResolvers<ContextType>;
  ApiKey?: ApiKeyResolvers<ContextType>;
  ApiSearchResult?: ApiSearchResultResolvers<ContextType>;
  ApiVersion?: ApiVersionResolvers<ContextType>;
  ApiVersionEdge?: ApiVersionEdgeResolvers<ContextType>;
  ApiVersionSearchResult?: ApiVersionSearchResultResolvers<ContextType>;
  ApiVersionTranslation?: ApiVersionTranslationResolvers<ContextType>;
  ApiYou?: ApiYouResolvers<ContextType>;
  Award?: AwardResolvers<ContextType>;
  Comment?: CommentResolvers<ContextType>;
  CommentSearchResult?: CommentSearchResultResolvers<ContextType>;
  CommentThread?: CommentThreadResolvers<ContextType>;
  CommentTranslation?: CommentTranslationResolvers<ContextType>;
  CommentYou?: CommentYouResolvers<ContextType>;
  CommentedOn?: CommentedOnResolvers<ContextType>;
  CopyResult?: CopyResultResolvers<ContextType>;
  Count?: CountResolvers<ContextType>;
  Date?: GraphQLScalarType;
  DevelopResult?: DevelopResultResolvers<ContextType>;
  Email?: EmailResolvers<ContextType>;
  Handle?: HandleResolvers<ContextType>;
  HistoryResult?: HistoryResultResolvers<ContextType>;
  Issue?: IssueResolvers<ContextType>;
  IssueEdge?: IssueEdgeResolvers<ContextType>;
  IssueSearchResult?: IssueSearchResultResolvers<ContextType>;
  IssueTo?: IssueToResolvers<ContextType>;
  IssueTranslation?: IssueTranslationResolvers<ContextType>;
  IssueYou?: IssueYouResolvers<ContextType>;
  Label?: LabelResolvers<ContextType>;
  LabelEdge?: LabelEdgeResolvers<ContextType>;
  LabelSearchResult?: LabelSearchResultResolvers<ContextType>;
  LabelTranslation?: LabelTranslationResolvers<ContextType>;
  LabelYou?: LabelYouResolvers<ContextType>;
  LearnResult?: LearnResultResolvers<ContextType>;
  Meeting?: MeetingResolvers<ContextType>;
  MeetingEdge?: MeetingEdgeResolvers<ContextType>;
  MeetingInvite?: MeetingInviteResolvers<ContextType>;
  MeetingInviteEdge?: MeetingInviteEdgeResolvers<ContextType>;
  MeetingInviteSearchResult?: MeetingInviteSearchResultResolvers<ContextType>;
  MeetingInviteYou?: MeetingInviteYouResolvers<ContextType>;
  MeetingSearchResult?: MeetingSearchResultResolvers<ContextType>;
  MeetingTranslation?: MeetingTranslationResolvers<ContextType>;
  MeetingYou?: MeetingYouResolvers<ContextType>;
  Member?: MemberResolvers<ContextType>;
  MemberEdge?: MemberEdgeResolvers<ContextType>;
  MemberInvite?: MemberInviteResolvers<ContextType>;
  MemberInviteEdge?: MemberInviteEdgeResolvers<ContextType>;
  MemberInviteSearchResult?: MemberInviteSearchResultResolvers<ContextType>;
  MemberInviteYou?: MemberInviteYouResolvers<ContextType>;
  MemberSearchResult?: MemberSearchResultResolvers<ContextType>;
  Mutation?: MutationResolvers<ContextType>;
  Node?: NodeResolvers<ContextType>;
  NodeEnd?: NodeEndResolvers<ContextType>;
  NodeLink?: NodeLinkResolvers<ContextType>;
  NodeLinkWhen?: NodeLinkWhenResolvers<ContextType>;
  NodeLinkWhenTranslation?: NodeLinkWhenTranslationResolvers<ContextType>;
  NodeLoop?: NodeLoopResolvers<ContextType>;
  NodeLoopWhile?: NodeLoopWhileResolvers<ContextType>;
  NodeLoopWhileTranslation?: NodeLoopWhileTranslationResolvers<ContextType>;
  NodeRoutineList?: NodeRoutineListResolvers<ContextType>;
  NodeRoutineListItem?: NodeRoutineListItemResolvers<ContextType>;
  NodeRoutineListItemTranslation?: NodeRoutineListItemTranslationResolvers<ContextType>;
  NodeTranslation?: NodeTranslationResolvers<ContextType>;
  Note?: NoteResolvers<ContextType>;
  NoteEdge?: NoteEdgeResolvers<ContextType>;
  NoteSearchResult?: NoteSearchResultResolvers<ContextType>;
  NoteVersion?: NoteVersionResolvers<ContextType>;
  NoteVersionEdge?: NoteVersionEdgeResolvers<ContextType>;
  NoteVersionSearchResult?: NoteVersionSearchResultResolvers<ContextType>;
  NoteVersionTranslation?: NoteVersionTranslationResolvers<ContextType>;
  NoteYou?: NoteYouResolvers<ContextType>;
  Notification?: NotificationResolvers<ContextType>;
  NotificationEdge?: NotificationEdgeResolvers<ContextType>;
  NotificationSearchResult?: NotificationSearchResultResolvers<ContextType>;
  NotificationSettings?: NotificationSettingsResolvers<ContextType>;
  NotificationSettingsCategory?: NotificationSettingsCategoryResolvers<ContextType>;
  NotificationSubscription?: NotificationSubscriptionResolvers<ContextType>;
  NotificationSubscriptionEdge?: NotificationSubscriptionEdgeResolvers<ContextType>;
  NotificationSubscriptionSearchResult?: NotificationSubscriptionSearchResultResolvers<ContextType>;
  Organization?: OrganizationResolvers<ContextType>;
  OrganizationEdge?: OrganizationEdgeResolvers<ContextType>;
  OrganizationSearchResult?: OrganizationSearchResultResolvers<ContextType>;
  OrganizationTranslation?: OrganizationTranslationResolvers<ContextType>;
  OrganizationYou?: OrganizationYouResolvers<ContextType>;
  Owner?: OwnerResolvers<ContextType>;
  PageInfo?: PageInfoResolvers<ContextType>;
  Payment?: PaymentResolvers<ContextType>;
  Phone?: PhoneResolvers<ContextType>;
  PopularResult?: PopularResultResolvers<ContextType>;
  Post?: PostResolvers<ContextType>;
  PostEdge?: PostEdgeResolvers<ContextType>;
  PostSearchResult?: PostSearchResultResolvers<ContextType>;
  PostTranslation?: PostTranslationResolvers<ContextType>;
  Premium?: PremiumResolvers<ContextType>;
  Project?: ProjectResolvers<ContextType>;
  ProjectEdge?: ProjectEdgeResolvers<ContextType>;
  ProjectOrOrganization?: ProjectOrOrganizationResolvers<ContextType>;
  ProjectOrOrganizationEdge?: ProjectOrOrganizationEdgeResolvers<ContextType>;
  ProjectOrOrganizationPageInfo?: ProjectOrOrganizationPageInfoResolvers<ContextType>;
  ProjectOrOrganizationSearchResult?: ProjectOrOrganizationSearchResultResolvers<ContextType>;
  ProjectOrRoutine?: ProjectOrRoutineResolvers<ContextType>;
  ProjectOrRoutineEdge?: ProjectOrRoutineEdgeResolvers<ContextType>;
  ProjectOrRoutinePageInfo?: ProjectOrRoutinePageInfoResolvers<ContextType>;
  ProjectOrRoutineSearchResult?: ProjectOrRoutineSearchResultResolvers<ContextType>;
  ProjectSearchResult?: ProjectSearchResultResolvers<ContextType>;
  ProjectVersion?: ProjectVersionResolvers<ContextType>;
  ProjectVersionDirectory?: ProjectVersionDirectoryResolvers<ContextType>;
  ProjectVersionDirectoryTranslation?: ProjectVersionDirectoryTranslationResolvers<ContextType>;
  ProjectVersionEdge?: ProjectVersionEdgeResolvers<ContextType>;
  ProjectVersionSearchResult?: ProjectVersionSearchResultResolvers<ContextType>;
  ProjectVersionTranslation?: ProjectVersionTranslationResolvers<ContextType>;
  ProjectVersionYou?: ProjectVersionYouResolvers<ContextType>;
  ProjectYou?: ProjectYouResolvers<ContextType>;
  PullRequest?: PullRequestResolvers<ContextType>;
  PullRequestEdge?: PullRequestEdgeResolvers<ContextType>;
  PullRequestFrom?: PullRequestFromResolvers<ContextType>;
  PullRequestSearchResult?: PullRequestSearchResultResolvers<ContextType>;
  PullRequestTo?: PullRequestToResolvers<ContextType>;
  PullRequestYou?: PullRequestYouResolvers<ContextType>;
  PushDevice?: PushDeviceResolvers<ContextType>;
  Query?: QueryResolvers<ContextType>;
  Question?: QuestionResolvers<ContextType>;
  QuestionAnswer?: QuestionAnswerResolvers<ContextType>;
  QuestionAnswerEdge?: QuestionAnswerEdgeResolvers<ContextType>;
  QuestionAnswerSearchResult?: QuestionAnswerSearchResultResolvers<ContextType>;
  QuestionAnswerTranslation?: QuestionAnswerTranslationResolvers<ContextType>;
  QuestionEdge?: QuestionEdgeResolvers<ContextType>;
  QuestionFor?: QuestionForResolvers<ContextType>;
  QuestionSearchResult?: QuestionSearchResultResolvers<ContextType>;
  QuestionTranslation?: QuestionTranslationResolvers<ContextType>;
  QuestionYou?: QuestionYouResolvers<ContextType>;
  Quiz?: QuizResolvers<ContextType>;
  QuizAttempt?: QuizAttemptResolvers<ContextType>;
  QuizAttemptEdge?: QuizAttemptEdgeResolvers<ContextType>;
  QuizAttemptSearchResult?: QuizAttemptSearchResultResolvers<ContextType>;
  QuizAttemptYou?: QuizAttemptYouResolvers<ContextType>;
  QuizEdge?: QuizEdgeResolvers<ContextType>;
  QuizQuestion?: QuizQuestionResolvers<ContextType>;
  QuizQuestionEdge?: QuizQuestionEdgeResolvers<ContextType>;
  QuizQuestionResponse?: QuizQuestionResponseResolvers<ContextType>;
  QuizQuestionResponseEdge?: QuizQuestionResponseEdgeResolvers<ContextType>;
  QuizQuestionResponseSearchResult?: QuizQuestionResponseSearchResultResolvers<ContextType>;
  QuizQuestionResponseTranslation?: QuizQuestionResponseTranslationResolvers<ContextType>;
  QuizQuestionResponseYou?: QuizQuestionResponseYouResolvers<ContextType>;
  QuizQuestionSearchResult?: QuizQuestionSearchResultResolvers<ContextType>;
  QuizQuestionTranslation?: QuizQuestionTranslationResolvers<ContextType>;
  QuizQuestionYou?: QuizQuestionYouResolvers<ContextType>;
  QuizSearchResult?: QuizSearchResultResolvers<ContextType>;
  QuizTranslation?: QuizTranslationResolvers<ContextType>;
  QuizYou?: QuizYouResolvers<ContextType>;
  Reminder?: ReminderResolvers<ContextType>;
  ReminderEdge?: ReminderEdgeResolvers<ContextType>;
  ReminderItem?: ReminderItemResolvers<ContextType>;
  ReminderList?: ReminderListResolvers<ContextType>;
  ReminderListEdge?: ReminderListEdgeResolvers<ContextType>;
  ReminderListSearchResult?: ReminderListSearchResultResolvers<ContextType>;
  ReminderSearchResult?: ReminderSearchResultResolvers<ContextType>;
  Report?: ReportResolvers<ContextType>;
  ReportEdge?: ReportEdgeResolvers<ContextType>;
  ReportResponse?: ReportResponseResolvers<ContextType>;
  ReportResponseEdge?: ReportResponseEdgeResolvers<ContextType>;
  ReportResponseSearchResult?: ReportResponseSearchResultResolvers<ContextType>;
  ReportResponseYou?: ReportResponseYouResolvers<ContextType>;
  ReportSearchResult?: ReportSearchResultResolvers<ContextType>;
  ReportYou?: ReportYouResolvers<ContextType>;
  ReputationHistory?: ReputationHistoryResolvers<ContextType>;
  ReputationHistoryEdge?: ReputationHistoryEdgeResolvers<ContextType>;
  ReputationHistorySearchResult?: ReputationHistorySearchResultResolvers<ContextType>;
  ResearchResult?: ResearchResultResolvers<ContextType>;
  Resource?: ResourceResolvers<ContextType>;
  ResourceEdge?: ResourceEdgeResolvers<ContextType>;
  ResourceList?: ResourceListResolvers<ContextType>;
  ResourceListEdge?: ResourceListEdgeResolvers<ContextType>;
  ResourceListSearchResult?: ResourceListSearchResultResolvers<ContextType>;
  ResourceListTranslation?: ResourceListTranslationResolvers<ContextType>;
  ResourceSearchResult?: ResourceSearchResultResolvers<ContextType>;
  ResourceTranslation?: ResourceTranslationResolvers<ContextType>;
  Response?: ResponseResolvers<ContextType>;
  Role?: RoleResolvers<ContextType>;
  RoleEdge?: RoleEdgeResolvers<ContextType>;
  RoleSearchResult?: RoleSearchResultResolvers<ContextType>;
  RoleTranslation?: RoleTranslationResolvers<ContextType>;
  Routine?: RoutineResolvers<ContextType>;
  RoutineEdge?: RoutineEdgeResolvers<ContextType>;
  RoutineSearchResult?: RoutineSearchResultResolvers<ContextType>;
  RoutineVersion?: RoutineVersionResolvers<ContextType>;
  RoutineVersionEdge?: RoutineVersionEdgeResolvers<ContextType>;
  RoutineVersionInput?: RoutineVersionInputResolvers<ContextType>;
  RoutineVersionInputTranslation?: RoutineVersionInputTranslationResolvers<ContextType>;
  RoutineVersionOutput?: RoutineVersionOutputResolvers<ContextType>;
  RoutineVersionOutputTranslation?: RoutineVersionOutputTranslationResolvers<ContextType>;
  RoutineVersionSearchResult?: RoutineVersionSearchResultResolvers<ContextType>;
  RoutineVersionTranslation?: RoutineVersionTranslationResolvers<ContextType>;
  RoutineVersionYou?: RoutineVersionYouResolvers<ContextType>;
  RoutineYou?: RoutineYouResolvers<ContextType>;
  RunProject?: RunProjectResolvers<ContextType>;
  RunProjectEdge?: RunProjectEdgeResolvers<ContextType>;
  RunProjectOrRunRoutine?: RunProjectOrRunRoutineResolvers<ContextType>;
  RunProjectOrRunRoutineEdge?: RunProjectOrRunRoutineEdgeResolvers<ContextType>;
  RunProjectOrRunRoutinePageInfo?: RunProjectOrRunRoutinePageInfoResolvers<ContextType>;
  RunProjectOrRunRoutineSearchResult?: RunProjectOrRunRoutineSearchResultResolvers<ContextType>;
  RunProjectSchedule?: RunProjectScheduleResolvers<ContextType>;
  RunProjectScheduleEdge?: RunProjectScheduleEdgeResolvers<ContextType>;
  RunProjectScheduleSearchResult?: RunProjectScheduleSearchResultResolvers<ContextType>;
  RunProjectScheduleTranslation?: RunProjectScheduleTranslationResolvers<ContextType>;
  RunProjectSearchResult?: RunProjectSearchResultResolvers<ContextType>;
  RunProjectStep?: RunProjectStepResolvers<ContextType>;
  RunProjectYou?: RunProjectYouResolvers<ContextType>;
  RunRoutine?: RunRoutineResolvers<ContextType>;
  RunRoutineEdge?: RunRoutineEdgeResolvers<ContextType>;
  RunRoutineInput?: RunRoutineInputResolvers<ContextType>;
  RunRoutineInputEdge?: RunRoutineInputEdgeResolvers<ContextType>;
  RunRoutineInputSearchResult?: RunRoutineInputSearchResultResolvers<ContextType>;
  RunRoutineSchedule?: RunRoutineScheduleResolvers<ContextType>;
  RunRoutineScheduleEdge?: RunRoutineScheduleEdgeResolvers<ContextType>;
  RunRoutineScheduleSearchResult?: RunRoutineScheduleSearchResultResolvers<ContextType>;
  RunRoutineScheduleTranslation?: RunRoutineScheduleTranslationResolvers<ContextType>;
  RunRoutineSearchResult?: RunRoutineSearchResultResolvers<ContextType>;
  RunRoutineStep?: RunRoutineStepResolvers<ContextType>;
  RunRoutineYou?: RunRoutineYouResolvers<ContextType>;
  Session?: SessionResolvers<ContextType>;
  SessionUser?: SessionUserResolvers<ContextType>;
  SmartContract?: SmartContractResolvers<ContextType>;
  SmartContractEdge?: SmartContractEdgeResolvers<ContextType>;
  SmartContractSearchResult?: SmartContractSearchResultResolvers<ContextType>;
  SmartContractVersion?: SmartContractVersionResolvers<ContextType>;
  SmartContractVersionEdge?: SmartContractVersionEdgeResolvers<ContextType>;
  SmartContractVersionSearchResult?: SmartContractVersionSearchResultResolvers<ContextType>;
  SmartContractVersionTranslation?: SmartContractVersionTranslationResolvers<ContextType>;
  SmartContractYou?: SmartContractYouResolvers<ContextType>;
  Standard?: StandardResolvers<ContextType>;
  StandardEdge?: StandardEdgeResolvers<ContextType>;
  StandardSearchResult?: StandardSearchResultResolvers<ContextType>;
  StandardVersion?: StandardVersionResolvers<ContextType>;
  StandardVersionEdge?: StandardVersionEdgeResolvers<ContextType>;
  StandardVersionSearchResult?: StandardVersionSearchResultResolvers<ContextType>;
  StandardVersionTranslation?: StandardVersionTranslationResolvers<ContextType>;
  StandardYou?: StandardYouResolvers<ContextType>;
  Star?: StarResolvers<ContextType>;
  Bookmark?: BookmarkResolvers<ContextType>;
  BookmarkSearchResult?: BookmarkSearchResultResolvers<ContextType>;
  BookmarkTo?: BookmarkToResolvers<ContextType>;
  StatsApi?: StatsApiResolvers<ContextType>;
  StatsApiEdge?: StatsApiEdgeResolvers<ContextType>;
  StatsApiSearchResult?: StatsApiSearchResultResolvers<ContextType>;
  StatsOrganization?: StatsOrganizationResolvers<ContextType>;
  StatsOrganizationEdge?: StatsOrganizationEdgeResolvers<ContextType>;
  StatsOrganizationSearchResult?: StatsOrganizationSearchResultResolvers<ContextType>;
  StatsProject?: StatsProjectResolvers<ContextType>;
  StatsProjectEdge?: StatsProjectEdgeResolvers<ContextType>;
  StatsProjectSearchResult?: StatsProjectSearchResultResolvers<ContextType>;
  StatsQuiz?: StatsQuizResolvers<ContextType>;
  StatsQuizEdge?: StatsQuizEdgeResolvers<ContextType>;
  StatsQuizSearchResult?: StatsQuizSearchResultResolvers<ContextType>;
  StatsRoutine?: StatsRoutineResolvers<ContextType>;
  StatsRoutineEdge?: StatsRoutineEdgeResolvers<ContextType>;
  StatsRoutineSearchResult?: StatsRoutineSearchResultResolvers<ContextType>;
  StatsSite?: StatsSiteResolvers<ContextType>;
  StatsSiteEdge?: StatsSiteEdgeResolvers<ContextType>;
  StatsSiteSearchResult?: StatsSiteSearchResultResolvers<ContextType>;
  StatsSmartContract?: StatsSmartContractResolvers<ContextType>;
  StatsSmartContractEdge?: StatsSmartContractEdgeResolvers<ContextType>;
  StatsSmartContractSearchResult?: StatsSmartContractSearchResultResolvers<ContextType>;
  StatsStandard?: StatsStandardResolvers<ContextType>;
  StatsStandardEdge?: StatsStandardEdgeResolvers<ContextType>;
  StatsStandardSearchResult?: StatsStandardSearchResultResolvers<ContextType>;
  StatsUser?: StatsUserResolvers<ContextType>;
  StatsUserEdge?: StatsUserEdgeResolvers<ContextType>;
  StatsUserSearchResult?: StatsUserSearchResultResolvers<ContextType>;
  SubscribedObject?: SubscribedObjectResolvers<ContextType>;
  Success?: SuccessResolvers<ContextType>;
  Tag?: TagResolvers<ContextType>;
  TagEdge?: TagEdgeResolvers<ContextType>;
  TagSearchResult?: TagSearchResultResolvers<ContextType>;
  TagTranslation?: TagTranslationResolvers<ContextType>;
  TagYou?: TagYouResolvers<ContextType>;
  Transfer?: TransferResolvers<ContextType>;
  TransferEdge?: TransferEdgeResolvers<ContextType>;
  TransferObject?: TransferObjectResolvers<ContextType>;
  TransferSearchResult?: TransferSearchResultResolvers<ContextType>;
  TransferYou?: TransferYouResolvers<ContextType>;
  Translate?: TranslateResolvers<ContextType>;
  Upload?: GraphQLScalarType;
  User?: UserResolvers<ContextType>;
  UserEdge?: UserEdgeResolvers<ContextType>;
  UserSchedule?: UserScheduleResolvers<ContextType>;
  UserScheduleEdge?: UserScheduleEdgeResolvers<ContextType>;
  UserScheduleFilter?: UserScheduleFilterResolvers<ContextType>;
  UserScheduleSearchResult?: UserScheduleSearchResultResolvers<ContextType>;
  UserSearchResult?: UserSearchResultResolvers<ContextType>;
  UserTranslation?: UserTranslationResolvers<ContextType>;
  UserYou?: UserYouResolvers<ContextType>;
  VersionYou?: VersionYouResolvers<ContextType>;
  View?: ViewResolvers<ContextType>;
  ViewEdge?: ViewEdgeResolvers<ContextType>;
  ViewSearchResult?: ViewSearchResultResolvers<ContextType>;
  ViewTo?: ViewToResolvers<ContextType>;
  Vote?: VoteResolvers<ContextType>;
  VoteEdge?: VoteEdgeResolvers<ContextType>;
  VoteSearchResult?: VoteSearchResultResolvers<ContextType>;
  VoteTo?: VoteToResolvers<ContextType>;
  Wallet?: WalletResolvers<ContextType>;
  WalletComplete?: WalletCompleteResolvers<ContextType>;
};

