import { RoutineModelLogic } from "../base/types";
import { Formatter } from "../types";

export const RoutineFormat: Formatter<RoutineModelLogic> = {
    gqlRelMap: {
        __typename: "Routine",
        createdBy: "User",
        owner: {
            ownedByUser: "User",
            ownedByOrganization: "Organization",
        },
        forks: "Routine",
        issues: "Issue",
        labels: "Label",
        parent: "Routine",
        bookmarkedBy: "User",
        tags: "Tag",
        versions: "RoutineVersion",
    },
    prismaRelMap: {
        __typename: "Routine",
        createdBy: "User",
        ownedByUser: "User",
        ownedByOrganization: "Organization",
        parent: "RoutineVersion",
        questions: "Question",
        quizzes: "Quiz",
        issues: "Issue",
        labels: "Label",
        pullRequests: "PullRequest",
        bookmarkedBy: "User",
        stats: "StatsRoutine",
        tags: "Tag",
        transfers: "Transfer",
        versions: "RoutineVersion",
        viewedBy: "View",
    },
    joinMap: { labels: "label", tags: "tag", bookmarkedBy: "user" },
    countFields: {
        forksCount: true,
        issuesCount: true,
        pullRequestsCount: true,
        questionsCount: true,
        quizzesCount: true,
        transfersCount: true,
        versionsCount: true,
    },
};
