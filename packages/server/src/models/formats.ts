import { type ModelType } from "@vrooli/shared";
import { type ApiKeyExternalModelInfo, type ApiKeyModelInfo, type AwardModelInfo, type BookmarkListModelInfo, type BookmarkModelInfo, type ChatInviteModelInfo, type ChatMessageModelInfo, type ChatModelInfo, type ChatParticipantModelInfo, type CommentModelInfo, type EmailModelInfo, type IssueModelInfo, type MeetingInviteModelInfo, type MeetingModelInfo, type MemberInviteModelInfo, type MemberModelInfo, type NotificationModelInfo, type NotificationSubscriptionModelInfo, type PaymentModelInfo, type PhoneModelInfo, type PullRequestModelInfo, type PushDeviceModelInfo, type ReactionModelInfo, type ReactionSummaryModelInfo, type ReminderItemModelInfo, type ReminderListModelInfo, type ReminderModelInfo, type ReportModelInfo, type ReportResponseModelInfo, type ResourceModelInfo, type ResourceVersionModelInfo, type ResourceVersionRelationModelInfo, type RunIOModelInfo, type RunModelInfo, type RunStepModelInfo, type ScheduleExceptionModelInfo, type ScheduleModelInfo, type ScheduleRecurrenceModelInfo, type StatsResourceModelInfo, type StatsSiteModelInfo, type StatsTeamModelInfo, type StatsUserModelInfo, type TagModelInfo, type TeamModelInfo, type TransferModelInfo, type UserModelInfo, type ViewModelInfo, type WalletModelInfo } from "./base/types.js";
import { type Formatter } from "./types.js";

export const ApiKeyFormat: Formatter<ApiKeyModelInfo> = {
    apiRelMap: {
        __typename: "ApiKey",
    },
    prismaRelMap: {
        __typename: "ApiKey",
    },
    countFields: {},
};

export const ApiKeyExternalFormat: Formatter<ApiKeyExternalModelInfo> = {
    apiRelMap: {
        __typename: "ApiKeyExternal",
    },
    prismaRelMap: {
        __typename: "ApiKeyExternal",
    },
    countFields: {},
    hiddenFields: ["key"],
};

export const AwardFormat: Formatter<AwardModelInfo> = {
    apiRelMap: {
        __typename: "Award",
    },
    prismaRelMap: {
        __typename: "Award",
        user: "User",
    },
    countFields: {},
};

export const BookmarkFormat: Formatter<BookmarkModelInfo> = {
    apiRelMap: {
        __typename: "Bookmark",
        by: "User",
        list: "BookmarkList",
        to: {
            comment: "Comment",
            issue: "Issue",
            list: "BookmarkList",
            resource: "Resource",
            tag: "Tag",
            team: "Team",
            user: "User",
        },
    },
    unionFields: {
        to: { connectField: "forConnect", typeField: "bookmarkFor" },
    },
    prismaRelMap: {
        __typename: "Bookmark",
        comment: "Comment",
        issue: "Issue",
        list: "BookmarkList",
        resource: "Resource",
        tag: "Tag",
        team: "Team",
        user: "User",
    },
    countFields: {},
};

export const BookmarkListFormat: Formatter<BookmarkListModelInfo> = {
    apiRelMap: {
        __typename: "BookmarkList",
        bookmarks: "Bookmark",
    },
    prismaRelMap: {
        __typename: "BookmarkList",
        bookmarks: "Bookmark",
    },
    countFields: {
        bookmarksCount: true,
    },
};

export const ChatFormat: Formatter<ChatModelInfo> = {
    apiRelMap: {
        __typename: "Chat",
        team: "Team",
        messages: "ChatMessage",
        participants: "ChatParticipant",
        invites: "ChatInvite",
    },
    prismaRelMap: {
        __typename: "Chat",
        creator: "User",
        team: "Team",
        messages: "ChatMessage",
        participants: "ChatParticipant",
        invites: "ChatInvite",
    },
    countFields: {
        participantsCount: true,
        invitesCount: true,
        translationsCount: true,
    },
};

