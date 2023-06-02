import { AwardKey, awardNames, AwardSortBy, MaxObjects } from "@local/shared";
import i18next from "i18next";
import { defaultPermissions } from "../../utils";
import { AwardFormat } from "../format/award";
import { ModelLogic } from "../types";
import { AwardModelLogic } from "./types";

const __typename = "Award" as const;
const suppFields = ["title", "description"] as const;
export const AwardModel: ModelLogic<AwardModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.award,
    display: {
        label: {
            select: () => ({ id: true, category: true, progress: true }),
            get: (select, languages) => {
                // Find name of highest tier achieved
                const { name, nameVariables } = awardNames[select.category](select.progress);
                // If key is not found, return empty string
                if (!name) return "";
                return i18next.t(`award:${name}`, { lng: languages[0], ...(nameVariables ?? {}) });
            },
        },
    },
    format: AwardFormat,
    search: {
        defaultSort: AwardSortBy.DateUpdatedDesc,
        sortBy: AwardSortBy,
        searchFields: {
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({}),
        /**
         * Can only ever search your own awards
         */
        customQueryData: (_, user) => ({ user: { id: user!.id } }),
        supplemental: {
            dbFields: ["category", "progress"],
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, objects, prisma, userData }) => {
                // Find name and description of highest tier achieved
                const titles: (string | null)[] = [];
                const descriptions: (string | null)[] = [];
                for (const award of objects) {
                    const { name, nameVariables, body, bodyVariables } = awardNames[award.category](award.progress);
                    if (userData && name) titles.push(i18next.t(`award:${name as AwardKey}`, { lng: userData!.languages[0], ...(nameVariables ?? {}) }) as any);
                    else titles.push(null);
                    if (userData && body) descriptions.push(i18next.t(`award:${body as AwardKey}`, { lng: userData!.languages[0], ...(bodyVariables ?? {}) }) as any);
                    else descriptions.push(null);
                }
                return {
                    title: titles,
                    description: descriptions,
                };
            },
        },
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data.user,
        }),
        permissionsSelect: () => ({
            id: true,
            user: "User",
        }),
        permissionResolvers: defaultPermissions,
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                user: { id: userId },
            }),
        },
    },
});
