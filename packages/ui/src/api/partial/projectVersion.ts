import { ProjectVersion, ProjectVersionTranslation, ProjectVersionYou } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";
import { versionYouPartial } from "./root";

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
        runs: () => relPartial(require('./runProject').runProjectPartial, 'full', { omit: 'projectVersion' }),
    },
    list: {
        runs: () => relPartial(require('./runProject').runProjectPartial, 'list', { omit: 'projectVersion' }),
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
        directories: () => relPartial(require('./projectVersionDirectory').projectVersionDirectoryPartial, 'full', { omit: 'projectVersion '}),
        pullRequest: () => relPartial(require('./pullRequest').pullRequestPartial, 'full'),
        root: () => relPartial(require('./project').projectPartial, 'full', { omit: 'versions' }),
        translations: () => relPartial(projectVersionTranslationPartial, 'full'),
        versionNotes: true,
    },
    list: {
        directories: () => relPartial(require('./projectVersionDirectory').projectVersionDirectoryPartial, 'list', { omit: 'projectVersion '}),
        root: () => relPartial(require('./project').projectPartial, 'list', { omit: 'versions' }),
        translations: () => relPartial(projectVersionTranslationPartial, 'list'),
    }
}