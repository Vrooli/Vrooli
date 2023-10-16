import { GqlModelType } from "@local/shared";
import { ApiKeyModelInfo, ApiModelInfo, ApiVersionModelInfo, AwardModelInfo, BookmarkListModelInfo, BookmarkModelInfo, ChatInviteModelInfo, ChatMessageModelInfo, ChatModelInfo, ChatParticipantModelInfo, CommentModelInfo, EmailModelInfo, FocusModeFilterModelInfo, FocusModeModelInfo, IssueModelInfo, LabelModelInfo, MeetingInviteModelInfo, MeetingModelInfo, MemberInviteModelInfo, MemberModelInfo, NodeEndModelInfo, NodeLinkModelInfo, NodeLinkWhenModelInfo, NodeLoopModelInfo, NodeLoopWhileModelInfo, NodeModelInfo, NodeRoutineListItemModelInfo, NodeRoutineListModelInfo, NoteModelInfo, NoteVersionModelInfo, NotificationModelInfo, NotificationSubscriptionModelInfo, OrganizationModelInfo, PaymentModelInfo, PhoneModelInfo, PostModelInfo, PremiumModelInfo, ProjectModelInfo, ProjectVersionDirectoryModelInfo, ProjectVersionModelInfo, PullRequestModelInfo, PushDeviceModelInfo, QuestionAnswerModelInfo, QuestionModelInfo, QuizAttemptModelInfo, QuizModelInfo, QuizQuestionModelInfo, QuizQuestionResponseModelInfo, ReactionModelInfo, ReactionSummaryModelInfo, ReminderItemModelInfo, ReminderListModelInfo, ReminderModelInfo, ReportModelInfo, ReportResponseModelInfo, ResourceListModelInfo, ResourceModelInfo, RoleModelInfo, RoutineModelInfo, RoutineVersionInputModelInfo, RoutineVersionModelInfo, RoutineVersionOutputModelInfo, RunProjectModelInfo, RunProjectStepModelInfo, RunRoutineInputModelInfo, RunRoutineModelInfo, RunRoutineStepModelInfo, ScheduleExceptionModelInfo, ScheduleModelInfo, ScheduleRecurrenceModelInfo, SmartContractModelInfo, SmartContractVersionModelInfo, StandardModelInfo, StandardVersionModelInfo, StatsApiModelInfo, StatsOrganizationModelInfo, StatsProjectModelInfo, StatsQuizModelInfo, StatsRoutineModelInfo, StatsSiteModelInfo, StatsSmartContractModelInfo, StatsStandardModelInfo, StatsUserModelInfo, TagModelInfo, TransferModelInfo, UserModelInfo, ViewModelInfo, WalletModelInfo } from "./base/types";
import { Formatter } from "./types";

export const ApiFormat: Formatter<ApiModelInfo> = {
    gqlRelMap: {
        __typename: "Api",
        createdBy: "User",
        owner: {
            ownedByUser: "User",
            ownedByOrganization: "Organization",
        },
        parent: "Api",
        tags: "Tag",
        versions: "ApiVersion",
        labels: "Label",
        issues: "Issue",
        pullRequests: "PullRequest",
        questions: "Question",
        bookmarkedBy: "User",
        stats: "StatsApi",
        transfers: "Transfer",
    },
    unionFields: {
        owner: {},
    },
    prismaRelMap: {
        __typename: "Api",
        createdBy: "User",
        ownedByUser: "User",
        ownedByOrganization: "Organization",
        parent: "ApiVersion",
        tags: "Tag",
        issues: "Issue",
        bookmarkedBy: "User",
        reactions: "Reaction",
        viewedBy: "View",
        pullRequests: "PullRequest",
        versions: "ApiVersion",
        labels: "Label",
        stats: "StatsApi",
        questions: "Question",
        transfers: "Transfer",
    },
    joinMap: { labels: "label", bookmarkedBy: "user", tags: "tag" },
    countFields: {
        issuesCount: true,
        pullRequestsCount: true,
        questionsCount: true,
        transfersCount: true,
        versionsCount: true,
    },
};

export const ApiKeyFormat: Formatter<ApiKeyModelInfo> = {
    gqlRelMap: {
        __typename: "ApiKey",
    },
    prismaRelMap: {
        __typename: "ApiKey",
    },
    countFields: {},
};

export const ApiVersionFormat: Formatter<ApiVersionModelInfo> = {
    gqlRelMap: {
        __typename: "ApiVersion",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        forks: "ApiVersion",
        pullRequest: "PullRequest",
        reports: "Report",
        resourceList: "ResourceList",
        root: "Api",
    },
    prismaRelMap: {
        __typename: "ApiVersion",
        calledByRoutineVersions: "RoutineVersion",
        comments: "Comment",
        reports: "Report",
        root: "Api",
        forks: "Api",
        resourceList: "ResourceList",
        pullRequest: "PullRequest",
        directoryListings: "ProjectVersionDirectory",
    },
    countFields: {
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        reportsCount: true,
    },
};

export const AwardFormat: Formatter<AwardModelInfo> = {
    gqlRelMap: {
        __typename: "Award",
    },
    prismaRelMap: {
        __typename: "Award",
        user: "User",
    },
    countFields: {},
};

export const BookmarkFormat: Formatter<BookmarkModelInfo> = {
    gqlRelMap: {
        __typename: "Bookmark",
        by: "User",
        list: "BookmarkList",
        to: {
            api: "Api",
            comment: "Comment",
            issue: "Issue",
            note: "Note",
            organization: "Organization",
            post: "Post",
            project: "Project",
            question: "Question",
            questionAnswer: "QuestionAnswer",
            quiz: "Quiz",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
            tag: "Tag",
            user: "User",
        },
    },
    unionFields: {
        to: { connectField: "forConnect", typeField: "bookmarkFor" },
    },
    prismaRelMap: {
        __typename: "Bookmark",
        api: "Api",
        comment: "Comment",
        issue: "Issue",
        list: "BookmarkList",
        note: "Note",
        organization: "Organization",
        post: "Post",
        project: "Project",
        question: "Question",
        questionAnswer: "QuestionAnswer",
        quiz: "Quiz",
        routine: "Routine",
        smartContract: "SmartContract",
        standard: "Standard",
        tag: "Tag",
        user: "User",
    },
    countFields: {},
};

