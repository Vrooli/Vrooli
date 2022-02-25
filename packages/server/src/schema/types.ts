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

export type AutocompleteInput = {
  searchString: Scalars['String'];
  take?: InputMaybe<Scalars['Int']>;
};

export type AutocompleteResult = {
  __typename?: 'AutocompleteResult';
  organizations: Array<Organization>;
  projects: Array<Project>;
  routines: Array<Routine>;
  standards: Array<Standard>;
  users: Array<User>;
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
  role?: Maybe<MemberRole>;
  score?: Maybe<Scalars['Int']>;
  starredBy?: Maybe<Array<User>>;
  stars?: Maybe<Scalars['Int']>;
  text?: Maybe<Scalars['String']>;
  updated_at: Scalars['Date'];
};

export type CommentCreateInput = {
  createdFor: CommentFor;
  forId: Scalars['ID'];
  text: Scalars['String'];
};

export enum CommentFor {
  Organization = 'Organization',
  Project = 'Project',
  Routine = 'Routine',
  Standard = 'Standard',
  User = 'User'
}

export type CommentUpdateInput = {
  id: Scalars['ID'];
  text?: InputMaybe<Scalars['String']>;
};

export type CommentedOn = Project | Routine | Standard;

export type Contributor = Organization | User;

export type Count = {
  __typename?: 'Count';
  count?: Maybe<Scalars['Int']>;
};

export type DeleteManyInput = {
  ids: Array<Scalars['ID']>;
};

export type DeleteOneInput = {
  id: Scalars['ID'];
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
  password: Scalars['String'];
  theme: Scalars['String'];
  username: Scalars['String'];
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

export type InputItem = {
  __typename?: 'InputItem';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isRequired?: Maybe<Scalars['Boolean']>;
  name?: Maybe<Scalars['String']>;
  routine: Routine;
  standard?: Maybe<Standard>;
};

export type InputItemCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  isRequired?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  standardConnect?: InputMaybe<Scalars['ID']>;
  standardCreate?: InputMaybe<StandardCreateInput>;
};

export type InputItemUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isRequired?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  standardConnect?: InputMaybe<Scalars['ID']>;
  standardCreate?: InputMaybe<StandardCreateInput>;
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
  commentDeleteOne: Success;
  commentUpdate: Comment;
  emailCreate: Email;
  emailDeleteOne: Success;
  emailLogIn: Session;
  emailRequestPasswordChange: Success;
  emailResetPassword: Session;
  emailSignUp: Session;
  emailUpdate: Email;
  exportData: Scalars['String'];
  feedbackCreate: Success;
  guestLogIn: Session;
  logOut: Success;
  nodeCreate: Node;
  nodeDeleteOne: Success;
  nodeUpdate: Node;
  organizationCreate: Organization;
  organizationDeleteOne: Success;
  organizationUpdate: Organization;
  profileEmailUpdate: Profile;
  profileUpdate: Profile;
  projectCreate: Project;
  projectDeleteOne: Success;
  projectUpdate: Project;
  reportCreate: Report;
  reportDeleteOne: Success;
  reportUpdate: Report;
  resourceCreate: Resource;
  resourceDeleteMany: Count;
  resourceUpdate: Resource;
  routineCreate: Routine;
  routineDeleteOne: Success;
  routineUpdate: Routine;
  standardCreate: Standard;
  standardDeleteOne: Success;
  standardUpdate: Standard;
  star: Success;
  tagCreate: Tag;
  tagDeleteMany: Count;
  tagUpdate: Tag;
  userDeleteOne: Success;
  validateSession: Session;
  vote: Success;
  walletComplete: WalletComplete;
  walletInit: Scalars['String'];
  walletRemove: Success;
  writeAssets?: Maybe<Scalars['Boolean']>;
};


export type MutationCommentCreateArgs = {
  input: CommentCreateInput;
};


export type MutationCommentDeleteOneArgs = {
  input: DeleteOneInput;
};


export type MutationCommentUpdateArgs = {
  input: CommentUpdateInput;
};


