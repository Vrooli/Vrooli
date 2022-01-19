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
  isUpvoted?: Maybe<Scalars['Boolean']>;
  reports: Array<Report>;
  score?: Maybe<Scalars['Int']>;
  starredBy?: Maybe<Array<User>>;
  stars?: Maybe<Scalars['Int']>;
  text?: Maybe<Scalars['String']>;
  updated_at: Scalars['Date'];
};

export enum CommentFor {
  Organization = 'Organization',
  Project = 'Project',
  Routine = 'Routine',
  Standard = 'Standard',
  User = 'User'
}

export type CommentInput = {
  createdFor: CommentFor;
  forId: Scalars['ID'];
  id?: InputMaybe<Scalars['ID']>;
  text?: InputMaybe<Scalars['String']>;
};

export type CommentedOn = Project | Routine | Standard;

export type Contributor = Organization | User;

export type Count = {
  __typename?: 'Count';
  count?: Maybe<Scalars['Int']>;
};

export type DeleteCommentInput = {
  createdFor: CommentFor;
  forId: Scalars['ID'];
  id: Scalars['ID'];
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

export type EmailInput = {
  emailAddress: Scalars['String'];
  id?: InputMaybe<Scalars['ID']>;
  receivesAccountUpdates?: InputMaybe<Scalars['Boolean']>;
  receivesBusinessUpdates?: InputMaybe<Scalars['Boolean']>;
  userId?: InputMaybe<Scalars['ID']>;
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
  email: Scalars['String'];
  marketingEmails: Scalars['Boolean'];
  password: Scalars['String'];
  confirmPassword: Scalars['String'];
  theme: Scalars['String'];
  username: Scalars['String'];
};

export type FeedbackInput = {
  text: Scalars['String'];
  userId?: InputMaybe<Scalars['ID']>;
};

export type FindByIdInput = {
  id: Scalars['ID'];
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
  commentAdd: Comment;
  commentDeleteOne: Success;
  commentUpdate: Comment;
  emailAdd: Email;
  emailDeleteMany: Count;
  emailLogIn: Session;
  emailRequestPasswordChange: Success;
  emailResetPassword: Session;
  emailSignUp: Session;
  emailUpdate: Email;
  exportData: Scalars['String'];
  feedbackAdd: Success;
  guestLogIn: Session;
  logOut: Success;
  nodeAdd: Node;
  nodeDeleteOne: Success;
  nodeUpdate: Node;
  organizationAdd: Organization;
  organizationDeleteOne: Success;
  organizationUpdate: Organization;
  profileUpdate: Profile;
  projectAdd: Project;
  projectDeleteOne: Success;
  projectUpdate: Project;
  reportAdd: Report;
  reportDeleteOne: Success;
  reportUpdate: Report;
  resourceAdd: Resource;
  resourceDeleteMany: Count;
  resourceUpdate: Resource;
  routineAdd: Routine;
  routineDeleteOne: Success;
  routineUpdate: Routine;
  standardAdd: Standard;
  standardDeleteOne: Success;
  tagAdd: Tag;
  tagDeleteMany: Count;
  tagUpdate: Tag;
  userDeleteOne: Success;
  validateSession: Session;
  vote: Success;
  walletComplete: Session;
  walletInit: Scalars['String'];
  walletRemove: Success;
  writeAssets?: Maybe<Scalars['Boolean']>;
};


export type MutationCommentAddArgs = {
  input: CommentInput;
};


export type MutationCommentDeleteOneArgs = {
  input: DeleteCommentInput;
};


export type MutationCommentUpdateArgs = {
  input: CommentInput;
};


export type MutationEmailAddArgs = {
  input: EmailInput;
};


export type MutationEmailDeleteManyArgs = {
  input: DeleteManyInput;
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
  input: EmailInput;
};


export type MutationFeedbackAddArgs = {
  input: FeedbackInput;
};


export type MutationNodeAddArgs = {
  input: NodeInput;
};


export type MutationNodeDeleteOneArgs = {
  input: DeleteOneInput;
};


export type MutationNodeUpdateArgs = {
  input: NodeInput;
};


export type MutationOrganizationAddArgs = {
  input: OrganizationInput;
};


export type MutationOrganizationDeleteOneArgs = {
  input?: InputMaybe<DeleteOneInput>;
};


export type MutationOrganizationUpdateArgs = {
  input: OrganizationInput;
};


export type MutationProfileUpdateArgs = {
  input: ProfileUpdateInput;
};


export type MutationProjectAddArgs = {
  input: ProjectInput;
};


export type MutationProjectDeleteOneArgs = {
  input: DeleteOneInput;
};


export type MutationProjectUpdateArgs = {
  input: ProjectInput;
};


export type MutationReportAddArgs = {
  input: ReportInput;
};


export type MutationReportDeleteOneArgs = {
  input: DeleteOneInput;
};


export type MutationReportUpdateArgs = {
  input: ReportInput;
};


export type MutationResourceAddArgs = {
  input: ResourceInput;
};


export type MutationResourceDeleteManyArgs = {
  input: DeleteManyInput;
};


export type MutationResourceUpdateArgs = {
  input: ResourceInput;
};


export type MutationRoutineAddArgs = {
  input: RoutineInput;
};


export type MutationRoutineDeleteOneArgs = {
  input: DeleteOneInput;
};


export type MutationRoutineUpdateArgs = {
  input: RoutineInput;
};


export type MutationStandardAddArgs = {
  input: StandardInput;
};


export type MutationStandardDeleteOneArgs = {
  input: DeleteOneInput;
};


export type MutationTagAddArgs = {
  input: TagInput;
};


export type MutationTagDeleteManyArgs = {
  input: DeleteManyInput;
};


export type MutationTagUpdateArgs = {
  input: TagInput;
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
  DecisionItem: Array<NodeDecisionItem>;
  From: Array<Node>;
  Next: Array<Node>;
  Previous: Array<Node>;
  To: Array<Node>;
  created_at: Scalars['Date'];
  data?: Maybe<NodeData>;
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  next?: Maybe<Scalars['ID']>;
  previous?: Maybe<Scalars['ID']>;
  routine: Routine;
  routineId: Scalars['ID'];
  title: Scalars['String'];
  type: NodeType;
  updated_at: Scalars['Date'];
};

export type NodeCombine = {
  __typename?: 'NodeCombine';
  from: Array<Scalars['ID']>;
  id: Scalars['ID'];
  to: Scalars['ID'];
};

export type NodeCombineInput = {
  from?: InputMaybe<Array<Scalars['ID']>>;
  id?: InputMaybe<Scalars['ID']>;
  to?: InputMaybe<Scalars['ID']>;
};

export type NodeData = NodeCombine | NodeDecision | NodeEnd | NodeLoop | NodeRedirect | NodeRoutineList | NodeStart;

export type NodeDecision = {
  __typename?: 'NodeDecision';
  decisions: Array<NodeDecisionItem>;
  id: Scalars['ID'];
};

export type NodeDecisionInput = {
  decisions: Array<NodeDecisionItemInput>;
  id?: InputMaybe<Scalars['ID']>;
};

export type NodeDecisionItem = {
  __typename?: 'NodeDecisionItem';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  title: Scalars['String'];
  toId?: Maybe<Scalars['ID']>;
  when: Array<Maybe<NodeDecisionItemCase>>;
};

export type NodeDecisionItemCase = {
  __typename?: 'NodeDecisionItemCase';
  condition: Scalars['String'];
  id: Scalars['ID'];
};

export type NodeDecisionItemCaseInput = {
  condition?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
};

export type NodeDecisionItemInput = {
  description?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
  title?: InputMaybe<Scalars['String']>;
  toId?: InputMaybe<Scalars['ID']>;
  when?: InputMaybe<Array<InputMaybe<NodeDecisionItemCaseInput>>>;
};

export type NodeEnd = {
  __typename?: 'NodeEnd';
  id: Scalars['ID'];
  wasSuccessful: Scalars['Boolean'];
};

export type NodeEndInput = {
  id?: InputMaybe<Scalars['ID']>;
  wasSuccessful?: InputMaybe<Scalars['Boolean']>;
};

export type NodeInput = {
  combineData?: InputMaybe<NodeCombineInput>;
  decisionData?: InputMaybe<NodeDecisionInput>;
  description?: InputMaybe<Scalars['String']>;
  endData?: InputMaybe<NodeEndInput>;
  id?: InputMaybe<Scalars['ID']>;
  loopData?: InputMaybe<NodeLoopInput>;
  redirectData?: InputMaybe<NodeRedirectInput>;
  routineId?: InputMaybe<Scalars['ID']>;
  routineListData?: InputMaybe<NodeRoutineListInput>;
  startData?: InputMaybe<NodeStartInput>;
  title?: InputMaybe<Scalars['String']>;
  type?: InputMaybe<NodeType>;
};

export type NodeLoop = {
  __typename?: 'NodeLoop';
  id: Scalars['ID'];
};

export type NodeLoopInput = {
  id?: InputMaybe<Scalars['ID']>;
};

export type NodeRedirect = {
  __typename?: 'NodeRedirect';
  id: Scalars['ID'];
};

export type NodeRedirectInput = {
  id?: InputMaybe<Scalars['ID']>;
};

export type NodeRoutineList = {
  __typename?: 'NodeRoutineList';
  id: Scalars['ID'];
  isOptional: Scalars['Boolean'];
  isOrdered: Scalars['Boolean'];
  routines: Array<NodeRoutineListItem>;
};

export type NodeRoutineListInput = {
  id?: InputMaybe<Scalars['ID']>;
  isOptional?: InputMaybe<Scalars['Boolean']>;
  isOrdered?: InputMaybe<Scalars['Boolean']>;
  routines: Array<NodeRoutineListItemInput>;
};

export type NodeRoutineListItem = {
  __typename?: 'NodeRoutineListItem';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isOptional: Scalars['Boolean'];
  routine?: Maybe<Routine>;
  title: Scalars['String'];
};

export type NodeRoutineListItemInput = {
  description?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
  isOptional?: InputMaybe<Scalars['Boolean']>;
  listId?: InputMaybe<Scalars['ID']>;
  routineId?: InputMaybe<Scalars['ID']>;
  title?: InputMaybe<Scalars['String']>;
};

export type NodeStart = {
  __typename?: 'NodeStart';
  id: Scalars['ID'];
};

export type NodeStartInput = {
  id?: InputMaybe<Scalars['ID']>;
};

export enum NodeType {
  Combine = 'Combine',
  Decision = 'Decision',
  End = 'End',
  Loop = 'Loop',
  Redirect = 'Redirect',
  RoutineList = 'RoutineList',
  Start = 'Start'
}

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
  members: Array<Member>;
  name: Scalars['String'];
  projects: Array<Project>;
  reports: Array<Report>;
  resources: Array<Resource>;
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

export type OrganizationEdge = {
  __typename?: 'OrganizationEdge';
  cursor: Scalars['String'];
  node: Organization;
};

export type OrganizationInput = {
  bio?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
  name: Scalars['String'];
  resources?: InputMaybe<Array<ResourceInput>>;
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
  id: Scalars['ID'];
  projects: Array<Project>;
  reports: Array<Report>;
  resources: Array<Resource>;
  roles: Array<Role>;
  routines: Array<Routine>;
  routinesCreated: Array<Routine>;
  sentReports: Array<Report>;
  starredBy: Array<User>;
  starredComments: Array<Comment>;
  starredOrganizations: Array<Organization>;
  starredProjects: Array<Project>;
  starredRoutines: Array<Routine>;
  starredStandards: Array<Standard>;
  starredTags: Array<Tag>;
  starredUsers: Array<User>;
  status: AccountStatus;
  theme: Scalars['String'];
  updated_at: Scalars['Date'];
  username?: Maybe<Scalars['String']>;
  wallets: Array<Wallet>;
};

export type ProfileUpdateInput = {
  currentPassword: Scalars['String'];
  data: UserInput;
  newPassword?: InputMaybe<Scalars['String']>;
};

export type Project = {
  __typename?: 'Project';
  comments: Array<Comment>;
  created_at: Scalars['Date'];
  creator?: Maybe<Contributor>;
  description?: Maybe<Scalars['String']>;
  forks: Array<Project>;
  id: Scalars['ID'];
  isUpvoted?: Maybe<Scalars['Boolean']>;
  name: Scalars['String'];
  owner?: Maybe<Contributor>;
  parent?: Maybe<Project>;
  reports: Array<Report>;
  resources?: Maybe<Array<Resource>>;
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

export type ProjectEdge = {
  __typename?: 'ProjectEdge';
  cursor: Scalars['String'];
  node: Project;
};

export type ProjectInput = {
  description?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
  name: Scalars['String'];
  organizationId?: InputMaybe<Scalars['ID']>;
  resources?: InputMaybe<Array<ResourceInput>>;
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

export type ReportInput = {
  createdFor: ReportFor;
  details?: InputMaybe<Scalars['String']>;
  forId: Scalars['ID'];
  id?: InputMaybe<Scalars['ID']>;
  reason: Scalars['String'];
};

export type Resource = {
  __typename?: 'Resource';
  created_at: Scalars['Date'];
  description?: Maybe<Scalars['String']>;
  displayUrl?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  link: Scalars['String'];
  organization_resources: Array<Organization>;
  project_resources: Array<Project>;
  routine_resources_contextual: Array<Routine>;
  routine_resources_external: Array<Routine>;
  title: Scalars['String'];
  updated_at: Scalars['Date'];
  usedFor: ResourceUsedFor;
  user_resources: Array<User>;
};

export type ResourceCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
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

export type ResourceInput = {
  createdFor: ResourceFor;
  description?: InputMaybe<Scalars['String']>;
  displayUrl?: InputMaybe<Scalars['String']>;
  forId: Scalars['ID'];
  id?: InputMaybe<Scalars['ID']>;
  link: Scalars['String'];
  title: Scalars['String'];
  usedFor?: InputMaybe<ResourceUsedFor>;
};

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

export enum ResourceUsedFor {
  Community = 'Community',
  Context = 'Context',
  Donation = 'Donation',
  Learning = 'Learning',
  OfficialWebsite = 'OfficialWebsite',
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
  inputs: Array<RoutineInputItem>;
  instructions?: Maybe<Scalars['String']>;
  isAutomatable?: Maybe<Scalars['Boolean']>;
  isUpvoted?: Maybe<Scalars['Boolean']>;
  nodeLists: Array<NodeRoutineList>;
  nodes: Array<Node>;
  outputs: Array<RoutineOutputItem>;
  owner?: Maybe<Contributor>;
  parent?: Maybe<Routine>;
  project?: Maybe<Project>;
  reports: Array<Report>;
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

export type RoutineEdge = {
  __typename?: 'RoutineEdge';
  cursor: Scalars['String'];
  node: Routine;
};

export type RoutineInput = {
  description?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
  inputs?: InputMaybe<Array<RoutineInputItemInput>>;
  instructions?: InputMaybe<Scalars['String']>;
  isAutomatable?: InputMaybe<Scalars['Boolean']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  outputs?: InputMaybe<Array<RoutineOutputItemInput>>;
  title?: InputMaybe<Scalars['String']>;
  version?: InputMaybe<Scalars['String']>;
};

export type RoutineInputItem = {
  __typename?: 'RoutineInputItem';
  id: Scalars['ID'];
  routine: Routine;
  standard: Standard;
};

export type RoutineInputItemInput = {
  id?: InputMaybe<Scalars['ID']>;
  routineId: Scalars['ID'];
  standardId?: InputMaybe<Scalars['ID']>;
};

export type RoutineOutputItem = {
  __typename?: 'RoutineOutputItem';
  id: Scalars['ID'];
  routine: Routine;
  standard: Standard;
};

export type RoutineOutputItemInput = {
  id?: InputMaybe<Scalars['ID']>;
  routineId: Scalars['ID'];
  standardId?: InputMaybe<Scalars['ID']>;
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
  isUpvoted?: Maybe<Scalars['Boolean']>;
  name: Scalars['String'];
  reports: Array<Report>;
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

export type StandardEdge = {
  __typename?: 'StandardEdge';
  cursor: Scalars['String'];
  node: Standard;
};

export type StandardInput = {
  default?: InputMaybe<Scalars['String']>;
  description?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
  isFile?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  organizationId?: InputMaybe<Scalars['ID']>;
  schema?: InputMaybe<Scalars['String']>;
  tags?: InputMaybe<Array<TagInput>>;
  type?: InputMaybe<StandardType>;
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
  isUpvoted?: Maybe<Scalars['Boolean']>;
  score: Scalars['Int'];
  starredBy: Array<User>;
  stars: Scalars['Int'];
  tag: Scalars['String'];
  updated_at: Scalars['Date'];
};

export type TagCountInput = {
  createdTimeFrame?: InputMaybe<TimeFrame>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type TagEdge = {
  __typename?: 'TagEdge';
  cursor: Scalars['String'];
  node: Tag;
};

export type TagInput = {
  id?: InputMaybe<Scalars['ID']>;
};

export type TagSearchInput = {
  after?: InputMaybe<Scalars['String']>;
  createdTimeFrame?: InputMaybe<TimeFrame>;
  ids?: InputMaybe<Array<Scalars['ID']>>;
  searchString?: InputMaybe<Scalars['String']>;
  sortBy?: InputMaybe<TagSortBy>;
  take?: InputMaybe<Scalars['Int']>;
  updatedTimeFrame?: InputMaybe<TimeFrame>;
  userId?: InputMaybe<Scalars['ID']>;
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
  id: Scalars['ID'];
  password: Scalars['String'];
};

export type UserEdge = {
  __typename?: 'UserEdge';
  cursor: Scalars['String'];
  node: User;
};

export type UserInput = {
  bio?: InputMaybe<Scalars['String']>;
  emails?: InputMaybe<Array<EmailInput>>;
  id?: InputMaybe<Scalars['ID']>;
  status?: InputMaybe<AccountStatus>;
  theme?: InputMaybe<Scalars['String']>;
  username?: InputMaybe<Scalars['String']>;
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

export type VoteRemoveInput = {
  forId: Scalars['ID'];
  voteFor: VoteFor;
};

export type Wallet = {
  __typename?: 'Wallet';
  id: Scalars['ID'];
  organization?: Maybe<Organization>;
  publicAddress: Scalars['String'];
  user?: Maybe<User>;
  verified: Scalars['Boolean'];
};

export type WalletCompleteInput = {
  key: Scalars['String'];
  publicAddress: Scalars['String'];
  signature: Scalars['String'];
};

export type WalletInitInput = {
  nonceDescription?: InputMaybe<Scalars['String']>;
  publicAddress: Scalars['String'];
};

export type WriteAssetsInput = {
  files: Array<Scalars['Upload']>;
};
