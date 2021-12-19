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
  Deleted = 'DELETED',
  HardLocked = 'HARD_LOCKED',
  SoftLocked = 'SOFT_LOCKED',
  Unlocked = 'UNLOCKED'
}

export type Comment = {
  __typename?: 'Comment';
  createdAt: Scalars['Date'];
  id: Scalars['ID'];
  organization?: Maybe<Organization>;
  organizationId?: Maybe<Scalars['ID']>;
  stars?: Maybe<Scalars['Int']>;
  text?: Maybe<Scalars['String']>;
  updatedAt: Scalars['Date'];
  user?: Maybe<User>;
  userId?: Maybe<Scalars['ID']>;
  vote?: Maybe<Scalars['Int']>;
};

export type CommentInput = {
  id?: InputMaybe<Scalars['ID']>;
  objectId?: InputMaybe<Scalars['ID']>;
  objectType?: InputMaybe<Scalars['String']>;
  text?: InputMaybe<Scalars['String']>;
};

export type CompleteValidateWalletInput = {
  publicAddress: Scalars['String'];
  signedMessage: Scalars['String'];
};

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

export type DeleteUserInput = {
  id: Scalars['ID'];
  password: Scalars['String'];
};

export type Email = {
  __typename?: 'Email';
  emailAddress: Scalars['String'];
  id: Scalars['ID'];
  receivesAccountUpdates: Scalars['Boolean'];
  receivesBusinessUpdates: Scalars['Boolean'];
  user?: Maybe<User>;
  userId: Scalars['ID'];
  verified: Scalars['Boolean'];
};

export type EmailInput = {
  emailAddress: Scalars['String'];
  id?: InputMaybe<Scalars['ID']>;
  receivesAccountUpdates?: InputMaybe<Scalars['Boolean']>;
  receivesBusinessUpdates?: InputMaybe<Scalars['Boolean']>;
  userId?: InputMaybe<Scalars['ID']>;
};

export type FeedbackInput = {
  text: Scalars['String'];
  userId?: InputMaybe<Scalars['ID']>;
};

export type FindByIdInput = {
  id: Scalars['ID'];
};

export type InitValidateWalletInput = {
  nonceDescription?: InputMaybe<Scalars['String']>;
  publicAddress: Scalars['String'];
};

export type LogInInput = {
  email?: InputMaybe<Scalars['String']>;
  password?: InputMaybe<Scalars['String']>;
  verificationCode?: InputMaybe<Scalars['String']>;
};

export type Mutation = {
  __typename?: 'Mutation';
  addComment: Email;
  addEmail: Email;
  addFeedback: Scalars['Boolean'];
  addNode: Node;
  addOrganization: Organization;
  addProject: Project;
  addResource: Resource;
  addRoutine: Routine;
  addStandard: Standard;
  addTag: Tag;
  completeValidateWallet: Scalars['Boolean'];
  deleteComment: Scalars['Boolean'];
  deleteEmails: Count;
  deleteNode: Scalars['Boolean'];
  deleteOrganization: Scalars['Boolean'];
  deleteProject: Scalars['Boolean'];
  deleteResources: Count;
  deleteRoutine: Scalars['Boolean'];
  deleteStandards: Count;
  deleteTags: Count;
  deleteUser?: Maybe<Scalars['Boolean']>;
  exportData: Scalars['String'];
  initValidateWallet: Scalars['String'];
  logIn: User;
  logOut?: Maybe<Scalars['Boolean']>;
  removeWallet: Scalars['Boolean'];
  reportComment: Scalars['Boolean'];
  reportOrganization: Scalars['Boolean'];
  reportProject: Scalars['Boolean'];
  reportResource: Scalars['Boolean'];
  reportRoutine: Scalars['Boolean'];
  reportStandard: Scalars['Boolean'];
  reportTag: Scalars['Boolean'];
  reportUser: Scalars['Boolean'];
  requestPasswordChange?: Maybe<Scalars['Boolean']>;
  resetPassword: User;
  signUp: User;
  updateComment: Email;
  updateEmail: Email;
  updateNode: Node;
  updateOrganization: Organization;
  updateProject: Project;
  updateResource: Resource;
  updateRoutine: Routine;
  updateStandard: Standard;
  updateTag: Tag;
  updateUser: User;
  voteComment: Scalars['Boolean'];
  voteTag: Scalars['Boolean'];
  writeAssets?: Maybe<Scalars['Boolean']>;
};