export type MutationEmailCreateArgs = {
  input: EmailCreateInput;
};


export type MutationEmailDeleteOneArgs = {
  input: DeleteOneInput;
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


export type MutationNodeCreateArgs = {
  input: NodeCreateInput;
};


export type MutationNodeDeleteOneArgs = {
  input: DeleteOneInput;
};


export type MutationNodeUpdateArgs = {
  input: NodeUpdateInput;
};


export type MutationOrganizationCreateArgs = {
  input: OrganizationCreateInput;
};


export type MutationOrganizationDeleteOneArgs = {
  input?: InputMaybe<DeleteOneInput>;
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


export type MutationProjectDeleteOneArgs = {
  input: DeleteOneInput;
};


export type MutationProjectUpdateArgs = {
  input: ProjectUpdateInput;
};


export type MutationReportCreateArgs = {
  input: ReportCreateInput;
};


export type MutationReportDeleteOneArgs = {
  input: DeleteOneInput;
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


export type MutationStandardCreateArgs = {
  input: StandardCreateInput;
};


export type MutationStandardDeleteOneArgs = {
  input: DeleteOneInput;
};


export type MutationStandardUpdateArgs = {
  input: StandardUpdateInput;
};


export type MutationStarArgs = {
  input: StarInput;
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


export type MutationWalletRemoveArgs = {
  input: DeleteOneInput;
};


export type MutationWriteAssetsArgs = {
  input: WriteAssetsInput;
};

export type Node = {
  __typename?: 'Node';
  created_at: Scalars['Date'];
  data?: Maybe<NodeData>;
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  routine: Routine;
  routineId: Scalars['ID'];
  title: Scalars['String'];
  type: NodeType;
  updated_at: Scalars['Date'];
};

export type NodeCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  nodeEndCreate?: InputMaybe<NodeEndCreateInput>;
  nodeLoopCreate?: InputMaybe<NodeLoopCreateInput>;
  nodeRoutineListCreate?: InputMaybe<NodeRoutineListCreateInput>;
  routineId: Scalars['ID'];
  title?: InputMaybe<Scalars['String']>;
  type?: InputMaybe<NodeType>;
};

export type NodeData = NodeEnd | NodeLoop | NodeRoutineList;

export type NodeEnd = {
  __typename?: 'NodeEnd';
  id: Scalars['ID'];
  wasSuccessful: Scalars['Boolean'];
};

export type NodeEndCreateInput = {
  wasSuccessful?: InputMaybe<Scalars['Boolean']>;
};

export type NodeEndUpdateInput = {
  id: Scalars['ID'];
  wasSuccessful?: InputMaybe<Scalars['Boolean']>;
};

export type NodeLink = {
  __typename?: 'NodeLink';
  conditions: Array<NodeLinkCondition>;
  fromId: Scalars['ID'];
  id: Scalars['ID'];
  toId: Scalars['ID'];
};

export type NodeLinkCondition = {
  __typename?: 'NodeLinkCondition';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  title: Scalars['String'];
  when: Array<NodeLinkConditionCase>;
};

export type NodeLinkConditionCase = {
  __typename?: 'NodeLinkConditionCase';
  condition: Scalars['String'];
  id: Scalars['ID'];
};

export type NodeLinkConditionCaseCreateInput = {
  condition: Scalars['String'];
};

export type NodeLinkConditionCaseUpdateInput = {
  condition?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
};

export type NodeLinkConditionCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  title: Scalars['String'];
  toId?: InputMaybe<Scalars['ID']>;
  whenCreate: Array<NodeLinkConditionCaseCreateInput>;
};

export type NodeLinkConditionUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  title?: InputMaybe<Scalars['String']>;
  toId?: InputMaybe<Scalars['ID']>;
  whenCreate?: InputMaybe<Array<NodeLinkConditionCaseCreateInput>>;
  whenDelete?: InputMaybe<Array<Scalars['ID']>>;
  whenUpdate?: InputMaybe<Array<NodeLinkConditionCaseUpdateInput>>;
};

export type NodeLinkCreateInput = {
  conditions: Array<NodeLinkConditionCreateInput>;
  fromId: Scalars['ID'];
  toId: Scalars['ID'];
};

export type NodeLinkUpdateInput = {
  conditionsCreate?: InputMaybe<Array<NodeLinkConditionCreateInput>>;
  conditionsDelete?: InputMaybe<Array<Scalars['ID']>>;
  conditionsUpdate?: InputMaybe<Array<NodeLinkConditionUpdateInput>>;
  fromId?: InputMaybe<Scalars['ID']>;
  id: Scalars['ID'];
  toId?: InputMaybe<Scalars['ID']>;
};

export type NodeLoop = {
  __typename?: 'NodeLoop';
  id: Scalars['ID'];
  loops?: Maybe<Scalars['Int']>;
  maxLoops?: Maybe<Scalars['Int']>;
  whiles: Array<NodeLoopWhile>;
};

export type NodeLoopCreateInput = {
  loops?: InputMaybe<Scalars['Int']>;
  maxLoops?: InputMaybe<Scalars['Int']>;
  whilesCreate: Array<NodeLoopWhileCreateInput>;
};

export type NodeLoopUpdateInput = {
  id: Scalars['ID'];
  loops?: InputMaybe<Scalars['Int']>;
  maxLoops?: InputMaybe<Scalars['Int']>;
  whilesCreate: Array<NodeLoopWhileCreateInput>;
  whilesDelete?: InputMaybe<Array<Scalars['ID']>>;
  whilesUpdate: Array<NodeLoopWhileUpdateInput>;
};

export type NodeLoopWhile = {
  __typename?: 'NodeLoopWhile';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  title: Scalars['String'];
  toId?: Maybe<Scalars['ID']>;
  when: Array<NodeLoopWhileCase>;
};

export type NodeLoopWhileCase = {
  __typename?: 'NodeLoopWhileCase';
  condition: Scalars['String'];
  id: Scalars['ID'];
};

export type NodeLoopWhileCaseCreateInput = {
  condition: Scalars['String'];
};

export type NodeLoopWhileCaseUpdateInput = {
  condition?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
};

export type NodeLoopWhileCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  title: Scalars['String'];
  toId?: InputMaybe<Scalars['ID']>;
  whenCreate: Array<NodeLoopWhileCaseCreateInput>;
};

