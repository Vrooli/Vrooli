import { ModelType } from "@local/shared";
import { ApiKeyExternalModelInfo, ApiKeyModelInfo, AwardModelInfo, BookmarkListModelInfo, BookmarkModelInfo, ChatInviteModelInfo, ChatMessageModelInfo, ChatModelInfo, ChatParticipantModelInfo, CommentModelInfo, EmailModelInfo, IssueModelInfo, MeetingInviteModelInfo, MeetingModelInfo, MemberInviteModelInfo, MemberModelInfo, NotificationModelInfo, NotificationSubscriptionModelInfo, PaymentModelInfo, PhoneModelInfo, PremiumModelInfo, PullRequestModelInfo, PushDeviceModelInfo, ReactionModelInfo, ReactionSummaryModelInfo, ReminderItemModelInfo, ReminderListModelInfo, ReminderModelInfo, ReportModelInfo, ReportResponseModelInfo, RoutineModelInfo, RoutineVersionInputModelInfo, RoutineVersionModelInfo, RoutineVersionOutputModelInfo, RunRoutineIOModelInfo, RunRoutineModelInfo, RunRoutineStepModelInfo, ScheduleExceptionModelInfo, ScheduleModelInfo, ScheduleRecurrenceModelInfo, SessionModelInfo, StatsSiteModelInfo, StatsTeamModelInfo, StatsUserModelInfo, TagModelInfo, TeamModelInfo, TransferModelInfo, UserModelInfo, ViewModelInfo, WalletModelInfo } from "./base/types.js";
import { Formatter } from "./types.js";

