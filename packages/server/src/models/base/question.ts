import { MaxObjects, ModelType, QuestionForType, QuestionSortBy, getTranslation, questionValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull } from "../../builders/noNull.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { getEmbeddableString } from "../../utils/embeddings/getEmbeddableString.js";
import { preShapeEmbeddableTranslatable, type PreShapeEmbeddableTranslatableResult } from "../../utils/shapes/preShapeEmbeddableTranslatable.js";
import { tagShapeHelper } from "../../utils/shapes/tagShapeHelper.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { afterMutationsPlain } from "../../utils/triggers/afterMutationsPlain.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { QuestionFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { BookmarkModelLogic, QuestionModelInfo, QuestionModelLogic, ReactionModelLogic } from "./types.js";

type QuestionPre = PreShapeEmbeddableTranslatableResult;

const forMapper: { [key in QuestionForType]: keyof Prisma.questionUpsertArgs["create"] } = {
    Api: "api",
    Code: "code",
    Note: "note",
    Project: "project",
    Routine: "routine",
    Standard: "standard",
    Team: "team",
};

const __typename = "Question" as const;
export const QuestionModel: QuestionModelLogic = ({
    __typename,
    dbTable: "question",
    dbTranslationTable: "question_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => getTranslation(select, languages).name ?? "",
        },
        embed: {
            select: () => ({ id: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } } }),
            get: ({ translations }, languages) => {
                const trans = getTranslation({ translations }, languages);
                return getEmbeddableString({
                    description: trans.description,
                    name: trans.name,
                }, languages?.[0]);
            },
        },
    }),
    format: QuestionFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Update }): Promise<QuestionPre> => {
                const maps = preShapeEmbeddableTranslatable<"id">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as QuestionPre;
                return {
                    id: data.id,
                    isPrivate: data.isPrivate,
                    referencing: noNull(data.referencing),
                    createdBy: { connect: { id: rest.userData.id } },
                    ...((data.forObjectConnect && data.forObjectType) ? ({ [forMapper[data.forObjectType]]: { connect: { id: data.forObjectConnect } } }) : {}),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Question", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as QuestionPre;
                return {
                    isPrivate: noNull(data.isPrivate),
                    ...(data.acceptedAnswerConnect ? {
                        answers: {
                            update: {
                                where: { id: data.acceptedAnswerConnect },
                                data: { isAccepted: true },
                            },
                        },
                    } : {}),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Question", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
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
            codeId: true,
            noteId: true,
            projectId: true,
            routineId: true,
            standardId: true,
            teamId: true,
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
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<QuestionModelInfo["ApiPermission"]>(__typename, ids, userData)),
                        isBookmarked: await ModelMap.get<BookmarkModelLogic>("Bookmark").query.getIsBookmarkeds(userData?.id, ids, __typename),
                        reaction: await ModelMap.get<ReactionModelLogic>("Reaction").query.getReactions(userData?.id, ids, __typename),
                    },
                };
            },
        },
    },
    validate: () => ({
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
            own: function getOwn(data) {
                return {
                    createdBy: { id: data.userId },
                    OR: [
                        // The question is not tied to an object
                        Object.fromEntries(Object.entries(forMapper).map(([key, value]) => [value + "Id", null])),
                        // The object the question is tied to is public
                        Object.fromEntries(Object.entries(forMapper).map(([key, value]) => [value, useVisibility(key as ModelType, "Public", data)])),
                    ],
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    OR: [
                        useVisibility("Question", "Own", data),
                        useVisibility("Question", "Public", data),
                    ],
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    ...useVisibility("Question", "Own", data),
                    isPrivate: true,
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    ...useVisibility("Question", "Own", data),
                    isPrivate: true,
                };
            },
            public: function getPublic(data) {
                return {
                    isPrivate: false,
                    OR: [
                        // The question is not tied to an object
                        Object.fromEntries(Object.entries(forMapper).map(([key, value]) => [value + "Id", null])),
                        // The object the question is tied to is public
                        Object.fromEntries(Object.entries(forMapper).map(([key, value]) => [value, useVisibility(key as ModelType, "Public", data)])),
                    ],
                };
            },
        },
    }),
});