export type NodeLoopWhileUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  title?: InputMaybe<Scalars['String']>;
  toId?: InputMaybe<Scalars['ID']>;
  whenCreate?: InputMaybe<Array<NodeLoopWhileCaseCreateInput>>;
  whenDelete?: InputMaybe<Array<Scalars['ID']>>;
  whenUpdate?: InputMaybe<Array<NodeLoopWhileCaseUpdateInput>>;
};

export type NodeRoutineList = {
  __typename?: 'NodeRoutineList';
  id: Scalars['ID'];
  isOptional: Scalars['Boolean'];
  isOrdered: Scalars['Boolean'];
  routines: Array<NodeRoutineListItem>;
};

export type NodeRoutineListCreateInput = {
  isOptional?: InputMaybe<Scalars['Boolean']>;
  isOrdered?: InputMaybe<Scalars['Boolean']>;
  routinesConnect?: InputMaybe<Array<Scalars['ID']>>;
  routinesCreate?: InputMaybe<Array<NodeRoutineListItemCreateInput>>;
};

export type NodeRoutineListItem = {
  __typename?: 'NodeRoutineListItem';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isOptional: Scalars['Boolean'];
  routine: Routine;
  title?: Maybe<Scalars['String']>;
};

export type NodeRoutineListItemCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  isOptional?: InputMaybe<Scalars['Boolean']>;
  routineConnect: Scalars['ID'];
  title?: InputMaybe<Scalars['String']>;
};

export type NodeRoutineListItemUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  isOptional?: InputMaybe<Scalars['Boolean']>;
  routineConnect?: InputMaybe<Scalars['ID']>;
  title?: InputMaybe<Scalars['String']>;
};

