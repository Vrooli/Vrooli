export const endpointGetApi = {
    endpoint: "/api/:id",
    method: "GET",
    tag: "api",
} as const;

export const endpointPutApi = {
    endpoint: "/api/:id",
    method: "PUT",
    tag: "api",
} as const;

export const endpointGetApis = {
    endpoint: "/apis",
    method: "GET",
    tag: "api",
} as const;

export const endpointPostApi = {
    endpoint: "/api",
    method: "POST",
    tag: "api",
} as const;

export const endpointPostApiKey = {
    endpoint: "/apiKey",
    method: "POST",
    tag: "apiKey",
} as const;

export const endpointPutApiKey = {
    endpoint: "/apiKey/:id",
    method: "PUT",
    tag: "apiKey",
} as const;

export const endpointDeleteApiKey = {
    endpoint: "/apiKey/:id",
    method: "DELETE",
    tag: "apiKey",
} as const;

export const endpointPostApiKeyValidate = {
    endpoint: "/apiKey/validate",
    method: "POST",
    tag: "apiKey",
} as const;

export const endpointGetApiVersion = {
    endpoint: "/apiVersion/:id",
    method: "GET",
    tag: "apiVersion",
} as const;

export const endpointPutApiVersion = {
    endpoint: "/apiVersion/:id",
    method: "PUT",
    tag: "apiVersion",
} as const;

export const endpointGetApiVersions = {
    endpoint: "/apiVersions",
    method: "GET",
    tag: "apiVersion",
} as const;

export const endpointPostApiVersion = {
    endpoint: "/apiVersion",
    method: "POST",
    tag: "apiVersion",
} as const;

export const endpointPostAuthEmailLogin = {
    endpoint: "/auth/email/login",
    method: "POST",
    tag: "auth",
} as const;

export const endpointPostAuthEmailSignup = {
    endpoint: "/auth/email/signup",
    method: "POST",
    tag: "auth",
} as const;

export const endpointPostAuthEmailRequestPasswordChange = {
    endpoint: "/auth/email/requestPasswordChange",
    method: "POST",
    tag: "auth",
} as const;

export const endpointPostAuthEmailResetPassword = {
    endpoint: "/auth/email/resetPassword",
    method: "POST",
    tag: "auth",
} as const;

export const endpointPostAuthGuestLogin = {
    endpoint: "/auth/guest/login",
    method: "POST",
    tag: "auth",
} as const;

export const endpointPostAuthLogout = {
    endpoint: "/auth/logout",
    method: "POST",
    tag: "auth",
} as const;

export const endpointPostAuthValidateSession = {
    endpoint: "/auth/validateSession",
    method: "POST",
    tag: "auth",
} as const;

export const endpointPostAuthSwitchCurrentAccount = {
    endpoint: "/auth/switchCurrentAccount",
    method: "POST",
    tag: "auth",
} as const;

export const endpointPostAuthWalletInit = {
    endpoint: "/auth/wallet/init",
    method: "POST",
    tag: "auth",
} as const;

export const endpointPostAuthWalletComplete = {
    endpoint: "/auth/wallet/complete",
    method: "POST",
    tag: "auth",
} as const;

export const endpointPostAwards = {
    endpoint: "/awards",
    method: "POST",
    tag: "award",
} as const;

export const endpointGetBookmark = {
    endpoint: "/bookmark/:id",
    method: "GET",
    tag: "bookmark",
} as const;

export const endpointPutBookmark = {
    endpoint: "/bookmark/:id",
    method: "PUT",
    tag: "bookmark",
} as const;

export const endpointGetBookmarks = {
    endpoint: "/bookmarks",
    method: "GET",
    tag: "bookmark",
} as const;

export const endpointPostBookmark = {
    endpoint: "/bookmark",
    method: "POST",
    tag: "bookmark",
} as const;

export const endpointGetBookmarkList = {
    endpoint: "/bookmarkList/:id",
    method: "GET",
    tag: "bookmarkList",
} as const;

export const endpointPutBookmarkList = {
    endpoint: "/bookmarkList/:id",
    method: "PUT",
    tag: "bookmarkList",
} as const;

export const endpointGetBookmarkLists = {
    endpoint: "/bookmarkLists",
    method: "GET",
    tag: "bookmarkList",
} as const;

export const endpointPostBookmarkList = {
    endpoint: "/bookmarkList",
    method: "POST",
    tag: "bookmarkList",
} as const;

export const endpointGetChat = {
    endpoint: "/chat/:id",
    method: "GET",
    tag: "chat",
} as const;

export const endpointPutChat = {
    endpoint: "/chat/:id",
    method: "PUT",
    tag: "chat",
} as const;

export const endpointGetChats = {
    endpoint: "/chats",
    method: "GET",
    tag: "chat",
} as const;

export const endpointPostChat = {
    endpoint: "/chat",
    method: "POST",
    tag: "chat",
} as const;

export const endpointGetChatInvite = {
    endpoint: "/chatInvite/:id",
    method: "GET",
    tag: "chatInvite",
} as const;

