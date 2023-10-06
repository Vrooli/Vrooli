import { QuizAttemptSortBy, quizAttemptValidation } from "@local/shared";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { QuizAttemptFormat } from "../formats";
import { ModelLogic } from "../types";
import { QuizModel } from "./quiz";
import { QuizAttemptModelLogic, QuizModelLogic } from "./types";

const __typename = "QuizAttempt" as const;
const suppFields = ["you"] as const;
export const QuizAttemptModel: ModelLogic<QuizAttemptModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.quiz_attempt,
    display: {
        label: {
            select: () => ({
                id: true,
                created_at: true,
                quiz: { select: QuizModel.display.label.select() },
            }),
            // Label is quiz name + created_at date
            get: (select, languages) => {
                const quizName = QuizModel.display.label.get(select.quiz as QuizModelLogic["PrismaModel"], languages);
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
        isPublic: (...rest) => oneIsPublic<QuizAttemptModelLogic["PrismaSelect"]>([["quiz", "Quiz"]], ...rest),
        isTransferable: false,
        maxObjects: 100000,
        owner: (data, userId) => QuizModel.validate.owner(data?.quiz as QuizModelLogic["PrismaModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({ id: true, quiz: "Quiz" }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                quiz: QuizModel.validate.visibility.owner(userId),
            }),
        },
    },
});
