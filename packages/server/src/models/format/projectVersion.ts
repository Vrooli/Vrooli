import { ProjectVersionModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ProjectVersionFormat: Formatter<ProjectVersionModelLogic> = {
    gqlRelMap: {
        __typename: "ProjectVersion",
        comments: "Comment",
        directories: "ProjectVersionDirectory",
        directoryListings: "ProjectVersionDirectory",
        forks: "Project",
        pullRequest: "PullRequest",
        reports: "Report",
        root: "Project",
        // 'runs.project': 'RunProject', //TODO
    },
    prismaRelMap: {
        __typename: "ProjectVersion",
        comments: "Comment",
        directories: "ProjectVersionDirectory",
        directoryListings: "ProjectVersionDirectory",
        pullRequest: "PullRequest",
        reports: "Report",
        resourceList: "ResourceList",
        root: "Project",
        forks: "Project",
        runProjects: "RunProject",
        suggestedNextByProject: "ProjectVersion",
    },
    joinMap: {
        suggestedNextByProject: "toProjectVersion",
    },
    countFields: {
        commentsCount: true,
        directoriesCount: true,
        directoryListingsCount: true,
        forksCount: true,
        reportsCount: true,
        runProjectsCount: true,
        translationsCount: true,
    },
};
