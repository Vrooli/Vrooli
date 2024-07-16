import { MaxObjects, QuestionForType, QuestionSortBy, questionValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { bestTranslation, defaultPermissions, getEmbeddableString } from "../../utils";
import { PreShapeEmbeddableTranslatableResult, preShapeEmbeddableTranslatable, tagShapeHelper, translationShapeHelper } from "../../utils/shapes";
import { afterMutationsPlain } from "../../utils/triggers";
import { getSingleTypePermissions } from "../../validators";
import { QuestionFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { BookmarkModelLogic, QuestionModelLogic, ReactionModelLogic } from "./types";

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
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, userData)),
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
            private: function getVisibilityPrivate() {
                return {
                    isPrivate: true,
                };
            },
            public: function getVisibilityPublic() {
                return {
                    isPrivate: false,
                };
            },
            owner: (userId) => ({
                createdBy: { id: userId },
            }),
        },
    }),
});