export type MutationAddCommentArgs = {
  input: CommentInput;
};


export type MutationAddEmailArgs = {
  input: EmailInput;
};


export type MutationAddFeedbackArgs = {
  input: FeedbackInput;
};


export type MutationAddNodeArgs = {
  input: NodeInput;
};


export type MutationAddOrganizationArgs = {
  input: OrganizationInput;
};


export type MutationAddProjectArgs = {
  input: ProjectInput;
};


export type MutationAddResourceArgs = {
  input: ResourceInput;
};


export type MutationAddRoutineArgs = {
  input: RoutineInput;
};


export type MutationAddStandardArgs = {
  input: StandardInput;
};


export type MutationAddTagArgs = {
  input: TagInput;
};


export type MutationCompleteValidateWalletArgs = {
  input: CompleteValidateWalletInput;
};


export type MutationDeleteCommentArgs = {
  input: DeleteOneInput;
};


export type MutationDeleteEmailsArgs = {
  input: DeleteManyInput;
};


export type MutationDeleteNodeArgs = {
  input: DeleteOneInput;
};


export type MutationDeleteOrganizationArgs = {
  input?: InputMaybe<DeleteOneInput>;
};


export type MutationDeleteProjectArgs = {
  input: DeleteOneInput;
};


export type MutationDeleteResourcesArgs = {
  input: DeleteManyInput;
};


export type MutationDeleteRoutineArgs = {
  input: DeleteOneInput;
};


export type MutationDeleteStandardsArgs = {
  input: DeleteManyInput;
};


export type MutationDeleteTagsArgs = {
  input: DeleteManyInput;
};


export type MutationDeleteUserArgs = {
  input: DeleteUserInput;
};


export type MutationInitValidateWalletArgs = {
  input: InitValidateWalletInput;
};


export type MutationLogInArgs = {
  input?: InputMaybe<LogInInput>;
};


export type MutationRemoveWalletArgs = {
  input: DeleteOneInput;
};


export type MutationReportCommentArgs = {
  input: ReportInput;
};


export type MutationReportOrganizationArgs = {
  input: ReportInput;
};


export type MutationReportProjectArgs = {
  input: ReportInput;
};


export type MutationReportResourceArgs = {
  input: ReportInput;
};


export type MutationReportRoutineArgs = {
  input: ReportInput;
};


export type MutationReportStandardArgs = {
  input: ReportInput;
};


export type MutationReportTagArgs = {
  input: ReportInput;
};


export type MutationReportUserArgs = {
  input: ReportInput;
};


export type MutationRequestPasswordChangeArgs = {
  input: RequestPasswordChangeInput;
};


export type MutationResetPasswordArgs = {
  input: ResetPasswordInput;
};


export type MutationSignUpArgs = {
  input: SignUpInput;
};


export type MutationUpdateCommentArgs = {
  input: CommentInput;
};


export type MutationUpdateEmailArgs = {
  input: EmailInput;
};


export type MutationUpdateNodeArgs = {
  input: NodeInput;
};


export type MutationUpdateOrganizationArgs = {
  input: OrganizationInput;
};


export type MutationUpdateProjectArgs = {
  input: ProjectInput;
};


export type MutationUpdateResourceArgs = {
  input: ResourceInput;
};


export type MutationUpdateRoutineArgs = {
  input: RoutineInput;
};


export type MutationUpdateStandardArgs = {
  input: StandardInput;
};


