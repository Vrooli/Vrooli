import { MaxObjects, SmartContractVersionSortBy, smartContractVersionValidation } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { bestTranslation, defaultPermissions, getEmbeddableString, postShapeVersion, translationShapeHelper } from "../../utils";
import { preShapeVersion } from "../../utils/preShapeVersion";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../../validators";
import { SmartContractVersionFormat } from "../format/smartContractVersion";
import { ModelLogic } from "../types";
import { SmartContractModel } from "./smartContract";
import { SmartContractVersionModelLogic } from "./types";

const __typename = "SmartContractVersion" as const;
const suppFields = ["you"] as const;
export const SmartContractVersionModel: ModelLogic<SmartContractVersionModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.smart_contract_version,
    display: {
        label: {
            select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
        embed: {
            select: () => ({
                id: true,
                root: { select: { tags: { select: { tag: true } } } },
                translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } },
            }),
            get: ({ root, translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return getEmbeddableString({
                    name: trans?.name,
                    tags: (root as any).tags.map(({ tag }) => tag),
                    description: trans?.description,
                }, languages[0]);
            },
        },
    },
    format: SmartContractVersionFormat,
    mutate: {
        shape: {
            pre: async (params) => {
                const { createList, updateList, deleteList, prisma, userData } = params;
                await versionsCheck({
                    createList,
                    deleteList,
                    objectType: __typename,
                    prisma,
                    updateList,
                    userData,
                });
                [...createList, ...updateList].forEach(input => lineBreaksCheck(input, ["description"], "LineBreaksBio", userData.languages));
                const maps = preShapeVersion({ createList, updateList, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                content: data.content,
                contractType: data.contractType,
                default: noNull(data.default),
                isPrivate: noNull(data.isPrivate),
                isComplete: noNull(data.isComplete),
                versionLabel: data.versionLabel,
                versionNotes: noNull(data.versionNotes),
                ...(await shapeHelper({ relation: "directoryListings", relTypes: ["Connect"], isOneToOne: false, isRequired: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childSmartContractVersions", data, ...rest })),
                ...(await shapeHelper({ relation: "resourceList", relTypes: ["Create"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "smartContractVersion", data, ...rest })),
                ...(await shapeHelper({ relation: "root", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: true, objectType: "SmartContract", parentRelationshipName: "versions", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                content: noNull(data.content),
                contractType: noNull(data.contractType),
                default: noNull(data.default),
                isPrivate: noNull(data.isPrivate),
                isComplete: noNull(data.isComplete),
                versionLabel: noNull(data.versionLabel),
                versionNotes: noNull(data.versionNotes),
                ...(await shapeHelper({ relation: "directoryListings", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "childSmartContractVersions", data, ...rest })),
                ...(await shapeHelper({ relation: "resourceList", relTypes: ["Create", "Update"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "smartContractVersion", data, ...rest })),
                ...(await shapeHelper({ relation: "root", relTypes: ["Update"], isOneToOne: true, isRequired: true, objectType: "SmartContract", parentRelationshipName: "versions", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
            post: async (params) => {
                await postShapeVersion({ ...params, objectType: __typename });
            },
        },
        yup: smartContractVersionValidation,
    },
    search: {
        defaultSort: SmartContractVersionSortBy.DateUpdatedDesc,
        sortBy: SmartContractVersionSortBy,
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
            tagsRoot: true,
            translationLanguages: true,
            updatedTimeFrame: true,
            userId: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transDescriptionWrapped",
                "transNameWrapped",
                { root: "tagsWrapped" },
                { root: "labelsWrapped" },
            ],
        }),
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
    validate: {
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, languages) => data.isPrivate === false &&
            data.isDeleted === false &&
            SmartContractModel.validate.isPublic(data.root as any, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => SmartContractModel.validate.owner(data.root as any, userId),
        permissionsSelect: (...params) => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: ["SmartContract", ["versions"]],
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
                root: SmartContractModel.validate.visibility.owner(userId),
            }),
        },
    },
});
