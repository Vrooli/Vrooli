import { LINKS } from "../consts/ui.js";

function findOne(one: string) {
    return {
        findOne: {
            endpoint: `/${one}/:publicId`,
            method: "GET" as const,
        },
    };
}

function findMany(many: string) {
    return {
        findMany: {
            endpoint: `/${many}`,
            method: "GET" as const,
        },
    };
}

function createOne(one: string) {
    return {
        createOne: {
            endpoint: `/${one}`,
            method: "POST" as const,
        },
    };
}

function createMany(many: string) {
    return {
        createMany: {
            endpoint: `/${many}`,
            method: "POST" as const,
        },
    };
}

function updateOne(one: string) {
    return {
        updateOne: {
            endpoint: `/${one}/:id`,
            method: "PUT" as const,
        },
    };
}

function updateMany(many: string) {
    return {
        updateMany: {
            endpoint: `/${many}`,
            method: "PUT" as const,
        },
    };
}

function standardCRUD(one: string, many: string) {
    return {
        ...findOne(one),
        ...findMany(many),
        ...createOne(one),
        ...updateOne(one),
    };
}

export const endpointsApiKey = {
    ...createOne("apiKey"),
    ...updateOne("apiKey"),
    validateOne: {
        endpoint: "/apiKey/:id/validate",
        method: "PUT" as const,
    },
} as const;

export const endpointsApiKeyExternal = {
    ...createOne("apiKeyExternal"),
    ...updateOne("apiKeyExternal"),
} as const;

export const endpointsAuth = {
    emailLogin: {
        endpoint: "/auth/email/login",
        method: "POST" as const,
    },
    emailSignup: {
        endpoint: "/auth/email/signup",
        method: "POST" as const,
    },
    emailRequestPasswordChange: {
        endpoint: "/auth/email/requestPasswordChange",
        method: "POST" as const,
    },
    emailResetPassword: {
        endpoint: "/auth/email/resetPassword",
        method: "POST" as const,
    },
    guestLogin: {
        endpoint: "/auth/guest/login",
        method: "POST" as const,
    },
    logout: {
        endpoint: "/auth/logout",
        method: "POST" as const,
    },
    logoutAll: {
        endpoint: "/auth/logoutAll",
        method: "POST" as const,
    },
    validateSession: {
        endpoint: "/auth/validateSession",
        method: "POST" as const,
    },
    switchCurrentAccount: {
        endpoint: "/auth/switchCurrentAccount",
        method: "POST" as const,
    },
    walletInit: {
        endpoint: "/auth/wallet/init",
        method: "POST" as const,
    },
    walletComplete: {
        endpoint: "/auth/wallet/complete",
        method: "POST" as const,
    },
};

export const endpointsAward = {
    ...findMany("awards"),
} as const;

export const endpointsBookmark = standardCRUD("bookmark", "bookmarks");

export const endpointsBookmarkList = standardCRUD(LINKS.BookmarkList, "bookmarkLists");

export const endpointsChat = standardCRUD(LINKS.Chat, "chats");

export const endpointsChatInvite = {
    ...standardCRUD("chatInvite", "chatInvites"),
    ...createMany("chatInvites"),
    ...updateMany("chatInvites"),
    acceptOne: {
        endpoint: "/chatInvite/:id/accept",
        method: "PUT" as const,
    },
    declineOne: {
        endpoint: "/chatInvite/:id/decline",
        method: "PUT" as const,
    },
} as const;

export const endpointsChatMessage = {
    ...standardCRUD("chatMessage", "chatMessages"),
    findTree: {
        endpoint: "/chatMessageTree",
        method: "GET" as const,
    },
    regenerateResponse: {
        endpoint: "/regenerateResponse",
        method: "POST" as const,
    },
} as const;

export const endpointsChatParticipant = {
    ...findOne("chatParticipant"),
    ...findMany("chatParticipants"),
    ...updateOne("chatParticipant"),
} as const;

export const endpointsComment = standardCRUD("comment", "comments");

