/* c8 ignore start */
import { type RunConfig } from "../run/types.js";

export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Omit<T, K extends keyof T> = Pick<T, Exclude<keyof T, K>>;
/** All built-in and custom scalars, mapped to their actual values */
type Scalars = {
    ID: string;
    String: string;
    BigInt: string; // Stringified because BigInt is not serializable
    Boolean: boolean;
    Int: number;
    Float: number;
    Date: any;
    JSONObject: any;
    Upload: any;
};

type Typed<Typename extends string> = {
    __typename: Typename;
}

export type DbObject<Typename extends string> = Typed<Typename> & {
    id: Scalars["ID"];
};

type Edge<NodeType, Typename extends string> = Typed<Typename> & {
    cursor: string;
    node: NodeType;
};

type SearchResult<
    EdgeType extends { __typename: string },
    Typename extends string,
    PInfo = PageInfo,
> = Typed<Typename> & {
    edges: Array<EdgeType>;
    pageInfo: PInfo;
};

interface MultiObjectSearchInput<TSortBy = unknown> {
    ids?: InputMaybe<Array<Scalars["ID"]>>;
    sortBy?: InputMaybe<TSortBy>;
    take?: InputMaybe<Scalars["Int"]>;
}

interface BaseSearchInput<TSortBy = unknown> extends MultiObjectSearchInput<TSortBy> {
    after?: InputMaybe<Scalars["String"]>;
}

interface BaseTranslation<Typename extends string> extends DbObject<Typename> {
    language: Scalars["String"];
}

interface BaseTranslationCreateInput {
    id: Scalars["ID"];
    language: Scalars["String"];
}

interface BaseTranslationUpdateInput {
    id: Scalars["ID"];
    language: Scalars["String"];
}

interface BaseTranslatableCreateInput<TCreateInput extends BaseTranslationCreateInput> {
    translationsCreate?: InputMaybe<Array<TCreateInput>>;
}

interface BaseTranslatableUpdateInput<TCreateInput extends BaseTranslationCreateInput, TUpdateInput extends BaseTranslationUpdateInput> {
    translationsCreate?: InputMaybe<Array<TCreateInput>>;
    translationsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    translationsUpdate?: InputMaybe<Array<TUpdateInput>>;
}

export enum AccountStatus {
    Deleted = "Deleted",
    HardLocked = "HardLocked",
    SoftLocked = "SoftLocked",
    Unlocked = "Unlocked"
}

export type ActiveFocusMode = {
    __typename: "ActiveFocusMode";
    focusMode: ActiveFocusModeFocusMode;
    stopCondition: FocusModeStopCondition;
    stopTime?: Maybe<Scalars["Date"]>;
};

export type ActiveFocusModeFocusMode = {
    __typename: "ActiveFocusModeFocusMode";
    id: Scalars["String"];
    reminderListId?: Maybe<Scalars["String"]>;
};

export type Api = DbObject<"Api"> & {
    bookmarkedBy: Array<User>;
    bookmarks: Scalars["Int"];
    completedAt?: Maybe<Scalars["Date"]>;
    createdBy?: Maybe<User>;
    created_at: Scalars["Date"];
    hasCompleteVersion: Scalars["Boolean"];
    isDeleted: Scalars["Boolean"];
    isPrivate: Scalars["Boolean"];
    issues: Array<Issue>;
    issuesCount: Scalars["Int"];
    labels: Array<Label>;
    owner?: Maybe<Owner>;
    parent?: Maybe<ApiVersion>;
    permissions: Scalars["String"];
    pullRequests: Array<PullRequest>;
    pullRequestsCount: Scalars["Int"];
    questions: Array<Question>;
    questionsCount: Scalars["Int"];
    score: Scalars["Int"];
    stats: Array<StatsApi>;
    tags: Array<Tag>;
    transfers: Array<Transfer>;
    transfersCount: Scalars["Int"];
    updated_at: Scalars["Date"];
    versions: Array<ApiVersion>;
    versionsCount: Scalars["Int"];
    views: Scalars["Int"];
    you: ApiYou;
};