export const ChatInviteFormat: Formatter<ChatInviteModelInfo> = {
    apiRelMap: {
        __typename: "ChatInvite",
        chat: "Chat",
        user: "User",
    },
    prismaRelMap: {
        __typename: "ChatInvite",
        chat: "Chat",
        user: "User",
    },
    joinMap: {},
    countFields: {},
};

export const ChatMessageFormat: Formatter<ChatMessageModelInfo> = {
    apiRelMap: {
        __typename: "ChatMessage",
        chat: "Chat",
        parent: "ChatMessage",
        user: "User",
        reactionSummaries: "ReactionSummary",
        reports: "Report",
    },
    prismaRelMap: {
        __typename: "ChatMessage",
        chat: "Chat",
        parent: "ChatMessage",
        children: "ChatMessage",
        user: "User",
        reactionSummaries: "ReactionSummary",
        reports: "Report",
    },
    joinMap: {},
    countFields: {
        reportsCount: true,
    },
};

export const ChatParticipantFormat: Formatter<ChatParticipantModelInfo> = {
    apiRelMap: {
        __typename: "ChatParticipant",
        chat: "Chat",
        user: "User",
    },
    prismaRelMap: {
        __typename: "ChatParticipant",
        chat: "Chat",
        user: "User",
    },
    joinMap: {},
    countFields: {},
};

export const CommentFormat: Formatter<CommentModelInfo> = {
    apiRelMap: {
        __typename: "Comment",
        owner: {
            ownedByTeam: "Team",
            ownedByUser: "User",
        },
        commentedOn: {
            issue: "Issue",
            pullRequest: "PullRequest",
            resourceVersion: "ResourceVersion",
        },
        reports: "Report",
        bookmarkedBy: "User",
    },
    unionFields: {
        commentedOn: { connectField: "forConnect", typeField: "createdFor" },
        owner: {},
    },
    prismaRelMap: {
        __typename: "Comment",
        ownedByTeam: "Team",
        ownedByUser: "User",
        issue: "Issue",
        parent: "Comment",
        pullRequest: "PullRequest",
        resourceVersion: "ResourceVersion",
        reports: "Report",
        bookmarkedBy: "User",
        reactions: "Reaction",
        parents: "Comment",
    },
    joinMap: { bookmarkedBy: "user" },
    countFields: {
        reportsCount: true,
        translationsCount: true,
    },
};

export const EmailFormat: Formatter<EmailModelInfo> = {
    apiRelMap: {
        __typename: "Email",
    },
    prismaRelMap: {
        __typename: "Email",
        user: "User",
    },
    countFields: {},
};

export const IssueFormat: Formatter<IssueModelInfo> = {
    apiRelMap: {
        __typename: "Issue",
        closedBy: "User",
        comments: "Comment",
        createdBy: "User",
        reports: "Report",
        bookmarkedBy: "User",
        to: {
            resource: "Resource",
            team: "Team",
        },
    },
    unionFields: {
        to: { connectField: "forConnect", typeField: "issueFor" },
    },
    prismaRelMap: {
        __typename: "Issue",
        resource: "Resource",
        closedBy: "User",
        comments: "Comment",
        reports: "Report",
        reactions: "Reaction",
        team: "Team",
        bookmarkedBy: "User",
        viewedBy: "View",
    },
    joinMap: { bookmarkedBy: "user" },
    countFields: {
        commentsCount: true,
        reportsCount: true,
        translationsCount: true,
    },
};

export const MeetingFormat: Formatter<MeetingModelInfo> = {
    apiRelMap: {
        __typename: "Meeting",
        attendees: "User",
        invites: "MeetingInvite",
        schedule: "Schedule",
        team: "Team",
    },
    prismaRelMap: {
        __typename: "Meeting",
        attendees: "User",
        invites: "MeetingInvite",
        schedule: "Schedule",
        team: "Team",
    },
    joinMap: {
        attendees: "user",
    },
    countFields: {
        attendeesCount: true,
        invitesCount: true,
        translationsCount: true,
    },
};

