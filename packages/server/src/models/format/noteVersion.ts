import { NoteVersionModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "NoteVersion" as const;
export const NoteVersionFormat: Formatter<NoteVersionModelLogic> = {
    gqlRelMap: {
        __typename,
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        pullRequest: "PullRequest",
        forks: "NoteVersion",
        reports: "Report",
        root: "Note",
    },
    prismaRelMap: {
        __typename,
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
