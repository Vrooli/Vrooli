export const endpointGetApi = {
    endpoint: "/api/:id",
    method: "GET",
} as const;

export const endpointPutApi = {
    endpoint: "/api/:id",
    method: "PUT",
} as const;

export const endpointGetApis = {
    endpoint: "/apis",
    method: "GET",
} as const;

export const endpointPostApi = {
    endpoint: "/api",
    method: "POST",
} as const;

export const endpointPostApiKey = {
    endpoint: "/apiKey",
    method: "POST",
} as const;

export const endpointPutApiKey = {
    endpoint: "/apiKey/:id",
    method: "PUT",
} as const;

export const endpointDeleteApiKey = {
    endpoint: "/apiKey/:id",
    method: "DELETE",
} as const;

export const endpointPostApiKeyValidate = {
    endpoint: "/apiKey/validate",
    method: "POST",
} as const;

export const endpointGetApiVersion = {
    endpoint: "/apiVersion/:id",
    method: "GET",
} as const;

export const endpointPutApiVersion = {
    endpoint: "/apiVersion/:id",
    method: "PUT",
} as const;

export const endpointGetApiVersions = {
    endpoint: "/apiVersions",
    method: "GET",
} as const;

export const endpointPostApiVersion = {
    endpoint: "/apiVersion",
    method: "POST",
} as const;

export const endpointPostAuthEmailLogin = {
    endpoint: "/auth/email/login",
    method: "POST",
} as const;

export const endpointPostAuthEmailSignup = {
    endpoint: "/auth/email/signup",
    method: "POST",
} as const;

export const endpointPostAuthEmailRequestPasswordChange = {
    endpoint: "/auth/email/requestPasswordChange",
    method: "POST",
} as const;

export const endpointPostAuthEmailResetPassword = {
    endpoint: "/auth/email/resetPassword",
    method: "POST",
} as const;

export const endpointPostAuthGuestLogin = {
    endpoint: "/auth/guest/login",
    method: "POST",
} as const;

export const endpointPostAuthLogout = {
    endpoint: "/auth/logout",
    method: "POST",
} as const;

export const endpointPostAuthValidateSession = {
    endpoint: "/auth/validateSession",
    method: "POST",
} as const;

export const endpointPostAuthSwitchCurrentAccount = {
    endpoint: "/auth/switchCurrentAccount",
    method: "POST",
} as const;

export const endpointPostAuthWalletInit = {
    endpoint: "/auth/wallet/init",
    method: "POST",
} as const;

export const endpointPostAuthWalletComplete = {
    endpoint: "/auth/wallet/complete",
    method: "POST",
} as const;

export const endpointPostAwards = {
    endpoint: "/awards",
    method: "POST",
} as const;

export const endpointGetBookmark = {
    endpoint: "/bookmark/:id",
    method: "GET",
} as const;

export const endpointPutBookmark = {
    endpoint: "/bookmark/:id",
    method: "PUT",
} as const;

export const endpointGetBookmarks = {
    endpoint: "/bookmarks",
    method: "GET",
} as const;

export const endpointPostBookmark = {
    endpoint: "/bookmark",
    method: "POST",
} as const;

export const endpointGetBookmarkList = {
    endpoint: "/bookmarkList/:id",
    method: "GET",
} as const;

export const endpointPutBookmarkList = {
    endpoint: "/bookmarkList/:id",
    method: "PUT",
} as const;

export const endpointGetBookmarkLists = {
    endpoint: "/bookmarkLists",
    method: "GET",
} as const;

export const endpointPostBookmarkList = {
    endpoint: "/bookmarkList",
    method: "POST",
} as const;

export const endpointGetChat = {
    endpoint: "/chat/:id",
    method: "GET",
} as const;

export const endpointPutChat = {
    endpoint: "/chat/:id",
    method: "PUT",
} as const;

export const endpointGetChats = {
    endpoint: "/chats",
    method: "GET",
} as const;

export const endpointPostChat = {
    endpoint: "/chat",
    method: "POST",
} as const;

export const endpointGetChatInvite = {
    endpoint: "/chatInvite/:id",
    method: "GET",
} as const;

export const endpointPutChatInvite = {
    endpoint: "/chatInvite/:id",
    method: "PUT",
} as const;

export const endpointGetChatInvites = {
    endpoint: "/chatInvites",
    method: "GET",
} as const;

export const endpointPostChatInvite = {
    endpoint: "/chatInvite",
    method: "POST",
} as const;

export const endpointPutChatInviteAccept = {
    endpoint: "/chatInvite/:id/accept",
    method: "PUT",
} as const;

export const endpointPutChatInviteDecline = {
    endpoint: "/chatInvite/:id/decline",
    method: "PUT",
} as const;

export const endpointGetChatMessage = {
    endpoint: "/chatMessage/:id",
    method: "GET",
} as const;