export const MeetingInviteFormat: Formatter<MeetingInviteModelInfo> = {
    apiRelMap: {
        __typename: "MeetingInvite",
        meeting: "Meeting",
        user: "User",
    },
    prismaRelMap: {
        __typename: "MeetingInvite",
        meeting: "Meeting",
        user: "User",
    },
    countFields: {},
};

export const MemberFormat: Formatter<MemberModelInfo> = {
    apiRelMap: {
        __typename: "Member",
        team: "Team",
        user: "User",
    },
    prismaRelMap: {
        __typename: "Member",
        team: "Team",
        user: "User",
    },
    countFields: {},
};

export const MemberInviteFormat: Formatter<MemberInviteModelInfo> = {
    apiRelMap: {
        __typename: "MemberInvite",
        team: "Team",
        user: "User",
    },
    prismaRelMap: {
        __typename: "MemberInvite",
        team: "Team",
        user: "User",
    },
    countFields: {},
};

export const NotificationFormat: Formatter<NotificationModelInfo> = {
    apiRelMap: {
        __typename: "Notification",
    },
    prismaRelMap: {
        __typename: "Notification",
    },
    countFields: {},
};

export const NotificationSubscriptionFormat: Formatter<NotificationSubscriptionModelInfo> = {
    apiRelMap: {
        __typename: "NotificationSubscription",
        object: {
            comment: "Comment",
            issue: "Issue",
            meeting: "Meeting",
            pullRequest: "PullRequest",
            report: "Report",
            resource: "Resource",
            schedule: "Schedule",
            team: "Team",
        },
    },
    unionFields: {
        object: { connectField: "objectConnect", typeField: "objectType" },
    },
    prismaRelMap: {
        __typename: "NotificationSubscription",
        comment: "Comment",
        issue: "Issue",
        meeting: "Meeting",
        pullRequest: "PullRequest",
        report: "Report",
        resource: "Resource",
        schedule: "Schedule",
        subscriber: "User",
        team: "Team",
    },
    countFields: {},
};

export const TeamFormat: Formatter<TeamModelInfo> = {
    apiRelMap: {
        __typename: "Team",
        comments: "Comment",
        forks: "Team",
        issues: "Issue",
        meetings: "Meeting",
        members: "Member",
        parent: "Team",
        paymentHistory: "Payment",
        premium: "Premium",
        reports: "Report",
        resources: "Resource",
        bookmarkedBy: "User",
        tags: "Tag",
        transfersIncoming: "Transfer",
        transfersOutgoing: "Transfer",
        wallets: "Wallet",
    },
    prismaRelMap: {
        __typename: "Team",
        createdBy: "User",
        issues: "Issue",
        comments: "Comment",
        forks: "Team",
        meetings: "Meeting",
        parent: "Team",
        tags: "Tag",
        members: "Member",
        memberInvites: "MemberInvite",
        reports: "Report",
        resources: "Resource",
        runs: "Run",
        bookmarkedBy: "User",
        transfersIncoming: "Transfer",
        transfersOutgoing: "Transfer",
        wallets: "Wallet",
    },
    joinMap: { bookmarkedBy: "user", tags: "tag" },
    countFields: {
        commentsCount: true,
        issuesCount: true,
        meetingsCount: true,
        membersCount: true,
        reportsCount: true,
        resourcesCount: true,
        translationsCount: true,
    },
};

export const PaymentFormat: Formatter<PaymentModelInfo> = {
    apiRelMap: {
        __typename: "Payment",
        team: "Team",
        user: "User",
    },
    prismaRelMap: {
        __typename: "Payment",
        team: "Team",
        user: "User",
    },
    countFields: {},
};

