import { Prisma } from "@prisma/client";
import { Count, GqlModelType, SessionUser, View, ViewFor, ViewSearchInput, ViewSortBy } from "@shared/consts";
import { ApiModel, IssueModel, NoteModel, PostModel, QuestionModel, SmartContractModel } from ".";
import { onlyValidIds, selPad } from "../builders";
import { SelectWrap } from "../builders/types";
import { CustomError } from "../events";
import { getLabels, getLogic } from "../getters";
import { initializeRedis } from "../redisConn";
import { PrismaType } from "../types";
import { OrganizationModel } from "./organization";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { StandardModel } from "./standard";
import { ModelLogic } from "./types";
import { UserModel } from "./user";

const toWhere = (key: string, nestedKey: string | null, id: string) => {
    if (nestedKey) return { [key]: { [nestedKey]: { some: { id } } } };
    return { [key]: { id } };
};

const toSelect = (key?: string) => {
    if (key) return { [key]: { select: { id: true, views: true } } };
    return { id: true, views: true };
};

const toData = (object: any, key?: string) => {
    if (key) return object[key];
    return object;
}

const toCreate = (object: any, relName: string, key?: string) => {
    if (key) return { [relName]: { connect: { id: object[key].id } } };
    return { [relName]: { connect: { id: object.id } } };
}

/**
 * Maps ViewFor types to partial query objects, used to determine 
 * if a view exists for a given object.
 */
const whereMapper = {
    Api: (id: string) => toWhere('api', null, id),
    ApiVersion: (id: string) => toWhere('api', 'versions', id),
    Note: (id: string) => toWhere('note', null, id),
    NoteVersion: (id: string) => toWhere('note', 'versions', id),
    Organization: (id: string) => toWhere('organization', null, id),
    Project: (id: string) => toWhere('project', null, id),
    ProjectVersion: (id: string) => toWhere('project', 'versions', id),
    Question: (id: string) => toWhere('question', null, id),
    Routine: (id: string) => toWhere('routine', null, id),
    RoutineVersion: (id: string) => toWhere('routine', 'versions', id),
    SmartContract: (id: string) => toWhere('smartContract', null, id),
    SmartContractVersion: (id: string) => toWhere('smartContract', 'versions', id),
    Standard: (id: string) => toWhere('standard', null, id),
    StandardVersion: (id: string) => toWhere('standard', 'versions', id),
    User: (id: string) => toWhere('user', null, id),
} as const;

/**
 * Maps ViewFor types to partial query objects, used to find data required to create a view.
 */
const selectMapper = {
    Api: toSelect(),
    ApiVersion: toSelect('root'),
    Note: toSelect(),
    NoteVersion: toSelect('root'),
    Organization: toSelect(),
    Project: toSelect(),
    ProjectVersion: toSelect('root'),
    Question: toSelect(),
    Routine: toSelect(),
    RoutineVersion: toSelect('root'),
    SmartContract: toSelect(),
    SmartContractVersion: toSelect('root'),
    Standard: toSelect(),
    StandardVersion: toSelect('root'),
    User: toSelect(),
} as const

/**
 * Maps object with selectMapper data to its corresponding id
 */
const dataMapper = {
    Api: (object: any) => toData(object),
    ApiVersion: (object: any) => toData(object, 'root'),
    Note: (object: any) => toData(object),
    NoteVersion: (object: any) => toData(object, 'root'),
    Organization: (object: any) => toData(object),
    Project: (object: any) => toData(object),
    ProjectVersion: (object: any) => toData(object, 'root'),
    Question: (object: any) => toData(object),
    Routine: (object: any) => toData(object),
    RoutineVersion: (object: any) => toData(object, 'root'),
    SmartContract: (object: any) => toData(object),
    SmartContractVersion: (object: any) => toData(object, 'root'),
    Standard: (object: any) => toData(object),
    StandardVersion: (object: any) => toData(object, 'root'),
    User: (object: any) => toData(object),
}

/**
 * Maps object with selectMapper data to a Prisma data object, to create a view.
 */
