import { Api, ApiCreateInput, ApiUpdateInput, ApiVersion, ApiVersionCreateInput, ApiVersionTranslation, ApiVersionTranslationCreateInput, ApiVersionTranslationUpdateInput, ApiVersionUpdateInput, Bookmark, BookmarkCreateInput, BookmarkFor, BookmarkList, BookmarkListCreateInput, BookmarkListUpdateInput, BookmarkUpdateInput, BotCreateInput, BotUpdateInput, Chat, ChatCreateInput, ChatInvite, ChatInviteCreateInput, ChatInviteStatus, ChatInviteUpdateInput, ChatInviteYou, ChatMessage, ChatMessageCreateInput, ChatMessageParent, ChatMessageTranslation, ChatMessageTranslationCreateInput, ChatMessageTranslationUpdateInput, ChatMessageUpdateInput, ChatMessageYou, ChatParticipant, ChatParticipantUpdateInput, ChatTranslation, ChatTranslationCreateInput, ChatTranslationUpdateInput, ChatUpdateInput, Code, CodeCreateInput, CodeUpdateInput, CodeVersion, CodeVersionCreateInput, CodeVersionTranslation, CodeVersionTranslationCreateInput, CodeVersionTranslationUpdateInput, CodeVersionUpdateInput, Comment, CommentCreateInput, CommentFor, CommentTranslation, CommentTranslationCreateInput, CommentTranslationUpdateInput, CommentUpdateInput, CommentedOn, FocusMode, FocusModeCreateInput, FocusModeFilter, FocusModeFilterCreateInput, FocusModeUpdateInput, Issue, IssueCreateInput, IssueFor, IssueTranslation, IssueTranslationCreateInput, IssueTranslationUpdateInput, IssueUpdateInput, Label, LabelCreateInput, LabelTranslation, LabelTranslationCreateInput, LabelTranslationUpdateInput, LabelUpdateInput, Meeting, MeetingCreateInput, MeetingInvite, MeetingInviteCreateInput, MeetingInviteUpdateInput, MeetingTranslation, MeetingTranslationCreateInput, MeetingTranslationUpdateInput, MeetingUpdateInput, Member, MemberInvite, MemberInviteCreateInput, MemberInviteUpdateInput, MemberUpdateInput, Node, NodeCreateInput, NodeEnd, NodeEndCreateInput, NodeEndUpdateInput, NodeLink, NodeLinkCreateInput, NodeLinkUpdateInput, NodeLinkWhen, NodeLinkWhenCreateInput, NodeLinkWhenTranslation, NodeLinkWhenTranslationCreateInput, NodeLinkWhenTranslationUpdateInput, NodeLinkWhenUpdateInput, NodeRoutineList, NodeRoutineListCreateInput, NodeRoutineListItem, NodeRoutineListItemCreateInput, NodeRoutineListItemTranslation, NodeRoutineListItemTranslationCreateInput, NodeRoutineListItemTranslationUpdateInput, NodeRoutineListItemUpdateInput, NodeRoutineListUpdateInput, NodeTranslation, NodeTranslationCreateInput, NodeTranslationUpdateInput, NodeUpdateInput, Note, NoteCreateInput, NotePage, NotePageCreateInput, NotePageUpdateInput, NoteUpdateInput, NoteVersion, NoteVersionCreateInput, NoteVersionTranslation, NoteVersionTranslationCreateInput, NoteVersionTranslationUpdateInput, NoteVersionUpdateInput, Post, PostCreateInput, PostUpdateInput, ProfileUpdateInput, Project, ProjectCreateInput, ProjectUpdateInput, ProjectVersion, ProjectVersionCreateInput, ProjectVersionDirectory, ProjectVersionDirectoryCreateInput, ProjectVersionDirectoryUpdateInput, ProjectVersionTranslation, ProjectVersionTranslationCreateInput, ProjectVersionTranslationUpdateInput, ProjectVersionUpdateInput, PullRequest, PullRequestCreateInput, PullRequestTranslation, PullRequestTranslationCreateInput, PullRequestTranslationUpdateInput, PullRequestUpdateInput, Question, QuestionAnswer, QuestionAnswerCreateInput, QuestionAnswerUpdateInput, QuestionCreateInput, QuestionForType, QuestionTranslation, QuestionTranslationCreateInput, QuestionTranslationUpdateInput, QuestionUpdateInput, Quiz, QuizAttempt, QuizAttemptCreateInput, QuizAttemptUpdateInput, QuizCreateInput, QuizQuestion, QuizQuestionCreateInput, QuizQuestionResponse, QuizQuestionResponseCreateInput, QuizQuestionResponseUpdateInput, QuizQuestionUpdateInput, QuizUpdateInput, ReactionSummary, Reminder, ReminderCreateInput, ReminderItem, ReminderItemCreateInput, ReminderItemUpdateInput, ReminderList, ReminderListCreateInput, ReminderListUpdateInput, ReminderUpdateInput, Report, ReportCreateInput, ReportFor, ReportResponse, ReportResponseCreateInput, ReportResponseUpdateInput, ReportUpdateInput, Resource, ResourceCreateInput, ResourceList, ResourceListCreateInput, ResourceListFor, ResourceListTranslation, ResourceListTranslationCreateInput, ResourceListTranslationUpdateInput, ResourceListUpdateInput, ResourceTranslation, ResourceTranslationCreateInput, ResourceTranslationUpdateInput, ResourceUpdateInput, Role, RoleCreateInput, RoleTranslation, RoleTranslationCreateInput, RoleTranslationUpdateInput, RoleUpdateInput, Routine, RoutineCreateInput, RoutineUpdateInput, RoutineVersion, RoutineVersionCreateInput, RoutineVersionInput, RoutineVersionInputCreateInput, RoutineVersionInputTranslation, RoutineVersionInputTranslationCreateInput, RoutineVersionInputTranslationUpdateInput, RoutineVersionInputUpdateInput, RoutineVersionOutput, RoutineVersionOutputCreateInput, RoutineVersionOutputTranslation, RoutineVersionOutputTranslationCreateInput, RoutineVersionOutputTranslationUpdateInput, RoutineVersionOutputUpdateInput, RoutineVersionTranslation, RoutineVersionTranslationCreateInput, RoutineVersionTranslationUpdateInput, RoutineVersionUpdateInput, RunProject, RunProjectCreateInput, RunProjectStep, RunProjectStepCreateInput, RunProjectStepUpdateInput, RunProjectUpdateInput, RunRoutine, RunRoutineCreateInput, RunRoutineInput, RunRoutineInputCreateInput, RunRoutineInputUpdateInput, RunRoutineStep, RunRoutineStepCreateInput, RunRoutineStepUpdateInput, RunRoutineUpdateInput, Schedule, ScheduleCreateInput, ScheduleException, ScheduleExceptionCreateInput, ScheduleExceptionUpdateInput, ScheduleRecurrence, ScheduleRecurrenceCreateInput, ScheduleRecurrenceUpdateInput, ScheduleUpdateInput, Standard, StandardCreateInput, StandardUpdateInput, StandardVersion, StandardVersionCreateInput, StandardVersionTranslation, StandardVersionTranslationCreateInput, StandardVersionTranslationUpdateInput, StandardVersionUpdateInput, Tag, TagCreateInput, TagTranslation, TagTranslationCreateInput, TagTranslationUpdateInput, TagUpdateInput, Team, TeamCreateInput, TeamTranslation, TeamTranslationCreateInput, TeamTranslationUpdateInput, TeamUpdateInput, User, UserTranslation, UserTranslationCreateInput, UserTranslationUpdateInput } from "../../api/generated/graphqlTypes";
import { CanConnect, ShapeModel } from "../../consts/commonTypes";
import { DUMMY_ID } from "../../id/uuid";
import { LlmModel } from "../../utils/bot";
import { addHttps } from "../../validation/utils/builders/addHttps";
import { hasObjectChanged } from "../general/objectTools";
import { createOwner, createPrims, createRel, createVersion, shapeDate, shapeUpdate, updateOwner, updatePrims, updateRel, updateTranslationPrims, updateVersion } from "./tools";
import { OwnerShape } from "./types";

export type ApiShape = Pick<Api, "id" | "isPrivate"> & {
    __typename: "Api";
    labels?: CanConnect<LabelShape>[] | null;
    owner: CanConnect<OwnerShape> | null | undefined;
    parent?: CanConnect<ApiVersionShape> | null;
    tags?: CanConnect<TagShape, "tag">[] | null;
    versions?: Omit<ApiVersionShape, "root">[] | null;
}
export const shapeApi: ShapeModel<ApiShape, ApiCreateInput, ApiUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isPrivate"),
        ...createOwner(d, "ownedBy"),
        ...createRel(d, "labels", ["Connect", "Create"], "many", shapeLabel),
        ...createRel(d, "parent", ["Connect"], "one"),
        ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
        ...createVersion(d, shapeApiVersion),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate"),
        ...updateOwner(o, u, "ownedBy"),
        ...updateRel(o, u, "labels", ["Connect", "Create", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateVersion(o, u, shapeApiVersion),
    }),
};

