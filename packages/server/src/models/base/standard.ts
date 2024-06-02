import { MaxObjects, StandardCreateInput, StandardSortBy, standardValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { getLabels } from "../../getters";
import { SessionUserToken } from "../../types";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { rootObjectDisplay } from "../../utils/rootObjectDisplay";
import { PreShapeRootResult, labelShapeHelper, ownerFields, preShapeRoot, tagShapeHelper } from "../../utils/shapes";
import { afterMutationsRoot } from "../../utils/triggers";
import { getSingleTypePermissions } from "../../validators";
import { StandardFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { BookmarkModelLogic, ReactionModelLogic, StandardModelInfo, StandardModelLogic, StandardVersionModelLogic, TeamModelLogic, ViewModelLogic } from "./types";

type StandardPre = PreShapeRootResult;

const __typename = "Standard" as const;
export const StandardModel: StandardModelLogic = ({
    __typename,
    dbTable: "standard",
    display: () => rootObjectDisplay(ModelMap.get<StandardVersionModelLogic>("StandardVersion")),
    format: StandardFormat,
    mutate: {
        shape: {
            pre: async (params): Promise<StandardPre> => {
                const maps = await preShapeRoot({ ...params, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as StandardPre;
                return {
                    id: data.id,
                    isInternal: noNull(data.isInternal),
                    isPrivate: data.isPrivate,
                    permissions: noNull(data.permissions) ?? JSON.stringify({}),
                    createdBy: rest.userData?.id ? { connect: { id: rest.userData.id } } : undefined,
                    ...preData.versionMap[data.id],
                    ...(await ownerFields({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "standards", isCreate: true, objectType: __typename, data, ...rest })),
                    parent: await shapeHelper({ relation: "parent", relTypes: ["Connect"], isOneToOne: true, objectType: "StandardVersion", parentRelationshipName: "forks", data, ...rest }),
                    versions: await shapeHelper({ relation: "versions", relTypes: ["Create"], isOneToOne: false, objectType: "StandardVersion", parentRelationshipName: "root", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Standard", data, ...rest }),
                    labels: await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Standard", data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as StandardPre;
                return {
                    isInternal: noNull(data.isInternal),
                    isPrivate: noNull(data.isPrivate),
                    permissions: noNull(data.permissions),
                    ...preData.versionMap[data.id],
                    ...(await ownerFields({ relation: "ownedBy", relTypes: ["Connect"], parentRelationshipName: "standards", isCreate: false, objectType: __typename, data, ...rest })),
                    versions: await shapeHelper({ relation: "versions", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "StandardVersion", parentRelationshipName: "root", data, ...rest }),
                    tags: await tagShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Standard", data, ...rest }),
                    labels: await labelShapeHelper({ relTypes: ["Connect", "Create", "Disconnect"], parentType: "Standard", data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsRoot({ ...params, objectType: __typename });
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
        //             const exists = await querier().findMatchingStandardVersion(standard, userId, true, false)
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
         * @param data StandardCreateData to check
         * @param userData The ID of the user creating the standard
         * @param uniqueToCreator Whether to check if the standard is unique to the user/team 
         * @param isInternal Used to determine if the standard should show up in search results
         * @returns data of matching standard, or null if no match
         */
        async findMatchingStandardVersion(
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
            // const standards = await prismaInstance.standard_version.findMany({
            //     where: {
            //         root: {
            //             isInternal: (isInternal === true || isInternal === false) ? isInternal : undefined,
            //             isDeleted: false,
            //             isPrivate: false,
            //             createdByUserId: (uniqueToCreator && !data.createdByTeamId) ? userData.id : undefined,
            //             createdByTeamId: (uniqueToCreator && data.createdByTeamId) ? data.createdByTeamId : undefined,
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
            ownedByTeamId: true,
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
        supplemental: {
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, userData)),
                        isBookmarked: await ModelMap.get<BookmarkModelLogic>("Bookmark").query.getIsBookmarkeds(userData?.id, ids, __typename),
                        isViewed: await ModelMap.get<ViewModelLogic>("View").query.getIsVieweds(userData?.id, ids, __typename),
                        reaction: await ModelMap.get<ReactionModelLogic>("Reaction").query.getReactions(userData?.id, ids, __typename),
                    },
                    "translatedName": await getLabels(ids, __typename, userData?.languages ?? ["en"], "project.translatedName"),
                };
            },
        },
    },
    validate: () => ({
        hasCompleteVersion: (data) => data.hasCompleteVersion === true,
        hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
        isDeleted: (data) => data.isDeleted,
        isPublic: (data, ...rest) => data.isPrivate === false &&
            data.isDeleted === false &&
            oneIsPublic<StandardModelInfo["PrismaSelect"]>([
                ["ownedByTeam", "Team"],
                ["ownedByUser", "User"],
            ], data, ...rest),
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
            ownedByTeam: "Team",
            ownedByUser: "User",
            versions: ["StandardVersion", ["root"]],
        }),
        owner: (data) => ({
            Team: data?.ownedByTeam,
            User: data?.ownedByUser,
        }),
        visibility: {
            private: { isPrivate: true, isDeleted: false },
            public: { isPrivate: false, isDeleted: false },
            owner: (userId) => ({
                OR: [
                    { ownedByTeam: ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(userId) },
                    { ownedByUser: { id: userId } },
                ],
            }),
        },
        // TODO perform unique checks: Check if standard with same createdByUserId, createdByTeamId, name, and version already exists with the same creator
        // TODO when deleting, anonymize standards which are being used by inputs/outputs
    }),
});
