import { NoteVersionModelLogic } from "../base/types";
import { Formatter } from "../types";

export const NoteVersionFormat: Formatter<NoteVersionModelLogic> = {
    gqlRelMap: {
        __typename: "NoteVersion",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        pullRequest: "PullRequest",
        forks: "NoteVersion",
        reports: "Report",
        root: "Note",
    },
    prismaRelMap: {
        __typename: "NoteVersion",
        root: "Note",
        forks: "Note",
        pullRequest: "PullRequest",
        comments: "Comment",
        reports: "Report",
        directoryListings: "ProjectVersionDirectory",
    },
    countFields: {
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        reportsCount: true,
    },
};
