import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MaxObjects, Note, NoteCreateInput, NoteSearchInput, NoteSortBy, NoteUpdateInput, NoteYou } from '@shared/consts';
import { PrismaType } from "../types";
import { getSingleTypePermissions } from "../validators";
import { NoteVersionModel } from "./noteVersion";
import { BookmarkModel } from "./bookmark";
import { ModelLogic } from "./types";
import { ViewModel } from "./view";
import { VoteModel } from "./vote";
import { rootObjectDisplay } from "../utils/rootObjectDisplay";
import { defaultPermissions, labelShapeHelper, onCommonRoot, ownerShapeHelper, preShapeRoot, tagShapeHelper } from "../utils";
import { OrganizationModel } from "./organization";
import { noteValidation } from "@shared/validation";
import { noNull, shapeHelper } from "../builders";

const __typename = 'Note' as const;
type Permissions = Pick<NoteYou, 'canDelete' | 'canUpdate' | 'canBookmark' | 'canTransfer' | 'canRead' | 'canVote'>;
const suppFields = ['you'] as const;
export const NoteModel: ModelLogic<{
    IsTransferable: true,
    IsVersioned: true,
    GqlCreate: NoteCreateInput,
    GqlUpdate: NoteUpdateInput,
    GqlModel: Note,
    GqlSearch: NoteSearchInput,
    GqlSort: NoteSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.noteUpsertArgs['create'],
    PrismaUpdate: Prisma.noteUpsertArgs['update'],
    PrismaModel: Prisma.noteGetPayload<SelectWrap<Prisma.noteSelect>>,
    PrismaSelect: Prisma.noteSelect,
    PrismaWhere: Prisma.noteWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.note,
    display: rootObjectDisplay(NoteVersionModel),
    format: {
        gqlRelMap: {
            __typename,
            createdBy: 'User',
            issues: 'Issue',
            labels: 'Label',
            owner: {
                ownedByUser: 'User',
                ownedByOrganization: 'Organization',
            },
            parent: 'Note',
            pullRequests: 'PullRequest',
            questions: 'Question',
            bookmarkedBy: 'User',
            tags: 'Tag',
            transfers: 'Transfer',
            versions: 'NoteVersion',
        },
        prismaRelMap: {
            __typename,
            parent: 'NoteVersion',
            createdBy: 'User',
            ownedByUser: 'User',
            ownedByOrganization: 'Organization',
            versions: 'NoteVersion',
            pullRequests: 'PullRequest',
            labels: 'Label',
            issues: 'Issue',
            tags: 'Tag',
            bookmarkedBy: 'User',
            questions: 'Question',
        },
        joinMap: { labels: 'label', bookmarkedBy: 'user', tags: 'tag' },
        countFields: {
            issuesCount: true,
            labelsCount: true,
            pullRequestsCount: true,
            questionsCount: true,
            transfersCount: true,
            versionsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        isBookmarked: await BookmarkModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        isViewed: await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename),
                        isUpvoted: await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename),
                    }
                }
            },
        },
    },
    mutate: {
        shape: {
            pre: async (params) => {
                const maps = await preShapeRoot({ ...params, objectType: __typename });
                return { ...maps }
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                isPrivate: noNull(data.isPrivate),
                permissions: noNull(data.permissions) ?? JSON.stringify({}),
                createdBy: rest.userData?.id ? { connect: { id: rest.userData.id } } : undefined,
                ...rest.preMap[__typename].versionMap[data.id],
                ...(await ownerShapeHelper({ relation: 'ownedBy', relTypes: ['Connect'], parentRelationshipName: 'notes', isCreate: true, objectType: __typename, data, ...rest })),
                ...(await shapeHelper({ relation: 'parent', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'NoteVersion', parentRelationshipName: 'forks', data, ...rest })),
                ...(await shapeHelper({ relation: 'versions', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'NoteVersion', parentRelationshipName: 'root', data, ...rest })),
                ...(await tagShapeHelper({ relTypes: ['Connect', 'Create'], parentType: 'Note', relation: 'tags', data, ...rest })),
                ...(await labelShapeHelper({ relTypes: ['Connect', 'Create'], parentType: 'Note', relation: 'labels', data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                isPrivate: noNull(data.isPrivate),
                permissions: noNull(data.permissions),
                ...rest.preMap[__typename].versionMap[data.id],
                ...(await ownerShapeHelper({ relation: 'ownedBy', relTypes: ['Connect'], parentRelationshipName: 'notes', isCreate: false, objectType: __typename, data, ...rest })),
                ...(await shapeHelper({ relation: 'versions', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'NoteVersion', parentRelationshipName: 'root', data, ...rest })),
                ...(await tagShapeHelper({ relTypes: ['Connect', 'Create', 'Disconnect'], parentType: 'Note', relation: 'tags', data, ...rest })),
                ...(await labelShapeHelper({ relTypes: ['Connect', 'Create', 'Disconnect'], parentType: 'Note', relation: 'labels', data, ...rest })),
            })
        },
        trigger: {
            onCommon: async (params) => {
                await onCommonRoot({ ...params, objectType: __typename });
            },
        },
        yup: noteValidation,
    },
    search: {
        defaultSort: NoteSortBy.DateUpdatedDesc,
        sortBy: NoteSortBy,
        searchFields: {
            createdById: true,
            createdTimeFrame: true,
            maxBookmarks: true,
            maxScore: true,
            minBookmarks: true,
            minScore: true,
            ownedByOrganizationId: true,
            ownedByUserId: true,
            parentId: true,
            tags: true,
            translationLanguagesLatestVersion: true,
            updatedTimeFrame: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                'tagsWrapped',
                'labelsWrapped',
                { versions: { some: 'transDescriptionWrapped' } },
                { versions: { some: 'transNameWrapped' } },
            ]
        })
    },
    validate: {
        hasCompleteVersion: () => true,
        hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
        isDeleted: () => false,
        isPublic: (data) => data.isPrivate === false,
        isTransferable: true,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Organization: data.ownedByOrganization,
            User: data.ownedByUser,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            isPrivate: true,
            permissions: true,
            createdBy: 'User',
            ownedByOrganization: 'Organization',
            ownedByUser: 'User',
            versions: ['NoteVersion', ['root']],
        }),
        visibility: {
            private: { isPrivate: true },
            public: { isPrivate: false },
            owner: (userId) => ({
                OR: [
                    { ownedByUser: { id: userId } },
                    { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
                ]
            }),
        },
    },
})