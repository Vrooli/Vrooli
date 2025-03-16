import { Api, ApiCreateInput, ApiKey, ApiKeyCreateInput, ApiKeyExternal, ApiKeyExternalCreateInput, ApiKeyExternalUpdateInput, ApiKeyUpdateInput, ApiSearchInput, ApiSortBy, ApiUpdateInput, ApiVersion, ApiVersionCreateInput, ApiVersionSearchInput, ApiVersionSortBy, ApiVersionUpdateInput, ApiYou, Award, AwardSearchInput, AwardSortBy, Bookmark, BookmarkCreateInput, BookmarkList, BookmarkListCreateInput, BookmarkListSearchInput, BookmarkListSortBy, BookmarkListUpdateInput, BookmarkSearchInput, BookmarkSortBy, BookmarkUpdateInput, BotCreateInput, BotUpdateInput, Chat, ChatCreateInput, ChatInvite, ChatInviteCreateInput, ChatInviteSearchInput, ChatInviteSortBy, ChatInviteUpdateInput, ChatInviteYou, ChatMessage, ChatMessageCreateInput, ChatMessageSearchInput, ChatMessageSortBy, ChatMessageUpdateInput, ChatMessageYou, ChatParticipant, ChatParticipantSearchInput, ChatParticipantSortBy, ChatParticipantUpdateInput, ChatSearchInput, ChatSortBy, ChatUpdateInput, ChatYou, Code, CodeCreateInput, CodeSearchInput, CodeSortBy, CodeUpdateInput, CodeVersion, CodeVersionCreateInput, CodeVersionSearchInput, CodeVersionSortBy, CodeVersionUpdateInput, CodeYou, Comment, CommentCreateInput, CommentSearchInput, CommentSortBy, CommentUpdateInput, CommentYou, Email, EmailCreateInput, FocusMode, FocusModeCreateInput, FocusModeFilter, FocusModeFilterCreateInput, FocusModeSearchInput, FocusModeSortBy, FocusModeUpdateInput, FocusModeYou, Issue, IssueCreateInput, IssueSearchInput, IssueSortBy, IssueUpdateInput, IssueYou, Label, LabelCreateInput, LabelSearchInput, LabelSortBy, LabelUpdateInput, LabelYou, Meeting, MeetingCreateInput, MeetingInvite, MeetingInviteCreateInput, MeetingInviteSearchInput, MeetingInviteSortBy, MeetingInviteUpdateInput, MeetingInviteYou, MeetingSearchInput, MeetingSortBy, MeetingUpdateInput, MeetingYou, Member, MemberInvite, MemberInviteCreateInput, MemberInviteSearchInput, MemberInviteSortBy, MemberInviteUpdateInput, MemberInviteYou, MemberSearchInput, MemberSortBy, MemberUpdateInput, MemberYou, Note, NoteCreateInput, NoteSearchInput, NoteSortBy, NoteUpdateInput, NoteVersion, NoteVersionCreateInput, NoteVersionSearchInput, NoteVersionSortBy, NoteVersionUpdateInput, NoteYou, Notification, NotificationSearchInput, NotificationSortBy, NotificationSubscription, NotificationSubscriptionCreateInput, NotificationSubscriptionSearchInput, NotificationSubscriptionSortBy, NotificationSubscriptionUpdateInput, Payment, PaymentSearchInput, PaymentSortBy, Phone, PhoneCreateInput, Post, PostCreateInput, PostSearchInput, PostSortBy, PostUpdateInput, Premium, ProfileUpdateInput, Project, ProjectCreateInput, ProjectSearchInput, ProjectSortBy, ProjectUpdateInput, ProjectVersion, ProjectVersionCreateInput, ProjectVersionDirectory, ProjectVersionDirectoryCreateInput, ProjectVersionDirectorySearchInput, ProjectVersionDirectorySortBy, ProjectVersionDirectoryUpdateInput, ProjectVersionSearchInput, ProjectVersionSortBy, ProjectVersionUpdateInput, ProjectYou, PullRequest, PullRequestCreateInput, PullRequestSearchInput, PullRequestSortBy, PullRequestUpdateInput, PullRequestYou, PushDevice, PushDeviceCreateInput, PushDeviceUpdateInput, Question, QuestionAnswer, QuestionAnswerCreateInput, QuestionAnswerSearchInput, QuestionAnswerSortBy, QuestionAnswerUpdateInput, QuestionCreateInput, QuestionSearchInput, QuestionSortBy, QuestionUpdateInput, QuestionYou, Quiz, QuizAttempt, QuizAttemptCreateInput, QuizAttemptSearchInput, QuizAttemptSortBy, QuizAttemptUpdateInput, QuizAttemptYou, QuizCreateInput, QuizQuestion, QuizQuestionCreateInput, QuizQuestionResponse, QuizQuestionResponseCreateInput, QuizQuestionResponseSearchInput, QuizQuestionResponseSortBy, QuizQuestionResponseUpdateInput, QuizQuestionResponseYou, QuizQuestionUpdateInput, QuizQuestionYou, QuizSearchInput, QuizSortBy, QuizUpdateInput, QuizYou, ReactInput, Reaction, ReactionSearchInput, ReactionSortBy, ReactionSummary, Reminder, ReminderCreateInput, ReminderItem, ReminderItemCreateInput, ReminderItemUpdateInput, ReminderList, ReminderListCreateInput, ReminderListUpdateInput, ReminderSearchInput, ReminderSortBy, ReminderUpdateInput, Report, ReportCreateInput, ReportResponse, ReportResponseCreateInput, ReportResponseSearchInput, ReportResponseSortBy, ReportResponseUpdateInput, ReportResponseYou, ReportSearchInput, ReportSortBy, ReportUpdateInput, ReportYou, ResourceCreateInput, ResourceList, ResourceListCreateInput, ResourceListSearchInput, ResourceListSortBy, ResourceListUpdateInput, ResourceSearchInput, ResourceSortBy, ResourceUpdateInput, Role, RoleCreateInput, RoleSearchInput, RoleSortBy, RoleUpdateInput, Routine, RoutineCreateInput, RoutineSearchInput, RoutineSortBy, RoutineUpdateInput, RoutineVersion, RoutineVersionCreateInput, RoutineVersionInput, RoutineVersionInputCreateInput, RoutineVersionInputUpdateInput, RoutineVersionOutput, RoutineVersionOutputCreateInput, RoutineVersionOutputUpdateInput, RoutineVersionSearchInput, RoutineVersionSortBy, RoutineVersionUpdateInput, RoutineVersionYou, RoutineYou, RunProject, RunProjectCreateInput, RunProjectSearchInput, RunProjectSortBy, RunProjectStep, RunProjectStepCreateInput, RunProjectStepUpdateInput, RunProjectUpdateInput, RunProjectYou, RunRoutine, RunRoutineCreateInput, RunRoutineIO, RunRoutineIOCreateInput, RunRoutineIOSearchInput, RunRoutineIOSortBy, RunRoutineIOUpdateInput, RunRoutineSearchInput, RunRoutineSortBy, RunRoutineStep, RunRoutineStepCreateInput, RunRoutineStepUpdateInput, RunRoutineUpdateInput, RunRoutineYou, Schedule, ScheduleCreateInput, ScheduleException, ScheduleExceptionCreateInput, ScheduleExceptionUpdateInput, ScheduleRecurrence, ScheduleRecurrenceCreateInput, ScheduleRecurrenceUpdateInput, ScheduleSearchInput, ScheduleSortBy, ScheduleUpdateInput, Session, Standard, StandardCreateInput, StandardSearchInput, StandardSortBy, StandardUpdateInput, StandardVersion, StandardVersionCreateInput, StandardVersionSearchInput, StandardVersionSortBy, StandardVersionUpdateInput, StandardYou, StatsApi, StatsApiSearchInput, StatsApiSortBy, StatsCode, StatsCodeSearchInput, StatsCodeSortBy, StatsProject, StatsProjectSearchInput, StatsProjectSortBy, StatsQuiz, StatsQuizSearchInput, StatsQuizSortBy, StatsRoutine, StatsRoutineSearchInput, StatsRoutineSortBy, StatsSite, StatsSiteSearchInput, StatsSiteSortBy, StatsStandard, StatsStandardSearchInput, StatsStandardSortBy, StatsTeam, StatsTeamSearchInput, StatsTeamSortBy, StatsUser, StatsUserSearchInput, StatsUserSortBy, Tag, TagCreateInput, TagSearchInput, TagSortBy, TagUpdateInput, Team, TeamCreateInput, TeamSearchInput, TeamSortBy, TeamUpdateInput, TeamYou, Transfer, TransferRequestReceiveInput, TransferRequestSendInput, TransferSearchInput, TransferSortBy, TransferUpdateInput, TransferYou, User, UserSearchInput, UserSortBy, UserYou, VersionYou, View, ViewSearchInput, ViewSortBy, Wallet, WalletUpdateInput } from "@local/shared";
import { Prisma } from "@prisma/client";
import { Resource } from "i18next";
import { SelectWrap } from "../../builders/types.js";
import { SuppFields } from "../suppFields.js";
import { ModelLogic } from "../types.js";

export type ApiPermissions = Omit<ApiYou, "__typename" | "isBookmarked" | "isViewed" | "reaction">;
export type ApiModelInfo = {
    __typename: "Api",
    IsTransferable: true,
    IsVersioned: true,
    ApiCreate: ApiCreateInput,
    ApiUpdate: ApiUpdateInput,
    ApiModel: Api,
    ApiPermission: ApiPermissions,
    ApiSearch: ApiSearchInput,
    ApiSort: ApiSortBy,
    DbCreate: Prisma.apiUpsertArgs["create"],
    DbUpdate: Prisma.apiUpsertArgs["update"],
    DbModel: Prisma.apiGetPayload<SelectWrap<Prisma.apiSelect>>,
    DbSelect: Prisma.apiSelect,
    DbWhere: Prisma.apiWhereInput,
}
export type ApiModelLogic = ModelLogic<ApiModelInfo, typeof SuppFields.Api>;