export type ApiVersionTranslationShape = Pick<ApiVersionTranslation, "id" | "language" | "details" | "name" | "summary"> & {
    __typename?: "ApiVersionTranslation";
}
export type ApiVersionShape = Pick<ApiVersion, "id" | "callLink" | "documentationLink" | "isComplete" | "isPrivate" | "schemaLanguage" | "schemaText" | "versionLabel" | "versionNotes"> & {
    __typename: "ApiVersion";
    directoryListings?: CanConnect<ProjectVersionDirectoryShape>[] | null;
    resourceList?: ResourceListShape | null;
    root?: CanConnect<ApiShape> | null;
    translations?: ApiVersionTranslationShape[] | null;
}
export const shapeApiVersionTranslation: ShapeModel<ApiVersionTranslationShape, ApiVersionTranslationCreateInput, ApiVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "details", "name", "summary"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "details", "summary")),
};
export const shapeApiVersion: ShapeModel<ApiVersionShape, ApiVersionCreateInput, ApiVersionUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "callLink", "documentationLink", "isComplete", "isPrivate", "schemaLanguage", "schemaText", "versionLabel", "versionNotes");
        return {
            ...prims,
            ...createRel(d, "directoryListings", ["Connect"], "many"),
            ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: prims.id, __typename: "ApiVersion" } })),
            ...createRel(d, "root", ["Connect", "Create"], "one", shapeApi, (r) => ({ ...r, isPrivate: prims.isPrivate })),
            ...createRel(d, "translations", ["Create"], "many", shapeApiVersionTranslation),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "callLink", "documentationLink", "isComplete", "isPrivate", "schemaLanguage", "schemaText", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "directoryListings", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: o.id, __typename: "ApiVersion" } })),
        ...updateRel(o, u, "root", ["Update"], "one", shapeApi),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeApiVersionTranslation),
    }),
};

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

/** Translation for bot-specific properties (which are stringified and stored in `botSettings` field) */
export type BotTranslationShape = {
    id: string;
    language: string;
    bias?: string | null;
    bio?: string | null;
    domainKnowledge?: string | null;
    keyPhrases?: string | null;
    occupation?: string | null;
    persona?: string | null;
    startingMessage?: string | null;
    tone?: string | null;
}

export type BotShape = Pick<User, "id" | "handle" | "isBotDepictingPerson" | "isPrivate" | "name"> & {
    __typename: "User";
    bannerImage?: string | File | null;
    creativity?: number | null;
    isBot?: true;
    model: LlmModel["value"],
    profileImage?: string | File | null;
    translations?: BotTranslationShape[] | null;
    verbosity?: number | null;
}

export const shapeBotTranslation: ShapeModel<BotTranslationShape, Record<string, string | number>, Record<string, string | number>> = {
    create: (d) => createPrims(d, "language", "bias", "domainKnowledge", "keyPhrases", "occupation", "persona", "startingMessage", "tone"),
    /** 
     * Unlike typical updates, we want to include every field so that 
     * we can stringify the entire object and store it in the `botSettings` field. 
     * This means we'll use `createPrims` again.
     **/
    update: (_, u) => createPrims(u, "language", "bias", "domainKnowledge", "keyPhrases", "occupation", "persona", "startingMessage", "tone"),
};

export const shapeBot: ShapeModel<BotShape, BotCreateInput, BotUpdateInput> = {
    create: (d) => {
        // Extract bot settings from translations
        const textData = createRel(d, "translations", ["Create"], "many", shapeBotTranslation);
        // Convert to object, where keys are language codes and values are the bot settings
        const textSettings = Object.fromEntries(textData.translationsCreate?.map(({ language, ...rest }) => [language, rest]) ?? []);
        return {
            isBot: true,
            botSettings: JSON.stringify({
                translations: textSettings,
                model: d.model,
                creativity: d.creativity ?? undefined,
                verbosity: d.verbosity ?? undefined,
            }),
            ...createPrims(d, "id", "bannerImage", "handle", "isBotDepictingPerson", "isPrivate", "name", "profileImage"),
            ...createRel(d, "translations", ["Create"], "many", shapeUserTranslation),
        };
    },
    update: (o, u) => {
        // Extract bot settings from translations. 
        // NOTE: We're using createRel again because we want to include every field
        const textData = createRel(u, "translations", ["Create"], "many", shapeBotTranslation);
        // Convert created to object, where keys are language codes and values are the bot settings
        const textSettings = Object.fromEntries(textData.translationsCreate?.map(({ language, ...rest }) => [language, rest]) ?? []);
        // Since we set the original to empty array, we need to manually remove the deleted translations (i.e. translations in the original but not in the update)
        const deletedTranslations = o.translations?.filter(t => !u.translations?.some(t2 => t2.id === t.id));
        if (deletedTranslations) {
            deletedTranslations.forEach(t => delete textSettings[t.language]);
        }
        return shapeUpdate(u, {
            botSettings: JSON.stringify({
                translations: textSettings,
                model: u.model,
                creativity: u.creativity ?? undefined,
                verbosity: u.verbosity ?? undefined,
            }),
            ...updatePrims(o, u, "id", "bannerImage", "handle", "isBotDepictingPerson", "isPrivate", "name", "profileImage"),
            ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeUserTranslation),
        });
    },
};

export type ChatTranslationShape = Pick<ChatTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "ChatTranslation";
}
export type ChatShape = Pick<Chat, "id" | "openToAnyoneWithInvite"> & {
    __typename: "Chat";
    /** Invites for new participants */
    invites: ChatInviteShape[];
    labels?: CanConnect<LabelShape>[] | null;
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
            ...createRel(d, "labels", ["Connect", "Create"], "many", shapeLabel),
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
        ...updateRel(o, u, "labels", ["Connect", "Create", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "messages", ["Create", "Update", "Delete"], "many", shapeChatMessage, (m, i) => ({ ...m, chat: { id: i.id } })),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeChatTranslation),
        ...(u.participantsDelete?.length ? { participantsDelete: u.participantsDelete.map(m => m.id) } : {}),
    }),
};

export type ChatInviteShape = Pick<ChatInvite, "id" | "message"> & {
    __typename: "ChatInvite";
    created_at: string; // Only used by the UI
    updated_at: string; // Only used by the UI
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

export type ChatMessageTranslationShape = Pick<ChatMessageTranslation, "id" | "language" | "text"> & {
    __typename?: "ChatMessageTranslation";
}
export type ChatMessageStatus = "unsent" | "editing" | "sending" | "sent" | "failed";
export type ChatMessageShape = Pick<ChatMessage, "id" | "versionIndex"> & {
    __typename: "ChatMessage";
    created_at: string; // Only used by the UI
    updated_at: string; // Only used by the UI
    chat?: CanConnect<ChatShape> | null;
    sequence?: number; // Only used by the UI
    /** If not provided, we'll assume it's sent */
    status?: ChatMessageStatus;
    parent?: CanConnect<ChatMessageParent> | null;
    reactionSummaries: ReactionSummary[]; // Only used by the UI
    translations: ChatMessageTranslationShape[];
    user?: CanConnect<User> | null;
    you?: ChatMessageYou; // Only used by the UI
}
export const shapeChatMessageTranslation: ShapeModel<ChatMessageTranslationShape, ChatMessageTranslationCreateInput, ChatMessageTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "text"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "text")),
};
export const shapeChatMessage: ShapeModel<ChatMessageShape, ChatMessageCreateInput, ChatMessageUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "versionIndex"),
        ...createRel(d, "chat", ["Connect"], "one"),
        ...createRel(d, "translations", ["Create"], "many", shapeChatMessageTranslation),
        ...createRel(d, "user", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeChatMessageTranslation),
    }),
};

export type ChatParticipantShape = Pick<ChatParticipant, "id"> & {
    __typename: "ChatParticipant";
    user: Pick<User, "__typename" | "updated_at" | "handle" | "id" | "isBot" | "name" | "profileImage">;
}
export const shapeChatParticipant: ShapeModel<ChatParticipantShape, null, ChatParticipantUpdateInput> = {
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
    }),
};

export type CodeShape = Pick<Code, "id" | "isPrivate" | "permissions"> & {
    __typename: "Code";
    labels?: CanConnect<LabelShape>[] | null;
    owner: OwnerShape | null | undefined;
    parent?: CanConnect<CodeVersionShape> | null;
    tags?: CanConnect<TagShape, "tag">[] | null;
    versions?: Omit<CodeVersionShape, "root">[] | null;
}
export const shapeCode: ShapeModel<CodeShape, CodeCreateInput, CodeUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isPrivate", "permissions"),
        ...createOwner(d, "ownedBy"),
        ...createRel(d, "labels", ["Connect", "Create"], "many", shapeLabel),
        ...createRel(d, "parent", ["Connect"], "one"),
        ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
        ...createVersion(d, shapeCodeVersion),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate", "permissions"),
        ...updateOwner(o, u, "ownedBy"),
        ...updateRel(o, u, "labels", ["Connect", "Create", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateVersion(o, u, shapeCodeVersion),
    }),
};