export const PhoneFormat: Formatter<PhoneModelInfo> = {
    apiRelMap: {
        __typename: "Phone",
    },
    prismaRelMap: {
        __typename: "Phone",
        user: "User",
    },
    countFields: {},
};


export const PullRequestFormat: Formatter<PullRequestModelInfo> = {
    apiRelMap: {
        __typename: "PullRequest",
        createdBy: "User",
        comments: "Comment",
        from: {
            fromResourceVersion: "ResourceVersion",
        },
        to: {
            toResource: "Resource",
        },
    },
    unionFields: {
        from: { connectField: "fromConnect", typeField: "fromObjectType" },
        to: { connectField: "toConnect", typeField: "toObjectType" },
    },
    prismaRelMap: {
        __typename: "PullRequest",
        fromResourceVersion: "ResourceVersion",
        toResource: "Resource",
        createdBy: "User",
        comments: "Comment",
    },
    countFields: {
        commentsCount: true,
        translationsCount: true,
    },
};

export const PushDeviceFormat: Formatter<PushDeviceModelInfo> = {
    apiRelMap: {
        __typename: "PushDevice",
    },
    prismaRelMap: {
        __typename: "PushDevice",
        user: "User",
    },
    countFields: {},
};

export const ReactionFormat: Formatter<ReactionModelInfo> = {
    apiRelMap: {
        __typename: "Reaction",
        by: "User",
        to: {
            chatMessage: "ChatMessage",
            comment: "Comment",
            issue: "Issue",
            resource: "Resource",
        },
    },
    unionFields: {
        to: { connectField: "forConnect", typeField: "reactionFor" },
    },
    prismaRelMap: {
        __typename: "Reaction",
        by: "User",
        chatMessage: "ChatMessage",
        comment: "Comment",
        issue: "Issue",
        resource: "Resource",
    },
    countFields: {},
};

export const ReactionSummaryFormat: Formatter<ReactionSummaryModelInfo> = {
    apiRelMap: {
        __typename: "ReactionSummary",
    },
    prismaRelMap: {
        __typename: "ReactionSummary",
    },
    countFields: {},
};

export const ReminderFormat: Formatter<ReminderModelInfo> = {
    apiRelMap: {
        __typename: "Reminder",
        reminderItems: "ReminderItem",
        reminderList: "ReminderList",
    },
    prismaRelMap: {
        __typename: "Reminder",
        reminderItems: "ReminderItem",
        reminderList: "ReminderList",
    },
    countFields: {},
};

export const ReminderItemFormat: Formatter<ReminderItemModelInfo> = {
    apiRelMap: {
        __typename: "ReminderItem",
        reminder: "Reminder",
    },
    prismaRelMap: {
        __typename: "ReminderItem",
        reminder: "Reminder",
    },
    countFields: {},
};

export const ReminderListFormat: Formatter<ReminderListModelInfo> = {
    apiRelMap: {
        __typename: "ReminderList",
        reminders: "Reminder",
    },
    prismaRelMap: {
        __typename: "ReminderList",
        reminders: "Reminder",
        user: "User",
    },
    countFields: {},
};

export const ReportFormat: Formatter<ReportModelInfo> = {
    apiRelMap: {
        __typename: "Report",
        responses: "ReportResponse",
        createdFor: {
            chatMessage: "ChatMessage",
            comment: "Comment",
            issue: "Issue",
            resourceVersion: "ResourceVersion",
            tag: "Tag",
            team: "Team",
            user: "User",
        },
    },
    unionFields: {
        createdFor: { connectField: "createdForConnect", typeField: "createdForType" },
    },
    prismaRelMap: {
        __typename: "Report",
        responses: "ReportResponse",
        chatMessage: "ChatMessage",
        comment: "Comment",
        issue: "Issue",
        resourceVersion: "ResourceVersion",
        tag: "Tag",
        team: "Team",
        user: "User",
    },
    hiddenFields: ["createdById"], // Always hide report creator,
    countFields: {
        responsesCount: true,
    },
};

