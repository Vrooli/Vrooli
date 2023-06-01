import { MaxObjects, StandardCreateInput, StandardSortBy, standardValidation, StandardYou } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../builders";
import { getLabels } from "../getters";
import { PrismaType, SessionUserToken } from "../types";
import { defaultPermissions, labelShapeHelper, onCommonRoot, oneIsPublic, ownerShapeHelper, preShapeRoot, tagShapeHelper } from "../utils";
import { rootObjectDisplay } from "../utils/rootObjectDisplay";
import { getSingleTypePermissions } from "../validators";
import { BookmarkModel } from "./bookmark";
import { OrganizationModel } from "./organization";
import { ReactionModel } from "./reaction";
import { StandardVersionModel } from "./standardVersion";
import { ModelLogic, StandardModelLogic } from "./types";
import { ViewModel } from "./view";

const __typename = "Standard" as const;
type Permissions = Pick<StandardYou, "canDelete" | "canUpdate" | "canBookmark" | "canTransfer" | "canRead" | "canReact">;
const suppFields = ["you", "translatedName"] as const;
export const StandardModel: ModelLogic<StandardModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.standard,
    display: rootObjectDisplay(StandardVersionModel),
    format: {
        gqlRelMap: {
            __typename,
            createdBy: "User",
            issues: "Issue",
            labels: "Label",
            owner: {
                ownedByUser: "User",
                ownedByOrganization: "Organization",
            },
            parent: "Project",
            pullRequests: "PullRequest",
            questions: "Question",
            bookmarkedBy: "User",
            tags: "Tag",
            transfers: "Transfer",
            versions: "StandardVersion",
        },
        prismaRelMap: {
            __typename,
            createdBy: "User",
            ownedByOrganization: "Organization",
            ownedByUser: "User",
            issues: "Issue",
            labels: "Label",
            parent: "StandardVersion",
            tags: "Tag",
            bookmarkedBy: "User",
            versions: "StandardVersion",
            pullRequests: "PullRequest",
            stats: "StatsStandard",
            questions: "Question",
            transfers: "Transfer",
        },
        joinMap: { labels: "label", tags: "tag", bookmarkedBy: "user" },
        countFields: {
            forksCount: true,
            issuesCount: true,
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
                        reaction: await ReactionModel.query.getReactions(prisma, userData?.id, ids, __typename),
                    },
                    "translatedName": await getLabels(ids, __typename, prisma, userData?.languages ?? ["en"], "project.translatedName"),
                };
            },
        },
    },
    mutate: {
        shape: {
            pre: async (params) => {
                const maps = await preShapeRoot({ ...params, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                isInternal: noNull(data.isInternal),
                isPrivate: noNull(data.isPrivate),
                permissions: noNull(data.permissions) ?? JSON.stringify({}),
                createdBy: rest.userData?.id ? { connect: { id: rest.userData.id } } : undefined,
                ...rest.preMap[__typename].versionMap[data.id],
                ...(await ownerShapeHelper({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "standards", isCreate: true, objectType: __typename, data, ...rest })),
                ...(await shapeHelper({ relation: "parent", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "StandardVersion", parentRelationshipName: "forks", data, ...rest })),
                ...(await shapeHelper({ relation: "versions", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "StandardVersion", parentRelationshipName: "root", data, ...rest })),
                ...(await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Standard", relation: "tags", data, ...rest })),
                ...(await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Standard", relation: "labels", data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                isInternal: noNull(data.isInternal),
                isPrivate: noNull(data.isPrivate),
                permissions: noNull(data.permissions),
                ...rest.preMap[__typename].versionMap[data.id],
                ...(await ownerShapeHelper({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "standards", isCreate: false, objectType: __typename, data, ...rest })),
                ...(await shapeHelper({ relation: "versions", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "StandardVersion", parentRelationshipName: "root", data, ...rest })),
                ...(await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Standard", relation: "tags", data, ...rest })),
                ...(await labelShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Standard", relation: "labels", data, ...rest })),
            }),
        },
        trigger: {
            onCommon: async (params) => {
                await onCommonRoot({ ...params, objectType: __typename });
            },
        },
        yup: standardValidation,
        /**
         * Add, update, or remove a one-to-one standard relationship. 
         * Due to some unknown Prisma bug, it won't let us create/update a standard directly
         * in the main mutation query like most other relationship builders. Instead, we 
         * must do this separately, and return the standard's ID.
         */
        // async relationshipBuilder(
        //     userId: string,
        //     data: { [x: string]: any },
        //     isAdd: boolean = true,
        // ): Promise<{ [x: string]: any } | undefined> {
        //     // Check if standard of same shape already exists with the same creator. 
        //     // If so, connect instead of creating a new standard
        //     const initialCreate = Array.isArray(data[`standardCreate`]) ?
        //         data[`standardCreate`].map((c: any) => c.id) :
        //         typeof data[`standardCreate`] === 'object' ? [data[`standardCreate`].id] :
        //             [];
        //     const initialConnect = Array.isArray(data[`standardConnect`]) ?
        //         data[`standardConnect`] :
        //         typeof data[`standardConnect`] === 'object' ? [data[`standardConnect`]] :
        //             [];
        //     const initialUpdate = Array.isArray(data[`standardUpdate`]) ?
        //         data[`standardUpdate`].map((c: any) => c.id) :
        //         typeof data[`standardUpdate`] === 'object' ? [data[`standardUpdate`].id] :
        //             [];
        //     const initialCombined = [...initialCreate, ...initialConnect, ...initialUpdate];
        //     // TODO this will have bugs
        //     if (initialCombined.length > 0) {
        //         const existingIds: string[] = [];
        //         // Find shapes of all initial standards
        //         for (const standard of initialCombined) {
        //             const initialShape = await this.shapeCreate(userId, standard);
        //             const exists = await querier().findMatchingStandardVersion(prisma, standard, userId, true, false)
        //             if (exists) existingIds.push(exists.id);
        //         }
        //         // All existing shapes are the new connects
        //         data[`standardConnect`] = existingIds;
        //         data[`standardCreate`] = initialCombined.filter((s: any) => !existingIds.includes(s.id));
        //     }
        //     return relationshipBuilderHelper({
        //         data,
        //         relationshipName: 'standard',
        //         isAdd,
        //         userId,
        //         shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
        //     });
        // },
    },
    query: {
        /**
         * Checks for existing standards with the same shape. Useful to avoid duplicates
         * @param prisma Prisma client
         * @param data StandardCreateData to check
         * @param userData The ID of the user creating the standard
         * @param uniqueToCreator Whether to check if the standard is unique to the user/organization 
         * @param isInternal Used to determine if the standard should show up in search results
         * @returns data of matching standard, or null if no match
         */
        async findMatchingStandardVersion(
            prisma: PrismaType,
            data: StandardCreateInput,
            userData: SessionUserToken,
            uniqueToCreator: boolean,
            isInternal: boolean,
        ): Promise<{ [x: string]: any } | null> {
            return null;
            // // Sort all JSON properties that are part of the comparison
            // const props = sortify(data.props, userData.languages);
            // const yup = data.yup ? sortify(data.yup, userData.languages) : null;
            // // Find all standards that match the given standard
            // const standards = await prisma.standard_version.findMany({
            //     where: {
            //         root: {
            //             isInternal: (isInternal === true || isInternal === false) ? isInternal : undefined,
            //             isDeleted: false,
            //             isPrivate: false,
            //             createdByUserId: (uniqueToCreator && !data.createdByOrganizationId) ? userData.id : undefined,
            //             createdByOrganizationId: (uniqueToCreator && data.createdByOrganizationId) ? data.createdByOrganizationId : undefined,
            //         },
            //         default: data.default ?? null,
            //         props: props,
            //         yup: yup,
            //     }
            // });
            // // If any standards match (should only ever be 0 or 1, but you never know) return the first one
            // if (standards.length > 0) {
            //     return standards[0];
            // }
            // // If no standards match, then data is unique. Return null
            // return null;
        },
    },
    search: {
        defaultSort: StandardSortBy.ScoreDesc,
        sortBy: StandardSortBy,
        searchFields: {
            createdById: true,
            createdTimeFrame: true,
            excludeIds: true,
            hasCompleteVersion: true,
            isInternal: true,
            issuesId: true,
            labelsIds: true,
            maxScore: true,
            maxBookmarks: true,
            maxViews: true,
            minScore: true,
            minBookmarks: true,
            minViews: true,
            ownedByOrganizationId: true,
            ownedByUserId: true,
            parentId: true,
            pullRequestsId: true,
            tags: true,
            translationLanguagesLatestVersion: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "tagsWrapped",
                "labelsWrapped",
                { versions: { some: "transNameWrapped" } },
                { versions: { some: "transDescriptionWrapped" } },
            ],
        }),
        /**
         * isInternal routines should never appear in the query, since they are 
         * only meant for a single input/output
         */
        customQueryData: () => ({ isInternal: true }),
    },
    validate: {
        hasCompleteVersion: (data) => data.hasCompleteVersion === true,
        hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
        isDeleted: (data) => data.isDeleted,
        isPublic: (data, languages) => data.isPrivate === false &&
            data.isDeleted === false &&
            oneIsPublic<Prisma.smart_contractSelect>(data, [
                ["ownedByOrganization", "Organization"],
                ["ownedByUser", "User"],
            ], languages),
        isTransferable: true,
        maxObjects: MaxObjects[__typename],
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            isInternal: true,
            isPrivate: true,
            isDeleted: true,
            permissions: true,
            createdBy: "User",
            ownedByOrganization: "Organization",
            ownedByUser: "User",
            versions: ["StandardVersion", ["root"]],
        }),
        owner: (data) => ({
            Organization: data.ownedByOrganization,
            User: data.ownedByUser,
        }),
        visibility: {
            private: { isPrivate: true, isDeleted: false },
            public: { isPrivate: false, isDeleted: false },
            owner: (userId) => ({
                OR: [
                    { ownedByUser: { id: userId } },
                    { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
                ],
            }),
        },
        // TODO perform unique checks: Check if standard with same createdByUserId, createdByOrganizationId, name, and version already exists with the same creator
        // TODO when deleting, anonymize standards which are being used by inputs/outputs
    },
});
