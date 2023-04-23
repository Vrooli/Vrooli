import { LINKS, SubscribableObject } from "@local/consts";
import i18next from "i18next";
import { selectHelper, toPartialGqlInfo } from "../builders";
import { CustomError, logger } from "../events";
import { getLogic } from "../getters";
import { OrganizationModel, PushDeviceModel, subscribableMapper } from "../models";
import { initializeRedis } from "../redisConn";
import { sendMail } from "./email";
import { findRecipientsAndLimit, updateNotificationSettings } from "./notificationSettings";
import { sendPush } from "./push";
export const isObjectSubscribable = (objectType) => {
    return objectType in SubscribableObject;
};
const push = async ({ bodyKey, category, link, prisma, titleKey, users, }) => {
    const devicesAndLimits = await findRecipientsAndLimit(category, prisma, users.map(u => u.userId));
    const userTitles = {};
    const userBodies = {};
    for (const user of users) {
        const lng = user.languages.length > 0 ? user.languages[0] : "en";
        const title = titleKey ? i18next.t(`notify:${titleKey}`, { lng, ...(user.titleVariables ?? {}) }) : undefined;
        const body = bodyKey ? i18next.t(`notify:${bodyKey}`, { lng, ...(user.bodyVariables ?? {}) }) : undefined;
        if (!title && !body)
            throw new CustomError("0362", "InternalError", user.languages);
        userTitles[user.userId] = title ?? `${body.substring(0, 10)}...`;
        userBodies[user.userId] = body ?? title;
    }
    const icon = "https://vrooli.com/Logo.png";
    try {
        const client = await initializeRedis();
        for (let i = 0; i < users.length; i++) {
            for (const delay of users[i].delays ?? [0]) {
                const { pushDevices, emails, phoneNumbers, dailyLimit } = devicesAndLimits[i];
                let currSilent = users[i].silent ?? false;
                const currTitle = userTitles[users[i].userId];
                const currBody = userBodies[users[i].userId];
                const count = await client.incr(`notification:${users[i].userId}:${category}`);
                if (dailyLimit && count > dailyLimit)
                    currSilent = true;
                if (!currSilent) {
                    for (const device of pushDevices) {
                        try {
                            const subscription = {
                                endpoint: device.endpoint,
                                keys: {
                                    p256dh: device.p256dh,
                                    auth: device.auth,
                                },
                            };
                            const payload = { body: currBody, icon, link, title: currTitle };
                            sendPush(subscription, payload, delay);
                        }
                        catch (err) {
                            logger.error("Error sending push notification", { trace: "0306" });
                        }
                    }
                    if (emails.length && currTitle) {
                        sendMail(emails.map(e => e.emailAddress), currTitle, currBody, "", delay);
                    }
                }
            }
        }
        await prisma.notification.createMany({
            data: users.map(({ userId }) => ({
                category,
                title: userTitles[userId],
                description: userBodies[userId],
                link,
                imgLink: icon,
                userId,
            })),
        });
    }
    catch (error) {
        logger.error("Error occured while connecting or accessing redis server", { trace: "0305", error });
    }
};
const getEventStartLabel = (date) => {
    const now = new Date();
    const diff = date.getTime() - now.getTime();
    if (diff < 0)
        return "now";
    if (diff < 1000 * 60)
        return "in a few seconds";
    if (diff < 1000 * 60 * 60)
        return `in ${Math.round(diff / (1000 * 60))} minutes`;
    if (diff < 1000 * 60 * 60 * 24)
        return `in ${Math.round(diff / (1000 * 60 * 60))} hours`;
    return `in ${Math.round(diff / (1000 * 60 * 60 * 24))} days`;
};
const replaceLabels = async (bodyVariables, titleVariables, silent, prisma, languages, users) => {
    const labelRegex = /<Label\|([A-z]+):([0-9\-]+)>/;
    const result = users.map(u => ({ ...u, bodyVariables, titleVariables, silent }));
    let labelTranslations = {};
    const findTranslations = async (objectType, objectId) => {
        if (Object.keys(labelTranslations).length > 0)
            return;
        const { delegate, display } = getLogic(["delegate", "display"], objectType, languages, "replaceLabels");
        const labels = await delegate(prisma).findUnique({
            where: { id: objectId },
            select: display.select(),
        });
        labelTranslations = labels ?? {};
    };
    if (titleVariables) {
        for (const key of Object.keys(titleVariables)) {
            if (typeof titleVariables[key] === "string") {
                const match = titleVariables[key].match(labelRegex);
                if (match) {
                    await findTranslations(match[1], match[2]);
                    const { display } = getLogic(["display"], match[1], languages, "replaceLabels");
                    for (let i = 0; i < result.length; i++) {
                        result[i][key] = result[i][key].replace(match[0], display.label(labelTranslations, result[i].languages));
                    }
                }
            }
        }
    }
    if (bodyVariables) {
        for (const key of Object.keys(bodyVariables)) {
            if (typeof bodyVariables[key] === "string") {
                const match = bodyVariables[key].match(labelRegex);
                if (match) {
                    await findTranslations(match[1], match[2]);
                    const { display } = getLogic(["display"], match[1], languages, "replaceLabels");
                    for (let i = 0; i < result.length; i++) {
                        result[i][key] = result[i][key].replace(match[0], display.label(labelTranslations, result[i].languages));
                    }
                }
            }
        }
    }
    return result;
};
const NotifyResult = ({ bodyKey, bodyVariables, category, languages, link, prisma, silent, titleKey, titleVariables, }) => ({
    toUser: async (userId) => {
        const users = await replaceLabels(bodyVariables, titleVariables, silent, prisma, languages, [{ languages, userId }]);
        await push({ bodyKey, category, link, prisma, titleKey, users });
    },
    toUsers: async (userIds) => {
        const users = await replaceLabels(bodyVariables, titleVariables, silent, prisma, languages, userIds.map(data => ({
            languages,
            userId: typeof data === "string" ? data : data.userId,
            delays: typeof data === "string" ? undefined : data.delays,
        })));
        await push({ bodyKey, category, link, prisma, titleKey, users });
    },
    toOrganization: async (organizationId, excludeUserId) => {
        const adminData = await OrganizationModel.query.findAdminInfo(prisma, organizationId, excludeUserId);
        const users = await replaceLabels(bodyVariables, titleVariables, silent, prisma, languages, adminData.map(({ id, languages }) => ({
            languages,
            userId: id,
        })));
        await push({ bodyKey, category, link, prisma, titleKey, users });
    },
    toOwner: async (owner, excludeUserId) => {
        if (owner.__typename === "User") {
            await NotifyResult({ bodyKey, bodyVariables, category, languages, link, prisma, silent, titleKey, titleVariables }).toUser(owner.id);
        }
        else if (owner.__typename === "Organization") {
            await NotifyResult({ bodyKey, bodyVariables, category, languages, link, prisma, silent, titleKey, titleVariables }).toOrganization(owner.id, excludeUserId);
        }
    },
    toSubscribers: async (objectType, objectId, excludeUserId) => {
        const batchSize = 100;
        let skip = 0;
        let currentBatchSize = 0;
        do {
            const batch = await prisma.notification_subscription.findMany({
                where: {
                    AND: [
                        { [subscribableMapper[objectType]]: { id: objectId } },
                        { subscriberId: { not: excludeUserId ?? undefined } },
                    ],
                },
                select: { subscriberId: true, silent: true },
                skip,
                take: batchSize,
            });
            skip += batchSize;
            currentBatchSize = batch.length;
            const users = await replaceLabels(bodyVariables, titleVariables, silent, prisma, languages, batch.map(({ subscriberId, silent }) => ({
                languages,
                silent,
                userId: subscriberId,
            })));
            await push({ bodyKey, category, link, prisma, titleKey, users });
        } while (currentBatchSize === batchSize);
    },
});
export const Notify = (prisma, languages) => ({
    registerPushDevice: async ({ endpoint, p256dh, auth, expires, userData, info }) => {
        const partialInfo = toPartialGqlInfo(info, PushDeviceModel.format.gqlRelMap, userData.languages, true);
        let select;
        let result = {};
        try {
            select = selectHelper(partialInfo)?.select;
            const device = await prisma.push_device.findUnique({
                where: { endpoint },
                select: { id: true },
            });
            if (device) {
                result = await prisma.push_device.update({
                    where: { id: device.id },
                    data: { auth, p256dh, expires },
                    select,
                });
            }
            else {
                result = await prisma.push_device.create({
                    data: {
                        endpoint,
                        auth,
                        p256dh,
                        expires,
                        user: { connect: { id: userData.id } },
                    },
                    select,
                });
            }
            return result;
        }
        catch (error) {
            throw new CustomError("0452", "InternalError", userData.languages, { error, select, result });
        }
    },
    unregisterPushDevice: async (deviceId, userId) => {
        const device = await prisma.push_device.findUnique({
            where: { id: deviceId },
            select: { userId: true },
        });
        if (!device || device.userId !== userId)
            throw new CustomError("0307", "PushDeviceNotFound", languages);
        const deletedDevice = await prisma.push_device.delete({ where: { id: deviceId } });
        return { __typename: "Success", success: Boolean(deletedDevice) };
    },
    updateSettings: async (settings, userId) => {
        await updateNotificationSettings(settings, prisma, userId);
    },
    pushApiOutOfCredits: () => NotifyResult({
        bodyKey: "ApiOutOfCreditsBody",
        category: "AccountCreditsOrApi",
        languages,
        prisma,
        titleKey: "ApiOutOfCreditsTitle",
    }),
    pushAward: (awardName, awardDescription) => NotifyResult({
        bodyKey: "AwardEarnedBody",
        bodyVariables: { awardName, awardDescription },
        category: "Award",
        languages,
        link: "/awards",
        prisma,
        titleKey: "AwardEarnedTitle",
    }),
    pushIssueStatusChange: (issueId, objectId, objectType, status) => NotifyResult({
        bodyKey: `IssueStatus${status}Body`,
        bodyVariables: { issueName: `<Label|Issue:${issueId}>`, objectName: `<Label|${objectType}:${objectId}>` },
        category: "IssueStatus",
        languages,
        link: `/issues/${issueId}`,
        prisma,
        titleKey: `IssueStatus${status}Title`,
    }),
    pushNewDeviceSignIn: () => NotifyResult({
        bodyKey: "NewDeviceBody",
        category: "Security",
        languages,
        prisma,
        titleKey: "NewDeviceTitle",
    }),
    pushNewEmailVerification: () => NotifyResult({
        bodyKey: "NewEmailVerificationBody",
        category: "Security",
        languages,
        prisma,
        titleKey: "NewEmailVerificationTitle",
    }),
    pushNewQuestionOnObject: (objectType, objectId, questionId) => NotifyResult({
        bodyKey: "NewQuestionOnObjectBody",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
        category: "NewQuestionOrIssue",
        languages,
        link: `/questions/${questionId}`,
        prisma,
        titleKey: "NewQuestionOnObjectTitle",
    }),
    pushNewObjectInOrganization: (objectType, objectId, organizationId) => NotifyResult({
        bodyKey: "NewObjectInOrganizationBody",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, organizationName: `<Label|Organization:${organizationId}>` },
        category: "NewObjectInOrganization",
        languages,
        link: `/${LINKS[objectType]}/${objectId}`,
        prisma,
        titleKey: "NewObjectInOrganizationTitle",
        titleVariables: { organizationName: `<Label|Organization:${organizationId}>` },
    }),
    pushNewObjectInProject: (objectType, objectId, projectId) => NotifyResult({
        bodyKey: "NewObjectInProjectBody",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, projectName: `<Label|Project:${projectId}>` },
        category: "NewObjectInProject",
        languages,
        link: `/${LINKS[objectType]}/${objectId}`,
        prisma,
        titleKey: "NewObjectInProjectTitle",
        titleVariables: { projectName: `<Label|Project:${projectId}>` },
    }),
    pushObjectReceivedBookmark: (objectType, objectId, totalBookmarks) => NotifyResult({
        bodyKey: "ObjectReceivedBookmarkBody",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, count: totalBookmarks },
        category: "ObjectActivity",
        languages,
        link: `/${LINKS[objectType]}/${objectId}`,
        prisma,
        titleKey: "ObjectReceivedBookmarkTitle",
    }),
    pushObjectReceivedComment: (objectType, objectId, totalComments) => NotifyResult({
        bodyKey: "ObjectReceivedCommentBody",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, count: totalComments },
        category: "ObjectActivity",
        languages,
        link: `/${LINKS[objectType]}/${objectId}?comments`,
        prisma,
        titleKey: "ObjectReceivedCommentTitle",
    }),
    pushObjectReceivedCopy: (objectType, objectId) => NotifyResult({
        category: "ObjectActivity",
        languages,
        link: `/${LINKS[objectType]}/${objectId}`,
        prisma,
        titleKey: "ObjectReceivedCopyTitle",
        titleVariables: { objectName: `<Label|${objectType}:${objectId}>` },
    }),
    pushObjectReceivedUpvote: (objectType, objectId, totalScore) => NotifyResult({
        bodyKey: "ObjectReceivedUpvoteBody",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, count: totalScore },
        category: "ObjectActivity",
        languages,
        link: `/${LINKS[objectType]}/${objectId}`,
        prisma,
        titleKey: "ObjectReceivedUpvoteTitle",
    }),
    pushReportStatusChange: (reportId, objectId, objectType, status) => NotifyResult({
        bodyKey: `ReportStatus${status}Body`,
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
        category: "ReportStatus",
        languages,
        link: `/${LINKS[objectType]}/${objectId}/reports/${reportId}`,
        prisma,
        titleKey: `ReportStatus${status}Title`,
    }),
    pushPullRequestStatusChange: (reportId, objectId, objectType, status) => NotifyResult({
        bodyKey: `PullRequestStatus${status}Body`,
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
        category: "PullRequestStatus",
        languages,
        link: `/${LINKS[objectType]}/${objectId}/pulls/${reportId}`,
        prisma,
        titleKey: `PullRequestStatus${status}Title`,
    }),
    pushQuestionActivity: (questionId) => NotifyResult({
        bodyKey: "QuestionActivityBody",
        bodyVariables: { objectName: `<Label|Question:${questionId}>` },
        category: "QuestionActivity",
        languages,
        link: `/questions/${questionId}`,
        prisma,
        titleKey: "QuestionActivityTitle",
    }),
    pushRunStartedAutomatically: (runId) => NotifyResult({
        bodyKey: "RunStartedAutomaticallyBody",
        bodyVariables: { runName: `<Label|RunRoutine:${runId}>` },
        category: "Run",
        languages,
        link: `/runs/${runId}`,
        prisma,
        titleKey: "RunStartedAutomaticallyTitle",
    }),
    pushRunCompletedAutomatically: (runId) => NotifyResult({
        bodyKey: "RunCompletedAutomaticallyBody",
        bodyVariables: { runName: `<Label|RunRoutine:${runId}>` },
        category: "Run",
        languages,
        link: `/runs/${runId}`,
        prisma,
        titleKey: "RunCompletedAutomaticallyTitle",
    }),
    pushRunFailedAutomatically: (runId) => NotifyResult({
        bodyKey: "RunFailedAutomaticallyBody",
        bodyVariables: { runName: `<Label|RunRoutine:${runId}>` },
        category: "Run",
        languages,
        link: `/runs/${runId}`,
        prisma,
        titleKey: "RunFailedAutomaticallyTitle",
    }),
    pushScheduleReminder: (scheduleForId, scheduleForType, startTime) => NotifyResult({
        bodyKey: "ScheduleUserBody",
        bodyVariables: { title: `<Label|${scheduleForType}:${scheduleForId}>`, startLabel: getEventStartLabel(startTime) },
        category: "Schedule",
        languages,
        link: scheduleForType === "Meeting" ? `/meeting/${scheduleForId}` : undefined,
        prisma,
    }),
    pushStreakReminder: (timeToReset) => NotifyResult({
        bodyKey: "StreakReminderBody",
        bodyVariables: { endLabel: getEventStartLabel(timeToReset) },
        category: "Streak",
        languages,
        prisma,
    }),
    pushStreakBroken: () => NotifyResult({
        bodyKey: "StreakBrokenBody",
        category: "Streak",
        languages,
        prisma,
        titleKey: "StreakBrokenTitle",
    }),
    pushTransferRequestSend: (transferId, objectType, objectId) => NotifyResult({
        bodyKey: "TransferRequestSendBody",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
        category: "Transfer",
        languages,
        link: `/${LINKS[objectType]}/transfers/${transferId}`,
        prisma,
        titleKey: "TransferRequestSendTitle",
    }),
    pushTransferRequestReceive: (transferId, objectType, objectId) => NotifyResult({
        bodyKey: "TransferRequestReceiveBody",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
        category: "Transfer",
        languages,
        link: `/${LINKS[objectType]}/transfers/${transferId}`,
        prisma,
        titleKey: "TransferRequestReceiveTitle",
    }),
    pushTransferAccepted: (objectType, objectId) => NotifyResult({
        bodyKey: "TransferAcceptedTitle",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
        category: "Transfer",
        languages,
        link: `/${LINKS[objectType]}/${objectId}`,
        prisma,
        titleKey: "TransferAcceptedTitle",
    }),
    pushTransferRejected: (objectType, objectId) => NotifyResult({
        bodyKey: "TransferRejectedTitle",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
        category: "Transfer",
        languages,
        link: `/${LINKS[objectType]}/${objectId}`,
        prisma,
        titleKey: "TransferRejectedTitle",
    }),
    pushUserInvite: (friendUsername) => NotifyResult({
        bodyKey: "UserInviteBody",
        bodyVariables: { friendUsername },
        category: "UserInvite",
        languages,
        prisma,
    }),
});
//# sourceMappingURL=notify.js.map