import { IssueFor, IssueSortBy, issueValidation, MaxObjects } from "@local/shared";
import { Prisma } from "@prisma/client";
import { bestTranslation, defaultPermissions, getEmbeddableString, labelShapeHelper, oneIsPublic, translationShapeHelper } from "../../utils";
import { preShapeEmbeddableTranslatable } from "../../utils/preShapeEmbeddableTranslatable";
import { getSingleTypePermissions } from "../../validators";
import { IssueFormat } from "../format/issue";
import { ModelLogic } from "../types";
import { BookmarkModel } from "./bookmark";
import { ReactionModel } from "./reaction";
import { IssueModelLogic } from "./types";

const forMapper: { [key in IssueFor]: keyof Prisma.issueUpsertArgs["create"] } = {
    Api: "api",
    Note: "note",
    Organization: "organization",
    Project: "project",
    Routine: "routine",
    SmartContract: "smartContract",
    Standard: "standard",
};

const __typename = "Issue" as const;
const suppFields = ["you"] as const;
export const IssueModel: ModelLogic<IssueModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.issue,
    display: {
        label: {
            select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
        embed: {
            select: () => ({ id: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } } }),
            get: ({ translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return getEmbeddableString({
                    description: trans.description,
                    name: trans.name,
                }, languages[0]);
            },
        },
    },
    format: IssueFormat,
    mutate: {
        shape: {
            pre: async ({ createList, updateList }) => {
                const maps = preShapeEmbeddableTranslatable({ createList, updateList, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                referencedVersion: data.referencedVersionIdConnect ? { connect: { id: data.referencedVersionIdConnect } } : undefined,
                [forMapper[data.issueFor]]: { connect: { id: data.forConnect } },
                ...(await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Issue", relation: "labels", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                ...(await labelShapeHelper({ relTypes: ["Connect", "Disconnect", "Create"], parentType: "Issue", relation: "labels", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
        },
        yup: issueValidation,
    },
    search: {
        defaultSort: IssueSortBy.ScoreDesc,
        sortBy: IssueSortBy,
        searchFields: {
            apiId: true,
            closedById: true,
            createdById: true,
            createdTimeFrame: true,
            minScore: true,
            minBookmarks: true,
            minViews: true,
            noteId: true,
            organizationId: true,
            projectId: true,
            referencedVersionId: true,
            routineId: true,
            smartContractId: true,
            standardId: true,
            status: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({ OR: ["transDescriptionWrapped", "transNameWrapped"] }),
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        isBookmarked: await BookmarkModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        reaction: await ReactionModel.query.getReactions(prisma, userData?.id, ids, __typename),
                    },
                };
            },
        },
    },
    validate: {
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.issueSelect>(data, [
            ["api", "Api"],
            ["organization", "Organization"],
            ["note", "Note"],
            ["project", "Project"],
            ["routine", "Routine"],
            ["smartContract", "SmartContract"],
            ["standard", "Standard"],
        ], languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data.createdBy,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            api: "Api",
            createdBy: "User",
            organization: "Organization",
            note: "Note",
            project: "Project",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({ createdBy: { id: userId } }),
        },
    },
});