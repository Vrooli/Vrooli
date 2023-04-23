import { MaxObjects, QuestionSortBy } from "@local/consts";
import { questionValidation } from "@local/validation";
import { noNull } from "../builders";
import { bestLabel, defaultPermissions, onCommonPlain, tagShapeHelper, translationShapeHelper } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { BookmarkModel } from "./bookmark";
import { ReactionModel } from "./reaction";
const forMapper = {
    Api: "api",
    Note: "note",
    Organization: "organization",
    Project: "project",
    Routine: "routine",
    SmartContract: "smartContract",
    Standard: "standard",
};
const __typename = "Question";
const suppFields = ["you"];
export const QuestionModel = ({
    __typename,
    delegate: (prisma) => prisma.question,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, "name", languages),
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
                        ...(await getSingleTypePermissions(__typename, ids, prisma, userData)),
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
                referencing: noNull(data.referencing),
                createdBy: { connect: { id: rest.userData.id } },
                ...((data.forObjectConnect && data.forObjectType) ? ({ [forMapper[data.forObjectType]]: { connect: { id: data.forObjectConnect } } }) : {}),
                ...(await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Question", relation: "tags", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
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
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
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
//# sourceMappingURL=question.js.map