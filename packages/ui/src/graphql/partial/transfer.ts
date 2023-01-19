import { TransferYou } from "@shared/consts";
import { GqlPartial } from "types";

export const transferYouPartial: GqlPartial<TransferYou> = {
    __typename: 'TransferYou',
    full: () => ({
        canDelete: true,
        canEdit: true,
    }),
}

export const listTransferFields = ['Transfer', `{
    id
}`] as const;
export const transferFields = ['Transfer', `{
    id
}`] as const;