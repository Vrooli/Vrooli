import { RunProjectYou } from "@shared/consts";
import { GqlPartial } from "types";

export const runProjectYouPartial: GqlPartial<RunProjectYou> = {
    __typename: 'RunProjectYou',
    full: {
        canDelete: true,
        canEdit: true,
        canView: true,
    },
}

export const listRunProjectFields = ['RunProject', `{
    id
}`] as const;
export const runProjectFields = ['RunProject', `{
    id
}`] as const;