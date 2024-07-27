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
  JSONObject: any;
  Upload: any;
};

export enum AccountStatus {
  Deleted = 'Deleted',
  HardLocked = 'HardLocked',
  SoftLocked = 'SoftLocked',
  Unlocked = 'Unlocked'
}

export type ActiveFocusMode = {
  __typename: 'ActiveFocusMode';
  mode: FocusMode;
  stopCondition: FocusModeStopCondition;
  stopTime?: Maybe<Scalars['Date']>;
};

export type Api = {
  __typename: 'Api';
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
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
  isPrivate: Scalars['Boolean'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  ownedByTeamConnect?: InputMaybe<Scalars['ID']>;
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
  teamConnect?: InputMaybe<Scalars['ID']>;
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
  maxBookmarks?: InputMaybe<Scalars['Int']>;
  maxScore?: InputMaybe<Scalars['Int']>;
  maxViews?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  ownedByTeamId?: InputMaybe<Scalars['ID']>;
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
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
  ownedByTeamConnect?: InputMaybe<Scalars['ID']>;
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
  isComplete: Scalars['Boolean'];
  isLatest: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  pullRequest?: Maybe<PullRequest>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceList?: Maybe<ResourceList>;
  root: Api;
  schemaText?: Maybe<Scalars['String']>;
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
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate: Scalars['Boolean'];
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  rootConnect?: InputMaybe<Scalars['ID']>;
  rootCreate?: InputMaybe<ApiCreateInput>;
  schemaText?: InputMaybe<Scalars['String']>;
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
  createdByIdRoot?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isCompleteWithRoot?: InputMaybe<Scalars['Boolean']>;
  isLatest?: InputMaybe<Scalars['Boolean']>;
  maxBookmarksRoot?: InputMaybe<Scalars['Int']>;
  maxScoreRoot?: InputMaybe<Scalars['Int']>;
  maxViewsRoot?: InputMaybe<Scalars['Int']>;
  minBookmarksRoot?: InputMaybe<Scalars['Int']>;
  minScoreRoot?: InputMaybe<Scalars['Int']>;
  minViewsRoot?: InputMaybe<Scalars['Int']>;
  ownedByTeamIdRoot?: InputMaybe<Scalars['ID']>;
  ownedByUserIdRoot?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ApiVersionSortBy>;
  tagsRoot?: InputMaybe<Array<Scalars['String']>>;
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
  language: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
  summary?: InputMaybe<Scalars['String']>;
};

export type ApiVersionUpdateInput = {
  callLink?: InputMaybe<Scalars['String']>;
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  directoryListingsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  documentationLink?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
  rootUpdate?: InputMaybe<ApiUpdateInput>;
  schemaText?: InputMaybe<Scalars['String']>;
  translationsCreate?: InputMaybe<Array<ApiVersionTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ApiVersionTranslationUpdateInput>>;
  versionLabel?: InputMaybe<Scalars['String']>;
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type ApiYou = {
  __typename: 'ApiYou';
  canBookmark: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canReact: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canTransfer: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isViewed: Scalars['Boolean'];
  reaction?: Maybe<Scalars['String']>;
};

export type AutoFillInput = {
  data: Scalars['JSONObject'];
  task: LlmTask;
};

export type AutoFillResult = {
  __typename: 'AutoFillResult';
  data: Scalars['JSONObject'];
};

export type Award = {
  __typename: 'Award';
  category: AwardCategory;
  created_at: Scalars['Date'];
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  progress: Scalars['Int'];
  timeCurrentTierCompleted?: Maybe<Scalars['Date']>;
  title?: Maybe<Scalars['String']>;
  updated_at: Scalars['Date'];
};

export enum AwardCategory {
  AccountAnniversary = 'AccountAnniversary',
  AccountNew = 'AccountNew',
  ApiCreate = 'ApiCreate',
  CommentCreate = 'CommentCreate',
  IssueCreate = 'IssueCreate',
  NoteCreate = 'NoteCreate',
  ObjectBookmark = 'ObjectBookmark',
  ObjectReact = 'ObjectReact',
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

export type AwardEdge = {
  __typename: 'AwardEdge';
  cursor: Scalars['String'];
  node: Award;
};

export type AwardSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  sortBy?: InputMaybe<AwardSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type AwardSearchResult = {
  __typename: 'AwardSearchResult';
  edges: Array<AwardEdge>;
  pageInfo: PageInfo;
};

export enum AwardSortBy {
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  ProgressAsc = 'ProgressAsc',
  ProgressDesc = 'ProgressDesc'
}

export type Bookmark = {
  __typename: 'Bookmark';
  by: User;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  list: BookmarkList;
  to: BookmarkTo;
  updated_at: Scalars['Date'];
};

export type BookmarkCreateInput = {
  bookmarkFor: BookmarkFor;
  forConnect: Scalars['ID'];
  id: Scalars['ID'];
  listConnect?: InputMaybe<Scalars['ID']>;
  listCreate?: InputMaybe<BookmarkListCreateInput>;
};

export type BookmarkEdge = {
  __typename: 'BookmarkEdge';
  cursor: Scalars['String'];
  node: Bookmark;
};

export enum BookmarkFor {
  Api = 'Api',
  Code = 'Code',
  Comment = 'Comment',
  Issue = 'Issue',
  Note = 'Note',
  Post = 'Post',
  Project = 'Project',
  Question = 'Question',
  QuestionAnswer = 'QuestionAnswer',
  Quiz = 'Quiz',
  Routine = 'Routine',
  Standard = 'Standard',
  Tag = 'Tag',
  Team = 'Team',
  User = 'User'
}

export type BookmarkList = {
  __typename: 'BookmarkList';
  bookmarks: Array<Bookmark>;
  bookmarksCount: Scalars['Int'];
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  label: Scalars['String'];
  updated_at: Scalars['Date'];
};

export type BookmarkListCreateInput = {
  bookmarksConnect?: InputMaybe<Array<Scalars['ID']>>;
  bookmarksCreate?: InputMaybe<Array<BookmarkCreateInput>>;
  id: Scalars['ID'];
  label: Scalars['String'];
};

export type BookmarkListEdge = {
  __typename: 'BookmarkListEdge';
  cursor: Scalars['String'];
  node: BookmarkList;
};

export type BookmarkListSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  bookmarksContainsId?: InputMaybe<Scalars['ID']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  labelsIds?: InputMaybe<Array<Scalars['String']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<BookmarkListSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type BookmarkListSearchResult = {
  __typename: 'BookmarkListSearchResult';
  edges: Array<BookmarkListEdge>;
  pageInfo: PageInfo;
};

export enum BookmarkListSortBy {
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  IndexAsc = 'IndexAsc',
  IndexDesc = 'IndexDesc',
  LabelAsc = 'LabelAsc',
  LabelDesc = 'LabelDesc'
}

export type BookmarkListUpdateInput = {
  bookmarksConnect?: InputMaybe<Array<Scalars['ID']>>;
  bookmarksCreate?: InputMaybe<Array<BookmarkCreateInput>>;
  bookmarksDelete?: InputMaybe<Array<Scalars['ID']>>;
  bookmarksUpdate?: InputMaybe<Array<BookmarkUpdateInput>>;
  id: Scalars['ID'];
  label?: InputMaybe<Scalars['String']>;
};

export type BookmarkSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  apiId?: InputMaybe<Scalars['ID']>;
  codeId?: InputMaybe<Scalars['ID']>;
  commentId?: InputMaybe<Scalars['ID']>;
  excludeLinkedToTag?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  issueId?: InputMaybe<Scalars['ID']>;
  limitTo?: InputMaybe<Array<BookmarkFor>>;
  listId?: InputMaybe<Scalars['ID']>;
  listLabel?: InputMaybe<Scalars['String']>;
  noteId?: InputMaybe<Scalars['ID']>;
  postId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  questionAnswerId?: InputMaybe<Scalars['ID']>;
  questionId?: InputMaybe<Scalars['ID']>;
  quizId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<BookmarkSortBy>;
  standardId?: InputMaybe<Scalars['ID']>;
  tagId?: InputMaybe<Scalars['ID']>;
  take?: InputMaybe<Scalars['Int']>;
  teamId?: InputMaybe<Scalars['ID']>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
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

export type BookmarkTo = Api | Code | Comment | Issue | Note | Post | Project | Question | QuestionAnswer | Quiz | Routine | Standard | Tag | Team | User;

export type BookmarkUpdateInput = {
  id: Scalars['ID'];
  listConnect?: InputMaybe<Scalars['ID']>;
  listUpdate?: InputMaybe<BookmarkListUpdateInput>;
};

export type BotCreateInput = {
  bannerImage?: InputMaybe<Scalars['Upload']>;
  botSettings: Scalars['String'];
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isBotDepictingPerson: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  name: Scalars['String'];
  profileImage?: InputMaybe<Scalars['Upload']>;
  translationsCreate?: InputMaybe<Array<UserTranslationCreateInput>>;
};

export type BotUpdateInput = {
  bannerImage?: InputMaybe<Scalars['Upload']>;
  botSettings?: InputMaybe<Scalars['String']>;
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isBotDepictingPerson?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  profileImage?: InputMaybe<Scalars['Upload']>;
  translationsCreate?: InputMaybe<Array<UserTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<UserTranslationUpdateInput>>;
};

export type CancelTaskInput = {
  taskId: Scalars['ID'];
};

export type Chat = {
  __typename: 'Chat';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  invites: Array<ChatInvite>;
  invitesCount: Scalars['Int'];
  labels: Array<Label>;
  labelsCount: Scalars['Int'];
  messages: Array<ChatMessage>;
  openToAnyoneWithInvite: Scalars['Boolean'];
  participants: Array<ChatParticipant>;
  participantsCount: Scalars['Int'];
  restrictedToRoles: Array<Role>;
  team?: Maybe<Team>;
  translations: Array<ChatTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  you: ChatYou;
};

export type ChatCreateInput = {
  id: Scalars['ID'];
  invitesCreate?: InputMaybe<Array<ChatInviteCreateInput>>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  messagesCreate?: InputMaybe<Array<ChatMessageCreateInput>>;
  openToAnyoneWithInvite?: InputMaybe<Scalars['Boolean']>;
  restrictedToRolesConnect?: InputMaybe<Array<Scalars['ID']>>;
  task?: InputMaybe<Scalars['String']>;
  teamConnect?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<ChatTranslationCreateInput>>;
};

export type ChatEdge = {
  __typename: 'ChatEdge';
  cursor: Scalars['String'];
  node: Chat;
};

export type ChatInvite = {
  __typename: 'ChatInvite';
  chat: Chat;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  message?: Maybe<Scalars['String']>;
  status: ChatInviteStatus;
  updated_at: Scalars['Date'];
  user: User;
  you: ChatInviteYou;
};

export type ChatInviteCreateInput = {
  chatConnect: Scalars['ID'];
  id: Scalars['ID'];
  message?: InputMaybe<Scalars['String']>;
  userConnect: Scalars['ID'];
};

export type ChatInviteEdge = {
  __typename: 'ChatInviteEdge';
  cursor: Scalars['String'];
  node: ChatInvite;
};

export type ChatInviteSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  chatId?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ChatInviteSortBy>;
  status?: InputMaybe<ChatInviteStatus>;
  statuses?: InputMaybe<Array<ChatInviteStatus>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ChatInviteSearchResult = {
  __typename: 'ChatInviteSearchResult';
  edges: Array<ChatInviteEdge>;
  pageInfo: PageInfo;
};

export enum ChatInviteSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  UserNameAsc = 'UserNameAsc',
  UserNameDesc = 'UserNameDesc'
}

export enum ChatInviteStatus {
  Accepted = 'Accepted',
  Declined = 'Declined',
  Pending = 'Pending'
}

export type ChatInviteUpdateInput = {
  id: Scalars['ID'];
  message?: InputMaybe<Scalars['String']>;
};

export type ChatInviteYou = {
  __typename: 'ChatInviteYou';
  canDelete: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
};

export type ChatMessage = {
  __typename: 'ChatMessage';
  chat: Chat;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  parent?: Maybe<ChatMessageParent>;
  reactionSummaries: Array<ReactionSummary>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  score: Scalars['Int'];
  sequence: Scalars['Int'];
  translations: Array<ChatMessageTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  user: User;
  versionIndex: Scalars['Int'];
  you: ChatMessageYou;
};

export type ChatMessageCreateInput = {
  chatConnect: Scalars['ID'];
  id: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<ChatMessageTranslationCreateInput>>;
  userConnect: Scalars['ID'];
  versionIndex: Scalars['Int'];
};

export type ChatMessageEdge = {
  __typename: 'ChatMessageEdge';
  cursor: Scalars['String'];
  node: ChatMessage;
};

export type ChatMessageParent = {
  __typename: 'ChatMessageParent';
  created_at: Scalars['Date'];
  id: Scalars['ID'];
};

export type ChatMessageSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  chatId: Scalars['ID'];
  createdTimeFrame?: InputMaybe<TimeFrame>;
  minScore?: InputMaybe<Scalars['Int']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ChatMessageSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type ChatMessageSearchResult = {
  __typename: 'ChatMessageSearchResult';
  edges: Array<ChatMessageEdge>;
  pageInfo: PageInfo;
};

export type ChatMessageSearchTreeInput = {
  chatId: Scalars['ID'];
  excludeDown?: InputMaybe<Scalars['Boolean']>;
  excludeUp?: InputMaybe<Scalars['Boolean']>;
  sortBy?: InputMaybe<ChatMessageSortBy>;
  startId?: InputMaybe<Scalars['ID']>;
  take?: InputMaybe<Scalars['Int']>;
};

export type ChatMessageSearchTreeResult = {
  __typename: 'ChatMessageSearchTreeResult';
  hasMoreDown: Scalars['Boolean'];
  hasMoreUp: Scalars['Boolean'];
  messages: Array<ChatMessage>;
};

export enum ChatMessageSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc'
}

export type ChatMessageTranslation = {
  __typename: 'ChatMessageTranslation';
  id: Scalars['ID'];
  language: Scalars['String'];
  text: Scalars['String'];
};

export type ChatMessageTranslationCreateInput = {
  id: Scalars['ID'];
  language: Scalars['String'];
  text: Scalars['String'];
};

export type ChatMessageTranslationUpdateInput = {
  id: Scalars['ID'];
  language: Scalars['String'];
  text?: InputMaybe<Scalars['String']>;
};

export type ChatMessageUpdateInput = {
  id: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<ChatMessageTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ChatMessageTranslationUpdateInput>>;
};

export type ChatMessageYou = {
  __typename: 'ChatMessageYou';
  canDelete: Scalars['Boolean'];
  canReact: Scalars['Boolean'];
  canReply: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  reaction?: Maybe<Scalars['String']>;
};

export type ChatMessageedOn = ApiVersion | CodeVersion | Issue | NoteVersion | Post | ProjectVersion | PullRequest | Question | QuestionAnswer | RoutineVersion | StandardVersion;

export type ChatParticipant = {
  __typename: 'ChatParticipant';
  chat: Chat;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  updated_at: Scalars['Date'];
  user: User;
};

export type ChatParticipantEdge = {
  __typename: 'ChatParticipantEdge';
  cursor: Scalars['String'];
  node: ChatParticipant;
};

export type ChatParticipantSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  chatId?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ChatParticipantSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ChatParticipantSearchResult = {
  __typename: 'ChatParticipantSearchResult';
  edges: Array<ChatParticipantEdge>;
  pageInfo: PageInfo;
};

export enum ChatParticipantSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  UserNameAsc = 'UserNameAsc',
  UserNameDesc = 'UserNameDesc'
}

export type ChatParticipantUpdateInput = {
  id: Scalars['ID'];
};

export type ChatSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  creatorId?: InputMaybe<Scalars['ID']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  labelsIds?: InputMaybe<Array<Scalars['ID']>>;
  openToAnyoneWithInvite?: InputMaybe<Scalars['Boolean']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ChatSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  teamId?: InputMaybe<Scalars['ID']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ChatSearchResult = {
  __typename: 'ChatSearchResult';
  edges: Array<ChatEdge>;
  pageInfo: PageInfo;
};

export enum ChatSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  InvitesAsc = 'InvitesAsc',
  InvitesDesc = 'InvitesDesc',
  MessagesAsc = 'MessagesAsc',
  MessagesDesc = 'MessagesDesc',
  ParticipantsAsc = 'ParticipantsAsc',
  ParticipantsDesc = 'ParticipantsDesc'
}

export type ChatTranslation = {
  __typename: 'ChatTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name?: Maybe<Scalars['String']>;
};

export type ChatTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
};

export type ChatTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
};

export type ChatUpdateInput = {
  id: Scalars['ID'];
  invitesCreate?: InputMaybe<Array<ChatInviteCreateInput>>;
  invitesDelete?: InputMaybe<Array<Scalars['ID']>>;
  invitesUpdate?: InputMaybe<Array<ChatInviteUpdateInput>>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  labelsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  messagesCreate?: InputMaybe<Array<ChatMessageCreateInput>>;
  messagesDelete?: InputMaybe<Array<Scalars['ID']>>;
  messagesUpdate?: InputMaybe<Array<ChatMessageUpdateInput>>;
  openToAnyoneWithInvite?: InputMaybe<Scalars['Boolean']>;
  participantsDelete?: InputMaybe<Array<Scalars['ID']>>;
  restrictedToRolesConnect?: InputMaybe<Array<Scalars['ID']>>;
  restrictedToRolesDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  translationsCreate?: InputMaybe<Array<ChatTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ChatTranslationUpdateInput>>;
};

export type ChatYou = {
  __typename: 'ChatYou';
  canDelete: Scalars['Boolean'];
  canInvite: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
};

export type CheckTaskStatusesInput = {
  taskIds: Array<Scalars['ID']>;
};

export type CheckTaskStatusesResult = {
  __typename: 'CheckTaskStatusesResult';
  statuses: Array<LlmTaskStatusInfo>;
};

export type Code = {
  __typename: 'Code';
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
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
  parent?: Maybe<CodeVersion>;
  permissions: Scalars['String'];
  pullRequests: Array<PullRequest>;
  pullRequestsCount: Scalars['Int'];
  questions: Array<Question>;
  questionsCount: Scalars['Int'];
  score: Scalars['Int'];
  stats: Array<StatsCode>;
  tags: Array<Tag>;
  transfers: Array<Transfer>;
  transfersCount: Scalars['Int'];
  translatedName: Scalars['String'];
  updated_at: Scalars['Date'];
  versions: Array<CodeVersion>;
  versionsCount?: Maybe<Scalars['Int']>;
  views: Scalars['Int'];
  you: CodeYou;
};

export type CodeCreateInput = {
  id: Scalars['ID'];
  isPrivate: Scalars['Boolean'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  ownedByTeamConnect?: InputMaybe<Scalars['ID']>;
  ownedByUserConnect?: InputMaybe<Scalars['ID']>;
  parentConnect?: InputMaybe<Scalars['ID']>;
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  versionsCreate?: InputMaybe<Array<CodeVersionCreateInput>>;
};

export type CodeEdge = {
  __typename: 'CodeEdge';
  cursor: Scalars['String'];
  node: Code;
};

export type CodeSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  hasCompleteVersion?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  issuesId?: InputMaybe<Scalars['ID']>;
  labelsIds?: InputMaybe<Array<Scalars['ID']>>;
  maxBookmarks?: InputMaybe<Scalars['Int']>;
  maxScore?: InputMaybe<Scalars['Int']>;
  maxViews?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  ownedByTeamId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  pullRequestsId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<CodeSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguagesLatestVersion?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type CodeSearchResult = {
  __typename: 'CodeSearchResult';
  edges: Array<CodeEdge>;
  pageInfo: PageInfo;
};

export enum CodeSortBy {
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
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
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc'
}

export enum CodeType {
  DataConvert = 'DataConvert',
  SmartContract = 'SmartContract'
}

export type CodeUpdateInput = {
  id: Scalars['ID'];
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  labelsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  ownedByTeamConnect?: InputMaybe<Scalars['ID']>;
  ownedByUserConnect?: InputMaybe<Scalars['ID']>;
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  versionsCreate?: InputMaybe<Array<CodeVersionCreateInput>>;
  versionsDelete?: InputMaybe<Array<Scalars['ID']>>;
  versionsUpdate?: InputMaybe<Array<CodeVersionUpdateInput>>;
};

export type CodeVersion = {
  __typename: 'CodeVersion';
  calledByRoutineVersionsCount: Scalars['Int'];
  codeLanguage: Scalars['String'];
  codeType: CodeType;
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  content: Scalars['String'];
  created_at: Scalars['Date'];
  default?: Maybe<Scalars['String']>;
  directoryListings: Array<ProjectVersionDirectory>;
  directoryListingsCount: Scalars['Int'];
  forks: Array<Code>;
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
  root: Code;
  translations: Array<CodeVersionTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  versionIndex: Scalars['Int'];
  versionLabel: Scalars['String'];
  versionNotes?: Maybe<Scalars['String']>;
  you: VersionYou;
};

export type CodeVersionCreateInput = {
  codeLanguage: Scalars['String'];
  codeType: CodeType;
  content: Scalars['String'];
  default?: InputMaybe<Scalars['String']>;
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate: Scalars['Boolean'];
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  rootConnect: Scalars['ID'];
  rootCreate?: InputMaybe<CodeCreateInput>;
  translationsCreate?: InputMaybe<Array<CodeVersionTranslationCreateInput>>;
  versionLabel: Scalars['String'];
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type CodeVersionEdge = {
  __typename: 'CodeVersionEdge';
  cursor: Scalars['String'];
  node: CodeVersion;
};

export type CodeVersionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  calledByRoutineVersionId?: InputMaybe<Scalars['ID']>;
  completedTimeFrame?: InputMaybe<TimeFrame>;
  createdByIdRoot?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isCompleteWithRoot?: InputMaybe<Scalars['Boolean']>;
  isLatest?: InputMaybe<Scalars['Boolean']>;
  maxBookmarksRoot?: InputMaybe<Scalars['Int']>;
  maxScoreRoot?: InputMaybe<Scalars['Int']>;
  maxViewsRoot?: InputMaybe<Scalars['Int']>;
  minBookmarksRoot?: InputMaybe<Scalars['Int']>;
  minScoreRoot?: InputMaybe<Scalars['Int']>;
  minViewsRoot?: InputMaybe<Scalars['Int']>;
  ownedByTeamIdRoot?: InputMaybe<Scalars['ID']>;
  ownedByUserIdRoot?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  rootId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<CodeVersionSortBy>;
  tagsRoot?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type CodeVersionSearchResult = {
  __typename: 'CodeVersionSearchResult';
  edges: Array<CodeVersionEdge>;
  pageInfo: PageInfo;
};

export enum CodeVersionSortBy {
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

export type CodeVersionTranslation = {
  __typename: 'CodeVersionTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  jsonVariable?: Maybe<Scalars['String']>;
  language: Scalars['String'];
  name: Scalars['String'];
};

export type CodeVersionTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  jsonVariable?: InputMaybe<Scalars['String']>;
  language: Scalars['String'];
  name: Scalars['String'];
};

export type CodeVersionTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  jsonVariable?: InputMaybe<Scalars['String']>;
  language: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
};

export type CodeVersionUpdateInput = {
  codeLanguage?: InputMaybe<Scalars['String']>;
  content?: InputMaybe<Scalars['String']>;
  default?: InputMaybe<Scalars['String']>;
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  directoryListingsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
  rootUpdate?: InputMaybe<CodeUpdateInput>;
  translationsCreate?: InputMaybe<Array<CodeVersionTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<CodeVersionTranslationUpdateInput>>;
  versionLabel?: InputMaybe<Scalars['String']>;
  versionNotes?: InputMaybe<Scalars['String']>;
};

export type CodeYou = {
  __typename: 'CodeYou';
  canBookmark: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canReact: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canTransfer: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isViewed: Scalars['Boolean'];
  reaction?: Maybe<Scalars['String']>;
};

export type Comment = {
  __typename: 'Comment';
  bookmarkedBy?: Maybe<Array<User>>;
  bookmarks: Scalars['Int'];
  commentedOn: CommentedOn;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  owner?: Maybe<Owner>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  score: Scalars['Int'];
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
  CodeVersion = 'CodeVersion',
  Issue = 'Issue',
  NoteVersion = 'NoteVersion',
  Post = 'Post',
  ProjectVersion = 'ProjectVersion',
  PullRequest = 'PullRequest',
  Question = 'Question',
  QuestionAnswer = 'QuestionAnswer',
  RoutineVersion = 'RoutineVersion',
  StandardVersion = 'StandardVersion'
}

export type CommentSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  apiVersionId?: InputMaybe<Scalars['ID']>;
  codeVersionId?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  issueId?: InputMaybe<Scalars['ID']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  noteVersionId?: InputMaybe<Scalars['ID']>;
  ownedByTeamId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  postId?: InputMaybe<Scalars['ID']>;
  projectVersionId?: InputMaybe<Scalars['ID']>;
  pullRequestId?: InputMaybe<Scalars['ID']>;
  questionAnswerId?: InputMaybe<Scalars['ID']>;
  questionId?: InputMaybe<Scalars['ID']>;
  routineVersionId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  ScoreAsc = 'ScoreAsc',
  ScoreDesc = 'ScoreDesc'
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
  language: Scalars['String'];
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
  canBookmark: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canReact: Scalars['Boolean'];
  canReply: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  reaction?: Maybe<Scalars['String']>;
};

export type CommentedOn = ApiVersion | CodeVersion | Issue | NoteVersion | Post | ProjectVersion | PullRequest | Question | QuestionAnswer | RoutineVersion | StandardVersion;

export type CopyInput = {
  id: Scalars['ID'];
  intendToPullRequest: Scalars['Boolean'];
  objectType: CopyType;
};

export type CopyResult = {
  __typename: 'CopyResult';
  apiVersion?: Maybe<ApiVersion>;
  codeVersion?: Maybe<CodeVersion>;
  noteVersion?: Maybe<NoteVersion>;
  projectVersion?: Maybe<ProjectVersion>;
  routineVersion?: Maybe<RoutineVersion>;
  standardVersion?: Maybe<StandardVersion>;
  team?: Maybe<Team>;
};

export enum CopyType {
  ApiVersion = 'ApiVersion',
  CodeVersion = 'CodeVersion',
  NoteVersion = 'NoteVersion',
  ProjectVersion = 'ProjectVersion',
  RoutineVersion = 'RoutineVersion',
  StandardVersion = 'StandardVersion',
  Team = 'Team'
}

export type Count = {
  __typename: 'Count';
  count: Scalars['Int'];
};

export type DeleteManyInput = {
  objects: Array<DeleteOneInput>;
};

export type DeleteOneInput = {
  id: Scalars['ID'];
  objectType: DeleteType;
};

export enum DeleteType {
  Api = 'Api',
  ApiKey = 'ApiKey',
  ApiVersion = 'ApiVersion',
  Bookmark = 'Bookmark',
  BookmarkList = 'BookmarkList',
  Chat = 'Chat',
  ChatInvite = 'ChatInvite',
  ChatMessage = 'ChatMessage',
  ChatParticipant = 'ChatParticipant',
  Code = 'Code',
  CodeVersion = 'CodeVersion',
  Comment = 'Comment',
  Email = 'Email',
  FocusMode = 'FocusMode',
  Issue = 'Issue',
  Meeting = 'Meeting',
  MeetingInvite = 'MeetingInvite',
  Member = 'Member',
  MemberInvite = 'MemberInvite',
  Node = 'Node',
  Note = 'Note',
  NoteVersion = 'NoteVersion',
  Notification = 'Notification',
  Phone = 'Phone',
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
  Resource = 'Resource',
  Role = 'Role',
  Routine = 'Routine',
  RoutineVersion = 'RoutineVersion',
  RunProject = 'RunProject',
  RunRoutine = 'RunRoutine',
  Schedule = 'Schedule',
  Standard = 'Standard',
  StandardVersion = 'StandardVersion',
  Team = 'Team',
  Transfer = 'Transfer',
  User = 'User',
  Wallet = 'Wallet'
}

export type Email = {
  __typename: 'Email';
  emailAddress: Scalars['String'];
  id: Scalars['ID'];
  verified: Scalars['Boolean'];
};

export type EmailCreateInput = {
  emailAddress: Scalars['String'];
  id: Scalars['ID'];
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

export type FindVersionInput = {
  handleRoot?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
  idRoot?: InputMaybe<Scalars['ID']>;
};

export type FocusMode = {
  __typename: 'FocusMode';
  created_at: Scalars['Date'];
  description?: Maybe<Scalars['String']>;
  filters: Array<FocusModeFilter>;
  id: Scalars['ID'];
  labels: Array<Label>;
  name: Scalars['String'];
  reminderList?: Maybe<ReminderList>;
  resourceList?: Maybe<ResourceList>;
  schedule?: Maybe<Schedule>;
  updated_at: Scalars['Date'];
  you: FocusModeYou;
};

export type FocusModeCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  filtersCreate?: InputMaybe<Array<FocusModeFilterCreateInput>>;
  id: Scalars['ID'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  name: Scalars['String'];
  reminderListConnect?: InputMaybe<Scalars['ID']>;
  reminderListCreate?: InputMaybe<ReminderListCreateInput>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  scheduleCreate?: InputMaybe<ScheduleCreateInput>;
};

export type FocusModeEdge = {
  __typename: 'FocusModeEdge';
  cursor: Scalars['String'];
  node: FocusMode;
};

export type FocusModeFilter = {
  __typename: 'FocusModeFilter';
  filterType: FocusModeFilterType;
  focusMode: FocusMode;
  id: Scalars['ID'];
  tag: Tag;
};

export type FocusModeFilterCreateInput = {
  filterType: FocusModeFilterType;
  focusModeConnect: Scalars['ID'];
  id: Scalars['ID'];
  tagConnect?: InputMaybe<Scalars['ID']>;
  tagCreate?: InputMaybe<TagCreateInput>;
};

export enum FocusModeFilterType {
  Blur = 'Blur',
  Hide = 'Hide',
  ShowMore = 'ShowMore'
}

export type FocusModeSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  labelsIds?: InputMaybe<Array<Scalars['ID']>>;
  scheduleEndTimeFrame?: InputMaybe<TimeFrame>;
  scheduleStartTimeFrame?: InputMaybe<TimeFrame>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<FocusModeSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  timeZone?: InputMaybe<Scalars['String']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type FocusModeSearchResult = {
  __typename: 'FocusModeSearchResult';
  edges: Array<FocusModeEdge>;
  pageInfo: PageInfo;
};

export enum FocusModeSortBy {
  NameAsc = 'NameAsc',
  NameDesc = 'NameDesc'
}

export enum FocusModeStopCondition {
  AfterStopTime = 'AfterStopTime',
  Automatic = 'Automatic',
  Never = 'Never',
  NextBegins = 'NextBegins'
}

export type FocusModeUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  filtersCreate?: InputMaybe<Array<FocusModeFilterCreateInput>>;
  filtersDelete?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  labelsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  name?: InputMaybe<Scalars['String']>;
  reminderListConnect?: InputMaybe<Scalars['ID']>;
  reminderListCreate?: InputMaybe<ReminderListCreateInput>;
  reminderListDisconnect?: InputMaybe<Scalars['Boolean']>;
  reminderListUpdate?: InputMaybe<ReminderListUpdateInput>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
  scheduleCreate?: InputMaybe<ScheduleCreateInput>;
  scheduleUpdate?: InputMaybe<ScheduleUpdateInput>;
};

export type FocusModeYou = {
  __typename: 'FocusModeYou';
  canDelete: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
};

export enum GqlModelType {
  Api = 'Api',
  ApiKey = 'ApiKey',
  ApiVersion = 'ApiVersion',
  Award = 'Award',
  Bookmark = 'Bookmark',
  BookmarkList = 'BookmarkList',
  Chat = 'Chat',
  ChatInvite = 'ChatInvite',
  ChatMessage = 'ChatMessage',
  ChatMessageSearchTreeResult = 'ChatMessageSearchTreeResult',
  ChatParticipant = 'ChatParticipant',
  Code = 'Code',
  CodeVersion = 'CodeVersion',
  Comment = 'Comment',
  Copy = 'Copy',
  Email = 'Email',
  FocusMode = 'FocusMode',
  FocusModeFilter = 'FocusModeFilter',
  Fork = 'Fork',
  Handle = 'Handle',
  HomeResult = 'HomeResult',
  Issue = 'Issue',
  Label = 'Label',
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
  Payment = 'Payment',
  Phone = 'Phone',
  PopularResult = 'PopularResult',
  Post = 'Post',
  Premium = 'Premium',
  Project = 'Project',
  ProjectOrRoutineSearchResult = 'ProjectOrRoutineSearchResult',
  ProjectOrTeamSearchResult = 'ProjectOrTeamSearchResult',
  ProjectVersion = 'ProjectVersion',
  ProjectVersionContentsSearchResult = 'ProjectVersionContentsSearchResult',
  ProjectVersionDirectory = 'ProjectVersionDirectory',
  PullRequest = 'PullRequest',
  PushDevice = 'PushDevice',
  Question = 'Question',
  QuestionAnswer = 'QuestionAnswer',
  Quiz = 'Quiz',
  QuizAttempt = 'QuizAttempt',
  QuizQuestion = 'QuizQuestion',
  QuizQuestionResponse = 'QuizQuestionResponse',
  Reaction = 'Reaction',
  ReactionSummary = 'ReactionSummary',
  Reminder = 'Reminder',
  ReminderItem = 'ReminderItem',
  ReminderList = 'ReminderList',
  Report = 'Report',
  ReportResponse = 'ReportResponse',
  ReputationHistory = 'ReputationHistory',
  Resource = 'Resource',
  ResourceList = 'ResourceList',
  Role = 'Role',
  Routine = 'Routine',
  RoutineVersion = 'RoutineVersion',
  RoutineVersionInput = 'RoutineVersionInput',
  RoutineVersionOutput = 'RoutineVersionOutput',
  RunProject = 'RunProject',
  RunProjectOrRunRoutineSearchResult = 'RunProjectOrRunRoutineSearchResult',
  RunProjectStep = 'RunProjectStep',
  RunRoutine = 'RunRoutine',
  RunRoutineInput = 'RunRoutineInput',
  RunRoutineOutput = 'RunRoutineOutput',
  RunRoutineStep = 'RunRoutineStep',
  Schedule = 'Schedule',
  ScheduleException = 'ScheduleException',
  ScheduleRecurrence = 'ScheduleRecurrence',
  Session = 'Session',
  SessionUser = 'SessionUser',
  Standard = 'Standard',
  StandardVersion = 'StandardVersion',
  StatsApi = 'StatsApi',
  StatsCode = 'StatsCode',
  StatsProject = 'StatsProject',
  StatsQuiz = 'StatsQuiz',
  StatsRoutine = 'StatsRoutine',
  StatsSite = 'StatsSite',
  StatsStandard = 'StatsStandard',
  StatsTeam = 'StatsTeam',
  StatsUser = 'StatsUser',
  Tag = 'Tag',
  Team = 'Team',
  Transfer = 'Transfer',
  User = 'User',
  View = 'View',
  Wallet = 'Wallet'
}

export type HomeInput = {
  activeFocusModeId?: InputMaybe<Scalars['ID']>;
};

export type HomeResult = {
  __typename: 'HomeResult';
  recommended: Array<Resource>;
  reminders: Array<Reminder>;
  resources: Array<Resource>;
  schedules: Array<Schedule>;
};

export type ImportCalendarInput = {
  file: Scalars['Upload'];
};

export type Issue = {
  __typename: 'Issue';
  bookmarkedBy?: Maybe<Array<Bookmark>>;
  bookmarks: Scalars['Int'];
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
  Code = 'Code',
  Note = 'Note',
  Project = 'Project',
  Routine = 'Routine',
  Standard = 'Standard',
  Team = 'Team'
}

export type IssueSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  apiId?: InputMaybe<Scalars['ID']>;
  closedById?: InputMaybe<Scalars['ID']>;
  codeId?: InputMaybe<Scalars['ID']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  noteId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  referencedVersionId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<IssueSortBy>;
  standardId?: InputMaybe<Scalars['ID']>;
  status?: InputMaybe<IssueStatus>;
  take?: InputMaybe<Scalars['Int']>;
  teamId?: InputMaybe<Scalars['ID']>;
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
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
  ScoreDesc = 'ScoreDesc'
}

export enum IssueStatus {
  ClosedResolved = 'ClosedResolved',
  ClosedUnresolved = 'ClosedUnresolved',
  Open = 'Open',
  Rejected = 'Rejected'
}

export type IssueTo = Api | Code | Note | Project | Routine | Standard | Team;

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
  language: Scalars['String'];
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
  canBookmark: Scalars['Boolean'];
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canReact: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  reaction?: Maybe<Scalars['String']>;
};

export type Label = {
  __typename: 'Label';
  apis?: Maybe<Array<Api>>;
  apisCount: Scalars['Int'];
  codes?: Maybe<Array<Code>>;
  codesCount: Scalars['Int'];
  color?: Maybe<Scalars['String']>;
  created_at: Scalars['Date'];
  focusModes?: Maybe<Array<FocusMode>>;
  focusModesCount: Scalars['Int'];
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
  schedules?: Maybe<Array<Schedule>>;
  schedulesCount: Scalars['Int'];
  standards?: Maybe<Array<Standard>>;
  standardsCount: Scalars['Int'];
  translations: Array<LabelTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  you: LabelYou;
};

export type LabelCreateInput = {
  color?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  label: Scalars['String'];
  teamConnect?: InputMaybe<Scalars['ID']>;
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
  ownedByTeamId?: InputMaybe<Scalars['ID']>;
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
  language: Scalars['String'];
};

