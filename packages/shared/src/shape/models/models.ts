import { CommentFor, type Bookmark, type BookmarkCreateInput, type BookmarkFor, type BookmarkList, type BookmarkListCreateInput, type BookmarkListUpdateInput, type BookmarkUpdateInput, type BotCreateInput, type BotUpdateInput, type Chat, type ChatCreateInput, type ChatInvite, type ChatInviteCreateInput, type ChatInviteStatus, type ChatInviteUpdateInput, type ChatInviteYou, type ChatMessage, type ChatMessageCreateInput, type ChatMessageParent, type ChatMessageUpdateInput, type ChatMessageYou, type ChatParticipant, type ChatParticipantUpdateInput, type ChatTranslation, type ChatTranslationCreateInput, type ChatTranslationUpdateInput, type ChatUpdateInput, type Comment, type CommentCreateInput, type CommentTranslation, type CommentTranslationCreateInput, type CommentTranslationUpdateInput, type CommentUpdateInput, type CommentedOn, type Issue, type IssueCreateInput, type IssueFor, type IssueTranslation, type IssueTranslationCreateInput, type IssueTranslationUpdateInput, type IssueUpdateInput, type Meeting, type MeetingCreateInput, type MeetingInvite, type MeetingInviteCreateInput, type MeetingInviteUpdateInput, type MeetingTranslation, type MeetingTranslationCreateInput, type MeetingTranslationUpdateInput, type MeetingUpdateInput, type Member, type MemberInvite, type MemberInviteCreateInput, type MemberInviteUpdateInput, type MemberUpdateInput, type ProfileUpdateInput, type PullRequest, type PullRequestCreateInput, type PullRequestTranslation, type PullRequestTranslationCreateInput, type PullRequestTranslationUpdateInput, type PullRequestUpdateInput, type ReactionSummary, type Reminder, type ReminderCreateInput, type ReminderItem, type ReminderItemCreateInput, type ReminderItemUpdateInput, type ReminderList, type ReminderListCreateInput, type ReminderListUpdateInput, type ReminderUpdateInput, type Report, type ReportCreateInput, type ReportFor, type ReportResponse, type ReportResponseCreateInput, type ReportResponseUpdateInput, type ReportUpdateInput, type Resource, type ResourceCreateInput, type ResourceUpdateInput, type ResourceVersion, type ResourceVersionCreateInput, type ResourceVersionRelation, type ResourceVersionRelationCreateInput, type ResourceVersionRelationUpdateInput, type ResourceVersionTranslation, type ResourceVersionTranslationCreateInput, type ResourceVersionTranslationUpdateInput, type ResourceVersionUpdateInput, type Run, type RunCreateInput, type RunIO, type RunIOCreateInput, type RunIOUpdateInput, type RunStep, type RunStepCreateInput, type RunStepUpdateInput, type RunUpdateInput, type Schedule, type ScheduleCreateInput, type ScheduleException, type ScheduleExceptionCreateInput, type ScheduleExceptionUpdateInput, type ScheduleRecurrence, type ScheduleRecurrenceCreateInput, type ScheduleRecurrenceUpdateInput, type ScheduleUpdateInput, type Tag, type TagCreateInput, type TagTranslation, type TagTranslationCreateInput, type TagTranslationUpdateInput, type TagUpdateInput, type Team, type TeamCreateInput, type TeamTranslation, type TeamTranslationCreateInput, type TeamTranslationUpdateInput, type TeamUpdateInput, type User, type UserTranslation, type UserTranslationCreateInput, type UserTranslationUpdateInput } from "../../api/types.js";
import { type CanConnect, type ShapeModel } from "../../consts/commonTypes.js";
import { DUMMY_ID } from "../../id/snowflake.js";
import { LATEST_CONFIG_VERSION } from "../../shape/configs/utils.js";
import { createOwner, createPrims, createRel, createVersion, shapeDate, shapeUpdate, updateOwner, updatePrims, updateRel, updateTagsRel, updateTransRel, updateTranslationPrims, updateVersion } from "./tools.js";
import { type OwnerShape } from "./types.js";

