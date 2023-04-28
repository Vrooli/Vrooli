import { MaxObjects, Tag, TagCreateInput, TagSearchInput, TagSortBy, TagUpdateInput, tagValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions, translationShapeHelper } from "../utils";
import { BookmarkModel } from "./bookmark";
import { ModelLogic } from "./types";

const __typename = "Tag" as const;
const suppFields = ["you"] as const;
export const TagModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: TagCreateInput,
    GqlUpdate: TagUpdateInput,
    GqlModel: Tag,
    GqlSearch: TagSearchInput,
    GqlSort: TagSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.tagUpsertArgs["create"],
    PrismaUpdate: Prisma.tagUpsertArgs["update"],
    PrismaModel: Prisma.tagGetPayload<SelectWrap<Prisma.tagSelect>>,
    PrismaSelect: Prisma.tagSelect,
    PrismaWhere: Prisma.tagWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.tag,
    display: {
        select: () => ({ id: true, tag: true }),
        label: (select) => select.tag,
    },
    format: {
        gqlRelMap: {
            __typename,
            apis: "Api",
            notes: "Note",
            organizations: "Organization",
            posts: "Post",
            projects: "Project",
            reports: "Report",
            routines: "Routine",
            smartContracts: "SmartContract",
            standards: "Standard",
            bookmarkedBy: "User",
        },
        prismaRelMap: {
            __typename,
            createdBy: "User",
            apis: "Api",
            notes: "Note",
            organizations: "Organization",
            posts: "Post",
            projects: "Project",
            reports: "Report",
            routines: "Routine",
            smartContracts: "SmartContract",
            standards: "Standard",
            bookmarkedBy: "User",
            focusModeFilters: "FocusModeFilter",
        },
        joinMap: {
            apis: "tagged",
            notes: "tagged",
            organizations: "tagged",
            posts: "tagged",
            projects: "tagged",
            reports: "tagged",
            routines: "tagged",
            smartContracts: "tagged",
            standards: "tagged",
            bookmarkedBy: "user",
        },
        countFields: {},
        supplemental: {
            graphqlFields: suppFields,
            dbFields: ["createdById", "id"],
            toGraphQL: async ({ ids, objects, prisma, userData }) => ({
                you: {
                    isBookmarked: await BookmarkModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                    isOwn: objects.map((x) => Boolean(userData) && x.createdByUserId === userData?.id),
                },
            }),
        },
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                tag: data.tag,
                createdBy: data.anonymous ? undefined : { connect: { id: rest.userData.id } },
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                ...(data.anonymous ? { createdBy: { disconnect: true } } : {}),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
        },
        yup: tagValidation,
    },
    search: {
        defaultSort: TagSortBy.BookmarksDesc,
        sortBy: TagSortBy,
        searchFields: {
            createdById: true,
            createdTimeFrame: true,
            excludeIds: true,
            maxBookmarks: true,
            minBookmarks: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transDescriptionWrapped",
                "tagWrapped",
            ],
        }),
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true }),
        permissionResolvers: defaultPermissions,
        owner: () => ({}),
        isDeleted: () => false,
        isPublic: () => true,
        profanityFields: ["tag"],
        visibility: {
            private: {},
            public: {},
            owner: () => ({}),
        },
    },
});