export const endpointPutChatInvite = {
    endpoint: "/chatInvite/:id",
    method: "PUT",
    tag: "chatInvite",
} as const;

export const endpointGetChatInvites = {
    endpoint: "/chatInvites",
    method: "GET",
    tag: "chatInvite",
} as const;

export const endpointPostChatInvites = {
    endpoint: "/chatInvites",
    method: "POST",
    tag: "chatInvite",
} as const;

export const endpointPutChatInvites = {
    endpoint: "/chatInvites",
    method: "PUT",
    tag: "chatInvite",
} as const;

export const endpointPostChatInvite = {
    endpoint: "/chatInvite",
    method: "POST",
    tag: "chatInvite",
} as const;

export const endpointPutChatInviteAccept = {
    endpoint: "/chatInvite/:id/accept",
    method: "PUT",
    tag: "chatInvite",
} as const;

export const endpointPutChatInviteDecline = {
    endpoint: "/chatInvite/:id/decline",
    method: "PUT",
    tag: "chatInvite",
} as const;

export const endpointGetChatMessage = {
    endpoint: "/chatMessage/:id",
    method: "GET",
    tag: "chatMessage",
} as const;

export const endpointPutChatMessage = {
    endpoint: "/chatMessage/:id",
    method: "PUT",
    tag: "chatMessage",
} as const;

export const endpointGetChatMessages = {
    endpoint: "/chatMessages",
    method: "GET",
    tag: "chatMessage",
} as const;

export const endpointGetChatMessageTree = {
    endpoint: "/chatMessageTree",
    method: "GET",
    tag: "chatMessage",
} as const;

export const endpointPostChatMessage = {
    endpoint: "/chatMessage",
    method: "POST",
    tag: "chatMessage",
} as const;

export const endpointPostRegenerateResponse = {
    endpoint: "/regenerateResponse",
    method: "POST",
    tag: "chatMessage",
} as const;

export const endpointGetAutoFill = {
    endpoint: "/autoFill",
    method: "GET",
    tag: "chatMessage",
} as const;

export const endpointPostStartTask = {
    endpoint: "/startTask",
    method: "POST",
    tag: "chatMessage",
} as const;

export const endpointPostCancelTask = {
    endpoint: "/cancelTask",
    method: "POST",
    tag: "chatMessage",
} as const;

export const endpointGetCheckTaskStatuses = {
    endpoint: "/checkTaskStatuses",
    method: "GET",
    tag: "chatMessage",
} as const;

export const endpointGetChatParticipant = {
    endpoint: "/chatParticipant/:id",
    method: "GET",
    tag: "chatParticipant",
} as const;

export const endpointPutChatParticipant = {
    endpoint: "/chatParticipant/:id",
    method: "PUT",
    tag: "chatParticipant",
} as const;

export const endpointGetChatParticipants = {
    endpoint: "/chatParticipants",
    method: "GET",
    tag: "chatParticipant",
} as const;

export const endpointGetCode = {
    endpoint: "/code/:id",
    method: "GET",
    tag: "code",
} as const;

export const endpointPutCode = {
    endpoint: "/code/:id",
    method: "PUT",
    tag: "code",
} as const;

export const endpointGetCodes = {
    endpoint: "/codes",
    method: "GET",
    tag: "code",
} as const;

export const endpointPostCode = {
    endpoint: "/code",
    method: "POST",
    tag: "code",
} as const;

export const endpointGetCodeVersion = {
    endpoint: "/codeVersion/:id",
    method: "GET",
    tag: "codeVersion",
} as const;

export const endpointPutCodeVersion = {
    endpoint: "/codeVersion/:id",
    method: "PUT",
    tag: "codeVersion",
} as const;

export const endpointGetCodeVersions = {
    endpoint: "/codeVersions",
    method: "GET",
    tag: "codeVersion",
} as const;

export const endpointPostCodeVersion = {
    endpoint: "/codeVersion",
    method: "POST",
    tag: "codeVersion",
} as const;

export const endpointGetComment = {
    endpoint: "/comment/:id",
    method: "GET",
    tag: "comment",
} as const;

export const endpointPutComment = {
    endpoint: "/comment/:id",
    method: "PUT",
    tag: "comment",
} as const;

export const endpointGetComments = {
    endpoint: "/comments",
    method: "GET",
    tag: "comment",
} as const;

export const endpointPostComment = {
    endpoint: "/comment",
    method: "POST",
    tag: "comment",
} as const;

export const endpointPostCopy = {
    endpoint: "/copy",
    method: "POST",
    tag: "copy",
} as const;

export const endpointPostDeleteOne = {
    endpoint: "/deleteOne",
    method: "POST",
    tag: "deleteOneOrMany",
} as const;

export const endpointPostDeleteMany = {
    endpoint: "/deleteMany",
    method: "POST",
    tag: "deleteOneOrMany",
} as const;

export const endpointPostEmail = {
    endpoint: "/email",
    method: "POST",
    tag: "email",
} as const;

