import { VersionYou } from "@shared/consts";
import { GqlPartial } from "types";

export const versionYouPartial: GqlPartial<VersionYou> = {
    __typename: 'VersionYou',
    full: {
        canComment: true,
        canCopy: true,
        canDelete: true,
        canEdit: true,
        canReport: true,
        canUse: true,
        canView: true,
    },
}