import { type ApiKey, type ApiKeyCreateInput, type ApiKeyExternal, type ApiKeyExternalCreateInput, type ApiKeyExternalUpdateInput, type ApiKeyUpdateInput, type Award, type AwardSearchInput, type AwardSortBy, type Bookmark, type BookmarkCreateInput, type BookmarkList, type BookmarkListCreateInput, type BookmarkListSearchInput, type BookmarkListSortBy, type BookmarkListUpdateInput, type BookmarkSearchInput, type BookmarkSortBy, type BookmarkUpdateInput, type BotCreateInput, type BotUpdateInput, type Chat, type ChatCreateInput, type ChatInvite, type ChatInviteCreateInput, type ChatInviteSearchInput, type ChatInviteSortBy, type ChatInviteUpdateInput, type ChatInviteYou, type ChatMessage, type ChatMessageCreateInput, type ChatMessageSearchInput, type ChatMessageSortBy, type ChatMessageUpdateInput, type ChatMessageYou, type ChatParticipant, type ChatParticipantSearchInput, type ChatParticipantSortBy, type ChatParticipantUpdateInput, type ChatSearchInput, type ChatSortBy, type ChatUpdateInput, type ChatYou, type Comment, type CommentCreateInput, type CommentSearchInput, type CommentSortBy, type CommentUpdateInput, type CommentYou, type Email, type EmailCreateInput, type Issue, type IssueCreateInput, type IssueSearchInput, type IssueSortBy, type IssueUpdateInput, type IssueYou, type Meeting, type MeetingCreateInput, type MeetingInvite, type MeetingInviteCreateInput, type MeetingInviteSearchInput, type MeetingInviteSortBy, type MeetingInviteUpdateInput, type MeetingInviteYou, type MeetingSearchInput, type MeetingSortBy, type MeetingUpdateInput, type MeetingYou, type Member, type MemberInvite, type MemberInviteCreateInput, type MemberInviteSearchInput, type MemberInviteSortBy, type MemberInviteUpdateInput, type MemberInviteYou, type MemberSearchInput, type MemberSortBy, type MemberUpdateInput, type MemberYou, type Notification, type NotificationSearchInput, type NotificationSortBy, type NotificationSubscription, type NotificationSubscriptionCreateInput, type NotificationSubscriptionSearchInput, type NotificationSubscriptionSortBy, type NotificationSubscriptionUpdateInput, type Payment, type PaymentSearchInput, type PaymentSortBy, type Phone, type PhoneCreateInput, type Premium, type ProfileUpdateInput, type PullRequest, type PullRequestCreateInput, type PullRequestSearchInput, type PullRequestSortBy, type PullRequestUpdateInput, type PullRequestYou, type PushDevice, type PushDeviceCreateInput, type PushDeviceUpdateInput, type ReactInput, type Reaction, type ReactionSearchInput, type ReactionSortBy, type ReactionSummary, type Reminder, type ReminderCreateInput, type ReminderItem, type ReminderItemCreateInput, type ReminderItemUpdateInput, type ReminderList, type ReminderListCreateInput, type ReminderListUpdateInput, type ReminderSearchInput, type ReminderSortBy, type ReminderUpdateInput, type Report, type ReportCreateInput, type ReportResponse, type ReportResponseCreateInput, type ReportResponseSearchInput, type ReportResponseSortBy, type ReportResponseUpdateInput, type ReportResponseYou, type ReportSearchInput, type ReportSortBy, type ReportUpdateInput, type ReportYou, type ResourceCreateInput, type ResourceSearchInput, type ResourceSortBy, type ResourceUpdateInput, type ResourceVersion, type ResourceVersionCreateInput, type ResourceVersionRelation, type ResourceVersionRelationCreateInput, type ResourceVersionRelationUpdateInput, type ResourceVersionSearchInput, type ResourceVersionSortBy, type ResourceVersionUpdateInput, type ResourceVersionYou, type ResourceYou, type Run, type RunCreateInput, type RunIO, type RunIOCreateInput, type RunIOSearchInput, type RunIOSortBy, type RunIOUpdateInput, type RunSearchInput, type RunSortBy, type RunStep, type RunStepCreateInput, type RunStepUpdateInput, type RunUpdateInput, type RunYou, type Schedule, type ScheduleCreateInput, type ScheduleException, type ScheduleExceptionCreateInput, type ScheduleExceptionSearchInput, type ScheduleExceptionSortBy, type ScheduleExceptionUpdateInput, type ScheduleRecurrence, type ScheduleRecurrenceCreateInput, type ScheduleRecurrenceSearchInput, type ScheduleRecurrenceSortBy, type ScheduleRecurrenceUpdateInput, type ScheduleSearchInput, type ScheduleSortBy, type ScheduleUpdateInput, type Session, type StatsResource, type StatsResourceSearchInput, type StatsResourceSortBy, type StatsSite, type StatsSiteSearchInput, type StatsSiteSortBy, type StatsTeam, type StatsTeamSearchInput, type StatsTeamSortBy, type StatsUser, type StatsUserSearchInput, type StatsUserSortBy, type Tag, type TagCreateInput, type TagSearchInput, type TagSortBy, type TagUpdateInput, type Team, type TeamCreateInput, type TeamSearchInput, type TeamSortBy, type TeamUpdateInput, type TeamYou, type Transfer, type TransferRequestReceiveInput, type TransferRequestSendInput, type TransferSearchInput, type TransferSortBy, type TransferUpdateInput, type TransferYou, type User, type UserSearchInput, type UserSortBy, type UserYou, type View, type ViewSearchInput, type ViewSortBy, type Wallet, type WalletUpdateInput } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";
import { type Resource } from "i18next";
import { type SelectWrap } from "../../builders/types.js";
import { type SuppFields } from "../suppFields.js";
import { type ModelLogic } from "../types.js";

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