export const endpointPostEmailVerification = {
    endpoint: "/email/verification",
    method: "POST",
    tag: "email",
} as const;

export const endpointGetFeedHome = {
    endpoint: "/feed/home",
    method: "GET",
    tag: "feed",
} as const;

export const endpointGetFeedPopular = {
    endpoint: "/feed/popular",
    method: "GET",
    tag: "feed",
} as const;

export const endpointGetFocusMode = {
    endpoint: "/focusMode/:id",
    method: "GET",
    tag: "focusMode",
} as const;

export const endpointPutFocusMode = {
    endpoint: "/focusMode/:id",
    method: "PUT",
    tag: "focusMode",
} as const;

export const endpointGetFocusModes = {
    endpoint: "/focusModes",
    method: "GET",
    tag: "focusMode",
} as const;

export const endpointPostFocusMode = {
    endpoint: "/focusMode",
    method: "POST",
    tag: "focusMode",
} as const;

export const endpointPutFocusModeActive = {
    endpoint: "/focusMode/active",
    method: "PUT",
    tag: "focusMode",
} as const;

export const endpointGetIssue = {
    endpoint: "/issue/:id",
    method: "GET",
    tag: "issue",
} as const;

export const endpointPutIssue = {
    endpoint: "/issue/:id",
    method: "PUT",
    tag: "issue",
} as const;

export const endpointGetIssues = {
    endpoint: "/issues",
    method: "GET",
    tag: "issue",
} as const;

export const endpointPostIssue = {
    endpoint: "/issue",
    method: "POST",
    tag: "issue",
} as const;

export const endpointPutIssueClose = {
    endpoint: "/issue/:id/close",
    method: "PUT",
    tag: "issue",
} as const;

export const endpointGetLabel = {
    endpoint: "/label/:id",
    method: "GET",
    tag: "label",
} as const;

export const endpointPutLabel = {
    endpoint: "/label/:id",
    method: "PUT",
    tag: "label",
} as const;

export const endpointGetLabels = {
    endpoint: "/labels",
    method: "GET",
    tag: "label",
} as const;

export const endpointPostLabel = {
    endpoint: "/label",
    method: "POST",
    tag: "label",
} as const;

export const endpointGetMeeting = {
    endpoint: "/meeting/:id",
    method: "GET",
    tag: "meeting",
} as const;

export const endpointPutMeeting = {
    endpoint: "/meeting/:id",
    method: "PUT",
    tag: "meeting",
} as const;

export const endpointGetMeetings = {
    endpoint: "/meetings",
    method: "GET",
    tag: "meeting",
} as const;

export const endpointPostMeeting = {
    endpoint: "/meeting",
    method: "POST",
    tag: "meeting",
} as const;

export const endpointGetMeetingInvite = {
    endpoint: "/meetingInvite/:id",
    method: "GET",
    tag: "meetingInvite",
} as const;

export const endpointPutMeetingInvite = {
    endpoint: "/meetingInvite/:id",
    method: "PUT",
    tag: "meetingInvite",
} as const;

export const endpointGetMeetingInvites = {
    endpoint: "/meetingInvites",
    method: "GET",
    tag: "meetingInvite",
} as const;

export const endpointPostMeetingInvites = {
    endpoint: "/meetingInvites",
    method: "POST",
    tag: "meetingInvite",
} as const;

export const endpointPutMeetingInvites = {
    endpoint: "/meetingInvites",
    method: "PUT",
    tag: "meetingInvite",
} as const;

export const endpointPostMeetingInvite = {
    endpoint: "/meetingInvite",
    method: "POST",
    tag: "meetingInvite",
} as const;

export const endpointPutMeetingInviteAccept = {
    endpoint: "/meetingInvite/:id/accept",
    method: "PUT",
    tag: "meetingInvite",
} as const;

export const endpointPutMeetingInviteDecline = {
    endpoint: "/meetingInvite/:id/decline",
    method: "PUT",
    tag: "meetingInvite",
} as const;

export const endpointGetMember = {
    endpoint: "/member/:id",
    method: "GET",
    tag: "member",
} as const;

export const endpointPutMember = {
    endpoint: "/member/:id",
    method: "PUT",
    tag: "member",
} as const;

export const endpointGetMembers = {
    endpoint: "/members",
    method: "GET",
    tag: "member",
} as const;

export const endpointGetMemberInvite = {
    endpoint: "/memberInvite/:id",
    method: "GET",
    tag: "memberInvite",
} as const;

export const endpointPutMemberInvite = {
    endpoint: "/memberInvite/:id",
    method: "PUT",
    tag: "memberInvite",
} as const;

export const endpointGetMemberInvites = {
    endpoint: "/memberInvites",
    method: "GET",
    tag: "memberInvite",
} as const;

export const endpointPostMemberInvites = {
    endpoint: "/memberInvites",
    method: "POST",
    tag: "memberInvite",
} as const;

export const endpointPutMemberInvites = {
    endpoint: "/memberInvites",
    method: "PUT",
    tag: "memberInvite",
} as const;

