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

export type Award = DbObject<"Award"> & {
    category: AwardCategory;
    createdAt: Scalars["Date"];
    description?: Maybe<Scalars["String"]>;
    progress: Scalars["Int"];
    tierCompletedAt?: Maybe<Scalars["Date"]>;
    title?: Maybe<Scalars["String"]>;
    updatedAt: Scalars["Date"];
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
    ProjectCreate = "ProjectCreate",
    PullRequestComplete = "PullRequestComplete",
    PullRequestCreate = "PullRequestCreate",
    ReportContribute = "ReportContribute",
    ReportEnd = "ReportEnd",
    Reputation = "Reputation",
    RoutineCreate = "RoutineCreate",
    Run = "Run",
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
    createdAt: Scalars["Date"];
    list: BookmarkList;
    to: BookmarkTo;
    updatedAt: Scalars["Date"];
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
    Comment = "Comment",
    Issue = "Issue",
    Resource = "Resource",
    Tag = "Tag",
    Team = "Team",
    User = "User"
}

export type BookmarkList = DbObject<"BookmarkList"> & {
    bookmarks: Array<Bookmark>;
    bookmarksCount: Scalars["Int"];
    createdAt: Scalars["Date"];
    label: Scalars["String"];
    updatedAt: Scalars["Date"];
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
    commentId?: InputMaybe<Scalars["ID"]>;
    excludeLinkedToTag?: InputMaybe<Scalars["Boolean"]>;
    issueId?: InputMaybe<Scalars["ID"]>;
    limitTo?: InputMaybe<Array<BookmarkFor>>;
    listId?: InputMaybe<Scalars["ID"]>;
    listLabel?: InputMaybe<Scalars["String"]>;
    resourceId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
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

export type BookmarkTo = Comment | Issue | Resource | Tag | Team | User;

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
    createdAt: Scalars["Date"];
    invites: Array<ChatInvite>;
    invitesCount: Scalars["Int"];
    messages: Array<ChatMessage>;
    openToAnyoneWithInvite: Scalars["Boolean"];
    participants: Array<ChatParticipant>;
    participantsCount: Scalars["Int"];
    publicId: Scalars["String"];
    team?: Maybe<Team>;
    translations: Array<ChatTranslation>;
    translationsCount: Scalars["Int"];
    updatedAt: Scalars["Date"];
    you: ChatYou;
};

export type ChatCreateInput = BaseTranslatableCreateInput<ChatTranslationCreateInput> & {
    id: Scalars["ID"];
    invitesCreate?: InputMaybe<Array<ChatInviteCreateInput>>;
    messagesCreate?: InputMaybe<Array<ChatMessageCreateInput>>;
    openToAnyoneWithInvite?: InputMaybe<Scalars["Boolean"]>;
    task?: InputMaybe<Scalars["String"]>;
    teamConnect?: InputMaybe<Scalars["ID"]>;
};

export type ChatEdge = Edge<Chat, "ChatEdge">;

export type ChatInvite = DbObject<"ChatInvite"> & {
    chat: Chat;
    createdAt: Scalars["Date"];
    message?: Maybe<Scalars["String"]>;
    status: ChatInviteStatus;
    updatedAt: Scalars["Date"];
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
    createdAt: Scalars["Date"];
    language: Scalars["String"];
    parent?: Maybe<ChatMessageParent>;
    reactionSummaries: Array<ReactionSummary>;
    reports: Array<Report>;
    reportsCount: Scalars["Int"];
    score: Scalars["Int"];
    sequence: Scalars["Int"];
    text: Scalars["String"];
    updatedAt: Scalars["Date"];
    user: User;
    versionIndex: Scalars["Int"];
    you: ChatMessageYou;
};

export type ChatMessageCreateInput = {
    chatConnect: Scalars["ID"];
    id: Scalars["ID"];
    language: Scalars["String"];
    parentConnect?: InputMaybe<Scalars["ID"]>;
    text: Scalars["String"];
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
    createdAt: Scalars["Date"];
    id: Scalars["ID"];
};