export type LabelUpdateInput = {
  apisConnect?: InputMaybe<Array<Scalars['ID']>>;
  apisDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  codesConnect?: InputMaybe<Array<Scalars['ID']>>;
  codesDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  color?: InputMaybe<Scalars['String']>;
  focusModesConnect?: InputMaybe<Array<Scalars['ID']>>;
  focusModesDisconnect?: InputMaybe<Array<Scalars['ID']>>;
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
  schedulesConnect?: InputMaybe<Array<Scalars['ID']>>;
  schedulesDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  standardsConnect?: InputMaybe<Array<Scalars['ID']>>;
  standardsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  translationsCreate?: InputMaybe<Array<LabelTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<LabelTranslationUpdateInput>>;
};

export type LabelYou = {
  __typename: 'LabelYou';
  canDelete: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
};

export enum LlmTask {
  ApiAdd = 'ApiAdd',
  ApiDelete = 'ApiDelete',
  ApiFind = 'ApiFind',
  ApiUpdate = 'ApiUpdate',
  BotAdd = 'BotAdd',
  BotDelete = 'BotDelete',
  BotFind = 'BotFind',
  BotUpdate = 'BotUpdate',
  DataConverterAdd = 'DataConverterAdd',
  DataConverterDelete = 'DataConverterDelete',
  DataConverterFind = 'DataConverterFind',
  DataConverterUpdate = 'DataConverterUpdate',
  MembersAdd = 'MembersAdd',
  MembersDelete = 'MembersDelete',
  MembersFind = 'MembersFind',
  MembersUpdate = 'MembersUpdate',
  NoteAdd = 'NoteAdd',
  NoteDelete = 'NoteDelete',
  NoteFind = 'NoteFind',
  NoteUpdate = 'NoteUpdate',
  ProjectAdd = 'ProjectAdd',
  ProjectDelete = 'ProjectDelete',
  ProjectFind = 'ProjectFind',
  ProjectUpdate = 'ProjectUpdate',
  ReminderAdd = 'ReminderAdd',
  ReminderDelete = 'ReminderDelete',
  ReminderFind = 'ReminderFind',
  ReminderUpdate = 'ReminderUpdate',
  RoleAdd = 'RoleAdd',
  RoleDelete = 'RoleDelete',
  RoleFind = 'RoleFind',
  RoleUpdate = 'RoleUpdate',
  RoutineAdd = 'RoutineAdd',
  RoutineDelete = 'RoutineDelete',
  RoutineFind = 'RoutineFind',
  RoutineUpdate = 'RoutineUpdate',
  RunProjectStart = 'RunProjectStart',
  RunRoutineStart = 'RunRoutineStart',
  ScheduleAdd = 'ScheduleAdd',
  ScheduleDelete = 'ScheduleDelete',
  ScheduleFind = 'ScheduleFind',
  ScheduleUpdate = 'ScheduleUpdate',
  SmartContractAdd = 'SmartContractAdd',
  SmartContractDelete = 'SmartContractDelete',
  SmartContractFind = 'SmartContractFind',
  SmartContractUpdate = 'SmartContractUpdate',
  StandardAdd = 'StandardAdd',
  StandardDelete = 'StandardDelete',
  StandardFind = 'StandardFind',
  StandardUpdate = 'StandardUpdate',
  Start = 'Start',
  TeamAdd = 'TeamAdd',
  TeamDelete = 'TeamDelete',
  TeamFind = 'TeamFind',
  TeamUpdate = 'TeamUpdate'
}

export enum LlmTaskStatus {
  Canceling = 'Canceling',
  Completed = 'Completed',
  Failed = 'Failed',
  Running = 'Running',
  Scheduled = 'Scheduled',
  Suggested = 'Suggested'
}

export type LlmTaskStatusInfo = {
  __typename: 'LlmTaskStatusInfo';
  id: Scalars['ID'];
  status?: Maybe<LlmTaskStatus>;
};

export type LogOutInput = {
  id?: InputMaybe<Scalars['ID']>;
};

export type Meeting = {
  __typename: 'Meeting';
  attendees: Array<User>;
  attendeesCount: Scalars['Int'];
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  invites: Array<MeetingInvite>;
  invitesCount: Scalars['Int'];
  labels: Array<Label>;
  labelsCount: Scalars['Int'];
  openToAnyoneWithInvite: Scalars['Boolean'];
  restrictedToRoles: Array<Role>;
  schedule?: Maybe<Schedule>;
  showOnTeamProfile: Scalars['Boolean'];
  team: Team;
  translations: Array<MeetingTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  you: MeetingYou;
};

export type MeetingCreateInput = {
  id: Scalars['ID'];
  invitesCreate?: InputMaybe<Array<MeetingInviteCreateInput>>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  openToAnyoneWithInvite?: InputMaybe<Scalars['Boolean']>;
  restrictedToRolesConnect?: InputMaybe<Array<Scalars['ID']>>;
  scheduleCreate?: InputMaybe<ScheduleCreateInput>;
  showOnTeamProfile?: InputMaybe<Scalars['Boolean']>;
  teamConnect: Scalars['ID'];
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
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<MeetingInviteSortBy>;
  status?: InputMaybe<MeetingInviteStatus>;
  statuses?: InputMaybe<Array<MeetingInviteStatus>>;
  take?: InputMaybe<Scalars['Int']>;
  teamId?: InputMaybe<Scalars['ID']>;
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
  MeetingEndAsc = 'MeetingEndAsc',
  MeetingEndDesc = 'MeetingEndDesc',
  MeetingStartAsc = 'MeetingStartAsc',
  MeetingStartDesc = 'MeetingStartDesc',
  UserNameAsc = 'UserNameAsc',
  UserNameDesc = 'UserNameDesc'
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
  openToAnyoneWithInvite?: InputMaybe<Scalars['Boolean']>;
  scheduleEndTimeFrame?: InputMaybe<TimeFrame>;
  scheduleStartTimeFrame?: InputMaybe<TimeFrame>;
  searchString?: InputMaybe<Scalars['String']>;
  showOnTeamProfile?: InputMaybe<Scalars['Boolean']>;
  sortBy?: InputMaybe<MeetingSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  teamId?: InputMaybe<Scalars['ID']>;
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
  InvitesAsc = 'InvitesAsc',
  InvitesDesc = 'InvitesDesc'
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
  language: Scalars['String'];
  link?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
};

export type MeetingUpdateInput = {
  id: Scalars['ID'];
  invitesCreate?: InputMaybe<Array<MeetingInviteCreateInput>>;
  invitesDelete?: InputMaybe<Array<Scalars['ID']>>;
  invitesUpdate?: InputMaybe<Array<MeetingInviteUpdateInput>>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  labelsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  openToAnyoneWithInvite?: InputMaybe<Scalars['Boolean']>;
  restrictedToRolesConnect?: InputMaybe<Array<Scalars['ID']>>;
  restrictedToRolesDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  scheduleCreate?: InputMaybe<ScheduleCreateInput>;
  scheduleUpdate?: InputMaybe<ScheduleUpdateInput>;
  showOnTeamProfile?: InputMaybe<Scalars['Boolean']>;
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
  permissions: Scalars['String'];
  roles: Array<Role>;
  team: Team;
  updated_at: Scalars['Date'];
  user: User;
  you: MemberYou;
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
  status: MemberInviteStatus;
  team: Team;
  updated_at: Scalars['Date'];
  user: User;
  willBeAdmin: Scalars['Boolean'];
  willHavePermissions?: Maybe<Scalars['String']>;
  you: MemberInviteYou;
};