export const endpointPostMemberInvite = {
    endpoint: "/memberInvite",
    method: "POST",
    tag: "memberInvite",
} as const;

export const endpointPutMemberInviteAccept = {
    endpoint: "/memberInvite/:id/accept",
    method: "PUT",
    tag: "memberInvite",
} as const;

export const endpointPutMemberInviteDecline = {
    endpoint: "/memberInvite/:id/decline",
    method: "PUT",
    tag: "memberInvite",
} as const;

export const endpointPostNode = {
    endpoint: "/node",
    method: "POST",
    tag: "node",
} as const;

export const endpointPutNode = {
    endpoint: "/node/:id",
    method: "PUT",
    tag: "node",
} as const;

export const endpointGetNote = {
    endpoint: "/note/:id",
    method: "GET",
    tag: "note",
} as const;

export const endpointPutNote = {
    endpoint: "/note/:id",
    method: "PUT",
    tag: "note",
} as const;

export const endpointGetNotes = {
    endpoint: "/notes",
    method: "GET",
    tag: "note",
} as const;

export const endpointPostNote = {
    endpoint: "/note",
    method: "POST",
    tag: "note",
} as const;

export const endpointGetNoteVersion = {
    endpoint: "/noteVersion/:id",
    method: "GET",
    tag: "noteVersion",
} as const;

export const endpointPutNoteVersion = {
    endpoint: "/noteVersion/:id",
    method: "PUT",
    tag: "noteVersion",
} as const;

export const endpointGetNoteVersions = {
    endpoint: "/noteVersions",
    method: "GET",
    tag: "noteVersion",
} as const;

export const endpointPostNoteVersion = {
    endpoint: "/noteVersion",
    method: "POST",
    tag: "noteVersion",
} as const;

export const endpointGetNotification = {
    endpoint: "/notification/:id",
    method: "GET",
    tag: "notification",
} as const;

export const endpointPutNotification = {
    endpoint: "/notification/:id",
    method: "PUT",
    tag: "notification",
} as const;

export const endpointGetNotifications = {
    endpoint: "/notifications",
    method: "GET",
    tag: "notification",
} as const;

export const endpointPutNotificationsMarkAllAsRead = {
    endpoint: "/notifications/markAllAsRead",
    method: "PUT",
    tag: "notification",
} as const;

export const endpointGetNotificationSettings = {
    endpoint: "/notificationSettings",
    method: "GET",
    tag: "notification",
} as const;

export const endpointPutNotificationSettings = {
    endpoint: "/notificationSettings",
    method: "PUT",
    tag: "notification",
} as const;

export const endpointGetNotificationSubscription = {
    endpoint: "/notificationSubscription/:id",
    method: "GET",
    tag: "notificationSubscription",
} as const;

export const endpointPutNotificationSubscription = {
    endpoint: "/notificationSubscription/:id",
    method: "PUT",
    tag: "notificationSubscription",
} as const;

export const endpointGetNotificationSubscriptions = {
    endpoint: "/notificationSubscriptions",
    method: "GET",
    tag: "notificationSubscription",
} as const;

export const endpointPostNotificationSubscription = {
    endpoint: "/notificationSubscription",
    method: "POST",
    tag: "notificationSubscription",
} as const;

export const endpointPostPhone = {
    endpoint: "/phone",
    method: "POST",
    tag: "phone",
} as const;

export const endpointPostPhoneVerificationText = {
    endpoint: "/phone/verificationText",
    method: "POST",
    tag: "phone",
} as const;

export const endpointPostPhoneValidateText = {
    endpoint: "/phone/validateText",
    method: "POST",
    tag: "phone",
} as const;

export const endpointGetPost = {
    endpoint: "/post/:id",
    method: "GET",
    tag: "post",
} as const;

export const endpointPutPost = {
    endpoint: "/post/:id",
    method: "PUT",
    tag: "post",
} as const;

export const endpointGetPosts = {
    endpoint: "/posts",
    method: "GET",
    tag: "post",
} as const;

export const endpointPostPost = {
    endpoint: "/post",
    method: "POST",
    tag: "post",
} as const;

export const endpointGetProject = {
    endpoint: "/project/:id",
    method: "GET",
    tag: "project",
} as const;

export const endpointPutProject = {
    endpoint: "/project/:id",
    method: "PUT",
    tag: "project",
} as const;

export const endpointGetProjects = {
    endpoint: "/projects",
    method: "GET",
    tag: "project",
} as const;

export const endpointPostProject = {
    endpoint: "/project",
    method: "POST",
    tag: "project",
} as const;

export const endpointGetProjectVersion = {
    endpoint: "/projectVersion/:id",
    method: "GET",
    tag: "projectVersion",
} as const;

export const endpointPutProjectVersion = {
    endpoint: "/projectVersion/:id",
    method: "PUT",
    tag: "projectVersion",
} as const;

export const endpointGetProjectVersions = {
    endpoint: "/projectVersions",
    method: "GET",
    tag: "projectVersion",
} as const;

export const endpointGetProjectVersionContents = {
    endpoint: "/projectVersionContents",
    method: "GET",
    tag: "projectVersion",
} as const;