export type BookmarkShape = Pick<Bookmark, "id"> & {
    __typename: "Bookmark";
    to: { __typename: Bookmark["to"]["__typename"], id: string };
    list: CanConnect<BookmarkListShape> | null;
}
export const shapeBookmark: ShapeModel<BookmarkShape, BookmarkCreateInput, BookmarkUpdateInput> = {
    create: (d) => ({
        forConnect: d.to.id,
        bookmarkFor: d.to.__typename as BookmarkFor,
        ...createPrims(d, "id"),
        ...createRel(d, "list", ["Create", "Connect"], "one", shapeBookmarkList),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateRel(o, u, "list", ["Create", "Update"], "one", shapeBookmarkList),
    }),
};

export type BookmarkListShape = Pick<BookmarkList, "id" | "label"> & {
    __typename: "BookmarkList";
    bookmarks?: BookmarkShape[] | null;
}
export const shapeBookmarkList: ShapeModel<BookmarkListShape, BookmarkListCreateInput, BookmarkListUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "label"),
        ...createRel(d, "bookmarks", ["Create"], "many", shapeBookmark),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "label"),
        ...updateRel(o, u, "bookmarks", ["Create", "Delete"], "many", shapeBookmark),
    }),
};

export type BotShape = Pick<User, "id" | "botSettings" | "handle" | "isBotDepictingPerson" | "isPrivate" | "name"> & {
    __typename: "User";
    bannerImage?: string | File | null;
    isBot?: true;
    profileImage?: string | File | null;
    translations?: UserTranslationShape[] | null;
}

export const shapeBot: ShapeModel<BotShape, BotCreateInput, BotUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "bannerImage", "handle", "isBot", "isBotDepictingPerson", "isPrivate", "name", "profileImage"),
        botSettings: d.botSettings ?? { __version: LATEST_CONFIG_VERSION },
        ...createRel(d, "translations", ["Create"], "many", shapeUserTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "bannerImage", "handle", "isBot", "isBotDepictingPerson", "isPrivate", "name", "profileImage"),
        botSettings: u.botSettings ?? o.botSettings ?? { __version: LATEST_CONFIG_VERSION },
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeUserTranslation),
    }),
};

export type ChatTranslationShape = Pick<ChatTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "ChatTranslation";
}
export type ChatShape = Pick<Chat, "id" | "openToAnyoneWithInvite"> & {
    __typename: "Chat";
    /** Invites for new participants */
    invites: ChatInviteShape[];
    messages: ChatMessageShape[];
    /** Potentially non-exhaustive list of current participants. Ignored by shapers */
    participants?: ChatParticipantShape[];
    /** Participants being removed from the chst */
    participantsDelete?: { id: string }[] | null;
    team?: CanConnect<TeamShape> | null;
    translations?: ChatTranslationShape[] | null;
}
export const shapeChatTranslation: ShapeModel<ChatTranslationShape, ChatTranslationCreateInput, ChatTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "name")),
};
export const shapeChat: ShapeModel<ChatShape, ChatCreateInput, ChatUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "openToAnyoneWithInvite");
        return {
            ...prims,
            ...createRel(d, "invites", ["Create"], "many", shapeChatInvite, (m) => ({ ...m, chat: { id: prims.id } })),
            ...createRel(d, "messages", ["Create"], "many", shapeChatMessage, (m) => {
                console.log("preshaping chat message", m, prims);
                return { ...m, chat: { id: prims.id } };
            }),
            ...createRel(d, "translations", ["Create"], "many", shapeChatTranslation),
            ...(d.team ? { teamConnect: d.team.id } : {}),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "openToAnyoneWithInvite"),
        ...updateRel(o, u, "invites", ["Create", "Update", "Delete"], "many", shapeChatInvite, (m, i) => ({ ...m, chat: { id: i.id } })),
        ...updateRel(o, u, "messages", ["Create", "Update", "Delete"], "many", shapeChatMessage, (m, i) => ({ ...m, chat: { id: i.id } })),
        ...updateTransRel(o, u, shapeChatTranslation),
        ...(u.participantsDelete?.length ? { participantsDelete: u.participantsDelete.map(m => m.id) } : {}),
    }),
};