export type MutationUpdateTagArgs = {
  input: TagInput;
};


export type MutationUpdateUserArgs = {
  input: UpdateUserInput;
};


export type MutationVoteCommentArgs = {
  input: VoteInput;
};


export type MutationVoteTagArgs = {
  input: TagVoteInput;
};


export type MutationWriteAssetsArgs = {
  files: Array<Scalars['Upload']>;
};

export type Node = {
  __typename?: 'Node';
  data?: Maybe<NodeData>;
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  routine: Routine;
  routineId: Scalars['ID'];
  title: Scalars['String'];
  type: NodeType;
};

export type NodeCombine = {
  __typename?: 'NodeCombine';
  from: Array<NodeCombineFrom>;
  id: Scalars['ID'];
  to?: Maybe<Node>;
};

export type NodeCombineFrom = {
  __typename?: 'NodeCombineFrom';
  combine?: Maybe<NodeCombine>;
  combineId: Scalars['ID'];
  from?: Maybe<Node>;
  fromId: Scalars['ID'];
  id: Scalars['ID'];
};

export type NodeCombineFromInput = {
  combineId?: InputMaybe<Scalars['ID']>;
  fromId?: InputMaybe<Scalars['ID']>;
  id?: InputMaybe<Scalars['ID']>;
};

export type NodeCombineInput = {
  from: Array<NodeCombineFromInput>;
  id?: InputMaybe<Scalars['ID']>;
  to?: InputMaybe<NodeInput>;
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
  id: Scalars['ID'];
  title: Scalars['String'];
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
  id?: InputMaybe<Scalars['ID']>;
  title?: InputMaybe<Scalars['String']>;
  when?: InputMaybe<Array<InputMaybe<NodeDecisionItemCaseInput>>>;
};

export type NodeEnd = {
  __typename?: 'NodeEnd';
  id: Scalars['ID'];
};

export type NodeEndInput = {
  id?: InputMaybe<Scalars['ID']>;
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
  isOrdered: Scalars['Boolean'];
  routines: Array<NodeRoutineListItem>;
};

export type NodeRoutineListInput = {
  id?: InputMaybe<Scalars['ID']>;
  isOrdered?: InputMaybe<Scalars['Boolean']>;
  routines: Array<NodeRoutineListItemInput>;
};

export type NodeRoutineListItem = {
  __typename?: 'NodeRoutineListItem';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isOptional: Scalars['Boolean'];
  list?: Maybe<NodeRoutineList>;
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
  Combine = 'COMBINE',
  Decision = 'DECISION',
  End = 'END',
  Loop = 'LOOP',
  Redirect = 'REDIRECT',
  RoutineList = 'ROUTINE_LIST',
  Start = 'START'
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
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  name: Scalars['String'];
  projects?: Maybe<Array<Project>>;
  resources?: Maybe<Array<Resource>>;
  routines?: Maybe<Array<Routine>>;
  starredBy?: Maybe<Array<User>>;
  wallets?: Maybe<Array<Wallet>>;
};

export type OrganizationInput = {
  description?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
  name: Scalars['String'];
  resources?: InputMaybe<Array<ResourceInput>>;
};

export type OrganizationsQueryInput = {
  first?: InputMaybe<Scalars['Int']>;
  skip?: InputMaybe<Scalars['Int']>;
};

export type Project = {
  __typename?: 'Project';
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  name: Scalars['String'];
  organizations?: Maybe<Array<Organization>>;
  resources?: Maybe<Array<Resource>>;
  starredBy?: Maybe<Array<User>>;
  users?: Maybe<Array<User>>;
  wallets?: Maybe<Array<Wallet>>;
};

export type ProjectInput = {
  description?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
  name: Scalars['String'];
  organizations?: InputMaybe<Array<OrganizationInput>>;
  resources?: InputMaybe<Array<ResourceInput>>;
  users?: InputMaybe<Array<UserInput>>;
};