export type NodeRoutineListUpdateInput = {
  id: Scalars['ID'];
  isOptional?: InputMaybe<Scalars['Boolean']>;
  isOrdered?: InputMaybe<Scalars['Boolean']>;
  routinesConnect?: InputMaybe<Array<Scalars['ID']>>;
  routinesCreate?: InputMaybe<Array<NodeRoutineListItemCreateInput>>;
  routinesDelete?: InputMaybe<Array<Scalars['ID']>>;
  routinesDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  routinesUpdate?: InputMaybe<Array<NodeRoutineListItemUpdateInput>>;
};

export enum NodeType {
  End = 'End',
  Loop = 'Loop',
  Redirect = 'Redirect',
  RoutineList = 'RoutineList',
  Start = 'Start'
}

export type NodeUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  nodeEndCreate?: InputMaybe<NodeEndCreateInput>;
  nodeEndUpdate?: InputMaybe<NodeEndUpdateInput>;
  nodeLoopCreate?: InputMaybe<NodeLoopCreateInput>;
  nodeLoopUpdate?: InputMaybe<NodeLoopUpdateInput>;
  nodeRoutineListCreate?: InputMaybe<NodeRoutineListCreateInput>;
  nodeRoutineListUpdate?: InputMaybe<NodeRoutineListUpdateInput>;
  routineId?: InputMaybe<Scalars['ID']>;
  title?: InputMaybe<Scalars['String']>;
  type?: InputMaybe<NodeType>;
};

export type OpenGraphResponse = {
  __typename?: 'OpenGraphResponse';
  description?: Maybe<Scalars['String']>;
  imageUrl?: Maybe<Scalars['String']>;
  site?: Maybe<Scalars['String']>;
  title?: Maybe<Scalars['String']>;
};

export type Organization = {
  __typename?: 'Organization';
  bio?: Maybe<Scalars['String']>;
  comments: Array<Comment>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isStarred: Scalars['Boolean'];
  members: Array<Member>;
  name: Scalars['String'];
  projects: Array<Project>;
  reports: Array<Report>;
  resources: Array<Resource>;
  role?: Maybe<MemberRole>;
  routines: Array<Routine>;
  routinesCreated: Array<Routine>;
  starredBy: Array<User>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  updated_at: Scalars['Date'];
  wallets: Array<Wallet>;
};

export type OrganizationCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type OrganizationCreateInput = {
  bio?: InputMaybe<Scalars['String']>;
  name: Scalars['String'];
  resourcesCreate?: InputMaybe<Array<ResourceCreateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
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
  projectId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<OrganizationSortBy>;
  standardId?: InputMaybe<Scalars['ID']>;
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
  AlphabeticalAsc = 'AlphabeticalAsc',
  AlphabeticalDesc = 'AlphabeticalDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc'
}

export type OrganizationUpdateInput = {
  bio?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  membersConnect?: InputMaybe<Array<Scalars['ID']>>;
  membersDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  name?: InputMaybe<Scalars['String']>;
  resourcesCreate?: InputMaybe<Array<ResourceCreateInput>>;
  resourcesDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourcesUpdate?: InputMaybe<Array<ResourceUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
};

export type OutputItem = {
  __typename?: 'OutputItem';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  name?: Maybe<Scalars['String']>;
  routine: Routine;
  standard?: Maybe<Standard>;
};

export type OutputItemCreateInput = {
  description?: InputMaybe<Scalars['String']>;
  name?: InputMaybe<Scalars['String']>;
  standardConnect?: InputMaybe<Scalars['ID']>;
  standardCreate?: InputMaybe<StandardCreateInput>;
};

export type OutputItemUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  name?: InputMaybe<Scalars['String']>;
  standardConnect?: InputMaybe<Scalars['ID']>;
  standardCreate?: InputMaybe<StandardCreateInput>;
};

export type PageInfo = {
  __typename?: 'PageInfo';
  endCursor?: Maybe<Scalars['String']>;
  hasNextPage: Scalars['Boolean'];
};

