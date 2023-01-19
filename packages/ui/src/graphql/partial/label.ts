import { LabelYou } from "@shared/consts";
import { GqlPartial } from "types";

export const labelYouPartial: GqlPartial<LabelYou> = {
    __typename: 'LabelYou',
    full: () => ({
        canDelete: true,
        canEdit: true,
    }),
}

export const labelFields = ['Label', `{
    id
}`] as const;