type ResourcePermissions = Omit<ResourceYou, "__typename" | "isBookmarked" | "isViewed" | "reaction">;
export type ResourceModelInfo = {
    __typename: "Resource",
    IsTransferable: true,
    IsVersioned: true,
    ApiCreate: ResourceCreateInput,
    ApiUpdate: ResourceUpdateInput,
    ApiModel: Resource,
    ApiSearch: ResourceSearchInput,
    ApiSort: ResourceSortBy,
    ApiPermission: ResourcePermissions,
    DbCreate: Prisma.resourceUpsertArgs["create"],
    DbUpdate: Prisma.resourceUpsertArgs["update"],
    DbModel: Prisma.resourceGetPayload<SelectWrap<Prisma.resourceSelect>>,
    DbSelect: Prisma.resourceSelect,
    DbWhere: Prisma.resourceWhereInput,
}
export type ResourceModelLogic = ModelLogic<ResourceModelInfo, typeof SuppFields.Resource>;

export type ResourceVersionPermissions = Omit<ResourceVersionYou, "__typename">;
export type ResourceVersionModelInfo = {
    __typename: "ResourceVersion",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ResourceVersionCreateInput,
    ApiUpdate: ResourceVersionUpdateInput,
    ApiModel: ResourceVersion,
    ApiSearch: ResourceVersionSearchInput,
    ApiSort: ResourceVersionSortBy,
    ApiPermission: ResourceVersionPermissions,
    DbCreate: Prisma.resource_versionUpsertArgs["create"],
    DbUpdate: Prisma.resource_versionUpsertArgs["update"],
    DbModel: Prisma.resource_versionGetPayload<SelectWrap<Prisma.resource_versionSelect>>,
    DbSelect: Prisma.resource_versionSelect,
    DbWhere: Prisma.resource_versionWhereInput,
}
export type ResourceVersionModelLogic = ModelLogic<ResourceVersionModelInfo, typeof SuppFields.ResourceVersion>;