export const ReportResponseFormat: Formatter<ReportResponseModelInfo> = {
    apiRelMap: {
        __typename: "ReportResponse",
        report: "Report",
    },
    prismaRelMap: {
        __typename: "ReportResponse",
        report: "Report",
    },
    hiddenFields: ["createdById"], // Always hide report creator
    countFields: {
        responsesCount: true,
    },
};

export const ResourceFormat: Formatter<ResourceModelInfo> = {
    apiRelMap: {
        __typename: "Resource",
        createdBy: "User",
        owner: {
            ownedByTeam: "Team",
            ownedByUser: "User",
        },
        issues: "Issue",
        parent: "ResourceVersion",
        bookmarkedBy: "User",
        tags: "Tag",
        versions: "ResourceVersion",
    },
    unionFields: {
        owner: {},
    },
    prismaRelMap: {
        __typename: "Resource",
        createdBy: "User",
        ownedByTeam: "Team",
        ownedByUser: "User",
        parent: "ResourceVersion",
        issues: "Issue",
        pullRequests: "PullRequest",
        bookmarkedBy: "User",
        stats: "StatsResource",
        tags: "Tag",
        transfers: "Transfer",
        versions: "ResourceVersion",
        viewedBy: "View",
    },
    joinMap: { tags: "tag", bookmarkedBy: "user" },
    countFields: {
        issuesCount: true,
        pullRequestsCount: true,
        transfersCount: true,
        versionsCount: true,
    },
};

export const ResourceVersionFormat: Formatter<ResourceVersionModelInfo> = {
    apiRelMap: {
        __typename: "ResourceVersion",
        comments: "Comment",
        forks: "Resource",
        pullRequest: "PullRequest",
        reports: "Report",
        root: "Resource",
        relatedVersions: "ResourceVersionRelation",
    },
    prismaRelMap: {
        __typename: "ResourceVersion",
        comments: "Comment",
        reports: "Report",
        root: "Resource",
        forks: "Resource",
        pullRequest: "PullRequest",
        runs: "Run",
        relatedVersions: "ResourceVersionRelation",
    },
    countFields: {
        commentsCount: true,
        forksCount: true,
        reportsCount: true,
        translationsCount: true,
    },
};

export const ResourceVersionRelationFormat: Formatter<ResourceVersionRelationModelInfo> = {
    apiRelMap: {
        __typename: "ResourceVersionRelation",
        toVersion: "ResourceVersion",
    },
    prismaRelMap: {
        __typename: "ResourceVersionRelation",
        fromVersion: "ResourceVersion",
        toVersion: "ResourceVersion",
    },
    countFields: {},
};

export const RunFormat: Formatter<RunModelInfo> = {
    apiRelMap: {
        __typename: "Run",
        io: "RunIO",
        team: "Team",
        resourceVersion: "ResourceVersion",
        schedule: "Schedule",
        steps: "RunStep",
        user: "User",
    },
    prismaRelMap: {
        __typename: "Run",
        io: "RunIO",
        team: "Team",
        resourceVersion: "ResourceVersion",
        schedule: "Schedule",
        steps: "RunStep",
        user: "User",
    },
    countFields: {
        ioCount: true,
        stepsCount: true,
    },
};

export const RunIOFormat: Formatter<RunIOModelInfo> = {
    apiRelMap: {
        __typename: "RunIO",
        run: "Run",
    },
    prismaRelMap: {
        __typename: "RunIO",
        run: "Run",
    },
    countFields: {},
};

export const RunStepFormat: Formatter<RunStepModelInfo> = {
    apiRelMap: {
        __typename: "RunStep",
        resourceVersion: "ResourceVersion",
    },
    prismaRelMap: {
        __typename: "RunStep",
        run: "Run",
        resourceVersion: "ResourceVersion",
    },
    countFields: {},
};

