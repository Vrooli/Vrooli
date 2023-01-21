import { StandardVersion, StandardVersionTranslation } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { pullRequestPartial } from "./pullRequest";
import { resourceListPartial } from "./resourceList";
import { versionYouPartial } from "./root";
import { standardPartial } from "./standard";

export const standardVersionTranslationPartial: GqlPartial<StandardVersionTranslation> = {
    __typename: 'StandardVersionTranslation',
    full: {
        id: true,
        language: true,
        description: true,
        jsonVariable: true,
    },
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
        pullRequest: () => relPartial(pullRequestPartial, 'full'),
        resourceList: () => relPartial(resourceListPartial, 'full'),
        root: () => relPartial(standardPartial, 'full', { omit: 'versions' }),
        translations: () => relPartial(standardVersionTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(standardVersionTranslationPartial, 'list'),
    }
}