import { ApiVersion, ApiVersionTranslation } from "@shared/consts";
import { GqlPartial } from "types";
import { apiPartial } from "./api";
import { versionYouPartial } from "./root";

export const apiVersionTranslationPartial: GqlPartial<ApiVersionTranslation> = {
    __typename: 'ApiVersionTranslation',
    full: () => ({
        id: true,
        language: true,
        details: true,
        summary: true,
    }),
}

export const apiVersionPartial: GqlPartial<ApiVersion> = {
    __typename: 'ApiVersion',
    common: () => ({
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
        you: versionYouPartial.full,
    }),
    full: () => ({
        pullRequest: pullRequestPartial.full,
        root: without(apiPartial.full, 'versions'),
        translations: apiVersionTranslationPartial.full,
        versionNotes: true,
    }),
    list: () => ({
        root: without(apiPartial.list, 'versions'),
        translations: apiVersionTranslationPartial.list,
    })
}