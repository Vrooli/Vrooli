import { toMutation, toQuery, toSearch } from "./utils";

export const endpoints = {
    api: async () => {
        const { api: apiPartial } = await import("./partial/api");
        return {
            findOne: toQuery("api", "FindByIdInput", apiPartial, "full"),
            findMany: toQuery("apis", "ApiSearchInput", ...(await toSearch(apiPartial))),
            create: toMutation("apiCreate", "ApiCreateInput", apiPartial, "full"),
            update: toMutation("apiUpdate", "ApiUpdateInput", apiPartial, "full"),
        };
    },
    apiKey: async () => {
        const { apiKey: apiKeyPartial } = await import("./partial/apiKey");
        const { success: successPartial } = await import("./partial/success");
        return {
            create: toMutation("apiKeyCreate", "ApiKeyCreateInput", apiKeyPartial, "full"),
            update: toMutation("apiKeyUpdate", "ApiKeyUpdateInput", apiKeyPartial, "full"),
            deleteOne: toMutation("apiKeyDeleteOne", "ApiKeyDeleteOneInput", successPartial, "full"),
            validate: toMutation("apiKeyValidate", "ApiKeyValidateInput", apiKeyPartial, "full"),
        };
    },
    apiVersion: async () => {
        const { apiVersion: apiVersionPartial } = await import("./partial/apiVersion");
        return {
            findOne: toQuery("apiVersion", "FindVersionInput", apiVersionPartial, "full"),
            findMany: toQuery("apiVersions", "ApiVersionSearchInput", ...(await toSearch(apiVersionPartial))),
            create: toMutation("apiVersionCreate", "ApiVersionCreateInput", apiVersionPartial, "full"),
            update: toMutation("apiVersionUpdate", "ApiVersionUpdateInput", apiVersionPartial, "full"),
        };
    },
    auth: async () => {
        const { session: sessionPartial } = await import("./partial/session");
        const { success: successPartial } = await import("./partial/success");
        const { walletComplete: walletCompletePartial } = await import("./partial/walletComplete");
        return {
            emailLogIn: toMutation("emailLogIn", "EmailLogInInput", sessionPartial, "full"),
            emailSignUp: toMutation("emailSignUp", "EmailSignUpInput", sessionPartial, "full"),
            emailRequestPasswordChange: toMutation("emailRequestPasswordChange", "EmailRequestPasswordChangeInput", successPartial, "full"),
            emailResetPassword: toMutation("emailResetPassword", "EmailResetPasswordInput", sessionPartial, "full"),
            guestLogIn: toMutation("guestLogIn", null, sessionPartial, "full"),
            logOut: toMutation("logOut", "LogOutInput", sessionPartial, "full"),
            validateSession: toMutation("validateSession", "ValidateSessionInput", sessionPartial, "full"),
            switchCurrentAccount: toMutation("switchCurrentAccount", "SwitchCurrentAccountInput", sessionPartial, "full"),
            walletInit: toMutation("walletInit", "WalletInitInput"),
            walletComplete: toMutation("walletComplete", "WalletCompleteInput", walletCompletePartial, "full"),
        };
    },
    award: async () => {
        const { award: awardPartial } = await import("./partial/award");
        return {
            findMany: toQuery("awards", "AwardSearchInput", ...(await toSearch(awardPartial))),
        };
    },
    bookmark: async () => {
        const { bookmark: bookmarkPartial } = await import("./partial/bookmark");
        return {
            findOne: toQuery("bookmark", "FindByIdInput", bookmarkPartial, "full"),
            findMany: toQuery("bookmarks", "BookmarkSearchInput", ...(await toSearch(bookmarkPartial))),
            create: toMutation("bookmarkCreate", "BookmarkCreateInput", bookmarkPartial, "full"),
            update: toMutation("bookmarkUpdate", "BookmarkUpdateInput", bookmarkPartial, "full"),
        };
    },
    bookmarkList: async () => {
        const { bookmarkList: bookmarkListPartial } = await import("./partial/bookmarkList");
        return {
            findOne: toQuery("bookmarkList", "FindByIdInput", bookmarkListPartial, "full"),
            findMany: toQuery("bookmarkLists", "BookmarkListSearchInput", ...(await toSearch(bookmarkListPartial))),
            create: toMutation("bookmarkListCreate", "BookmarkListCreateInput", bookmarkListPartial, "full"),
            update: toMutation("bookmarkListUpdate", "BookmarkListUpdateInput", bookmarkListPartial, "full"),
        };
    },
    chat: async () => {
        const { chat: chatPartial } = await import("./partial/chat");
        return {
            findOne: toQuery("chat", "FindByIdInput", chatPartial, "full"),
            findMany: toQuery("chats", "ChatSearchInput", ...(await toSearch(chatPartial))),
            create: toMutation("chatCreate", "ChatCreateInput", chatPartial, "full"),
            update: toMutation("chatUpdate", "ChatUpdateInput", chatPartial, "full"),
        };
    },
    chatInvite: async () => {
        const { chatInvite: chatInvitePartial } = await import("./partial/chatInvite");
        return {
            findOne: toQuery("chatInvite", "FindByIdInput", chatInvitePartial, "full"),
            findMany: toQuery("chatInvites", "ChatInviteSearchInput", ...(await toSearch(chatInvitePartial))),
            createOne: toMutation("chatInviteCreate", "ChatInviteCreateInput", chatInvitePartial, "full"),
            createMany: toMutation("chatInvitesCreate", "[ChatInviteCreateInput!]", chatInvitePartial, "full"),
            updateOne: toMutation("chatInviteUpdate", "ChatInviteUpdateInput", chatInvitePartial, "full"),
            updateMany: toMutation("chatInvitesUpdate", "[ChatInviteUpdateInput!]", chatInvitePartial, "full"),
            accept: toMutation("chatInviteAccept", "FindByIdInput", chatInvitePartial, "full"),
            decline: toMutation("chatInviteDecline", "FindByIdInput", chatInvitePartial, "full"),
        };
    },
    chatMessage: async () => {
        const { chatMessage: chatMessagePartial, chatMessageSearchTreeResult: chatMessageSearchTreeResultPartial } = await import("./partial/chatMessage");
        return {
            findOne: toQuery("chatMessage", "FindByIdInput", chatMessagePartial, "full"),
            findMany: toQuery("chatMessages", "ChatMessageSearchInput", ...(await toSearch(chatMessagePartial))),
            findTree: toQuery("chatMessageTree", "ChatMessageSearchTreeInput", chatMessageSearchTreeResultPartial, "full"),
            create: toMutation("chatMessageCreate", "ChatMessageCreateInput", chatMessagePartial, "full"),
            update: toMutation("chatMessageUpdate", "ChatMessageUpdateInput", chatMessagePartial, "full"),
        };
    },
    chatParticipant: async () => {
        const { chatParticipant: chatParticipantPartial } = await import("./partial/chatParticipant");
        return {
            findOne: toQuery("chatParticipant", "FindByIdInput", chatParticipantPartial, "full"),
            findMany: toQuery("chatParticipants", "ChatParticipantSearchInput", ...(await toSearch(chatParticipantPartial))),
            update: toMutation("chatParticipantUpdate", "ChatParticipantUpdateInput", chatParticipantPartial, "full"),
        };
    },
    comment: async () => {
        const { comment: commentPartial, commentSearchResult: commentSearchResultPartial } = await import("./partial/comment");
        return {
            findOne: toQuery("comment", "FindByIdInput", commentPartial, "full"),
            findMany: toQuery("comments", "CommentSearchInput", commentSearchResultPartial, "full"),
            create: toMutation("commentCreate", "CommentCreateInput", commentPartial, "full"),
            update: toMutation("commentUpdate", "CommentUpdateInput", commentPartial, "full"),
        };
    },
    copy: async () => {
        const { copyResult: copyResultPartial } = await import("./partial/copyResult");
        return {
            copy: toMutation("copy", "CopyInput", copyResultPartial, "full"),
        };
    },
    deleteOneOrMany: async () => {
        const { success: successPartial } = await import("./partial/success");
        const { count: countPartial } = await import("./partial/count");
        return {
            deleteOne: toMutation("deleteOne", "DeleteOneInput", successPartial, "full"),
            deleteMany: toMutation("deleteMany", "DeleteManyInput", countPartial, "full"),
        };
    },
    email: async () => {
        const { email: emailPartial } = await import("./partial/email");
        const { success: successPartial } = await import("./partial/success");
        return {
            create: toMutation("emailCreate", "EmailCreateInput", emailPartial, "full"),
            verify: toMutation("sendVerificationEmail", "SendVerificationEmailInput", successPartial, "full"),
        };
    },
    feed: async () => {
        const { homeResult: homeResultPartial } = await import("./partial/feed");
        return {
            home: toQuery("home", "HomeInput", homeResultPartial, "list"),
        };
    },
    issue: async () => {
        const { issue: issuePartial } = await import("./partial/issue");
        return {
            findOne: toQuery("issue", "FindByIdInput", issuePartial, "full"),
            findMany: toQuery("issues", "IssueSearchInput", ...(await toSearch(issuePartial))),
            create: toMutation("issueCreate", "IssueCreateInput", issuePartial, "full"),
            update: toMutation("issueUpdate", "IssueUpdateInput", issuePartial, "full"),
            close: toMutation("issueClose", "IssueCloseInput", issuePartial, "full"),
        };
    },
    focusMode: async () => {
        const { activeFocusMode: activeFocusModePartial } = await import("./partial/activeFocusMode");
        const { focusMode: focusModePartial } = await import("./partial/focusMode");
        return {
            findOne: toQuery("focusMode", "FindByIdInput", focusModePartial, "full"),
            findMany: toQuery("focusModes", "FocusModeSearchInput", ...(await toSearch(focusModePartial))),
            create: toMutation("focusModeCreate", "FocusModeCreateInput", focusModePartial, "full"),
            update: toMutation("focusModeUpdate", "FocusModeUpdateInput", focusModePartial, "full"),
            setActive: toMutation("setActiveFocusMode", "SetActiveFocusModeInput", activeFocusModePartial, "full"),
        };
    },
    label: async () => {
        const { label: labelPartial } = await import("./partial/label");
        return {
            findOne: toQuery("label", "FindByIdInput", labelPartial, "full"),
            findMany: toQuery("labels", "LabelSearchInput", ...(await toSearch(labelPartial))),
            create: toMutation("labelCreate", "LabelCreateInput", labelPartial, "full"),
            update: toMutation("labelUpdate", "LabelUpdateInput", labelPartial, "full"),
        };
    },
    meeting: async () => {
        const { meeting: meetingPartial } = await import("./partial/meeting");
        return {
            findOne: toQuery("meeting", "FindByIdInput", meetingPartial, "full"),
            findMany: toQuery("meetings", "MeetingSearchInput", ...(await toSearch(meetingPartial))),
            create: toMutation("meetingCreate", "MeetingCreateInput", meetingPartial, "full"),
            update: toMutation("meetingUpdate", "MeetingUpdateInput", meetingPartial, "full"),
        };
    },
    meetingInvite: async () => {
        const { meetingInvite: meetingInvitePartial } = await import("./partial/meetingInvite");
        return {
            findOne: toQuery("meetingInvite", "FindByIdInput", meetingInvitePartial, "full"),
            findMany: toQuery("meetingInvites", "MeetingInviteSearchInput", ...(await toSearch(meetingInvitePartial))),
            createOne: toMutation("meetingInviteCreate", "MeetingInviteCreateInput", meetingInvitePartial, "full"),
            createMany: toMutation("meetingInvitesCreate", "[MeetingInviteCreateInput!]", meetingInvitePartial, "full"),
            updateOne: toMutation("meetingInviteUpdate", "MeetingInviteUpdateInput", meetingInvitePartial, "full"),
            updateMany: toMutation("meetingInvitesUpdate", "[MeetingInviteUpdateInput!]", meetingInvitePartial, "full"),
            accept: toMutation("meetingInviteAccept", "FindByIdInput", meetingInvitePartial, "full"),
            decline: toMutation("meetingInviteDecline", "FindByIdInput", meetingInvitePartial, "full"),
        };
    },
    member: async () => {
        const { member: memberPartial } = await import("./partial/member");
        return {
            findOne: toQuery("member", "FindByIdInput", memberPartial, "full"),
            findMany: toQuery("members", "MemberSearchInput", ...(await toSearch(memberPartial))),
            update: toMutation("memberUpdate", "MemberUpdateInput", memberPartial, "full"),
        };
    },
    memberInvite: async () => {
        const { memberInvite: memberInvitePartial } = await import("./partial/memberInvite");
        return {
            findOne: toQuery("memberInvite", "FindByIdInput", memberInvitePartial, "full"),
            findMany: toQuery("memberInvites", "MemberInviteSearchInput", ...(await toSearch(memberInvitePartial))),
            createOne: toMutation("memberInviteCreate", "MemberInviteCreateInput", memberInvitePartial, "full"),
            createMany: toMutation("memberInvitesCreate", "[MemberInviteCreateInput!]", memberInvitePartial, "full"),
            updateOne: toMutation("memberInviteUpdate", "MemberInviteUpdateInput", memberInvitePartial, "full"),
            updateMany: toMutation("memberInvitesUpdate", "[MemberInviteUpdateInput!]", memberInvitePartial, "full"),
            accept: toMutation("memberInviteAccept", "FindByIdInput", memberInvitePartial, "full"),
            decline: toMutation("memberInviteDecline", "FindByIdInput", memberInvitePartial, "full"),
        };
    },
    node: async () => {
        const { node: nodePartial } = await import("./partial/node");
        return {
            create: toMutation("nodeCreate", "NodeCreateInput", nodePartial, "full"),
            update: toMutation("nodeUpdate", "NodeUpdateInput", nodePartial, "full"),
        };
    },
    note: async () => {
        const { note: notePartial } = await import("./partial/note");
        return {
            findOne: toQuery("note", "FindByIdInput", notePartial, "full"),
            findMany: toQuery("notes", "NoteSearchInput", ...(await toSearch(notePartial))),
            create: toMutation("noteCreate", "NoteCreateInput", notePartial, "full"),
            update: toMutation("noteUpdate", "NoteUpdateInput", notePartial, "full"),
        };
    },
    noteVersion: async () => {
        const { noteVersion: noteVersionPartial } = await import("./partial/noteVersion");
        return {
            findOne: toQuery("noteVersion", "FindVersionInput", noteVersionPartial, "full"),
            findMany: toQuery("noteVersions", "NoteVersionSearchInput", ...(await toSearch(noteVersionPartial))),
            create: toMutation("noteVersionCreate", "NoteVersionCreateInput", noteVersionPartial, "full"),
            update: toMutation("noteVersionUpdate", "NoteVersionUpdateInput", noteVersionPartial, "full"),
        };
    },
    notification: async () => {
        const { notification: notificationPartial, notificationSettings: notificationSettingsPartial } = await import("./partial/notification");
        const { success: successPartial } = await import("./partial/success");
        const { count: countPartial } = await import("./partial/count");
        return {
            findOne: toQuery("notification", "FindByIdInput", notificationPartial, "full"),
            findMany: toQuery("notifications", "NotificationSearchInput", ...(await toSearch(notificationPartial))),
            settings: toQuery("notificationSettings", null, notificationSettingsPartial, "full"),
            markAsRead: toMutation("notificationMarkAsRead", "FindByIdInput", successPartial, "full"),
            markAllAsRead: toMutation("notificationMarkAllAsRead", null, countPartial, "full"),
            update: toMutation("notificationMarkAllAsRead", null, countPartial, "full"),
            settingsUpdate: toMutation("notificationSettingsUpdate", "NotificationSettingsUpdateInput", notificationSettingsPartial, "full"),
        };
    },
    notificationSubscription: async () => {
        const { notificationSubscription: notificationSubscriptionPartial } = await import("./partial/notificationSubscription");
        return {
            findOne: toQuery("notificationSubscription", "FindByIdInput", notificationSubscriptionPartial, "full"),
            findMany: toQuery("notificationSubscriptions", "NotificationSubscriptionSearchInput", ...(await toSearch(notificationSubscriptionPartial))),
            create: toMutation("notificationSubscriptionCreate", "NotificationSubscriptionCreateInput", notificationSubscriptionPartial, "full"),
            update: toMutation("notificationSubscriptionUpdate", "NotificationSubscriptionUpdateInput", notificationSubscriptionPartial, "full"),
        };
    },
    organization: async () => {
        const { organization: organizationPartial } = await import("./partial/organization");
        return {
            findOne: toQuery("organization", "FindByIdOrHandleInput", organizationPartial, "full"),
            findMany: toQuery("organizations", "OrganizationSearchInput", ...(await toSearch(organizationPartial))),
            create: toMutation("organizationCreate", "OrganizationCreateInput", organizationPartial, "full"),
            update: toMutation("organizationUpdate", "OrganizationUpdateInput", organizationPartial, "full"),
        };
    },
    phone: async () => {
        const { phone: phonePartial } = await import("./partial/phone");
        const { success: successPartial } = await import("./partial/success");
        return {
            create: toMutation("phoneCreate", "PhoneCreateInput", phonePartial, "full"),
            update: toMutation("sendVerificationText", "SendVerificationTextInput", successPartial, "full"),
        };
    },
    popular: async () => {
        const { popular: popularPartial } = await import("./partial/popular");
        return {
            findMany: toQuery("populars", "PopularSearchInput", ...(await toSearch(popularPartial, {
                pageInfo: {
                    hasNextPage: true,
                    endCursorApi: true,
                    endCursorNote: true,
                    endCursorOrganization: true,
                    endCursorProject: true,
                    endCursorQuestion: true,
                    endCursorRoutine: true,
                    endCursorSmartContract: true,
                    endCursorStandard: true,
                    endCursorUser: true,
                },
            }))),
        };
    },
    post: async () => {
        const { post: postPartial } = await import("./partial/post");
        return {
            findOne: toQuery("post", "FindByIdInput", postPartial, "full"),
            findMany: toQuery("posts", "PostSearchInput", ...(await toSearch(postPartial))),
            create: toMutation("postCreate", "PostCreateInput", postPartial, "full"),
            update: toMutation("postUpdate", "PostUpdateInput", postPartial, "full"),
        };
    },
    project: async () => {
        const { project: projectPartial } = await import("./partial/project");
        return {
            findOne: toQuery("project", "FindByIdOrHandleInput", projectPartial, "full"),
            findMany: toQuery("projects", "ProjectSearchInput", ...(await toSearch(projectPartial))),
            create: toMutation("projectCreate", "ProjectCreateInput", projectPartial, "full"),
            update: toMutation("projectUpdate", "ProjectUpdateInput", projectPartial, "full"),
        };
    },
    projectOrOrganization: async () => {
        const { projectOrOrganization: projectOrOrganizationPartial } = await import("./partial/projectOrOrganization");
        return {
            findMany: toQuery("projectOrOrganizations", "ProjectOrOrganizationSearchInput", ...(await toSearch(projectOrOrganizationPartial, {
                pageInfo: {
                    hasNextPage: true,
                    endCursorProject: true,
                    endCursorOrganization: true,
                },
            }))),
        };
    },
    projectOrRoutine: async () => {
        const { projectOrRoutine: projectOrRoutinePartial } = await import("./partial/projectOrRoutine");
        return {
            findMany: toQuery("projectOrRoutines", "ProjectOrRoutineSearchInput", ...(await toSearch(projectOrRoutinePartial, {
                pageInfo: {
                    hasNextPage: true,
                    endCursorProject: true,
                    endCursorRoutine: true,
                },
            }))),
        };
    },
    projectVersion: async () => {
        const { projectVersion: projectVersionPartial } = await import("./partial/projectVersion");
        return {
            findOne: toQuery("projectVersion", "FindVersionInput", projectVersionPartial, "full"),
            findMany: toQuery("projectVersions", "ProjectVersionSearchInput", ...(await toSearch(projectVersionPartial))),
            create: toMutation("projectVersionCreate", "ProjectVersionCreateInput", projectVersionPartial, "full"),
            update: toMutation("projectVersionUpdate", "ProjectVersionUpdateInput", projectVersionPartial, "full"),
        };
    },
    projectVersionDirectory: async () => {
        const { projectVersionDirectory: projectVersionDirectoryPartial } = await import("./partial/projectVersionDirectory");
        return {
            findOne: toQuery("projectVersionDirectory", "FindByIdInput", projectVersionDirectoryPartial, "full"),
            findMany: toQuery("projectVersionDirectories", "ProjectVersionDirectorySearchInput", ...(await toSearch(projectVersionDirectoryPartial))),
            create: toMutation("projectVersionDirectoryCreate", "ProjectVersionDirectoryCreateInput", projectVersionDirectoryPartial, "full"),
            update: toMutation("projectVersionDirectoryUpdate", "ProjectVersionDirectoryUpdateInput", projectVersionDirectoryPartial, "full"),
        };
    },
    pullRequest: async () => {
        const { pullRequest: pullRequestPartial } = await import("./partial/pullRequest");
        return {
            findOne: toQuery("pullRequest", "FindByIdInput", pullRequestPartial, "full"),
            findMany: toQuery("pullRequests", "PullRequestSearchInput", ...(await toSearch(pullRequestPartial))),
            create: toMutation("pullRequestCreate", "PullRequestCreateInput", pullRequestPartial, "full"),
            update: toMutation("pullRequestUpdate", "PullRequestUpdateInput", pullRequestPartial, "full"),
            accept: toMutation("pullRequestAcdept", "FindByIdInput", pullRequestPartial, "full"),
            reject: toMutation("pullRequestReject", "FindByIdInput", pullRequestPartial, "full"),
        };
    },
    pushDevice: async () => {
        const { pushDevice: pushDevicePartial } = await import("./partial/pushDevice");
        return {
            findMany: toQuery("pushDevices", "PushDeviceSearchInput", ...(await toSearch(pushDevicePartial))),
            create: toMutation("pushDeviceCreate", "PushDeviceCreateInput", pushDevicePartial, "full"),
            update: toMutation("pushDeviceUpdate", "PushDeviceUpdateInput", pushDevicePartial, "full"),
        };
    },
    question: async () => {
        const { question: questionPartial } = await import("./partial/question");
        return {
            findOne: toQuery("question", "FindByIdInput", questionPartial, "full"),
            findMany: toQuery("questions", "QuestionSearchInput", ...(await toSearch(questionPartial))),
            create: toMutation("questionCreate", "QuestionCreateInput", questionPartial, "full"),
            update: toMutation("questionUpdate", "QuestionUpdateInput", questionPartial, "full"),
        };
    },
    questionAnswer: async () => {
        const { questionAnswer: questionAnswerPartial } = await import("./partial/questionAnswer");
        return {
            findOne: toQuery("questionAnswer", "FindByIdInput", questionAnswerPartial, "full"),
            findMany: toQuery("questionAnswers", "QuestionAnswerSearchInput", ...(await toSearch(questionAnswerPartial))),
            create: toMutation("questionAnswerCreate", "QuestionAnswerCreateInput", questionAnswerPartial, "full"),
            update: toMutation("questionAnswerUpdate", "QuestionAnswerUpdateInput", questionAnswerPartial, "full"),
            accept: toMutation("questionAnswerMarkAsAccepted", "FindByIdInput", questionAnswerPartial, "full"),
        };
    },
    quiz: async () => {
        const { quiz: quizPartial } = await import("./partial/quiz");
        return {
            findOne: toQuery("quiz", "FindByIdInput", quizPartial, "full"),
            findMany: toQuery("quizzes", "QuizSearchInput", ...(await toSearch(quizPartial))),
            create: toMutation("quizCreate", "QuizCreateInput", quizPartial, "full"),
            update: toMutation("quizUpdate", "QuizUpdateInput", quizPartial, "full"),
        };
    },
    quizAttempt: async () => {
        const { quizAttempt: quizAttemptPartial } = await import("./partial/quizAttempt");
        return {
            findOne: toQuery("quizAttempt", "FindByIdInput", quizAttemptPartial, "full"),
            findMany: toQuery("quizAttempts", "QuizAttemptSearchInput", ...(await toSearch(quizAttemptPartial))),
            create: toMutation("quizAttemptCreate", "QuizAttemptCreateInput", quizAttemptPartial, "full"),
            update: toMutation("quizAttemptUpdate", "QuizAttemptUpdateInput", quizAttemptPartial, "full"),
        };
    },
    quizQuestion: async () => {
        const { quizQuestion: quizQuestionPartial } = await import("./partial/quizQuestion");
        return {
            findOne: toQuery("quizQuestion", "FindByIdInput", quizQuestionPartial, "full"),
            findMany: toQuery("quizQuestions", "QuizQuestionSearchInput", ...(await toSearch(quizQuestionPartial))),
        };
    },
    quizQuestionResponse: async () => {
        const { quizQuestionResponse: quizQuestionResponsePartial } = await import("./partial/quizQuestionResponse");
        return {
            findOne: toQuery("quizQuestionResponse", "FindByIdInput", quizQuestionResponsePartial, "full"),
            findMany: toQuery("quizQuestionResponses", "QuizQuestionResponseSearchInput", ...(await toSearch(quizQuestionResponsePartial))),
            create: toMutation("quizQuestionResponseCreate", "QuizQuestionResponseCreateInput", quizQuestionResponsePartial, "full"),
            update: toMutation("quizQuestionResponseUpdate", "QuizQuestionResponseUpdateInput", quizQuestionResponsePartial, "full"),
        };
    },
    reaction: async () => {
        const { reaction: reactionPartial } = await import("./partial/reaction");
        const { success: successPartial } = await import("./partial/success");
        return {
            findMany: toQuery("reactions", "ReactionSearchInput", ...(await toSearch(reactionPartial))),
            react: toMutation("react", "ReactInput", successPartial, "full"),
        };
    },
    reminder: async () => {
        const { reminder: reminderPartial } = await import("./partial/reminder");
        return {
            findOne: toQuery("reminder", "FindByIdInput", reminderPartial, "full"),
            findMany: toQuery("reminders", "ReminderSearchInput", ...(await toSearch(reminderPartial))),
            create: toMutation("reminderCreate", "ReminderCreateInput", reminderPartial, "full"),
            update: toMutation("reminderUpdate", "ReminderUpdateInput", reminderPartial, "full"),
        };
    },
    reminderList: async () => {
        const { reminderList: reminderListPartial } = await import("./partial/reminderList");
        return {
            findOne: toQuery("reminderList", "FindByIdInput", reminderListPartial, "full"),
            findMany: toQuery("reminderLists", "ReminderListSearchInput", ...(await toSearch(reminderListPartial))),
            create: toMutation("reminderListCreate", "ReminderListCreateInput", reminderListPartial, "full"),
            update: toMutation("reminderListUpdate", "ReminderListUpdateInput", reminderListPartial, "full"),
        };
    },
    report: async () => {
        const { report: reportPartial } = await import("./partial/report");
        return {
            findOne: toQuery("report", "FindByIdInput", reportPartial, "full"),
            findMany: toQuery("reports", "ReportSearchInput", ...(await toSearch(reportPartial))),
            create: toMutation("reportCreate", "ReportCreateInput", reportPartial, "full"),
            update: toMutation("reportUpdate", "ReportUpdateInput", reportPartial, "full"),
        };
    },
    reportResponse: async () => {
        const { reportResponse: reportResponsePartial } = await import("./partial/reportResponse");
        return {
            findOne: toQuery("reportResponse", "FindByIdInput", reportResponsePartial, "full"),
            findMany: toQuery("reportResponses", "ReportResponseSearchInput", ...(await toSearch(reportResponsePartial))),
            create: toMutation("reportResponseCreate", "ReportResponseCreateInput", reportResponsePartial, "full"),
            update: toMutation("reportResponseUpdate", "ReportResponseUpdateInput", reportResponsePartial, "full"),
        };
    },
    reputationHistory: async () => {
        const { reputationHistory: reputationHistoryPartial } = await import("./partial/reputationHistory");
        return {
            findOne: toQuery("reputationHistory", "FindByIdInput", reputationHistoryPartial, "full"),
            findMany: toQuery("reputationHistories", "ReputationHistorySearchInput", ...(await toSearch(reputationHistoryPartial))),
        };
    },
    resource: async () => {
        const { resource: resourcePartial } = await import("./partial/resource");
        return {
            findOne: toQuery("resource", "FindByIdInput", resourcePartial, "full"),
            findMany: toQuery("resources", "ResourceSearchInput", ...(await toSearch(resourcePartial))),
            create: toMutation("resourceCreate", "ResourceCreateInput", resourcePartial, "full"),
            update: toMutation("resourceUpdate", "ResourceUpdateInput", resourcePartial, "full"),
        };
    },
    resourceList: async () => {
        const { resourceList: resourceListPartial } = await import("./partial/resourceList");
        return {
            findOne: toQuery("resourceList", "FindByIdInput", resourceListPartial, "full"),
            findMany: toQuery("resourceLists", "ResourceListSearchInput", ...(await toSearch(resourceListPartial))),
            create: toMutation("resourceListCreate", "ResourceListCreateInput", resourceListPartial, "full"),
            update: toMutation("resourceListUpdate", "ResourceListUpdateInput", resourceListPartial, "full"),
        };
    },
    role: async () => {
        const { role: rolePartial } = await import("./partial/role");
        return {
            findOne: toQuery("role", "FindByIdInput", rolePartial, "full"),
            findMany: toQuery("roles", "RoleSearchInput", ...(await toSearch(rolePartial))),
            create: toMutation("roleCreate", "RoleCreateInput", rolePartial, "full"),
            update: toMutation("roleUpdate", "RoleUpdateInput", rolePartial, "full"),
        };
    },
    routine: async () => {
        const { routine: routinePartial } = await import("./partial/routine");
        return {
            findOne: toQuery("routine", "FindByIdInput", routinePartial, "full"),
            findMany: toQuery("routines", "RoutineSearchInput", ...(await toSearch(routinePartial))),
            create: toMutation("routineCreate", "RoutineCreateInput", routinePartial, "full"),
            update: toMutation("routineUpdate", "RoutineUpdateInput", routinePartial, "full"),
        };
    },
    routineVersion: async () => {
        const { routineVersion: routineVersionPartial } = await import("./partial/routineVersion");
        return {
            findOne: toQuery("routineVersion", "FindVersionInput", routineVersionPartial, "full"),
            findMany: toQuery("routineVersions", "RoutineVersionSearchInput", ...(await toSearch(routineVersionPartial))),
            create: toMutation("routineVersionCreate", "RoutineVersionCreateInput", routineVersionPartial, "full"),
            update: toMutation("routineVersionUpdate", "RoutineVersionUpdateInput", routineVersionPartial, "full"),
        };
    },
    runProject: async () => {
        const { runProject: runProjectPartial } = await import("./partial/runProject");
        const { count: countPartial } = await import("./partial/count");
        return {
            findOne: toQuery("runProject", "FindByIdInput", runProjectPartial, "full"),
            findMany: toQuery("runProjects", "RunProjectSearchInput", ...(await toSearch(runProjectPartial))),
            create: toMutation("runProjectCreate", "RunProjectCreateInput", runProjectPartial, "full"),
            update: toMutation("runProjectUpdate", "RunProjectUpdateInput", runProjectPartial, "full"),
            deleteAll: toMutation("runProjectDeleteAll", null, countPartial, "full"),
            complete: toMutation("runProjectComplete", "RunProjectCompleteInput", runProjectPartial, "full"),
            cancel: toMutation("runProjectCancel", "RunProjectCancelInput", runProjectPartial, "full"),
        };
    },
    runProjectOrRunRoutine: async () => {
        const { runProjectOrRunRoutine: runProjectOrRunRoutinePartial } = await import("./partial/runProjectOrRunRoutine");
        return {
            findMany: toQuery("runProjectOrRunRoutines", "RunProjectOrRunRoutineSearchInput", ...(await toSearch(runProjectOrRunRoutinePartial, {
                pageInfo: {
                    hasNextPage: true,
                    endCursorRunProject: true,
                    endCursorRunRoutine: true,
                },
            }))),
        };
    },
    runRoutine: async () => {
        const { runRoutine: runRoutinePartial } = await import("./partial/runRoutine");
        const { count: countPartial } = await import("./partial/count");
        return {
            findOne: toQuery("runRoutine", "FindByIdInput", runRoutinePartial, "full"),
            findMany: toQuery("runRoutines", "RunRoutineSearchInput", ...(await toSearch(runRoutinePartial))),
            create: toMutation("runRoutineCreate", "RunRoutineCreateInput", runRoutinePartial, "full"),
            update: toMutation("runRoutineUpdate", "RunRoutineUpdateInput", runRoutinePartial, "full"),
            deleteAll: toMutation("runRoutineDeleteAll", null, countPartial, "full"),
            complete: toMutation("runRoutineComplete", "RunRoutineCompleteInput", runRoutinePartial, "full"),
            cancel: toMutation("runRoutineCancel", "RunRoutineCancelInput", runRoutinePartial, "full"),
        };
    },
    runRoutineInput: async () => {
        const { runRoutineInput: runRoutineInputPartial } = await import("./partial/runRoutineInput");
        return {
            findMany: toQuery("runRoutineInputs", "RunRoutineInputSearchInput", ...(await toSearch(runRoutineInputPartial))),
        };
    },
    schedule: async () => {
        const { schedule: schedulePartial } = await import("./partial/schedule");
        return {
            findOne: toQuery("schedule", "FindByIdInput", schedulePartial, "full"),
            findMany: toQuery("schedules", "ScheduleSearchInput", ...(await toSearch(schedulePartial))),
            create: toMutation("scheduleCreate", "ScheduleCreateInput", schedulePartial, "full"),
            update: toMutation("scheduleUpdate", "ScheduleUpdateInput", schedulePartial, "full"),
        };
    },
    scheduleException: async () => {
        const { scheduleException: scheduleExceptionPartial } = await import("./partial/scheduleException");
        return {
            findOne: toQuery("scheduleException", "FindByIdInput", scheduleExceptionPartial, "full"),
            findMany: toQuery("scheduleExceptions", "ScheduleExceptionSearchInput", ...(await toSearch(scheduleExceptionPartial))),
            create: toMutation("scheduleExceptionCreate", "ScheduleExceptionCreateInput", scheduleExceptionPartial, "full"),
            update: toMutation("scheduleExceptionUpdate", "ScheduleExceptionUpdateInput", scheduleExceptionPartial, "full"),
        };
    },
    scheduleRecurrence: async () => {
        const { scheduleRecurrence: scheduleRecurrencePartial } = await import("./partial/scheduleRecurrence");
        return {
            findOne: toQuery("scheduleRecurrence", "FindByIdInput", scheduleRecurrencePartial, "full"),
            findMany: toQuery("scheduleRecurrences", "ScheduleRecurrenceSearchInput", ...(await toSearch(scheduleRecurrencePartial))),
            create: toMutation("scheduleRecurrenceCreate", "ScheduleRecurrenceCreateInput", scheduleRecurrencePartial, "full"),
            update: toMutation("scheduleRecurrenceUpdate", "ScheduleRecurrenceUpdateInput", scheduleRecurrencePartial, "full"),
        };
    },
    smartContract: async () => {
        const { smartContract: smartContractPartial } = await import("./partial/smartContract");
        return {
            findOne: toQuery("smartContract", "FindByIdInput", smartContractPartial, "full"),
            findMany: toQuery("smartContracts", "SmartContractSearchInput", ...(await toSearch(smartContractPartial))),
            create: toMutation("smartContractCreate", "SmartContractCreateInput", smartContractPartial, "full"),
            update: toMutation("smartContractUpdate", "SmartContractUpdateInput", smartContractPartial, "full"),
        };
    },
    smartContractVersion: async () => {
        const { smartContractVersion: smartContractVersionPartial } = await import("./partial/smartContractVersion");
        return {
            findOne: toQuery("smartContractVersion", "FindVersionInput", smartContractVersionPartial, "full"),
            findMany: toQuery("smartContractVersions", "SmartContractVersionSearchInput", ...(await toSearch(smartContractVersionPartial))),
            create: toMutation("smartContractVersionCreate", "SmartContractVersionCreateInput", smartContractVersionPartial, "full"),
            update: toMutation("smartContractVersionUpdate", "SmartContractVersionUpdateInput", smartContractVersionPartial, "full"),
        };
    },
    standard: async () => {
        const { standard: standardPartial } = await import("./partial/standard");
        return {
            findOne: toQuery("standard", "FindByIdInput", standardPartial, "full"),
            findMany: toQuery("standards", "StandardSearchInput", ...(await toSearch(standardPartial))),
            create: toMutation("standardCreate", "StandardCreateInput", standardPartial, "full"),
            update: toMutation("standardUpdate", "StandardUpdateInput", standardPartial, "full"),
        };
    },
    standardVersion: async () => {
        const { standardVersion: standardVersionPartial } = await import("./partial/standardVersion");
        return {
            findOne: toQuery("standardVersion", "FindVersionInput", standardVersionPartial, "full"),
            findMany: toQuery("standardVersions", "StandardVersionSearchInput", ...(await toSearch(standardVersionPartial))),
            create: toMutation("standardVersionCreate", "StandardVersionCreateInput", standardVersionPartial, "full"),
            update: toMutation("standardVersionUpdate", "StandardVersionUpdateInput", standardVersionPartial, "full"),
        };
    },
    statsApi: async () => {
        const { statsApi: statsApiPartial } = await import("./partial/statsApi");
        return {
            findMany: toQuery("statsApi", "StatsApiSearchInput", ...(await toSearch(statsApiPartial))),
        };
    },
    statsOrganization: async () => {
        const { statsOrganization: statsOrganizationPartial } = await import("./partial/statsOrganization");
        return {
            findMany: toQuery("statsOrganization", "StatsOrganizationSearchInput", ...(await toSearch(statsOrganizationPartial))),
        };
    },
    statsProject: async () => {
        const { statsProject: statsProjectPartial } = await import("./partial/statsProject");
        return {
            findMany: toQuery("statsProject", "StatsProjectSearchInput", ...(await toSearch(statsProjectPartial))),
        };
    },
    statsQuiz: async () => {
        const { statsQuiz: statsQuizPartial } = await import("./partial/statsQuiz");
        return {
            findMany: toQuery("statsQuiz", "StatsQuizSearchInput", ...(await toSearch(statsQuizPartial))),
        };
    },
    statsRoutine: async () => {
        const { statsRoutine: statsRoutinePartial } = await import("./partial/statsRoutine");
        return {
            findMany: toQuery("statsRoutine", "StatsRoutineSearchInput", ...(await toSearch(statsRoutinePartial))),
        };
    },
    statsSite: async () => {
        const { statsSite: statsSitePartial } = await import("./partial/statsSite");
        return {
            findMany: toQuery("statsSite", "StatsSiteSearchInput", ...(await toSearch(statsSitePartial))),
        };
    },
    statsSmartContract: async () => {
        const { statsSmartContract: statsSmartContractPartial } = await import("./partial/statsSmartContract");
        return {
            findMany: toQuery("statsSmartContract", "StatsSmartContractSearchInput", ...(await toSearch(statsSmartContractPartial))),
        };
    },
    statsStandard: async () => {
        const { statsStandard: statsStandardPartial } = await import("./partial/statsStandard");
        return {
            findMany: toQuery("statsStandard", "StatsStandardSearchInput", ...(await toSearch(statsStandardPartial))),
        };
    },
    statsUser: async () => {
        const { statsUser: statsUserPartial } = await import("./partial/statsUser");
        return {
            findMany: toQuery("statsUser", "StatsUserSearchInput", ...(await toSearch(statsUserPartial))),
        };
    },
    tag: async () => {
        const { tag: tagPartial } = await import("./partial/tag");
        return {
            findOne: toQuery("tag", "FindByIdInput", tagPartial, "full"),
            findMany: toQuery("tags", "TagSearchInput", ...(await toSearch(tagPartial))),
            create: toMutation("tagCreate", "TagCreateInput", tagPartial, "full"),
            update: toMutation("tagUpdate", "TagUpdateInput", tagPartial, "full"),
        };
    },
    transfer: async () => {
        const { transfer: transferPartial } = await import("./partial/transfer");
        return {
            findOne: toQuery("transfer", "FindByIdInput", transferPartial, "full"),
            findMany: toQuery("transfers", "TransferSearchInput", ...(await toSearch(transferPartial))),
            requestSend: toMutation("transferRequestSend", "TransferRequestSendInput", transferPartial, "full"),
            requestReceive: toMutation("transferRequestReceive", "TransferRequestReceiveInput", transferPartial, "full"),
            update: toMutation("transferUpdate", "TransferUpdateInput", transferPartial, "full"),
            cancel: toMutation("transferCancel", "FindByIdInput", transferPartial, "full"),
            accept: toMutation("transferAccept", "FindByIdInput", transferPartial, "full"),
            deny: toMutation("transferDeny", "TransferDenyInput", transferPartial, "full"),
        };
    },
    translate: async () => {
        const { translate: translatePartial } = await import("./partial/translate");
        return {
            translate: toQuery("translate", "FindByIdInput", translatePartial, "full"),
        };
    },
    user: async () => {
        const { user: userPartial, profile: profilePartial } = await import("./partial/user");
        const { session: sessionPartial } = await import("./partial/session");
        return {
            profile: toQuery("profile", null, profilePartial, "full"),
            findOne: toQuery("user", "FindByIdOrHandleInput", userPartial, "full"),
            findMany: toQuery("users", "UserSearchInput", ...(await toSearch(userPartial))),
            botCreate: toMutation("botCreate", "BotCreateInput", userPartial, "full"),
            botUpdate: toMutation("botUpdate", "BotUpdateInput", userPartial, "full"),
            profileUpdate: toMutation("profileUpdate", "ProfileUpdateInput", profilePartial, "full"),
            profileEmailUpdate: toMutation("profileEmailUpdate", "ProfileEmailUpdateInput", profilePartial, "full"),
            deleteOne: toMutation("userDeleteOne", "UserDeleteInput", sessionPartial, "full"),
            exportData: toMutation("exportData", null),
        };
    },
    view: async () => {
        const { view: viewPartial } = await import("./partial/view");
        return {
            findMany: toQuery("views", "ViewSearchInput", ...(await toSearch(viewPartial))),
        };
    },
    wallet: async () => {
        const { wallet: walletPartial } = await import("./partial/wallet");
        return {
            update: toMutation("walletUpdate", "WalletUpdateInput", walletPartial, "full"),
        };
    },
};