export const endpointsActions = {
    copy: {
        endpoint: "/copy",
        method: "POST" as const,
    },
    deleteOne: {
        endpoint: "/deleteOne",
        method: "POST" as const,
    },
    deleteMany: {
        endpoint: "/deleteMany",
        method: "POST" as const,
    },
    deleteAll: {
        endpoint: "/deleteAll",
        method: "POST" as const,
    },
    deleteAccount: {
        endpoint: "/deleteAccount",
        method: "POST" as const,
    },
} as const;

export const endpointsEmail = {
    ...createOne("email"),
    verify: {
        endpoint: "/email/verify",
        method: "POST" as const,
    },
} as const;

export const endpointsFeed = {
    home: {
        endpoint: "/feed/home",
        method: "GET" as const,
    },
    popular: {
        endpoint: "/feed/popular",
        method: "GET" as const,
    },
} as const;

export const endpointsIssue = {
    ...standardCRUD(LINKS.Issue, "issues"),
    closeOne: {
        endpoint: "/issue/:id/close",
        method: "PUT" as const,
    },
} as const;

export const endpointsMeeting = standardCRUD(LINKS.Meeting, "meetings");

export const endpointsMeetingInvite = {
    ...standardCRUD("meetingInvite", "meetingInvites"),
    ...createMany("meetingInvites"),
    ...updateMany("meetingInvites"),
    acceptOne: {
        endpoint: "/meetingInvite/:id/accept",
        method: "PUT" as const,
    },
    declineOne: {
        endpoint: "/meetingInvite/:id/decline",
        method: "PUT" as const,
    },
} as const;

export const endpointsMember = {
    ...findOne("member"),
    ...findMany("members"),
    ...updateOne("member"),
} as const;

export const endpointsMemberInvite = {
    ...standardCRUD("memberInvite", "memberInvites"),
    ...createMany("memberInvites"),
    ...updateMany("memberInvites"),
    acceptOne: {
        endpoint: "/memberInvite/:id/accept",
        method: "PUT" as const,
    },
    declineOne: {
        endpoint: "/memberInvite/:id/decline",
        method: "PUT" as const,
    },
} as const;

export const endpointsNotification = {
    ...findOne("notification"),
    ...findMany("notifications"),
    markAsRead: {
        endpoint: "/notification/:id/markAsRead",
        method: "PUT" as const,
    },
    markAllAsRead: {
        endpoint: "/notifications/markAllAsRead",
        method: "PUT" as const,
    },
    getSettings: {
        endpoint: "/notificationSettings",
        method: "GET" as const,
    },
    updateSettings: {
        endpoint: "/notificationSettings",
        method: "PUT" as const,
    },
} as const;

export const endpointsNotificationSubscription = standardCRUD("notificationSubscription", "notificationSubscriptions");

export const endpointsPhone = {
    ...createOne("phone"),
    verify: {
        endpoint: "/phone/verify",
        method: "POST" as const,
    },
    validate: {
        endpoint: "/phone/validate",
        method: "POST" as const,
    },
} as const;

export const endpointsPullRequest = standardCRUD("pullRequest", "pullRequests");

export const endpointsPushDevice = {
    ...findMany("pushDevices"),
    ...createOne("pushDevice"),
    ...updateOne("pushDevice"),
    testOne: {
        endpoint: "/pushDeviceTest/:id",
        method: "PUT" as const,
    },
} as const;

export const endpointsReaction = {
    ...findMany("reactions"),
    ...createOne("react"),
};

export const endpointsReminder = standardCRUD(LINKS.Reminder, "reminders");

export const endpointsReminderList = {
    ...createOne("reminderList"),
    ...updateOne("reminderList"),
} as const;

export const endpointsReport = standardCRUD(LINKS.Report, "reports");

export const endpointsReportResponse = standardCRUD("reportResponse", "reportResponses");

export const endpointsReputationHistory = {
    ...findOne("reputationHistory"),
    ...findMany("reputationHistories"),
} as const;