export type ChatInviteShape = Pick<ChatInvite, "id" | "message"> & {
    __typename: "ChatInvite";
    createdAt: string; // Only used by the UI
    updatedAt: string; // Only used by the UI
    status: ChatInviteStatus; // Ignored when mutating, so don't get any ideas
    chat: CanConnect<ChatShape> | null;
    user: { __typename: "User", id: string };
    you?: ChatInviteYou; // Only used by the UI
}
export const shapeChatInvite: ShapeModel<ChatInviteShape, ChatInviteCreateInput, ChatInviteUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "message"),
        ...createRel(d, "chat", ["Connect"], "one"),
        ...createRel(d, "user", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "message"),
    }),
};

export type ChatMessageStatus = "unsent" | "editing" | "sending" | "sent" | "failed";
export type ChatMessageShape = Pick<ChatMessage, "id" | "config" | "language" | "text" | "versionIndex"> & {
    __typename: "ChatMessage";
    createdAt: string; // Only used by the UI
    updatedAt: string; // Only used by the UI
    chat?: CanConnect<ChatShape> | null;
    sequence?: number; // Only used by the UI
    /** If not provided, we'll assume it's sent */
    status?: ChatMessageStatus;
    parent?: CanConnect<ChatMessageParent> | null;
    parentId?: string | null;
    reactionSummaries: ReactionSummary[]; // Only used by the UI
    user?: CanConnect<User> | null;
    you?: ChatMessageYou; // Only used by the UI
}
export const shapeChatMessage: ShapeModel<ChatMessageShape, ChatMessageCreateInput, ChatMessageUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "config", "language", "text", "versionIndex"),
        ...createRel(d, "chat", ["Connect"], "one"),
        ...createRel(d, "user", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "text"),
    }),
};

export type ChatParticipantShape = Pick<ChatParticipant, "id"> & {
    __typename: "ChatParticipant";
    user: Pick<User, "__typename" | "updatedAt" | "handle" | "id" | "isBot" | "name" | "profileImage" | "publicId">;
}
export const shapeChatParticipant: ShapeModel<ChatParticipantShape, null, ChatParticipantUpdateInput> = {
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
    }),
};

export type CommentTranslationShape = Pick<CommentTranslation, "id" | "language" | "text"> & {
    __typename?: "CommentTranslation";
}
export type CommentShape = Pick<Comment, "id"> & {
    __typename: "Comment";
    commentedOn: { __typename: CommentedOn["__typename"], id: string };
    threadId?: string | null;
    translations: CommentTranslationShape[];
}
export const shapeCommentTranslation: ShapeModel<CommentTranslationShape, CommentTranslationCreateInput, CommentTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "text"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "text")),
};
export const shapeComment: ShapeModel<CommentShape, CommentCreateInput, CommentUpdateInput> = {
    create: (d) => {
        if (!Object.prototype.hasOwnProperty.call(d, "commentedOn")) {
            console.error("Comment must have a commentedOn field");
        }
        return {
            ...createPrims(d, "id", "threadId"),
            createdFor: d.commentedOn ? (d.commentedOn.__typename as CommentFor) : CommentFor.ResourceVersion,
            forConnect: d.commentedOn ? d.commentedOn.id : DUMMY_ID,
            ...createRel(d, "translations", ["Create"], "many", shapeCommentTranslation),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateTransRel(o, u, shapeCommentTranslation),
    }),
};

export type IssueTranslationShape = Pick<IssueTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "IssueTranslation";
}
export type IssueShape = Pick<Issue, "id"> & {
    __typename: "Issue";
    issueFor: IssueFor;
    for: { id: string };
    translations: IssueTranslationShape[];
}
export const shapeIssueTranslation: ShapeModel<IssueTranslationShape, IssueTranslationCreateInput, IssueTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "name")),
};
export const shapeIssue: ShapeModel<IssueShape, IssueCreateInput, IssueUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "issueFor"),
        ...createRel(d, "for", ["Connect"], "one"),
        ...createRel(d, "translations", ["Create"], "many", shapeIssueTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateTransRel(o, u, shapeIssueTranslation),
    }),
};

