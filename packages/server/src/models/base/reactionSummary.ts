import { defaultPermissions } from "../../utils";
import { ReactionSummaryFormat } from "../formats";
import { ReactionSummaryModelLogic } from "./types";

const __typename = "ReactionSummary" as const;
export const ReactionSummaryModel: ReactionSummaryModelLogic = ({
    __typename,
    delegate: (p) => p.stats_api,
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
        maxObjects: 0,
        permissionsSelect: () => ({}),
        permissionResolvers: defaultPermissions,
        owner: () => ({
            Organization: null,
            User: null,
        }),
        isDeleted: () => false,
        isPublic: () => false,
        visibility: {
            private: {},
            public: {},
            owner: () => ({}),
        },
    }),
});
