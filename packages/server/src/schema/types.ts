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
  /** Custom description for the date scalar */
  Date: any;
  /** The `Upload` scalar type represents a file upload. */
  Upload: any;
};

export enum AccountStatus {
  Deleted = 'Deleted',
  HardLocked = 'HardLocked',
  SoftLocked = 'SoftLocked',
  Unlocked = 'Unlocked'
}

export type BooleanSearchException = {
  id: Scalars['ID'];
  relation: Scalars['String'];
};

export type Comment = {
  __typename?: 'Comment';
  commentedOn: CommentedOn;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  id: Scalars['ID'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  role?: Maybe<MemberRole>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  translations: Array<CommentTranslation>;
  updated_at: Scalars['Date'];
};

export type CommentCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type CommentCreateInput = {
  createdFor: CommentFor;
  forId: Scalars['ID'];
  id?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  translationsCreate?: InputMaybe<Array<CommentTranslationCreateInput>>;
};

export enum CommentFor {
  Project = 'Project',
  Routine = 'Routine',
  Standard = 'Standard'
}

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
  id?: InputMaybe<Scalars['ID']>;
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
  objectType: CopyType;
};

export type CopyResult = {
  __typename?: 'CopyResult';
  node?: Maybe<Node>;
  organization?: Maybe<Organization>;
  project?: Maybe<Project>;
  routine?: Maybe<Routine>;
  standard?: Maybe<Standard>;
};

export enum CopyType {
  Node = 'Node',
  Organization = 'Organization',
  Project = 'Project',
  Routine = 'Routine',
  Standard = 'Standard'
}

export type Count = {
  __typename?: 'Count';
  count?: Maybe<Scalars['Int']>;
};

export type DeleteManyInput = {
  ids: Array<Scalars['ID']>;
};

export type DeleteOneInput = {
  id: Scalars['ID'];
  objectType: DeleteOneType;
};

export enum DeleteOneType {
  Comment = 'Comment',
  Email = 'Email',
  Node = 'Node',
  Organization = 'Organization',
  Project = 'Project',
  Report = 'Report',
  Routine = 'Routine',
  Run = 'Run',
  Standard = 'Standard',
  Wallet = 'Wallet'
}

export type DevelopPageResult = {
  __typename?: 'DevelopPageResult';
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

export type FindHandlesInput = {
  organizationId?: InputMaybe<Scalars['ID']>;
};

export type ForkInput = {
  id: Scalars['ID'];
  objectType: ForkType;
};

export type ForkResult = {
  __typename?: 'ForkResult';
  organization?: Maybe<Organization>;
  project?: Maybe<Project>;
  routine?: Maybe<Routine>;
  standard?: Maybe<Standard>;
};

export enum ForkType {
  Organization = 'Organization',
  Project = 'Project',
  Routine = 'Routine',
  Standard = 'Standard'
}

export type Handle = {
  __typename?: 'Handle';
  handle: Scalars['String'];
  id: Scalars['ID'];
  wallet: Wallet;
};

export type HistoryPageInput = {
  searchString: Scalars['String'];
  take?: InputMaybe<Scalars['Int']>;
};

export type HistoryPageResult = {
  __typename?: 'HistoryPageResult';
  activeRuns: Array<Run>;
  completedRuns: Array<Run>;
  recentlyStarred: Array<Star>;
  recentlyViewed: Array<View>;
};

export type HomePageInput = {
  searchString: Scalars['String'];
  take?: InputMaybe<Scalars['Int']>;
};

export type HomePageResult = {
  __typename?: 'HomePageResult';
  organizations: Array<Organization>;
  projects: Array<Project>;
  routines: Array<Routine>;
  standards: Array<Standard>;
  users: Array<User>;
};

export type InputItem = {
  __typename?: 'InputItem';
  id: Scalars['ID'];
  isRequired?: Maybe<Scalars['Boolean']>;
  name?: Maybe<Scalars['String']>;
  routine: Routine;
  standard?: Maybe<Standard>;
  translations: Array<InputItemTranslation>;
};

export type InputItemCreateInput = {
  id?: InputMaybe<Scalars['ID']>;
  isRequired?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  standardConnect?: InputMaybe<Scalars['ID']>;
  standardCreate?: InputMaybe<StandardCreateInput>;
  translationsCreate?: InputMaybe<Array<InputItemTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<InputItemTranslationUpdateInput>>;
};

export type InputItemTranslation = {
  __typename?: 'InputItemTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type InputItemTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
  language: Scalars['String'];
};

export type InputItemTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
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

export type LearnPageResult = {
  __typename?: 'LearnPageResult';
  courses: Array<Project>;
  tutorials: Array<Routine>;
};

export type Log = {
  __typename?: 'Log';
  action: LogType;
  data?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  object1Id?: Maybe<Scalars['ID']>;
  object1Type?: Maybe<Scalars['String']>;
  object2Id?: Maybe<Scalars['ID']>;
  object2Type?: Maybe<Scalars['String']>;
  timestamp: Scalars['Date'];
};

export type LogCreateInput = {
  action: LogType;
  data?: InputMaybe<Scalars['String']>;
  object1Id?: InputMaybe<Scalars['ID']>;
  object1Type?: InputMaybe<Scalars['String']>;
  object2Id?: InputMaybe<Scalars['ID']>;
  object2Type?: InputMaybe<Scalars['String']>;
};

export type LogEdge = {
  __typename?: 'LogEdge';
  cursor: Scalars['String'];
  node: Log;
};

export type LogSearchInput = {
  actions?: InputMaybe<Array<Scalars['String']>>;
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  data?: InputMaybe<Scalars['String']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  object1Id?: InputMaybe<Scalars['ID']>;
  object1Type?: InputMaybe<Scalars['String']>;
  object2Id?: InputMaybe<Scalars['ID']>;
  object2Type?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<LogSortBy>;
  take?: InputMaybe<Scalars['Int']>;
};

export type LogSearchResult = {
  __typename?: 'LogSearchResult';
  edges: Array<LogEdge>;
  pageInfo: PageInfo;
};

export enum LogSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc'
}