export type ApiKeyModelInfo = {
    __typename: "ApiKey",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ApiKeyCreateInput,
    ApiUpdate: ApiKeyUpdateInput,
    ApiPermission: never,
    ApiModel: ApiKey,
    ApiSearch: undefined,
    ApiSort: undefined,
    DbCreate: Prisma.api_keyUpsertArgs["create"],
    DbUpdate: Prisma.api_keyUpsertArgs["update"],
    DbModel: Prisma.api_keyGetPayload<SelectWrap<Prisma.api_keySelect>>,
    DbSelect: Prisma.api_keySelect,
    DbWhere: Prisma.api_keyWhereInput,
}
export type ApiKeyModelLogic = ModelLogic<ApiKeyModelInfo, typeof SuppFields.ApiKey>;

export type ApiKeyExternalModelInfo = {
    __typename: "ApiKeyExternal",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ApiKeyExternalCreateInput,
    ApiUpdate: ApiKeyExternalUpdateInput,
    ApiPermission: never,
    ApiModel: ApiKeyExternal,
    ApiSearch: undefined,
    ApiSort: undefined,
    DbCreate: Prisma.api_key_externalUpsertArgs["create"],
    DbUpdate: Prisma.api_key_externalUpsertArgs["update"],
    DbModel: Prisma.api_key_externalGetPayload<SelectWrap<Prisma.api_key_externalSelect>>,
    DbSelect: Prisma.api_key_externalSelect,
    DbWhere: Prisma.api_key_externalWhereInput,
}
export type ApiKeyExternalModelLogic = ModelLogic<ApiKeyExternalModelInfo, typeof SuppFields.ApiKeyExternal>;

export type ApiVersionPermissions = Omit<VersionYou, "__typename">;
export type ApiVersionModelInfo = {
    __typename: "ApiVersion",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ApiVersionCreateInput,
    ApiUpdate: ApiVersionUpdateInput,
    ApiPermission: ApiVersionPermissions,
    ApiModel: ApiVersion,
    ApiSearch: ApiVersionSearchInput,
    ApiSort: ApiVersionSortBy,
    DbCreate: Prisma.api_versionUpsertArgs["create"],
    DbUpdate: Prisma.api_versionUpsertArgs["update"],
    DbModel: Prisma.api_versionGetPayload<SelectWrap<Prisma.api_versionSelect>>,
    DbSelect: Prisma.api_versionSelect,
    DbWhere: Prisma.api_versionWhereInput,
}
export type ApiVersionModelLogic = ModelLogic<ApiVersionModelInfo, typeof SuppFields.ApiVersion>;

export type AwardModelInfo = {
    __typename: "Award",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: undefined,
    ApiPermission: never,
    ApiModel: Award,
    ApiSearch: AwardSearchInput,
    ApiSort: AwardSortBy,
    DbCreate: undefined,
    DbUpdate: undefined,
    DbModel: Prisma.awardGetPayload<SelectWrap<Prisma.awardSelect>>,
    DbSelect: Prisma.awardSelect,
    DbWhere: Prisma.awardWhereInput,
}
export type AwardModelLogic = ModelLogic<AwardModelInfo, typeof SuppFields.Award>;

export type BookmarkModelInfo = {
    __typename: "Bookmark",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: BookmarkCreateInput,
    ApiUpdate: BookmarkUpdateInput,
    ApiModel: Bookmark,
    ApiSearch: BookmarkSearchInput,
    ApiSort: BookmarkSortBy,
    ApiPermission: never,
    DbCreate: Prisma.bookmarkUpsertArgs["create"],
    DbUpdate: Prisma.bookmarkUpsertArgs["update"],
    DbModel: Prisma.bookmarkGetPayload<SelectWrap<Prisma.bookmarkSelect>>,
    DbSelect: Prisma.bookmarkSelect,
    DbWhere: Prisma.bookmarkWhereInput,
}
export type BookmarkModelLogic = ModelLogic<BookmarkModelInfo, typeof SuppFields.Bookmark>;

export type BookmarkListModelInfo = {
    __typename: "BookmarkList",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: BookmarkListCreateInput,
    ApiUpdate: BookmarkListUpdateInput,
    ApiModel: BookmarkList,
    ApiSearch: BookmarkListSearchInput,
    ApiSort: BookmarkListSortBy,
    ApiPermission: never,
    DbCreate: Prisma.bookmark_listUpsertArgs["create"],
    DbUpdate: Prisma.bookmark_listUpsertArgs["update"],
    DbModel: Prisma.bookmark_listGetPayload<SelectWrap<Prisma.bookmark_listSelect>>,
    DbSelect: Prisma.bookmark_listSelect,
    DbWhere: Prisma.bookmark_listWhereInput,
}
export type BookmarkListModelLogic = ModelLogic<BookmarkListModelInfo, typeof SuppFields.BookmarkList>;

export type ChatPermissions = Omit<ChatYou, "__typename">;
export type ChatModelInfo = {
    __typename: "Chat",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ChatCreateInput,
    ApiUpdate: ChatUpdateInput,
    ApiModel: Chat,
    ApiSearch: ChatSearchInput,
    ApiSort: ChatSortBy,
    ApiPermission: ChatPermissions,
    DbCreate: Prisma.chatUpsertArgs["create"],
    DbUpdate: Prisma.chatUpsertArgs["update"],
    DbModel: Prisma.chatGetPayload<SelectWrap<Prisma.chatSelect>>,
    DbSelect: Prisma.chatSelect,
    DbWhere: Prisma.chatWhereInput,
}
export type ChatModelLogic = ModelLogic<ChatModelInfo, typeof SuppFields.Chat>;

export type ChatInvitePermissions = Omit<ChatInviteYou, "__typename">;
export type ChatInviteModelInfo = {
    __typename: "ChatInvite",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ChatInviteCreateInput,
    ApiUpdate: ChatInviteUpdateInput,
    ApiModel: ChatInvite,
    ApiSearch: ChatInviteSearchInput,
    ApiSort: ChatInviteSortBy,
    ApiPermission: ChatInvitePermissions,
    DbCreate: Prisma.chat_inviteUpsertArgs["create"],
    DbUpdate: Prisma.chat_inviteUpsertArgs["update"],
    DbModel: Prisma.chat_inviteGetPayload<SelectWrap<Prisma.chat_inviteSelect>>,
    DbSelect: Prisma.chat_inviteSelect,
    DbWhere: Prisma.chat_inviteWhereInput,
}
export type ChatInviteModelLogic = ModelLogic<ChatInviteModelInfo, typeof SuppFields.ChatInvite>;

export type ChatMessagePermissions = Omit<ChatMessageYou, "__typename" | "reaction">;
export type ChatMessageModelInfo = {
    __typename: "ChatMessage",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ChatMessageCreateInput,
    ApiUpdate: ChatMessageUpdateInput,
    ApiModel: ChatMessage,
    ApiSearch: ChatMessageSearchInput,
    ApiSort: ChatMessageSortBy,
    ApiPermission: ChatMessagePermissions,
    DbCreate: Prisma.chat_messageUpsertArgs["create"],
    DbUpdate: Prisma.chat_messageUpsertArgs["update"],
    DbModel: Prisma.chat_messageGetPayload<SelectWrap<Prisma.chat_messageSelect>>,
    DbSelect: Prisma.chat_messageSelect,
    DbWhere: Prisma.chat_messageWhereInput,
}
export type ChatMessageModelLogic = ModelLogic<ChatMessageModelInfo, typeof SuppFields.ChatMessage>;

export type ChatParticipantModelInfo = {
    __typename: "ChatParticipant",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: ChatParticipantUpdateInput,
    ApiModel: ChatParticipant,
    ApiSearch: ChatParticipantSearchInput,
    ApiSort: ChatParticipantSortBy,
    ApiPermission: never,
    DbCreate: Prisma.chat_participantsUpsertArgs["create"],
    DbUpdate: Prisma.chat_participantsUpsertArgs["update"],
    DbModel: Prisma.chat_participantsGetPayload<SelectWrap<Prisma.chat_participantsSelect>>,
    DbSelect: Prisma.chat_participantsSelect,
    DbWhere: Prisma.chat_participantsWhereInput,
}
export type ChatParticipantModelLogic = ModelLogic<ChatParticipantModelInfo, typeof SuppFields.ChatParticipant>;

export type CodePermissions = Omit<CodeYou, "__typename" | "isBookmarked" | "isViewed" | "reaction">;
export type CodeModelInfo = {
    __typename: "Code",
    IsTransferable: true,
    IsVersioned: true,
    ApiCreate: CodeCreateInput,
    ApiUpdate: CodeUpdateInput,
    ApiModel: Code,
    ApiSearch: CodeSearchInput,
    ApiSort: CodeSortBy,
    ApiPermission: CodePermissions,
    DbCreate: Prisma.codeUpsertArgs["create"],
    DbUpdate: Prisma.codeUpsertArgs["update"],
    DbModel: Prisma.codeGetPayload<SelectWrap<Prisma.codeSelect>>,
    DbSelect: Prisma.codeSelect,
    DbWhere: Prisma.codeWhereInput,
}
export type CodeModelLogic = ModelLogic<CodeModelInfo, typeof SuppFields.Code>;