export type ChatMessageSearchInput = Omit<BaseSearchInput<ChatMessageSortBy>, "ids"> & {
    chatId: Scalars["ID"];
    createdTimeFrame?: InputMaybe<TimeFrame>;
    languages?: InputMaybe<Array<Scalars["String"]>>;
    minScore?: InputMaybe<Scalars["Int"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    userId?: InputMaybe<Scalars["ID"]>;
};

export type ChatMessageSearchResult = SearchResult<ChatMessageEdge, "ChatMessageSearchResult">;

export type ChatMessageSearchTreeInput = {
    chatId: Scalars["ID"];
    excludeDown?: InputMaybe<Scalars["Boolean"]>;
    excludeUp?: InputMaybe<Scalars["Boolean"]>;
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

export type ChatMessageUpdateInput = {
    id: Scalars["ID"];
    text?: InputMaybe<Scalars["String"]>;
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

export type ChatMessageedOn = Issue | PullRequest | ResourceVersion;

export type ChatParticipant = DbObject<"ChatParticipant"> & {
    chat: Chat;
    createdAt: Scalars["Date"];
    updatedAt: Scalars["Date"];
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
    messagesCreate?: InputMaybe<Array<ChatMessageCreateInput>>;
    messagesDelete?: InputMaybe<Array<Scalars["ID"]>>;
    messagesUpdate?: InputMaybe<Array<ChatMessageUpdateInput>>;
    openToAnyoneWithInvite?: InputMaybe<Scalars["Boolean"]>;
    participantsDelete?: InputMaybe<Array<Scalars["ID"]>>;
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

export type Comment = DbObject<"Comment"> & {
    bookmarkedBy?: Maybe<Array<User>>;
    bookmarks: Scalars["Int"];
    commentedOn: CommentedOn;
    createdAt: Scalars["Date"];
    owner?: Maybe<Owner>;
    reports: Array<Report>;
    reportsCount: Scalars["Int"];
    score: Scalars["Int"];
    translations: Array<CommentTranslation>;
    translationsCount: Scalars["Int"];
    updatedAt: Scalars["Date"];
    you: CommentYou;
};

export type CommentCreateInput = BaseTranslatableCreateInput<CommentTranslationCreateInput> & {
    createdFor: CommentFor;
    forConnect: Scalars["ID"];
    id: Scalars["ID"];
    parentConnect?: InputMaybe<Scalars["ID"]>;
};

export enum CommentFor {
    Issue = "Issue",
    PullRequest = "PullRequest",
    ResourceVersion = "ResourceVersion",
}

export type CommentSearchInput = Omit<BaseSearchInput<CommentSortBy>, "ids"> & {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    issueId?: InputMaybe<Scalars["ID"]>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    minScore?: InputMaybe<Scalars["Int"]>;
    ownedByTeamId?: InputMaybe<Scalars["ID"]>;
    ownedByUserId?: InputMaybe<Scalars["ID"]>;
    pullRequestId?: InputMaybe<Scalars["ID"]>;
    resourceVersionId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
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

export type CommentedOn = Issue | PullRequest | ResourceVersion;

export type CopyInput = {
    id: Scalars["ID"];
    intendToPullRequest: Scalars["Boolean"];
    objectType: CopyType;
};

export type CopyResult = {
    __typename: "CopyResult";
    resourceVersion?: Maybe<ResourceVersion>;
    team?: Maybe<Team>;
};

export enum CopyType {
    Resource = "Resource",
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
    ApiKey = "ApiKey",
    ApiKeyExternal = "ApiKeyExternal",
    Bookmark = "Bookmark",
    BookmarkList = "BookmarkList",
    Chat = "Chat",
    ChatInvite = "ChatInvite",
    ChatMessage = "ChatMessage",
    ChatParticipant = "ChatParticipant",
    Comment = "Comment",
    Email = "Email",
    Issue = "Issue",
    Meeting = "Meeting",
    MeetingInvite = "MeetingInvite",
    Member = "Member",
    MemberInvite = "MemberInvite",
    Notification = "Notification",
    Phone = "Phone",
    PullRequest = "PullRequest",
    PushDevice = "PushDevice",
    Reminder = "Reminder",
    ReminderList = "ReminderList",
    Report = "Report",
    Resource = "Resource",
    ResourceVersion = "ResourceVersion",
    Run = "Run",
    Schedule = "Schedule",
    Team = "Team",
    Transfer = "Transfer",
    User = "User",
    Wallet = "Wallet"
}

export type Email = {
    __typename: "Email";
    emailAddress: Scalars["String"];
    id: Scalars["ID"];
    verifiedAt: Scalars["Date"];
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
    id?: InputMaybe<Scalars["ID"]>;
    publicId?: InputMaybe<Scalars["String"]>;
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

export type FindByPublicIdInput = {
    publicId: Scalars["String"];
};

export type FindVersionInput = {
    publicId: Scalars["String"];
    versionLabel?: InputMaybe<Scalars["String"]>;
};

export enum ModelType {
    ApiKey = "ApiKey",
    ApiKeyExternal = "ApiKeyExternal",
    Award = "Award",
    Bookmark = "Bookmark",
    BookmarkList = "BookmarkList",
    Chat = "Chat",
    ChatInvite = "ChatInvite",
    ChatMessage = "ChatMessage",
    ChatMessageSearchTreeResult = "ChatMessageSearchTreeResult",
    ChatParticipant = "ChatParticipant",
    Comment = "Comment",
    Copy = "Copy",
    Email = "Email",
    Fork = "Fork",
    Handle = "Handle",
    HomeResult = "HomeResult",
    Issue = "Issue",
    Meeting = "Meeting",
    MeetingInvite = "MeetingInvite",
    Member = "Member",
    MemberInvite = "MemberInvite",
    Notification = "Notification",
    NotificationSubscription = "NotificationSubscription",
    Payment = "Payment",
    Phone = "Phone",
    PopularResult = "PopularResult",
    Premium = "Premium",
    PullRequest = "PullRequest",
    PushDevice = "PushDevice",
    Reaction = "Reaction",
    ReactionSummary = "ReactionSummary",
    Reminder = "Reminder",
    ReminderItem = "ReminderItem",
    ReminderList = "ReminderList",
    Report = "Report",
    ReportResponse = "ReportResponse",
    ReputationHistory = "ReputationHistory",
    Resource = "Resource",
    ResourceVersion = "ResourceVersion",
    ResourceVersionRelation = "ResourceVersionRelation",
    Run = "Run",
    RunStep = "RunStep",
    RunIO = "RunIO",
    Schedule = "Schedule",
    ScheduleException = "ScheduleException",
    ScheduleRecurrence = "ScheduleRecurrence",
    Session = "Session",
    SessionUser = "SessionUser",
    StatsResource = "StatsResource",
    StatsSite = "StatsSite",
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
    reminders: Array<Reminder>;
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
    createdAt: Scalars["Date"];
    publicId: Scalars["String"];
    referencedVersionId?: Maybe<Scalars["String"]>;
    reports: Array<Report>;
    reportsCount: Scalars["Int"];
    score: Scalars["Int"];
    status: IssueStatus;
    to: IssueTo;
    translations: Array<IssueTranslation>;
    translationsCount: Scalars["Int"];
    updatedAt: Scalars["Date"];
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
    referencedVersionIdConnect?: InputMaybe<Scalars["ID"]>;
};

export type IssueEdge = Edge<Issue, "IssueEdge">;

export enum IssueFor {
    Resource = "Resource",
    Team = "Team"
}

export type IssueSearchInput = BaseSearchInput<IssueSortBy> & {
    closedById?: InputMaybe<Scalars["ID"]>;
    createdById?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    minBookmarks?: InputMaybe<Scalars["Int"]>;
    minScore?: InputMaybe<Scalars["Int"]>;
    minViews?: InputMaybe<Scalars["Int"]>;
    referencedVersionId?: InputMaybe<Scalars["ID"]>;
    resourceId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
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

export type IssueTo = Resource | Team;

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

export enum LlmTask {
    BotAdd = "BotAdd",
    BotDelete = "BotDelete",
    BotFind = "BotFind",
    BotUpdate = "BotUpdate",
    MembersAdd = "MembersAdd",
    MembersDelete = "MembersDelete",
    MembersFind = "MembersFind",
    MembersUpdate = "MembersUpdate",
    ReminderAdd = "ReminderAdd",
    ReminderDelete = "ReminderDelete",
    ReminderFind = "ReminderFind",
    ReminderUpdate = "ReminderUpdate",
    ResourceAdd = "ResourceAdd",
    ResourceDelete = "ResourceDelete",
    ResourceFind = "ResourceFind",
    ResourceUpdate = "ResourceUpdate",
    RunStart = "RunStart",
    ScheduleAdd = "ScheduleAdd",
    ScheduleDelete = "ScheduleDelete",
    ScheduleFind = "ScheduleFind",
    ScheduleUpdate = "ScheduleUpdate",
    Start = "Start",
    TeamAdd = "TeamAdd",
    TeamDelete = "TeamDelete",
    TeamFind = "TeamFind",
    TeamUpdate = "TeamUpdate"
}

export type Meeting = DbObject<"Meeting"> & {
    attendees: Array<User>;
    attendeesCount: Scalars["Int"];
    createdAt: Scalars["Date"];
    invites: Array<MeetingInvite>;
    invitesCount: Scalars["Int"];
    openToAnyoneWithInvite: Scalars["Boolean"];
    publicId: Scalars["String"];
    schedule?: Maybe<Schedule>;
    showOnTeamProfile: Scalars["Boolean"];
    team: Team;
    translations: Array<MeetingTranslation>;
    translationsCount: Scalars["Int"];
    updatedAt: Scalars["Date"];
    you: MeetingYou;
};

export type MeetingCreateInput = BaseTranslatableCreateInput<MeetingTranslationCreateInput> & {
    id: Scalars["ID"];
    invitesCreate?: InputMaybe<Array<MeetingInviteCreateInput>>;
    openToAnyoneWithInvite?: InputMaybe<Scalars["Boolean"]>;
    scheduleCreate?: InputMaybe<ScheduleCreateInput>;
    showOnTeamProfile?: InputMaybe<Scalars["Boolean"]>;
    teamConnect: Scalars["ID"];
};

export type MeetingEdge = Edge<Meeting, "MeetingEdge">;

export type MeetingInvite = DbObject<"MeetingInvite"> & {
    createdAt: Scalars["Date"];
    meeting: Meeting;
    message?: Maybe<Scalars["String"]>;
    status: MeetingInviteStatus;
    updatedAt: Scalars["Date"];
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
    openToAnyoneWithInvite?: InputMaybe<Scalars["Boolean"]>;
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
    createdAt: Scalars["Date"];
    isAdmin: Scalars["Boolean"];
    permissions: Scalars["String"];
    publicId: Scalars["String"];
    team: Team;
    updatedAt: Scalars["Date"];
    user: User;
    you: MemberYou;
};

export type MemberEdge = Edge<Member, "MemberEdge">;

export type MemberInvite = DbObject<"MemberInvite"> & {
    createdAt: Scalars["Date"];
    message?: Maybe<Scalars["String"]>;
    status: MemberInviteStatus;
    team: Team;
    updatedAt: Scalars["Date"];
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

export type Notification = DbObject<"Notification"> & {
    category: Scalars["String"];
    createdAt: Scalars["Date"];
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
    createdAt: Scalars["Date"];
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
    createdAt: Scalars["Date"];
    currency: Scalars["String"];
    description: Scalars["String"];
    paymentMethod: Scalars["String"];
    paymentType: PaymentType;
    status: PaymentStatus;
    team: Team;
    updatedAt: Scalars["Date"];
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
    verifiedAt: Scalars["Date"];
};

export type PhoneCreateInput = {
    id: Scalars["ID"];
    phoneNumber: Scalars["String"];
};

export type Popular = Resource | Team | User;

export type PopularEdge = Edge<Popular, "PopularEdge">;

export enum PopularObjectType {
    Resource = "Resource",
    Team = "Team",
    User = "User"
}

export type PopularPageInfo = {
    __typename: "PopularPageInfo";
    endCursorResource?: Maybe<Scalars["String"]>;
    endCursorTeam?: Maybe<Scalars["String"]>;
    endCursorUser?: Maybe<Scalars["String"]>;
    hasNextPage: Scalars["Boolean"];
};

export type PopularSearchInput = {
    createdTimeFrame?: InputMaybe<TimeFrame>;
    objectType?: InputMaybe<PopularObjectType>;
    resourceAfter?: InputMaybe<Scalars["String"]>;
    searchString?: InputMaybe<Scalars["String"]>;
    sortBy?: InputMaybe<PopularSortBy>;
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

export type Premium = DbObject<"Premium"> & {
    credits: Scalars["Int"];
    customPlan?: Maybe<Scalars["String"]>;
    enabledAt?: Maybe<Scalars["Date"]>;
    expiresAt?: Maybe<Scalars["Date"]>;
};

export type ProfileEmailUpdateInput = {
    currentPassword: Scalars["String"];
    emailsCreate?: InputMaybe<Array<EmailCreateInput>>;
    emailsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    newPassword?: InputMaybe<Scalars["String"]>;
};

export type ProfileUpdateInput = BaseTranslatableUpdateInput<UserTranslationCreateInput, UserTranslationUpdateInput> & {
    bannerImage?: InputMaybe<Scalars["Upload"]>;
    handle?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    isPrivateBookmarks?: InputMaybe<Scalars["Boolean"]>;
    isPrivateMemberships?: InputMaybe<Scalars["Boolean"]>;
    isPrivatePullRequests?: InputMaybe<Scalars["Boolean"]>;
    isPrivateResources?: InputMaybe<Scalars["Boolean"]>;
    isPrivateResourcesCreated?: InputMaybe<Scalars["Boolean"]>;
    isPrivateTeamsCreated?: InputMaybe<Scalars["Boolean"]>;
    isPrivateVotes?: InputMaybe<Scalars["Boolean"]>;
    languages?: InputMaybe<Array<Scalars["String"]>>;
    name?: InputMaybe<Scalars["String"]>;
    notificationSettings?: InputMaybe<Scalars["String"]>;
    profileImage?: InputMaybe<Scalars["Upload"]>;
    theme?: InputMaybe<Scalars["String"]>;
};

export type PullRequest = DbObject<"PullRequest"> & {
    comments: Array<Comment>;
    commentsCount: Scalars["Int"];
    createdBy?: Maybe<User>;
    createdAt: Scalars["Date"];
    from: PullRequestFrom;
    closedAt?: Maybe<Scalars["Date"]>;
    publicId: Scalars["String"];
    status: PullRequestStatus;
    to: PullRequestTo;
    translations: Array<CommentTranslation>;
    translationsCount: Scalars["Int"];
    updatedAt: Scalars["Date"];
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

export type PullRequestFrom = ResourceVersion;

export enum PullRequestFromObjectType {
    ResourceVersion = "ResourceVersion",
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

export type PullRequestTo = Resource;

export enum PullRequestToObjectType {
    Resource = "Resource",
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
    ChatMessage = "ChatMessage",
    Comment = "Comment",
    Issue = "Issue",
    Resource = "Resource",
}

export type ReactionSearchInput = BaseSearchInput<ReactionSortBy> & {
    chatMessageId?: InputMaybe<Scalars["ID"]>;
    commentId?: InputMaybe<Scalars["ID"]>;
    excludeLinkedToTag?: InputMaybe<Scalars["Boolean"]>;
    issueId?: InputMaybe<Scalars["ID"]>;
    resourceId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
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

export type ReactionTo = ChatMessage | Comment | Issue | Resource;

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
    createdAt: Scalars["Date"];
    description?: Maybe<Scalars["String"]>;
    dueDate?: Maybe<Scalars["Date"]>;
    index: Scalars["Int"];
    isComplete: Scalars["Boolean"];
    name: Scalars["String"];
    reminderItems: Array<ReminderItem>;
    reminderList: ReminderList;
    updatedAt: Scalars["Date"];
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
    createdAt: Scalars["Date"];
    description?: Maybe<Scalars["String"]>;
    dueDate?: Maybe<Scalars["Date"]>;
    index: Scalars["Int"];
    isComplete: Scalars["Boolean"];
    name: Scalars["String"];
    reminder: Reminder;
    updatedAt: Scalars["Date"];
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
    createdAt: Scalars["Date"];
    reminders: Array<Reminder>;
    updatedAt: Scalars["Date"];
};

export type ReminderListCreateInput = {
    id: Scalars["ID"];
    remindersCreate?: InputMaybe<Array<ReminderCreateInput>>;
};

export type ReminderListUpdateInput = {
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
    createdAt: Scalars["Date"];
    details?: Maybe<Scalars["String"]>;
    language: Scalars["String"];
    publicId: Scalars["String"];
    reason: Scalars["String"];
    responses: Array<ReportResponse>;
    responsesCount: Scalars["Int"];
    status: ReportStatus;
    updatedAt: Scalars["Date"];
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
    ChatMessage = "ChatMessage",
    Comment = "Comment",
    Issue = "Issue",
    ResourceVersion = "ResourceVersion",
    Tag = "Tag",
    Team = "Team",
    User = "User"
}

export type ReportResponse = DbObject<"ReportResponse"> & {
    actionSuggested: ReportSuggestedAction;
    createdAt: Scalars["Date"];
    details?: Maybe<Scalars["String"]>;
    language?: Maybe<Scalars["String"]>;
    report: Report;
    updatedAt: Scalars["Date"];
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
    chatMessageId?: InputMaybe<Scalars["ID"]>;
    commentId?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    fromId?: InputMaybe<Scalars["ID"]>;
    issueId?: InputMaybe<Scalars["ID"]>;
    languageIn?: InputMaybe<Array<Scalars["String"]>>;
    resourceVersionId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
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
    createdAt: Scalars["Date"];
    event: Scalars["String"];
    objectId1?: Maybe<Scalars["ID"]>;
    objectId2?: Maybe<Scalars["ID"]>;
    updatedAt: Scalars["Date"];
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

export type Response = {
    __typename: "Response";
    code?: Maybe<Scalars["Int"]>;
    message: Scalars["String"];
};

export type Resource = DbObject<"Resource"> & {
    bookmarkedBy: Array<User>;
    bookmarks: Scalars["Int"];
    completedAt?: Maybe<Scalars["Date"]>;
    createdBy?: Maybe<User>;
    createdAt: Scalars["Date"];
    hasCompleteVersion: Scalars["Boolean"];
    isDeleted: Scalars["Boolean"];
    isInternal?: Maybe<Scalars["Boolean"]>;
    isPrivate: Scalars["Boolean"];
    issues: Array<Issue>;
    issuesCount: Scalars["Int"];
    owner?: Maybe<Owner>;
    parent?: Maybe<ResourceVersion>;
    permissions: Scalars["String"];
    publicId: Scalars["String"];
    pullRequests: Array<PullRequest>;
    pullRequestsCount: Scalars["Int"];
    resourceType: ResourceType;
    score: Scalars["Int"];
    stats: Array<StatsResource>;
    tags: Array<Tag>;
    transfers: Array<Transfer>;
    transfersCount: Scalars["Int"];
    translatedName: Scalars["String"];
    updatedAt: Scalars["Date"];
    versions: Array<ResourceVersion>;
    versionsCount?: Maybe<Scalars["Int"]>;
    views: Scalars["Int"];
    you: ResourceYou;
};

export type ResourceCreateInput = {
    id: Scalars["ID"];
    publicId?: InputMaybe<Scalars["String"]>;
    isInternal?: InputMaybe<Scalars["Boolean"]>;
    isPrivate: Scalars["Boolean"];
    ownedByTeamConnect?: InputMaybe<Scalars["ID"]>;
    ownedByUserConnect?: InputMaybe<Scalars["ID"]>;
    parentConnect?: InputMaybe<Scalars["ID"]>;
    permissions?: InputMaybe<Scalars["String"]>;
    resourceType: ResourceType;
    tagsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
    versionsCreate?: InputMaybe<Array<ResourceVersionCreateInput>>;
};

export type ResourceEdge = Edge<Resource, "ResourceEdge">;

export type ResourceSearchInput = BaseSearchInput<ResourceSortBy> & {
    createdById?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    hasCompleteVersion?: InputMaybe<Scalars["Boolean"]>;
    isInternal?: InputMaybe<Scalars["Boolean"]>;
    issuesId?: InputMaybe<Scalars["ID"]>;
    latestVersionResourceSubType?: InputMaybe<ResourceSubType>;
    latestVersionResourceSubTypes?: InputMaybe<Array<ResourceSubType>>;
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
    resourceType?: InputMaybe<ResourceType>;
    searchString?: InputMaybe<Scalars["String"]>;
    tags?: InputMaybe<Array<Scalars["String"]>>;
    translationLanguagesLatestVersion?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type ResourceSearchResult = SearchResult<ResourceEdge, "ResourceSearchResult">;

export enum ResourceSortBy {
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
    ScoreAsc = "ScoreAsc",
    ScoreDesc = "ScoreDesc",
    VersionsAsc = "VersionsAsc",
    VersionsDesc = "VersionsDesc",
    ViewsAsc = "ViewsAsc",
    ViewsDesc = "ViewsDesc"
}

export enum ResourceType {
    Api = "Api",
    Code = "Code",
    Note = "Note",
    Project = "Project",
    Routine = "Routine",
    Standard = "Standard"
}

export enum ResourceSubTypeRoutine {
    RoutineAction = "RoutineAction",
    RoutineApi = "RoutineApi",
    RoutineCode = "RoutineCode",
    RoutineData = "RoutineData",
    RoutineGenerate = "RoutineGenerate",
    RoutineInformational = "RoutineInformational",
    RoutineMultiStep = "RoutineMultiStep",
    RoutineSmartContract = "RoutineSmartContract",
    RoutineWeb = "RoutineWeb"
}

export enum ResourceSubTypeCode {
    CodeDataConverter = "CodeDataConverter",
    CodeSmartContract = "CodeSmartContract",
}

export enum ResourceSubTypeStandard {
    StandardDataStructure = "StandardDataStructure",
    StandardPrompt = "StandardPrompt",
}

export enum ResourceSubType {
    RoutineAction = "RoutineAction",
    RoutineApi = "RoutineApi",
    RoutineCode = "RoutineCode",
    RoutineData = "RoutineData",
    RoutineGenerate = "RoutineGenerate",
    RoutineInformational = "RoutineInformational",
    RoutineMultiStep = "RoutineMultiStep",
    RoutineSmartContract = "RoutineSmartContract",
    RoutineWeb = "RoutineWeb",
    CodeDataConverter = "CodeDataConverter",
    CodeSmartContract = "CodeSmartContract",
    StandardDataStructure = "StandardDataStructure",
    StandardPrompt = "StandardPrompt"
}

export type ResourceUpdateInput = {
    id: Scalars["ID"];
    isInternal?: InputMaybe<Scalars["Boolean"]>;
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    ownedByTeamConnect?: InputMaybe<Scalars["ID"]>;
    ownedByUserConnect?: InputMaybe<Scalars["ID"]>;
    permissions?: InputMaybe<Scalars["String"]>;
    tagsConnect?: InputMaybe<Array<Scalars["ID"]>>;
    tagsCreate?: InputMaybe<Array<TagCreateInput>>;
    tagsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    versionsCreate?: InputMaybe<Array<ResourceVersionCreateInput>>;
    versionsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    versionsUpdate?: InputMaybe<Array<ResourceVersionUpdateInput>>;
};

export type ResourceVersion = DbObject<"ResourceVersion"> & {
    codeLanguage?: Maybe<Scalars["String"]>;
    comments: Array<Comment>;
    commentsCount: Scalars["Int"];
    completedAt?: Maybe<Scalars["Date"]>;
    complexity: Scalars["Int"];
    config?: Maybe<Scalars["String"]>;
    createdAt: Scalars["Date"];
    forks: Array<Resource>;
    forksCount: Scalars["Int"];
    isAutomatable?: Maybe<Scalars["Boolean"]>;
    isComplete: Scalars["Boolean"];
    isDeleted: Scalars["Boolean"];
    isLatest: Scalars["Boolean"];
    isPrivate: Scalars["Boolean"];
    publicId: Scalars["String"];
    pullRequest?: Maybe<PullRequest>;
    relatedVersions: Array<ResourceVersionRelation>;
    reports: Array<Report>;
    reportsCount: Scalars["Int"];
    root: Resource;
    resourceSubType: ResourceSubType;
    simplicity: Scalars["Int"];
    timesCompleted: Scalars["Int"];
    timesStarted: Scalars["Int"];
    translations: Array<ResourceVersionTranslation>;
    translationsCount: Scalars["Int"];
    updatedAt: Scalars["Date"];
    versionIndex: Scalars["Int"];
    versionLabel: Scalars["String"];
    versionNotes?: Maybe<Scalars["String"]>;
    you: ResourceVersionYou;
};

export type ResourceVersionCreateInput = BaseTranslatableCreateInput<ResourceVersionTranslationCreateInput> & {
    codeLanguage?: InputMaybe<Scalars["String"]>;
    config?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    isAutomatable?: InputMaybe<Scalars["Boolean"]>;
    isComplete?: InputMaybe<Scalars["Boolean"]>;
    isPrivate: Scalars["Boolean"];
    publicId?: InputMaybe<Scalars["String"]>;
    relatedVersionsCreate?: InputMaybe<Array<ResourceVersionRelationCreateInput>>;
    resourceSubType: ResourceSubType;
    rootConnect?: InputMaybe<Scalars["ID"]>;
    rootCreate?: InputMaybe<ResourceCreateInput>;
    versionLabel: Scalars["String"];
    versionNotes?: InputMaybe<Scalars["String"]>;
};

export type ResourceVersionEdge = Edge<ResourceVersion, "ResourceVersionEdge">;

export type ResourceVersionRelation = DbObject<"ResourceVersionRelation"> & {
    toVersion: ResourceVersion;
    labels: Array<Scalars["String"]>;
};

export type ResourceVersionRelationCreateInput = {
    id: Scalars["ID"];
    labels: Array<Scalars["String"]>;
    toVersionConnect: Scalars["ID"];
};

export type ResourceVersionRelationUpdateInput = {
    id: Scalars["ID"];
    labels?: InputMaybe<Array<Scalars["String"]>>;
};

export type ResourceVersionSearchInput = BaseSearchInput<ResourceVersionSortBy> & {
    codeLanguage?: InputMaybe<Scalars["String"]>;
    createdByIdRoot?: InputMaybe<Scalars["ID"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
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
    resourceSubType?: InputMaybe<ResourceSubType>;
    resourceSubTypes?: InputMaybe<Array<ResourceSubType>>;
    searchString?: InputMaybe<Scalars["String"]>;
    tagsRoot?: InputMaybe<Array<Scalars["String"]>>;
    translationLanguages?: InputMaybe<Array<Scalars["String"]>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type ResourceVersionSearchResult = SearchResult<ResourceVersionEdge, "ResourceVersionSearchResult">;

export enum ResourceVersionSortBy {
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
    ForksAsc = "ForksAsc",
    ForksDesc = "ForksDesc",
    RunsAsc = "RunsAsc",
    RunsDesc = "RunsDesc",
    SimplicityAsc = "SimplicityAsc",
    SimplicityDesc = "SimplicityDesc"
}

export type ResourceVersionTranslation = BaseTranslation<"ResourceVersionTranslation"> & {
    description?: Maybe<Scalars["String"]>;
    details?: Maybe<Scalars["String"]>;
    instructions?: Maybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type ResourceVersionTranslationCreateInput = BaseTranslationCreateInput & {
    description?: InputMaybe<Scalars["String"]>;
    details?: InputMaybe<Scalars["String"]>;
    instructions?: InputMaybe<Scalars["String"]>;
    name: Scalars["String"];
};

export type ResourceVersionTranslationUpdateInput = BaseTranslationUpdateInput & {
    description?: InputMaybe<Scalars["String"]>;
    details?: InputMaybe<Scalars["String"]>;
    instructions?: InputMaybe<Scalars["String"]>;
    name?: InputMaybe<Scalars["String"]>;
};

export type ResourceVersionUpdateInput = BaseTranslatableUpdateInput<ResourceVersionTranslationCreateInput, ResourceVersionTranslationUpdateInput> & {
    codeLanguage?: InputMaybe<Scalars["String"]>;
    config?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    isAutomatable?: InputMaybe<Scalars["Boolean"]>;
    isComplete?: InputMaybe<Scalars["Boolean"]>;
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    rootUpdate?: InputMaybe<ResourceUpdateInput>;
    relatedVersionsCreate?: InputMaybe<Array<ResourceVersionRelationCreateInput>>;
    relatedVersionsUpdate?: InputMaybe<Array<ResourceVersionRelationUpdateInput>>;
    relatedVersionsDisconnect?: InputMaybe<Array<Scalars["ID"]>>;
    versionLabel?: InputMaybe<Scalars["String"]>;
    versionNotes?: InputMaybe<Scalars["String"]>;
};

export type ResourceVersionYou = {
    __typename: "ResourceVersionYou";
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

export type ResourceYou = {
    __typename: "ResourceYou";
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

export type Run = DbObject<"Run"> & {
    completedAt?: Maybe<Scalars["Date"]>;
    completedComplexity: Scalars["Int"];
    contextSwitches: Scalars["Int"];
    data?: Maybe<Scalars["String"]>;
    io: Array<RunIO>;
    ioCount: Scalars["Int"];
    isPrivate: Scalars["Boolean"];
    lastStep?: Maybe<Array<Scalars["Int"]>>;
    name: Scalars["String"];
    resourceVersion?: Maybe<ResourceVersion>;
    schedule?: Maybe<Schedule>;
    startedAt?: Maybe<Scalars["Date"]>;
    status: RunStatus;
    steps: Array<RunStep>;
    stepsCount: Scalars["Int"];
    team?: Maybe<Team>;
    timeElapsed?: Maybe<Scalars["Int"]>;
    user?: Maybe<User>;
    wasRunAutomatically: Scalars["Boolean"];
    you: RunYou;
};

export type RunCreateInput = {
    completedComplexity?: InputMaybe<Scalars["Int"]>;
    contextSwitches?: InputMaybe<Scalars["Int"]>;
    data?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    ioCreate?: InputMaybe<Array<RunIOCreateInput>>;
    isPrivate: Scalars["Boolean"];
    name: Scalars["String"];
    resourceVersionConnect: Scalars["ID"];
    scheduleCreate?: InputMaybe<ScheduleCreateInput>;
    startedAt?: InputMaybe<Scalars["Date"]>;
    status: RunStatus;
    stepsCreate?: InputMaybe<Array<RunStepCreateInput>>;
    teamConnect?: InputMaybe<Scalars["ID"]>;
    timeElapsed?: InputMaybe<Scalars["Int"]>;
};

export type RunEdge = Edge<Run, "RunEdge">;

export type RunIO = DbObject<"RunIO"> & {
    data: Scalars["String"];
    nodeInputName: Scalars["String"];
    nodeName: Scalars["String"];
    run: Run;
};

export type RunIOCreateInput = {
    data: Scalars["String"];
    id: Scalars["ID"];
    nodeInputName: Scalars["String"];
    nodeName: Scalars["String"];
    runConnect: Scalars["ID"];
};

export type RunIOEdge = Edge<RunIO, "RunIOEdge">;

export type RunIOSearchInput = {
    after?: InputMaybe<Scalars["String"]>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    ids?: InputMaybe<Array<Scalars["ID"]>>;
    runIds?: InputMaybe<Array<Scalars["ID"]>>;
    take?: InputMaybe<Scalars["Int"]>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
};

export type RunIOSearchResult = SearchResult<RunIOEdge, "RunIOSearchResult">;

export enum RunIOSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc"
}

export type RunIOUpdateInput = {
    data: Scalars["String"];
    id: Scalars["ID"];
};

export type RunSearchInput = BaseSearchInput<RunSortBy> & {
    completedTimeFrame?: InputMaybe<TimeFrame>;
    createdTimeFrame?: InputMaybe<TimeFrame>;
    excludeIds?: InputMaybe<Array<Scalars["ID"]>>;
    resourceVersionId?: InputMaybe<Scalars["ID"]>;
    scheduleEndTimeFrame?: InputMaybe<TimeFrame>;
    scheduleStartTimeFrame?: InputMaybe<TimeFrame>;
    searchString?: InputMaybe<Scalars["String"]>;
    startedTimeFrame?: InputMaybe<TimeFrame>;
    status?: InputMaybe<RunStatus>;
    statuses?: InputMaybe<Array<RunStatus>>;
    updatedTimeFrame?: InputMaybe<TimeFrame>;
    visibility?: InputMaybe<VisibilityType>;
};

export type RunSearchResult = SearchResult<RunEdge, "RunSearchResult">;

export enum RunSortBy {
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

export type RunStep = DbObject<"RunStep"> & {
    completedAt?: Maybe<Scalars["Date"]>;
    complexity: Scalars["Int"];
    contextSwitches: Scalars["Int"];
    name: Scalars["String"];
    nodeId: Scalars["String"];
    order: Scalars["Int"];
    startedAt?: Maybe<Scalars["Date"]>;
    status: RunStepStatus;
    resourceVersion?: Maybe<ResourceVersion>;
    resourceInId: Scalars["ID"];
    timeElapsed?: Maybe<Scalars["Int"]>;
};

export type RunStepCreateInput = {
    complexity: Scalars["Int"];
    contextSwitches?: InputMaybe<Scalars["Int"]>;
    id: Scalars["ID"];
    name: Scalars["String"];
    nodeId: Scalars["String"];
    order: Scalars["Int"];
    runConnect: Scalars["ID"];
    status?: InputMaybe<RunStepStatus>;
    resourceVersionConnect?: InputMaybe<Scalars["ID"]>;
    resourceInId: Scalars["ID"];
    timeElapsed?: InputMaybe<Scalars["Int"]>;
};

export enum RunStepSortBy {
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

export type RunStepUpdateInput = {
    contextSwitches?: InputMaybe<Scalars["Int"]>;
    id: Scalars["ID"];
    status?: InputMaybe<RunStepStatus>;
    timeElapsed?: InputMaybe<Scalars["Int"]>;
};

export type RunUpdateInput = {
    completedComplexity?: InputMaybe<Scalars["Int"]>;
    contextSwitches?: InputMaybe<Scalars["Int"]>;
    data?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    ioCreate?: InputMaybe<Array<RunIOCreateInput>>;
    ioDelete?: InputMaybe<Array<Scalars["ID"]>>;
    ioUpdate?: InputMaybe<Array<RunIOUpdateInput>>;
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    scheduleCreate?: InputMaybe<ScheduleCreateInput>;
    scheduleUpdate?: InputMaybe<ScheduleUpdateInput>;
    scheduleDelete?: InputMaybe<Scalars["Boolean"]>;
    startedAt?: InputMaybe<Scalars["Date"]>;
    status?: InputMaybe<RunStatus>;
    stepsCreate?: InputMaybe<Array<RunStepCreateInput>>;
    stepsDelete?: InputMaybe<Array<Scalars["ID"]>>;
    stepsUpdate?: InputMaybe<Array<RunStepUpdateInput>>;
    timeElapsed?: InputMaybe<Scalars["Int"]>;
};

export type RunYou = {
    __typename: "RunYou";
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

export enum SandboxTask {
    CallApi = "CallApi",
    RunDataTransform = "RunDataTransform",
    RunSmartContract = "RunSmartContract"
}

export type Schedule = DbObject<"Schedule"> & {
    createdAt: Scalars["Date"];
    endTime: Scalars["Date"];
    exceptions: Array<ScheduleException>;
    meetings: Array<Meeting>;
    publicId: Scalars["String"];
    recurrences: Array<ScheduleRecurrence>;
    runs: Array<Run>;
    startTime: Scalars["Date"];
    timezone: Scalars["String"];
    updatedAt: Scalars["Date"];
};

export type ScheduleCreateInput = {
    endTime?: InputMaybe<Scalars["Date"]>;
    exceptionsCreate?: InputMaybe<Array<ScheduleExceptionCreateInput>>;
    id: Scalars["ID"];
    meetingConnect?: InputMaybe<Scalars["ID"]>;
    recurrencesCreate?: InputMaybe<Array<ScheduleRecurrenceCreateInput>>;
    runConnect?: InputMaybe<Scalars["ID"]>;
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
    Meeting = "Meeting",
    Run = "Run",
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
    credits: Scalars["String"];
    handle?: Maybe<Scalars["String"]>;
    hasPremium: Scalars["Boolean"];
    id: Scalars["String"];
    languages: Array<Scalars["String"]>;
    name?: Maybe<Scalars["String"]>;
    profileImage?: Maybe<Scalars["String"]>;
    publicId: Scalars["String"];
    session: SessionUserSession;
    theme?: Maybe<Scalars["String"]>;
    updatedAt: Scalars["Date"];
};

export type SessionUserSession = {
    __typename: "SessionUserSession";
    id: Scalars["String"];
    lastRefreshAt: Scalars["Date"];
};

export type StartLlmTaskInput = {
    chatId: Scalars["ID"];
    model: string;
    parentId?: InputMaybe<Scalars["ID"]>;
    respondingBot: { id?: string, publicId?: string, handle?: string };
    shouldNotRunTasks: Scalars["Boolean"];
    task: LlmTask;
    taskContexts: Array<TaskContextInfoInput>;
};

export type StartRunTaskInput = {
    config: RunConfig;
    formValues?: InputMaybe<Scalars["JSONObject"]>;
    isNewRun: Scalars["Boolean"];
    resourceVersionId: Scalars["ID"];
    runId: Scalars["ID"];
};

export enum StatPeriodType {
    Daily = "Daily",
    Hourly = "Hourly",
    Monthly = "Monthly",
    Weekly = "Weekly",
    Yearly = "Yearly"
}

export type StatsResource = DbObject<"StatsResource"> & {
    periodEnd: Scalars["Date"];
    periodStart: Scalars["Date"];
    periodType: StatPeriodType;
    references: Scalars["Int"];
    referencedBy: Scalars["Int"];
    runCompletionTimeAverage: Scalars["Float"];
    runContextSwitchesAverage: Scalars["Float"];
    runsCompleted: Scalars["Int"];
    runsStarted: Scalars["Int"];
};

export type StatsResourceEdge = Edge<StatsResource, "StatsResourceEdge">;

export type StatsResourceSearchInput = BaseSearchInput<StatsResourceSortBy> & {
    periodTimeFrame?: InputMaybe<TimeFrame>;
    periodType: StatPeriodType;
    searchString?: InputMaybe<Scalars["String"]>;
};

export type StatsResourceSearchResult = SearchResult<StatsResourceEdge, "StatsResourceSearchResult">;

export enum StatsResourceSortBy {
    PeriodStartAsc = "PeriodStartAsc",
    PeriodStartDesc = "PeriodStartDesc"
}

export type StatsSite = DbObject<"StatsSite"> & {
    periodEnd: Scalars["Date"];
    periodStart: Scalars["Date"];
    periodType: StatPeriodType;
    activeUsers: Scalars["Int"];
    teamsCreated: Scalars["Int"];
    verifiedEmailsCreated: Scalars["Int"];
    verifiedWalletsCreated: Scalars["Int"];
    resourcesCreatedByType: Scalars["String"];
    resourcesCompletedByType: Scalars["String"];
    resourceCompletionTimeAverageByType: Scalars["String"];
    routineSimplicityAverage: Scalars["Float"];
    routineComplexityAverage: Scalars["Float"];
    runCompletionTimeAverage: Scalars["Float"];
    runContextSwitchesAverage: Scalars["Float"];
    runsStarted: Scalars["Int"];
    runsCompleted: Scalars["Int"];
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

export type StatsTeam = DbObject<"StatsTeam"> & {
    members: Scalars["Int"];
    periodEnd: Scalars["Date"];
    periodStart: Scalars["Date"];
    periodType: StatPeriodType;
    projects: Scalars["Int"];
    resources: Scalars["Int"];
    runCompletionTimeAverage: Scalars["Float"];
    runContextSwitchesAverage: Scalars["Float"];
    runsCompleted: Scalars["Int"];
    runsStarted: Scalars["Int"];
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
    periodEnd: Scalars["Date"];
    periodStart: Scalars["Date"];
    periodType: StatPeriodType;
    resourcesCreatedByType: Scalars["String"];
    resourcesCompletedByType: Scalars["String"];
    resourceCompletionTimeAverageByType: Scalars["String"];
    runsStarted: Scalars["Int"];
    runsCompleted: Scalars["Int"];
    runCompletionTimeAverage: Scalars["Float"];
    runContextSwitchesAverage: Scalars["Float"];
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
    Comment = "Comment",
    Issue = "Issue",
    Meeting = "Meeting",
    PullRequest = "PullRequest",
    Report = "Report",
    Resource = "Resource",
    Schedule = "Schedule",
    Team = "Team"
}

export type SubscribedObject = Comment | Issue | Meeting | PullRequest | Report | Resource | Schedule | Team;

export type Success = {
    __typename: "Success";
    success: Scalars["Boolean"];
};

export type SwitchCurrentAccountInput = {
    id: Scalars["ID"];
};

export type Tag = DbObject<"Tag"> & {
    bookmarkedBy: Array<User>;
    bookmarks: Scalars["Int"];
    createdAt: Scalars["Date"];
    reports: Array<Report>;
    resources: Array<Resource>;
    tag: Scalars["String"];
    teams: Array<Team>;
    translations: Array<TagTranslation>;
    updatedAt: Scalars["Date"];
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
    bannerImage?: Maybe<Scalars["String"]>;
    bookmarkedBy: Array<User>;
    bookmarks: Scalars["Int"];
    comments: Array<Comment>;
    commentsCount: Scalars["Int"];
    config?: Maybe<Scalars["String"]>;
    createdAt: Scalars["Date"];
    forks: Array<Team>;
    handle?: Maybe<Scalars["String"]>;
    isOpenToNewMembers: Scalars["Boolean"];
    isPrivate: Scalars["Boolean"];
    issues: Array<Issue>;
    issuesCount: Scalars["Int"];
    meetings: Array<Meeting>;
    meetingsCount: Scalars["Int"];
    members: Array<Member>;
    membersCount: Scalars["Int"];
    parent?: Maybe<Team>;
    paymentHistory: Array<Payment>;
    permissions: Scalars["String"];
    premium?: Maybe<Premium>;
    profileImage?: Maybe<Scalars["String"]>;
    publicId: Scalars["String"];
    reports: Array<Report>;
    reportsCount: Scalars["Int"];
    resources: Array<Resource>;
    resourcesCount: Scalars["Int"];
    stats: Array<StatsTeam>;
    tags: Array<Tag>;
    transfersIncoming: Array<Transfer>;
    transfersOutgoing: Array<Transfer>;
    translatedName: Scalars["String"];
    translations: Array<TeamTranslation>;
    translationsCount: Scalars["Int"];
    updatedAt: Scalars["Date"];
    views: Scalars["Int"];
    wallets: Array<Wallet>;
    you: TeamYou;
};

export type TeamCreateInput = BaseTranslatableCreateInput<TeamTranslationCreateInput> & {
    bannerImage?: InputMaybe<Scalars["Upload"]>;
    config?: InputMaybe<Scalars["String"]>;
    handle?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    isOpenToNewMembers?: InputMaybe<Scalars["Boolean"]>;
    isPrivate: Scalars["Boolean"];
    memberInvitesCreate?: InputMaybe<Array<MemberInviteCreateInput>>;
    permissions?: InputMaybe<Scalars["String"]>;
    profileImage?: InputMaybe<Scalars["Upload"]>;
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
    reportId?: InputMaybe<Scalars["ID"]>;
    resourceId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
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
    config?: InputMaybe<Scalars["String"]>;
    handle?: InputMaybe<Scalars["String"]>;
    id: Scalars["ID"];
    isOpenToNewMembers?: InputMaybe<Scalars["Boolean"]>;
    isPrivate?: InputMaybe<Scalars["Boolean"]>;
    memberInvitesCreate?: InputMaybe<Array<MemberInviteCreateInput>>;
    memberInvitesDelete?: InputMaybe<Array<Scalars["ID"]>>;
    membersDelete?: InputMaybe<Array<Scalars["ID"]>>;
    permissions?: InputMaybe<Scalars["String"]>;
    profileImage?: InputMaybe<Scalars["Upload"]>;
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
    createdAt: Scalars["Date"];
    fromOwner?: Maybe<Owner>;
    closedAt?: Maybe<Scalars["Date"]>;
    object: TransferObject;
    status: TransferStatus;
    toOwner?: Maybe<Owner>;
    updatedAt: Scalars["Date"];
    you: TransferYou;
};

export type TransferDenyInput = {
    id: Scalars["ID"];
    reason?: InputMaybe<Scalars["String"]>;
};

export type TransferEdge = Edge<Transfer, "TransferEdge">;

export type TransferObject = Resource;

export enum TransferObjectType {
    Resource = "Resource",
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
    createdTimeFrame?: InputMaybe<TimeFrame>;
    fromTeamId?: InputMaybe<Scalars["ID"]>;
    resourceId?: InputMaybe<Scalars["ID"]>;
    searchString?: InputMaybe<Scalars["String"]>;
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

export type User = DbObject<"User"> & {
    apiKeys?: Maybe<Array<ApiKey>>;
    apiKeysExternal?: Maybe<Array<ApiKeyExternal>>;
    awards?: Maybe<Array<Award>>;
    bannerImage?: Maybe<Scalars["String"]>;
    bookmarked?: Maybe<Array<Bookmark>>;
    bookmarkedBy: Array<User>;
    bookmarks: Scalars["Int"];
    botSettings?: Maybe<Scalars["String"]>;
    comments?: Maybe<Array<Comment>>;
    createdAt: Scalars["Date"];
    emails?: Maybe<Array<Email>>;
    handle?: Maybe<Scalars["String"]>;
    invitedByUser?: Maybe<User>;
    invitedUsers?: Maybe<Array<User>>;
    isBot: Scalars["Boolean"];
    isBotDepictingPerson: Scalars["Boolean"];
    isPrivate: Scalars["Boolean"];
    isPrivateBookmarks: Scalars["Boolean"];
    isPrivateMemberships: Scalars["Boolean"];
    isPrivatePullRequests: Scalars["Boolean"];
    isPrivateResources: Scalars["Boolean"];
    isPrivateResourcesCreated: Scalars["Boolean"];
    isPrivateTeamsCreated: Scalars["Boolean"];
    isPrivateVotes: Scalars["Boolean"];
    issuesClosed?: Maybe<Array<Issue>>;
    issuesCreated?: Maybe<Array<Issue>>;
    meetingsAttending?: Maybe<Array<Meeting>>;
    meetingsInvited?: Maybe<Array<MeetingInvite>>;
    memberships?: Maybe<Array<Member>>;
    membershipsCount: Scalars["Int"];
    membershipsInvited?: Maybe<Array<MemberInvite>>;
    name: Scalars["String"];
    notificationSettings?: Maybe<Scalars["String"]>;
    notificationSubscriptions?: Maybe<Array<NotificationSubscription>>;
    notifications?: Maybe<Array<Notification>>;
    paymentHistory?: Maybe<Array<Payment>>;
    phones?: Maybe<Array<Phone>>;
    premium?: Maybe<Premium>;
    profileImage?: Maybe<Scalars["String"]>;
    publicId: Scalars["String"];
    pullRequests?: Maybe<Array<PullRequest>>;
    pushDevices?: Maybe<Array<PushDevice>>;
    reacted?: Maybe<Array<Reaction>>;
    reportResponses?: Maybe<Array<ReportResponse>>;
    reportsCreated?: Maybe<Array<Report>>;
    reportsReceived: Array<Report>;
    reportsReceivedCount: Scalars["Int"];
    reputationHistory?: Maybe<Array<ReputationHistory>>;
    resources?: Maybe<Array<Resource>>;
    resourcesCount: Scalars["Int"];
    resourcesCreated?: Maybe<Array<Resource>>;
    runs?: Maybe<Array<Run>>;
    sentReports?: Maybe<Array<Report>>;
    status?: Maybe<AccountStatus>;
    tags?: Maybe<Array<Tag>>;
    teamsCreated?: Maybe<Array<Team>>;
    theme?: Maybe<Scalars["String"]>;
    transfersIncoming?: Maybe<Array<Transfer>>;
    transfersOutgoing?: Maybe<Array<Transfer>>;
    translationLanguages?: Maybe<Array<Scalars["String"]>>;
    translations: Array<UserTranslation>;
    updatedAt?: Maybe<Scalars["Date"]>;
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

export type ViewTo = Issue | Resource | Team | User;

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
    verifiedAt: Scalars["Date"];
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
