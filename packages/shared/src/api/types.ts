export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Omit<T, K extends keyof T> = Pick<T, Exclude<keyof T, K>>;
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
    focusMode: ActiveFocusModeFocusMode;
    stopCondition: FocusModeStopCondition;
    stopTime?: Maybe<Scalars['Date']>;
};

export type ActiveFocusModeFocusMode = {
    __typename: 'ActiveFocusModeFocusMode';
    id: Scalars['String'];
    reminderListId?: Maybe<Scalars['String']>;
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
    schemaLanguage?: Maybe<Scalars['String']>;
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
    schemaLanguage?: InputMaybe<Scalars['String']>;
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
    schemaLanguage?: InputMaybe<Scalars['String']>;
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
    taskType: TaskType;
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
    parentConnect?: InputMaybe<Scalars['ID']>;
    translationsCreate?: InputMaybe<Array<ChatMessageTranslationCreateInput>>;
    userConnect: Scalars['ID'];
    versionIndex: Scalars['Int'];
};

export type ChatMessageCreateWithTaskInfoInput = {
    message: ChatMessageCreateInput;
    task: LlmTask;
    taskContexts: Array<TaskContextInfoInput>;
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

export type ChatMessageUpdateWithTaskInfoInput = {
    message: ChatMessageUpdateInput;
    task: LlmTask;
    taskContexts: Array<TaskContextInfoInput>;
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
    taskType: TaskType;
};

export type CheckTaskStatusesResult = {
    __typename: 'CheckTaskStatusesResult';
    statuses: Array<TaskStatusInfo>;
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
    codeLanguageLatestVersion?: InputMaybe<Scalars['String']>;
    codeTypeLatestVersion?: InputMaybe<CodeType>;
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
    codeLanguage?: InputMaybe<Scalars['String']>;
    codeType?: InputMaybe<CodeType>;
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
    ActiveFocusMode = 'ActiveFocusMode',
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
    Draft = 'Draft',
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
    QuestionAdd = 'QuestionAdd',
    QuestionDelete = 'QuestionDelete',
    QuestionFind = 'QuestionFind',
    QuestionUpdate = 'QuestionUpdate',
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
    logOutAll: Session;
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
    setActiveFocusMode?: Maybe<ActiveFocusMode>;
    standardCreate: Standard;
    standardUpdate: Standard;
    standardVersionCreate: StandardVersion;
    standardVersionUpdate: StandardVersion;
    startLlmTask: Success;
    startRunTask: Success;
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
    input: ChatMessageCreateWithTaskInfoInput;
};


export type MutationChatMessageUpdateArgs = {
    input: ChatMessageUpdateWithTaskInfoInput;
};


export type MutationChatParticipantUpdateArgs = {
    input: ChatParticipantUpdateInput;
};


export type MutationChatUpdateArgs = {
    input: ChatUpdateInput;
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


export type MutationStartLlmTaskArgs = {
    input: StartLlmTaskInput;
};


export type MutationStartRunTaskArgs = {
    input: StartRunTaskInput;
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
    Draft = 'Draft',
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
    checkTaskStatuses: CheckTaskStatusesResult;
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


export type QueryCheckTaskStatusesArgs = {
    input: CheckTaskStatusesInput;
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
    visibility?: InputMaybe<VisibilityType>;
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
    apiId?: InputMaybe<Scalars['ID']>;
    chatMessageId?: InputMaybe<Scalars['ID']>;
    codeId?: InputMaybe<Scalars['ID']>;
    commentId?: InputMaybe<Scalars['ID']>;
    excludeLinkedToTag?: InputMaybe<Scalars['Boolean']>;
    ids?: InputMaybe<Array<Scalars['ID']>>;
    issueId?: InputMaybe<Scalars['ID']>;
    noteId?: InputMaybe<Scalars['ID']>;
    postId?: InputMaybe<Scalars['ID']>;
    projectId?: InputMaybe<Scalars['ID']>;
    questionAnswerId?: InputMaybe<Scalars['ID']>;
    questionId?: InputMaybe<Scalars['ID']>;
    quizId?: InputMaybe<Scalars['ID']>;
    routineId?: InputMaybe<Scalars['ID']>;
    searchString?: InputMaybe<Scalars['String']>;
    sortBy?: InputMaybe<ReactionSortBy>;
    standardId?: InputMaybe<Scalars['ID']>;
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
    task: LlmTask;
    taskContexts: Array<TaskContextInfoInput>;
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
    createdFor: ReportFor;
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
    createdForConnect: Scalars['ID'];
    createdForType: ReportFor;
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
    visibility?: InputMaybe<VisibilityType>;
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
    isOwn: Scalars['Boolean'];
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
    latestVersionRoutineTypes?: InputMaybe<Array<RoutineType>>;
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
    routineTypes?: InputMaybe<Array<RoutineType>>;
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

export enum RunTask {
    RunProject = 'RunProject',
    RunRoutine = 'RunRoutine'
}

export enum SandboxTask {
    CallApi = 'CallApi',
    RunDataTransform = 'RunDataTransform',
    RunSmartContract = 'RunSmartContract'
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
    codesCount: Scalars['Int'];
    credits: Scalars['String'];
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
    session: SessionUserSession;
    standardsCount: Scalars['Int'];
    theme?: Maybe<Scalars['String']>;
    updated_at: Scalars['Date'];
};

export type SessionUserSession = {
    __typename: 'SessionUserSession';
    id: Scalars['String'];
    lastRefreshAt: Scalars['Date'];
};

export type SetActiveFocusModeInput = {
    id?: InputMaybe<Scalars['ID']>;
    stopCondition?: InputMaybe<FocusModeStopCondition>;
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
    codeLanguageLatestVersion?: InputMaybe<Scalars['String']>;
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
    variantLatestVersion?: InputMaybe<StandardType>;
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

export enum StandardType {
    DataStructure = 'DataStructure',
    Prompt = 'Prompt'
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
    codeLanguage: Scalars['String'];
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
    translations: Array<StandardVersionTranslation>;
    translationsCount: Scalars['Int'];
    updated_at: Scalars['Date'];
    variant: StandardType;
    versionIndex: Scalars['Int'];
    versionLabel: Scalars['String'];
    versionNotes?: Maybe<Scalars['String']>;
    you: VersionYou;
    yup?: Maybe<Scalars['String']>;
};

export type StandardVersionCreateInput = {
    codeLanguage: Scalars['String'];
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
    translationsCreate?: InputMaybe<Array<StandardVersionTranslationCreateInput>>;
    variant: StandardType;
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
    codeLanguage?: InputMaybe<Scalars['String']>;
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
    tagsRoot?: InputMaybe<Array<Scalars['String']>>;
    take?: InputMaybe<Scalars['Int']>;
    translationLanguages?: InputMaybe<Array<Scalars['String']>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userId?: InputMaybe<Scalars['ID']>;
    variant?: InputMaybe<StandardType>;
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
    codeLanguage?: InputMaybe<Scalars['String']>;
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
    translationsCreate?: InputMaybe<Array<StandardVersionTranslationCreateInput>>;
    translationsDelete?: InputMaybe<Array<Scalars['ID']>>;
    translationsUpdate?: InputMaybe<Array<StandardVersionTranslationUpdateInput>>;
    variant?: InputMaybe<StandardType>;
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

export type StartLlmTaskInput = {
    chatId: Scalars['ID'];
    parentId?: InputMaybe<Scalars['ID']>;
    respondingBotId: Scalars['ID'];
    shouldNotRunTasks: Scalars['Boolean'];
    task: LlmTask;
    taskContexts: Array<TaskContextInfoInput>;
};

export type StartRunTaskInput = {
    formValues?: InputMaybe<Scalars['JSONObject']>;
    projectVerisonId?: InputMaybe<Scalars['ID']>;
    routineVersionId?: InputMaybe<Scalars['ID']>;
    runId: Scalars['ID'];
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

export type TaskContextInfoInput = {
    data: Scalars['JSONObject'];
    id: Scalars['ID'];
    label: Scalars['String'];
    template?: InputMaybe<Scalars['String']>;
    templateVariables?: InputMaybe<TaskContextInfoTemplateVariablesInput>;
};

export type TaskContextInfoTemplateVariablesInput = {
    data?: InputMaybe<Scalars['String']>;
    task?: InputMaybe<Scalars['String']>;
};

export enum TaskStatus {
    Canceling = 'Canceling',
    Completed = 'Completed',
    Failed = 'Failed',
    Running = 'Running',
    Scheduled = 'Scheduled',
    Suggested = 'Suggested'
}

export type TaskStatusInfo = {
    __typename: 'TaskStatusInfo';
    id: Scalars['ID'];
    status?: Maybe<TaskStatus>;
};

export enum TaskType {
    Llm = 'Llm',
    Run = 'Run',
    Sandbox = 'Sandbox'
}

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
    notInvitedToTeamId?: InputMaybe<Scalars['ID']>;
    notMemberInTeamId?: InputMaybe<Scalars['ID']>;
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
    Own = 'Own',
    OwnOrPublic = 'OwnOrPublic',
    OwnPrivate = 'OwnPrivate',
    OwnPublic = 'OwnPublic',
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
