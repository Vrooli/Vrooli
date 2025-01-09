import { AwardSortBy, DEFAULT_LANGUAGE, MaxObjects, TranslationKeyAward, awardNames } from "@local/shared";
import i18next from "i18next";
import { useVisibility } from "../../builders/visibilityBuilder";
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
                return i18next.t(`award:${name}`, { lng: languages && languages.length > 0 ? languages[0] : DEFAULT_LANGUAGE, ...(nameVariables ?? {}) });
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
        supplemental: {
            dbFields: ["category", "progress"],
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, objects, userData }) => {
                // Find name and description of highest tier achieved
                const titles: (string | null)[] = [];
                const descriptions: (string | null)[] = [];
                for (const award of objects) {
                    const { name, nameVariables, body, bodyVariables } = awardNames[award.category](award.progress);
                    if (userData && name) titles.push(i18next.t(`award:${name as TranslationKeyAward}`, { lng: userData!.languages[0], ...(nameVariables ?? {}) }) as any);
                    else titles.push(null);
                    if (userData && body) descriptions.push(i18next.t(`award:${body as TranslationKeyAward}`, { lng: userData!.languages[0], ...(bodyVariables ?? {}) }) as any);
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
            own: function getOwn(data) {
                return {
                    user: { id: data.userId },
                };
            },
            // Always private, so it's the same as "own"
            ownOrPublic: function getOwnOrPublic(data) {
                return useVisibility("Award", "Own", data);
            },
            // Always private, so it's the same as "own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("Award", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});