export type CodeVersionPermissions = Omit<VersionYou, "__typename">;
export type CodeVersionModelInfo = {
    __typename: "CodeVersion",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: CodeVersionCreateInput,
    ApiUpdate: CodeVersionUpdateInput,
    ApiModel: CodeVersion,
    ApiSearch: CodeVersionSearchInput,
    ApiSort: CodeVersionSortBy,
    ApiPermission: CodeVersionPermissions,
    DbCreate: Prisma.code_versionUpsertArgs["create"],
    DbUpdate: Prisma.code_versionUpsertArgs["update"],
    DbModel: Prisma.code_versionGetPayload<SelectWrap<Prisma.code_versionSelect>>,
    DbSelect: Prisma.code_versionSelect,
    DbWhere: Prisma.code_versionWhereInput,
}
export type CodeVersionModelLogic = ModelLogic<CodeVersionModelInfo, typeof SuppFields.CodeVersion>;

export type CommentPermissions = Omit<CommentYou, "__typename" | "isBookmarked" | "reaction">;
export type CommentModelInfo = {
    __typename: "Comment",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: CommentCreateInput,
    ApiUpdate: CommentUpdateInput,
    ApiModel: Comment,
    ApiPermission: CommentPermissions,
    ApiSearch: CommentSearchInput,
    ApiSort: CommentSortBy,
    DbCreate: Prisma.commentUpsertArgs["create"],
    DbUpdate: Prisma.commentUpsertArgs["update"],
    DbModel: Prisma.commentGetPayload<SelectWrap<Prisma.commentSelect>>,
    DbSelect: Prisma.commentSelect,
    DbWhere: Prisma.commentWhereInput,
}
export type CommentModelLogic = ModelLogic<CommentModelInfo, typeof SuppFields.Comment>;

export type EmailModelInfo = {
    __typename: "Email",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: EmailCreateInput,
    ApiUpdate: undefined,
    ApiModel: Email,
    ApiPermission: never,
    ApiSearch: undefined,
    ApiSort: undefined,
    DbCreate: Prisma.emailUpsertArgs["create"],
    DbUpdate: Prisma.emailUpsertArgs["update"],
    DbModel: Prisma.emailGetPayload<SelectWrap<Prisma.emailSelect>>,
    DbSelect: Prisma.emailSelect,
    DbWhere: Prisma.emailWhereInput,
}
export type EmailModelLogic = ModelLogic<EmailModelInfo, typeof SuppFields.Email>;

export type FocusModePermissions = Omit<FocusModeYou, "__typename">;
export type FocusModeModelInfo = {
    __typename: "FocusMode",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: FocusModeCreateInput,
    ApiUpdate: FocusModeUpdateInput,
    ApiModel: FocusMode,
    ApiSearch: FocusModeSearchInput,
    ApiSort: FocusModeSortBy,
    ApiPermission: FocusModePermissions,
    DbCreate: Prisma.focus_modeUpsertArgs["create"],
    DbUpdate: Prisma.focus_modeUpsertArgs["update"],
    DbModel: Prisma.focus_modeGetPayload<SelectWrap<Prisma.focus_modeSelect>>,
    DbSelect: Prisma.focus_modeSelect,
    DbWhere: Prisma.focus_modeWhereInput,
}
export type FocusModeModelLogic = ModelLogic<FocusModeModelInfo, typeof SuppFields.FocusMode>;

export type FocusModeFilterModelInfo = {
    __typename: "FocusModeFilter",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: FocusModeFilterCreateInput,
    ApiUpdate: undefined,
    ApiModel: FocusModeFilter,
    ApiPermission: never,
    ApiSearch: undefined,
    ApiSort: undefined,
    DbCreate: Prisma.focus_mode_filterUpsertArgs["create"],
    DbUpdate: Prisma.focus_mode_filterUpsertArgs["update"],
    DbModel: Prisma.focus_mode_filterGetPayload<SelectWrap<Prisma.focus_mode_filterSelect>>,
    DbSelect: Prisma.focus_mode_filterSelect,
    DbWhere: Prisma.focus_mode_filterWhereInput,
}
export type FocusModeFilterModelLogic = ModelLogic<FocusModeFilterModelInfo, typeof SuppFields.FocusModeFilter>;

export type IssuePermissions = Omit<IssueYou, "__typename" | "isBookmarked" | "reaction">;
export type IssueModelInfo = {
    __typename: "Issue",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: IssueCreateInput,
    ApiUpdate: IssueUpdateInput,
    ApiModel: Issue,
    ApiSearch: IssueSearchInput,
    ApiSort: IssueSortBy,
    ApiPermission: IssuePermissions,
    DbCreate: Prisma.issueUpsertArgs["create"],
    DbUpdate: Prisma.issueUpsertArgs["update"],
    DbModel: Prisma.issueGetPayload<SelectWrap<Prisma.issueSelect>>,
    DbSelect: Prisma.issueSelect,
    DbWhere: Prisma.issueWhereInput,
}
export type IssueModelLogic = ModelLogic<IssueModelInfo, typeof SuppFields.Issue>;

export type LabelPermissions = Omit<LabelYou, "__typename">;
export type LabelModelInfo = {
    __typename: "Label",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: LabelCreateInput,
    ApiUpdate: LabelUpdateInput,
    ApiModel: Label,
    ApiSearch: LabelSearchInput,
    ApiSort: LabelSortBy,
    ApiPermission: LabelPermissions,
    DbCreate: Prisma.labelUpsertArgs["create"],
    DbUpdate: Prisma.labelUpsertArgs["update"],
    DbModel: Prisma.labelGetPayload<SelectWrap<Prisma.labelSelect>>,
    DbSelect: Prisma.labelSelect,
    DbWhere: Prisma.labelWhereInput,
}
export type LabelModelLogic = ModelLogic<LabelModelInfo, typeof SuppFields.Label>;

export type MeetingPermissions = Omit<MeetingYou, "__typename">;
export type MeetingModelInfo = {
    __typename: "Meeting",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: MeetingCreateInput,
    ApiUpdate: MeetingUpdateInput,
    ApiModel: Meeting,
    ApiSearch: MeetingSearchInput,
    ApiSort: MeetingSortBy,
    ApiPermission: MeetingPermissions,
    DbCreate: Prisma.meetingUpsertArgs["create"],
    DbUpdate: Prisma.meetingUpsertArgs["update"],
    DbModel: Prisma.meetingGetPayload<SelectWrap<Prisma.meetingSelect>>,
    DbSelect: Prisma.meetingSelect,
    DbWhere: Prisma.meetingWhereInput,
}
export type MeetingModelLogic = ModelLogic<MeetingModelInfo, typeof SuppFields.Meeting>;

export type MeetingInvitePermissions = Omit<MeetingInviteYou, "__typename">;
export type MeetingInviteModelInfo = {
    __typename: "MeetingInvite",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: MeetingInviteCreateInput,
    ApiUpdate: MeetingInviteUpdateInput,
    ApiModel: MeetingInvite,
    ApiSearch: MeetingInviteSearchInput,
    ApiSort: MeetingInviteSortBy,
    ApiPermission: MeetingInvitePermissions,
    DbCreate: Prisma.meeting_inviteUpsertArgs["create"],
    DbUpdate: Prisma.meeting_inviteUpsertArgs["update"],
    DbModel: Prisma.meeting_inviteGetPayload<SelectWrap<Prisma.meeting_inviteSelect>>,
    DbSelect: Prisma.meeting_inviteSelect,
    DbWhere: Prisma.meeting_inviteWhereInput,
}
export type MeetingInviteModelLogic = ModelLogic<MeetingInviteModelInfo, typeof SuppFields.MeetingInvite>;

export type MemberPermissions = Omit<MemberYou, "__typename">;
export type MemberModelInfo = {
    __typename: "Member",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: MemberUpdateInput,
    ApiModel: Member,
    ApiSearch: MemberSearchInput,
    ApiSort: MemberSortBy,
    ApiPermission: MemberPermissions,
    DbCreate: Prisma.memberUpsertArgs["create"],
    DbUpdate: Prisma.memberUpsertArgs["update"],
    DbModel: Prisma.memberGetPayload<SelectWrap<Prisma.memberSelect>>,
    DbSelect: Prisma.memberSelect,
    DbWhere: Prisma.memberWhereInput,
}
export type MemberModelLogic = ModelLogic<MemberModelInfo, typeof SuppFields.Member>;

export type MemberInvitePermissions = Omit<MemberInviteYou, "__typename">;
export type MemberInviteModelInfo = {
    __typename: "MemberInvite",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: MemberInviteCreateInput,
    ApiUpdate: MemberInviteUpdateInput,
    ApiModel: MemberInvite,
    ApiSearch: MemberInviteSearchInput,
    ApiSort: MemberInviteSortBy,
    ApiPermission: MemberInvitePermissions,
    DbCreate: Prisma.member_inviteUpsertArgs["create"],
    DbUpdate: Prisma.member_inviteUpsertArgs["update"],
    DbModel: Prisma.member_inviteGetPayload<SelectWrap<Prisma.member_inviteSelect>>,
    DbSelect: Prisma.member_inviteSelect,
    DbWhere: Prisma.member_inviteWhereInput,
}
export type MemberInviteModelLogic = ModelLogic<MemberInviteModelInfo, typeof SuppFields.MemberInvite>;