const createMapper = {
    Api: (object: any) => toCreate(object, 'api'),
    ApiVersion: (object: any) => toCreate(object, 'api', 'root'),
    Note: (object: any) => toCreate(object, 'note'),
    NoteVersion: (object: any) => toCreate(object, 'note', 'root'),
    Organization: (object: any) => toCreate(object, 'organization'),
    Project: (object: any) => toCreate(object, 'project'),
    ProjectVersion: (object: any) => toCreate(object, 'project', 'root'),
    Question: (object: any) => toCreate(object, 'question'),
    Routine: (object: any) => toCreate(object, 'routine'),
    RoutineVersion: (object: any) => toCreate(object, 'routine', 'root'),
    SmartContract: (object: any) => toCreate(object, 'smartContract'),
    SmartContractVersion: (object: any) => toCreate(object, 'smartContract', 'root'),
    Standard: (object: any) => toCreate(object, 'standard'),
    StandardVersion: (object: any) => toCreate(object, 'standard', 'root'),
    User: (object: any) => toCreate(object, 'user'),
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
    }).then(({ count }) => ({ __typename: 'Count' as const, count }))
}

/**
 * Removes all of user's views, but does not affect view count or logs.
 */
const clearViews = async (prisma: PrismaType, userId: string): Promise<Count> => {
    return await prisma.view.deleteMany({
        where: { byId: userId }
    }).then(({ count }) => ({ __typename: 'Count' as const, count }))
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
    GqlPermission: {},
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
            api: selPad(ApiModel.display.select),
            organization: selPad(OrganizationModel.display.select),
            question: selPad(QuestionModel.display.select),
            note: selPad(NoteModel.display.select),
            post: selPad(PostModel.display.select),
            project: selPad(ProjectModel.display.select),
            routine: selPad(RoutineModel.display.select),
            smartContract: selPad(SmartContractModel.display.select),
            standard: selPad(StandardModel.display.select),
            user: selPad(UserModel.display.select),
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
    /**
     * Marks objects as viewed. If view exists, updates last viewed time.
     * A user may view their own objects, but it does not count towards its view count.
     * @returns True if view updated correctly
     */
    view: async (prisma: PrismaType, userData: SessionUser, input: ViewInput): Promise<boolean> => {
        // Get Prisma delegate for viewed object
        const { delegate } = getLogic(['delegate'], input.viewFor, userData.languages, 'ViewModel.view');
        // Check if object being viewed on exists
        const objectToView: { [x: string]: any } | null = await delegate(prisma).findUnique({
            where: { id: input.forId },
            select: selectMapper[input.viewFor]
        });
        if (!objectToView)
            throw new CustomError('0173', 'NotFound', userData.languages);
        // Check if view exists
        let view = await prisma.view.findFirst({
            where: {
                by: { id: userData.id },
                ...whereMapper[input.viewFor](input.forId)
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
                    by: { connect: { id: userData.id } },
                    name: labels[0],
                    ...createMapper[input.viewFor](objectToView),
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
            case ViewFor.StandardVersion:
                // Check if ROOT object is owned by this user or by an organization they are a member of
                const { delegate: rootObjectDelegate } = getLogic(['delegate'], input.viewFor.replace('Version', '') as GqlModelType, userData.languages, 'ViewModel.view2');
                const rootObject = await rootObjectDelegate(prisma).findFirst({
                    where: {
                        AND: [
                            { id: dataMapper[input.viewFor](objectToView).id },
                            {
                                OR: [
                                    OrganizationModel.query.isMemberOfOrganizationQuery(userData.id),
                                    { ownedByUser: { id: userData.id } },
                                ]
                            }
                        ]
                    }
                })
                if (rootObject) isOwn = true;
                break;
            case ViewFor.Question:
                // Check if question was created by this user
                const question = await prisma.question.findFirst({ where: { id: input.forId, createdBy: { id: userData.id } } });
                if (question) isOwn = true;
                break;
            case ViewFor.User:
                isOwn = userData.id === input.forId;
                break;
        }
        // If user is owner, don't do anything else
        if (isOwn) return true;
        // Check the last time the user viewed this object
        const redisKey = `view:${userData.id}_${dataMapper[input.viewFor](objectToView).id}_${input.viewFor}`
        const client = await initializeRedis();
        const lastViewed = await client.get(redisKey);
        // If object viewed more than 1 hour ago, update view count
        if (!lastViewed || new Date(lastViewed).getTime() < new Date().getTime() - 3600000) {
            // View counts don't exist on versioned objects, so we must make sure we are updating the root object
            const { delegate: rootObjectDelegate } = getLogic(['delegate'], input.viewFor.replace('Version', '') as GqlModelType, userData.languages, 'ViewModel.view3');
            await rootObjectDelegate(prisma).update({
                where: { id: input.forId },
                data: { views: dataMapper[input.viewFor](objectToView).views + 1 }
            })
        }
        // Update last viewed time
        await client.set(redisKey, new Date().toISOString());
        return true;
    },
    deleteViews,
    clearViews,
})