export type ApiCreateInput = {
    id: Scalars["ID"];
    isPrivate: Scalars["Boolean"];
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    ownedByTeamConnect?: InputMaybe<Scalars["ID"]>;
    ownedByUserConnect?: InputMaybe<Scalars["ID"]>;
    parentConnect?: InputMaybe<Scalars["ID"]>;
    permissions?: InputMaybe<Scalars["String"]>;
    tagsConnect?: InputMaybe<Array<Scalars["String"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
    versionsCreate?: InputMaybe<Array<ApiVersionCreateInput>>;
};

export type ApiEdge = Edge<Api, "ApiEdge">;

export type ApiKey = DbObject<"ApiKey"> & {
    creditsUsed: Scalars["BigInt"];
    disabledAt?: Maybe<Scalars["Date"]>;
    limitHard: Scalars["BigInt"];
    limitSoft?: Maybe<Scalars["BigInt"]>;
    name: Scalars["String"];
    permissions: Scalars["String"];
    stopAtLimit: Scalars["Boolean"];
};

// Only show the key to the user once it's created, and never again
export type ApiKeyCreated = ApiKey & {
    key: Scalars["String"];
}

export type ApiKeyCreateInput = {
    disabled?: InputMaybe<Scalars["Boolean"]>;
    limitHard: Scalars["BigInt"];
    limitSoft?: InputMaybe<Scalars["BigInt"]>;
    name: Scalars["String"];
    stopAtLimit: Scalars["Boolean"];
    teamConnect?: InputMaybe<Scalars["ID"]>;
    permissions: Scalars["String"];
};

export type ApiKeyUpdateInput = {
    disabled?: InputMaybe<Scalars["Boolean"]>;
    id: Scalars["ID"];
    limitHard?: InputMaybe<Scalars["BigInt"]>;
    limitSoft?: InputMaybe<Scalars["BigInt"]>;
    name?: InputMaybe<Scalars["String"]>;
    stopAtLimit?: InputMaybe<Scalars["Boolean"]>;
    permissions?: InputMaybe<Scalars["String"]>;
};

export type ApiKeyValidateInput = {
    id: Scalars["ID"];
    secret: Scalars["String"];
};

export type ApiKeyExternal = DbObject<"ApiKeyExternal"> & {
    disabledAt?: Maybe<Scalars["Date"]>;
    name: Scalars["String"];
    service: Scalars["String"];
}

export type ApiKeyExternalCreateInput = {
    disabled?: InputMaybe<Scalars["Boolean"]>;
    key: Scalars["String"];
    name: Scalars["String"];
    service: Scalars["String"];
    teamConnect?: InputMaybe<Scalars["ID"]>;
}

export type ApiKeyExternalDeleteOneInput = {
    id: Scalars["ID"];
}

export type ApiKeyExternalUpdateInput = {
    id: Scalars["ID"];
    disabled?: InputMaybe<Scalars["Boolean"]>;
    key?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
    service?: InputMaybe<Scalars["String"]>;
}

export type ApiSearchInput = BaseSearchInput<ApiSortBy> & {
    createdById?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    hasCompleteVersion?: InputMaybe<Scalars["Boolean"]>;
    issuesId?: InputMaybe<Scalars["ID"]>;
    labelsIds?: InputMaybe<Array<Scalars["ID"]>>;
    maxBookmarks?: InputMaybe<Scalars["Int"]>;
    maxScore?: InputMaybe<Scalars["Int"]>;
    maxViews?: InputMaybe<Scalars["Int"]>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    minScore?: InputMaybe<Scalars["Int"]>;
    minViews?: InputMaybe<Scalars["Int"]>;
    ownedByTeamId?: InputMaybe<Scalars["ID"]>;
    ownedByUserId?: InputMaybe<Scalars["ID"]>;
    parentId?: InputMaybe<Scalars["ID"]>;
    pullRequestsId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    tags?: InputMaybe<Array<Scalars["String"]>>;
    translationLanguagesLatestVersion?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type ApiSearchResult = SearchResult<ApiEdge, "ApiSearchResult">;

export enum ApiSortBy {
    BookmarksAsc = "BookmarksAsc",
    BookmarksDesc = "BookmarksDesc",
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
    ScoreAsc = "ScoreAsc",
    ScoreDesc = "ScoreDesc",
    VersionsAsc = "VersionsAsc",
    VersionsDesc = "VersionsDesc",
    ViewsAsc = "ViewsAsc",
    ViewsDesc = "ViewsDesc"
}

export type ApiUpdateInput = {
    id: Scalars["ID"];
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    labelsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    ownedByTeamConnect?: InputMaybe<Scalars["ID"]>;
    ownedByUserConnect?: InputMaybe<Scalars["ID"]>;
    permissions?: InputMaybe<Scalars["String"]>;
    tagsConnect?: InputMaybe<Array<Scalars["String"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
    tagsDisconnect?: InputMaybe<Array<Scalars["String"]>>;
    versionsCreate?: InputMaybe<Array<ApiVersionCreateInput>>;
    versionsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    versionsUpdate?: InputMaybe<Array<ApiVersionUpdateInput>>;
};

export type ApiVersion = DbObject<"ApiVersion"> & {
    calledByRoutineVersionsCount: Scalars["Int"];
    callLink: Scalars["String"];
    comments: Array<Comment>;
    commentsCount: Scalars["Int"];
    created_at: Scalars["Date"];
    directoryListings: Array<ProjectVersionDirectory>;
    directoryListingsCount: Scalars["Int"];
    documentationLink?: Maybe<Scalars["String"]>;
    forks: Array<Api>;
    forksCount: Scalars["Int"];
    isComplete: Scalars["Boolean"];
    isLatest: Scalars["Boolean"];
    isPrivate: Scalars["Boolean"];
    pullRequest?: Maybe<PullRequest>;
    reports: Array<Report>;
    reportsCount: Scalars["Int"];
    resourceList?: Maybe<ResourceList>;
    root: Api;
    schemaLanguage?: Maybe<Scalars["String"]>;
    schemaText?: Maybe<Scalars["String"]>;
    translations: Array<ApiVersionTranslation>;
    updated_at: Scalars["Date"];
    versionIndex: Scalars["Int"];
    versionLabel: Scalars["String"];
    versionNotes?: Maybe<Scalars["String"]>;
    you: VersionYou;
};

export type ApiVersionCreateInput = BaseTranslatableCreateInput<ApiVersionTranslationCreateInput> & {
    callLink: Scalars["String"];
    directoryListingsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    documentationLink?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    isComplete?: InputMaybe<Scalars["Boolean"]>;
    isPrivate: Scalars["Boolean"];
    resourceListCreate?: InputMaybe<ResourceListCreateInput>;
    rootConnect?: InputMaybe<Scalars["ID"]>;
    rootCreate?: InputMaybe<ApiCreateInput>;
    schemaLanguage?: InputMaybe<Scalars["String"]>;
    schemaText?: InputMaybe<Scalars["String"]>;
    versionLabel: Scalars["String"];
    versionNotes?: InputMaybe<Scalars["String"]>;
};

export type ApiVersionEdge = Edge<ApiVersion, "ApiVersionEdge">;

export type ApiVersionSearchInput = BaseSearchInput<ApiVersionSortBy> & {
    calledByRoutineVersionId?: InputMaybe<Scalars["ID"]>;
    createdByIdRoot?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    isCompleteWithRoot?: InputMaybe<Scalars["Boolean"]>;
    isLatest?: InputMaybe<Scalars["Boolean"]>;
    maxBookmarksRoot?: InputMaybe<Scalars["Int"]>;
    maxScoreRoot?: InputMaybe<Scalars["Int"]>;
    maxViewsRoot?: InputMaybe<Scalars["Int"]>;
    minBookmarksRoot?: InputMaybe<Scalars["Int"]>;
    minScoreRoot?: InputMaybe<Scalars["Int"]>;
    minViewsRoot?: InputMaybe<Scalars["Int"]>;
    ownedByTeamIdRoot?: InputMaybe<Scalars["ID"]>;
    ownedByUserIdRoot?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    tagsRoot?: InputMaybe<Array<Scalars["String"]>>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type ApiVersionSearchResult = SearchResult<ApiVersionEdge, "ApiVersionSearchResult">;

export enum ApiVersionSortBy {
    CalledByRoutinesAsc = "CalledByRoutinesAsc",
    CalledByRoutinesDesc = "CalledByRoutinesDesc",
    CommentsAsc = "CommentsAsc",
    CommentsDesc = "CommentsDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DirectoryListingsAsc = "DirectoryListingsAsc",
    DirectoryListingsDesc = "DirectoryListingsDesc",
    ForksAsc = "ForksAsc",
    ForksDesc = "ForksDesc",
    ReportsAsc = "ReportsAsc",
    ReportsDesc = "ReportsDesc"
}

export type ApiVersionTranslation = BaseTranslation<"ApiVersionTranslation"> & {
    details?: Maybe<Scalars["String"]>;
    name: Scalars["String"];
    summary?: Maybe<Scalars["String"]>;
};

export type ApiVersionTranslationCreateInput = BaseTranslationCreateInput & {
    details?: InputMaybe<Scalars["String"]>;
    name: Scalars["String"];
    summary?: InputMaybe<Scalars["String"]>;
};

export type ApiVersionTranslationUpdateInput = BaseTranslationUpdateInput & {
    details?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
    summary?: InputMaybe<Scalars["String"]>;
};

export type ApiVersionUpdateInput = BaseTranslatableUpdateInput<ApiVersionTranslationCreateInput, ApiVersionTranslationUpdateInput> & {
    callLink?: InputMaybe<Scalars["String"]>;
    directoryListingsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    directoryListingsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    documentationLink?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    isComplete?: InputMaybe<Scalars["Boolean"]>;
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    resourceListCreate?: InputMaybe<ResourceListCreateInput>;
    resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
    rootUpdate?: InputMaybe<ApiUpdateInput>;
    schemaLanguage?: InputMaybe<Scalars["String"]>;
    schemaText?: InputMaybe<Scalars["String"]>;
    versionLabel?: InputMaybe<Scalars["String"]>;
    versionNotes?: InputMaybe<Scalars["String"]>;
};

export type ApiYou = {
    __typename: "ApiYou";
    canBookmark: Scalars["Boolean"];
    canDelete: Scalars["Boolean"];
    canReact: Scalars["Boolean"];
    canRead: Scalars["Boolean"];
    canTransfer: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
    isBookmarked: Scalars["Boolean"];
    isViewed: Scalars["Boolean"];
    reaction?: Maybe<Scalars["String"]>;
};

export type Award = DbObject<"Award"> & {
    category: AwardCategory;
    created_at: Scalars["Date"];
    description?: Maybe<Scalars["String"]>;
    progress: Scalars["Int"];
    timeCurrentTierCompleted?: Maybe<Scalars["Date"]>;
    title?: Maybe<Scalars["String"]>;
    updated_at: Scalars["Date"];
};

export enum AwardCategory {
    AccountAnniversary = "AccountAnniversary",
    AccountNew = "AccountNew",
    ApiCreate = "ApiCreate",
    CommentCreate = "CommentCreate",
    IssueCreate = "IssueCreate",
    NoteCreate = "NoteCreate",
    ObjectBookmark = "ObjectBookmark",
    ObjectReact = "ObjectReact",
    OrganizationCreate = "OrganizationCreate",
    OrganizationJoin = "OrganizationJoin",
    PostCreate = "PostCreate",
    ProjectCreate = "ProjectCreate",
    PullRequestComplete = "PullRequestComplete",
    PullRequestCreate = "PullRequestCreate",
    QuestionAnswer = "QuestionAnswer",
    QuestionCreate = "QuestionCreate",
    QuizPass = "QuizPass",
    ReportContribute = "ReportContribute",
    ReportEnd = "ReportEnd",
    Reputation = "Reputation",
    RoutineCreate = "RoutineCreate",
    RunProject = "RunProject",
    RunRoutine = "RunRoutine",
    SmartContractCreate = "SmartContractCreate",
    StandardCreate = "StandardCreate",
    Streak = "Streak",
    UserInvite = "UserInvite"
}

export type AwardEdge = Edge<Award, "AwardEdge">;

export type AwardSearchInput = BaseSearchInput<Award> & {
    updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type AwardSearchResult = SearchResult<AwardEdge, "AwardSearchResult">;

export enum AwardSortBy {
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    ProgressAsc = "ProgressAsc",
    ProgressDesc = "ProgressDesc"
}

export type Bookmark = DbObject<"Bookmark"> & {
    by: User;
    created_at: Scalars["Date"];
    list: BookmarkList;
    to: BookmarkTo;
    updated_at: Scalars["Date"];
};

export type BookmarkCreateInput = {
    bookmarkFor: BookmarkFor;
    forConnect: Scalars["ID"];
    id: Scalars["ID"];
    listConnect?: InputMaybe<Scalars["ID"]>;
    listCreate?: InputMaybe<BookmarkListCreateInput>;
};

export type BookmarkEdge = Edge<Bookmark, "BookmarkEdge">;

export enum BookmarkFor {
    Api = "Api",
    Code = "Code",
    Comment = "Comment",
    Issue = "Issue",
    Note = "Note",
    Post = "Post",
    Project = "Project",
    Question = "Question",
    QuestionAnswer = "QuestionAnswer",
    Quiz = "Quiz",
    Routine = "Routine",
    Standard = "Standard",
    Tag = "Tag",
    Team = "Team",
    User = "User"
}

export type BookmarkList = DbObject<"BookmarkList"> & {
    bookmarks: Array<Bookmark>;
    bookmarksCount: Scalars["Int"];
    created_at: Scalars["Date"];
    label: Scalars["String"];
    updated_at: Scalars["Date"];
};

export type BookmarkListCreateInput = {
    bookmarksConnect?: InputMaybe<Array<Scalars["ID"]>>;
    bookmarksCreate?: InputMaybe<Array<BookmarkCreateInput>>;
    id: Scalars["ID"];
    label: Scalars["String"];
};

export type BookmarkListEdge = Edge<BookmarkList, "BookmarkListEdge">;

export type BookmarkListSearchInput = BaseSearchInput<BookmarkListSortBy> & {
    bookmarksContainsId?: InputMaybe<Scalars["ID"]>;
    labelsIds?: InputMaybe<Array<Scalars["String"]>>;
    searchString?: InputMaybe<Scalars["String"]>;
};

export type BookmarkListSearchResult = SearchResult<BookmarkListEdge, "BookmarkListSearchResult">;

export enum BookmarkListSortBy {
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    IndexAsc = "IndexAsc",
    IndexDesc = "IndexDesc",
    LabelAsc = "LabelAsc",
    LabelDesc = "LabelDesc"
}

export type BookmarkListUpdateInput = {
    bookmarksConnect?: InputMaybe<Array<Scalars["ID"]>>;
    bookmarksCreate?: InputMaybe<Array<BookmarkCreateInput>>;
    bookmarksDelete?: InputMaybe<Array<Scalars["ID"]>>;
    bookmarksUpdate?: InputMaybe<Array<BookmarkUpdateInput>>;
    id: Scalars["ID"];
    label?: InputMaybe<Scalars["String"]>;
};

export type BookmarkSearchInput = BaseSearchInput<BookmarkSortBy> & {
    apiId?: InputMaybe<Scalars["ID"]>;
    codeId?: InputMaybe<Scalars["ID"]>;
    commentId?: InputMaybe<Scalars["ID"]>;
    excludeLinkedToTag?: InputMaybe<Scalars["Boolean"]>;
    issueId?: InputMaybe<Scalars["ID"]>;
    limitTo?: InputMaybe<Array<BookmarkFor>>;
    listId?: InputMaybe<Scalars["ID"]>;
    listLabel?: InputMaybe<Scalars["String"]>;
    noteId?: InputMaybe<Scalars["ID"]>;
    postId?: InputMaybe<Scalars["ID"]>;
    projectId?: InputMaybe<Scalars["ID"]>;
    questionAnswerId?: InputMaybe<Scalars["ID"]>;
    questionId?: InputMaybe<Scalars["ID"]>;
    quizId?: InputMaybe<Scalars["ID"]>;
    routineId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    standardId?: InputMaybe<Scalars["ID"]>;
    tagId?: InputMaybe<Scalars["ID"]>;
    teamId?: InputMaybe<Scalars["ID"]>;
    userId?: InputMaybe<Scalars["ID"]>;
    visibility?: InputMaybe<VisibilityType>;
};

export type BookmarkSearchResult = SearchResult<BookmarkEdge, "BookmarkSearchResult">;

export enum BookmarkSortBy {
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc"
}

export type BookmarkTo = Api | Code | Comment | Issue | Note | Post | Project | Question | QuestionAnswer | Quiz | Routine | Standard | Tag | Team | User;

export type BookmarkUpdateInput = {
    id: Scalars["ID"];
    listConnect?: InputMaybe<Scalars["ID"]>;
    listUpdate?: InputMaybe<BookmarkListUpdateInput>;
};

export type BotCreateInput = BaseTranslatableCreateInput<UserTranslationCreateInput> & {
    bannerImage?: InputMaybe<Scalars["Upload"]>;
    botSettings: Scalars["String"];
    handle?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    isBotDepictingPerson: Scalars["Boolean"];
    isPrivate: Scalars["Boolean"];
    name: Scalars["String"];
    profileImage?: InputMaybe<Scalars["Upload"]>;
};

export type BotUpdateInput = BaseTranslatableUpdateInput<UserTranslationCreateInput, UserTranslationUpdateInput> & {
    bannerImage?: InputMaybe<Scalars["Upload"]>;
    botSettings?: InputMaybe<Scalars["String"]>;
    handle?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    isBotDepictingPerson?: InputMaybe<Scalars["Boolean"]>;
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    name?: InputMaybe<Scalars["String"]>;
    profileImage?: InputMaybe<Scalars["Upload"]>;
};

export type CancelTaskInput = {
    taskId: Scalars["ID"];
    taskType: TaskType;
};

export type Chat = DbObject<"Chat"> & {
    created_at: Scalars["Date"];
    invites: Array<ChatInvite>;
    invitesCount: Scalars["Int"];
    labels: Array<Label>;
    labelsCount: Scalars["Int"];
    messages: Array<ChatMessage>;
    openToAnyoneWithInvite: Scalars["Boolean"];
    participants: Array<ChatParticipant>;
    participantsCount: Scalars["Int"];
    restrictedToRoles: Array<Role>;
    team?: Maybe<Team>;
    translations: Array<ChatTranslation>;
    translationsCount: Scalars["Int"];
    updated_at: Scalars["Date"];
    you: ChatYou;
};

export type ChatCreateInput = BaseTranslatableCreateInput<ChatTranslationCreateInput> & {
    id: Scalars["ID"];
    invitesCreate?: InputMaybe<Array<ChatInviteCreateInput>>;
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    messagesCreate?: InputMaybe<Array<ChatMessageCreateInput>>;
    openToAnyoneWithInvite?: InputMaybe<Scalars["Boolean"]>;
    restrictedToRolesConnect?: InputMaybe<Array<Scalars["ID"]>>;
    task?: InputMaybe<Scalars["String"]>;
    teamConnect?: InputMaybe<Scalars["ID"]>;
};

export type ChatEdge = Edge<Chat, "ChatEdge">;

export type ChatInvite = DbObject<"ChatInvite"> & {
    chat: Chat;
    created_at: Scalars["Date"];
    message?: Maybe<Scalars["String"]>;
    status: ChatInviteStatus;
    updated_at: Scalars["Date"];
    user: User;
    you: ChatInviteYou;
};

export type ChatInviteCreateInput = {
    chatConnect: Scalars["ID"];
    id: Scalars["ID"];
    message?: InputMaybe<Scalars["String"]>;
    userConnect: Scalars["ID"];
};

export type ChatInviteEdge = Edge<ChatInvite, "ChatInviteEdge">;

export type ChatInviteSearchInput = BaseSearchInput<ChatInviteSortBy> & {
    chatId?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    searchString?: InputMaybe<Scalars["String"]>;
    status?: InputMaybe<ChatInviteStatus>;
    statuses?: InputMaybe<Array<ChatInviteStatus>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userId?: InputMaybe<Scalars["ID"]>;
    visibility?: InputMaybe<VisibilityType>;
};

export type ChatInviteSearchResult = SearchResult<ChatInviteEdge, "ChatInviteSearchResult">;

export enum ChatInviteSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    UserNameAsc = "UserNameAsc",
    UserNameDesc = "UserNameDesc"
}

export enum ChatInviteStatus {
    Accepted = "Accepted",
    Declined = "Declined",
    Pending = "Pending"
}

export type ChatInviteUpdateInput = {
    id: Scalars["ID"];
    message?: InputMaybe<Scalars["String"]>;
};

export type ChatInviteYou = {
    __typename: "ChatInviteYou";
    canDelete: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
};

export type ChatMessage = DbObject<"ChatMessage"> & {
    chat: Chat;
    created_at: Scalars["Date"];
    parent?: Maybe<ChatMessageParent>;
    reactionSummaries: Array<ReactionSummary>;
    reports: Array<Report>;
    reportsCount: Scalars["Int"];
    score: Scalars["Int"];
    sequence: Scalars["Int"];
    translations: Array<ChatMessageTranslation>;
    translationsCount: Scalars["Int"];
    updated_at: Scalars["Date"];
    user: User;
    versionIndex: Scalars["Int"];
    you: ChatMessageYou;
};

export type ChatMessageCreateInput = BaseTranslatableCreateInput<ChatMessageTranslationCreateInput> & {
    chatConnect: Scalars["ID"];
    id: Scalars["ID"];
    parentConnect?: InputMaybe<Scalars["ID"]>;
    userConnect: Scalars["ID"];
    versionIndex: Scalars["Int"];
};

export type ChatMessageCreateWithTaskInfoInput = {
    message: ChatMessageCreateInput;
    model: string;
    task: LlmTask;
    taskContexts: Array<TaskContextInfoInput>;
};

export type ChatMessageEdge = Edge<ChatMessage, "ChatMessageEdge">;

export type ChatMessageParent = {
    __typename: "ChatMessageParent";
    created_at: Scalars["Date"];
    id: Scalars["ID"];
};

export type ChatMessageSearchInput = Omit<BaseSearchInput<ChatMessageSortBy>, "ids"> & {
    chatId: Scalars["ID"];
    createdTimeFrame?: InputMaybe<TimeFrame>;
    minScore?: InputMaybe<Scalars["Int"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userId?: InputMaybe<Scalars["ID"]>;
};

export type ChatMessageSearchResult = SearchResult<ChatMessageEdge, "ChatMessageSearchResult">;

export type ChatMessageSearchTreeInput = {
    chatId: Scalars["ID"];
    excludeDown?: InputMaybe<Scalars["Boolean"]>;
    excludeUp?: InputMaybe<Scalars["Boolean"]>;
    sortBy?: InputMaybe<ChatMessageSortBy>;
    startId?: InputMaybe<Scalars["ID"]>;
    take?: InputMaybe<Scalars["Int"]>;
};

export type ChatMessageSearchTreeResult = {
    __typename: "ChatMessageSearchTreeResult";
    hasMoreDown: Scalars["Boolean"];
    hasMoreUp: Scalars["Boolean"];
    messages: Array<ChatMessage>;
};

export enum ChatMessageSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc"
}

export type ChatMessageTranslation = BaseTranslation<"ChatMessageTranslation"> & {
    text: Scalars["String"];
};

export type ChatMessageTranslationCreateInput = BaseTranslationCreateInput & {
    text: Scalars["String"];
};

export type ChatMessageTranslationUpdateInput = BaseTranslationUpdateInput & {
    text?: InputMaybe<Scalars["String"]>;
};

export type ChatMessageUpdateInput = BaseTranslatableUpdateInput<ChatMessageTranslationCreateInput, ChatMessageTranslationUpdateInput> & {
    id: Scalars["ID"];
};

export type ChatMessageUpdateWithTaskInfoInput = {
    message: ChatMessageUpdateInput;
    model: string;
    task: LlmTask;
    taskContexts: Array<TaskContextInfoInput>;
};

export type ChatMessageYou = {
    __typename: "ChatMessageYou";
    canDelete: Scalars["Boolean"];
    canReact: Scalars["Boolean"];
    canReply: Scalars["Boolean"];
    canReport: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
    reaction?: Maybe<Scalars["String"]>;
};

export type ChatMessageedOn = ApiVersion | CodeVersion | Issue | NoteVersion | Post | ProjectVersion | PullRequest | Question | QuestionAnswer | RoutineVersion | StandardVersion;

export type ChatParticipant = DbObject<"ChatParticipant"> & {
    chat: Chat;
    created_at: Scalars["Date"];
    updated_at: Scalars["Date"];
    user: User;
};

export type ChatParticipantEdge = Edge<ChatParticipant, "ChatParticipantEdge">;

export type ChatParticipantSearchInput = BaseSearchInput<ChatParticipantSortBy> & {
    chatId?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    searchString?: InputMaybe<Scalars["String"]>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userId?: InputMaybe<Scalars["ID"]>;
    visibility?: InputMaybe<VisibilityType>;
};

export type ChatParticipantSearchResult = SearchResult<ChatParticipantEdge, "ChatParticipantSearchResult">;

export enum ChatParticipantSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    UserNameAsc = "UserNameAsc",
    UserNameDesc = "UserNameDesc"
}

export type ChatParticipantUpdateInput = {
    id: Scalars["ID"];
};

export type ChatSearchInput = BaseSearchInput<ChatSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    creatorId?: InputMaybe<Scalars["ID"]>;
    labelsIds?: InputMaybe<Array<Scalars["ID"]>>;
    openToAnyoneWithInvite?: InputMaybe<Scalars["Boolean"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    teamId?: InputMaybe<Scalars["ID"]>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type ChatSearchResult = SearchResult<ChatEdge, "ChatSearchResult">;

export enum ChatSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    InvitesAsc = "InvitesAsc",
    InvitesDesc = "InvitesDesc",
    MessagesAsc = "MessagesAsc",
    MessagesDesc = "MessagesDesc",
    ParticipantsAsc = "ParticipantsAsc",
    ParticipantsDesc = "ParticipantsDesc"
}

export type ChatTranslation = BaseTranslation<"ChatTranslation"> & {
    description?: Maybe<Scalars["String"]>;
    name?: Maybe<Scalars["String"]>;
};

export type ChatTranslationCreateInput = BaseTranslationCreateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type ChatTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type ChatUpdateInput = BaseTranslatableUpdateInput<ChatTranslationCreateInput, ChatTranslationUpdateInput> & {
    id: Scalars["ID"];
    invitesCreate?: InputMaybe<Array<ChatInviteCreateInput>>;
    invitesDelete?: InputMaybe<Array<Scalars["ID"]>>;
    invitesUpdate?: InputMaybe<Array<ChatInviteUpdateInput>>;
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    labelsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    messagesCreate?: InputMaybe<Array<ChatMessageCreateInput>>;
    messagesDelete?: InputMaybe<Array<Scalars["ID"]>>;
    messagesUpdate?: InputMaybe<Array<ChatMessageUpdateInput>>;
    openToAnyoneWithInvite?: InputMaybe<Scalars["Boolean"]>;
    participantsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    restrictedToRolesConnect?: InputMaybe<Array<Scalars["ID"]>>;
    restrictedToRolesDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
};

export type ChatYou = {
    __typename: "ChatYou";
    canDelete: Scalars["Boolean"];
    canInvite: Scalars["Boolean"];
    canRead: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
};

export type CheckTaskStatusesInput = {
    taskIds: Array<Scalars["ID"]>;
    taskType: TaskType;
};

export type CheckTaskStatusesResult = {
    __typename: "CheckTaskStatusesResult";
    statuses: Array<TaskStatusInfo>;
};

export type Code = DbObject<"Code"> & {
    bookmarkedBy: Array<User>;
    bookmarks: Scalars["Int"];
    completedAt?: Maybe<Scalars["Date"]>;
    createdBy?: Maybe<User>;
    created_at: Scalars["Date"];
    hasCompleteVersion: Scalars["Boolean"];
    isDeleted: Scalars["Boolean"];
    isPrivate: Scalars["Boolean"];
    issues: Array<Issue>;
    issuesCount: Scalars["Int"];
    labels: Array<Label>;
    owner?: Maybe<Owner>;
    parent?: Maybe<CodeVersion>;
    permissions: Scalars["String"];
    pullRequests: Array<PullRequest>;
    pullRequestsCount: Scalars["Int"];
    questions: Array<Question>;
    questionsCount: Scalars["Int"];
    score: Scalars["Int"];
    stats: Array<StatsCode>;
    tags: Array<Tag>;
    transfers: Array<Transfer>;
    transfersCount: Scalars["Int"];
    translatedName: Scalars["String"];
    updated_at: Scalars["Date"];
    versions: Array<CodeVersion>;
    versionsCount?: Maybe<Scalars["Int"]>;
    views: Scalars["Int"];
    you: CodeYou;
};

export type CodeCreateInput = {
    id: Scalars["ID"];
    data?: InputMaybe<Scalars["String"]>;
    isPrivate: Scalars["Boolean"];
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    ownedByTeamConnect?: InputMaybe<Scalars["ID"]>;
    ownedByUserConnect?: InputMaybe<Scalars["ID"]>;
    parentConnect?: InputMaybe<Scalars["ID"]>;
    permissions?: InputMaybe<Scalars["String"]>;
    tagsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
    versionsCreate?: InputMaybe<Array<CodeVersionCreateInput>>;
};

export type CodeEdge = Edge<Code, "CodeEdge">;

export type CodeSearchInput = BaseSearchInput<CodeSortBy> & {
    codeLanguageLatestVersion?: InputMaybe<Scalars["String"]>;
    codeTypeLatestVersion?: InputMaybe<CodeType>;
    createdById?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    hasCompleteVersion?: InputMaybe<Scalars["Boolean"]>;
    issuesId?: InputMaybe<Scalars["ID"]>;
    labelsIds?: InputMaybe<Array<Scalars["ID"]>>;
    maxBookmarks?: InputMaybe<Scalars["Int"]>;
    maxScore?: InputMaybe<Scalars["Int"]>;
    maxViews?: InputMaybe<Scalars["Int"]>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    minScore?: InputMaybe<Scalars["Int"]>;
    minViews?: InputMaybe<Scalars["Int"]>;
    ownedByTeamId?: InputMaybe<Scalars["ID"]>;
    ownedByUserId?: InputMaybe<Scalars["ID"]>;
    parentId?: InputMaybe<Scalars["ID"]>;
    pullRequestsId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    tags?: InputMaybe<Array<Scalars["String"]>>;
    translationLanguagesLatestVersion?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type CodeSearchResult = SearchResult<CodeEdge, "CodeSearchResult">;

export enum CodeSortBy {
    BookmarksAsc = "BookmarksAsc",
    BookmarksDesc = "BookmarksDesc",
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
    ScoreAsc = "ScoreAsc",
    ScoreDesc = "ScoreDesc",
    VersionsAsc = "VersionsAsc",
    VersionsDesc = "VersionsDesc",
    ViewsAsc = "ViewsAsc",
    ViewsDesc = "ViewsDesc"
}

export enum CodeType {
    DataConvert = "DataConvert",
    SmartContract = "SmartContract"
}

export type CodeUpdateInput = {
    id: Scalars["ID"];
    data?: InputMaybe<Scalars["String"]>;
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    labelsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    ownedByTeamConnect?: InputMaybe<Scalars["ID"]>;
    ownedByUserConnect?: InputMaybe<Scalars["ID"]>;
    permissions?: InputMaybe<Scalars["String"]>;
    tagsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
    tagsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    versionsCreate?: InputMaybe<Array<CodeVersionCreateInput>>;
    versionsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    versionsUpdate?: InputMaybe<Array<CodeVersionUpdateInput>>;
};

export type CodeVersion = DbObject<"CodeVersion"> & {
    calledByRoutineVersionsCount: Scalars["Int"];
    codeLanguage: Scalars["String"];
    codeType: CodeType;
    comments: Array<Comment>;
    commentsCount: Scalars["Int"];
    completedAt?: Maybe<Scalars["Date"]>;
    content: Scalars["String"];
    created_at: Scalars["Date"];
    data?: Maybe<Scalars["String"]>;
    default?: Maybe<Scalars["String"]>;
    directoryListings: Array<ProjectVersionDirectory>;
    directoryListingsCount: Scalars["Int"];
    forks: Array<Code>;
    forksCount: Scalars["Int"];
    isComplete: Scalars["Boolean"];
    isDeleted: Scalars["Boolean"];
    isLatest: Scalars["Boolean"];
    isPrivate: Scalars["Boolean"];
    pullRequest?: Maybe<PullRequest>;
    reports: Array<Report>;
    reportsCount: Scalars["Int"];
    resourceList?: Maybe<ResourceList>;
    root: Code;
    translations: Array<CodeVersionTranslation>;
    translationsCount: Scalars["Int"];
    updated_at: Scalars["Date"];
    versionIndex: Scalars["Int"];
    versionLabel: Scalars["String"];
    versionNotes?: Maybe<Scalars["String"]>;
    you: VersionYou;
};

export type CodeVersionCreateInput = BaseTranslatableCreateInput<CodeVersionTranslationCreateInput> & {
    codeLanguage: Scalars["String"];
    codeType: CodeType;
    content: Scalars["String"];
    default?: InputMaybe<Scalars["String"]>;
    directoryListingsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    id: Scalars["ID"];
    isComplete?: InputMaybe<Scalars["Boolean"]>;
    isPrivate: Scalars["Boolean"];
    resourceListCreate?: InputMaybe<ResourceListCreateInput>;
    rootConnect: Scalars["ID"];
    rootCreate?: InputMaybe<CodeCreateInput>;
    versionLabel: Scalars["String"];
    versionNotes?: InputMaybe<Scalars["String"]>;
};

export type CodeVersionEdge = Edge<CodeVersion, "CodeVersionEdge">;

export type CodeVersionSearchInput = BaseSearchInput<CodeVersionSortBy> & {
    calledByRoutineVersionId?: InputMaybe<Scalars["ID"]>;
    codeLanguage?: InputMaybe<Scalars["String"]>;
    codeType?: InputMaybe<CodeType>;
    completedTimeFrame?: InputMaybe<TimeFrame>;
    createdByIdRoot?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    isCompleteWithRoot?: InputMaybe<Scalars["Boolean"]>;
    isLatest?: InputMaybe<Scalars["Boolean"]>;
    maxBookmarksRoot?: InputMaybe<Scalars["Int"]>;
    maxScoreRoot?: InputMaybe<Scalars["Int"]>;
    maxViewsRoot?: InputMaybe<Scalars["Int"]>;
    minBookmarksRoot?: InputMaybe<Scalars["Int"]>;
    minScoreRoot?: InputMaybe<Scalars["Int"]>;
    minViewsRoot?: InputMaybe<Scalars["Int"]>;
    ownedByTeamIdRoot?: InputMaybe<Scalars["ID"]>;
    ownedByUserIdRoot?: InputMaybe<Scalars["ID"]>;
    reportId?: InputMaybe<Scalars["ID"]>;
    rootId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    tagsRoot?: InputMaybe<Array<Scalars["String"]>>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userId?: InputMaybe<Scalars["ID"]>;
    visibility?: InputMaybe<VisibilityType>;
};

export type CodeVersionSearchResult = SearchResult<CodeVersionEdge, "CodeVersionSearchResult">;

export enum CodeVersionSortBy {
    CalledByRoutinesAsc = "CalledByRoutinesAsc",
    CalledByRoutinesDesc = "CalledByRoutinesDesc",
    CommentsAsc = "CommentsAsc",
    CommentsDesc = "CommentsDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DirectoryListingsAsc = "DirectoryListingsAsc",
    DirectoryListingsDesc = "DirectoryListingsDesc",
    ForksAsc = "ForksAsc",
    ForksDesc = "ForksDesc",
    ReportsAsc = "ReportsAsc",
    ReportsDesc = "ReportsDesc"
}

export type CodeVersionTranslation = BaseTranslation<"CodeVersionTranslation"> & {
    description?: Maybe<Scalars["String"]>;
    jsonVariable?: Maybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type CodeVersionTranslationCreateInput = BaseTranslationCreateInput & {
    description?: InputMaybe<Scalars["String"]>;
    jsonVariable?: InputMaybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type CodeVersionTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
    jsonVariable?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type CodeVersionUpdateInput = BaseTranslatableUpdateInput<CodeVersionTranslationCreateInput, CodeVersionTranslationUpdateInput> & {
    codeLanguage?: InputMaybe<Scalars["String"]>;
    content?: InputMaybe<Scalars["String"]>;
    default?: InputMaybe<Scalars["String"]>;
    directoryListingsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    directoryListingsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    id: Scalars["ID"];
    isComplete?: InputMaybe<Scalars["Boolean"]>;
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    resourceListCreate?: InputMaybe<ResourceListCreateInput>;
    resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
    rootUpdate?: InputMaybe<CodeUpdateInput>;
    versionLabel?: InputMaybe<Scalars["String"]>;
    versionNotes?: InputMaybe<Scalars["String"]>;
};

export type CodeYou = {
    __typename: "CodeYou";
    canBookmark: Scalars["Boolean"];
    canDelete: Scalars["Boolean"];
    canReact: Scalars["Boolean"];
    canRead: Scalars["Boolean"];
    canTransfer: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
    isBookmarked: Scalars["Boolean"];
    isViewed: Scalars["Boolean"];
    reaction?: Maybe<Scalars["String"]>;
};

export type Comment = DbObject<"Comment"> & {
    bookmarkedBy?: Maybe<Array<User>>;
    bookmarks: Scalars["Int"];
    commentedOn: CommentedOn;
    created_at: Scalars["Date"];
    owner?: Maybe<Owner>;
    reports: Array<Report>;
    reportsCount: Scalars["Int"];
    score: Scalars["Int"];
    translations: Array<CommentTranslation>;
    translationsCount: Scalars["Int"];
    updated_at: Scalars["Date"];
    you: CommentYou;
};

export type CommentCreateInput = BaseTranslatableCreateInput<CommentTranslationCreateInput> & {
    createdFor: CommentFor;
    forConnect: Scalars["ID"];
    id: Scalars["ID"];
    parentConnect?: InputMaybe<Scalars["ID"]>;
};

export enum CommentFor {
    ApiVersion = "ApiVersion",
    CodeVersion = "CodeVersion",
    Issue = "Issue",
    NoteVersion = "NoteVersion",
    Post = "Post",
    ProjectVersion = "ProjectVersion",
    PullRequest = "PullRequest",
    Question = "Question",
    QuestionAnswer = "QuestionAnswer",
    RoutineVersion = "RoutineVersion",
    StandardVersion = "StandardVersion"
}

export type CommentSearchInput = Omit<BaseSearchInput<CommentSortBy>, "ids"> & {
    apiVersionId?: InputMaybe<Scalars["ID"]>;
    codeVersionId?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    issueId?: InputMaybe<Scalars["ID"]>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    minScore?: InputMaybe<Scalars["Int"]>;
    noteVersionId?: InputMaybe<Scalars["ID"]>;
    ownedByTeamId?: InputMaybe<Scalars["ID"]>;
    ownedByUserId?: InputMaybe<Scalars["ID"]>;
    postId?: InputMaybe<Scalars["ID"]>;
    projectVersionId?: InputMaybe<Scalars["ID"]>;
    pullRequestId?: InputMaybe<Scalars["ID"]>;
    questionAnswerId?: InputMaybe<Scalars["ID"]>;
    questionId?: InputMaybe<Scalars["ID"]>;
    routineVersionId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    standardVersionId?: InputMaybe<Scalars["ID"]>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type CommentSearchResult = {
    __typename: "CommentSearchResult";
    endCursor?: Maybe<Scalars["String"]>;
    threads?: Maybe<Array<CommentThread>>;
    totalThreads?: Maybe<Scalars["Int"]>;
};

export enum CommentSortBy {
    BookmarksAsc = "BookmarksAsc",
    BookmarksDesc = "BookmarksDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    ScoreAsc = "ScoreAsc",
    ScoreDesc = "ScoreDesc"
}

export type CommentThread = {
    __typename: "CommentThread";
    childThreads: Array<CommentThread>;
    comment: Comment;
    endCursor?: Maybe<Scalars["String"]>;
    totalInThread?: Maybe<Scalars["Int"]>;
};

export type CommentTranslation = BaseTranslation<"CommentTranslation"> & {
    text: Scalars["String"];
};

export type CommentTranslationCreateInput = BaseTranslationCreateInput & {
    text: Scalars["String"];
};

export type CommentTranslationUpdateInput = BaseTranslationUpdateInput & {
    text?: InputMaybe<Scalars["String"]>;
};

export type CommentUpdateInput = BaseTranslatableUpdateInput<CommentTranslationCreateInput, CommentTranslationUpdateInput> & {
    id: Scalars["ID"];
};

export type CommentYou = {
    __typename: "CommentYou";
    canBookmark: Scalars["Boolean"];
    canDelete: Scalars["Boolean"];
    canReact: Scalars["Boolean"];
    canReply: Scalars["Boolean"];
    canReport: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
    isBookmarked: Scalars["Boolean"];
    reaction?: Maybe<Scalars["String"]>;
};

export type CommentedOn = ApiVersion | CodeVersion | Issue | NoteVersion | Post | ProjectVersion | PullRequest | Question | QuestionAnswer | RoutineVersion | StandardVersion;

export type CopyInput = {
    id: Scalars["ID"];
    intendToPullRequest: Scalars["Boolean"];
    objectType: CopyType;
};

export type CopyResult = {
    __typename: "CopyResult";
    apiVersion?: Maybe<ApiVersion>;
    codeVersion?: Maybe<CodeVersion>;
    noteVersion?: Maybe<NoteVersion>;
    projectVersion?: Maybe<ProjectVersion>;
    routineVersion?: Maybe<RoutineVersion>;
    standardVersion?: Maybe<StandardVersion>;
    team?: Maybe<Team>;
};

export enum CopyType {
    ApiVersion = "ApiVersion",
    CodeVersion = "CodeVersion",
    NoteVersion = "NoteVersion",
    ProjectVersion = "ProjectVersion",
    RoutineVersion = "RoutineVersion",
    StandardVersion = "StandardVersion",
    Team = "Team"
}

export type Count = {
    __typename: "Count";
    count: Scalars["Int"];
};

export type DeleteManyInput = {
    objects: Array<DeleteOneInput>;
};

export type DeleteOneInput = {
    id: Scalars["ID"];
    objectType: DeleteType;
};

export type DeleteAllInput = {
    objectTypes: Array<DeleteType>;
}

export type DeleteAccountInput = {
    deletePublicData: Scalars["Boolean"];
    password: Scalars["String"];
};

export enum DeleteType {
    Api = "Api",
    ApiKey = "ApiKey",
    ApiKeyExternal = "ApiKeyExternal",
    ApiVersion = "ApiVersion",
    Bookmark = "Bookmark",
    BookmarkList = "BookmarkList",
    Chat = "Chat",
    ChatInvite = "ChatInvite",
    ChatMessage = "ChatMessage",
    ChatParticipant = "ChatParticipant",
    Code = "Code",
    CodeVersion = "CodeVersion",
    Comment = "Comment",
    Email = "Email",
    FocusMode = "FocusMode",
    Issue = "Issue",
    Meeting = "Meeting",
    MeetingInvite = "MeetingInvite",
    Member = "Member",
    MemberInvite = "MemberInvite",
    Note = "Note",
    NoteVersion = "NoteVersion",
    Notification = "Notification",
    Phone = "Phone",
    Post = "Post",
    Project = "Project",
    ProjectVersion = "ProjectVersion",
    PullRequest = "PullRequest",
    PushDevice = "PushDevice",
    Question = "Question",
    QuestionAnswer = "QuestionAnswer",
    Quiz = "Quiz",
    Reminder = "Reminder",
    ReminderList = "ReminderList",
    Report = "Report",
    Resource = "Resource",
    Role = "Role",
    Routine = "Routine",
    RoutineVersion = "RoutineVersion",
    RunProject = "RunProject",
    RunRoutine = "RunRoutine",
    Schedule = "Schedule",
    Standard = "Standard",
    StandardVersion = "StandardVersion",
    Team = "Team",
    Transfer = "Transfer",
    User = "User",
    Wallet = "Wallet"
}

export type Email = {
    __typename: "Email";
    emailAddress: Scalars["String"];
    id: Scalars["ID"];
    verified: Scalars["Boolean"];
};

export type EmailCreateInput = {
    emailAddress: Scalars["String"];
    id: Scalars["ID"];
};

export type EmailLogInInput = {
    email?: InputMaybe<Scalars["String"]>;
    password?: InputMaybe<Scalars["String"]>;
    verificationCode?: InputMaybe<Scalars["String"]>;
};

export type EmailRequestPasswordChangeInput = {
    email: Scalars["String"];
};

export type EmailResetPasswordInput = {
    code: Scalars["String"];
    id: Scalars["ID"];
    newPassword: Scalars["String"];
};

export type EmailSignUpInput = {
    confirmPassword: Scalars["String"];
    email: Scalars["String"];
    marketingEmails: Scalars["Boolean"];
    name: Scalars["String"];
    password: Scalars["String"];
    theme: Scalars["String"];
};

export type FindByIdInput = {
    id: Scalars["ID"];
};

export type FindByIdOrHandleInput = {
    handle?: InputMaybe<Scalars["String"]>;
    id?: InputMaybe<Scalars["ID"]>;
};

export type FindVersionInput = {
    handleRoot?: InputMaybe<Scalars["String"]>;
    id?: InputMaybe<Scalars["ID"]>;
    idRoot?: InputMaybe<Scalars["ID"]>;
};

export type FocusMode = DbObject<"FocusMode"> & {
    created_at: Scalars["Date"];
    description?: Maybe<Scalars["String"]>;
    filters: Array<FocusModeFilter>;
    labels: Array<Label>;
    name: Scalars["String"];
    reminderList?: Maybe<ReminderList>;
    resourceList?: Maybe<ResourceList>;
    schedule?: Maybe<Schedule>;
    updated_at: Scalars["Date"];
    you: FocusModeYou;
};

export type FocusModeCreateInput = {
    description?: InputMaybe<Scalars["String"]>;
    filtersCreate?: InputMaybe<Array<FocusModeFilterCreateInput>>;
    id: Scalars["ID"];
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    name: Scalars["String"];
    reminderListConnect?: InputMaybe<Scalars["ID"]>;
    reminderListCreate?: InputMaybe<ReminderListCreateInput>;
    resourceListCreate?: InputMaybe<ResourceListCreateInput>;
    scheduleCreate?: InputMaybe<ScheduleCreateInput>;
};

export type FocusModeEdge = Edge<FocusMode, "FocusModeEdge">;

export type FocusModeFilter = DbObject<"FocusModeFilter"> & {
    filterType: FocusModeFilterType;
    focusMode: FocusMode;
    tag: Tag;
};

export type FocusModeFilterCreateInput = {
    filterType: FocusModeFilterType;
    focusModeConnect: Scalars["ID"];
    id: Scalars["ID"];
    tagConnect?: InputMaybe<Scalars["ID"]>;
    tagCreate?: InputMaybe<TagCreateInput>;
};

export enum FocusModeFilterType {
    Blur = "Blur",
    Hide = "Hide",
    ShowMore = "ShowMore"
}

export type FocusModeSearchInput = BaseSearchInput<FocusModeSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    labelsIds?: InputMaybe<Array<Scalars["ID"]>>;
    scheduleEndTimeFrame?: InputMaybe<TimeFrame>;
    scheduleStartTimeFrame?: InputMaybe<TimeFrame>;
    searchString?: InputMaybe<Scalars["String"]>;
    timeZone?: InputMaybe<Scalars["String"]>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type FocusModeSearchResult = SearchResult<FocusModeEdge, "FocusModeSearchResult">;

export enum FocusModeSortBy {
    NameAsc = "NameAsc",
    NameDesc = "NameDesc"
}

export enum FocusModeStopCondition {
    AfterStopTime = "AfterStopTime",
    Automatic = "Automatic",
    Never = "Never",
    NextBegins = "NextBegins"
}

export type FocusModeUpdateInput = {
    description?: InputMaybe<Scalars["String"]>;
    filtersCreate?: InputMaybe<Array<FocusModeFilterCreateInput>>;
    filtersDelete?: InputMaybe<Array<Scalars["ID"]>>;
    id: Scalars["ID"];
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    labelsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    name?: InputMaybe<Scalars["String"]>;
    reminderListConnect?: InputMaybe<Scalars["ID"]>;
    reminderListCreate?: InputMaybe<ReminderListCreateInput>;
    reminderListDisconnect?: InputMaybe<Scalars["Boolean"]>;
    reminderListUpdate?: InputMaybe<ReminderListUpdateInput>;
    resourceListCreate?: InputMaybe<ResourceListCreateInput>;
    resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
    scheduleCreate?: InputMaybe<ScheduleCreateInput>;
    scheduleUpdate?: InputMaybe<ScheduleUpdateInput>;
};

export type FocusModeYou = {
    __typename: "FocusModeYou";
    canDelete: Scalars["Boolean"];
    canRead: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
};

export enum ModelType {
    ActiveFocusMode = "ActiveFocusMode",
    Api = "Api",
    ApiKey = "ApiKey",
    ApiKeyExternal = "ApiKeyExternal",
    ApiVersion = "ApiVersion",
    Award = "Award",
    Bookmark = "Bookmark",
    BookmarkList = "BookmarkList",
    Chat = "Chat",
    ChatInvite = "ChatInvite",
    ChatMessage = "ChatMessage",
    ChatMessageSearchTreeResult = "ChatMessageSearchTreeResult",
    ChatParticipant = "ChatParticipant",
    Code = "Code",
    CodeVersion = "CodeVersion",
    Comment = "Comment",
    Copy = "Copy",
    Email = "Email",
    FocusMode = "FocusMode",
    FocusModeFilter = "FocusModeFilter",
    Fork = "Fork",
    Handle = "Handle",
    HomeResult = "HomeResult",
    Issue = "Issue",
    Label = "Label",
    Meeting = "Meeting",
    MeetingInvite = "MeetingInvite",
    Member = "Member",
    MemberInvite = "MemberInvite",
    Note = "Note",
    NoteVersion = "NoteVersion",
    Notification = "Notification",
    NotificationSubscription = "NotificationSubscription",
    Payment = "Payment",
    Phone = "Phone",
    PopularResult = "PopularResult",
    Post = "Post",
    Premium = "Premium",
    Project = "Project",
    ProjectOrRoutineSearchResult = "ProjectOrRoutineSearchResult",
    ProjectOrTeamSearchResult = "ProjectOrTeamSearchResult",
    ProjectVersion = "ProjectVersion",
    ProjectVersionContentsSearchResult = "ProjectVersionContentsSearchResult",
    ProjectVersionDirectory = "ProjectVersionDirectory",
    PullRequest = "PullRequest",
    PushDevice = "PushDevice",
    Question = "Question",
    QuestionAnswer = "QuestionAnswer",
    Quiz = "Quiz",
    QuizAttempt = "QuizAttempt",
    QuizQuestion = "QuizQuestion",
    QuizQuestionResponse = "QuizQuestionResponse",
    Reaction = "Reaction",
    ReactionSummary = "ReactionSummary",
    Reminder = "Reminder",
    ReminderItem = "ReminderItem",
    ReminderList = "ReminderList",
    Report = "Report",
    ReportResponse = "ReportResponse",
    ReputationHistory = "ReputationHistory",
    Resource = "Resource",
    ResourceList = "ResourceList",
    Role = "Role",
    Routine = "Routine",
    RoutineVersion = "RoutineVersion",
    RoutineVersionInput = "RoutineVersionInput",
    RoutineVersionOutput = "RoutineVersionOutput",
    RunProject = "RunProject",
    RunProjectOrRunRoutineSearchResult = "RunProjectOrRunRoutineSearchResult",
    RunProjectStep = "RunProjectStep",
    RunRoutine = "RunRoutine",
    RunRoutineIO = "RunRoutineIO",
    RunRoutineStep = "RunRoutineStep",
    Schedule = "Schedule",
    ScheduleException = "ScheduleException",
    ScheduleRecurrence = "ScheduleRecurrence",
    Session = "Session",
    SessionUser = "SessionUser",
    Standard = "Standard",
    StandardVersion = "StandardVersion",
    StatsApi = "StatsApi",
    StatsCode = "StatsCode",
    StatsProject = "StatsProject",
    StatsQuiz = "StatsQuiz",
    StatsRoutine = "StatsRoutine",
    StatsSite = "StatsSite",
    StatsStandard = "StatsStandard",
    StatsTeam = "StatsTeam",
    StatsUser = "StatsUser",
    Tag = "Tag",
    Team = "Team",
    Transfer = "Transfer",
    User = "User",
    View = "View",
    Wallet = "Wallet"
}

export type HomeResult = {
    __typename: "HomeResult";
    recommended: Array<Resource>;
    reminders: Array<Reminder>;
    resources: Array<Resource>;
    schedules: Array<Schedule>;
};

export type ImportCalendarInput = {
    file: Scalars["Upload"];
};

export type Issue = DbObject<"Issue"> & {
    bookmarkedBy?: Maybe<Array<Bookmark>>;
    bookmarks: Scalars["Int"];
    closedAt?: Maybe<Scalars["Date"]>;
    closedBy?: Maybe<User>;
    comments: Array<Comment>;
    commentsCount: Scalars["Int"];
    createdBy?: Maybe<User>;
    created_at: Scalars["Date"];
    labels: Array<Label>;
    labelsCount: Scalars["Int"];
    referencedVersionId?: Maybe<Scalars["String"]>;
    reports: Array<Report>;
    reportsCount: Scalars["Int"];
    score: Scalars["Int"];
    status: IssueStatus;
    to: IssueTo;
    translations: Array<IssueTranslation>;
    translationsCount: Scalars["Int"];
    updated_at: Scalars["Date"];
    views: Scalars["Int"];
    you: IssueYou;
};

export type IssueCloseInput = {
    id: Scalars["ID"];
    status: IssueStatus;
};

export type IssueCreateInput = BaseTranslatableCreateInput<IssueTranslationCreateInput> & {
    forConnect: Scalars["ID"];
    id: Scalars["ID"];
    issueFor: IssueFor;
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    referencedVersionIdConnect?: InputMaybe<Scalars["ID"]>;
};

export type IssueEdge = Edge<Issue, "IssueEdge">;

export enum IssueFor {
    Api = "Api",
    Code = "Code",
    Note = "Note",
    Project = "Project",
    Routine = "Routine",
    Standard = "Standard",
    Team = "Team"
}

export type IssueSearchInput = BaseSearchInput<IssueSortBy> & {
    apiId?: InputMaybe<Scalars["ID"]>;
    closedById?: InputMaybe<Scalars["ID"]>;
    codeId?: InputMaybe<Scalars["ID"]>;
    createdById?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    minScore?: InputMaybe<Scalars["Int"]>;
    minViews?: InputMaybe<Scalars["Int"]>;
    noteId?: InputMaybe<Scalars["ID"]>;
    projectId?: InputMaybe<Scalars["ID"]>;
    referencedVersionId?: InputMaybe<Scalars["ID"]>;
    routineId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    standardId?: InputMaybe<Scalars["ID"]>;
    status?: InputMaybe<IssueStatus>;
    teamId?: InputMaybe<Scalars["ID"]>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type IssueSearchResult = SearchResult<IssueEdge, "IssueSearchResult">;

export enum IssueSortBy {
    BookmarksAsc = "BookmarksAsc",
    BookmarksDesc = "BookmarksDesc",
    CommentsAsc = "CommentsAsc",
    CommentsDesc = "CommentsDesc",
    DateCompletedAsc = "DateCompletedAsc",
    DateCompletedDesc = "DateCompletedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    ForksAsc = "ForksAsc",
    ForksDesc = "ForksDesc",
    ScoreAsc = "ScoreAsc",
    ScoreDesc = "ScoreDesc"
}

export enum IssueStatus {
    ClosedResolved = "ClosedResolved",
    ClosedUnresolved = "ClosedUnresolved",
    Draft = "Draft",
    Open = "Open",
    Rejected = "Rejected"
}

export type IssueTo = Api | Code | Note | Project | Routine | Standard | Team;

export type IssueTranslation = BaseTranslation<"IssueTranslation"> & {
    description?: Maybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type IssueTranslationCreateInput = BaseTranslationCreateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type IssueTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type IssueUpdateInput = BaseTranslatableUpdateInput<IssueTranslationCreateInput, IssueTranslationUpdateInput> & {
    id: Scalars["ID"];
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    labelsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
};

export type IssueYou = {
    __typename: "IssueYou";
    canBookmark: Scalars["Boolean"];
    canComment: Scalars["Boolean"];
    canDelete: Scalars["Boolean"];
    canReact: Scalars["Boolean"];
    canRead: Scalars["Boolean"];
    canReport: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
    isBookmarked: Scalars["Boolean"];
    reaction?: Maybe<Scalars["String"]>;
};

export type Label = DbObject<"Label"> & {
    apis?: Maybe<Array<Api>>;
    apisCount: Scalars["Int"];
    codes?: Maybe<Array<Code>>;
    codesCount: Scalars["Int"];
    color?: Maybe<Scalars["String"]>;
    created_at: Scalars["Date"];
    focusModes?: Maybe<Array<FocusMode>>;
    focusModesCount: Scalars["Int"];
    issues?: Maybe<Array<Issue>>;
    issuesCount: Scalars["Int"];
    label: Scalars["String"];
    meetings?: Maybe<Array<Meeting>>;
    meetingsCount: Scalars["Int"];
    notes?: Maybe<Array<Note>>;
    notesCount: Scalars["Int"];
    owner: Owner;
    projects?: Maybe<Array<Project>>;
    projectsCount: Scalars["Int"];
    routines?: Maybe<Array<Routine>>;
    routinesCount: Scalars["Int"];
    schedules?: Maybe<Array<Schedule>>;
    schedulesCount: Scalars["Int"];
    standards?: Maybe<Array<Standard>>;
    standardsCount: Scalars["Int"];
    translations: Array<LabelTranslation>;
    translationsCount: Scalars["Int"];
    updated_at: Scalars["Date"];
    you: LabelYou;
};

export type LabelCreateInput = BaseTranslatableCreateInput<LabelTranslationCreateInput> & {
    color?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    label: Scalars["String"];
    teamConnect?: InputMaybe<Scalars["ID"]>;
};

export type LabelEdge = Edge<Label, "LabelEdge">;

export type LabelSearchInput = BaseSearchInput<LabelSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    ownedByTeamId?: InputMaybe<Scalars["ID"]>;
    ownedByUserId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type LabelSearchResult = SearchResult<LabelEdge, "LabelSearchResult">;

export enum LabelSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc"
}

export type LabelTranslation = BaseTranslation<"LabelTranslation"> & {
    description: Scalars["String"];
};

export type LabelTranslationCreateInput = BaseTranslationCreateInput & {
    description: Scalars["String"];
};

export type LabelTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
};

export type LabelUpdateInput = BaseTranslatableUpdateInput<LabelTranslationCreateInput, LabelTranslationUpdateInput> & {
    apisConnect?: InputMaybe<Array<Scalars["ID"]>>;
    apisDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    codesConnect?: InputMaybe<Array<Scalars["ID"]>>;
    codesDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    color?: InputMaybe<Scalars["String"]>;
    focusModesConnect?: InputMaybe<Array<Scalars["ID"]>>;
    focusModesDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    id: Scalars["ID"];
    issuesConnect?: InputMaybe<Array<Scalars["ID"]>>;
    issuesDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    label?: InputMaybe<Scalars["String"]>;
    meetingsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    meetingsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    notesConnect?: InputMaybe<Array<Scalars["ID"]>>;
    notesDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    projectsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    projectsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    routinesConnect?: InputMaybe<Array<Scalars["ID"]>>;
    routinesDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    schedulesConnect?: InputMaybe<Array<Scalars["ID"]>>;
    schedulesDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    standardsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    standardsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
};

export type LabelYou = {
    __typename: "LabelYou";
    canDelete: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
};

export enum LlmTask {
    ApiAdd = "ApiAdd",
    ApiDelete = "ApiDelete",
    ApiFind = "ApiFind",
    ApiUpdate = "ApiUpdate",
    BotAdd = "BotAdd",
    BotDelete = "BotDelete",
    BotFind = "BotFind",
    BotUpdate = "BotUpdate",
    DataConverterAdd = "DataConverterAdd",
    DataConverterDelete = "DataConverterDelete",
    DataConverterFind = "DataConverterFind",
    DataConverterUpdate = "DataConverterUpdate",
    MembersAdd = "MembersAdd",
    MembersDelete = "MembersDelete",
    MembersFind = "MembersFind",
    MembersUpdate = "MembersUpdate",
    NoteAdd = "NoteAdd",
    NoteDelete = "NoteDelete",
    NoteFind = "NoteFind",
    NoteUpdate = "NoteUpdate",
    ProjectAdd = "ProjectAdd",
    ProjectDelete = "ProjectDelete",
    ProjectFind = "ProjectFind",
    ProjectUpdate = "ProjectUpdate",
    QuestionAdd = "QuestionAdd",
    QuestionDelete = "QuestionDelete",
    QuestionFind = "QuestionFind",
    QuestionUpdate = "QuestionUpdate",
    ReminderAdd = "ReminderAdd",
    ReminderDelete = "ReminderDelete",
    ReminderFind = "ReminderFind",
    ReminderUpdate = "ReminderUpdate",
    RoleAdd = "RoleAdd",
    RoleDelete = "RoleDelete",
    RoleFind = "RoleFind",
    RoleUpdate = "RoleUpdate",
    RoutineAdd = "RoutineAdd",
    RoutineDelete = "RoutineDelete",
    RoutineFind = "RoutineFind",
    RoutineUpdate = "RoutineUpdate",
    RunProjectStart = "RunProjectStart",
    RunRoutineStart = "RunRoutineStart",
    ScheduleAdd = "ScheduleAdd",
    ScheduleDelete = "ScheduleDelete",
    ScheduleFind = "ScheduleFind",
    ScheduleUpdate = "ScheduleUpdate",
    SmartContractAdd = "SmartContractAdd",
    SmartContractDelete = "SmartContractDelete",
    SmartContractFind = "SmartContractFind",
    SmartContractUpdate = "SmartContractUpdate",
    StandardAdd = "StandardAdd",
    StandardDelete = "StandardDelete",
    StandardFind = "StandardFind",
    StandardUpdate = "StandardUpdate",
    Start = "Start",
    TeamAdd = "TeamAdd",
    TeamDelete = "TeamDelete",
    TeamFind = "TeamFind",
    TeamUpdate = "TeamUpdate"
}

export type Meeting = DbObject<"Meeting"> & {
    attendees: Array<User>;
    attendeesCount: Scalars["Int"];
    created_at: Scalars["Date"];
    invites: Array<MeetingInvite>;
    invitesCount: Scalars["Int"];
    labels: Array<Label>;
    labelsCount: Scalars["Int"];
    openToAnyoneWithInvite: Scalars["Boolean"];
    restrictedToRoles: Array<Role>;
    schedule?: Maybe<Schedule>;
    showOnTeamProfile: Scalars["Boolean"];
    team: Team;
    translations: Array<MeetingTranslation>;
    translationsCount: Scalars["Int"];
    updated_at: Scalars["Date"];
    you: MeetingYou;
};

export type MeetingCreateInput = BaseTranslatableCreateInput<MeetingTranslationCreateInput> & {
    id: Scalars["ID"];
    invitesCreate?: InputMaybe<Array<MeetingInviteCreateInput>>;
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    openToAnyoneWithInvite?: InputMaybe<Scalars["Boolean"]>;
    restrictedToRolesConnect?: InputMaybe<Array<Scalars["ID"]>>;
    scheduleCreate?: InputMaybe<ScheduleCreateInput>;
    showOnTeamProfile?: InputMaybe<Scalars["Boolean"]>;
    teamConnect: Scalars["ID"];
};

export type MeetingEdge = Edge<Meeting, "MeetingEdge">;

export type MeetingInvite = DbObject<"MeetingInvite"> & {
    created_at: Scalars["Date"];
    meeting: Meeting;
    message?: Maybe<Scalars["String"]>;
    status: MeetingInviteStatus;
    updated_at: Scalars["Date"];
    user: User;
    you: MeetingInviteYou;
};

export type MeetingInviteCreateInput = {
    id: Scalars["ID"];
    meetingConnect: Scalars["ID"];
    message?: InputMaybe<Scalars["String"]>;
    userConnect: Scalars["ID"];
};

export type MeetingInviteEdge = Edge<MeetingInvite, "MeetingInviteEdge">;

export type MeetingInviteSearchInput = BaseSearchInput<MeetingInviteSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    meetingId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    status?: InputMaybe<MeetingInviteStatus>;
    statuses?: InputMaybe<Array<MeetingInviteStatus>>;
    teamId?: InputMaybe<Scalars["ID"]>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userId?: InputMaybe<Scalars["ID"]>;
    visibility?: InputMaybe<VisibilityType>;
};

export type MeetingInviteSearchResult = SearchResult<MeetingInviteEdge, "MeetingInviteSearchResult">;

export enum MeetingInviteSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    MeetingEndAsc = "MeetingEndAsc",
    MeetingEndDesc = "MeetingEndDesc",
    MeetingStartAsc = "MeetingStartAsc",
    MeetingStartDesc = "MeetingStartDesc",
    UserNameAsc = "UserNameAsc",
    UserNameDesc = "UserNameDesc"
}

export enum MeetingInviteStatus {
    Accepted = "Accepted",
    Declined = "Declined",
    Pending = "Pending"
}

export type MeetingInviteUpdateInput = {
    id: Scalars["ID"];
    message?: InputMaybe<Scalars["String"]>;
};

export type MeetingInviteYou = {
    __typename: "MeetingInviteYou";
    canDelete: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
};

export type MeetingSearchInput = BaseSearchInput<MeetingSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    labelsIds?: InputMaybe<Array<Scalars["ID"]>>;
    openToAnyoneWithInvite?: InputMaybe<Scalars["Boolean"]>;
    scheduleEndTimeFrame?: InputMaybe<TimeFrame>;
    scheduleStartTimeFrame?: InputMaybe<TimeFrame>;
    searchString?: InputMaybe<Scalars["String"]>;
    showOnTeamProfile?: InputMaybe<Scalars["Boolean"]>;
    teamId?: InputMaybe<Scalars["ID"]>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type MeetingSearchResult = SearchResult<MeetingEdge, "MeetingSearchResult">;

export enum MeetingSortBy {
    AttendeesAsc = "AttendeesAsc",
    AttendeesDesc = "AttendeesDesc",
    InvitesAsc = "InvitesAsc",
    InvitesDesc = "InvitesDesc"
}

export type MeetingTranslation = BaseTranslation<"MeetingTranslation"> & {
    description?: Maybe<Scalars["String"]>;
    link?: Maybe<Scalars["String"]>;
    name?: Maybe<Scalars["String"]>;
};

export type MeetingTranslationCreateInput = BaseTranslationCreateInput & {
    description?: InputMaybe<Scalars["String"]>;
    link?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type MeetingTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
    link?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type MeetingUpdateInput = BaseTranslatableUpdateInput<MeetingTranslationCreateInput, MeetingTranslationUpdateInput> & {
    id: Scalars["ID"];
    invitesCreate?: InputMaybe<Array<MeetingInviteCreateInput>>;
    invitesDelete?: InputMaybe<Array<Scalars["ID"]>>;
    invitesUpdate?: InputMaybe<Array<MeetingInviteUpdateInput>>;
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    labelsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    openToAnyoneWithInvite?: InputMaybe<Scalars["Boolean"]>;
    restrictedToRolesConnect?: InputMaybe<Array<Scalars["ID"]>>;
    restrictedToRolesDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    scheduleCreate?: InputMaybe<ScheduleCreateInput>;
    scheduleUpdate?: InputMaybe<ScheduleUpdateInput>;
    showOnTeamProfile?: InputMaybe<Scalars["Boolean"]>;
};

export type MeetingYou = {
    __typename: "MeetingYou";
    canDelete: Scalars["Boolean"];
    canInvite: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
};

export type Member = DbObject<"Member"> & {
    created_at: Scalars["Date"];
    isAdmin: Scalars["Boolean"];
    permissions: Scalars["String"];
    roles: Array<Role>;
    team: Team;
    updated_at: Scalars["Date"];
    user: User;
    you: MemberYou;
};

export type MemberEdge = Edge<Member, "MemberEdge">;

export type MemberInvite = DbObject<"MemberInvite"> & {
    created_at: Scalars["Date"];
    message?: Maybe<Scalars["String"]>;
    status: MemberInviteStatus;
    team: Team;
    updated_at: Scalars["Date"];
    user: User;
    willBeAdmin: Scalars["Boolean"];
    willHavePermissions?: Maybe<Scalars["String"]>;
    you: MemberInviteYou;
};

export type MemberInviteCreateInput = {
    id: Scalars["ID"];
    message?: InputMaybe<Scalars["String"]>;
    teamConnect: Scalars["ID"];
    userConnect: Scalars["ID"];
    willBeAdmin?: InputMaybe<Scalars["Boolean"]>;
    willHavePermissions?: InputMaybe<Scalars["String"]>;
};

export type MemberInviteEdge = Edge<MemberInvite, "MemberInviteEdge">;

export type MemberInviteSearchInput = BaseSearchInput<MemberInviteSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    searchString?: InputMaybe<Scalars["String"]>;
    status?: InputMaybe<MemberInviteStatus>;
    statuses?: InputMaybe<Array<MemberInviteStatus>>;
    teamId?: InputMaybe<Scalars["ID"]>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userId?: InputMaybe<Scalars["ID"]>;
    visibility?: InputMaybe<VisibilityType>;
};

export type MemberInviteSearchResult = SearchResult<MemberInviteEdge, "MemberInviteSearchResult">;

export enum MemberInviteSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    UserNameAsc = "UserNameAsc",
    UserNameDesc = "UserNameDesc"
}

export enum MemberInviteStatus {
    Accepted = "Accepted",
    Declined = "Declined",
    Pending = "Pending"
}

export type MemberInviteUpdateInput = {
    id: Scalars["ID"];
    message?: InputMaybe<Scalars["String"]>;
    willBeAdmin?: InputMaybe<Scalars["Boolean"]>;
    willHavePermissions?: InputMaybe<Scalars["String"]>;
};

export type MemberInviteYou = {
    __typename: "MemberInviteYou";
    canDelete: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
};

export type MemberSearchInput = BaseSearchInput<MemberSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    isAdmin?: InputMaybe<Scalars["Boolean"]>;
    roles?: InputMaybe<Array<Scalars["String"]>>;
    searchString?: InputMaybe<Scalars["String"]>;
    teamId?: InputMaybe<Scalars["ID"]>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userId?: InputMaybe<Scalars["ID"]>;
    visibility?: InputMaybe<VisibilityType>;
};

export type MemberSearchResult = SearchResult<MemberEdge, "MemberSearchResult">;

export enum MemberSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc"
}

export type MemberUpdateInput = {
    id: Scalars["ID"];
    isAdmin?: InputMaybe<Scalars["Boolean"]>;
    permissions?: InputMaybe<Scalars["String"]>;
};

export type MemberYou = {
    __typename: "MemberYou";
    canDelete: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
};

export type Note = DbObject<"Note"> & {
    bookmarkedBy: Array<User>;
    bookmarks: Scalars["Int"];
    createdBy?: Maybe<User>;
    created_at: Scalars["Date"];
    isPrivate: Scalars["Boolean"];
    issues: Array<Issue>;
    issuesCount: Scalars["Int"];
    labels: Array<Label>;
    labelsCount: Scalars["Int"];
    owner?: Maybe<Owner>;
    parent?: Maybe<NoteVersion>;
    permissions: Scalars["String"];
    pullRequests: Array<PullRequest>;
    pullRequestsCount: Scalars["Int"];
    questions: Array<Question>;
    questionsCount: Scalars["Int"];
    score: Scalars["Int"];
    tags: Array<Tag>;
    transfers: Array<Transfer>;
    transfersCount: Scalars["Int"];
    updated_at: Scalars["Date"];
    versions: Array<NoteVersion>;
    versionsCount: Scalars["Int"];
    views: Scalars["Int"];
    you: NoteYou;
};

export type NoteCreateInput = {
    id: Scalars["ID"];
    isPrivate: Scalars["Boolean"];
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    ownedByTeamConnect?: InputMaybe<Scalars["ID"]>;
    ownedByUserConnect?: InputMaybe<Scalars["ID"]>;
    parentConnect?: InputMaybe<Scalars["ID"]>;
    permissions?: InputMaybe<Scalars["String"]>;
    tagsConnect?: InputMaybe<Array<Scalars["String"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
    versionsCreate?: InputMaybe<Array<NoteVersionCreateInput>>;
};

export type NoteEdge = Edge<Note, "NoteEdge">;

export type NotePage = DbObject<"NotePage"> & {
    pageIndex: Scalars["Int"];
    text: Scalars["String"];
};

export type NotePageCreateInput = {
    id: Scalars["ID"];
    pageIndex: Scalars["Int"];
    text: Scalars["String"];
};

export type NotePageUpdateInput = {
    id: Scalars["ID"];
    pageIndex?: InputMaybe<Scalars["Int"]>;
    text?: InputMaybe<Scalars["String"]>;
};

export type NoteSearchInput = BaseSearchInput<NoteSortBy> & {
    createdById?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    maxBookmarks?: InputMaybe<Scalars["Int"]>;
    maxScore?: InputMaybe<Scalars["Int"]>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    minScore?: InputMaybe<Scalars["Int"]>;
    ownedByTeamId?: InputMaybe<Scalars["ID"]>;
    ownedByUserId?: InputMaybe<Scalars["ID"]>;
    parentId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    tags?: InputMaybe<Array<Scalars["String"]>>;
    translationLanguagesLatestVersion?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type NoteSearchResult = SearchResult<NoteEdge, "NoteSearchResult">;

export enum NoteSortBy {
    BookmarksAsc = "BookmarksAsc",
    BookmarksDesc = "BookmarksDesc",
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
    ScoreAsc = "ScoreAsc",
    ScoreDesc = "ScoreDesc",
    VersionsAsc = "VersionsAsc",
    VersionsDesc = "VersionsDesc",
    ViewsAsc = "ViewsAsc",
    ViewsDesc = "ViewsDesc"
}

export type NoteUpdateInput = {
    id: Scalars["ID"];
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    labelsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    ownedByTeamConnect?: InputMaybe<Scalars["ID"]>;
    ownedByUserConnect?: InputMaybe<Scalars["ID"]>;
    permissions?: InputMaybe<Scalars["String"]>;
    tagsConnect?: InputMaybe<Array<Scalars["String"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
    tagsDisconnect?: InputMaybe<Array<Scalars["String"]>>;
    versionsCreate?: InputMaybe<Array<NoteVersionCreateInput>>;
    versionsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    versionsUpdate?: InputMaybe<Array<NoteVersionUpdateInput>>;
};

export type NoteVersion = DbObject<"NoteVersion"> & {
    comments: Array<Comment>;
    commentsCount: Scalars["Int"];
    created_at: Scalars["Date"];
    directoryListings: Array<ProjectVersionDirectory>;
    directoryListingsCount: Scalars["Int"];
    forks: Array<Note>;
    forksCount: Scalars["Int"];
    isLatest: Scalars["Boolean"];
    isPrivate: Scalars["Boolean"];
    pullRequest?: Maybe<PullRequest>;
    reports: Array<Report>;
    reportsCount: Scalars["Int"];
    root: Note;
    translations: Array<NoteVersionTranslation>;
    updated_at: Scalars["Date"];
    versionIndex: Scalars["Int"];
    versionLabel: Scalars["String"];
    versionNotes?: Maybe<Scalars["String"]>;
    you: VersionYou;
};

export type NoteVersionCreateInput = BaseTranslatableCreateInput<NoteVersionTranslationCreateInput> & {
    directoryListingsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    id: Scalars["ID"];
    isPrivate: Scalars["Boolean"];
    rootConnect?: InputMaybe<Scalars["ID"]>;
    rootCreate?: InputMaybe<NoteCreateInput>;
    versionLabel: Scalars["String"];
    versionNotes?: InputMaybe<Scalars["String"]>;
};

export type NoteVersionEdge = Edge<NoteVersion, "NoteVersionEdge">;

export type NoteVersionSearchInput = BaseSearchInput<NoteVersionSortBy> & {
    createdByIdRoot?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    isLatest?: InputMaybe<Scalars["Boolean"]>;
    maxBookmarksRoot?: InputMaybe<Scalars["Int"]>;
    maxScoreRoot?: InputMaybe<Scalars["Int"]>;
    maxViewsRoot?: InputMaybe<Scalars["Int"]>;
    minBookmarksRoot?: InputMaybe<Scalars["Int"]>;
    minScoreRoot?: InputMaybe<Scalars["Int"]>;
    minViewsRoot?: InputMaybe<Scalars["Int"]>;
    ownedByTeamIdRoot?: InputMaybe<Scalars["ID"]>;
    ownedByUserIdRoot?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    tagsRoot?: InputMaybe<Array<Scalars["String"]>>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type NoteVersionSearchResult = SearchResult<NoteVersionEdge, "NoteVersionSearchResult">;

export enum NoteVersionSortBy {
    CommentsAsc = "CommentsAsc",
    CommentsDesc = "CommentsDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DirectoryListingsAsc = "DirectoryListingsAsc",
    DirectoryListingsDesc = "DirectoryListingsDesc",
    ForksAsc = "ForksAsc",
    ForksDesc = "ForksDesc",
    ReportsAsc = "ReportsAsc",
    ReportsDesc = "ReportsDesc"
}

export type NoteVersionTranslation = BaseTranslation<"NoteVersionTranslation"> & {
    description?: Maybe<Scalars["String"]>;
    name: Scalars["String"];
    pages: Array<NotePage>;
};

export type NoteVersionTranslationCreateInput = BaseTranslationCreateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name: Scalars["String"];
    pagesCreate?: InputMaybe<Array<NotePageCreateInput>>;
};

export type NoteVersionTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
    pagesCreate?: InputMaybe<Array<NotePageCreateInput>>;
    pagesDelete?: InputMaybe<Array<Scalars["ID"]>>;
    pagesUpdate?: InputMaybe<Array<NotePageUpdateInput>>;
};

export type NoteVersionUpdateInput = BaseTranslatableUpdateInput<NoteVersionTranslationCreateInput, NoteVersionTranslationUpdateInput> & {
    directoryListingsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    directoryListingsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    id: Scalars["ID"];
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    rootUpdate?: InputMaybe<NoteUpdateInput>;
    versionLabel?: InputMaybe<Scalars["String"]>;
    versionNotes?: InputMaybe<Scalars["String"]>;
};

export type NoteYou = {
    __typename: "NoteYou";
    canBookmark: Scalars["Boolean"];
    canDelete: Scalars["Boolean"];
    canReact: Scalars["Boolean"];
    canRead: Scalars["Boolean"];
    canTransfer: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
    isBookmarked: Scalars["Boolean"];
    isViewed: Scalars["Boolean"];
    reaction?: Maybe<Scalars["String"]>;
};

export type Notification = DbObject<"Notification"> & {
    category: Scalars["String"];
    created_at: Scalars["Date"];
    description?: Maybe<Scalars["String"]>;
    imgLink?: Maybe<Scalars["String"]>;
    isRead: Scalars["Boolean"];
    link?: Maybe<Scalars["String"]>;
    title: Scalars["String"];
};

export type NotificationEdge = Edge<Notification, "NotificationEdge">;

export type NotificationSearchInput = BaseSearchInput<NotificationSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    searchString?: InputMaybe<Scalars["String"]>;
    userId?: InputMaybe<Scalars["ID"]>;
    visibility?: InputMaybe<VisibilityType>;
};

export type NotificationSearchResult = SearchResult<NotificationEdge, "NotificationSearchResult">;

export type NotificationSettings = {
    __typename: "NotificationSettings";
    categories?: Maybe<Array<NotificationSettingsCategory>>;
    dailyLimit?: Maybe<Scalars["Int"]>;
    enabled: Scalars["Boolean"];
    includedEmails?: Maybe<Array<Email>>;
    includedPush?: Maybe<Array<PushDevice>>;
    includedSms?: Maybe<Array<Phone>>;
    toEmails?: Maybe<Scalars["Boolean"]>;
    toPush?: Maybe<Scalars["Boolean"]>;
    toSms?: Maybe<Scalars["Boolean"]>;
};

export type NotificationSettingsCategory = {
    __typename: "NotificationSettingsCategory";
    category: Scalars["String"];
    dailyLimit?: Maybe<Scalars["Int"]>;
    enabled: Scalars["Boolean"];
    toEmails?: Maybe<Scalars["Boolean"]>;
    toPush?: Maybe<Scalars["Boolean"]>;
    toSms?: Maybe<Scalars["Boolean"]>;
};

export type NotificationSettingsCategoryUpdateInput = {
    category: Scalars["String"];
    dailyLimit?: InputMaybe<Scalars["Int"]>;
    enabled?: InputMaybe<Scalars["Boolean"]>;
    toEmails?: InputMaybe<Scalars["Boolean"]>;
    toPush?: InputMaybe<Scalars["Boolean"]>;
    toSms?: InputMaybe<Scalars["Boolean"]>;
};

export type NotificationSettingsUpdateInput = {
    categories?: InputMaybe<Array<NotificationSettingsCategoryUpdateInput>>;
    dailyLimit?: InputMaybe<Scalars["Int"]>;
    enabled?: InputMaybe<Scalars["Boolean"]>;
    includedEmails?: InputMaybe<Array<Scalars["ID"]>>;
    includedPush?: InputMaybe<Array<Scalars["ID"]>>;
    includedSms?: InputMaybe<Array<Scalars["ID"]>>;
    toEmails?: InputMaybe<Scalars["Boolean"]>;
    toPush?: InputMaybe<Scalars["Boolean"]>;
    toSms?: InputMaybe<Scalars["Boolean"]>;
};

export enum NotificationSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc"
}

export type NotificationSubscription = {
    __typename: "NotificationSubscription";
    context?: Maybe<Scalars["String"]>;
    created_at: Scalars["Date"];
    id: Scalars["ID"];
    object: SubscribedObject;
    silent: Scalars["Boolean"];
};

export type NotificationSubscriptionCreateInput = {
    context?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    objectConnect: Scalars["ID"];
    objectType: SubscribableObject;
    silent?: InputMaybe<Scalars["Boolean"]>;
};

export type NotificationSubscriptionEdge = Edge<NotificationSubscription, "NotificationSubscriptionEdge">;

export type NotificationSubscriptionSearchInput = BaseSearchInput<NotificationSubscriptionSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    objectId?: InputMaybe<Scalars["ID"]>;
    objectType?: InputMaybe<SubscribableObject>;
    silent?: InputMaybe<Scalars["Boolean"]>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type NotificationSubscriptionSearchResult = SearchResult<NotificationSubscriptionEdge, "NotificationSubscriptionSearchResult">;

export enum NotificationSubscriptionSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc"
}

export type NotificationSubscriptionUpdateInput = {
    context?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    silent?: InputMaybe<Scalars["Boolean"]>;
};

export type Owner = Team | User;

export type PageInfo = {
    __typename: "PageInfo";
    endCursor?: Maybe<Scalars["String"]>;
    hasNextPage: Scalars["Boolean"];
};

export type Payment = DbObject<"Payment"> & {
    amount: Scalars["Int"];
    cardExpDate?: Maybe<Scalars["String"]>;
    cardLast4?: Maybe<Scalars["String"]>;
    cardType?: Maybe<Scalars["String"]>;
    checkoutId: Scalars["String"];
    created_at: Scalars["Date"];
    currency: Scalars["String"];
    description: Scalars["String"];
    paymentMethod: Scalars["String"];
    paymentType: PaymentType;
    status: PaymentStatus;
    team: Team;
    updated_at: Scalars["Date"];
    user: User;
};

export type PaymentEdge = Edge<Payment, "PaymentEdge">;

export type PaymentSearchInput = BaseSearchInput<PaymentSortBy> & {
    cardLast4?: InputMaybe<Scalars["String"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    currency?: InputMaybe<Scalars["String"]>;
    maxAmount?: InputMaybe<Scalars["Int"]>;
    minAmount?: InputMaybe<Scalars["Int"]>;
    status?: InputMaybe<PaymentStatus>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type PaymentSearchResult = SearchResult<PaymentEdge, "PaymentSearchResult">;

export enum PaymentSortBy {
    AmountAsc = "AmountAsc",
    AmountDesc = "AmountDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc"
}

export enum PaymentStatus {
    Failed = "Failed",
    Paid = "Paid",
    Pending = "Pending"
}

export enum PaymentType {
    Credits = "Credits",
    Donation = "Donation",
    PremiumMonthly = "PremiumMonthly",
    PremiumYearly = "PremiumYearly"
}

export type Phone = {
    __typename: "Phone";
    id: Scalars["ID"];
    phoneNumber: Scalars["String"];
    verified: Scalars["Boolean"];
};

export type PhoneCreateInput = {
    id: Scalars["ID"];
    phoneNumber: Scalars["String"];
};

export type Popular = Api | Code | Note | Project | Question | Routine | Standard | Team | User;

export type PopularEdge = Edge<Popular, "PopularEdge">;

export enum PopularObjectType {
    Api = "Api",
    Code = "Code",
    Note = "Note",
    Project = "Project",
    Question = "Question",
    Routine = "Routine",
    Standard = "Standard",
    Team = "Team",
    User = "User"
}

export type PopularPageInfo = {
    __typename: "PopularPageInfo";
    endCursorApi?: Maybe<Scalars["String"]>;
    endCursorCode?: Maybe<Scalars["String"]>;
    endCursorNote?: Maybe<Scalars["String"]>;
    endCursorProject?: Maybe<Scalars["String"]>;
    endCursorQuestion?: Maybe<Scalars["String"]>;
    endCursorRoutine?: Maybe<Scalars["String"]>;
    endCursorStandard?: Maybe<Scalars["String"]>;
    endCursorTeam?: Maybe<Scalars["String"]>;
    endCursorUser?: Maybe<Scalars["String"]>;
    hasNextPage: Scalars["Boolean"];
};

export type PopularSearchInput = {
    apiAfter?: InputMaybe<Scalars["String"]>;
    codeAfter?: InputMaybe<Scalars["String"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    noteAfter?: InputMaybe<Scalars["String"]>;
    objectType?: InputMaybe<PopularObjectType>;
    projectAfter?: InputMaybe<Scalars["String"]>;
    questionAfter?: InputMaybe<Scalars["String"]>;
    routineAfter?: InputMaybe<Scalars["String"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    sortBy?: InputMaybe<PopularSortBy>;
    standardAfter?: InputMaybe<Scalars["String"]>;
    take?: InputMaybe<Scalars["Int"]>;
    teamAfter?: InputMaybe<Scalars["String"]>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userAfter?: InputMaybe<Scalars["String"]>;
    visibility?: InputMaybe<VisibilityType>;
};

export type PopularSearchResult = SearchResult<PopularEdge, "PopularSearchResult", PopularPageInfo>;

export enum PopularSortBy {
    BookmarksAsc = "BookmarksAsc",
    BookmarksDesc = "BookmarksDesc",
    ViewsAsc = "ViewsAsc",
    ViewsDesc = "ViewsDesc"
}

export type Post = DbObject<"Post"> & {
    bookmarkedBy: Array<User>;
    bookmarks: Scalars["Int"];
    comments: Array<Comment>;
    commentsCount: Scalars["Int"];
    created_at: Scalars["Date"];
    owner: Owner;
    reports: Array<Report>;
    repostedFrom?: Maybe<Post>;
    reposts: Array<Post>;
    repostsCount: Scalars["Int"];
    resourceList: ResourceList;
    score: Scalars["Int"];
    tags: Array<Tag>;
    translations: Array<PostTranslation>;
    updated_at: Scalars["Date"];
    views: Scalars["Int"];
};

export type PostCreateInput = BaseTranslatableCreateInput<PostTranslationCreateInput> & {
    id: Scalars["ID"];
    isPinned?: InputMaybe<Scalars["Boolean"]>;
    isPrivate: Scalars["Boolean"];
    repostedFromConnect?: InputMaybe<Scalars["ID"]>;
    resourceListCreate?: InputMaybe<ResourceListCreateInput>;
    tagsConnect?: InputMaybe<Array<Scalars["String"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
    teamConnect?: InputMaybe<Scalars["ID"]>;
    userConnect?: InputMaybe<Scalars["ID"]>;
};

export type PostEdge = Edge<Post, "PostEdge">;

export type PostSearchInput = BaseSearchInput<PostSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    isPinned?: InputMaybe<Scalars["Boolean"]>;
    maxBookmarks?: InputMaybe<Scalars["Int"]>;
    maxScore?: InputMaybe<Scalars["Int"]>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    minScore?: InputMaybe<Scalars["Int"]>;
    repostedFromIds?: InputMaybe<Array<Scalars["ID"]>>;
    searchString?: InputMaybe<Scalars["String"]>;
    tags?: InputMaybe<Array<Scalars["String"]>>;
    teamId?: InputMaybe<Scalars["ID"]>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userId?: InputMaybe<Scalars["ID"]>;
};

export type PostSearchResult = SearchResult<PostEdge, "PostSearchResult">;

export enum PostSortBy {
    BookmarksAsc = "BookmarksAsc",
    BookmarksDesc = "BookmarksDesc",
    CommentsAsc = "CommentsAsc",
    CommentsDesc = "CommentsDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    ReportsAsc = "ReportsAsc",
    ReportsDesc = "ReportsDesc",
    RepostsAsc = "RepostsAsc",
    RepostsDesc = "RepostsDesc",
    ScoreAsc = "ScoreAsc",
    ScoreDesc = "ScoreDesc",
    ViewsAsc = "ViewsAsc",
    ViewsDesc = "ViewsDesc"
}

export type PostTranslation = BaseTranslation<"PostTranslation"> & {
    description?: Maybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type PostTranslationCreateInput = BaseTranslationCreateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type PostTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type PostUpdateInput = BaseTranslatableUpdateInput<PostTranslationCreateInput, PostTranslationUpdateInput> & {
    id: Scalars["ID"];
    isPinned?: InputMaybe<Scalars["Boolean"]>;
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
    tagsConnect?: InputMaybe<Array<Scalars["String"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
    tagsDisconnect?: InputMaybe<Array<Scalars["String"]>>;
};

export type Premium = DbObject<"Premium"> & {
    credits: Scalars["Int"];
    customPlan?: Maybe<Scalars["String"]>;
    enabledAt?: Maybe<Scalars["Date"]>;
    expiresAt?: Maybe<Scalars["Date"]>;
    isActive: Scalars["Boolean"];
};

export type ProfileEmailUpdateInput = {
    currentPassword: Scalars["String"];
    emailsCreate?: InputMaybe<Array<EmailCreateInput>>;
    emailsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    newPassword?: InputMaybe<Scalars["String"]>;
};

export type ProfileUpdateInput = BaseTranslatableUpdateInput<UserTranslationCreateInput, UserTranslationUpdateInput> & {
    bannerImage?: InputMaybe<Scalars["Upload"]>;
    focusModesCreate?: InputMaybe<Array<FocusModeCreateInput>>;
    focusModesDelete?: InputMaybe<Array<Scalars["ID"]>>;
    focusModesUpdate?: InputMaybe<Array<FocusModeUpdateInput>>;
    handle?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    isPrivateApis?: InputMaybe<Scalars["Boolean"]>;
    isPrivateApisCreated?: InputMaybe<Scalars["Boolean"]>;
    isPrivateBookmarks?: InputMaybe<Scalars["Boolean"]>;
    isPrivateCodes?: InputMaybe<Scalars["Boolean"]>;
    isPrivateCodesCreated?: InputMaybe<Scalars["Boolean"]>;
    isPrivateMemberships?: InputMaybe<Scalars["Boolean"]>;
    isPrivateProjects?: InputMaybe<Scalars["Boolean"]>;
    isPrivateProjectsCreated?: InputMaybe<Scalars["Boolean"]>;
    isPrivatePullRequests?: InputMaybe<Scalars["Boolean"]>;
    isPrivateQuestionsAnswered?: InputMaybe<Scalars["Boolean"]>;
    isPrivateQuestionsAsked?: InputMaybe<Scalars["Boolean"]>;
    isPrivateQuizzesCreated?: InputMaybe<Scalars["Boolean"]>;
    isPrivateRoles?: InputMaybe<Scalars["Boolean"]>;
    isPrivateRoutines?: InputMaybe<Scalars["Boolean"]>;
    isPrivateRoutinesCreated?: InputMaybe<Scalars["Boolean"]>;
    isPrivateStandards?: InputMaybe<Scalars["Boolean"]>;
    isPrivateStandardsCreated?: InputMaybe<Scalars["Boolean"]>;
    isPrivateTeamsCreated?: InputMaybe<Scalars["Boolean"]>;
    isPrivateVotes?: InputMaybe<Scalars["Boolean"]>;
    languages?: InputMaybe<Array<Scalars["String"]>>;
    name?: InputMaybe<Scalars["String"]>;
    notificationSettings?: InputMaybe<Scalars["String"]>;
    profileImage?: InputMaybe<Scalars["Upload"]>;
    theme?: InputMaybe<Scalars["String"]>;
};

export type Project = DbObject<"Project"> & {
    bookmarkedBy: Array<User>;
    bookmarks: Scalars["Int"];
    createdBy?: Maybe<User>;
    created_at: Scalars["Date"];
    handle?: Maybe<Scalars["String"]>;
    hasCompleteVersion: Scalars["Boolean"];
    isPrivate: Scalars["Boolean"];
    issues: Array<Issue>;
    issuesCount: Scalars["Int"];
    labels: Array<Label>;
    labelsCount: Scalars["Int"];
    owner?: Maybe<Owner>;
    parent?: Maybe<ProjectVersion>;
    permissions: Scalars["String"];
    pullRequests: Array<PullRequest>;
    pullRequestsCount: Scalars["Int"];
    questions: Array<Question>;
    questionsCount: Scalars["Int"];
    quizzes: Array<Quiz>;
    quizzesCount: Scalars["Int"];
    score: Scalars["Int"];
    stats: Array<StatsProject>;
    tags: Array<Tag>;
    transfers: Array<Transfer>;
    transfersCount: Scalars["Int"];
    translatedName: Scalars["String"];
    updated_at: Scalars["Date"];
    versions: Array<ProjectVersion>;
    versionsCount: Scalars["Int"];
    views: Scalars["Int"];
    you: ProjectYou;
};

export type ProjectCreateInput = {
    handle?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    isPrivate: Scalars["Boolean"];
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    ownedByTeamConnect?: InputMaybe<Scalars["ID"]>;
    ownedByUserConnect?: InputMaybe<Scalars["ID"]>;
    parentConnect?: InputMaybe<Scalars["ID"]>;
    permissions?: InputMaybe<Scalars["String"]>;
    tagsConnect?: InputMaybe<Array<Scalars["String"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
    versionsCreate?: InputMaybe<Array<ProjectVersionCreateInput>>;
};

export type ProjectEdge = Edge<Project, "ProjectEdge">;

export type ProjectOrRoutine = Project | Routine;

export type ProjectOrRoutineEdge = Edge<ProjectOrRoutine, "ProjectOrRoutineEdge">;

export type ProjectOrRoutinePageInfo = {
    __typename: "ProjectOrRoutinePageInfo";
    endCursorProject?: Maybe<Scalars["String"]>;
    endCursorRoutine?: Maybe<Scalars["String"]>;
    hasNextPage: Scalars["Boolean"];
};

export type ProjectOrRoutineSearchInput = MultiObjectSearchInput<ProjectOrRoutineSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    hasCompleteVersion?: InputMaybe<Scalars["Boolean"]>;
    hasCompleteVersionExceptions?: InputMaybe<Array<SearchException>>;
    maxBookmarks?: InputMaybe<Scalars["Int"]>;
    maxScore?: InputMaybe<Scalars["Int"]>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    minScore?: InputMaybe<Scalars["Int"]>;
    minViews?: InputMaybe<Scalars["Int"]>;
    objectType?: InputMaybe<Scalars["String"]>;
    parentId?: InputMaybe<Scalars["ID"]>;
    projectAfter?: InputMaybe<Scalars["String"]>;
    reportId?: InputMaybe<Scalars["ID"]>;
    resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
    routineAfter?: InputMaybe<Scalars["String"]>;
    routineIsInternal?: InputMaybe<Scalars["Boolean"]>;
    routineMaxComplexity?: InputMaybe<Scalars["Int"]>;
    routineMaxSimplicity?: InputMaybe<Scalars["Int"]>;
    routineMaxTimesCompleted?: InputMaybe<Scalars["Int"]>;
    routineMinComplexity?: InputMaybe<Scalars["Int"]>;
    routineMinSimplicity?: InputMaybe<Scalars["Int"]>;
    routineMinTimesCompleted?: InputMaybe<Scalars["Int"]>;
    routineProjectId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    tags?: InputMaybe<Array<Scalars["String"]>>;
    teamId?: InputMaybe<Scalars["ID"]>;
    translationLanguagesLatestVersion?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userId?: InputMaybe<Scalars["ID"]>;
    visibility?: InputMaybe<VisibilityType>;
};

export type ProjectOrRoutineSearchResult = SearchResult<ProjectOrRoutineEdge, "ProjectOrRoutineSearchResult", ProjectOrRoutinePageInfo>;

export enum ProjectOrRoutineSortBy {
    BookmarksAsc = "BookmarksAsc",
    BookmarksDesc = "BookmarksDesc",
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
    ScoreAsc = "ScoreAsc",
    ScoreDesc = "ScoreDesc",
    VersionsAsc = "VersionsAsc",
    VersionsDesc = "VersionsDesc",
    ViewsAsc = "ViewsAsc",
    ViewsDesc = "ViewsDesc"
}

export type ProjectOrTeam = Project | Team;

export type ProjectOrTeamEdge = Edge<ProjectOrTeam, "ProjectOrTeamEdge">;

export type ProjectOrTeamPageInfo = {
    __typename: "ProjectOrTeamPageInfo";
    endCursorProject?: Maybe<Scalars["String"]>;
    endCursorTeam?: Maybe<Scalars["String"]>;
    hasNextPage: Scalars["Boolean"];
};

export type ProjectOrTeamSearchInput = MultiObjectSearchInput<ProjectOrTeamSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    maxBookmarks?: InputMaybe<Scalars["Int"]>;
    maxViews?: InputMaybe<Scalars["Int"]>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    minViews?: InputMaybe<Scalars["Int"]>;
    objectType?: InputMaybe<Scalars["String"]>;
    projectAfter?: InputMaybe<Scalars["String"]>;
    projectIsComplete?: InputMaybe<Scalars["Boolean"]>;
    projectIsCompleteExceptions?: InputMaybe<Array<SearchException>>;
    projectMaxScore?: InputMaybe<Scalars["Int"]>;
    projectMinScore?: InputMaybe<Scalars["Int"]>;
    projectParentId?: InputMaybe<Scalars["ID"]>;
    projectTeamId?: InputMaybe<Scalars["ID"]>;
    reportId?: InputMaybe<Scalars["ID"]>;
    resourceTypes?: InputMaybe<Array<ResourceUsedFor>>;
    searchString?: InputMaybe<Scalars["String"]>;
    tags?: InputMaybe<Array<Scalars["String"]>>;
    teamAfter?: InputMaybe<Scalars["String"]>;
    teamIsOpenToNewMembers?: InputMaybe<Scalars["Boolean"]>;
    teamProjectId?: InputMaybe<Scalars["ID"]>;
    teamRoutineId?: InputMaybe<Scalars["ID"]>;
    translationLanguagesLatestVersion?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userId?: InputMaybe<Scalars["ID"]>;
    visibility?: InputMaybe<VisibilityType>;
};

export type ProjectOrTeamSearchResult = SearchResult<ProjectOrTeamEdge, "ProjectOrTeamSearchResult", ProjectOrTeamPageInfo>;

export enum ProjectOrTeamSortBy {
    BookmarksAsc = "BookmarksAsc",
    BookmarksDesc = "BookmarksDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc"
}

export type ProjectSearchInput = BaseSearchInput<ProjectSortBy> & {
    createdById?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    hasCompleteVersion?: InputMaybe<Scalars["Boolean"]>;
    issuesId?: InputMaybe<Scalars["ID"]>;
    labelsIds?: InputMaybe<Array<Scalars["ID"]>>;
    maxBookmarks?: InputMaybe<Scalars["Int"]>;
    maxScore?: InputMaybe<Scalars["Int"]>;
    maxViews?: InputMaybe<Scalars["Int"]>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    minScore?: InputMaybe<Scalars["Int"]>;
    minViews?: InputMaybe<Scalars["Int"]>;
    ownedByTeamId?: InputMaybe<Scalars["ID"]>;
    ownedByUserId?: InputMaybe<Scalars["ID"]>;
    parentId?: InputMaybe<Scalars["ID"]>;
    pullRequestsId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    tags?: InputMaybe<Array<Scalars["String"]>>;
    translationLanguagesLatestVersion?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type ProjectSearchResult = SearchResult<ProjectEdge, "ProjectSearchResult">;

export enum ProjectSortBy {
    BookmarksAsc = "BookmarksAsc",
    BookmarksDesc = "BookmarksDesc",
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
    ScoreAsc = "ScoreAsc",
    ScoreDesc = "ScoreDesc",
    VersionsAsc = "VersionsAsc",
    VersionsDesc = "VersionsDesc",
    ViewsAsc = "ViewsAsc",
    ViewsDesc = "ViewsDesc"
}

export type ProjectUpdateInput = {
    handle?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    labelsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    ownedByTeamConnect?: InputMaybe<Scalars["ID"]>;
    ownedByUserConnect?: InputMaybe<Scalars["ID"]>;
    permissions?: InputMaybe<Scalars["String"]>;
    tagsConnect?: InputMaybe<Array<Scalars["String"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
    tagsDisconnect?: InputMaybe<Array<Scalars["String"]>>;
    versionsCreate?: InputMaybe<Array<ProjectVersionCreateInput>>;
    versionsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    versionsUpdate?: InputMaybe<Array<ProjectVersionUpdateInput>>;
};

export type ProjectVersion = DbObject<"ProjectVersion"> & {
    comments: Array<Comment>;
    commentsCount: Scalars["Int"];
    completedAt?: Maybe<Scalars["Date"]>;
    complexity: Scalars["Int"];
    created_at: Scalars["Date"];
    directories: Array<ProjectVersionDirectory>;
    directoriesCount: Scalars["Int"];
    directoryListings: Array<ProjectVersionDirectory>;
    directoryListingsCount: Scalars["Int"];
    forks: Array<Project>;
    forksCount: Scalars["Int"];
    isComplete: Scalars["Boolean"];
    isLatest: Scalars["Boolean"];
    isPrivate: Scalars["Boolean"];
    pullRequest?: Maybe<PullRequest>;
    reports: Array<Report>;
    reportsCount: Scalars["Int"];
    root: Project;
    runProjectsCount: Scalars["Int"];
    simplicity: Scalars["Int"];
    suggestedNextByProject: Array<Project>;
    timesCompleted: Scalars["Int"];
    timesStarted: Scalars["Int"];
    translations: Array<ProjectVersionTranslation>;
    translationsCount: Scalars["Int"];
    updated_at: Scalars["Date"];
    versionIndex: Scalars["Int"];
    versionLabel: Scalars["String"];
    versionNotes?: Maybe<Scalars["String"]>;
    you: ProjectVersionYou;
};

export type ProjectVersionContentsSearchInput = Omit<BaseSearchInput<ProjectVersionContentsSortBy>, "ids"> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    directoryIds?: InputMaybe<Array<Scalars["ID"]>>;
    projectVersionId: Scalars["ID"];
    searchString?: InputMaybe<Scalars["String"]>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type ProjectVersionContentsSearchResult = {
    __typename: "ProjectVersionContentsSearchResult";
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
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc"
}

export type ProjectVersionCreateInput = BaseTranslatableCreateInput<ProjectVersionTranslationCreateInput> & {
    directoriesCreate?: InputMaybe<Array<ProjectVersionDirectoryCreateInput>>;
    id: Scalars["ID"];
    isComplete?: InputMaybe<Scalars["Boolean"]>;
    isPrivate: Scalars["Boolean"];
    rootConnect?: InputMaybe<Scalars["ID"]>;
    rootCreate?: InputMaybe<ProjectCreateInput>;
    suggestedNextByProjectConnect?: InputMaybe<Array<Scalars["ID"]>>;
    versionLabel: Scalars["String"];
    versionNotes?: InputMaybe<Scalars["String"]>;
};

export type ProjectVersionDirectory = DbObject<"ProjectVersionDirectory"> & {
    childApiVersions: Array<ApiVersion>;
    childCodeVersions: Array<CodeVersion>;
    childNoteVersions: Array<NoteVersion>;
    childOrder?: Maybe<Scalars["String"]>;
    childProjectVersions: Array<ProjectVersion>;
    childRoutineVersions: Array<RoutineVersion>;
    childStandardVersions: Array<StandardVersion>;
    childTeams: Array<Team>;
    children: Array<ProjectVersionDirectory>;
    created_at: Scalars["Date"];
    isRoot: Scalars["Boolean"];
    parentDirectory?: Maybe<ProjectVersionDirectory>;
    projectVersion?: Maybe<ProjectVersion>;
    runProjectSteps: Array<RunProjectStep>;
    translations: Array<ProjectVersionDirectoryTranslation>;
    updated_at: Scalars["Date"];
};

export type ProjectVersionDirectoryCreateInput = BaseTranslatableCreateInput<ProjectVersionDirectoryTranslationCreateInput> & {
    childApiVersionsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    childCodeVersionsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    childNoteVersionsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    childOrder?: InputMaybe<Scalars["String"]>;
    childProjectVersionsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    childRoutineVersionsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    childStandardVersionsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    childTeamsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    id: Scalars["ID"];
    isRoot?: InputMaybe<Scalars["Boolean"]>;
    parentDirectoryConnect?: InputMaybe<Scalars["ID"]>;
    projectVersionConnect: Scalars["ID"];
};

export type ProjectVersionDirectoryEdge = Edge<ProjectVersionDirectory, "ProjectVersionDirectoryEdge">;

export type ProjectVersionDirectorySearchInput = BaseSearchInput<ProjectVersionDirectorySortBy> & {
    isRoot?: InputMaybe<Scalars["Boolean"]>;
    parentDirectoryId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    visibility?: InputMaybe<VisibilityType>;
};

export type ProjectVersionDirectorySearchResult = SearchResult<ProjectVersionDirectoryEdge, "ProjectVersionDirectorySearchResult">;

export enum ProjectVersionDirectorySortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc"
}

export type ProjectVersionDirectoryTranslation = BaseTranslation<"ProjectVersionDirectoryTranslation"> & {
    description?: Maybe<Scalars["String"]>;
    name?: Maybe<Scalars["String"]>;
};

export type ProjectVersionDirectoryTranslationCreateInput = BaseTranslationCreateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type ProjectVersionDirectoryTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type ProjectVersionDirectoryUpdateInput = BaseTranslatableUpdateInput<ProjectVersionDirectoryTranslationCreateInput, ProjectVersionDirectoryTranslationUpdateInput> & {
    childApiVersionsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    childApiVersionsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    childCodeVersionsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    childCodeVersionsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    childNoteVersionsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    childNoteVersionsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    childOrder?: InputMaybe<Scalars["String"]>;
    childProjectVersionsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    childProjectVersionsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    childRoutineVersionsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    childRoutineVersionsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    childStandardVersionsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    childStandardVersionsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    childTeamsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    childTeamsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    id: Scalars["ID"];
    isRoot?: InputMaybe<Scalars["Boolean"]>;
    parentDirectoryConnect?: InputMaybe<Scalars["ID"]>;
    parentDirectoryDisconnect?: InputMaybe<Scalars["Boolean"]>;
    projectVersionConnect?: InputMaybe<Scalars["ID"]>;
};

export type ProjectVersionEdge = Edge<ProjectVersion, "ProjectVersionEdge">;

export type ProjectVersionSearchInput = BaseSearchInput<ProjectVersionSortBy> & {
    createdByIdRoot?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    directoryListingsId?: InputMaybe<Scalars["ID"]>;
    isCompleteWithRoot?: InputMaybe<Scalars["Boolean"]>;
    isCompleteWithRootExcludeOwnedByTeamId?: InputMaybe<Scalars["ID"]>;
    isCompleteWithRootExcludeOwnedByUserId?: InputMaybe<Scalars["ID"]>;
    isLatest?: InputMaybe<Scalars["Boolean"]>;
    maxBookmarksRoot?: InputMaybe<Scalars["Int"]>;
    maxComplexity?: InputMaybe<Scalars["Int"]>;
    maxScoreRoot?: InputMaybe<Scalars["Int"]>;
    maxSimplicity?: InputMaybe<Scalars["Int"]>;
    maxTimesCompleted?: InputMaybe<Scalars["Int"]>;
    maxViewsRoot?: InputMaybe<Scalars["Int"]>;
    minBookmarksRoot?: InputMaybe<Scalars["Int"]>;
    minComplexity?: InputMaybe<Scalars["Int"]>;
    minScoreRoot?: InputMaybe<Scalars["Int"]>;
    minSimplicity?: InputMaybe<Scalars["Int"]>;
    minTimesCompleted?: InputMaybe<Scalars["Int"]>;
    minViewsRoot?: InputMaybe<Scalars["Int"]>;
    ownedByTeamIdRoot?: InputMaybe<Scalars["ID"]>;
    ownedByUserIdRoot?: InputMaybe<Scalars["ID"]>;
    rootId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    tagsRoot?: InputMaybe<Array<Scalars["String"]>>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type ProjectVersionSearchResult = SearchResult<ProjectVersionEdge, "ProjectVersionSearchResult">;

export enum ProjectVersionSortBy {
    CommentsAsc = "CommentsAsc",
    CommentsDesc = "CommentsDesc",
    ComplexityAsc = "ComplexityAsc",
    ComplexityDesc = "ComplexityDesc",
    DateCompletedAsc = "DateCompletedAsc",
    DateCompletedDesc = "DateCompletedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DirectoryListingsAsc = "DirectoryListingsAsc",
    DirectoryListingsDesc = "DirectoryListingsDesc",
    ForksAsc = "ForksAsc",
    ForksDesc = "ForksDesc",
    RunProjectsAsc = "RunProjectsAsc",
    RunProjectsDesc = "RunProjectsDesc",
    SimplicityAsc = "SimplicityAsc",
    SimplicityDesc = "SimplicityDesc"
}

export type ProjectVersionTranslation = BaseTranslation<"ProjectVersionTranslation"> & {
    description?: Maybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type ProjectVersionTranslationCreateInput = BaseTranslationCreateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type ProjectVersionTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type ProjectVersionUpdateInput = BaseTranslatableUpdateInput<ProjectVersionTranslationCreateInput, ProjectVersionTranslationUpdateInput> & {
    directoriesCreate?: InputMaybe<Array<ProjectVersionDirectoryCreateInput>>;
    directoriesDelete?: InputMaybe<Array<Scalars["ID"]>>;
    directoriesUpdate?: InputMaybe<Array<ProjectVersionDirectoryUpdateInput>>;
    id: Scalars["ID"];
    isComplete?: InputMaybe<Scalars["Boolean"]>;
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    rootUpdate?: InputMaybe<ProjectUpdateInput>;
    suggestedNextByProjectConnect?: InputMaybe<Array<Scalars["ID"]>>;
    suggestedNextByProjectDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    versionLabel?: InputMaybe<Scalars["String"]>;
    versionNotes?: InputMaybe<Scalars["String"]>;
};

export type ProjectVersionYou = {
    __typename: "ProjectVersionYou";
    canComment: Scalars["Boolean"];
    canCopy: Scalars["Boolean"];
    canDelete: Scalars["Boolean"];
    canRead: Scalars["Boolean"];
    canReport: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
    canUse: Scalars["Boolean"];
    runs: Array<RunProject>;
};

export type ProjectYou = {
    __typename: "ProjectYou";
    canBookmark: Scalars["Boolean"];
    canDelete: Scalars["Boolean"];
    canReact: Scalars["Boolean"];
    canRead: Scalars["Boolean"];
    canTransfer: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
    isBookmarked: Scalars["Boolean"];
    isViewed: Scalars["Boolean"];
    reaction?: Maybe<Scalars["String"]>;
};

export type PullRequest = DbObject<"PullRequest"> & {
    comments: Array<Comment>;
    commentsCount: Scalars["Int"];
    createdBy?: Maybe<User>;
    created_at: Scalars["Date"];
    from: PullRequestFrom;
    mergedOrRejectedAt?: Maybe<Scalars["Date"]>;
    status: PullRequestStatus;
    to: PullRequestTo;
    translations: Array<CommentTranslation>;
    translationsCount: Scalars["Int"];
    updated_at: Scalars["Date"];
    you: PullRequestYou;
};

export type PullRequestCreateInput = BaseTranslatableCreateInput<PullRequestTranslationCreateInput> & {
    fromConnect: Scalars["ID"];
    fromObjectType: PullRequestFromObjectType;
    id: Scalars["ID"];
    toConnect: Scalars["ID"];
    toObjectType: PullRequestToObjectType;
};

export type PullRequestEdge = Edge<PullRequest, "PullRequestEdge">;

export type PullRequestFrom = ApiVersion | CodeVersion | NoteVersion | ProjectVersion | RoutineVersion | StandardVersion;

export enum PullRequestFromObjectType {
    ApiVersion = "ApiVersion",
    CodeVersion = "CodeVersion",
    NoteVersion = "NoteVersion",
    ProjectVersion = "ProjectVersion",
    RoutineVersion = "RoutineVersion",
    StandardVersion = "StandardVersion"
}

export type PullRequestSearchInput = BaseSearchInput<PullRequestSortBy> & {
    createdById?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    isMergedOrRejected?: InputMaybe<Scalars["Boolean"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    tags?: InputMaybe<Array<Scalars["String"]>>;
    toId?: InputMaybe<Scalars["ID"]>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userId?: InputMaybe<Scalars["ID"]>;
    visibility?: InputMaybe<VisibilityType>;
};

export type PullRequestSearchResult = SearchResult<PullRequestEdge, "PullRequestSearchResult">;

export enum PullRequestSortBy {
    CommentsAsc = "CommentsAsc",
    CommentsDesc = "CommentsDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc"
}

export enum PullRequestStatus {
    Canceled = "Canceled",
    Draft = "Draft",
    Merged = "Merged",
    Open = "Open",
    Rejected = "Rejected"
}

export type PullRequestTo = Api | Code | Note | Project | Routine | Standard;

export enum PullRequestToObjectType {
    Api = "Api",
    Code = "Code",
    Note = "Note",
    Project = "Project",
    Routine = "Routine",
    Standard = "Standard"
}

export type PullRequestTranslation = BaseTranslation<"PullRequestTranslation"> & {
    text: Scalars["String"];
};

export type PullRequestTranslationCreateInput = BaseTranslationCreateInput & {
    text: Scalars["String"];
};

export type PullRequestTranslationUpdateInput = BaseTranslationUpdateInput & {
    text?: InputMaybe<Scalars["String"]>;
};

export type PullRequestUpdateInput = BaseTranslatableUpdateInput<PullRequestTranslationCreateInput, PullRequestTranslationUpdateInput> & {
    id: Scalars["ID"];
    status?: InputMaybe<PullRequestStatus>;
};

export type PullRequestYou = {
    __typename: "PullRequestYou";
    canComment: Scalars["Boolean"];
    canDelete: Scalars["Boolean"];
    canReport: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
};

export type PushDevice = DbObject<"PushDevice"> & {
    deviceId: Scalars["String"];
    expires?: Maybe<Scalars["Date"]>;
    name?: Maybe<Scalars["String"]>;
};

export type PushDeviceCreateInput = {
    endpoint: Scalars["String"];
    expires?: InputMaybe<Scalars["Date"]>;
    keys: PushDeviceKeysInput;
    name?: InputMaybe<Scalars["String"]>;
};

export type PushDeviceKeysInput = {
    auth: Scalars["String"];
    p256dh: Scalars["String"];
};

export type PushDeviceTestInput = {
    id: Scalars["ID"];
};

export type PushDeviceUpdateInput = {
    id: Scalars["ID"];
    name?: InputMaybe<Scalars["String"]>;
};

export type Question = DbObject<"Question"> & {
    answers: Array<QuestionAnswer>;
    answersCount: Scalars["Int"];
    bookmarkedBy: Array<User>;
    bookmarks: Scalars["Int"];
    comments: Array<Comment>;
    commentsCount: Scalars["Int"];
    createdBy?: Maybe<User>;
    created_at: Scalars["Date"];
    forObject?: Maybe<QuestionFor>;
    hasAcceptedAnswer: Scalars["Boolean"];
    isPrivate: Scalars["Boolean"];
    reports: Array<Report>;
    reportsCount: Scalars["Int"];
    score: Scalars["Int"];
    tags: Array<Tag>;
    translations: Array<QuestionTranslation>;
    translationsCount: Scalars["Int"];
    updated_at: Scalars["Date"];
    you: QuestionYou;
};

export type QuestionAnswer = DbObject<"QuestionAnswer"> & {
    bookmarkedBy: Array<User>;
    bookmarks: Scalars["Int"];
    comments: Array<Comment>;
    commentsCount: Scalars["Int"];
    createdBy?: Maybe<User>;
    created_at: Scalars["Date"];
    isAccepted: Scalars["Boolean"];
    question: Question;
    score: Scalars["Int"];
    translations: Array<QuestionAnswerTranslation>;
    updated_at: Scalars["Date"];
};

export type QuestionAnswerCreateInput = BaseTranslatableCreateInput<QuestionAnswerTranslationCreateInput> & {
    id: Scalars["ID"];
    questionConnect: Scalars["ID"];
};

export type QuestionAnswerEdge = Edge<QuestionAnswer, "QuestionAnswerEdge">;

export type QuestionAnswerSearchInput = BaseSearchInput<QuestionAnswerSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    minScore?: InputMaybe<Scalars["Int"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type QuestionAnswerSearchResult = SearchResult<QuestionAnswerEdge, "QuestionAnswerSearchResult">;

export enum QuestionAnswerSortBy {
    BookmarksAsc = "BookmarksAsc",
    BookmarksDesc = "BookmarksDesc",
    CommentsAsc = "CommentsAsc",
    CommentsDesc = "CommentsDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    ScoreAsc = "ScoreAsc",
    ScoreDesc = "ScoreDesc"
}

export type QuestionAnswerTranslation = BaseTranslation<"QuestionAnswerTranslation"> & {
    text: Scalars["String"];
};

export type QuestionAnswerTranslationCreateInput = BaseTranslationCreateInput & {
    text: Scalars["String"];
};

export type QuestionAnswerTranslationUpdateInput = BaseTranslationUpdateInput & {
    text?: InputMaybe<Scalars["String"]>;
};

export type QuestionAnswerUpdateInput = BaseTranslatableUpdateInput<QuestionAnswerTranslationCreateInput, QuestionAnswerTranslationUpdateInput> & {
    id: Scalars["ID"];
};

export type QuestionCreateInput = BaseTranslatableCreateInput<QuestionTranslationCreateInput> & {
    forObjectConnect?: InputMaybe<Scalars["ID"]>;
    forObjectType?: InputMaybe<QuestionForType>;
    id: Scalars["ID"];
    isPrivate: Scalars["Boolean"];
    referencing?: InputMaybe<Scalars["String"]>;
    tagsConnect?: InputMaybe<Array<Scalars["String"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
};

export type QuestionEdge = Edge<Question, "QuestionEdge">;

export type QuestionFor = Api | Code | Note | Project | Routine | Standard | Team;

export enum QuestionForType {
    Api = "Api",
    Code = "Code",
    Note = "Note",
    Project = "Project",
    Routine = "Routine",
    Standard = "Standard",
    Team = "Team"
}

export type QuestionSearchInput = BaseSearchInput<QuestionSortBy> & {
    apiId?: InputMaybe<Scalars["ID"]>;
    codeId?: InputMaybe<Scalars["ID"]>;
    createdById?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    hasAcceptedAnswer?: InputMaybe<Scalars["Boolean"]>;
    maxBookmarks?: InputMaybe<Scalars["Int"]>;
    maxScore?: InputMaybe<Scalars["Int"]>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    minScore?: InputMaybe<Scalars["Int"]>;
    noteId?: InputMaybe<Scalars["ID"]>;
    projectId?: InputMaybe<Scalars["ID"]>;
    routineId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    standardId?: InputMaybe<Scalars["ID"]>;
    teamId?: InputMaybe<Scalars["ID"]>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type QuestionSearchResult = SearchResult<QuestionEdge, "QuestionSearchResult">;

export enum QuestionSortBy {
    AnswersAsc = "AnswersAsc",
    AnswersDesc = "AnswersDesc",
    BookmarksAsc = "BookmarksAsc",
    BookmarksDesc = "BookmarksDesc",
    CommentsAsc = "CommentsAsc",
    CommentsDesc = "CommentsDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    ScoreAsc = "ScoreAsc",
    ScoreDesc = "ScoreDesc"
}

export type QuestionTranslation = BaseTranslation<"QuestionTranslation"> & {
    description?: Maybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type QuestionTranslationCreateInput = BaseTranslationCreateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type QuestionTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type QuestionUpdateInput = BaseTranslatableUpdateInput<QuestionTranslationCreateInput, QuestionTranslationUpdateInput> & {
    acceptedAnswerConnect?: InputMaybe<Scalars["ID"]>;
    acceptedAnswerDisconnect?: InputMaybe<Scalars["Boolean"]>;
    id: Scalars["ID"];
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    tagsConnect?: InputMaybe<Array<Scalars["String"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
    tagsDisconnect?: InputMaybe<Array<Scalars["String"]>>;
};

export type QuestionYou = {
    __typename: "QuestionYou";
    canBookmark: Scalars["Boolean"];
    canDelete: Scalars["Boolean"];
    canReact: Scalars["Boolean"];
    canRead: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
    isBookmarked: Scalars["Boolean"];
    reaction?: Maybe<Scalars["String"]>;
};

export type Quiz = DbObject<"Quiz"> & {
    attempts: Array<QuizAttempt>;
    attemptsCount: Scalars["Int"];
    bookmarkedBy: Array<User>;
    bookmarks: Scalars["Int"];
    createdBy?: Maybe<User>;
    created_at: Scalars["Date"];
    isPrivate: Scalars["Boolean"];
    project?: Maybe<Project>;
    quizQuestions: Array<QuizQuestion>;
    quizQuestionsCount: Scalars["Int"];
    randomizeQuestionOrder: Scalars["Boolean"];
    routine?: Maybe<Routine>;
    score: Scalars["Int"];
    stats: Array<StatsQuiz>;
    translations: Array<QuizTranslation>;
    updated_at: Scalars["Date"];
    you: QuizYou;
};

export type QuizAttempt = DbObject<"QuizAttempt"> & {
    contextSwitches: Scalars["Int"];
    created_at: Scalars["Date"];
    pointsEarned: Scalars["Int"];
    quiz: Quiz;
    responses: Array<QuizQuestionResponse>;
    responsesCount: Scalars["Int"];
    status: QuizAttemptStatus;
    timeTaken?: Maybe<Scalars["Int"]>;
    updated_at: Scalars["Date"];
    user: User;
    you: QuizAttemptYou;
};

export type QuizAttemptCreateInput = {
    contextSwitches?: InputMaybe<Scalars["Int"]>;
    id: Scalars["ID"];
    language: Scalars["String"];
    quizConnect: Scalars["ID"];
    responsesCreate?: InputMaybe<Array<QuizQuestionResponseCreateInput>>;
    timeTaken?: InputMaybe<Scalars["Int"]>;
};

export type QuizAttemptEdge = Edge<QuizAttempt, "QuizAttemptEdge">;

export type QuizAttemptSearchInput = BaseSearchInput<QuizAttemptSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    languageIn?: InputMaybe<Array<Scalars["String"]>>;
    maxPointsEarned?: InputMaybe<Scalars["Int"]>;
    minPointsEarned?: InputMaybe<Scalars["Int"]>;
    quizId?: InputMaybe<Scalars["ID"]>;
    status?: InputMaybe<QuizAttemptStatus>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userId?: InputMaybe<Scalars["ID"]>;
    visibility?: InputMaybe<VisibilityType>;
};

export type QuizAttemptSearchResult = SearchResult<QuizAttemptEdge, "QuizAttemptSearchResult">;

export enum QuizAttemptSortBy {
    ContextSwitchesAsc = "ContextSwitchesAsc",
    ContextSwitchesDesc = "ContextSwitchesDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    PointsEarnedAsc = "PointsEarnedAsc",
    PointsEarnedDesc = "PointsEarnedDesc",
    TimeTakenAsc = "TimeTakenAsc",
    TimeTakenDesc = "TimeTakenDesc"
}

export enum QuizAttemptStatus {
    Failed = "Failed",
    InProgress = "InProgress",
    NotStarted = "NotStarted",
    Passed = "Passed"
}

export type QuizAttemptUpdateInput = {
    contextSwitches?: InputMaybe<Scalars["Int"]>;
    id: Scalars["ID"];
    responsesCreate?: InputMaybe<Array<QuizQuestionResponseCreateInput>>;
    responsesDelete?: InputMaybe<Array<Scalars["ID"]>>;
    responsesUpdate?: InputMaybe<Array<QuizQuestionResponseUpdateInput>>;
    timeTaken?: InputMaybe<Scalars["Int"]>;
};

export type QuizAttemptYou = {
    __typename: "QuizAttemptYou";
    canDelete: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
};

export type QuizCreateInput = BaseTranslatableCreateInput<QuizTranslationCreateInput> & {
    id: Scalars["ID"];
    isPrivate: Scalars["Boolean"];
    maxAttempts?: InputMaybe<Scalars["Int"]>;
    pointsToPass?: InputMaybe<Scalars["Int"]>;
    projectConnect?: InputMaybe<Scalars["ID"]>;
    quizQuestionsCreate?: InputMaybe<Array<QuizQuestionCreateInput>>;
    randomizeQuestionOrder?: InputMaybe<Scalars["Boolean"]>;
    revealCorrectAnswers?: InputMaybe<Scalars["Boolean"]>;
    routineConnect?: InputMaybe<Scalars["ID"]>;
    timeLimit?: InputMaybe<Scalars["Int"]>;
};

export type QuizEdge = Edge<Quiz, "QuizEdge">;

export type QuizQuestion = DbObject<"QuizQuestion"> & {
    created_at: Scalars["Date"];
    order?: Maybe<Scalars["Int"]>;
    points: Scalars["Int"];
    quiz: Quiz;
    responses?: Maybe<Array<QuizQuestionResponse>>;
    responsesCount: Scalars["Int"];
    standardVersion?: Maybe<StandardVersion>;
    translations?: Maybe<Array<QuizQuestionTranslation>>;
    updated_at: Scalars["Date"];
    you: QuizQuestionYou;
};

export type QuizQuestionCreateInput = BaseTranslatableCreateInput<QuizQuestionTranslationCreateInput> & {
    id: Scalars["ID"];
    order?: InputMaybe<Scalars["Int"]>;
    points?: InputMaybe<Scalars["Int"]>;
    quizConnect: Scalars["ID"];
    standardVersionConnect?: InputMaybe<Scalars["ID"]>;
    standardVersionCreate?: InputMaybe<StandardVersionCreateInput>;
};

export type QuizQuestionResponse = DbObject<"QuizQuestionResponse"> & {
    created_at: Scalars["Date"];
    quizAttempt: QuizAttempt;
    quizQuestion: QuizQuestion;
    response?: Maybe<Scalars["String"]>;
    updated_at: Scalars["Date"];
    you: QuizQuestionResponseYou;
};

export type QuizQuestionResponseCreateInput = {
    id: Scalars["ID"];
    quizAttemptConnect: Scalars["ID"];
    quizQuestionConnect: Scalars["ID"];
    response: Scalars["String"];
};

export type QuizQuestionResponseEdge = Edge<QuizQuestionResponse, "QuizQuestionResponseEdge">;

export type QuizQuestionResponseSearchInput = BaseSearchInput<QuizQuestionResponseSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    quizAttemptId?: InputMaybe<Scalars["ID"]>;
    quizQuestionId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type QuizQuestionResponseSearchResult = SearchResult<QuizQuestionResponseEdge, "QuizQuestionResponseSearchResult">;

export enum QuizQuestionResponseSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    QuestionOrderAsc = "QuestionOrderAsc",
    QuestionOrderDesc = "QuestionOrderDesc"
}

export type QuizQuestionResponseUpdateInput = {
    id: Scalars["ID"];
    response?: InputMaybe<Scalars["String"]>;
};

export type QuizQuestionResponseYou = {
    __typename: "QuizQuestionResponseYou";
    canDelete: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
};

export type QuizQuestionTranslation = BaseTranslation<"QuizQuestionTranslation"> & {
    helpText?: Maybe<Scalars["String"]>;
    questionText: Scalars["String"];
};

export type QuizQuestionTranslationCreateInput = BaseTranslationCreateInput & {
    helpText?: InputMaybe<Scalars["String"]>;
    questionText: Scalars["String"];
};

export type QuizQuestionTranslationUpdateInput = BaseTranslationUpdateInput & {
    helpText?: InputMaybe<Scalars["String"]>;
    questionText?: InputMaybe<Scalars["String"]>;
};

export type QuizQuestionUpdateInput = BaseTranslatableUpdateInput<QuizQuestionTranslationCreateInput, QuizQuestionTranslationUpdateInput> & {
    id: Scalars["ID"];
    order?: InputMaybe<Scalars["Int"]>;
    points?: InputMaybe<Scalars["Int"]>;
    standardVersionConnect?: InputMaybe<Scalars["ID"]>;
    standardVersionCreate?: InputMaybe<StandardVersionCreateInput>;
    standardVersionUpdate?: InputMaybe<StandardVersionUpdateInput>;
};

export type QuizQuestionYou = {
    __typename: "QuizQuestionYou";
    canDelete: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
};

export type QuizSearchInput = BaseSearchInput<QuizSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    isComplete?: InputMaybe<Scalars["Boolean"]>;
    maxBookmarks?: InputMaybe<Scalars["Int"]>;
    maxScore?: InputMaybe<Scalars["Int"]>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    minScore?: InputMaybe<Scalars["Int"]>;
    projectId?: InputMaybe<Scalars["ID"]>;
    routineId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userId?: InputMaybe<Scalars["ID"]>;
    visibility?: InputMaybe<VisibilityType>;
};

export type QuizSearchResult = SearchResult<QuizEdge, "QuizSearchResult">;

export enum QuizSortBy {
    AttemptsAsc = "AttemptsAsc",
    AttemptsDesc = "AttemptsDesc",
    BookmarksAsc = "BookmarksAsc",
    BookmarksDesc = "BookmarksDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    QuestionsAsc = "QuestionsAsc",
    QuestionsDesc = "QuestionsDesc",
    ScoreAsc = "ScoreAsc",
    ScoreDesc = "ScoreDesc"
}

export type QuizTranslation = BaseTranslation<"QuizTranslation"> & {
    description?: Maybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type QuizTranslationCreateInput = BaseTranslationCreateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type QuizTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type QuizUpdateInput = BaseTranslatableUpdateInput<QuizTranslationCreateInput, QuizTranslationUpdateInput> & {
    id: Scalars["ID"];
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    maxAttempts?: InputMaybe<Scalars["Int"]>;
    pointsToPass?: InputMaybe<Scalars["Int"]>;
    projectConnect?: InputMaybe<Scalars["ID"]>;
    projectDisconnect?: InputMaybe<Scalars["Boolean"]>;
    quizQuestionsCreate?: InputMaybe<Array<QuizQuestionCreateInput>>;
    quizQuestionsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    quizQuestionsUpdate?: InputMaybe<Array<QuizQuestionUpdateInput>>;
    randomizeQuestionOrder?: InputMaybe<Scalars["Boolean"]>;
    revealCorrectAnswers?: InputMaybe<Scalars["Boolean"]>;
    routineConnect?: InputMaybe<Scalars["ID"]>;
    routineDisconnect?: InputMaybe<Scalars["Boolean"]>;
    timeLimit?: InputMaybe<Scalars["Int"]>;
};

export type QuizYou = {
    __typename: "QuizYou";
    canBookmark: Scalars["Boolean"];
    canDelete: Scalars["Boolean"];
    canReact: Scalars["Boolean"];
    canRead: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
    hasCompleted: Scalars["Boolean"];
    isBookmarked: Scalars["Boolean"];
    reaction?: Maybe<Scalars["String"]>;
};

export type ReactInput = {
    emoji?: InputMaybe<Scalars["String"]>;
    forConnect: Scalars["ID"];
    reactionFor: ReactionFor;
};

export type Reaction = DbObject<"Reaction"> & {
    by: User;
    emoji: Scalars["String"];
    to: ReactionTo;
};

export type ReactionEdge = Edge<Reaction, "ReactionEdge">;

export enum ReactionFor {
    Api = "Api",
    ChatMessage = "ChatMessage",
    Code = "Code",
    Comment = "Comment",
    Issue = "Issue",
    Note = "Note",
    Post = "Post",
    Project = "Project",
    Question = "Question",
    QuestionAnswer = "QuestionAnswer",
    Quiz = "Quiz",
    Routine = "Routine",
    Standard = "Standard"
}

export type ReactionSearchInput = BaseSearchInput<ReactionSortBy> & {
    apiId?: InputMaybe<Scalars["ID"]>;
    chatMessageId?: InputMaybe<Scalars["ID"]>;
    codeId?: InputMaybe<Scalars["ID"]>;
    commentId?: InputMaybe<Scalars["ID"]>;
    excludeLinkedToTag?: InputMaybe<Scalars["Boolean"]>;
    issueId?: InputMaybe<Scalars["ID"]>;
    noteId?: InputMaybe<Scalars["ID"]>;
    postId?: InputMaybe<Scalars["ID"]>;
    projectId?: InputMaybe<Scalars["ID"]>;
    questionAnswerId?: InputMaybe<Scalars["ID"]>;
    questionId?: InputMaybe<Scalars["ID"]>;
    quizId?: InputMaybe<Scalars["ID"]>;
    routineId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    standardId?: InputMaybe<Scalars["ID"]>;
};

export type ReactionSearchResult = SearchResult<ReactionEdge, "ReactionSearchResult">;

export enum ReactionSortBy {
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc"
}

export type ReactionSummary = {
    __typename: "ReactionSummary";
    count: Scalars["Int"];
    emoji: Scalars["String"];
};

export type ReactionTo = Api | ChatMessage | Code | Comment | Issue | Note | Post | Project | Question | QuestionAnswer | Quiz | Routine | Standard;

export type ReadAssetsInput = {
    files: Array<Scalars["String"]>;
};

export type RegenerateResponseInput = {
    messageId: Scalars["ID"];
    model: string;
    task: LlmTask;
    taskContexts: Array<TaskContextInfoInput>;
};

export type Reminder = DbObject<"Reminder"> & {
    created_at: Scalars["Date"];
    description?: Maybe<Scalars["String"]>;
    dueDate?: Maybe<Scalars["Date"]>;
    index: Scalars["Int"];
    isComplete: Scalars["Boolean"];
    name: Scalars["String"];
    reminderItems: Array<ReminderItem>;
    reminderList: ReminderList;
    updated_at: Scalars["Date"];
};

export type ReminderCreateInput = {
    description?: InputMaybe<Scalars["String"]>;
    dueDate?: InputMaybe<Scalars["Date"]>;
    id: Scalars["ID"];
    index: Scalars["Int"];
    name: Scalars["String"];
    reminderItemsCreate?: InputMaybe<Array<ReminderItemCreateInput>>;
    reminderListConnect?: InputMaybe<Scalars["ID"]>;
    reminderListCreate?: InputMaybe<ReminderListCreateInput>;
};

export type ReminderEdge = Edge<Reminder, "ReminderEdge">;

export type ReminderItem = DbObject<"ReminderItem"> & {
    created_at: Scalars["Date"];
    description?: Maybe<Scalars["String"]>;
    dueDate?: Maybe<Scalars["Date"]>;
    index: Scalars["Int"];
    isComplete: Scalars["Boolean"];
    name: Scalars["String"];
    reminder: Reminder;
    updated_at: Scalars["Date"];
};

export type ReminderItemCreateInput = {
    description?: InputMaybe<Scalars["String"]>;
    dueDate?: InputMaybe<Scalars["Date"]>;
    id: Scalars["ID"];
    index: Scalars["Int"];
    isComplete?: InputMaybe<Scalars["Boolean"]>;
    name: Scalars["String"];
    reminderConnect: Scalars["ID"];
};

export type ReminderItemUpdateInput = {
    description?: InputMaybe<Scalars["String"]>;
    dueDate?: InputMaybe<Scalars["Date"]>;
    id: Scalars["ID"];
    index?: InputMaybe<Scalars["Int"]>;
    isComplete?: InputMaybe<Scalars["Boolean"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type ReminderList = DbObject<"ReminderList"> & {
    created_at: Scalars["Date"];
    focusMode?: Maybe<FocusMode>;
    reminders: Array<Reminder>;
    updated_at: Scalars["Date"];
};

export type ReminderListCreateInput = {
    focusModeConnect?: InputMaybe<Scalars["ID"]>;
    id: Scalars["ID"];
    remindersCreate?: InputMaybe<Array<ReminderCreateInput>>;
};

export type ReminderListUpdateInput = {
    focusModeConnect?: InputMaybe<Scalars["ID"]>;
    id: Scalars["ID"];
    remindersCreate?: InputMaybe<Array<ReminderCreateInput>>;
    remindersDelete?: InputMaybe<Array<Scalars["ID"]>>;
    remindersUpdate?: InputMaybe<Array<ReminderUpdateInput>>;
};

export type ReminderSearchInput = BaseSearchInput<ReminderSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    reminderListId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type ReminderSearchResult = SearchResult<ReminderEdge, "ReminderSearchResult">;

export enum ReminderSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DueDateAsc = "DueDateAsc",
    DueDateDesc = "DueDateDesc",
    NameAsc = "NameAsc",
    NameDesc = "NameDesc"
}

export type ReminderUpdateInput = {
    description?: InputMaybe<Scalars["String"]>;
    dueDate?: InputMaybe<Scalars["Date"]>;
    id: Scalars["ID"];
    index?: InputMaybe<Scalars["Int"]>;
    isComplete?: InputMaybe<Scalars["Boolean"]>;
    name?: InputMaybe<Scalars["String"]>;
    reminderItemsCreate?: InputMaybe<Array<ReminderItemCreateInput>>;
    reminderItemsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    reminderItemsUpdate?: InputMaybe<Array<ReminderItemUpdateInput>>;
    reminderListConnect?: InputMaybe<Scalars["ID"]>;
    reminderListCreate?: InputMaybe<ReminderListCreateInput>;
};

export type Report = DbObject<"Report"> & {
    createdFor: ReportFor;
    created_at: Scalars["Date"];
    details?: Maybe<Scalars["String"]>;
    language: Scalars["String"];
    reason: Scalars["String"];
    responses: Array<ReportResponse>;
    responsesCount: Scalars["Int"];
    status: ReportStatus;
    updated_at: Scalars["Date"];
    you: ReportYou;
};

export type ReportCreateInput = {
    createdForConnect: Scalars["ID"];
    createdForType: ReportFor;
    details?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    language: Scalars["String"];
    reason: Scalars["String"];
};

export type ReportEdge = Edge<Report, "ReportEdge">;

export enum ReportFor {
    ApiVersion = "ApiVersion",
    ChatMessage = "ChatMessage",
    CodeVersion = "CodeVersion",
    Comment = "Comment",
    Issue = "Issue",
    NoteVersion = "NoteVersion",
    Post = "Post",
    ProjectVersion = "ProjectVersion",
    RoutineVersion = "RoutineVersion",
    StandardVersion = "StandardVersion",
    Tag = "Tag",
    Team = "Team",
    User = "User"
}

export type ReportResponse = DbObject<"ReportResponse"> & {
    actionSuggested: ReportSuggestedAction;
    created_at: Scalars["Date"];
    details?: Maybe<Scalars["String"]>;
    language?: Maybe<Scalars["String"]>;
    report: Report;
    updated_at: Scalars["Date"];
    you: ReportResponseYou;
};

export type ReportResponseCreateInput = {
    actionSuggested: ReportSuggestedAction;
    details?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    language?: InputMaybe<Scalars["String"]>;
    reportConnect: Scalars["ID"];
};

export type ReportResponseEdge = Edge<ReportResponse, "ReportResponseEdge">;

export type ReportResponseSearchInput = BaseSearchInput<ReportResponseSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    languageIn?: InputMaybe<Array<Scalars["String"]>>;
    reportId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userId?: InputMaybe<Scalars["ID"]>;
    visibility?: InputMaybe<VisibilityType>;
};

export type ReportResponseSearchResult = SearchResult<ReportResponseEdge, "ReportResponseSearchResult">;

export enum ReportResponseSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc"
}

export type ReportResponseUpdateInput = {
    actionSuggested?: InputMaybe<ReportSuggestedAction>;
    details?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    language?: InputMaybe<Scalars["String"]>;
};

export type ReportResponseYou = {
    __typename: "ReportResponseYou";
    canDelete: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
};

export type ReportSearchInput = BaseSearchInput<ReportSortBy> & {
    apiVersionId?: InputMaybe<Scalars["ID"]>;
    chatMessageId?: InputMaybe<Scalars["ID"]>;
    codeVersionId?: InputMaybe<Scalars["ID"]>;
    commentId?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    fromId?: InputMaybe<Scalars["ID"]>;
    issueId?: InputMaybe<Scalars["ID"]>;
    languageIn?: InputMaybe<Array<Scalars["String"]>>;
    noteVersionId?: InputMaybe<Scalars["ID"]>;
    postId?: InputMaybe<Scalars["ID"]>;
    projectVersionId?: InputMaybe<Scalars["ID"]>;
    routineVersionId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    standardVersionId?: InputMaybe<Scalars["ID"]>;
    tagId?: InputMaybe<Scalars["ID"]>;
    teamId?: InputMaybe<Scalars["ID"]>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userId?: InputMaybe<Scalars["ID"]>;
    visibility?: InputMaybe<VisibilityType>;
};

export type ReportSearchResult = SearchResult<ReportEdge, "ReportSearchResult">;

export enum ReportSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    ResponsesAsc = "ResponsesAsc",
    ResponsesDesc = "ResponsesDesc"
}

export enum ReportStatus {
    ClosedDeleted = "ClosedDeleted",
    ClosedFalseReport = "ClosedFalseReport",
    ClosedHidden = "ClosedHidden",
    ClosedNonIssue = "ClosedNonIssue",
    ClosedSuspended = "ClosedSuspended",
    Open = "Open"
}

export enum ReportSuggestedAction {
    Delete = "Delete",
    FalseReport = "FalseReport",
    HideUntilFixed = "HideUntilFixed",
    NonIssue = "NonIssue",
    SuspendUser = "SuspendUser"
}

export type ReportUpdateInput = {
    details?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    language?: InputMaybe<Scalars["String"]>;
    reason?: InputMaybe<Scalars["String"]>;
};

export type ReportYou = {
    __typename: "ReportYou";
    canDelete: Scalars["Boolean"];
    canRespond: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
    isOwn: Scalars["Boolean"];
};

export type ReputationHistory = DbObject<"ReputationHistory"> & {
    amount: Scalars["Int"];
    created_at: Scalars["Date"];
    event: Scalars["String"];
    objectId1?: Maybe<Scalars["ID"]>;
    objectId2?: Maybe<Scalars["ID"]>;
    updated_at: Scalars["Date"];
};

export type ReputationHistoryEdge = Edge<ReputationHistory, "ReputationHistoryEdge">;

export type ReputationHistorySearchInput = BaseSearchInput<ReputationHistorySortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    searchString?: InputMaybe<Scalars["String"]>;
    tags?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type ReputationHistorySearchResult = SearchResult<ReputationHistoryEdge, "ReputationHistorySearchResult">;

export enum ReputationHistorySortBy {
    AmountAsc = "AmountAsc",
    AmountDesc = "AmountDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc"
}

export type Resource = DbObject<"Resource"> & {
    created_at: Scalars["Date"];
    index?: Maybe<Scalars["Int"]>;
    link: Scalars["String"];
    list: ResourceList;
    translations: Array<ResourceTranslation>;
    updated_at: Scalars["Date"];
    usedFor: ResourceUsedFor;
};

export type ResourceCreateInput = BaseTranslatableCreateInput<ResourceTranslationCreateInput> & {
    id: Scalars["ID"];
    index?: InputMaybe<Scalars["Int"]>;
    link: Scalars["String"];
    listConnect?: InputMaybe<Scalars["ID"]>;
    listCreate?: InputMaybe<ResourceListCreateInput>;
    usedFor: ResourceUsedFor;
};

export type ResourceEdge = Edge<Resource, "ResourceEdge">;

export type ResourceList = DbObject<"ResourceList"> & {
    created_at: Scalars["Date"];
    listFor: ResourceListOn;
    resources: Array<Resource>;
    translations: Array<ResourceListTranslation>;
    updated_at: Scalars["Date"];
};

export type ResourceListCreateInput = BaseTranslatableCreateInput<ResourceListTranslationCreateInput> & {
    id: Scalars["ID"];
    listForConnect: Scalars["ID"];
    listForType: ResourceListFor;
    resourcesCreate?: InputMaybe<Array<ResourceCreateInput>>;
};

export type ResourceListEdge = Edge<ResourceList, "ResourceListEdge">;

export enum ResourceListFor {
    ApiVersion = "ApiVersion",
    CodeVersion = "CodeVersion",
    FocusMode = "FocusMode",
    Post = "Post",
    ProjectVersion = "ProjectVersion",
    RoutineVersion = "RoutineVersion",
    StandardVersion = "StandardVersion",
    Team = "Team"
}

export type ResourceListOn = ApiVersion | CodeVersion | FocusMode | Post | ProjectVersion | RoutineVersion | StandardVersion | Team;

export type ResourceListSearchInput = BaseSearchInput<ResourceListSortBy> & {
    apiVersionId?: InputMaybe<Scalars["ID"]>;
    codeVersionId?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    focusModeId?: InputMaybe<Scalars["ID"]>;
    postId?: InputMaybe<Scalars["ID"]>;
    projectVersionId?: InputMaybe<Scalars["ID"]>;
    routineVersionId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    standardVersionId?: InputMaybe<Scalars["ID"]>;
    teamId?: InputMaybe<Scalars["ID"]>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type ResourceListSearchResult = SearchResult<ResourceListEdge, "ResourceListSearchResult">;

export enum ResourceListSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    IndexAsc = "IndexAsc",
    IndexDesc = "IndexDesc"
}

export type ResourceListTranslation = BaseTranslation<"ResourceListTranslation"> & {
    description?: Maybe<Scalars["String"]>;
    name?: Maybe<Scalars["String"]>;
};

export type ResourceListTranslationCreateInput = BaseTranslationCreateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type ResourceListTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type ResourceListUpdateInput = BaseTranslatableUpdateInput<ResourceListTranslationCreateInput, ResourceListTranslationUpdateInput> & {
    id: Scalars["ID"];
    resourcesCreate?: InputMaybe<Array<ResourceCreateInput>>;
    resourcesDelete?: InputMaybe<Array<Scalars["ID"]>>;
    resourcesUpdate?: InputMaybe<Array<ResourceUpdateInput>>;
};

export type ResourceSearchInput = BaseSearchInput<ResourceListSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    resourceListId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type ResourceSearchResult = SearchResult<ResourceEdge, "ResourceSearchResult">;

export enum ResourceSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    IndexAsc = "IndexAsc",
    IndexDesc = "IndexDesc",
    UsedForAsc = "UsedForAsc",
    UsedForDesc = "UsedForDesc"
}

export type ResourceTranslation = BaseTranslation<"ResourceTranslation"> & {
    description?: Maybe<Scalars["String"]>;
    name?: Maybe<Scalars["String"]>;
};

export type ResourceTranslationCreateInput = BaseTranslationCreateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type ResourceTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type ResourceUpdateInput = BaseTranslatableUpdateInput<ResourceTranslationCreateInput, ResourceTranslationUpdateInput> & {
    id: Scalars["ID"];
    index?: InputMaybe<Scalars["Int"]>;
    link?: InputMaybe<Scalars["String"]>;
    listConnect?: InputMaybe<Scalars["ID"]>;
    listCreate?: InputMaybe<ResourceListCreateInput>;
    usedFor?: InputMaybe<ResourceUsedFor>;
};

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
    Tutorial = "Tutorial"
}

export type Response = {
    __typename: "Response";
    code?: Maybe<Scalars["Int"]>;
    message: Scalars["String"];
};

export type Role = DbObject<"Role"> & {
    created_at: Scalars["Date"];
    members?: Maybe<Array<Member>>;
    membersCount: Scalars["Int"];
    name: Scalars["String"];
    permissions: Scalars["String"];
    team: Team;
    translations: Array<RoleTranslation>;
    updated_at: Scalars["Date"];
};

export type RoleCreateInput = BaseTranslatableCreateInput<RoleTranslationCreateInput> & {
    id: Scalars["ID"];
    membersConnect?: InputMaybe<Array<Scalars["ID"]>>;
    name: Scalars["String"];
    permissions: Scalars["String"];
    teamConnect: Scalars["ID"];
};

export type RoleEdge = Edge<Role, "RoleEdge">;

export type RoleSearchInput = BaseSearchInput<RoleSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    searchString?: InputMaybe<Scalars["String"]>;
    teamId: Scalars["ID"];
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type RoleSearchResult = SearchResult<RoleEdge, "RoleSearchResult">;

export enum RoleSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    MembersAsc = "MembersAsc",
    MembersDesc = "MembersDesc"
}

export type RoleTranslation = BaseTranslation<"RoleTranslation"> & {
    description: Scalars["String"];
};

export type RoleTranslationCreateInput = BaseTranslationCreateInput & {
    description: Scalars["String"];
};

export type RoleTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
};

export type RoleUpdateInput = BaseTranslatableUpdateInput<RoleTranslationCreateInput, RoleTranslationUpdateInput> & {
    id: Scalars["ID"];
    membersConnect?: InputMaybe<Array<Scalars["ID"]>>;
    membersDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    name?: InputMaybe<Scalars["String"]>;
    permissions?: InputMaybe<Scalars["String"]>;
};

export type Routine = DbObject<"Routine"> & {
    bookmarkedBy: Array<User>;
    bookmarks: Scalars["Int"];
    completedAt?: Maybe<Scalars["Date"]>;
    createdBy?: Maybe<User>;
    created_at: Scalars["Date"];
    forks: Array<Routine>;
    forksCount: Scalars["Int"];
    hasCompleteVersion: Scalars["Boolean"];
    isDeleted: Scalars["Boolean"];
    isInternal?: Maybe<Scalars["Boolean"]>;
    isPrivate: Scalars["Boolean"];
    issues: Array<Issue>;
    issuesCount: Scalars["Int"];
    labels: Array<Label>;
    owner?: Maybe<Owner>;
    parent?: Maybe<RoutineVersion>;
    permissions: Scalars["String"];
    pullRequests: Array<PullRequest>;
    pullRequestsCount: Scalars["Int"];
    questions: Array<Question>;
    questionsCount: Scalars["Int"];
    quizzes: Array<Quiz>;
    quizzesCount: Scalars["Int"];
    score: Scalars["Int"];
    stats: Array<StatsRoutine>;
    tags: Array<Tag>;
    transfers: Array<Transfer>;
    transfersCount: Scalars["Int"];
    translatedName: Scalars["String"];
    updated_at: Scalars["Date"];
    versions: Array<RoutineVersion>;
    versionsCount?: Maybe<Scalars["Int"]>;
    views: Scalars["Int"];
    you: RoutineYou;
};

export type RoutineCreateInput = {
    id: Scalars["ID"];
    isInternal?: InputMaybe<Scalars["Boolean"]>;
    isPrivate: Scalars["Boolean"];
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    ownedByTeamConnect?: InputMaybe<Scalars["ID"]>;
    ownedByUserConnect?: InputMaybe<Scalars["ID"]>;
    parentConnect?: InputMaybe<Scalars["ID"]>;
    permissions?: InputMaybe<Scalars["String"]>;
    tagsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
    versionsCreate?: InputMaybe<Array<RoutineVersionCreateInput>>;
};

export type RoutineEdge = Edge<Routine, "RoutineEdge">;

export type RoutineSearchInput = BaseSearchInput<RoutineSortBy> & {
    createdById?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    hasCompleteVersion?: InputMaybe<Scalars["Boolean"]>;
    isInternal?: InputMaybe<Scalars["Boolean"]>;
    issuesId?: InputMaybe<Scalars["ID"]>;
    labelsIds?: InputMaybe<Array<Scalars["ID"]>>;
    latestVersionRoutineType?: InputMaybe<RoutineType>;
    latestVersionRoutineTypes?: InputMaybe<Array<RoutineType>>;
    maxBookmarks?: InputMaybe<Scalars["Int"]>;
    maxScore?: InputMaybe<Scalars["Int"]>;
    maxViews?: InputMaybe<Scalars["Int"]>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    minScore?: InputMaybe<Scalars["Int"]>;
    minViews?: InputMaybe<Scalars["Int"]>;
    ownedByTeamId?: InputMaybe<Scalars["ID"]>;
    ownedByUserId?: InputMaybe<Scalars["ID"]>;
    parentId?: InputMaybe<Scalars["ID"]>;
    pullRequestsId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    tags?: InputMaybe<Array<Scalars["String"]>>;
    translationLanguagesLatestVersion?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type RoutineSearchResult = SearchResult<RoutineEdge, "RoutineSearchResult">;

export enum RoutineSortBy {
    BookmarksAsc = "BookmarksAsc",
    BookmarksDesc = "BookmarksDesc",
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
    ScoreAsc = "ScoreAsc",
    ScoreDesc = "ScoreDesc",
    VersionsAsc = "VersionsAsc",
    VersionsDesc = "VersionsDesc",
    ViewsAsc = "ViewsAsc",
    ViewsDesc = "ViewsDesc"
}

export enum RoutineType {
    Action = "Action",
    Api = "Api",
    Code = "Code",
    Data = "Data",
    Generate = "Generate",
    Informational = "Informational",
    MultiStep = "MultiStep",
    SmartContract = "SmartContract"
}

export type RoutineUpdateInput = {
    id: Scalars["ID"];
    isInternal?: InputMaybe<Scalars["Boolean"]>;
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    labelsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    ownedByTeamConnect?: InputMaybe<Scalars["ID"]>;
    ownedByUserConnect?: InputMaybe<Scalars["ID"]>;
    permissions?: InputMaybe<Scalars["String"]>;
    tagsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
    tagsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    versionsCreate?: InputMaybe<Array<RoutineVersionCreateInput>>;
    versionsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    versionsUpdate?: InputMaybe<Array<RoutineVersionUpdateInput>>;
};

export type RoutineVersion = DbObject<"RoutineVersion"> & {
    apiVersion?: Maybe<ApiVersion>;
    codeVersion?: Maybe<CodeVersion>;
    comments: Array<Comment>;
    commentsCount: Scalars["Int"];
    completedAt?: Maybe<Scalars["Date"]>;
    complexity: Scalars["Int"];
    config?: Maybe<Scalars["String"]>;
    created_at: Scalars["Date"];
    directoryListings: Array<ProjectVersionDirectory>;
    directoryListingsCount: Scalars["Int"];
    forks: Array<Routine>;
    forksCount: Scalars["Int"];
    inputs: Array<RoutineVersionInput>;
    inputsCount: Scalars["Int"];
    isAutomatable?: Maybe<Scalars["Boolean"]>;
    isComplete: Scalars["Boolean"];
    isDeleted: Scalars["Boolean"];
    isLatest: Scalars["Boolean"];
    isPrivate: Scalars["Boolean"];
    outputs: Array<RoutineVersionOutput>;
    outputsCount: Scalars["Int"];
    pullRequest?: Maybe<PullRequest>;
    reports: Array<Report>;
    reportsCount: Scalars["Int"];
    resourceList?: Maybe<ResourceList>;
    root: Routine;
    routineType: RoutineType;
    simplicity: Scalars["Int"];
    subroutineLinks: Array<RoutineVersion>;
    suggestedNextByRoutineVersion: Array<RoutineVersion>;
    suggestedNextByRoutineVersionCount: Scalars["Int"];
    timesCompleted: Scalars["Int"];
    timesStarted: Scalars["Int"];
    translations: Array<RoutineVersionTranslation>;
    translationsCount: Scalars["Int"];
    updated_at: Scalars["Date"];
    versionIndex: Scalars["Int"];
    versionLabel: Scalars["String"];
    versionNotes?: Maybe<Scalars["String"]>;
    you: RoutineVersionYou;
};

export type RoutineVersionCreateInput = BaseTranslatableCreateInput<RoutineVersionTranslationCreateInput> & {
    apiVersionConnect?: InputMaybe<Scalars["ID"]>;
    codeVersionConnect?: InputMaybe<Scalars["ID"]>;
    config?: InputMaybe<Scalars["String"]>;
    directoryListingsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    id: Scalars["ID"];
    inputsCreate?: InputMaybe<Array<RoutineVersionInputCreateInput>>;
    isAutomatable?: InputMaybe<Scalars["Boolean"]>;
    isComplete?: InputMaybe<Scalars["Boolean"]>;
    isPrivate: Scalars["Boolean"];
    outputsCreate?: InputMaybe<Array<RoutineVersionOutputCreateInput>>;
    resourceListCreate?: InputMaybe<ResourceListCreateInput>;
    rootConnect?: InputMaybe<Scalars["ID"]>;
    rootCreate?: InputMaybe<RoutineCreateInput>;
    routineType: RoutineType;
    subroutineLinksConnect?: InputMaybe<Array<Scalars["ID"]>>;
    suggestedNextByRoutineVersionConnect?: InputMaybe<Array<Scalars["ID"]>>;
    versionLabel: Scalars["String"];
    versionNotes?: InputMaybe<Scalars["String"]>;
};

export type RoutineVersionEdge = Edge<RoutineVersion, "RoutineVersionEdge">;

export type RoutineVersionInput = DbObject<"RoutineVersionInput"> & {
    index?: Maybe<Scalars["Int"]>;
    isRequired?: Maybe<Scalars["Boolean"]>;
    name?: Maybe<Scalars["String"]>;
    routineVersion: RoutineVersion;
    standardVersion?: Maybe<StandardVersion>;
    translations: Array<RoutineVersionInputTranslation>;
};

export type RoutineVersionInputCreateInput = BaseTranslatableCreateInput<RoutineVersionInputTranslationCreateInput> & {
    id: Scalars["ID"];
    index?: InputMaybe<Scalars["Int"]>;
    isRequired?: InputMaybe<Scalars["Boolean"]>;
    name?: InputMaybe<Scalars["String"]>;
    routineVersionConnect: Scalars["ID"];
    standardVersionConnect?: InputMaybe<Scalars["ID"]>;
    standardVersionCreate?: InputMaybe<StandardVersionCreateInput>;
};

export type RoutineVersionInputTranslation = BaseTranslation<"RoutineVersionInputTranslation"> & {
    description?: Maybe<Scalars["String"]>;
    helpText?: Maybe<Scalars["String"]>;
};

export type RoutineVersionInputTranslationCreateInput = BaseTranslationCreateInput & {
    description?: InputMaybe<Scalars["String"]>;
    helpText?: InputMaybe<Scalars["String"]>;
};

export type RoutineVersionInputTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
    helpText?: InputMaybe<Scalars["String"]>;
};

export type RoutineVersionInputUpdateInput = BaseTranslatableUpdateInput<RoutineVersionInputTranslationCreateInput, RoutineVersionInputTranslationUpdateInput> & {
    id: Scalars["ID"];
    index?: InputMaybe<Scalars["Int"]>;
    isRequired?: InputMaybe<Scalars["Boolean"]>;
    name?: InputMaybe<Scalars["String"]>;
    standardVersionConnect?: InputMaybe<Scalars["ID"]>;
    standardVersionCreate?: InputMaybe<StandardVersionCreateInput>;
    standardVersionDisconnect?: InputMaybe<Scalars["Boolean"]>;
};

export type RoutineVersionOutput = DbObject<"RoutineVersionOutput"> & {
    index?: Maybe<Scalars["Int"]>;
    name?: Maybe<Scalars["String"]>;
    routineVersion: RoutineVersion;
    standardVersion?: Maybe<StandardVersion>;
    translations: Array<RoutineVersionOutputTranslation>;
};

export type RoutineVersionOutputCreateInput = BaseTranslatableCreateInput<RoutineVersionOutputTranslationCreateInput> & {
    id: Scalars["ID"];
    index?: InputMaybe<Scalars["Int"]>;
    name?: InputMaybe<Scalars["String"]>;
    routineVersionConnect: Scalars["ID"];
    standardVersionConnect?: InputMaybe<Scalars["ID"]>;
    standardVersionCreate?: InputMaybe<StandardVersionCreateInput>;
};

export type RoutineVersionOutputTranslation = BaseTranslation<"RoutineVersionOutputTranslation"> & {
    description?: Maybe<Scalars["String"]>;
    helpText?: Maybe<Scalars["String"]>;
};

export type RoutineVersionOutputTranslationCreateInput = BaseTranslationCreateInput & {
    description?: InputMaybe<Scalars["String"]>;
    helpText?: InputMaybe<Scalars["String"]>;
};

export type RoutineVersionOutputTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
    helpText?: InputMaybe<Scalars["String"]>;
};

export type RoutineVersionOutputUpdateInput = BaseTranslatableUpdateInput<RoutineVersionOutputTranslationCreateInput, RoutineVersionOutputTranslationUpdateInput> & {
    id: Scalars["ID"];
    index?: InputMaybe<Scalars["Int"]>;
    name?: InputMaybe<Scalars["String"]>;
    standardVersionConnect?: InputMaybe<Scalars["ID"]>;
    standardVersionCreate?: InputMaybe<StandardVersionCreateInput>;
    standardVersionDisconnect?: InputMaybe<Scalars["Boolean"]>;
};

export type RoutineVersionSearchInput = BaseSearchInput<RoutineVersionSortBy> & {
    codeVersionId?: InputMaybe<Scalars["ID"]>;
    createdByIdRoot?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    directoryListingsId?: InputMaybe<Scalars["ID"]>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    isCompleteWithRoot?: InputMaybe<Scalars["Boolean"]>;
    isCompleteWithRootExcludeOwnedByTeamId?: InputMaybe<Scalars["ID"]>;
    isCompleteWithRootExcludeOwnedByUserId?: InputMaybe<Scalars["ID"]>;
    isExternalWithRootExcludeOwnedByTeamId?: InputMaybe<Scalars["ID"]>;
    isExternalWithRootExcludeOwnedByUserId?: InputMaybe<Scalars["ID"]>;
    isInternalWithRoot?: InputMaybe<Scalars["Boolean"]>;
    isInternalWithRootExcludeOwnedByTeamId?: InputMaybe<Scalars["ID"]>;
    isInternalWithRootExcludeOwnedByUserId?: InputMaybe<Scalars["ID"]>;
    isLatest?: InputMaybe<Scalars["Boolean"]>;
    maxBookmarksRoot?: InputMaybe<Scalars["Int"]>;
    maxComplexity?: InputMaybe<Scalars["Int"]>;
    maxScoreRoot?: InputMaybe<Scalars["Int"]>;
    maxSimplicity?: InputMaybe<Scalars["Int"]>;
    maxTimesCompleted?: InputMaybe<Scalars["Int"]>;
    maxViewsRoot?: InputMaybe<Scalars["Int"]>;
    minBookmarksRoot?: InputMaybe<Scalars["Int"]>;
    minComplexity?: InputMaybe<Scalars["Int"]>;
    minScoreRoot?: InputMaybe<Scalars["Int"]>;
    minSimplicity?: InputMaybe<Scalars["Int"]>;
    minTimesCompleted?: InputMaybe<Scalars["Int"]>;
    minViewsRoot?: InputMaybe<Scalars["Int"]>;
    ownedByTeamIdRoot?: InputMaybe<Scalars["ID"]>;
    ownedByUserIdRoot?: InputMaybe<Scalars["ID"]>;
    reportId?: InputMaybe<Scalars["ID"]>;
    rootId?: InputMaybe<Scalars["ID"]>;
    routineType?: InputMaybe<RoutineType>;
    routineTypes?: InputMaybe<Array<RoutineType>>;
    searchString?: InputMaybe<Scalars["String"]>;
    tagsRoot?: InputMaybe<Array<Scalars["String"]>>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type RoutineVersionSearchResult = SearchResult<RoutineVersionEdge, "RoutineVersionSearchResult">;

export enum RoutineVersionSortBy {
    CommentsAsc = "CommentsAsc",
    CommentsDesc = "CommentsDesc",
    ComplexityAsc = "ComplexityAsc",
    ComplexityDesc = "ComplexityDesc",
    DateCompletedAsc = "DateCompletedAsc",
    DateCompletedDesc = "DateCompletedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DirectoryListingsAsc = "DirectoryListingsAsc",
    DirectoryListingsDesc = "DirectoryListingsDesc",
    ForksAsc = "ForksAsc",
    ForksDesc = "ForksDesc",
    RunRoutinesAsc = "RunRoutinesAsc",
    RunRoutinesDesc = "RunRoutinesDesc",
    SimplicityAsc = "SimplicityAsc",
    SimplicityDesc = "SimplicityDesc"
}

export type RoutineVersionTranslation = BaseTranslation<"RoutineVersionTranslation"> & {
    description?: Maybe<Scalars["String"]>;
    instructions?: Maybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type RoutineVersionTranslationCreateInput = BaseTranslationCreateInput & {
    description?: InputMaybe<Scalars["String"]>;
    instructions?: InputMaybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type RoutineVersionTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
    instructions?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type RoutineVersionUpdateInput = BaseTranslatableUpdateInput<RoutineVersionTranslationCreateInput, RoutineVersionTranslationUpdateInput> & {
    apiVersionConnect?: InputMaybe<Scalars["ID"]>;
    apiVersionDisconnect?: InputMaybe<Scalars["Boolean"]>;
    codeVersionConnect?: InputMaybe<Scalars["ID"]>;
    codeVersionDisconnect?: InputMaybe<Scalars["Boolean"]>;
    config?: InputMaybe<Scalars["String"]>;
    directoryListingsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    directoryListingsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    id: Scalars["ID"];
    inputsCreate?: InputMaybe<Array<RoutineVersionInputCreateInput>>;
    inputsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    inputsUpdate?: InputMaybe<Array<RoutineVersionInputUpdateInput>>;
    isAutomatable?: InputMaybe<Scalars["Boolean"]>;
    isComplete?: InputMaybe<Scalars["Boolean"]>;
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    outputsCreate?: InputMaybe<Array<RoutineVersionOutputCreateInput>>;
    outputsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    outputsUpdate?: InputMaybe<Array<RoutineVersionOutputUpdateInput>>;
    resourceListCreate?: InputMaybe<ResourceListCreateInput>;
    resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
    rootUpdate?: InputMaybe<RoutineUpdateInput>;
    subroutineLinksConnect?: InputMaybe<Array<Scalars["ID"]>>;
    subroutineLinksDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    suggestedNextByRoutineVersionConnect?: InputMaybe<Array<Scalars["ID"]>>;
    suggestedNextByRoutineVersionDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    versionLabel?: InputMaybe<Scalars["String"]>;
    versionNotes?: InputMaybe<Scalars["String"]>;
};

export type RoutineVersionYou = {
    __typename: "RoutineVersionYou";
    canBookmark: Scalars["Boolean"];
    canComment: Scalars["Boolean"];
    canCopy: Scalars["Boolean"];
    canDelete: Scalars["Boolean"];
    canReact: Scalars["Boolean"];
    canRead: Scalars["Boolean"];
    canReport: Scalars["Boolean"];
    canRun: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
};

export type RoutineYou = {
    __typename: "RoutineYou";
    canBookmark: Scalars["Boolean"];
    canComment: Scalars["Boolean"];
    canDelete: Scalars["Boolean"];
    canReact: Scalars["Boolean"];
    canRead: Scalars["Boolean"];
    canTransfer: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
    isBookmarked: Scalars["Boolean"];
    isViewed: Scalars["Boolean"];
    reaction?: Maybe<Scalars["String"]>;
};

export type RunProject = DbObject<"RunProject"> & {
    completedAt?: Maybe<Scalars["Date"]>;
    completedComplexity: Scalars["Int"];
    contextSwitches: Scalars["Int"];
    data?: Maybe<Scalars["String"]>;
    isPrivate: Scalars["Boolean"];
    lastStep?: Maybe<Array<Scalars["Int"]>>;
    name: Scalars["String"];
    projectVersion?: Maybe<ProjectVersion>;
    schedule?: Maybe<Schedule>;
    startedAt?: Maybe<Scalars["Date"]>;
    status: RunStatus;
    steps: Array<RunProjectStep>;
    stepsCount: Scalars["Int"];
    team?: Maybe<Team>;
    timeElapsed?: Maybe<Scalars["Int"]>;
    user?: Maybe<User>;
    you: RunProjectYou;
};

export type RunProjectCreateInput = {
    completedComplexity?: InputMaybe<Scalars["Int"]>;
    contextSwitches?: InputMaybe<Scalars["Int"]>;
    data?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    isPrivate: Scalars["Boolean"];
    name: Scalars["String"];
    projectVersionConnect: Scalars["ID"];
    scheduleCreate?: InputMaybe<ScheduleCreateInput>;
    startedAt?: InputMaybe<Scalars["Date"]>;
    status: RunStatus;
    stepsCreate?: InputMaybe<Array<RunProjectStepCreateInput>>;
    teamConnect?: InputMaybe<Scalars["ID"]>;
    timeElapsed?: InputMaybe<Scalars["Int"]>;
};

export type RunProjectEdge = Edge<RunProject, "RunProjectEdge">;

export type RunProjectOrRunRoutine = RunProject | RunRoutine;

export type RunProjectOrRunRoutineEdge = Edge<RunProjectOrRunRoutine, "RunProjectOrRunRoutineEdge">;

export type RunProjectOrRunRoutinePageInfo = {
    __typename: "RunProjectOrRunRoutinePageInfo";
    endCursorRunProject?: Maybe<Scalars["String"]>;
    endCursorRunRoutine?: Maybe<Scalars["String"]>;
    hasNextPage: Scalars["Boolean"];
};

export type RunProjectOrRunRoutineSearchInput = MultiObjectSearchInput<RunProjectOrRunRoutineSortBy> & {
    completedTimeFrame?: InputMaybe<TimeFrame>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    objectType?: InputMaybe<Scalars["String"]>;
    projectVersionId?: InputMaybe<Scalars["ID"]>;
    routineVersionId?: InputMaybe<Scalars["ID"]>;
    runProjectAfter?: InputMaybe<Scalars["String"]>;
    runRoutineAfter?: InputMaybe<Scalars["String"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    startedTimeFrame?: InputMaybe<TimeFrame>;
    status?: InputMaybe<RunStatus>;
    statuses?: InputMaybe<Array<RunStatus>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type RunProjectOrRunRoutineSearchResult = SearchResult<RunProjectOrRunRoutineEdge, "RunProjectOrRunRoutineSearchResult", RunProjectOrRunRoutinePageInfo>;

export enum RunProjectOrRunRoutineSortBy {
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
    StepsAsc = "StepsAsc",
    StepsDesc = "StepsDesc"
}

export type RunProjectSearchInput = BaseSearchInput<RunProjectSortBy> & {
    completedTimeFrame?: InputMaybe<TimeFrame>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    projectVersionId?: InputMaybe<Scalars["ID"]>;
    scheduleEndTimeFrame?: InputMaybe<TimeFrame>;
    scheduleStartTimeFrame?: InputMaybe<TimeFrame>;
    searchString?: InputMaybe<Scalars["String"]>;
    startedTimeFrame?: InputMaybe<TimeFrame>;
    status?: InputMaybe<RunStatus>;
    statuses?: InputMaybe<Array<RunStatus>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type RunProjectSearchResult = SearchResult<RunProjectEdge, "RunProjectSearchResult">;

export enum RunProjectSortBy {
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
    StepsDesc = "StepsDesc"
}

export type RunProjectStep = DbObject<"RunProjectStep"> & {
    completedAt?: Maybe<Scalars["Date"]>;
    complexity: Scalars["Int"];
    contextSwitches: Scalars["Int"];
    directory?: Maybe<ProjectVersionDirectory>;
    directoryInId: Scalars["ID"];
    name: Scalars["String"];
    order: Scalars["Int"];
    runProject: RunProject;
    startedAt?: Maybe<Scalars["Date"]>;
    status: RunStepStatus;
    timeElapsed?: Maybe<Scalars["Int"]>;
};

export type RunProjectStepCreateInput = {
    complexity: Scalars["Int"];
    contextSwitches?: InputMaybe<Scalars["Int"]>;
    directoryConnect?: InputMaybe<Scalars["ID"]>;
    directoryInId: Scalars["ID"];
    id: Scalars["ID"];
    name: Scalars["String"];
    order: Scalars["Int"];
    runProjectConnect: Scalars["ID"];
    status?: InputMaybe<RunStepStatus>;
    timeElapsed?: InputMaybe<Scalars["Int"]>;
};

export type RunProjectStepUpdateInput = {
    contextSwitches?: InputMaybe<Scalars["Int"]>;
    id: Scalars["ID"];
    status?: InputMaybe<RunStepStatus>;
    timeElapsed?: InputMaybe<Scalars["Int"]>;
};

export type RunProjectUpdateInput = {
    completedComplexity?: InputMaybe<Scalars["Int"]>;
    contextSwitches?: InputMaybe<Scalars["Int"]>;
    data?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    scheduleCreate?: InputMaybe<ScheduleCreateInput>;
    scheduleUpdate?: InputMaybe<ScheduleUpdateInput>;
    scheduleDelete?: InputMaybe<Scalars["Boolean"]>;
    startedAt?: InputMaybe<Scalars["Date"]>;
    status?: InputMaybe<RunStatus>;
    stepsCreate?: InputMaybe<Array<RunProjectStepCreateInput>>;
    stepsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    stepsUpdate?: InputMaybe<Array<RunProjectStepUpdateInput>>;
    timeElapsed?: InputMaybe<Scalars["Int"]>;
};

export type RunProjectYou = {
    __typename: "RunProjectYou";
    canDelete: Scalars["Boolean"];
    canRead: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
};

export type RunRoutine = DbObject<"RunRoutine"> & {
    completedAt?: Maybe<Scalars["Date"]>;
    completedComplexity: Scalars["Int"];
    contextSwitches: Scalars["Int"];
    data?: Maybe<Scalars["String"]>;
    io: Array<RunRoutineIO>;
    ioCount: Scalars["Int"];
    isPrivate: Scalars["Boolean"];
    lastStep?: Maybe<Array<Scalars["Int"]>>;
    name: Scalars["String"];
    routineVersion?: Maybe<RoutineVersion>;
    schedule?: Maybe<Schedule>;
    startedAt?: Maybe<Scalars["Date"]>;
    status: RunStatus;
    steps: Array<RunRoutineStep>;
    stepsCount: Scalars["Int"];
    team?: Maybe<Team>;
    timeElapsed?: Maybe<Scalars["Int"]>;
    user?: Maybe<User>;
    wasRunAutomatically: Scalars["Boolean"];
    you: RunRoutineYou;
};

export type RunRoutineCreateInput = {
    completedComplexity?: InputMaybe<Scalars["Int"]>;
    contextSwitches?: InputMaybe<Scalars["Int"]>;
    data?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    ioCreate?: InputMaybe<Array<RunRoutineIOCreateInput>>;
    isPrivate: Scalars["Boolean"];
    name: Scalars["String"];
    routineVersionConnect: Scalars["ID"];
    scheduleCreate?: InputMaybe<ScheduleCreateInput>;
    startedAt?: InputMaybe<Scalars["Date"]>;
    status: RunStatus;
    stepsCreate?: InputMaybe<Array<RunRoutineStepCreateInput>>;
    teamConnect?: InputMaybe<Scalars["ID"]>;
    timeElapsed?: InputMaybe<Scalars["Int"]>;
};

export type RunRoutineEdge = Edge<RunRoutine, "RunRoutineEdge">;

export type RunRoutineIO = DbObject<"RunRoutineIO"> & {
    data: Scalars["String"];
    nodeInputName: Scalars["String"];
    nodeName: Scalars["String"];
    runRoutine: RunRoutine;
    routineVersionInput: Maybe<RoutineVersionInput>;
    routineVersionOutput: Maybe<RoutineVersionOutput>;
};

export type RunRoutineIOCreateInput = {
    data: Scalars["String"];
    id: Scalars["ID"];
    nodeInputName: Scalars["String"];
    nodeName: Scalars["String"];
    runRoutineConnect: Scalars["ID"];
    routineVersionInputConnect: InputMaybe<Scalars["ID"]>;
    routineVersionOutputConnect: InputMaybe<Scalars["ID"]>;
};

export type RunRoutineIOEdge = Edge<RunRoutineIO, "RunRoutineIOEdge">;

export type RunRoutineIOSearchInput = {
    after?: InputMaybe<Scalars["String"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    ids?: InputMaybe<Array<Scalars["ID"]>>;
    runRoutineIds?: InputMaybe<Array<Scalars["ID"]>>;
    take?: InputMaybe<Scalars["Int"]>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type RunRoutineIOSearchResult = SearchResult<RunRoutineIOEdge, "RunRoutineIOSearchResult">;

export enum RunRoutineIOSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc"
}

export type RunRoutineIOUpdateInput = {
    data: Scalars["String"];
    id: Scalars["ID"];
};

export type RunRoutineSearchInput = BaseSearchInput<RunRoutineSortBy> & {
    completedTimeFrame?: InputMaybe<TimeFrame>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    routineVersionId?: InputMaybe<Scalars["ID"]>;
    scheduleEndTimeFrame?: InputMaybe<TimeFrame>;
    scheduleStartTimeFrame?: InputMaybe<TimeFrame>;
    searchString?: InputMaybe<Scalars["String"]>;
    startedTimeFrame?: InputMaybe<TimeFrame>;
    status?: InputMaybe<RunStatus>;
    statuses?: InputMaybe<Array<RunStatus>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type RunRoutineSearchResult = SearchResult<RunRoutineEdge, "RunRoutineSearchResult">;

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
    StepsAsc = "StepsAsc",
    StepsDesc = "StepsDesc"
}

export type RunRoutineStep = DbObject<"RunRoutineStep"> & {
    completedAt?: Maybe<Scalars["Date"]>;
    complexity: Scalars["Int"];
    contextSwitches: Scalars["Int"];
    name: Scalars["String"];
    nodeId: Scalars["String"];
    order: Scalars["Int"];
    runRoutine: RunRoutine;
    startedAt?: Maybe<Scalars["Date"]>;
    status: RunStepStatus;
    subroutine?: Maybe<RoutineVersion>;
    subroutineInId: Scalars["ID"];
    timeElapsed?: Maybe<Scalars["Int"]>;
};

export type RunRoutineStepCreateInput = {
    complexity: Scalars["Int"];
    contextSwitches?: InputMaybe<Scalars["Int"]>;
    id: Scalars["ID"];
    name: Scalars["String"];
    nodeId: Scalars["String"];
    order: Scalars["Int"];
    runRoutineConnect: Scalars["ID"];
    status?: InputMaybe<RunStepStatus>;
    subroutineConnect?: InputMaybe<Scalars["ID"]>;
    subroutineInId: Scalars["ID"];
    timeElapsed?: InputMaybe<Scalars["Int"]>;
};

export enum RunRoutineStepSortBy {
    ContextSwitchesAsc = "ContextSwitchesAsc",
    ContextSwitchesDesc = "ContextSwitchesDesc",
    OrderAsc = "OrderAsc",
    OrderDesc = "OrderDesc",
    TimeCompletedAsc = "TimeCompletedAsc",
    TimeCompletedDesc = "TimeCompletedDesc",
    TimeElapsedAsc = "TimeElapsedAsc",
    TimeElapsedDesc = "TimeElapsedDesc",
    TimeStartedAsc = "TimeStartedAsc",
    TimeStartedDesc = "TimeStartedDesc"
}

export type RunRoutineStepUpdateInput = {
    contextSwitches?: InputMaybe<Scalars["Int"]>;
    id: Scalars["ID"];
    status?: InputMaybe<RunStepStatus>;
    timeElapsed?: InputMaybe<Scalars["Int"]>;
};

export type RunRoutineUpdateInput = {
    completedComplexity?: InputMaybe<Scalars["Int"]>;
    contextSwitches?: InputMaybe<Scalars["Int"]>;
    data?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    ioCreate?: InputMaybe<Array<RunRoutineIOCreateInput>>;
    ioDelete?: InputMaybe<Array<Scalars["ID"]>>;
    ioUpdate?: InputMaybe<Array<RunRoutineIOUpdateInput>>;
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    scheduleCreate?: InputMaybe<ScheduleCreateInput>;
    scheduleUpdate?: InputMaybe<ScheduleUpdateInput>;
    scheduleDelete?: InputMaybe<Scalars["Boolean"]>;
    startedAt?: InputMaybe<Scalars["Date"]>;
    status?: InputMaybe<RunStatus>;
    stepsCreate?: InputMaybe<Array<RunRoutineStepCreateInput>>;
    stepsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    stepsUpdate?: InputMaybe<Array<RunRoutineStepUpdateInput>>;
    timeElapsed?: InputMaybe<Scalars["Int"]>;
};

export type RunRoutineYou = {
    __typename: "RunRoutineYou";
    canDelete: Scalars["Boolean"];
    canRead: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
};

export enum RunStatus {
    Cancelled = "Cancelled",
    Completed = "Completed",
    Failed = "Failed",
    InProgress = "InProgress",
    Paused = "Paused",
    Scheduled = "Scheduled"
}

export enum RunStepStatus {
    Completed = "Completed",
    InProgress = "InProgress",
    Skipped = "Skipped"
}

export enum RunTask {
    RunProject = "RunProject",
    RunRoutine = "RunRoutine"
}

export enum SandboxTask {
    CallApi = "CallApi",
    RunDataTransform = "RunDataTransform",
    RunSmartContract = "RunSmartContract"
}

export type Schedule = DbObject<"Schedule"> & {
    created_at: Scalars["Date"];
    endTime: Scalars["Date"];
    exceptions: Array<ScheduleException>;
    focusModes: Array<FocusMode>;
    labels: Array<Label>;
    meetings: Array<Meeting>;
    recurrences: Array<ScheduleRecurrence>;
    runProjects: Array<RunProject>;
    runRoutines: Array<RunRoutine>;
    startTime: Scalars["Date"];
    timezone: Scalars["String"];
    updated_at: Scalars["Date"];
};

export type ScheduleCreateInput = {
    endTime?: InputMaybe<Scalars["Date"]>;
    exceptionsCreate?: InputMaybe<Array<ScheduleExceptionCreateInput>>;
    focusModeConnect?: InputMaybe<Scalars["ID"]>;
    id: Scalars["ID"];
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    meetingConnect?: InputMaybe<Scalars["ID"]>;
    recurrencesCreate?: InputMaybe<Array<ScheduleRecurrenceCreateInput>>;
    runProjectConnect?: InputMaybe<Scalars["ID"]>;
    runRoutineConnect?: InputMaybe<Scalars["ID"]>;
    startTime?: InputMaybe<Scalars["Date"]>;
    timezone: Scalars["String"];
};

export type ScheduleEdge = Edge<Schedule, "ScheduleEdge">;

export type ScheduleException = DbObject<"ScheduleException"> & {
    newEndTime?: Maybe<Scalars["Date"]>;
    newStartTime?: Maybe<Scalars["Date"]>;
    originalStartTime: Scalars["Date"];
    schedule: Schedule;
};

export type ScheduleExceptionCreateInput = {
    id: Scalars["ID"];
    newEndTime?: InputMaybe<Scalars["Date"]>;
    newStartTime?: InputMaybe<Scalars["Date"]>;
    originalStartTime: Scalars["Date"];
    scheduleConnect: Scalars["ID"];
};

export type ScheduleExceptionEdge = Edge<ScheduleException, "ScheduleExceptionEdge">;

export type ScheduleExceptionSearchInput = BaseSearchInput<ScheduleExceptionSortBy> & {
    newEndTimeFrame?: InputMaybe<TimeFrame>;
    newStartTimeFrame?: InputMaybe<TimeFrame>;
    originalStartTimeFrame?: InputMaybe<TimeFrame>;
    searchString?: InputMaybe<Scalars["String"]>;
};

export type ScheduleExceptionSearchResult = SearchResult<ScheduleExceptionEdge, "ScheduleExceptionSearchResult">;

export enum ScheduleExceptionSortBy {
    NewEndTimeAsc = "NewEndTimeAsc",
    NewEndTimeDesc = "NewEndTimeDesc",
    NewStartTimeAsc = "NewStartTimeAsc",
    NewStartTimeDesc = "NewStartTimeDesc",
    OriginalStartTimeAsc = "OriginalStartTimeAsc",
    OriginalStartTimeDesc = "OriginalStartTimeDesc"
}

export type ScheduleExceptionUpdateInput = {
    id: Scalars["ID"];
    newEndTime?: InputMaybe<Scalars["Date"]>;
    newStartTime?: InputMaybe<Scalars["Date"]>;
    originalStartTime?: InputMaybe<Scalars["Date"]>;
};

export enum ScheduleFor {
    FocusMode = "FocusMode",
    Meeting = "Meeting",
    RunProject = "RunProject",
    RunRoutine = "RunRoutine"
}

export type ScheduleRecurrence = DbObject<"ScheduleRecurrence"> & {
    dayOfMonth?: Maybe<Scalars["Int"]>;
    dayOfWeek?: Maybe<Scalars["Int"]>;
    duration: Scalars["Int"];
    endDate?: Maybe<Scalars["Date"]>;
    interval: Scalars["Int"];
    month?: Maybe<Scalars["Int"]>;
    recurrenceType: ScheduleRecurrenceType;
    schedule: Schedule;
};

export type ScheduleRecurrenceCreateInput = {
    dayOfMonth?: InputMaybe<Scalars["Int"]>;
    dayOfWeek?: InputMaybe<Scalars["Int"]>;
    duration: Scalars["Int"];
    endDate?: InputMaybe<Scalars["Date"]>;
    id: Scalars["ID"];
    interval: Scalars["Int"];
    month?: InputMaybe<Scalars["Int"]>;
    recurrenceType: ScheduleRecurrenceType;
    scheduleConnect?: InputMaybe<Scalars["ID"]>;
    scheduleCreate?: InputMaybe<ScheduleCreateInput>;
};

export type ScheduleRecurrenceEdge = Edge<ScheduleRecurrence, "ScheduleRecurrenceEdge">;

export type ScheduleRecurrenceSearchInput = BaseSearchInput<ScheduleRecurrenceSortBy> & {
    dayOfMonth?: InputMaybe<Scalars["Int"]>;
    dayOfWeek?: InputMaybe<Scalars["Int"]>;
    endDateTimeFrame?: InputMaybe<TimeFrame>;
    interval?: InputMaybe<Scalars["Int"]>;
    month?: InputMaybe<Scalars["Int"]>;
    recurrenceType?: InputMaybe<ScheduleRecurrenceType>;
    searchString?: InputMaybe<Scalars["String"]>;
};

export type ScheduleRecurrenceSearchResult = SearchResult<ScheduleRecurrenceEdge, "ScheduleRecurrenceSearchResult">;

export enum ScheduleRecurrenceSortBy {
    DayOfMonthAsc = "DayOfMonthAsc",
    DayOfMonthDesc = "DayOfMonthDesc",
    DayOfWeekAsc = "DayOfWeekAsc",
    DayOfWeekDesc = "DayOfWeekDesc",
    EndDateAsc = "EndDateAsc",
    EndDateDesc = "EndDateDesc",
    MonthAsc = "MonthAsc",
    MonthDesc = "MonthDesc"
}

export enum ScheduleRecurrenceType {
    Daily = "Daily",
    Monthly = "Monthly",
    Weekly = "Weekly",
    Yearly = "Yearly"
}

export type ScheduleRecurrenceUpdateInput = {
    dayOfMonth?: InputMaybe<Scalars["Int"]>;
    dayOfWeek?: InputMaybe<Scalars["Int"]>;
    duration?: InputMaybe<Scalars["Int"]>;
    endDate?: InputMaybe<Scalars["Date"]>;
    id: Scalars["ID"];
    interval?: InputMaybe<Scalars["Int"]>;
    month?: InputMaybe<Scalars["Int"]>;
    recurrenceType?: InputMaybe<ScheduleRecurrenceType>;
};

export type ScheduleSearchInput = BaseSearchInput<ScheduleSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    endTimeFrame?: InputMaybe<TimeFrame>;
    scheduleFor?: InputMaybe<ScheduleFor>;
    scheduleForUserId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    startTimeFrame?: InputMaybe<TimeFrame>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type ScheduleSearchResult = SearchResult<ScheduleEdge, "ScheduleSearchResult">;

export enum ScheduleSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    EndTimeAsc = "EndTimeAsc",
    EndTimeDesc = "EndTimeDesc",
    StartTimeAsc = "StartTimeAsc",
    StartTimeDesc = "StartTimeDesc"
}

export type ScheduleUpdateInput = {
    endTime?: InputMaybe<Scalars["Date"]>;
    exceptionsCreate?: InputMaybe<Array<ScheduleExceptionCreateInput>>;
    exceptionsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    exceptionsUpdate?: InputMaybe<Array<ScheduleExceptionUpdateInput>>;
    id: Scalars["ID"];
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    labelsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    recurrencesCreate?: InputMaybe<Array<ScheduleRecurrenceCreateInput>>;
    recurrencesDelete?: InputMaybe<Array<Scalars["ID"]>>;
    recurrencesUpdate?: InputMaybe<Array<ScheduleRecurrenceUpdateInput>>;
    startTime?: InputMaybe<Scalars["Date"]>;
    timezone?: InputMaybe<Scalars["String"]>;
};

export type SearchException = {
    field: Scalars["String"];
    value: Scalars["String"];
};

export type SendVerificationEmailInput = {
    emailAddress: Scalars["String"];
};

export type SendVerificationTextInput = {
    phoneNumber: Scalars["String"];
};

export type Session = {
    __typename: "Session";
    isLoggedIn: Scalars["Boolean"];
    timeZone?: Maybe<Scalars["String"]>;
    users?: Maybe<Array<SessionUser>>;
};

export type SessionUser = {
    __typename: "SessionUser";
    activeFocusMode?: Maybe<ActiveFocusMode>;
    apisCount: Scalars["Int"];
    codesCount: Scalars["Int"];
    credits: Scalars["String"];
    handle?: Maybe<Scalars["String"]>;
    hasPremium: Scalars["Boolean"];
    id: Scalars["String"];
    languages: Array<Scalars["String"]>;
    membershipsCount: Scalars["Int"];
    name?: Maybe<Scalars["String"]>;
    notesCount: Scalars["Int"];
    profileImage?: Maybe<Scalars["String"]>;
    projectsCount: Scalars["Int"];
    questionsAskedCount: Scalars["Int"];
    routinesCount: Scalars["Int"];
    session: SessionUserSession;
    standardsCount: Scalars["Int"];
    theme?: Maybe<Scalars["String"]>;
    updated_at: Scalars["Date"];
};

export type SessionUserSession = {
    __typename: "SessionUserSession";
    id: Scalars["String"];
    lastRefreshAt: Scalars["Date"];
};

export type SetActiveFocusModeInput = {
    id?: InputMaybe<Scalars["ID"]>;
    stopCondition?: InputMaybe<FocusModeStopCondition>;
    stopTime?: InputMaybe<Scalars["Date"]>;
};

export type Standard = DbObject<"Standard"> & {
    bookmarkedBy: Array<User>;
    bookmarks: Scalars["Int"];
    completedAt?: Maybe<Scalars["Date"]>;
    createdBy?: Maybe<User>;
    created_at: Scalars["Date"];
    forks: Array<Standard>;
    forksCount: Scalars["Int"];
    hasCompleteVersion: Scalars["Boolean"];
    isDeleted: Scalars["Boolean"];
    isInternal: Scalars["Boolean"];
    isPrivate: Scalars["Boolean"];
    issues: Array<Issue>;
    issuesCount: Scalars["Int"];
    labels: Array<Label>;
    owner?: Maybe<Owner>;
    parent?: Maybe<StandardVersion>;
    permissions: Scalars["String"];
    pullRequests: Array<PullRequest>;
    pullRequestsCount: Scalars["Int"];
    questions: Array<Question>;
    questionsCount: Scalars["Int"];
    score: Scalars["Int"];
    stats: Array<StatsStandard>;
    tags: Array<Tag>;
    transfers: Array<Transfer>;
    transfersCount: Scalars["Int"];
    translatedName: Scalars["String"];
    updated_at: Scalars["Date"];
    versions: Array<StandardVersion>;
    versionsCount?: Maybe<Scalars["Int"]>;
    views: Scalars["Int"];
    you: StandardYou;
};

export type StandardCreateInput = {
    id: Scalars["ID"];
    isInternal?: InputMaybe<Scalars["Boolean"]>;
    isPrivate: Scalars["Boolean"];
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    ownedByTeamConnect?: InputMaybe<Scalars["ID"]>;
    ownedByUserConnect?: InputMaybe<Scalars["ID"]>;
    parentConnect?: InputMaybe<Scalars["ID"]>;
    permissions?: InputMaybe<Scalars["String"]>;
    tagsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
    versionsCreate?: InputMaybe<Array<StandardVersionCreateInput>>;
};

export type StandardEdge = Edge<Standard, "StandardEdge">;

export type StandardSearchInput = BaseSearchInput<StandardSortBy> & {
    codeLanguageLatestVersion?: InputMaybe<Scalars["String"]>;
    createdById?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    hasCompleteVersion?: InputMaybe<Scalars["Boolean"]>;
    isInternal?: InputMaybe<Scalars["Boolean"]>;
    issuesId?: InputMaybe<Scalars["ID"]>;
    labelsIds?: InputMaybe<Array<Scalars["ID"]>>;
    maxBookmarks?: InputMaybe<Scalars["Int"]>;
    maxScore?: InputMaybe<Scalars["Int"]>;
    maxViews?: InputMaybe<Scalars["Int"]>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    minScore?: InputMaybe<Scalars["Int"]>;
    minViews?: InputMaybe<Scalars["Int"]>;
    ownedByTeamId?: InputMaybe<Scalars["ID"]>;
    ownedByUserId?: InputMaybe<Scalars["ID"]>;
    parentId?: InputMaybe<Scalars["ID"]>;
    pullRequestsId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    tags?: InputMaybe<Array<Scalars["String"]>>;
    translationLanguagesLatestVersion?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    variantLatestVersion?: InputMaybe<StandardType>;
    visibility?: InputMaybe<VisibilityType>;
};

export type StandardSearchResult = SearchResult<StandardEdge, "StandardSearchResult">;

export enum StandardSortBy {
    BookmarksAsc = "BookmarksAsc",
    BookmarksDesc = "BookmarksDesc",
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
    ScoreAsc = "ScoreAsc",
    ScoreDesc = "ScoreDesc",
    VersionsAsc = "VersionsAsc",
    VersionsDesc = "VersionsDesc",
    ViewsAsc = "ViewsAsc",
    ViewsDesc = "ViewsDesc"
}

export enum StandardType {
    DataStructure = "DataStructure",
    Prompt = "Prompt"
}

export type StandardUpdateInput = {
    id: Scalars["ID"];
    isInternal?: InputMaybe<Scalars["Boolean"]>;
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    labelsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    labelsCreate?: InputMaybe<Array<LabelCreateInput>>;
    labelsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    ownedByTeamConnect?: InputMaybe<Scalars["ID"]>;
    ownedByUserConnect?: InputMaybe<Scalars["ID"]>;
    permissions?: InputMaybe<Scalars["String"]>;
    tagsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
    tagsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    versionsCreate?: InputMaybe<Array<StandardVersionCreateInput>>;
    versionsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    versionsUpdate?: InputMaybe<Array<StandardVersionUpdateInput>>;
};

export type StandardVersion = DbObject<"StandardVersion"> & {
    codeLanguage: Scalars["String"];
    comments: Array<Comment>;
    commentsCount: Scalars["Int"];
    completedAt?: Maybe<Scalars["Date"]>;
    created_at: Scalars["Date"];
    default?: Maybe<Scalars["String"]>;
    directoryListings: Array<ProjectVersionDirectory>;
    directoryListingsCount: Scalars["Int"];
    forks: Array<Standard>;
    forksCount: Scalars["Int"];
    isComplete: Scalars["Boolean"];
    isDeleted: Scalars["Boolean"];
    isFile?: Maybe<Scalars["Boolean"]>;
    isLatest: Scalars["Boolean"];
    isPrivate: Scalars["Boolean"];
    props: Scalars["String"];
    pullRequest?: Maybe<PullRequest>;
    reports: Array<Report>;
    reportsCount: Scalars["Int"];
    resourceList?: Maybe<ResourceList>;
    root: Standard;
    translations: Array<StandardVersionTranslation>;
    translationsCount: Scalars["Int"];
    updated_at: Scalars["Date"];
    variant: StandardType;
    versionIndex: Scalars["Int"];
    versionLabel: Scalars["String"];
    versionNotes?: Maybe<Scalars["String"]>;
    you: VersionYou;
    yup?: Maybe<Scalars["String"]>;
};

export type StandardVersionCreateInput = BaseTranslatableCreateInput<StandardVersionTranslationCreateInput> & {
    codeLanguage: Scalars["String"];
    default?: InputMaybe<Scalars["String"]>;
    directoryListingsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    id: Scalars["ID"];
    isComplete?: InputMaybe<Scalars["Boolean"]>;
    isFile?: InputMaybe<Scalars["Boolean"]>;
    isPrivate: Scalars["Boolean"];
    props: Scalars["String"];
    resourceListCreate?: InputMaybe<ResourceListCreateInput>;
    rootConnect?: InputMaybe<Scalars["ID"]>;
    rootCreate?: InputMaybe<StandardCreateInput>;
    variant: StandardType;
    versionLabel: Scalars["String"];
    versionNotes?: InputMaybe<Scalars["String"]>;
    yup?: InputMaybe<Scalars["String"]>;
};

export type StandardVersionEdge = Edge<StandardVersion, "StandardVersionEdge">;

export type StandardVersionSearchInput = BaseSearchInput<StandardVersionSortBy> & {
    codeLanguage?: InputMaybe<Scalars["String"]>;
    completedTimeFrame?: InputMaybe<TimeFrame>;
    createdByIdRoot?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    isCompleteWithRoot?: InputMaybe<Scalars["Boolean"]>;
    isInternalWithRoot?: InputMaybe<Scalars["Boolean"]>;
    isLatest?: InputMaybe<Scalars["Boolean"]>;
    maxBookmarksRoot?: InputMaybe<Scalars["Int"]>;
    maxScoreRoot?: InputMaybe<Scalars["Int"]>;
    maxViewsRoot?: InputMaybe<Scalars["Int"]>;
    minBookmarksRoot?: InputMaybe<Scalars["Int"]>;
    minScoreRoot?: InputMaybe<Scalars["Int"]>;
    minViewsRoot?: InputMaybe<Scalars["Int"]>;
    ownedByTeamIdRoot?: InputMaybe<Scalars["ID"]>;
    ownedByUserIdRoot?: InputMaybe<Scalars["ID"]>;
    reportId?: InputMaybe<Scalars["ID"]>;
    rootId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    tagsRoot?: InputMaybe<Array<Scalars["String"]>>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userId?: InputMaybe<Scalars["ID"]>;
    variant?: InputMaybe<StandardType>;
    visibility?: InputMaybe<VisibilityType>;
};

export type StandardVersionSearchResult = SearchResult<StandardVersionEdge, "StandardVersionSearchResult">;

export enum StandardVersionSortBy {
    CommentsAsc = "CommentsAsc",
    CommentsDesc = "CommentsDesc",
    DateCompletedAsc = "DateCompletedAsc",
    DateCompletedDesc = "DateCompletedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DirectoryListingsAsc = "DirectoryListingsAsc",
    DirectoryListingsDesc = "DirectoryListingsDesc",
    ForksAsc = "ForksAsc",
    ForksDesc = "ForksDesc",
    ReportsAsc = "ReportsAsc",
    ReportsDesc = "ReportsDesc"
}

export type StandardVersionTranslation = BaseTranslation<"StandardVersionTranslation"> & {
    description?: Maybe<Scalars["String"]>;
    jsonVariable?: Maybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type StandardVersionTranslationCreateInput = BaseTranslationCreateInput & {
    description?: InputMaybe<Scalars["String"]>;
    jsonVariable?: InputMaybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type StandardVersionTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
    jsonVariable?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type StandardVersionUpdateInput = BaseTranslatableUpdateInput<StandardVersionTranslationCreateInput, StandardVersionTranslationUpdateInput> & {
    codeLanguage?: InputMaybe<Scalars["String"]>;
    default?: InputMaybe<Scalars["String"]>;
    directoryListingsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    directoryListingsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    id: Scalars["ID"];
    isComplete?: InputMaybe<Scalars["Boolean"]>;
    isFile?: InputMaybe<Scalars["Boolean"]>;
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    props?: InputMaybe<Scalars["String"]>;
    resourceListCreate?: InputMaybe<ResourceListCreateInput>;
    resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
    rootUpdate?: InputMaybe<StandardUpdateInput>;
    variant?: InputMaybe<StandardType>;
    versionLabel?: InputMaybe<Scalars["String"]>;
    versionNotes?: InputMaybe<Scalars["String"]>;
    yup?: InputMaybe<Scalars["String"]>;
};

export type StandardYou = {
    __typename: "StandardYou";
    canBookmark: Scalars["Boolean"];
    canDelete: Scalars["Boolean"];
    canReact: Scalars["Boolean"];
    canRead: Scalars["Boolean"];
    canTransfer: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
    isBookmarked: Scalars["Boolean"];
    isViewed: Scalars["Boolean"];
    reaction?: Maybe<Scalars["String"]>;
};

export type StartLlmTaskInput = {
    chatId: Scalars["ID"];
    model: string;
    parentId?: InputMaybe<Scalars["ID"]>;
    respondingBotId: Scalars["ID"];
    shouldNotRunTasks: Scalars["Boolean"];
    task: LlmTask;
    taskContexts: Array<TaskContextInfoInput>;
};

export type StartRunTaskInput = {
    config: RunConfig;
    formValues?: InputMaybe<Scalars["JSONObject"]>;
    isNewRun: Scalars["Boolean"];
    projectVersionId?: InputMaybe<Scalars["ID"]>;
    routineVersionId?: InputMaybe<Scalars["ID"]>;
    runId: Scalars["ID"];
};

export enum StatPeriodType {
    Daily = "Daily",
    Hourly = "Hourly",
    Monthly = "Monthly",
    Weekly = "Weekly",
    Yearly = "Yearly"
}

export type StatsApi = DbObject<"StatsApi"> & {
    calls: Scalars["Int"];
    periodEnd: Scalars["Date"];
    periodStart: Scalars["Date"];
    periodType: StatPeriodType;
    routineVersions: Scalars["Int"];
};

export type StatsApiEdge = Edge<StatsApi, "StatsApiEdge">;

export type StatsApiSearchInput = BaseSearchInput<StatsApiSortBy> & {
    periodTimeFrame?: InputMaybe<TimeFrame>;
    periodType: StatPeriodType;
    searchString?: InputMaybe<Scalars["String"]>;
};

export type StatsApiSearchResult = SearchResult<StatsApiEdge, "StatsApiSearchResult">;

export enum StatsApiSortBy {
    PeriodStartAsc = "PeriodStartAsc",
    PeriodStartDesc = "PeriodStartDesc"
}

export type StatsCode = DbObject<"StatsCode"> & {
    calls: Scalars["Int"];
    periodEnd: Scalars["Date"];
    periodStart: Scalars["Date"];
    periodType: StatPeriodType;
    routineVersions: Scalars["Int"];
};

export type StatsCodeEdge = Edge<StatsCode, "StatsCodeEdge">;

export type StatsCodeSearchInput = BaseSearchInput<StatsCodeSortBy> & {
    periodTimeFrame?: InputMaybe<TimeFrame>;
    periodType: StatPeriodType;
    searchString?: InputMaybe<Scalars["String"]>;
};

export type StatsCodeSearchResult = SearchResult<StatsCodeEdge, "StatsCodeSearchResult">;

export enum StatsCodeSortBy {
    PeriodStartAsc = "PeriodStartAsc",
    PeriodStartDesc = "PeriodStartDesc"
}

export type StatsProject = DbObject<"StatsProject"> & {
    apis: Scalars["Int"];
    codes: Scalars["Int"];
    directories: Scalars["Int"];
    notes: Scalars["Int"];
    periodEnd: Scalars["Date"];
    periodStart: Scalars["Date"];
    periodType: StatPeriodType;
    projects: Scalars["Int"];
    routines: Scalars["Int"];
    runCompletionTimeAverage: Scalars["Float"];
    runContextSwitchesAverage: Scalars["Float"];
    runsCompleted: Scalars["Int"];
    runsStarted: Scalars["Int"];
    standards: Scalars["Int"];
    teams: Scalars["Int"];
};

export type StatsProjectEdge = Edge<StatsProject, "StatsProjectEdge">;

export type StatsProjectSearchInput = BaseSearchInput<StatsProjectSortBy> & {
    periodTimeFrame?: InputMaybe<TimeFrame>;
    periodType: StatPeriodType;
    searchString?: InputMaybe<Scalars["String"]>;
};

export type StatsProjectSearchResult = SearchResult<StatsProjectEdge, "StatsProjectSearchResult">;

export enum StatsProjectSortBy {
    PeriodStartAsc = "PeriodStartAsc",
    PeriodStartDesc = "PeriodStartDesc"
}

export type StatsQuiz = DbObject<"StatsQuiz"> & {
    completionTimeAverage: Scalars["Float"];
    periodEnd: Scalars["Date"];
    periodStart: Scalars["Date"];
    periodType: StatPeriodType;
    scoreAverage: Scalars["Float"];
    timesFailed: Scalars["Int"];
    timesPassed: Scalars["Int"];
    timesStarted: Scalars["Int"];
};

export type StatsQuizEdge = Edge<StatsQuiz, "StatsQuizEdge">;

export type StatsQuizSearchInput = BaseSearchInput<StatsQuizSortBy> & {
    periodTimeFrame?: InputMaybe<TimeFrame>;
    periodType: StatPeriodType;
    searchString?: InputMaybe<Scalars["String"]>;
};

export type StatsQuizSearchResult = SearchResult<StatsQuizEdge, "StatsQuizSearchResult">;

export enum StatsQuizSortBy {
    PeriodStartAsc = "PeriodStartAsc",
    PeriodStartDesc = "PeriodStartDesc"
}

export type StatsRoutine = DbObject<"StatsRoutine"> & {
    periodEnd: Scalars["Date"];
    periodStart: Scalars["Date"];
    periodType: StatPeriodType;
    runCompletionTimeAverage: Scalars["Float"];
    runContextSwitchesAverage: Scalars["Float"];
    runsCompleted: Scalars["Int"];
    runsStarted: Scalars["Int"];
};

export type StatsRoutineEdge = Edge<StatsRoutine, "StatsRoutineEdge">;

export type StatsRoutineSearchInput = BaseSearchInput<StatsRoutineSortBy> & {
    periodTimeFrame?: InputMaybe<TimeFrame>;
    periodType: StatPeriodType;
    searchString?: InputMaybe<Scalars["String"]>;
};

export type StatsRoutineSearchResult = SearchResult<StatsRoutineEdge, "StatsRoutineSearchResult">;

export enum StatsRoutineSortBy {
    PeriodStartAsc = "PeriodStartAsc",
    PeriodStartDesc = "PeriodStartDesc"
}

export type StatsSite = DbObject<"StatsSite"> & {
    activeUsers: Scalars["Int"];
    apiCalls: Scalars["Int"];
    apisCreated: Scalars["Int"];
    codeCalls: Scalars["Int"];
    codeCompletionTimeAverage: Scalars["Float"];
    codesCompleted: Scalars["Int"];
    codesCreated: Scalars["Int"];
    periodEnd: Scalars["Date"];
    periodStart: Scalars["Date"];
    periodType: StatPeriodType;
    projectCompletionTimeAverage: Scalars["Float"];
    projectsCompleted: Scalars["Int"];
    projectsCreated: Scalars["Int"];
    quizzesCompleted: Scalars["Int"];
    quizzesCreated: Scalars["Int"];
    routineCompletionTimeAverage: Scalars["Float"];
    routineComplexityAverage: Scalars["Float"];
    routineSimplicityAverage: Scalars["Float"];
    routinesCompleted: Scalars["Int"];
    routinesCreated: Scalars["Int"];
    runProjectCompletionTimeAverage: Scalars["Float"];
    runProjectContextSwitchesAverage: Scalars["Float"];
    runProjectsCompleted: Scalars["Int"];
    runProjectsStarted: Scalars["Int"];
    runRoutineCompletionTimeAverage: Scalars["Float"];
    runRoutineContextSwitchesAverage: Scalars["Float"];
    runRoutinesCompleted: Scalars["Int"];
    runRoutinesStarted: Scalars["Int"];
    standardCompletionTimeAverage: Scalars["Float"];
    standardsCompleted: Scalars["Int"];
    standardsCreated: Scalars["Int"];
    teamsCreated: Scalars["Int"];
    verifiedEmailsCreated: Scalars["Int"];
    verifiedWalletsCreated: Scalars["Int"];
};

export type StatsSiteEdge = Edge<StatsSite, "StatsSiteEdge">;

export type StatsSiteSearchInput = BaseSearchInput<StatsSiteSortBy> & {
    periodTimeFrame?: InputMaybe<TimeFrame>;
    periodType: StatPeriodType;
    searchString?: InputMaybe<Scalars["String"]>;
};

export type StatsSiteSearchResult = SearchResult<StatsSiteEdge, "StatsSiteSearchResult">;

export enum StatsSiteSortBy {
    PeriodStartAsc = "PeriodStartAsc",
    PeriodStartDesc = "PeriodStartDesc"
}

export type StatsStandard = DbObject<"StatsStandard"> & {
    linksToInputs: Scalars["Int"];
    linksToOutputs: Scalars["Int"];
    periodEnd: Scalars["Date"];
    periodStart: Scalars["Date"];
    periodType: StatPeriodType;
};

export type StatsStandardEdge = Edge<StatsStandard, "StatsStandardEdge">;

export type StatsStandardSearchInput = BaseSearchInput<StatsStandardSortBy> & {
    periodTimeFrame?: InputMaybe<TimeFrame>;
    periodType: StatPeriodType;
    searchString?: InputMaybe<Scalars["String"]>;
};

export type StatsStandardSearchResult = SearchResult<StatsStandardEdge, "StatsStandardSearchResult">;

export enum StatsStandardSortBy {
    PeriodStartAsc = "PeriodStartAsc",
    PeriodStartDesc = "PeriodStartDesc"
}

export type StatsTeam = DbObject<"StatsTeam"> & {
    apis: Scalars["Int"];
    codes: Scalars["Int"];
    members: Scalars["Int"];
    notes: Scalars["Int"];
    periodEnd: Scalars["Date"];
    periodStart: Scalars["Date"];
    periodType: StatPeriodType;
    projects: Scalars["Int"];
    routines: Scalars["Int"];
    runRoutineCompletionTimeAverage: Scalars["Float"];
    runRoutineContextSwitchesAverage: Scalars["Float"];
    runRoutinesCompleted: Scalars["Int"];
    runRoutinesStarted: Scalars["Int"];
    standards: Scalars["Int"];
};

export type StatsTeamEdge = Edge<StatsTeam, "StatsTeamEdge">;

export type StatsTeamSearchInput = BaseSearchInput<StatsTeamSortBy> & {
    periodTimeFrame?: InputMaybe<TimeFrame>;
    periodType: StatPeriodType;
    searchString?: InputMaybe<Scalars["String"]>;
};

export type StatsTeamSearchResult = SearchResult<StatsTeamEdge, "StatsTeamSearchResult">;

export enum StatsTeamSortBy {
    PeriodStartAsc = "PeriodStartAsc",
    PeriodStartDesc = "PeriodStartDesc"
}

export type StatsUser = DbObject<"StatsUser"> & {
    apisCreated: Scalars["Int"];
    codeCompletionTimeAverage: Scalars["Float"];
    codesCompleted: Scalars["Int"];
    codesCreated: Scalars["Int"];
    periodEnd: Scalars["Date"];
    periodStart: Scalars["Date"];
    periodType: StatPeriodType;
    projectCompletionTimeAverage: Scalars["Float"];
    projectsCompleted: Scalars["Int"];
    projectsCreated: Scalars["Int"];
    quizzesFailed: Scalars["Int"];
    quizzesPassed: Scalars["Int"];
    routineCompletionTimeAverage: Scalars["Float"];
    routinesCompleted: Scalars["Int"];
    routinesCreated: Scalars["Int"];
    runProjectCompletionTimeAverage: Scalars["Float"];
    runProjectContextSwitchesAverage: Scalars["Float"];
    runProjectsCompleted: Scalars["Int"];
    runProjectsStarted: Scalars["Int"];
    runRoutineCompletionTimeAverage: Scalars["Float"];
    runRoutineContextSwitchesAverage: Scalars["Float"];
    runRoutinesCompleted: Scalars["Int"];
    runRoutinesStarted: Scalars["Int"];
    standardCompletionTimeAverage: Scalars["Float"];
    standardsCompleted: Scalars["Int"];
    standardsCreated: Scalars["Int"];
    teamsCreated: Scalars["Int"];
};

export type StatsUserEdge = Edge<StatsUser, "StatsUserEdge">;

export type StatsUserSearchInput = BaseSearchInput<StatsUserSortBy> & {
    periodTimeFrame?: InputMaybe<TimeFrame>;
    periodType: StatPeriodType;
    searchString?: InputMaybe<Scalars["String"]>;
};

export type StatsUserSearchResult = SearchResult<StatsUserEdge, "StatsUserSearchResult">;

export enum StatsUserSortBy {
    PeriodStartAsc = "PeriodStartAsc",
    PeriodStartDesc = "PeriodStartDesc"
}

export enum SubscribableObject {
    Api = "Api",
    Code = "Code",
    Comment = "Comment",
    Issue = "Issue",
    Meeting = "Meeting",
    Note = "Note",
    Project = "Project",
    PullRequest = "PullRequest",
    Question = "Question",
    Quiz = "Quiz",
    Report = "Report",
    Routine = "Routine",
    Schedule = "Schedule",
    Standard = "Standard",
    Team = "Team"
}

export type SubscribedObject = Api | Code | Comment | Issue | Meeting | Note | Project | PullRequest | Question | Quiz | Report | Routine | Schedule | Standard | Team;

export type Success = {
    __typename: "Success";
    success: Scalars["Boolean"];
};

export type SwitchCurrentAccountInput = {
    id: Scalars["ID"];
};

export type Tag = DbObject<"Tag"> & {
    apis: Array<Api>;
    bookmarkedBy: Array<User>;
    bookmarks: Scalars["Int"];
    codes: Array<Code>;
    created_at: Scalars["Date"];
    notes: Array<Note>;
    posts: Array<Post>;
    projects: Array<Project>;
    reports: Array<Report>;
    routines: Array<Routine>;
    standards: Array<Standard>;
    tag: Scalars["String"];
    teams: Array<Team>;
    translations: Array<TagTranslation>;
    updated_at: Scalars["Date"];
    you: TagYou;
};

export type TagCreateInput = BaseTranslatableCreateInput<TagTranslationCreateInput> & {
    anonymous?: InputMaybe<Scalars["Boolean"]>;
    id: Scalars["ID"];
    tag: Scalars["String"];
};

export type TagEdge = Edge<Tag, "TagEdge">;

export type TagSearchInput = BaseSearchInput<TagSortBy> & {
    createdById?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    maxBookmarks?: InputMaybe<Scalars["Int"]>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    offset?: InputMaybe<Scalars["Int"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type TagSearchResult = SearchResult<TagEdge, "TagSearchResult">;

export enum TagSortBy {
    EmbedDateCreatedAsc = "EmbedDateCreatedAsc",
    EmbedDateCreatedDesc = "EmbedDateCreatedDesc",
    EmbedDateUpdatedAsc = "EmbedDateUpdatedAsc",
    EmbedDateUpdatedDesc = "EmbedDateUpdatedDesc",
    EmbedTopAsc = "EmbedTopAsc",
    EmbedTopDesc = "EmbedTopDesc"
}

export type TagTranslation = BaseTranslation<"TagTranslation"> & {
    description?: Maybe<Scalars["String"]>;
};

export type TagTranslationCreateInput = BaseTranslationCreateInput & {
    description?: InputMaybe<Scalars["String"]>;
};

export type TagTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
};

export type TagUpdateInput = BaseTranslatableUpdateInput<TagTranslationCreateInput, TagTranslationUpdateInput> & {
    anonymous?: InputMaybe<Scalars["Boolean"]>;
    id: Scalars["ID"];
    tag: Scalars["String"];
};

export type TagYou = {
    __typename: "TagYou";
    isBookmarked: Scalars["Boolean"];
    isOwn: Scalars["Boolean"];
};

export type TaskContextInfoInput = {
    data: Scalars["JSONObject"];
    id: Scalars["ID"];
    label: Scalars["String"];
    template?: InputMaybe<Scalars["String"]>;
    templateVariables?: InputMaybe<TaskContextInfoTemplateVariablesInput>;
};

export type TaskContextInfoTemplateVariablesInput = {
    data?: InputMaybe<Scalars["String"]>;
    task?: InputMaybe<Scalars["String"]>;
};

export enum TaskStatus {
    Canceling = "Canceling",
    Completed = "Completed",
    Failed = "Failed",
    Paused = "Paused",
    Running = "Running",
    Scheduled = "Scheduled",
    Suggested = "Suggested"
}

export type TaskStatusInfo = {
    __typename: "TaskStatusInfo";
    id: Scalars["ID"];
    status?: Maybe<TaskStatus>;
};

export enum TaskType {
    Llm = "Llm",
    Run = "Run",
    Sandbox = "Sandbox"
}

export type Team = DbObject<"Team"> & {
    apis: Array<Api>;
    apisCount: Scalars["Int"];
    bannerImage?: Maybe<Scalars["String"]>;
    bookmarkedBy: Array<User>;
    bookmarks: Scalars["Int"];
    codes: Array<Code>;
    codesCount: Scalars["Int"];
    comments: Array<Comment>;
    commentsCount: Scalars["Int"];
    created_at: Scalars["Date"];
    directoryListings: Array<ProjectVersionDirectory>;
    forks: Array<Team>;
    handle?: Maybe<Scalars["String"]>;
    isOpenToNewMembers: Scalars["Boolean"];
    isPrivate: Scalars["Boolean"];
    issues: Array<Issue>;
    issuesCount: Scalars["Int"];
    labels: Array<Label>;
    labelsCount: Scalars["Int"];
    meetings: Array<Meeting>;
    meetingsCount: Scalars["Int"];
    members: Array<Member>;
    membersCount: Scalars["Int"];
    notes: Array<Note>;
    notesCount: Scalars["Int"];
    parent?: Maybe<Team>;
    paymentHistory: Array<Payment>;
    permissions: Scalars["String"];
    posts: Array<Post>;
    postsCount: Scalars["Int"];
    premium?: Maybe<Premium>;
    profileImage?: Maybe<Scalars["String"]>;
    projects: Array<Project>;
    projectsCount: Scalars["Int"];
    questions: Array<Question>;
    questionsCount: Scalars["Int"];
    reports: Array<Report>;
    reportsCount: Scalars["Int"];
    resourceList?: Maybe<ResourceList>;
    roles?: Maybe<Array<Role>>;
    rolesCount: Scalars["Int"];
    routines: Array<Routine>;
    routinesCount: Scalars["Int"];
    standards: Array<Standard>;
    standardsCount: Scalars["Int"];
    stats: Array<StatsTeam>;
    tags: Array<Tag>;
    transfersIncoming: Array<Transfer>;
    transfersOutgoing: Array<Transfer>;
    translatedName: Scalars["String"];
    translations: Array<TeamTranslation>;
    translationsCount: Scalars["Int"];
    updated_at: Scalars["Date"];
    views: Scalars["Int"];
    wallets: Array<Wallet>;
    you: TeamYou;
};

export type TeamCreateInput = BaseTranslatableCreateInput<TeamTranslationCreateInput> & {
    bannerImage?: InputMaybe<Scalars["Upload"]>;
    handle?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    isOpenToNewMembers?: InputMaybe<Scalars["Boolean"]>;
    isPrivate: Scalars["Boolean"];
    memberInvitesCreate?: InputMaybe<Array<MemberInviteCreateInput>>;
    permissions?: InputMaybe<Scalars["String"]>;
    profileImage?: InputMaybe<Scalars["Upload"]>;
    resourceListCreate?: InputMaybe<ResourceListCreateInput>;
    rolesCreate?: InputMaybe<Array<RoleCreateInput>>;
    tagsConnect?: InputMaybe<Array<Scalars["String"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
};

export type TeamEdge = Edge<Team, "TeamEdge">;

export type TeamSearchInput = BaseSearchInput<TeamSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    isOpenToNewMembers?: InputMaybe<Scalars["Boolean"]>;
    maxBookmarks?: InputMaybe<Scalars["Int"]>;
    maxViews?: InputMaybe<Scalars["Int"]>;
    memberUserIds?: InputMaybe<Array<Scalars["ID"]>>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    minViews?: InputMaybe<Scalars["Int"]>;
    projectId?: InputMaybe<Scalars["ID"]>;
    reportId?: InputMaybe<Scalars["ID"]>;
    routineId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    standardId?: InputMaybe<Scalars["ID"]>;
    tags?: InputMaybe<Array<Scalars["String"]>>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type TeamSearchResult = SearchResult<TeamEdge, "TeamSearchResult">;

export enum TeamSortBy {
    BookmarksAsc = "BookmarksAsc",
    BookmarksDesc = "BookmarksDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc"
}

export type TeamTranslation = BaseTranslation<"TeamTranslation"> & {
    bio?: Maybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type TeamTranslationCreateInput = BaseTranslationCreateInput & {
    bio?: InputMaybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type TeamTranslationUpdateInput = BaseTranslationUpdateInput & {
    bio?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type TeamUpdateInput = BaseTranslatableUpdateInput<TeamTranslationCreateInput, TeamTranslationUpdateInput> & {
    bannerImage?: InputMaybe<Scalars["Upload"]>;
    handle?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    isOpenToNewMembers?: InputMaybe<Scalars["Boolean"]>;
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    memberInvitesCreate?: InputMaybe<Array<MemberInviteCreateInput>>;
    memberInvitesDelete?: InputMaybe<Array<Scalars["ID"]>>;
    membersDelete?: InputMaybe<Array<Scalars["ID"]>>;
    permissions?: InputMaybe<Scalars["String"]>;
    profileImage?: InputMaybe<Scalars["Upload"]>;
    resourceListCreate?: InputMaybe<ResourceListCreateInput>;
    resourceListUpdate?: InputMaybe<ResourceListUpdateInput>;
    rolesCreate?: InputMaybe<Array<RoleCreateInput>>;
    rolesDelete?: InputMaybe<Array<Scalars["ID"]>>;
    rolesUpdate?: InputMaybe<Array<RoleUpdateInput>>;
    tagsConnect?: InputMaybe<Array<Scalars["String"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
    tagsDisconnect?: InputMaybe<Array<Scalars["String"]>>;
};

export type TeamYou = {
    __typename: "TeamYou";
    canAddMembers: Scalars["Boolean"];
    canBookmark: Scalars["Boolean"];
    canDelete: Scalars["Boolean"];
    canRead: Scalars["Boolean"];
    canReport: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
    isBookmarked: Scalars["Boolean"];
    isViewed: Scalars["Boolean"];
    yourMembership?: Maybe<Member>;
};

export type TimeFrame = {
    after?: InputMaybe<Scalars["Date"]>;
    before?: InputMaybe<Scalars["Date"]>;
};

export type Transfer = DbObject<"Transfer"> & {
    created_at: Scalars["Date"];
    fromOwner?: Maybe<Owner>;
    mergedOrRejectedAt?: Maybe<Scalars["Date"]>;
    object: TransferObject;
    status: TransferStatus;
    toOwner?: Maybe<Owner>;
    updated_at: Scalars["Date"];
    you: TransferYou;
};

export type TransferDenyInput = {
    id: Scalars["ID"];
    reason?: InputMaybe<Scalars["String"]>;
};

export type TransferEdge = Edge<Transfer, "TransferEdge">;

export type TransferObject = Api | Code | Note | Project | Routine | Standard;

export enum TransferObjectType {
    Api = "Api",
    Code = "Code",
    Note = "Note",
    Project = "Project",
    Routine = "Routine",
    Standard = "Standard"
}

export type TransferRequestReceiveInput = {
    message?: InputMaybe<Scalars["String"]>;
    objectConnect: Scalars["ID"];
    objectType: TransferObjectType;
    toTeamConnect?: InputMaybe<Scalars["ID"]>;
};

export type TransferRequestSendInput = {
    message?: InputMaybe<Scalars["String"]>;
    objectConnect: Scalars["ID"];
    objectType: TransferObjectType;
    toTeamConnect?: InputMaybe<Scalars["ID"]>;
    toUserConnect?: InputMaybe<Scalars["ID"]>;
};

export type TransferSearchInput = BaseSearchInput<TransferSortBy> & {
    apiId?: InputMaybe<Scalars["ID"]>;
    codeId?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    fromTeamId?: InputMaybe<Scalars["ID"]>;
    noteId?: InputMaybe<Scalars["ID"]>;
    projectId?: InputMaybe<Scalars["ID"]>;
    routineId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    standardId?: InputMaybe<Scalars["ID"]>;
    status?: InputMaybe<TransferStatus>;
    toTeamId?: InputMaybe<Scalars["ID"]>;
    toUserId?: InputMaybe<Scalars["ID"]>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type TransferSearchResult = SearchResult<TransferEdge, "TransferSearchResult">;

export enum TransferSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc"
}

export enum TransferStatus {
    Accepted = "Accepted",
    Denied = "Denied",
    Pending = "Pending"
}

export type TransferUpdateInput = {
    id: Scalars["ID"];
    message?: InputMaybe<Scalars["String"]>;
};

export type TransferYou = {
    __typename: "TransferYou";
    canDelete: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
};

export type Translate = {
    __typename: "Translate";
    fields: Scalars["String"];
    language: Scalars["String"];
};

export type TranslateInput = {
    fields: Scalars["String"];
    languageSource: Scalars["String"];
    languageTarget: Scalars["String"];
};

export type User = DbObject<"User"> & {
    apiKeys?: Maybe<Array<ApiKey>>;
    apiKeysExternal?: Maybe<Array<ApiKeyExternal>>;
    apis: Array<Api>;
    apisCount: Scalars["Int"];
    apisCreated?: Maybe<Array<Api>>;
    awards?: Maybe<Array<Award>>;
    bannerImage?: Maybe<Scalars["String"]>;
    bookmarked?: Maybe<Array<Bookmark>>;
    bookmarkedBy: Array<User>;
    bookmarks: Scalars["Int"];
    botSettings?: Maybe<Scalars["String"]>;
    codes?: Maybe<Array<Code>>;
    codesCount: Scalars["Int"];
    codesCreated?: Maybe<Array<Code>>;
    comments?: Maybe<Array<Comment>>;
    created_at: Scalars["Date"];
    emails?: Maybe<Array<Email>>;
    focusModes?: Maybe<Array<FocusMode>>;
    handle?: Maybe<Scalars["String"]>;
    invitedByUser?: Maybe<User>;
    invitedUsers?: Maybe<Array<User>>;
    isBot: Scalars["Boolean"];
    isBotDepictingPerson: Scalars["Boolean"];
    isPrivate: Scalars["Boolean"];
    isPrivateApis: Scalars["Boolean"];
    isPrivateApisCreated: Scalars["Boolean"];
    isPrivateBookmarks: Scalars["Boolean"];
    isPrivateCodes: Scalars["Boolean"];
    isPrivateCodesCreated: Scalars["Boolean"];
    isPrivateMemberships: Scalars["Boolean"];
    isPrivateProjects: Scalars["Boolean"];
    isPrivateProjectsCreated: Scalars["Boolean"];
    isPrivatePullRequests: Scalars["Boolean"];
    isPrivateQuestionsAnswered: Scalars["Boolean"];
    isPrivateQuestionsAsked: Scalars["Boolean"];
    isPrivateQuizzesCreated: Scalars["Boolean"];
    isPrivateRoles: Scalars["Boolean"];
    isPrivateRoutines: Scalars["Boolean"];
    isPrivateRoutinesCreated: Scalars["Boolean"];
    isPrivateStandards: Scalars["Boolean"];
    isPrivateStandardsCreated: Scalars["Boolean"];
    isPrivateTeamsCreated: Scalars["Boolean"];
    isPrivateVotes: Scalars["Boolean"];
    issuesClosed?: Maybe<Array<Issue>>;
    issuesCreated?: Maybe<Array<Issue>>;
    labels?: Maybe<Array<Label>>;
    meetingsAttending?: Maybe<Array<Meeting>>;
    meetingsInvited?: Maybe<Array<MeetingInvite>>;
    memberships?: Maybe<Array<Member>>;
    membershipsCount: Scalars["Int"];
    membershipsInvited?: Maybe<Array<MemberInvite>>;
    name: Scalars["String"];
    notes?: Maybe<Array<Note>>;
    notesCount: Scalars["Int"];
    notesCreated?: Maybe<Array<Note>>;
    notificationSettings?: Maybe<Scalars["String"]>;
    notificationSubscriptions?: Maybe<Array<NotificationSubscription>>;
    notifications?: Maybe<Array<Notification>>;
    paymentHistory?: Maybe<Array<Payment>>;
    phones?: Maybe<Array<Phone>>;
    premium?: Maybe<Premium>;
    profileImage?: Maybe<Scalars["String"]>;
    projects?: Maybe<Array<Project>>;
    projectsCount: Scalars["Int"];
    projectsCreated?: Maybe<Array<Project>>;
    pullRequests?: Maybe<Array<PullRequest>>;
    pushDevices?: Maybe<Array<PushDevice>>;
    questionsAnswered?: Maybe<Array<QuestionAnswer>>;
    questionsAsked?: Maybe<Array<Question>>;
    questionsAskedCount: Scalars["Int"];
    quizzesCreated?: Maybe<Array<Quiz>>;
    quizzesTaken?: Maybe<Array<Quiz>>;
    reacted?: Maybe<Array<Reaction>>;
    reportResponses?: Maybe<Array<ReportResponse>>;
    reportsCreated?: Maybe<Array<Report>>;
    reportsReceived: Array<Report>;
    reportsReceivedCount: Scalars["Int"];
    reputationHistory?: Maybe<Array<ReputationHistory>>;
    roles?: Maybe<Array<Role>>;
    routines?: Maybe<Array<Routine>>;
    routinesCount: Scalars["Int"];
    routinesCreated?: Maybe<Array<Routine>>;
    runProjects?: Maybe<Array<RunProject>>;
    runRoutines?: Maybe<Array<RunRoutine>>;
    sentReports?: Maybe<Array<Report>>;
    standards?: Maybe<Array<Standard>>;
    standardsCount: Scalars["Int"];
    standardsCreated?: Maybe<Array<Standard>>;
    status?: Maybe<AccountStatus>;
    tags?: Maybe<Array<Tag>>;
    teamsCreated?: Maybe<Array<Team>>;
    theme?: Maybe<Scalars["String"]>;
    transfersIncoming?: Maybe<Array<Transfer>>;
    transfersOutgoing?: Maybe<Array<Transfer>>;
    translationLanguages?: Maybe<Array<Scalars["String"]>>;
    translations: Array<UserTranslation>;
    updated_at?: Maybe<Scalars["Date"]>;
    viewed?: Maybe<Array<View>>;
    viewedBy?: Maybe<Array<View>>;
    views: Scalars["Int"];
    wallets?: Maybe<Array<Wallet>>;
    you: UserYou;
};

export type UserEdge = Edge<User, "UserEdge">;

export type UserSearchInput = BaseSearchInput<UserSortBy> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    isBot?: InputMaybe<Scalars["Boolean"]>;
    isBotDepictingPerson?: InputMaybe<Scalars["Boolean"]>;
    maxBookmarks?: InputMaybe<Scalars["Int"]>;
    maxViews?: InputMaybe<Scalars["Int"]>;
    memberInTeamId?: InputMaybe<Scalars["ID"]>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    minViews?: InputMaybe<Scalars["Int"]>;
    notInChatId?: InputMaybe<Scalars["ID"]>;
    notInvitedToTeamId?: InputMaybe<Scalars["ID"]>;
    notMemberInTeamId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type UserSearchResult = SearchResult<UserEdge, "UserSearchResult">;

export enum UserSortBy {
    BookmarksAsc = "BookmarksAsc",
    BookmarksDesc = "BookmarksDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc"
}

export type UserTranslation = BaseTranslation<"UserTranslation"> & {
    bio?: Maybe<Scalars["String"]>;
};

export type UserTranslationCreateInput = BaseTranslationCreateInput & {
    bio?: InputMaybe<Scalars["String"]>;
};

export type UserTranslationUpdateInput = BaseTranslationUpdateInput & {
    bio?: InputMaybe<Scalars["String"]>;
};

export type UserYou = {
    __typename: "UserYou";
    canDelete: Scalars["Boolean"];
    canReport: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
    isBookmarked: Scalars["Boolean"];
    isViewed: Scalars["Boolean"];
};

export type ValidateSessionInput = {
    timeZone: Scalars["String"];
};

export type ValidateVerificationTextInput = {
    phoneNumber: Scalars["String"];
    verificationCode: Scalars["String"];
};

export type VersionYou = {
    __typename: "VersionYou";
    canComment: Scalars["Boolean"];
    canCopy: Scalars["Boolean"];
    canDelete: Scalars["Boolean"];
    canRead: Scalars["Boolean"];
    canReport: Scalars["Boolean"];
    canUpdate: Scalars["Boolean"];
    canUse: Scalars["Boolean"];
};

export type View = DbObject<"View"> & {
    by: User;
    lastViewedAt: Scalars["Date"];
    name: Scalars["String"];
    to: ViewTo;
};

export type ViewEdge = Edge<View, "ViewEdge">;

export type ViewSearchInput = BaseSearchInput<ViewSortBy> & {
    lastViewedTimeFrame?: InputMaybe<TimeFrame>;
    searchString?: InputMaybe<Scalars["String"]>;
};

export type ViewSearchResult = SearchResult<ViewEdge, "ViewSearchResult">;

export enum ViewSortBy {
    LastViewedAsc = "LastViewedAsc",
    LastViewedDesc = "LastViewedDesc"
}

export type ViewTo = Api | Code | Issue | Note | Post | Project | Question | Routine | Standard | Team | User;

export enum VisibilityType {
    Own = "Own",
    OwnOrPublic = "OwnOrPublic",
    OwnPrivate = "OwnPrivate",
    OwnPublic = "OwnPublic",
    Public = "Public"
}

export type Wallet = DbObject<"Wallet"> & {
    name?: Maybe<Scalars["String"]>;
    publicAddress?: Maybe<Scalars["String"]>;
    stakingAddress: Scalars["String"];
    team?: Maybe<Team>;
    user?: Maybe<User>;
    verified: Scalars["Boolean"];
};

export type WalletInit = {
    __typename: "WalletInit";
    nonce: Scalars["String"];
};

export type WalletComplete = {
    __typename: "WalletComplete";
    firstLogIn: Scalars["Boolean"];
    session?: Maybe<Session>;
    wallet?: Maybe<Wallet>;
};

export type WalletCompleteInput = {
    signedPayload: Scalars["String"];
    stakingAddress: Scalars["String"];
};

export type WalletInitInput = {
    nonceDescription?: InputMaybe<Scalars["String"]>;
    stakingAddress: Scalars["String"];
};

export type WalletUpdateInput = {
    id: Scalars["ID"];
    name?: InputMaybe<Scalars["String"]>;
};

export type WriteAssetsInput = {
    files: Array<Scalars["Upload"]>;
};