export type NotePermissions = Omit<NoteYou, "__typename" | "isBookmarked" | "isViewed" | "reaction">;
export type NoteModelInfo = {
    __typename: "Note",
    IsTransferable: true,
    IsVersioned: true,
    ApiCreate: NoteCreateInput,
    ApiUpdate: NoteUpdateInput,
    ApiModel: Note,
    ApiSearch: NoteSearchInput,
    ApiSort: NoteSortBy,
    ApiPermission: NotePermissions,
    DbCreate: Prisma.noteUpsertArgs["create"],
    DbUpdate: Prisma.noteUpsertArgs["update"],
    DbModel: Prisma.noteGetPayload<SelectWrap<Prisma.noteSelect>>,
    DbSelect: Prisma.noteSelect,
    DbWhere: Prisma.noteWhereInput,
}
export type NoteModelLogic = ModelLogic<NoteModelInfo, typeof SuppFields.Note>;

export type NoteVersionPermissions = Omit<VersionYou, "__typename">;
export type NoteVersionModelInfo = {
    __typename: "NoteVersion",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: NoteVersionCreateInput,
    ApiUpdate: NoteVersionUpdateInput,
    ApiModel: NoteVersion,
    ApiSearch: NoteVersionSearchInput,
    ApiSort: NoteVersionSortBy,
    ApiPermission: NoteVersionPermissions,
    DbCreate: Prisma.note_versionUpsertArgs["create"],
    DbUpdate: Prisma.note_versionUpsertArgs["update"],
    DbModel: Prisma.note_versionGetPayload<SelectWrap<Prisma.note_versionSelect>>,
    DbSelect: Prisma.note_versionSelect,
    DbWhere: Prisma.note_versionWhereInput,
}
export type NoteVersionModelLogic = ModelLogic<NoteVersionModelInfo, typeof SuppFields.NoteVersion>;

export type NotificationModelInfo = {
    __typename: "Notification",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: undefined,
    ApiModel: Notification,
    ApiSearch: NotificationSearchInput,
    ApiSort: NotificationSortBy,
    ApiPermission: never,
    DbCreate: Prisma.notificationUpsertArgs["create"],
    DbUpdate: Prisma.notificationUpsertArgs["update"],
    DbModel: Prisma.notificationGetPayload<SelectWrap<Prisma.notificationSelect>>,
    DbSelect: Prisma.notificationSelect,
    DbWhere: Prisma.notificationWhereInput,
}
export type NotificationModelLogic = ModelLogic<NotificationModelInfo, typeof SuppFields.Notification>;

export type NotificationSubscriptionModelInfo = {
    __typename: "NotificationSubscription",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: NotificationSubscriptionCreateInput,
    ApiUpdate: NotificationSubscriptionUpdateInput,
    ApiModel: NotificationSubscription,
    ApiSearch: NotificationSubscriptionSearchInput,
    ApiSort: NotificationSubscriptionSortBy,
    ApiPermission: never,
    DbCreate: Prisma.notification_subscriptionUpsertArgs["create"],
    DbUpdate: Prisma.notification_subscriptionUpsertArgs["update"],
    DbModel: Prisma.notification_subscriptionGetPayload<SelectWrap<Prisma.notification_subscriptionSelect>>,
    DbSelect: Prisma.notification_subscriptionSelect,
    DbWhere: Prisma.notification_subscriptionWhereInput,
}
export type NotificationSubscriptionModelLogic = ModelLogic<NotificationSubscriptionModelInfo, typeof SuppFields.NotificationSubscription>;

export type PaymentModelInfo = {
    __typename: "Payment",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: undefined,
    ApiModel: Payment,
    ApiPermission: never,
    ApiSearch: PaymentSearchInput,
    ApiSort: PaymentSortBy,
    DbCreate: Prisma.paymentUpsertArgs["create"],
    DbUpdate: Prisma.paymentUpsertArgs["update"],
    DbModel: Prisma.paymentGetPayload<SelectWrap<Prisma.paymentSelect>>,
    DbSelect: Prisma.paymentSelect,
    DbWhere: Prisma.paymentWhereInput,
}
export type PaymentModelLogic = ModelLogic<PaymentModelInfo, typeof SuppFields.Payment>;

export type PhoneModelInfo = {
    __typename: "Phone",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: PhoneCreateInput,
    ApiUpdate: undefined,
    ApiModel: Phone,
    ApiPermission: never,
    ApiSearch: undefined,
    ApiSort: undefined,
    DbCreate: Prisma.phoneUpsertArgs["create"],
    DbUpdate: Prisma.phoneUpsertArgs["update"],
    DbModel: Prisma.phoneGetPayload<SelectWrap<Prisma.phoneSelect>>,
    DbSelect: Prisma.phoneSelect,
    DbWhere: Prisma.phoneWhereInput,
}
export type PhoneModelLogic = ModelLogic<PhoneModelInfo, typeof SuppFields.Phone>;

export type PostModelInfo = {
    __typename: "Post",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: PostCreateInput,
    ApiUpdate: PostUpdateInput,
    ApiModel: Post,
    ApiSearch: PostSearchInput,
    ApiSort: PostSortBy,
    ApiPermission: never,
    DbCreate: Prisma.postUpsertArgs["create"],
    DbUpdate: Prisma.postUpsertArgs["update"],
    DbModel: Prisma.postGetPayload<SelectWrap<Prisma.postSelect>>,
    DbSelect: Prisma.postSelect,
    DbWhere: Prisma.postWhereInput,
}
export type PostModelLogic = ModelLogic<PostModelInfo, typeof SuppFields.Post>;

export type PremiumModelInfo = {
    __typename: "Premium",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: undefined,
    ApiModel: Premium,
    ApiPermission: never,
    ApiSearch: undefined,
    ApiSort: undefined,
    DbCreate: Prisma.premiumUpsertArgs["create"],
    DbUpdate: Prisma.premiumUpsertArgs["update"],
    DbModel: Prisma.premiumGetPayload<SelectWrap<Prisma.premiumSelect>>,
    DbSelect: Prisma.premiumSelect,
    DbWhere: Prisma.premiumWhereInput,
}
export type PremiumModelLogic = ModelLogic<PremiumModelInfo, typeof SuppFields.Premium>;

export type ProjectPermissions = Omit<ProjectYou, "__typename" | "isBookmarked" | "isViewed" | "reaction">;
export type ProjectModelInfo = {
    __typename: "Project",
    IsTransferable: true,
    IsVersioned: true,
    ApiCreate: ProjectCreateInput,
    ApiUpdate: ProjectUpdateInput,
    ApiModel: Project,
    ApiSearch: ProjectSearchInput,
    ApiSort: ProjectSortBy,
    ApiPermission: ProjectPermissions,
    DbCreate: Prisma.projectUpsertArgs["create"],
    DbUpdate: Prisma.projectUpsertArgs["update"],
    DbModel: Prisma.projectGetPayload<SelectWrap<Prisma.projectSelect>>,
    DbSelect: Prisma.projectSelect,
    DbWhere: Prisma.projectWhereInput,
}
export type ProjectModelLogic = ModelLogic<ProjectModelInfo, typeof SuppFields.Project>;

export type ProjectVersionPermissions = Omit<VersionYou, "__typename">;
export type ProjectVersionModelInfo = {
    __typename: "ProjectVersion",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ProjectVersionCreateInput,
    ApiUpdate: ProjectVersionUpdateInput,
    ApiModel: ProjectVersion,
    ApiSearch: ProjectVersionSearchInput,
    ApiSort: ProjectVersionSortBy,
    ApiPermission: ProjectVersionPermissions,
    DbCreate: Prisma.project_versionUpsertArgs["create"],
    DbUpdate: Prisma.project_versionUpsertArgs["update"],
    DbModel: Prisma.project_versionGetPayload<SelectWrap<Prisma.project_versionSelect>>,
    DbSelect: Prisma.project_versionSelect,
    DbWhere: Prisma.project_versionWhereInput,
}
export type ProjectVersionModelLogic = ModelLogic<ProjectVersionModelInfo, typeof SuppFields.ProjectVersion>;

export type ProjectVersionDirectoryModelInfo = {
    __typename: "ProjectVersionDirectory",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ProjectVersionDirectoryCreateInput,
    ApiUpdate: ProjectVersionDirectoryUpdateInput,
    ApiModel: ProjectVersionDirectory,
    ApiPermission: never,
    ApiSearch: ProjectVersionDirectorySearchInput,
    ApiSort: ProjectVersionDirectorySortBy,
    DbCreate: Prisma.project_version_directoryUpsertArgs["create"],
    DbUpdate: Prisma.project_version_directoryUpsertArgs["update"],
    DbModel: Prisma.project_version_directoryGetPayload<SelectWrap<Prisma.project_version_directorySelect>>,
    DbSelect: Prisma.project_version_directorySelect,
    DbWhere: Prisma.project_version_directoryWhereInput,
}
export type ProjectVersionDirectoryModelLogic = ModelLogic<ProjectVersionDirectoryModelInfo, typeof SuppFields.ProjectVersionDirectory>;

export type PullRequestPermissions = Omit<PullRequestYou, "__typename">;
export type PullRequestModelInfo = {
    __typename: "PullRequest",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: PullRequestCreateInput,
    ApiUpdate: PullRequestUpdateInput,
    ApiModel: PullRequest,
    ApiSearch: PullRequestSearchInput,
    ApiSort: PullRequestSortBy,
    ApiPermission: PullRequestPermissions,
    DbCreate: Prisma.pull_requestUpsertArgs["create"],
    DbUpdate: Prisma.pull_requestUpsertArgs["update"],
    DbModel: Prisma.pull_requestGetPayload<SelectWrap<Prisma.pull_requestSelect>>,
    DbSelect: Prisma.pull_requestSelect,
    DbWhere: Prisma.pull_requestWhereInput,
}
export type PullRequestModelLogic = ModelLogic<PullRequestModelInfo, typeof SuppFields.PullRequest>;

