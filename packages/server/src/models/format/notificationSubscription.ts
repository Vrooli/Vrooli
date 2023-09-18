import { NotificationSubscriptionModelLogic } from "../base/types";
import { Formatter } from "../types";

export const NotificationSubscriptionFormat: Formatter<NotificationSubscriptionModelLogic> = {
    gqlRelMap: {
        __typename: "NotificationSubscription",
        object: {
            api: "Api",
            comment: "Comment",
            issue: "Issue",
            meeting: "Meeting",
            note: "Note",
            organization: "Organization",
            project: "Project",
            pullRequest: "PullRequest",
            question: "Question",
            quiz: "Quiz",
            report: "Report",
            routine: "Routine",
            schedule: "Schedule",
            smartContract: "SmartContract",
            standard: "Standard",
        },
    },
    prismaRelMap: {
        __typename: "NotificationSubscription",
        api: "Api",
        comment: "Comment",
        issue: "Issue",
        meeting: "Meeting",
        note: "Note",
        organization: "Organization",
        project: "Project",
        pullRequest: "PullRequest",
        question: "Question",
        quiz: "Quiz",
        report: "Report",
        routine: "Routine",
        schedule: "Schedule",
        smartContract: "SmartContract",
        standard: "Standard",
        subscriber: "User",
    },
    countFields: {},
};
