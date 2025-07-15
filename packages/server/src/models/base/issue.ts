import { type Prisma } from "@prisma/client";
import { generatePublicId, getTranslation, type IssueFor, type IssueSearchInput, IssueSortBy, IssueStatus, issueValidation, MaxObjects, type ModelType } from "@vrooli/shared";
import { useVisibility, useVisibilityMapper } from "../../builders/visibilityBuilder.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { preShapeEmbeddableTranslatable, type PreShapeEmbeddableTranslatableResult } from "../../utils/shapes/preShapeEmbeddableTranslatable.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { defaultPermissions, filterPermissions, getSingleTypePermissions } from "../../validators/permissions.js";
import { IssueFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { type BookmarkModelLogic, type IssueModelInfo, type IssueModelLogic, type ReactionModelLogic } from "./types.js";

type IssuePre = PreShapeEmbeddableTranslatableResult;

const forMapper: { [key in IssueFor]: keyof Prisma.issueUpsertArgs["create"] } = {
    Resource: "resource",
    Team: "team",
};
const reversedForMapper: { [key in keyof Prisma.issueUpsertArgs["create"]]: IssueFor } = Object.fromEntries(
    Object.entries(forMapper).map(([key, value]) => [value, key]),
) as { [key in keyof Prisma.issueUpsertArgs["create"]]: IssueFor };

const __typename = "Issue" as const;
export const IssueModel: IssueModelLogic = ({
    __typename,
    dbTable: "issue",
    dbTranslationTable: "issue_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => getTranslation(select, languages).name ?? "",
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
                    id: BigInt(data.id),
                    publicId: generatePublicId(),
                    referencedVersion: data.referencedVersionIdConnect ? { connect: { id: BigInt(data.referencedVersionIdConnect) } } : undefined,
                    [forMapper[data.issueFor]]: { connect: { id: BigInt(data.forConnect) } },
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
            // NOTE: Pull request creator can only set status to 'Canceled'. 
            // Owner of object that pull request is on can set status to anything but 'Canceled'.
            // Once out of 'Draft' status, status cannot be set back to 'Draft'.
            // TODO need to update params for shape to account for this (probably). Then need to update this function
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as IssuePre;
                return {
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
        },
        yup: issueValidation,
    },
    search: {
        defaultSort: IssueSortBy.ScoreDesc,
        sortBy: IssueSortBy,
        searchFields: {
            closedById: true,
            createdById: true,
            createdTimeFrame: true,
            minScore: true,
            minBookmarks: true,
            minViews: true,
            referencedVersionId: true,
            resourceId: true,
            status: true,
            teamId: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({ OR: ["transDescriptionWrapped", "transNameWrapped"] }),
        supplemental: {
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<IssueModelInfo["ApiPermission"]>(__typename, ids, userData)),
                        isBookmarked: await ModelMap.get<BookmarkModelLogic>("Bookmark").query.getIsBookmarkeds(userData?.id, ids, __typename),
                        reaction: await ModelMap.get<ReactionModelLogic>("Reaction").query.getReactions(userData?.id, ids, __typename),
                    },
                };
            },
        },
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: (data, getParentInfo?) => oneIsPublic<IssueModelInfo["DbSelect"]>([
            ["resource", "Resource"],
            ["team", "Team"],
        ], data, getParentInfo),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, _userId) => ({
            User: data?.createdBy,
        }),
        permissionResolvers: ({ isAdmin, isDeleted, isLoggedIn, isPublic }) => {
            const base = defaultPermissions({ isAdmin, isDeleted, isLoggedIn, isPublic });
            return filterPermissions(base, [
                "canBookmark", "canComment", "canDelete", "canReact", "canRead", "canReport", "canUpdate",
                "canConnect", "canDisconnect",
            ]);
        },
        permissionsSelect: () => ({
            id: true,
            createdBy: "User",
            resource: "Resource",
            team: "Team",
        }),
        visibility: {
            own: function getOwn(data) {
                return {
                    createdBy: { id: BigInt(data.userId) },
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    OR: [
                        useVisibility("Issue", "Own", data),
                        useVisibility("Issue", "Public", data),
                    ],
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    ...useVisibility("Issue", "Own", data),
                    status: IssueStatus.Draft,
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    ...useVisibility("Issue", "Own", data),
                    status: { not: IssueStatus.Draft },
                };
            },
            public: function getPublic(data) {
                const common = {
                    status: { not: IssueStatus.Draft },
                } as const;
                const searchInput = data.searchInput as IssueSearchInput;
                // If the search input has a relation ID, return that relation only
                const forSearch = Object.keys(searchInput).find(searchKey =>
                    searchKey.endsWith("Id") &&
                    reversedForMapper[searchKey.substring(0, searchKey.length - "Id".length)],
                );
                if (forSearch) {
                    const relation = forSearch.substring(0, forSearch.length - "Id".length);
                    return {
                        ...common,
                        [relation]: useVisibility(reversedForMapper[relation] as ModelType, "Public", data),
                    };
                }
                // Otherwise, use an OR on all relations
                return {
                    ...common,
                    // Can use OR because only one relation will be present
                    OR: [
                        ...useVisibilityMapper("Public", data, forMapper, false),
                    ],
                };
            },
        },
    }),
});