export type CodeVersionTranslationShape = Pick<CodeVersionTranslation, "id" | "language" | "description" | "name" | "jsonVariable"> & {
    __typename?: "CodeVersionTranslation";
}
export type CodeVersionShape = Pick<CodeVersion, "id" | "calledByRoutineVersionsCount" | "codeLanguage" | "codeType" | "content" | "default" | "isComplete" | "isPrivate" | "versionLabel" | "versionNotes"> & {
    __typename: "CodeVersion";
    directoryListings?: ProjectVersionDirectoryShape[] | null;
    resourceList?: ResourceListShape | null;
    root?: CanConnect<CodeShape> | null;
    suggestedNextByCode?: CanConnect<CodeVersionShape>[] | null;
    translations?: CodeVersionTranslationShape[] | null;
}
export const shapeCodeVersionTranslation: ShapeModel<CodeVersionTranslationShape, CodeVersionTranslationCreateInput, CodeVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name", "jsonVariable"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "language", "description", "name", "jsonVariable")),
};
export const shapeCodeVersion: ShapeModel<CodeVersionShape, CodeVersionCreateInput, CodeVersionUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "codeLanguage", "codeType", "content", "default", "isComplete", "isPrivate", "versionLabel", "versionNotes");
        return {
            ...prims,
            ...createRel(d, "directoryListings", ["Create"], "many", shapeProjectVersionDirectory),
            ...createRel(d, "root", ["Connect", "Create"], "one", shapeCode, (r) => ({ ...r, isPrivate: prims.isPrivate })),
            ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: prims.id, __typename: "CodeVersion" } })),
            ...createRel(d, "translations", ["Create"], "many", shapeCodeVersionTranslation),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "codeLanguage", "content", "default", "isPrivate", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "directoryListings", ["Create", "Update", "Delete"], "many", shapeProjectVersionDirectory),
        ...updateRel(o, u, "root", ["Update"], "one", shapeCode),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: o.id, __typename: "CodeVersion" } })),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeCodeVersionTranslation),
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
    create: (d) => ({
        ...createPrims(d, "id", "threadId"),
        createdFor: d.commentedOn.__typename as CommentFor,
        forConnect: d.commentedOn.id,
        ...createRel(d, "translations", ["Create"], "many", shapeCommentTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeCommentTranslation),
    }),
};

export type FocusModeShape = Pick<FocusMode, "id" | "name" | "description"> & {
    __typename: "FocusMode",
    reminderList?: ReminderListShape | null,
    resourceList?: ResourceListShape | null;
    labels?: LabelShape[] | null,
    filters?: FocusModeFilterShape[] | null,
    schedule?: ScheduleShape | null,
}
export const shapeFocusMode: ShapeModel<FocusModeShape, FocusModeCreateInput, FocusModeUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "name", "description");
        return {
            ...prims,
            ...createRel(d, "reminderList", ["Create", "Connect"], "one", shapeReminderList),
            ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: prims.id, __typename: "FocusMode" } })),
            ...createRel(d, "labels", ["Create", "Connect"], "many", shapeLabel),
            ...createRel(d, "filters", ["Create"], "many", shapeFocusModeFilter),
            ...createRel(d, "schedule", ["Create"], "one", shapeSchedule),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "name", "description"),
        ...updateRel(o, u, "reminderList", ["Create", "Connect", "Disconnect", "Update"], "one", shapeReminderList),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: o.id, __typename: "FocusMode" } })),
        ...updateRel(o, u, "labels", ["Create", "Connect", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "filters", ["Create", "Delete"], "many", shapeFocusModeFilter),
        ...updateRel(o, u, "schedule", ["Create", "Update"], "one", shapeSchedule),
    }),
};

export type FocusModeFilterShape = Pick<FocusModeFilter, "id" | "filterType"> & {
    __typename: "FocusModeFilter";
    focusMode: CanConnect<FocusModeShape>;
    tag: TagShape,
}
export const shapeFocusModeFilter: ShapeModel<FocusModeFilterShape, FocusModeFilterCreateInput, null> = {
    create: (d) => ({
        ...createPrims(d, "id", "filterType"),
        ...createRel(d, "focusMode", ["Connect"], "one"),
        ...createRel(d, "tag", ["Create", "Connect"], "one", shapeTag),
    }),
};

export type IssueTranslationShape = Pick<IssueTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "IssueTranslation";
}
export type IssueShape = Pick<Issue, "id"> & {
    __typename: "Issue";
    issueFor: IssueFor;
    for: { id: string };
    labels?: CanConnect<LabelShape>[] | null;
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
        ...createRel(d, "labels", ["Connect", "Create"], "many", shapeLabel),
        ...createRel(d, "translations", ["Create"], "many", shapeIssueTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateRel(o, u, "labels", ["Connect", "Disconnect", "Create"], "many", shapeLabel),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeIssueTranslation),
    }),
};

export type LabelTranslationShape = Pick<LabelTranslation, "id" | "language" | "description"> & {
    __typename?: "LabelTranslation";
}
export type LabelShape = Pick<Label, "id" | "label" | "color"> & {
    __typename: "Label";
    team?: CanConnect<TeamShape> | null; // If no team specified, assumes current user
    translations: LabelTranslationShape[];
    // Connects and disconnects of labels to other objects are handled separately, or in parent shape
}
export const shapeLabelTranslation: ShapeModel<LabelTranslationShape, LabelTranslationCreateInput, LabelTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description")),
};
export const shapeLabel: ShapeModel<LabelShape, LabelCreateInput, LabelUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "label", "color"),
        ...createRel(d, "team", ["Connect"], "one"),
        ...createRel(d, "translations", ["Create"], "many", shapeLabelTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "label", "color"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeLabelTranslation),
    }),
};

export type MeetingTranslationShape = Pick<MeetingTranslation, "id" | "language" | "description" | "link" | "name"> & {
    __typename?: "MeetingTranslation";
}
export type MeetingShape = Pick<Meeting, "id" | "openToAnyoneWithInvite" | "showOnTeamProfile"> & {
    __typename: "Meeting";
    restrictedToRoles?: CanConnect<RoleShape>[] | null;
    invites?: CanConnect<MeetingInviteShape>[] | null;
    labels?: CanConnect<LabelShape>[] | null;
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
            ...createRel(d, "restrictedToRoles", ["Connect"], "many"),
            ...createRel(d, "invites", ["Create"], "many", shapeMeetingInvite, (i) => ({ ...i, meeting: { id: prims.id } })),
            ...createRel(d, "labels", ["Connect"], "many"),
            ...createRel(d, "schedule", ["Create"], "one", shapeSchedule),
            ...createRel(d, "team", ["Connect"], "one"),
            ...createRel(d, "translations", ["Create"], "many", shapeMeetingTranslation),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "openToAnyoneWithInvite", "showOnTeamProfile"),
        ...updateRel(o, u, "restrictedToRoles", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "invites", ["Create", "Update", "Delete"], "many", shapeMeetingInvite, (i) => ({ ...i, meeting: { id: o.id } })),
        ...updateRel(o, u, "labels", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "schedule", ["Create", "Update"], "one", shapeSchedule),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeMeetingTranslation),
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
    user: Pick<User, "updated_at" | "handle" | "id" | "isBot" | "name" | "profileImage">;
}
export const shapeMember: ShapeModel<MemberShape, null, MemberUpdateInput> = {
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isAdmin", "permissions"),
    }),
};

export type MemberInviteShape = Pick<MemberInvite, "id" | "message" | "willBeAdmin" | "willHavePermissions"> & {
    __typename: "MemberInvite";
    team: CanConnect<TeamShape>;
    user: Pick<User, "__typename" | "updated_at" | "handle" | "id" | "isBot" | "name" | "profileImage">;
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

export type NodeTranslationShape = Pick<NodeTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "NodeTranslation";
}
export type NodeShape = Pick<Node, "id" | "columnIndex" | "rowIndex" | "nodeType"> & {
    __typename: "Node";
    // loop?: LoopShape | null
    end?: NodeEndShape | null;
    routineList?: NodeRoutineListShape | null;
    routineVersion: CanConnect<Omit<RoutineVersionShape, "nodes">>;
    translations: NodeTranslationShape[];
}
export const shapeNodeTranslation: ShapeModel<NodeTranslationShape, NodeTranslationCreateInput, NodeTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "name")),
};
export const shapeNode: ShapeModel<NodeShape, NodeCreateInput, NodeUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "columnIndex", "nodeType", "rowIndex");
        return {
            ...prims,
            ...createRel(d, "routineVersion", ["Connect"], "one"),
            // ...createRel(d, "loop", ['Create'], "one", shapeLoop, (n) => ({ node: { id: prims.id }, ...n })),
            ...createRel(d, "end", ["Create"], "one", shapeNodeEnd, (n) => ({ node: { id: prims.id }, ...n })),
            ...createRel(d, "routineList", ["Create"], "one", shapeNodeRoutineList, (n) => ({ node: { id: prims.id }, ...n })),
            ...createRel(d, "translations", ["Create"], "many", shapeNodeTranslation),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "columnIndex", "nodeType", "rowIndex"),
        ...updateRel(o, u, "routineVersion", ["Connect"], "one"),
        // ...updateRel(o, u, "loop", ['Create', 'Update', 'Delete'], "one", shapeLoop, (n, i) => ({ node: { id: i.id }, ...n })),
        ...updateRel(o, u, "end", ["Create", "Update"], "one", shapeNodeEnd, (n, i) => ({ node: { id: i.id }, ...n })),
        ...updateRel(o, u, "routineList", ["Create", "Update"], "one", shapeNodeRoutineList, (n, i) => ({ node: { id: i.id }, ...n })),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeNodeTranslation),
    }),
};