export const endpointPostProjectVersion = {
    endpoint: "/projectVersion",
    method: "POST",
    tag: "projectVersion",
} as const;

export const endpointGetProjectVersionDirectory = {
    endpoint: "/projectVersionDirectory/:id",
    method: "GET",
    tag: "projectVersionDirectory",
} as const;

export const endpointPutProjectVersionDirectory = {
    endpoint: "/projectVersionDirectory/:id",
    method: "PUT",
    tag: "projectVersionDirectory",
} as const;

export const endpointGetProjectVersionDirectories = {
    endpoint: "/projectVersionDirectories",
    method: "GET",
    tag: "projectVersionDirectory",
} as const;

export const endpointPostProjectVersionDirectory = {
    endpoint: "/projectVersionDirectory",
    method: "POST",
    tag: "projectVersionDirectory",
} as const;

export const endpointGetPullRequest = {
    endpoint: "/pullRequest/:id",
    method: "GET",
    tag: "pullRequest",
} as const;

export const endpointPutPullRequest = {
    endpoint: "/pullRequest/:id",
    method: "PUT",
    tag: "pullRequest",
} as const;

export const endpointGetPullRequests = {
    endpoint: "/pullRequests",
    method: "GET",
    tag: "pullRequest",
} as const;

export const endpointPostPullRequest = {
    endpoint: "/pullRequest",
    method: "POST",
    tag: "pullRequest",
} as const;

export const endpointPostPushDevice = {
    endpoint: "/pushDevice",
    method: "POST",
    tag: "pushDevice",
} as const;

export const endpointGetPushDevices = {
    endpoint: "/pushDevices",
    method: "GET",
    tag: "pushDevice",
} as const;

export const endpointPutPushDevice = {
    endpoint: "/pushDevice/:id",
    method: "PUT",
    tag: "pushDevice",
} as const;

export const endpointGetQuestion = {
    endpoint: "/question/:id",
    method: "GET",
    tag: "question",
} as const;

export const endpointPutQuestion = {
    endpoint: "/question/:id",
    method: "PUT",
    tag: "question",
} as const;

export const endpointGetQuestions = {
    endpoint: "/questions",
    method: "GET",
    tag: "question",
} as const;

export const endpointPostQuestion = {
    endpoint: "/question",
    method: "POST",
    tag: "question",
} as const;

export const endpointGetQuestionAnswer = {
    endpoint: "/questionAnswer/:id",
    method: "GET",
    tag: "questionAnswer",
} as const;

export const endpointPutQuestionAnswer = {
    endpoint: "/questionAnswer/:id",
    method: "PUT",
    tag: "questionAnswer",
} as const;

export const endpointGetQuestionAnswers = {
    endpoint: "/questionAnswers",
    method: "GET",
    tag: "questionAnswer",
} as const;

export const endpointPostQuestionAnswer = {
    endpoint: "/questionAnswer",
    method: "POST",
    tag: "questionAnswer",
} as const;

export const endpointPutQuestionAnswerAccept = {
    endpoint: "/questionAnswer/:id/accept",
    method: "PUT",
    tag: "questionAnswer",
} as const;

export const endpointGetQuiz = {
    endpoint: "/quiz/:id",
    method: "GET",
    tag: "quiz",
} as const;

export const endpointPutQuiz = {
    endpoint: "/quiz/:id",
    method: "PUT",
    tag: "quiz",
} as const;

export const endpointGetQuizzes = {
    endpoint: "/quizzes",
    method: "GET",
    tag: "quiz",
} as const;

export const endpointPostQuiz = {
    endpoint: "/quiz",
    method: "POST",
    tag: "quiz",
} as const;

export const endpointGetQuizAttempt = {
    endpoint: "/quizAttempt/:id",
    method: "GET",
    tag: "quizAttempt",
} as const;

export const endpointPutQuizAttempt = {
    endpoint: "/quizAttempt/:id",
    method: "PUT",
    tag: "quizAttempt",
} as const;

export const endpointGetQuizAttempts = {
    endpoint: "/quizAttempts",
    method: "GET",
    tag: "quizAttempt",
} as const;

export const endpointPostQuizAttempt = {
    endpoint: "/quizAttempt",
    method: "POST",
    tag: "quizAttempt",
} as const;

export const endpointGetQuizQuestion = {
    endpoint: "/quizQuestion/:id",
    method: "GET",
    tag: "quizQuestion",
} as const;

export const endpointGetQuizQuestions = {
    endpoint: "/quizQuestions",
    method: "GET",
    tag: "quizQuestion",
} as const;

export const endpointGetQuizQuestionResponse = {
    endpoint: "/quizQuestionResponse/:id",
    method: "GET",
    tag: "quizQuestionResponse",
} as const;

export const endpointGetQuizQuestionResponses = {
    endpoint: "/quizQuestionResponses",
    method: "GET",
    tag: "quizQuestionResponse",
} as const;

export const endpointGetReactions = {
    endpoint: "/reactions",
    method: "GET",
    tag: "reaction",
} as const;

