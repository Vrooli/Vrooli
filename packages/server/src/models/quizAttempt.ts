import { QuizAttempt, QuizAttemptCreateInput, QuizAttemptSearchInput, QuizAttemptSortBy, QuizAttemptUpdateInput, quizAttemptValidation, QuizAttemptYou } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { getSingleTypePermissions } from "../validators";
import { QuizModel } from "./quiz";
import { ModelLogic } from "./types";

const __typename = "QuizAttempt" as const;
type Permissions = Pick<QuizAttemptYou, "canDelete" | "canUpdate">;
const suppFields = ["you"] as const;
export const QuizAttemptModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: QuizAttemptCreateInput,
    GqlUpdate: QuizAttemptUpdateInput,
    GqlModel: QuizAttempt,
    GqlSearch: QuizAttemptSearchInput,
    GqlSort: QuizAttemptSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.quiz_attemptUpsertArgs["create"],
    PrismaUpdate: Prisma.quiz_attemptUpsertArgs["update"],
    PrismaModel: Prisma.quiz_attemptGetPayload<SelectWrap<Prisma.quiz_attemptSelect>>,
    PrismaSelect: Prisma.quiz_attemptSelect,
    PrismaWhere: Prisma.quiz_attemptWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.quiz_attempt,
    display: {
        select: () => ({
            id: true,
            created_at: true,
            quiz: { select: QuizModel.display.select() },
        }),
        // Label is quiz name + created_at date
        label: (select, languages) => {
            const quizName = QuizModel.display.label(select.quiz as any, languages);
            const date = new Date(select.created_at).toLocaleDateString();
            return `${quizName} - ${date}`;
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            quiz: "Quiz",
            responses: "QuizQuestionResponse",
            user: "User",
        },
        prismaRelMap: {
            __typename,
            quiz: "Quiz",
            responses: "QuizQuestionResponse",
            user: "User",
        },
        countFields: {
            responsesCount: true,
        },
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
            visibility: true,
        },
        searchStringQuery: () => ({}), // No strings to search
    },
    validate: {} as any,
});