export type NodeEndShape = Pick<NodeEnd, "id" | "wasSuccessful" | "suggestedNextRoutineVersions"> & {
    __typename: "NodeEnd";
    node: CanConnect<Omit<NodeShape, "end">>;
    suggestedNextRoutineVersions?: CanConnect<RoutineVersionShape>[] | null;
}
export const shapeNodeEnd: ShapeModel<NodeEndShape, NodeEndCreateInput, NodeEndUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "wasSuccessful"),
        ...createRel(d, "node", ["Connect"], "one"),
        ...createRel(d, "suggestedNextRoutineVersions", ["Connect"], "many"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "wasSuccessful"),
        ...updateRel(o, u, "suggestedNextRoutineVersions", ["Connect", "Disconnect"], "many"),
    }),
};

export type NodeLinkShape = Pick<NodeLink, "id" | "operation"> & {
    __typename: "NodeLink";
    from: CanConnect<NodeShape>;
    to: CanConnect<NodeShape>;
    routineVersion: CanConnect<RoutineVersionShape>;
    whens?: NodeLinkWhenShape[];
}
export const shapeNodeLink: ShapeModel<NodeLinkShape, NodeLinkCreateInput, NodeLinkUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "operation"),
        ...createRel(d, "from", ["Connect"], "one"),
        ...createRel(d, "to", ["Connect"], "one"),
        ...createRel(d, "routineVersion", ["Connect"], "one"),
        ...createRel(d, "whens", ["Create"], "many", shapeNodeLinkWhen),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "operation"),
        ...updateRel(o, u, "from", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "to", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "whens", ["Create", "Update", "Delete"], "many", shapeNodeLinkWhen),
    }),
};

export type NodeLinkWhenTranslationShape = Pick<NodeLinkWhenTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "NodeLinkWhenTranslation";
}
export type NodeLinkWhenShape = Pick<NodeLinkWhen, "id" | "condition"> & {
    __typename: "NodeLinkWhen";
    link: CanConnect<NodeLinkShape>;
    translations?: NodeLinkWhenTranslationShape[] | null;
}
export const shapeNodeLinkWhenTranslation: ShapeModel<NodeLinkWhenTranslationShape, NodeLinkWhenTranslationCreateInput, NodeLinkWhenTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "name")),
};
export const shapeNodeLinkWhen: ShapeModel<NodeLinkWhenShape, NodeLinkWhenCreateInput, NodeLinkWhenUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "condition"),
        ...createRel(d, "link", ["Connect"], "one"),
        ...createRel(d, "translations", ["Create"], "many", shapeNodeLinkWhenTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "condition"),
        ...updateRel(o, u, "link", ["Connect"], "one"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeNodeLinkWhenTranslation),
    }),
};

export type NodeRoutineListShape = Pick<NodeRoutineList, "id" | "isOptional" | "isOrdered"> & {
    __typename: "NodeRoutineList";
    items: NodeRoutineListItemShape[];
    node: CanConnect<Omit<NodeShape, "routineList">>;
}
export const shapeNodeRoutineList: ShapeModel<NodeRoutineListShape, NodeRoutineListCreateInput, NodeRoutineListUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "isOptional", "isOrdered");
        return {
            ...prims,
            ...createRel(d, "items", ["Create"], "many", shapeNodeRoutineListItem, (r) => ({ list: { id: prims.id }, ...r })),
            ...createRel(d, "node", ["Connect"], "one"),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isOptional", "isOrdered"),
        ...updateRel(o, u, "items", ["Create", "Update", "Delete"], "many", shapeNodeRoutineListItem, (r, i) => ({ list: { id: i.id }, ...r })),
    }),
};

export type NodeRoutineListItemTranslationShape = Pick<NodeRoutineListItemTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "NodeRoutineListItemTranslation";
}
export type NodeRoutineListItemShape = Pick<NodeRoutineListItem, "id" | "index" | "isOptional"> & {
    __typename: "NodeRoutineListItem";
    list: CanConnect<NodeRoutineListShape>;
    routineVersion: RoutineVersionShape;
    translations: NodeRoutineListItemTranslationShape[];
}
export const shapeNodeRoutineListItemTranslation: ShapeModel<NodeRoutineListItemTranslationShape, NodeRoutineListItemTranslationCreateInput, NodeRoutineListItemTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "name")),
};
export const shapeNodeRoutineListItem: ShapeModel<NodeRoutineListItemShape, NodeRoutineListItemCreateInput, NodeRoutineListItemUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "index", "isOptional"),
        ...createRel(d, "list", ["Connect"], "one"),
        ...createRel(d, "routineVersion", ["Connect"], "one"),
        ...createRel(d, "translations", ["Create"], "many", shapeNodeRoutineListItemTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "index", "isOptional"),
        ...updateRel(o, u, "routineVersion", ["Update"], "one", shapeRoutineVersion),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeNodeRoutineListItemTranslation),
    }),
};

export type NoteShape = Pick<Note, "id" | "isPrivate"> & {
    __typename: "Note";
    labels?: CanConnect<LabelShape>[] | null;
    owner: OwnerShape | null | undefined;
    parent?: CanConnect<NoteVersionShape> | null;
    tags?: CanConnect<TagShape, "tag">[] | null;
    versions?: Omit<NoteVersionShape, "root">[] | null;
}
export const shapeNote: ShapeModel<NoteShape, NoteCreateInput, NoteUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isPrivate"),
        ...createOwner(d, "ownedBy"),
        ...createRel(d, "labels", ["Connect", "Create"], "many", shapeLabel),
        ...createRel(d, "parent", ["Connect"], "one"),
        ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
        ...createVersion(d, shapeNoteVersion),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate"),
        ...updateOwner(o, u, "ownedBy"),
        ...updateRel(o, u, "labels", ["Connect", "Create", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateVersion(o, u, shapeNoteVersion),
    }),
};

export type NotePageShape = Pick<NotePage, "id" | "pageIndex" | "text"> & {
    __typename: "NotePage";
}
export type NoteVersionTranslationShape = Pick<NoteVersionTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "NoteVersionTranslation";
    pages?: NotePageShape[] | null;
}
export type NoteVersionShape = Pick<NoteVersion, "id" | "isPrivate" | "versionLabel" | "versionNotes"> & {
    __typename: "NoteVersion";
    directoryListings?: CanConnect<ProjectVersionDirectoryShape>[] | null;
    resourceList?: CanConnect<ResourceListShape> | null;
    root?: CanConnect<NoteShape> | null;
    translations?: NoteVersionTranslationShape[] | null;
}
export const shapeNotePage: ShapeModel<NotePageShape, NotePageCreateInput, NotePageUpdateInput> = {
    create: (d) => createPrims(d, "id", "pageIndex", "text"),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, "id", "pageIndex", "text")),
};
export const shapeNoteVersionTranslation: ShapeModel<NoteVersionTranslationShape, NoteVersionTranslationCreateInput, NoteVersionTranslationUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "language", "description", "name"),
        ...createRel(d, "pages", ["Create"], "many", shapeNotePage),
    }),
    update: (o, u) => shapeUpdate(u, ({
        ...updatePrims(o, u, "id", "language", "description", "name"),
        ...updateRel(o, u, "pages", ["Create", "Update", "Delete"], "many", shapeNotePage),
    })),
};
export const shapeNoteVersion: ShapeModel<NoteVersionShape, NoteVersionCreateInput, NoteVersionUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "isPrivate", "versionLabel", "versionNotes");
        return {
            ...prims,
            ...createRel(d, "directoryListings", ["Connect"], "many"),
            ...createRel(d, "root", ["Connect", "Create"], "one", shapeNote, (r) => ({ ...r, isPrivate: prims.isPrivate })),
            ...createRel(d, "translations", ["Create"], "many", shapeNoteVersionTranslation),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "directoryListings", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "root", ["Update"], "one", shapeNote),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeNoteVersionTranslation),
    }),
};

