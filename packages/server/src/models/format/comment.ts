import { CommentModelLogic } from "../base/types";
import { Formatter } from "../types";

export const CommentFormat: Formatter<CommentModelLogic> = {
    gqlRelMap: {
        __typename: "Comment",
        owner: {
            ownedByUser: "User",
            ownedByOrganization: "Organization",
        },
        commentedOn: {
            apiVersion: "ApiVersion",
            issue: "Issue",
            noteVersion: "NoteVersion",
            post: "Post",
            projectVersion: "ProjectVersion",
            pullRequest: "PullRequest",
            question: "Question",
            questionAnswer: "QuestionAnswer",
            routineVersion: "RoutineVersion",
            smartContractVersion: "SmartContractVersion",
            standardVersion: "StandardVersion",
        },
        reports: "Report",
        bookmarkedBy: "User",
    },
    prismaRelMap: {
        __typename: "Comment",
        ownedByUser: "User",
        ownedByOrganization: "Organization",
        apiVersion: "ApiVersion",
        issue: "Issue",
        noteVersion: "NoteVersion",
        parent: "Comment",
        post: "Post",
        projectVersion: "ProjectVersion",
        pullRequest: "PullRequest",
        question: "Question",
        questionAnswer: "QuestionAnswer",
        routineVersion: "RoutineVersion",
        smartContractVersion: "SmartContractVersion",
        standardVersion: "StandardVersion",
        reports: "Report",
        bookmarkedBy: "User",
        reactions: "Reaction",
        parents: "Comment",
    },
    joinMap: { bookmarkedBy: "user" },
    countFields: {
        reportsCount: true,
        translationsCount: true,
    },
};