export type MeetingTranslationShape = Pick<MeetingTranslation, "id" | "language" | "description" | "link" | "name"> & {
    __typename?: "MeetingTranslation";
}
export type MeetingShape = Pick<Meeting, "id" | "openToAnyoneWithInvite" | "showOnTeamProfile"> & {
    __typename: "Meeting";
    invites?: CanConnect<MeetingInviteShape>[] | null;
    schedule?: CanConnect<ScheduleShape> | null;
    team: CanConnect<TeamShape> | null;
    translations?: MeetingTranslationShape[] | null;
}
export const shapeMeetingTranslation: ShapeModel<MeetingTranslationShape, MeetingTranslationCreateInput, MeetingTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "link", "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "link", "name")),
};
export const shapeMeeting: ShapeModel<MeetingShape, MeetingCreateInput, MeetingUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "openToAnyoneWithInvite", "showOnTeamProfile");
        return {
            ...prims,
            ...createRel(d, "invites", ["Create"], "many", shapeMeetingInvite, (i) => ({ ...i, meeting: { id: prims.id } })),
            ...createRel(d, "schedule", ["Create"], "one", shapeSchedule),
            ...createRel(d, "team", ["Connect"], "one"),
            ...createRel(d, "translations", ["Create"], "many", shapeMeetingTranslation),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "openToAnyoneWithInvite", "showOnTeamProfile"),
        ...updateRel(o, u, "invites", ["Create", "Update", "Delete"], "many", shapeMeetingInvite, (i) => ({ ...i, meeting: { id: o.id } })),
        ...updateRel(o, u, "schedule", ["Create", "Update"], "one", shapeSchedule),
        ...updateTransRel(o, u, shapeMeetingTranslation),
    }),
};

export type MeetingInviteShape = Pick<MeetingInvite, "id" | "message"> & {
    __typename: "MeetingInvite";
    meeting: CanConnect<MeetingShape>;
    user: { id: string };
}
export const shapeMeetingInvite: ShapeModel<MeetingInviteShape, MeetingInviteCreateInput, MeetingInviteUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "message"),
        ...createRel(d, "meeting", ["Connect"], "one"),
        ...createRel(d, "user", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "message"),
    }),
};

export type MemberShape = Pick<Member, "id" | "isAdmin" | "permissions"> & {
    __typename: "Member";
    user: Pick<User, "updatedAt" | "handle" | "id" | "isBot" | "name" | "profileImage">;
}
export const shapeMember: ShapeModel<MemberShape, null, MemberUpdateInput> = {
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isAdmin", "permissions"),
    }),
};

export type MemberInviteShape = Pick<MemberInvite, "id" | "message" | "willBeAdmin" | "willHavePermissions"> & {
    __typename: "MemberInvite";
    team: CanConnect<TeamShape>;
    user: Pick<User, "__typename" | "updatedAt" | "handle" | "id" | "isBot" | "name" | "profileImage">;
}
export const shapeMemberInvite: ShapeModel<MemberInviteShape, MemberInviteCreateInput, MemberInviteUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "message", "willBeAdmin", "willHavePermissions"),
        ...createRel(d, "team", ["Connect"], "one"),
        ...createRel(d, "user", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "message", "willBeAdmin", "willHavePermissions"),
    }),
};

export type ProfileTranslationShape = Pick<UserTranslation, "id" | "language" | "bio"> & {
    __typename?: "UserTranslation";
}
export type ProfileShape = Partial<Pick<User, "handle" | "isPrivate" | "isPrivateMemberships" | "isPrivatePullRequests" | "isPrivateResources" | "isPrivateResourcesCreated" | "isPrivateTeamsCreated" | "isPrivateBookmarks" | "isPrivateVotes" | "name" | "theme">> & {
    __typename: "User",
    id: string;
    bannerImage?: string | File | null;
    profileImage?: string | File | null;
    translations?: ProfileTranslationShape[] | null;
}
export const shapeProfileTranslation: ShapeModel<ProfileTranslationShape, UserTranslationCreateInput, UserTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "bio"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "bio")),
};
export const shapeProfile: ShapeModel<ProfileShape, null, ProfileUpdateInput> = {
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, null,
            "bannerImage",
            "handle",
            "id",
            "isPrivateMemberships",
            "isPrivatePullRequests",
            "isPrivateResources",
            "isPrivateResourcesCreated",
            "isPrivateTeamsCreated",
            "isPrivateBookmarks",
            "isPrivateVotes",
            "name",
            "profileImage",
            "theme",
        ),
        ...updateTransRel(o, u, shapeProfileTranslation),
    }),
};