export const endpointPutChatMessage = {
    endpoint: "/chatMessage/:id",
    method: "PUT",
} as const;

export const endpointGetChatMessages = {
    endpoint: "/chatMessages",
    method: "GET",
} as const;

export const endpointPostChatMessage = {
    endpoint: "/chatMessage",
    method: "POST",
} as const;

export const endpointGetChatParticipant = {
    endpoint: "/chatParticipant/:id",
    method: "GET",
} as const;

export const endpointPutChatParticipant = {
    endpoint: "/chatParticipant/:id",
    method: "PUT",
} as const;

export const endpointGetChatParticipants = {
    endpoint: "/chatParticipants",
    method: "GET",
} as const;

export const endpointGetComment = {
    endpoint: "/comment/:id",
    method: "GET",
} as const;

export const endpointPutComment = {
    endpoint: "/comment/:id",
    method: "PUT",
} as const;

export const endpointGetComments = {
    endpoint: "/comments",
    method: "GET",
} as const;

export const endpointPostComment = {
    endpoint: "/comment",
    method: "POST",
} as const;

export const endpointPostCopy = {
    endpoint: "/copy",
    method: "POST",
} as const;

export const endpointPostDeleteOne = {
    endpoint: "/deleteOne",
    method: "POST",
} as const;

export const endpointPostDeleteMany = {
    endpoint: "/deleteMany",
    method: "POST",
} as const;

export const endpointPostEmail = {
    endpoint: "/email",
    method: "POST",
} as const;

export const endpointPostEmailVerification = {
    endpoint: "/email/verification",
    method: "POST",
} as const;

export const endpointGetFeedHome = {
    endpoint: "/feed/home",
    method: "GET",
} as const;

export const endpointGetFeedPopular = {
    endpoint: "/feed/popular",
    method: "GET",
} as const;

export const endpointGetFocusMode = {
    endpoint: "/focusMode/:id",
    method: "GET",
} as const;

export const endpointPutFocusMode = {
    endpoint: "/focusMode/:id",
    method: "PUT",
} as const;

export const endpointGetFocusModes = {
    endpoint: "/focusModes",
    method: "GET",
} as const;

export const endpointPostFocusMode = {
    endpoint: "/focusMode",
    method: "POST",
} as const;