export type PushDeviceModelInfo = {
    __typename: "PushDevice",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: PushDeviceCreateInput,
    ApiUpdate: PushDeviceUpdateInput,
    ApiModel: PushDevice,
    ApiPermission: never,
    ApiSearch: undefined,
    ApiSort: undefined,
    DbCreate: Prisma.push_deviceUpsertArgs["create"],
    DbUpdate: Prisma.push_deviceUpsertArgs["update"],
    DbModel: Prisma.push_deviceGetPayload<SelectWrap<Prisma.push_deviceSelect>>,
    DbSelect: Prisma.push_deviceSelect,
    DbWhere: Prisma.push_deviceWhereInput,
}
export type PushDeviceModelLogic = ModelLogic<PushDeviceModelInfo, typeof SuppFields.PushDevice>;

export type QuestionPermissions = Omit<QuestionYou, "__typename" | "isBookmarked" | "reaction">;
export type QuestionModelInfo = {
    __typename: "Question",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: QuestionCreateInput,
    ApiUpdate: QuestionUpdateInput,
    ApiModel: Question,
    ApiSearch: QuestionSearchInput,
    ApiSort: QuestionSortBy,
    ApiPermission: QuestionPermissions,
    DbCreate: Prisma.questionUpsertArgs["create"],
    DbUpdate: Prisma.questionUpsertArgs["update"],
    DbModel: Prisma.questionGetPayload<SelectWrap<Prisma.questionSelect>>,
    DbSelect: Prisma.questionSelect,
    DbWhere: Prisma.questionWhereInput,
}
export type QuestionModelLogic = ModelLogic<QuestionModelInfo, typeof SuppFields.Question>;

export type QuestionAnswerModelInfo = {
    __typename: "QuestionAnswer",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: QuestionAnswerCreateInput,
    ApiUpdate: QuestionAnswerUpdateInput,
    ApiModel: QuestionAnswer,
    ApiSearch: QuestionAnswerSearchInput,
    ApiSort: QuestionAnswerSortBy,
    ApiPermission: never,
    DbCreate: Prisma.question_answerUpsertArgs["create"],
    DbUpdate: Prisma.question_answerUpsertArgs["update"],
    DbModel: Prisma.question_answerGetPayload<SelectWrap<Prisma.question_answerSelect>>,
    DbSelect: Prisma.question_answerSelect,
    DbWhere: Prisma.question_answerWhereInput,
}
export type QuestionAnswerModelLogic = ModelLogic<QuestionAnswerModelInfo, typeof SuppFields.QuestionAnswer>;

export type QuizPermissions = Omit<QuizYou, "__typename" | "hasCompleted" | "isBookmarked" | "reaction">;
export type QuizModelInfo = {
    __typename: "Quiz",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: QuizCreateInput,
    ApiUpdate: QuizUpdateInput,
    ApiModel: Quiz,
    ApiSearch: QuizSearchInput,
    ApiSort: QuizSortBy,
    ApiPermission: QuizPermissions,
    DbCreate: Prisma.quizUpsertArgs["create"],
    DbUpdate: Prisma.quizUpsertArgs["update"],
    DbModel: Prisma.quizGetPayload<SelectWrap<Prisma.quizSelect>>,
    DbSelect: Prisma.quizSelect,
    DbWhere: Prisma.quizWhereInput,
}
export type QuizModelLogic = ModelLogic<QuizModelInfo, typeof SuppFields.Quiz>;

export type QuizAttemptPermissions = Omit<QuizAttemptYou, "__typename">;
export type QuizAttemptModelInfo = {
    __typename: "QuizAttempt",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: QuizAttemptCreateInput,
    ApiUpdate: QuizAttemptUpdateInput,
    ApiModel: QuizAttempt,
    ApiSearch: QuizAttemptSearchInput,
    ApiSort: QuizAttemptSortBy,
    ApiPermission: QuizAttemptPermissions,
    DbCreate: Prisma.quiz_attemptUpsertArgs["create"],
    DbUpdate: Prisma.quiz_attemptUpsertArgs["update"],
    DbModel: Prisma.quiz_attemptGetPayload<SelectWrap<Prisma.quiz_attemptSelect>>,
    DbSelect: Prisma.quiz_attemptSelect,
    DbWhere: Prisma.quiz_attemptWhereInput,
}
export type QuizAttemptModelLogic = ModelLogic<QuizAttemptModelInfo, typeof SuppFields.QuizAttempt>;

export type QuizQuestionPermissions = Omit<QuizQuestionYou, "__typename">;
export type QuizQuestionModelInfo = {
    __typename: "QuizQuestion",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: QuizQuestionCreateInput,
    ApiUpdate: QuizQuestionUpdateInput,
    ApiModel: QuizQuestion,
    ApiSearch: undefined,
    ApiSort: undefined,
    ApiPermission: QuizQuestionPermissions,
    DbCreate: Prisma.quiz_questionUpsertArgs["create"],
    DbUpdate: Prisma.quiz_questionUpsertArgs["update"],
    DbModel: Prisma.quiz_questionGetPayload<SelectWrap<Prisma.quiz_questionSelect>>,
    DbSelect: Prisma.quiz_questionSelect,
    DbWhere: Prisma.quiz_questionWhereInput,
}
export type QuizQuestionModelLogic = ModelLogic<QuizQuestionModelInfo, typeof SuppFields.QuizQuestion>;

type QuizQuestionResponsePermissions = Omit<QuizQuestionResponseYou, "__typename">;
export type QuizQuestionResponseModelInfo = {
    __typename: "QuizQuestionResponse",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: QuizQuestionResponseCreateInput,
    ApiUpdate: QuizQuestionResponseUpdateInput,
    ApiModel: QuizQuestionResponse,
    ApiSearch: QuizQuestionResponseSearchInput,
    ApiSort: QuizQuestionResponseSortBy,
    ApiPermission: QuizQuestionResponsePermissions,
    DbCreate: Prisma.quiz_question_responseUpsertArgs["create"],
    DbUpdate: Prisma.quiz_question_responseUpsertArgs["update"],
    DbModel: Prisma.quiz_question_responseGetPayload<SelectWrap<Prisma.quiz_question_responseSelect>>,
    DbSelect: Prisma.quiz_question_responseSelect,
    DbWhere: Prisma.quiz_question_responseWhereInput,
}
export type QuizQuestionResponseModelLogic = ModelLogic<QuizQuestionResponseModelInfo, typeof SuppFields.QuizQuestionResponse>;

export type ReactionModelInfo = {
    __typename: "Reaction",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ReactInput,
    ApiUpdate: undefined,
    ApiModel: Reaction,
    ApiSearch: ReactionSearchInput,
    ApiSort: ReactionSortBy,
    ApiPermission: never,
    DbCreate: Prisma.reactionUpsertArgs["create"],
    DbUpdate: Prisma.reactionUpsertArgs["update"],
    DbModel: Prisma.reactionGetPayload<SelectWrap<Prisma.reactionSelect>>,
    DbSelect: Prisma.reactionSelect,
    DbWhere: Prisma.reactionWhereInput,
}
export type ReactionModelLogic = ModelLogic<ReactionModelInfo, typeof SuppFields.Reaction>;

export type ReactionSummaryModelInfo = {
    __typename: "ReactionSummary",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: undefined,
    ApiModel: ReactionSummary,
    ApiSearch: undefined,
    ApiSort: undefined,
    ApiPermission: never,
    DbCreate: Prisma.reaction_summaryUpsertArgs["create"],
    DbUpdate: Prisma.reaction_summaryUpsertArgs["update"],
    DbModel: Prisma.reaction_summaryGetPayload<SelectWrap<Prisma.reaction_summarySelect>>,
    DbSelect: Prisma.reaction_summarySelect,
    DbWhere: Prisma.reaction_summaryWhereInput,
}
export type ReactionSummaryModelLogic = ModelLogic<ReactionSummaryModelInfo, typeof SuppFields.ReactionSummary>;

export type ReminderModelInfo = {
    __typename: "Reminder",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ReminderCreateInput,
    ApiUpdate: ReminderUpdateInput,
    ApiModel: Reminder,
    ApiSearch: ReminderSearchInput,
    ApiSort: ReminderSortBy,
    ApiPermission: never,
    DbCreate: Prisma.reminderUpsertArgs["create"],
    DbUpdate: Prisma.reminderUpsertArgs["update"],
    DbModel: Prisma.reminderGetPayload<SelectWrap<Prisma.reminderSelect>>,
    DbSelect: Prisma.reminderSelect,
    DbWhere: Prisma.reminderWhereInput,
}
export type ReminderModelLogic = ModelLogic<ReminderModelInfo, typeof SuppFields.Reminder>;

export type ReminderItemModelInfo = {
    __typename: "ReminderItem",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ReminderItemCreateInput,
    ApiUpdate: ReminderItemUpdateInput,
    ApiModel: ReminderItem,
    ApiPermission: never,
    ApiSearch: undefined,
    ApiSort: undefined,
    DbCreate: Prisma.reminder_itemUpsertArgs["create"],
    DbUpdate: Prisma.reminder_itemUpsertArgs["update"],
    DbModel: Prisma.reminder_itemGetPayload<SelectWrap<Prisma.reminder_itemSelect>>,
    DbSelect: Prisma.reminder_itemSelect,
    DbWhere: Prisma.reminder_itemWhereInput,
}
export type ReminderItemModelLogic = ModelLogic<ReminderItemModelInfo, typeof SuppFields.ReminderItem>;

