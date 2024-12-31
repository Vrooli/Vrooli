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

function deleteMany(many: string) {
    return {
        deleteMany: {
            endpoint: `/${many}`,
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
//TODO for morning: finish rewriting these and updating endpoints/rest/* files. Move this out of the generated folder and delete the generated folder. Remoe API_GENERATE script for ui package, leaving only the server script for generating prisma select objects and the shared script for generating OpenAPI schema

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

export const endpointGetIssue = {
    endpoint: "/issue/:id",
    method: "GET",
} as const;

export const endpointPutIssue = {
    endpoint: "/issue/:id",
    method: "PUT",
} as const;

export const endpointGetIssues = {
    endpoint: "/issues",
    method: "GET",
} as const;

export const endpointPostIssue = {
    endpoint: "/issue",
    method: "POST",
} as const;

export const endpointPutIssueClose = {
    endpoint: "/issue/:id/close",
    method: "PUT",
} as const;

export const endpointsLabel = standardCRUD("label", "labels");

export const endpointsMeeting = standardCRUD("meeting", "meetings");

export const endpointGetMeetingInvite = {
    endpoint: "/meetingInvite/:id",
    method: "GET",
} as const;

export const endpointPutMeetingInvite = {
    endpoint: "/meetingInvite/:id",
    method: "PUT",
} as const;

export const endpointGetMeetingInvites = {
    endpoint: "/meetingInvites",
    method: "GET",
} as const;

export const endpointPostMeetingInvites = {
    endpoint: "/meetingInvites",
    method: "POST",
} as const;

export const endpointPutMeetingInvites = {
    endpoint: "/meetingInvites",
    method: "PUT",
} as const;

export const endpointPostMeetingInvite = {
    endpoint: "/meetingInvite",
    method: "POST",
} as const;

export const endpointPutMeetingInviteAccept = {
    endpoint: "/meetingInvite/:id/accept",
    method: "PUT",
} as const;

export const endpointPutMeetingInviteDecline = {
    endpoint: "/meetingInvite/:id/decline",
    method: "PUT",
} as const;

export const endpointGetMember = {
    endpoint: "/member/:id",
    method: "GET",
} as const;

export const endpointPutMember = {
    endpoint: "/member/:id",
    method: "PUT",
} as const;

export const endpointGetMembers = {
    endpoint: "/members",
    method: "GET",
} as const;

export const endpointGetMemberInvite = {
    endpoint: "/memberInvite/:id",
    method: "GET",
} as const;

export const endpointPutMemberInvite = {
    endpoint: "/memberInvite/:id",
    method: "PUT",
} as const;

export const endpointGetMemberInvites = {
    endpoint: "/memberInvites",
    method: "GET",
} as const;

export const endpointPostMemberInvites = {
    endpoint: "/memberInvites",
    method: "POST",
} as const;

export const endpointPutMemberInvites = {
    endpoint: "/memberInvites",
    method: "PUT",
} as const;

export const endpointPostMemberInvite = {
    endpoint: "/memberInvite",
    method: "POST",
} as const;

export const endpointPutMemberInviteAccept = {
    endpoint: "/memberInvite/:id/accept",
    method: "PUT",
} as const;

export const endpointPutMemberInviteDecline = {
    endpoint: "/memberInvite/:id/decline",
    method: "PUT",
} as const;

export const endpointPostNode = {
    endpoint: "/node",
    method: "POST",
} as const;

export const endpointPutNode = {
    endpoint: "/node/:id",
    method: "PUT",
} as const;

export const endpointsNote = standardCRUD("note", "notes");

export const endpointsNoteVersion = standardCRUD("noteVersion", "noteVersions");

export const endpointGetNotification = {
    endpoint: "/notification/:id",
    method: "GET",
} as const;

export const endpointPutNotification = {
    endpoint: "/notification/:id",
    method: "PUT",
} as const;

export const endpointGetNotifications = {
    endpoint: "/notifications",
    method: "GET",
} as const;

export const endpointPutNotificationsMarkAllAsRead = {
    endpoint: "/notifications/markAllAsRead",
    method: "PUT",
} as const;

export const endpointGetNotificationSettings = {
    endpoint: "/notificationSettings",
    method: "GET",
} as const;

export const endpointPutNotificationSettings = {
    endpoint: "/notificationSettings",
    method: "PUT",
} as const;

export const endpointsNotificationSubscription = standardCRUD("notificationSubscription", "notificationSubscriptions");

export const endpointPostPhone = {
    endpoint: "/phone",
    method: "POST",
} as const;

export const endpointPostPhoneVerificationText = {
    endpoint: "/phone/verificationText",
    method: "POST",
} as const;

export const endpointPostPhoneValidateText = {
    endpoint: "/phone/validateText",
    method: "POST",
} as const;

export const endpointsPost = standardCRUD("post", "posts");

export const endpointsProject = standardCRUD("project", "projects");

export const endpointGetProjectVersion = {
    endpoint: "/projectVersion/:id",
    method: "GET",
} as const;

export const endpointPutProjectVersion = {
    endpoint: "/projectVersion/:id",
    method: "PUT",
} as const;

export const endpointGetProjectVersions = {
    endpoint: "/projectVersions",
    method: "GET",
} as const;

export const endpointGetProjectVersionContents = {
    endpoint: "/projectVersionContents",
    method: "GET",
} as const;

export const endpointPostProjectVersion = {
    endpoint: "/projectVersion",
    method: "POST",
} as const;

export const endpointsProjectVersionDirectory = standardCRUD("projectVersionDirectory", "projectVersionDirectories");

export const endpointsPullRequest = standardCRUD("pullRequest", "pullRequests");

export const endpointPostPushDevice = {
    endpoint: "/pushDevice",
    method: "POST",
} as const;

export const endpointGetPushDevices = {
    endpoint: "/pushDevices",
    method: "GET",
} as const;

export const endpointPutPushDevice = {
    endpoint: "/pushDevice/:id",
    method: "PUT",
} as const;

export const endpointPutPushDeviceTest = {
    endpoint: "/pushDeviceTest/:id",
    method: "PUT",
} as const;

export const endpointsQuestion = standardCRUD("question", "questions");

export const endpointGetQuestionAnswer = {
    endpoint: "/questionAnswer/:id",
    method: "GET",
} as const;

export const endpointPutQuestionAnswer = {
    endpoint: "/questionAnswer/:id",
    method: "PUT",
} as const;

export const endpointGetQuestionAnswers = {
    endpoint: "/questionAnswers",
    method: "GET",
} as const;

export const endpointPostQuestionAnswer = {
    endpoint: "/questionAnswer",
    method: "POST",
} as const;

export const endpointPutQuestionAnswerAccept = {
    endpoint: "/questionAnswer/:id/accept",
    method: "PUT",
} as const;

export const endpointsQuiz = standardCRUD("quiz", "quizzes");

export const endpointGetQuizAttempt = {
    endpoint: "/quizAttempt/:id",
    method: "GET",
} as const;

export const endpointPutQuizAttempt = {
    endpoint: "/quizAttempt/:id",
    method: "PUT",
} as const;

export const endpointGetQuizAttempts = {
    endpoint: "/quizAttempts",
    method: "GET",
} as const;

export const endpointPostQuizAttempt = {
    endpoint: "/quizAttempt",
    method: "POST",
} as const;

export const endpointGetQuizQuestion = {
    endpoint: "/quizQuestion/:id",
    method: "GET",
} as const;

export const endpointGetQuizQuestions = {
    endpoint: "/quizQuestions",
    method: "GET",
} as const;

export const endpointGetQuizQuestionResponse = {
    endpoint: "/quizQuestionResponse/:id",
    method: "GET",
} as const;

export const endpointGetQuizQuestionResponses = {
    endpoint: "/quizQuestionResponses",
    method: "GET",
} as const;

export const endpointGetReactions = {
    endpoint: "/reactions",
    method: "GET",
} as const;

export const endpointPostReact = {
    endpoint: "/react",
    method: "POST",
} as const;

export const endpointGetReminder = {
    endpoint: "/reminder/:id",
    method: "GET",
} as const;

export const endpointPutReminder = {
    endpoint: "/reminder/:id",
    method: "PUT",
} as const;

export const endpointGetReminders = {
    endpoint: "/reminders",
    method: "GET",
} as const;

export const endpointPostReminder = {
    endpoint: "/reminder",
    method: "POST",
} as const;

export const endpointPostReminderList = {
    endpoint: "/reminderList",
    method: "POST",
} as const;

export const endpointPutReminderList = {
    endpoint: "/reminderList/:id",
    method: "PUT",
} as const;

export const endpointGetReport = {
    endpoint: "/report/:id",
    method: "GET",
} as const;

export const endpointPutReport = {
    endpoint: "/report/:id",
    method: "PUT",
} as const;

export const endpointGetReports = {
    endpoint: "/reports",
    method: "GET",
} as const;

export const endpointPostReport = {
    endpoint: "/report",
    method: "POST",
} as const;

export const endpointGetReportResponse = {
    endpoint: "/reportResponse/:id",
    method: "GET",
} as const;

export const endpointPutReportResponse = {
    endpoint: "/reportResponse/:id",
    method: "PUT",
} as const;

export const endpointGetReportResponses = {
    endpoint: "/reportResponses",
    method: "GET",
} as const;

export const endpointPostReportResponse = {
    endpoint: "/reportResponse",
    method: "POST",
} as const;

export const endpointGetReputationHistory = {
    endpoint: "/reputationHistory/:id",
    method: "GET",
} as const;

export const endpointGetReputationHistories = {
    endpoint: "/reputationHistories",
    method: "GET",
} as const;

export const endpointGetResource = {
    endpoint: "/resource/:id",
    method: "GET",
} as const;

export const endpointPutResource = {
    endpoint: "/resource/:id",
    method: "PUT",
} as const;

export const endpointGetResources = {
    endpoint: "/resources",
    method: "GET",
} as const;

export const endpointPostResource = {
    endpoint: "/resource",
    method: "POST",
} as const;

export const endpointGetResourceList = {
    endpoint: "/resourceList/:id",
    method: "GET",
} as const;

export const endpointPutResourceList = {
    endpoint: "/resourceList/:id",
    method: "PUT",
} as const;

export const endpointGetResourceLists = {
    endpoint: "/resourceLists",
    method: "GET",
} as const;

export const endpointPostResourceList = {
    endpoint: "/resourceList",
    method: "POST",
} as const;

export const endpointGetRole = {
    endpoint: "/role/:id",
    method: "GET",
} as const;

export const endpointPutRole = {
    endpoint: "/role/:id",
    method: "PUT",
} as const;

export const endpointGetRoles = {
    endpoint: "/roles",
    method: "GET",
} as const;

export const endpointPostRole = {
    endpoint: "/role",
    method: "POST",
} as const;

export const endpointGetRoutine = {
    endpoint: "/routine/:id",
    method: "GET",
} as const;

export const endpointPutRoutine = {
    endpoint: "/routine/:id",
    method: "PUT",
} as const;

export const endpointGetRoutines = {
    endpoint: "/routines",
    method: "GET",
} as const;

export const endpointPostRoutine = {
    endpoint: "/routine",
    method: "POST",
} as const;

export const endpointGetRoutineVersion = {
    endpoint: "/routineVersion/:id",
    method: "GET",
} as const;

export const endpointPutRoutineVersion = {
    endpoint: "/routineVersion/:id",
    method: "PUT",
} as const;

export const endpointGetRoutineVersions = {
    endpoint: "/routineVersions",
    method: "GET",
} as const;

export const endpointPostRoutineVersion = {
    endpoint: "/routineVersion",
    method: "POST",
} as const;

export const endpointGetRunProject = {
    endpoint: "/runProject/:id",
    method: "GET",
} as const;

export const endpointPutRunProject = {
    endpoint: "/runProject/:id",
    method: "PUT",
} as const;

export const endpointGetRunProjects = {
    endpoint: "/runProjects",
    method: "GET",
} as const;

export const endpointPostRunProject = {
    endpoint: "/runProject",
    method: "POST",
} as const;

export const endpointDeleteRunProjectDeleteAll = {
    endpoint: "/runProject/deleteAll",
    method: "DELETE",
} as const;

export const endpointGetRunRoutine = {
    endpoint: "/runRoutine/:id",
    method: "GET",
} as const;

export const endpointPutRunRoutine = {
    endpoint: "/runRoutine/:id",
    method: "PUT",
} as const;

export const endpointGetRunRoutines = {
    endpoint: "/runRoutines",
    method: "GET",
} as const;

export const endpointPostRunRoutine = {
    endpoint: "/runRoutine",
    method: "POST",
} as const;

export const endpointDeleteRunRoutineDeleteAll = {
    endpoint: "/runRoutine/deleteAll",
    method: "DELETE",
} as const;

export const endpointGetRunRoutineInputs = {
    endpoint: "/runRoutineInputs",
    method: "GET",
} as const;

export const endpointGetRunRoutineOutputs = {
    endpoint: "/runRoutineOutputs",
    method: "GET",
} as const;

export const endpointGetSchedule = {
    endpoint: "/schedule/:id",
    method: "GET",
} as const;

export const endpointPutSchedule = {
    endpoint: "/schedule/:id",
    method: "PUT",
} as const;

export const endpointGetSchedules = {
    endpoint: "/schedules",
    method: "GET",
} as const;

export const endpointPostSchedule = {
    endpoint: "/schedule",
    method: "POST",
} as const;

export const endpointGetScheduleException = {
    endpoint: "/scheduleException/:id",
    method: "GET",
} as const;

export const endpointPutScheduleException = {
    endpoint: "/scheduleException/:id",
    method: "PUT",
} as const;

export const endpointGetScheduleExceptions = {
    endpoint: "/scheduleExceptions",
    method: "GET",
} as const;

export const endpointPostScheduleException = {
    endpoint: "/scheduleException",
    method: "POST",
} as const;

export const endpointGetScheduleRecurrence = {
    endpoint: "/scheduleRecurrence/:id",
    method: "GET",
} as const;

export const endpointPutScheduleRecurrence = {
    endpoint: "/scheduleRecurrence/:id",
    method: "PUT",
} as const;

export const endpointGetScheduleRecurrences = {
    endpoint: "/scheduleRecurrences",
    method: "GET",
} as const;

export const endpointPostScheduleRecurrence = {
    endpoint: "/scheduleRecurrence",
    method: "POST",
} as const;

export const endpointGetStandard = {
    endpoint: "/standard/:id",
    method: "GET",
} as const;

export const endpointPutStandard = {
    endpoint: "/standard/:id",
    method: "PUT",
} as const;

export const endpointGetStandards = {
    endpoint: "/standards",
    method: "GET",
} as const;

export const endpointPostStandard = {
    endpoint: "/standard",
    method: "POST",
} as const;

export const endpointGetStandardVersion = {
    endpoint: "/standardVersion/:id",
    method: "GET",
} as const;

export const endpointPutStandardVersion = {
    endpoint: "/standardVersion/:id",
    method: "PUT",
} as const;

export const endpointGetStandardVersions = {
    endpoint: "/standardVersions",
    method: "GET",
} as const;

export const endpointPostStandardVersion = {
    endpoint: "/standardVersion",
    method: "POST",
} as const;

export const endpointGetStatsApi = {
    endpoint: "/stats/api",
    method: "GET",
} as const;

export const endpointGetStatsCode = {
    endpoint: "/stats/code",
    method: "GET",
} as const;

export const endpointGetStatsProject = {
    endpoint: "/stats/project",
    method: "GET",
} as const;

export const endpointGetStatsQuiz = {
    endpoint: "/stats/quiz",
    method: "GET",
} as const;

export const endpointGetStatsRoutine = {
    endpoint: "/stats/routine",
    method: "GET",
} as const;

export const endpointGetStatsSite = {
    endpoint: "/stats/site",
    method: "GET",
} as const;

export const endpointGetStatsStandard = {
    endpoint: "/stats/standard",
    method: "GET",
} as const;

export const endpointGetStatsTeam = {
    endpoint: "/stats/team",
    method: "GET",
} as const;

export const endpointGetStatsUser = {
    endpoint: "/stats/user",
    method: "GET",
} as const;

export const endpointGetTag = {
    endpoint: "/tag/:id",
    method: "GET",
} as const;

export const endpointPutTag = {
    endpoint: "/tag/:id",
    method: "PUT",
} as const;

export const endpointGetTags = {
    endpoint: "/tags",
    method: "GET",
} as const;

export const endpointPostTag = {
    endpoint: "/tag",
    method: "POST",
} as const;

export const endpointPostStartLlmTask = {
    endpoint: "/startLlmTask",
    method: "POST",
} as const;

export const endpointPostStartRunTask = {
    endpoint: "/startRunTask",
    method: "POST",
} as const;

export const endpointPostCancelTask = {
    endpoint: "/cancelTask",
    method: "POST",
} as const;

export const endpointGetCheckTaskStatuses = {
    endpoint: "/checkTaskStatuses",
    method: "GET",
} as const;

export const endpointGetTeam = {
    endpoint: "/team/:id",
    method: "GET",
} as const;

export const endpointPutTeam = {
    endpoint: "/team/:id",
    method: "PUT",
} as const;

export const endpointGetTeams = {
    endpoint: "/teams",
    method: "GET",
} as const;

export const endpointPostTeam = {
    endpoint: "/team",
    method: "POST",
} as const;

export const endpointGetTransfer = {
    endpoint: "/transfer/:id",
    method: "GET",
} as const;

export const endpointPutTransfer = {
    endpoint: "/transfer/:id",
    method: "PUT",
} as const;

export const endpointGetTransfers = {
    endpoint: "/transfers",
    method: "GET",
} as const;

export const endpointPostTransferRequestSend = {
    endpoint: "/transfer/requestSend",
    method: "POST",
} as const;

export const endpointPostTransferRequestReceive = {
    endpoint: "/transfer/requestReceive",
    method: "POST",
} as const;

export const endpointPutTransferCancel = {
    endpoint: "/transfer/:id/cancel",
    method: "PUT",
} as const;

export const endpointPutTransferAccept = {
    endpoint: "/transfer/:id/accept",
    method: "PUT",
} as const;

export const endpointPutTransferDeny = {
    endpoint: "/transfer/deny",
    method: "PUT",
} as const;

export const endpointGetTranslate = {
    endpoint: "/translate",
    method: "GET",
} as const;

export const endpointGetUnionsProjectOrRoutines = {
    endpoint: "/unions/projectOrRoutines",
    method: "GET",
} as const;

export const endpointGetUnionsProjectOrTeams = {
    endpoint: "/unions/projectOrTeams",
    method: "GET",
} as const;

export const endpointGetUnionsRunProjectOrRunRoutines = {
    endpoint: "/unions/runProjectOrRunRoutines",
    method: "GET",
} as const;

export const endpointPutBot = {
    endpoint: "/bot/:id",
    method: "PUT",
} as const;

export const endpointPostBot = {
    endpoint: "/bot",
    method: "POST",
} as const;

export const endpointGetProfile = {
    endpoint: "/profile",
    method: "GET",
} as const;

export const endpointPutProfile = {
    endpoint: "/profile",
    method: "PUT",
} as const;

export const endpointGetUser = {
    endpoint: "/user/:id",
    method: "GET",
} as const;

export const endpointGetUsers = {
    endpoint: "/users",
    method: "GET",
} as const;

export const endpointPutProfileEmail = {
    endpoint: "/profile/email",
    method: "PUT",
} as const;

export const endpointDeleteUser = {
    endpoint: "/user",
    method: "DELETE",
} as const;

export const endpointGetViews = {
    endpoint: "/views",
    method: "GET",
} as const;

export const endpointPutWallet = {
    endpoint: "/wallet/:id",
    method: "PUT",
} as const;