export type PullRequestTranslationShape = Pick<PullRequestTranslation, "id" | "language" | "text"> & {
    __typename?: "PullRequestTranslation";
}
export type PullRequestShape = Pick<PullRequest, "id"> & {
    __typename: "PullRequest";
}
export const shapePullRequestTranslation: ShapeModel<PullRequestTranslationShape, PullRequestTranslationCreateInput, PullRequestTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "text"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "text")),
};
export const shapePullRequest: ShapeModel<PullRequestShape, PullRequestCreateInput, PullRequestUpdateInput> = {
    create: (_d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any,
};

export type ReminderShape = Pick<Reminder, "id" | "name" | "description" | "dueDate" | "index" | "isComplete"> & {
    __typename: "Reminder";
    reminderList: CanConnect<ReminderListShape>;
    reminderItems?: CanConnect<ReminderItemShape>[] | null;
}
export const shapeReminder: ShapeModel<ReminderShape, ReminderCreateInput, ReminderUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "name", "description", ["dueDate", shapeDate], "index");
        return {
            ...prims,
            ...createRel(d, "reminderItems", ["Create"], "many", shapeReminderItem, (r) => ({ ...r, reminder: { id: prims.id } })),
            // Treat as connect when the reminderList has createdAt
            ...createRel(d, "reminderList", ["Connect", "Create"], "one", shapeReminderList, (l) => {
                if (l.createdAt) return { id: l.id };
                return l;
            }),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "name", "description", ["dueDate", shapeDate], "index", "isComplete"),
        ...updateRel(o, u, "reminderItems", ["Create", "Update", "Delete"], "many", shapeReminderItem, (r, i) => ({ ...r, reminder: { id: i.id } })),
        // Treat as connect when the reminderList has createdAt
        ...updateRel(o, u, "reminderList", ["Connect", "Create", "Disconnect"], "one", shapeReminderList, (l) => {
            if (l.createdAt) return { id: l.id };
            return l;
        }),
    }),
};

export type ReminderItemShape = Pick<ReminderItem, "id" | "name" | "description" | "dueDate" | "index" | "isComplete"> & {
    __typename: "ReminderItem";
    reminder: CanConnect<ReminderShape>;
}
export const shapeReminderItem: ShapeModel<ReminderItemShape, ReminderItemCreateInput, ReminderItemUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "name", "description", ["dueDate", shapeDate], "index", "isComplete"),
        ...createRel(d, "reminder", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "name", "description", ["dueDate", shapeDate], "index", "isComplete"),
    }),
};

export type ReminderListShape = Pick<ReminderList, "id"> & {
    __typename: "ReminderList";
    reminders?: CanConnect<ReminderShape>[] | null;
}
export const shapeReminderList: ShapeModel<ReminderListShape, ReminderListCreateInput, ReminderListUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id");
        return {
            ...prims,
            ...createRel(d, "reminders", ["Create"], "many", shapeReminder, (r) => ({ ...r, reminderList: { id: prims.id } })),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateRel(o, u, "reminders", ["Create", "Update", "Delete"], "many", shapeReminder, (r, i) => ({ ...r, reminderList: { id: i.id } })),
    }),
};

export type ReportShape = Pick<Report, "id" | "details" | "language" | "reason"> & {
    __typename: "Report";
    createdFor: { __typename: ReportFor, id: string };
    otherReason?: string | null;
}
export const shapeReport: ShapeModel<ReportShape, ReportCreateInput, ReportUpdateInput> = {
    create: (d) => ({
        createdForConnect: d.createdFor?.id ?? DUMMY_ID,
        createdForType: d.createdFor?.__typename ?? "RoutineVersion",
        reason: d.otherReason?.trim() || d.reason,
        ...createPrims(d, "id", "details", "language"),
    }),
    update: (o, u) => shapeUpdate(u, {
        reason: (u.otherReason ?? u.reason) !== o.reason ? (u.otherReason ?? u.reason) : undefined,
        ...updatePrims(o, u, "id", "details", "language"),
    }),
};