export type ResourceVersionRelationModelInfo = {
    __typename: "ResourceVersionRelation",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: ResourceVersionRelationCreateInput,
    ApiUpdate: ResourceVersionRelationUpdateInput,
    ApiModel: ResourceVersionRelation,
    ApiSearch: undefined,
    ApiSort: undefined,
    ApiPermission: never,
    DbCreate: Prisma.resource_version_relationUpsertArgs["create"],
    DbUpdate: Prisma.resource_version_relationUpsertArgs["update"],
    DbModel: Prisma.resource_version_relationGetPayload<SelectWrap<Prisma.resource_version_relationSelect>>,
    DbSelect: Prisma.resource_version_relationSelect,
    DbWhere: Prisma.resource_version_relationWhereInput,
}
export type ResourceVersionRelationModelLogic = ModelLogic<ResourceVersionRelationModelInfo, typeof SuppFields.ResourceVersionRelation>;

export type RunPermissions = Omit<RunYou, "__typename">;
export type RunModelInfo = {
    __typename: "Run",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: RunCreateInput,
    ApiUpdate: RunUpdateInput,
    ApiModel: Run,
    ApiSearch: RunSearchInput,
    ApiSort: RunSortBy,
    ApiPermission: RunPermissions,
    DbCreate: Prisma.runUpsertArgs["create"],
    DbUpdate: Prisma.runUpsertArgs["update"],
    DbModel: Prisma.runGetPayload<SelectWrap<Prisma.runSelect>>,
    DbSelect: Prisma.runSelect,
    DbWhere: Prisma.runWhereInput,
}
export type RunModelLogic = ModelLogic<RunModelInfo, typeof SuppFields.Run>;

export type RunIOModelInfo = {
    __typename: "RunIO",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: RunIOCreateInput,
    ApiUpdate: RunIOUpdateInput,
    ApiModel: RunIO,
    ApiSearch: RunIOSearchInput,
    ApiSort: RunIOSortBy,
    ApiPermission: never,
    DbCreate: Prisma.run_ioUpsertArgs["create"],
    DbUpdate: Prisma.run_ioUpsertArgs["update"],
    DbModel: Prisma.run_ioGetPayload<SelectWrap<Prisma.run_ioSelect>>,
    DbSelect: Prisma.run_ioSelect,
    DbWhere: Prisma.run_ioWhereInput,
}
export type RunIOModelLogic = ModelLogic<RunIOModelInfo, typeof SuppFields.RunIO>;

export type RunStepModelInfo = {
    __typename: "RunStep",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: RunStepCreateInput,
    ApiUpdate: RunStepUpdateInput,
    ApiModel: RunStep,
    ApiPermission: never,
    ApiSearch: undefined,
    ApiSort: undefined,
    DbCreate: Prisma.run_stepUpsertArgs["create"],
    DbUpdate: Prisma.run_stepUpsertArgs["update"],
    DbModel: Prisma.run_stepGetPayload<SelectWrap<Prisma.run_stepSelect>>,
    DbSelect: Prisma.run_stepSelect,
    DbWhere: Prisma.run_stepWhereInput,
}
export type RunStepModelLogic = ModelLogic<RunStepModelInfo, typeof SuppFields.RunStep>;

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
    ApiSearch: ScheduleExceptionSearchInput,
    ApiSort: ScheduleExceptionSortBy,
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
    ApiSearch: ScheduleRecurrenceSearchInput,
    ApiSort: ScheduleRecurrenceSortBy,
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

export type StatsResourceModelInfo = {
    __typename: "StatsResource",
    IsTransferable: false,
    IsVersioned: false,
    ApiCreate: undefined,
    ApiUpdate: undefined,
    ApiModel: StatsResource,
    ApiSearch: StatsResourceSearchInput,
    ApiSort: StatsResourceSortBy,
    ApiPermission: never,
    DbCreate: Prisma.stats_resourceUpsertArgs["create"],
    DbUpdate: Prisma.stats_resourceUpsertArgs["update"],
    DbModel: Prisma.stats_resourceGetPayload<SelectWrap<Prisma.stats_resourceSelect>>,
    DbSelect: Prisma.stats_resourceSelect,
    DbWhere: Prisma.stats_resourceWhereInput,
}
export type StatsResourceModelLogic = ModelLogic<StatsResourceModelInfo, typeof SuppFields.StatsResource>;

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
    ApiCreate: BotCreateInput | (ProfileUpdateInput & { id: string, isBot: boolean }), // Profile creation only used by admin for seeding or testing
    ApiUpdate: BotUpdateInput | ProfileUpdateInput,
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
