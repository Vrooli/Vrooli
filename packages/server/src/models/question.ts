import { MaxObjects, Question, QuestionCreateInput, QuestionForType, QuestionSearchInput, QuestionSortBy, QuestionUpdateInput, questionValidation, QuestionYou } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestTranslation, defaultPermissions, onCommonPlain, tagShapeHelper, translationShapeHelper } from "../utils";
import { preShapeEmbeddableTranslatable } from "../utils/preShapeEmbeddableTranslatable";
import { getSingleTypePermissions } from "../validators";
import { BookmarkModel } from "./bookmark";
import { ReactionModel } from "./reaction";
import { ModelLogic } from "./types";

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
type Permissions = Pick<QuestionYou, "canDelete" | "canUpdate" | "canBookmark" | "canRead" | "canReact">;
const suppFields = ["you"] as const;
export const QuestionModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: QuestionCreateInput,
    GqlUpdate: QuestionUpdateInput,
    GqlModel: Question,
    GqlSearch: QuestionSearchInput,
    GqlSort: QuestionSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.questionUpsertArgs["create"],
    PrismaUpdate: Prisma.questionUpsertArgs["update"],
    PrismaModel: Prisma.questionGetPayload<SelectWrap<Prisma.questionSelect>>,
    PrismaSelect: Prisma.questionSelect,
    PrismaWhere: Prisma.questionWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.question,
    display: {
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
        embed: {
            select: () => ({ id: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } } }),
            get: ({ translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return JSON.stringify({
                    description: trans.description,
                    name: trans.name,
                });
            },
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            createdBy: "User",
            answers: "QuestionAnswer",
            comments: "Comment",
            forObject: {
                api: "Api",
                note: "Note",
                organization: "Organization",
                project: "Project",
                routine: "Routine",
                smartContract: "SmartContract",
                standard: "Standard",
            },
            reports: "Report",
            bookmarkedBy: "User",
            tags: "Tag",
        },
        prismaRelMap: {
            __typename,
            createdBy: "User",
            api: "Api",
            note: "Note",
            organization: "Organization",
            project: "Project",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
            comments: "Comment",
            answers: "QuestionAnswer",
            reports: "Report",
            tags: "Tag",
            bookmarkedBy: "User",
            reactions: "Reaction",
            viewedBy: "User",
        },
        joinMap: { bookmarkedBy: "user", tags: "tag" },
        countFields: {
            answersCount: true,
            commentsCount: true,
            reportsCount: true,
            translationsCount: true,
        },
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
    mutate: {
        shape: {
            pre: async ({ createList, updateList }) => {
                const maps = preShapeEmbeddableTranslatable({ createList, updateList, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                referencing: noNull(data.referencing),
                createdBy: { connect: { id: rest.userData.id } },
                ...((data.forObjectConnect && data.forObjectType) ? ({ [forMapper[data.forObjectType]]: { connect: { id: data.forObjectConnect } } }) : {}),
                ...(await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Question", relation: "tags", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
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
            onCommon: async (params) => {
                await onCommonPlain({
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
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => true,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data.createdBy,
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
