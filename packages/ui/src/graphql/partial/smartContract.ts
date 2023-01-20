import { SmartContract, SmartContractYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { labelPartial } from "./label";
import { organizationPartial } from "./organization";
import { statsSmartContractPartial } from "./statsSmartContract";
import { tagPartial } from "./tag";
import { userPartial } from "./user";

export const smartContractYouPartial: GqlPartial<SmartContractYou> = {
    __typename: 'SmartContractYou',
    full: {
        canDelete: true,
        canEdit: true,
        canStar: true,
        canTransfer: true,
        canView: true,
        canVote: true,
        isStarred: true,
        isUpvoted: true,
        isViewed: true,
    },
}

export const smartContractPartial: GqlPartial<SmartContract> = {
    __typename: 'SmartContract',
    common: {
        id: true,
        created_at: true,
        isPrivate: true,
        issuesCount: true,
        labels: () => relPartial(labelPartial, 'list'),
        owner: {
            __union: {
                Organization: () => relPartial(organizationPartial, 'nav'),
                User: () => relPartial(userPartial, 'nav'),
            }
        },
        permissions: true,
        questionsCount: true,
        score: true,
        stars: true,
        tags: () => relPartial(tagPartial, 'list'),
        transfersCount: true,
        views: true,
        you: () => relPartial(smartContractYouPartial, 'full'),
    },
    full: {
        versions: () => relPartial(smartContractVersionPartial, 'full', { omit: 'root' }),
        stats: () => relPartial(statsSmartContractPartial, 'full'),
    },
    list: {
        versions: () => relPartial(smartContractVersionPartial, 'list'),
    }
}