export const ScheduleFormat: Formatter<ScheduleModelInfo> = {
    apiRelMap: {
        __typename: "Schedule",
        exceptions: "ScheduleException",
        recurrences: "ScheduleRecurrence",
        runs: "Run",
        meetings: "Meeting",
    },
    prismaRelMap: {
        __typename: "Schedule",
        exceptions: "ScheduleException",
        recurrences: "ScheduleRecurrence",
        runs: "Run",
        meetings: "Meeting",
    },
    countFields: {},
};

export const ScheduleExceptionFormat: Formatter<ScheduleExceptionModelInfo> = {
    apiRelMap: {
        __typename: "ScheduleException",
        schedule: "Schedule",
    },
    prismaRelMap: {
        __typename: "ScheduleException",
        schedule: "Schedule",
    },
    countFields: {},
};

export const ScheduleRecurrenceFormat: Formatter<ScheduleRecurrenceModelInfo> = {
    apiRelMap: {
        __typename: "ScheduleRecurrence",
        schedule: "Schedule",
    },
    prismaRelMap: {
        __typename: "ScheduleRecurrence",
        schedule: "Schedule",
    },
    countFields: {},
};

export const StatsResourceFormat: Formatter<StatsResourceModelInfo> = {
    apiRelMap: {
        __typename: "StatsResource",
    },
    prismaRelMap: {
        __typename: "StatsResource",
        resource: "Resource",
    },
    countFields: {},
};

export const StatsSiteFormat: Formatter<StatsSiteModelInfo> = {
    apiRelMap: {
        __typename: "StatsSite",
    },
    prismaRelMap: {
        __typename: "StatsSite",
    },
    countFields: {},
};

export const StatsTeamFormat: Formatter<StatsTeamModelInfo> = {
    apiRelMap: {
        __typename: "StatsTeam",
    },
    prismaRelMap: {
        __typename: "StatsTeam",
        team: "Team",
    },
    countFields: {},
};

export const StatsUserFormat: Formatter<StatsUserModelInfo> = {
    apiRelMap: {
        __typename: "StatsUser",
    },
    prismaRelMap: {
        __typename: "StatsUser",
        user: "User",
    },
    countFields: {},
};

export const TagFormat: Formatter<TagModelInfo> = {
    apiRelMap: {
        __typename: "Tag",
        reports: "Report",
        resources: "Resource",
        teams: "Team",
        bookmarkedBy: "User",
    },
    prismaRelMap: {
        __typename: "Tag",
        createdBy: "User",
        reports: "Report",
        resources: "Resource",
        teams: "Team",
        bookmarkedBy: "User",
    },
    joinMap: {
        reports: "tagged",
        resources: "tagged",
        teams: "tagged",
        bookmarkedBy: "user",
    },
    countFields: {},
};

export const TransferFormat: Formatter<TransferModelInfo> = {
    apiRelMap: {
        __typename: "Transfer",
        fromOwner: {
            fromTeam: "Team",
            fromUser: "User",
        },
        object: {
            resource: "Resource",
        },
        toOwner: {
            toTeam: "Team",
            toUser: "User",
        },
    },
    unionFields: {
        // fromOwner is yourself, so it's not included in the Create input - and thus not needed here (for now)
        object: { connectField: "objectConnect", typeField: "objectType" },
        toOwner: {},
    },
    prismaRelMap: {
        __typename: "Transfer",
        fromTeam: "Team",
        fromUser: "User",
        toTeam: "Team",
        toUser: "User",
        resource: "Resource",
    },
    countFields: {},
};

