import { Count, GqlModelType, View, ViewFor, ViewSearchInput, ViewSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import { ApiModel, IssueModel, NoteModel, PostModel, QuestionModel, SmartContractModel } from ".";
import { onlyValidIds, selPad } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { CustomError } from "../../events";
import { getLabels, getLogic } from "../../getters";
import { initializeRedis } from "../../redisConn";
import { PrismaType, SessionUserToken } from "../../types";
import { defaultPermissions } from "../../utils";
import { OrganizationModel } from "./organization";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { StandardModel } from "./standard";
import { ModelLogic } from "./types";
import { UserModel } from "./user";
import { Formatter } from "../types";

const __typename = "View" as const;
export const ViewFormat: Formatter<ModelViewLogic> = {
        gqlRelMap: {
            __typename,
            by: "User",
            to: {
                api: "Api",
                issue: "Issue",
                note: "Note",
                organization: "Organization",
                post: "Post",
                project: "Project",
                question: "Question",
                routine: "Routine",
                smartContract: "SmartContract",
                standard: "Standard",
                user: "User",
            },
        },
        prismaRelMap: {
            __typename,
            by: "User",
            api: "Api",
            issue: "Issue",
            note: "Note",
            organization: "Organization",
            post: "Post",
            project: "Project",
            question: "Question",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
            user: "User",
        },
        countFields: {},
    },
    search: {
        defaultSort: ViewSortBy.LastViewedDesc,
        sortBy: ViewSortBy,
        searchFields: {
            lastViewedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "nameWrapped",
                { api: ApiModel.search!.searchStringQuery() },
                { issue: IssueModel.search!.searchStringQuery() },
                { note: NoteModel.search!.searchStringQuery() },
                { organization: OrganizationModel.search!.searchStringQuery() },
                { question: QuestionModel.search!.searchStringQuery() },
                { post: PostModel.search!.searchStringQuery() },
                { project: ProjectModel.search!.searchStringQuery() },
                { routine: RoutineModel.search!.searchStringQuery() },
                { smartContract: SmartContractModel.search!.searchStringQuery() },
                { standard: StandardModel.search!.searchStringQuery() },
                { user: UserModel.search!.searchStringQuery() },
            ],
        }),
    },
    query: {
        async getIsVieweds(
            prisma: PrismaType,
            userId: string | null | undefined,
            ids: string[],
            viewFor: keyof typeof ViewFor,
        ): Promise<Array<boolean | null>> {
            // Create result array that is the same length as ids
            const result = new Array(ids.length).fill(false);
            // If userId not provided, return result
            if (!userId) return result;
            // Filter out nulls and undefineds from ids
            const idsFiltered = onlyValidIds(ids);
            const fieldName = `${viewFor.toLowerCase()}Id`;
            const isViewedArray = await prisma.view.findMany({ where: { byId: userId, [fieldName]: { in: idsFiltered } } });
            // Replace the nulls in the result array with true if viewed
            for (let i = 0; i < ids.length; i++) {
                // Try to find this id in the isViewed array
                result[i] = Boolean(isViewedArray.find((view: any) => view[fieldName] === ids[i]));
            }
            return result;
        },
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: 10000000,
        owner: (data) => ({
            User: data.by,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            by: "User",
        }),
        //TODO should set private/public based on viewed object's visibility, 
        //since you could view it when it was public and then it became private.
        //Should look into doing this for more than just views.
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                by: { id: userId },
            }),
        },
    },
    /**
     * Marks objects as viewed. If view exists, updates last viewed time.
     * A user may view their own objects, but it does not count towards its view count.
     * @returns True if view updated correctly
     */
    view: async (prisma: PrismaType, userData: SessionUserToken, input: ViewInput): Promise<boolean> => {
        // Get Prisma delegate for viewed object
        const { delegate } = getLogic(["delegate"], input.viewFor, userData.languages, "ViewModel.view");
        // Check if object being viewed on exists
        const objectToView: { [x: string]: any } | null = await delegate(prisma).findUnique({
            where: { id: input.forId },
            select: selectMapper[input.viewFor],
        });
        if (!objectToView)
            throw new CustomError("0173", "NotFound", userData.languages);
        // Check if view exists
        let view = await prisma.view.findFirst({
            where: {
                by: { id: userData.id },
                ...whereMapper[input.viewFor](input.forId),
            },
        });
        // If view already existed, update view time
        if (view) {
            await prisma.view.update({
                where: { id: view.id },
                data: {
                    lastViewedAt: new Date(),
                },
            });
        }
        // If view did not exist, create it
        else {
            const labels = await getLabels([{ id: input.forId, languages: userData.languages }], input.viewFor, prisma, userData.languages, "view");
            view = await prisma.view.create({
                data: {
                    by: { connect: { id: userData.id } },
                    name: labels[0],
                    ...createMapper[input.viewFor](objectToView),
                },
            });
        }
        // Check if a view from this user should increment the view count
        let isOwn = false;
        switch (input.viewFor) {
            case ViewFor.Organization: {
                // Check if user is an admin or owner of the organization
                const roles = await OrganizationModel.query.hasRole(prisma, userData.id, [input.forId]);
                isOwn = Boolean(roles[0]);
                break;
            }
            case ViewFor.Api:
            case ViewFor.ApiVersion:
            case ViewFor.Note:
            case ViewFor.NoteVersion:
            case ViewFor.Project:
            case ViewFor.ProjectVersion:
            case ViewFor.Routine:
            case ViewFor.RoutineVersion:
            case ViewFor.SmartContract:
            case ViewFor.SmartContractVersion:
            case ViewFor.Standard:
            case ViewFor.StandardVersion: {
                // Check if ROOT object is owned by this user or by an organization they are a member of
                const { delegate: rootObjectDelegate } = getLogic(["delegate"], input.viewFor.replace("Version", "") as GqlModelType, userData.languages, "ViewModel.view2");
                const rootObject = await rootObjectDelegate(prisma).findFirst({
                    where: {
                        AND: [
                            { id: dataMapper[input.viewFor](objectToView).id },
                            {
                                OR: [
                                    OrganizationModel.query.isMemberOfOrganizationQuery(userData.id),
                                    { ownedByUser: { id: userData.id } },
                                ],
                            },
                        ],
                    },
                });
                if (rootObject) isOwn = true;
                break;
            }
            case ViewFor.Question: {
                // Check if question was created by this user
                const question = await prisma.question.findFirst({ where: { id: input.forId, createdBy: { id: userData.id } } });
                if (question) isOwn = true;
                break;
            }
            case ViewFor.User:
                isOwn = userData.id === input.forId;
                break;
        }
        // If user is owner, don't do anything else
        if (isOwn) return true;
        // Check the last time the user viewed this object
        const redisKey = `view:${userData.id}_${dataMapper[input.viewFor](objectToView).id}_${input.viewFor}`;
        const client = await initializeRedis();
        const lastViewed = await client.get(redisKey);
        // If object viewed more than 1 hour ago, update view count
        if (!lastViewed || new Date(lastViewed).getTime() < new Date().getTime() - 3600000) {
            // View counts don't exist on versioned objects, so we must make sure we are updating the root object
            const { delegate: rootObjectDelegate } = getLogic(["delegate"], input.viewFor.replace("Version", "") as GqlModelType, userData.languages, "ViewModel.view3");
            await rootObjectDelegate(prisma).update({
                where: { id: input.forId },
                data: { views: dataMapper[input.viewFor](objectToView).views + 1 },
            });
        }
        // Update last viewed time
        await client.set(redisKey, new Date().toISOString());
        return true;
};
