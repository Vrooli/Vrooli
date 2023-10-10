import { MaxObjects, QuizQuestionSortBy, quizQuestionValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { bestTranslation, defaultPermissions, oneIsPublic } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions } from "../../validators";
import { QuizQuestionFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { QuizModelInfo, QuizModelLogic, QuizQuestionModelInfo, QuizQuestionModelLogic } from "./types";

const __typename = "QuizQuestion" as const;
export const QuizQuestionModel: QuizQuestionModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.quiz_question,
    display: {
        label: {
            select: () => ({ id: true, translations: { select: { language: true, questionText: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.questionText ?? "",
        },
    },
    format: QuizQuestionFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                order: noNull(data.order),
                points: noNull(data.points),
                ...(await shapeHelper({ relation: "standardVersion", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: true, objectType: "StandardVersion", parentRelationshipName: "quizQuestions", data, ...rest })),
                ...(await shapeHelper({ relation: "quiz", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Quiz", parentRelationshipName: "quizQuestions", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                order: noNull(data.order),
                points: noNull(data.points),
                ...(await shapeHelper({ relation: "standardVersion", relTypes: ["Connect", "Create", "Update"], isOneToOne: true, isRequired: false, objectType: "StandardVersion", parentRelationshipName: "quizQuestions", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
        },
        yup: quizQuestionValidation,
    },
    search: {
        defaultSort: QuizQuestionSortBy.OrderAsc,
        sortBy: QuizQuestionSortBy,
        searchFields: {
            createdTimeFrame: true,
            translationLanguages: true,
            quizId: true,
            standardId: true,
            userId: true,
            responseId: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transQuestionTextWrapped",
            ],
        }),
        supplemental: {
            graphqlFields: SuppFields[__typename],
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
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<QuizQuestionModelInfo["PrismaSelect"]>([["quiz", "Quiz"]], ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<QuizModelLogic>("Quiz").validate.owner(data?.quiz as QuizModelInfo["PrismaModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            quiz: "Quiz",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                quiz: ModelMap.get<QuizModelLogic>("Quiz").validate.visibility.owner(userId),
            }),
        },
    },
});