export type PostShape = Pick<Post, "id"> & {
    __typename: "Post";
}
export const shapePost: ShapeModel<PostShape, PostCreateInput, PostUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any,
};

export type ProfileTranslationShape = Pick<UserTranslation, "id" | "language" | "bio"> & {
    __typename?: "UserTranslation";
}
export type ProfileShape = Partial<Pick<User, "handle" | "isPrivate" | "isPrivateApis" | "isPrivateApisCreated" | "isPrivateMemberships" | "isPrivateTeamsCreated" | "isPrivateProjects" | "isPrivateProjectsCreated" | "isPrivatePullRequests" | "isPrivateQuestionsAnswered" | "isPrivateQuestionsAsked" | "isPrivateQuizzesCreated" | "isPrivateRoles" | "isPrivateRoutines" | "isPrivateRoutinesCreated" | "isPrivateStandards" | "isPrivateStandardsCreated" | "isPrivateBookmarks" | "isPrivateVotes" | "name" | "theme">> & {
    __typename: "User",
    id: string;
    bannerImage?: string | File | null;
    focusModes?: FocusModeShape[] | null;
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
            "isPrivate",
            "isPrivateApis",
            "isPrivateApisCreated",
            "isPrivateMemberships",
            "isPrivateProjects",
            "isPrivateProjectsCreated",
            "isPrivatePullRequests",
            "isPrivateQuestionsAnswered",
            "isPrivateQuestionsAsked",
            "isPrivateQuizzesCreated",
            "isPrivateRoles",
            "isPrivateRoutines",
            "isPrivateRoutinesCreated",
            "isPrivateStandards",
            "isPrivateStandardsCreated",
            "isPrivateTeamsCreated",
            "isPrivateBookmarks",
            "isPrivateVotes",
            "name",
            "profileImage",
            "theme",
        ),
        ...updateRel(o, u, "focusModes", ["Create", "Update", "Delete"], "many", shapeFocusMode),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeProfileTranslation),
    }),
};

export type ProjectShape = Pick<Project, "id" | "handle" | "isPrivate" | "permissions"> & {
    __typename: "Project";
    labels?: CanConnect<LabelShape>[] | null;
    owner: OwnerShape | null | undefined;
    parent?: CanConnect<ProjectVersionShape> | null;
    tags?: CanConnect<TagShape, "tag">[] | null;
    versions?: Omit<ProjectVersionShape, "root">[] | null;
}
export const shapeProject: ShapeModel<ProjectShape, ProjectCreateInput, ProjectUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "handle", "isPrivate", "permissions"),
        ...createOwner(d, "ownedBy"),
        ...createRel(d, "labels", ["Connect", "Create"], "many", shapeLabel),
        ...createRel(d, "parent", ["Connect"], "one"),
        ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
        ...createVersion(d, shapeProjectVersion),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "handle", "isPrivate", "permissions"),
        ...updateOwner(o, u, "ownedBy"),
        ...updateRel(o, u, "labels", ["Connect", "Create", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateVersion(o, u, shapeProjectVersion),
    }),
};

export type ProjectVersionTranslationShape = Pick<ProjectVersionTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "ProjectVersionTranslation";
}
export type ProjectVersionShape = Pick<ProjectVersion, "id" | "isComplete" | "isPrivate" | "versionLabel" | "versionNotes"> & {
    __typename: "ProjectVersion";
    directories?: ProjectVersionDirectoryShape[] | null;
    resourceList?: CanConnect<ResourceListShape> | null;
    root?: CanConnect<ProjectShape> | null;
    suggestedNextByProject?: CanConnect<ProjectShape>[] | null;
    translations?: ProjectVersionTranslationShape[] | null;
}
export const shapeProjectVersionTranslation: ShapeModel<ProjectVersionTranslationShape, ProjectVersionTranslationCreateInput, ProjectVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "name")),
};
export const shapeProjectVersion: ShapeModel<ProjectVersionShape, ProjectVersionCreateInput, ProjectVersionUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "isComplete", "isPrivate", "versionLabel", "versionNotes");
        return {
            ...prims,
            ...createRel(d, "directories", ["Create"], "many", shapeProjectVersionDirectory, (r) => ({ ...r, projectVersion: { id: prims.id } })),
            ...createRel(d, "root", ["Connect", "Create"], "one", shapeProject, (r) => ({ ...r, isPrivate: prims.isPrivate })),
            ...createRel(d, "suggestedNextByProject", ["Connect"], "many"),
            ...createRel(d, "translations", ["Create"], "many", shapeProjectVersionTranslation),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "directories", ["Create", "Update", "Delete"], "many", shapeProjectVersionDirectory),
        ...updateRel(o, u, "root", ["Update"], "one", shapeProject),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeProjectVersionTranslation),
        ...updateRel(o, u, "suggestedNextByProject", ["Connect", "Disconnect"], "many"),
    }),
};

export type ProjectVersionDirectoryShape = Pick<ProjectVersionDirectory, "id" | "isRoot" | "childOrder"> & {
    __typename: "ProjectVersionDirectory";
    childApiVersions?: CanConnect<ApiVersionShape>[] | null;
    childCodeVersions?: CanConnect<CodeVersionShape>[] | null;
    childNoteVersions?: CanConnect<NoteVersionShape>[] | null;
    childProjectVersions?: CanConnect<ProjectVersionShape>[] | null;
    childRoutineVersions?: CanConnect<RoutineVersionShape>[] | null;
    childStandardVersions?: CanConnect<StandardVersionShape>[] | null;
    childTeams?: CanConnect<TeamShape>[] | null;
    parentDirectory?: CanConnect<ProjectVersionDirectoryShape> | null;
    projectVersion?: CanConnect<ProjectVersionShape> | null;
}
export const shapeProjectVersionDirectory: ShapeModel<ProjectVersionDirectoryShape, ProjectVersionDirectoryCreateInput, ProjectVersionDirectoryUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isRoot", "childOrder"),
        ...createRel(d, "childApiVersions", ["Connect"], "many"),
        ...createRel(d, "childCodeVersions", ["Connect"], "many"),
        ...createRel(d, "childNoteVersions", ["Connect"], "many"),
        ...createRel(d, "childProjectVersions", ["Connect"], "many"),
        ...createRel(d, "childRoutineVersions", ["Connect"], "many"),
        ...createRel(d, "childStandardVersions", ["Connect"], "many"),
        ...createRel(d, "childTeams", ["Connect"], "many"),
        ...createRel(d, "parentDirectory", ["Connect"], "one"),
        ...createRel(d, "projectVersion", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isRoot", "childOrder"),
        ...updateRel(o, u, "childApiVersions", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "childCodeVersions", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "childNoteVersions", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "childProjectVersions", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "childRoutineVersions", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "childStandardVersions", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "childTeams", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "parentDirectory", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "projectVersion", ["Connect", "Disconnect"], "one"),
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
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any,
};

export type QuestionTranslationShape = Pick<QuestionTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "QuestionTranslation";
}
export type QuestionShape = Pick<Question, "id" | "isPrivate"> & {
    __typename: "Question";
    acceptedAnswer?: CanConnect<QuestionAnswerShape> | null;
    forObject?: { __typename: QuestionForType | `${QuestionForType}`, id: string } | null;
    referencing?: string;
    tags?: CanConnect<TagShape, "tag">[] | null;
    translations?: QuestionTranslationShape[] | null;
}
export const shapeQuestionTranslation: ShapeModel<QuestionTranslationShape, QuestionTranslationCreateInput, QuestionTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "language", "description", "name")),
};
export const shapeQuestion: ShapeModel<QuestionShape, QuestionCreateInput, QuestionUpdateInput> = {
    create: (d) => ({
        forObjectConnect: d.forObject?.id ?? undefined,
        forObjectType: d.forObject?.__typename as any ?? undefined,
        ...createPrims(d, "id", "isPrivate", "referencing"),
        ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
        ...createRel(d, "translations", ["Create"], "many", shapeQuestionTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate"),
        ...updateRel(o, u, "acceptedAnswer", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeQuestionTranslation),
    }),
};

export type QuestionAnswerShape = Pick<QuestionAnswer, "id"> & {
    __typename: "QuestionAnswer";
}
export const shapeQuestionAnswer: ShapeModel<QuestionAnswerShape, QuestionAnswerCreateInput, QuestionAnswerUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any,
};

export type QuizShape = Pick<Quiz, "id"> & {
    __typename: "Quiz";
}
export const shapeQuiz: ShapeModel<QuizShape, QuizCreateInput, QuizUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any,
};

export type QuizAttemptShape = Pick<QuizAttempt, "id"> & {
    __typename: "QuizAttempt";
}
export const shapeQuizAttempt: ShapeModel<QuizAttemptShape, QuizAttemptCreateInput, QuizAttemptUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any,
};

