import { MaxObjects, QuizAttemptSortBy, quizAttemptValidation } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { QuizAttemptFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { QuizAttemptModelInfo, QuizAttemptModelLogic, QuizModelInfo, QuizModelLogic } from "./types.js";

const __typename = "QuizAttempt" as const;
export const QuizAttemptModel: QuizAttemptModelLogic = ({
    __typename,
    dbTable: "quiz_attempt",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                created_at: true,
                quiz: { select: ModelMap.get<QuizModelLogic>("Quiz").display().label.select() },
            }),
            // Label is quiz name + created_at date
            get: (select, languages) => {
                const quizName = ModelMap.get<QuizModelLogic>("Quiz").display().label.get(select.quiz as QuizModelInfo["DbModel"], languages);
                const date = new Date(select.created_at).toLocaleDateString();
                return `${quizName} - ${date}`;
            },
        },
    }),
    format: QuizAttemptFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                contextSwitches: noNull(data.contextSwitches),
                timeTaken: noNull(data.timeTaken),
                language: data.language,
                user: { connect: { id: rest.userData.id } },
                quiz: await shapeHelper({ relation: "quiz", relTypes: ["Connect"], isOneToOne: true, objectType: "Quiz", parentRelationshipName: "attempts", data, ...rest }),
                responses: await shapeHelper({ relation: "responses", relTypes: ["Create"], isOneToOne: false, objectType: "QuizQuestionResponse", parentRelationshipName: "quizAttempt", data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                contextSwitches: noNull(data.contextSwitches),
                timeTaken: noNull(data.timeTaken),
                responses: await shapeHelper({ relation: "responses", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "QuizQuestionResponse", parentRelationshipName: "quizAttempt", data, ...rest }),
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
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<QuizAttemptModelInfo["ApiPermission"]>(__typename, ids, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<QuizAttemptModelInfo["DbSelect"]>([["quiz", "Quiz"]], ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<QuizModelLogic>("Quiz").validate().owner(data?.quiz as QuizModelInfo["DbModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({ id: true, quiz: "Quiz" }),
        visibility: {
            own: function getOwn(data) {
                return {
                    user: { id: data.userId },
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    OR: [
                        { user: { id: data.userId } },
                        { quiz: useVisibility("Quiz", "Public", data) },
                    ],
                };
            },
            // Search method not useful for this object because answers are not explicitly set as private, so we'll return "Own"
            // TODO when drafts (i.e. in-progress attempts) are added, this and related functions will have to change
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("QuizAttempt", "Own", data);
            },
            ownPublic: function getOwnPublic(data) {
                return useVisibility("QuizAttempt", "Own", data);
            },
            public: function getPublic(data) {
                return {
                    quiz: useVisibility("QuizAttempt", "Public", data),
                };
            },
        },
    }),
});