export const endpointPostReact = {
    endpoint: "/react",
    method: "POST",
    tag: "reaction",
} as const;

export const endpointGetReminder = {
    endpoint: "/reminder/:id",
    method: "GET",
    tag: "reminder",
} as const;

export const endpointPutReminder = {
    endpoint: "/reminder/:id",
    method: "PUT",
    tag: "reminder",
} as const;

export const endpointGetReminders = {
    endpoint: "/reminders",
    method: "GET",
    tag: "reminder",
} as const;

export const endpointPostReminder = {
    endpoint: "/reminder",
    method: "POST",
    tag: "reminder",
} as const;

export const endpointPostReminderList = {
    endpoint: "/reminderList",
    method: "POST",
    tag: "reminderList",
} as const;

export const endpointPutReminderList = {
    endpoint: "/reminderList/:id",
    method: "PUT",
    tag: "reminderList",
} as const;

export const endpointGetReport = {
    endpoint: "/report/:id",
    method: "GET",
    tag: "report",
} as const;

export const endpointPutReport = {
    endpoint: "/report/:id",
    method: "PUT",
    tag: "report",
} as const;

export const endpointGetReports = {
    endpoint: "/reports",
    method: "GET",
    tag: "report",
} as const;

export const endpointPostReport = {
    endpoint: "/report",
    method: "POST",
    tag: "report",
} as const;

export const endpointGetReportResponse = {
    endpoint: "/reportResponse/:id",
    method: "GET",
    tag: "reportResponse",
} as const;

export const endpointPutReportResponse = {
    endpoint: "/reportResponse/:id",
    method: "PUT",
    tag: "reportResponse",
} as const;

export const endpointGetReportResponses = {
    endpoint: "/reportResponses",
    method: "GET",
    tag: "reportResponse",
} as const;

export const endpointPostReportResponse = {
    endpoint: "/reportResponse",
    method: "POST",
    tag: "reportResponse",
} as const;

export const endpointGetReputationHistory = {
    endpoint: "/reputationHistory/:id",
    method: "GET",
    tag: "reputationHistory",
} as const;

export const endpointGetReputationHistories = {
    endpoint: "/reputationHistories",
    method: "GET",
    tag: "reputationHistory",
} as const;

export const endpointGetResource = {
    endpoint: "/resource/:id",
    method: "GET",
    tag: "resource",
} as const;

export const endpointPutResource = {
    endpoint: "/resource/:id",
    method: "PUT",
    tag: "resource",
} as const;

export const endpointGetResources = {
    endpoint: "/resources",
    method: "GET",
    tag: "resource",
} as const;

export const endpointPostResource = {
    endpoint: "/resource",
    method: "POST",
    tag: "resource",
} as const;

export const endpointGetResourceList = {
    endpoint: "/resourceList/:id",
    method: "GET",
    tag: "resourceList",
} as const;

export const endpointPutResourceList = {
    endpoint: "/resourceList/:id",
    method: "PUT",
    tag: "resourceList",
} as const;

export const endpointGetResourceLists = {
    endpoint: "/resourceLists",
    method: "GET",
    tag: "resourceList",
} as const;

export const endpointPostResourceList = {
    endpoint: "/resourceList",
    method: "POST",
    tag: "resourceList",
} as const;

export const endpointGetRole = {
    endpoint: "/role/:id",
    method: "GET",
    tag: "role",
} as const;

export const endpointPutRole = {
    endpoint: "/role/:id",
    method: "PUT",
    tag: "role",
} as const;

export const endpointGetRoles = {
    endpoint: "/roles",
    method: "GET",
    tag: "role",
} as const;

export const endpointPostRole = {
    endpoint: "/role",
    method: "POST",
    tag: "role",
} as const;

export const endpointGetRoutine = {
    endpoint: "/routine/:id",
    method: "GET",
    tag: "routine",
} as const;

export const endpointPutRoutine = {
    endpoint: "/routine/:id",
    method: "PUT",
    tag: "routine",
} as const;

export const endpointGetRoutines = {
    endpoint: "/routines",
    method: "GET",
    tag: "routine",
} as const;

export const endpointPostRoutine = {
    endpoint: "/routine",
    method: "POST",
    tag: "routine",
} as const;

export const endpointGetRoutineVersion = {
    endpoint: "/routineVersion/:id",
    method: "GET",
    tag: "routineVersion",
} as const;

export const endpointPutRoutineVersion = {
    endpoint: "/routineVersion/:id",
    method: "PUT",
    tag: "routineVersion",
} as const;

export const endpointGetRoutineVersions = {
    endpoint: "/routineVersions",
    method: "GET",
    tag: "routineVersion",
} as const;

export const endpointPostRoutineVersion = {
    endpoint: "/routineVersion",
    method: "POST",
    tag: "routineVersion",
} as const;

export const endpointGetRunProject = {
    endpoint: "/runProject/:id",
    method: "GET",
    tag: "runProject",
} as const;

