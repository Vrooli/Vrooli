import { MaxObjects, QuizSortBy, quizValidation } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { bestTranslation, defaultPermissions, getEmbeddableString, oneIsPublic } from "../../utils";
import { preShapeEmbeddableTranslatable, translationShapeHelper } from "../../utils/shapes";
import { afterMutationsPlain } from "../../utils/triggers";
import { getSingleTypePermissions } from "../../validators";
import { QuizFormat } from "../formats";
import { ModelLogic } from "../types";
import { BookmarkModel } from "./bookmark";
import { ProjectModel } from "./project";
import { ReactionModel } from "./reaction";
import { RoutineModel } from "./routine";
import { QuizModelLogic } from "./types";

const __typename = "Quiz" as const;
const suppFields = ["you"] as const;
export const QuizModel: ModelLogic<QuizModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.quiz,
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
                    description: trans?.description,
                    name: trans?.name,
                }, languages[0]);
            },
        },
    },
    format: QuizFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Update }) => {
                const maps = preShapeEmbeddableTranslatable<"id">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                isPrivate: data.isPrivate,
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
            afterMutations: async (params) => {
                await afterMutationsPlain({
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
    validate: {
        isDeleted: () => false,
        isPublic: (data, ...rest) => data.isPrivate === false && oneIsPublic<QuizModelLogic["PrismaSelect"]>([
            ["project", "Project"],
            ["routine", "Routine"],
        ], data, ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data?.createdBy,
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
                    { project: ProjectModel.validate.visibility.private },
                    { routine: RoutineModel.validate.visibility.private },
                ],
            },
            public: {
                AND: [
                    { isPrivate: false },
                    { project: ProjectModel.validate.visibility.public },
                    { routine: RoutineModel.validate.visibility.public },
                ],
            },
            owner: (userId) => ({ createdBy: { id: userId } }),
        },
    },
});
