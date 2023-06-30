import { ProjectModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Project" as const;
export const ProjectFormat: Formatter<ProjectModelLogic> = {
    gqlRelMap: {
        __typename,
        createdBy: "User",
        issues: "Issue",
        labels: "Label",
        owner: {
            ownedByUser: "User",
            ownedByOrganization: "Organization",
        },
        parent: "Project",
        pullRequests: "PullRequest",
        questions: "Question",
        quizzes: "Quiz",
        bookmarkedBy: "User",
        tags: "Tag",
        transfers: "Transfer",
        versions: "ProjectVersion",
    },
    prismaRelMap: {
        __typename,
        createdBy: "User",
        ownedByOrganization: "Organization",
        ownedByUser: "User",
        parent: "ProjectVersion",
        issues: "Issue",
        labels: "Label",
        tags: "Tag",
        versions: "ProjectVersion",
        bookmarkedBy: "User",
        pullRequests: "PullRequest",
        stats: "StatsProject",
        questions: "Question",
        transfers: "Transfer",
        quizzes: "Quiz",
    },
    joinMap: { labels: "label", bookmarkedBy: "user", tags: "tag" },
    countFields: {
        issuesCount: true,
        labelsCount: true,
        pullRequestsCount: true,
        questionsCount: true,
        quizzesCount: true,
        transfersCount: true,
        versionsCount: true,
    },
};