export const endpointPutRunProject = {
    endpoint: "/runProject/:id",
    method: "PUT",
    tag: "runProject",
} as const;

export const endpointGetRunProjects = {
    endpoint: "/runProjects",
    method: "GET",
    tag: "runProject",
} as const;

export const endpointPostRunProject = {
    endpoint: "/runProject",
    method: "POST",
    tag: "runProject",
} as const;

export const endpointDeleteRunProjectDeleteAll = {
    endpoint: "/runProject/deleteAll",
    method: "DELETE",
    tag: "runProject",
} as const;

export const endpointPutRunProjectComplete = {
    endpoint: "/runProject/:id/complete",
    method: "PUT",
    tag: "runProject",
} as const;

export const endpointPutRunProjectCancel = {
    endpoint: "/runProject/:id/cancel",
    method: "PUT",
    tag: "runProject",
} as const;

export const endpointGetRunRoutine = {
    endpoint: "/runRoutine/:id",
    method: "GET",
    tag: "runRoutine",
} as const;

export const endpointPutRunRoutine = {
    endpoint: "/runRoutine/:id",
    method: "PUT",
    tag: "runRoutine",
} as const;

export const endpointGetRunRoutines = {
    endpoint: "/runRoutines",
    method: "GET",
    tag: "runRoutine",
} as const;

export const endpointPostRunRoutine = {
    endpoint: "/runRoutine",
    method: "POST",
    tag: "runRoutine",
} as const;

export const endpointDeleteRunRoutineDeleteAll = {
    endpoint: "/runRoutine/deleteAll",
    method: "DELETE",
    tag: "runRoutine",
} as const;

export const endpointPutRunRoutineComplete = {
    endpoint: "/runRoutine/:id/complete",
    method: "PUT",
    tag: "runRoutine",
} as const;

export const endpointPutRunRoutineCancel = {
    endpoint: "/runRoutine/:id/cancel",
    method: "PUT",
    tag: "runRoutine",
} as const;

export const endpointGetRunRoutineInputs = {
    endpoint: "/runRoutineInputs",
    method: "GET",
    tag: "runRoutineInput",
} as const;

export const endpointGetSchedule = {
    endpoint: "/schedule/:id",
    method: "GET",
    tag: "schedule",
} as const;

export const endpointPutSchedule = {
    endpoint: "/schedule/:id",
    method: "PUT",
    tag: "schedule",
} as const;

export const endpointGetSchedules = {
    endpoint: "/schedules",
    method: "GET",
    tag: "schedule",
} as const;

export const endpointPostSchedule = {
    endpoint: "/schedule",
    method: "POST",
    tag: "schedule",
} as const;

export const endpointGetScheduleException = {
    endpoint: "/scheduleException/:id",
    method: "GET",
    tag: "scheduleException",
} as const;

export const endpointPutScheduleException = {
    endpoint: "/scheduleException/:id",
    method: "PUT",
    tag: "scheduleException",
} as const;

export const endpointGetScheduleExceptions = {
    endpoint: "/scheduleExceptions",
    method: "GET",
    tag: "scheduleException",
} as const;

export const endpointPostScheduleException = {
    endpoint: "/scheduleException",
    method: "POST",
    tag: "scheduleException",
} as const;

export const endpointGetScheduleRecurrence = {
    endpoint: "/scheduleRecurrence/:id",
    method: "GET",
    tag: "scheduleRecurrence",
} as const;

export const endpointPutScheduleRecurrence = {
    endpoint: "/scheduleRecurrence/:id",
    method: "PUT",
    tag: "scheduleRecurrence",
} as const;

export const endpointGetScheduleRecurrences = {
    endpoint: "/scheduleRecurrences",
    method: "GET",
    tag: "scheduleRecurrence",
} as const;

export const endpointPostScheduleRecurrence = {
    endpoint: "/scheduleRecurrence",
    method: "POST",
    tag: "scheduleRecurrence",
} as const;

export const endpointGetStandard = {
    endpoint: "/standard/:id",
    method: "GET",
    tag: "standard",
} as const;

export const endpointPutStandard = {
    endpoint: "/standard/:id",
    method: "PUT",
    tag: "standard",
} as const;

export const endpointGetStandards = {
    endpoint: "/standards",
    method: "GET",
    tag: "standard",
} as const;

export const endpointPostStandard = {
    endpoint: "/standard",
    method: "POST",
    tag: "standard",
} as const;

export const endpointGetStandardVersion = {
    endpoint: "/standardVersion/:id",
    method: "GET",
    tag: "standardVersion",
} as const;

export const endpointPutStandardVersion = {
    endpoint: "/standardVersion/:id",
    method: "PUT",
    tag: "standardVersion",
} as const;

export const endpointGetStandardVersions = {
    endpoint: "/standardVersions",
    method: "GET",
    tag: "standardVersion",
} as const;

export const endpointPostStandardVersion = {
    endpoint: "/standardVersion",
    method: "POST",
    tag: "standardVersion",
} as const;

export const endpointGetStatsApi = {
    endpoint: "/stats/api",
    method: "GET",
    tag: "statsApi",
} as const;