export type QuizQuestionShape = Pick<QuizQuestion, "id"> & {
    __typename: "QuizQuestion";
}
export const shapeQuizQuestion: ShapeModel<QuizQuestionShape, QuizQuestionCreateInput, QuizQuestionUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any,
};

export type QuizQuestionResponseShape = Pick<QuizQuestionResponse, "id"> & {
    __typename: "QuizQuestionResponse";
}
export const shapeQuizQuestionResponse: ShapeModel<QuizQuestionResponseShape, QuizQuestionResponseCreateInput, QuizQuestionResponseUpdateInput> = {
    create: (d) => ({}) as any,
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
            // Treat as connect when the reminderList has created_at
            ...createRel(d, "reminderList", ["Connect", "Create"], "one", shapeReminderList, (l) => {
                if (l.created_at) return { id: l.id };
                return l;
            }),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "name", "description", ["dueDate", shapeDate], "index", "isComplete"),
        ...updateRel(o, u, "reminderItems", ["Create", "Update", "Delete"], "many", shapeReminderItem, (r, i) => ({ ...r, reminder: { id: i.id } })),
        // Treat as connect when the reminderList has created_at
        ...updateRel(o, u, "reminderList", ["Connect", "Create", "Disconnect"], "one", shapeReminderList, (l) => {
            if (l.created_at) return { id: l.id };
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
    focusMode?: CanConnect<FocusModeShape> | null;
    reminders?: CanConnect<ReminderShape>[] | null;
}
export const shapeReminderList: ShapeModel<ReminderListShape, ReminderListCreateInput, ReminderListUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id");
        return {
            ...prims,
            ...createRel(d, "focusMode", ["Connect"], "one"),
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
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any,
};

export type ResourceTranslationShape = Pick<ResourceTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "ResourceTranslation";
}
export type ResourceShape = Pick<Resource, "id" | "index" | "link" | "usedFor"> & {
    __typename: "Resource";
    list: CanConnect<ResourceListShape>;
    translations: ResourceTranslationShape[];
}
export const shapeResourceTranslation: ShapeModel<ResourceTranslationShape, ResourceTranslationCreateInput, ResourceTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "name")),
};
export const shapeResource: ShapeModel<ResourceShape, ResourceCreateInput, ResourceUpdateInput> = {
    create: (d) => ({
        // Make sure link is properly shaped
        link: addHttps(d.link),
        ...createPrims(d, "id", "index", "usedFor"),
        ...createRel(d, "list", ["Connect", "Create"], "one", shapeResourceList),
        ...createRel(d, "translations", ["Create"], "many", shapeResourceTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        // Make sure link is properly shaped
        link: o.link !== u.link ? addHttps(u.link) : undefined,
        ...updatePrims(o, u, "id", "index", "usedFor"),
        ...updateRel(o, u, "list", ["Connect", "Create"], "one", shapeResourceList),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeResourceTranslation),
    }),
};

export type ResourceListTranslationShape = Pick<ResourceListTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "ResourceListTranslation";
}
export type ResourceListShape = Pick<ResourceList, "id"> & {
    __typename: "ResourceList";
    listFor: Pick<ResourceList["listFor"], "id" | "__typename">;
    resources?: ResourceShape[] | null;
    translations?: ResourceListTranslationShape[] | null;
}
export const shapeResourceListTranslation: ShapeModel<ResourceListTranslationShape, ResourceListTranslationCreateInput, ResourceListTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "name")),
};
export const shapeResourceList: ShapeModel<ResourceListShape, ResourceListCreateInput, ResourceListUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id");
        let listForConnect: string | undefined;
        let listForType: ResourceListFor | undefined;
        console.log("in shapeResourceListCreate", d, "listFor" in d);
        if ("listFor" in d) {
            listForConnect = d.listFor.id;
            listForType = d.listFor.__typename as ResourceListFor;
        }
        return {
            ...prims,
            listForConnect: listForConnect as string,
            listForType: listForType as ResourceListFor,
            ...createRel(d, "resources", ["Create"], "many", shapeResource, (r) => ({ list: { id: prims.id }, ...r })),
            ...createRel(d, "translations", ["Create"], "many", shapeResourceListTranslation),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateRel(o, u, "resources", ["Create", "Update", "Delete"], "many", shapeResource, (r, i) => ({ list: { id: i.id }, ...r })),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeResourceListTranslation),
    }),
};

export type RoleTranslationShape = Pick<RoleTranslation, "id" | "language" | "description"> & {
    __typename?: "RoleTranslation";
}
export type RoleShape = Pick<Role, "id" | "name" | "permissions"> & {
    __typename: "Role";
    members?: CanConnect<MemberShape>[] | null;
    team: CanConnect<TeamShape>;
    translations?: RoleTranslationShape[] | null;
}
export const shapeRoleTranslation: ShapeModel<RoleTranslationShape, RoleTranslationCreateInput, RoleTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description")),
};
export const shapeRole: ShapeModel<RoleShape, RoleCreateInput, RoleUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "name", "permissions"),
        ...createRel(d, "members", ["Connect"], "many"),
        ...createRel(d, "team", ["Connect"], "one"),
        ...createRel(d, "translations", ["Create"], "many", shapeRoleTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "name", "permissions"),
        ...updateRel(o, u, "members", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeRoleTranslation),
    }),
};

export type RoutineShape = Pick<Routine, "id" | "isInternal" | "isPrivate" | "permissions"> & {
    __typename: "Routine";
    labels?: CanConnect<LabelShape>[] | null;
    owner: OwnerShape | null | undefined;
    parent?: CanConnect<RoutineVersionShape> | null;
    tags?: CanConnect<TagShape, "tag">[] | null;
    versions?: Omit<RoutineVersionShape, "root">[] | null;
}
export const shapeRoutine: ShapeModel<RoutineShape, RoutineCreateInput, RoutineUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isInternal", "isPrivate", "permissions"),
        ...createOwner(d, "ownedBy"),
        ...createRel(d, "labels", ["Connect", "Create"], "many", shapeLabel),
        ...createRel(d, "parent", ["Connect"], "one"),
        ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
        ...createVersion(d, shapeRoutineVersion),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isInternal", "isPrivate", "permissions"),
        ...updateOwner(o, u, "ownedBy"),
        ...updateRel(o, u, "labels", ["Connect", "Create", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateVersion(o, u, shapeRoutineVersion),
    }),
};

export type RoutineVersionTranslationShape = Pick<RoutineVersionTranslation, "id" | "language" | "description" | "instructions" | "name"> & {
    __typename?: "RoutineVersionTranslation";
}
export type RoutineVersionShape = Pick<RoutineVersion, "id" | "configCallData" | "configFormInput" | "configFormOutput" | "isAutomatable" | "isComplete" | "isPrivate" | "routineType" | "versionLabel" | "versionNotes"> & {
    __typename: "RoutineVersion";
    apiVersion?: CanConnect<ApiVersionShape> | null;
    codeVersion?: CanConnect<CodeVersionShape> | null;
    directoryListings?: CanConnect<ProjectVersionDirectoryShape>[] | null;
    inputs?: RoutineVersionInputShape[] | null;
    nodes?: NodeShape[] | null;
    nodeLinks?: NodeLinkShape[] | null;
    outputs?: RoutineVersionOutputShape[] | null;
    resourceList?: ResourceListShape | null;
    root?: CanConnect<RoutineShape> | null;
    suggestedNextByRoutineVersion?: CanConnect<RoutineVersionShape>[] | null;
    translations?: RoutineVersionTranslationShape[] | null;
}
export const shapeRoutineVersionTranslation: ShapeModel<RoutineVersionTranslationShape, RoutineVersionTranslationCreateInput, RoutineVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", ["instructions", (instructions) => instructions ?? ""], "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "instructions", "name")),
};
export const shapeRoutineVersion: ShapeModel<RoutineVersionShape, RoutineVersionCreateInput, RoutineVersionUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "configCallData", "configFormInput", "configFormOutput", "isAutomatable", "isComplete", "isPrivate", "routineType", "versionLabel", "versionNotes");
        return {
            ...prims,
            ...createRel(d, "apiVersion", ["Connect"], "one"),
            ...createRel(d, "codeVersion", ["Connect"], "one"),
            ...createRel(d, "directoryListings", ["Connect"], "many"),
            ...createRel(d, "inputs", ["Create"], "many", shapeRoutineVersionInput, (i) => ({ ...i, routineVersion: { id: prims.id } })),
            ...createRel(d, "nodes", ["Create"], "many", shapeNode, (n) => ({ ...n, routineVersion: { id: prims.id } })),
            ...createRel(d, "nodeLinks", ["Create"], "many", shapeNodeLink, (nl) => ({ ...nl, routineVersion: { id: prims.id } })),
            ...createRel(d, "outputs", ["Create"], "many", shapeRoutineVersionOutput, (out) => ({ ...out, routineVersion: { id: prims.id } })),
            ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: prims.id, __typename: "RoutineVersion" } })),
            ...createRel(d, "root", ["Connect", "Create"], "one", shapeRoutine, (r) => ({ ...r, isPrivate: d.isPrivate })),
            ...createRel(d, "suggestedNextByRoutineVersion", ["Connect"], "many"),
            ...createRel(d, "translations", ["Create"], "many", shapeRoutineVersionTranslation),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "configCallData", "configFormInput", "configFormOutput", "isAutomatable", "isComplete", "isPrivate", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "apiVersion", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "codeVersion", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "directoryListings", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "inputs", ["Create", "Update", "Delete"], "many", shapeRoutineVersionInput, (i) => ({ ...i, routineVersion: { id: o.id } })),
        ...updateRel(o, u, "nodes", ["Create", "Update", "Delete"], "many", shapeNode, (n) => ({ ...n, routineVersion: { id: o.id } })),
        ...updateRel(o, u, "nodeLinks", ["Create", "Update", "Delete"], "many", shapeNodeLink, (nl) => ({ ...nl, routineVersion: { id: o.id } })),
        ...updateRel(o, u, "outputs", ["Create", "Update", "Delete"], "many", shapeRoutineVersionOutput, (out) => ({ ...out, routineVersion: { id: o.id } })),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: o.id, __typename: "RoutineVersion" } })),
        ...updateRel(o, u, "root", ["Update"], "one", shapeRoutine),
        ...updateRel(o, u, "suggestedNextByRoutineVersion", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeRoutineVersionTranslation),
    }),
};

