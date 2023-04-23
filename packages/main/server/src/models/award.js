import { awardNames, AwardSortBy, MaxObjects } from "@local/consts";
import i18next from "i18next";
import { defaultPermissions } from "../utils";
const __typename = "Award";
const suppFields = ["title", "description"];
export const AwardModel = ({
    __typename,
    delegate: (prisma) => prisma.award,
    display: {
        select: () => ({ id: true, category: true, progress: true }),
        label: (select, languages) => {
            const { name, nameVariables } = awardNames[select.category](select.progress);
            if (!name)
                return "";
            return i18next.t(`award:${name}`, { lng: languages[0], ...(nameVariables ?? {}) });
        },
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            user: "User",
        },
        countFields: {},
        supplemental: {
            dbFields: ["category", "progress"],
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, objects, prisma, userData }) => {
                const titles = [];
                const descriptions = [];
                for (const award of objects) {
                    const { name, nameVariables, body, bodyVariables } = awardNames[award.category](award.progress);
                    if (userData && name)
                        titles.push(i18next.t(`award:${name}`, { lng: userData.languages[0], ...(nameVariables ?? {}) }));
                    else
                        titles.push(null);
                    if (userData && body)
                        descriptions.push(i18next.t(`award:${body}`, { lng: userData.languages[0], ...(bodyVariables ?? {}) }));
                    else
                        descriptions.push(null);
                }
                return {
                    title: titles,
                    description: descriptions,
                };
            },
        },
    },
    search: {
        defaultSort: AwardSortBy.DateUpdatedDesc,
        sortBy: AwardSortBy,
        searchFields: {
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({}),
        customQueryData: (_, user) => ({ user: { id: user.id } }),
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
//# sourceMappingURL=award.js.map