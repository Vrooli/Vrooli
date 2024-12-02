import { toMutation, toQuery, toSearch } from "./utils";

export const endpoints = {
    api: async () => {
        const { api } = await import("./partial/api");
        return {
            findOne: toQuery("api", "FindByIdInput", api, "full"),
            findMany: toQuery("apis", "ApiSearchInput", ...(await toSearch(api))),
            create: toMutation("apiCreate", "ApiCreateInput", api, "full"),
            update: toMutation("apiUpdate", "ApiUpdateInput", api, "full"),
        };
    },
    apiKey: async () => {
        const { apiKey } = await import("./partial/apiKey");
        const { success } = await import("./partial/success");
        return {
            create: toMutation("apiKeyCreate", "ApiKeyCreateInput", apiKey, "full"),
            update: toMutation("apiKeyUpdate", "ApiKeyUpdateInput", apiKey, "full"),
            deleteOne: toMutation("apiKeyDeleteOne", "ApiKeyDeleteOneInput", success, "full"),
            validate: toMutation("apiKeyValidate", "ApiKeyValidateInput", apiKey, "full"),
        };
    },
    apiVersion: async () => {
        const { apiVersion } = await import("./partial/apiVersion");
        return {
            findOne: toQuery("apiVersion", "FindVersionInput", apiVersion, "full"),
            findMany: toQuery("apiVersions", "ApiVersionSearchInput", ...(await toSearch(apiVersion))),
            create: toMutation("apiVersionCreate", "ApiVersionCreateInput", apiVersion, "full"),
            update: toMutation("apiVersionUpdate", "ApiVersionUpdateInput", apiVersion, "full"),
        };
    },
    auth: async () => {
        const { session } = await import("./partial/session");
        const { success } = await import("./partial/success");
        const { walletComplete } = await import("./partial/walletComplete");
        return {
            emailLogIn: toMutation("emailLogIn", "EmailLogInInput", session, "full"),
            emailSignUp: toMutation("emailSignUp", "EmailSignUpInput", session, "full"),
            emailRequestPasswordChange: toMutation("emailRequestPasswordChange", "EmailRequestPasswordChangeInput", success, "full"),
            emailResetPassword: toMutation("emailResetPassword", "EmailResetPasswordInput", session, "full"),
            guestLogIn: toMutation("guestLogIn", null, session, "full"),
            logOut: toMutation("logOut", null, session, "full"),
            logOutAll: toMutation("logOutAll", null, session, "full"),
            validateSession: toMutation("validateSession", "ValidateSessionInput", session, "full"),
            switchCurrentAccount: toMutation("switchCurrentAccount", "SwitchCurrentAccountInput", session, "full"),
            walletInit: toMutation("walletInit", "WalletInitInput"),
            walletComplete: toMutation("walletComplete", "WalletCompleteInput", walletComplete, "full"),
        };
    },
    award: async () => {
        const { award } = await import("./partial/award");
        return {
            findMany: toQuery("awards", "AwardSearchInput", ...(await toSearch(award))),
        };
    },
    bookmark: async () => {
        const { bookmark } = await import("./partial/bookmark");
        return {
            findOne: toQuery("bookmark", "FindByIdInput", bookmark, "full"),
            findMany: toQuery("bookmarks", "BookmarkSearchInput", ...(await toSearch(bookmark))),
            create: toMutation("bookmarkCreate", "BookmarkCreateInput", bookmark, "full"),
            update: toMutation("bookmarkUpdate", "BookmarkUpdateInput", bookmark, "full"),
        };
    },
    bookmarkList: async () => {
        const { bookmarkList } = await import("./partial/bookmarkList");
        return {
            findOne: toQuery("bookmarkList", "FindByIdInput", bookmarkList, "full"),
            findMany: toQuery("bookmarkLists", "BookmarkListSearchInput", ...(await toSearch(bookmarkList))),
            create: toMutation("bookmarkListCreate", "BookmarkListCreateInput", bookmarkList, "full"),
            update: toMutation("bookmarkListUpdate", "BookmarkListUpdateInput", bookmarkList, "full"),
        };
    },
    chat: async () => {
        const { chat } = await import("./partial/chat");
        return {
            findOne: toQuery("chat", "FindByIdInput", chat, "full"),
            findMany: toQuery("chats", "ChatSearchInput", ...(await toSearch(chat))),
            create: toMutation("chatCreate", "ChatCreateInput", chat, "full"),
            update: toMutation("chatUpdate", "ChatUpdateInput", chat, "full"),
        };
    },
    chatInvite: async () => {
        const { chatInvite } = await import("./partial/chatInvite");
        return {
            findOne: toQuery("chatInvite", "FindByIdInput", chatInvite, "full"),
            findMany: toQuery("chatInvites", "ChatInviteSearchInput", ...(await toSearch(chatInvite))),
            createOne: toMutation("chatInviteCreate", "ChatInviteCreateInput", chatInvite, "full"),
            createMany: toMutation("chatInvitesCreate", "[ChatInviteCreateInput!]", chatInvite, "full"),
            updateOne: toMutation("chatInviteUpdate", "ChatInviteUpdateInput", chatInvite, "full"),
            updateMany: toMutation("chatInvitesUpdate", "[ChatInviteUpdateInput!]", chatInvite, "full"),
            accept: toMutation("chatInviteAccept", "FindByIdInput", chatInvite, "full"),
            decline: toMutation("chatInviteDecline", "FindByIdInput", chatInvite, "full"),
        };
    },
    chatMessage: async () => {
        const { chatMessage, chatMessageSearchTreeResult } = await import("./partial/chatMessage");
        const { success } = await import("./partial/success");
        return {
            findOne: toQuery("chatMessage", "FindByIdInput", chatMessage, "full"),
            findMany: toQuery("chatMessages", "ChatMessageSearchInput", ...(await toSearch(chatMessage))),
            findTree: toQuery("chatMessageTree", "ChatMessageSearchTreeInput", chatMessageSearchTreeResult, "common"),
            create: toMutation("chatMessageCreate", "ChatMessageCreateWithTaskInfoInput", chatMessage, "full"),
            update: toMutation("chatMessageUpdate", "ChatMessageUpdateWithTaskInfoInput", chatMessage, "full"),
            regenerateResponse: toMutation("regenerateResponse", "RegenerateResponseInput", success, "full"),
        };
    },
    chatParticipant: async () => {
        const { chatParticipant } = await import("./partial/chatParticipant");
        return {
            findOne: toQuery("chatParticipant", "FindByIdInput", chatParticipant, "full"),
            findMany: toQuery("chatParticipants", "ChatParticipantSearchInput", ...(await toSearch(chatParticipant))),
            update: toMutation("chatParticipantUpdate", "ChatParticipantUpdateInput", chatParticipant, "full"),
        };
    },
    code: async () => {
        const { code } = await import("./partial/code");
        return {
            findOne: toQuery("code", "FindByIdInput", code, "full"),
            findMany: toQuery("codes", "CodeSearchInput", ...(await toSearch(code))),
            create: toMutation("codeCreate", "CodeCreateInput", code, "full"),
            update: toMutation("codeUpdate", "CodeUpdateInput", code, "full"),
        };
    },
    codeVersion: async () => {
        const { codeVersion } = await import("./partial/codeVersion");
        return {
            findOne: toQuery("codeVersion", "FindVersionInput", codeVersion, "full"),
            findMany: toQuery("codeVersions", "CodeVersionSearchInput", ...(await toSearch(codeVersion))),
            create: toMutation("codeVersionCreate", "CodeVersionCreateInput", codeVersion, "full"),
            update: toMutation("codeVersionUpdate", "CodeVersionUpdateInput", codeVersion, "full"),
        };
    },
    comment: async () => {
        const { comment, commentSearchResult } = await import("./partial/comment");
        return {
            findOne: toQuery("comment", "FindByIdInput", comment, "full"),
            findMany: toQuery("comments", "CommentSearchInput", commentSearchResult, "full"),
            create: toMutation("commentCreate", "CommentCreateInput", comment, "full"),
            update: toMutation("commentUpdate", "CommentUpdateInput", comment, "full"),
        };
    },
    copy: async () => {
        const { copyResult } = await import("./partial/copyResult");
        return {
            copy: toMutation("copy", "CopyInput", copyResult, "full"),
        };
    },
    deleteOneOrMany: async () => {
        const { success } = await import("./partial/success");
        const { count } = await import("./partial/count");
        return {
            deleteOne: toMutation("deleteOne", "DeleteOneInput", success, "full"),
            deleteMany: toMutation("deleteMany", "DeleteManyInput", count, "full"),
        };
    },
    email: async () => {
        const { email } = await import("./partial/email");
        const { success } = await import("./partial/success");
        return {
            create: toMutation("emailCreate", "EmailCreateInput", email, "full"),
            verify: toMutation("sendVerificationEmail", "SendVerificationEmailInput", success, "full"),
        };
    },
    feed: async () => {
        const { homeResult } = await import("./partial/feed");
        return {
            home: toQuery("home", "HomeInput", homeResult, "list"),
        };
    },
    issue: async () => {
        const { issue } = await import("./partial/issue");
        return {
            findOne: toQuery("issue", "FindByIdInput", issue, "full"),
            findMany: toQuery("issues", "IssueSearchInput", ...(await toSearch(issue))),
            create: toMutation("issueCreate", "IssueCreateInput", issue, "full"),
            update: toMutation("issueUpdate", "IssueUpdateInput", issue, "full"),
            close: toMutation("issueClose", "IssueCloseInput", issue, "full"),
        };
    },
    focusMode: async () => {
        const { activeFocusMode } = await import("./partial/activeFocusMode");
        const { focusMode } = await import("./partial/focusMode");
        return {
            findOne: toQuery("focusMode", "FindByIdInput", focusMode, "full"),
            findMany: toQuery("focusModes", "FocusModeSearchInput", ...(await toSearch(focusMode))),
            create: toMutation("focusModeCreate", "FocusModeCreateInput", focusMode, "full"),
            update: toMutation("focusModeUpdate", "FocusModeUpdateInput", focusMode, "full"),
            setActive: toMutation("setActiveFocusMode", "SetActiveFocusModeInput", activeFocusMode, "full"),
        };
    },
    label: async () => {
        const { label } = await import("./partial/label");
        return {
            findOne: toQuery("label", "FindByIdInput", label, "full"),
            findMany: toQuery("labels", "LabelSearchInput", ...(await toSearch(label))),
            create: toMutation("labelCreate", "LabelCreateInput", label, "full"),
            update: toMutation("labelUpdate", "LabelUpdateInput", label, "full"),
        };
    },
    meeting: async () => {
        const { meeting } = await import("./partial/meeting");
        return {
            findOne: toQuery("meeting", "FindByIdInput", meeting, "full"),
            findMany: toQuery("meetings", "MeetingSearchInput", ...(await toSearch(meeting))),
            create: toMutation("meetingCreate", "MeetingCreateInput", meeting, "full"),
            update: toMutation("meetingUpdate", "MeetingUpdateInput", meeting, "full"),
        };
    },
    meetingInvite: async () => {
        const { meetingInvite } = await import("./partial/meetingInvite");
        return {
            findOne: toQuery("meetingInvite", "FindByIdInput", meetingInvite, "full"),
            findMany: toQuery("meetingInvites", "MeetingInviteSearchInput", ...(await toSearch(meetingInvite))),
            createOne: toMutation("meetingInviteCreate", "MeetingInviteCreateInput", meetingInvite, "full"),
            createMany: toMutation("meetingInvitesCreate", "[MeetingInviteCreateInput!]", meetingInvite, "full"),
            updateOne: toMutation("meetingInviteUpdate", "MeetingInviteUpdateInput", meetingInvite, "full"),
            updateMany: toMutation("meetingInvitesUpdate", "[MeetingInviteUpdateInput!]", meetingInvite, "full"),
            accept: toMutation("meetingInviteAccept", "FindByIdInput", meetingInvite, "full"),
            decline: toMutation("meetingInviteDecline", "FindByIdInput", meetingInvite, "full"),
        };
    },
    member: async () => {
        const { member } = await import("./partial/member");
        return {
            findOne: toQuery("member", "FindByIdInput", member, "full"),
            findMany: toQuery("members", "MemberSearchInput", ...(await toSearch(member))),
            update: toMutation("memberUpdate", "MemberUpdateInput", member, "full"),
        };
    },
    memberInvite: async () => {
        const { memberInvite } = await import("./partial/memberInvite");
        return {
            findOne: toQuery("memberInvite", "FindByIdInput", memberInvite, "full"),
            findMany: toQuery("memberInvites", "MemberInviteSearchInput", ...(await toSearch(memberInvite))),
            createOne: toMutation("memberInviteCreate", "MemberInviteCreateInput", memberInvite, "full"),
            createMany: toMutation("memberInvitesCreate", "[MemberInviteCreateInput!]", memberInvite, "full"),
            updateOne: toMutation("memberInviteUpdate", "MemberInviteUpdateInput", memberInvite, "full"),
            updateMany: toMutation("memberInvitesUpdate", "[MemberInviteUpdateInput!]", memberInvite, "full"),
            accept: toMutation("memberInviteAccept", "FindByIdInput", memberInvite, "full"),
            decline: toMutation("memberInviteDecline", "FindByIdInput", memberInvite, "full"),
        };
    },
    node: async () => {
        const { node } = await import("./partial/node");
        return {
            create: toMutation("nodeCreate", "NodeCreateInput", node, "full"),
            update: toMutation("nodeUpdate", "NodeUpdateInput", node, "full"),
        };
    },
    note: async () => {
        const { note } = await import("./partial/note");
        return {
            findOne: toQuery("note", "FindByIdInput", note, "full"),
            findMany: toQuery("notes", "NoteSearchInput", ...(await toSearch(note))),
            create: toMutation("noteCreate", "NoteCreateInput", note, "full"),
            update: toMutation("noteUpdate", "NoteUpdateInput", note, "full"),
        };
    },
    noteVersion: async () => {
        const { noteVersion } = await import("./partial/noteVersion");
        return {
            findOne: toQuery("noteVersion", "FindVersionInput", noteVersion, "full"),
            findMany: toQuery("noteVersions", "NoteVersionSearchInput", ...(await toSearch(noteVersion))),
            create: toMutation("noteVersionCreate", "NoteVersionCreateInput", noteVersion, "full"),
            update: toMutation("noteVersionUpdate", "NoteVersionUpdateInput", noteVersion, "full"),
        };
    },
    notification: async () => {
        const { notification, notificationSettings } = await import("./partial/notification");
        const { success } = await import("./partial/success");
        const { count } = await import("./partial/count");
        return {
            findOne: toQuery("notification", "FindByIdInput", notification, "full"),
            findMany: toQuery("notifications", "NotificationSearchInput", ...(await toSearch(notification))),
            settings: toQuery("notificationSettings", null, notificationSettings, "full"),
            markAsRead: toMutation("notificationMarkAsRead", "FindByIdInput", success, "full"),
            markAllAsRead: toMutation("notificationMarkAllAsRead", null, count, "full"),
            update: toMutation("notificationMarkAllAsRead", null, count, "full"),
            settingsUpdate: toMutation("notificationSettingsUpdate", "NotificationSettingsUpdateInput", notificationSettings, "full"),
        };
    },
    notificationSubscription: async () => {
        const { notificationSubscription } = await import("./partial/notificationSubscription");
        return {
            findOne: toQuery("notificationSubscription", "FindByIdInput", notificationSubscription, "full"),
            findMany: toQuery("notificationSubscriptions", "NotificationSubscriptionSearchInput", ...(await toSearch(notificationSubscription))),
            create: toMutation("notificationSubscriptionCreate", "NotificationSubscriptionCreateInput", notificationSubscription, "full"),
            update: toMutation("notificationSubscriptionUpdate", "NotificationSubscriptionUpdateInput", notificationSubscription, "full"),
        };
    },
    phone: async () => {
        const { phone } = await import("./partial/phone");
        const { success } = await import("./partial/success");
        return {
            create: toMutation("phoneCreate", "PhoneCreateInput", phone, "full"),
            verify: toMutation("sendVerificationText", "SendVerificationTextInput", success, "full"),
            validate: toMutation("validateVerificationText", "ValidateVerificationTextInput", success, "full"),
        };
    },
    popular: async () => {
        const { popular } = await import("./partial/popular");
        return {
            findMany: toQuery("populars", "PopularSearchInput", ...(await toSearch(popular, {
                pageInfo: {
                    hasNextPage: true,
                    endCursorApi: true,
                    endCursorCode: true,
                    endCursorNote: true,
                    endCursorProject: true,
                    endCursorQuestion: true,
                    endCursorRoutine: true,
                    endCursorStandard: true,
                    endCursorTeam: true,
                    endCursorUser: true,
                },
            }))),
        };
    },
    post: async () => {
        const { post } = await import("./partial/post");
        return {
            findOne: toQuery("post", "FindByIdInput", post, "full"),
            findMany: toQuery("posts", "PostSearchInput", ...(await toSearch(post))),
            create: toMutation("postCreate", "PostCreateInput", post, "full"),
            update: toMutation("postUpdate", "PostUpdateInput", post, "full"),
        };
    },
    project: async () => {
        const { project } = await import("./partial/project");
        return {
            findOne: toQuery("project", "FindByIdOrHandleInput", project, "full"),
            findMany: toQuery("projects", "ProjectSearchInput", ...(await toSearch(project))),
            create: toMutation("projectCreate", "ProjectCreateInput", project, "full"),
            update: toMutation("projectUpdate", "ProjectUpdateInput", project, "full"),
        };
    },
    projectOrRoutine: async () => {
        const { projectOrRoutine } = await import("./partial/projectOrRoutine");
        return {
            findMany: toQuery("projectOrRoutines", "ProjectOrRoutineSearchInput", ...(await toSearch(projectOrRoutine, {
                pageInfo: {
                    hasNextPage: true,
                    endCursorProject: true,
                    endCursorRoutine: true,
                },
            }))),
        };
    },
    projectOrTeam: async () => {
        const { projectOrTeam } = await import("./partial/projectOrTeam");
        return {
            findMany: toQuery("projectOrTeams", "ProjectOrTeamSearchInput", ...(await toSearch(projectOrTeam, {
                pageInfo: {
                    hasNextPage: true,
                    endCursorProject: true,
                    endCursorTeam: true,
                },
            }))),
        };
    },
    projectVersion: async () => {
        const { projectVersion } = await import("./partial/projectVersion");
        return {
            findOne: toQuery("projectVersion", "FindVersionInput", projectVersion, "full"),
            findMany: toQuery("projectVersions", "ProjectVersionSearchInput", ...(await toSearch(projectVersion))),
            create: toMutation("projectVersionCreate", "ProjectVersionCreateInput", projectVersion, "full"),
            update: toMutation("projectVersionUpdate", "ProjectVersionUpdateInput", projectVersion, "full"),
        };
    },
    projectVersionDirectory: async () => {
        const { projectVersionDirectory } = await import("./partial/projectVersionDirectory");
        return {
            findOne: toQuery("projectVersionDirectory", "FindByIdInput", projectVersionDirectory, "full"),
            findMany: toQuery("projectVersionDirectories", "ProjectVersionDirectorySearchInput", ...(await toSearch(projectVersionDirectory))),
            create: toMutation("projectVersionDirectoryCreate", "ProjectVersionDirectoryCreateInput", projectVersionDirectory, "full"),
            update: toMutation("projectVersionDirectoryUpdate", "ProjectVersionDirectoryUpdateInput", projectVersionDirectory, "full"),
        };
    },
    pullRequest: async () => {
        const { pullRequest } = await import("./partial/pullRequest");
        return {
            findOne: toQuery("pullRequest", "FindByIdInput", pullRequest, "full"),
            findMany: toQuery("pullRequests", "PullRequestSearchInput", ...(await toSearch(pullRequest))),
            create: toMutation("pullRequestCreate", "PullRequestCreateInput", pullRequest, "full"),
            update: toMutation("pullRequestUpdate", "PullRequestUpdateInput", pullRequest, "full"),
            accept: toMutation("pullRequestAcdept", "FindByIdInput", pullRequest, "full"),
            reject: toMutation("pullRequestReject", "FindByIdInput", pullRequest, "full"),
        };
    },
    pushDevice: async () => {
        const { pushDevice } = await import("./partial/pushDevice");
        const { success } = await import("./partial/success");
        return {
            findMany: toQuery("pushDevices", "PushDeviceSearchInput", ...(await toSearch(pushDevice))),
            create: toMutation("pushDeviceCreate", "PushDeviceCreateInput", pushDevice, "full"),
            test: toMutation("pushDeviceTest", "PushDeviceTestInput", success, "full"),
            update: toMutation("pushDeviceUpdate", "PushDeviceUpdateInput", pushDevice, "full"),
        };
    },
    question: async () => {
        const { question } = await import("./partial/question");
        return {
            findOne: toQuery("question", "FindByIdInput", question, "full"),
            findMany: toQuery("questions", "QuestionSearchInput", ...(await toSearch(question))),
            create: toMutation("questionCreate", "QuestionCreateInput", question, "full"),
            update: toMutation("questionUpdate", "QuestionUpdateInput", question, "full"),
        };
    },
    questionAnswer: async () => {
        const { questionAnswer } = await import("./partial/questionAnswer");
        return {
            findOne: toQuery("questionAnswer", "FindByIdInput", questionAnswer, "full"),
            findMany: toQuery("questionAnswers", "QuestionAnswerSearchInput", ...(await toSearch(questionAnswer))),
            create: toMutation("questionAnswerCreate", "QuestionAnswerCreateInput", questionAnswer, "full"),
            update: toMutation("questionAnswerUpdate", "QuestionAnswerUpdateInput", questionAnswer, "full"),
            accept: toMutation("questionAnswerMarkAsAccepted", "FindByIdInput", questionAnswer, "full"),
        };
    },
    quiz: async () => {
        const { quiz } = await import("./partial/quiz");
        return {
            findOne: toQuery("quiz", "FindByIdInput", quiz, "full"),
            findMany: toQuery("quizzes", "QuizSearchInput", ...(await toSearch(quiz))),
            create: toMutation("quizCreate", "QuizCreateInput", quiz, "full"),
            update: toMutation("quizUpdate", "QuizUpdateInput", quiz, "full"),
        };
    },
    quizAttempt: async () => {
        const { quizAttempt } = await import("./partial/quizAttempt");
        return {
            findOne: toQuery("quizAttempt", "FindByIdInput", quizAttempt, "full"),
            findMany: toQuery("quizAttempts", "QuizAttemptSearchInput", ...(await toSearch(quizAttempt))),
            create: toMutation("quizAttemptCreate", "QuizAttemptCreateInput", quizAttempt, "full"),
            update: toMutation("quizAttemptUpdate", "QuizAttemptUpdateInput", quizAttempt, "full"),
        };
    },
    quizQuestion: async () => {
        const { quizQuestion } = await import("./partial/quizQuestion");
        return {
            findOne: toQuery("quizQuestion", "FindByIdInput", quizQuestion, "full"),
            findMany: toQuery("quizQuestions", "QuizQuestionSearchInput", ...(await toSearch(quizQuestion))),
        };
    },
    quizQuestionResponse: async () => {
        const { quizQuestionResponse } = await import("./partial/quizQuestionResponse");
        return {
            findOne: toQuery("quizQuestionResponse", "FindByIdInput", quizQuestionResponse, "full"),
            findMany: toQuery("quizQuestionResponses", "QuizQuestionResponseSearchInput", ...(await toSearch(quizQuestionResponse))),
            create: toMutation("quizQuestionResponseCreate", "QuizQuestionResponseCreateInput", quizQuestionResponse, "full"),
            update: toMutation("quizQuestionResponseUpdate", "QuizQuestionResponseUpdateInput", quizQuestionResponse, "full"),
        };
    },
    reaction: async () => {
        const { reaction } = await import("./partial/reaction");
        const { success } = await import("./partial/success");
        return {
            findMany: toQuery("reactions", "ReactionSearchInput", ...(await toSearch(reaction))),
            react: toMutation("react", "ReactInput", success, "full"),
        };
    },
    reminder: async () => {
        const { reminder } = await import("./partial/reminder");
        return {
            findOne: toQuery("reminder", "FindByIdInput", reminder, "full"),
            findMany: toQuery("reminders", "ReminderSearchInput", ...(await toSearch(reminder))),
            create: toMutation("reminderCreate", "ReminderCreateInput", reminder, "full"),
            update: toMutation("reminderUpdate", "ReminderUpdateInput", reminder, "full"),
        };
    },
    reminderList: async () => {
        const { reminderList } = await import("./partial/reminderList");
        return {
            findOne: toQuery("reminderList", "FindByIdInput", reminderList, "full"),
            findMany: toQuery("reminderLists", "ReminderListSearchInput", ...(await toSearch(reminderList))),
            create: toMutation("reminderListCreate", "ReminderListCreateInput", reminderList, "full"),
            update: toMutation("reminderListUpdate", "ReminderListUpdateInput", reminderList, "full"),
        };
    },
    report: async () => {
        const { report } = await import("./partial/report");
        return {
            findOne: toQuery("report", "FindByIdInput", report, "full"),
            findMany: toQuery("reports", "ReportSearchInput", ...(await toSearch(report))),
            create: toMutation("reportCreate", "ReportCreateInput", report, "full"),
            update: toMutation("reportUpdate", "ReportUpdateInput", report, "full"),
        };
    },
    reportResponse: async () => {
        const { reportResponse } = await import("./partial/reportResponse");
        return {
            findOne: toQuery("reportResponse", "FindByIdInput", reportResponse, "full"),
            findMany: toQuery("reportResponses", "ReportResponseSearchInput", ...(await toSearch(reportResponse))),
            create: toMutation("reportResponseCreate", "ReportResponseCreateInput", reportResponse, "full"),
            update: toMutation("reportResponseUpdate", "ReportResponseUpdateInput", reportResponse, "full"),
        };
    },
    reputationHistory: async () => {
        const { reputationHistory } = await import("./partial/reputationHistory");
        return {
            findOne: toQuery("reputationHistory", "FindByIdInput", reputationHistory, "full"),
            findMany: toQuery("reputationHistories", "ReputationHistorySearchInput", ...(await toSearch(reputationHistory))),
        };
    },
    resource: async () => {
        const { resource } = await import("./partial/resource");
        return {
            findOne: toQuery("resource", "FindByIdInput", resource, "full"),
            findMany: toQuery("resources", "ResourceSearchInput", ...(await toSearch(resource))),
            create: toMutation("resourceCreate", "ResourceCreateInput", resource, "full"),
            update: toMutation("resourceUpdate", "ResourceUpdateInput", resource, "full"),
        };
    },
    resourceList: async () => {
        const { resourceList } = await import("./partial/resourceList");
        return {
            findOne: toQuery("resourceList", "FindByIdInput", resourceList, "full"),
            findMany: toQuery("resourceLists", "ResourceListSearchInput", ...(await toSearch(resourceList))),
            create: toMutation("resourceListCreate", "ResourceListCreateInput", resourceList, "full"),
            update: toMutation("resourceListUpdate", "ResourceListUpdateInput", resourceList, "full"),
        };
    },
    role: async () => {
        const { role } = await import("./partial/role");
        return {
            findOne: toQuery("role", "FindByIdInput", role, "full"),
            findMany: toQuery("roles", "RoleSearchInput", ...(await toSearch(role))),
            create: toMutation("roleCreate", "RoleCreateInput", role, "full"),
            update: toMutation("roleUpdate", "RoleUpdateInput", role, "full"),
        };
    },
    routine: async () => {
        const { routine } = await import("./partial/routine");
        return {
            findOne: toQuery("routine", "FindByIdInput", routine, "full"),
            findMany: toQuery("routines", "RoutineSearchInput", ...(await toSearch(routine))),
            create: toMutation("routineCreate", "RoutineCreateInput", routine, "full"),
            update: toMutation("routineUpdate", "RoutineUpdateInput", routine, "full"),
        };
    },
    routineVersion: async () => {
        const { routineVersion } = await import("./partial/routineVersion");
        return {
            findOne: toQuery("routineVersion", "FindVersionInput", routineVersion, "full"),
            findMany: toQuery("routineVersions", "RoutineVersionSearchInput", ...(await toSearch(routineVersion))),
            create: toMutation("routineVersionCreate", "RoutineVersionCreateInput", routineVersion, "full"),
            update: toMutation("routineVersionUpdate", "RoutineVersionUpdateInput", routineVersion, "full"),
        };
    },
    runProject: async () => {
        const { runProject } = await import("./partial/runProject");
        const { count } = await import("./partial/count");
        return {
            findOne: toQuery("runProject", "FindByIdInput", runProject, "full"),
            findMany: toQuery("runProjects", "RunProjectSearchInput", ...(await toSearch(runProject))),
            create: toMutation("runProjectCreate", "RunProjectCreateInput", runProject, "full"),
            update: toMutation("runProjectUpdate", "RunProjectUpdateInput", runProject, "full"),
            deleteAll: toMutation("runProjectDeleteAll", null, count, "full"),
        };
    },
    runProjectOrRunRoutine: async () => {
        const { runProjectOrRunRoutine } = await import("./partial/runProjectOrRunRoutine");
        return {
            findMany: toQuery("runProjectOrRunRoutines", "RunProjectOrRunRoutineSearchInput", ...(await toSearch(runProjectOrRunRoutine, {
                pageInfo: {
                    hasNextPage: true,
                    endCursorRunProject: true,
                    endCursorRunRoutine: true,
                },
            }))),
        };
    },
    runRoutine: async () => {
        const { runRoutine } = await import("./partial/runRoutine");
        const { count } = await import("./partial/count");
        return {
            findOne: toQuery("runRoutine", "FindByIdInput", runRoutine, "full"),
            findMany: toQuery("runRoutines", "RunRoutineSearchInput", ...(await toSearch(runRoutine))),
            create: toMutation("runRoutineCreate", "RunRoutineCreateInput", runRoutine, "full"),
            update: toMutation("runRoutineUpdate", "RunRoutineUpdateInput", runRoutine, "full"),
            deleteAll: toMutation("runRoutineDeleteAll", null, count, "full"),
        };
    },
    runRoutineInput: async () => {
        const { runRoutineInput } = await import("./partial/runRoutineInput");
        return {
            findMany: toQuery("runRoutineInputs", "RunRoutineInputSearchInput", ...(await toSearch(runRoutineInput))),
        };
    },
    runRoutineOutput: async () => {
        const { runRoutineOutput } = await import("./partial/runRoutineOutput");
        return {
            findMany: toQuery("runRoutineOutputs", "RunRoutineOutputSearchInput", ...(await toSearch(runRoutineOutput))),
        };
    },
    schedule: async () => {
        const { schedule } = await import("./partial/schedule");
        return {
            findOne: toQuery("schedule", "FindByIdInput", schedule, "full"),
            findMany: toQuery("schedules", "ScheduleSearchInput", ...(await toSearch(schedule))),
            create: toMutation("scheduleCreate", "ScheduleCreateInput", schedule, "full"),
            update: toMutation("scheduleUpdate", "ScheduleUpdateInput", schedule, "full"),
        };
    },
    scheduleException: async () => {
        const { scheduleException } = await import("./partial/scheduleException");
        return {
            findOne: toQuery("scheduleException", "FindByIdInput", scheduleException, "full"),
            findMany: toQuery("scheduleExceptions", "ScheduleExceptionSearchInput", ...(await toSearch(scheduleException))),
            create: toMutation("scheduleExceptionCreate", "ScheduleExceptionCreateInput", scheduleException, "full"),
            update: toMutation("scheduleExceptionUpdate", "ScheduleExceptionUpdateInput", scheduleException, "full"),
        };
    },
    scheduleRecurrence: async () => {
        const { scheduleRecurrence } = await import("./partial/scheduleRecurrence");
        return {
            findOne: toQuery("scheduleRecurrence", "FindByIdInput", scheduleRecurrence, "full"),
            findMany: toQuery("scheduleRecurrences", "ScheduleRecurrenceSearchInput", ...(await toSearch(scheduleRecurrence))),
            create: toMutation("scheduleRecurrenceCreate", "ScheduleRecurrenceCreateInput", scheduleRecurrence, "full"),
            update: toMutation("scheduleRecurrenceUpdate", "ScheduleRecurrenceUpdateInput", scheduleRecurrence, "full"),
        };
    },
    standard: async () => {
        const { standard } = await import("./partial/standard");
        return {
            findOne: toQuery("standard", "FindByIdInput", standard, "full"),
            findMany: toQuery("standards", "StandardSearchInput", ...(await toSearch(standard))),
            create: toMutation("standardCreate", "StandardCreateInput", standard, "full"),
            update: toMutation("standardUpdate", "StandardUpdateInput", standard, "full"),
        };
    },
    standardVersion: async () => {
        const { standardVersion } = await import("./partial/standardVersion");
        return {
            findOne: toQuery("standardVersion", "FindVersionInput", standardVersion, "full"),
            findMany: toQuery("standardVersions", "StandardVersionSearchInput", ...(await toSearch(standardVersion))),
            create: toMutation("standardVersionCreate", "StandardVersionCreateInput", standardVersion, "full"),
            update: toMutation("standardVersionUpdate", "StandardVersionUpdateInput", standardVersion, "full"),
        };
    },
    statsApi: async () => {
        const { statsApi } = await import("./partial/statsApi");
        return {
            findMany: toQuery("statsApi", "StatsApiSearchInput", ...(await toSearch(statsApi))),
        };
    },
    statsCode: async () => {
        const { statsCode } = await import("./partial/statsCode");
        return {
            findMany: toQuery("statsCode", "StatsCodeSearchInput", ...(await toSearch(statsCode))),
        };
    },
    statsProject: async () => {
        const { statsProject } = await import("./partial/statsProject");
        return {
            findMany: toQuery("statsProject", "StatsProjectSearchInput", ...(await toSearch(statsProject))),
        };
    },
    statsQuiz: async () => {
        const { statsQuiz } = await import("./partial/statsQuiz");
        return {
            findMany: toQuery("statsQuiz", "StatsQuizSearchInput", ...(await toSearch(statsQuiz))),
        };
    },
    statsRoutine: async () => {
        const { statsRoutine } = await import("./partial/statsRoutine");
        return {
            findMany: toQuery("statsRoutine", "StatsRoutineSearchInput", ...(await toSearch(statsRoutine))),
        };
    },
    statsSite: async () => {
        const { statsSite } = await import("./partial/statsSite");
        return {
            findMany: toQuery("statsSite", "StatsSiteSearchInput", ...(await toSearch(statsSite))),
        };
    },
    statsStandard: async () => {
        const { statsStandard } = await import("./partial/statsStandard");
        return {
            findMany: toQuery("statsStandard", "StatsStandardSearchInput", ...(await toSearch(statsStandard))),
        };
    },
    statsTeam: async () => {
        const { statsTeam } = await import("./partial/statsTeam");
        return {
            findMany: toQuery("statsTeam", "StatsTeamSearchInput", ...(await toSearch(statsTeam))),
        };
    },
    statsUser: async () => {
        const { statsUser } = await import("./partial/statsUser");
        return {
            findMany: toQuery("statsUser", "StatsUserSearchInput", ...(await toSearch(statsUser))),
        };
    },
    tag: async () => {
        const { tag } = await import("./partial/tag");
        return {
            findOne: toQuery("tag", "FindByIdInput", tag, "full"),
            findMany: toQuery("tags", "TagSearchInput", ...(await toSearch(tag))),
            create: toMutation("tagCreate", "TagCreateInput", tag, "full"),
            update: toMutation("tagUpdate", "TagUpdateInput", tag, "full"),
        };
    },
    task: async () => {
        const { checkTaskStatusesResult } = await import("./partial/task");
        const { success } = await import("./partial/success");
        return {
            startLlmTask: toMutation("startLlmTask", "StartLlmTaskInput", success, "full"),
            startRunTask: toMutation("startRunTask", "StartRunTaskInput", success, "full"),
            cancelTask: toMutation("cancelTask", "CancelTaskInput", success, "full"),
            checkTaskStatuses: toMutation("checkTaskStatuses", "CheckTaskStatusesInput", checkTaskStatusesResult, "full"),
        };
    },
    team: async () => {
        const { team } = await import("./partial/team");
        return {
            findOne: toQuery("team", "FindByIdOrHandleInput", team, "full"),
            findMany: toQuery("teams", "TeamSearchInput", ...(await toSearch(team))),
            create: toMutation("teamCreate", "TeamCreateInput", team, "full"),
            update: toMutation("teamUpdate", "TeamUpdateInput", team, "full"),
        };
    },
    transfer: async () => {
        const { transfer } = await import("./partial/transfer");
        return {
            findOne: toQuery("transfer", "FindByIdInput", transfer, "full"),
            findMany: toQuery("transfers", "TransferSearchInput", ...(await toSearch(transfer))),
            requestSend: toMutation("transferRequestSend", "TransferRequestSendInput", transfer, "full"),
            requestReceive: toMutation("transferRequestReceive", "TransferRequestReceiveInput", transfer, "full"),
            update: toMutation("transferUpdate", "TransferUpdateInput", transfer, "full"),
            cancel: toMutation("transferCancel", "FindByIdInput", transfer, "full"),
            accept: toMutation("transferAccept", "FindByIdInput", transfer, "full"),
            deny: toMutation("transferDeny", "TransferDenyInput", transfer, "full"),
        };
    },
    translate: async () => {
        const { translate } = await import("./partial/translate");
        return {
            translate: toQuery("translate", "FindByIdInput", translate, "full"),
        };
    },
    user: async () => {
        const { user, profile } = await import("./partial/user");
        const { session } = await import("./partial/session");
        return {
            profile: toQuery("profile", null, profile, "full"),
            findOne: toQuery("user", "FindByIdOrHandleInput", user, "full"),
            findMany: toQuery("users", "UserSearchInput", ...(await toSearch(user))),
            botCreate: toMutation("botCreate", "BotCreateInput", user, "full"),
            botUpdate: toMutation("botUpdate", "BotUpdateInput", user, "full"),
            profileUpdate: toMutation("profileUpdate", "ProfileUpdateInput", profile, "full"),
            profileEmailUpdate: toMutation("profileEmailUpdate", "ProfileEmailUpdateInput", profile, "full"),
            deleteOne: toMutation("userDeleteOne", "UserDeleteInput", session, "full"),
            exportData: toMutation("exportData", null),
        };
    },
    view: async () => {
        const { view } = await import("./partial/view");
        return {
            findMany: toQuery("views", "ViewSearchInput", ...(await toSearch(view))),
        };
    },
    wallet: async () => {
        const { wallet } = await import("./partial/wallet");
        return {
            update: toMutation("walletUpdate", "WalletUpdateInput", wallet, "full"),
        };
    },
};