export const BookmarkListFormat: Formatter<BookmarkListModelInfo> = {
    gqlRelMap: {
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
    gqlRelMap: {
        __typename: "Chat",
        organization: "Organization",
        restrictedToRoles: "Role",
        messages: "ChatMessage",
        participants: "ChatParticipant",
        invites: "ChatInvite",
        labels: "Label",
    },
    prismaRelMap: {
        __typename: "Chat",
        creator: "User",
        organization: "Organization",
        restrictedToRoles: "Role",
        messages: "ChatMessage",
        participants: "ChatParticipant",
        invites: "ChatInvite",
        labels: "Label",
    },
    joinMap: { labels: "label", restrictedToRoles: "role" },
    countFields: {
        participantsCount: true,
        invitesCount: true,
        labelsCount: true,
        translationsCount: true,
    },
};

export const ChatInviteFormat: Formatter<ChatInviteModelInfo> = {
    gqlRelMap: {
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
    gqlRelMap: {
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
    gqlRelMap: {
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
    gqlRelMap: {
        __typename: "Comment",
        owner: {
            ownedByUser: "User",
            ownedByOrganization: "Organization",
        },
        commentedOn: {
            apiVersion: "ApiVersion",
            issue: "Issue",
            noteVersion: "NoteVersion",
            post: "Post",
            projectVersion: "ProjectVersion",
            pullRequest: "PullRequest",
            question: "Question",
            questionAnswer: "QuestionAnswer",
            routineVersion: "RoutineVersion",
            smartContractVersion: "SmartContractVersion",
            standardVersion: "StandardVersion",
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
        ownedByUser: "User",
        ownedByOrganization: "Organization",
        apiVersion: "ApiVersion",
        issue: "Issue",
        noteVersion: "NoteVersion",
        parent: "Comment",
        post: "Post",
        projectVersion: "ProjectVersion",
        pullRequest: "PullRequest",
        question: "Question",
        questionAnswer: "QuestionAnswer",
        routineVersion: "RoutineVersion",
        smartContractVersion: "SmartContractVersion",
        standardVersion: "StandardVersion",
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
    gqlRelMap: {
        __typename: "Email",
    },
    prismaRelMap: {
        __typename: "Email",
        user: "User",
    },
    countFields: {},
};

export const FocusModeFormat: Formatter<FocusModeModelInfo> = {
    gqlRelMap: {
        __typename: "FocusMode",
        filters: "FocusModeFilter",
        labels: "Label",
        reminderList: "ReminderList",
        schedule: "Schedule",
    },
    prismaRelMap: {
        __typename: "FocusMode",
        reminderList: "ReminderList",
        resourceList: "ResourceList",
        user: "User",
        labels: "Label",
        filters: "FocusModeFilter",
        schedule: "Schedule",
    },
    countFields: {},
    joinMap: { labels: "label" },
};

export const FocusModeFilterFormat: Formatter<FocusModeFilterModelInfo> = {
    gqlRelMap: {
        __typename: "FocusModeFilter",
        focusMode: "FocusMode",
        tag: "Tag",
    },
    prismaRelMap: {
        __typename: "FocusModeFilter",
        focusMode: "FocusMode",
        tag: "Tag",
    },
    countFields: {},
};

export const IssueFormat: Formatter<IssueModelInfo> = {
    gqlRelMap: {
        __typename: "Issue",
        closedBy: "User",
        comments: "Comment",
        createdBy: "User",
        labels: "Label",
        reports: "Report",
        bookmarkedBy: "User",
        to: {
            api: "Api",
            organization: "Organization",
            note: "Note",
            project: "Project",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
        },
    },
    unionFields: {
        to: { connectField: "forConnect", typeField: "issueFor" },
    },
    prismaRelMap: {
        __typename: "Issue",
        api: "Api",
        organization: "Organization",
        note: "Note",
        project: "Project",
        routine: "Routine",
        smartContract: "SmartContract",
        standard: "Standard",
        closedBy: "User",
        comments: "Comment",
        labels: "Label",
        reports: "Report",
        reactions: "Reaction",
        bookmarkedBy: "User",
        viewedBy: "View",
    },
    joinMap: { labels: "label", bookmarkedBy: "user" },
    countFields: {
        commentsCount: true,
        labelsCount: true,
        reportsCount: true,
        translationsCount: true,
    },
};

export const LabelFormat: Formatter<LabelModelInfo> = {
    gqlRelMap: {
        __typename: "Label",
        apis: "Api",
        focusModes: "FocusMode",
        issues: "Issue",
        meetings: "Meeting",
        notes: "Note",
        owner: {
            ownedByUser: "User",
            ownedByOrganization: "Organization",
        },
        projects: "Project",
        routines: "Routine",
        schedules: "Schedule",
    },
    unionFields: {
        owner: {},
    },
    prismaRelMap: {
        __typename: "Label",
        apis: "Api",
        focusModes: "FocusMode",
        issues: "Issue",
        meetings: "Meeting",
        notes: "Note",
        ownedByUser: "User",
        ownedByOrganization: "Organization",
        projects: "Project",
        routines: "Routine",
        schedules: "Schedule",
    },
    joinMap: {
        apis: "labelled",
        focusModes: "labelled",
        issues: "labelled",
        meetings: "labelled",
        notes: "labelled",
        projects: "labelled",
        routines: "labelled",
        schedules: "labelled",
    },
    countFields: {
        apisCount: true,
        focusModesCount: true,
        issuesCount: true,
        meetingsCount: true,
        notesCount: true,
        projectsCount: true,
        smartContractsCount: true,
        standardsCount: true,
        routinesCount: true,
        schedulesCount: true,
        translationsCount: true,
    },
};

export const MeetingFormat: Formatter<MeetingModelInfo> = {
    gqlRelMap: {
        __typename: "Meeting",
        attendees: "User",
        invites: "MeetingInvite",
        labels: "Label",
        organization: "Organization",
        restrictedToRoles: "Role",
        schedule: "Schedule",
    },
    prismaRelMap: {
        __typename: "Meeting",
        organization: "Organization",
        restrictedToRoles: "Role",
        attendees: "User",
        invites: "MeetingInvite",
        labels: "Label",
        schedule: "Schedule",
    },
    joinMap: {
        labels: "label",
        restrictedToRoles: "role",
        attendees: "user",
    },
    countFields: {
        attendeesCount: true,
        invitesCount: true,
        labelsCount: true,
        translationsCount: true,
    },
};

export const MeetingInviteFormat: Formatter<MeetingInviteModelInfo> = {
    gqlRelMap: {
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
    gqlRelMap: {
        __typename: "Member",
        organization: "Organization",
        roles: "Role",
        user: "User",
    },
    prismaRelMap: {
        __typename: "Member",
        organization: "Organization",
        roles: "Role",
        user: "User",
    },
    countFields: {},
};

export const MemberInviteFormat: Formatter<MemberInviteModelInfo> = {
    gqlRelMap: {
        __typename: "MemberInvite",
        organization: "Organization",
        user: "User",
    },
    prismaRelMap: {
        __typename: "MemberInvite",
        organization: "Organization",
        user: "User",
    },
    countFields: {},
};

export const NodeFormat: Formatter<NodeModelInfo> = {
    gqlRelMap: {
        __typename: "Node",
        end: "NodeEnd",
        loop: "NodeLoop",
        routineList: "NodeRoutineList",
        routineVersion: "RoutineVersion",
    },
    prismaRelMap: {
        __typename: "Node",
        end: "NodeEnd",
        loop: "NodeLoop",
        next: "NodeLink",
        previous: "NodeLink",
        routineList: "NodeRoutineList",
        routineVersion: "RoutineVersion",
        runSteps: "RunRoutineStep",
    },
    countFields: {},
};

export const NodeEndFormat: Formatter<NodeEndModelInfo> = {
    gqlRelMap: {
        __typename: "NodeEnd",
        node: "Node",
        suggestedNextRoutineVersions: "RoutineVersion",
    },
    prismaRelMap: {
        __typename: "NodeEnd",
        node: "Node",
        suggestedNextRoutineVersions: "RoutineVersion",
    },
    joinMap: { suggestedNextRoutineVersions: "toRoutineVersion" },
    countFields: {},
};

export const NodeLinkFormat: Formatter<NodeLinkModelInfo> = {
    gqlRelMap: {
        __typename: "NodeLink",
        from: "Node",
        to: "Node",
        routineVersion: "RoutineVersion",
        whens: "NodeLinkWhen",
    },
    prismaRelMap: {
        __typename: "NodeLink",
        from: "Node",
        to: "Node",
        routineVersion: "RoutineVersion",
        whens: "NodeLinkWhen",
    },
    countFields: {},
};

export const NodeLinkWhenFormat: Formatter<NodeLinkWhenModelInfo> = {
    gqlRelMap: {
        __typename: "NodeLinkWhen",
    },
    prismaRelMap: {
        __typename: "NodeLinkWhen",
        link: "NodeLink",
    },
    countFields: {},
};

export const NodeLoopFormat: Formatter<NodeLoopModelInfo> = {
    gqlRelMap: {
        __typename: "NodeLoop",
        whiles: "NodeLoopWhile",
    },
    prismaRelMap: {
        __typename: "NodeLoop",
        node: "Node",
        whiles: "NodeLoopWhile",
    },
    countFields: {},
};

export const NodeLoopWhileFormat: Formatter<NodeLoopWhileModelInfo> = {
    gqlRelMap: {
        __typename: "NodeLoopWhile",
    },
    prismaRelMap: {
        __typename: "NodeLoopWhile",
        loop: "NodeLoop",
    },
    countFields: {},
};

export const NodeRoutineListFormat: Formatter<NodeRoutineListModelInfo> = {
    gqlRelMap: {
        __typename: "NodeRoutineList",
        items: "NodeRoutineListItem",
        node: "Node",
    },
    prismaRelMap: {
        __typename: "NodeRoutineList",
        items: "NodeRoutineListItem",
        node: "Node",
    },
    countFields: {},
};

export const NodeRoutineListItemFormat: Formatter<NodeRoutineListItemModelInfo> = {
    gqlRelMap: {
        __typename: "NodeRoutineListItem",
        list: "NodeRoutineList",
        routineVersion: "RoutineVersion",
    },
    prismaRelMap: {
        __typename: "NodeRoutineListItem",
        list: "NodeRoutineList",
        routineVersion: "RoutineVersion",
    },
    countFields: {},
};

export const NoteFormat: Formatter<NoteModelInfo> = {
    gqlRelMap: {
        __typename: "Note",
        createdBy: "User",
        issues: "Issue",
        labels: "Label",
        owner: {
            ownedByUser: "User",
            ownedByOrganization: "Organization",
        },
        parent: "Note",
        pullRequests: "PullRequest",
        questions: "Question",
        bookmarkedBy: "User",
        tags: "Tag",
        transfers: "Transfer",
        versions: "NoteVersion",
    },
    unionFields: {
        owner: {},
    },
    prismaRelMap: {
        __typename: "Note",
        parent: "NoteVersion",
        createdBy: "User",
        ownedByUser: "User",
        ownedByOrganization: "Organization",
        versions: "NoteVersion",
        pullRequests: "PullRequest",
        labels: "Label",
        issues: "Issue",
        tags: "Tag",
        bookmarkedBy: "User",
        questions: "Question",
    },
    joinMap: { labels: "label", bookmarkedBy: "user", tags: "tag" },
    countFields: {
        issuesCount: true,
        labelsCount: true,
        pullRequestsCount: true,
        questionsCount: true,
        transfersCount: true,
        versionsCount: true,
    },
};

export const NoteVersionFormat: Formatter<NoteVersionModelInfo> = {
    gqlRelMap: {
        __typename: "NoteVersion",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        pullRequest: "PullRequest",
        forks: "NoteVersion",
        reports: "Report",
        root: "Note",
    },
    prismaRelMap: {
        __typename: "NoteVersion",
        root: "Note",
        forks: "Note",
        pullRequest: "PullRequest",
        comments: "Comment",
        reports: "Report",
        directoryListings: "ProjectVersionDirectory",
    },
    countFields: {
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        reportsCount: true,
    },
};

export const NotificationFormat: Formatter<NotificationModelInfo> = {
    gqlRelMap: {
        __typename: "Notification",
    },
    prismaRelMap: {
        __typename: "Notification",
    },
    countFields: {},
};

export const NotificationSubscriptionFormat: Formatter<NotificationSubscriptionModelInfo> = {
    gqlRelMap: {
        __typename: "NotificationSubscription",
        object: {
            api: "Api",
            comment: "Comment",
            issue: "Issue",
            meeting: "Meeting",
            note: "Note",
            organization: "Organization",
            project: "Project",
            pullRequest: "PullRequest",
            question: "Question",
            quiz: "Quiz",
            report: "Report",
            routine: "Routine",
            schedule: "Schedule",
            smartContract: "SmartContract",
            standard: "Standard",
        },
    },
    unionFields: {
        object: { connectField: "objectConnect", typeField: "objectType" },
    },
    prismaRelMap: {
        __typename: "NotificationSubscription",
        api: "Api",
        comment: "Comment",
        issue: "Issue",
        meeting: "Meeting",
        note: "Note",
        organization: "Organization",
        project: "Project",
        pullRequest: "PullRequest",
        question: "Question",
        quiz: "Quiz",
        report: "Report",
        routine: "Routine",
        schedule: "Schedule",
        smartContract: "SmartContract",
        standard: "Standard",
        subscriber: "User",
    },
    countFields: {},
};

export const OrganizationFormat: Formatter<OrganizationModelInfo> = {
    gqlRelMap: {
        __typename: "Organization",
        apis: "Api",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        forks: "Organization",
        issues: "Issue",
        labels: "Label",
        meetings: "Meeting",
        members: "Member",
        notes: "Note",
        parent: "Organization",
        paymentHistory: "Payment",
        posts: "Post",
        premium: "Premium",
        projects: "Project",
        questions: "Question",
        reports: "Report",
        resourceList: "ResourceList",
        roles: "Role",
        routines: "Routine",
        smartContracts: "SmartContract",
        standards: "Standard",
        bookmarkedBy: "User",
        tags: "Tag",
        transfersIncoming: "Transfer",
        transfersOutgoing: "Transfer",
        wallets: "Wallet",
    },
    prismaRelMap: {
        __typename: "Organization",
        createdBy: "User",
        directoryListings: "ProjectVersionDirectory",
        issues: "Issue",
        labels: "Label",
        notes: "Note",
        apis: "Api",
        comments: "Comment",
        forks: "Organization",
        meetings: "Meeting",
        parent: "Organization",
        posts: "Post",
        smartContracts: "SmartContract",
        tags: "Tag",
        members: "Member",
        memberInvites: "MemberInvite",
        projects: "Project",
        questions: "Question",
        reports: "Report",
        resourceList: "ResourceList",
        roles: "Role",
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
        commentsCount: true,
        issuesCount: true,
        labelsCount: true,
        meetingsCount: true,
        membersCount: true,
        notesCount: true,
        postsCount: true,
        projectsCount: true,
        questionsCount: true,
        reportsCount: true,
        rolesCount: true,
        routinesCount: true,
        smartContractsCount: true,
        standardsCount: true,
        translationsCount: true,
    },
};

export const PaymentFormat: Formatter<PaymentModelInfo> = {
    gqlRelMap: {
        __typename: "Payment",
        organization: "Organization",
        user: "User",
    },
    prismaRelMap: {
        __typename: "Payment",
        organization: "Organization",
        user: "User",
    },
    countFields: {},
};

export const PhoneFormat: Formatter<PhoneModelInfo> = {
    gqlRelMap: {
        __typename: "Phone",
    },
    prismaRelMap: {
        __typename: "Phone",
        user: "User",
    },
    countFields: {},
};

export const PostFormat: Formatter<PostModelInfo> = {
    gqlRelMap: {
        __typename: "Post",
        comments: "Comment",
        owner: {
            user: "User",
            organization: "Organization",
        },
        reports: "Report",
        repostedFrom: "Post",
        reposts: "Post",
        resourceList: "ResourceList",
        bookmarkedBy: "User",
        tags: "Tag",
    },
    unionFields: {
        owner: {},
    },
    prismaRelMap: {
        __typename: "Post",
        organization: "Organization",
        user: "User",
        repostedFrom: "Post",
        reposts: "Post",
        resourceList: "ResourceList",
        comments: "Comment",
        bookmarkedBy: "User",
        reactions: "Reaction",
        viewedBy: "View",
        reports: "Report",
        tags: "Tag",
    },
    countFields: {
        commentsCount: true,
        repostsCount: true,
    },
};

export const PremiumFormat: Formatter<PremiumModelInfo> = {
    gqlRelMap: {
        __typename: "Premium",
    },
    prismaRelMap: {
        __typename: "Premium",
        organization: "Organization",
        user: "User",
    },
    countFields: {},
};

export const ProjectFormat: Formatter<ProjectModelInfo> = {
    gqlRelMap: {
        __typename: "Project",
        createdBy: "User",
        issues: "Issue",
        labels: "Label",
        owner: {
            ownedByUser: "User",
            ownedByOrganization: "Organization",
        },
        parent: "Project",
        pullRequests: "PullRequest",
        questions: "Question",
        quizzes: "Quiz",
        bookmarkedBy: "User",
        tags: "Tag",
        transfers: "Transfer",
        versions: "ProjectVersion",
    },
    unionFields: {
        owner: {},
    },
    prismaRelMap: {
        __typename: "Project",
        createdBy: "User",
        ownedByOrganization: "Organization",
        ownedByUser: "User",
        parent: "ProjectVersion",
        issues: "Issue",
        labels: "Label",
        tags: "Tag",
        versions: "ProjectVersion",
        bookmarkedBy: "User",
        pullRequests: "PullRequest",
        stats: "StatsProject",
        questions: "Question",
        transfers: "Transfer",
        quizzes: "Quiz",
    },
    joinMap: { labels: "label", bookmarkedBy: "user", tags: "tag" },
    countFields: {
        issuesCount: true,
        labelsCount: true,
        pullRequestsCount: true,
        questionsCount: true,
        quizzesCount: true,
        transfersCount: true,
        versionsCount: true,
    },
};

export const ProjectVersionFormat: Formatter<ProjectVersionModelInfo> = {
    gqlRelMap: {
        __typename: "ProjectVersion",
        comments: "Comment",
        directories: "ProjectVersionDirectory",
        directoryListings: "ProjectVersionDirectory",
        forks: "Project",
        pullRequest: "PullRequest",
        reports: "Report",
        root: "Project",
        // 'runs.project': 'RunProject', //TODO
    },
    prismaRelMap: {
        __typename: "ProjectVersion",
        comments: "Comment",
        directories: "ProjectVersionDirectory",
        directoryListings: "ProjectVersionDirectory",
        pullRequest: "PullRequest",
        reports: "Report",
        resourceList: "ResourceList",
        root: "Project",
        forks: "Project",
        runProjects: "RunProject",
        suggestedNextByProject: "ProjectVersion",
    },
    joinMap: {
        suggestedNextByProject: "toProjectVersion",
    },
    countFields: {
        commentsCount: true,
        directoriesCount: true,
        directoryListingsCount: true,
        forksCount: true,
        reportsCount: true,
        runProjectsCount: true,
        translationsCount: true,
    },
};

export const ProjectVersionDirectoryFormat: Formatter<ProjectVersionDirectoryModelInfo> = {
    gqlRelMap: {
        __typename: "ProjectVersionDirectory",
        parentDirectory: "ProjectVersionDirectory",
        projectVersion: "ProjectVersion",
        children: "ProjectVersionDirectory",
        childApiVersions: "ApiVersion",
        childNoteVersions: "NoteVersion",
        childOrganizations: "Organization",
        childProjectVersions: "ProjectVersion",
        childRoutineVersions: "RoutineVersion",
        childSmartContractVersions: "SmartContractVersion",
        childStandardVersions: "StandardVersion",
        runProjectSteps: "RunProjectStep",
    },
    prismaRelMap: {
        __typename: "ProjectVersionDirectory",
        parentDirectory: "ProjectVersionDirectory",
        projectVersion: "ProjectVersion",
        children: "ProjectVersionDirectory",
        childApiVersions: "ApiVersion",
        childNoteVersions: "NoteVersion",
        childOrganizations: "Organization",
        childProjectVersions: "ProjectVersion",
        childRoutineVersions: "RoutineVersion",
        childSmartContractVersions: "SmartContractVersion",
        childStandardVersions: "StandardVersion",
        runProjectSteps: "RunProjectStep",
    },
    countFields: {},
};

export const PullRequestFormat: Formatter<PullRequestModelInfo> = {
    gqlRelMap: {
        __typename: "PullRequest",
        createdBy: "User",
        comments: "Comment",
        from: {
            fromApiVersion: "ApiVersion",
            fromNoteVersion: "NoteVersion",
            fromProjectVersion: "ProjectVersion",
            fromRoutineVersion: "RoutineVersion",
            fromSmartContractVersion: "SmartContractVersion",
            fromStandardVersion: "StandardVersion",
        },
        to: {
            toApi: "Api",
            toNote: "Note",
            toProject: "Project",
            toRoutine: "Routine",
            toSmartContract: "SmartContract",
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
        fromNoteVersion: "NoteVersion",
        fromProjectVersion: "ProjectVersion",
        fromRoutineVersion: "RoutineVersion",
        fromSmartContractVersion: "SmartContractVersion",
        fromStandardVersion: "StandardVersion",
        toApi: "Api",
        toNote: "Note",
        toProject: "Project",
        toRoutine: "Routine",
        toSmartContract: "SmartContract",
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
    gqlRelMap: {
        __typename: "PushDevice",
    },
    prismaRelMap: {
        __typename: "PushDevice",
        user: "User",
    },
    countFields: {},
};

export const QuestionFormat: Formatter<QuestionModelInfo> = {
    gqlRelMap: {
        __typename: "Question",
        createdBy: "User",
        answers: "QuestionAnswer",
        comments: "Comment",
        forObject: {
            api: "Api",
            note: "Note",
            organization: "Organization",
            project: "Project",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
        },
        reports: "Report",
        bookmarkedBy: "User",
        tags: "Tag",
    },
    unionFields: {
        forObject: { connectField: "forObjectConnect", typeField: "forObjectType" },
    },
    prismaRelMap: {
        __typename: "Question",
        createdBy: "User",
        api: "Api",
        note: "Note",
        organization: "Organization",
        project: "Project",
        routine: "Routine",
        smartContract: "SmartContract",
        standard: "Standard",
        comments: "Comment",
        answers: "QuestionAnswer",
        reports: "Report",
        tags: "Tag",
        bookmarkedBy: "User",
        reactions: "Reaction",
        viewedBy: "User",
    },
    joinMap: { bookmarkedBy: "user", tags: "tag" },
    countFields: {
        answersCount: true,
        commentsCount: true,
        reportsCount: true,
        translationsCount: true,
    },
};

export const QuestionAnswerFormat: Formatter<QuestionAnswerModelInfo> = {
    gqlRelMap: {
        __typename: "QuestionAnswer",
        bookmarkedBy: "User",
        createdBy: "User",
        comments: "Comment",
        question: "Question",
    },
    prismaRelMap: {
        __typename: "QuestionAnswer",
        bookmarkedBy: "User",
        createdBy: "User",
        comments: "Comment",
        question: "Question",
        reactions: "Reaction",
    },
    countFields: {
        commentsCount: true,
    },
    joinMap: { bookmarkedBy: "user" },
};

export const QuizFormat: Formatter<QuizModelInfo> = {
    gqlRelMap: {
        __typename: "Quiz",
        attempts: "QuizAttempt",
        createdBy: "User",
        project: "Project",
        quizQuestions: "QuizQuestion",
        routine: "Routine",
        bookmarkedBy: "User",
    },
    prismaRelMap: {
        __typename: "Quiz",
        attempts: "QuizAttempt",
        createdBy: "User",
        project: "Project",
        quizQuestions: "QuizQuestion",
        routine: "Routine",
        bookmarkedBy: "User",
    },
    joinMap: { bookmarkedBy: "user" },
    countFields: {
        attemptsCount: true,
        quizQuestionsCount: true,
    },
};

export const QuizAttemptFormat: Formatter<QuizAttemptModelInfo> = {
    gqlRelMap: {
        __typename: "QuizAttempt",
        quiz: "Quiz",
        responses: "QuizQuestionResponse",
        user: "User",
    },
    prismaRelMap: {
        __typename: "QuizAttempt",
        quiz: "Quiz",
        responses: "QuizQuestionResponse",
        user: "User",
    },
    countFields: {
        responsesCount: true,
    },
};

export const QuizQuestionFormat: Formatter<QuizQuestionModelInfo> = {
    gqlRelMap: {
        __typename: "QuizQuestion",
        quiz: "Quiz",
        responses: "QuizQuestionResponse",
        standardVersion: "StandardVersion",
    },
    prismaRelMap: {
        __typename: "QuizQuestion",
        quiz: "Quiz",
        responses: "QuizQuestionResponse",
        standardVersion: "StandardVersion",
    },
    countFields: {
        responsesCount: true,
    },
};

export const QuizQuestionResponseFormat: Formatter<QuizQuestionResponseModelInfo> = {
    gqlRelMap: {
        __typename: "QuizQuestionResponse",
        quizAttempt: "QuizAttempt",
        quizQuestion: "QuizQuestion",
    },
    prismaRelMap: {
        __typename: "QuizQuestionResponse",
        quizAttempt: "QuizAttempt",
        quizQuestion: "QuizQuestion",
    },
    countFields: {},
};

export const ReactionFormat: Formatter<ReactionModelInfo> = {
    gqlRelMap: {
        __typename: "Reaction",
        by: "User",
        to: {
            api: "Api",
            chatMessage: "ChatMessage",
            comment: "Comment",
            issue: "Issue",
            note: "Note",
            post: "Post",
            project: "Project",
            question: "Question",
            questionAnswer: "QuestionAnswer",
            quiz: "Quiz",
            routine: "Routine",
            smartContract: "SmartContract",
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
        comment: "Comment",
        issue: "Issue",
        note: "Note",
        post: "Post",
        project: "Project",
        question: "Question",
        questionAnswer: "QuestionAnswer",
        quiz: "Quiz",
        routine: "Routine",
        smartContract: "SmartContract",
        standard: "Standard",
    },
    countFields: {},
};

export const ReactionSummaryFormat: Formatter<ReactionSummaryModelInfo> = {
    gqlRelMap: {
        __typename: "ReactionSummary",
    },
    prismaRelMap: {
        __typename: "ReactionSummary",
    },
    countFields: {},
};

export const ReminderFormat: Formatter<ReminderModelInfo> = {
    gqlRelMap: {
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
    gqlRelMap: {
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
    gqlRelMap: {
        __typename: "ReminderList",
        focusMode: "FocusMode",
        reminders: "Reminder",
    },
    prismaRelMap: {
        __typename: "ReminderList",
        focusMode: "FocusMode",
        reminders: "Reminder",
    },
    countFields: {},
};

export const ReportFormat: Formatter<ReportModelInfo> = {
    gqlRelMap: {
        __typename: "Report",
        responses: "ReportResponse",
    },
    prismaRelMap: {
        __typename: "Report",
        responses: "ReportResponse",
    },
    hiddenFields: ["createdById"], // Always hide report creator,
    countFields: {
        responsesCount: true,
    },
};

export const ReportResponseFormat: Formatter<ReportResponseModelInfo> = {
    gqlRelMap: {
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
    gqlRelMap: {
        __typename: "Resource",
        list: "ResourceList",
    },
    prismaRelMap: {
        __typename: "Resource",
        list: "ResourceList",
    },
    countFields: {},
};

export const ResourceListFormat: Formatter<ResourceListModelInfo> = {
    gqlRelMap: {
        __typename: "ResourceList",
        resources: "Resource",
        listFor: {
            apiVersion: "ApiVersion",
            focusMode: "FocusMode",
            organization: "Organization",
            post: "Post",
            projectVersion: "ProjectVersion",
            routineVersion: "RoutineVersion",
            smartContractVersion: "SmartContractVersion",
            standardVersion: "StandardVersion",
        },
    },
    unionFields: {
        listFor: { connectField: "listForConnect", typeField: "listForType" },
    },
    prismaRelMap: {
        __typename: "ResourceList",
        resources: "Resource",
        apiVersion: "ApiVersion",
        organization: "Organization",
        post: "Post",
        projectVersion: "ProjectVersion",
        routineVersion: "RoutineVersion",
        smartContractVersion: "SmartContractVersion",
        standardVersion: "StandardVersion",
        focusMode: "FocusMode",
    },
    countFields: {},
};

export const RoleFormat: Formatter<RoleModelInfo> = {
    gqlRelMap: {
        __typename: "Role",
        members: "Member",
        organization: "Organization",
    },
    prismaRelMap: {
        __typename: "Role",
        members: "Member",
        meetings: "Meeting",
        organization: "Organization",
    },
    countFields: {
        membersCount: true,
    },
};

export const RoutineFormat: Formatter<RoutineModelInfo> = {
    gqlRelMap: {
        __typename: "Routine",
        createdBy: "User",
        owner: {
            ownedByUser: "User",
            ownedByOrganization: "Organization",
        },
        forks: "Routine",
        issues: "Issue",
        labels: "Label",
        parent: "Routine",
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
        ownedByUser: "User",
        ownedByOrganization: "Organization",
        parent: "RoutineVersion",
        questions: "Question",
        quizzes: "Quiz",
        issues: "Issue",
        labels: "Label",
        pullRequests: "PullRequest",
        bookmarkedBy: "User",
        stats: "StatsRoutine",
        tags: "Tag",
        transfers: "Transfer",
        versions: "RoutineVersion",
        viewedBy: "View",
    },
    joinMap: { labels: "label", tags: "tag", bookmarkedBy: "user" },
    countFields: {
        forksCount: true,
        issuesCount: true,
        pullRequestsCount: true,
        questionsCount: true,
        quizzesCount: true,
        transfersCount: true,
        versionsCount: true,
    },
};

export const RoutineVersionFormat: Formatter<RoutineVersionModelInfo> = {
    gqlRelMap: {
        __typename: "RoutineVersion",
        apiVersion: "ApiVersion",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        forks: "Routine",
        inputs: "RoutineVersionInput",
        nodes: "Node",
        nodeLinks: "NodeLink",
        outputs: "RoutineVersionOutput",
        pullRequest: "PullRequest",
        resourceList: "ResourceList",
        reports: "Report",
        root: "Routine",
        smartContractVersion: "SmartContractVersion",
        suggestedNextByRoutineVersion: "RoutineVersion",
        // you.runs: 'RunRoutine', //TODO
    },
    prismaRelMap: {
        __typename: "RoutineVersion",
        apiVersion: "ApiVersion",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        reports: "Report",
        smartContractVersion: "SmartContractVersion",
        nodes: "Node",
        nodeLinks: "NodeLink",
        resourceList: "ResourceList",
        root: "Routine",
        forks: "Routine",
        inputs: "RoutineVersionInput",
        outputs: "RoutineVersionOutput",
        pullRequest: "PullRequest",
        runRoutines: "RunRoutine",
        runSteps: "RunRoutineStep",
        suggestedNextByRoutineVersion: "RoutineVersion",
    },
    joinMap: {
        suggestedNextByRoutineVersion: "toRoutineVersion",
    },
    countFields: {
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        inputsCount: true,
        nodeLinksCount: true,
        nodesCount: true,
        outputsCount: true,
        reportsCount: true,
        suggestedNextByRoutineVersionCount: true,
        translationsCount: true,
    },
};

export const RoutineVersionInputFormat: Formatter<RoutineVersionInputModelInfo> = {
    gqlRelMap: {
        __typename: "RoutineVersionInput",
        routineVersion: "RoutineVersion",
        standardVersion: "StandardVersion",
    },
    prismaRelMap: {
        __typename: "RoutineVersionInput",
        routineVersion: "RoutineVersion",
        standardVersion: "StandardVersion",
        runInputs: "RunRoutineInput",
    },
    countFields: {},
};

export const RoutineVersionOutputFormat: Formatter<RoutineVersionOutputModelInfo> = {
    gqlRelMap: {
        __typename: "RoutineVersionOutput",
        routineVersion: "RoutineVersion",
        standardVersion: "StandardVersion",
    },
    prismaRelMap: {
        __typename: "RoutineVersionOutput",
        routineVersion: "RoutineVersion",
        standardVersion: "StandardVersion",
    },
    countFields: {},
};

export const RunProjectFormat: Formatter<RunProjectModelInfo> = {
    gqlRelMap: {
        __typename: "RunProject",
        projectVersion: "ProjectVersion",
        schedule: "Schedule",
        steps: "RunProjectStep",
        user: "User",
        organization: "Organization",
    },
    prismaRelMap: {
        __typename: "RunProject",
        projectVersion: "ProjectVersion",
        schedule: "Schedule",
        steps: "RunProjectStep",
        user: "User",
        organization: "Organization",
    },
    countFields: {
        stepsCount: true,
    },
};

export const RunProjectStepFormat: Formatter<RunProjectStepModelInfo> = {
    gqlRelMap: {
        __typename: "RunProjectStep",
        directory: "ProjectVersionDirectory",
        run: "RunProject",
    },
    prismaRelMap: {
        __typename: "RunProjectStep",
        directory: "ProjectVersionDirectory",
        runProject: "RunProject",
    },
    countFields: {},
};

export const RunRoutineFormat: Formatter<RunRoutineModelInfo> = {
    gqlRelMap: {
        __typename: "RunRoutine",
        inputs: "RunRoutineInput",
        organization: "Organization",
        routineVersion: "RoutineVersion",
        runProject: "RunProject",
        schedule: "Schedule",
        steps: "RunRoutineStep",
        user: "User",
    },
    prismaRelMap: {
        __typename: "RunRoutine",
        inputs: "RunRoutineInput",
        organization: "Organization",
        routineVersion: "RoutineVersion",
        runProject: "RunProject",
        schedule: "Schedule",
        steps: "RunRoutineStep",
        user: "User",
    },
    countFields: {
        inputsCount: true,
        stepsCount: true,
    },
};

export const RunRoutineInputFormat: Formatter<RunRoutineInputModelInfo> = {
    gqlRelMap: {
        __typename: "RunRoutineInput",
        input: "RoutineVersionInput",
        runRoutine: "RunRoutine",
    },
    prismaRelMap: {
        __typename: "RunRoutineInput",
        input: "RunRoutineInput",
        runRoutine: "RunRoutine",
    },
    countFields: {},
};

export const RunRoutineStepFormat: Formatter<RunRoutineStepModelInfo> = {
    gqlRelMap: {
        __typename: "RunRoutineStep",
        run: "RunRoutine",
        node: "Node",
        subroutine: "Routine",
    },
    prismaRelMap: {
        __typename: "RunRoutineStep",
        node: "Node",
        runRoutine: "RunRoutine",
        subroutine: "RoutineVersion",
    },
    countFields: {},
};

export const ScheduleFormat: Formatter<ScheduleModelInfo> = {
    gqlRelMap: {
        __typename: "Schedule",
        exceptions: "ScheduleException",
        labels: "Label",
        recurrences: "ScheduleRecurrence",
        runProjects: "RunProject",
        runRoutines: "RunRoutine",
        focusModes: "FocusMode",
        meetings: "Meeting",
    },
    prismaRelMap: {
        __typename: "Schedule",
        exceptions: "ScheduleException",
        labels: "Label",
        recurrences: "ScheduleRecurrence",
        runProjects: "RunProject",
        runRoutines: "RunRoutine",
        focusModes: "FocusMode",
        meetings: "Meeting",
    },
    joinMap: { labels: "label" },
    countFields: {},
};

export const ScheduleExceptionFormat: Formatter<ScheduleExceptionModelInfo> = {
    gqlRelMap: {
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
    gqlRelMap: {
        __typename: "ScheduleRecurrence",
        schedule: "Schedule",
    },
    prismaRelMap: {
        __typename: "ScheduleRecurrence",
        schedule: "Schedule",
    },
    countFields: {},
};

export const SmartContractFormat: Formatter<SmartContractModelInfo> = {
    gqlRelMap: {
        __typename: "SmartContract",
        createdBy: "User",
        issues: "Issue",
        labels: "Label",
        owner: {
            ownedByUser: "User",
            ownedByOrganization: "Organization",
        },
        parent: "SmartContract",
        pullRequests: "PullRequest",
        questions: "Question",
        bookmarkedBy: "User",
        tags: "Tag",
        transfers: "Transfer",
        versions: "NoteVersion",
    },
    unionFields: {
        owner: {},
    },
    prismaRelMap: {
        __typename: "SmartContract",
        createdBy: "User",
        issues: "Issue",
        labels: "Label",
        ownedByUser: "User",
        ownedByOrganization: "Organization",
        parent: "NoteVersion",
        pullRequests: "PullRequest",
        questions: "Question",
        bookmarkedBy: "User",
        tags: "Tag",
        transfers: "Transfer",
        versions: "NoteVersion",
    },
    joinMap: { labels: "label", bookmarkedBy: "user", tags: "tag" },
    countFields: {
        issuesCount: true,
        pullRequestsCount: true,
        questionsCount: true,
        transfersCount: true,
        versionsCount: true,
    },
};

export const SmartContractVersionFormat: Formatter<SmartContractVersionModelInfo> = {
    gqlRelMap: {
        __typename: "SmartContractVersion",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        forks: "SmartContractVersion",
        pullRequest: "PullRequest",
        reports: "Report",
        root: "SmartContract",
    },
    prismaRelMap: {
        __typename: "SmartContractVersion",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        forks: "SmartContractVersion",
        pullRequest: "PullRequest",
        reports: "Report",
        root: "SmartContract",
    },
    countFields: {
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        reportsCount: true,
        translationsCount: true,
    },
};

export const StandardFormat: Formatter<StandardModelInfo> = {
    gqlRelMap: {
        __typename: "Standard",
        createdBy: "User",
        issues: "Issue",
        labels: "Label",
        owner: {
            ownedByUser: "User",
            ownedByOrganization: "Organization",
        },
        parent: "Project",
        pullRequests: "PullRequest",
        questions: "Question",
        bookmarkedBy: "User",
        tags: "Tag",
        transfers: "Transfer",
        versions: "StandardVersion",
    },
    unionFields: {
        owner: {},
    },
    prismaRelMap: {
        __typename: "Standard",
        createdBy: "User",
        ownedByOrganization: "Organization",
        ownedByUser: "User",
        issues: "Issue",
        labels: "Label",
        parent: "StandardVersion",
        tags: "Tag",
        bookmarkedBy: "User",
        versions: "StandardVersion",
        pullRequests: "PullRequest",
        stats: "StatsStandard",
        questions: "Question",
        transfers: "Transfer",
    },
    joinMap: { labels: "label", tags: "tag", bookmarkedBy: "user" },
    countFields: {
        forksCount: true,
        issuesCount: true,
        pullRequestsCount: true,
        questionsCount: true,
        transfersCount: true,
        versionsCount: true,
    },
};

export const StandardVersionFormat: Formatter<StandardVersionModelInfo> = {
    gqlRelMap: {
        __typename: "StandardVersion",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        forks: "StandardVersion",
        pullRequest: "PullRequest",
        reports: "Report",
        resourceList: "ResourceList",
        root: "Standard",
    },
    prismaRelMap: {
        __typename: "StandardVersion",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        forks: "StandardVersion",
        pullRequest: "PullRequest",
        reports: "Report",
        resourceList: "ResourceList",
        root: "Standard",
    },
    countFields: {
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        reportsCount: true,
        translationsCount: true,
    },
};

export const StatsApiFormat: Formatter<StatsApiModelInfo> = {
    gqlRelMap: {
        __typename: "StatsApi",
    },
    prismaRelMap: {
        __typename: "StatsApi",
        api: "Api",
    },
    countFields: {},
};

export const StatsOrganizationFormat: Formatter<StatsOrganizationModelInfo> = {
    gqlRelMap: {
        __typename: "StatsOrganization",
    },
    prismaRelMap: {
        __typename: "StatsOrganization",
        organization: "Organization",
    },
    countFields: {},
};

export const StatsProjectFormat: Formatter<StatsProjectModelInfo> = {
    gqlRelMap: {
        __typename: "StatsProject",
    },
    prismaRelMap: {
        __typename: "StatsProject",
        project: "Api",
    },
    countFields: {},
};

export const StatsQuizFormat: Formatter<StatsQuizModelInfo> = {
    gqlRelMap: {
        __typename: "StatsQuiz",
    },
    prismaRelMap: {
        __typename: "StatsQuiz",
        quiz: "Quiz",
    },
    countFields: {},
};

export const StatsRoutineFormat: Formatter<StatsRoutineModelInfo> = {
    gqlRelMap: {
        __typename: "StatsRoutine",
    },
    prismaRelMap: {
        __typename: "StatsRoutine",
        routine: "Routine",
    },
    countFields: {},
};

export const StatsSiteFormat: Formatter<StatsSiteModelInfo> = {
    gqlRelMap: {
        __typename: "StatsSite",
    },
    prismaRelMap: {
        __typename: "StatsSite",
    },
    countFields: {},
};

export const StatsSmartContractFormat: Formatter<StatsSmartContractModelInfo> = {
    gqlRelMap: {
        __typename: "StatsSmartContract",
    },
    prismaRelMap: {
        __typename: "StatsSmartContract",
        smartContract: "SmartContract",
    },
    countFields: {},
};

export const StatsStandardFormat: Formatter<StatsStandardModelInfo> = {
    gqlRelMap: {
        __typename: "StatsStandard",
    },
    prismaRelMap: {
        __typename: "StatsStandard",
        standard: "Standard",
    },
    countFields: {},
};

export const StatsUserFormat: Formatter<StatsUserModelInfo> = {
    gqlRelMap: {
        __typename: "StatsUser",
    },
    prismaRelMap: {
        __typename: "StatsUser",
        user: "User",
    },
    countFields: {},
};

export const TagFormat: Formatter<TagModelInfo> = {
    gqlRelMap: {
        __typename: "Tag",
        apis: "Api",
        notes: "Note",
        organizations: "Organization",
        posts: "Post",
        projects: "Project",
        reports: "Report",
        routines: "Routine",
        smartContracts: "SmartContract",
        standards: "Standard",
        bookmarkedBy: "User",
    },
    prismaRelMap: {
        __typename: "Tag",
        createdBy: "User",
        apis: "Api",
        notes: "Note",
        organizations: "Organization",
        posts: "Post",
        projects: "Project",
        reports: "Report",
        routines: "Routine",
        smartContracts: "SmartContract",
        standards: "Standard",
        bookmarkedBy: "User",
        focusModeFilters: "FocusModeFilter",
    },
    joinMap: {
        apis: "tagged",
        notes: "tagged",
        organizations: "tagged",
        posts: "tagged",
        projects: "tagged",
        reports: "tagged",
        routines: "tagged",
        smartContracts: "tagged",
        standards: "tagged",
        bookmarkedBy: "user",
    },
    countFields: {},
};

export const TransferFormat: Formatter<TransferModelInfo> = {
    gqlRelMap: {
        __typename: "Transfer",
        fromOwner: {
            fromUser: "User",
            fromOrganization: "Organization",
        },
        object: {
            api: "Api",
            note: "Note",
            project: "Project",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
        },
        toOwner: {
            toUser: "User",
            toOrganization: "Organization",
        },
    },
    unionFields: {
        // fromOwner is yourself, so it's not included in the Create input - and thus not needed here (for now)
        object: { connectField: "objectConnect", typeField: "objectType" },
        toOwner: {},
    },
    prismaRelMap: {
        __typename: "Transfer",
        fromUser: "User",
        fromOrganization: "Organization",
        toUser: "User",
        toOrganization: "Organization",
        api: "Api",
        note: "Note",
        project: "Project",
        routine: "Routine",
        smartContract: "SmartContract",
        standard: "Standard",
    },
    countFields: {},
};

export const UserFormat: Formatter<UserModelInfo> = {
    gqlRelMap: {
        __typename: "User",
        comments: "Comment",
        emails: "Email",
        focusModes: "FocusMode",
        labels: "Label",
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
        comments: "Comment",
        emails: "Email",
        organizationsCreated: "Organization",
        phones: "Phone",
        posts: "Post",
        focusModes: "FocusMode",
        invitedByUser: "User",
        invitedUsers: "User",
        issuesCreated: "Issue",
        issuesClosed: "Issue",
        labels: "Label",
        meetingsAttending: "Meeting",
        meetingsInvited: "MeetingInvite",
        pushDevices: "PushDevice",
        notifications: "Notification",
        memberships: "Member",
        projectsCreated: "Project",
        projects: "Project",
        pullRequests: "PullRequest",
        questionAnswered: "QuestionAnswer",
        questionsAsked: "Question",
        quizzesCreated: "Quiz",
        quizzesTaken: "QuizAttempt",
        reportsCreated: "Report",
        reportsReceived: "Report",
        reportResponses: "ReportResponse",
        routinesCreated: "Routine",
        routines: "Routine",
        runProjects: "RunProject",
        runRoutines: "RunRoutine",
        smartContractsCreated: "SmartContract",
        smartContracts: "SmartContract",
        standardsCreated: "Standard",
        standards: "Standard",
        bookmarkedBy: "User",
        tags: "Tag",
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
        membershipsCount: true,
        notesCount: true,
        projectsCount: true,
        questionsAskedCount: true,
        reportsReceivedCount: true,
        routinesCount: true,
        smartContractsCount: true,
        standardsCount: true,
    },
};

export const ViewFormat: Formatter<ViewModelInfo> = {
    gqlRelMap: {
        __typename: "View",
        by: "User",
        to: {
            api: "Api",
            issue: "Issue",
            note: "Note",
            organization: "Organization",
            post: "Post",
            project: "Project",
            question: "Question",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
            user: "User",
        },
    },
    // View is never called directly, so we don't need to worry about the unionFields - at least for now
    prismaRelMap: {
        __typename: "View",
        by: "User",
        api: "Api",
        issue: "Issue",
        note: "Note",
        organization: "Organization",
        post: "Post",
        project: "Project",
        question: "Question",
        routine: "Routine",
        smartContract: "SmartContract",
        standard: "Standard",
        user: "User",
    },
    countFields: {},
};

export const WalletFormat: Formatter<WalletModelInfo> = {
    gqlRelMap: {
        __typename: "Wallet",
        user: "User",
        organization: "Organization",
    },
    prismaRelMap: {
        __typename: "Wallet",
        user: "User",
        organization: "Organization",
    },
    countFields: {},
};

/** Maps model types to their respective formatter logic */
export const FormatMap: { [key in GqlModelType]?: Formatter<any> } = {
    Api: ApiFormat,
    ApiKey: ApiKeyFormat,
    ApiVersion: ApiVersionFormat,
    Award: AwardFormat,
    Bookmark: BookmarkFormat,
    BookmarkList: BookmarkListFormat,
    Chat: ChatFormat,
    ChatInvite: ChatInviteFormat,
    ChatMessage: ChatMessageFormat,
    ChatParticipant: ChatParticipantFormat,
    Comment: CommentFormat,
    Email: EmailFormat,
    FocusMode: FocusModeFormat,
    FocusModeFilter: FocusModeFilterFormat,
    Issue: IssueFormat,
    Label: LabelFormat,
    Meeting: MeetingFormat,
    MeetingInvite: MeetingInviteFormat,
    Member: MemberFormat,
    MemberInvite: MemberInviteFormat,
    Node: NodeFormat,
    NodeEnd: NodeEndFormat,
    NodeLink: NodeLinkFormat,
    NodeLinkWhen: NodeLinkWhenFormat,
    NodeLoop: NodeLoopFormat,
    NodeLoopWhile: NodeLoopWhileFormat,
    NodeRoutineList: NodeRoutineListFormat,
    NodeRoutineListItem: NodeRoutineListItemFormat,
    Note: NoteFormat,
    NoteVersion: NoteVersionFormat,
    Notification: NotificationFormat,
    NotificationSubscription: NotificationSubscriptionFormat,
    Organization: OrganizationFormat,
    Payment: PaymentFormat,
    Phone: PhoneFormat,
    Post: PostFormat,
    Premium: PremiumFormat,
    Project: ProjectFormat,
    ProjectVersion: ProjectVersionFormat,
    ProjectVersionDirectory: ProjectVersionDirectoryFormat,
    PullRequest: PullRequestFormat,
    PushDevice: PushDeviceFormat,
    Question: QuestionFormat,
    QuestionAnswer: QuestionAnswerFormat,
    Quiz: QuizFormat,
    QuizAttempt: QuizAttemptFormat,
    QuizQuestion: QuizQuestionFormat,
    QuizQuestionResponse: QuizQuestionResponseFormat,
    Reaction: ReactionFormat,
    ReactionSummary: ReactionSummaryFormat,
    Reminder: ReminderFormat,
    ReminderItem: ReminderItemFormat,
    ReminderList: ReminderListFormat,
    Report: ReportFormat,
    ReportResponse: ReportResponseFormat,
    Resource: ResourceFormat,
    ResourceList: ResourceListFormat,
    Role: RoleFormat,
    Routine: RoutineFormat,
    RoutineVersion: RoutineVersionFormat,
    RoutineVersionInput: RoutineVersionInputFormat,
    RoutineVersionOutput: RoutineVersionOutputFormat,
    RunProject: RunProjectFormat,
    RunProjectStep: RunProjectStepFormat,
    RunRoutine: RunRoutineFormat,
    RunRoutineInput: RunRoutineInputFormat,
    RunRoutineStep: RunRoutineStepFormat,
    Schedule: ScheduleFormat,
    ScheduleException: ScheduleExceptionFormat,
    ScheduleRecurrence: ScheduleRecurrenceFormat,
    SmartContract: SmartContractFormat,
    SmartContractVersion: SmartContractVersionFormat,
    Standard: StandardFormat,
    StandardVersion: StandardVersionFormat,
    StatsApi: StatsApiFormat,
    StatsOrganization: StatsOrganizationFormat,
    StatsProject: StatsProjectFormat,
    StatsQuiz: StatsQuizFormat,
    StatsRoutine: StatsRoutineFormat,
    StatsSite: StatsSiteFormat,
    StatsSmartContract: StatsSmartContractFormat,
    StatsStandard: StatsStandardFormat,
    StatsUser: StatsUserFormat,
    Tag: TagFormat,
    Transfer: TransferFormat,
    User: UserFormat,
    View: ViewFormat,
    Wallet: WalletFormat,
};