export type ReportResponseShape = Pick<ReportResponse, "id"> & {
    __typename: "ReportResponse";
}
export const shapeReportResponse: ShapeModel<ReportResponseShape, ReportResponseCreateInput, ReportResponseUpdateInput> = {
    create: (_d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any,
};

export type ResourceShape = Pick<Resource, "id" | "isInternal" | "isPrivate" | "permissions" | "resourceType"> & {
    __typename: "Resource";
    owner: OwnerShape | null | undefined;
    parent?: CanConnect<ResourceVersionShape> | null;
    publicId?: string | null;
    tags?: CanConnect<TagShape, "tag">[] | null;
    versions?: Omit<ResourceVersionShape, "root">[] | null;
}
export const shapeResource: ShapeModel<ResourceShape, ResourceCreateInput, ResourceUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isInternal", "isPrivate", "permissions", "publicId", "resourceType"),
        ...createOwner(d, "ownedBy"),
        ...createRel(d, "parent", ["Connect"], "one"),
        ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
        ...createVersion(d, shapeResourceVersion),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isInternal", "isPrivate", "permissions"),
        ...updateOwner(o, u, "ownedBy"),
        ...updateTagsRel(o, u, shapeTag),
        ...updateVersion(o, u, shapeResourceVersion),
    }),
};

export type ResourceVersionTranslationShape = Pick<ResourceVersionTranslation, "id" | "language" | "description" | "details" | "instructions" | "name"> & {
    __typename?: "ResourceVersionTranslation";
}
export type ResourceVersionShape = Pick<ResourceVersion, "id" | "codeLanguage" | "config" | "isAutomatable" | "isComplete" | "isPrivate" | "resourceSubType" | "versionLabel" | "versionNotes"> & {
    __typename: "ResourceVersion";
    publicId?: string | null;
    relatedVersions?: ResourceVersionRelationShape[] | null;
    root?: CanConnect<ResourceShape> | null;
    translations?: ResourceVersionTranslationShape[] | null;
}
export const shapeResourceVersionTranslation: ShapeModel<ResourceVersionTranslationShape, ResourceVersionTranslationCreateInput, ResourceVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", ["instructions", (instructions) => instructions ?? ""], "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "instructions", "name")),
};
export const shapeResourceVersion: ShapeModel<ResourceVersionShape, ResourceVersionCreateInput, ResourceVersionUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "codeLanguage", "config", "isAutomatable", "isComplete", "isPrivate", "publicId", "resourceSubType", "versionLabel", "versionNotes");
        return {
            ...prims,
            ...createRel(d, "relatedVersions", ["Create"], "many", shapeResourceVersionRelation),
            ...createRel(d, "root", ["Connect", "Create"], "one", shapeResource, (r) => ({ ...r, isPrivate: d.isPrivate })),
            ...createRel(d, "translations", ["Create"], "many", shapeResourceVersionTranslation),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "codeLanguage", "config", "isAutomatable", "isComplete", "isPrivate", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "relatedVersions", ["Create", "Update", "Delete"], "many", shapeResourceVersionRelation),
        ...updateRel(o, u, "root", ["Update"], "one", shapeResource),
        ...updateTransRel(o, u, shapeResourceVersionTranslation),
    }),
};

export type ResourceVersionRelationShape = Pick<ResourceVersionRelation, "id" | "labels"> & {
    __typename: "ResourceVersionRelation";
    toVersion: CanConnect<ResourceVersionShape>;
}
export const shapeResourceVersionRelation: ShapeModel<ResourceVersionRelation, ResourceVersionRelationCreateInput, ResourceVersionRelationUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "labels");
        return {
            ...prims,
            ...createRel(d, "toVersion", ["Connect"], "one"),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "labels"),
    }),
};

