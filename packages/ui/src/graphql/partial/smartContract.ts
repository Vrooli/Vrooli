import { SmartContractYou } from "@shared/consts";
import { GqlPartial } from "types";

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

export const smartContractNameFields = ['SmartContract', `{
    id
    translatedName
}`] as const;
export const listSmartContractFields = ['SmartContract', `{
    id
}`] as const;
export const smartContractFields = ['SmartContract', `{
    id
}`] as const;