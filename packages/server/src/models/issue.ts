import { Issue, IssueCreateInput, IssueFor, IssueSearchInput, IssueSortBy, IssueUpdateInput, issueValidation, IssueYou, MaxObjects } from "@local/shared";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestTranslation, defaultPermissions, getEmbeddableString, labelShapeHelper, oneIsPublic, translationShapeHelper } from "../utils";
import { preShapeEmbeddableTranslatable } from "../utils/preShapeEmbeddableTranslatable";
import { getSingleTypePermissions } from "../validators";
import { BookmarkModel } from "./bookmark";
import { ReactionModel } from "./reaction";
import { ModelLogic } from "./types";

const forMapper: { [key in IssueFor]: keyof Prisma.issueUpsertArgs["create"] } = {
    Api: "api",
    Note: "note",
    Organization: "organization",
    Project: "project",
    Routine: "routine",
    SmartContract: "smartContract",
    Standard: "standard",
};

const __typename = "Issue" as const;
type Permissions = Pick<IssueYou, "canComment" | "canDelete" | "canUpdate" | "canBookmark" | "canReport" | "canRead" | "canReact">;
const suppFields = ["you"] as const;
export const IssueModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: IssueCreateInput,
    GqlUpdate: IssueUpdateInput,
    GqlModel: Issue,
    GqlSearch: IssueSearchInput,
    GqlSort: IssueSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.issueUpsertArgs["create"],
    PrismaUpdate: Prisma.issueUpsertArgs["update"],
    PrismaModel: Prisma.issueGetPayload<SelectWrap<Prisma.issueSelect>>,
    PrismaSelect: Prisma.issueSelect,
    PrismaWhere: Prisma.issueWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.issue,
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
                    description: trans.description,
                    name: trans.name,
                }, languages[0]);
            },
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            closedBy: "User",
            comments: "Comment",
            createdBy: "User",
            labels: "Label",
            reports: "Report",
            bookmarkedBy: "User",
            to: {
                api: "Api",
                organization: "Organization",
                note: "Note",
                project: "Project",
                routine: "Routine",
                smartContract: "SmartContract",
                standard: "Standard",
            },
        },
        prismaRelMap: {
            __typename,
            api: "Api",
            organization: "Organization",
            note: "Note",
            project: "Project",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
            closedBy: "User",
            comments: "Comment",
            labels: "Label",
            reports: "Report",
            reactions: "Reaction",
            bookmarkedBy: "User",
            viewedBy: "View",
        },
        joinMap: { labels: "label", bookmarkedBy: "user" },
        countFields: {
            commentsCount: true,
            labelsCount: true,
            reportsCount: true,
            translationsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        isBookmarked: await BookmarkModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        reaction: await ReactionModel.query.getReactions(prisma, userData?.id, ids, __typename),
                    },
                };
            },
        },
    },
    mutate: {
        shape: {
            pre: async ({ createList, updateList }) => {
                const maps = preShapeEmbeddableTranslatable({ createList, updateList, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                referencedVersion: data.referencedVersionIdConnect ? { connect: { id: data.referencedVersionIdConnect } } : undefined,
                [forMapper[data.issueFor]]: { connect: { id: data.forConnect } },
                ...(await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Issue", relation: "labels", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                ...(await labelShapeHelper({ relTypes: ["Connect", "Disconnect", "Create"], parentType: "Issue", relation: "labels", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
        },
        yup: issueValidation,
    },
    search: {
        defaultSort: IssueSortBy.ScoreDesc,
        sortBy: IssueSortBy,
        searchFields: {
            apiId: true,
            closedById: true,
            createdById: true,
            createdTimeFrame: true,
            minScore: true,
            minBookmarks: true,
            minViews: true,
            noteId: true,
            organizationId: true,
            projectId: true,
            referencedVersionId: true,
            routineId: true,
            smartContractId: true,
            standardId: true,
            status: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({ OR: ["transDescriptionWrapped", "transNameWrapped"] }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.issueSelect>(data, [
            ["api", "Api"],
            ["organization", "Organization"],
            ["note", "Note"],
            ["project", "Project"],
            ["routine", "Routine"],
            ["smartContract", "SmartContract"],
            ["standard", "Standard"],
        ], languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data.createdBy,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            api: "Api",
            createdBy: "User",
            organization: "Organization",
            note: "Note",
            project: "Project",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({ createdBy: { id: userId } }),
        },
    },
});
