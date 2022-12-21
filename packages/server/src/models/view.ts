import { ViewFor, ViewSortBy } from "@shared/consts";
import { OrganizationModel } from "./organization";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { UserModel } from "./user";
import { StandardModel } from "./standard";
import { CustomError } from "../events";
import { initializeRedis } from "../redisConn";
import { ViewSearchInput, Count, SessionUser, View } from "../endpoints/types";
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { lowercaseFirstLetter, onlyValidIds, padSelect } from "../builders";
import { getLabels } from "../getters";
import { ApiModel, IssueModel, NoteModel, PostModel, QuestionModel, SmartContractModel } from ".";

const forMapper: { [key in ViewFor]: keyof PrismaType } = {
    Api: "api",
    Note: "note",
    Organization: "organization",
    Project: "project",
    Routine: "routine",
    Standard: "standard",
    User: "user",
}

interface ViewInput {
    forId: string;
    viewFor: ViewFor;
}

/**
 * Deletes views from user's view list, but does not affect view count or logs.
 */
const deleteViews = async (prisma: PrismaType, userId: string, ids: string[]): Promise<Count> => {
    return await prisma.view.deleteMany({
        where: {
            AND: [
                { id: { in: ids } },
                { byId: userId },
            ]
        }
    })
}

/**
 * Removes all of user's views, but does not affect view count or logs.
 */
const clearViews = async (prisma: PrismaType, userId: string): Promise<Count> => {
    return await prisma.view.deleteMany({
        where: { byId: userId }
    })
}

