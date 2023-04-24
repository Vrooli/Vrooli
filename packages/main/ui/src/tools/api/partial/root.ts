import { VersionYou } from ":local/consts";
import { GqlPartial } from "../types";

export const versionYou: GqlPartial<VersionYou> = {
    __typename: "VersionYou",
    full: {
        canComment: true,
        canCopy: true,
        canDelete: true,
        canReport: true,
        canUpdate: true,
        canUse: true,
        canRead: true,
    },
};
