import { ProjectVersion, ProjectVersionTranslation, ProjectVersionYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { projectPartial } from "./project";
import { projectVersionDirectoryPartial } from "./projectVersionDirectory";
import { pullRequestPartial } from "./pullRequest";
import { versionYouPartial } from "./root";
import { runProjectPartial } from "./runProject";

export const projectVersionTranslationPartial: GqlPartial<ProjectVersionTranslation> = {
    __typename: 'ProjectVersionTranslation',
    full: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
}

export const projectVersionYouPartial: GqlPartial<ProjectVersionYou> = {
    __typename: 'ProjectVersionYou',
    common: {
        canComment: true,
        canCopy: true,
        canDelete: true,
        canEdit: true,
        canReport: true,
        canUse: true,
        canView: true,
    },
    full: {
        runs: () => relPartial(runProjectPartial, 'full', { omit: 'projectVersion' }),
    },
    list: {
        runs: () => relPartial(runProjectPartial, 'list', { omit: 'projectVersion' }),
    },
}

export const projectVersionPartial: GqlPartial<ProjectVersion> = {
    __typename: 'ProjectVersion',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        directoriesCount: true,
        isLatest: true,
        isPrivate: true,
        reportsCount: true,
        runsCount: true,
        simplicity: true,
        versionIndex: true,
        versionLabel: true,
        you: () => relPartial(versionYouPartial, 'full'),
    },
    full: {
        directories: () => relPartial(projectVersionDirectoryPartial, 'full', { omit: 'projectVersion '}),
        pullRequest: () => relPartial(pullRequestPartial, 'full'),
        root: () => relPartial(projectPartial, 'full', { omit: 'versions' }),
        translations: () => relPartial(projectVersionTranslationPartial, 'full'),
        versionNotes: true,
    },
    list: {
        directories: () => relPartial(projectVersionDirectoryPartial, 'list', { omit: 'projectVersion '}),
        root: () => relPartial(projectPartial, 'list', { omit: 'versions' }),
        translations: () => relPartial(projectVersionTranslationPartial, 'list'),
    }
}