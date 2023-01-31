import { StandardVersion, StandardVersionTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";
import { versionYouPartial } from "./root";

export const standardVersionTranslationPartial: GqlPartial<StandardVersionTranslation> = {
    __typename: 'StandardVersionTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        jsonVariable: true,
    },
    full: {},
    list: {},
}

export const standardVersionPartial: GqlPartial<StandardVersion> = {
    __typename: 'StandardVersion',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        isComplete: true,
        isFile: true,
        isLatest: true,
        isPrivate: true,
        default: true,
        standardType: true,
        props: true,
        yup: true,
        versionIndex: true,
        versionLabel: true,
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        reportsCount: true,
        you: () => relPartial(versionYouPartial, 'full'),
    },
    full: {
        versionNotes: true,
        pullRequest: async () => relPartial((await import('./pullRequest')).pullRequestPartial, 'full'),
        resourceList: async () => relPartial((await import('./resourceList')).resourceListPartial, 'full'),
        root: async () => relPartial((await import('./standard')).standardPartial, 'full', { omit: 'versions' }),
        translations: () => relPartial(standardVersionTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(standardVersionTranslationPartial, 'list'),
    },
    nav: {
        id: true,
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: async () => relPartial((await import('./standard')).standardPartial, 'nav'),
        translations: () => relPartial(standardVersionTranslationPartial, 'list'),
    }
}