function resourceVersion(one: string) {
    return {
        endpoint: `/${one}/:publicId/v/:versionLabel`,
        method: "GET" as const,
    };
}
// There are several types of resources, but they all end up pointing to the same resource endpoints
export const endpointsResource = {
    ...standardCRUD("resource", "resources"),
    findResourceVersion: resourceVersion("resource"),
    findApiVersion: resourceVersion(LINKS.Api.replace("/", "")),
    findDataConverterVersion: resourceVersion(LINKS.DataConverter.replace("/", "")),
    findDataStructureVersion: resourceVersion(LINKS.DataStructure.replace("/", "")),
    findNoteVersion: resourceVersion(LINKS.Note.replace("/", "")),
    findProjectVersion: resourceVersion(LINKS.Project.replace("/", "")),
    findPromptVersion: resourceVersion(LINKS.Prompt.replace("/", "")),
    findRoutineMultiStepVersion: resourceVersion(LINKS.RoutineMultiStep.replace("/", "")),
    findRoutineSingleStepVersion: resourceVersion(LINKS.RoutineSingleStep.replace("/", "")),
    findSmartContractVersion: resourceVersion(LINKS.SmartContract.replace("/", "")),
} as const;

export const endpointsRun = standardCRUD(LINKS.Run, "runs");

export const endpointsRunIO = {
    ...findMany("run/io"),
} as const;

export const endpointsSchedule = standardCRUD(LINKS.Schedule, "schedules");

export const endpointsStatsResource = {
    ...findMany("stats/resource"),
} as const;

export const endpointsStatsSite = {
    ...findMany("stats/site"),
} as const;

export const endpointsStatsTeam = {
    ...findMany("stats/team"),
} as const;

export const endpointsStatsUser = {
    ...findMany("stats/user"),
} as const;

export const endpointsTag = standardCRUD("tag", "tags");

export const endpointsTask = {
    checkStatuses: {
        endpoint: "/task/checkStatuses",
        method: "GET" as const,
    },
    startLlmTask: {
        endpoint: "/task/start/llm",
        method: "POST" as const,
    },
    startRunTask: {
        endpoint: "/task/start/run",
        method: "POST" as const,
    },
    cancelTask: {
        endpoint: "/task/cancel",
        method: "POST" as const,
    },
};

export const endpointsTeam = standardCRUD(LINKS.Team, "teams");

export const endpointsTransfer = {
    ...findOne("transfer"),
    ...findMany("transfers"),
    ...updateOne("transfer"),
    requestSendOne: {
        endpoint: "/transfer/requestSend",
        method: "POST" as const,
    },
    requestReceiveOne: {
        endpoint: "/transfer/requestReceive",
        method: "POST" as const,
    },
    cancelOne: {
        endpoint: "/transfer/:id/cancel",
        method: "PUT" as const,
    },
    acceptOne: {
        endpoint: "/transfer/:id/accept",
        method: "PUT" as const,
    },
    denyOne: {
        endpoint: "/transfer/deny",
        method: "PUT" as const,
    },
};

export const endpointsUser = {
    ...findOne(LINKS.User),
    ...findMany("users"),
    profile: {
        endpoint: "/profile",
        method: "GET" as const,
    },
    profileUpdate: {
        endpoint: "/profile",
        method: "PUT" as const,
    },
    profileEmailUpdate: {
        endpoint: "/profile/email",
        method: "PUT" as const,
    },
    botCreateOne: {
        endpoint: "/bot",
        method: "POST" as const,
    },
    botUpdateOne: {
        endpoint: "/bot/:id",
        method: "PUT" as const,
    },
    importCalendar: {
        endpoint: "/import/calendar",
        method: "POST" as const,
    },
    importUserData: {
        endpoint: "/import/userData",
        method: "POST" as const,
    },
    exportCalendar: {
        endpoint: "/export/calendar",
        method: "GET" as const,
    },
    exportData: {
        endpoint: "/export/data",
        method: "GET" as const,
    },
} as const;

export const endpointsView = {
    ...findMany("views"),
} as const;

export const endpointsWallet = {
    ...updateOne("wallet"),
} as const;

