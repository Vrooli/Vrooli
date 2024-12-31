import { toObject, toSearch } from "./utils";

export const endpoints = {
    api: async () => {
        const { api } = await import("./partial/api");
        return {
            findOne: await toObject(api, "full"),
            findMany: await toObject(...(await toSearch(api))),
            create: await toObject(api, "full"),
            update: await toObject(api, "full"),
        };
    },
    apiKey: async () => {
        const { apiKey } = await import("./partial/apiKey");
        const { success } = await import("./partial/success");
        return {
            create: await toObject(apiKey, "full"),
            update: await toObject(apiKey, "full"),
            deleteOne: await toObject(success, "full"),
            validate: await toObject(apiKey, "full"),
        };
    },
    apiVersion: async () => {
        const { apiVersion } = await import("./partial/apiVersion");
        return {
            findOne: await toObject(apiVersion, "full"),
            findMany: await toObject(...(await toSearch(apiVersion))),
            create: await toObject(apiVersion, "full"),
            update: await toObject(apiVersion, "full"),
        };
    },
    auth: async () => {
        const { session } = await import("./partial/session");
        const { success } = await import("./partial/success");
        const { walletInit, walletComplete } = await import("./partial/wallet");
        return {
            emailLogIn: await toObject(session, "full"),
            emailSignUp: await toObject(session, "full"),
            emailRequestPasswordChange: await toObject(success, "full"),
            emailResetPassword: await toObject(session, "full"),
            guestLogIn: await toObject(session, "full"),
            logOut: await toObject(session, "full"),
            logOutAll: await toObject(session, "full"),
            validateSession: await toObject(session, "full"),
            switchCurrentAccount: await toObject(session, "full"),
            walletInit: await toObject(walletInit, "full"),
            walletComplete: await toObject(walletComplete, "full"),
        };
    },
    award: async () => {
        const { award } = await import("./partial/award");
        return {
            findMany: await toObject(...(await toSearch(award))),
        };
    },
    bookmark: async () => {
        const { bookmark } = await import("./partial/bookmark");
        return {
            findOne: await toObject(bookmark, "full"),
            findMany: await toObject(...(await toSearch(bookmark))),
            create: await toObject(bookmark, "full"),
            update: await toObject(bookmark, "full"),
        };
    },
    bookmarkList: async () => {
        const { bookmarkList } = await import("./partial/bookmarkList");
        return {
            findOne: await toObject(bookmarkList, "full"),
            findMany: await toObject(...(await toSearch(bookmarkList))),
            create: await toObject(bookmarkList, "full"),
            update: await toObject(bookmarkList, "full"),
        };
    },
    chat: async () => {
        const { chat } = await import("./partial/chat");
        return {
            findOne: await toObject(chat, "full"),
            findMany: await toObject(...(await toSearch(chat))),
            create: await toObject(chat, "full"),
            update: await toObject(chat, "full"),
        };
    },
    chatInvite: async () => {
        const { chatInvite } = await import("./partial/chatInvite");
        return {
            findOne: await toObject(chatInvite, "full"),
            findMany: await toObject(...(await toSearch(chatInvite))),
            createOne: await toObject(chatInvite, "full"),
            createMany: await toObject(chatInvite, "full"),
            updateOne: await toObject(chatInvite, "full"),
            updateMany: await toObject(chatInvite, "full"),
            accept: await toObject(chatInvite, "full"),
            decline: await toObject(chatInvite, "full"),
        };
    },
    chatMessage: async () => {
        const { chatMessage, chatMessageSearchTreeResult } = await import("./partial/chatMessage");
        const { success } = await import("./partial/success");
        return {
            findOne: await toObject(chatMessage, "full"),
            findMany: await toObject(...(await toSearch(chatMessage))),
            findTree: await toObject(chatMessageSearchTreeResult, "common"),
            create: await toObject(chatMessage, "full"),
            update: await toObject(chatMessage, "full"),
            regenerateResponse: await toObject(success, "full"),
        };
    },
    chatParticipant: async () => {
        const { chatParticipant } = await import("./partial/chatParticipant");
        return {
            findOne: await toObject(chatParticipant, "full"),
            findMany: await toObject(...(await toSearch(chatParticipant))),
            update: await toObject(chatParticipant, "full"),
        };
    },
    code: async () => {
        const { code } = await import("./partial/code");
        return {
            findOne: await toObject(code, "full"),
            findMany: await toObject(...(await toSearch(code))),
            create: await toObject(code, "full"),
            update: await toObject(code, "full"),
        };
    },
    codeVersion: async () => {
        const { codeVersion } = await import("./partial/codeVersion");
        return {
            findOne: await toObject(codeVersion, "full"),
            findMany: await toObject(...(await toSearch(codeVersion))),
            create: await toObject(codeVersion, "full"),
            update: await toObject(codeVersion, "full"),
        };
    },
    comment: async () => {
        const { comment, commentSearchResult } = await import("./partial/comment");
        return {
            findOne: await toObject(comment, "full"),
            findMany: await toObject(commentSearchResult, "full"),
            create: await toObject(comment, "full"),
            update: await toObject(comment, "full"),
        };
    },
    copy: async () => {
        const { copyResult } = await import("./partial/copyResult");
        return {
            copy: await toObject(copyResult, "full"),
        };
    },
    deleteOneOrMany: async () => {
        const { success } = await import("./partial/success");
        const { count } = await import("./partial/count");
        return {
            deleteOne: await toObject(success, "full"),
            deleteMany: await toObject(count, "full"),
        };
    },
    email: async () => {
        const { email } = await import("./partial/email");
        const { success } = await import("./partial/success");
        return {
            create: await toObject(email, "full"),
            verify: await toObject(success, "full"),
        };
    },
    feed: async () => {
        const { homeResult } = await import("./partial/feed");
        return {
            home: await toObject(homeResult, "list"),
        };
    },
    issue: async () => {
        const { issue } = await import("./partial/issue");
        return {
            findOne: await toObject(issue, "full"),
            findMany: await toObject(...(await toSearch(issue))),
            create: await toObject(issue, "full"),
            update: await toObject(issue, "full"),
            close: await toObject(issue, "full"),
        };
    },
    focusMode: async () => {
        const { activeFocusMode } = await import("./partial/activeFocusMode");
        const { focusMode } = await import("./partial/focusMode");
        return {
            findOne: await toObject(focusMode, "full"),
            findMany: await toObject(...(await toSearch(focusMode))),
            create: await toObject(focusMode, "full"),
            update: await toObject(focusMode, "full"),
            setActive: await toObject(activeFocusMode, "full"),
        };
    },
    label: async () => {
        const { label } = await import("./partial/label");
        return {
            findOne: await toObject(label, "full"),
            findMany: await toObject(...(await toSearch(label))),
            create: await toObject(label, "full"),
            update: await toObject(label, "full"),
        };
    },
    meeting: async () => {
        const { meeting } = await import("./partial/meeting");
        return {
            findOne: await toObject(meeting, "full"),
            findMany: await toObject(...(await toSearch(meeting))),
            create: await toObject(meeting, "full"),
            update: await toObject(meeting, "full"),
        };
    },
    meetingInvite: async () => {
        const { meetingInvite } = await import("./partial/meetingInvite");
        return {
            findOne: await toObject(meetingInvite, "full"),
            findMany: await toObject(...(await toSearch(meetingInvite))),
            createOne: await toObject(meetingInvite, "full"),
            createMany: await toObject(meetingInvite, "full"),
            updateOne: await toObject(meetingInvite, "full"),
            updateMany: await toObject(meetingInvite, "full"),
            accept: await toObject(meetingInvite, "full"),
            decline: await toObject(meetingInvite, "full"),
        };
    },
    member: async () => {
        const { member } = await import("./partial/member");
        return {
            findOne: await toObject(member, "full"),
            findMany: await toObject(...(await toSearch(member))),
            update: await toObject(member, "full"),
        };
    },
    memberInvite: async () => {
        const { memberInvite } = await import("./partial/memberInvite");
        return {
            findOne: await toObject(memberInvite, "full"),
            findMany: await toObject(...(await toSearch(memberInvite))),
            createOne: await toObject(memberInvite, "full"),
            createMany: await toObject(memberInvite, "full"),
            updateOne: await toObject(memberInvite, "full"),
            updateMany: await toObject(memberInvite, "full"),
            accept: await toObject(memberInvite, "full"),
            decline: await toObject(memberInvite, "full"),
        };
    },
    node: async () => {
        const { node } = await import("./partial/node");
        return {
            create: await toObject(node, "full"),
            update: await toObject(node, "full"),
        };
    },
    note: async () => {
        const { note } = await import("./partial/note");
        return {
            findOne: await toObject(note, "full"),
            findMany: await toObject(...(await toSearch(note))),
            create: await toObject(note, "full"),
            update: await toObject(note, "full"),
        };
    },
    noteVersion: async () => {
        const { noteVersion } = await import("./partial/noteVersion");
        return {
            findOne: await toObject(noteVersion, "full"),
            findMany: await toObject(...(await toSearch(noteVersion))),
            create: await toObject(noteVersion, "full"),
            update: await toObject(noteVersion, "full"),
        };
    },
    notification: async () => {
        const { notification, notificationSettings } = await import("./partial/notification");
        const { success } = await import("./partial/success");
        const { count } = await import("./partial/count");
        return {
            findOne: await toObject(notification, "full"),
            findMany: await toObject(...(await toSearch(notification))),
            settings: await toObject(notificationSettings, "full"),
            markAsRead: await toObject(success, "full"),
            markAllAsRead: await toObject(count, "full"),
            update: await toObject(count, "full"),
            settingsUpdate: await toObject(notificationSettings, "full"),
        };
    },
    notificationSubscription: async () => {
        const { notificationSubscription } = await import("./partial/notificationSubscription");
        return {
            findOne: await toObject(notificationSubscription, "full"),
            findMany: await toObject(...(await toSearch(notificationSubscription))),
            create: await toObject(notificationSubscription, "full"),
            update: await toObject(notificationSubscription, "full"),
        };
    },
    phone: async () => {
        const { phone } = await import("./partial/phone");
        const { success } = await import("./partial/success");
        return {
            create: await toObject(phone, "full"),
            verify: await toObject(success, "full"),
            validate: await toObject(success, "full"),
        };
    },
    popular: async () => {
        const { popular } = await import("./partial/popular");
        return {
            findMany: await toObject(...(await toSearch(popular, {
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
            findOne: await toObject(post, "full"),
            findMany: await toObject(...(await toSearch(post))),
            create: await toObject(post, "full"),
            update: await toObject(post, "full"),
        };
    },
    project: async () => {
        const { project } = await import("./partial/project");
        return {
            findOne: await toObject(project, "full"),
            findMany: await toObject(...(await toSearch(project))),
            create: await toObject(project, "full"),
            update: await toObject(project, "full"),
        };
    },
    projectOrRoutine: async () => {
        const { projectOrRoutine } = await import("./partial/projectOrRoutine");
        return {
            findMany: await toObject(...(await toSearch(projectOrRoutine, {
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
            findMany: await toObject(...(await toSearch(projectOrTeam, {
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
            findOne: await toObject(projectVersion, "full"),
            findMany: await toObject(...(await toSearch(projectVersion))),
            create: await toObject(projectVersion, "full"),
            update: await toObject(projectVersion, "full"),
        };
    },
    projectVersionDirectory: async () => {
        const { projectVersionDirectory } = await import("./partial/projectVersionDirectory");
        return {
            findOne: await toObject(projectVersionDirectory, "full"),
            findMany: await toObject(...(await toSearch(projectVersionDirectory))),
            create: await toObject(projectVersionDirectory, "full"),
            update: await toObject(projectVersionDirectory, "full"),
        };
    },
    pullRequest: async () => {
        const { pullRequest } = await import("./partial/pullRequest");
        return {
            findOne: await toObject(pullRequest, "full"),
            findMany: await toObject(...(await toSearch(pullRequest))),
            create: await toObject(pullRequest, "full"),
            update: await toObject(pullRequest, "full"),
            accept: await toObject(pullRequest, "full"),
            reject: await toObject(pullRequest, "full"),
        };
    },
    pushDevice: async () => {
        const { pushDevice } = await import("./partial/pushDevice");
        const { success } = await import("./partial/success");
        return {
            findMany: await toObject(...(await toSearch(pushDevice))),
            create: await toObject(pushDevice, "full"),
            test: await toObject(success, "full"),
            update: await toObject(pushDevice, "full"),
        };
    },
    question: async () => {
        const { question } = await import("./partial/question");
        return {
            findOne: await toObject(question, "full"),
            findMany: await toObject(...(await toSearch(question))),
            create: await toObject(question, "full"),
            update: await toObject(question, "full"),
        };
    },
    questionAnswer: async () => {
        const { questionAnswer } = await import("./partial/questionAnswer");
        return {
            findOne: await toObject(questionAnswer, "full"),
            findMany: await toObject(...(await toSearch(questionAnswer))),
            create: await toObject(questionAnswer, "full"),
            update: await toObject(questionAnswer, "full"),
            accept: await toObject(questionAnswer, "full"),
        };
    },
    quiz: async () => {
        const { quiz } = await import("./partial/quiz");
        return {
            findOne: await toObject(quiz, "full"),
            findMany: await toObject(...(await toSearch(quiz))),
            create: await toObject(quiz, "full"),
            update: await toObject(quiz, "full"),
        };
    },
    quizAttempt: async () => {
        const { quizAttempt } = await import("./partial/quizAttempt");
        return {
            findOne: await toObject(quizAttempt, "full"),
            findMany: await toObject(...(await toSearch(quizAttempt))),
            create: await toObject(quizAttempt, "full"),
            update: await toObject(quizAttempt, "full"),
        };
    },
    quizQuestion: async () => {
        const { quizQuestion } = await import("./partial/quizQuestion");
        return {
            findOne: await toObject(quizQuestion, "full"),
            findMany: await toObject(...(await toSearch(quizQuestion))),
        };
    },
    quizQuestionResponse: async () => {
        const { quizQuestionResponse } = await import("./partial/quizQuestionResponse");
        return {
            findOne: await toObject(quizQuestionResponse, "full"),
            findMany: await toObject(...(await toSearch(quizQuestionResponse))),
            create: await toObject(quizQuestionResponse, "full"),
            update: await toObject(quizQuestionResponse, "full"),
        };
    },
    reaction: async () => {
        const { reaction } = await import("./partial/reaction");
        const { success } = await import("./partial/success");
        return {
            findMany: await toObject(...(await toSearch(reaction))),
            react: await toObject(success, "full"),
        };
    },
    reminder: async () => {
        const { reminder } = await import("./partial/reminder");
        return {
            findOne: await toObject(reminder, "full"),
            findMany: await toObject(...(await toSearch(reminder))),
            create: await toObject(reminder, "full"),
            update: await toObject(reminder, "full"),
        };
    },
    reminderList: async () => {
        const { reminderList } = await import("./partial/reminderList");
        return {
            findOne: await toObject(reminderList, "full"),
            findMany: await toObject(...(await toSearch(reminderList))),
            create: await toObject(reminderList, "full"),
            update: await toObject(reminderList, "full"),
        };
    },
    report: async () => {
        const { report } = await import("./partial/report");
        return {
            findOne: await toObject(report, "full"),
            findMany: await toObject(...(await toSearch(report))),
            create: await toObject(report, "full"),
            update: await toObject(report, "full"),
        };
    },
    reportResponse: async () => {
        const { reportResponse } = await import("./partial/reportResponse");
        return {
            findOne: await toObject(reportResponse, "full"),
            findMany: await toObject(...(await toSearch(reportResponse))),
            create: await toObject(reportResponse, "full"),
            update: await toObject(reportResponse, "full"),
        };
    },
    reputationHistory: async () => {
        const { reputationHistory } = await import("./partial/reputationHistory");
        return {
            findOne: await toObject(reputationHistory, "full"),
            findMany: await toObject(...(await toSearch(reputationHistory))),
        };
    },
    resource: async () => {
        const { resource } = await import("./partial/resource");
        return {
            findOne: await toObject(resource, "full"),
            findMany: await toObject(...(await toSearch(resource))),
            create: await toObject(resource, "full"),
            update: await toObject(resource, "full"),
        };
    },
    resourceList: async () => {
        const { resourceList } = await import("./partial/resourceList");
        return {
            findOne: await toObject(resourceList, "full"),
            findMany: await toObject(...(await toSearch(resourceList))),
            create: await toObject(resourceList, "full"),
            update: await toObject(resourceList, "full"),
        };
    },
    role: async () => {
        const { role } = await import("./partial/role");
        return {
            findOne: await toObject(role, "full"),
            findMany: await toObject(...(await toSearch(role))),
            create: await toObject(role, "full"),
            update: await toObject(role, "full"),
        };
    },
    routine: async () => {
        const { routine } = await import("./partial/routine");
        return {
            findOne: await toObject(routine, "full"),
            findMany: await toObject(...(await toSearch(routine))),
            create: await toObject(routine, "full"),
            update: await toObject(routine, "full"),
        };
    },
    routineVersion: async () => {
        const { routineVersion } = await import("./partial/routineVersion");
        return {
            findOne: await toObject(routineVersion, "full"),
            findMany: await toObject(...(await toSearch(routineVersion))),
            create: await toObject(routineVersion, "full"),
            update: await toObject(routineVersion, "full"),
        };
    },
    runProject: async () => {
        const { runProject } = await import("./partial/runProject");
        const { count } = await import("./partial/count");
        return {
            findOne: await toObject(runProject, "full"),
            findMany: await toObject(...(await toSearch(runProject))),
            create: await toObject(runProject, "full"),
            update: await toObject(runProject, "full"),
            deleteAll: await toObject(count, "full"),
        };
    },
    runProjectOrRunRoutine: async () => {
        const { runProjectOrRunRoutine } = await import("./partial/runProjectOrRunRoutine");
        return {
            findMany: await toObject(...(await toSearch(runProjectOrRunRoutine, {
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
            findOne: await toObject(runRoutine, "full"),
            findMany: await toObject(...(await toSearch(runRoutine))),
            create: await toObject(runRoutine, "full"),
            update: await toObject(runRoutine, "full"),
            deleteAll: await toObject(count, "full"),
        };
    },
    runRoutineInput: async () => {
        const { runRoutineInput } = await import("./partial/runRoutineInput");
        return {
            findMany: await toObject(...(await toSearch(runRoutineInput))),
        };
    },
    runRoutineOutput: async () => {
        const { runRoutineOutput } = await import("./partial/runRoutineOutput");
        return {
            findMany: await toObject(...(await toSearch(runRoutineOutput))),
        };
    },
    schedule: async () => {
        const { schedule } = await import("./partial/schedule");
        return {
            findOne: await toObject(schedule, "full"),
            findMany: await toObject(...(await toSearch(schedule))),
            create: await toObject(schedule, "full"),
            update: await toObject(schedule, "full"),
        };
    },
    scheduleException: async () => {
        const { scheduleException } = await import("./partial/scheduleException");
        return {
            findOne: await toObject(scheduleException, "full"),
            findMany: await toObject(...(await toSearch(scheduleException))),
            create: await toObject(scheduleException, "full"),
            update: await toObject(scheduleException, "full"),
        };
    },
    scheduleRecurrence: async () => {
        const { scheduleRecurrence } = await import("./partial/scheduleRecurrence");
        return {
            findOne: await toObject(scheduleRecurrence, "full"),
            findMany: await toObject(...(await toSearch(scheduleRecurrence))),
            create: await toObject(scheduleRecurrence, "full"),
            update: await toObject(scheduleRecurrence, "full"),
        };
    },
    standard: async () => {
        const { standard } = await import("./partial/standard");
        return {
            findOne: await toObject(standard, "full"),
            findMany: await toObject(...(await toSearch(standard))),
            create: await toObject(standard, "full"),
            update: await toObject(standard, "full"),
        };
    },
    standardVersion: async () => {
        const { standardVersion } = await import("./partial/standardVersion");
        return {
            findOne: await toObject(standardVersion, "full"),
            findMany: await toObject(...(await toSearch(standardVersion))),
            create: await toObject(standardVersion, "full"),
            update: await toObject(standardVersion, "full"),
        };
    },
    statsApi: async () => {
        const { statsApi } = await import("./partial/statsApi");
        return {
            findMany: await toObject(...(await toSearch(statsApi))),
        };
    },
    statsCode: async () => {
        const { statsCode } = await import("./partial/statsCode");
        return {
            findMany: await toObject(...(await toSearch(statsCode))),
        };
    },
    statsProject: async () => {
        const { statsProject } = await import("./partial/statsProject");
        return {
            findMany: await toObject(...(await toSearch(statsProject))),
        };
    },
    statsQuiz: async () => {
        const { statsQuiz } = await import("./partial/statsQuiz");
        return {
            findMany: await toObject(...(await toSearch(statsQuiz))),
        };
    },
    statsRoutine: async () => {
        const { statsRoutine } = await import("./partial/statsRoutine");
        return {
            findMany: await toObject(...(await toSearch(statsRoutine))),
        };
    },
    statsSite: async () => {
        const { statsSite } = await import("./partial/statsSite");
        return {
            findMany: await toObject(...(await toSearch(statsSite))),
        };
    },
    statsStandard: async () => {
        const { statsStandard } = await import("./partial/statsStandard");
        return {
            findMany: await toObject(...(await toSearch(statsStandard))),
        };
    },
    statsTeam: async () => {
        const { statsTeam } = await import("./partial/statsTeam");
        return {
            findMany: await toObject(...(await toSearch(statsTeam))),
        };
    },
    statsUser: async () => {
        const { statsUser } = await import("./partial/statsUser");
        return {
            findMany: await toObject(...(await toSearch(statsUser))),
        };
    },
    tag: async () => {
        const { tag } = await import("./partial/tag");
        return {
            findOne: await toObject(tag, "full"),
            findMany: await toObject(...(await toSearch(tag))),
            create: await toObject(tag, "full"),
            update: await toObject(tag, "full"),
        };
    },
    task: async () => {
        const { checkTaskStatusesResult } = await import("./partial/task");
        const { success } = await import("./partial/success");
        return {
            startLlmTask: await toObject(success, "full"),
            startRunTask: await toObject(success, "full"),
            cancelTask: await toObject(success, "full"),
            checkTaskStatuses: await toObject(checkTaskStatusesResult, "full"),
        };
    },
    team: async () => {
        const { team } = await import("./partial/team");
        return {
            findOne: await toObject(team, "full"),
            findMany: await toObject(...(await toSearch(team))),
            create: await toObject(team, "full"),
            update: await toObject(team, "full"),
        };
    },
    transfer: async () => {
        const { transfer } = await import("./partial/transfer");
        return {
            findOne: await toObject(transfer, "full"),
            findMany: await toObject(...(await toSearch(transfer))),
            requestSend: await toObject(transfer, "full"),
            requestReceive: await toObject(transfer, "full"),
            update: await toObject(transfer, "full"),
            cancel: await toObject(transfer, "full"),
            accept: await toObject(transfer, "full"),
            deny: await toObject(transfer, "full"),
        };
    },
    translate: async () => {
        const { translate } = await import("./partial/translate");
        return {
            translate: await toObject(translate, "full"),
        };
    },
    user: async () => {
        const { user, profile } = await import("./partial/user");
        const { session } = await import("./partial/session");
        const { success } = await import("./partial/success");
        return {
            profile: await toObject(profile, "full"),
            findOne: await toObject(user, "full"),
            findMany: await toObject(...(await toSearch(user))),
            botCreate: await toObject(user, "full"),
            botUpdate: await toObject(user, "full"),
            profileUpdate: await toObject(profile, "full"),
            profileEmailUpdate: await toObject(profile, "full"),
            deleteOne: await toObject(session, "full"),
            exportData: await toObject(success, "full"),
        };
    },
    view: async () => {
        const { view } = await import("./partial/view");
        return {
            findMany: await toObject(...(await toSearch(view))),
        };
    },
    wallet: async () => {
        const { wallet } = await import("./partial/wallet");
        return {
            update: await toObject(wallet, "full"),
        };
    },
};
