import { type VersionYou } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";

export const versionYou: ApiPartial<VersionYou> = {
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