export enum LogType {
  Copy = 'Copy',
  Create = 'Create',
  Delete = 'Delete',
  Downvote = 'Downvote',
  Fork = 'Fork',
  OrganizationAddMember = 'OrganizationAddMember',
  OrganizationJoin = 'OrganizationJoin',
  OrganizationLeave = 'OrganizationLeave',
  OrganizationRemoveMember = 'OrganizationRemoveMember',
  OrganizationUpdateMember = 'OrganizationUpdateMember',
  ProjectComplete = 'ProjectComplete',
  RemoveStar = 'RemoveStar',
  RemoveVote = 'RemoveVote',
  RoutineCancel = 'RoutineCancel',
  RoutineComplete = 'RoutineComplete',
  RoutineStartCanceled = 'RoutineStartCanceled',
  RoutineStartCompleted = 'RoutineStartCompleted',
  RoutineStartIncomplete = 'RoutineStartIncomplete',
  Star = 'Star',
  Update = 'Update',
  Upvote = 'Upvote',
  View = 'View'
}

export type Loop = {
  __typename?: 'Loop';
  id: Scalars['ID'];
  loops?: Maybe<Scalars['Int']>;
  maxLoops?: Maybe<Scalars['Int']>;
  operation?: Maybe<Scalars['String']>;
  whiles: Array<LoopWhile>;
};

