import { IssueFor, IssueSortBy, issueValidation, MaxObjects } from "@local/shared";
import { Prisma } from "@prisma/client";
import { ModelMap } from ".";
import { bestTranslation, defaultPermissions, getEmbeddableString, oneIsPublic } from "../../utils";
import { labelShapeHelper, preShapeEmbeddableTranslatable, PreShapeEmbeddableTranslatableResult, translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions } from "../../validators";
import { IssueFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { BookmarkModelLogic, IssueModelInfo, IssueModelLogic, ReactionModelLogic } from "./types";

type IssuePre = PreShapeEmbeddableTranslatableResult;

const forMapper: { [key in IssueFor]: keyof Prisma.issueUpsertArgs["create"] } = {
    Api: "api",
    Code: "code",
    Note: "note",
    Project: "project",
    Routine: "routine",
    Standard: "standard",
    Team: "team",
};

const __typename = "Issue" as const;
export const IssueModel: IssueModelLogic = ({
    __typename,
    dbTable: "issue",
    dbTranslationTable: "issue_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
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
    }),
    format: IssueFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Update }): Promise<IssuePre> => {
                const maps = preShapeEmbeddableTranslatable<"id">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as IssuePre;
                return {
                    id: data.id,
                    referencedVersion: data.referencedVersionIdConnect ? { connect: { id: data.referencedVersionIdConnect } } : undefined,
                    [forMapper[data.issueFor]]: { connect: { id: data.forConnect } },
                    labels: await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Issue", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                }
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as IssuePre;
                return {
                    labels: await labelShapeHelper({ relTypes: ["Connect", "Disconnect", "Create"], parentType: "Issue", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                }
            },
        },
        yup: issueValidation,
    },
    search: {
        defaultSort: IssueSortBy.ScoreDesc,
        sortBy: IssueSortBy,
        searchFields: {
            apiId: true,
            closedById: true,
            codeId: true,
            createdById: true,
            createdTimeFrame: true,
            minScore: true,
            minBookmarks: true,
            minViews: true,
            noteId: true,
            projectId: true,
            referencedVersionId: true,
            routineId: true,
            standardId: true,
            status: true,
            teamId: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({ OR: ["transDescriptionWrapped", "transNameWrapped"] }),
        supplemental: {
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, userData)),
                        isBookmarked: await ModelMap.get<BookmarkModelLogic>("Bookmark").query.getIsBookmarkeds(userData?.id, ids, __typename),
                        reaction: await ModelMap.get<ReactionModelLogic>("Reaction").query.getReactions(userData?.id, ids, __typename),
                    },
                };
            },
        },
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<IssueModelInfo["PrismaSelect"]>([
            ["api", "Api"],
            ["code", "Code"],
            ["note", "Note"],
            ["project", "Project"],
            ["routine", "Routine"],
            ["standard", "Standard"],
            ["team", "Team"],
        ], ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data?.createdBy,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            api: "Api",
            code: "Code",
            createdBy: "User",
            note: "Note",
            project: "Project",
            routine: "Routine",
            standard: "Standard",
            team: "Team",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({ createdBy: { id: userId } }),
        },
    }),
});