export type ReminderListModelInfo = {
    __typename: "ReminderList",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ReminderListCreateInput,
    ApiUpdate: ReminderListUpdateInput,
    ApiModel: ReminderList,
    ApiSearch: undefined,
    ApiSort: undefined,
    ApiPermission: never,
    DbCreate: Prisma.reminder_listUpsertArgs["create"],
    DbUpdate: Prisma.reminder_listUpsertArgs["update"],
    DbModel: Prisma.reminder_listGetPayload<SelectWrap<Prisma.reminder_listSelect>>,
    DbSelect: Prisma.reminder_listSelect,
    DbWhere: Prisma.reminder_listWhereInput,
}
export type ReminderListModelLogic = ModelLogic<ReminderListModelInfo, typeof SuppFields.ReminderList>;

export type ReportPermissions = Omit<ReportYou, "__typename">;
export type ReportModelInfo = {
    __typename: "Report",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ReportCreateInput,
    ApiUpdate: ReportUpdateInput,
    ApiModel: Report,
    ApiSearch: ReportSearchInput,
    ApiSort: ReportSortBy,
    ApiPermission: ReportPermissions,
    DbCreate: Prisma.reportUpsertArgs["create"],
    DbUpdate: Prisma.reportUpsertArgs["update"],
    DbModel: Prisma.reportGetPayload<SelectWrap<Prisma.reportSelect>>,
    DbSelect: Prisma.reportSelect,
    DbWhere: Prisma.reportWhereInput,
}
export type ReportModelLogic = ModelLogic<ReportModelInfo, typeof SuppFields.Report>;

export type ReportResponsePermissions = Omit<ReportResponseYou, "__typename">;
export type ReportResponseModelInfo = {
    __typename: "ReportResponse",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ReportResponseCreateInput,
    ApiUpdate: ReportResponseUpdateInput,
    ApiModel: ReportResponse,
    ApiSearch: ReportResponseSearchInput,
    ApiSort: ReportResponseSortBy,
    ApiPermission: ReportResponsePermissions,
    DbCreate: Prisma.report_responseUpsertArgs["create"],
    DbUpdate: Prisma.report_responseUpsertArgs["update"],
    DbModel: Prisma.report_responseGetPayload<SelectWrap<Prisma.report_responseSelect>>,
    DbSelect: Prisma.report_responseSelect,
    DbWhere: Prisma.report_responseWhereInput,
}
export type ReportResponseModelLogic = ModelLogic<ReportResponseModelInfo, typeof SuppFields.ReportResponse>;

export type ResourceModelInfo = {
    __typename: "Resource",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ResourceCreateInput,
    ApiUpdate: ResourceUpdateInput,
    ApiModel: Resource,
    ApiSearch: ResourceSearchInput,
    ApiSort: ResourceSortBy,
    ApiPermission: never,
    DbCreate: Prisma.resourceUpsertArgs["create"],
    DbUpdate: Prisma.resourceUpsertArgs["update"],
    DbModel: Prisma.resourceGetPayload<SelectWrap<Prisma.resourceSelect>>,
    DbSelect: Prisma.resourceSelect,
    DbWhere: Prisma.resourceWhereInput,
}
export type ResourceModelLogic = ModelLogic<ResourceModelInfo, typeof SuppFields.Resource>;

export type ResourceListModelInfo = {
    __typename: "ResourceList",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ResourceListCreateInput,
    ApiUpdate: ResourceListUpdateInput,
    ApiModel: ResourceList,
    ApiSearch: ResourceListSearchInput,
    ApiSort: ResourceListSortBy,
    ApiPermission: never,
    DbCreate: Prisma.resource_listUpsertArgs["create"],
    DbUpdate: Prisma.resource_listUpsertArgs["update"],
    DbModel: Prisma.resource_listGetPayload<SelectWrap<Prisma.resource_listSelect>>,
    DbSelect: Prisma.resource_listSelect,
    DbWhere: Prisma.resource_listWhereInput,
}
export type ResourceListModelLogic = ModelLogic<ResourceListModelInfo, typeof SuppFields.ResourceList>;

export type RoleModelInfo = {
    __typename: "Role",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: RoleCreateInput,
    ApiUpdate: RoleUpdateInput,
    ApiModel: Role,
    ApiPermission: never,
    ApiSearch: RoleSearchInput,
    ApiSort: RoleSortBy,
    DbCreate: Prisma.roleUpsertArgs["create"],
    DbUpdate: Prisma.roleUpsertArgs["update"],
    DbModel: Prisma.roleGetPayload<SelectWrap<Prisma.roleSelect>>,
    DbSelect: Prisma.roleSelect,
    DbWhere: Prisma.roleWhereInput,
}
export type RoleModelLogic = ModelLogic<RoleModelInfo, typeof SuppFields.Role>;

type RoutinePermissions = Omit<RoutineYou, "__typename" | "isBookmarked" | "isViewed" | "reaction">;
export type RoutineModelInfo = {
    __typename: "Routine",
    IsTransferable: true,
    IsVersioned: true,
    ApiCreate: RoutineCreateInput,
    ApiUpdate: RoutineUpdateInput,
    ApiModel: Routine,
    ApiSearch: RoutineSearchInput,
    ApiSort: RoutineSortBy,
    ApiPermission: RoutinePermissions,
    DbCreate: Prisma.routineUpsertArgs["create"],
    DbUpdate: Prisma.routineUpsertArgs["update"],
    DbModel: Prisma.routineGetPayload<SelectWrap<Prisma.routineSelect>>,
    DbSelect: Prisma.routineSelect,
    DbWhere: Prisma.routineWhereInput,
}
export type RoutineModelLogic = ModelLogic<RoutineModelInfo, typeof SuppFields.Routine>;

export type RoutineVersionPermissions = Omit<RoutineVersionYou, "__typename">;
export type RoutineVersionModelInfo = {
    __typename: "RoutineVersion",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: RoutineVersionCreateInput,
    ApiUpdate: RoutineVersionUpdateInput,
    ApiModel: RoutineVersion,
    ApiSearch: RoutineVersionSearchInput,
    ApiSort: RoutineVersionSortBy,
    ApiPermission: RoutineVersionPermissions,
    DbCreate: Prisma.routine_versionUpsertArgs["create"],
    DbUpdate: Prisma.routine_versionUpsertArgs["update"],
    DbModel: Prisma.routine_versionGetPayload<SelectWrap<Prisma.routine_versionSelect>>,
    DbSelect: Prisma.routine_versionSelect,
    DbWhere: Prisma.routine_versionWhereInput,
}
export type RoutineVersionModelLogic = ModelLogic<RoutineVersionModelInfo, typeof SuppFields.RoutineVersion>;

export type RoutineVersionInputModelInfo = {
    __typename: "RoutineVersionInput",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: RoutineVersionInputCreateInput,
    ApiUpdate: RoutineVersionInputUpdateInput,
    ApiModel: RoutineVersionInput,
    ApiPermission: never,
    ApiSearch: undefined,
    ApiSort: undefined,
    DbCreate: Prisma.routine_version_inputUpsertArgs["create"],
    DbUpdate: Prisma.routine_version_inputUpsertArgs["update"],
    DbModel: Prisma.routine_version_inputGetPayload<SelectWrap<Prisma.routine_version_inputSelect>>,
    DbSelect: Prisma.routine_version_inputSelect,
    DbWhere: Prisma.routine_version_inputWhereInput,
}
export type RoutineVersionInputModelLogic = ModelLogic<RoutineVersionInputModelInfo, typeof SuppFields.RoutineVersionInput>;

export type RoutineVersionOutputModelInfo = {
    __typename: "RoutineVersionOutput",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: RoutineVersionOutputCreateInput,
    ApiUpdate: RoutineVersionOutputUpdateInput,
    ApiModel: RoutineVersionOutput,
    ApiPermission: never,
    ApiSearch: undefined,
    ApiSort: undefined,
    DbCreate: Prisma.routine_version_outputUpsertArgs["create"],
    DbUpdate: Prisma.routine_version_outputUpsertArgs["update"],
    DbModel: Prisma.routine_version_outputGetPayload<SelectWrap<Prisma.routine_version_outputSelect>>,
    DbSelect: Prisma.routine_version_outputSelect,
    DbWhere: Prisma.routine_version_outputWhereInput,
}
export type RoutineVersionOutputModelLogic = ModelLogic<RoutineVersionOutputModelInfo, typeof SuppFields.RoutineVersionOutput>;

export type RunProjectPermissions = Omit<RunProjectYou, "__typename">;
export type RunProjectModelInfo = {
    __typename: "RunProject",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: RunProjectCreateInput,
    ApiUpdate: RunProjectUpdateInput,
    ApiModel: RunProject,
    ApiSearch: RunProjectSearchInput,
    ApiSort: RunProjectSortBy,
    ApiPermission: RunProjectPermissions,
    DbCreate: Prisma.run_projectUpsertArgs["create"],
    DbUpdate: Prisma.run_projectUpsertArgs["update"],
    DbModel: Prisma.run_projectGetPayload<SelectWrap<Prisma.run_projectSelect>>,
    DbSelect: Prisma.run_projectSelect,
    DbWhere: Prisma.run_projectWhereInput,
}
export type RunProjectModelLogic = ModelLogic<RunProjectModelInfo, typeof SuppFields.RunProject>;

export type RunProjectStepModelInfo = {
    __typename: "RunProjectStep",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: RunProjectStepCreateInput,
    ApiUpdate: RunProjectStepUpdateInput,
    ApiModel: RunProjectStep,
    ApiPermission: never,
    ApiSearch: undefined,
    ApiSort: undefined,
    DbCreate: Prisma.run_project_stepUpsertArgs["create"],
    DbUpdate: Prisma.run_project_stepUpsertArgs["update"],
    DbModel: Prisma.run_project_stepGetPayload<SelectWrap<Prisma.run_project_stepSelect>>,
    DbSelect: Prisma.run_project_stepSelect,
    DbWhere: Prisma.run_project_stepWhereInput,
}
export type RunProjectStepModelLogic = ModelLogic<RunProjectStepModelInfo, typeof SuppFields.RunProjectStep>;

