import { MaxObjects } from "@local/shared";
import { defaultPermissions } from "../../utils";
import { ReactionSummaryFormat } from "../formats";
import { ReactionSummaryModelLogic } from "./types";

const __typename = "ReactionSummary" as const;
export const ReactionSummaryModel: ReactionSummaryModelLogic = ({
    __typename,
    dbTable: "stats_api",
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
        owner: () => ({
            Team: null,
            User: null,
        }),
        isDeleted: () => false,
        isPublic: () => false,
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