export const UserFormat: Formatter<UserModelInfo> = {
    apiRelMap: {
        __typename: "User",
        comments: "Comment",
        emails: "Email",
        // phones: 'Phone',
        pushDevices: "PushDevice",
        bookmarkedBy: "User",
        reportsCreated: "Report",
        reportsReceived: "Report",
        resourcesCreated: "Resource",
        resources: "Resource",
    },
    prismaRelMap: {
        __typename: "User",
        apiKeys: "ApiKey",
        comments: "Comment",
        emails: "Email",
        phones: "Phone",
        invitedByUser: "User",
        invitedUsers: "User",
        issuesCreated: "Issue",
        issuesClosed: "Issue",
        meetingsAttending: "Meeting",
        meetingsInvited: "MeetingInvite",
        pushDevices: "PushDevice",
        notifications: "Notification",
        memberships: "Member",
        pullRequests: "PullRequest",
        reportsCreated: "Report",
        reportsReceived: "Report",
        reportResponses: "ReportResponse",
        resourcesCreated: "Resource",
        resources: "Resource",
        runs: "Run",
        bookmarkedBy: "User",
        tags: "Tag",
        teamsCreated: "Team",
        transfersIncoming: "Transfer",
        transfersOutgoing: "Transfer",
        wallets: "Wallet",
        stats: "StatsUser",
    },
    joinMap: {
        meetingsAttending: "user",
        bookmarkedBy: "user",
    },
    countFields: {
        membershipsCount: true,
        reportsReceivedCount: true,
        resourcesCount: true,
    },
};

export const ViewFormat: Formatter<ViewModelInfo> = {
    apiRelMap: {
        __typename: "View",
        by: "User",
        to: {
            issue: "Issue",
            resource: "Resource",
            team: "Team",
            user: "User",
        },
    },
    // View is never called directly, so we don't need to worry about the unionFields - at least for now
    prismaRelMap: {
        __typename: "View",
        by: "User",
        issue: "Issue",
        resource: "Resource",
        team: "Team",
        user: "User",
    },
    countFields: {},
};

export const WalletFormat: Formatter<WalletModelInfo> = {
    apiRelMap: {
        __typename: "Wallet",
        team: "Team",
        user: "User",
    },
    prismaRelMap: {
        __typename: "Wallet",
        team: "Team",
        user: "User",
    },
    countFields: {},
};

/** Maps model types to their respective formatter logic */
export const FormatMap: { [key in ModelType]?: Formatter<any> } = {
    ApiKey: ApiKeyFormat,
    Award: AwardFormat,
    Bookmark: BookmarkFormat,
    BookmarkList: BookmarkListFormat,
    Chat: ChatFormat,
    ChatInvite: ChatInviteFormat,
    ChatMessage: ChatMessageFormat,
    ChatParticipant: ChatParticipantFormat,
    Comment: CommentFormat,
    Email: EmailFormat,
    Issue: IssueFormat,
    Meeting: MeetingFormat,
    MeetingInvite: MeetingInviteFormat,
    Member: MemberFormat,
    MemberInvite: MemberInviteFormat,
    Notification: NotificationFormat,
    NotificationSubscription: NotificationSubscriptionFormat,
    Payment: PaymentFormat,
    Phone: PhoneFormat,
    PullRequest: PullRequestFormat,
    PushDevice: PushDeviceFormat,
    Reaction: ReactionFormat,
    ReactionSummary: ReactionSummaryFormat,
    Reminder: ReminderFormat,
    ReminderItem: ReminderItemFormat,
    ReminderList: ReminderListFormat,
    Report: ReportFormat,
    ReportResponse: ReportResponseFormat,
    Resource: ResourceFormat,
    ResourceVersion: ResourceVersionFormat,
    ResourceVersionRelation: ResourceVersionRelationFormat,
    Run: RunFormat,
    RunIO: RunIOFormat,
    RunStep: RunStepFormat,
    Schedule: ScheduleFormat,
    ScheduleException: ScheduleExceptionFormat,
    ScheduleRecurrence: ScheduleRecurrenceFormat,
    StatsResource: StatsResourceFormat,
    StatsSite: StatsSiteFormat,
    StatsTeam: StatsTeamFormat,
    StatsUser: StatsUserFormat,
    Tag: TagFormat,
    Team: TeamFormat,
    Transfer: TransferFormat,
    User: UserFormat,
    View: ViewFormat,
    Wallet: WalletFormat,
};
