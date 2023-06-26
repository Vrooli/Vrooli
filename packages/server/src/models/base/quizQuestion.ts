import { MaxObjects, QuizQuestionSortBy, quizQuestionValidation } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { bestTranslation, defaultPermissions, translationShapeHelper } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { QuizQuestionFormat } from "../format/quizQuestion";
import { ModelLogic } from "../types";
import { QuizModel } from "./quiz";
import { QuizQuestionModelLogic } from "./types";

const __typename = "QuizQuestion" as const;
const suppFields = ["you"] as const;
export const QuizQuestionModel: ModelLogic<QuizQuestionModelLogic, typeof suppFields> = ({
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
        isDeleted: () => false,
        isPublic: (data, languages) => QuizModel.validate.isPublic(data.quiz as any, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => QuizModel.validate.owner(data.quiz as any, userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            quiz: "Quiz",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                quiz: QuizModel.validate.visibility.owner(userId),
            }),
        },
    },
});