export const endpointGetStatsCode = {
    endpoint: "/stats/code",
    method: "GET",
    tag: "statsCode",
} as const;

export const endpointGetStatsProject = {
    endpoint: "/stats/project",
    method: "GET",
    tag: "statsProject",
} as const;

export const endpointGetStatsQuiz = {
    endpoint: "/stats/quiz",
    method: "GET",
    tag: "statsQuiz",
} as const;

export const endpointGetStatsRoutine = {
    endpoint: "/stats/routine",
    method: "GET",
    tag: "statsRoutine",
} as const;

export const endpointGetStatsSite = {
    endpoint: "/stats/site",
    method: "GET",
    tag: "statsSite",
} as const;

export const endpointGetStatsStandard = {
    endpoint: "/stats/standard",
    method: "GET",
    tag: "statsStandard",
} as const;

export const endpointGetStatsTeam = {
    endpoint: "/stats/team",
    method: "GET",
    tag: "statsTeam",
} as const;

export const endpointGetStatsUser = {
    endpoint: "/stats/user",
    method: "GET",
    tag: "statsUser",
} as const;

export const endpointGetTag = {
    endpoint: "/tag/:id",
    method: "GET",
    tag: "tag",
} as const;

export const endpointPutTag = {
    endpoint: "/tag/:id",
    method: "PUT",
    tag: "tag",
} as const;

export const endpointGetTags = {
    endpoint: "/tags",
    method: "GET",
    tag: "tag",
} as const;

export const endpointPostTag = {
    endpoint: "/tag",
    method: "POST",
    tag: "tag",
} as const;

export const endpointGetTeam = {
    endpoint: "/team/:id",
    method: "GET",
    tag: "team",
} as const;

export const endpointPutTeam = {
    endpoint: "/team/:id",
    method: "PUT",
    tag: "team",
} as const;

export const endpointGetTeams = {
    endpoint: "/teams",
    method: "GET",
    tag: "team",
} as const;

export const endpointPostTeam = {
    endpoint: "/team",
    method: "POST",
    tag: "team",
} as const;

export const endpointGetTransfer = {
    endpoint: "/transfer/:id",
    method: "GET",
    tag: "transfer",
} as const;

export const endpointPutTransfer = {
    endpoint: "/transfer/:id",
    method: "PUT",
    tag: "transfer",
} as const;

export const endpointGetTransfers = {
    endpoint: "/transfers",
    method: "GET",
    tag: "transfer",
} as const;

export const endpointPostTransferRequestSend = {
    endpoint: "/transfer/requestSend",
    method: "POST",
    tag: "transfer",
} as const;

export const endpointPostTransferRequestReceive = {
    endpoint: "/transfer/requestReceive",
    method: "POST",
    tag: "transfer",
} as const;

export const endpointPutTransferCancel = {
    endpoint: "/transfer/:id/cancel",
    method: "PUT",
    tag: "transfer",
} as const;

export const endpointPutTransferAccept = {
    endpoint: "/transfer/:id/accept",
    method: "PUT",
    tag: "transfer",
} as const;

export const endpointPutTransferDeny = {
    endpoint: "/transfer/deny",
    method: "PUT",
    tag: "transfer",
} as const;

export const endpointGetTranslate = {
    endpoint: "/translate",
    method: "GET",
    tag: "translate",
} as const;

export const endpointGetUnionsProjectOrRoutines = {
    endpoint: "/unions/projectOrRoutines",
    method: "GET",
    tag: "unions",
} as const;

export const endpointGetUnionsProjectOrTeams = {
    endpoint: "/unions/projectOrTeams",
    method: "GET",
    tag: "unions",
} as const;

export const endpointGetUnionsRunProjectOrRunRoutines = {
    endpoint: "/unions/runProjectOrRunRoutines",
    method: "GET",
    tag: "unions",
} as const;

export const endpointPutBot = {
    endpoint: "/bot/:id",
    method: "PUT",
    tag: "user",
} as const;

export const endpointPostBot = {
    endpoint: "/bot",
    method: "POST",
    tag: "user",
} as const;

export const endpointGetProfile = {
    endpoint: "/profile",
    method: "GET",
    tag: "user",
} as const;

export const endpointPutProfile = {
    endpoint: "/profile",
    method: "PUT",
    tag: "user",
} as const;

export const endpointGetUser = {
    endpoint: "/user/:id",
    method: "GET",
    tag: "user",
} as const;

export const endpointGetUsers = {
    endpoint: "/users",
    method: "GET",
    tag: "user",
} as const;

export const endpointPutProfileEmail = {
    endpoint: "/profile/email",
    method: "PUT",
    tag: "user",
} as const;

export const endpointDeleteUser = {
    endpoint: "/user",
    method: "DELETE",
    tag: "user",
} as const;

export const endpointGetViews = {
    endpoint: "/views",
    method: "GET",
    tag: "view",
} as const;

export const endpointPutWallet = {
    endpoint: "/wallet/:id",
    method: "PUT",
    tag: "wallet",
} as const;