export type RunShape = Pick<Run, "id" | "isPrivate" | "completedComplexity" | "contextSwitches" | "data" | "name" | "startedAt" | "status" | "timeElapsed"> & {
    __typename: "Run";
    steps?: RunStepShape[] | null;
    io?: RunIOShape[] | null;
    schedule?: ScheduleShape | null;
    resourceVersion?: CanConnect<ResourceVersionShape> | null;
    team?: CanConnect<TeamShape> | null;
}
export const shapeRun: ShapeModel<RunShape, RunCreateInput, RunUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isPrivate", "completedComplexity", "contextSwitches", "data", "name", "startedAt", "status", "timeElapsed"),
        ...createRel(d, "steps", ["Create"], "many", shapeRunStep),
        ...createRel(d, "io", ["Create"], "many", shapeRunIO),
        ...createRel(d, "schedule", ["Create"], "one", shapeSchedule),
        ...createRel(d, "resourceVersion", ["Connect"], "one"),
        ...createRel(d, "team", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate", "completedComplexity", "contextSwitches", "data", "name", "startedAt", "status", "timeElapsed"),
        ...updateRel(o, u, "io", ["Create", "Update", "Delete"], "many", shapeRunIO),
        ...updateRel(o, u, "steps", ["Create", "Update", "Delete"], "many", shapeRunStep),
        ...updateRel(o, u, "schedule", ["Create", "Update", "Delete"], "one", shapeSchedule),
    }),
};

export type RunIOShape = Pick<RunIO, "id" | "data" | "nodeInputName" | "nodeName"> & {
    __typename: "RunIO";
    run: CanConnect<Run>;
}
export const shapeRunIO: ShapeModel<RunIOShape, RunIOCreateInput, RunIOUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "data", "nodeInputName", "nodeName"),
        ...createRel(d, "run", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "data"),
    }),
};

export type RunStepShape = Pick<RunStep, "id" | "completedAt" | "complexity" | "contextSwitches" | "name" | "nodeId" | "order" | "startedAt" | "status" | "resourceInId" | "timeElapsed"> & {
    __typename: "RunStep";
    run: CanConnect<RunShape>;
    resourceVersion?: CanConnect<ResourceVersionShape> | null;
}
export const shapeRunStep: ShapeModel<RunStepShape, RunStepCreateInput, RunStepUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "completedAt", "complexity", "contextSwitches", "name", "nodeId", "order", "startedAt", "status", "resourceInId", "timeElapsed"),
        ...createRel(d, "run", ["Connect"], "one"),
        ...createRel(d, "resourceVersion", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "contextSwitches", "status", "timeElapsed"),
    }),
};

export type ScheduleShape = Pick<Schedule, "id" | "startTime" | "endTime" | "timezone"> & {
    __typename: "Schedule";
    exceptions: ScheduleExceptionShape[];
    meeting?: CanConnect<MeetingShape> | null;
    recurrences: ScheduleRecurrenceShape[];
    run?: CanConnect<RunShape> | null;
}
export const shapeSchedule: ShapeModel<ScheduleShape, ScheduleCreateInput, ScheduleUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", ["startTime", shapeDate], ["endTime", shapeDate], "timezone");
        return {
            ...prims,
            ...createRel(d, "exceptions", ["Create"], "many", shapeScheduleException, (e) => ({
                ...e,
                schedule: { __typename: "Schedule" as const, id: prims.id },
            })),
            ...createRel(d, "meeting", ["Connect"], "one"),
            ...createRel(d, "recurrences", ["Create"], "many", shapeScheduleRecurrence, (e) => ({
                ...e,
                schedule: { __typename: "Schedule" as const, id: prims.id },
            })),
            ...createRel(d, "run", ["Connect"], "one"),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", ["startTime", shapeDate], ["endTime", shapeDate], "timezone"),
        ...updateRel(o, u, "exceptions", ["Create", "Update", "Delete"], "many", shapeScheduleException, (e, i) => ({
            ...e,
            schedule: { __typename: "Schedule" as const, id: i.id },
        })),
        ...updateRel(o, u, "recurrences", ["Create", "Update", "Delete"], "many", shapeScheduleRecurrence, (e, i) => ({
            ...e,
            schedule: { __typename: "Schedule" as const, id: i.id },
        })),
    }),
};

