import { MaxObjects } from "@vrooli/shared";
import { defaultPermissions } from "../../validators/permissions.js";
import { ReactionSummaryFormat } from "../formats.js";
import { type ReactionSummaryModelLogic } from "./types.js";

const __typename = "ReactionSummary" as const;
export const ReactionSummaryModel: ReactionSummaryModelLogic = ({
    __typename,
    dbTable: "reaction_summary",
    display: () => ({
        label: {
            select: () => ({ id: true, emoji: true, count: true }),
            get: (select) => `${select.emoji} (${select.count})`,
        },
    }),
    format: ReactionSummaryFormat,
    search: undefined,
    // Never queried directly, so should be fine without validation
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({}),
        permissionResolvers: defaultPermissions,
        owner: (_data, _userId) => ({
            Team: null,
            User: null,
        }),
        isDeleted: () => false,
        isPublic: (_data, _getParentInfo?) => false,
        // These are never searched directly, so all search methods can be disabled
        visibility: {
            own: null,
            ownOrPublic: null,
            ownPrivate: null,
            ownPublic: null,
            public: null,
        },
    }),
});