export type RoutineVersionInputTranslationShape = Pick<RoutineVersionInputTranslation, "id" | "language" | "description" | "helpText"> & {
    __typename?: "RoutineVersionInputTranslation";
}
export type RoutineVersionInputShape = Pick<RoutineVersionInput, "id" | "index" | "isRequired" | "name"> & {
    __typename: "RoutineVersionInput";
    routineVersion: CanConnect<RoutineVersionShape>;
    standardVersion?: StandardVersionShape | null;
    translations?: RoutineVersionInputTranslationShape[] | null;
}
export const shapeRoutineVersionInputTranslation: ShapeModel<RoutineVersionInputTranslationShape, RoutineVersionInputTranslationCreateInput, RoutineVersionInputTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "helpText"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "helpText")),
};
export const shapeRoutineVersionInput: ShapeModel<RoutineVersionInputShape, RoutineVersionInputCreateInput, RoutineVersionInputUpdateInput> = {
    create: (d) => {
        // Connect to standard if it's marked as external. 
        // Otherwise, set as create. The backend will handle the rest
        const shouldConnectToStandard = d.standardVersion && !d.standardVersion.root.isInternal && d.standardVersion.id;
        return {
            ...createPrims(d, "id", "index", "isRequired", "name"),
            ...createRel(d, "routineVersion", ["Connect"], "one"),
            standardVersionConnect: shouldConnectToStandard ? d.standardVersion!.id : undefined,
            standardVersionCreate: d.standardVersion && !shouldConnectToStandard ? shapeStandardVersion.create(d.standardVersion) : undefined,
            ...createRel(d, "translations", ["Create"], "many", shapeRoutineVersionInputTranslation),
        };
    },
    update: (o, u) => shapeUpdate(u, () => {
        // Connect to standard if it's marked as external. 
        // Otherwise, set as create. The backend will handle the rest
        const shouldConnectToStandard = u.standardVersion && !u.standardVersion.root.isInternal && u.standardVersion.id;
        const hasStandardChanged = hasObjectChanged(o.standardVersion, u.standardVersion);
        return {
            ...updatePrims(o, u, "id", "index", "isRequired", "name"),
            standardVersionConnect: (hasStandardChanged && shouldConnectToStandard) ? u.standardVersion!.id : undefined,
            standardVersionCreate: (u.standardVersion && hasStandardChanged && !shouldConnectToStandard) ? shapeStandardVersion.create(u.standardVersion!) : undefined,
            ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeRoutineVersionInputTranslation),
        };
    }),
};

export type RoutineVersionOutputTranslationShape = Pick<RoutineVersionOutputTranslation, "id" | "language" | "description" | "helpText"> & {
    __typename?: "RoutineVersionOutputTranslation";
}
export type RoutineVersionOutputShape = Pick<RoutineVersionOutput, "id" | "index" | "name"> & {
    __typename: "RoutineVersionOutput";
    routineVersion: CanConnect<RoutineVersionShape>;
    standardVersion?: StandardVersionShape | null;
    translations?: RoutineVersionOutputTranslationShape[] | null;
}
export const shapeRoutineVersionOutputTranslation: ShapeModel<RoutineVersionOutputTranslationShape, RoutineVersionOutputTranslationCreateInput, RoutineVersionOutputTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "helpText"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "helpText")),
};
export const shapeRoutineVersionOutput: ShapeModel<RoutineVersionOutputShape, RoutineVersionOutputCreateInput, RoutineVersionOutputUpdateInput> = {
    create: (d) => {
        // Connect to standard if it's marked as external. 
        // Otherwise, set as create. The backend will handle the rest
        const shouldConnectToStandard = d.standardVersion && !d.standardVersion.root.isInternal && d.standardVersion.id;
        return {
            ...createPrims(d, "id", "index", "name"),
            ...createRel(d, "routineVersion", ["Connect"], "one"),
            standardVersionConnect: shouldConnectToStandard ? d.standardVersion!.id : undefined,
            standardVersionCreate: d.standardVersion && !shouldConnectToStandard ? shapeStandardVersion.create(d.standardVersion) : undefined,
            ...createRel(d, "translations", ["Create"], "many", shapeRoutineVersionOutputTranslation),
        };
    },
    update: (o, u) => shapeUpdate(u, () => {
        // Connect to standard if it's marked as external. 
        // Otherwise, set as create. The backend will handle the rest
        const shouldConnectToStandard = u.standardVersion && !u.standardVersion.root.isInternal && u.standardVersion.id;
        const hasStandardChanged = hasObjectChanged(o.standardVersion, u.standardVersion);
        return {
            ...updatePrims(o, u, "id", "index", "name"),
            standardVersionConnect: (hasStandardChanged && shouldConnectToStandard) ? u.standardVersion!.id : undefined,
            standardVersionCreate: (u.standardVersion && hasStandardChanged && !shouldConnectToStandard) ? shapeStandardVersion.create(u.standardVersion!) : undefined,
            ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeRoutineVersionOutputTranslation),
        };
    }),
};

export type RunProjectShape = Pick<RunProject, "id" | "isPrivate" | "completedComplexity" | "contextSwitches" | "name" | "status" | "timeElapsed"> & {
    __typename: "RunProject";
    steps?: RunProjectStepShape[] | null;
    schedule?: ScheduleShape | null;
    projectVersion?: CanConnect<ProjectVersionShape> | null;
    team?: CanConnect<TeamShape> | null;
}
export const shapeRunProject: ShapeModel<RunProjectShape, RunProjectCreateInput, RunProjectUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isPrivate", "completedComplexity", "contextSwitches", "name", "status", "timeElapsed"),
        ...createRel(d, "steps", ["Create"], "many", shapeRunProjectStep),
        ...createRel(d, "schedule", ["Create"], "one", shapeSchedule),
        ...createRel(d, "projectVersion", ["Connect"], "one"),
        ...createRel(d, "team", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate", "completedComplexity", "contextSwitches", "status", "timeElapsed"),
        ...updateRel(o, u, "steps", ["Create", "Update", "Delete"], "many", shapeRunProjectStep),
        ...updateRel(o, u, "schedule", ["Create", "Update"], "one", shapeSchedule),
    }),
};

export type RunProjectStepShape = Pick<RunProjectStep, "id" | "contextSwitches" | "name" | "order" | "status" | "step" | "timeElapsed"> & {
    __typename: "RunProjectStep";
    directory?: CanConnect<ProjectVersionDirectoryShape> | null;
    node?: CanConnect<NodeShape> | null;
    runProject: CanConnect<RunProjectShape>;
}
export const shapeRunProjectStep: ShapeModel<RunProjectStepShape, RunProjectStepCreateInput, RunProjectStepUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "contextSwitches", "name", "order", "status", "step", "timeElapsed"),
        ...createRel(d, "directory", ["Connect"], "one"),
        ...createRel(d, "node", ["Connect"], "one"),
        ...createRel(d, "runProject", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "contextSwitches", "status", "timeElapsed"),
    }),
};