const __typename = 'View' as const;
const suppFields = [] as const;
export const ViewModel: ModelLogic<{
    IsTransferable: true,
    IsVersioned: true,
    GqlCreate: undefined,
    GqlUpdate: undefined,
    GqlModel: View,
    GqlSearch: ViewSearchInput,
    GqlSort: ViewSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.viewUpsertArgs['create'],
    PrismaUpdate: Prisma.viewUpsertArgs['update'],
    PrismaModel: Prisma.viewGetPayload<SelectWrap<Prisma.viewSelect>>,
    PrismaSelect: Prisma.viewSelect,
    PrismaWhere: Prisma.viewWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.view,
    display: {
        select: () => ({
            id: true,
            api: padSelect(ApiModel.display.select),
            organization: padSelect(OrganizationModel.display.select),
            question: padSelect(QuestionModel.display.select),
            note: padSelect(NoteModel.display.select),
            post: padSelect(PostModel.display.select),
            project: padSelect(ProjectModel.display.select),
            routine: padSelect(RoutineModel.display.select),
            smartContract: padSelect(SmartContractModel.display.select),
            standard: padSelect(StandardModel.display.select),
            user: padSelect(UserModel.display.select),
        }),
        label: (select, languages) => {
            if (select.api) return ApiModel.display.label(select.api as any, languages);
            if (select.organization) return OrganizationModel.display.label(select.organization as any, languages);
            if (select.question) return QuestionModel.display.label(select.question as any, languages);
            if (select.note) return NoteModel.display.label(select.note as any, languages);
            if (select.post) return PostModel.display.label(select.post as any, languages);
            if (select.project) return ProjectModel.display.label(select.project as any, languages);
            if (select.routine) return RoutineModel.display.label(select.routine as any, languages);
            if (select.smartContract) return SmartContractModel.display.label(select.smartContract as any, languages);
            if (select.standard) return StandardModel.display.label(select.standard as any, languages);
            if (select.user) return UserModel.display.label(select.user as any, languages);
            return '';
        }
    },
    format: {
        gqlRelMap: {
            __typename,
            by: 'User',
            to: {
                api: 'Api',
                issue: 'Issue',
                note: 'Note',
                organization: 'Organization',
                post: 'Post',
                project: 'Project',
                question: 'Question',
                routine: 'Routine',
                smartContract: 'SmartContract',
                standard: 'Standard',
                user: 'User',
            }
        },
        prismaRelMap: {
            __typename,
            by: 'User',
            api: 'Api',
            issue: 'Issue',
            note: 'Note',
            organization: 'Organization',
            post: 'Post',
            project: 'Project',
            question: 'Question',
            routine: 'Routine',
            smartContract: 'SmartContract',
            standard: 'Standard',
            user: 'User',
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
                'nameWrapped',
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
            ]
        }),
    },
    query: {
        async getIsVieweds(
            prisma: PrismaType,
            userId: string | null | undefined,
            ids: string[],
            viewFor: keyof typeof ViewFor
        ): Promise<Array<boolean | null>> {
            // Create result array that is the same length as ids
            const result = new Array(ids.length).fill(null);
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
    /**
     * Marks objects as viewed. If view exists, updates last viewed time.
     * A user may view their own objects, but it does not count towards its view count.
     * @returns True if view updated correctly
     */
    view: async (prisma: PrismaType, userData: SessionUser, input: ViewInput): Promise<boolean> => {
        // Define prisma type for viewed object
        const prismaFor = (prisma[forMapper[input.viewFor] as keyof PrismaType] as any);
        // Check if object being viewed on exists
        const viewFor: null | { id: string, views: number } = await prismaFor.findUnique({ where: { id: input.forId }, select: { id: true, views: true } });
        if (!viewFor)
            throw new CustomError('0173', 'NotFound', userData.languages);
        // Check if view exists
        let view = await prisma.view.findFirst({
            where: {
                byId: userData.id,
                [`${forMapper[input.viewFor]}Id`]: input.forId
            }
        })
        // If view already existed, update view time
        if (view) {
            await prisma.view.update({
                where: { id: view.id },
                data: {
                    lastViewedAt: new Date(),
                }
            })
        }
        // If view did not exist, create it
        else {
            const labels = await getLabels([{ id: input.forId, languages: userData.languages }], input.viewFor, prisma, userData.languages, 'view');
            view = await prisma.view.create({
                data: {
                    byId: userData.id,
                    [`${forMapper[input.viewFor]}Id`]: input.forId,
                    name: labels[0],
                }
            })
        }
        // Check if a view from this user should increment the view count
        let isOwn = false;
        switch (input.viewFor) {
            case ViewFor.Organization:
                // Check if user is an admin or owner of the organization
                const roles = await OrganizationModel.query.hasRole(prisma, userData.id, [input.forId]);
                isOwn = Boolean(roles[0]);
                break;
            case ViewFor.Api:
            case ViewFor.Note:
            case ViewFor.Project:
            case ViewFor.Routine:
            case ViewFor.Standard:
                // Check if project/routine is owned by this user or by an organization they are a member of
                const object = await (prisma[lowercaseFirstLetter(input.viewFor) as 'api' | 'project' | 'routine' | 'standard'] as any).findFirst({
                    where: {
                        AND: [
                            { id: input.forId },
                            {
                                OR: [
                                    OrganizationModel.query.isMemberOfOrganizationQuery(userData.id),
                                    { ownedByUser: { id: userData.id } },
                                ]
                            }
                        ]
                    }
                })
                if (object) isOwn = true;
                break;
            case ViewFor.User:
                isOwn = userData.id === input.forId;
                break;
        }
        // If user is owner, don't do anything else
        if (isOwn) return true;
        // Check the last time the user viewed this object
        const redisKey = `view:${userData.id}_${input.forId}_${input.viewFor}`
        const client = await initializeRedis();
        const lastViewed = await client.get(redisKey);
        // If object viewed more than 1 hour ago, update view count
        if (!lastViewed || new Date(lastViewed).getTime() < new Date().getTime() - 3600000) {
            await prismaFor.update({
                where: { id: input.forId },
                data: { views: viewFor.views + 1 }
            })
        }
        // Update last viewed time
        await client.set(redisKey, new Date().toISOString());
        return true;
    },
    deleteViews,
    clearViews,
})