export const ApiKeyFormat: Formatter<ApiKeyModelInfo> = {
    apiRelMap: {
        __typename: "ApiKey",
    },
    prismaRelMap: {
        __typename: "ApiKey",
    },
    countFields: {},
    hiddenFields: ["key"],
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
        translationsCount: true,
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
        apis: "Api",
        codes: "Code",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        forks: "Team",
        issues: "Issue",
        meetings: "Meeting",
        members: "Member",
        notes: "Note",
        parent: "Team",
        paymentHistory: "Payment",
        premium: "Premium",
        projects: "Project",
        reports: "Report",
        routines: "Routine",
        standards: "Standard",
        bookmarkedBy: "User",
        tags: "Tag",
        transfersIncoming: "Transfer",
        transfersOutgoing: "Transfer",
        wallets: "Wallet",
    },
    prismaRelMap: {
        __typename: "Team",
        createdBy: "User",
        directoryListings: "ProjectVersionDirectory",
        issues: "Issue",
        notes: "Note",
        apis: "Api",
        codes: "Code",
        comments: "Comment",
        forks: "Team",
        meetings: "Meeting",
        parent: "Team",
        tags: "Tag",
        members: "Member",
        memberInvites: "MemberInvite",
        projects: "Project",
        reports: "Report",
        routines: "Routine",
        runRoutines: "RunRoutine",
        standards: "Standard",
        bookmarkedBy: "User",
        transfersIncoming: "Transfer",
        transfersOutgoing: "Transfer",
        wallets: "Wallet",
    },
    joinMap: { bookmarkedBy: "user", tags: "tag" },
    countFields: {
        apisCount: true,
        codesCount: true,
        commentsCount: true,
        issuesCount: true,
        meetingsCount: true,
        membersCount: true,
        notesCount: true,
        projectsCount: true,
        reportsCount: true,
        routinesCount: true,
        standardsCount: true,
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

export const PremiumFormat: Formatter<PremiumModelInfo> = {
    apiRelMap: {
        __typename: "Premium",
    },
    prismaRelMap: {
        __typename: "Premium",
        team: "Team",
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
            fromApiVersion: "ApiVersion",
            fromCodeVersion: "CodeVersion",
            fromNoteVersion: "NoteVersion",
            fromProjectVersion: "ProjectVersion",
            fromRoutineVersion: "RoutineVersion",
            fromStandardVersion: "StandardVersion",
        },
        to: {
            toApi: "Api",
            toCode: "Code",
            toNote: "Note",
            toProject: "Project",
            toRoutine: "Routine",
            toStandard: "Standard",
        },
    },
    unionFields: {
        from: { connectField: "fromConnect", typeField: "fromObjectType" },
        to: { connectField: "toConnect", typeField: "toObjectType" },
    },
    prismaRelMap: {
        __typename: "PullRequest",
        fromApiVersion: "ApiVersion",
        fromCodeVersion: "CodeVersion",
        fromNoteVersion: "NoteVersion",
        fromProjectVersion: "ProjectVersion",
        fromRoutineVersion: "RoutineVersion",
        fromStandardVersion: "StandardVersion",
        toApi: "Api",
        toCode: "Code",
        toNote: "Note",
        toProject: "Project",
        toRoutine: "Routine",
        toStandard: "Standard",
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
            api: "Api",
            chatMessage: "ChatMessage",
            code: "Code",
            comment: "Comment",
            issue: "Issue",
            note: "Note",
            project: "Project",
            routine: "Routine",
            standard: "Standard",
        },
    },
    unionFields: {
        to: { connectField: "forConnect", typeField: "reactionFor" },
    },
    prismaRelMap: {
        __typename: "Reaction",
        by: "User",
        api: "Api",
        chatMessage: "ChatMessage",
        code: "Code",
        comment: "Comment",
        issue: "Issue",
        note: "Note",
        project: "Project",
        routine: "Routine",
        standard: "Standard",
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
            apiVersion: "ApiVersion",
            chatMessage: "ChatMessage",
            codeVersion: "CodeVersion",
            comment: "Comment",
            issue: "Issue",
            noteVersion: "NoteVersion",
            projectVersion: "ProjectVersion",
            routineVersion: "RoutineVersion",
            standardVersion: "StandardVersion",
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
        apiVersion: "ApiVersion",
        chatMessage: "ChatMessage",
        codeVersion: "CodeVersion",
        comment: "Comment",
        issue: "Issue",
        noteVersion: "NoteVersion",
        projectVersion: "ProjectVersion",
        routineVersion: "RoutineVersion",
        standardVersion: "StandardVersion",
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

export const RoutineFormat: Formatter<RoutineModelInfo> = {
    apiRelMap: {
        __typename: "Routine",
        createdBy: "User",
        owner: {
            ownedByTeam: "Team",
            ownedByUser: "User",
        },
        forks: "Routine",
        issues: "Issue",
        parent: "RoutineVersion",
        bookmarkedBy: "User",
        tags: "Tag",
        versions: "RoutineVersion",
    },
    unionFields: {
        owner: {},
    },
    prismaRelMap: {
        __typename: "Routine",
        createdBy: "User",
        ownedByTeam: "Team",
        ownedByUser: "User",
        parent: "RoutineVersion",
        issues: "Issue",
        pullRequests: "PullRequest",
        bookmarkedBy: "User",
        stats: "StatsRoutine",
        tags: "Tag",
        transfers: "Transfer",
        versions: "RoutineVersion",
        viewedBy: "View",
    },
    joinMap: { tags: "tag", bookmarkedBy: "user" },
    countFields: {
        forksCount: true,
        issuesCount: true,
        pullRequestsCount: true,
        transfersCount: true,
        versionsCount: true,
    },
};

export const RoutineVersionFormat: Formatter<RoutineVersionModelInfo> = {
    apiRelMap: {
        __typename: "RoutineVersion",
        apiVersion: "ApiVersion",
        codeVersion: "CodeVersion",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        forks: "Routine",
        inputs: "RoutineVersionInput",
        outputs: "RoutineVersionOutput",
        pullRequest: "PullRequest",
        reports: "Report",
        root: "Routine",
        subroutineLinks: "RoutineVersion",
    },
    prismaRelMap: {
        __typename: "RoutineVersion",
        apiVersion: "ApiVersion",
        codeVersion: "CodeVersion",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        reports: "Report",
        root: "Routine",
        forks: "Routine",
        inputs: "RoutineVersionInput",
        outputs: "RoutineVersionOutput",
        parentRoutineLinks: "RoutineVersion",
        pullRequest: "PullRequest",
        runRoutines: "RunRoutine",
        runSteps: "RunRoutineStep",
        subroutineLinks: "RoutineVersion",
    },
    joinMap: {
        subroutineLinks: "subroutine",
    },
    countFields: {
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        inputsCount: true,
        outputsCount: true,
        reportsCount: true,
        translationsCount: true,
    },
};

export const RoutineVersionInputFormat: Formatter<RoutineVersionInputModelInfo> = {
    apiRelMap: {
        __typename: "RoutineVersionInput",
        routineVersion: "RoutineVersion",
        standardVersion: "StandardVersion",
    },
    prismaRelMap: {
        __typename: "RoutineVersionInput",
        routineVersion: "RoutineVersion",
        standardVersion: "StandardVersion",
        runIO: "RunRoutineIO",
    },
    countFields: {},
};

export const RoutineVersionOutputFormat: Formatter<RoutineVersionOutputModelInfo> = {
    apiRelMap: {
        __typename: "RoutineVersionOutput",
        routineVersion: "RoutineVersion",
        standardVersion: "StandardVersion",
    },
    prismaRelMap: {
        __typename: "RoutineVersionOutput",
        routineVersion: "RoutineVersion",
        standardVersion: "StandardVersion",
        runIO: "RunRoutineIO",
    },
    countFields: {},
};

export const RunRoutineFormat: Formatter<RunRoutineModelInfo> = {
    apiRelMap: {
        __typename: "RunRoutine",
        io: "RunRoutineIO",
        team: "Team",
        routineVersion: "RoutineVersion",
        schedule: "Schedule",
        steps: "RunRoutineStep",
        user: "User",
    },
    prismaRelMap: {
        __typename: "RunRoutine",
        io: "RunRoutineIO",
        team: "Team",
        routineVersion: "RoutineVersion",
        schedule: "Schedule",
        steps: "RunRoutineStep",
        user: "User",
    },
    countFields: {
        ioCount: true,
        stepsCount: true,
    },
};

export const RunRoutineIOFormat: Formatter<RunRoutineIOModelInfo> = {
    apiRelMap: {
        __typename: "RunRoutineIO",
        runRoutine: "RunRoutine",
        routineVersionInput: "RoutineVersionInput",
        routineVersionOutput: "RoutineVersionOutput",
    },
    prismaRelMap: {
        __typename: "RunRoutineIO",
        runRoutine: "RunRoutine",
        routineVersionInput: "RoutineVersionInput",
        routineVersionOutput: "RoutineVersionOutput",
    },
    countFields: {},
};

export const RunRoutineStepFormat: Formatter<RunRoutineStepModelInfo> = {
    apiRelMap: {
        __typename: "RunRoutineStep",
        runRoutine: "RunRoutine",
        subroutine: "RoutineVersion",
    },
    prismaRelMap: {
        __typename: "RunRoutineStep",
        runRoutine: "RunRoutine",
        subroutine: "RoutineVersion",
    },
    countFields: {},
};

export const ScheduleFormat: Formatter<ScheduleModelInfo> = {
    apiRelMap: {
        __typename: "Schedule",
        exceptions: "ScheduleException",
        recurrences: "ScheduleRecurrence",
        runProjects: "RunProject",
        runRoutines: "RunRoutine",
        meetings: "Meeting",
    },
    prismaRelMap: {
        __typename: "Schedule",
        exceptions: "ScheduleException",
        recurrences: "ScheduleRecurrence",
        runProjects: "RunProject",
        runRoutines: "RunRoutine",
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

export const SessionFormat: Formatter<SessionModelInfo> = {
    apiRelMap: {
        __typename: "Session",
    },
    prismaRelMap: {
        __typename: "Session",
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
        apis: "Api",
        codes: "Code",
        notes: "Note",
        projects: "Project",
        reports: "Report",
        routines: "Routine",
        standards: "Standard",
        teams: "Team",
        bookmarkedBy: "User",
    },
    prismaRelMap: {
        __typename: "Tag",
        createdBy: "User",
        apis: "Api",
        codes: "Code",
        notes: "Note",
        projects: "Project",
        reports: "Report",
        routines: "Routine",
        standards: "Standard",
        teams: "Team",
        bookmarkedBy: "User",
    },
    joinMap: {
        apis: "tagged",
        codes: "tagged",
        notes: "tagged",
        projects: "tagged",
        reports: "tagged",
        routines: "tagged",
        standards: "tagged",
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
            api: "Api",
            code: "Code",
            note: "Note",
            project: "Project",
            routine: "Routine",
            standard: "Standard",
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
        api: "Api",
        code: "Code",
        note: "Note",
        project: "Project",
        routine: "Routine",
        standard: "Standard",
    },
    countFields: {},
};

export const UserFormat: Formatter<UserModelInfo> = {
    apiRelMap: {
        __typename: "User",
        comments: "Comment",
        emails: "Email",
        // phones: 'Phone',
        projects: "Project",
        pushDevices: "PushDevice",
        bookmarkedBy: "User",
        reportsCreated: "Report",
        reportsReceived: "Report",
        routines: "Routine",
    },
    prismaRelMap: {
        __typename: "User",
        apis: "Api",
        apiKeys: "ApiKey",
        codesCreated: "Code",
        codes: "Code",
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
        projectsCreated: "Project",
        projects: "Project",
        pullRequests: "PullRequest",
        reportsCreated: "Report",
        reportsReceived: "Report",
        reportResponses: "ReportResponse",
        routinesCreated: "Routine",
        routines: "Routine",
        runProjects: "RunProject",
        runRoutines: "RunRoutine",
        standardsCreated: "Standard",
        standards: "Standard",
        bookmarkedBy: "User",
        tags: "Tag",
        teamsCreated: "Team",
        transfersIncoming: "Transfer",
        transfersOutgoing: "Transfer",
        notesCreated: "Note",
        notes: "Note",
        wallets: "Wallet",
        stats: "StatsUser",
    },
    joinMap: {
        meetingsAttending: "user",
        bookmarkedBy: "user",
    },
    countFields: {
        apisCount: true,
        codesCount: true,
        membershipsCount: true,
        notesCount: true,
        projectsCount: true,
        reportsReceivedCount: true,
        routinesCount: true,
        standardsCount: true,
    },
};

export const ViewFormat: Formatter<ViewModelInfo> = {
    apiRelMap: {
        __typename: "View",
        by: "User",
        to: {
            api: "Api",
            code: "Code",
            issue: "Issue",
            note: "Note",
            project: "Project",
            routine: "Routine",
            standard: "Standard",
            team: "Team",
            user: "User",
        },
    },
    // View is never called directly, so we don't need to worry about the unionFields - at least for now
    prismaRelMap: {
        __typename: "View",
        by: "User",
        api: "Api",
        code: "Code",
        issue: "Issue",
        note: "Note",
        project: "Project",
        routine: "Routine",
        standard: "Standard",
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
    Premium: PremiumFormat,
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
    RoutineVersionInput: RoutineVersionInputFormat,
    RoutineVersionOutput: RoutineVersionOutputFormat,
    Run: RunFormat,
    RunIO: RunIOFormat,
    RunStep: RunStepFormat,
    Schedule: ScheduleFormat,
    ScheduleException: ScheduleExceptionFormat,
    ScheduleRecurrence: ScheduleRecurrenceFormat,
    Session: SessionFormat,
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