export type ProjectsQueryInput = {
  first?: InputMaybe<Scalars['Int']>;
  skip?: InputMaybe<Scalars['Int']>;
};

export type Query = {
  __typename?: 'Query';
  organization?: Maybe<Organization>;
  organizations: Array<Organization>;
  organizationsCount: Count;
  profile: User;
  project?: Maybe<Project>;
  projects: Array<Project>;
  projectsCount: Scalars['Int'];
  readAssets: Array<Maybe<Scalars['String']>>;
  readOpenGraph: OpenGraphResponse;
  resource?: Maybe<Resource>;
  resources: Array<Resource>;
  resourcesCount: Scalars['Int'];
  routine?: Maybe<Routine>;
  routines: Array<Routine>;
  routinesCount: Scalars['Int'];
  standard?: Maybe<Standard>;
  standards: Array<Standard>;
  standardsCount: Scalars['Int'];
  tag?: Maybe<Tag>;
  tags: Array<Tag>;
  tagsCount: Scalars['Int'];
};


export type QueryOrganizationArgs = {
  input: FindByIdInput;
};


export type QueryOrganizationsArgs = {
  input: OrganizationsQueryInput;
};


export type QueryProjectArgs = {
  input: FindByIdInput;
};


export type QueryProjectsArgs = {
  input: ProjectsQueryInput;
};


export type QueryReadAssetsArgs = {
  files: Array<Scalars['String']>;
};


export type QueryReadOpenGraphArgs = {
  url: Scalars['String'];
};


export type QueryResourceArgs = {
  input: FindByIdInput;
};


export type QueryResourcesArgs = {
  input: ResourcesQueryInput;
};


export type QueryRoutineArgs = {
  input: FindByIdInput;
};


export type QueryRoutinesArgs = {
  input: RoutinesQueryInput;
};


export type QueryStandardArgs = {
  input: FindByIdInput;
};


export type QueryStandardsArgs = {
  input: StandardsQueryInput;
};


export type QueryTagArgs = {
  input: FindByIdInput;
};


export type QueryTagsArgs = {
  input: TagsQueryInput;
};

export type ReportInput = {
  id: Scalars['ID'];
  reason?: InputMaybe<Scalars['String']>;
};

export type RequestPasswordChangeInput = {
  email: Scalars['String'];
};

export type ResetPasswordInput = {
  code: Scalars['String'];
  id: Scalars['ID'];
  newPassword: Scalars['String'];
};

export type Resource = {
  __typename?: 'Resource';
  description?: Maybe<Scalars['String']>;
  displayUrl?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  link: Scalars['String'];
  name: Scalars['String'];
};

export enum ResourceFor {
  Organization = 'ORGANIZATION',
  Project = 'PROJECT',
  RoutineContextual = 'ROUTINE_CONTEXTUAL',
  RoutineDonation = 'ROUTINE_DONATION',
  RoutineExternal = 'ROUTINE_EXTERNAL',
  User = 'USER'
}

export type ResourceInput = {
  createdFor: ResourceFor;
  description?: InputMaybe<Scalars['String']>;
  displayUrl?: InputMaybe<Scalars['String']>;
  forId: Scalars['ID'];
  id?: InputMaybe<Scalars['ID']>;
  link: Scalars['String'];
  name: Scalars['String'];
};

export type ResourcesQueryInput = {
  first?: InputMaybe<Scalars['Int']>;
  skip?: InputMaybe<Scalars['Int']>;
};

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
  id: Scalars['ID'];
};

export type RoutineInput = {
  description?: InputMaybe<Scalars['String']>;
  externalLink?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
  inputs?: InputMaybe<Array<RoutineInputItemInput>>;
  instructions?: InputMaybe<Scalars['String']>;
  isAutomatable?: InputMaybe<Scalars['Boolean']>;
  outputs?: InputMaybe<Array<RoutineOutputItemInput>>;
  title?: InputMaybe<Scalars['String']>;
  version?: InputMaybe<Scalars['String']>;
};