export const endpointPutFocusModeActive = {
    endpoint: "/focusMode/active",
    method: "PUT",
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

export const endpointGetLabel = {
    endpoint: "/label/:id",
    method: "GET",
} as const;

export const endpointPutLabel = {
    endpoint: "/label/:id",
    method: "PUT",
} as const;

export const endpointGetLabels = {
    endpoint: "/labels",
    method: "GET",
} as const;

export const endpointPostLabel = {
    endpoint: "/label",
    method: "POST",
} as const;

export const endpointGetMeeting = {
    endpoint: "/meeting/:id",
    method: "GET",
} as const;

export const endpointPutMeeting = {
    endpoint: "/meeting/:id",
    method: "PUT",
} as const;

export const endpointGetMeetings = {
    endpoint: "/meetings",
    method: "GET",
} as const;

export const endpointPostMeeting = {
    endpoint: "/meeting",
    method: "POST",
} as const;

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

export const endpointGetNote = {
    endpoint: "/note/:id",
    method: "GET",
} as const;

export const endpointPutNote = {
    endpoint: "/note/:id",
    method: "PUT",
} as const;

export const endpointGetNotes = {
    endpoint: "/notes",
    method: "GET",
} as const;

export const endpointPostNote = {
    endpoint: "/note",
    method: "POST",
} as const;

export const endpointGetNoteVersion = {
    endpoint: "/noteVersion/:id",
    method: "GET",
} as const;

export const endpointPutNoteVersion = {
    endpoint: "/noteVersion/:id",
    method: "PUT",
} as const;

export const endpointGetNoteVersions = {
    endpoint: "/noteVersions",
    method: "GET",
} as const;

export const endpointPostNoteVersion = {
    endpoint: "/noteVersion",
    method: "POST",
} as const;

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

export const endpointGetNotificationSubscription = {
    endpoint: "/notificationSubscription/:id",
    method: "GET",
} as const;

export const endpointPutNotificationSubscription = {
    endpoint: "/notificationSubscription/:id",
    method: "PUT",
} as const;

export const endpointGetNotificationSubscriptions = {
    endpoint: "/notificationSubscriptions",
    method: "GET",
} as const;

export const endpointPostNotificationSubscription = {
    endpoint: "/notificationSubscription",
    method: "POST",
} as const;

export const endpointGetOrganization = {
    endpoint: "/organization/:id",
    method: "GET",
} as const;

export const endpointPutOrganization = {
    endpoint: "/organization/:id",
    method: "PUT",
} as const;

export const endpointGetOrganizations = {
    endpoint: "/organizations",
    method: "GET",
} as const;

export const endpointPostOrganization = {
    endpoint: "/organization",
    method: "POST",
} as const;

export const endpointPostPhone = {
    endpoint: "/phone",
    method: "POST",
} as const;

export const endpointGetPost = {
    endpoint: "/post/:id",
    method: "GET",
} as const;

export const endpointPutPost = {
    endpoint: "/post/:id",
    method: "PUT",
} as const;

export const endpointGetPosts = {
    endpoint: "/posts",
    method: "GET",
} as const;

export const endpointPostPost = {
    endpoint: "/post",
    method: "POST",
} as const;

export const endpointGetProject = {
    endpoint: "/project/:id",
    method: "GET",
} as const;

export const endpointPutProject = {
    endpoint: "/project/:id",
    method: "PUT",
} as const;

export const endpointGetProjects = {
    endpoint: "/projects",
    method: "GET",
} as const;

export const endpointPostProject = {
    endpoint: "/project",
    method: "POST",
} as const;

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

export const endpointGetProjectVersionDirectories = {
    endpoint: "/projectVersionDirectories",
    method: "GET",
} as const;

export const endpointGetPullRequest = {
    endpoint: "/pullRequest/:id",
    method: "GET",
} as const;

export const endpointPutPullRequest = {
    endpoint: "/pullRequest/:id",
    method: "PUT",
} as const;

export const endpointGetPullRequests = {
    endpoint: "/pullRequests",
    method: "GET",
} as const;

export const endpointPostPullRequest = {
    endpoint: "/pullRequest",
    method: "POST",
} as const;

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

export const endpointGetQuestion = {
    endpoint: "/question/:id",
    method: "GET",
} as const;

export const endpointPutQuestion = {
    endpoint: "/question/:id",
    method: "PUT",
} as const;

export const endpointGetQuestions = {
    endpoint: "/questions",
    method: "GET",
} as const;

export const endpointPostQuestion = {
    endpoint: "/question",
    method: "POST",
} as const;

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

export const endpointGetQuiz = {
    endpoint: "/quiz/:id",
    method: "GET",
} as const;

export const endpointPutQuiz = {
    endpoint: "/quiz/:id",
    method: "PUT",
} as const;

export const endpointGetQuizzes = {
    endpoint: "/quizzes",
    method: "GET",
} as const;

export const endpointPostQuiz = {
    endpoint: "/quiz",
    method: "POST",
} as const;

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

export const endpointPutRunProjectComplete = {
    endpoint: "/runProject/:id/complete",
    method: "PUT",
} as const;

export const endpointPutRunProjectCancel = {
    endpoint: "/runProject/:id/cancel",
    method: "PUT",
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

export const endpointPutRunRoutineComplete = {
    endpoint: "/runRoutine/:id/complete",
    method: "PUT",
} as const;

export const endpointPutRunRoutineCancel = {
    endpoint: "/runRoutine/:id/cancel",
    method: "PUT",
} as const;

export const endpointGetRunRoutineInputs = {
    endpoint: "/runRoutineInputs",
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

export const endpointGetSmartContract = {
    endpoint: "/smartContract/:id",
    method: "GET",
} as const;

export const endpointPutSmartContract = {
    endpoint: "/smartContract/:id",
    method: "PUT",
} as const;

export const endpointGetSmartContracts = {
    endpoint: "/smartContracts",
    method: "GET",
} as const;

export const endpointPostSmartContract = {
    endpoint: "/smartContract",
    method: "POST",
} as const;

export const endpointGetSmartContractVersion = {
    endpoint: "/smartContractVersion/:id",
    method: "GET",
} as const;

export const endpointPutSmartContractVersion = {
    endpoint: "/smartContractVersion/:id",
    method: "PUT",
} as const;

export const endpointGetSmartContractVersions = {
    endpoint: "/smartContractVersions",
    method: "GET",
} as const;

export const endpointPostSmartContractVersion = {
    endpoint: "/smartContractVersion",
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

export const endpointGetStatsOrganization = {
    endpoint: "/stats/organization",
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

export const endpointGetStatsSmartContract = {
    endpoint: "/stats/smartContract",
    method: "GET",
} as const;

export const endpointGetStatsStandard = {
    endpoint: "/stats/standard",
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

export const endpointGetUnionsProjectOrOrganizations = {
    endpoint: "/unions/projectOrOrganizations",
    method: "GET",
} as const;

export const endpointGetUnionsRunProjectOrRunRoutines = {
    endpoint: "/unions/runProjectOrRunRoutines",
    method: "GET",
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

export const endpointPostWalletHandles = {
    endpoint: "/wallet/handles",
    method: "POST",
} as const;

export const endpointPutWallet = {
    endpoint: "/wallet/:id",
    method: "PUT",
} as const;