import { MaxObjects, QuestionForType, QuestionSortBy, questionValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull } from "../../builders/noNull";
import { bestTranslation, defaultPermissions, getEmbeddableString } from "../../utils";
import { preShapeEmbeddableTranslatable, tagShapeHelper, translationShapeHelper } from "../../utils/shapes";
import { afterMutationsPlain } from "../../utils/triggers";
import { getSingleTypePermissions } from "../../validators";
import { QuestionFormat } from "../formats";
import { ModelLogic } from "../types";
import { BookmarkModel } from "./bookmark";
import { ReactionModel } from "./reaction";
import { QuestionModelLogic } from "./types";

const forMapper: { [key in QuestionForType]: keyof Prisma.questionUpsertArgs["create"] } = {
    Api: "api",
    Note: "note",
    Organization: "organization",
    Project: "project",
    Routine: "routine",
    SmartContract: "smartContract",
    Standard: "standard",
};

const __typename = "Question" as const;
const suppFields = ["you"] as const;
export const QuestionModel: ModelLogic<QuestionModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.question,
    display: {
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
        embed: {
            select: () => ({ id: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } } }),
            get: ({ translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return getEmbeddableString({
                    description: trans?.description,
                    name: trans?.name,
                }, languages[0]);
            },
        },
    },
    format: QuestionFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Update }) => {
                const maps = preShapeEmbeddableTranslatable<"id">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                isPrivate: data.isPrivate,
                referencing: noNull(data.referencing),
                createdBy: { connect: { id: rest.userData.id } },
                ...((data.forObjectConnect && data.forObjectType) ? ({ [forMapper[data.forObjectType]]: { connect: { id: data.forObjectConnect } } }) : {}),
                ...(await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Question", relation: "tags", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                isPrivate: noNull(data.isPrivate),
                ...(data.acceptedAnswerConnect ? {
                    answers: {
                        update: {
                            where: { id: data.acceptedAnswerConnect },
                            data: { isAccepted: true },
                        },
                    },
                } : {}),
                ...(await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Question", relation: "tags", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsPlain({
                    ...params,
                    objectType: __typename,
                    ownerUserField: "createdBy",
                });
            },
        },
        yup: questionValidation,
    },
    search: {
        defaultSort: QuestionSortBy.ScoreDesc,
        sortBy: QuestionSortBy,
        searchFields: {
            createdTimeFrame: true,
            excludeIds: true,
            hasAcceptedAnswer: true,
            createdById: true,
            apiId: true,
            noteId: true,
            organizationId: true,
            projectId: true,
            routineId: true,
            smartContractId: true,
            standardId: true,
            translationLanguages: true,
            maxScore: true,
            maxBookmarks: true,
            minScore: true,
            minBookmarks: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transDescriptionWrapped",
                "transNameWrapped",
            ],
        }),
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
        isPublic: () => true,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data?.createdBy,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            createdBy: "User",
        }),
        visibility: {
            private: { isPrivate: true },
            public: { isPrivate: false },
            owner: (userId) => ({
                createdBy: { id: userId },
            }),
        },
    },
});
