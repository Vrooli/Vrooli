import { MaxObjects, QuizSortBy } from "@local/consts";
import { quizValidation } from "@local/validation";
import { noNull, shapeHelper } from "../builders";
import { bestLabel, defaultPermissions, onCommonPlain, oneIsPublic, translationShapeHelper } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { BookmarkModel } from "./bookmark";
import { ProjectModel } from "./project";
import { ReactionModel } from "./reaction";
import { RoutineModel } from "./routine";
const __typename = "Quiz";
const suppFields = ["you"];
export const QuizModel = ({
    __typename,
    delegate: (prisma) => prisma.quiz,
    display: {
        select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, "name", languages),
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
                        ...(await getSingleTypePermissions(__typename, ids, prisma, userData)),
                        hasCompleted: new Array(ids.length).fill(false),
                        isBookmarked: await BookmarkModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        reaction: await ReactionModel.query.getReactions(prisma, userData?.id, ids, __typename),
                    },
                };
            },
        },
    },
    mutate: {
        shape: {
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
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
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
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
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
            visibility: true,
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
        isPublic: (data, languages) => data.isPrivate === false && oneIsPublic(data, [
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
//# sourceMappingURL=quiz.js.map