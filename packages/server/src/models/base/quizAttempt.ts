import { QuizAttemptSortBy, quizAttemptValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { QuizAttemptFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { QuizAttemptModelInfo, QuizAttemptModelLogic, QuizModelInfo, QuizModelLogic } from "./types";

const __typename = "QuizAttempt" as const;
export const QuizAttemptModel: QuizAttemptModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.quiz_attempt,
    display: {
        label: {
            select: () => ({
                id: true,
                created_at: true,
                quiz: { select: ModelMap.get<QuizModelLogic>("Quiz").display.label.select() },
            }),
            // Label is quiz name + created_at date
            get: (select, languages) => {
                const quizName = ModelMap.get<QuizModelLogic>("Quiz").display.label.get(select.quiz as QuizModelInfo["PrismaModel"], languages);
                const date = new Date(select.created_at).toLocaleDateString();
                return `${quizName} - ${date}`;
            },
        },
    },
    format: QuizAttemptFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                contextSwitches: noNull(data.contextSwitches),
                timeTaken: noNull(data.timeTaken),
                language: data.language,
                user: { connect: { id: rest.userData.id } },
                ...(await shapeHelper({ relation: "quiz", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Quiz", parentRelationshipName: "attempts", data, ...rest })),
                ...(await shapeHelper({ relation: "responses", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "QuizQuestionResponse", parentRelationshipName: "quizAttempt", data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                contextSwitches: noNull(data.contextSwitches),
                timeTaken: noNull(data.timeTaken),
                ...(await shapeHelper({ relation: "responses", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "QuizQuestionResponse", parentRelationshipName: "quizAttempt", data, ...rest })),
            }),
        },
        yup: quizAttemptValidation,
    },
    search: {
        defaultSort: QuizAttemptSortBy.TimeTakenDesc,
        sortBy: QuizAttemptSortBy,
        searchFields: {
            createdTimeFrame: true,
            status: true,
            languageIn: true,
            maxPointsEarned: true,
            minPointsEarned: true,
            userId: true,
            quizId: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({}), // No strings to search
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
        isPublic: (...rest) => oneIsPublic<QuizAttemptModelInfo["PrismaSelect"]>([["quiz", "Quiz"]], ...rest),
        isTransferable: false,
        maxObjects: 100000,
        owner: (data, userId) => ModelMap.get<QuizModelLogic>("Quiz").validate.owner(data?.quiz as QuizModelInfo["PrismaModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({ id: true, quiz: "Quiz" }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                quiz: ModelMap.get<QuizModelLogic>("Quiz").validate.visibility.owner(userId),
            }),
        },
    },
});
