import { MaxObjects, Quiz, QuizCreateInput, QuizSearchInput, QuizSortBy, QuizUpdateInput, quizValidation, QuizYou } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestTranslation, defaultPermissions, getEmbeddableString, onCommonPlain, oneIsPublic, translationShapeHelper } from "../utils";
import { preShapeEmbeddableTranslatable } from "../utils/preShapeEmbeddableTranslatable";
import { getSingleTypePermissions } from "../validators";
import { BookmarkModel } from "./bookmark";
import { ProjectModel } from "./project";
import { ReactionModel } from "./reaction";
import { RoutineModel } from "./routine";
import { ModelLogic } from "./types";

const __typename = "Quiz" as const;
type Permissions = Pick<QuizYou, "canDelete" | "canUpdate" | "canBookmark" | "canRead" | "canReact">;
const suppFields = ["you"] as const;
export const QuizModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: QuizCreateInput,
    GqlUpdate: QuizUpdateInput,
    GqlModel: Quiz,
    GqlSearch: QuizSearchInput,
    GqlSort: QuizSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.quizUpsertArgs["create"],
    PrismaUpdate: Prisma.quizUpsertArgs["update"],
    PrismaModel: Prisma.quizGetPayload<SelectWrap<Prisma.quizSelect>>,
    PrismaSelect: Prisma.quizSelect,
    PrismaWhere: Prisma.quizWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.quiz,
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
    format: {
        gqlRelMap: {
            __typename,
            attempts: "QuizAttempt",
            createdBy: "User",
            project: "Project",
            quizQuestions: "QuizQuestion",
            routine: "Routine",
            bookmarkedBy: "User",
        },
        prismaRelMap: {
            __typename,
            attempts: "QuizAttempt",
            createdBy: "User",
            project: "Project",
            quizQuestions: "QuizQuestion",
            routine: "Routine",
            bookmarkedBy: "User",
        },
        joinMap: { bookmarkedBy: "user" },
        countFields: {
            attemptsCount: true,
            quizQuestionsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        hasCompleted: new Array(ids.length).fill(false), // TODO: Implement
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
                isPrivate: noNull(data.isPrivate),
                maxAttempts: noNull(data.maxAttempts),
                randomizeQuestionOrder: noNull(data.randomizeQuestionOrder),
                revealCorrectAnswers: noNull(data.revealCorrectAnswers),
                timeLimit: noNull(data.timeLimit),
                pointsToPass: noNull(data.pointsToPass),
                createdBy: { connect: { id: rest.userData.id } },
                ...(await shapeHelper({ relation: "project", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "Project", parentRelationshipName: "quizzes", data, ...rest })),
                ...(await shapeHelper({ relation: "routine", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "Routine", parentRelationshipName: "quizzes", data, ...rest })),
                ...(await shapeHelper({ relation: "quizQuestions", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "QuizQuestion", parentRelationshipName: "answers", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                isPrivate: noNull(data.isPrivate),
                maxAttempts: noNull(data.maxAttempts),
                randomizeQuestionOrder: noNull(data.randomizeQuestionOrder),
                revealCorrectAnswers: noNull(data.revealCorrectAnswers),
                timeLimit: noNull(data.timeLimit),
                pointsToPass: noNull(data.pointsToPass),
                createdBy: { connect: { id: rest.userData.id } },
                ...(await shapeHelper({ relation: "project", relTypes: ["Connect", "Disconnect"], isOneToOne: true, isRequired: false, objectType: "Project", parentRelationshipName: "quizzes", data, ...rest })),
                ...(await shapeHelper({ relation: "routine", relTypes: ["Connect", "Disconnect"], isOneToOne: true, isRequired: false, objectType: "Routine", parentRelationshipName: "quizzes", data, ...rest })),
                ...(await shapeHelper({ relation: "quizQuestions", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "QuizQuestion", parentRelationshipName: "answers", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
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
        yup: quizValidation,
    },
    search: {
        defaultSort: QuizSortBy.ScoreDesc,
        sortBy: QuizSortBy,
        searchFields: {
            createdTimeFrame: true,
            isComplete: true,
            translationLanguages: true,
            maxBookmarks: true,
            maxScore: true,
            minBookmarks: true,
            minScore: true,
            routineId: true,
            projectId: true,
            userId: true,
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
        isPublic: (data, languages) => data.isPrivate === false && oneIsPublic<Prisma.quizSelect>(data, [
            ["project", "Project"],
            ["routine", "Routine"],
        ], languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data.createdBy,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            isPrivate: true,
            createdBy: "User",
            project: "Project",
            routine: "Routine",
        }),
        visibility: {
            private: {
                OR: [
                    { isPrivate: true },
                    { project: ProjectModel.validate!.visibility.private },
                    { routine: RoutineModel.validate!.visibility.private },
                ],
            },
            public: {
                AND: [
                    { isPrivate: false },
                    { project: ProjectModel.validate!.visibility.public },
                    { routine: RoutineModel.validate!.visibility.public },
                ],
            },
            owner: (userId) => ({ createdBy: { id: userId } }),
        },
    },
});