export type MemberInviteCreateInput = {
  id: Scalars['ID'];
  message?: InputMaybe<Scalars['String']>;
  teamConnect: Scalars['ID'];
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
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<MemberInviteSortBy>;
  status?: InputMaybe<MemberInviteStatus>;
  statuses?: InputMaybe<Array<MemberInviteStatus>>;
  take?: InputMaybe<Scalars['Int']>;
  teamId?: InputMaybe<Scalars['ID']>;
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
  UserNameAsc = 'UserNameAsc',
  UserNameDesc = 'UserNameDesc'
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
  roles?: InputMaybe<Array<Scalars['String']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<MemberSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  teamId?: InputMaybe<Scalars['ID']>;
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

export type MemberYou = {
  __typename: 'MemberYou';
  canDelete: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
};

export type Mutation = {
  __typename: 'Mutation';
  _empty?: Maybe<Scalars['String']>;
  apiCreate: Api;
  apiKeyCreate: ApiKey;
  apiKeyDeleteOne: Success;
  apiKeyUpdate: ApiKey;
  apiKeyValidate: ApiKey;
  apiUpdate: Api;
  apiVersionCreate: ApiVersion;
  apiVersionUpdate: ApiVersion;
  autoFill: AutoFillResult;
  bookmarkCreate: Bookmark;
  bookmarkListCreate: BookmarkList;
  bookmarkListUpdate: BookmarkList;
  bookmarkUpdate: Bookmark;
  botCreate: User;
  botUpdate: User;
  cancelTask: Success;
  chatCreate: Chat;
  chatInviteAccept: ChatInvite;
  chatInviteCreate: ChatInvite;
  chatInviteDecline: ChatInvite;
  chatInviteUpdate: ChatInvite;
  chatInvitesCreate: Array<ChatInvite>;
  chatInvitesUpdate: Array<ChatInvite>;
  chatMessageCreate: ChatMessage;
  chatMessageUpdate: ChatMessage;
  chatParticipantUpdate: ChatParticipant;
  chatUpdate: Chat;
  checkTaskStatuses: CheckTaskStatusesResult;
  codeCreate: Code;
  codeUpdate: Api;
  codeVersionCreate: CodeVersion;
  codeVersionUpdate: CodeVersion;
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
  exportCalendar: Scalars['String'];
  exportData: Scalars['String'];
  focusModeCreate: FocusMode;
  focusModeUpdate: FocusMode;
  guestLogIn: Session;
  importCalendar: Success;
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
  meetingInvitesCreate: Array<MeetingInvite>;
  meetingInvitesUpdate: Array<MeetingInvite>;
  meetingUpdate: Meeting;
  memberInviteAccept: MemberInvite;
  memberInviteCreate: MemberInvite;
  memberInviteDecline: MemberInvite;
  memberInviteUpdate: MemberInvite;
  memberInvitesCreate: Array<MemberInvite>;
  memberInvitesUpdate: Array<MemberInvite>;
  memberUpdate: Member;
  nodeCreate: Node;
  nodeUpdate: Node;
  noteCreate: Note;
  noteUpdate: Note;
  noteVersionCreate: NoteVersion;
  noteVersionUpdate: NoteVersion;
  notificationMarkAllAsRead: Success;
  notificationMarkAsRead: Success;
  notificationSettingsUpdate: NotificationSettings;
  notificationSubscriptionCreate: NotificationSubscription;
  notificationSubscriptionUpdate: NotificationSubscription;
  phoneCreate: Phone;
  postCreate: Post;
  postUpdate: Post;
  profileEmailUpdate: User;
  profileUpdate: User;
  projectCreate: Project;
  projectUpdate: Project;
  projectVersionCreate: ProjectVersion;
  projectVersionDirectoryCreate: ProjectVersionDirectory;
  projectVersionDirectoryUpdate: ProjectVersionDirectory;
  projectVersionUpdate: ProjectVersion;
  pullRequestAccept: PullRequest;
  pullRequestCreate: PullRequest;
  pullRequestReject: PullRequest;
  pullRequestUpdate: PullRequest;
  pushDeviceCreate: PushDevice;
  pushDeviceTest: Success;
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
  react: Success;
  regenerateResponse: Success;
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
  runProjectCreate: RunProject;
  runProjectDeleteAll: Count;
  runProjectUpdate: RunProject;
  runRoutineCreate: RunRoutine;
  runRoutineDeleteAll: Count;
  runRoutineUpdate: RunRoutine;
  scheduleCreate: Schedule;
  scheduleExceptionCreate: ScheduleException;
  scheduleExceptionUpdate: ScheduleException;
  scheduleRecurrenceCreate: ScheduleRecurrence;
  scheduleRecurrenceUpdate: ScheduleRecurrence;
  scheduleUpdate: Schedule;
  sendVerificationEmail: Success;
  sendVerificationText: Success;
  setActiveFocusMode: ActiveFocusMode;
  standardCreate: Standard;
  standardUpdate: Standard;
  standardVersionCreate: StandardVersion;
  standardVersionUpdate: StandardVersion;
  startTask: Success;
  switchCurrentAccount: Session;
  tagCreate: Tag;
  tagUpdate: Tag;
  teamCreate: Team;
  teamUpdate: Team;
  transferAccept: Transfer;
  transferCancel: Transfer;
  transferDeny: Transfer;
  transferRequestReceive: Transfer;
  transferRequestSend: Transfer;
  transferUpdate: Transfer;
  userDeleteOne: Session;
  validateSession: Session;
  validateVerificationText: Success;
  walletComplete: WalletComplete;
  walletInit: Scalars['String'];
  walletUpdate: Wallet;
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


export type MutationAutoFillArgs = {
  input: AutoFillInput;
};


export type MutationBookmarkCreateArgs = {
  input: BookmarkCreateInput;
};


export type MutationBookmarkListCreateArgs = {
  input: BookmarkListCreateInput;
};


export type MutationBookmarkListUpdateArgs = {
  input: BookmarkListUpdateInput;
};


export type MutationBookmarkUpdateArgs = {
  input: BookmarkUpdateInput;
};


export type MutationBotCreateArgs = {
  input: BotCreateInput;
};


export type MutationBotUpdateArgs = {
  input: BotUpdateInput;
};


export type MutationCancelTaskArgs = {
  input: CancelTaskInput;
};


export type MutationChatCreateArgs = {
  input: ChatCreateInput;
};


export type MutationChatInviteAcceptArgs = {
  input: FindByIdInput;
};


export type MutationChatInviteCreateArgs = {
  input: ChatInviteCreateInput;
};


export type MutationChatInviteDeclineArgs = {
  input: FindByIdInput;
};


export type MutationChatInviteUpdateArgs = {
  input: ChatInviteUpdateInput;
};


export type MutationChatInvitesCreateArgs = {
  input: Array<ChatInviteCreateInput>;
};


export type MutationChatInvitesUpdateArgs = {
  input: Array<ChatInviteUpdateInput>;
};


export type MutationChatMessageCreateArgs = {
  input: ChatMessageCreateInput;
};


export type MutationChatMessageUpdateArgs = {
  input: ChatMessageUpdateInput;
};


export type MutationChatParticipantUpdateArgs = {
  input: ChatParticipantUpdateInput;
};


export type MutationChatUpdateArgs = {
  input: ChatUpdateInput;
};


export type MutationCheckTaskStatusesArgs = {
  input: CheckTaskStatusesInput;
};


export type MutationCodeCreateArgs = {
  input: CodeCreateInput;
};


export type MutationCodeUpdateArgs = {
  input: CodeUpdateInput;
};


export type MutationCodeVersionCreateArgs = {
  input: CodeVersionCreateInput;
};


export type MutationCodeVersionUpdateArgs = {
  input: CodeVersionUpdateInput;
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


export type MutationFocusModeCreateArgs = {
  input: FocusModeCreateInput;
};


export type MutationFocusModeUpdateArgs = {
  input: FocusModeUpdateInput;
};


export type MutationImportCalendarArgs = {
  input: ImportCalendarInput;
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


export type MutationMeetingInvitesCreateArgs = {
  input: Array<MeetingInviteCreateInput>;
};


export type MutationMeetingInvitesUpdateArgs = {
  input: Array<MeetingInviteUpdateInput>;
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


export type MutationMemberInvitesCreateArgs = {
  input: Array<MemberInviteCreateInput>;
};


export type MutationMemberInvitesUpdateArgs = {
  input: Array<MemberInviteUpdateInput>;
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


export type MutationProjectVersionDirectoryCreateArgs = {
  input: ProjectVersionDirectoryCreateInput;
};


export type MutationProjectVersionDirectoryUpdateArgs = {
  input: ProjectVersionDirectoryUpdateInput;
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


export type MutationPushDeviceTestArgs = {
  input: PushDeviceTestInput;
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


export type MutationReactArgs = {
  input: ReactInput;
};


export type MutationRegenerateResponseArgs = {
  input: RegenerateResponseInput;
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


export type MutationRunProjectCreateArgs = {
  input: RunProjectCreateInput;
};


export type MutationRunProjectUpdateArgs = {
  input: RunProjectUpdateInput;
};


export type MutationRunRoutineCreateArgs = {
  input: RunRoutineCreateInput;
};


export type MutationRunRoutineUpdateArgs = {
  input: RunRoutineUpdateInput;
};


export type MutationScheduleCreateArgs = {
  input: ScheduleCreateInput;
};


export type MutationScheduleExceptionCreateArgs = {
  input: ScheduleExceptionCreateInput;
};


export type MutationScheduleExceptionUpdateArgs = {
  input: ScheduleExceptionUpdateInput;
};


export type MutationScheduleRecurrenceCreateArgs = {
  input: ScheduleRecurrenceCreateInput;
};


export type MutationScheduleRecurrenceUpdateArgs = {
  input: ScheduleRecurrenceUpdateInput;
};


export type MutationScheduleUpdateArgs = {
  input: ScheduleUpdateInput;
};


export type MutationSendVerificationEmailArgs = {
  input: SendVerificationEmailInput;
};


export type MutationSendVerificationTextArgs = {
  input: SendVerificationTextInput;
};


export type MutationSetActiveFocusModeArgs = {
  input: SetActiveFocusModeInput;
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


export type MutationStartTaskArgs = {
  input: StartTaskInput;
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


export type MutationTeamCreateArgs = {
  input: TeamCreateInput;
};


export type MutationTeamUpdateArgs = {
  input: TeamUpdateInput;
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


export type MutationValidateSessionArgs = {
  input: ValidateSessionInput;
};


export type MutationValidateVerificationTextArgs = {
  input: ValidateVerificationTextInput;
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
  language: Scalars['String'];
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
  nodeConnect: Scalars['ID'];
  operation?: InputMaybe<Scalars['String']>;
  whilesCreate?: InputMaybe<Array<NodeLoopWhileCreateInput>>;
};

export type NodeLoopUpdateInput = {
  id: Scalars['ID'];
  loops?: InputMaybe<Scalars['Int']>;
  maxLoops?: InputMaybe<Scalars['Int']>;
  nodeConnect?: InputMaybe<Scalars['ID']>;
  operation?: InputMaybe<Scalars['String']>;
  whilesCreate?: InputMaybe<Array<NodeLoopWhileCreateInput>>;
  whilesDelete?: InputMaybe<Array<Scalars['ID']>>;
  whilesUpdate?: InputMaybe<Array<NodeLoopWhileUpdateInput>>;
};

export type NodeLoopWhile = {
  __typename: 'NodeLoopWhile';
  condition: Scalars['String'];
  id: Scalars['ID'];
  translations: Array<NodeLoopWhileTranslation>;
};

export type NodeLoopWhileCreateInput = {
  condition: Scalars['String'];
  id: Scalars['ID'];
  loopConnect: Scalars['ID'];
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
  language: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
};

export type NodeLoopWhileUpdateInput = {
  condition?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
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
  language: Scalars['String'];
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
  language: Scalars['String'];
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
  endCreate?: InputMaybe<NodeEndCreateInput>;
  endUpdate?: InputMaybe<NodeEndUpdateInput>;
  id: Scalars['ID'];
  loopCreate?: InputMaybe<NodeLoopCreateInput>;
  loopDelete?: InputMaybe<Scalars['Boolean']>;
  loopUpdate?: InputMaybe<NodeLoopUpdateInput>;
  nodeType?: InputMaybe<NodeType>;
  routineListCreate?: InputMaybe<NodeRoutineListCreateInput>;
  routineListUpdate?: InputMaybe<NodeRoutineListUpdateInput>;
  routineVersionConnect?: InputMaybe<Scalars['ID']>;
  rowIndex?: InputMaybe<Scalars['Int']>;
  translationsCreate?: InputMaybe<Array<NodeTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<NodeTranslationUpdateInput>>;
};

export type Note = {
  __typename: 'Note';
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
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
  isPrivate: Scalars['Boolean'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  ownedByTeamConnect?: InputMaybe<Scalars['ID']>;
  ownedByUserConnect?: InputMaybe<Scalars['ID']>;
  parentConnect?: InputMaybe<Scalars['ID']>;
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  versionsCreate?: InputMaybe<Array<NoteVersionCreateInput>>;
};

export type NoteEdge = {
  __typename: 'NoteEdge';
  cursor: Scalars['String'];
  node: Note;
};

export type NotePage = {
  __typename: 'NotePage';
  id: Scalars['ID'];
  pageIndex: Scalars['Int'];
  text: Scalars['String'];
};

export type NotePageCreateInput = {
  id: Scalars['ID'];
  pageIndex: Scalars['Int'];
  text: Scalars['String'];
};

export type NotePageUpdateInput = {
  id: Scalars['ID'];
  pageIndex?: InputMaybe<Scalars['Int']>;
  text?: InputMaybe<Scalars['String']>;
};

export type NoteSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  maxBookmarks?: InputMaybe<Scalars['Int']>;
  maxScore?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  ownedByTeamId?: InputMaybe<Scalars['ID']>;
  ownedByUserId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<NoteSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguagesLatestVersion?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type NoteSearchResult = {
  __typename: 'NoteSearchResult';
  edges: Array<NoteEdge>;
  pageInfo: PageInfo;
};

export enum NoteSortBy {
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
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
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc'
}

export type NoteUpdateInput = {
  id: Scalars['ID'];
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  labelsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  ownedByTeamConnect?: InputMaybe<Scalars['ID']>;
  ownedByUserConnect?: InputMaybe<Scalars['ID']>;
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  versionsCreate?: InputMaybe<Array<NoteVersionCreateInput>>;
  versionsDelete?: InputMaybe<Array<Scalars['ID']>>;
  versionsUpdate?: InputMaybe<Array<NoteVersionUpdateInput>>;
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
  isPrivate: Scalars['Boolean'];
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
  createdByIdRoot?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isLatest?: InputMaybe<Scalars['Boolean']>;
  maxBookmarksRoot?: InputMaybe<Scalars['Int']>;
  maxScoreRoot?: InputMaybe<Scalars['Int']>;
  maxViewsRoot?: InputMaybe<Scalars['Int']>;
  minBookmarksRoot?: InputMaybe<Scalars['Int']>;
  minScoreRoot?: InputMaybe<Scalars['Int']>;
  minViewsRoot?: InputMaybe<Scalars['Int']>;
  ownedByTeamIdRoot?: InputMaybe<Scalars['ID']>;
  ownedByUserIdRoot?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<NoteVersionSortBy>;
  tagsRoot?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
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
  pages: Array<NotePage>;
};

export type NoteVersionTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
  pagesCreate?: InputMaybe<Array<NotePageCreateInput>>;
};

export type NoteVersionTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
  pagesCreate?: InputMaybe<Array<NotePageCreateInput>>;
  pagesDelete?: InputMaybe<Array<Scalars['ID']>>;
  pagesUpdate?: InputMaybe<Array<NotePageUpdateInput>>;
};

export type NoteVersionUpdateInput = {
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  directoryListingsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
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
  canBookmark: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canReact: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canTransfer: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isViewed: Scalars['Boolean'];
  reaction?: Maybe<Scalars['String']>;
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
  userId?: InputMaybe<Scalars['ID']>;
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
  includedEmails?: Maybe<Array<Email>>;
  includedPush?: Maybe<Array<PushDevice>>;
  includedSms?: Maybe<Array<Phone>>;
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
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type NotificationSubscription = {
  __typename: 'NotificationSubscription';
  context?: Maybe<Scalars['String']>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  object: SubscribedObject;
  silent: Scalars['Boolean'];
};

export type NotificationSubscriptionCreateInput = {
  context?: InputMaybe<Scalars['String']>;
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
  DateCreatedDesc = 'DateCreatedDesc'
}

export type NotificationSubscriptionUpdateInput = {
  context?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  silent?: InputMaybe<Scalars['Boolean']>;
};

export type Owner = Team | User;

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
  checkoutId: Scalars['String'];
  created_at: Scalars['Date'];
  currency: Scalars['String'];
  description: Scalars['String'];
  id: Scalars['ID'];
  paymentMethod: Scalars['String'];
  paymentType: PaymentType;
  status: PaymentStatus;
  team: Team;
  updated_at: Scalars['Date'];
  user: User;
};

export type PaymentEdge = {
  __typename: 'PaymentEdge';
  cursor: Scalars['String'];
  node: Payment;
};

export type PaymentSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  cardLast4?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  currency?: InputMaybe<Scalars['String']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  maxAmount?: InputMaybe<Scalars['Int']>;
  minAmount?: InputMaybe<Scalars['Int']>;
  sortBy?: InputMaybe<PaymentSortBy>;
  status?: InputMaybe<PaymentStatus>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type PaymentSearchResult = {
  __typename: 'PaymentSearchResult';
  edges: Array<PaymentEdge>;
  pageInfo: PageInfo;
};

export enum PaymentSortBy {
  AmountAsc = 'AmountAsc',
  AmountDesc = 'AmountDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export enum PaymentStatus {
  Failed = 'Failed',
  Paid = 'Paid',
  Pending = 'Pending'
}

export enum PaymentType {
  Credits = 'Credits',
  Donation = 'Donation',
  PremiumMonthly = 'PremiumMonthly',
  PremiumYearly = 'PremiumYearly'
}

export type Phone = {
  __typename: 'Phone';
  id: Scalars['ID'];
  phoneNumber: Scalars['String'];
  verified: Scalars['Boolean'];
};

export type PhoneCreateInput = {
  id: Scalars['ID'];
  phoneNumber: Scalars['String'];
};

export type Popular = Api | Code | Note | Project | Question | Routine | Standard | Team | User;

export type PopularEdge = {
  __typename: 'PopularEdge';
  cursor: Scalars['String'];
  node: Popular;
};

export enum PopularObjectType {
  Api = 'Api',
  Code = 'Code',
  Note = 'Note',
  Project = 'Project',
  Question = 'Question',
  Routine = 'Routine',
  Standard = 'Standard',
  Team = 'Team',
  User = 'User'
}

export type PopularPageInfo = {
  __typename: 'PopularPageInfo';
  endCursorApi?: Maybe<Scalars['String']>;
  endCursorCode?: Maybe<Scalars['String']>;
  endCursorNote?: Maybe<Scalars['String']>;
  endCursorProject?: Maybe<Scalars['String']>;
  endCursorQuestion?: Maybe<Scalars['String']>;
  endCursorRoutine?: Maybe<Scalars['String']>;
  endCursorStandard?: Maybe<Scalars['String']>;
  endCursorTeam?: Maybe<Scalars['String']>;
  endCursorUser?: Maybe<Scalars['String']>;
  hasNextPage: Scalars['Boolean'];
};

export type PopularSearchInput = {
  apiAfter?: InputMaybe<Scalars['String']>;
  codeAfter?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  noteAfter?: InputMaybe<Scalars['String']>;
  objectType?: InputMaybe<PopularObjectType>;
  projectAfter?: InputMaybe<Scalars['String']>;
  questionAfter?: InputMaybe<Scalars['String']>;
  routineAfter?: InputMaybe<Scalars['String']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<PopularSortBy>;
  standardAfter?: InputMaybe<Scalars['String']>;
  take?: InputMaybe<Scalars['Int']>;
  teamAfter?: InputMaybe<Scalars['String']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userAfter?: InputMaybe<Scalars['String']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type PopularSearchResult = {
  __typename: 'PopularSearchResult';
  edges: Array<PopularEdge>;
  pageInfo: PopularPageInfo;
};

export enum PopularSortBy {
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc'
}

export type Post = {
  __typename: 'Post';
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
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
  tags: Array<Tag>;
  translations: Array<PostTranslation>;
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
};

export type PostCreateInput = {
  id: Scalars['ID'];
  isPinned?: InputMaybe<Scalars['Boolean']>;
  isPrivate: Scalars['Boolean'];
  repostedFromConnect?: InputMaybe<Scalars['ID']>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  teamConnect?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<ApiVersionTranslationCreateInput>>;
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
  maxBookmarks?: InputMaybe<Scalars['Int']>;
  maxScore?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  repostedFromIds?: InputMaybe<Array<Scalars['ID']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<PostSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  teamId?: InputMaybe<Scalars['ID']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type PostSearchResult = {
  __typename: 'PostSearchResult';
  edges: Array<PostEdge>;
  pageInfo: PageInfo;
};

export enum PostSortBy {
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
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
  language: Scalars['String'];
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
  translationsCreate?: InputMaybe<Array<ApiVersionTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ApiVersionTranslationUpdateInput>>;
};

export type Premium = {
  __typename: 'Premium';
  credits: Scalars['Int'];
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
  bannerImage?: InputMaybe<Scalars['Upload']>;
  focusModesCreate?: InputMaybe<Array<FocusModeCreateInput>>;
  focusModesDelete?: InputMaybe<Array<Scalars['ID']>>;
  focusModesUpdate?: InputMaybe<Array<FocusModeUpdateInput>>;
  handle?: InputMaybe<Scalars['String']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  isPrivateApis?: InputMaybe<Scalars['Boolean']>;
  isPrivateApisCreated?: InputMaybe<Scalars['Boolean']>;
  isPrivateBookmarks?: InputMaybe<Scalars['Boolean']>;
  isPrivateCodes?: InputMaybe<Scalars['Boolean']>;
  isPrivateCodesCreated?: InputMaybe<Scalars['Boolean']>;
  isPrivateMemberships?: InputMaybe<Scalars['Boolean']>;
  isPrivateProjects?: InputMaybe<Scalars['Boolean']>;
  isPrivateProjectsCreated?: InputMaybe<Scalars['Boolean']>;
  isPrivatePullRequests?: InputMaybe<Scalars['Boolean']>;
  isPrivateQuestionsAnswered?: InputMaybe<Scalars['Boolean']>;
  isPrivateQuestionsAsked?: InputMaybe<Scalars['Boolean']>;
  isPrivateQuizzesCreated?: InputMaybe<Scalars['Boolean']>;
  isPrivateRoles?: InputMaybe<Scalars['Boolean']>;
  isPrivateRoutines?: InputMaybe<Scalars['Boolean']>;
  isPrivateRoutinesCreated?: InputMaybe<Scalars['Boolean']>;
  isPrivateStandards?: InputMaybe<Scalars['Boolean']>;
  isPrivateStandardsCreated?: InputMaybe<Scalars['Boolean']>;
  isPrivateTeamsCreated?: InputMaybe<Scalars['Boolean']>;
  isPrivateVotes?: InputMaybe<Scalars['Boolean']>;
  languages?: InputMaybe<Array<Scalars['String']>>;
  name?: InputMaybe<Scalars['String']>;
  notificationSettings?: InputMaybe<Scalars['String']>;
  profileImage?: InputMaybe<Scalars['Upload']>;
  theme?: InputMaybe<Scalars['String']>;
  translationsCreate?: InputMaybe<Array<UserTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<UserTranslationUpdateInput>>;
};

export type Project = {
  __typename: 'Project';
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
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
  isPrivate: Scalars['Boolean'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  ownedByTeamConnect?: InputMaybe<Scalars['ID']>;
  ownedByUserConnect?: InputMaybe<Scalars['ID']>;
  parentConnect?: InputMaybe<Scalars['ID']>;
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  versionsCreate?: InputMaybe<Array<ProjectVersionCreateInput>>;
};

export type ProjectEdge = {
  __typename: 'ProjectEdge';
  cursor: Scalars['String'];
  node: Project;
};

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
  hasCompleteVersion?: InputMaybe<Scalars['Boolean']>;
  hasCompleteVersionExceptions?: InputMaybe<Array<SearchException>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  maxBookmarks?: InputMaybe<Scalars['Int']>;
  maxScore?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  objectType?: InputMaybe<Scalars['String']>;
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
  teamId?: InputMaybe<Scalars['ID']>;
  translationLanguagesLatestVersion?: InputMaybe<Array<Scalars['String']>>;
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
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
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc'
}

export type ProjectOrTeam = Project | Team;

export type ProjectOrTeamEdge = {
  __typename: 'ProjectOrTeamEdge';
  cursor: Scalars['String'];
  node: ProjectOrTeam;
};

export type ProjectOrTeamPageInfo = {
  __typename: 'ProjectOrTeamPageInfo';
  endCursorProject?: Maybe<Scalars['String']>;
  endCursorTeam?: Maybe<Scalars['String']>;
  hasNextPage: Scalars['Boolean'];
};

export type ProjectOrTeamSearchInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  maxBookmarks?: InputMaybe<Scalars['Int']>;
  maxViews?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  objectType?: InputMaybe<Scalars['String']>;
  projectAfter?: InputMaybe<Scalars['String']>;
  projectIsComplete?: InputMaybe<Scalars['Boolean']>;
  projectIsCompleteExceptions?: InputMaybe<Array<SearchException>>;
  projectMaxScore?: InputMaybe<Scalars['Int']>;
  projectMinScore?: InputMaybe<Scalars['Int']>;
  projectParentId?: InputMaybe<Scalars['ID']>;
  projectTeamId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectOrTeamSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  teamAfter?: InputMaybe<Scalars['String']>;
  teamIsOpenToNewMembers?: InputMaybe<Scalars['Boolean']>;
  teamProjectId?: InputMaybe<Scalars['ID']>;
  teamRoutineId?: InputMaybe<Scalars['ID']>;
  translationLanguagesLatestVersion?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ProjectOrTeamSearchResult = {
  __typename: 'ProjectOrTeamSearchResult';
  edges: Array<ProjectOrTeamEdge>;
  pageInfo: ProjectOrTeamPageInfo;
};

export enum ProjectOrTeamSortBy {
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
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
  maxBookmarks?: InputMaybe<Scalars['Int']>;
  maxScore?: InputMaybe<Scalars['Int']>;
  maxViews?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  ownedByTeamId?: InputMaybe<Scalars['ID']>;
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
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
  ownedByTeamConnect?: InputMaybe<Scalars['ID']>;
  ownedByUserConnect?: InputMaybe<Scalars['ID']>;
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
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

export type ProjectVersionContentsSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  directoryIds?: InputMaybe<Array<Scalars['ID']>>;
  projectVersionId: Scalars['ID'];
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectVersionContentsSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ProjectVersionContentsSearchResult = {
  __typename: 'ProjectVersionContentsSearchResult';
  apiVersion?: Maybe<ApiVersion>;
  codeVersion?: Maybe<CodeVersion>;
  directory?: Maybe<ProjectVersionDirectory>;
  noteVersion?: Maybe<NoteVersion>;
  projectVersion?: Maybe<ProjectVersion>;
  routineVersion?: Maybe<RoutineVersion>;
  standardVersion?: Maybe<StandardVersion>;
  team?: Maybe<Team>;
};

export enum ProjectVersionContentsSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type ProjectVersionCreateInput = {
  directoriesCreate?: InputMaybe<Array<ProjectVersionDirectoryCreateInput>>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate: Scalars['Boolean'];
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
  childCodeVersions: Array<CodeVersion>;
  childNoteVersions: Array<NoteVersion>;
  childOrder?: Maybe<Scalars['String']>;
  childProjectVersions: Array<ProjectVersion>;
  childRoutineVersions: Array<RoutineVersion>;
  childStandardVersions: Array<StandardVersion>;
  childTeams: Array<Team>;
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
  childApiVersionsConnect?: InputMaybe<Array<Scalars['ID']>>;
  childCodeVersionsConnect?: InputMaybe<Array<Scalars['ID']>>;
  childNoteVersionsConnect?: InputMaybe<Array<Scalars['ID']>>;
  childOrder?: InputMaybe<Scalars['String']>;
  childProjectVersionsConnect?: InputMaybe<Array<Scalars['ID']>>;
  childRoutineVersionsConnect?: InputMaybe<Array<Scalars['ID']>>;
  childStandardVersionsConnect?: InputMaybe<Array<Scalars['ID']>>;
  childTeamsConnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  isRoot?: InputMaybe<Scalars['Boolean']>;
  parentDirectoryConnect?: InputMaybe<Scalars['ID']>;
  projectVersionConnect: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<ProjectVersionDirectoryTranslationCreateInput>>;
};

export type ProjectVersionDirectoryEdge = {
  __typename: 'ProjectVersionDirectoryEdge';
  cursor: Scalars['String'];
  node: ProjectVersionDirectory;
};

export type ProjectVersionDirectorySearchInput = {
  after?: InputMaybe<Scalars['String']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isRoot?: InputMaybe<Scalars['Boolean']>;
  parentDirectoryId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectVersionDirectorySortBy>;
  take?: InputMaybe<Scalars['Int']>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ProjectVersionDirectorySearchResult = {
  __typename: 'ProjectVersionDirectorySearchResult';
  edges: Array<ProjectVersionDirectoryEdge>;
  pageInfo: PageInfo;
};

export enum ProjectVersionDirectorySortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

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
  language: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
};

export type ProjectVersionDirectoryUpdateInput = {
  childApiVersionsConnect?: InputMaybe<Array<Scalars['ID']>>;
  childApiVersionsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  childCodeVersionsConnect?: InputMaybe<Array<Scalars['ID']>>;
  childCodeVersionsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  childNoteVersionsConnect?: InputMaybe<Array<Scalars['ID']>>;
  childNoteVersionsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  childOrder?: InputMaybe<Scalars['String']>;
  childProjectVersionsConnect?: InputMaybe<Array<Scalars['ID']>>;
  childProjectVersionsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  childRoutineVersionsConnect?: InputMaybe<Array<Scalars['ID']>>;
  childRoutineVersionsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  childStandardVersionsConnect?: InputMaybe<Array<Scalars['ID']>>;
  childStandardVersionsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  childTeamsConnect?: InputMaybe<Array<Scalars['ID']>>;
  childTeamsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  isRoot?: InputMaybe<Scalars['Boolean']>;
  parentDirectoryConnect?: InputMaybe<Scalars['ID']>;
  parentDirectoryDisconnect?: InputMaybe<Scalars['Boolean']>;
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
  createdByIdRoot?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  directoryListingsId?: InputMaybe<Scalars['ID']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isCompleteWithRoot?: InputMaybe<Scalars['Boolean']>;
  isCompleteWithRootExcludeOwnedByTeamId?: InputMaybe<Scalars['ID']>;
  isCompleteWithRootExcludeOwnedByUserId?: InputMaybe<Scalars['ID']>;
  isLatest?: InputMaybe<Scalars['Boolean']>;
  maxBookmarksRoot?: InputMaybe<Scalars['Int']>;
  maxComplexity?: InputMaybe<Scalars['Int']>;
  maxScoreRoot?: InputMaybe<Scalars['Int']>;
  maxSimplicity?: InputMaybe<Scalars['Int']>;
  maxTimesCompleted?: InputMaybe<Scalars['Int']>;
  maxViewsRoot?: InputMaybe<Scalars['Int']>;
  minBookmarksRoot?: InputMaybe<Scalars['Int']>;
  minComplexity?: InputMaybe<Scalars['Int']>;
  minScoreRoot?: InputMaybe<Scalars['Int']>;
  minSimplicity?: InputMaybe<Scalars['Int']>;
  minTimesCompleted?: InputMaybe<Scalars['Int']>;
  minViewsRoot?: InputMaybe<Scalars['Int']>;
  ownedByTeamIdRoot?: InputMaybe<Scalars['ID']>;
  ownedByUserIdRoot?: InputMaybe<Scalars['ID']>;
  rootId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectVersionSortBy>;
  tagsRoot?: InputMaybe<Array<Scalars['String']>>;
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
  language: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
};

export type ProjectVersionUpdateInput = {
  directoriesCreate?: InputMaybe<Array<ProjectVersionDirectoryCreateInput>>;
  directoriesDelete?: InputMaybe<Array<Scalars['ID']>>;
  directoriesUpdate?: InputMaybe<Array<ProjectVersionDirectoryUpdateInput>>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
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
  canBookmark: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canReact: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canTransfer: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isViewed: Scalars['Boolean'];
  reaction?: Maybe<Scalars['String']>;
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
  translations: Array<CommentTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  you: PullRequestYou;
};

export type PullRequestCreateInput = {
  fromConnect: Scalars['ID'];
  fromObjectType: PullRequestFromObjectType;
  id: Scalars['ID'];
  toConnect: Scalars['ID'];
  toObjectType: PullRequestToObjectType;
  translationsCreate?: InputMaybe<Array<PullRequestTranslationCreateInput>>;
};

export type PullRequestEdge = {
  __typename: 'PullRequestEdge';
  cursor: Scalars['String'];
  node: PullRequest;
};

export type PullRequestFrom = ApiVersion | CodeVersion | NoteVersion | ProjectVersion | RoutineVersion | StandardVersion;

export enum PullRequestFromObjectType {
  ApiVersion = 'ApiVersion',
  CodeVersion = 'CodeVersion',
  NoteVersion = 'NoteVersion',
  ProjectVersion = 'ProjectVersion',
  RoutineVersion = 'RoutineVersion',
  StandardVersion = 'StandardVersion'
}

export type PullRequestSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isMergedOrRejected?: InputMaybe<Scalars['Boolean']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<PullRequestSortBy>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  toId?: InputMaybe<Scalars['ID']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
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
  Canceled = 'Canceled',
  Merged = 'Merged',
  Open = 'Open',
  Rejected = 'Rejected'
}

export type PullRequestTo = Api | Code | Note | Project | Routine | Standard;

export enum PullRequestToObjectType {
  Api = 'Api',
  Code = 'Code',
  Note = 'Note',
  Project = 'Project',
  Routine = 'Routine',
  Standard = 'Standard'
}

export type PullRequestTranslation = {
  __typename: 'PullRequestTranslation';
  id: Scalars['ID'];
  language: Scalars['String'];
  text: Scalars['String'];
};

export type PullRequestTranslationCreateInput = {
  id: Scalars['ID'];
  language: Scalars['String'];
  text: Scalars['String'];
};

export type PullRequestTranslationUpdateInput = {
  id: Scalars['ID'];
  language: Scalars['String'];
  text?: InputMaybe<Scalars['String']>;
};

export type PullRequestUpdateInput = {
  id: Scalars['ID'];
  status?: InputMaybe<PullRequestStatus>;
  translationsCreate?: InputMaybe<Array<PullRequestTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<PullRequestTranslationUpdateInput>>;
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
  deviceId: Scalars['String'];
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

export type PushDeviceTestInput = {
  id: Scalars['ID'];
};

export type PushDeviceUpdateInput = {
  id: Scalars['ID'];
  name?: InputMaybe<Scalars['String']>;
};

export type Query = {
  __typename: 'Query';
  _empty?: Maybe<Scalars['String']>;
  api?: Maybe<Api>;
  apiVersion?: Maybe<ApiVersion>;
  apiVersions: ApiVersionSearchResult;
  apis: ApiSearchResult;
  awards: AwardSearchResult;
  bookmark?: Maybe<Bookmark>;
  bookmarkList?: Maybe<BookmarkList>;
  bookmarkLists: BookmarkListSearchResult;
  bookmarks: BookmarkSearchResult;
  chat?: Maybe<Chat>;
  chatInvite?: Maybe<ChatInvite>;
  chatInvites: ChatInviteSearchResult;
  chatMessage?: Maybe<ChatMessage>;
  chatMessageTree: ChatMessageSearchTreeResult;
  chatMessages: ChatMessageSearchResult;
  chatParticipant?: Maybe<ChatParticipant>;
  chatParticipants: ChatParticipantSearchResult;
  chats: ChatSearchResult;
  code?: Maybe<Code>;
  codeVersion?: Maybe<CodeVersion>;
  codeVersions: CodeVersionSearchResult;
  codes: CodeSearchResult;
  comment?: Maybe<Comment>;
  comments: CommentSearchResult;
  focusMode?: Maybe<FocusMode>;
  focusModes: FocusModeSearchResult;
  home: HomeResult;
  issue?: Maybe<Issue>;
  issues: IssueSearchResult;
  label?: Maybe<Label>;
  labels: LabelSearchResult;
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
  notificationSettings: NotificationSettings;
  notificationSubscription?: Maybe<NotificationSubscription>;
  notificationSubscriptions: NotificationSubscriptionSearchResult;
  notifications: NotificationSearchResult;
  payment?: Maybe<Payment>;
  payments: PaymentSearchResult;
  popular: PopularSearchResult;
  post?: Maybe<Post>;
  posts: PostSearchResult;
  profile: User;
  project?: Maybe<Project>;
  projectOrRoutines: ProjectOrRoutineSearchResult;
  projectOrTeams: ProjectOrTeamSearchResult;
  projectVersion?: Maybe<ProjectVersion>;
  projectVersionContents: ProjectVersionContentsSearchResult;
  projectVersionDirectories: ProjectVersionDirectorySearchResult;
  projectVersionDirectory?: Maybe<ProjectVersionDirectory>;
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
  reactions: ReactionSearchResult;
  reminder?: Maybe<Reminder>;
  reminders: ReminderSearchResult;
  report?: Maybe<Report>;
  reportResponse?: Maybe<ReportResponse>;
  reportResponses: ReportResponseSearchResult;
  reports: ReportSearchResult;
  reputationHistories: ReputationHistorySearchResult;
  reputationHistory?: Maybe<ReputationHistory>;
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
  runProjects: RunProjectSearchResult;
  runRoutine?: Maybe<RunRoutine>;
  runRoutineInputs: RunRoutineInputSearchResult;
  runRoutineOutputs: RunRoutineOutputSearchResult;
  runRoutines: RunRoutineSearchResult;
  schedule?: Maybe<Schedule>;
  scheduleException?: Maybe<ScheduleException>;
  scheduleExceptions: ScheduleExceptionSearchResult;
  scheduleRecurrence?: Maybe<ScheduleRecurrence>;
  scheduleRecurrences: ScheduleRecurrenceSearchResult;
  schedules: ScheduleSearchResult;
  standard?: Maybe<Standard>;
  standardVersion?: Maybe<StandardVersion>;
  standardVersions: StandardVersionSearchResult;
  standards: StandardSearchResult;
  statsApi: StatsApiSearchResult;
  statsCode: StatsCodeSearchResult;
  statsProject: StatsProjectSearchResult;
  statsQuiz: StatsQuizSearchResult;
  statsRoutine: StatsRoutineSearchResult;
  statsSite: StatsSiteSearchResult;
  statsStandard: StatsStandardSearchResult;
  statsTeam: StatsTeamSearchResult;
  statsUser: StatsUserSearchResult;
  tag?: Maybe<Tag>;
  tags: TagSearchResult;
  team?: Maybe<Team>;
  teams: TeamSearchResult;
  transfer?: Maybe<Transfer>;
  transfers: TransferSearchResult;
  translate: Translate;
  user?: Maybe<User>;
  users: UserSearchResult;
  views: ViewSearchResult;
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


export type QueryAwardsArgs = {
  input: AwardSearchInput;
};


export type QueryBookmarkArgs = {
  input: FindByIdInput;
};


export type QueryBookmarkListArgs = {
  input: FindByIdInput;
};


export type QueryBookmarkListsArgs = {
  input: BookmarkListSearchInput;
};


export type QueryBookmarksArgs = {
  input: BookmarkSearchInput;
};


export type QueryChatArgs = {
  input: FindByIdInput;
};


export type QueryChatInviteArgs = {
  input: FindByIdInput;
};


export type QueryChatInvitesArgs = {
  input: ChatInviteSearchInput;
};


export type QueryChatMessageArgs = {
  input: FindByIdInput;
};


export type QueryChatMessageTreeArgs = {
  input: ChatMessageSearchTreeInput;
};


export type QueryChatMessagesArgs = {
  input: ChatMessageSearchInput;
};


export type QueryChatParticipantArgs = {
  input: FindByIdInput;
};


export type QueryChatParticipantsArgs = {
  input: ChatParticipantSearchInput;
};


export type QueryChatsArgs = {
  input: ChatSearchInput;
};


export type QueryCodeArgs = {
  input: FindByIdInput;
};


export type QueryCodeVersionArgs = {
  input: FindVersionInput;
};


export type QueryCodeVersionsArgs = {
  input: CodeVersionSearchInput;
};


export type QueryCodesArgs = {
  input: CodeSearchInput;
};


export type QueryCommentArgs = {
  input: FindByIdInput;
};


export type QueryCommentsArgs = {
  input: CommentSearchInput;
};


export type QueryFocusModeArgs = {
  input: FindByIdInput;
};


export type QueryFocusModesArgs = {
  input: FocusModeSearchInput;
};


export type QueryHomeArgs = {
  input: HomeInput;
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


export type QueryPaymentArgs = {
  input: FindByIdInput;
};


export type QueryPaymentsArgs = {
  input: PaymentSearchInput;
};


export type QueryPopularArgs = {
  input: PopularSearchInput;
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


export type QueryProjectOrRoutinesArgs = {
  input: ProjectOrRoutineSearchInput;
};


export type QueryProjectOrTeamsArgs = {
  input: ProjectOrTeamSearchInput;
};


export type QueryProjectVersionArgs = {
  input: FindVersionInput;
};


export type QueryProjectVersionContentsArgs = {
  input: ProjectVersionContentsSearchInput;
};


export type QueryProjectVersionDirectoriesArgs = {
  input: ProjectVersionDirectorySearchInput;
};


export type QueryProjectVersionDirectoryArgs = {
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


export type QueryReactionsArgs = {
  input: ReactionSearchInput;
};


export type QueryReminderArgs = {
  input: FindByIdInput;
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


export type QueryRunProjectsArgs = {
  input: RunProjectSearchInput;
};


export type QueryRunRoutineArgs = {
  input: FindByIdInput;
};


export type QueryRunRoutineInputsArgs = {
  input: RunRoutineInputSearchInput;
};


export type QueryRunRoutineOutputsArgs = {
  input: RunRoutineOutputSearchInput;
};


export type QueryRunRoutinesArgs = {
  input: RunRoutineSearchInput;
};


export type QueryScheduleArgs = {
  input: FindVersionInput;
};


export type QueryScheduleExceptionArgs = {
  input: FindByIdInput;
};


export type QueryScheduleExceptionsArgs = {
  input: ScheduleExceptionSearchInput;
};


export type QueryScheduleRecurrenceArgs = {
  input: FindByIdInput;
};


export type QueryScheduleRecurrencesArgs = {
  input: ScheduleRecurrenceSearchInput;
};


export type QuerySchedulesArgs = {
  input: ScheduleSearchInput;
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


export type QueryStatsApiArgs = {
  input: StatsApiSearchInput;
};


export type QueryStatsCodeArgs = {
  input: StatsCodeSearchInput;
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


export type QueryStatsStandardArgs = {
  input: StatsStandardSearchInput;
};


export type QueryStatsTeamArgs = {
  input: StatsTeamSearchInput;
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


export type QueryTeamArgs = {
  input: FindByIdOrHandleInput;
};


export type QueryTeamsArgs = {
  input: TeamSearchInput;
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


export type QueryUsersArgs = {
  input: UserSearchInput;
};


export type QueryViewsArgs = {
  input: ViewSearchInput;
};

export type Question = {
  __typename: 'Question';
  answers: Array<QuestionAnswer>;
  answersCount: Scalars['Int'];
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  forObject?: Maybe<QuestionFor>;
  hasAcceptedAnswer: Scalars['Boolean'];
  id: Scalars['ID'];
  isPrivate: Scalars['Boolean'];
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  score: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<QuestionTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  you: QuestionYou;
};

export type QuestionAnswer = {
  __typename: 'QuestionAnswer';
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  createdBy?: Maybe<User>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isAccepted: Scalars['Boolean'];
  question: Question;
  score: Scalars['Int'];
  translations: Array<QuestionAnswerTranslation>;
  updated_at: Scalars['Date'];
};

export type QuestionAnswerCreateInput = {
  id: Scalars['ID'];
  questionConnect: Scalars['ID'];
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
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<QuestionAnswerSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type QuestionAnswerSearchResult = {
  __typename: 'QuestionAnswerSearchResult';
  edges: Array<QuestionAnswerEdge>;
  pageInfo: PageInfo;
};

export enum QuestionAnswerSortBy {
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
  CommentsAsc = 'CommentsAsc',
  CommentsDesc = 'CommentsDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  ScoreAsc = 'ScoreAsc',
  ScoreDesc = 'ScoreDesc'
}

export type QuestionAnswerTranslation = {
  __typename: 'QuestionAnswerTranslation';
  id: Scalars['ID'];
  language: Scalars['String'];
  text: Scalars['String'];
};

export type QuestionAnswerTranslationCreateInput = {
  id: Scalars['ID'];
  language: Scalars['String'];
  text: Scalars['String'];
};

export type QuestionAnswerTranslationUpdateInput = {
  id: Scalars['ID'];
  language: Scalars['String'];
  text?: InputMaybe<Scalars['String']>;
};

export type QuestionAnswerUpdateInput = {
  id: Scalars['ID'];
  translationsCreate?: InputMaybe<Array<QuestionAnswerTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<QuestionAnswerTranslationUpdateInput>>;
};

export type QuestionCreateInput = {
  forObjectConnect?: InputMaybe<Scalars['ID']>;
  forObjectType?: InputMaybe<QuestionForType>;
  id: Scalars['ID'];
  isPrivate: Scalars['Boolean'];
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

export type QuestionFor = Api | Code | Note | Project | Routine | Standard | Team;

export enum QuestionForType {
  Api = 'Api',
  Code = 'Code',
  Note = 'Note',
  Project = 'Project',
  Routine = 'Routine',
  Standard = 'Standard',
  Team = 'Team'
}

export type QuestionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  apiId?: InputMaybe<Scalars['ID']>;
  codeId?: InputMaybe<Scalars['ID']>;
  createdById?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  hasAcceptedAnswer?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  maxBookmarks?: InputMaybe<Scalars['Int']>;
  maxScore?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  noteId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<QuestionSortBy>;
  standardId?: InputMaybe<Scalars['ID']>;
  take?: InputMaybe<Scalars['Int']>;
  teamId?: InputMaybe<Scalars['ID']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
  CommentsAsc = 'CommentsAsc',
  CommentsDesc = 'CommentsDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  ScoreAsc = 'ScoreAsc',
  ScoreDesc = 'ScoreDesc'
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
  language: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
};

export type QuestionUpdateInput = {
  acceptedAnswerConnect?: InputMaybe<Scalars['ID']>;
  acceptedAnswerDisconnect?: InputMaybe<Scalars['Boolean']>;
  id: Scalars['ID'];
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<QuestionTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<QuestionTranslationUpdateInput>>;
};

export type QuestionYou = {
  __typename: 'QuestionYou';
  canBookmark: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canReact: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  reaction?: Maybe<Scalars['String']>;
};

export type Quiz = {
  __typename: 'Quiz';
  attempts: Array<QuizAttempt>;
  attemptsCount: Scalars['Int'];
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
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
  stats: Array<StatsQuiz>;
  translations: Array<QuizTranslation>;
  updated_at: Scalars['Date'];
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
  quizConnect: Scalars['ID'];
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
  maxPointsEarned?: InputMaybe<Scalars['Int']>;
  minPointsEarned?: InputMaybe<Scalars['Int']>;
  quizId?: InputMaybe<Scalars['ID']>;
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
  isPrivate: Scalars['Boolean'];
  maxAttempts?: InputMaybe<Scalars['Int']>;
  pointsToPass?: InputMaybe<Scalars['Int']>;
  projectConnect?: InputMaybe<Scalars['ID']>;
  quizQuestionsCreate?: InputMaybe<Array<QuizQuestionCreateInput>>;
  randomizeQuestionOrder?: InputMaybe<Scalars['Boolean']>;
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
  quizConnect: Scalars['ID'];
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
  updated_at: Scalars['Date'];
  you: QuizQuestionResponseYou;
};

export type QuizQuestionResponseCreateInput = {
  id: Scalars['ID'];
  quizAttemptConnect: Scalars['ID'];
  quizQuestionConnect: Scalars['ID'];
  response: Scalars['String'];
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
  DateUpdatedDesc = 'DateUpdatedDesc',
  QuestionOrderAsc = 'QuestionOrderAsc',
  QuestionOrderDesc = 'QuestionOrderDesc'
}

export type QuizQuestionResponseUpdateInput = {
  id: Scalars['ID'];
  response?: InputMaybe<Scalars['String']>;
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
  quizId?: InputMaybe<Scalars['ID']>;
  responseId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<QuizQuestionSortBy>;
  standardId?: InputMaybe<Scalars['ID']>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
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
  language: Scalars['String'];
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
  maxBookmarks?: InputMaybe<Scalars['Int']>;
  maxScore?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  projectId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<QuizSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  QuestionsAsc = 'QuestionsAsc',
  QuestionsDesc = 'QuestionsDesc',
  ScoreAsc = 'ScoreAsc',
  ScoreDesc = 'ScoreDesc'
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
  language: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
};

export type QuizUpdateInput = {
  id: Scalars['ID'];
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  maxAttempts?: InputMaybe<Scalars['Int']>;
  pointsToPass?: InputMaybe<Scalars['Int']>;
  projectConnect?: InputMaybe<Scalars['ID']>;
  projectDisconnect?: InputMaybe<Scalars['Boolean']>;
  quizQuestionsCreate?: InputMaybe<Array<QuizQuestionCreateInput>>;
  quizQuestionsDelete?: InputMaybe<Array<Scalars['ID']>>;
  quizQuestionsUpdate?: InputMaybe<Array<QuizQuestionUpdateInput>>;
  randomizeQuestionOrder?: InputMaybe<Scalars['Boolean']>;
  revealCorrectAnswers?: InputMaybe<Scalars['Boolean']>;
  routineConnect?: InputMaybe<Scalars['ID']>;
  routineDisconnect?: InputMaybe<Scalars['Boolean']>;
  timeLimit?: InputMaybe<Scalars['Int']>;
  translationsCreate?: InputMaybe<Array<QuizTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<QuizTranslationUpdateInput>>;
};

export type QuizYou = {
  __typename: 'QuizYou';
  canBookmark: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canReact: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  hasCompleted: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  reaction?: Maybe<Scalars['String']>;
};

export type ReactInput = {
  emoji?: InputMaybe<Scalars['String']>;
  forConnect: Scalars['ID'];
  reactionFor: ReactionFor;
};

export type Reaction = {
  __typename: 'Reaction';
  by: User;
  emoji: Scalars['String'];
  id: Scalars['ID'];
  to: ReactionTo;
};

export type ReactionEdge = {
  __typename: 'ReactionEdge';
  cursor: Scalars['String'];
  node: Reaction;
};

export enum ReactionFor {
  Api = 'Api',
  ChatMessage = 'ChatMessage',
  Code = 'Code',
  Comment = 'Comment',
  Issue = 'Issue',
  Note = 'Note',
  Post = 'Post',
  Project = 'Project',
  Question = 'Question',
  QuestionAnswer = 'QuestionAnswer',
  Quiz = 'Quiz',
  Routine = 'Routine',
  Standard = 'Standard'
}

export type ReactionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  excludeLinkedToTag?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ReactionSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type ReactionSearchResult = {
  __typename: 'ReactionSearchResult';
  edges: Array<ReactionEdge>;
  pageInfo: PageInfo;
};

export enum ReactionSortBy {
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type ReactionSummary = {
  __typename: 'ReactionSummary';
  count: Scalars['Int'];
  emoji: Scalars['String'];
};

export type ReactionTo = Api | ChatMessage | Code | Comment | Issue | Note | Post | Project | Question | QuestionAnswer | Quiz | Routine | Standard;

export type ReadAssetsInput = {
  files: Array<Scalars['String']>;
};

export type RegenerateResponseInput = {
  messageId: Scalars['ID'];
};

export type Reminder = {
  __typename: 'Reminder';
  created_at: Scalars['Date'];
  description?: Maybe<Scalars['String']>;
  dueDate?: Maybe<Scalars['Date']>;
  id: Scalars['ID'];
  index: Scalars['Int'];
  isComplete: Scalars['Boolean'];
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
  reminderListConnect?: InputMaybe<Scalars['ID']>;
  reminderListCreate?: InputMaybe<ReminderListCreateInput>;
};

export type ReminderEdge = {
  __typename: 'ReminderEdge';
  cursor: Scalars['String'];
  node: Reminder;
};

export type ReminderItem = {
  __typename: 'ReminderItem';
  created_at: Scalars['Date'];
  description?: Maybe<Scalars['String']>;
  dueDate?: Maybe<Scalars['Date']>;
  id: Scalars['ID'];
  index: Scalars['Int'];
  isComplete: Scalars['Boolean'];
  name: Scalars['String'];
  reminder: Reminder;
  updated_at: Scalars['Date'];
};

export type ReminderItemCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  dueDate?: InputMaybe<Scalars['Date']>;
  id: Scalars['ID'];
  index: Scalars['Int'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  name: Scalars['String'];
  reminderConnect: Scalars['ID'];
};

export type ReminderItemUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  dueDate?: InputMaybe<Scalars['Date']>;
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
};

export type ReminderList = {
  __typename: 'ReminderList';
  created_at: Scalars['Date'];
  focusMode?: Maybe<FocusMode>;
  id: Scalars['ID'];
  reminders: Array<Reminder>;
  updated_at: Scalars['Date'];
};

export type ReminderListCreateInput = {
  focusModeConnect?: InputMaybe<Scalars['ID']>;
  id: Scalars['ID'];
  remindersCreate?: InputMaybe<Array<ReminderCreateInput>>;
};

export type ReminderListUpdateInput = {
  focusModeConnect?: InputMaybe<Scalars['ID']>;
  id: Scalars['ID'];
  remindersCreate?: InputMaybe<Array<ReminderCreateInput>>;
  remindersDelete?: InputMaybe<Array<Scalars['ID']>>;
  remindersUpdate?: InputMaybe<Array<ReminderUpdateInput>>;
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
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
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
  isComplete?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  reminderItemsCreate?: InputMaybe<Array<ReminderItemCreateInput>>;
  reminderItemsDelete?: InputMaybe<Array<Scalars['ID']>>;
  reminderItemsUpdate?: InputMaybe<Array<ReminderItemUpdateInput>>;
  reminderListConnect?: InputMaybe<Scalars['ID']>;
  reminderListCreate?: InputMaybe<ReminderListCreateInput>;
};

export type Report = {
  __typename: 'Report';
  created_at: Scalars['Date'];
  details?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  reason: Scalars['String'];
  responses: Array<ReportResponse>;
  responsesCount: Scalars['Int'];
  status: ReportStatus;
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
  ChatMessage = 'ChatMessage',
  CodeVersion = 'CodeVersion',
  Comment = 'Comment',
  Issue = 'Issue',
  NoteVersion = 'NoteVersion',
  Post = 'Post',
  ProjectVersion = 'ProjectVersion',
  RoutineVersion = 'RoutineVersion',
  StandardVersion = 'StandardVersion',
  Tag = 'Tag',
  Team = 'Team',
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
  chatMessageId?: InputMaybe<Scalars['ID']>;
  codeVersionId?: InputMaybe<Scalars['ID']>;
  commentId?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  fromId?: InputMaybe<Scalars['ID']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  issueId?: InputMaybe<Scalars['ID']>;
  languageIn?: InputMaybe<Array<Scalars['String']>>;
  noteVersionId?: InputMaybe<Scalars['ID']>;
  postId?: InputMaybe<Scalars['ID']>;
  projectVersionId?: InputMaybe<Scalars['ID']>;
  routineVersionId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ReportSortBy>;
  standardVersionId?: InputMaybe<Scalars['ID']>;
  tagId?: InputMaybe<Scalars['ID']>;
  take?: InputMaybe<Scalars['Int']>;
  teamId?: InputMaybe<Scalars['ID']>;
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

export enum ReportStatus {
  ClosedDeleted = 'ClosedDeleted',
  ClosedFalseReport = 'ClosedFalseReport',
  ClosedHidden = 'ClosedHidden',
  ClosedNonIssue = 'ClosedNonIssue',
  ClosedSuspended = 'ClosedSuspended',
  Open = 'Open'
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
  listConnect?: InputMaybe<Scalars['ID']>;
  listCreate?: InputMaybe<ResourceListCreateInput>;
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
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  listFor: ResourceListOn;
  resources: Array<Resource>;
  translations: Array<ResourceListTranslation>;
  updated_at: Scalars['Date'];
};

export type ResourceListCreateInput = {
  id: Scalars['ID'];
  listForConnect: Scalars['ID'];
  listForType: ResourceListFor;
  resourcesCreate?: InputMaybe<Array<ResourceCreateInput>>;
  translationsCreate?: InputMaybe<Array<ResourceListTranslationCreateInput>>;
};

export type ResourceListEdge = {
  __typename: 'ResourceListEdge';
  cursor: Scalars['String'];
  node: ResourceList;
};

export enum ResourceListFor {
  ApiVersion = 'ApiVersion',
  CodeVersion = 'CodeVersion',
  FocusMode = 'FocusMode',
  Post = 'Post',
  ProjectVersion = 'ProjectVersion',
  RoutineVersion = 'RoutineVersion',
  StandardVersion = 'StandardVersion',
  Team = 'Team'
}

export type ResourceListOn = ApiVersion | CodeVersion | FocusMode | Post | ProjectVersion | RoutineVersion | StandardVersion | Team;

export type ResourceListSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  apiVersionId?: InputMaybe<Scalars['ID']>;
  codeVersionId?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  focusModeId?: InputMaybe<Scalars['ID']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  postId?: InputMaybe<Scalars['ID']>;
  projectVersionId?: InputMaybe<Scalars['ID']>;
  routineVersionId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ResourceListSortBy>;
  standardVersionId?: InputMaybe<Scalars['ID']>;
  take?: InputMaybe<Scalars['Int']>;
  teamId?: InputMaybe<Scalars['ID']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
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
  language: Scalars['String'];
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
  language: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
};

export type ResourceUpdateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  link?: InputMaybe<Scalars['String']>;
  listConnect?: InputMaybe<Scalars['ID']>;
  listCreate?: InputMaybe<ResourceListCreateInput>;
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
  permissions: Scalars['String'];
  team: Team;
  translations: Array<RoleTranslation>;
  updated_at: Scalars['Date'];
};

export type RoleCreateInput = {
  id: Scalars['ID'];
  membersConnect?: InputMaybe<Array<Scalars['ID']>>;
  name: Scalars['String'];
  permissions: Scalars['String'];
  teamConnect: Scalars['ID'];
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
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RoleSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  teamId: Scalars['ID'];
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
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
  language: Scalars['String'];
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
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
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
  isPrivate: Scalars['Boolean'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  ownedByTeamConnect?: InputMaybe<Scalars['ID']>;
  ownedByUserConnect?: InputMaybe<Scalars['ID']>;
  parentConnect?: InputMaybe<Scalars['ID']>;
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
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
  latestVersionRoutineType?: InputMaybe<RoutineType>;
  maxBookmarks?: InputMaybe<Scalars['Int']>;
  maxScore?: InputMaybe<Scalars['Int']>;
  maxViews?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  ownedByTeamId?: InputMaybe<Scalars['ID']>;
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
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
  VersionsAsc = 'VersionsAsc',
  VersionsDesc = 'VersionsDesc',
  ViewsAsc = 'ViewsAsc',
  ViewsDesc = 'ViewsDesc'
}

export enum RoutineType {
  Action = 'Action',
  Api = 'Api',
  Code = 'Code',
  Data = 'Data',
  Generate = 'Generate',
  Informational = 'Informational',
  MultiStep = 'MultiStep',
  SmartContract = 'SmartContract'
}

export type RoutineUpdateInput = {
  id: Scalars['ID'];
  isInternal?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  labelsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  ownedByTeamConnect?: InputMaybe<Scalars['ID']>;
  ownedByUserConnect?: InputMaybe<Scalars['ID']>;
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  versionsCreate?: InputMaybe<Array<RoutineVersionCreateInput>>;
  versionsDelete?: InputMaybe<Array<Scalars['ID']>>;
  versionsUpdate?: InputMaybe<Array<RoutineVersionUpdateInput>>;
};

export type RoutineVersion = {
  __typename: 'RoutineVersion';
  apiVersion?: Maybe<ApiVersion>;
  codeVersion?: Maybe<CodeVersion>;
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  completedAt?: Maybe<Scalars['Date']>;
  complexity: Scalars['Int'];
  configCallData?: Maybe<Scalars['String']>;
  configFormInput?: Maybe<Scalars['String']>;
  configFormOutput?: Maybe<Scalars['String']>;
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
  routineType: RoutineType;
  simplicity: Scalars['Int'];
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
  apiVersionConnect?: InputMaybe<Scalars['ID']>;
  codeVersionConnect?: InputMaybe<Scalars['ID']>;
  configCallData?: InputMaybe<Scalars['String']>;
  configFormInput?: InputMaybe<Scalars['String']>;
  configFormOutput?: InputMaybe<Scalars['String']>;
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  inputsCreate?: InputMaybe<Array<RoutineVersionInputCreateInput>>;
  isAutomatable?: InputMaybe<Scalars['Boolean']>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isPrivate: Scalars['Boolean'];
  nodeLinksCreate?: InputMaybe<Array<NodeLinkCreateInput>>;
  nodesCreate?: InputMaybe<Array<NodeCreateInput>>;
  outputsCreate?: InputMaybe<Array<RoutineVersionOutputCreateInput>>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  rootConnect?: InputMaybe<Scalars['ID']>;
  rootCreate?: InputMaybe<RoutineCreateInput>;
  routineType: RoutineType;
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
  language: Scalars['String'];
};

export type RoutineVersionInputUpdateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  isRequired?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  standardVersionConnect?: InputMaybe<Scalars['ID']>;
  standardVersionCreate?: InputMaybe<StandardVersionCreateInput>;
  standardVersionDisconnect?: InputMaybe<Scalars['Boolean']>;
  translationsCreate?: InputMaybe<Array<RoutineVersionInputTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<RoutineVersionInputTranslationUpdateInput>>;
};

export type RoutineVersionOutput = {
  __typename: 'RoutineVersionOutput';
  id: Scalars['ID'];
  index?: Maybe<Scalars['Int']>;
  name?: Maybe<Scalars['String']>;
  routineVersion: RoutineVersion;
  standardVersion?: Maybe<StandardVersion>;
  translations: Array<RoutineVersionOutputTranslation>;
};

export type RoutineVersionOutputCreateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  name?: InputMaybe<Scalars['String']>;
  routineVersionConnect: Scalars['ID'];
  standardVersionConnect?: InputMaybe<Scalars['ID']>;
  standardVersionCreate?: InputMaybe<StandardVersionCreateInput>;
  translationsCreate?: InputMaybe<Array<RoutineVersionOutputTranslationCreateInput>>;
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
  language: Scalars['String'];
};

export type RoutineVersionOutputUpdateInput = {
  id: Scalars['ID'];
  index?: InputMaybe<Scalars['Int']>;
  name?: InputMaybe<Scalars['String']>;
  standardVersionConnect?: InputMaybe<Scalars['ID']>;
  standardVersionCreate?: InputMaybe<StandardVersionCreateInput>;
  standardVersionDisconnect?: InputMaybe<Scalars['Boolean']>;
  translationsCreate?: InputMaybe<Array<RoutineVersionOutputTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<RoutineVersionOutputTranslationUpdateInput>>;
};

export type RoutineVersionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  codeVersionId?: InputMaybe<Scalars['ID']>;
  createdByIdRoot?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  directoryListingsId?: InputMaybe<Scalars['ID']>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isCompleteWithRoot?: InputMaybe<Scalars['Boolean']>;
  isCompleteWithRootExcludeOwnedByTeamId?: InputMaybe<Scalars['ID']>;
  isCompleteWithRootExcludeOwnedByUserId?: InputMaybe<Scalars['ID']>;
  isExternalWithRootExcludeOwnedByTeamId?: InputMaybe<Scalars['ID']>;
  isExternalWithRootExcludeOwnedByUserId?: InputMaybe<Scalars['ID']>;
  isInternalWithRoot?: InputMaybe<Scalars['Boolean']>;
  isInternalWithRootExcludeOwnedByTeamId?: InputMaybe<Scalars['ID']>;
  isInternalWithRootExcludeOwnedByUserId?: InputMaybe<Scalars['ID']>;
  isLatest?: InputMaybe<Scalars['Boolean']>;
  maxBookmarksRoot?: InputMaybe<Scalars['Int']>;
  maxComplexity?: InputMaybe<Scalars['Int']>;
  maxScoreRoot?: InputMaybe<Scalars['Int']>;
  maxSimplicity?: InputMaybe<Scalars['Int']>;
  maxTimesCompleted?: InputMaybe<Scalars['Int']>;
  maxViewsRoot?: InputMaybe<Scalars['Int']>;
  minBookmarksRoot?: InputMaybe<Scalars['Int']>;
  minComplexity?: InputMaybe<Scalars['Int']>;
  minScoreRoot?: InputMaybe<Scalars['Int']>;
  minSimplicity?: InputMaybe<Scalars['Int']>;
  minTimesCompleted?: InputMaybe<Scalars['Int']>;
  minViewsRoot?: InputMaybe<Scalars['Int']>;
  ownedByTeamIdRoot?: InputMaybe<Scalars['ID']>;
  ownedByUserIdRoot?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  rootId?: InputMaybe<Scalars['ID']>;
  routineType?: InputMaybe<RoutineType>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RoutineVersionSortBy>;
  tagsRoot?: InputMaybe<Array<Scalars['String']>>;
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
  instructions?: Maybe<Scalars['String']>;
  language: Scalars['String'];
  name: Scalars['String'];
};

export type RoutineVersionTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  instructions?: InputMaybe<Scalars['String']>;
  language: Scalars['String'];
  name: Scalars['String'];
};

export type RoutineVersionTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  instructions?: InputMaybe<Scalars['String']>;
  language: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
};

export type RoutineVersionUpdateInput = {
  apiVersionConnect?: InputMaybe<Scalars['ID']>;
  apiVersionDisconnect?: InputMaybe<Scalars['Boolean']>;
  codeVersionConnect?: InputMaybe<Scalars['ID']>;
  codeVersionDisconnect?: InputMaybe<Scalars['Boolean']>;
  configCallData?: InputMaybe<Scalars['String']>;
  configFormInput?: InputMaybe<Scalars['String']>;
  configFormOutput?: InputMaybe<Scalars['String']>;
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  directoryListingsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  inputsCreate?: InputMaybe<Array<RoutineVersionInputCreateInput>>;
  inputsDelete?: InputMaybe<Array<Scalars['ID']>>;
  inputsUpdate?: InputMaybe<Array<RoutineVersionInputUpdateInput>>;
  isAutomatable?: InputMaybe<Scalars['Boolean']>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
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
  canBookmark: Scalars['Boolean'];
  canComment: Scalars['Boolean'];
  canCopy: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canReact: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canRun: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  runs: Array<RunRoutine>;
};

export type RoutineYou = {
  __typename: 'RoutineYou';
  canBookmark: Scalars['Boolean'];
  canComment: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canReact: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canTransfer: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isViewed: Scalars['Boolean'];
  reaction?: Maybe<Scalars['String']>;
};

export type RunProject = {
  __typename: 'RunProject';
  completedAt?: Maybe<Scalars['Date']>;
  completedComplexity: Scalars['Int'];
  contextSwitches: Scalars['Int'];
  id: Scalars['ID'];
  isPrivate: Scalars['Boolean'];
  lastStep?: Maybe<Array<Scalars['Int']>>;
  name: Scalars['String'];
  projectVersion?: Maybe<ProjectVersion>;
  schedule?: Maybe<Schedule>;
  startedAt?: Maybe<Scalars['Date']>;
  status: RunStatus;
  steps: Array<RunProjectStep>;
  stepsCount: Scalars['Int'];
  team?: Maybe<Team>;
  timeElapsed?: Maybe<Scalars['Int']>;
  user?: Maybe<User>;
  you: RunProjectYou;
};

export type RunProjectCreateInput = {
  completedComplexity?: InputMaybe<Scalars['Int']>;
  contextSwitches?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  isPrivate: Scalars['Boolean'];
  name: Scalars['String'];
  projectVersionConnect: Scalars['ID'];
  scheduleCreate?: InputMaybe<ScheduleCreateInput>;
  status: RunStatus;
  stepsCreate?: InputMaybe<Array<RunProjectStepCreateInput>>;
  teamConnect?: InputMaybe<Scalars['ID']>;
  timeElapsed?: InputMaybe<Scalars['Int']>;
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
  endCursorRunProject?: Maybe<Scalars['String']>;
  endCursorRunRoutine?: Maybe<Scalars['String']>;
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
  statuses?: InputMaybe<Array<RunStatus>>;
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

export type RunProjectSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  completedTimeFrame?: InputMaybe<TimeFrame>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  projectVersionId?: InputMaybe<Scalars['ID']>;
  scheduleEndTimeFrame?: InputMaybe<TimeFrame>;
  scheduleStartTimeFrame?: InputMaybe<TimeFrame>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RunProjectSortBy>;
  startedTimeFrame?: InputMaybe<TimeFrame>;
  status?: InputMaybe<RunStatus>;
  statuses?: InputMaybe<Array<RunStatus>>;
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
  runProject: RunProject;
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
  runProjectConnect: Scalars['ID'];
  status?: InputMaybe<RunProjectStepStatus>;
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
  scheduleCreate?: InputMaybe<ScheduleCreateInput>;
  scheduleUpdate?: InputMaybe<ScheduleUpdateInput>;
  status?: InputMaybe<RunStatus>;
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
  lastStep?: Maybe<Array<Scalars['Int']>>;
  name: Scalars['String'];
  outputs: Array<RunRoutineOutput>;
  outputsCount: Scalars['Int'];
  routineVersion?: Maybe<RoutineVersion>;
  runProject?: Maybe<RunProject>;
  schedule?: Maybe<Schedule>;
  startedAt?: Maybe<Scalars['Date']>;
  status: RunStatus;
  steps: Array<RunRoutineStep>;
  stepsCount: Scalars['Int'];
  team?: Maybe<Team>;
  timeElapsed?: Maybe<Scalars['Int']>;
  user?: Maybe<User>;
  wasRunAutomatically: Scalars['Boolean'];
  you: RunRoutineYou;
};

export type RunRoutineCreateInput = {
  completedComplexity?: InputMaybe<Scalars['Int']>;
  contextSwitches?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  inputsCreate?: InputMaybe<Array<RunRoutineInputCreateInput>>;
  isPrivate: Scalars['Boolean'];
  name: Scalars['String'];
  outputsCreate?: InputMaybe<Array<RunRoutineOutputCreateInput>>;
  routineVersionConnect: Scalars['ID'];
  runProjectConnect?: InputMaybe<Scalars['ID']>;
  scheduleCreate?: InputMaybe<ScheduleCreateInput>;
  status: RunStatus;
  stepsCreate?: InputMaybe<Array<RunRoutineStepCreateInput>>;
  teamConnect?: InputMaybe<Scalars['ID']>;
  timeElapsed?: InputMaybe<Scalars['Int']>;
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
  runRoutineIds?: InputMaybe<Array<Scalars['ID']>>;
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

export type RunRoutineOutput = {
  __typename: 'RunRoutineOutput';
  data: Scalars['String'];
  id: Scalars['ID'];
  output: RoutineVersionOutput;
  runRoutine: RunRoutine;
};

export type RunRoutineOutputCreateInput = {
  data: Scalars['String'];
  id: Scalars['ID'];
  outputConnect: Scalars['ID'];
  runRoutineConnect: Scalars['ID'];
};

export type RunRoutineOutputEdge = {
  __typename: 'RunRoutineOutputEdge';
  cursor: Scalars['String'];
  node: RunRoutineOutput;
};

export type RunRoutineOutputSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  runRoutineIds?: InputMaybe<Array<Scalars['ID']>>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type RunRoutineOutputSearchResult = {
  __typename: 'RunRoutineOutputSearchResult';
  edges: Array<RunRoutineOutputEdge>;
  pageInfo: PageInfo;
};

export enum RunRoutineOutputSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type RunRoutineOutputUpdateInput = {
  data: Scalars['String'];
  id: Scalars['ID'];
};

export type RunRoutineSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  completedTimeFrame?: InputMaybe<TimeFrame>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  routineVersionId?: InputMaybe<Scalars['ID']>;
  scheduleEndTimeFrame?: InputMaybe<TimeFrame>;
  scheduleStartTimeFrame?: InputMaybe<TimeFrame>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RunRoutineSortBy>;
  startedTimeFrame?: InputMaybe<TimeFrame>;
  status?: InputMaybe<RunStatus>;
  statuses?: InputMaybe<Array<RunStatus>>;
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
  runRoutine: RunRoutine;
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
  runRoutineConnect: Scalars['ID'];
  status?: InputMaybe<RunRoutineStepStatus>;
  step: Array<Scalars['Int']>;
  subroutineConnect?: InputMaybe<Scalars['ID']>;
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
  outputsCreate?: InputMaybe<Array<RunRoutineOutputCreateInput>>;
  outputsDelete?: InputMaybe<Array<Scalars['ID']>>;
  outputsUpdate?: InputMaybe<Array<RunRoutineOutputUpdateInput>>;
  scheduleCreate?: InputMaybe<ScheduleCreateInput>;
  scheduleUpdate?: InputMaybe<ScheduleUpdateInput>;
  status?: InputMaybe<RunStatus>;
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

export type Schedule = {
  __typename: 'Schedule';
  created_at: Scalars['Date'];
  endTime: Scalars['Date'];
  exceptions: Array<ScheduleException>;
  focusModes: Array<FocusMode>;
  id: Scalars['ID'];
  labels: Array<Label>;
  meetings: Array<Meeting>;
  recurrences: Array<ScheduleRecurrence>;
  runProjects: Array<RunProject>;
  runRoutines: Array<RunRoutine>;
  startTime: Scalars['Date'];
  timezone: Scalars['String'];
  updated_at: Scalars['Date'];
};

export type ScheduleCreateInput = {
  endTime?: InputMaybe<Scalars['Date']>;
  exceptionsCreate?: InputMaybe<Array<ScheduleExceptionCreateInput>>;
  focusModeConnect?: InputMaybe<Scalars['ID']>;
  id: Scalars['ID'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  meetingConnect?: InputMaybe<Scalars['ID']>;
  recurrencesCreate?: InputMaybe<Array<ScheduleRecurrenceCreateInput>>;
  runProjectConnect?: InputMaybe<Scalars['ID']>;
  runRoutineConnect?: InputMaybe<Scalars['ID']>;
  startTime?: InputMaybe<Scalars['Date']>;
  timezone: Scalars['String'];
};

export type ScheduleEdge = {
  __typename: 'ScheduleEdge';
  cursor: Scalars['String'];
  node: Schedule;
};

export type ScheduleException = {
  __typename: 'ScheduleException';
  id: Scalars['ID'];
  newEndTime?: Maybe<Scalars['Date']>;
  newStartTime?: Maybe<Scalars['Date']>;
  originalStartTime: Scalars['Date'];
  schedule: Schedule;
};

export type ScheduleExceptionCreateInput = {
  id: Scalars['ID'];
  newEndTime?: InputMaybe<Scalars['Date']>;
  newStartTime?: InputMaybe<Scalars['Date']>;
  originalStartTime: Scalars['Date'];
  scheduleConnect: Scalars['ID'];
};

export type ScheduleExceptionEdge = {
  __typename: 'ScheduleExceptionEdge';
  cursor: Scalars['String'];
  node: ScheduleException;
};

export type ScheduleExceptionSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  newEndTimeFrame?: InputMaybe<TimeFrame>;
  newStartTimeFrame?: InputMaybe<TimeFrame>;
  originalStartTimeFrame?: InputMaybe<TimeFrame>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ScheduleExceptionSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type ScheduleExceptionSearchResult = {
  __typename: 'ScheduleExceptionSearchResult';
  edges: Array<ScheduleExceptionEdge>;
  pageInfo: PageInfo;
};

export enum ScheduleExceptionSortBy {
  NewEndTimeAsc = 'NewEndTimeAsc',
  NewEndTimeDesc = 'NewEndTimeDesc',
  NewStartTimeAsc = 'NewStartTimeAsc',
  NewStartTimeDesc = 'NewStartTimeDesc',
  OriginalStartTimeAsc = 'OriginalStartTimeAsc',
  OriginalStartTimeDesc = 'OriginalStartTimeDesc'
}

export type ScheduleExceptionUpdateInput = {
  id: Scalars['ID'];
  newEndTime?: InputMaybe<Scalars['Date']>;
  newStartTime?: InputMaybe<Scalars['Date']>;
  originalStartTime?: InputMaybe<Scalars['Date']>;
};

export enum ScheduleFor {
  FocusMode = 'FocusMode',
  Meeting = 'Meeting',
  RunProject = 'RunProject',
  RunRoutine = 'RunRoutine'
}

export type ScheduleRecurrence = {
  __typename: 'ScheduleRecurrence';
  dayOfMonth?: Maybe<Scalars['Int']>;
  dayOfWeek?: Maybe<Scalars['Int']>;
  duration: Scalars['Int'];
  endDate?: Maybe<Scalars['Date']>;
  id: Scalars['ID'];
  interval: Scalars['Int'];
  month?: Maybe<Scalars['Int']>;
  recurrenceType: ScheduleRecurrenceType;
  schedule: Schedule;
};

export type ScheduleRecurrenceCreateInput = {
  dayOfMonth?: InputMaybe<Scalars['Int']>;
  dayOfWeek?: InputMaybe<Scalars['Int']>;
  duration: Scalars['Int'];
  endDate?: InputMaybe<Scalars['Date']>;
  id: Scalars['ID'];
  interval: Scalars['Int'];
  month?: InputMaybe<Scalars['Int']>;
  recurrenceType: ScheduleRecurrenceType;
  scheduleConnect?: InputMaybe<Scalars['ID']>;
  scheduleCreate?: InputMaybe<ScheduleCreateInput>;
};

export type ScheduleRecurrenceEdge = {
  __typename: 'ScheduleRecurrenceEdge';
  cursor: Scalars['String'];
  node: ScheduleRecurrence;
};

export type ScheduleRecurrenceSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  dayOfMonth?: InputMaybe<Scalars['Int']>;
  dayOfWeek?: InputMaybe<Scalars['Int']>;
  endDateTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  interval?: InputMaybe<Scalars['Int']>;
  month?: InputMaybe<Scalars['Int']>;
  recurrenceType?: InputMaybe<ScheduleRecurrenceType>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ScheduleRecurrenceSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type ScheduleRecurrenceSearchResult = {
  __typename: 'ScheduleRecurrenceSearchResult';
  edges: Array<ScheduleRecurrenceEdge>;
  pageInfo: PageInfo;
};

export enum ScheduleRecurrenceSortBy {
  DayOfMonthAsc = 'DayOfMonthAsc',
  DayOfMonthDesc = 'DayOfMonthDesc',
  DayOfWeekAsc = 'DayOfWeekAsc',
  DayOfWeekDesc = 'DayOfWeekDesc',
  EndDateAsc = 'EndDateAsc',
  EndDateDesc = 'EndDateDesc',
  MonthAsc = 'MonthAsc',
  MonthDesc = 'MonthDesc'
}

export enum ScheduleRecurrenceType {
  Daily = 'Daily',
  Monthly = 'Monthly',
  Weekly = 'Weekly',
  Yearly = 'Yearly'
}

export type ScheduleRecurrenceUpdateInput = {
  dayOfMonth?: InputMaybe<Scalars['Int']>;
  dayOfWeek?: InputMaybe<Scalars['Int']>;
  duration?: InputMaybe<Scalars['Int']>;
  endDate?: InputMaybe<Scalars['Date']>;
  id: Scalars['ID'];
  interval?: InputMaybe<Scalars['Int']>;
  month?: InputMaybe<Scalars['Int']>;
  recurrenceType?: InputMaybe<ScheduleRecurrenceType>;
};

export type ScheduleSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  endTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  scheduleFor?: InputMaybe<ScheduleFor>;
  scheduleForUserId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ScheduleSortBy>;
  startTimeFrame?: InputMaybe<TimeFrame>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type ScheduleSearchResult = {
  __typename: 'ScheduleSearchResult';
  edges: Array<ScheduleEdge>;
  pageInfo: PageInfo;
};

export enum ScheduleSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  EndTimeAsc = 'EndTimeAsc',
  EndTimeDesc = 'EndTimeDesc',
  StartTimeAsc = 'StartTimeAsc',
  StartTimeDesc = 'StartTimeDesc'
}

export type ScheduleUpdateInput = {
  endTime?: InputMaybe<Scalars['Date']>;
  exceptionsCreate?: InputMaybe<Array<ScheduleExceptionCreateInput>>;
  exceptionsDelete?: InputMaybe<Array<Scalars['ID']>>;
  exceptionsUpdate?: InputMaybe<Array<ScheduleExceptionUpdateInput>>;
  id: Scalars['ID'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  labelsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  recurrencesCreate?: InputMaybe<Array<ScheduleRecurrenceCreateInput>>;
  recurrencesDelete?: InputMaybe<Array<Scalars['ID']>>;
  recurrencesUpdate?: InputMaybe<Array<ScheduleRecurrenceUpdateInput>>;
  startTime?: InputMaybe<Scalars['Date']>;
  timezone?: InputMaybe<Scalars['String']>;
};

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
  activeFocusMode?: Maybe<ActiveFocusMode>;
  apisCount: Scalars['Int'];
  bookmarkLists: Array<BookmarkList>;
  codesCount: Scalars['Int'];
  credits: Scalars['String'];
  focusModes: Array<FocusMode>;
  handle?: Maybe<Scalars['String']>;
  hasPremium: Scalars['Boolean'];
  id: Scalars['String'];
  languages: Array<Scalars['String']>;
  membershipsCount: Scalars['Int'];
  name?: Maybe<Scalars['String']>;
  notesCount: Scalars['Int'];
  profileImage?: Maybe<Scalars['String']>;
  projectsCount: Scalars['Int'];
  questionsAskedCount: Scalars['Int'];
  routinesCount: Scalars['Int'];
  standardsCount: Scalars['Int'];
  theme?: Maybe<Scalars['String']>;
  updated_at: Scalars['Date'];
};

export type SetActiveFocusModeInput = {
  id: Scalars['ID'];
  stopCondition: FocusModeStopCondition;
  stopTime?: InputMaybe<Scalars['Date']>;
};

export type Standard = {
  __typename: 'Standard';
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
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
  owner?: Maybe<Owner>;
  parent?: Maybe<StandardVersion>;
  permissions: Scalars['String'];
  pullRequests: Array<PullRequest>;
  pullRequestsCount: Scalars['Int'];
  questions: Array<Question>;
  questionsCount: Scalars['Int'];
  score: Scalars['Int'];
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
  isPrivate: Scalars['Boolean'];
  labelsConnect?: InputMaybe<Array<Scalars['ID']>>;
  labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
  ownedByTeamConnect?: InputMaybe<Scalars['ID']>;
  ownedByUserConnect?: InputMaybe<Scalars['ID']>;
  parentConnect?: InputMaybe<Scalars['ID']>;
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
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
  maxBookmarks?: InputMaybe<Scalars['Int']>;
  maxScore?: InputMaybe<Scalars['Int']>;
  maxViews?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minScore?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  ownedByTeamId?: InputMaybe<Scalars['ID']>;
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
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
  ownedByTeamConnect?: InputMaybe<Scalars['ID']>;
  ownedByUserConnect?: InputMaybe<Scalars['ID']>;
  permissions?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
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
  isPrivate: Scalars['Boolean'];
  props: Scalars['String'];
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  rootConnect?: InputMaybe<Scalars['ID']>;
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
  createdByIdRoot?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isCompleteWithRoot?: InputMaybe<Scalars['Boolean']>;
  isInternalWithRoot?: InputMaybe<Scalars['Boolean']>;
  isLatest?: InputMaybe<Scalars['Boolean']>;
  maxBookmarksRoot?: InputMaybe<Scalars['Int']>;
  maxScoreRoot?: InputMaybe<Scalars['Int']>;
  maxViewsRoot?: InputMaybe<Scalars['Int']>;
  minBookmarksRoot?: InputMaybe<Scalars['Int']>;
  minScoreRoot?: InputMaybe<Scalars['Int']>;
  minViewsRoot?: InputMaybe<Scalars['Int']>;
  ownedByTeamIdRoot?: InputMaybe<Scalars['ID']>;
  ownedByUserIdRoot?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  rootId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StandardVersionSortBy>;
  standardType?: InputMaybe<Scalars['String']>;
  tagsRoot?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
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
  name: Scalars['String'];
};

export type StandardVersionTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  jsonVariable?: InputMaybe<Scalars['String']>;
  language: Scalars['String'];
  name: Scalars['String'];
};

export type StandardVersionTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  jsonVariable?: InputMaybe<Scalars['String']>;
  language: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
};

export type StandardVersionUpdateInput = {
  default?: InputMaybe<Scalars['String']>;
  directoryListingsConnect?: InputMaybe<Array<Scalars['ID']>>;
  directoryListingsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  id: Scalars['ID'];
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isFile?: InputMaybe<Scalars['Boolean']>;
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
  canBookmark: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canReact: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canTransfer: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isViewed: Scalars['Boolean'];
  reaction?: Maybe<Scalars['String']>;
};

export type StartTaskInput = {
  botId: Scalars['String'];
  label: Scalars['String'];
  messageId: Scalars['ID'];
  properties: Scalars['JSONObject'];
  task: LlmTask;
  taskId: Scalars['ID'];
};

export enum StatPeriodType {
  Daily = 'Daily',
  Hourly = 'Hourly',
  Monthly = 'Monthly',
  Weekly = 'Weekly',
  Yearly = 'Yearly'
}

export type StatsApi = {
  __typename: 'StatsApi';
  calls: Scalars['Int'];
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
  PeriodStartAsc = 'PeriodStartAsc',
  PeriodStartDesc = 'PeriodStartDesc'
}

export type StatsCode = {
  __typename: 'StatsCode';
  calls: Scalars['Int'];
  id: Scalars['ID'];
  periodEnd: Scalars['Date'];
  periodStart: Scalars['Date'];
  periodType: StatPeriodType;
  routineVersions: Scalars['Int'];
};

export type StatsCodeEdge = {
  __typename: 'StatsCodeEdge';
  cursor: Scalars['String'];
  node: StatsCode;
};

export type StatsCodeSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  periodTimeFrame?: InputMaybe<TimeFrame>;
  periodType: StatPeriodType;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StatsCodeSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type StatsCodeSearchResult = {
  __typename: 'StatsCodeSearchResult';
  edges: Array<StatsCodeEdge>;
  pageInfo: PageInfo;
};

export enum StatsCodeSortBy {
  PeriodStartAsc = 'PeriodStartAsc',
  PeriodStartDesc = 'PeriodStartDesc'
}

export type StatsProject = {
  __typename: 'StatsProject';
  apis: Scalars['Int'];
  codes: Scalars['Int'];
  directories: Scalars['Int'];
  id: Scalars['ID'];
  notes: Scalars['Int'];
  periodEnd: Scalars['Date'];
  periodStart: Scalars['Date'];
  periodType: StatPeriodType;
  projects: Scalars['Int'];
  routines: Scalars['Int'];
  runCompletionTimeAverage: Scalars['Float'];
  runContextSwitchesAverage: Scalars['Float'];
  runsCompleted: Scalars['Int'];
  runsStarted: Scalars['Int'];
  standards: Scalars['Int'];
  teams: Scalars['Int'];
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
  PeriodStartAsc = 'PeriodStartAsc',
  PeriodStartDesc = 'PeriodStartDesc'
}

export type StatsQuiz = {
  __typename: 'StatsQuiz';
  completionTimeAverage: Scalars['Float'];
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
  PeriodStartAsc = 'PeriodStartAsc',
  PeriodStartDesc = 'PeriodStartDesc'
}

export type StatsRoutine = {
  __typename: 'StatsRoutine';
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
  PeriodStartAsc = 'PeriodStartAsc',
  PeriodStartDesc = 'PeriodStartDesc'
}

export type StatsSite = {
  __typename: 'StatsSite';
  activeUsers: Scalars['Int'];
  apiCalls: Scalars['Int'];
  apisCreated: Scalars['Int'];
  codeCalls: Scalars['Int'];
  codeCompletionTimeAverage: Scalars['Float'];
  codesCompleted: Scalars['Int'];
  codesCreated: Scalars['Int'];
  id: Scalars['ID'];
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
  standardCompletionTimeAverage: Scalars['Float'];
  standardsCompleted: Scalars['Int'];
  standardsCreated: Scalars['Int'];
  teamsCreated: Scalars['Int'];
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
  PeriodStartAsc = 'PeriodStartAsc',
  PeriodStartDesc = 'PeriodStartDesc'
}

export type StatsStandard = {
  __typename: 'StatsStandard';
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
  PeriodStartAsc = 'PeriodStartAsc',
  PeriodStartDesc = 'PeriodStartDesc'
}

export type StatsTeam = {
  __typename: 'StatsTeam';
  apis: Scalars['Int'];
  codes: Scalars['Int'];
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
  standards: Scalars['Int'];
};

export type StatsTeamEdge = {
  __typename: 'StatsTeamEdge';
  cursor: Scalars['String'];
  node: StatsTeam;
};

export type StatsTeamSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  periodTimeFrame?: InputMaybe<TimeFrame>;
  periodType: StatPeriodType;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StatsTeamSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type StatsTeamSearchResult = {
  __typename: 'StatsTeamSearchResult';
  edges: Array<StatsTeamEdge>;
  pageInfo: PageInfo;
};

export enum StatsTeamSortBy {
  PeriodStartAsc = 'PeriodStartAsc',
  PeriodStartDesc = 'PeriodStartDesc'
}

export type StatsUser = {
  __typename: 'StatsUser';
  apisCreated: Scalars['Int'];
  codeCompletionTimeAverage: Scalars['Float'];
  codesCompleted: Scalars['Int'];
  codesCreated: Scalars['Int'];
  id: Scalars['ID'];
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
  standardCompletionTimeAverage: Scalars['Float'];
  standardsCompleted: Scalars['Int'];
  standardsCreated: Scalars['Int'];
  teamssCreated: Scalars['Int'];
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
  PeriodStartAsc = 'PeriodStartAsc',
  PeriodStartDesc = 'PeriodStartDesc'
}

export enum SubscribableObject {
  Api = 'Api',
  Code = 'Code',
  Comment = 'Comment',
  Issue = 'Issue',
  Meeting = 'Meeting',
  Note = 'Note',
  Project = 'Project',
  PullRequest = 'PullRequest',
  Question = 'Question',
  Quiz = 'Quiz',
  Report = 'Report',
  Routine = 'Routine',
  Schedule = 'Schedule',
  Standard = 'Standard',
  Team = 'Team'
}

export type SubscribedObject = Api | Code | Comment | Issue | Meeting | Note | Project | PullRequest | Question | Quiz | Report | Routine | Schedule | Standard | Team;

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
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
  codes: Array<Code>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  notes: Array<Note>;
  posts: Array<Post>;
  projects: Array<Project>;
  reports: Array<Report>;
  routines: Array<Routine>;
  standards: Array<Standard>;
  tag: Scalars['String'];
  teams: Array<Team>;
  translations: Array<TagTranslation>;
  updated_at: Scalars['Date'];
  you: TagYou;
};

export type TagCreateInput = {
  anonymous?: InputMaybe<Scalars['Boolean']>;
  id: Scalars['ID'];
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
  maxBookmarks?: InputMaybe<Scalars['Int']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  offset?: InputMaybe<Scalars['Int']>;
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
  EmbedDateCreatedAsc = 'EmbedDateCreatedAsc',
  EmbedDateCreatedDesc = 'EmbedDateCreatedDesc',
  EmbedDateUpdatedAsc = 'EmbedDateUpdatedAsc',
  EmbedDateUpdatedDesc = 'EmbedDateUpdatedDesc',
  EmbedTopAsc = 'EmbedTopAsc',
  EmbedTopDesc = 'EmbedTopDesc'
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
  language: Scalars['String'];
};

export type TagUpdateInput = {
  anonymous?: InputMaybe<Scalars['Boolean']>;
  id: Scalars['ID'];
  tag: Scalars['String'];
  translationsCreate?: InputMaybe<Array<TagTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<TagTranslationUpdateInput>>;
};

export type TagYou = {
  __typename: 'TagYou';
  isBookmarked: Scalars['Boolean'];
  isOwn: Scalars['Boolean'];
};

export type Team = {
  __typename: 'Team';
  apis: Array<Api>;
  apisCount: Scalars['Int'];
  bannerImage?: Maybe<Scalars['String']>;
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
  codes: Array<Code>;
  codesCount: Scalars['Int'];
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  created_at: Scalars['Date'];
  directoryListings: Array<ProjectVersionDirectory>;
  forks: Array<Team>;
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
  parent?: Maybe<Team>;
  paymentHistory: Array<Payment>;
  permissions: Scalars['String'];
  posts: Array<Post>;
  postsCount: Scalars['Int'];
  premium?: Maybe<Premium>;
  profileImage?: Maybe<Scalars['String']>;
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
  standards: Array<Standard>;
  standardsCount: Scalars['Int'];
  stats: Array<StatsTeam>;
  tags: Array<Tag>;
  transfersIncoming: Array<Transfer>;
  transfersOutgoing: Array<Transfer>;
  translatedName: Scalars['String'];
  translations: Array<TeamTranslation>;
  translationsCount: Scalars['Int'];
  updated_at: Scalars['Date'];
  views: Scalars['Int'];
  wallets: Array<Wallet>;
  you: TeamYou;
};

export type TeamCreateInput = {
  bannerImage?: InputMaybe<Scalars['Upload']>;
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isOpenToNewMembers?: InputMaybe<Scalars['Boolean']>;
  isPrivate: Scalars['Boolean'];
  memberInvitesCreate?: InputMaybe<Array<MemberInviteCreateInput>>;
  permissions?: InputMaybe<Scalars['String']>;
  profileImage?: InputMaybe<Scalars['Upload']>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  rolesCreate?: InputMaybe<Array<RoleCreateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<TeamTranslationCreateInput>>;
};

export type TeamEdge = {
  __typename: 'TeamEdge';
  cursor: Scalars['String'];
  node: Team;
};

export type TeamSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isOpenToNewMembers?: InputMaybe<Scalars['Boolean']>;
  maxBookmarks?: InputMaybe<Scalars['Int']>;
  maxViews?: InputMaybe<Scalars['Int']>;
  memberUserIds?: InputMaybe<Array<Scalars['ID']>>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  projectId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<TeamSortBy>;
  standardId?: InputMaybe<Scalars['ID']>;
  tags?: InputMaybe<Array<Scalars['String']>>;
  take?: InputMaybe<Scalars['Int']>;
  translationLanguages?: InputMaybe<Array<Scalars['String']>>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  visibility?: InputMaybe<VisibilityType>;
};

export type TeamSearchResult = {
  __typename: 'TeamSearchResult';
  edges: Array<TeamEdge>;
  pageInfo: PageInfo;
};

export enum TeamSortBy {
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type TeamTranslation = {
  __typename: 'TeamTranslation';
  bio?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type TeamTranslationCreateInput = {
  bio?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type TeamTranslationUpdateInput = {
  bio?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name?: InputMaybe<Scalars['String']>;
};

export type TeamUpdateInput = {
  bannerImage?: InputMaybe<Scalars['Upload']>;
  handle?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isOpenToNewMembers?: InputMaybe<Scalars['Boolean']>;
  isPrivate?: InputMaybe<Scalars['Boolean']>;
  memberInvitesCreate?: InputMaybe<Array<MemberInviteCreateInput>>;
  memberInvitesDelete?: InputMaybe<Array<Scalars['ID']>>;
  membersDelete?: InputMaybe<Array<Scalars['ID']>>;
  permissions?: InputMaybe<Scalars['String']>;
  profileImage?: InputMaybe<Scalars['Upload']>;
  resourceListCreate?: InputMaybe<ResourceListCreateInput>;
  resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
  rolesCreate?: InputMaybe<Array<RoleCreateInput>>;
  rolesDelete?: InputMaybe<Array<Scalars['ID']>>;
  rolesUpdate?: InputMaybe<Array<RoleUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['String']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['String']>>;
  translationsCreate?: InputMaybe<Array<TeamTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<TeamTranslationUpdateInput>>;
};

export type TeamYou = {
  __typename: 'TeamYou';
  canAddMembers: Scalars['Boolean'];
  canBookmark: Scalars['Boolean'];
  canDelete: Scalars['Boolean'];
  canRead: Scalars['Boolean'];
  canReport: Scalars['Boolean'];
  canUpdate: Scalars['Boolean'];
  isBookmarked: Scalars['Boolean'];
  isViewed: Scalars['Boolean'];
  yourMembership?: Maybe<Member>;
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

export type TransferObject = Api | Code | Note | Project | Routine | Standard;

export enum TransferObjectType {
  Api = 'Api',
  Code = 'Code',
  Note = 'Note',
  Project = 'Project',
  Routine = 'Routine',
  Standard = 'Standard'
}

export type TransferRequestReceiveInput = {
  message?: InputMaybe<Scalars['String']>;
  objectConnect: Scalars['ID'];
  objectType: TransferObjectType;
  toTeamConnect?: InputMaybe<Scalars['ID']>;
};

export type TransferRequestSendInput = {
  message?: InputMaybe<Scalars['String']>;
  objectConnect: Scalars['ID'];
  objectType: TransferObjectType;
  toTeamConnect?: InputMaybe<Scalars['ID']>;
  toUserConnect?: InputMaybe<Scalars['ID']>;
};

export type TransferSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  apiId?: InputMaybe<Scalars['ID']>;
  codeId?: InputMaybe<Scalars['ID']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  fromTeamId?: InputMaybe<Scalars['ID']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  noteId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<TransferSortBy>;
  standardId?: InputMaybe<Scalars['ID']>;
  status?: InputMaybe<TransferStatus>;
  take?: InputMaybe<Scalars['Int']>;
  toTeamId?: InputMaybe<Scalars['ID']>;
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
  apisCount: Scalars['Int'];
  apisCreated?: Maybe<Array<Api>>;
  awards?: Maybe<Array<Award>>;
  bannerImage?: Maybe<Scalars['String']>;
  bookmarked?: Maybe<Array<Bookmark>>;
  bookmarkedBy: Array<User>;
  bookmarks: Scalars['Int'];
  botSettings?: Maybe<Scalars['String']>;
  codes?: Maybe<Array<Code>>;
  codesCount: Scalars['Int'];
  codesCreated?: Maybe<Array<Code>>;
  comments?: Maybe<Array<Comment>>;
  created_at: Scalars['Date'];
  emails?: Maybe<Array<Email>>;
  focusModes?: Maybe<Array<FocusMode>>;
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  invitedByUser?: Maybe<User>;
  invitedUsers?: Maybe<Array<User>>;
  isBot: Scalars['Boolean'];
  isBotDepictingPerson: Scalars['Boolean'];
  isPrivate: Scalars['Boolean'];
  isPrivateApis: Scalars['Boolean'];
  isPrivateApisCreated: Scalars['Boolean'];
  isPrivateBookmarks: Scalars['Boolean'];
  isPrivateCodes: Scalars['Boolean'];
  isPrivateCodesCreated: Scalars['Boolean'];
  isPrivateMemberships: Scalars['Boolean'];
  isPrivateProjects: Scalars['Boolean'];
  isPrivateProjectsCreated: Scalars['Boolean'];
  isPrivatePullRequests: Scalars['Boolean'];
  isPrivateQuestionsAnswered: Scalars['Boolean'];
  isPrivateQuestionsAsked: Scalars['Boolean'];
  isPrivateQuizzesCreated: Scalars['Boolean'];
  isPrivateRoles: Scalars['Boolean'];
  isPrivateRoutines: Scalars['Boolean'];
  isPrivateRoutinesCreated: Scalars['Boolean'];
  isPrivateStandards: Scalars['Boolean'];
  isPrivateStandardsCreated: Scalars['Boolean'];
  isPrivateTeamsCreated: Scalars['Boolean'];
  isPrivateVotes: Scalars['Boolean'];
  issuesClosed?: Maybe<Array<Issue>>;
  issuesCreated?: Maybe<Array<Issue>>;
  labels?: Maybe<Array<Label>>;
  meetingsAttending?: Maybe<Array<Meeting>>;
  meetingsInvited?: Maybe<Array<MeetingInvite>>;
  memberships?: Maybe<Array<Member>>;
  membershipsCount: Scalars['Int'];
  membershipsInvited?: Maybe<Array<MemberInvite>>;
  name: Scalars['String'];
  notes?: Maybe<Array<Note>>;
  notesCount: Scalars['Int'];
  notesCreated?: Maybe<Array<Note>>;
  notificationSettings?: Maybe<Scalars['String']>;
  notificationSubscriptions?: Maybe<Array<NotificationSubscription>>;
  notifications?: Maybe<Array<Notification>>;
  paymentHistory?: Maybe<Array<Payment>>;
  phones?: Maybe<Array<Phone>>;
  premium?: Maybe<Premium>;
  profileImage?: Maybe<Scalars['String']>;
  projects?: Maybe<Array<Project>>;
  projectsCount: Scalars['Int'];
  projectsCreated?: Maybe<Array<Project>>;
  pullRequests?: Maybe<Array<PullRequest>>;
  pushDevices?: Maybe<Array<PushDevice>>;
  questionsAnswered?: Maybe<Array<QuestionAnswer>>;
  questionsAsked?: Maybe<Array<Question>>;
  questionsAskedCount: Scalars['Int'];
  quizzesCreated?: Maybe<Array<Quiz>>;
  quizzesTaken?: Maybe<Array<Quiz>>;
  reacted?: Maybe<Array<Reaction>>;
  reportResponses?: Maybe<Array<ReportResponse>>;
  reportsCreated?: Maybe<Array<Report>>;
  reportsReceived: Array<Report>;
  reportsReceivedCount: Scalars['Int'];
  reputationHistory?: Maybe<Array<ReputationHistory>>;
  roles?: Maybe<Array<Role>>;
  routines?: Maybe<Array<Routine>>;
  routinesCount: Scalars['Int'];
  routinesCreated?: Maybe<Array<Routine>>;
  runProjects?: Maybe<Array<RunProject>>;
  runRoutines?: Maybe<Array<RunRoutine>>;
  sentReports?: Maybe<Array<Report>>;
  standards?: Maybe<Array<Standard>>;
  standardsCount: Scalars['Int'];
  standardsCreated?: Maybe<Array<Standard>>;
  status?: Maybe<AccountStatus>;
  tags?: Maybe<Array<Tag>>;
  teamsCreated?: Maybe<Array<Team>>;
  theme?: Maybe<Scalars['String']>;
  transfersIncoming?: Maybe<Array<Transfer>>;
  transfersOutgoing?: Maybe<Array<Transfer>>;
  translationLanguages?: Maybe<Array<Scalars['String']>>;
  translations: Array<UserTranslation>;
  updated_at?: Maybe<Scalars['Date']>;
  viewed?: Maybe<Array<View>>;
  viewedBy?: Maybe<Array<View>>;
  views: Scalars['Int'];
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

export type UserSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isBot?: InputMaybe<Scalars['Boolean']>;
  isBotDepictingPerson?: InputMaybe<Scalars['Boolean']>;
  maxBookmarks?: InputMaybe<Scalars['Int']>;
  maxViews?: InputMaybe<Scalars['Int']>;
  memberInTeamId?: InputMaybe<Scalars['ID']>;
  minBookmarks?: InputMaybe<Scalars['Int']>;
  minViews?: InputMaybe<Scalars['Int']>;
  notInChatId?: InputMaybe<Scalars['ID']>;
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
  BookmarksAsc = 'BookmarksAsc',
  BookmarksDesc = 'BookmarksDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
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
  language: Scalars['String'];
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

export type ValidateVerificationTextInput = {
  phoneNumber: Scalars['String'];
  verificationCode: Scalars['String'];
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

export type ViewTo = Api | Code | Issue | Note | Post | Project | Question | Routine | Standard | Team | User;

export enum VisibilityType {
  All = 'All',
  Own = 'Own',
  Private = 'Private',
  Public = 'Public'
}

export type Wallet = {
  __typename: 'Wallet';
  id: Scalars['ID'];
  name?: Maybe<Scalars['String']>;
  publicAddress?: Maybe<Scalars['String']>;
  stakingAddress: Scalars['String'];
  team?: Maybe<Team>;
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
  ActiveFocusMode: ResolverTypeWrapper<ActiveFocusMode>;
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
  AutoFillInput: AutoFillInput;
  AutoFillResult: ResolverTypeWrapper<AutoFillResult>;
  Award: ResolverTypeWrapper<Award>;
  AwardCategory: AwardCategory;
  AwardEdge: ResolverTypeWrapper<AwardEdge>;
  AwardSearchInput: AwardSearchInput;
  AwardSearchResult: ResolverTypeWrapper<AwardSearchResult>;
  AwardSortBy: AwardSortBy;
  Bookmark: ResolverTypeWrapper<Omit<Bookmark, 'to'> & { to: ResolversTypes['BookmarkTo'] }>;
  BookmarkCreateInput: BookmarkCreateInput;
  BookmarkEdge: ResolverTypeWrapper<BookmarkEdge>;
  BookmarkFor: BookmarkFor;
  BookmarkList: ResolverTypeWrapper<BookmarkList>;
  BookmarkListCreateInput: BookmarkListCreateInput;
  BookmarkListEdge: ResolverTypeWrapper<BookmarkListEdge>;
  BookmarkListSearchInput: BookmarkListSearchInput;
  BookmarkListSearchResult: ResolverTypeWrapper<BookmarkListSearchResult>;
  BookmarkListSortBy: BookmarkListSortBy;
  BookmarkListUpdateInput: BookmarkListUpdateInput;
  BookmarkSearchInput: BookmarkSearchInput;
  BookmarkSearchResult: ResolverTypeWrapper<BookmarkSearchResult>;
  BookmarkSortBy: BookmarkSortBy;
  BookmarkTo: ResolversTypes['Api'] | ResolversTypes['Code'] | ResolversTypes['Comment'] | ResolversTypes['Issue'] | ResolversTypes['Note'] | ResolversTypes['Post'] | ResolversTypes['Project'] | ResolversTypes['Question'] | ResolversTypes['QuestionAnswer'] | ResolversTypes['Quiz'] | ResolversTypes['Routine'] | ResolversTypes['Standard'] | ResolversTypes['Tag'] | ResolversTypes['Team'] | ResolversTypes['User'];
  BookmarkUpdateInput: BookmarkUpdateInput;
  Boolean: ResolverTypeWrapper<Scalars['Boolean']>;
  BotCreateInput: BotCreateInput;
  BotUpdateInput: BotUpdateInput;
  CancelTaskInput: CancelTaskInput;
  Chat: ResolverTypeWrapper<Chat>;
  ChatCreateInput: ChatCreateInput;
  ChatEdge: ResolverTypeWrapper<ChatEdge>;
  ChatInvite: ResolverTypeWrapper<ChatInvite>;
  ChatInviteCreateInput: ChatInviteCreateInput;
  ChatInviteEdge: ResolverTypeWrapper<ChatInviteEdge>;
  ChatInviteSearchInput: ChatInviteSearchInput;
  ChatInviteSearchResult: ResolverTypeWrapper<ChatInviteSearchResult>;
  ChatInviteSortBy: ChatInviteSortBy;
  ChatInviteStatus: ChatInviteStatus;
  ChatInviteUpdateInput: ChatInviteUpdateInput;
  ChatInviteYou: ResolverTypeWrapper<ChatInviteYou>;
  ChatMessage: ResolverTypeWrapper<ChatMessage>;
  ChatMessageCreateInput: ChatMessageCreateInput;
  ChatMessageEdge: ResolverTypeWrapper<ChatMessageEdge>;
  ChatMessageParent: ResolverTypeWrapper<ChatMessageParent>;
  ChatMessageSearchInput: ChatMessageSearchInput;
  ChatMessageSearchResult: ResolverTypeWrapper<ChatMessageSearchResult>;
  ChatMessageSearchTreeInput: ChatMessageSearchTreeInput;
  ChatMessageSearchTreeResult: ResolverTypeWrapper<ChatMessageSearchTreeResult>;
  ChatMessageSortBy: ChatMessageSortBy;
  ChatMessageTranslation: ResolverTypeWrapper<ChatMessageTranslation>;
  ChatMessageTranslationCreateInput: ChatMessageTranslationCreateInput;
  ChatMessageTranslationUpdateInput: ChatMessageTranslationUpdateInput;
  ChatMessageUpdateInput: ChatMessageUpdateInput;
  ChatMessageYou: ResolverTypeWrapper<ChatMessageYou>;
  ChatMessageedOn: ResolversTypes['ApiVersion'] | ResolversTypes['CodeVersion'] | ResolversTypes['Issue'] | ResolversTypes['NoteVersion'] | ResolversTypes['Post'] | ResolversTypes['ProjectVersion'] | ResolversTypes['PullRequest'] | ResolversTypes['Question'] | ResolversTypes['QuestionAnswer'] | ResolversTypes['RoutineVersion'] | ResolversTypes['StandardVersion'];
  ChatParticipant: ResolverTypeWrapper<ChatParticipant>;
  ChatParticipantEdge: ResolverTypeWrapper<ChatParticipantEdge>;
  ChatParticipantSearchInput: ChatParticipantSearchInput;
  ChatParticipantSearchResult: ResolverTypeWrapper<ChatParticipantSearchResult>;
  ChatParticipantSortBy: ChatParticipantSortBy;
  ChatParticipantUpdateInput: ChatParticipantUpdateInput;
  ChatSearchInput: ChatSearchInput;
  ChatSearchResult: ResolverTypeWrapper<ChatSearchResult>;
  ChatSortBy: ChatSortBy;
  ChatTranslation: ResolverTypeWrapper<ChatTranslation>;
  ChatTranslationCreateInput: ChatTranslationCreateInput;
  ChatTranslationUpdateInput: ChatTranslationUpdateInput;
  ChatUpdateInput: ChatUpdateInput;
  ChatYou: ResolverTypeWrapper<ChatYou>;
  CheckTaskStatusesInput: CheckTaskStatusesInput;
  CheckTaskStatusesResult: ResolverTypeWrapper<CheckTaskStatusesResult>;
  Code: ResolverTypeWrapper<Omit<Code, 'owner'> & { owner?: Maybe<ResolversTypes['Owner']> }>;
  CodeCreateInput: CodeCreateInput;
  CodeEdge: ResolverTypeWrapper<CodeEdge>;
  CodeSearchInput: CodeSearchInput;
  CodeSearchResult: ResolverTypeWrapper<CodeSearchResult>;
  CodeSortBy: CodeSortBy;
  CodeType: CodeType;
  CodeUpdateInput: CodeUpdateInput;
  CodeVersion: ResolverTypeWrapper<CodeVersion>;
  CodeVersionCreateInput: CodeVersionCreateInput;
  CodeVersionEdge: ResolverTypeWrapper<CodeVersionEdge>;
  CodeVersionSearchInput: CodeVersionSearchInput;
  CodeVersionSearchResult: ResolverTypeWrapper<CodeVersionSearchResult>;
  CodeVersionSortBy: CodeVersionSortBy;
  CodeVersionTranslation: ResolverTypeWrapper<CodeVersionTranslation>;
  CodeVersionTranslationCreateInput: CodeVersionTranslationCreateInput;
  CodeVersionTranslationUpdateInput: CodeVersionTranslationUpdateInput;
  CodeVersionUpdateInput: CodeVersionUpdateInput;
  CodeYou: ResolverTypeWrapper<CodeYou>;
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
  CommentedOn: ResolversTypes['ApiVersion'] | ResolversTypes['CodeVersion'] | ResolversTypes['Issue'] | ResolversTypes['NoteVersion'] | ResolversTypes['Post'] | ResolversTypes['ProjectVersion'] | ResolversTypes['PullRequest'] | ResolversTypes['Question'] | ResolversTypes['QuestionAnswer'] | ResolversTypes['RoutineVersion'] | ResolversTypes['StandardVersion'];
  CopyInput: CopyInput;
  CopyResult: ResolverTypeWrapper<CopyResult>;
  CopyType: CopyType;
  Count: ResolverTypeWrapper<Count>;
  Date: ResolverTypeWrapper<Scalars['Date']>;
  DeleteManyInput: DeleteManyInput;
  DeleteOneInput: DeleteOneInput;
  DeleteType: DeleteType;
  Email: ResolverTypeWrapper<Email>;
  EmailCreateInput: EmailCreateInput;
  EmailLogInInput: EmailLogInInput;
  EmailRequestPasswordChangeInput: EmailRequestPasswordChangeInput;
  EmailResetPasswordInput: EmailResetPasswordInput;
  EmailSignUpInput: EmailSignUpInput;
  FindByIdInput: FindByIdInput;
  FindByIdOrHandleInput: FindByIdOrHandleInput;
  FindVersionInput: FindVersionInput;
  Float: ResolverTypeWrapper<Scalars['Float']>;
  FocusMode: ResolverTypeWrapper<FocusMode>;
  FocusModeCreateInput: FocusModeCreateInput;
  FocusModeEdge: ResolverTypeWrapper<FocusModeEdge>;
  FocusModeFilter: ResolverTypeWrapper<FocusModeFilter>;
  FocusModeFilterCreateInput: FocusModeFilterCreateInput;
  FocusModeFilterType: FocusModeFilterType;
  FocusModeSearchInput: FocusModeSearchInput;
  FocusModeSearchResult: ResolverTypeWrapper<FocusModeSearchResult>;
  FocusModeSortBy: FocusModeSortBy;
  FocusModeStopCondition: FocusModeStopCondition;
  FocusModeUpdateInput: FocusModeUpdateInput;
  FocusModeYou: ResolverTypeWrapper<FocusModeYou>;
  GqlModelType: GqlModelType;
  HomeInput: HomeInput;
  HomeResult: ResolverTypeWrapper<HomeResult>;
  ID: ResolverTypeWrapper<Scalars['ID']>;
  ImportCalendarInput: ImportCalendarInput;
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
  IssueTo: ResolversTypes['Api'] | ResolversTypes['Code'] | ResolversTypes['Note'] | ResolversTypes['Project'] | ResolversTypes['Routine'] | ResolversTypes['Standard'] | ResolversTypes['Team'];
  IssueTranslation: ResolverTypeWrapper<IssueTranslation>;
  IssueTranslationCreateInput: IssueTranslationCreateInput;
  IssueTranslationUpdateInput: IssueTranslationUpdateInput;
  IssueUpdateInput: IssueUpdateInput;
  IssueYou: ResolverTypeWrapper<IssueYou>;
  JSONObject: ResolverTypeWrapper<Scalars['JSONObject']>;
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
  LlmTask: LlmTask;
  LlmTaskStatus: LlmTaskStatus;
  LlmTaskStatusInfo: ResolverTypeWrapper<LlmTaskStatusInfo>;
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
  MemberYou: ResolverTypeWrapper<MemberYou>;
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
  NotePage: ResolverTypeWrapper<NotePage>;
  NotePageCreateInput: NotePageCreateInput;
  NotePageUpdateInput: NotePageUpdateInput;
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
  Owner: ResolversTypes['Team'] | ResolversTypes['User'];
  PageInfo: ResolverTypeWrapper<PageInfo>;
  Payment: ResolverTypeWrapper<Payment>;
  PaymentEdge: ResolverTypeWrapper<PaymentEdge>;
  PaymentSearchInput: PaymentSearchInput;
  PaymentSearchResult: ResolverTypeWrapper<PaymentSearchResult>;
  PaymentSortBy: PaymentSortBy;
  PaymentStatus: PaymentStatus;
  PaymentType: PaymentType;
  Phone: ResolverTypeWrapper<Phone>;
  PhoneCreateInput: PhoneCreateInput;
  Popular: ResolversTypes['Api'] | ResolversTypes['Code'] | ResolversTypes['Note'] | ResolversTypes['Project'] | ResolversTypes['Question'] | ResolversTypes['Routine'] | ResolversTypes['Standard'] | ResolversTypes['Team'] | ResolversTypes['User'];
  PopularEdge: ResolverTypeWrapper<Omit<PopularEdge, 'node'> & { node: ResolversTypes['Popular'] }>;
  PopularObjectType: PopularObjectType;
  PopularPageInfo: ResolverTypeWrapper<PopularPageInfo>;
  PopularSearchInput: PopularSearchInput;
  PopularSearchResult: ResolverTypeWrapper<PopularSearchResult>;
  PopularSortBy: PopularSortBy;
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
  ProjectOrRoutine: ResolversTypes['Project'] | ResolversTypes['Routine'];
  ProjectOrRoutineEdge: ResolverTypeWrapper<Omit<ProjectOrRoutineEdge, 'node'> & { node: ResolversTypes['ProjectOrRoutine'] }>;
  ProjectOrRoutinePageInfo: ResolverTypeWrapper<ProjectOrRoutinePageInfo>;
  ProjectOrRoutineSearchInput: ProjectOrRoutineSearchInput;
  ProjectOrRoutineSearchResult: ResolverTypeWrapper<ProjectOrRoutineSearchResult>;
  ProjectOrRoutineSortBy: ProjectOrRoutineSortBy;
  ProjectOrTeam: ResolversTypes['Project'] | ResolversTypes['Team'];
  ProjectOrTeamEdge: ResolverTypeWrapper<Omit<ProjectOrTeamEdge, 'node'> & { node: ResolversTypes['ProjectOrTeam'] }>;
  ProjectOrTeamPageInfo: ResolverTypeWrapper<ProjectOrTeamPageInfo>;
  ProjectOrTeamSearchInput: ProjectOrTeamSearchInput;
  ProjectOrTeamSearchResult: ResolverTypeWrapper<ProjectOrTeamSearchResult>;
  ProjectOrTeamSortBy: ProjectOrTeamSortBy;
  ProjectSearchInput: ProjectSearchInput;
  ProjectSearchResult: ResolverTypeWrapper<ProjectSearchResult>;
  ProjectSortBy: ProjectSortBy;
  ProjectUpdateInput: ProjectUpdateInput;
  ProjectVersion: ResolverTypeWrapper<ProjectVersion>;
  ProjectVersionContentsSearchInput: ProjectVersionContentsSearchInput;
  ProjectVersionContentsSearchResult: ResolverTypeWrapper<ProjectVersionContentsSearchResult>;
  ProjectVersionContentsSortBy: ProjectVersionContentsSortBy;
  ProjectVersionCreateInput: ProjectVersionCreateInput;
  ProjectVersionDirectory: ResolverTypeWrapper<ProjectVersionDirectory>;
  ProjectVersionDirectoryCreateInput: ProjectVersionDirectoryCreateInput;
  ProjectVersionDirectoryEdge: ResolverTypeWrapper<ProjectVersionDirectoryEdge>;
  ProjectVersionDirectorySearchInput: ProjectVersionDirectorySearchInput;
  ProjectVersionDirectorySearchResult: ResolverTypeWrapper<ProjectVersionDirectorySearchResult>;
  ProjectVersionDirectorySortBy: ProjectVersionDirectorySortBy;
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
  PullRequestFrom: ResolversTypes['ApiVersion'] | ResolversTypes['CodeVersion'] | ResolversTypes['NoteVersion'] | ResolversTypes['ProjectVersion'] | ResolversTypes['RoutineVersion'] | ResolversTypes['StandardVersion'];
  PullRequestFromObjectType: PullRequestFromObjectType;
  PullRequestSearchInput: PullRequestSearchInput;
  PullRequestSearchResult: ResolverTypeWrapper<PullRequestSearchResult>;
  PullRequestSortBy: PullRequestSortBy;
  PullRequestStatus: PullRequestStatus;
  PullRequestTo: ResolversTypes['Api'] | ResolversTypes['Code'] | ResolversTypes['Note'] | ResolversTypes['Project'] | ResolversTypes['Routine'] | ResolversTypes['Standard'];
  PullRequestToObjectType: PullRequestToObjectType;
  PullRequestTranslation: ResolverTypeWrapper<PullRequestTranslation>;
  PullRequestTranslationCreateInput: PullRequestTranslationCreateInput;
  PullRequestTranslationUpdateInput: PullRequestTranslationUpdateInput;
  PullRequestUpdateInput: PullRequestUpdateInput;
  PullRequestYou: ResolverTypeWrapper<PullRequestYou>;
  PushDevice: ResolverTypeWrapper<PushDevice>;
  PushDeviceCreateInput: PushDeviceCreateInput;
  PushDeviceKeysInput: PushDeviceKeysInput;
  PushDeviceTestInput: PushDeviceTestInput;
  PushDeviceUpdateInput: PushDeviceUpdateInput;
  Query: ResolverTypeWrapper<{}>;
  Question: ResolverTypeWrapper<Omit<Question, 'forObject'> & { forObject?: Maybe<ResolversTypes['QuestionFor']> }>;
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
  QuestionFor: ResolversTypes['Api'] | ResolversTypes['Code'] | ResolversTypes['Note'] | ResolversTypes['Project'] | ResolversTypes['Routine'] | ResolversTypes['Standard'] | ResolversTypes['Team'];
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
  ReactInput: ReactInput;
  Reaction: ResolverTypeWrapper<Omit<Reaction, 'to'> & { to: ResolversTypes['ReactionTo'] }>;
  ReactionEdge: ResolverTypeWrapper<ReactionEdge>;
  ReactionFor: ReactionFor;
  ReactionSearchInput: ReactionSearchInput;
  ReactionSearchResult: ResolverTypeWrapper<ReactionSearchResult>;
  ReactionSortBy: ReactionSortBy;
  ReactionSummary: ResolverTypeWrapper<ReactionSummary>;
  ReactionTo: ResolversTypes['Api'] | ResolversTypes['ChatMessage'] | ResolversTypes['Code'] | ResolversTypes['Comment'] | ResolversTypes['Issue'] | ResolversTypes['Note'] | ResolversTypes['Post'] | ResolversTypes['Project'] | ResolversTypes['Question'] | ResolversTypes['QuestionAnswer'] | ResolversTypes['Quiz'] | ResolversTypes['Routine'] | ResolversTypes['Standard'];
  ReadAssetsInput: ReadAssetsInput;
  RegenerateResponseInput: RegenerateResponseInput;
  Reminder: ResolverTypeWrapper<Reminder>;
  ReminderCreateInput: ReminderCreateInput;
  ReminderEdge: ResolverTypeWrapper<ReminderEdge>;
  ReminderItem: ResolverTypeWrapper<ReminderItem>;
  ReminderItemCreateInput: ReminderItemCreateInput;
  ReminderItemUpdateInput: ReminderItemUpdateInput;
  ReminderList: ResolverTypeWrapper<ReminderList>;
  ReminderListCreateInput: ReminderListCreateInput;
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
  ReportStatus: ReportStatus;
  ReportSuggestedAction: ReportSuggestedAction;
  ReportUpdateInput: ReportUpdateInput;
  ReportYou: ResolverTypeWrapper<ReportYou>;
  ReputationHistory: ResolverTypeWrapper<ReputationHistory>;
  ReputationHistoryEdge: ResolverTypeWrapper<ReputationHistoryEdge>;
  ReputationHistorySearchInput: ReputationHistorySearchInput;
  ReputationHistorySearchResult: ResolverTypeWrapper<ReputationHistorySearchResult>;
  ReputationHistorySortBy: ReputationHistorySortBy;
  Resource: ResolverTypeWrapper<Resource>;
  ResourceCreateInput: ResourceCreateInput;
  ResourceEdge: ResolverTypeWrapper<ResourceEdge>;
  ResourceList: ResolverTypeWrapper<Omit<ResourceList, 'listFor'> & { listFor: ResolversTypes['ResourceListOn'] }>;
  ResourceListCreateInput: ResourceListCreateInput;
  ResourceListEdge: ResolverTypeWrapper<ResourceListEdge>;
  ResourceListFor: ResourceListFor;
  ResourceListOn: ResolversTypes['ApiVersion'] | ResolversTypes['CodeVersion'] | ResolversTypes['FocusMode'] | ResolversTypes['Post'] | ResolversTypes['ProjectVersion'] | ResolversTypes['RoutineVersion'] | ResolversTypes['StandardVersion'] | ResolversTypes['Team'];
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
  RoutineType: RoutineType;
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
  RunProjectCreateInput: RunProjectCreateInput;
  RunProjectEdge: ResolverTypeWrapper<RunProjectEdge>;
  RunProjectOrRunRoutine: ResolversTypes['RunProject'] | ResolversTypes['RunRoutine'];
  RunProjectOrRunRoutineEdge: ResolverTypeWrapper<Omit<RunProjectOrRunRoutineEdge, 'node'> & { node: ResolversTypes['RunProjectOrRunRoutine'] }>;
  RunProjectOrRunRoutinePageInfo: ResolverTypeWrapper<RunProjectOrRunRoutinePageInfo>;
  RunProjectOrRunRoutineSearchInput: RunProjectOrRunRoutineSearchInput;
  RunProjectOrRunRoutineSearchResult: ResolverTypeWrapper<RunProjectOrRunRoutineSearchResult>;
  RunProjectOrRunRoutineSortBy: RunProjectOrRunRoutineSortBy;
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
  RunRoutineCreateInput: RunRoutineCreateInput;
  RunRoutineEdge: ResolverTypeWrapper<RunRoutineEdge>;
  RunRoutineInput: ResolverTypeWrapper<RunRoutineInput>;
  RunRoutineInputCreateInput: RunRoutineInputCreateInput;
  RunRoutineInputEdge: ResolverTypeWrapper<RunRoutineInputEdge>;
  RunRoutineInputSearchInput: RunRoutineInputSearchInput;
  RunRoutineInputSearchResult: ResolverTypeWrapper<RunRoutineInputSearchResult>;
  RunRoutineInputSortBy: RunRoutineInputSortBy;
  RunRoutineInputUpdateInput: RunRoutineInputUpdateInput;
  RunRoutineOutput: ResolverTypeWrapper<RunRoutineOutput>;
  RunRoutineOutputCreateInput: RunRoutineOutputCreateInput;
  RunRoutineOutputEdge: ResolverTypeWrapper<RunRoutineOutputEdge>;
  RunRoutineOutputSearchInput: RunRoutineOutputSearchInput;
  RunRoutineOutputSearchResult: ResolverTypeWrapper<RunRoutineOutputSearchResult>;
  RunRoutineOutputSortBy: RunRoutineOutputSortBy;
  RunRoutineOutputUpdateInput: RunRoutineOutputUpdateInput;
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
  Schedule: ResolverTypeWrapper<Schedule>;
  ScheduleCreateInput: ScheduleCreateInput;
  ScheduleEdge: ResolverTypeWrapper<ScheduleEdge>;
  ScheduleException: ResolverTypeWrapper<ScheduleException>;
  ScheduleExceptionCreateInput: ScheduleExceptionCreateInput;
  ScheduleExceptionEdge: ResolverTypeWrapper<ScheduleExceptionEdge>;
  ScheduleExceptionSearchInput: ScheduleExceptionSearchInput;
  ScheduleExceptionSearchResult: ResolverTypeWrapper<ScheduleExceptionSearchResult>;
  ScheduleExceptionSortBy: ScheduleExceptionSortBy;
  ScheduleExceptionUpdateInput: ScheduleExceptionUpdateInput;
  ScheduleFor: ScheduleFor;
  ScheduleRecurrence: ResolverTypeWrapper<ScheduleRecurrence>;
  ScheduleRecurrenceCreateInput: ScheduleRecurrenceCreateInput;
  ScheduleRecurrenceEdge: ResolverTypeWrapper<ScheduleRecurrenceEdge>;
  ScheduleRecurrenceSearchInput: ScheduleRecurrenceSearchInput;
  ScheduleRecurrenceSearchResult: ResolverTypeWrapper<ScheduleRecurrenceSearchResult>;
  ScheduleRecurrenceSortBy: ScheduleRecurrenceSortBy;
  ScheduleRecurrenceType: ScheduleRecurrenceType;
  ScheduleRecurrenceUpdateInput: ScheduleRecurrenceUpdateInput;
  ScheduleSearchInput: ScheduleSearchInput;
  ScheduleSearchResult: ResolverTypeWrapper<ScheduleSearchResult>;
  ScheduleSortBy: ScheduleSortBy;
  ScheduleUpdateInput: ScheduleUpdateInput;
  SearchException: SearchException;
  SendVerificationEmailInput: SendVerificationEmailInput;
  SendVerificationTextInput: SendVerificationTextInput;
  Session: ResolverTypeWrapper<Session>;
  SessionUser: ResolverTypeWrapper<SessionUser>;
  SetActiveFocusModeInput: SetActiveFocusModeInput;
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
  StartTaskInput: StartTaskInput;
  StatPeriodType: StatPeriodType;
  StatsApi: ResolverTypeWrapper<StatsApi>;
  StatsApiEdge: ResolverTypeWrapper<StatsApiEdge>;
  StatsApiSearchInput: StatsApiSearchInput;
  StatsApiSearchResult: ResolverTypeWrapper<StatsApiSearchResult>;
  StatsApiSortBy: StatsApiSortBy;
  StatsCode: ResolverTypeWrapper<StatsCode>;
  StatsCodeEdge: ResolverTypeWrapper<StatsCodeEdge>;
  StatsCodeSearchInput: StatsCodeSearchInput;
  StatsCodeSearchResult: ResolverTypeWrapper<StatsCodeSearchResult>;
  StatsCodeSortBy: StatsCodeSortBy;
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
  StatsStandard: ResolverTypeWrapper<StatsStandard>;
  StatsStandardEdge: ResolverTypeWrapper<StatsStandardEdge>;
  StatsStandardSearchInput: StatsStandardSearchInput;
  StatsStandardSearchResult: ResolverTypeWrapper<StatsStandardSearchResult>;
  StatsStandardSortBy: StatsStandardSortBy;
  StatsTeam: ResolverTypeWrapper<StatsTeam>;
  StatsTeamEdge: ResolverTypeWrapper<StatsTeamEdge>;
  StatsTeamSearchInput: StatsTeamSearchInput;
  StatsTeamSearchResult: ResolverTypeWrapper<StatsTeamSearchResult>;
  StatsTeamSortBy: StatsTeamSortBy;
  StatsUser: ResolverTypeWrapper<StatsUser>;
  StatsUserEdge: ResolverTypeWrapper<StatsUserEdge>;
  StatsUserSearchInput: StatsUserSearchInput;
  StatsUserSearchResult: ResolverTypeWrapper<StatsUserSearchResult>;
  StatsUserSortBy: StatsUserSortBy;
  String: ResolverTypeWrapper<Scalars['String']>;
  SubscribableObject: SubscribableObject;
  SubscribedObject: ResolversTypes['Api'] | ResolversTypes['Code'] | ResolversTypes['Comment'] | ResolversTypes['Issue'] | ResolversTypes['Meeting'] | ResolversTypes['Note'] | ResolversTypes['Project'] | ResolversTypes['PullRequest'] | ResolversTypes['Question'] | ResolversTypes['Quiz'] | ResolversTypes['Report'] | ResolversTypes['Routine'] | ResolversTypes['Schedule'] | ResolversTypes['Standard'] | ResolversTypes['Team'];
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
  Team: ResolverTypeWrapper<Team>;
  TeamCreateInput: TeamCreateInput;
  TeamEdge: ResolverTypeWrapper<TeamEdge>;
  TeamSearchInput: TeamSearchInput;
  TeamSearchResult: ResolverTypeWrapper<TeamSearchResult>;
  TeamSortBy: TeamSortBy;
  TeamTranslation: ResolverTypeWrapper<TeamTranslation>;
  TeamTranslationCreateInput: TeamTranslationCreateInput;
  TeamTranslationUpdateInput: TeamTranslationUpdateInput;
  TeamUpdateInput: TeamUpdateInput;
  TeamYou: ResolverTypeWrapper<TeamYou>;
  TimeFrame: TimeFrame;
  Transfer: ResolverTypeWrapper<Omit<Transfer, 'fromOwner' | 'object' | 'toOwner'> & { fromOwner?: Maybe<ResolversTypes['Owner']>, object: ResolversTypes['TransferObject'], toOwner?: Maybe<ResolversTypes['Owner']> }>;
  TransferDenyInput: TransferDenyInput;
  TransferEdge: ResolverTypeWrapper<TransferEdge>;
  TransferObject: ResolversTypes['Api'] | ResolversTypes['Code'] | ResolversTypes['Note'] | ResolversTypes['Project'] | ResolversTypes['Routine'] | ResolversTypes['Standard'];
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
  UserSearchInput: UserSearchInput;
  UserSearchResult: ResolverTypeWrapper<UserSearchResult>;
  UserSortBy: UserSortBy;
  UserTranslation: ResolverTypeWrapper<UserTranslation>;
  UserTranslationCreateInput: UserTranslationCreateInput;
  UserTranslationUpdateInput: UserTranslationUpdateInput;
  UserYou: ResolverTypeWrapper<UserYou>;
  ValidateSessionInput: ValidateSessionInput;
  ValidateVerificationTextInput: ValidateVerificationTextInput;
  VersionYou: ResolverTypeWrapper<VersionYou>;
  View: ResolverTypeWrapper<Omit<View, 'to'> & { to: ResolversTypes['ViewTo'] }>;
  ViewEdge: ResolverTypeWrapper<ViewEdge>;
  ViewSearchInput: ViewSearchInput;
  ViewSearchResult: ResolverTypeWrapper<ViewSearchResult>;
  ViewSortBy: ViewSortBy;
  ViewTo: ResolversTypes['Api'] | ResolversTypes['Code'] | ResolversTypes['Issue'] | ResolversTypes['Note'] | ResolversTypes['Post'] | ResolversTypes['Project'] | ResolversTypes['Question'] | ResolversTypes['Routine'] | ResolversTypes['Standard'] | ResolversTypes['Team'] | ResolversTypes['User'];
  VisibilityType: VisibilityType;
  Wallet: ResolverTypeWrapper<Wallet>;
  WalletComplete: ResolverTypeWrapper<WalletComplete>;
  WalletCompleteInput: WalletCompleteInput;
  WalletInitInput: WalletInitInput;
  WalletUpdateInput: WalletUpdateInput;
  WriteAssetsInput: WriteAssetsInput;
};

/** Mapping between all available schema types and the resolvers parents */
export type ResolversParentTypes = {
  ActiveFocusMode: ActiveFocusMode;
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
  AutoFillInput: AutoFillInput;
  AutoFillResult: AutoFillResult;
  Award: Award;
  AwardEdge: AwardEdge;
  AwardSearchInput: AwardSearchInput;
  AwardSearchResult: AwardSearchResult;
  Bookmark: Omit<Bookmark, 'to'> & { to: ResolversParentTypes['BookmarkTo'] };
  BookmarkCreateInput: BookmarkCreateInput;
  BookmarkEdge: BookmarkEdge;
  BookmarkList: BookmarkList;
  BookmarkListCreateInput: BookmarkListCreateInput;
  BookmarkListEdge: BookmarkListEdge;
  BookmarkListSearchInput: BookmarkListSearchInput;
  BookmarkListSearchResult: BookmarkListSearchResult;
  BookmarkListUpdateInput: BookmarkListUpdateInput;
  BookmarkSearchInput: BookmarkSearchInput;
  BookmarkSearchResult: BookmarkSearchResult;
  BookmarkTo: ResolversParentTypes['Api'] | ResolversParentTypes['Code'] | ResolversParentTypes['Comment'] | ResolversParentTypes['Issue'] | ResolversParentTypes['Note'] | ResolversParentTypes['Post'] | ResolversParentTypes['Project'] | ResolversParentTypes['Question'] | ResolversParentTypes['QuestionAnswer'] | ResolversParentTypes['Quiz'] | ResolversParentTypes['Routine'] | ResolversParentTypes['Standard'] | ResolversParentTypes['Tag'] | ResolversParentTypes['Team'] | ResolversParentTypes['User'];
  BookmarkUpdateInput: BookmarkUpdateInput;
  Boolean: Scalars['Boolean'];
  BotCreateInput: BotCreateInput;
  BotUpdateInput: BotUpdateInput;
  CancelTaskInput: CancelTaskInput;
  Chat: Chat;
  ChatCreateInput: ChatCreateInput;
  ChatEdge: ChatEdge;
  ChatInvite: ChatInvite;
  ChatInviteCreateInput: ChatInviteCreateInput;
  ChatInviteEdge: ChatInviteEdge;
  ChatInviteSearchInput: ChatInviteSearchInput;
  ChatInviteSearchResult: ChatInviteSearchResult;
  ChatInviteUpdateInput: ChatInviteUpdateInput;
  ChatInviteYou: ChatInviteYou;
  ChatMessage: ChatMessage;
  ChatMessageCreateInput: ChatMessageCreateInput;
  ChatMessageEdge: ChatMessageEdge;
  ChatMessageParent: ChatMessageParent;
  ChatMessageSearchInput: ChatMessageSearchInput;
  ChatMessageSearchResult: ChatMessageSearchResult;
  ChatMessageSearchTreeInput: ChatMessageSearchTreeInput;
  ChatMessageSearchTreeResult: ChatMessageSearchTreeResult;
  ChatMessageTranslation: ChatMessageTranslation;
  ChatMessageTranslationCreateInput: ChatMessageTranslationCreateInput;
  ChatMessageTranslationUpdateInput: ChatMessageTranslationUpdateInput;
  ChatMessageUpdateInput: ChatMessageUpdateInput;
  ChatMessageYou: ChatMessageYou;
  ChatMessageedOn: ResolversParentTypes['ApiVersion'] | ResolversParentTypes['CodeVersion'] | ResolversParentTypes['Issue'] | ResolversParentTypes['NoteVersion'] | ResolversParentTypes['Post'] | ResolversParentTypes['ProjectVersion'] | ResolversParentTypes['PullRequest'] | ResolversParentTypes['Question'] | ResolversParentTypes['QuestionAnswer'] | ResolversParentTypes['RoutineVersion'] | ResolversParentTypes['StandardVersion'];
  ChatParticipant: ChatParticipant;
  ChatParticipantEdge: ChatParticipantEdge;
  ChatParticipantSearchInput: ChatParticipantSearchInput;
  ChatParticipantSearchResult: ChatParticipantSearchResult;
  ChatParticipantUpdateInput: ChatParticipantUpdateInput;
  ChatSearchInput: ChatSearchInput;
  ChatSearchResult: ChatSearchResult;
  ChatTranslation: ChatTranslation;
  ChatTranslationCreateInput: ChatTranslationCreateInput;
  ChatTranslationUpdateInput: ChatTranslationUpdateInput;
  ChatUpdateInput: ChatUpdateInput;
  ChatYou: ChatYou;
  CheckTaskStatusesInput: CheckTaskStatusesInput;
  CheckTaskStatusesResult: CheckTaskStatusesResult;
  Code: Omit<Code, 'owner'> & { owner?: Maybe<ResolversParentTypes['Owner']> };
  CodeCreateInput: CodeCreateInput;
  CodeEdge: CodeEdge;
  CodeSearchInput: CodeSearchInput;
  CodeSearchResult: CodeSearchResult;
  CodeUpdateInput: CodeUpdateInput;
  CodeVersion: CodeVersion;
  CodeVersionCreateInput: CodeVersionCreateInput;
  CodeVersionEdge: CodeVersionEdge;
  CodeVersionSearchInput: CodeVersionSearchInput;
  CodeVersionSearchResult: CodeVersionSearchResult;
  CodeVersionTranslation: CodeVersionTranslation;
  CodeVersionTranslationCreateInput: CodeVersionTranslationCreateInput;
  CodeVersionTranslationUpdateInput: CodeVersionTranslationUpdateInput;
  CodeVersionUpdateInput: CodeVersionUpdateInput;
  CodeYou: CodeYou;
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
  CommentedOn: ResolversParentTypes['ApiVersion'] | ResolversParentTypes['CodeVersion'] | ResolversParentTypes['Issue'] | ResolversParentTypes['NoteVersion'] | ResolversParentTypes['Post'] | ResolversParentTypes['ProjectVersion'] | ResolversParentTypes['PullRequest'] | ResolversParentTypes['Question'] | ResolversParentTypes['QuestionAnswer'] | ResolversParentTypes['RoutineVersion'] | ResolversParentTypes['StandardVersion'];
  CopyInput: CopyInput;
  CopyResult: CopyResult;
  Count: Count;
  Date: Scalars['Date'];
  DeleteManyInput: DeleteManyInput;
  DeleteOneInput: DeleteOneInput;
  Email: Email;
  EmailCreateInput: EmailCreateInput;
  EmailLogInInput: EmailLogInInput;
  EmailRequestPasswordChangeInput: EmailRequestPasswordChangeInput;
  EmailResetPasswordInput: EmailResetPasswordInput;
  EmailSignUpInput: EmailSignUpInput;
  FindByIdInput: FindByIdInput;
  FindByIdOrHandleInput: FindByIdOrHandleInput;
  FindVersionInput: FindVersionInput;
  Float: Scalars['Float'];
  FocusMode: FocusMode;
  FocusModeCreateInput: FocusModeCreateInput;
  FocusModeEdge: FocusModeEdge;
  FocusModeFilter: FocusModeFilter;
  FocusModeFilterCreateInput: FocusModeFilterCreateInput;
  FocusModeSearchInput: FocusModeSearchInput;
  FocusModeSearchResult: FocusModeSearchResult;
  FocusModeUpdateInput: FocusModeUpdateInput;
  FocusModeYou: FocusModeYou;
  HomeInput: HomeInput;
  HomeResult: HomeResult;
  ID: Scalars['ID'];
  ImportCalendarInput: ImportCalendarInput;
  Int: Scalars['Int'];
  Issue: Omit<Issue, 'to'> & { to: ResolversParentTypes['IssueTo'] };
  IssueCloseInput: IssueCloseInput;
  IssueCreateInput: IssueCreateInput;
  IssueEdge: IssueEdge;
  IssueSearchInput: IssueSearchInput;
  IssueSearchResult: IssueSearchResult;
  IssueTo: ResolversParentTypes['Api'] | ResolversParentTypes['Code'] | ResolversParentTypes['Note'] | ResolversParentTypes['Project'] | ResolversParentTypes['Routine'] | ResolversParentTypes['Standard'] | ResolversParentTypes['Team'];
  IssueTranslation: IssueTranslation;
  IssueTranslationCreateInput: IssueTranslationCreateInput;
  IssueTranslationUpdateInput: IssueTranslationUpdateInput;
  IssueUpdateInput: IssueUpdateInput;
  IssueYou: IssueYou;
  JSONObject: Scalars['JSONObject'];
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
  LlmTaskStatusInfo: LlmTaskStatusInfo;
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
  MemberYou: MemberYou;
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
  NotePage: NotePage;
  NotePageCreateInput: NotePageCreateInput;
  NotePageUpdateInput: NotePageUpdateInput;
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
  Owner: ResolversParentTypes['Team'] | ResolversParentTypes['User'];
  PageInfo: PageInfo;
  Payment: Payment;
  PaymentEdge: PaymentEdge;
  PaymentSearchInput: PaymentSearchInput;
  PaymentSearchResult: PaymentSearchResult;
  Phone: Phone;
  PhoneCreateInput: PhoneCreateInput;
  Popular: ResolversParentTypes['Api'] | ResolversParentTypes['Code'] | ResolversParentTypes['Note'] | ResolversParentTypes['Project'] | ResolversParentTypes['Question'] | ResolversParentTypes['Routine'] | ResolversParentTypes['Standard'] | ResolversParentTypes['Team'] | ResolversParentTypes['User'];
  PopularEdge: Omit<PopularEdge, 'node'> & { node: ResolversParentTypes['Popular'] };
  PopularPageInfo: PopularPageInfo;
  PopularSearchInput: PopularSearchInput;
  PopularSearchResult: PopularSearchResult;
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
  ProjectOrRoutine: ResolversParentTypes['Project'] | ResolversParentTypes['Routine'];
  ProjectOrRoutineEdge: Omit<ProjectOrRoutineEdge, 'node'> & { node: ResolversParentTypes['ProjectOrRoutine'] };
  ProjectOrRoutinePageInfo: ProjectOrRoutinePageInfo;
  ProjectOrRoutineSearchInput: ProjectOrRoutineSearchInput;
  ProjectOrRoutineSearchResult: ProjectOrRoutineSearchResult;
  ProjectOrTeam: ResolversParentTypes['Project'] | ResolversParentTypes['Team'];
  ProjectOrTeamEdge: Omit<ProjectOrTeamEdge, 'node'> & { node: ResolversParentTypes['ProjectOrTeam'] };
  ProjectOrTeamPageInfo: ProjectOrTeamPageInfo;
  ProjectOrTeamSearchInput: ProjectOrTeamSearchInput;
  ProjectOrTeamSearchResult: ProjectOrTeamSearchResult;
  ProjectSearchInput: ProjectSearchInput;
  ProjectSearchResult: ProjectSearchResult;
  ProjectUpdateInput: ProjectUpdateInput;
  ProjectVersion: ProjectVersion;
  ProjectVersionContentsSearchInput: ProjectVersionContentsSearchInput;
  ProjectVersionContentsSearchResult: ProjectVersionContentsSearchResult;
  ProjectVersionCreateInput: ProjectVersionCreateInput;
  ProjectVersionDirectory: ProjectVersionDirectory;
  ProjectVersionDirectoryCreateInput: ProjectVersionDirectoryCreateInput;
  ProjectVersionDirectoryEdge: ProjectVersionDirectoryEdge;
  ProjectVersionDirectorySearchInput: ProjectVersionDirectorySearchInput;
  ProjectVersionDirectorySearchResult: ProjectVersionDirectorySearchResult;
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
  PullRequestFrom: ResolversParentTypes['ApiVersion'] | ResolversParentTypes['CodeVersion'] | ResolversParentTypes['NoteVersion'] | ResolversParentTypes['ProjectVersion'] | ResolversParentTypes['RoutineVersion'] | ResolversParentTypes['StandardVersion'];
  PullRequestSearchInput: PullRequestSearchInput;
  PullRequestSearchResult: PullRequestSearchResult;
  PullRequestTo: ResolversParentTypes['Api'] | ResolversParentTypes['Code'] | ResolversParentTypes['Note'] | ResolversParentTypes['Project'] | ResolversParentTypes['Routine'] | ResolversParentTypes['Standard'];
  PullRequestTranslation: PullRequestTranslation;
  PullRequestTranslationCreateInput: PullRequestTranslationCreateInput;
  PullRequestTranslationUpdateInput: PullRequestTranslationUpdateInput;
  PullRequestUpdateInput: PullRequestUpdateInput;
  PullRequestYou: PullRequestYou;
  PushDevice: PushDevice;
  PushDeviceCreateInput: PushDeviceCreateInput;
  PushDeviceKeysInput: PushDeviceKeysInput;
  PushDeviceTestInput: PushDeviceTestInput;
  PushDeviceUpdateInput: PushDeviceUpdateInput;
  Query: {};
  Question: Omit<Question, 'forObject'> & { forObject?: Maybe<ResolversParentTypes['QuestionFor']> };
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
  QuestionFor: ResolversParentTypes['Api'] | ResolversParentTypes['Code'] | ResolversParentTypes['Note'] | ResolversParentTypes['Project'] | ResolversParentTypes['Routine'] | ResolversParentTypes['Standard'] | ResolversParentTypes['Team'];
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
  ReactInput: ReactInput;
  Reaction: Omit<Reaction, 'to'> & { to: ResolversParentTypes['ReactionTo'] };
  ReactionEdge: ReactionEdge;
  ReactionSearchInput: ReactionSearchInput;
  ReactionSearchResult: ReactionSearchResult;
  ReactionSummary: ReactionSummary;
  ReactionTo: ResolversParentTypes['Api'] | ResolversParentTypes['ChatMessage'] | ResolversParentTypes['Code'] | ResolversParentTypes['Comment'] | ResolversParentTypes['Issue'] | ResolversParentTypes['Note'] | ResolversParentTypes['Post'] | ResolversParentTypes['Project'] | ResolversParentTypes['Question'] | ResolversParentTypes['QuestionAnswer'] | ResolversParentTypes['Quiz'] | ResolversParentTypes['Routine'] | ResolversParentTypes['Standard'];
  ReadAssetsInput: ReadAssetsInput;
  RegenerateResponseInput: RegenerateResponseInput;
  Reminder: Reminder;
  ReminderCreateInput: ReminderCreateInput;
  ReminderEdge: ReminderEdge;
  ReminderItem: ReminderItem;
  ReminderItemCreateInput: ReminderItemCreateInput;
  ReminderItemUpdateInput: ReminderItemUpdateInput;
  ReminderList: ReminderList;
  ReminderListCreateInput: ReminderListCreateInput;
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
  Resource: Resource;
  ResourceCreateInput: ResourceCreateInput;
  ResourceEdge: ResourceEdge;
  ResourceList: Omit<ResourceList, 'listFor'> & { listFor: ResolversParentTypes['ResourceListOn'] };
  ResourceListCreateInput: ResourceListCreateInput;
  ResourceListEdge: ResourceListEdge;
  ResourceListOn: ResolversParentTypes['ApiVersion'] | ResolversParentTypes['CodeVersion'] | ResolversParentTypes['FocusMode'] | ResolversParentTypes['Post'] | ResolversParentTypes['ProjectVersion'] | ResolversParentTypes['RoutineVersion'] | ResolversParentTypes['StandardVersion'] | ResolversParentTypes['Team'];
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
  RunProjectCreateInput: RunProjectCreateInput;
  RunProjectEdge: RunProjectEdge;
  RunProjectOrRunRoutine: ResolversParentTypes['RunProject'] | ResolversParentTypes['RunRoutine'];
  RunProjectOrRunRoutineEdge: Omit<RunProjectOrRunRoutineEdge, 'node'> & { node: ResolversParentTypes['RunProjectOrRunRoutine'] };
  RunProjectOrRunRoutinePageInfo: RunProjectOrRunRoutinePageInfo;
  RunProjectOrRunRoutineSearchInput: RunProjectOrRunRoutineSearchInput;
  RunProjectOrRunRoutineSearchResult: RunProjectOrRunRoutineSearchResult;
  RunProjectSearchInput: RunProjectSearchInput;
  RunProjectSearchResult: RunProjectSearchResult;
  RunProjectStep: RunProjectStep;
  RunProjectStepCreateInput: RunProjectStepCreateInput;
  RunProjectStepUpdateInput: RunProjectStepUpdateInput;
  RunProjectUpdateInput: RunProjectUpdateInput;
  RunProjectYou: RunProjectYou;
  RunRoutine: RunRoutine;
  RunRoutineCreateInput: RunRoutineCreateInput;
  RunRoutineEdge: RunRoutineEdge;
  RunRoutineInput: RunRoutineInput;
  RunRoutineInputCreateInput: RunRoutineInputCreateInput;
  RunRoutineInputEdge: RunRoutineInputEdge;
  RunRoutineInputSearchInput: RunRoutineInputSearchInput;
  RunRoutineInputSearchResult: RunRoutineInputSearchResult;
  RunRoutineInputUpdateInput: RunRoutineInputUpdateInput;
  RunRoutineOutput: RunRoutineOutput;
  RunRoutineOutputCreateInput: RunRoutineOutputCreateInput;
  RunRoutineOutputEdge: RunRoutineOutputEdge;
  RunRoutineOutputSearchInput: RunRoutineOutputSearchInput;
  RunRoutineOutputSearchResult: RunRoutineOutputSearchResult;
  RunRoutineOutputUpdateInput: RunRoutineOutputUpdateInput;
  RunRoutineSearchInput: RunRoutineSearchInput;
  RunRoutineSearchResult: RunRoutineSearchResult;
  RunRoutineStep: RunRoutineStep;
  RunRoutineStepCreateInput: RunRoutineStepCreateInput;
  RunRoutineStepUpdateInput: RunRoutineStepUpdateInput;
  RunRoutineUpdateInput: RunRoutineUpdateInput;
  RunRoutineYou: RunRoutineYou;
  Schedule: Schedule;
  ScheduleCreateInput: ScheduleCreateInput;
  ScheduleEdge: ScheduleEdge;
  ScheduleException: ScheduleException;
  ScheduleExceptionCreateInput: ScheduleExceptionCreateInput;
  ScheduleExceptionEdge: ScheduleExceptionEdge;
  ScheduleExceptionSearchInput: ScheduleExceptionSearchInput;
  ScheduleExceptionSearchResult: ScheduleExceptionSearchResult;
  ScheduleExceptionUpdateInput: ScheduleExceptionUpdateInput;
  ScheduleRecurrence: ScheduleRecurrence;
  ScheduleRecurrenceCreateInput: ScheduleRecurrenceCreateInput;
  ScheduleRecurrenceEdge: ScheduleRecurrenceEdge;
  ScheduleRecurrenceSearchInput: ScheduleRecurrenceSearchInput;
  ScheduleRecurrenceSearchResult: ScheduleRecurrenceSearchResult;
  ScheduleRecurrenceUpdateInput: ScheduleRecurrenceUpdateInput;
  ScheduleSearchInput: ScheduleSearchInput;
  ScheduleSearchResult: ScheduleSearchResult;
  ScheduleUpdateInput: ScheduleUpdateInput;
  SearchException: SearchException;
  SendVerificationEmailInput: SendVerificationEmailInput;
  SendVerificationTextInput: SendVerificationTextInput;
  Session: Session;
  SessionUser: SessionUser;
  SetActiveFocusModeInput: SetActiveFocusModeInput;
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
  StartTaskInput: StartTaskInput;
  StatsApi: StatsApi;
  StatsApiEdge: StatsApiEdge;
  StatsApiSearchInput: StatsApiSearchInput;
  StatsApiSearchResult: StatsApiSearchResult;
  StatsCode: StatsCode;
  StatsCodeEdge: StatsCodeEdge;
  StatsCodeSearchInput: StatsCodeSearchInput;
  StatsCodeSearchResult: StatsCodeSearchResult;
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
  StatsStandard: StatsStandard;
  StatsStandardEdge: StatsStandardEdge;
  StatsStandardSearchInput: StatsStandardSearchInput;
  StatsStandardSearchResult: StatsStandardSearchResult;
  StatsTeam: StatsTeam;
  StatsTeamEdge: StatsTeamEdge;
  StatsTeamSearchInput: StatsTeamSearchInput;
  StatsTeamSearchResult: StatsTeamSearchResult;
  StatsUser: StatsUser;
  StatsUserEdge: StatsUserEdge;
  StatsUserSearchInput: StatsUserSearchInput;
  StatsUserSearchResult: StatsUserSearchResult;
  String: Scalars['String'];
  SubscribedObject: ResolversParentTypes['Api'] | ResolversParentTypes['Code'] | ResolversParentTypes['Comment'] | ResolversParentTypes['Issue'] | ResolversParentTypes['Meeting'] | ResolversParentTypes['Note'] | ResolversParentTypes['Project'] | ResolversParentTypes['PullRequest'] | ResolversParentTypes['Question'] | ResolversParentTypes['Quiz'] | ResolversParentTypes['Report'] | ResolversParentTypes['Routine'] | ResolversParentTypes['Schedule'] | ResolversParentTypes['Standard'] | ResolversParentTypes['Team'];
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
  Team: Team;
  TeamCreateInput: TeamCreateInput;
  TeamEdge: TeamEdge;
  TeamSearchInput: TeamSearchInput;
  TeamSearchResult: TeamSearchResult;
  TeamTranslation: TeamTranslation;
  TeamTranslationCreateInput: TeamTranslationCreateInput;
  TeamTranslationUpdateInput: TeamTranslationUpdateInput;
  TeamUpdateInput: TeamUpdateInput;
  TeamYou: TeamYou;
  TimeFrame: TimeFrame;
  Transfer: Omit<Transfer, 'fromOwner' | 'object' | 'toOwner'> & { fromOwner?: Maybe<ResolversParentTypes['Owner']>, object: ResolversParentTypes['TransferObject'], toOwner?: Maybe<ResolversParentTypes['Owner']> };
  TransferDenyInput: TransferDenyInput;
  TransferEdge: TransferEdge;
  TransferObject: ResolversParentTypes['Api'] | ResolversParentTypes['Code'] | ResolversParentTypes['Note'] | ResolversParentTypes['Project'] | ResolversParentTypes['Routine'] | ResolversParentTypes['Standard'];
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
  UserSearchInput: UserSearchInput;
  UserSearchResult: UserSearchResult;
  UserTranslation: UserTranslation;
  UserTranslationCreateInput: UserTranslationCreateInput;
  UserTranslationUpdateInput: UserTranslationUpdateInput;
  UserYou: UserYou;
  ValidateSessionInput: ValidateSessionInput;
  ValidateVerificationTextInput: ValidateVerificationTextInput;
  VersionYou: VersionYou;
  View: Omit<View, 'to'> & { to: ResolversParentTypes['ViewTo'] };
  ViewEdge: ViewEdge;
  ViewSearchInput: ViewSearchInput;
  ViewSearchResult: ViewSearchResult;
  ViewTo: ResolversParentTypes['Api'] | ResolversParentTypes['Code'] | ResolversParentTypes['Issue'] | ResolversParentTypes['Note'] | ResolversParentTypes['Post'] | ResolversParentTypes['Project'] | ResolversParentTypes['Question'] | ResolversParentTypes['Routine'] | ResolversParentTypes['Standard'] | ResolversParentTypes['Team'] | ResolversParentTypes['User'];
  Wallet: Wallet;
  WalletComplete: WalletComplete;
  WalletCompleteInput: WalletCompleteInput;
  WalletInitInput: WalletInitInput;
  WalletUpdateInput: WalletUpdateInput;
  WriteAssetsInput: WriteAssetsInput;
};

export type ActiveFocusModeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ActiveFocusMode'] = ResolversParentTypes['ActiveFocusMode']> = {
  mode?: Resolver<ResolversTypes['FocusMode'], ParentType, ContextType>;
  stopCondition?: Resolver<ResolversTypes['FocusModeStopCondition'], ParentType, ContextType>;
  stopTime?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ApiResolvers<ContextType = any, ParentType extends ResolversParentTypes['Api'] = ResolversParentTypes['Api']> = {
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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
  isComplete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isLatest?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  pullRequest?: Resolver<Maybe<ResolversTypes['PullRequest']>, ParentType, ContextType>;
  reports?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  reportsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  resourceList?: Resolver<Maybe<ResolversTypes['ResourceList']>, ParentType, ContextType>;
  root?: Resolver<ResolversTypes['Api'], ParentType, ContextType>;
  schemaText?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
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
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReact?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canTransfer?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isViewed?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  reaction?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type AutoFillResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['AutoFillResult'] = ResolversParentTypes['AutoFillResult']> = {
  data?: Resolver<ResolversTypes['JSONObject'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type AwardResolvers<ContextType = any, ParentType extends ResolversParentTypes['Award'] = ResolversParentTypes['Award']> = {
  category?: Resolver<ResolversTypes['AwardCategory'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  progress?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  timeCurrentTierCompleted?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  title?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type AwardEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['AwardEdge'] = ResolversParentTypes['AwardEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Award'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type AwardSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['AwardSearchResult'] = ResolversParentTypes['AwardSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['AwardEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type BookmarkResolvers<ContextType = any, ParentType extends ResolversParentTypes['Bookmark'] = ResolversParentTypes['Bookmark']> = {
  by?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  list?: Resolver<ResolversTypes['BookmarkList'], ParentType, ContextType>;
  to?: Resolver<ResolversTypes['BookmarkTo'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type BookmarkEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['BookmarkEdge'] = ResolversParentTypes['BookmarkEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Bookmark'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type BookmarkListResolvers<ContextType = any, ParentType extends ResolversParentTypes['BookmarkList'] = ResolversParentTypes['BookmarkList']> = {
  bookmarks?: Resolver<Array<ResolversTypes['Bookmark']>, ParentType, ContextType>;
  bookmarksCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  label?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type BookmarkListEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['BookmarkListEdge'] = ResolversParentTypes['BookmarkListEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['BookmarkList'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type BookmarkListSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['BookmarkListSearchResult'] = ResolversParentTypes['BookmarkListSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['BookmarkListEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type BookmarkSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['BookmarkSearchResult'] = ResolversParentTypes['BookmarkSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['BookmarkEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type BookmarkToResolvers<ContextType = any, ParentType extends ResolversParentTypes['BookmarkTo'] = ResolversParentTypes['BookmarkTo']> = {
  __resolveType: TypeResolveFn<'Api' | 'Code' | 'Comment' | 'Issue' | 'Note' | 'Post' | 'Project' | 'Question' | 'QuestionAnswer' | 'Quiz' | 'Routine' | 'Standard' | 'Tag' | 'Team' | 'User', ParentType, ContextType>;
};

export type ChatResolvers<ContextType = any, ParentType extends ResolversParentTypes['Chat'] = ResolversParentTypes['Chat']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  invites?: Resolver<Array<ResolversTypes['ChatInvite']>, ParentType, ContextType>;
  invitesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  labels?: Resolver<Array<ResolversTypes['Label']>, ParentType, ContextType>;
  labelsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  messages?: Resolver<Array<ResolversTypes['ChatMessage']>, ParentType, ContextType>;
  openToAnyoneWithInvite?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  participants?: Resolver<Array<ResolversTypes['ChatParticipant']>, ParentType, ContextType>;
  participantsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  restrictedToRoles?: Resolver<Array<ResolversTypes['Role']>, ParentType, ContextType>;
  team?: Resolver<Maybe<ResolversTypes['Team']>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['ChatTranslation']>, ParentType, ContextType>;
  translationsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['ChatYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ChatEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatEdge'] = ResolversParentTypes['ChatEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Chat'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ChatInviteResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatInvite'] = ResolversParentTypes['ChatInvite']> = {
  chat?: Resolver<ResolversTypes['Chat'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  message?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  status?: Resolver<ResolversTypes['ChatInviteStatus'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  user?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['ChatInviteYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ChatInviteEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatInviteEdge'] = ResolversParentTypes['ChatInviteEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['ChatInvite'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ChatInviteSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatInviteSearchResult'] = ResolversParentTypes['ChatInviteSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ChatInviteEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ChatInviteYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatInviteYou'] = ResolversParentTypes['ChatInviteYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ChatMessageResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatMessage'] = ResolversParentTypes['ChatMessage']> = {
  chat?: Resolver<ResolversTypes['Chat'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  parent?: Resolver<Maybe<ResolversTypes['ChatMessageParent']>, ParentType, ContextType>;
  reactionSummaries?: Resolver<Array<ResolversTypes['ReactionSummary']>, ParentType, ContextType>;
  reports?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  reportsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  sequence?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['ChatMessageTranslation']>, ParentType, ContextType>;
  translationsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  user?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
  versionIndex?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['ChatMessageYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ChatMessageEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatMessageEdge'] = ResolversParentTypes['ChatMessageEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['ChatMessage'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ChatMessageParentResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatMessageParent'] = ResolversParentTypes['ChatMessageParent']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ChatMessageSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatMessageSearchResult'] = ResolversParentTypes['ChatMessageSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ChatMessageEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ChatMessageSearchTreeResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatMessageSearchTreeResult'] = ResolversParentTypes['ChatMessageSearchTreeResult']> = {
  hasMoreDown?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  hasMoreUp?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  messages?: Resolver<Array<ResolversTypes['ChatMessage']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ChatMessageTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatMessageTranslation'] = ResolversParentTypes['ChatMessageTranslation']> = {
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  text?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ChatMessageYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatMessageYou'] = ResolversParentTypes['ChatMessageYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReact?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReply?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReport?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  reaction?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ChatMessageedOnResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatMessageedOn'] = ResolversParentTypes['ChatMessageedOn']> = {
  __resolveType: TypeResolveFn<'ApiVersion' | 'CodeVersion' | 'Issue' | 'NoteVersion' | 'Post' | 'ProjectVersion' | 'PullRequest' | 'Question' | 'QuestionAnswer' | 'RoutineVersion' | 'StandardVersion', ParentType, ContextType>;
};

export type ChatParticipantResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatParticipant'] = ResolversParentTypes['ChatParticipant']> = {
  chat?: Resolver<ResolversTypes['Chat'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  user?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ChatParticipantEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatParticipantEdge'] = ResolversParentTypes['ChatParticipantEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['ChatParticipant'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ChatParticipantSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatParticipantSearchResult'] = ResolversParentTypes['ChatParticipantSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ChatParticipantEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ChatSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatSearchResult'] = ResolversParentTypes['ChatSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ChatEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ChatTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatTranslation'] = ResolversParentTypes['ChatTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ChatYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['ChatYou'] = ResolversParentTypes['ChatYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canInvite?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CheckTaskStatusesResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['CheckTaskStatusesResult'] = ResolversParentTypes['CheckTaskStatusesResult']> = {
  statuses?: Resolver<Array<ResolversTypes['LlmTaskStatusInfo']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CodeResolvers<ContextType = any, ParentType extends ResolversParentTypes['Code'] = ResolversParentTypes['Code']> = {
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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
  parent?: Resolver<Maybe<ResolversTypes['CodeVersion']>, ParentType, ContextType>;
  permissions?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  pullRequests?: Resolver<Array<ResolversTypes['PullRequest']>, ParentType, ContextType>;
  pullRequestsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  questions?: Resolver<Array<ResolversTypes['Question']>, ParentType, ContextType>;
  questionsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  stats?: Resolver<Array<ResolversTypes['StatsCode']>, ParentType, ContextType>;
  tags?: Resolver<Array<ResolversTypes['Tag']>, ParentType, ContextType>;
  transfers?: Resolver<Array<ResolversTypes['Transfer']>, ParentType, ContextType>;
  transfersCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  translatedName?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  versions?: Resolver<Array<ResolversTypes['CodeVersion']>, ParentType, ContextType>;
  versionsCount?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  views?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['CodeYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CodeEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['CodeEdge'] = ResolversParentTypes['CodeEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Code'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CodeSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['CodeSearchResult'] = ResolversParentTypes['CodeSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['CodeEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CodeVersionResolvers<ContextType = any, ParentType extends ResolversParentTypes['CodeVersion'] = ResolversParentTypes['CodeVersion']> = {
  calledByRoutineVersionsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  codeLanguage?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  codeType?: Resolver<ResolversTypes['CodeType'], ParentType, ContextType>;
  comments?: Resolver<Array<ResolversTypes['Comment']>, ParentType, ContextType>;
  commentsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  completedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  content?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  default?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  directoryListings?: Resolver<Array<ResolversTypes['ProjectVersionDirectory']>, ParentType, ContextType>;
  directoryListingsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  forks?: Resolver<Array<ResolversTypes['Code']>, ParentType, ContextType>;
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
  root?: Resolver<ResolversTypes['Code'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['CodeVersionTranslation']>, ParentType, ContextType>;
  translationsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  versionIndex?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  versionLabel?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  versionNotes?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  you?: Resolver<ResolversTypes['VersionYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CodeVersionEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['CodeVersionEdge'] = ResolversParentTypes['CodeVersionEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['CodeVersion'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CodeVersionSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['CodeVersionSearchResult'] = ResolversParentTypes['CodeVersionSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['CodeVersionEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CodeVersionTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['CodeVersionTranslation'] = ResolversParentTypes['CodeVersionTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  jsonVariable?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CodeYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['CodeYou'] = ResolversParentTypes['CodeYou']> = {
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReact?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canTransfer?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isViewed?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  reaction?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CommentResolvers<ContextType = any, ParentType extends ResolversParentTypes['Comment'] = ResolversParentTypes['Comment']> = {
  bookmarkedBy?: Resolver<Maybe<Array<ResolversTypes['User']>>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  commentedOn?: Resolver<ResolversTypes['CommentedOn'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  owner?: Resolver<Maybe<ResolversTypes['Owner']>, ParentType, ContextType>;
  reports?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  reportsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReact?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReply?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReport?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  reaction?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CommentedOnResolvers<ContextType = any, ParentType extends ResolversParentTypes['CommentedOn'] = ResolversParentTypes['CommentedOn']> = {
  __resolveType: TypeResolveFn<'ApiVersion' | 'CodeVersion' | 'Issue' | 'NoteVersion' | 'Post' | 'ProjectVersion' | 'PullRequest' | 'Question' | 'QuestionAnswer' | 'RoutineVersion' | 'StandardVersion', ParentType, ContextType>;
};

export type CopyResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['CopyResult'] = ResolversParentTypes['CopyResult']> = {
  apiVersion?: Resolver<Maybe<ResolversTypes['ApiVersion']>, ParentType, ContextType>;
  codeVersion?: Resolver<Maybe<ResolversTypes['CodeVersion']>, ParentType, ContextType>;
  noteVersion?: Resolver<Maybe<ResolversTypes['NoteVersion']>, ParentType, ContextType>;
  projectVersion?: Resolver<Maybe<ResolversTypes['ProjectVersion']>, ParentType, ContextType>;
  routineVersion?: Resolver<Maybe<ResolversTypes['RoutineVersion']>, ParentType, ContextType>;
  standardVersion?: Resolver<Maybe<ResolversTypes['StandardVersion']>, ParentType, ContextType>;
  team?: Resolver<Maybe<ResolversTypes['Team']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type CountResolvers<ContextType = any, ParentType extends ResolversParentTypes['Count'] = ResolversParentTypes['Count']> = {
  count?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export interface DateScalarConfig extends GraphQLScalarTypeConfig<ResolversTypes['Date'], any> {
  name: 'Date';
}

export type EmailResolvers<ContextType = any, ParentType extends ResolversParentTypes['Email'] = ResolversParentTypes['Email']> = {
  emailAddress?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  verified?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type FocusModeResolvers<ContextType = any, ParentType extends ResolversParentTypes['FocusMode'] = ResolversParentTypes['FocusMode']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  filters?: Resolver<Array<ResolversTypes['FocusModeFilter']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  labels?: Resolver<Array<ResolversTypes['Label']>, ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  reminderList?: Resolver<Maybe<ResolversTypes['ReminderList']>, ParentType, ContextType>;
  resourceList?: Resolver<Maybe<ResolversTypes['ResourceList']>, ParentType, ContextType>;
  schedule?: Resolver<Maybe<ResolversTypes['Schedule']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['FocusModeYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type FocusModeEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['FocusModeEdge'] = ResolversParentTypes['FocusModeEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['FocusMode'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type FocusModeFilterResolvers<ContextType = any, ParentType extends ResolversParentTypes['FocusModeFilter'] = ResolversParentTypes['FocusModeFilter']> = {
  filterType?: Resolver<ResolversTypes['FocusModeFilterType'], ParentType, ContextType>;
  focusMode?: Resolver<ResolversTypes['FocusMode'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  tag?: Resolver<ResolversTypes['Tag'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type FocusModeSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['FocusModeSearchResult'] = ResolversParentTypes['FocusModeSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['FocusModeEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type FocusModeYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['FocusModeYou'] = ResolversParentTypes['FocusModeYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type HomeResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['HomeResult'] = ResolversParentTypes['HomeResult']> = {
  recommended?: Resolver<Array<ResolversTypes['Resource']>, ParentType, ContextType>;
  reminders?: Resolver<Array<ResolversTypes['Reminder']>, ParentType, ContextType>;
  resources?: Resolver<Array<ResolversTypes['Resource']>, ParentType, ContextType>;
  schedules?: Resolver<Array<ResolversTypes['Schedule']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type IssueResolvers<ContextType = any, ParentType extends ResolversParentTypes['Issue'] = ResolversParentTypes['Issue']> = {
  bookmarkedBy?: Resolver<Maybe<Array<ResolversTypes['Bookmark']>>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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
  __resolveType: TypeResolveFn<'Api' | 'Code' | 'Note' | 'Project' | 'Routine' | 'Standard' | 'Team', ParentType, ContextType>;
};

export type IssueTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['IssueTranslation'] = ResolversParentTypes['IssueTranslation']> = {
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type IssueYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['IssueYou'] = ResolversParentTypes['IssueYou']> = {
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canComment?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReact?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReport?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  reaction?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export interface JsonObjectScalarConfig extends GraphQLScalarTypeConfig<ResolversTypes['JSONObject'], any> {
  name: 'JSONObject';
}

export type LabelResolvers<ContextType = any, ParentType extends ResolversParentTypes['Label'] = ResolversParentTypes['Label']> = {
  apis?: Resolver<Maybe<Array<ResolversTypes['Api']>>, ParentType, ContextType>;
  apisCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  codes?: Resolver<Maybe<Array<ResolversTypes['Code']>>, ParentType, ContextType>;
  codesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  color?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  focusModes?: Resolver<Maybe<Array<ResolversTypes['FocusMode']>>, ParentType, ContextType>;
  focusModesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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
  schedules?: Resolver<Maybe<Array<ResolversTypes['Schedule']>>, ParentType, ContextType>;
  schedulesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  standards?: Resolver<Maybe<Array<ResolversTypes['Standard']>>, ParentType, ContextType>;
  standardsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['LabelTranslation']>, ParentType, ContextType>;
  translationsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
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

export type LlmTaskStatusInfoResolvers<ContextType = any, ParentType extends ResolversParentTypes['LlmTaskStatusInfo'] = ResolversParentTypes['LlmTaskStatusInfo']> = {
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  status?: Resolver<Maybe<ResolversTypes['LlmTaskStatus']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MeetingResolvers<ContextType = any, ParentType extends ResolversParentTypes['Meeting'] = ResolversParentTypes['Meeting']> = {
  attendees?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  attendeesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  invites?: Resolver<Array<ResolversTypes['MeetingInvite']>, ParentType, ContextType>;
  invitesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  labels?: Resolver<Array<ResolversTypes['Label']>, ParentType, ContextType>;
  labelsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  openToAnyoneWithInvite?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  restrictedToRoles?: Resolver<Array<ResolversTypes['Role']>, ParentType, ContextType>;
  schedule?: Resolver<Maybe<ResolversTypes['Schedule']>, ParentType, ContextType>;
  showOnTeamProfile?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  team?: Resolver<ResolversTypes['Team'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['MeetingTranslation']>, ParentType, ContextType>;
  translationsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
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
  permissions?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  roles?: Resolver<Array<ResolversTypes['Role']>, ParentType, ContextType>;
  team?: Resolver<ResolversTypes['Team'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  user?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['MemberYou'], ParentType, ContextType>;
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
  status?: Resolver<ResolversTypes['MemberInviteStatus'], ParentType, ContextType>;
  team?: Resolver<ResolversTypes['Team'], ParentType, ContextType>;
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

export type MemberYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['MemberYou'] = ResolversParentTypes['MemberYou']> = {
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MutationResolvers<ContextType = any, ParentType extends ResolversParentTypes['Mutation'] = ResolversParentTypes['Mutation']> = {
  _empty?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  apiCreate?: Resolver<ResolversTypes['Api'], ParentType, ContextType, RequireFields<MutationApiCreateArgs, 'input'>>;
  apiKeyCreate?: Resolver<ResolversTypes['ApiKey'], ParentType, ContextType, RequireFields<MutationApiKeyCreateArgs, 'input'>>;
  apiKeyDeleteOne?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationApiKeyDeleteOneArgs, 'input'>>;
  apiKeyUpdate?: Resolver<ResolversTypes['ApiKey'], ParentType, ContextType, RequireFields<MutationApiKeyUpdateArgs, 'input'>>;
  apiKeyValidate?: Resolver<ResolversTypes['ApiKey'], ParentType, ContextType, RequireFields<MutationApiKeyValidateArgs, 'input'>>;
  apiUpdate?: Resolver<ResolversTypes['Api'], ParentType, ContextType, RequireFields<MutationApiUpdateArgs, 'input'>>;
  apiVersionCreate?: Resolver<ResolversTypes['ApiVersion'], ParentType, ContextType, RequireFields<MutationApiVersionCreateArgs, 'input'>>;
  apiVersionUpdate?: Resolver<ResolversTypes['ApiVersion'], ParentType, ContextType, RequireFields<MutationApiVersionUpdateArgs, 'input'>>;
  autoFill?: Resolver<ResolversTypes['AutoFillResult'], ParentType, ContextType, RequireFields<MutationAutoFillArgs, 'input'>>;
  bookmarkCreate?: Resolver<ResolversTypes['Bookmark'], ParentType, ContextType, RequireFields<MutationBookmarkCreateArgs, 'input'>>;
  bookmarkListCreate?: Resolver<ResolversTypes['BookmarkList'], ParentType, ContextType, RequireFields<MutationBookmarkListCreateArgs, 'input'>>;
  bookmarkListUpdate?: Resolver<ResolversTypes['BookmarkList'], ParentType, ContextType, RequireFields<MutationBookmarkListUpdateArgs, 'input'>>;
  bookmarkUpdate?: Resolver<ResolversTypes['Bookmark'], ParentType, ContextType, RequireFields<MutationBookmarkUpdateArgs, 'input'>>;
  botCreate?: Resolver<ResolversTypes['User'], ParentType, ContextType, RequireFields<MutationBotCreateArgs, 'input'>>;
  botUpdate?: Resolver<ResolversTypes['User'], ParentType, ContextType, RequireFields<MutationBotUpdateArgs, 'input'>>;
  cancelTask?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationCancelTaskArgs, 'input'>>;
  chatCreate?: Resolver<ResolversTypes['Chat'], ParentType, ContextType, RequireFields<MutationChatCreateArgs, 'input'>>;
  chatInviteAccept?: Resolver<ResolversTypes['ChatInvite'], ParentType, ContextType, RequireFields<MutationChatInviteAcceptArgs, 'input'>>;
  chatInviteCreate?: Resolver<ResolversTypes['ChatInvite'], ParentType, ContextType, RequireFields<MutationChatInviteCreateArgs, 'input'>>;
  chatInviteDecline?: Resolver<ResolversTypes['ChatInvite'], ParentType, ContextType, RequireFields<MutationChatInviteDeclineArgs, 'input'>>;
  chatInviteUpdate?: Resolver<ResolversTypes['ChatInvite'], ParentType, ContextType, RequireFields<MutationChatInviteUpdateArgs, 'input'>>;
  chatInvitesCreate?: Resolver<Array<ResolversTypes['ChatInvite']>, ParentType, ContextType, RequireFields<MutationChatInvitesCreateArgs, 'input'>>;
  chatInvitesUpdate?: Resolver<Array<ResolversTypes['ChatInvite']>, ParentType, ContextType, RequireFields<MutationChatInvitesUpdateArgs, 'input'>>;
  chatMessageCreate?: Resolver<ResolversTypes['ChatMessage'], ParentType, ContextType, RequireFields<MutationChatMessageCreateArgs, 'input'>>;
  chatMessageUpdate?: Resolver<ResolversTypes['ChatMessage'], ParentType, ContextType, RequireFields<MutationChatMessageUpdateArgs, 'input'>>;
  chatParticipantUpdate?: Resolver<ResolversTypes['ChatParticipant'], ParentType, ContextType, RequireFields<MutationChatParticipantUpdateArgs, 'input'>>;
  chatUpdate?: Resolver<ResolversTypes['Chat'], ParentType, ContextType, RequireFields<MutationChatUpdateArgs, 'input'>>;
  checkTaskStatuses?: Resolver<ResolversTypes['CheckTaskStatusesResult'], ParentType, ContextType, RequireFields<MutationCheckTaskStatusesArgs, 'input'>>;
  codeCreate?: Resolver<ResolversTypes['Code'], ParentType, ContextType, RequireFields<MutationCodeCreateArgs, 'input'>>;
  codeUpdate?: Resolver<ResolversTypes['Api'], ParentType, ContextType, RequireFields<MutationCodeUpdateArgs, 'input'>>;
  codeVersionCreate?: Resolver<ResolversTypes['CodeVersion'], ParentType, ContextType, RequireFields<MutationCodeVersionCreateArgs, 'input'>>;
  codeVersionUpdate?: Resolver<ResolversTypes['CodeVersion'], ParentType, ContextType, RequireFields<MutationCodeVersionUpdateArgs, 'input'>>;
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
  exportCalendar?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  exportData?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  focusModeCreate?: Resolver<ResolversTypes['FocusMode'], ParentType, ContextType, RequireFields<MutationFocusModeCreateArgs, 'input'>>;
  focusModeUpdate?: Resolver<ResolversTypes['FocusMode'], ParentType, ContextType, RequireFields<MutationFocusModeUpdateArgs, 'input'>>;
  guestLogIn?: Resolver<ResolversTypes['Session'], ParentType, ContextType>;
  importCalendar?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationImportCalendarArgs, 'input'>>;
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
  meetingInvitesCreate?: Resolver<Array<ResolversTypes['MeetingInvite']>, ParentType, ContextType, RequireFields<MutationMeetingInvitesCreateArgs, 'input'>>;
  meetingInvitesUpdate?: Resolver<Array<ResolversTypes['MeetingInvite']>, ParentType, ContextType, RequireFields<MutationMeetingInvitesUpdateArgs, 'input'>>;
  meetingUpdate?: Resolver<ResolversTypes['Meeting'], ParentType, ContextType, RequireFields<MutationMeetingUpdateArgs, 'input'>>;
  memberInviteAccept?: Resolver<ResolversTypes['MemberInvite'], ParentType, ContextType, RequireFields<MutationMemberInviteAcceptArgs, 'input'>>;
  memberInviteCreate?: Resolver<ResolversTypes['MemberInvite'], ParentType, ContextType, RequireFields<MutationMemberInviteCreateArgs, 'input'>>;
  memberInviteDecline?: Resolver<ResolversTypes['MemberInvite'], ParentType, ContextType, RequireFields<MutationMemberInviteDeclineArgs, 'input'>>;
  memberInviteUpdate?: Resolver<ResolversTypes['MemberInvite'], ParentType, ContextType, RequireFields<MutationMemberInviteUpdateArgs, 'input'>>;
  memberInvitesCreate?: Resolver<Array<ResolversTypes['MemberInvite']>, ParentType, ContextType, RequireFields<MutationMemberInvitesCreateArgs, 'input'>>;
  memberInvitesUpdate?: Resolver<Array<ResolversTypes['MemberInvite']>, ParentType, ContextType, RequireFields<MutationMemberInvitesUpdateArgs, 'input'>>;
  memberUpdate?: Resolver<ResolversTypes['Member'], ParentType, ContextType, RequireFields<MutationMemberUpdateArgs, 'input'>>;
  nodeCreate?: Resolver<ResolversTypes['Node'], ParentType, ContextType, RequireFields<MutationNodeCreateArgs, 'input'>>;
  nodeUpdate?: Resolver<ResolversTypes['Node'], ParentType, ContextType, RequireFields<MutationNodeUpdateArgs, 'input'>>;
  noteCreate?: Resolver<ResolversTypes['Note'], ParentType, ContextType, RequireFields<MutationNoteCreateArgs, 'input'>>;
  noteUpdate?: Resolver<ResolversTypes['Note'], ParentType, ContextType, RequireFields<MutationNoteUpdateArgs, 'input'>>;
  noteVersionCreate?: Resolver<ResolversTypes['NoteVersion'], ParentType, ContextType, RequireFields<MutationNoteVersionCreateArgs, 'input'>>;
  noteVersionUpdate?: Resolver<ResolversTypes['NoteVersion'], ParentType, ContextType, RequireFields<MutationNoteVersionUpdateArgs, 'input'>>;
  notificationMarkAllAsRead?: Resolver<ResolversTypes['Success'], ParentType, ContextType>;
  notificationMarkAsRead?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationNotificationMarkAsReadArgs, 'input'>>;
  notificationSettingsUpdate?: Resolver<ResolversTypes['NotificationSettings'], ParentType, ContextType, RequireFields<MutationNotificationSettingsUpdateArgs, 'input'>>;
  notificationSubscriptionCreate?: Resolver<ResolversTypes['NotificationSubscription'], ParentType, ContextType, RequireFields<MutationNotificationSubscriptionCreateArgs, 'input'>>;
  notificationSubscriptionUpdate?: Resolver<ResolversTypes['NotificationSubscription'], ParentType, ContextType, RequireFields<MutationNotificationSubscriptionUpdateArgs, 'input'>>;
  phoneCreate?: Resolver<ResolversTypes['Phone'], ParentType, ContextType, RequireFields<MutationPhoneCreateArgs, 'input'>>;
  postCreate?: Resolver<ResolversTypes['Post'], ParentType, ContextType, RequireFields<MutationPostCreateArgs, 'input'>>;
  postUpdate?: Resolver<ResolversTypes['Post'], ParentType, ContextType, RequireFields<MutationPostUpdateArgs, 'input'>>;
  profileEmailUpdate?: Resolver<ResolversTypes['User'], ParentType, ContextType, RequireFields<MutationProfileEmailUpdateArgs, 'input'>>;
  profileUpdate?: Resolver<ResolversTypes['User'], ParentType, ContextType, RequireFields<MutationProfileUpdateArgs, 'input'>>;
  projectCreate?: Resolver<ResolversTypes['Project'], ParentType, ContextType, RequireFields<MutationProjectCreateArgs, 'input'>>;
  projectUpdate?: Resolver<ResolversTypes['Project'], ParentType, ContextType, RequireFields<MutationProjectUpdateArgs, 'input'>>;
  projectVersionCreate?: Resolver<ResolversTypes['ProjectVersion'], ParentType, ContextType, RequireFields<MutationProjectVersionCreateArgs, 'input'>>;
  projectVersionDirectoryCreate?: Resolver<ResolversTypes['ProjectVersionDirectory'], ParentType, ContextType, RequireFields<MutationProjectVersionDirectoryCreateArgs, 'input'>>;
  projectVersionDirectoryUpdate?: Resolver<ResolversTypes['ProjectVersionDirectory'], ParentType, ContextType, RequireFields<MutationProjectVersionDirectoryUpdateArgs, 'input'>>;
  projectVersionUpdate?: Resolver<ResolversTypes['ProjectVersion'], ParentType, ContextType, RequireFields<MutationProjectVersionUpdateArgs, 'input'>>;
  pullRequestAccept?: Resolver<ResolversTypes['PullRequest'], ParentType, ContextType, RequireFields<MutationPullRequestAcceptArgs, 'input'>>;
  pullRequestCreate?: Resolver<ResolversTypes['PullRequest'], ParentType, ContextType, RequireFields<MutationPullRequestCreateArgs, 'input'>>;
  pullRequestReject?: Resolver<ResolversTypes['PullRequest'], ParentType, ContextType, RequireFields<MutationPullRequestRejectArgs, 'input'>>;
  pullRequestUpdate?: Resolver<ResolversTypes['PullRequest'], ParentType, ContextType, RequireFields<MutationPullRequestUpdateArgs, 'input'>>;
  pushDeviceCreate?: Resolver<ResolversTypes['PushDevice'], ParentType, ContextType, RequireFields<MutationPushDeviceCreateArgs, 'input'>>;
  pushDeviceTest?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationPushDeviceTestArgs, 'input'>>;
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
  react?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationReactArgs, 'input'>>;
  regenerateResponse?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationRegenerateResponseArgs, 'input'>>;
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
  runProjectCreate?: Resolver<ResolversTypes['RunProject'], ParentType, ContextType, RequireFields<MutationRunProjectCreateArgs, 'input'>>;
  runProjectDeleteAll?: Resolver<ResolversTypes['Count'], ParentType, ContextType>;
  runProjectUpdate?: Resolver<ResolversTypes['RunProject'], ParentType, ContextType, RequireFields<MutationRunProjectUpdateArgs, 'input'>>;
  runRoutineCreate?: Resolver<ResolversTypes['RunRoutine'], ParentType, ContextType, RequireFields<MutationRunRoutineCreateArgs, 'input'>>;
  runRoutineDeleteAll?: Resolver<ResolversTypes['Count'], ParentType, ContextType>;
  runRoutineUpdate?: Resolver<ResolversTypes['RunRoutine'], ParentType, ContextType, RequireFields<MutationRunRoutineUpdateArgs, 'input'>>;
  scheduleCreate?: Resolver<ResolversTypes['Schedule'], ParentType, ContextType, RequireFields<MutationScheduleCreateArgs, 'input'>>;
  scheduleExceptionCreate?: Resolver<ResolversTypes['ScheduleException'], ParentType, ContextType, RequireFields<MutationScheduleExceptionCreateArgs, 'input'>>;
  scheduleExceptionUpdate?: Resolver<ResolversTypes['ScheduleException'], ParentType, ContextType, RequireFields<MutationScheduleExceptionUpdateArgs, 'input'>>;
  scheduleRecurrenceCreate?: Resolver<ResolversTypes['ScheduleRecurrence'], ParentType, ContextType, RequireFields<MutationScheduleRecurrenceCreateArgs, 'input'>>;
  scheduleRecurrenceUpdate?: Resolver<ResolversTypes['ScheduleRecurrence'], ParentType, ContextType, RequireFields<MutationScheduleRecurrenceUpdateArgs, 'input'>>;
  scheduleUpdate?: Resolver<ResolversTypes['Schedule'], ParentType, ContextType, RequireFields<MutationScheduleUpdateArgs, 'input'>>;
  sendVerificationEmail?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationSendVerificationEmailArgs, 'input'>>;
  sendVerificationText?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationSendVerificationTextArgs, 'input'>>;
  setActiveFocusMode?: Resolver<ResolversTypes['ActiveFocusMode'], ParentType, ContextType, RequireFields<MutationSetActiveFocusModeArgs, 'input'>>;
  standardCreate?: Resolver<ResolversTypes['Standard'], ParentType, ContextType, RequireFields<MutationStandardCreateArgs, 'input'>>;
  standardUpdate?: Resolver<ResolversTypes['Standard'], ParentType, ContextType, RequireFields<MutationStandardUpdateArgs, 'input'>>;
  standardVersionCreate?: Resolver<ResolversTypes['StandardVersion'], ParentType, ContextType, RequireFields<MutationStandardVersionCreateArgs, 'input'>>;
  standardVersionUpdate?: Resolver<ResolversTypes['StandardVersion'], ParentType, ContextType, RequireFields<MutationStandardVersionUpdateArgs, 'input'>>;
  startTask?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationStartTaskArgs, 'input'>>;
  switchCurrentAccount?: Resolver<ResolversTypes['Session'], ParentType, ContextType, RequireFields<MutationSwitchCurrentAccountArgs, 'input'>>;
  tagCreate?: Resolver<ResolversTypes['Tag'], ParentType, ContextType, RequireFields<MutationTagCreateArgs, 'input'>>;
  tagUpdate?: Resolver<ResolversTypes['Tag'], ParentType, ContextType, RequireFields<MutationTagUpdateArgs, 'input'>>;
  teamCreate?: Resolver<ResolversTypes['Team'], ParentType, ContextType, RequireFields<MutationTeamCreateArgs, 'input'>>;
  teamUpdate?: Resolver<ResolversTypes['Team'], ParentType, ContextType, RequireFields<MutationTeamUpdateArgs, 'input'>>;
  transferAccept?: Resolver<ResolversTypes['Transfer'], ParentType, ContextType, RequireFields<MutationTransferAcceptArgs, 'input'>>;
  transferCancel?: Resolver<ResolversTypes['Transfer'], ParentType, ContextType, RequireFields<MutationTransferCancelArgs, 'input'>>;
  transferDeny?: Resolver<ResolversTypes['Transfer'], ParentType, ContextType, RequireFields<MutationTransferDenyArgs, 'input'>>;
  transferRequestReceive?: Resolver<ResolversTypes['Transfer'], ParentType, ContextType, RequireFields<MutationTransferRequestReceiveArgs, 'input'>>;
  transferRequestSend?: Resolver<ResolversTypes['Transfer'], ParentType, ContextType, RequireFields<MutationTransferRequestSendArgs, 'input'>>;
  transferUpdate?: Resolver<ResolversTypes['Transfer'], ParentType, ContextType, RequireFields<MutationTransferUpdateArgs, 'input'>>;
  userDeleteOne?: Resolver<ResolversTypes['Session'], ParentType, ContextType, RequireFields<MutationUserDeleteOneArgs, 'input'>>;
  validateSession?: Resolver<ResolversTypes['Session'], ParentType, ContextType, RequireFields<MutationValidateSessionArgs, 'input'>>;
  validateVerificationText?: Resolver<ResolversTypes['Success'], ParentType, ContextType, RequireFields<MutationValidateVerificationTextArgs, 'input'>>;
  walletComplete?: Resolver<ResolversTypes['WalletComplete'], ParentType, ContextType, RequireFields<MutationWalletCompleteArgs, 'input'>>;
  walletInit?: Resolver<ResolversTypes['String'], ParentType, ContextType, RequireFields<MutationWalletInitArgs, 'input'>>;
  walletUpdate?: Resolver<ResolversTypes['Wallet'], ParentType, ContextType, RequireFields<MutationWalletUpdateArgs, 'input'>>;
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
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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

export type NotePageResolvers<ContextType = any, ParentType extends ResolversParentTypes['NotePage'] = ResolversParentTypes['NotePage']> = {
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  pageIndex?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  text?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
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
  pages?: Resolver<Array<ResolversTypes['NotePage']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type NoteYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['NoteYou'] = ResolversParentTypes['NoteYou']> = {
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReact?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canTransfer?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isViewed?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  reaction?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
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
  includedEmails?: Resolver<Maybe<Array<ResolversTypes['Email']>>, ParentType, ContextType>;
  includedPush?: Resolver<Maybe<Array<ResolversTypes['PushDevice']>>, ParentType, ContextType>;
  includedSms?: Resolver<Maybe<Array<ResolversTypes['Phone']>>, ParentType, ContextType>;
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
  context?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
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

export type OwnerResolvers<ContextType = any, ParentType extends ResolversParentTypes['Owner'] = ResolversParentTypes['Owner']> = {
  __resolveType: TypeResolveFn<'Team' | 'User', ParentType, ContextType>;
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
  checkoutId?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  currency?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  description?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  paymentMethod?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  paymentType?: Resolver<ResolversTypes['PaymentType'], ParentType, ContextType>;
  status?: Resolver<ResolversTypes['PaymentStatus'], ParentType, ContextType>;
  team?: Resolver<ResolversTypes['Team'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  user?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PaymentEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['PaymentEdge'] = ResolversParentTypes['PaymentEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Payment'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PaymentSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['PaymentSearchResult'] = ResolversParentTypes['PaymentSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['PaymentEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PhoneResolvers<ContextType = any, ParentType extends ResolversParentTypes['Phone'] = ResolversParentTypes['Phone']> = {
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  phoneNumber?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  verified?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PopularResolvers<ContextType = any, ParentType extends ResolversParentTypes['Popular'] = ResolversParentTypes['Popular']> = {
  __resolveType: TypeResolveFn<'Api' | 'Code' | 'Note' | 'Project' | 'Question' | 'Routine' | 'Standard' | 'Team' | 'User', ParentType, ContextType>;
};

export type PopularEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['PopularEdge'] = ResolversParentTypes['PopularEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Popular'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PopularPageInfoResolvers<ContextType = any, ParentType extends ResolversParentTypes['PopularPageInfo'] = ResolversParentTypes['PopularPageInfo']> = {
  endCursorApi?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  endCursorCode?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  endCursorNote?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  endCursorProject?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  endCursorQuestion?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  endCursorRoutine?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  endCursorStandard?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  endCursorTeam?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  endCursorUser?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  hasNextPage?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PopularSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['PopularSearchResult'] = ResolversParentTypes['PopularSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['PopularEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PopularPageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PostResolvers<ContextType = any, ParentType extends ResolversParentTypes['Post'] = ResolversParentTypes['Post']> = {
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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
  credits?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  customPlan?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  enabledAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  expiresAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isActive?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectResolvers<ContextType = any, ParentType extends ResolversParentTypes['Project'] = ResolversParentTypes['Project']> = {
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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

export type ProjectOrTeamResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectOrTeam'] = ResolversParentTypes['ProjectOrTeam']> = {
  __resolveType: TypeResolveFn<'Project' | 'Team', ParentType, ContextType>;
};

export type ProjectOrTeamEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectOrTeamEdge'] = ResolversParentTypes['ProjectOrTeamEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['ProjectOrTeam'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectOrTeamPageInfoResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectOrTeamPageInfo'] = ResolversParentTypes['ProjectOrTeamPageInfo']> = {
  endCursorProject?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  endCursorTeam?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  hasNextPage?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectOrTeamSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectOrTeamSearchResult'] = ResolversParentTypes['ProjectOrTeamSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ProjectOrTeamEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['ProjectOrTeamPageInfo'], ParentType, ContextType>;
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

export type ProjectVersionContentsSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectVersionContentsSearchResult'] = ResolversParentTypes['ProjectVersionContentsSearchResult']> = {
  apiVersion?: Resolver<Maybe<ResolversTypes['ApiVersion']>, ParentType, ContextType>;
  codeVersion?: Resolver<Maybe<ResolversTypes['CodeVersion']>, ParentType, ContextType>;
  directory?: Resolver<Maybe<ResolversTypes['ProjectVersionDirectory']>, ParentType, ContextType>;
  noteVersion?: Resolver<Maybe<ResolversTypes['NoteVersion']>, ParentType, ContextType>;
  projectVersion?: Resolver<Maybe<ResolversTypes['ProjectVersion']>, ParentType, ContextType>;
  routineVersion?: Resolver<Maybe<ResolversTypes['RoutineVersion']>, ParentType, ContextType>;
  standardVersion?: Resolver<Maybe<ResolversTypes['StandardVersion']>, ParentType, ContextType>;
  team?: Resolver<Maybe<ResolversTypes['Team']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectVersionDirectoryResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectVersionDirectory'] = ResolversParentTypes['ProjectVersionDirectory']> = {
  childApiVersions?: Resolver<Array<ResolversTypes['ApiVersion']>, ParentType, ContextType>;
  childCodeVersions?: Resolver<Array<ResolversTypes['CodeVersion']>, ParentType, ContextType>;
  childNoteVersions?: Resolver<Array<ResolversTypes['NoteVersion']>, ParentType, ContextType>;
  childOrder?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  childProjectVersions?: Resolver<Array<ResolversTypes['ProjectVersion']>, ParentType, ContextType>;
  childRoutineVersions?: Resolver<Array<ResolversTypes['RoutineVersion']>, ParentType, ContextType>;
  childStandardVersions?: Resolver<Array<ResolversTypes['StandardVersion']>, ParentType, ContextType>;
  childTeams?: Resolver<Array<ResolversTypes['Team']>, ParentType, ContextType>;
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

export type ProjectVersionDirectoryEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectVersionDirectoryEdge'] = ResolversParentTypes['ProjectVersionDirectoryEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['ProjectVersionDirectory'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ProjectVersionDirectorySearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ProjectVersionDirectorySearchResult'] = ResolversParentTypes['ProjectVersionDirectorySearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ProjectVersionDirectoryEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
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
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReact?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canTransfer?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isViewed?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  reaction?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
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
  translations?: Resolver<Array<ResolversTypes['CommentTranslation']>, ParentType, ContextType>;
  translationsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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
  __resolveType: TypeResolveFn<'ApiVersion' | 'CodeVersion' | 'NoteVersion' | 'ProjectVersion' | 'RoutineVersion' | 'StandardVersion', ParentType, ContextType>;
};

export type PullRequestSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['PullRequestSearchResult'] = ResolversParentTypes['PullRequestSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['PullRequestEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PullRequestToResolvers<ContextType = any, ParentType extends ResolversParentTypes['PullRequestTo'] = ResolversParentTypes['PullRequestTo']> = {
  __resolveType: TypeResolveFn<'Api' | 'Code' | 'Note' | 'Project' | 'Routine' | 'Standard', ParentType, ContextType>;
};

export type PullRequestTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['PullRequestTranslation'] = ResolversParentTypes['PullRequestTranslation']> = {
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  text?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PullRequestYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['PullRequestYou'] = ResolversParentTypes['PullRequestYou']> = {
  canComment?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReport?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type PushDeviceResolvers<ContextType = any, ParentType extends ResolversParentTypes['PushDevice'] = ResolversParentTypes['PushDevice']> = {
  deviceId?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  expires?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  name?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QueryResolvers<ContextType = any, ParentType extends ResolversParentTypes['Query'] = ResolversParentTypes['Query']> = {
  _empty?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  api?: Resolver<Maybe<ResolversTypes['Api']>, ParentType, ContextType, RequireFields<QueryApiArgs, 'input'>>;
  apiVersion?: Resolver<Maybe<ResolversTypes['ApiVersion']>, ParentType, ContextType, RequireFields<QueryApiVersionArgs, 'input'>>;
  apiVersions?: Resolver<ResolversTypes['ApiVersionSearchResult'], ParentType, ContextType, RequireFields<QueryApiVersionsArgs, 'input'>>;
  apis?: Resolver<ResolversTypes['ApiSearchResult'], ParentType, ContextType, RequireFields<QueryApisArgs, 'input'>>;
  awards?: Resolver<ResolversTypes['AwardSearchResult'], ParentType, ContextType, RequireFields<QueryAwardsArgs, 'input'>>;
  bookmark?: Resolver<Maybe<ResolversTypes['Bookmark']>, ParentType, ContextType, RequireFields<QueryBookmarkArgs, 'input'>>;
  bookmarkList?: Resolver<Maybe<ResolversTypes['BookmarkList']>, ParentType, ContextType, RequireFields<QueryBookmarkListArgs, 'input'>>;
  bookmarkLists?: Resolver<ResolversTypes['BookmarkListSearchResult'], ParentType, ContextType, RequireFields<QueryBookmarkListsArgs, 'input'>>;
  bookmarks?: Resolver<ResolversTypes['BookmarkSearchResult'], ParentType, ContextType, RequireFields<QueryBookmarksArgs, 'input'>>;
  chat?: Resolver<Maybe<ResolversTypes['Chat']>, ParentType, ContextType, RequireFields<QueryChatArgs, 'input'>>;
  chatInvite?: Resolver<Maybe<ResolversTypes['ChatInvite']>, ParentType, ContextType, RequireFields<QueryChatInviteArgs, 'input'>>;
  chatInvites?: Resolver<ResolversTypes['ChatInviteSearchResult'], ParentType, ContextType, RequireFields<QueryChatInvitesArgs, 'input'>>;
  chatMessage?: Resolver<Maybe<ResolversTypes['ChatMessage']>, ParentType, ContextType, RequireFields<QueryChatMessageArgs, 'input'>>;
  chatMessageTree?: Resolver<ResolversTypes['ChatMessageSearchTreeResult'], ParentType, ContextType, RequireFields<QueryChatMessageTreeArgs, 'input'>>;
  chatMessages?: Resolver<ResolversTypes['ChatMessageSearchResult'], ParentType, ContextType, RequireFields<QueryChatMessagesArgs, 'input'>>;
  chatParticipant?: Resolver<Maybe<ResolversTypes['ChatParticipant']>, ParentType, ContextType, RequireFields<QueryChatParticipantArgs, 'input'>>;
  chatParticipants?: Resolver<ResolversTypes['ChatParticipantSearchResult'], ParentType, ContextType, RequireFields<QueryChatParticipantsArgs, 'input'>>;
  chats?: Resolver<ResolversTypes['ChatSearchResult'], ParentType, ContextType, RequireFields<QueryChatsArgs, 'input'>>;
  code?: Resolver<Maybe<ResolversTypes['Code']>, ParentType, ContextType, RequireFields<QueryCodeArgs, 'input'>>;
  codeVersion?: Resolver<Maybe<ResolversTypes['CodeVersion']>, ParentType, ContextType, RequireFields<QueryCodeVersionArgs, 'input'>>;
  codeVersions?: Resolver<ResolversTypes['CodeVersionSearchResult'], ParentType, ContextType, RequireFields<QueryCodeVersionsArgs, 'input'>>;
  codes?: Resolver<ResolversTypes['CodeSearchResult'], ParentType, ContextType, RequireFields<QueryCodesArgs, 'input'>>;
  comment?: Resolver<Maybe<ResolversTypes['Comment']>, ParentType, ContextType, RequireFields<QueryCommentArgs, 'input'>>;
  comments?: Resolver<ResolversTypes['CommentSearchResult'], ParentType, ContextType, RequireFields<QueryCommentsArgs, 'input'>>;
  focusMode?: Resolver<Maybe<ResolversTypes['FocusMode']>, ParentType, ContextType, RequireFields<QueryFocusModeArgs, 'input'>>;
  focusModes?: Resolver<ResolversTypes['FocusModeSearchResult'], ParentType, ContextType, RequireFields<QueryFocusModesArgs, 'input'>>;
  home?: Resolver<ResolversTypes['HomeResult'], ParentType, ContextType, RequireFields<QueryHomeArgs, 'input'>>;
  issue?: Resolver<Maybe<ResolversTypes['Issue']>, ParentType, ContextType, RequireFields<QueryIssueArgs, 'input'>>;
  issues?: Resolver<ResolversTypes['IssueSearchResult'], ParentType, ContextType, RequireFields<QueryIssuesArgs, 'input'>>;
  label?: Resolver<Maybe<ResolversTypes['Label']>, ParentType, ContextType, RequireFields<QueryLabelArgs, 'input'>>;
  labels?: Resolver<ResolversTypes['LabelSearchResult'], ParentType, ContextType, RequireFields<QueryLabelsArgs, 'input'>>;
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
  notificationSettings?: Resolver<ResolversTypes['NotificationSettings'], ParentType, ContextType>;
  notificationSubscription?: Resolver<Maybe<ResolversTypes['NotificationSubscription']>, ParentType, ContextType, RequireFields<QueryNotificationSubscriptionArgs, 'input'>>;
  notificationSubscriptions?: Resolver<ResolversTypes['NotificationSubscriptionSearchResult'], ParentType, ContextType, RequireFields<QueryNotificationSubscriptionsArgs, 'input'>>;
  notifications?: Resolver<ResolversTypes['NotificationSearchResult'], ParentType, ContextType, RequireFields<QueryNotificationsArgs, 'input'>>;
  payment?: Resolver<Maybe<ResolversTypes['Payment']>, ParentType, ContextType, RequireFields<QueryPaymentArgs, 'input'>>;
  payments?: Resolver<ResolversTypes['PaymentSearchResult'], ParentType, ContextType, RequireFields<QueryPaymentsArgs, 'input'>>;
  popular?: Resolver<ResolversTypes['PopularSearchResult'], ParentType, ContextType, RequireFields<QueryPopularArgs, 'input'>>;
  post?: Resolver<Maybe<ResolversTypes['Post']>, ParentType, ContextType, RequireFields<QueryPostArgs, 'input'>>;
  posts?: Resolver<ResolversTypes['PostSearchResult'], ParentType, ContextType, RequireFields<QueryPostsArgs, 'input'>>;
  profile?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
  project?: Resolver<Maybe<ResolversTypes['Project']>, ParentType, ContextType, RequireFields<QueryProjectArgs, 'input'>>;
  projectOrRoutines?: Resolver<ResolversTypes['ProjectOrRoutineSearchResult'], ParentType, ContextType, RequireFields<QueryProjectOrRoutinesArgs, 'input'>>;
  projectOrTeams?: Resolver<ResolversTypes['ProjectOrTeamSearchResult'], ParentType, ContextType, RequireFields<QueryProjectOrTeamsArgs, 'input'>>;
  projectVersion?: Resolver<Maybe<ResolversTypes['ProjectVersion']>, ParentType, ContextType, RequireFields<QueryProjectVersionArgs, 'input'>>;
  projectVersionContents?: Resolver<ResolversTypes['ProjectVersionContentsSearchResult'], ParentType, ContextType, RequireFields<QueryProjectVersionContentsArgs, 'input'>>;
  projectVersionDirectories?: Resolver<ResolversTypes['ProjectVersionDirectorySearchResult'], ParentType, ContextType, RequireFields<QueryProjectVersionDirectoriesArgs, 'input'>>;
  projectVersionDirectory?: Resolver<Maybe<ResolversTypes['ProjectVersionDirectory']>, ParentType, ContextType, RequireFields<QueryProjectVersionDirectoryArgs, 'input'>>;
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
  reactions?: Resolver<ResolversTypes['ReactionSearchResult'], ParentType, ContextType, RequireFields<QueryReactionsArgs, 'input'>>;
  reminder?: Resolver<Maybe<ResolversTypes['Reminder']>, ParentType, ContextType, RequireFields<QueryReminderArgs, 'input'>>;
  reminders?: Resolver<ResolversTypes['ReminderSearchResult'], ParentType, ContextType, RequireFields<QueryRemindersArgs, 'input'>>;
  report?: Resolver<Maybe<ResolversTypes['Report']>, ParentType, ContextType, RequireFields<QueryReportArgs, 'input'>>;
  reportResponse?: Resolver<Maybe<ResolversTypes['ReportResponse']>, ParentType, ContextType, RequireFields<QueryReportResponseArgs, 'input'>>;
  reportResponses?: Resolver<ResolversTypes['ReportResponseSearchResult'], ParentType, ContextType, RequireFields<QueryReportResponsesArgs, 'input'>>;
  reports?: Resolver<ResolversTypes['ReportSearchResult'], ParentType, ContextType, RequireFields<QueryReportsArgs, 'input'>>;
  reputationHistories?: Resolver<ResolversTypes['ReputationHistorySearchResult'], ParentType, ContextType, RequireFields<QueryReputationHistoriesArgs, 'input'>>;
  reputationHistory?: Resolver<Maybe<ResolversTypes['ReputationHistory']>, ParentType, ContextType, RequireFields<QueryReputationHistoryArgs, 'input'>>;
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
  runProjects?: Resolver<ResolversTypes['RunProjectSearchResult'], ParentType, ContextType, RequireFields<QueryRunProjectsArgs, 'input'>>;
  runRoutine?: Resolver<Maybe<ResolversTypes['RunRoutine']>, ParentType, ContextType, RequireFields<QueryRunRoutineArgs, 'input'>>;
  runRoutineInputs?: Resolver<ResolversTypes['RunRoutineInputSearchResult'], ParentType, ContextType, RequireFields<QueryRunRoutineInputsArgs, 'input'>>;
  runRoutineOutputs?: Resolver<ResolversTypes['RunRoutineOutputSearchResult'], ParentType, ContextType, RequireFields<QueryRunRoutineOutputsArgs, 'input'>>;
  runRoutines?: Resolver<ResolversTypes['RunRoutineSearchResult'], ParentType, ContextType, RequireFields<QueryRunRoutinesArgs, 'input'>>;
  schedule?: Resolver<Maybe<ResolversTypes['Schedule']>, ParentType, ContextType, RequireFields<QueryScheduleArgs, 'input'>>;
  scheduleException?: Resolver<Maybe<ResolversTypes['ScheduleException']>, ParentType, ContextType, RequireFields<QueryScheduleExceptionArgs, 'input'>>;
  scheduleExceptions?: Resolver<ResolversTypes['ScheduleExceptionSearchResult'], ParentType, ContextType, RequireFields<QueryScheduleExceptionsArgs, 'input'>>;
  scheduleRecurrence?: Resolver<Maybe<ResolversTypes['ScheduleRecurrence']>, ParentType, ContextType, RequireFields<QueryScheduleRecurrenceArgs, 'input'>>;
  scheduleRecurrences?: Resolver<ResolversTypes['ScheduleRecurrenceSearchResult'], ParentType, ContextType, RequireFields<QueryScheduleRecurrencesArgs, 'input'>>;
  schedules?: Resolver<ResolversTypes['ScheduleSearchResult'], ParentType, ContextType, RequireFields<QuerySchedulesArgs, 'input'>>;
  standard?: Resolver<Maybe<ResolversTypes['Standard']>, ParentType, ContextType, RequireFields<QueryStandardArgs, 'input'>>;
  standardVersion?: Resolver<Maybe<ResolversTypes['StandardVersion']>, ParentType, ContextType, RequireFields<QueryStandardVersionArgs, 'input'>>;
  standardVersions?: Resolver<ResolversTypes['StandardVersionSearchResult'], ParentType, ContextType, RequireFields<QueryStandardVersionsArgs, 'input'>>;
  standards?: Resolver<ResolversTypes['StandardSearchResult'], ParentType, ContextType, RequireFields<QueryStandardsArgs, 'input'>>;
  statsApi?: Resolver<ResolversTypes['StatsApiSearchResult'], ParentType, ContextType, RequireFields<QueryStatsApiArgs, 'input'>>;
  statsCode?: Resolver<ResolversTypes['StatsCodeSearchResult'], ParentType, ContextType, RequireFields<QueryStatsCodeArgs, 'input'>>;
  statsProject?: Resolver<ResolversTypes['StatsProjectSearchResult'], ParentType, ContextType, RequireFields<QueryStatsProjectArgs, 'input'>>;
  statsQuiz?: Resolver<ResolversTypes['StatsQuizSearchResult'], ParentType, ContextType, RequireFields<QueryStatsQuizArgs, 'input'>>;
  statsRoutine?: Resolver<ResolversTypes['StatsRoutineSearchResult'], ParentType, ContextType, RequireFields<QueryStatsRoutineArgs, 'input'>>;
  statsSite?: Resolver<ResolversTypes['StatsSiteSearchResult'], ParentType, ContextType, RequireFields<QueryStatsSiteArgs, 'input'>>;
  statsStandard?: Resolver<ResolversTypes['StatsStandardSearchResult'], ParentType, ContextType, RequireFields<QueryStatsStandardArgs, 'input'>>;
  statsTeam?: Resolver<ResolversTypes['StatsTeamSearchResult'], ParentType, ContextType, RequireFields<QueryStatsTeamArgs, 'input'>>;
  statsUser?: Resolver<ResolversTypes['StatsUserSearchResult'], ParentType, ContextType, RequireFields<QueryStatsUserArgs, 'input'>>;
  tag?: Resolver<Maybe<ResolversTypes['Tag']>, ParentType, ContextType, RequireFields<QueryTagArgs, 'input'>>;
  tags?: Resolver<ResolversTypes['TagSearchResult'], ParentType, ContextType, RequireFields<QueryTagsArgs, 'input'>>;
  team?: Resolver<Maybe<ResolversTypes['Team']>, ParentType, ContextType, RequireFields<QueryTeamArgs, 'input'>>;
  teams?: Resolver<ResolversTypes['TeamSearchResult'], ParentType, ContextType, RequireFields<QueryTeamsArgs, 'input'>>;
  transfer?: Resolver<Maybe<ResolversTypes['Transfer']>, ParentType, ContextType, RequireFields<QueryTransferArgs, 'input'>>;
  transfers?: Resolver<ResolversTypes['TransferSearchResult'], ParentType, ContextType, RequireFields<QueryTransfersArgs, 'input'>>;
  translate?: Resolver<ResolversTypes['Translate'], ParentType, ContextType, RequireFields<QueryTranslateArgs, 'input'>>;
  user?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType, RequireFields<QueryUserArgs, 'input'>>;
  users?: Resolver<ResolversTypes['UserSearchResult'], ParentType, ContextType, RequireFields<QueryUsersArgs, 'input'>>;
  views?: Resolver<ResolversTypes['ViewSearchResult'], ParentType, ContextType, RequireFields<QueryViewsArgs, 'input'>>;
};

export type QuestionResolvers<ContextType = any, ParentType extends ResolversParentTypes['Question'] = ResolversParentTypes['Question']> = {
  answers?: Resolver<Array<ResolversTypes['QuestionAnswer']>, ParentType, ContextType>;
  answersCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  comments?: Resolver<Array<ResolversTypes['Comment']>, ParentType, ContextType>;
  commentsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  createdBy?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  forObject?: Resolver<Maybe<ResolversTypes['QuestionFor']>, ParentType, ContextType>;
  hasAcceptedAnswer?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  reports?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  reportsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  tags?: Resolver<Array<ResolversTypes['Tag']>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['QuestionTranslation']>, ParentType, ContextType>;
  translationsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  you?: Resolver<ResolversTypes['QuestionYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuestionAnswerResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuestionAnswer'] = ResolversParentTypes['QuestionAnswer']> = {
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  comments?: Resolver<Array<ResolversTypes['Comment']>, ParentType, ContextType>;
  commentsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  createdBy?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isAccepted?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  question?: Resolver<ResolversTypes['Question'], ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  text?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuestionEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuestionEdge'] = ResolversParentTypes['QuestionEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Question'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuestionForResolvers<ContextType = any, ParentType extends ResolversParentTypes['QuestionFor'] = ResolversParentTypes['QuestionFor']> = {
  __resolveType: TypeResolveFn<'Api' | 'Code' | 'Note' | 'Project' | 'Routine' | 'Standard' | 'Team', ParentType, ContextType>;
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
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReact?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  reaction?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QuizResolvers<ContextType = any, ParentType extends ResolversParentTypes['Quiz'] = ResolversParentTypes['Quiz']> = {
  attempts?: Resolver<Array<ResolversTypes['QuizAttempt']>, ParentType, ContextType>;
  attemptsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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
  stats?: Resolver<Array<ResolversTypes['StatsQuiz']>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['QuizTranslation']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
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
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReact?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  hasCompleted?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  reaction?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReactionResolvers<ContextType = any, ParentType extends ResolversParentTypes['Reaction'] = ResolversParentTypes['Reaction']> = {
  by?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
  emoji?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  to?: Resolver<ResolversTypes['ReactionTo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReactionEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReactionEdge'] = ResolversParentTypes['ReactionEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Reaction'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReactionSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReactionSearchResult'] = ResolversParentTypes['ReactionSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ReactionEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReactionSummaryResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReactionSummary'] = ResolversParentTypes['ReactionSummary']> = {
  count?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  emoji?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReactionToResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReactionTo'] = ResolversParentTypes['ReactionTo']> = {
  __resolveType: TypeResolveFn<'Api' | 'ChatMessage' | 'Code' | 'Comment' | 'Issue' | 'Note' | 'Post' | 'Project' | 'Question' | 'QuestionAnswer' | 'Quiz' | 'Routine' | 'Standard', ParentType, ContextType>;
};

export type ReminderResolvers<ContextType = any, ParentType extends ResolversParentTypes['Reminder'] = ResolversParentTypes['Reminder']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  dueDate?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  index?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  isComplete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
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
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  description?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  dueDate?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  index?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  isComplete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  reminder?: Resolver<ResolversTypes['Reminder'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ReminderListResolvers<ContextType = any, ParentType extends ResolversParentTypes['ReminderList'] = ResolversParentTypes['ReminderList']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  focusMode?: Resolver<Maybe<ResolversTypes['FocusMode']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  reminders?: Resolver<Array<ResolversTypes['Reminder']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
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
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  reason?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  responses?: Resolver<Array<ResolversTypes['ReportResponse']>, ParentType, ContextType>;
  responsesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  status?: Resolver<ResolversTypes['ReportStatus'], ParentType, ContextType>;
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
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  listFor?: Resolver<ResolversTypes['ResourceListOn'], ParentType, ContextType>;
  resources?: Resolver<Array<ResolversTypes['Resource']>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['ResourceListTranslation']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ResourceListEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ResourceListEdge'] = ResolversParentTypes['ResourceListEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['ResourceList'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ResourceListOnResolvers<ContextType = any, ParentType extends ResolversParentTypes['ResourceListOn'] = ResolversParentTypes['ResourceListOn']> = {
  __resolveType: TypeResolveFn<'ApiVersion' | 'CodeVersion' | 'FocusMode' | 'Post' | 'ProjectVersion' | 'RoutineVersion' | 'StandardVersion' | 'Team', ParentType, ContextType>;
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
  permissions?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  team?: Resolver<ResolversTypes['Team'], ParentType, ContextType>;
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
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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
  apiVersion?: Resolver<Maybe<ResolversTypes['ApiVersion']>, ParentType, ContextType>;
  codeVersion?: Resolver<Maybe<ResolversTypes['CodeVersion']>, ParentType, ContextType>;
  comments?: Resolver<Array<ResolversTypes['Comment']>, ParentType, ContextType>;
  commentsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  completedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  complexity?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  configCallData?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  configFormInput?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  configFormOutput?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
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
  routineType?: Resolver<ResolversTypes['RoutineType'], ParentType, ContextType>;
  simplicity?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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
  instructions?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoutineVersionYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['RoutineVersionYou'] = ResolversParentTypes['RoutineVersionYou']> = {
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canComment?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canCopy?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReact?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReport?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRun?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  runs?: Resolver<Array<ResolversTypes['RunRoutine']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RoutineYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['RoutineYou'] = ResolversParentTypes['RoutineYou']> = {
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canComment?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReact?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canTransfer?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isViewed?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  reaction?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunProjectResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunProject'] = ResolversParentTypes['RunProject']> = {
  completedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  completedComplexity?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  contextSwitches?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  lastStep?: Resolver<Maybe<Array<ResolversTypes['Int']>>, ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  projectVersion?: Resolver<Maybe<ResolversTypes['ProjectVersion']>, ParentType, ContextType>;
  schedule?: Resolver<Maybe<ResolversTypes['Schedule']>, ParentType, ContextType>;
  startedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  status?: Resolver<ResolversTypes['RunStatus'], ParentType, ContextType>;
  steps?: Resolver<Array<ResolversTypes['RunProjectStep']>, ParentType, ContextType>;
  stepsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  team?: Resolver<Maybe<ResolversTypes['Team']>, ParentType, ContextType>;
  timeElapsed?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  user?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
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
  endCursorRunProject?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  endCursorRunRoutine?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  hasNextPage?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunProjectOrRunRoutineSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunProjectOrRunRoutineSearchResult'] = ResolversParentTypes['RunProjectOrRunRoutineSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['RunProjectOrRunRoutineEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['RunProjectOrRunRoutinePageInfo'], ParentType, ContextType>;
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
  runProject?: Resolver<ResolversTypes['RunProject'], ParentType, ContextType>;
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
  lastStep?: Resolver<Maybe<Array<ResolversTypes['Int']>>, ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  outputs?: Resolver<Array<ResolversTypes['RunRoutineOutput']>, ParentType, ContextType>;
  outputsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  routineVersion?: Resolver<Maybe<ResolversTypes['RoutineVersion']>, ParentType, ContextType>;
  runProject?: Resolver<Maybe<ResolversTypes['RunProject']>, ParentType, ContextType>;
  schedule?: Resolver<Maybe<ResolversTypes['Schedule']>, ParentType, ContextType>;
  startedAt?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  status?: Resolver<ResolversTypes['RunStatus'], ParentType, ContextType>;
  steps?: Resolver<Array<ResolversTypes['RunRoutineStep']>, ParentType, ContextType>;
  stepsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  team?: Resolver<Maybe<ResolversTypes['Team']>, ParentType, ContextType>;
  timeElapsed?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  user?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  wasRunAutomatically?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
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

export type RunRoutineOutputResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunRoutineOutput'] = ResolversParentTypes['RunRoutineOutput']> = {
  data?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  output?: Resolver<ResolversTypes['RoutineVersionOutput'], ParentType, ContextType>;
  runRoutine?: Resolver<ResolversTypes['RunRoutine'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunRoutineOutputEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunRoutineOutputEdge'] = ResolversParentTypes['RunRoutineOutputEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['RunRoutineOutput'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type RunRoutineOutputSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['RunRoutineOutputSearchResult'] = ResolversParentTypes['RunRoutineOutputSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['RunRoutineOutputEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
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
  runRoutine?: Resolver<ResolversTypes['RunRoutine'], ParentType, ContextType>;
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

export type ScheduleResolvers<ContextType = any, ParentType extends ResolversParentTypes['Schedule'] = ResolversParentTypes['Schedule']> = {
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  endTime?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  exceptions?: Resolver<Array<ResolversTypes['ScheduleException']>, ParentType, ContextType>;
  focusModes?: Resolver<Array<ResolversTypes['FocusMode']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  labels?: Resolver<Array<ResolversTypes['Label']>, ParentType, ContextType>;
  meetings?: Resolver<Array<ResolversTypes['Meeting']>, ParentType, ContextType>;
  recurrences?: Resolver<Array<ResolversTypes['ScheduleRecurrence']>, ParentType, ContextType>;
  runProjects?: Resolver<Array<ResolversTypes['RunProject']>, ParentType, ContextType>;
  runRoutines?: Resolver<Array<ResolversTypes['RunRoutine']>, ParentType, ContextType>;
  startTime?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  timezone?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ScheduleEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ScheduleEdge'] = ResolversParentTypes['ScheduleEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Schedule'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ScheduleExceptionResolvers<ContextType = any, ParentType extends ResolversParentTypes['ScheduleException'] = ResolversParentTypes['ScheduleException']> = {
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  newEndTime?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  newStartTime?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  originalStartTime?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  schedule?: Resolver<ResolversTypes['Schedule'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ScheduleExceptionEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ScheduleExceptionEdge'] = ResolversParentTypes['ScheduleExceptionEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['ScheduleException'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ScheduleExceptionSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ScheduleExceptionSearchResult'] = ResolversParentTypes['ScheduleExceptionSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ScheduleExceptionEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ScheduleRecurrenceResolvers<ContextType = any, ParentType extends ResolversParentTypes['ScheduleRecurrence'] = ResolversParentTypes['ScheduleRecurrence']> = {
  dayOfMonth?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  dayOfWeek?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  duration?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  endDate?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  interval?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  month?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  recurrenceType?: Resolver<ResolversTypes['ScheduleRecurrenceType'], ParentType, ContextType>;
  schedule?: Resolver<ResolversTypes['Schedule'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ScheduleRecurrenceEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['ScheduleRecurrenceEdge'] = ResolversParentTypes['ScheduleRecurrenceEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['ScheduleRecurrence'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ScheduleRecurrenceSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ScheduleRecurrenceSearchResult'] = ResolversParentTypes['ScheduleRecurrenceSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ScheduleRecurrenceEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ScheduleSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['ScheduleSearchResult'] = ResolversParentTypes['ScheduleSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['ScheduleEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type SessionResolvers<ContextType = any, ParentType extends ResolversParentTypes['Session'] = ResolversParentTypes['Session']> = {
  isLoggedIn?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  timeZone?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  users?: Resolver<Maybe<Array<ResolversTypes['SessionUser']>>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type SessionUserResolvers<ContextType = any, ParentType extends ResolversParentTypes['SessionUser'] = ResolversParentTypes['SessionUser']> = {
  activeFocusMode?: Resolver<Maybe<ResolversTypes['ActiveFocusMode']>, ParentType, ContextType>;
  apisCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  bookmarkLists?: Resolver<Array<ResolversTypes['BookmarkList']>, ParentType, ContextType>;
  codesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  credits?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  focusModes?: Resolver<Array<ResolversTypes['FocusMode']>, ParentType, ContextType>;
  handle?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  hasPremium?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  languages?: Resolver<Array<ResolversTypes['String']>, ParentType, ContextType>;
  membershipsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  name?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  notesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  profileImage?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  projectsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  questionsAskedCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  routinesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  standardsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  theme?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StandardResolvers<ContextType = any, ParentType extends ResolversParentTypes['Standard'] = ResolversParentTypes['Standard']> = {
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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
  owner?: Resolver<Maybe<ResolversTypes['Owner']>, ParentType, ContextType>;
  parent?: Resolver<Maybe<ResolversTypes['StandardVersion']>, ParentType, ContextType>;
  permissions?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  pullRequests?: Resolver<Array<ResolversTypes['PullRequest']>, ParentType, ContextType>;
  pullRequestsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  questions?: Resolver<Array<ResolversTypes['Question']>, ParentType, ContextType>;
  questionsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StandardYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['StandardYou'] = ResolversParentTypes['StandardYou']> = {
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReact?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canTransfer?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isViewed?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  reaction?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsApiResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsApi'] = ResolversParentTypes['StatsApi']> = {
  calls?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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

export type StatsCodeResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsCode'] = ResolversParentTypes['StatsCode']> = {
  calls?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  periodEnd?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodStart?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodType?: Resolver<ResolversTypes['StatPeriodType'], ParentType, ContextType>;
  routineVersions?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsCodeEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsCodeEdge'] = ResolversParentTypes['StatsCodeEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['StatsCode'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsCodeSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsCodeSearchResult'] = ResolversParentTypes['StatsCodeSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['StatsCodeEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsProjectResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsProject'] = ResolversParentTypes['StatsProject']> = {
  apis?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  codes?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  directories?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  notes?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  periodEnd?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodStart?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  periodType?: Resolver<ResolversTypes['StatPeriodType'], ParentType, ContextType>;
  projects?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  routines?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  runCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  runContextSwitchesAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  runsCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  runsStarted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  standards?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  teams?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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
  codeCalls?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  codeCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  codesCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  codesCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
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
  standardCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  standardsCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  standardsCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  teamsCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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

export type StatsStandardResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsStandard'] = ResolversParentTypes['StatsStandard']> = {
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

export type StatsTeamResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsTeam'] = ResolversParentTypes['StatsTeam']> = {
  apis?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  codes?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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
  standards?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsTeamEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsTeamEdge'] = ResolversParentTypes['StatsTeamEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['StatsTeam'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsTeamSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsTeamSearchResult'] = ResolversParentTypes['StatsTeamSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['StatsTeamEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type StatsUserResolvers<ContextType = any, ParentType extends ResolversParentTypes['StatsUser'] = ResolversParentTypes['StatsUser']> = {
  apisCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  codeCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  codesCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  codesCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
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
  standardCompletionTimeAverage?: Resolver<ResolversTypes['Float'], ParentType, ContextType>;
  standardsCompleted?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  standardsCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  teamssCreated?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
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
  __resolveType: TypeResolveFn<'Api' | 'Code' | 'Comment' | 'Issue' | 'Meeting' | 'Note' | 'Project' | 'PullRequest' | 'Question' | 'Quiz' | 'Report' | 'Routine' | 'Schedule' | 'Standard' | 'Team', ParentType, ContextType>;
};

export type SuccessResolvers<ContextType = any, ParentType extends ResolversParentTypes['Success'] = ResolversParentTypes['Success']> = {
  success?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type TagResolvers<ContextType = any, ParentType extends ResolversParentTypes['Tag'] = ResolversParentTypes['Tag']> = {
  apis?: Resolver<Array<ResolversTypes['Api']>, ParentType, ContextType>;
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  codes?: Resolver<Array<ResolversTypes['Code']>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  notes?: Resolver<Array<ResolversTypes['Note']>, ParentType, ContextType>;
  posts?: Resolver<Array<ResolversTypes['Post']>, ParentType, ContextType>;
  projects?: Resolver<Array<ResolversTypes['Project']>, ParentType, ContextType>;
  reports?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  routines?: Resolver<Array<ResolversTypes['Routine']>, ParentType, ContextType>;
  standards?: Resolver<Array<ResolversTypes['Standard']>, ParentType, ContextType>;
  tag?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  teams?: Resolver<Array<ResolversTypes['Team']>, ParentType, ContextType>;
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
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isOwn?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type TeamResolvers<ContextType = any, ParentType extends ResolversParentTypes['Team'] = ResolversParentTypes['Team']> = {
  apis?: Resolver<Array<ResolversTypes['Api']>, ParentType, ContextType>;
  apisCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  bannerImage?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  codes?: Resolver<Array<ResolversTypes['Code']>, ParentType, ContextType>;
  codesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  comments?: Resolver<Array<ResolversTypes['Comment']>, ParentType, ContextType>;
  commentsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  directoryListings?: Resolver<Array<ResolversTypes['ProjectVersionDirectory']>, ParentType, ContextType>;
  forks?: Resolver<Array<ResolversTypes['Team']>, ParentType, ContextType>;
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
  parent?: Resolver<Maybe<ResolversTypes['Team']>, ParentType, ContextType>;
  paymentHistory?: Resolver<Array<ResolversTypes['Payment']>, ParentType, ContextType>;
  permissions?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  posts?: Resolver<Array<ResolversTypes['Post']>, ParentType, ContextType>;
  postsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  premium?: Resolver<Maybe<ResolversTypes['Premium']>, ParentType, ContextType>;
  profileImage?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
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
  standards?: Resolver<Array<ResolversTypes['Standard']>, ParentType, ContextType>;
  standardsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  stats?: Resolver<Array<ResolversTypes['StatsTeam']>, ParentType, ContextType>;
  tags?: Resolver<Array<ResolversTypes['Tag']>, ParentType, ContextType>;
  transfersIncoming?: Resolver<Array<ResolversTypes['Transfer']>, ParentType, ContextType>;
  transfersOutgoing?: Resolver<Array<ResolversTypes['Transfer']>, ParentType, ContextType>;
  translatedName?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['TeamTranslation']>, ParentType, ContextType>;
  translationsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  updated_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  views?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  wallets?: Resolver<Array<ResolversTypes['Wallet']>, ParentType, ContextType>;
  you?: Resolver<ResolversTypes['TeamYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type TeamEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['TeamEdge'] = ResolversParentTypes['TeamEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['Team'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type TeamSearchResultResolvers<ContextType = any, ParentType extends ResolversParentTypes['TeamSearchResult'] = ResolversParentTypes['TeamSearchResult']> = {
  edges?: Resolver<Array<ResolversTypes['TeamEdge']>, ParentType, ContextType>;
  pageInfo?: Resolver<ResolversTypes['PageInfo'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type TeamTranslationResolvers<ContextType = any, ParentType extends ResolversParentTypes['TeamTranslation'] = ResolversParentTypes['TeamTranslation']> = {
  bio?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  language?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type TeamYouResolvers<ContextType = any, ParentType extends ResolversParentTypes['TeamYou'] = ResolversParentTypes['TeamYou']> = {
  canAddMembers?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canBookmark?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canDelete?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canRead?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canReport?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  canUpdate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBookmarked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isViewed?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  yourMembership?: Resolver<Maybe<ResolversTypes['Member']>, ParentType, ContextType>;
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
  __resolveType: TypeResolveFn<'Api' | 'Code' | 'Note' | 'Project' | 'Routine' | 'Standard', ParentType, ContextType>;
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
  apisCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  apisCreated?: Resolver<Maybe<Array<ResolversTypes['Api']>>, ParentType, ContextType>;
  awards?: Resolver<Maybe<Array<ResolversTypes['Award']>>, ParentType, ContextType>;
  bannerImage?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  bookmarked?: Resolver<Maybe<Array<ResolversTypes['Bookmark']>>, ParentType, ContextType>;
  bookmarkedBy?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  bookmarks?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  botSettings?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  codes?: Resolver<Maybe<Array<ResolversTypes['Code']>>, ParentType, ContextType>;
  codesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  codesCreated?: Resolver<Maybe<Array<ResolversTypes['Code']>>, ParentType, ContextType>;
  comments?: Resolver<Maybe<Array<ResolversTypes['Comment']>>, ParentType, ContextType>;
  created_at?: Resolver<ResolversTypes['Date'], ParentType, ContextType>;
  emails?: Resolver<Maybe<Array<ResolversTypes['Email']>>, ParentType, ContextType>;
  focusModes?: Resolver<Maybe<Array<ResolversTypes['FocusMode']>>, ParentType, ContextType>;
  handle?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  invitedByUser?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  invitedUsers?: Resolver<Maybe<Array<ResolversTypes['User']>>, ParentType, ContextType>;
  isBot?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isBotDepictingPerson?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivate?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateApis?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateApisCreated?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateBookmarks?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateCodes?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateCodesCreated?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateMemberships?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateProjects?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateProjectsCreated?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivatePullRequests?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateQuestionsAnswered?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateQuestionsAsked?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateQuizzesCreated?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateRoles?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateRoutines?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateRoutinesCreated?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateStandards?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateStandardsCreated?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateTeamsCreated?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  isPrivateVotes?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  issuesClosed?: Resolver<Maybe<Array<ResolversTypes['Issue']>>, ParentType, ContextType>;
  issuesCreated?: Resolver<Maybe<Array<ResolversTypes['Issue']>>, ParentType, ContextType>;
  labels?: Resolver<Maybe<Array<ResolversTypes['Label']>>, ParentType, ContextType>;
  meetingsAttending?: Resolver<Maybe<Array<ResolversTypes['Meeting']>>, ParentType, ContextType>;
  meetingsInvited?: Resolver<Maybe<Array<ResolversTypes['MeetingInvite']>>, ParentType, ContextType>;
  memberships?: Resolver<Maybe<Array<ResolversTypes['Member']>>, ParentType, ContextType>;
  membershipsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  membershipsInvited?: Resolver<Maybe<Array<ResolversTypes['MemberInvite']>>, ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  notes?: Resolver<Maybe<Array<ResolversTypes['Note']>>, ParentType, ContextType>;
  notesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  notesCreated?: Resolver<Maybe<Array<ResolversTypes['Note']>>, ParentType, ContextType>;
  notificationSettings?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  notificationSubscriptions?: Resolver<Maybe<Array<ResolversTypes['NotificationSubscription']>>, ParentType, ContextType>;
  notifications?: Resolver<Maybe<Array<ResolversTypes['Notification']>>, ParentType, ContextType>;
  paymentHistory?: Resolver<Maybe<Array<ResolversTypes['Payment']>>, ParentType, ContextType>;
  phones?: Resolver<Maybe<Array<ResolversTypes['Phone']>>, ParentType, ContextType>;
  premium?: Resolver<Maybe<ResolversTypes['Premium']>, ParentType, ContextType>;
  profileImage?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  projects?: Resolver<Maybe<Array<ResolversTypes['Project']>>, ParentType, ContextType>;
  projectsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  projectsCreated?: Resolver<Maybe<Array<ResolversTypes['Project']>>, ParentType, ContextType>;
  pullRequests?: Resolver<Maybe<Array<ResolversTypes['PullRequest']>>, ParentType, ContextType>;
  pushDevices?: Resolver<Maybe<Array<ResolversTypes['PushDevice']>>, ParentType, ContextType>;
  questionsAnswered?: Resolver<Maybe<Array<ResolversTypes['QuestionAnswer']>>, ParentType, ContextType>;
  questionsAsked?: Resolver<Maybe<Array<ResolversTypes['Question']>>, ParentType, ContextType>;
  questionsAskedCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  quizzesCreated?: Resolver<Maybe<Array<ResolversTypes['Quiz']>>, ParentType, ContextType>;
  quizzesTaken?: Resolver<Maybe<Array<ResolversTypes['Quiz']>>, ParentType, ContextType>;
  reacted?: Resolver<Maybe<Array<ResolversTypes['Reaction']>>, ParentType, ContextType>;
  reportResponses?: Resolver<Maybe<Array<ResolversTypes['ReportResponse']>>, ParentType, ContextType>;
  reportsCreated?: Resolver<Maybe<Array<ResolversTypes['Report']>>, ParentType, ContextType>;
  reportsReceived?: Resolver<Array<ResolversTypes['Report']>, ParentType, ContextType>;
  reportsReceivedCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  reputationHistory?: Resolver<Maybe<Array<ResolversTypes['ReputationHistory']>>, ParentType, ContextType>;
  roles?: Resolver<Maybe<Array<ResolversTypes['Role']>>, ParentType, ContextType>;
  routines?: Resolver<Maybe<Array<ResolversTypes['Routine']>>, ParentType, ContextType>;
  routinesCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  routinesCreated?: Resolver<Maybe<Array<ResolversTypes['Routine']>>, ParentType, ContextType>;
  runProjects?: Resolver<Maybe<Array<ResolversTypes['RunProject']>>, ParentType, ContextType>;
  runRoutines?: Resolver<Maybe<Array<ResolversTypes['RunRoutine']>>, ParentType, ContextType>;
  sentReports?: Resolver<Maybe<Array<ResolversTypes['Report']>>, ParentType, ContextType>;
  standards?: Resolver<Maybe<Array<ResolversTypes['Standard']>>, ParentType, ContextType>;
  standardsCount?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  standardsCreated?: Resolver<Maybe<Array<ResolversTypes['Standard']>>, ParentType, ContextType>;
  status?: Resolver<Maybe<ResolversTypes['AccountStatus']>, ParentType, ContextType>;
  tags?: Resolver<Maybe<Array<ResolversTypes['Tag']>>, ParentType, ContextType>;
  teamsCreated?: Resolver<Maybe<Array<ResolversTypes['Team']>>, ParentType, ContextType>;
  theme?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  transfersIncoming?: Resolver<Maybe<Array<ResolversTypes['Transfer']>>, ParentType, ContextType>;
  transfersOutgoing?: Resolver<Maybe<Array<ResolversTypes['Transfer']>>, ParentType, ContextType>;
  translationLanguages?: Resolver<Maybe<Array<ResolversTypes['String']>>, ParentType, ContextType>;
  translations?: Resolver<Array<ResolversTypes['UserTranslation']>, ParentType, ContextType>;
  updated_at?: Resolver<Maybe<ResolversTypes['Date']>, ParentType, ContextType>;
  viewed?: Resolver<Maybe<Array<ResolversTypes['View']>>, ParentType, ContextType>;
  viewedBy?: Resolver<Maybe<Array<ResolversTypes['View']>>, ParentType, ContextType>;
  views?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  wallets?: Resolver<Maybe<Array<ResolversTypes['Wallet']>>, ParentType, ContextType>;
  you?: Resolver<ResolversTypes['UserYou'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type UserEdgeResolvers<ContextType = any, ParentType extends ResolversParentTypes['UserEdge'] = ResolversParentTypes['UserEdge']> = {
  cursor?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  node?: Resolver<ResolversTypes['User'], ParentType, ContextType>;
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
  __resolveType: TypeResolveFn<'Api' | 'Code' | 'Issue' | 'Note' | 'Post' | 'Project' | 'Question' | 'Routine' | 'Standard' | 'Team' | 'User', ParentType, ContextType>;
};

export type WalletResolvers<ContextType = any, ParentType extends ResolversParentTypes['Wallet'] = ResolversParentTypes['Wallet']> = {
  id?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  name?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  publicAddress?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  stakingAddress?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  team?: Resolver<Maybe<ResolversTypes['Team']>, ParentType, ContextType>;
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
  ActiveFocusMode?: ActiveFocusModeResolvers<ContextType>;
  Api?: ApiResolvers<ContextType>;
  ApiEdge?: ApiEdgeResolvers<ContextType>;
  ApiKey?: ApiKeyResolvers<ContextType>;
  ApiSearchResult?: ApiSearchResultResolvers<ContextType>;
  ApiVersion?: ApiVersionResolvers<ContextType>;
  ApiVersionEdge?: ApiVersionEdgeResolvers<ContextType>;
  ApiVersionSearchResult?: ApiVersionSearchResultResolvers<ContextType>;
  ApiVersionTranslation?: ApiVersionTranslationResolvers<ContextType>;
  ApiYou?: ApiYouResolvers<ContextType>;
  AutoFillResult?: AutoFillResultResolvers<ContextType>;
  Award?: AwardResolvers<ContextType>;
  AwardEdge?: AwardEdgeResolvers<ContextType>;
  AwardSearchResult?: AwardSearchResultResolvers<ContextType>;
  Bookmark?: BookmarkResolvers<ContextType>;
  BookmarkEdge?: BookmarkEdgeResolvers<ContextType>;
  BookmarkList?: BookmarkListResolvers<ContextType>;
  BookmarkListEdge?: BookmarkListEdgeResolvers<ContextType>;
  BookmarkListSearchResult?: BookmarkListSearchResultResolvers<ContextType>;
  BookmarkSearchResult?: BookmarkSearchResultResolvers<ContextType>;
  BookmarkTo?: BookmarkToResolvers<ContextType>;
  Chat?: ChatResolvers<ContextType>;
  ChatEdge?: ChatEdgeResolvers<ContextType>;
  ChatInvite?: ChatInviteResolvers<ContextType>;
  ChatInviteEdge?: ChatInviteEdgeResolvers<ContextType>;
  ChatInviteSearchResult?: ChatInviteSearchResultResolvers<ContextType>;
  ChatInviteYou?: ChatInviteYouResolvers<ContextType>;
  ChatMessage?: ChatMessageResolvers<ContextType>;
  ChatMessageEdge?: ChatMessageEdgeResolvers<ContextType>;
  ChatMessageParent?: ChatMessageParentResolvers<ContextType>;
  ChatMessageSearchResult?: ChatMessageSearchResultResolvers<ContextType>;
  ChatMessageSearchTreeResult?: ChatMessageSearchTreeResultResolvers<ContextType>;
  ChatMessageTranslation?: ChatMessageTranslationResolvers<ContextType>;
  ChatMessageYou?: ChatMessageYouResolvers<ContextType>;
  ChatMessageedOn?: ChatMessageedOnResolvers<ContextType>;
  ChatParticipant?: ChatParticipantResolvers<ContextType>;
  ChatParticipantEdge?: ChatParticipantEdgeResolvers<ContextType>;
  ChatParticipantSearchResult?: ChatParticipantSearchResultResolvers<ContextType>;
  ChatSearchResult?: ChatSearchResultResolvers<ContextType>;
  ChatTranslation?: ChatTranslationResolvers<ContextType>;
  ChatYou?: ChatYouResolvers<ContextType>;
  CheckTaskStatusesResult?: CheckTaskStatusesResultResolvers<ContextType>;
  Code?: CodeResolvers<ContextType>;
  CodeEdge?: CodeEdgeResolvers<ContextType>;
  CodeSearchResult?: CodeSearchResultResolvers<ContextType>;
  CodeVersion?: CodeVersionResolvers<ContextType>;
  CodeVersionEdge?: CodeVersionEdgeResolvers<ContextType>;
  CodeVersionSearchResult?: CodeVersionSearchResultResolvers<ContextType>;
  CodeVersionTranslation?: CodeVersionTranslationResolvers<ContextType>;
  CodeYou?: CodeYouResolvers<ContextType>;
  Comment?: CommentResolvers<ContextType>;
  CommentSearchResult?: CommentSearchResultResolvers<ContextType>;
  CommentThread?: CommentThreadResolvers<ContextType>;
  CommentTranslation?: CommentTranslationResolvers<ContextType>;
  CommentYou?: CommentYouResolvers<ContextType>;
  CommentedOn?: CommentedOnResolvers<ContextType>;
  CopyResult?: CopyResultResolvers<ContextType>;
  Count?: CountResolvers<ContextType>;
  Date?: GraphQLScalarType;
  Email?: EmailResolvers<ContextType>;
  FocusMode?: FocusModeResolvers<ContextType>;
  FocusModeEdge?: FocusModeEdgeResolvers<ContextType>;
  FocusModeFilter?: FocusModeFilterResolvers<ContextType>;
  FocusModeSearchResult?: FocusModeSearchResultResolvers<ContextType>;
  FocusModeYou?: FocusModeYouResolvers<ContextType>;
  HomeResult?: HomeResultResolvers<ContextType>;
  Issue?: IssueResolvers<ContextType>;
  IssueEdge?: IssueEdgeResolvers<ContextType>;
  IssueSearchResult?: IssueSearchResultResolvers<ContextType>;
  IssueTo?: IssueToResolvers<ContextType>;
  IssueTranslation?: IssueTranslationResolvers<ContextType>;
  IssueYou?: IssueYouResolvers<ContextType>;
  JSONObject?: GraphQLScalarType;
  Label?: LabelResolvers<ContextType>;
  LabelEdge?: LabelEdgeResolvers<ContextType>;
  LabelSearchResult?: LabelSearchResultResolvers<ContextType>;
  LabelTranslation?: LabelTranslationResolvers<ContextType>;
  LabelYou?: LabelYouResolvers<ContextType>;
  LlmTaskStatusInfo?: LlmTaskStatusInfoResolvers<ContextType>;
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
  MemberYou?: MemberYouResolvers<ContextType>;
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
  NotePage?: NotePageResolvers<ContextType>;
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
  Owner?: OwnerResolvers<ContextType>;
  PageInfo?: PageInfoResolvers<ContextType>;
  Payment?: PaymentResolvers<ContextType>;
  PaymentEdge?: PaymentEdgeResolvers<ContextType>;
  PaymentSearchResult?: PaymentSearchResultResolvers<ContextType>;
  Phone?: PhoneResolvers<ContextType>;
  Popular?: PopularResolvers<ContextType>;
  PopularEdge?: PopularEdgeResolvers<ContextType>;
  PopularPageInfo?: PopularPageInfoResolvers<ContextType>;
  PopularSearchResult?: PopularSearchResultResolvers<ContextType>;
  Post?: PostResolvers<ContextType>;
  PostEdge?: PostEdgeResolvers<ContextType>;
  PostSearchResult?: PostSearchResultResolvers<ContextType>;
  PostTranslation?: PostTranslationResolvers<ContextType>;
  Premium?: PremiumResolvers<ContextType>;
  Project?: ProjectResolvers<ContextType>;
  ProjectEdge?: ProjectEdgeResolvers<ContextType>;
  ProjectOrRoutine?: ProjectOrRoutineResolvers<ContextType>;
  ProjectOrRoutineEdge?: ProjectOrRoutineEdgeResolvers<ContextType>;
  ProjectOrRoutinePageInfo?: ProjectOrRoutinePageInfoResolvers<ContextType>;
  ProjectOrRoutineSearchResult?: ProjectOrRoutineSearchResultResolvers<ContextType>;
  ProjectOrTeam?: ProjectOrTeamResolvers<ContextType>;
  ProjectOrTeamEdge?: ProjectOrTeamEdgeResolvers<ContextType>;
  ProjectOrTeamPageInfo?: ProjectOrTeamPageInfoResolvers<ContextType>;
  ProjectOrTeamSearchResult?: ProjectOrTeamSearchResultResolvers<ContextType>;
  ProjectSearchResult?: ProjectSearchResultResolvers<ContextType>;
  ProjectVersion?: ProjectVersionResolvers<ContextType>;
  ProjectVersionContentsSearchResult?: ProjectVersionContentsSearchResultResolvers<ContextType>;
  ProjectVersionDirectory?: ProjectVersionDirectoryResolvers<ContextType>;
  ProjectVersionDirectoryEdge?: ProjectVersionDirectoryEdgeResolvers<ContextType>;
  ProjectVersionDirectorySearchResult?: ProjectVersionDirectorySearchResultResolvers<ContextType>;
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
  PullRequestTranslation?: PullRequestTranslationResolvers<ContextType>;
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
  QuizQuestionResponseYou?: QuizQuestionResponseYouResolvers<ContextType>;
  QuizQuestionSearchResult?: QuizQuestionSearchResultResolvers<ContextType>;
  QuizQuestionTranslation?: QuizQuestionTranslationResolvers<ContextType>;
  QuizQuestionYou?: QuizQuestionYouResolvers<ContextType>;
  QuizSearchResult?: QuizSearchResultResolvers<ContextType>;
  QuizTranslation?: QuizTranslationResolvers<ContextType>;
  QuizYou?: QuizYouResolvers<ContextType>;
  Reaction?: ReactionResolvers<ContextType>;
  ReactionEdge?: ReactionEdgeResolvers<ContextType>;
  ReactionSearchResult?: ReactionSearchResultResolvers<ContextType>;
  ReactionSummary?: ReactionSummaryResolvers<ContextType>;
  ReactionTo?: ReactionToResolvers<ContextType>;
  Reminder?: ReminderResolvers<ContextType>;
  ReminderEdge?: ReminderEdgeResolvers<ContextType>;
  ReminderItem?: ReminderItemResolvers<ContextType>;
  ReminderList?: ReminderListResolvers<ContextType>;
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
  Resource?: ResourceResolvers<ContextType>;
  ResourceEdge?: ResourceEdgeResolvers<ContextType>;
  ResourceList?: ResourceListResolvers<ContextType>;
  ResourceListEdge?: ResourceListEdgeResolvers<ContextType>;
  ResourceListOn?: ResourceListOnResolvers<ContextType>;
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
  RunProjectSearchResult?: RunProjectSearchResultResolvers<ContextType>;
  RunProjectStep?: RunProjectStepResolvers<ContextType>;
  RunProjectYou?: RunProjectYouResolvers<ContextType>;
  RunRoutine?: RunRoutineResolvers<ContextType>;
  RunRoutineEdge?: RunRoutineEdgeResolvers<ContextType>;
  RunRoutineInput?: RunRoutineInputResolvers<ContextType>;
  RunRoutineInputEdge?: RunRoutineInputEdgeResolvers<ContextType>;
  RunRoutineInputSearchResult?: RunRoutineInputSearchResultResolvers<ContextType>;
  RunRoutineOutput?: RunRoutineOutputResolvers<ContextType>;
  RunRoutineOutputEdge?: RunRoutineOutputEdgeResolvers<ContextType>;
  RunRoutineOutputSearchResult?: RunRoutineOutputSearchResultResolvers<ContextType>;
  RunRoutineSearchResult?: RunRoutineSearchResultResolvers<ContextType>;
  RunRoutineStep?: RunRoutineStepResolvers<ContextType>;
  RunRoutineYou?: RunRoutineYouResolvers<ContextType>;
  Schedule?: ScheduleResolvers<ContextType>;
  ScheduleEdge?: ScheduleEdgeResolvers<ContextType>;
  ScheduleException?: ScheduleExceptionResolvers<ContextType>;
  ScheduleExceptionEdge?: ScheduleExceptionEdgeResolvers<ContextType>;
  ScheduleExceptionSearchResult?: ScheduleExceptionSearchResultResolvers<ContextType>;
  ScheduleRecurrence?: ScheduleRecurrenceResolvers<ContextType>;
  ScheduleRecurrenceEdge?: ScheduleRecurrenceEdgeResolvers<ContextType>;
  ScheduleRecurrenceSearchResult?: ScheduleRecurrenceSearchResultResolvers<ContextType>;
  ScheduleSearchResult?: ScheduleSearchResultResolvers<ContextType>;
  Session?: SessionResolvers<ContextType>;
  SessionUser?: SessionUserResolvers<ContextType>;
  Standard?: StandardResolvers<ContextType>;
  StandardEdge?: StandardEdgeResolvers<ContextType>;
  StandardSearchResult?: StandardSearchResultResolvers<ContextType>;
  StandardVersion?: StandardVersionResolvers<ContextType>;
  StandardVersionEdge?: StandardVersionEdgeResolvers<ContextType>;
  StandardVersionSearchResult?: StandardVersionSearchResultResolvers<ContextType>;
  StandardVersionTranslation?: StandardVersionTranslationResolvers<ContextType>;
  StandardYou?: StandardYouResolvers<ContextType>;
  StatsApi?: StatsApiResolvers<ContextType>;
  StatsApiEdge?: StatsApiEdgeResolvers<ContextType>;
  StatsApiSearchResult?: StatsApiSearchResultResolvers<ContextType>;
  StatsCode?: StatsCodeResolvers<ContextType>;
  StatsCodeEdge?: StatsCodeEdgeResolvers<ContextType>;
  StatsCodeSearchResult?: StatsCodeSearchResultResolvers<ContextType>;
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
  StatsStandard?: StatsStandardResolvers<ContextType>;
  StatsStandardEdge?: StatsStandardEdgeResolvers<ContextType>;
  StatsStandardSearchResult?: StatsStandardSearchResultResolvers<ContextType>;
  StatsTeam?: StatsTeamResolvers<ContextType>;
  StatsTeamEdge?: StatsTeamEdgeResolvers<ContextType>;
  StatsTeamSearchResult?: StatsTeamSearchResultResolvers<ContextType>;
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
  Team?: TeamResolvers<ContextType>;
  TeamEdge?: TeamEdgeResolvers<ContextType>;
  TeamSearchResult?: TeamSearchResultResolvers<ContextType>;
  TeamTranslation?: TeamTranslationResolvers<ContextType>;
  TeamYou?: TeamYouResolvers<ContextType>;
  Transfer?: TransferResolvers<ContextType>;
  TransferEdge?: TransferEdgeResolvers<ContextType>;
  TransferObject?: TransferObjectResolvers<ContextType>;
  TransferSearchResult?: TransferSearchResultResolvers<ContextType>;
  TransferYou?: TransferYouResolvers<ContextType>;
  Translate?: TranslateResolvers<ContextType>;
  Upload?: GraphQLScalarType;
  User?: UserResolvers<ContextType>;
  UserEdge?: UserEdgeResolvers<ContextType>;
  UserSearchResult?: UserSearchResultResolvers<ContextType>;
  UserTranslation?: UserTranslationResolvers<ContextType>;
  UserYou?: UserYouResolvers<ContextType>;
  VersionYou?: VersionYouResolvers<ContextType>;
  View?: ViewResolvers<ContextType>;
  ViewEdge?: ViewEdgeResolvers<ContextType>;
  ViewSearchResult?: ViewSearchResultResolvers<ContextType>;
  ViewTo?: ViewToResolvers<ContextType>;
  Wallet?: WalletResolvers<ContextType>;
  WalletComplete?: WalletCompleteResolvers<ContextType>;
};

