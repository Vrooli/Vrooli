import { AwardKey, awardNames, AwardSortBy, MaxObjects } from "@local/shared";
import i18next from "i18next";
import { defaultPermissions } from "../../utils";
import { AwardFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { AwardModelLogic } from "./types";

const __typename = "Award" as const;
export const AwardModel: AwardModelLogic = ({
    __typename,
    dbTable: "award",
    display: () => ({
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
    }),
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
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, objects, userData }) => {
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
    validate: () => ({
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data?.user,
        }),
        permissionsSelect: () => ({
            id: true,
            user: "User",
        }),
        permissionResolvers: defaultPermissions,
        visibility: {
            private: null, // Search method disabled
            public: null, // Search method disabled
            owner: (userId) => ({
                user: { id: userId },
            }),
        },
    }),
});