export type RunRoutinePermissions = Omit<RunRoutineYou, "__typename">;
export type RunRoutineModelInfo = {
    __typename: "RunRoutine",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: RunRoutineCreateInput,
    ApiUpdate: RunRoutineUpdateInput,
    ApiModel: RunRoutine,
    ApiSearch: RunRoutineSearchInput,
    ApiSort: RunRoutineSortBy,
    ApiPermission: RunRoutinePermissions,
    DbCreate: Prisma.run_routineUpsertArgs["create"],
    DbUpdate: Prisma.run_routineUpsertArgs["update"],
    DbModel: Prisma.run_routineGetPayload<SelectWrap<Prisma.run_routineSelect>>,
    DbSelect: Prisma.run_routineSelect,
    DbWhere: Prisma.run_routineWhereInput,
}
export type RunRoutineModelLogic = ModelLogic<RunRoutineModelInfo, typeof SuppFields.RunRoutine>;

export type RunRoutineIOModelInfo = {
    __typename: "RunRoutineIO",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: RunRoutineIOCreateInput,
    ApiUpdate: RunRoutineIOUpdateInput,
    ApiModel: RunRoutineIO,
    ApiSearch: RunRoutineIOSearchInput,
    ApiSort: RunRoutineIOSortBy,
    ApiPermission: never,
    DbCreate: Prisma.run_routine_ioUpsertArgs["create"],
    DbUpdate: Prisma.run_routine_ioUpsertArgs["update"],
    DbModel: Prisma.run_routine_ioGetPayload<SelectWrap<Prisma.run_routine_ioSelect>>,
    DbSelect: Prisma.run_routine_ioSelect,
    DbWhere: Prisma.run_routine_ioWhereInput,
}
export type RunRoutineIOModelLogic = ModelLogic<RunRoutineIOModelInfo, typeof SuppFields.RunRoutineIO>;

export type RunRoutineStepModelInfo = {
    __typename: "RunRoutineStep",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: RunRoutineStepCreateInput,
    ApiUpdate: RunRoutineStepUpdateInput,
    ApiModel: RunRoutineStep,
    ApiPermission: never,
    ApiSearch: undefined,
    ApiSort: undefined,
    DbCreate: Prisma.run_routine_stepUpsertArgs["create"],
    DbUpdate: Prisma.run_routine_stepUpsertArgs["update"],
    DbModel: Prisma.run_routine_stepGetPayload<SelectWrap<Prisma.run_routine_stepSelect>>,
    DbSelect: Prisma.run_routine_stepSelect,
    DbWhere: Prisma.run_routine_stepWhereInput,
}
export type RunRoutineStepModelLogic = ModelLogic<RunRoutineStepModelInfo, typeof SuppFields.RunRoutineStep>;

export type ScheduleModelInfo = {
    __typename: "Schedule",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ScheduleCreateInput,
    ApiUpdate: ScheduleUpdateInput,
    ApiModel: Schedule,
    ApiPermission: never,
    ApiSearch: ScheduleSearchInput,
    ApiSort: ScheduleSortBy,
    DbCreate: Prisma.scheduleUpsertArgs["create"],
    DbUpdate: Prisma.scheduleUpsertArgs["update"],
    DbModel: Prisma.scheduleGetPayload<SelectWrap<Prisma.scheduleSelect>>,
    DbSelect: Prisma.scheduleSelect,
    DbWhere: Prisma.scheduleWhereInput,
}
export type ScheduleModelLogic = ModelLogic<ScheduleModelInfo, typeof SuppFields.Schedule>;

export type ScheduleExceptionModelInfo = {
    __typename: "ScheduleException",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ScheduleExceptionCreateInput,
    ApiUpdate: ScheduleExceptionUpdateInput,
    ApiModel: ScheduleException,
    ApiPermission: never,
    ApiSearch: RunRoutineSearchInput,
    ApiSort: RunRoutineSortBy,
    DbCreate: Prisma.schedule_exceptionUpsertArgs["create"],
    DbUpdate: Prisma.schedule_exceptionUpsertArgs["update"],
    DbModel: Prisma.schedule_exceptionGetPayload<SelectWrap<Prisma.schedule_exceptionSelect>>,
    DbSelect: Prisma.schedule_exceptionSelect,
    DbWhere: Prisma.schedule_exceptionWhereInput,
}
export type ScheduleExceptionModelLogic = ModelLogic<ScheduleExceptionModelInfo, typeof SuppFields.ScheduleException>;

export type ScheduleRecurrenceModelInfo = {
    __typename: "ScheduleRecurrence",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ScheduleRecurrenceCreateInput,
    ApiUpdate: ScheduleRecurrenceUpdateInput,
    ApiModel: ScheduleRecurrence,
    ApiPermission: never,
    ApiSearch: RunRoutineSearchInput,
    ApiSort: RunRoutineSortBy,
    DbCreate: Prisma.schedule_recurrenceUpsertArgs["create"],
    DbUpdate: Prisma.schedule_recurrenceUpsertArgs["update"],
    DbModel: Prisma.schedule_recurrenceGetPayload<SelectWrap<Prisma.schedule_recurrenceSelect>>,
    DbSelect: Prisma.schedule_recurrenceSelect,
    DbWhere: Prisma.schedule_recurrenceWhereInput,
}
export type ScheduleRecurrenceModelLogic = ModelLogic<ScheduleRecurrenceModelInfo, typeof SuppFields.ScheduleRecurrence>;

export type SessionModelInfo = {
    __typename: "Session",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: undefined,
    ApiModel: Session,
    ApiPermission: never,
    ApiSearch: undefined,
    ApiSort: undefined,
    DbCreate: Prisma.sessionUpsertArgs["create"],
    DbUpdate: Prisma.sessionUpsertArgs["update"],
    DbModel: Prisma.sessionGetPayload<SelectWrap<Prisma.sessionSelect>>,
    DbSelect: Prisma.sessionSelect,
    DbWhere: Prisma.sessionWhereInput,
}
export type SessionModelLogic = ModelLogic<SessionModelInfo, typeof SuppFields.Session>;

export type StandardPermissions = Omit<StandardYou, "__typename" | "isBookmarked" | "isViewed" | "reaction">;
export type StandardModelInfo = {
    __typename: "Standard",
    IsTransferable: true,
    IsVersioned: true,
    ApiCreate: StandardCreateInput,
    ApiUpdate: StandardUpdateInput,
    ApiModel: Standard,
    ApiSearch: StandardSearchInput,
    ApiSort: StandardSortBy,
    ApiPermission: StandardPermissions,
    DbCreate: Prisma.standardUpsertArgs["create"],
    DbUpdate: Prisma.standardUpsertArgs["update"],
    DbModel: Prisma.standardGetPayload<SelectWrap<Prisma.standardSelect>>,
    DbSelect: Prisma.standardSelect,
    DbWhere: Prisma.standardWhereInput,
}
export type StandardModelLogic = ModelLogic<StandardModelInfo, typeof SuppFields.Standard>;

export type StandardVersionPermissions = Omit<VersionYou, "__typename">;
export type StandardVersionModelInfo = {
    __typename: "StandardVersion",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: StandardVersionCreateInput,
    ApiUpdate: StandardVersionUpdateInput,
    ApiModel: StandardVersion,
    ApiSearch: StandardVersionSearchInput,
    ApiSort: StandardVersionSortBy,
    ApiPermission: StandardVersionPermissions,
    DbCreate: Prisma.standard_versionUpsertArgs["create"],
    DbUpdate: Prisma.standard_versionUpsertArgs["update"],
    DbModel: Prisma.standard_versionGetPayload<SelectWrap<Prisma.standard_versionSelect>>,
    DbSelect: Prisma.standard_versionSelect,
    DbWhere: Prisma.standard_versionWhereInput,
}
export type StandardVersionModelLogic = ModelLogic<StandardVersionModelInfo, typeof SuppFields.StandardVersion>;

export type StatsApiModelInfo = {
    __typename: "StatsApi",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: undefined,
    ApiModel: StatsApi,
    ApiSearch: StatsApiSearchInput,
    ApiSort: StatsApiSortBy,
    ApiPermission: never,
    DbCreate: Prisma.stats_apiUpsertArgs["create"],
    DbUpdate: Prisma.stats_apiUpsertArgs["update"],
    DbModel: Prisma.stats_apiGetPayload<SelectWrap<Prisma.stats_apiSelect>>,
    DbSelect: Prisma.stats_apiSelect,
    DbWhere: Prisma.stats_apiWhereInput,
}
export type StatsApiModelLogic = ModelLogic<StatsApiModelInfo, typeof SuppFields.StatsApi>;

export type StatsCodeModelInfo = {
    __typename: "StatsCode",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: undefined,
    ApiModel: StatsCode,
    ApiSearch: StatsCodeSearchInput,
    ApiSort: StatsCodeSortBy,
    ApiPermission: never,
    DbCreate: Prisma.stats_codeUpsertArgs["create"],
    DbUpdate: Prisma.stats_codeUpsertArgs["update"],
    DbModel: Prisma.stats_codeGetPayload<SelectWrap<Prisma.stats_codeSelect>>,
    DbSelect: Prisma.stats_codeSelect,
    DbWhere: Prisma.stats_codeWhereInput,
}
export type StatsCodeModelLogic = ModelLogic<StatsCodeModelInfo, typeof SuppFields.StatsCode>;