export type RunRoutineShape = Pick<RunRoutine, "id" | "isPrivate" | "completedComplexity" | "contextSwitches" | "name" | "status" | "timeElapsed"> & {
    __typename: "RunRoutine";
    steps?: RunRoutineStepShape[] | null;
    inputs?: RunRoutineInputShape[] | null;
    schedule?: ScheduleShape | null;
    routineVersion?: CanConnect<RoutineVersionShape> | null;
    runProject?: CanConnect<RunProjectShape> | null;
    team?: CanConnect<TeamShape> | null;
}
export const shapeRunRoutine: ShapeModel<RunRoutineShape, RunRoutineCreateInput, RunRoutineUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isPrivate", "completedComplexity", "contextSwitches", "name", "status", "timeElapsed"),
        ...createRel(d, "steps", ["Create"], "many", shapeRunRoutineStep),
        ...createRel(d, "inputs", ["Create"], "many", shapeRunRoutineInput),
        ...createRel(d, "schedule", ["Create"], "one", shapeSchedule),
        ...createRel(d, "routineVersion", ["Connect"], "one"),
        ...createRel(d, "runProject", ["Connect"], "one"),
        ...createRel(d, "team", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate", "completedComplexity", "contextSwitches", "status", "timeElapsed"),
        ...updateRel(o, u, "inputs", ["Create", "Update", "Delete"], "many", shapeRunRoutineInput),
        ...updateRel(o, u, "steps", ["Create", "Update", "Delete"], "many", shapeRunRoutineStep),
        ...updateRel(o, u, "schedule", ["Create", "Update"], "one", shapeSchedule),
    }),
};

export type RunRoutineInputShape = Pick<RunRoutineInput, "id" | "data"> & {
    __typename: "RunRoutineInput";
    input: CanConnect<RoutineVersionInput>;
    runRoutine: CanConnect<RunRoutine>;
}
export const shapeRunRoutineInput: ShapeModel<RunRoutineInputShape, RunRoutineInputCreateInput, RunRoutineInputUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "data"),
        ...createRel(d, "input", ["Connect"], "one"),
        ...createRel(d, "runRoutine", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "data"),
    }),
};

export type RunRoutineStepShape = Pick<RunRoutineStep, "id" | "contextSwitches" | "name" | "order" | "status" | "step" | "timeElapsed"> & {
    __typename: "RunRoutineStep";
    node?: CanConnect<NodeShape> | null;
    runRoutine: CanConnect<RunRoutineShape>;
    subroutine?: CanConnect<RoutineVersionShape> | null;
}
export const shapeRunRoutineStep: ShapeModel<RunRoutineStepShape, RunRoutineStepCreateInput, RunRoutineStepUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "contextSwitches", "name", "order", "status", "step", "timeElapsed"),
        ...createRel(d, "node", ["Connect"], "one"),
        ...createRel(d, "runRoutine", ["Connect"], "one"),
        ...createRel(d, "subroutine", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "contextSwitches", "status", "timeElapsed"),
    }),
};

export type ScheduleShape = Pick<Schedule, "id" | "startTime" | "endTime" | "timezone"> & {
    __typename: "Schedule";
    exceptions: ScheduleExceptionShape[];
    focusMode?: CanConnect<FocusModeShape> | null;
    labels?: LabelShape[] | null;
    meeting?: CanConnect<MeetingShape> | null;
    recurrences: ScheduleRecurrenceShape[];
    runProject?: CanConnect<RunProjectShape> | null;
    runRoutine?: CanConnect<RunRoutineShape> | null;
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
            ...createRel(d, "focusMode", ["Connect"], "one"),
            ...createRel(d, "labels", ["Create", "Connect"], "many", shapeLabel),
            ...createRel(d, "meeting", ["Connect"], "one"),
            ...createRel(d, "recurrences", ["Create"], "many", shapeScheduleRecurrence, (e) => ({
                ...e,
                schedule: { __typename: "Schedule" as const, id: prims.id },
            })),
            ...createRel(d, "runProject", ["Connect"], "one"),
            ...createRel(d, "runRoutine", ["Connect"], "one"),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", ["startTime", shapeDate], ["endTime", shapeDate], "timezone"),
        ...updateRel(o, u, "exceptions", ["Create", "Update", "Delete"], "many", shapeScheduleException, (e, i) => ({
            ...e,
            schedule: { __typename: "Schedule" as const, id: i.id },
        })),
        ...updateRel(o, u, "labels", ["Create", "Connect", "Disconnect"], "many", shapeLabel),
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

export type StandardShape = Pick<Standard, "id" | "isInternal" | "isPrivate" | "permissions"> & {
    __typename: "Standard";
    parent?: CanConnect<StandardVersionShape> | null;
    owner?: OwnerShape | null;
    labels?: CanConnect<LabelShape>[] | null;
    tags?: CanConnect<TagShape, "tag">[] | null;
    versions?: Omit<StandardVersionShape, "root">[] | null;
}
export type StandardShapeUpdate = Omit<StandardShape, "default" | "isInternal" | "name" | "props" | "yup" | "type" | "version" | "creator">;
export const shapeStandard: ShapeModel<StandardShape, StandardCreateInput, StandardUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isInternal", "isPrivate", "permissions"),
        ...createOwner(d, "ownedBy"),
        ...createRel(d, "labels", ["Connect", "Create"], "many", shapeLabel),
        ...createRel(d, "parent", ["Connect"], "one"),
        ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
        ...createVersion(d, shapeStandardVersion),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isInternal", "isPrivate", "permissions"),
        ...updateOwner(o, u, "ownedBy"),
        ...updateRel(o, u, "labels", ["Connect", "Create", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateVersion(o, u, shapeStandardVersion),
    }),
};

export type StandardVersionTranslationShape = Pick<StandardVersionTranslation, "id" | "language" | "description" | "jsonVariable" | "name"> & {
    __typename?: "StandardVersionTranslation";
}
export type StandardVersionShape = Pick<StandardVersion, "id" | "isComplete" | "isPrivate" | "isFile" | "codeLanguage" | "default" | "props" | "yup" | "variant" | "versionLabel" | "versionNotes"> & {
    __typename: "StandardVersion";
    directoryListings?: CanConnect<ProjectVersionDirectoryShape>[] | null;
    root: StandardShape;
    resourceList?: ResourceListShape | null;
    translations?: StandardVersionTranslationShape[] | null;
}
export const shapeStandardVersionTranslation: ShapeModel<StandardVersionTranslationShape, StandardVersionTranslationCreateInput, StandardVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "jsonVariable", "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "jsonVariable", "name")),
};
export const shapeStandardVersion: ShapeModel<StandardVersionShape, StandardVersionCreateInput, StandardVersionUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "isComplete", "isPrivate", "isFile", "codeLanguage", "default", "props", "yup", "variant", "versionLabel", "versionNotes");
        return {
            ...prims,
            ...createRel(d, "directoryListings", ["Create"], "many", shapeProjectVersionDirectory),
            ...createRel(d, "root", ["Connect", "Create"], "one", shapeStandard, (r) => ({ ...r, isPrivate: prims.isPrivate })),
            ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: prims.id, __typename: "StandardVersion" } })),
            ...createRel(d, "translations", ["Create"], "many", shapeStandardVersionTranslation),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isComplete", "isPrivate", "isFile", "codeLanguage", "default", "props", "yup", "variant", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "directoryListings", ["Create", "Update", "Delete"], "many", shapeProjectVersionDirectory),
        ...updateRel(o, u, "root", ["Update"], "one", shapeStandard),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: o.id, __typename: "StandardVersion" } })),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeStandardVersionTranslation),
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
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeTagTranslation),
    }),
};

export type TeamTranslationShape = Pick<TeamTranslation, "id" | "language" | "bio" | "name"> & {
    __typename?: "TeamTranslation";
}
export type TeamShape = Pick<Team, "id" | "handle" | "isOpenToNewMembers" | "isPrivate"> & {
    __typename: "Team";
    bannerImage?: string | File | null;
    /** Invites for new members */
    memberInvites?: MemberInviteShape[] | null;
    /** Potentially non-exhaustive list of current members. Ignored by shapers */
    members?: MemberShape[];
    /** Members being removed from the team */
    membersDelete?: { id: string }[] | null;
    profileImage?: string | File | null;
    resourceList?: Omit<ResourceListShape, "listFor"> | null;
    roles?: RoleShape[] | null;
    tags?: CanConnect<TagShape, "tag">[] | null;
    translations?: TeamTranslationShape[] | null;
}
export const shapeTeamTranslation: ShapeModel<TeamTranslationShape, TeamTranslationCreateInput, TeamTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "bio", "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "bio", "name")),
};
export const shapeTeam: ShapeModel<TeamShape, TeamCreateInput, TeamUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "bannerImage", "handle", "isOpenToNewMembers", "isPrivate", "profileImage");
        return {
            ...prims,
            ...createRel(d, "memberInvites", ["Create"], "many", shapeMemberInvite),
            ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: prims.id, __typename: "Team" } })),
            ...createRel(d, "roles", ["Create"], "many", shapeRole),
            ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
            ...createRel(d, "translations", ["Create"], "many", shapeTeamTranslation),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "bannerImage", "handle", "isOpenToNewMembers", "isPrivate", "profileImage"),
        ...updateRel(o, u, "memberInvites", ["Create", "Delete"], "many", shapeMemberInvite),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: o.id, __typename: "Team" } })),
        ...updateRel(o, u, "roles", ["Create", "Update", "Delete"], "many", shapeRole),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeTeamTranslation),
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
