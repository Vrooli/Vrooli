import { defaultPermissions } from "../../utils";
import { ReactionSummaryFormat } from "../formats";
import { ModelLogic } from "../types";
import { ReactionSummaryModelLogic } from "./types";

const __typename = "ReactionSummary" as const;
const suppFields = [] as const;
export const ReactionSummaryModel: ModelLogic<ReactionSummaryModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.stats_api,
    display: {
        label: {
            select: () => ({ id: true, emoji: true, count: true }),
            get: (select) => `${select.emoji} (${select.count})`,
        },
    },
    format: ReactionSummaryFormat,
    search: undefined,
    // Never queried directly, so should be fine without validation
    validate: {
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
    },
});