export type ScheduleExceptionShape = Pick<ScheduleException, "id" | "originalStartTime" | "newStartTime" | "newEndTime"> & {
    __typename: "ScheduleException";
    schedule: CanConnect<ScheduleShape>;
}
export const shapeScheduleException: ShapeModel<ScheduleExceptionShape, ScheduleExceptionCreateInput, ScheduleExceptionUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", ["originalStartTime", shapeDate], ["newStartTime", shapeDate], ["newEndTime", shapeDate]),
        ...createRel(d, "schedule", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", ["originalStartTime", shapeDate], ["newStartTime", shapeDate], ["newEndTime", shapeDate]),
    }),
};

export type ScheduleRecurrenceShape = Pick<ScheduleRecurrence, "id" | "recurrenceType" | "interval" | "dayOfMonth" | "dayOfWeek" | "duration" | "month" | "endDate"> & {
    __typename: "ScheduleRecurrence";
    schedule: CanConnect<ScheduleShape>;
}
export const shapeScheduleRecurrence: ShapeModel<ScheduleRecurrenceShape, ScheduleRecurrenceCreateInput, ScheduleRecurrenceUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "recurrenceType", "interval", "dayOfMonth", "dayOfWeek", "duration", "month", ["endDate", shapeDate]),
        ...createRel(d, "schedule", ["Connect", "Create"], "one", shapeSchedule),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "recurrenceType", "interval", "dayOfMonth", "dayOfWeek", "duration", "month", ["endDate", shapeDate]),
    }),
};

export type TagTranslationShape = Pick<TagTranslation, "id" | "language" | "description"> & {
    __typename?: "TagTranslation";
}
export type TagShape = Pick<Tag, "id" | "tag"> & {
    __typename: "Tag";
    anonymous?: boolean | null;
    translations?: TagTranslationShape[] | null;
    you?: Tag["you"] | null;
}
export const shapeTagTranslation: ShapeModel<TagTranslationShape, TagTranslationCreateInput, TagTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description")),
};
export const shapeTag: ShapeModel<TagShape, TagCreateInput, TagUpdateInput> = {
    idField: "tag",
    create: (d) => ({
        ...createPrims(d, "id", "tag"),
        ...createRel(d, "translations", ["Create"], "many", shapeTagTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "tag"),
        ...updateTransRel(o, u, shapeTagTranslation),
    }),
};

export type TeamTranslationShape = Pick<TeamTranslation, "id" | "language" | "bio" | "name"> & {
    __typename?: "TeamTranslation";
}
export type TeamShape = Pick<Team, "id" | "config" | "handle" | "isOpenToNewMembers" | "isPrivate"> & {
    __typename: "Team";
    bannerImage?: string | File | null;
    /** Invites for new members */
    memberInvites?: MemberInviteShape[] | null;
    /** Potentially non-exhaustive list of current members. Ignored by shapers */
    members?: MemberShape[];
    /** Members being removed from the team */
    membersDelete?: { id: string }[] | null;
    profileImage?: string | File | null;
    tags?: CanConnect<TagShape, "tag">[] | null;
    translations?: TeamTranslationShape[] | null;
}
export const shapeTeamTranslation: ShapeModel<TeamTranslationShape, TeamTranslationCreateInput, TeamTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "bio", "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "bio", "name")),
};
export const shapeTeam: ShapeModel<TeamShape, TeamCreateInput, TeamUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "bannerImage", "config", "handle", "isOpenToNewMembers", "isPrivate", "profileImage");
        return {
            ...prims,
            ...createRel(d, "memberInvites", ["Create"], "many", shapeMemberInvite),
            ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
            ...createRel(d, "translations", ["Create"], "many", shapeTeamTranslation),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "bannerImage", "config", "handle", "isOpenToNewMembers", "isPrivate", "profileImage"),
        ...updateRel(o, u, "memberInvites", ["Create", "Delete"], "many", shapeMemberInvite),
        ...updateTagsRel(o, u, shapeTag),
        ...updateTransRel(o, u, shapeTeamTranslation),
        ...(u.membersDelete ? { membersDelete: u.membersDelete.map(m => m.id) } : {}),
    }),
};

export type UserTranslationShape = Pick<UserTranslation, "id" | "language" | "bio"> & {
    __typename?: "UserTranslation";
}
export const shapeUserTranslation: ShapeModel<UserTranslationShape, UserTranslationCreateInput, UserTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "bio"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "language", "bio")),
};