export type StatsTeamModelInfo = {
    __typename: "StatsTeam",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: undefined,
    ApiModel: StatsTeam,
    ApiSearch: StatsTeamSearchInput,
    ApiSort: StatsTeamSortBy,
    ApiPermission: never,
    DbCreate: Prisma.stats_teamUpsertArgs["create"],
    DbUpdate: Prisma.stats_teamUpsertArgs["update"],
    DbModel: Prisma.stats_teamGetPayload<SelectWrap<Prisma.stats_teamSelect>>,
    DbSelect: Prisma.stats_teamSelect,
    DbWhere: Prisma.stats_teamWhereInput,
}
export type StatsTeamModelLogic = ModelLogic<StatsTeamModelInfo, typeof SuppFields.StatsTeam>;

export type StatsProjectModelInfo = {
    __typename: "StatsProject",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: undefined,
    ApiModel: StatsProject,
    ApiSearch: StatsProjectSearchInput,
    ApiSort: StatsProjectSortBy,
    ApiPermission: never,
    DbCreate: Prisma.stats_projectUpsertArgs["create"],
    DbUpdate: Prisma.stats_projectUpsertArgs["update"],
    DbModel: Prisma.stats_projectGetPayload<SelectWrap<Prisma.stats_projectSelect>>,
    DbSelect: Prisma.stats_projectSelect,
    DbWhere: Prisma.stats_projectWhereInput,
}
export type StatsProjectModelLogic = ModelLogic<StatsProjectModelInfo, typeof SuppFields.StatsProject>;

export type StatsQuizModelInfo = {
    __typename: "StatsQuiz",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: undefined,
    ApiModel: StatsQuiz,
    ApiSearch: StatsQuizSearchInput,
    ApiSort: StatsQuizSortBy,
    ApiPermission: never,
    DbCreate: Prisma.stats_quizUpsertArgs["create"],
    DbUpdate: Prisma.stats_quizUpsertArgs["update"],
    DbModel: Prisma.stats_quizGetPayload<SelectWrap<Prisma.stats_quizSelect>>,
    DbSelect: Prisma.stats_quizSelect,
    DbWhere: Prisma.stats_quizWhereInput,
}
export type StatsQuizModelLogic = ModelLogic<StatsQuizModelInfo, typeof SuppFields.StatsQuiz>;

export type StatsRoutineModelInfo = {
    __typename: "StatsRoutine",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: undefined,
    ApiModel: StatsRoutine,
    ApiSearch: StatsRoutineSearchInput,
    ApiSort: StatsRoutineSortBy,
    ApiPermission: never,
    DbCreate: Prisma.stats_routineUpsertArgs["create"],
    DbUpdate: Prisma.stats_routineUpsertArgs["update"],
    DbModel: Prisma.stats_routineGetPayload<SelectWrap<Prisma.stats_routineSelect>>,
    DbSelect: Prisma.stats_routineSelect,
    DbWhere: Prisma.stats_routineWhereInput,
}
export type StatsRoutineModelLogic = ModelLogic<StatsRoutineModelInfo, typeof SuppFields.StatsRoutine>;

export type StatsSiteModelInfo = {
    __typename: "StatsSite",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: undefined,
    ApiModel: StatsSite,
    ApiSearch: StatsSiteSearchInput,
    ApiSort: StatsSiteSortBy,
    ApiPermission: never,
    DbCreate: Prisma.stats_siteUpsertArgs["create"],
    DbUpdate: Prisma.stats_siteUpsertArgs["update"],
    DbModel: Prisma.stats_siteGetPayload<SelectWrap<Prisma.stats_siteSelect>>,
    DbSelect: Prisma.stats_siteSelect,
    DbWhere: Prisma.stats_siteWhereInput,
}
export type StatsSiteModelLogic = ModelLogic<StatsSiteModelInfo, typeof SuppFields.StatsSite>;

export type StatsStandardModelInfo = {
    __typename: "StatsStandard",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: undefined,
    ApiModel: StatsStandard,
    ApiSearch: StatsStandardSearchInput,
    ApiSort: StatsStandardSortBy,
    ApiPermission: never,
    DbCreate: Prisma.stats_standardUpsertArgs["create"],
    DbUpdate: Prisma.stats_standardUpsertArgs["update"],
    DbModel: Prisma.stats_standardGetPayload<SelectWrap<Prisma.stats_standardSelect>>,
    DbSelect: Prisma.stats_standardSelect,
    DbWhere: Prisma.stats_standardWhereInput,
}
export type StatsStandardModelLogic = ModelLogic<StatsStandardModelInfo, typeof SuppFields.StatsStandard>;

export type StatsUserModelInfo = {
    __typename: "StatsUser",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: undefined,
    ApiModel: StatsUser,
    ApiSearch: StatsUserSearchInput,
    ApiSort: StatsUserSortBy,
    ApiPermission: never,
    DbCreate: Prisma.stats_userUpsertArgs["create"],
    DbUpdate: Prisma.stats_userUpsertArgs["update"],
    DbModel: Prisma.stats_userGetPayload<SelectWrap<Prisma.stats_userSelect>>,
    DbSelect: Prisma.stats_userSelect,
    DbWhere: Prisma.stats_userWhereInput,
}
export type StatsUserModelLogic = ModelLogic<StatsUserModelInfo, typeof SuppFields.StatsUser>;

export type TagModelInfo = {
    __typename: "Tag",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: TagCreateInput,
    ApiUpdate: TagUpdateInput,
    ApiModel: Tag,
    ApiSearch: TagSearchInput,
    ApiSort: TagSortBy,
    ApiPermission: never,
    DbCreate: Prisma.tagUpsertArgs["create"],
    DbUpdate: Prisma.tagUpsertArgs["update"],
    DbModel: Prisma.tagGetPayload<SelectWrap<Prisma.tagSelect>>,
    DbSelect: Prisma.tagSelect,
    DbWhere: Prisma.tagWhereInput,
}
export type TagModelLogic = ModelLogic<TagModelInfo, typeof SuppFields.Tag, "tag">;

export type TeamPermissions = Omit<TeamYou, "__typename" | "isBookmarked" | "isViewed" | "yourMembership">;
export type TeamModelInfo = {
    __typename: "Team",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: TeamCreateInput,
    ApiUpdate: TeamUpdateInput,
    ApiModel: Team,
    ApiSearch: TeamSearchInput,
    ApiSort: TeamSortBy,
    ApiPermission: TeamPermissions,
    DbCreate: Prisma.teamUpsertArgs["create"],
    DbUpdate: Prisma.teamUpsertArgs["update"],
    DbModel: Prisma.teamGetPayload<SelectWrap<Prisma.teamSelect>>,
    DbSelect: Prisma.teamSelect,
    DbWhere: Prisma.teamWhereInput,
}
export type TeamModelLogic = ModelLogic<TeamModelInfo, typeof SuppFields.Team>;

export type TransferPermissions = Omit<TransferYou, "__typename">;
export type TransferModelInfo = {
    __typename: "Transfer",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: TransferRequestSendInput | TransferRequestReceiveInput,
    ApiUpdate: TransferUpdateInput,
    ApiModel: Transfer,
    ApiSearch: TransferSearchInput,
    ApiSort: TransferSortBy,
    ApiPermission: TransferPermissions,
    DbCreate: Prisma.transferUpsertArgs["create"],
    DbUpdate: Prisma.transferUpsertArgs["update"],
    DbModel: Prisma.transferGetPayload<SelectWrap<Prisma.transferSelect>>,
    DbSelect: Prisma.transferSelect,
    DbWhere: Prisma.transferWhereInput,
}
export type TransferModelLogic = ModelLogic<TransferModelInfo, typeof SuppFields.Transfer>;

export type UserPermissions = Omit<UserYou, "__typename" | "isBookmarked" | "isViewed">
export type UserModelInfo = {
    __typename: "User",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: BotCreateInput, // Can only create bot users
    ApiUpdate: BotUpdateInput | ProfileUpdateInput, // Can update either a bot or your own profile
    ApiModel: User,
    ApiSearch: UserSearchInput,
    ApiSort: UserSortBy,
    ApiPermission: UserPermissions,
    DbCreate: Prisma.userUpsertArgs["create"],
    DbUpdate: Prisma.userUpsertArgs["update"],
    DbModel: Prisma.userGetPayload<SelectWrap<Prisma.userSelect>>,
    DbSelect: Prisma.userSelect,
    DbWhere: Prisma.userWhereInput,
}
export type UserModelLogic = ModelLogic<UserModelInfo, typeof SuppFields.User>;

export type ViewModelInfo = {
    __typename: "View",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: undefined,
    ApiModel: View,
    ApiSearch: ViewSearchInput,
    ApiSort: ViewSortBy,
    ApiPermission: never,
    DbCreate: Prisma.viewUpsertArgs["create"],
    DbUpdate: Prisma.viewUpsertArgs["update"],
    DbModel: Prisma.viewGetPayload<SelectWrap<Prisma.viewSelect>>,
    DbSelect: Prisma.viewSelect,
    DbWhere: Prisma.viewWhereInput,
}
export type ViewModelLogic = ModelLogic<ViewModelInfo, typeof SuppFields.View>;

export type WalletModelInfo = {
    __typename: "Wallet",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: WalletUpdateInput,
    ApiModel: Wallet,
    ApiPermission: never,
    ApiSearch: undefined,
    ApiSort: undefined,
    DbCreate: Prisma.walletUpsertArgs["create"],
    DbUpdate: Prisma.walletUpsertArgs["update"],
    DbModel: Prisma.walletGetPayload<SelectWrap<Prisma.walletSelect>>,
    DbSelect: Prisma.walletSelect,
    DbWhere: Prisma.walletWhereInput,
}
export type WalletModelLogic = ModelLogic<WalletModelInfo, typeof SuppFields.Wallet>;