export type LoopCreateInput = {
  id?: InputMaybe<Scalars['ID']>;
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
  id?: InputMaybe<Scalars['ID']>;
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
  id?: InputMaybe<Scalars['ID']>;
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

export type Member = {
  __typename?: 'Member';
  role: MemberRole;
  user: User;
};

export enum MemberRole {
  Admin = 'Admin',
  Member = 'Member',
  Owner = 'Owner'
}

export type Mutation = {
  __typename?: 'Mutation';
  commentCreate: Comment;
  commentUpdate: Comment;
  copy: CopyResult;
  deleteOne: Success;
  emailCreate: Email;
  emailLogIn: Session;
  emailRequestPasswordChange: Success;
  emailResetPassword: Session;
  emailSignUp: Session;
  emailUpdate: Email;
  exportData: Scalars['String'];
  feedbackCreate: Success;
  fork: ForkResult;
  guestLogIn: Session;
  logDeleteMany: Count;
  logOut: Success;
  nodeCreate: Node;
  nodeUpdate: Node;
  organizationCreate: Organization;
  organizationUpdate: Organization;
  profileEmailUpdate: Profile;
  profileUpdate: Profile;
  projectCreate: Project;
  projectUpdate: Project;
  reportCreate: Report;
  reportUpdate: Report;
  resourceCreate: Resource;
  resourceDeleteMany: Count;
  resourceListCreate: ResourceList;
  resourceListDeleteMany: Count;
  resourceListUpdate: ResourceList;
  resourceUpdate: Resource;
  routineCreate: Routine;
  routineDeleteOne: Success;
  routineUpdate: Routine;
  runCancel: Run;
  runComplete: Run;
  runCreate: Run;
  runDeleteAll: Count;
  runDeleteMany: Count;
  runUpdate: Run;
  sendVerificationEmail: Success;
  standardCreate: Standard;
  standardUpdate: Standard;
  star: Success;
  stepInputDataCreate: StepInputData;
  stepInputDataDeleteMany: Count;
  stepInputDataUpdate: StepInputData;
  tagCreate: Tag;
  tagDeleteMany: Count;
  tagUpdate: Tag;
  userDeleteOne: Success;
  validateSession: Session;
  vote: Success;
  walletComplete: WalletComplete;
  walletInit: Scalars['String'];
  walletUpdate: Wallet;
  writeAssets?: Maybe<Scalars['Boolean']>;
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


export type MutationForkArgs = {
  input: ForkInput;
};


export type MutationLogDeleteManyArgs = {
  input?: InputMaybe<DeleteManyInput>;
};


export type MutationNodeCreateArgs = {
  input: NodeCreateInput;
};


export type MutationNodeUpdateArgs = {
  input: NodeUpdateInput;
};


export type MutationOrganizationCreateArgs = {
  input: OrganizationCreateInput;
};


export type MutationOrganizationUpdateArgs = {
  input: OrganizationUpdateInput;
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


export type MutationReportCreateArgs = {
  input: ReportCreateInput;
};


export type MutationReportUpdateArgs = {
  input: ReportUpdateInput;
};


export type MutationResourceCreateArgs = {
  input: ResourceCreateInput;
};


export type MutationResourceDeleteManyArgs = {
  input: DeleteManyInput;
};


export type MutationResourceListCreateArgs = {
  input: ResourceListCreateInput;
};


export type MutationResourceListDeleteManyArgs = {
  input: DeleteManyInput;
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


export type MutationRoutineDeleteOneArgs = {
  input: DeleteOneInput;
};


export type MutationRoutineUpdateArgs = {
  input: RoutineUpdateInput;
};


export type MutationRunCancelArgs = {
  input: RunCancelInput;
};


export type MutationRunCompleteArgs = {
  input: RunCompleteInput;
};


export type MutationRunCreateArgs = {
  input: RunCreateInput;
};


export type MutationRunDeleteManyArgs = {
  input: DeleteManyInput;
};


export type MutationRunUpdateArgs = {
  input: RunUpdateInput;
};


export type MutationSendVerificationEmailArgs = {
  input: SendVerificationEmailInput;
};


export type MutationStandardCreateArgs = {
  input: StandardCreateInput;
};


export type MutationStandardUpdateArgs = {
  input: StandardUpdateInput;
};


export type MutationStarArgs = {
  input: StarInput;
};


export type MutationStepInputDataCreateArgs = {
  input: StepInputDataCreateInput;
};


export type MutationStepInputDataDeleteManyArgs = {
  input?: InputMaybe<DeleteManyInput>;
};


export type MutationStepInputDataUpdateArgs = {
  input: StepInputDataUpdateInput;
};


export type MutationTagCreateArgs = {
  input: TagCreateInput;
};


export type MutationTagDeleteManyArgs = {
  input: DeleteManyInput;
};


export type MutationTagUpdateArgs = {
  input: TagUpdateInput;
};


export type MutationUserDeleteOneArgs = {
  input: UserDeleteInput;
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
  id?: InputMaybe<Scalars['ID']>;
  loopCreate?: InputMaybe<LoopCreateInput>;
  nodeEndCreate?: InputMaybe<NodeEndCreateInput>;
  nodeRoutineListCreate?: InputMaybe<NodeRoutineListCreateInput>;
  routineId?: InputMaybe<Scalars['ID']>;
  rowIndex?: InputMaybe<Scalars['Int']>;
  translationsCreate?: InputMaybe<Array<NodeTranslationCreateInput>>;
  type?: InputMaybe<NodeType>;
};

export type NodeData = NodeEnd | NodeRoutineList;

export type NodeEnd = {
  __typename?: 'NodeEnd';
  id: Scalars['ID'];
  wasSuccessful: Scalars['Boolean'];
};

export type NodeEndCreateInput = {
  id?: InputMaybe<Scalars['ID']>;
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
  id?: InputMaybe<Scalars['ID']>;
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
  id?: InputMaybe<Scalars['ID']>;
  toId?: InputMaybe<Scalars['ID']>;
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
  id?: InputMaybe<Scalars['ID']>;
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
  toId?: InputMaybe<Scalars['ID']>;
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
  id?: InputMaybe<Scalars['ID']>;
  isOptional?: InputMaybe<Scalars['Boolean']>;
  isOrdered?: InputMaybe<Scalars['Boolean']>;
  routinesCreate?: InputMaybe<Array<NodeRoutineListItemCreateInput>>;
};

export type NodeRoutineListItem = {
  __typename?: 'NodeRoutineListItem';
  id: Scalars['ID'];
  index: Scalars['Int'];
  isOptional: Scalars['Boolean'];
  routine: Routine;
  translations: Array<NodeRoutineListItemTranslation>;
};

export type NodeRoutineListItemCreateInput = {
  id?: InputMaybe<Scalars['ID']>;
  index: Scalars['Int'];
  isOptional?: InputMaybe<Scalars['Boolean']>;
  routineConnect: Scalars['ID'];
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
  id?: InputMaybe<Scalars['ID']>;
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
  id?: InputMaybe<Scalars['ID']>;
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
  nodeEndUpdate?: InputMaybe<NodeEndUpdateInput>;
  nodeRoutineListUpdate?: InputMaybe<NodeRoutineListUpdateInput>;
  routineId?: InputMaybe<Scalars['ID']>;
  rowIndex?: InputMaybe<Scalars['Int']>;
  translationsCreate?: InputMaybe<Array<NodeTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<NodeTranslationUpdateInput>>;
  type?: InputMaybe<NodeType>;
};

export type Organization = {
  __typename?: 'Organization';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  created_at: Scalars['Date'];
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isOpenToNewMembers: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isViewed: Scalars['Boolean'];
  members: Array<Member>;
  projects: Array<Project>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists: Array<ResourceList>;
  role?: Maybe<MemberRole>;
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

export type OrganizationCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type OrganizationCreateInput = {
  id?: InputMaybe<Scalars['ID']>;
  isOpenToNewMembers?: InputMaybe<Scalars['Boolean']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<OrganizationTranslationCreateInput>>;
};

export type OrganizationEdge = {
  __typename?: 'OrganizationEdge';
  cursor: Scalars['String'];
  node: Organization;
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
  id?: InputMaybe<Scalars['ID']>;
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
  membersConnect?: InputMaybe<Array<Scalars['ID']>>;
  membersDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  translationsCreate?: InputMaybe<Array<OrganizationTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<OrganizationTranslationUpdateInput>>;
};

export type OutputItem = {
  __typename?: 'OutputItem';
  id: Scalars['ID'];
  name?: Maybe<Scalars['String']>;
  routine: Routine;
  standard?: Maybe<Standard>;
  translations: Array<OutputItemTranslation>;
};

export type OutputItemCreateInput = {
  id?: InputMaybe<Scalars['ID']>;
  name?: InputMaybe<Scalars['String']>;
  standardConnect?: InputMaybe<Scalars['ID']>;
  standardCreate?: InputMaybe<StandardCreateInput>;
  translationsCreate?: InputMaybe<Array<OutputItemTranslationCreateInput>>;
};

export type OutputItemTranslation = {
  __typename?: 'OutputItemTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
};

export type OutputItemTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
  language: Scalars['String'];
};

export type OutputItemTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
};

export type OutputItemUpdateInput = {
  id: Scalars['ID'];
  name?: InputMaybe<Scalars['String']>;
  standardConnect?: InputMaybe<Scalars['ID']>;
  standardCreate?: InputMaybe<StandardCreateInput>;
  translationsCreate?: InputMaybe<Array<OutputItemTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<OutputItemTranslationUpdateInput>>;
};

export type PageInfo = {
  __typename?: 'PageInfo';
  endCursor?: Maybe<Scalars['String']>;
  hasNextPage: Scalars['Boolean'];
};

export type Profile = {
  __typename?: 'Profile';
  comments: Array<Comment>;
  created_at: Scalars['Date'];
  emails: Array<Email>;
  handle?: Maybe<Scalars['String']>;
  hiddenTags?: Maybe<Array<TagHidden>>;
  history: Array<Log>;
  id: Scalars['ID'];
  name: Scalars['String'];
  projects: Array<Project>;
  projectsCreated: Array<Project>;
  reports: Array<Report>;
  resourceLists: Array<ResourceList>;
  roles: Array<Role>;
  routines: Array<Routine>;
  routinesCreated: Array<Routine>;
  sentReports: Array<Report>;
  starredBy: Array<User>;
  starredTags?: Maybe<Array<Tag>>;
  stars: Array<Star>;
  status: AccountStatus;
  theme: Scalars['String'];
  translations: Array<UserTranslation>;
  updated_at: Scalars['Date'];
  votes: Array<Vote>;
  wallets: Array<Wallet>;
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
  resourceListsCreate?: InputMaybe<Array<ResourceCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceUpdateInput>>;
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
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists?: Maybe<Array<ResourceList>>;
  role?: Maybe<MemberRole>;
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

export type ProjectCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type ProjectCreateInput = {
  createdByOrganizationId?: InputMaybe<Scalars['ID']>;
  createdByUserId?: InputMaybe<Scalars['ID']>;
  id?: InputMaybe<Scalars['ID']>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
};

export type ProjectEdge = {
  __typename?: 'ProjectEdge';
  cursor: Scalars['String'];
  node: Project;
};

export type ProjectOrOrganization = Organization | Project;

export type ProjectOrOrganizationOrRoutineOrStandardOrUser = Organization | Project | Routine | Standard | User;

export type ProjectOrRoutine = Project | Routine;

export type ProjectSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<BooleanSearchException>>;
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
};

export type ProjectSearchResult = {
  __typename?: 'ProjectSearchResult';
  edges: Array<ProjectEdge>;
  pageInfo: PageInfo;
};

export enum ProjectSortBy {
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

export type ProjectTranslation = {
  __typename?: 'ProjectTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
  name: Scalars['String'];
};

export type ProjectTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
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
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  translationsCreate?: InputMaybe<Array<ProjectTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<ProjectTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type Query = {
  __typename?: 'Query';
  comment?: Maybe<Comment>;
  comments: CommentSearchResult;
  commentsCount: Scalars['Int'];
  developPage: DevelopPageResult;
  findHandles: Array<Scalars['String']>;
  historyPage: HistoryPageResult;
  homePage: HomePageResult;
  learnPage: LearnPageResult;
  logs: LogSearchResult;
  organization?: Maybe<Organization>;
  organizations: OrganizationSearchResult;
  organizationsCount: Scalars['Int'];
  profile: Profile;
  project?: Maybe<Project>;
  projects: ProjectSearchResult;
  projectsCount: Scalars['Int'];
  readAssets: Array<Maybe<Scalars['String']>>;
  report?: Maybe<Report>;
  reports: ReportSearchResult;
  reportsCount: Scalars['Int'];
  researchPage: ResearchPageResult;
  resource?: Maybe<Resource>;
  resourceList?: Maybe<Resource>;
  resourceLists: ResourceListSearchResult;
  resourceListsCount: Scalars['Int'];
  resources: ResourceSearchResult;
  resourcesCount: Scalars['Int'];
  routine?: Maybe<Routine>;
  routines: RoutineSearchResult;
  routinesCount: Scalars['Int'];
  run?: Maybe<Run>;
  runs: RunSearchResult;
  runsCount: Scalars['Int'];
  standard?: Maybe<Standard>;
  standards: StandardSearchResult;
  standardsCount: Scalars['Int'];
  statisticsPage: StatisticsPageResult;
  stepInputData?: Maybe<StepInputData>;
  stepInputDatas: StepInputDataSearchResult;
  tag?: Maybe<Tag>;
  tags: TagSearchResult;
  tagsCount: Scalars['Int'];
  user?: Maybe<User>;
  users: UserSearchResult;
  usersCount: Scalars['Int'];
};


export type QueryCommentArgs = {
  input: FindByIdInput;
};


export type QueryCommentsArgs = {
  input: CommentSearchInput;
};


export type QueryCommentsCountArgs = {
  input: CommentCountInput;
};


export type QueryFindHandlesArgs = {
  input: FindHandlesInput;
};


export type QueryHistoryPageArgs = {
  input: HistoryPageInput;
};


export type QueryHomePageArgs = {
  input: HomePageInput;
};


export type QueryLogsArgs = {
  input: LogSearchInput;
};


export type QueryOrganizationArgs = {
  input: FindByIdOrHandleInput;
};


export type QueryOrganizationsArgs = {
  input: OrganizationSearchInput;
};


export type QueryOrganizationsCountArgs = {
  input: OrganizationCountInput;
};


export type QueryProjectArgs = {
  input: FindByIdOrHandleInput;
};


export type QueryProjectsArgs = {
  input: ProjectSearchInput;
};


export type QueryProjectsCountArgs = {
  input: ProjectCountInput;
};


export type QueryReadAssetsArgs = {
  input: ReadAssetsInput;
};


export type QueryReportArgs = {
  input: FindByIdInput;
};


export type QueryReportsArgs = {
  input: ReportSearchInput;
};


export type QueryReportsCountArgs = {
  input: ReportCountInput;
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


export type QueryResourceListsCountArgs = {
  input: ResourceListCountInput;
};


export type QueryResourcesArgs = {
  input: ResourceSearchInput;
};


export type QueryResourcesCountArgs = {
  input: ResourceCountInput;
};


export type QueryRoutineArgs = {
  input: FindByIdInput;
};


export type QueryRoutinesArgs = {
  input: RoutineSearchInput;
};


export type QueryRoutinesCountArgs = {
  input: RoutineCountInput;
};


export type QueryRunArgs = {
  input: FindByIdInput;
};


export type QueryRunsArgs = {
  input: RunSearchInput;
};


export type QueryRunsCountArgs = {
  input: RunCountInput;
};


export type QueryStandardArgs = {
  input: FindByIdInput;
};


export type QueryStandardsArgs = {
  input: StandardSearchInput;
};


export type QueryStandardsCountArgs = {
  input: StandardCountInput;
};


export type QueryStatisticsPageArgs = {
  input: StatisticsPageInput;
};


export type QueryStepInputDataArgs = {
  input: FindByIdInput;
};


export type QueryStepInputDatasArgs = {
  input: StepInputDataSearchInput;
};


export type QueryTagArgs = {
  input: FindByIdInput;
};


export type QueryTagsArgs = {
  input: TagSearchInput;
};


export type QueryTagsCountArgs = {
  input: TagCountInput;
};


export type QueryUserArgs = {
  input: FindByIdOrHandleInput;
};


export type QueryUsersArgs = {
  input: UserSearchInput;
};


export type QueryUsersCountArgs = {
  input: UserCountInput;
};

export type ReadAssetsInput = {
  files: Array<Scalars['String']>;
};

export type Report = {
  __typename?: 'Report';
  details?: Maybe<Scalars['String']>;
  id?: Maybe<Scalars['ID']>;
  isOwn: Scalars['Boolean'];
  language: Scalars['String'];
  reason: Scalars['String'];
};

export type ReportCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type ReportCreateInput = {
  createdFor: ReportFor;
  createdForId: Scalars['ID'];
  details?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
  language: Scalars['String'];
  reason: Scalars['String'];
};

export type ReportEdge = {
  __typename?: 'ReportEdge';
  cursor: Scalars['String'];
  node: Report;
};

export enum ReportFor {
  Comment = 'Comment',
  Organization = 'Organization',
  Project = 'Project',
  Routine = 'Routine',
  Standard = 'Standard',
  Tag = 'Tag',
  User = 'User'
}

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
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type ReportUpdateInput = {
  details?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
  reason?: InputMaybe<Scalars['String']>;
};

export type ResearchPageResult = {
  __typename?: 'ResearchPageResult';
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

export type ResourceCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type ResourceCreateInput = {
  id?: InputMaybe<Scalars['ID']>;
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
  translations: Array<ResourceListTranslation>;
  updated_at: Scalars['Date'];
  usedFor?: Maybe<ResourceListUsedFor>;
  user?: Maybe<User>;
};

export type ResourceListCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type ResourceListCreateInput = {
  id?: InputMaybe<Scalars['ID']>;
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
  IndexDesc = 'IndexDesc'
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
  id?: InputMaybe<Scalars['ID']>;
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
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  title: Scalars['String'];
  users: Array<User>;
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
  isInternal?: Maybe<Scalars['Boolean']>;
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
  project?: Maybe<Project>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists: Array<ResourceList>;
  role?: Maybe<MemberRole>;
  runs: Array<Run>;
  score: Scalars['Int'];
  simplicity: Scalars['Int'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<RoutineTranslation>;
  updated_at: Scalars['Date'];
  version?: Maybe<Scalars['String']>;
  views: Scalars['Int'];
};

export type RoutineCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type RoutineCreateInput = {
  createdByOrganizationId?: InputMaybe<Scalars['ID']>;
  createdByUserId?: InputMaybe<Scalars['ID']>;
  id?: InputMaybe<Scalars['ID']>;
  inputsCreate?: InputMaybe<Array<InputItemCreateInput>>;
  isAutomatable?: InputMaybe<Scalars['Boolean']>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isInternal?: InputMaybe<Scalars['Boolean']>;
  nodeLinksCreate?: InputMaybe<Array<NodeLinkCreateInput>>;
  nodesCreate?: InputMaybe<Array<NodeCreateInput>>;
  outputsCreate?: InputMaybe<Array<OutputItemCreateInput>>;
  parentId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<RoutineTranslationCreateInput>>;
  version?: InputMaybe<Scalars['String']>;
};

export type RoutineEdge = {
  __typename?: 'RoutineEdge';
  cursor: Scalars['String'];
  node: Routine;
};

export type RoutineSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isCompleteExceptions?: InputMaybe<Array<BooleanSearchException>>;
  isInternal?: InputMaybe<Scalars['Boolean']>;
  isInternalExceptions?: InputMaybe<Array<BooleanSearchException>>;
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
};

export type RoutineSearchResult = {
  __typename?: 'RoutineSearchResult';
  edges: Array<RoutineEdge>;
  pageInfo: PageInfo;
};

export enum RoutineSortBy {
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
  id?: InputMaybe<Scalars['ID']>;
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
  id: Scalars['ID'];
  inputsCreate?: InputMaybe<Array<InputItemCreateInput>>;
  inputsDelete?: InputMaybe<Array<Scalars['ID']>>;
  inputsUpdate?: InputMaybe<Array<InputItemUpdateInput>>;
  isAutomatable?: InputMaybe<Scalars['Boolean']>;
  isComplete?: InputMaybe<Scalars['Boolean']>;
  isInternal?: InputMaybe<Scalars['Boolean']>;
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
  parentId?: InputMaybe<Scalars['ID']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  translationsCreate?: InputMaybe<Array<RoutineTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<RoutineTranslationUpdateInput>>;
  userId?: InputMaybe<Scalars['ID']>;
  version?: InputMaybe<Scalars['String']>;
};

export type Run = {
  __typename?: 'Run';
  completedComplexity: Scalars['Int'];
  contextSwitches: Scalars['Int'];
  id: Scalars['ID'];
  routine?: Maybe<Routine>;
  status: RunStatus;
  steps: Array<RunStep>;
  timeCompleted?: Maybe<Scalars['Date']>;
  timeElapsed?: Maybe<Scalars['Int']>;
  timeStarted?: Maybe<Scalars['Date']>;
  title: Scalars['String'];
  user: User;
};

export type RunCancelInput = {
  id: Scalars['ID'];
};

export type RunCompleteInput = {
  completedComplexity?: InputMaybe<Scalars['Int']>;
  exists?: InputMaybe<Scalars['Boolean']>;
  finalStepCreate?: InputMaybe<RunStepCreateInput>;
  finalStepUpdate?: InputMaybe<RunStepUpdateInput>;
  id: Scalars['ID'];
  title: Scalars['String'];
  version: Scalars['String'];
  wasSuccessful?: InputMaybe<Scalars['Boolean']>;
};

export type RunCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type RunCreateInput = {
  id?: InputMaybe<Scalars['ID']>;
  routineId: Scalars['ID'];
  stepsCreate?: InputMaybe<Array<RunStepCreateInput>>;
  title: Scalars['String'];
  version: Scalars['String'];
};

export type RunEdge = {
  __typename?: 'RunEdge';
  cursor: Scalars['String'];
  node: Run;
};

export type RunSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  completedTimeFrame?: InputMaybe<TimeFrame>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  excludeIds?: InputMaybe<Array<Scalars['ID']>>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RunSortBy>;
  startedTimeFrame?: InputMaybe<TimeFrame>;
  status?: InputMaybe<RunStatus>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type RunSearchResult = {
  __typename?: 'RunSearchResult';
  edges: Array<RunEdge>;
  pageInfo: PageInfo;
};

export enum RunSortBy {
  DateCompletedAsc = 'DateCompletedAsc',
  DateCompletedDesc = 'DateCompletedDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateStartedAsc = 'DateStartedAsc',
  DateStartedDesc = 'DateStartedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export enum RunStatus {
  Cancelled = 'Cancelled',
  Completed = 'Completed',
  Failed = 'Failed',
  InProgress = 'InProgress',
  Scheduled = 'Scheduled'
}

export type RunStep = {
  __typename?: 'RunStep';
  contextSwitches: Scalars['Int'];
  id: Scalars['ID'];
  node?: Maybe<Node>;
  order: Scalars['Int'];
  run: Run;
  status: RunStepStatus;
  step: Array<Scalars['Int']>;
  subroutine?: Maybe<Routine>;
  timeCompleted?: Maybe<Scalars['Date']>;
  timeElapsed?: Maybe<Scalars['Int']>;
  timeStarted?: Maybe<Scalars['Date']>;
  title: Scalars['String'];
};

export type RunStepCreateInput = {
  contextSwitches?: InputMaybe<Scalars['Int']>;
  id?: InputMaybe<Scalars['ID']>;
  nodeId?: InputMaybe<Scalars['ID']>;
  order: Scalars['Int'];
  step: Array<Scalars['Int']>;
  subroutineId?: InputMaybe<Scalars['ID']>;
  timeElapsed?: InputMaybe<Scalars['Int']>;
  title: Scalars['String'];
};

export enum RunStepStatus {
  Completed = 'Completed',
  InProgress = 'InProgress',
  Skipped = 'Skipped'
}

export type RunStepUpdateInput = {
  contextSwitches?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  status?: InputMaybe<RunStepStatus>;
  timeElapsed?: InputMaybe<Scalars['Int']>;
};

export type RunUpdateInput = {
  completedComplexity?: InputMaybe<Scalars['Int']>;
  contextSwitches?: InputMaybe<Scalars['Int']>;
  id: Scalars['ID'];
  stepsCreate?: InputMaybe<Array<RunStepCreateInput>>;
  stepsDelete?: InputMaybe<Array<Scalars['ID']>>;
  stepsUpdate?: InputMaybe<Array<RunStepUpdateInput>>;
  timeElapsed?: InputMaybe<Scalars['Int']>;
};

export type SendVerificationEmailInput = {
  emailAddress: Scalars['String'];
};

export type Session = {
  __typename?: 'Session';
  id?: Maybe<Scalars['ID']>;
  languages?: Maybe<Array<Scalars['String']>>;
  roles: Array<Scalars['String']>;
  theme: Scalars['String'];
};

export type Standard = {
  __typename?: 'Standard';
  comments: Array<Comment>;
  commentsCount: Scalars['Int'];
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  default?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isInternal: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  isViewed: Scalars['Boolean'];
  name: Scalars['String'];
  props: Scalars['String'];
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists: Array<ResourceList>;
  role?: Maybe<MemberRole>;
  routineInputs: Array<Routine>;
  routineOutputs: Array<Routine>;
  score: Scalars['Int'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  translations: Array<StandardTranslation>;
  type: Scalars['String'];
  updated_at: Scalars['Date'];
  version: Scalars['String'];
  views: Scalars['Int'];
  yup?: Maybe<Scalars['String']>;
};

export type StandardCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type StandardCreateInput = {
  createdByOrganizationId?: InputMaybe<Scalars['ID']>;
  createdByUserId?: InputMaybe<Scalars['ID']>;
  default?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
  isInternal?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  props: Scalars['String'];
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  translationsCreate?: InputMaybe<Array<StandardTranslationCreateInput>>;
  type: Scalars['String'];
  version?: InputMaybe<Scalars['String']>;
  yup?: InputMaybe<Scalars['String']>;
};

export type StandardEdge = {
  __typename?: 'StandardEdge';
  cursor: Scalars['String'];
  node: Standard;
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
};

export type StandardSearchResult = {
  __typename?: 'StandardSearchResult';
  edges: Array<StandardEdge>;
  pageInfo: PageInfo;
};

export enum StandardSortBy {
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

export type StandardTranslation = {
  __typename?: 'StandardTranslation';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  jsonVariable?: Maybe<Scalars['String']>;
  language: Scalars['String'];
};

export type StandardTranslationCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
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
  id: Scalars['ID'];
  makeAnonymous?: InputMaybe<Scalars['Boolean']>;
  resourceListsCreate?: InputMaybe<Array<ResourceListCreateInput>>;
  resourceListsDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourceListsUpdate?: InputMaybe<Array<ResourceListUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  translationsCreate?: InputMaybe<Array<StandardTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<StandardTranslationUpdateInput>>;
};

export type Star = {
  __typename?: 'Star';
  from: User;
  id: Scalars['ID'];
  to: StarTo;
};

export enum StarFor {
  Comment = 'Comment',
  Organization = 'Organization',
  Project = 'Project',
  Routine = 'Routine',
  Standard = 'Standard',
  Tag = 'Tag',
  User = 'User'
}

export type StarInput = {
  forId: Scalars['ID'];
  isStar: Scalars['Boolean'];
  starFor: StarFor;
};

export type StarTo = Comment | Organization | Project | Routine | Standard | Tag | User;

export type StatisticsPageInput = {
  searchString: Scalars['String'];
  take?: InputMaybe<Scalars['Int']>;
};

export type StatisticsPageResult = {
  __typename?: 'StatisticsPageResult';
  allTime: StatisticsTimeFrame;
  daily: StatisticsTimeFrame;
  monthly: StatisticsTimeFrame;
  weekly: StatisticsTimeFrame;
  yearly: StatisticsTimeFrame;
};

export type StatisticsTimeFrame = {
  __typename?: 'StatisticsTimeFrame';
  organizations: Array<Scalars['Int']>;
  projects: Array<Scalars['Int']>;
  routines: Array<Scalars['Int']>;
  standards: Array<Scalars['Int']>;
  users: Array<Scalars['Int']>;
};

export type StepInputData = {
  __typename?: 'StepInputData';
  id: Scalars['ID'];
  inputs: Array<StepInputDataInput>;
  nodeId: Scalars['ID'];
  routineId: Scalars['ID'];
  runId: Scalars['ID'];
  stepId: Scalars['ID'];
  subroutineId?: Maybe<Scalars['ID']>;
};

export type StepInputDataCreateInput = {
  inputsCreate?: InputMaybe<Array<StepInputDataInputsCreateInput>>;
  nodeId: Scalars['ID'];
  routineId: Scalars['ID'];
  runId: Scalars['ID'];
  stepId: Scalars['ID'];
  subroutineId?: InputMaybe<Scalars['ID']>;
};

export type StepInputDataEdge = {
  __typename?: 'StepInputDataEdge';
  cursor: Scalars['String'];
  node: StepInputData;
};

export type StepInputDataInput = {
  __typename?: 'StepInputDataInput';
  id: Scalars['ID'];
  inputId: Scalars['ID'];
  name: Scalars['String'];
  standardId?: Maybe<Scalars['ID']>;
  value: Scalars['String'];
};

export type StepInputDataInputsCreateInput = {
  inputId: Scalars['ID'];
  name: Scalars['String'];
  standardId?: InputMaybe<Scalars['ID']>;
  value: Scalars['String'];
};

export type StepInputDataInputsUpdateInput = {
  inputId: Scalars['ID'];
  value: Scalars['String'];
};

export type StepInputDataSearchInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  nodeId?: InputMaybe<Scalars['ID']>;
  runId?: InputMaybe<Scalars['ID']>;
  sortBy?: InputMaybe<StepInputDataSortBy>;
  stepId?: InputMaybe<Scalars['ID']>;
  subroutineId?: InputMaybe<Scalars['ID']>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type StepInputDataSearchResult = {
  __typename?: 'StepInputDataSearchResult';
  edges: Array<StepInputDataEdge>;
  pageInfo: PageInfo;
};

export enum StepInputDataSortBy {
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type StepInputDataUpdateInput = {
  inputsCreate?: InputMaybe<Array<StepInputDataInputsCreateInput>>;
  inputsUpdate?: InputMaybe<Array<StepInputDataInputsUpdateInput>>;
  stepId: Scalars['ID'];
};

export type Success = {
  __typename?: 'Success';
  success?: Maybe<Scalars['Boolean']>;
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

export type TagCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type TagCreateInput = {
  anonymous?: InputMaybe<Scalars['Boolean']>;
  id?: InputMaybe<Scalars['ID']>;
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
  id?: InputMaybe<Scalars['ID']>;
  isBlur?: InputMaybe<Scalars['Boolean']>;
  tagConnect?: InputMaybe<Scalars['ID']>;
  tagCreate?: InputMaybe<TagCreateInput>;
};

export type TagHiddenUpdateInput = {
  id?: InputMaybe<Scalars['ID']>;
  isBlur?: InputMaybe<Scalars['Boolean']>;
};

export type TagSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
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
  id?: InputMaybe<Scalars['ID']>;
  language: Scalars['String'];
};

export type TagTranslationUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language?: InputMaybe<Scalars['String']>;
};

export type TagUpdateInput = {
  anonymous?: InputMaybe<Scalars['Boolean']>;
  id: Scalars['ID'];
  tag?: InputMaybe<Scalars['String']>;
  translationsCreate?: InputMaybe<Array<TagTranslationCreateInput>>;
  translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
  translationsUpdate?: InputMaybe<Array<TagTranslationUpdateInput>>;
};

export type TimeFrame = {
  after?: InputMaybe<Scalars['Date']>;
  before?: InputMaybe<Scalars['Date']>;
};

export type User = {
  __typename?: 'User';
  comments: Array<Comment>;
  created_at: Scalars['Date'];
  handle?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isStarred: Scalars['Boolean'];
  isViewed: Scalars['Boolean'];
  name: Scalars['String'];
  projects: Array<Project>;
  projectsCreated: Array<Project>;
  reports: Array<Report>;
  reportsCount: Scalars['Int'];
  resourceLists: Array<ResourceList>;
  routines: Array<Routine>;
  routinesCreated: Array<Routine>;
  starredBy: Array<User>;
  stars: Scalars['Int'];
  translations: Array<UserTranslation>;
  views: Scalars['Int'];
};

export type UserCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type UserDeleteInput = {
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
  id?: InputMaybe<Scalars['ID']>;
  language: Scalars['String'];
};

export type UserTranslationUpdateInput = {
  bio?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  language: Scalars['String'];
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

export type Vote = {
  __typename?: 'Vote';
  from: User;
  isUpvote?: Maybe<Scalars['Boolean']>;
  to: VoteTo;
};

export enum VoteFor {
  Comment = 'Comment',
  Project = 'Project',
  Routine = 'Routine',
  Standard = 'Standard',
  Tag = 'Tag'
}

export type VoteInput = {
  forId: Scalars['ID'];
  isUpvote?: InputMaybe<Scalars['Boolean']>;
  voteFor: VoteFor;
};

export type VoteTo = Comment | Project | Routine | Standard | Tag;

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
