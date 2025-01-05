function findOne(one: string) {
    return {
        findOne: {
            endpoint: `/${one}/:id`,
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
    }
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

function deleteOne(one: string) {
    return {
        deleteOne: {
            endpoint: `/${one}/:id`,
            method: "DELETE" as const,
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

export const endpointsApi = standardCRUD("api", "apis");

export const endpointsApiKey = {
    ...createOne("apiKey"),
    ...updateOne("apiKey"),
    ...deleteOne("apiKey"),
    validateOne: {
        endpoint: "/apiKey/:id/validate",
        method: "PUT" as const,
    },
} as const;

export const endpointsApiVersion = standardCRUD("apiVersion", "apiVersions");

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
}

export const endpointsAward = {
    ...findMany("awards"),
} as const;

export const endpointsBookmark = standardCRUD("bookmark", "bookmarks");

export const endpointsBookmarkList = standardCRUD("bookmarkList", "bookmarkLists");

export const endpointsChat = standardCRUD("chat", "chats");

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
    ...standardCRUD("chatInvite", "chatInvites"),
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

export const endpointsCode = standardCRUD("code", "codes");

export const endpointsCodeVersion = standardCRUD("codeVersion", "codeVersions");

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

export const endpointsFocusMode = {
    ...standardCRUD("focusMode", "focusModes"),
    setActive: {
        endpoint: "/focusMode/active/:id",
        method: "PUT" as const,
    }
} as const;

export const endpointsIssue = {
    ...standardCRUD("issue", "issues"),
    closeOne: {
        endpoint: "/issue/:id/close",
        method: "PUT" as const,
    }
} as const;

export const endpointsLabel = standardCRUD("label", "labels");

export const endpointsMeeting = standardCRUD("meeting", "meetings");

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

export const endpointsNote = standardCRUD("note", "notes");

export const endpointsNoteVersion = standardCRUD("noteVersion", "noteVersions");

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

export const endpointsPost = standardCRUD("post", "posts");

export const endpointsProject = standardCRUD("project", "projects");

export const endpointsProjectVersion = {
    ...standardCRUD("projectVersion", "projectVersions"),
    contents: {
        endpoint: "/projectVersionContents",
        method: "GET" as const,
    },
} as const;

export const endpointsProjectVersionDirectory = standardCRUD("projectVersionDirectory", "projectVersionDirectories");

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

export const endpointsQuestion = standardCRUD("question", "questions");

export const endpointsQuestionAnswer = {
    ...standardCRUD("questionAnswer", "questionAnswers"),
    acceptOne: {
        endpoint: "/questionAnswer/:id/accept",
        method: "PUT" as const,
    },
} as const;

export const endpointsQuiz = standardCRUD("quiz", "quizzes");

export const endpointsQuizAttempt = standardCRUD("quizAttempt", "quizAttempts");

export const endpointsQuizQuestionResponse = {
    ...findOne("quizQuestionResponse"),
    ...findMany("quizQuestionResponses"),
} as const;

export const endpointsReaction = {
    ...findMany("reactions"),
    ...createOne("react"),
}

export const endpointsReminder = standardCRUD("reminder", "reminders");

export const endpointsReminderList = {
    ...createOne("reminderList"),
    ...updateOne("reminderList"),
} as const;

export const endpointsReport = standardCRUD("report", "reports");

export const endpointsReportResponse = standardCRUD("reportResponse", "reportResponses");

export const endpointsReputationHistory = {
    ...findOne("reputationHistory"),
    ...findMany("reputationHistories"),
} as const;

export const endpointsResourceList = standardCRUD("resourceList", "resourceLists");

export const endpointsRole = standardCRUD("role", "roles");

export const endpointsRoutine = standardCRUD("routine", "routines");

export const endpointsRoutineVersion = standardCRUD("routineVersion", "routineVersions");

export const endpointsRunProject = standardCRUD("run/project", "run/projects");

export const endpointsRunRoutine = standardCRUD("run/routine", "run/routines");

export const endpointsRunRoutineInput = {
    ...findMany("run/routine/inputs"),
} as const;

export const endpointsRunRoutineOutput = {
    ...findMany("run/routine/outputs"),
} as const;

export const endpointsSchedule = standardCRUD("schedule", "schedules");

export const endpointsStandard = standardCRUD("standard", "standards");

export const endpointsStandardVersion = standardCRUD("standardVersion", "standardVersions");

export const endpointsStatsApi = {
    ...findMany("stats/api"),
} as const;

export const endpointsStatsCode = {
    ...findMany("stats/code"),
} as const;

export const endpointsStatsProject = {
    ...findMany("stats/projects"),
} as const;

export const endpointsStatsQuiz = {
    ...findMany("stats/quiz"),
} as const;

export const endpointsStatsRoutine = {
    ...findMany("stats/routine"),
} as const;

export const endpointsStatsSite = {
    ...findMany("stats/site"),
} as const;

export const endpointsStatsStandard = {
    ...findMany("stats/standard"),
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
}

export const endpointsTeam = standardCRUD("team", "teams");

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
}

export const endpointsTranslate = {
    translate: {
        endpoint: "/translate",
        method: "GET" as const,
    },
} as const;

export const endpointsUnions = {
    projectOrRoutines: {
        endpoint: "/unions/projectOrRoutines",
        method: "GET" as const,
    },
    projectOrTeams: {
        endpoint: "/unions/projectOrTeams",
        method: "GET" as const,
    },
    runProjectOrRunRoutines: {
        endpoint: "/unions/runProjectOrRunRoutines",
        method: "GET" as const,
    },
} as const;

export const endpointsUser = {
    ...findOne("user"),
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