export type RoutineInputItem = {
  __typename?: 'RoutineInputItem';
  id: Scalars['ID'];
};

export type RoutineInputItemInput = {
  id?: InputMaybe<Scalars['ID']>;
};

export type RoutineOutputItem = {
  __typename?: 'RoutineOutputItem';
  id: Scalars['ID'];
};

export type RoutineOutputItemInput = {
  id?: InputMaybe<Scalars['ID']>;
};

export type RoutinesQueryInput = {
  first?: InputMaybe<Scalars['Int']>;
  skip?: InputMaybe<Scalars['Int']>;
};

export type SignUpInput = {
  email: Scalars['String'];
  marketingEmails: Scalars['Boolean'];
  password: Scalars['String'];
  pronouns?: InputMaybe<Scalars['String']>;
  theme: Scalars['String'];
  username: Scalars['String'];
};

export type Standard = {
  __typename?: 'Standard';
  default?: Maybe<Scalars['String']>;
  description?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  isFile: Scalars['Boolean'];
  name: Scalars['String'];
  schema: Scalars['String'];
  tags: Array<Tag>;
  type: StandardType;
};

export type StandardInput = {
  default?: InputMaybe<Scalars['String']>;
  description?: InputMaybe<Scalars['String']>;
  id?: InputMaybe<Scalars['ID']>;
  isFile?: InputMaybe<Scalars['Boolean']>;
  name?: InputMaybe<Scalars['String']>;
  schema?: InputMaybe<Scalars['String']>;
  tags?: InputMaybe<Array<TagInput>>;
  type?: InputMaybe<StandardType>;
};

export enum StandardType {
  Array = 'ARRAY',
  Boolean = 'BOOLEAN',
  File = 'FILE',
  Number = 'NUMBER',
  Object = 'OBJECT',
  String = 'STRING',
  Url = 'URL'
}

export type StandardsQueryInput = {
  first?: InputMaybe<Scalars['Int']>;
  skip?: InputMaybe<Scalars['Int']>;
};

export type Tag = {
  __typename?: 'Tag';
  id: Scalars['ID'];
};

export type TagInput = {
  id?: InputMaybe<Scalars['ID']>;
};

export type TagVoteInput = {
  id: Scalars['ID'];
  isUpvote: Scalars['Boolean'];
  objectId: Scalars['ID'];
  objectType: Scalars['String'];
};

export type TagsQueryInput = {
  first?: InputMaybe<Scalars['Int']>;
  skip?: InputMaybe<Scalars['Int']>;
};

export type UpdateUserInput = {
  currentPassword: Scalars['String'];
  data: UserInput;
  newPassword?: InputMaybe<Scalars['String']>;
};

export type User = {
  __typename?: 'User';
  emailVerified: Scalars['Boolean'];
  emails: Array<Email>;
  id: Scalars['ID'];
  pronouns: Scalars['String'];
  roles: Array<UserRole>;
  status: AccountStatus;
  theme: Scalars['String'];
  username?: Maybe<Scalars['String']>;
};

export type UserInput = {
  emails?: InputMaybe<Array<EmailInput>>;
  id?: InputMaybe<Scalars['ID']>;
  pronouns?: InputMaybe<Scalars['String']>;
  status?: InputMaybe<AccountStatus>;
  theme?: InputMaybe<Scalars['String']>;
  username?: InputMaybe<Scalars['String']>;
};

export type UserRole = {
  __typename?: 'UserRole';
  role: Role;
  user: User;
};

export type VoteInput = {
  id: Scalars['ID'];
  isUpvote: Scalars['Boolean'];
};

export type Wallet = {
  __typename?: 'Wallet';
  id: Scalars['ID'];
  organization?: Maybe<Organization>;
  publicAddress: Scalars['String'];
  user?: Maybe<User>;
  verified: Scalars['Boolean'];
};
