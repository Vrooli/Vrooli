import { ApiVersion, ApiVersionTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";
import { versionYouPartial } from "./root";

export const apiVersionTranslation: GqlPartial<ApiVersionTranslation> = {
    __typename: 'ApiVersionTranslation',
    common: {
        id: true,
        language: true,
        details: true,
        summary: true,
    },
    full: {},
    list: {},
}

export const apiVersion: GqlPartial<ApiVersion> = {
    __typename: 'ApiVersion',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        callLink: true,
        commentsCount: true,
        documentationLink: true,
        forksCount: true,
        isLatest: true,
        isPrivate: true,
        reportsCount: true,
        versionIndex: true,
        versionLabel: true,
        you: () => relPartial(versionYouPartial, 'full'),
    },
    full: {
        pullRequest: async () => relPartial((await import('./pullRequest')).pullRequestPartial, 'full'),
        root: async () => relPartial((await import('./api')).api, 'full', { omit: 'versions' }),
        translations: () => relPartial(apiVersionTranslation, 'full'),
        versionNotes: true,
    },
    list: {
        root: async () => relPartial((await import('./api')).api, 'list', { omit: 'versions' }),
        translations: () => relPartial(apiVersionTranslation, 'list'),
    },
    nav: {
        id: true,
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: async () => relPartial((await import('./api')).api, 'nav'),
        translations: () => relPartial(apiVersionTranslation, 'list'),
    }
}