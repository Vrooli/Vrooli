import { toObject, toSearch } from "./utils";

export const endpoints = {
    actions: async () => {
        const { copyResult } = await import("./partial/actions");
        const { success } = await import("./partial/success");
        const { count } = await import("./partial/count");
        return {
            copy: await toObject(copyResult, "full"),
            deleteOne: await toObject(success, "full"),
            deleteMany: await toObject(count, "full"),
            deleteAll: await toObject(count, "full"),
            deleteAccount: await toObject(success, "full"),
        };
    },
    api: async () => {
        const { api } = await import("./partial/api");
        return {
            findOne: await toObject(api, "full"),
            findMany: await toObject(...(await toSearch(api))),
            createOne: await toObject(api, "full"),
            updateUpdate: await toObject(api, "full"),
        };
    },
    apiKey: async () => {
        const { apiKey } = await import("./partial/apiKey");
        const { success } = await import("./partial/success");
        return {
            createOne: await toObject(apiKey, "full"),
            updateOne: await toObject(apiKey, "full"),
            deleteOne: await toObject(success, "full"),
            validateOne: await toObject(apiKey, "full"),
        };
    },
    apiVersion: async () => {
        const { apiVersion } = await import("./partial/apiVersion");
        return {
            findOne: await toObject(apiVersion, "full"),
            findMany: await toObject(...(await toSearch(apiVersion))),
            createOne: await toObject(apiVersion, "full"),
            updateOne: await toObject(apiVersion, "full"),
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
            createOne: await toObject(bookmark, "full"),
            updateOne: await toObject(bookmark, "full"),
        };
    },
    bookmarkList: async () => {
        const { bookmarkList } = await import("./partial/bookmarkList");
        return {
            findOne: await toObject(bookmarkList, "full"),
            findMany: await toObject(...(await toSearch(bookmarkList))),
            createOne: await toObject(bookmarkList, "full"),
            updateOne: await toObject(bookmarkList, "full"),
        };
    },
    chat: async () => {
        const { chat } = await import("./partial/chat");
        return {
            findOne: await toObject(chat, "full"),
            findMany: await toObject(...(await toSearch(chat))),
            createOne: await toObject(chat, "full"),
            updateOne: await toObject(chat, "full"),
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
            acceptOne: await toObject(chatInvite, "full"),
            declineOne: await toObject(chatInvite, "full"),
        };
    },
    chatMessage: async () => {
        const { chatMessage, chatMessageSearchTreeResult } = await import("./partial/chatMessage");
        const { success } = await import("./partial/success");
        return {
            findOne: await toObject(chatMessage, "full"),
            findMany: await toObject(...(await toSearch(chatMessage))),
            findTree: await toObject(chatMessageSearchTreeResult, "common"),
            createOne: await toObject(chatMessage, "full"),
            updateOne: await toObject(chatMessage, "full"),
            regenerateResponse: await toObject(success, "full"),
        };
    },
    chatParticipant: async () => {
        const { chatParticipant } = await import("./partial/chatParticipant");
        return {
            findOne: await toObject(chatParticipant, "full"),
            findMany: await toObject(...(await toSearch(chatParticipant))),
            updateOne: await toObject(chatParticipant, "full"),
        };
    },
    code: async () => {
        const { code } = await import("./partial/code");
        return {
            findOne: await toObject(code, "full"),
            findMany: await toObject(...(await toSearch(code))),
            createOne: await toObject(code, "full"),
            updateOne: await toObject(code, "full"),
        };
    },
    codeVersion: async () => {
        const { codeVersion } = await import("./partial/codeVersion");
        return {
            findOne: await toObject(codeVersion, "full"),
            findMany: await toObject(...(await toSearch(codeVersion))),
            createOne: await toObject(codeVersion, "full"),
            updateOne: await toObject(codeVersion, "full"),
        };
    },
    comment: async () => {
        const { comment, commentSearchResult } = await import("./partial/comment");
        return {
            findOne: await toObject(comment, "full"),
            findMany: await toObject(commentSearchResult, "full"),
            createOne: await toObject(comment, "full"),
            updateONe: await toObject(comment, "full"),
        };
    },
    email: async () => {
        const { email } = await import("./partial/email");
        const { success } = await import("./partial/success");
        return {
            createOne: await toObject(email, "full"),
            verify: await toObject(success, "full"),
        };
    },
    feed: async () => {
        const { homeResult, popular } = await import("./partial/feed");
        return {
            home: await toObject(homeResult, "list"),
            popular: await toObject(...(await toSearch(popular, {
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
    focusMode: async () => {
        const { activeFocusMode } = await import("./partial/activeFocusMode");
        const { focusMode } = await import("./partial/focusMode");
        return {
            findOne: await toObject(focusMode, "full"),
            findMany: await toObject(...(await toSearch(focusMode))),
            createOne: await toObject(focusMode, "full"),
            updateOne: await toObject(focusMode, "full"),
            setActive: await toObject(activeFocusMode, "full"),
        };
    },
    issue: async () => {
        const { issue } = await import("./partial/issue");
        return {
            findOne: await toObject(issue, "full"),
            findMany: await toObject(...(await toSearch(issue))),
            createOne: await toObject(issue, "full"),
            updateOne: await toObject(issue, "full"),
            closeOne: await toObject(issue, "full"),
        };
    },
    label: async () => {
        const { label } = await import("./partial/label");
        return {
            findOne: await toObject(label, "full"),
            findMany: await toObject(...(await toSearch(label))),
            createOne: await toObject(label, "full"),
            updateOne: await toObject(label, "full"),
        };
    },
    meeting: async () => {
        const { meeting } = await import("./partial/meeting");
        return {
            findOne: await toObject(meeting, "full"),
            findMany: await toObject(...(await toSearch(meeting))),
            createOne: await toObject(meeting, "full"),
            updateOne: await toObject(meeting, "full"),
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
            acceptOne: await toObject(meetingInvite, "full"),
            declineOne: await toObject(meetingInvite, "full"),
        };
    },
    member: async () => {
        const { member } = await import("./partial/member");
        return {
            findOne: await toObject(member, "full"),
            findMany: await toObject(...(await toSearch(member))),
            updateOne: await toObject(member, "full"),
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
            acceptOne: await toObject(memberInvite, "full"),
            declineOne: await toObject(memberInvite, "full"),
        };
    },
    note: async () => {
        const { note } = await import("./partial/note");
        return {
            findOne: await toObject(note, "full"),
            findMany: await toObject(...(await toSearch(note))),
            createOne: await toObject(note, "full"),
            updateOne: await toObject(note, "full"),
        };
    },
    noteVersion: async () => {
        const { noteVersion } = await import("./partial/noteVersion");
        return {
            findOne: await toObject(noteVersion, "full"),
            findMany: await toObject(...(await toSearch(noteVersion))),
            createOne: await toObject(noteVersion, "full"),
            updateOne: await toObject(noteVersion, "full"),
        };
    },
    notification: async () => {
        const { notification, notificationSettings } = await import("./partial/notification");
        const { success } = await import("./partial/success");
        const { count } = await import("./partial/count");
        return {
            findOne: await toObject(notification, "full"),
            findMany: await toObject(...(await toSearch(notification))),
            markAsRead: await toObject(success, "full"),
            markAllAsRead: await toObject(count, "full"),
            getSettings: await toObject(notificationSettings, "full"),
            updateSettings: await toObject(notificationSettings, "full"),
        };
    },
    notificationSubscription: async () => {
        const { notificationSubscription } = await import("./partial/notificationSubscription");
        return {
            findOne: await toObject(notificationSubscription, "full"),
            findMany: await toObject(...(await toSearch(notificationSubscription))),
            createOne: await toObject(notificationSubscription, "full"),
            updateOne: await toObject(notificationSubscription, "full"),
        };
    },
    phone: async () => {
        const { phone } = await import("./partial/phone");
        const { success } = await import("./partial/success");
        return {
            createOne: await toObject(phone, "full"),
            verify: await toObject(success, "full"),
            validate: await toObject(success, "full"),
        };
    },
    post: async () => {
        const { post } = await import("./partial/post");
        return {
            findOne: await toObject(post, "full"),
            findMany: await toObject(...(await toSearch(post))),
            createOne: await toObject(post, "full"),
            updateOne: await toObject(post, "full"),
        };
    },
    project: async () => {
        const { project } = await import("./partial/project");
        return {
            findOne: await toObject(project, "full"),
            findMany: await toObject(...(await toSearch(project))),
            createOne: await toObject(project, "full"),
            updateOne: await toObject(project, "full"),
        };
    },
    projectVersion: async () => {
        const { projectVersion } = await import("./partial/projectVersion");
        return {
            findOne: await toObject(projectVersion, "full"),
            findMany: await toObject(...(await toSearch(projectVersion))),
            createOne: await toObject(projectVersion, "full"),
            updateOne: await toObject(projectVersion, "full"),
        };
    },
    projectVersionDirectory: async () => {
        const { projectVersionDirectory } = await import("./partial/projectVersionDirectory");
        return {
            findOne: await toObject(projectVersionDirectory, "full"),
            findMany: await toObject(...(await toSearch(projectVersionDirectory))),
            createOne: await toObject(projectVersionDirectory, "full"),
            updateOne: await toObject(projectVersionDirectory, "full"),
        };
    },
    pullRequest: async () => {
        const { pullRequest } = await import("./partial/pullRequest");
        return {
            findOne: await toObject(pullRequest, "full"),
            findMany: await toObject(...(await toSearch(pullRequest))),
            createOne: await toObject(pullRequest, "full"),
            updateOne: await toObject(pullRequest, "full"),
        };
    },
    pushDevice: async () => {
        const { pushDevice } = await import("./partial/pushDevice");
        const { success } = await import("./partial/success");
        return {
            findMany: await toObject(...(await toSearch(pushDevice))),
            createOne: await toObject(pushDevice, "full"),
            updateOne: await toObject(pushDevice, "full"),
            testOne: await toObject(success, "full"),
        };
    },
    question: async () => {
        const { question } = await import("./partial/question");
        return {
            findOne: await toObject(question, "full"),
            findMany: await toObject(...(await toSearch(question))),
            createOne: await toObject(question, "full"),
            updateOne: await toObject(question, "full"),
        };
    },
    questionAnswer: async () => {
        const { questionAnswer } = await import("./partial/questionAnswer");
        return {
            findOne: await toObject(questionAnswer, "full"),
            findMany: await toObject(...(await toSearch(questionAnswer))),
            createOne: await toObject(questionAnswer, "full"),
            updateOne: await toObject(questionAnswer, "full"),
            acceptOne: await toObject(questionAnswer, "full"),
        };
    },
    quiz: async () => {
        const { quiz } = await import("./partial/quiz");
        return {
            findOne: await toObject(quiz, "full"),
            findMany: await toObject(...(await toSearch(quiz))),
            createOne: await toObject(quiz, "full"),
            updateOne: await toObject(quiz, "full"),
        };
    },
    quizAttempt: async () => {
        const { quizAttempt } = await import("./partial/quizAttempt");
        return {
            findOne: await toObject(quizAttempt, "full"),
            findMany: await toObject(...(await toSearch(quizAttempt))),
            createOne: await toObject(quizAttempt, "full"),
            updateOne: await toObject(quizAttempt, "full"),
        };
    },
    quizQuestionResponse: async () => {
        const { quizQuestionResponse } = await import("./partial/quizQuestionResponse");
        return {
            findOne: await toObject(quizQuestionResponse, "full"),
            findMany: await toObject(...(await toSearch(quizQuestionResponse))),
        };
    },
    reaction: async () => {
        const { reaction } = await import("./partial/reaction");
        const { success } = await import("./partial/success");
        return {
            findMany: await toObject(...(await toSearch(reaction))),
            createOne: await toObject(success, "full"),
        };
    },
    reminder: async () => {
        const { reminder } = await import("./partial/reminder");
        return {
            findOne: await toObject(reminder, "full"),
            findMany: await toObject(...(await toSearch(reminder))),
            createOne: await toObject(reminder, "full"),
            updateOne: await toObject(reminder, "full"),
        };
    },
    reminderList: async () => {
        const { reminderList } = await import("./partial/reminderList");
        return {
            createOne: await toObject(reminderList, "full"),
            updateOne: await toObject(reminderList, "full"),
        };
    },
    report: async () => {
        const { report } = await import("./partial/report");
        return {
            findOne: await toObject(report, "full"),
            findMany: await toObject(...(await toSearch(report))),
            createOne: await toObject(report, "full"),
            updateOne: await toObject(report, "full"),
        };
    },
    reportResponse: async () => {
        const { reportResponse } = await import("./partial/reportResponse");
        return {
            findOne: await toObject(reportResponse, "full"),
            findMany: await toObject(...(await toSearch(reportResponse))),
            createOne: await toObject(reportResponse, "full"),
            updateOne: await toObject(reportResponse, "full"),
        };
    },
    reputationHistory: async () => {
        const { reputationHistory } = await import("./partial/reputationHistory");
        return {
            findOne: await toObject(reputationHistory, "full"),
            findMany: await toObject(...(await toSearch(reputationHistory))),
        };
    },
    resourceList: async () => {
        const { resourceList } = await import("./partial/resourceList");
        return {
            findOne: await toObject(resourceList, "full"),
            findMany: await toObject(...(await toSearch(resourceList))),
            createOne: await toObject(resourceList, "full"),
            updateOne: await toObject(resourceList, "full"),
        };
    },
    role: async () => {
        const { role } = await import("./partial/role");
        return {
            findOne: await toObject(role, "full"),
            findMany: await toObject(...(await toSearch(role))),
            createOne: await toObject(role, "full"),
            updateOne: await toObject(role, "full"),
        };
    },
    routine: async () => {
        const { routine } = await import("./partial/routine");
        return {
            findOne: await toObject(routine, "full"),
            findMany: await toObject(...(await toSearch(routine))),
            createOne: await toObject(routine, "full"),
            updateOne: await toObject(routine, "full"),
        };
    },
    routineVersion: async () => {
        const { routineVersion } = await import("./partial/routineVersion");
        return {
            findOne: await toObject(routineVersion, "full"),
            findMany: await toObject(...(await toSearch(routineVersion))),
            createOne: await toObject(routineVersion, "full"),
            updateOne: await toObject(routineVersion, "full"),
        };
    },
    runProject: async () => {
        const { runProject } = await import("./partial/runProject");
        const { count } = await import("./partial/count");
        return {
            findOne: await toObject(runProject, "full"),
            findMany: await toObject(...(await toSearch(runProject))),
            createOne: await toObject(runProject, "full"),
            updateOne: await toObject(runProject, "full"),
        };
    },
    runRoutine: async () => {
        const { runRoutine } = await import("./partial/runRoutine");
        const { count } = await import("./partial/count");
        return {
            findOne: await toObject(runRoutine, "full"),
            findMany: await toObject(...(await toSearch(runRoutine))),
            createOne: await toObject(runRoutine, "full"),
            updateOne: await toObject(runRoutine, "full"),
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
            createOne: await toObject(schedule, "full"),
            updateOne: await toObject(schedule, "full"),
        };
    },
    standard: async () => {
        const { standard } = await import("./partial/standard");
        return {
            findOne: await toObject(standard, "full"),
            findMany: await toObject(...(await toSearch(standard))),
            createOne: await toObject(standard, "full"),
            updateOne: await toObject(standard, "full"),
        };
    },
    standardVersion: async () => {
        const { standardVersion } = await import("./partial/standardVersion");
        return {
            findOne: await toObject(standardVersion, "full"),
            findMany: await toObject(...(await toSearch(standardVersion))),
            createOne: await toObject(standardVersion, "full"),
            updateOne: await toObject(standardVersion, "full"),
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
            createOne: await toObject(tag, "full"),
            updateOne: await toObject(tag, "full"),
        };
    },
    task: async () => {
        const { checkTaskStatusesResult } = await import("./partial/task");
        const { success } = await import("./partial/success");
        return {
            startLlmTask: await toObject(success, "full"),
            startRunTask: await toObject(success, "full"),
            cancelTask: await toObject(success, "full"),
            checkStatuses: await toObject(checkTaskStatusesResult, "full"),
        };
    },
    team: async () => {
        const { team } = await import("./partial/team");
        return {
            findOne: await toObject(team, "full"),
            findMany: await toObject(...(await toSearch(team))),
            createOne: await toObject(team, "full"),
            updateOne: await toObject(team, "full"),
        };
    },
    transfer: async () => {
        const { transfer } = await import("./partial/transfer");
        return {
            findOne: await toObject(transfer, "full"),
            findMany: await toObject(...(await toSearch(transfer))),
            updateOne: await toObject(transfer, "full"),
            requestSendOne: await toObject(transfer, "full"),
            requestReceiveOne: await toObject(transfer, "full"),
            cancelOne: await toObject(transfer, "full"),
            acceptOne: await toObject(transfer, "full"),
            denyOne: await toObject(transfer, "full"),
        };
    },
    translate: async () => {
        const { translate } = await import("./partial/translate");
        return {
            translate: await toObject(translate, "full"),
        };
    },
    unions: async () => {
        const { projectOrRoutine } = await import("./partial/projectOrRoutine");
        const { projectOrTeam } = await import("./partial/projectOrTeam");
        const { runProjectOrRunRoutine } = await import("./partial/runProjectOrRunRoutine");
        return {
            projectOrRoutine: await toObject(...(await toSearch(projectOrRoutine, {
                pageInfo: {
                    hasNextPage: true,
                    endCursorProject: true,
                    endCursorRoutine: true,
                },
            }))),
            projectOrTeam: await toObject(...(await toSearch(projectOrTeam, {
                pageInfo: {
                    hasNextPage: true,
                    endCursorProject: true,
                    endCursorTeam: true,
                },
            }))),
            runProjectOrRunRoutine: await toObject(...(await toSearch(runProjectOrRunRoutine, {
                pageInfo: {
                    hasNextPage: true,
                    endCursorRunProject: true,
                    endCursorRunRoutine: true,
                },
            }))),
        };
    },
    user: async () => {
        const { user, profile } = await import("./partial/user");
        return {
            profile: await toObject(profile, "full"),
            findOne: await toObject(user, "full"),
            findMany: await toObject(...(await toSearch(user))),
            botCreateOne: await toObject(user, "full"),
            botUpdateOne: await toObject(user, "full"),
            profileUpdate: await toObject(profile, "full"),
            profileEmailUpdate: await toObject(profile, "full"),
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
            updateOne: await toObject(wallet, "full"),
        };
    },
};
