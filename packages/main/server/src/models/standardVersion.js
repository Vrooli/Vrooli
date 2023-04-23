import { MaxObjects, StandardVersionSortBy } from "@local/consts";
import { standardVersionValidation } from "@local/validation";
import { randomString } from "../auth/wallet";
import { noNull, shapeHelper } from "../builders";
import { bestLabel, defaultPermissions, postShapeVersion, translationShapeHelper } from "../utils";
import { sortify } from "../utils/objectTools";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../validators";
import { StandardModel } from "./standard";
const querier = () => ({
    async findMatchingStandardVersion(prisma, data, userData, uniqueToCreator, isInternal) {
        return null;
    },
    async generateName(prisma, userId, languages, data) {
        const translatedName = "";
        if (translatedName.length > 0)
            return translatedName;
        const name = `${data.standardType} ${randomString(5)}`;
        return name;
    },
});
const __typename = "StandardVersion";
const suppFields = ["you"];
export const StandardVersionModel = ({
    __typename,
    delegate: (prisma) => prisma.standard_version,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, "name", languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            comments: "Comment",
            directoryListings: "ProjectVersionDirectory",
            forks: "StandardVersion",
            pullRequest: "PullRequest",
            reports: "Report",
            root: "Standard",
        },
        prismaRelMap: {
            __typename,
            comments: "Comment",
            directoryListings: "ProjectVersionDirectory",
            forks: "StandardVersion",
            pullRequest: "PullRequest",
            reports: "Report",
            root: "Standard",
        },
        countFields: {
            commentsCount: true,
            directoryListingsCount: true,
            forksCount: true,
            reportsCount: true,
            translationsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
    mutate: {
        shape: {
            pre: async ({ createList, updateList, deleteList, prisma, userData }) => {
                await versionsCheck({
                    createList,
                    deleteList,
                    objectType: __typename,
                    prisma,
                    updateList,
                    userData,
                });
                const combined = [...createList, ...updateList.map(({ data }) => data)];
                combined.forEach(input => lineBreaksCheck(input, ["description"], "LineBreaksBio", userData.languages));
            },
            create: async ({ data, ...rest }) => {
                const { translations } = await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest });
                if (translations?.create?.length) {
                    translations.create = translations.create.map(t => {
                        t.jsonVariables = sortify(t.jsonVariables, rest.userData.languages);
                        return t;
                    });
                }
                return {
                    id: data.id,
                    default: noNull(data.default),
                    isPrivate: noNull(data.isPrivate),
                    isComplete: noNull(data.isComplete),
                    isFile: noNull(data.isFile),
                    props: sortify(data.props, rest.userData.languages),
                    standardType: data.standardType,
                    versionLabel: data.versionLabel,
                    versionNotes: noNull(data.versionNotes),
                    yup: data.yup ? sortify(data.yup, rest.userData.languages) : undefined,
                    ...(await shapeHelper({ relation: "directoryListings", relTypes: ["Connect"], isOneToOne: false, isRequired: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childStandardVersions", data, ...rest })),
                    ...(await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "standardVersion", data, ...rest })),
                    ...(await shapeHelper({ relation: "root", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: true, objectType: "Standard", parentRelationshipName: "versions", data, ...rest })),
                    translations,
                };
            },
            update: async ({ data, ...rest }) => {
                const { translations } = await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest });
                if (translations?.update?.length) {
                    translations.update = translations.update.map(t => {
                        t.data = {
                            ...t.data,
                            jsonVariables: sortify(t.data.jsonVariables, rest.userData.languages),
                        };
                        return t;
                    });
                }
                if (translations?.create?.length) {
                    translations.create = translations.create.map(t => {
                        t.jsonVariables = sortify(t.jsonVariables, rest.userData.languages);
                        return t;
                    });
                }
                return {
                    default: noNull(data.default),
                    isPrivate: noNull(data.isPrivate),
                    isComplete: noNull(data.isComplete),
                    isFile: noNull(data.isFile),
                    props: data.props ? sortify(data.props, rest.userData.languages) : undefined,
                    standardType: noNull(data.standardType),
                    versionLabel: noNull(data.versionLabel),
                    versionNotes: noNull(data.versionNotes),
                    yup: data.yup ? sortify(data.yup, rest.userData.languages) : undefined,
                    ...(await shapeHelper({ relation: "directoryListings", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childStandardVersions", data, ...rest })),
                    ...(await shapeHelper({ relation: "resourceList", relTypes: ["Create", "Update"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "standardVersion", data, ...rest })),
                    ...(await shapeHelper({ relation: "root", relTypes: ["Update"], isOneToOne: true, isRequired: true, objectType: "Standard", parentRelationshipName: "versions", data, ...rest })),
                    translations,
                };
            },
            post: async (params) => {
                await postShapeVersion({ ...params, objectType: __typename });
            },
        },
        yup: standardVersionValidation,
    },
    query: querier(),
    search: {
        defaultSort: StandardVersionSortBy.DateCompletedDesc,
        sortBy: StandardVersionSortBy,
        searchFields: {
            completedTimeFrame: true,
            createdByIdRoot: true,
            createdTimeFrame: true,
            isCompleteWithRoot: true,
            maxBookmarksRoot: true,
            maxScoreRoot: true,
            maxViewsRoot: true,
            minBookmarksRoot: true,
            minScoreRoot: true,
            minViewsRoot: true,
            ownedByOrganizationIdRoot: true,
            ownedByUserIdRoot: true,
            reportId: true,
            rootId: true,
            standardType: true,
            tagsRoot: true,
            translationLanguages: true,
            updatedTimeFrame: true,
            userId: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transDescriptionWrapped",
                { root: "tagsWrapped" },
                { root: "labelsWrapped" },
                { root: "nameWrapped" },
            ],
        }),
        customQueryData: () => ({ root: { isInternal: true } }),
    },
    validate: {
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, languages) => data.isPrivate === false &&
            data.isDeleted === false &&
            StandardModel.validate.isPublic(data.root, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => StandardModel.validate.owner(data.root, userId),
        permissionsSelect: () => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: ["Standard", ["versions"]],
        }),
        permissionResolvers: defaultPermissions,
        visibility: {
            private: {
                isDeleted: false,
                root: { isDeleted: false },
                OR: [
                    { isPrivate: true },
                    { root: { isPrivate: true } },
                ],
            },
            public: {
                isDeleted: false,
                root: { isDeleted: false },
                AND: [
                    { isPrivate: false },
                    { root: { isPrivate: false } },
                ],
            },
            owner: (userId) => ({
                root: StandardModel.validate.visibility.owner(userId),
            }),
        },
    },
});
//# sourceMappingURL=standardVersion.js.map