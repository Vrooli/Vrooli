import { toObject } from "./utils.js";

export const endpoints = {
    actions: async () => {
        const { copyResult } = await import("./partial/actions.js");
        const { success } = await import("./partial/success.js");
        const { count } = await import("./partial/count.js");
        return {
            copy: await toObject(copyResult, "full"),
            deleteOne: await toObject(success, "full"),
            deleteMany: await toObject(count, "full"),
            deleteAll: await toObject(count, "full"),
            deleteAccount: await toObject(success, "full"),
        };
    },
    api: async () => {
        const { api } = await import("./partial/api.js");
        return {
            findOne: await toObject(api, "full"),
            findMany: await toObject(api, "list", { asSearch: true }),
            createOne: await toObject(api, "full"),
            updateOne: await toObject(api, "full"),
        };
    },
    apiKey: async () => {
        const { apiKey } = await import("./partial/apiKey.js");
        const { success } = await import("./partial/success.js");
        return {
            createOne: await toObject(apiKey, "full"),
            updateOne: await toObject(apiKey, "full"),
            deleteOne: await toObject(success, "full"),
            validateOne: await toObject(apiKey, "full"),
        };
    },
    apiVersion: async () => {
        const { apiVersion } = await import("./partial/apiVersion.js");
        return {
            findOne: await toObject(apiVersion, "full"),
            findMany: await toObject(apiVersion, "list", { asSearch: true }),
            createOne: await toObject(apiVersion, "full"),
            updateOne: await toObject(apiVersion, "full"),
        };
    },
    auth: async () => {
        const { session } = await import("./partial/session.js");
        const { success } = await import("./partial/success.js");
        const { walletInit, walletComplete } = await import("./partial/wallet.js");
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
        const { award } = await import("./partial/award.js");
        return {
            findMany: await toObject(award, "list", { asSearch: true }),
        };
    },
    bookmark: async () => {
        const { bookmark } = await import("./partial/bookmark.js");
        return {
            findOne: await toObject(bookmark, "full"),
            findMany: await toObject(bookmark, "list", { asSearch: true }),
            createOne: await toObject(bookmark, "full"),
            updateOne: await toObject(bookmark, "full"),
        };
    },
    bookmarkList: async () => {
        const { bookmarkList } = await import("./partial/bookmarkList.js");
        return {
            findOne: await toObject(bookmarkList, "full"),
            findMany: await toObject(bookmarkList, "list", { asSearch: true }),
            createOne: await toObject(bookmarkList, "full"),
            updateOne: await toObject(bookmarkList, "full"),
        };
    },
    chat: async () => {
        const { chat } = await import("./partial/chat.js");
        return {
            findOne: await toObject(chat, "full"),
            findMany: await toObject(chat, "list", { asSearch: true }),
            createOne: await toObject(chat, "full"),
            updateOne: await toObject(chat, "full"),
        };
    },
    chatInvite: async () => {
        const { chatInvite } = await import("./partial/chatInvite.js");
        return {
            findOne: await toObject(chatInvite, "full"),
            findMany: await toObject(chatInvite, "list", { asSearch: true }),
            createOne: await toObject(chatInvite, "full"),
            createMany: await toObject(chatInvite, "full"),
            updateOne: await toObject(chatInvite, "full"),
            updateMany: await toObject(chatInvite, "full"),
            acceptOne: await toObject(chatInvite, "full"),
            declineOne: await toObject(chatInvite, "full"),
        };
    },
    chatMessage: async () => {
        const { chatMessage, chatMessageSearchTreeResult } = await import("./partial/chatMessage.js");
        const { success } = await import("./partial/success.js");
        return {
            findOne: await toObject(chatMessage, "full"),
            findMany: await toObject(chatMessage, "list", { asSearch: true }),
            findTree: await toObject(chatMessageSearchTreeResult, "common"),
            createOne: await toObject(chatMessage, "full"),
            updateOne: await toObject(chatMessage, "full"),
            regenerateResponse: await toObject(success, "full"),
        };
    },
    chatParticipant: async () => {
        const { chatParticipant } = await import("./partial/chatParticipant.js");
        return {
            findOne: await toObject(chatParticipant, "full"),
            findMany: await toObject(chatParticipant, "list", { asSearch: true }),
            updateOne: await toObject(chatParticipant, "full"),
        };
    },
    code: async () => {
        const { code } = await import("./partial/code.js");
        return {
            findOne: await toObject(code, "full"),
            findMany: await toObject(code, "list", { asSearch: true }),
            createOne: await toObject(code, "full"),
            updateOne: await toObject(code, "full"),
        };
    },
    codeVersion: async () => {
        const { codeVersion } = await import("./partial/codeVersion.js");
        return {
            findOne: await toObject(codeVersion, "full"),
            findMany: await toObject(codeVersion, "list", { asSearch: true }),
            createOne: await toObject(codeVersion, "full"),
            updateOne: await toObject(codeVersion, "full"),
        };
    },
    comment: async () => {
        const { comment, commentSearchResult } = await import("./partial/comment.js");
        return {
            findOne: await toObject(comment, "full"),
            findMany: await toObject(commentSearchResult, "full"),
            createOne: await toObject(comment, "full"),
            updateOne: await toObject(comment, "full"),
        };
    },
    email: async () => {
        const { email } = await import("./partial/email.js");
        const { success } = await import("./partial/success.js");
        return {
            createOne: await toObject(email, "full"),
            verify: await toObject(success, "full"),
        };
    },
    feed: async () => {
        const { homeResult, popular } = await import("./partial/feed.js");
        return {
            home: await toObject(homeResult, "list"),
            popular: await toObject(popular, "list", {
                asSearch: true,
                searchOverrides: {
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
                },
            }),
        };
    },
    focusMode: async () => {
        const { activeFocusMode } = await import("./partial/activeFocusMode.js");
        const { focusMode } = await import("./partial/focusMode.js");
        return {
            findOne: await toObject(focusMode, "full"),
            findMany: await toObject(focusMode, "list", { asSearch: true }),
            createOne: await toObject(focusMode, "full"),
            updateOne: await toObject(focusMode, "full"),
            setActive: await toObject(activeFocusMode, "full"),
        };
    },
    issue: async () => {
        const { issue } = await import("./partial/issue.js");
        return {
            findOne: await toObject(issue, "full"),
            findMany: await toObject(issue, "list", { asSearch: true }),
            createOne: await toObject(issue, "full"),
            updateOne: await toObject(issue, "full"),
            closeOne: await toObject(issue, "full"),
        };
    },
    label: async () => {
        const { label } = await import("./partial/label.js");
        return {
            findOne: await toObject(label, "full"),
            findMany: await toObject(label, "list", { asSearch: true }),
            createOne: await toObject(label, "full"),
            updateOne: await toObject(label, "full"),
        };
    },
    meeting: async () => {
        const { meeting } = await import("./partial/meeting.js");
        return {
            findOne: await toObject(meeting, "full"),
            findMany: await toObject(meeting, "list", { asSearch: true }),
            createOne: await toObject(meeting, "full"),
            updateOne: await toObject(meeting, "full"),
        };
    },
    meetingInvite: async () => {
        const { meetingInvite } = await import("./partial/meetingInvite.js");
        return {
            findOne: await toObject(meetingInvite, "full"),
            findMany: await toObject(meetingInvite, "list", { asSearch: true }),
            createOne: await toObject(meetingInvite, "full"),
            createMany: await toObject(meetingInvite, "full"),
            updateOne: await toObject(meetingInvite, "full"),
            updateMany: await toObject(meetingInvite, "full"),
            acceptOne: await toObject(meetingInvite, "full"),
            declineOne: await toObject(meetingInvite, "full"),
        };
    },
    member: async () => {
        const { member } = await import("./partial/member.js");
        return {
            findOne: await toObject(member, "full"),
            findMany: await toObject(member, "list", { asSearch: true }),
            updateOne: await toObject(member, "full"),
        };
    },
    memberInvite: async () => {
        const { memberInvite } = await import("./partial/memberInvite.js");
        return {
            findOne: await toObject(memberInvite, "full"),
            findMany: await toObject(memberInvite, "list", { asSearch: true }),
            createOne: await toObject(memberInvite, "full"),
            createMany: await toObject(memberInvite, "full"),
            updateOne: await toObject(memberInvite, "full"),
            updateMany: await toObject(memberInvite, "full"),
            acceptOne: await toObject(memberInvite, "full"),
            declineOne: await toObject(memberInvite, "full"),
        };
    },
    note: async () => {
        const { note } = await import("./partial/note.js");
        return {
            findOne: await toObject(note, "full"),
            findMany: await toObject(note, "list", { asSearch: true }),
            createOne: await toObject(note, "full"),
            updateOne: await toObject(note, "full"),
        };
    },
    noteVersion: async () => {
        const { noteVersion } = await import("./partial/noteVersion.js");
        return {
            findOne: await toObject(noteVersion, "full"),
            findMany: await toObject(noteVersion, "list", { asSearch: true }),
            createOne: await toObject(noteVersion, "full"),
            updateOne: await toObject(noteVersion, "full"),
        };
    },
    notification: async () => {
        const { notification, notificationSettings } = await import("./partial/notification.js");
        const { success } = await import("./partial/success.js");
        const { count } = await import("./partial/count.js");
        return {
            findOne: await toObject(notification, "full"),
            findMany: await toObject(notification, "list", { asSearch: true }),
            markAsRead: await toObject(success, "full"),
            markAllAsRead: await toObject(count, "full"),
            getSettings: await toObject(notificationSettings, "full"),
            updateSettings: await toObject(notificationSettings, "full"),
        };
    },
    notificationSubscription: async () => {
        const { notificationSubscription } = await import("./partial/notificationSubscription.js");
        return {
            findOne: await toObject(notificationSubscription, "full"),
            findMany: await toObject(notificationSubscription, "list", { asSearch: true }),
            createOne: await toObject(notificationSubscription, "full"),
            updateOne: await toObject(notificationSubscription, "full"),
        };
    },
    phone: async () => {
        const { phone } = await import("./partial/phone.js");
        const { success } = await import("./partial/success.js");
        return {
            createOne: await toObject(phone, "full"),
            verify: await toObject(success, "full"),
            validate: await toObject(success, "full"),
        };
    },
    post: async () => {
        const { post } = await import("./partial/post.js");
        return {
            findOne: await toObject(post, "full"),
            findMany: await toObject(post, "list", { asSearch: true }),
            createOne: await toObject(post, "full"),
            updateOne: await toObject(post, "full"),
        };
    },
    project: async () => {
        const { project } = await import("./partial/project.js");
        return {
            findOne: await toObject(project, "full"),
            findMany: await toObject(project, "list", { asSearch: true }),
            createOne: await toObject(project, "full"),
            updateOne: await toObject(project, "full"),
        };
    },
    projectVersion: async () => {
        const { projectVersion } = await import("./partial/projectVersion.js");
        return {
            findOne: await toObject(projectVersion, "full"),
            findMany: await toObject(projectVersion, "list", { asSearch: true }),
            createOne: await toObject(projectVersion, "full"),
            updateOne: await toObject(projectVersion, "full"),
        };
    },
    projectVersionDirectory: async () => {
        const { projectVersionDirectory } = await import("./partial/projectVersionDirectory.js");
        return {
            findOne: await toObject(projectVersionDirectory, "full"),
            findMany: await toObject(projectVersionDirectory, "list", { asSearch: true }),
            createOne: await toObject(projectVersionDirectory, "full"),
            updateOne: await toObject(projectVersionDirectory, "full"),
        };
    },
    pullRequest: async () => {
        const { pullRequest } = await import("./partial/pullRequest.js");
        return {
            findOne: await toObject(pullRequest, "full"),
            findMany: await toObject(pullRequest, "list", { asSearch: true }),
            createOne: await toObject(pullRequest, "full"),
            updateOne: await toObject(pullRequest, "full"),
        };
    },
    pushDevice: async () => {
        const { pushDevice } = await import("./partial/pushDevice.js");
        const { success } = await import("./partial/success.js");
        return {
            findMany: await toObject(pushDevice, "list", { asSearch: true }),
            createOne: await toObject(pushDevice, "full"),
            updateOne: await toObject(pushDevice, "full"),
            testOne: await toObject(success, "full"),
        };
    },
    question: async () => {
        const { question } = await import("./partial/question.js");
        return {
            findOne: await toObject(question, "full"),
            findMany: await toObject(question, "list", { asSearch: true }),
            createOne: await toObject(question, "full"),
            updateOne: await toObject(question, "full"),
        };
    },
    questionAnswer: async () => {
        const { questionAnswer } = await import("./partial/questionAnswer.js");
        return {
            findOne: await toObject(questionAnswer, "full"),
            findMany: await toObject(questionAnswer, "list", { asSearch: true }),
            createOne: await toObject(questionAnswer, "full"),
            updateOne: await toObject(questionAnswer, "full"),
            acceptOne: await toObject(questionAnswer, "full"),
        };
    },
    quiz: async () => {
        const { quiz } = await import("./partial/quiz.js");
        return {
            findOne: await toObject(quiz, "full"),
            findMany: await toObject(quiz, "list", { asSearch: true }),
            createOne: await toObject(quiz, "full"),
            updateOne: await toObject(quiz, "full"),
        };
    },
    quizAttempt: async () => {
        const { quizAttempt } = await import("./partial/quizAttempt.js");
        return {
            findOne: await toObject(quizAttempt, "full"),
            findMany: await toObject(quizAttempt, "list", { asSearch: true }),
            createOne: await toObject(quizAttempt, "full"),
            updateOne: await toObject(quizAttempt, "full"),
        };
    },
    quizQuestionResponse: async () => {
        const { quizQuestionResponse } = await import("./partial/quizQuestionResponse.js");
        return {
            findOne: await toObject(quizQuestionResponse, "full"),
            findMany: await toObject(quizQuestionResponse, "list", { asSearch: true }),
        };
    },
    reaction: async () => {
        const { reaction } = await import("./partial/reaction.js");
        const { success } = await import("./partial/success.js");
        return {
            findMany: await toObject(reaction, "list", { asSearch: true }),
            createOne: await toObject(success, "full"),
        };
    },
    reminder: async () => {
        const { reminder } = await import("./partial/reminder.js");
        return {
            findOne: await toObject(reminder, "full"),
            findMany: await toObject(reminder, "list", { asSearch: true }),
            createOne: await toObject(reminder, "full"),
            updateOne: await toObject(reminder, "full"),
        };
    },
    reminderList: async () => {
        const { reminderList } = await import("./partial/reminderList.js");
        return {
            createOne: await toObject(reminderList, "full"),
            updateOne: await toObject(reminderList, "full"),
        };
    },
    report: async () => {
        const { report } = await import("./partial/report.js");
        return {
            findOne: await toObject(report, "full"),
            findMany: await toObject(report, "list", { asSearch: true }),
            createOne: await toObject(report, "full"),
            updateOne: await toObject(report, "full"),
        };
    },
    reportResponse: async () => {
        const { reportResponse } = await import("./partial/reportResponse.js");
        return {
            findOne: await toObject(reportResponse, "full"),
            findMany: await toObject(reportResponse, "list", { asSearch: true }),
            createOne: await toObject(reportResponse, "full"),
            updateOne: await toObject(reportResponse, "full"),
        };
    },
    reputationHistory: async () => {
        const { reputationHistory } = await import("./partial/reputationHistory.js");
        return {
            findOne: await toObject(reputationHistory, "full"),
            findMany: await toObject(reputationHistory, "list", { asSearch: true }),
        };
    },
    resource: async () => {
        const { resource } = await import("./partial/resource.js");
        return {
            findOne: await toObject(resource, "full"),
            findMany: await toObject(resource, "list", { asSearch: true }),
            createOne: await toObject(resource, "full"),
            updateOne: await toObject(resource, "full"),
        };
    },
    resourceList: async () => {
        const { resourceList } = await import("./partial/resourceList.js");
        return {
            findOne: await toObject(resourceList, "full"),
            findMany: await toObject(resourceList, "list", { asSearch: true }),
            createOne: await toObject(resourceList, "full"),
            updateOne: await toObject(resourceList, "full"),
        };
    },
    role: async () => {
        const { role } = await import("./partial/role.js");
        return {
            findOne: await toObject(role, "full"),
            findMany: await toObject(role, "list", { asSearch: true }),
            createOne: await toObject(role, "full"),
            updateOne: await toObject(role, "full"),
        };
    },
    routine: async () => {
        const { routine } = await import("./partial/routine.js");
        return {
            findOne: await toObject(routine, "full"),
            findMany: await toObject(routine, "list", { asSearch: true }),
            createOne: await toObject(routine, "full"),
            updateOne: await toObject(routine, "full"),
        };
    },
    routineVersion: async () => {
        const { routineVersion } = await import("./partial/routineVersion.js");
        return {
            findOne: await toObject(routineVersion, "full"),
            findMany: await toObject(routineVersion, "list", { asSearch: true }),
            createOne: await toObject(routineVersion, "full"),
            updateOne: await toObject(routineVersion, "full"),
        };
    },
    runProject: async () => {
        const { runProject } = await import("./partial/runProject.js");
        return {
            findOne: await toObject(runProject, "full"),
            findMany: await toObject(runProject, "list", { asSearch: true }),
            createOne: await toObject(runProject, "full"),
            updateOne: await toObject(runProject, "full"),
        };
    },
    runRoutine: async () => {
        const { runRoutine } = await import("./partial/runRoutine.js");
        return {
            findOne: await toObject(runRoutine, "full"),
            findMany: await toObject(runRoutine, "list", { asSearch: true }),
            createOne: await toObject(runRoutine, "full"),
            updateOne: await toObject(runRoutine, "full"),
        };
    },
    runRoutineIO: async () => {
        const { runRoutineIO } = await import("./partial/runRoutineIO.js");
        return {
            findMany: await toObject(runRoutineIO, "list", { asSearch: true }),
        };
    },
    schedule: async () => {
        const { schedule } = await import("./partial/schedule.js");
        return {
            findOne: await toObject(schedule, "full"),
            findMany: await toObject(schedule, "list", { asSearch: true }),
            createOne: await toObject(schedule, "full"),
            updateOne: await toObject(schedule, "full"),
        };
    },
    standard: async () => {
        const { standard } = await import("./partial/standard.js");
        return {
            findOne: await toObject(standard, "full"),
            findMany: await toObject(standard, "list", { asSearch: true }),
            createOne: await toObject(standard, "full"),
            updateOne: await toObject(standard, "full"),
        };
    },
    standardVersion: async () => {
        const { standardVersion } = await import("./partial/standardVersion.js");
        return {
            findOne: await toObject(standardVersion, "full"),
            findMany: await toObject(standardVersion, "list", { asSearch: true }),
            createOne: await toObject(standardVersion, "full"),
            updateOne: await toObject(standardVersion, "full"),
        };
    },
    statsApi: async () => {
        const { statsApi } = await import("./partial/statsApi.js");
        return {
            findMany: await toObject(statsApi, "list", { asSearch: true }),
        };
    },
    statsCode: async () => {
        const { statsCode } = await import("./partial/statsCode.js");
        return {
            findMany: await toObject(statsCode, "list", { asSearch: true }),
        };
    },
    statsProject: async () => {
        const { statsProject } = await import("./partial/statsProject.js");
        return {
            findMany: await toObject(statsProject, "list", { asSearch: true }),
        };
    },
    statsQuiz: async () => {
        const { statsQuiz } = await import("./partial/statsQuiz.js");
        return {
            findMany: await toObject(statsQuiz, "list", { asSearch: true }),
        };
    },
    statsRoutine: async () => {
        const { statsRoutine } = await import("./partial/statsRoutine.js");
        return {
            findMany: await toObject(statsRoutine, "list", { asSearch: true }),
        };
    },
    statsSite: async () => {
        const { statsSite } = await import("./partial/statsSite.js");
        return {
            findMany: await toObject(statsSite, "list", { asSearch: true }),
        };
    },
    statsStandard: async () => {
        const { statsStandard } = await import("./partial/statsStandard.js");
        return {
            findMany: await toObject(statsStandard, "list", { asSearch: true }),
        };
    },
    statsTeam: async () => {
        const { statsTeam } = await import("./partial/statsTeam.js");
        return {
            findMany: await toObject(statsTeam, "list", { asSearch: true }),
        };
    },
    statsUser: async () => {
        const { statsUser } = await import("./partial/statsUser.js");
        return {
            findMany: await toObject(statsUser, "list", { asSearch: true }),
        };
    },
    tag: async () => {
        const { tag } = await import("./partial/tag.js");
        return {
            findOne: await toObject(tag, "full"),
            findMany: await toObject(tag, "list", { asSearch: true }),
            createOne: await toObject(tag, "full"),
            updateOne: await toObject(tag, "full"),
        };
    },
    task: async () => {
        const { checkTaskStatusesResult } = await import("./partial/task.js");
        const { success } = await import("./partial/success.js");
        return {
            startLlmTask: await toObject(success, "full"),
            startRunTask: await toObject(success, "full"),
            cancelTask: await toObject(success, "full"),
            checkStatuses: await toObject(checkTaskStatusesResult, "full"),
        };
    },
    team: async () => {
        const { team } = await import("./partial/team.js");
        return {
            findOne: await toObject(team, "full"),
            findMany: await toObject(team, "list", { asSearch: true }),
            createOne: await toObject(team, "full"),
            updateOne: await toObject(team, "full"),
        };
    },
    transfer: async () => {
        const { transfer } = await import("./partial/transfer.js");
        return {
            findOne: await toObject(transfer, "full"),
            findMany: await toObject(transfer, "list", { asSearch: true }),
            updateOne: await toObject(transfer, "full"),
            requestSendOne: await toObject(transfer, "full"),
            requestReceiveOne: await toObject(transfer, "full"),
            cancelOne: await toObject(transfer, "full"),
            acceptOne: await toObject(transfer, "full"),
            denyOne: await toObject(transfer, "full"),
        };
    },
    translate: async () => {
        const { translate } = await import("./partial/translate.js");
        return {
            translate: await toObject(translate, "full"),
        };
    },
    unions: async () => {
        const { projectOrRoutine } = await import("./partial/projectOrRoutine.js");
        const { projectOrTeam } = await import("./partial/projectOrTeam.js");
        const { runProjectOrRunRoutine } = await import("./partial/runProjectOrRunRoutine.js");
        return {
            projectOrRoutine: await toObject(projectOrRoutine, "list", {
                asSearch: true,
                searchOverrides: {
                    pageInfo: {
                        hasNextPage: true,
                        endCursorProject: true,
                        endCursorRoutine: true,
                    },
                },
            }),
            projectOrTeam: await toObject(projectOrTeam, "list", {
                asSearch: true,
                searchOverrides: {
                    pageInfo: {
                        hasNextPage: true,
                        endCursorProject: true,
                        endCursorTeam: true,
                    },
                },
            }),
            runProjectOrRunRoutine: await toObject(runProjectOrRunRoutine, "list", {
                asSearch: true,
                searchOverrides: {
                    pageInfo: {
                        hasNextPage: true,
                        endCursorRunProject: true,
                        endCursorRunRoutine: true,
                    },
                },
            }),
        };
    },
    user: async () => {
        const { user, profile } = await import("./partial/user.js");
        return {
            profile: await toObject(profile, "full"),
            findOne: await toObject(user, "full"),
            findMany: await toObject(user, "list", { asSearch: true }),
            botCreateOne: await toObject(user, "full"),
            botUpdateOne: await toObject(user, "full"),
            profileUpdate: await toObject(profile, "full"),
            profileEmailUpdate: await toObject(profile, "full"),
        };
    },
    view: async () => {
        const { view } = await import("./partial/view.js");
        return {
            findMany: await toObject(view, "list", { asSearch: true }),
        };
    },
    wallet: async () => {
        const { wallet } = await import("./partial/wallet.js");
        return {
            updateOne: await toObject(wallet, "full"),
        };
    },
};