export type Profile = {
  __typename?: 'Profile';
  bio?: Maybe<Scalars['String']>;
  comments: Array<Comment>;
  created_at: Scalars['Date'];
  emails: Array<Email>;
  hiddenTags?: Maybe<Array<Tag>>;
  id: Scalars['ID'];
  projects: Array<Project>;
  projectsCreated: Array<Project>;
  reports: Array<Report>;
  resources: Array<Resource>;
  roles: Array<Role>;
  routines: Array<Routine>;
  routinesCreated: Array<Routine>;
  sentReports: Array<Report>;
  starredBy: Array<User>;
  starredTags?: Maybe<Array<Tag>>;
  stars: Array<Star>;
  status: AccountStatus;
  theme: Scalars['String'];
  updated_at: Scalars['Date'];
  username?: Maybe<Scalars['String']>;
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
  bio?: InputMaybe<Scalars['String']>;
  hiddenTagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  hiddenTagsCreate?: InputMaybe<Array<TagCreateInput>>;
  hiddenTagsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  starredTagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  starredTagsCreate?: InputMaybe<Array<TagCreateInput>>;
  starredTagsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  theme?: InputMaybe<Scalars['String']>;
  username?: InputMaybe<Scalars['String']>;
};

export type Project = {
  __typename?: 'Project';
  comments: Array<Comment>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  description?: Maybe<Scalars['String']>;
  forks: Array<Project>;
  id: Scalars['ID'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  name: Scalars['String'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  reports: Array<Report>;
  resources?: Maybe<Array<Resource>>;
  role?: Maybe<MemberRole>;
  routines: Array<Routine>;
  score: Scalars['Int'];
  starredBy?: Maybe<Array<User>>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  updated_at: Scalars['Date'];
  wallets?: Maybe<Array<Wallet>>;
};

export type ProjectCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type ProjectCreateInput = {
  createdByOrganizationId?: InputMaybe<Scalars['ID']>;
  createdByUserId?: InputMaybe<Scalars['ID']>;
  description?: InputMaybe<Scalars['String']>;
  name: Scalars['String'];
  parentId?: InputMaybe<Scalars['ID']>;
  resourcesCreate?: InputMaybe<Array<ResourceCreateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
};

export type ProjectEdge = {
  __typename?: 'ProjectEdge';
  cursor: Scalars['String'];
  node: Project;
};

export type ProjectSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<ProjectSortBy>;
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
  AlphabeticalAsc = 'AlphabeticalAsc',
  AlphabeticalDesc = 'AlphabeticalDesc',
  CommentsAsc = 'CommentsAsc',
  CommentsDesc = 'CommentsDesc',
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

export type ProjectUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  name?: InputMaybe<Scalars['String']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  resourcesCreate?: InputMaybe<Array<ResourceCreateInput>>;
  resourcesDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourcesUpdate?: InputMaybe<Array<ResourceUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type Query = {
  __typename?: 'Query';
  autocomplete: AutocompleteResult;
  organization?: Maybe<Organization>;
  organizations: OrganizationSearchResult;
  organizationsCount: Scalars['Int'];
  profile: Profile;
  project?: Maybe<Project>;
  projects: ProjectSearchResult;
  projectsCount: Scalars['Int'];
  readAssets: Array<Maybe<Scalars['String']>>;
  readOpenGraph: OpenGraphResponse;
  resource?: Maybe<Resource>;
  resources: ResourceSearchResult;
  resourcesCount: Scalars['Int'];
  routine?: Maybe<Routine>;
  routines: RoutineSearchResult;
  routinesCount: Scalars['Int'];
  standard?: Maybe<Standard>;
  standards: StandardSearchResult;
  standardsCount: Scalars['Int'];
  statistics: StatisticsResult;
  tag?: Maybe<Tag>;
  tags: TagSearchResult;
  tagsCount: Scalars['Int'];
  user?: Maybe<User>;
  users: UserSearchResult;
  usersCount: Scalars['Int'];
};


export type QueryAutocompleteArgs = {
  input: AutocompleteInput;
};


export type QueryOrganizationArgs = {
  input: FindByIdInput;
};


export type QueryOrganizationsArgs = {
  input: OrganizationSearchInput;
};


export type QueryOrganizationsCountArgs = {
  input: OrganizationCountInput;
};


export type QueryProjectArgs = {
  input: FindByIdInput;
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


export type QueryReadOpenGraphArgs = {
  input: ReadOpenGraphInput;
};


export type QueryResourceArgs = {
  input: FindByIdInput;
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


export type QueryStandardArgs = {
  input: FindByIdInput;
};


export type QueryStandardsArgs = {
  input: StandardSearchInput;
};


export type QueryStandardsCountArgs = {
  input: StandardCountInput;
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
  input: FindByIdInput;
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

export type ReadOpenGraphInput = {
  url: Scalars['String'];
};

export type Report = {
  __typename?: 'Report';
  details?: Maybe<Scalars['String']>;
  id?: Maybe<Scalars['ID']>;
  isOwn: Scalars['Boolean'];
  reason: Scalars['String'];
};

export type ReportCreateInput = {
  createdFor: ReportFor;
  createdForId: Scalars['ID'];
  details?: InputMaybe<Scalars['String']>;
  reason: Scalars['String'];
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

export type ReportUpdateInput = {
  details?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  reason?: InputMaybe<Scalars['String']>;
};

export type Resource = {
  __typename?: 'Resource';
  createdFor: ResourceFor;
  createdForId: Scalars['ID'];
  created_at: Scalars['Date'];
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  link: Scalars['String'];
  title?: Maybe<Scalars['String']>;
  updated_at: Scalars['Date'];
  usedFor?: Maybe<ResourceUsedFor>;
};

export type ResourceCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type ResourceCreateInput = {
  createdFor: ResourceFor;
  createdForId: Scalars['ID'];
  description?: InputMaybe<Scalars['String']>;
  link: Scalars['String'];
  title?: InputMaybe<Scalars['String']>;
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
  RoutineContextual = 'RoutineContextual',
  RoutineExternal = 'RoutineExternal',
  User = 'User'
}

export type ResourceSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  forId?: InputMaybe<Scalars['ID']>;
  forType?: InputMaybe<ResourceFor>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
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
  AlphabeticalAsc = 'AlphabeticalAsc',
  AlphabeticalDesc = 'AlphabeticalDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc'
}

export type ResourceUpdateInput = {
  createdFor?: InputMaybe<ResourceFor>;
  createdForId?: InputMaybe<Scalars['ID']>;
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  link?: InputMaybe<Scalars['String']>;
  title?: InputMaybe<Scalars['String']>;
  usedFor?: InputMaybe<ResourceUsedFor>;
};

export enum ResourceUsedFor {
  Community = 'Community',
  Context = 'Context',
  Donation = 'Donation',
  Learning = 'Learning',
  OfficialWebsite = 'OfficialWebsite',
  Proposal = 'Proposal',
  Related = 'Related',
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
  contextualResources: Array<Resource>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  description?: Maybe<Scalars['String']>;
  externalResources: Array<Resource>;
  forks: Array<Routine>;
  id: Scalars['ID'];
  inputs: Array<InputItem>;
  instructions?: Maybe<Scalars['String']>;
  isAutomatable?: Maybe<Scalars['Boolean']>;
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  nodeLinks: Array<NodeLink>;
  nodeLists: Array<NodeRoutineList>;
  nodes: Array<Node>;
  outputs: Array<OutputItem>;
  owner?: Maybe<Contributor>;
  parent?: Maybe<Routine>;
  project?: Maybe<Project>;
  reports: Array<Report>;
  role?: Maybe<MemberRole>;
  score: Scalars['Int'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  title?: Maybe<Scalars['String']>;
  updated_at: Scalars['Date'];
  version?: Maybe<Scalars['String']>;
};

export type RoutineCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type RoutineCreateInput = {
  createdByOrganizationId?: InputMaybe<Scalars['ID']>;
  createdByUserId?: InputMaybe<Scalars['ID']>;
  description?: InputMaybe<Scalars['String']>;
  inputsCreate?: InputMaybe<Array<InputItemCreateInput>>;
  instructions?: InputMaybe<Scalars['String']>;
  isAutomatable?: InputMaybe<Scalars['Boolean']>;
  nodeLinksCreate?: InputMaybe<Array<NodeLinkCreateInput>>;
  nodesCreate?: InputMaybe<Array<NodeCreateInput>>;
  outputsCreate?: InputMaybe<Array<OutputItemCreateInput>>;
  parentId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  resourcesContextualCreate?: InputMaybe<Array<ResourceCreateInput>>;
  resourcesExternalCreate?: InputMaybe<Array<ResourceCreateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  title: Scalars['String'];
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
  ids?: InputMaybe<Array<Scalars['ID']>>;
  organizationId?: InputMaybe<Scalars['ID']>;
  parentId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<RoutineSortBy>;
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
  AlphabeticalAsc = 'AlphabeticalAsc',
  AlphabeticalDesc = 'AlphabeticalDesc',
  CommentsAsc = 'CommentsAsc',
  CommentsDesc = 'CommentsDesc',
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

export type RoutineUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  inputsCreate?: InputMaybe<Array<InputItemCreateInput>>;
  inputsDelete?: InputMaybe<Array<Scalars['ID']>>;
  inputsUpdate?: InputMaybe<Array<InputItemUpdateInput>>;
  instructions?: InputMaybe<Scalars['String']>;
  isAutomatable?: InputMaybe<Scalars['Boolean']>;
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
  resourcesContextualCreate?: InputMaybe<Array<ResourceCreateInput>>;
  resourcesContextualDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourcesContextualUpdate?: InputMaybe<Array<ResourceUpdateInput>>;
  resourcesExternalCreate?: InputMaybe<Array<ResourceCreateInput>>;
  resourcesExternalDelete?: InputMaybe<Array<Scalars['ID']>>;
  resourcesExternalUpdate?: InputMaybe<Array<ResourceUpdateInput>>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
  title?: InputMaybe<Scalars['String']>;
  userId?: InputMaybe<Scalars['ID']>;
  version?: InputMaybe<Scalars['String']>;
};

export type Session = {
  __typename?: 'Session';
  id?: Maybe<Scalars['ID']>;
  roles: Array<Scalars['String']>;
  theme: Scalars['String'];
};

export type Standard = {
  __typename?: 'Standard';
  comments: Array<Comment>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  default?: Maybe<Scalars['String']>;
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isFile: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  name: Scalars['String'];
  reports: Array<Report>;
  role?: Maybe<MemberRole>;
  routineInputs: Array<Routine>;
  routineOutputs: Array<Routine>;
  schema: Scalars['String'];
  score: Scalars['Int'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  tags: Array<Tag>;
  type: StandardType;
  updated_at: Scalars['Date'];
};

export type StandardCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type StandardCreateInput = {
  createdByOrganizationId?: InputMaybe<Scalars['ID']>;
  createdByUserId?: InputMaybe<Scalars['ID']>;
  default?: InputMaybe<Scalars['String']>;
  description?: InputMaybe<Scalars['String']>;
  isFile?: InputMaybe<Scalars['Boolean']>;
  name: Scalars['String'];
  schema?: InputMaybe<Scalars['String']>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  type?: InputMaybe<StandardType>;
  version?: InputMaybe<Scalars['String']>;
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
  organizationId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<StandardSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type StandardSearchResult = {
  __typename?: 'StandardSearchResult';
  edges: Array<StandardEdge>;
  pageInfo: PageInfo;
};

export enum StandardSortBy {
  AlphabeticalAsc = 'AlphabeticalAsc',
  AlphabeticalDesc = 'AlphabeticalDesc',
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

export enum StandardType {
  Array = 'Array',
  Boolean = 'Boolean',
  File = 'File',
  Number = 'Number',
  Object = 'Object',
  String = 'String',
  Url = 'Url'
}

export type StandardUpdateInput = {
  description?: InputMaybe<Scalars['String']>;
  id: Scalars['ID'];
  makeAnonymous?: InputMaybe<Scalars['Boolean']>;
  tagsConnect?: InputMaybe<Array<Scalars['ID']>>;
  tagsCreate?: InputMaybe<Array<TagCreateInput>>;
  tagsDisconnect?: InputMaybe<Array<Scalars['ID']>>;
};

export type Star = {
  __typename?: 'Star';
  from: User;
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

export type StarTo = Comment | Organization | Project | Routine | Standard | Tag;

export type StatisticsResult = {
  __typename?: 'StatisticsResult';
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

export type Success = {
  __typename?: 'Success';
  success?: Maybe<Scalars['Boolean']>;
};

export type Tag = {
  __typename?: 'Tag';
  created_at: Scalars['Date'];
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isOwn: Scalars['Boolean'];
  isStarred: Scalars['Boolean'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  tag: Scalars['String'];
  updated_at: Scalars['Date'];
};

export type TagCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type TagCreateInput = {
  anonymous?: InputMaybe<Scalars['Boolean']>;
  description?: InputMaybe<Scalars['String']>;
  tag: Scalars['String'];
};

export type TagEdge = {
  __typename?: 'TagEdge';
  cursor: Scalars['String'];
  node: Tag;
};

export type TagSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  hidden?: InputMaybe<Scalars['Boolean']>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
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
  AlphabeticalAsc = 'AlphabeticalAsc',
  AlphabeticalDesc = 'AlphabeticalDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc'
}

export type TagUpdateInput = {
  anonymous?: InputMaybe<Scalars['Boolean']>;
  description?: InputMaybe<Scalars['String']>;
  tag?: InputMaybe<Scalars['String']>;
};

export type TimeFrame = {
  after?: InputMaybe<Scalars['Date']>;
  before?: InputMaybe<Scalars['Date']>;
};

export type User = {
  __typename?: 'User';
  bio?: Maybe<Scalars['String']>;
  comments: Array<Comment>;
  created_at: Scalars['Date'];
  id: Scalars['ID'];
  isStarred: Scalars['Boolean'];
  projects: Array<Project>;
  reports: Array<Report>;
  resources: Array<Resource>;
  roles: Array<Role>;
  routines: Array<Routine>;
  routinesCreated: Array<Routine>;
  starredBy: Array<User>;
  stars: Scalars['Int'];
  username?: Maybe<Scalars['String']>;
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
  organizationId?: InputMaybe<Scalars['ID']>;
  projectId?: InputMaybe<Scalars['ID']>;
  reportId?: InputMaybe<Scalars['ID']>;
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
  AlphabeticalAsc = 'AlphabeticalAsc',
  AlphabeticalDesc = 'AlphabeticalDesc',
  DateCreatedAsc = 'DateCreatedAsc',
  DateCreatedDesc = 'DateCreatedDesc',
  DateUpdatedAsc = 'DateUpdatedAsc',
  DateUpdatedDesc = 'DateUpdatedDesc',
  StarsAsc = 'StarsAsc',
  StarsDesc = 'StarsDesc'
}

export type Vote = {
  __typename?: 'Vote';
  from: User;
  isUpvote?: Maybe<Scalars['Boolean']>;
  to: StarTo;
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
  id: Scalars['ID'];
  organization?: Maybe<Organization>;
  publicAddress: Scalars['String'];
  user?: Maybe<User>;
  verified: Scalars['Boolean'];
};

export type WalletComplete = {
  __typename?: 'WalletComplete';
  firstLogIn: Scalars['Boolean'];
  session?: Maybe<Session>;
};

export type WalletCompleteInput = {
  publicAddress: Scalars['String'];
  signedPayload: Scalars['String'];
};

export type WalletInitInput = {
  nonceDescription?: InputMaybe<Scalars['String']>;
  publicAddress: Scalars['String'];
};

export type WriteAssetsInput = {
  files: Array<Scalars['Upload']>;
};
