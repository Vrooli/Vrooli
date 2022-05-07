import { CODE, MemberRole, ViewFor } from "@local/shared";
import { CustomError } from "../../error";
import { Count, LogType, User } from "../../schema/types";
import { PrismaType } from "../../types";
import { deconstructUnion, FormatConverter, GraphQLModelType } from "./base";
import _ from "lodash";
import { genErrorCode } from "../../logger";
import { Log } from "../../models/nosql";
import { OrganizationModel } from "./organization";
import { initializeRedis } from "redisConn";

//==============================================================
/* #region Custom Components */
//==============================================================

export interface View {
    __typename?: 'View';
    from: User;
    to: ViewFor;
}

export const viewFormatter = (): FormatConverter<View> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.View,
        'from': GraphQLModelType.User,
        'to': {
            'Organization': GraphQLModelType.Organization,
            'Project': GraphQLModelType.Project,
            'Routine': GraphQLModelType.Routine,
            'Standard': GraphQLModelType.Standard,
            'User': GraphQLModelType.User,
        }
    },
    constructUnions: (data) => {
        let { organization, project, routine, standard, user, ...modified } = data;
        if (organization) modified.to = organization;
        else if (project) modified.to = project;
        else if (routine) modified.to = routine;
        else if (standard) modified.to = standard;
        else if (user) modified.to = user;
        return modified;
    },
    deconstructUnions: (partial) => {
        let modified = deconstructUnion(partial, 'to', [
            [GraphQLModelType.Organization, 'organization'],
            [GraphQLModelType.Project, 'project'],
            [GraphQLModelType.Routine, 'routine'],
            [GraphQLModelType.Standard, 'standard'],
            [GraphQLModelType.User, 'user'],
        ]);
        return modified;
    },
})

const forMapper = {
    [ViewFor.Organization]: 'organization',
    [ViewFor.Project]: 'project',
    [ViewFor.Routine]: 'routine',
    [ViewFor.Standard]: 'standard',
    [ViewFor.User]: 'user',
}

export interface ViewInput {
    forId: string;
    title: string;
    viewFor: ViewFor;
}

/**
 * Marks objects as viewed. If view exists, updates last viewed time.
 * A user may view their own objects, but it does not count towards its view count.
 * @returns True if view updated correctly
 */
const viewer = (prisma: PrismaType) => ({
    async view(userId: string, input: ViewInput): Promise<boolean> {
        console.log('creating view', JSON.stringify(input));
        // Define prisma type for viewed object
        const prismaFor = (prisma[forMapper[input.viewFor] as keyof PrismaType] as any);
        // Check if object being viewed on exists
        const viewFor: null | { id: string, views: number } = await prismaFor.findUnique({ where: { id: input.forId }, select: { id: true, views: true } });
        if (!viewFor)
            throw new CustomError(CODE.ErrorUnknown, 'Could not find object being viewed', { code: genErrorCode('0173') });
        // Check if view exists
        let view = await prisma.view.findFirst({
            where: {
                byId: userId,
                [`${forMapper[input.viewFor]}Id`]: input.forId
            }
        })
        // If view already existed, update view time
        if (view) {
            console.log('view existed. updating time', JSON.stringify(view))
            await prisma.view.update({
                where: { id: view.id },
                data: {
                    lastViewed: new Date(),
                    title: input.title,
                }
            })
        }
        // If view did not exist, create it
        else {
            console.log('view did not exist. creating', JSON.stringify(input))
            view = await prisma.view.create({
                data: {
                    byId: userId,
                    [`${forMapper[input.viewFor]}Id`]: input.forId,
                    title: input.title,
                }
            })
        }
        // Check if a view from this user should increment the view count
        let isOwn = false;
        switch (input.viewFor) {
            case ViewFor.Organization:
                // Check if user is an admin or owner of the organization
                const memberData = await OrganizationModel(prisma).isOwnerOrAdmin(userId, [input.forId]);
                isOwn = Boolean(memberData[0]);
                break;
            case ViewFor.Project:
                // Check if project is owned by this user or by an organization they are a member of
                const project = await prisma.project.findFirst({
                    where: {
                        AND: [
                            { id: input.forId },
                            {
                                OR: [
                                    { user: { id: userId } },
                                    {
                                        organization: {
                                            members: {
                                                some: {
                                                    id: userId,
                                                    role: { in: [MemberRole.Admin as any, MemberRole.Owner as any] },
                                                }
                                            }
                                        }
                                    }
                                ]
                            }
                        ]
                    }
                })
                if (project) isOwn = true;
                break;
            case ViewFor.Routine:
                // Check if routine is owned by this user or by an organization they are a member of
                const routine = await prisma.routine.findFirst({
                    where: {
                        AND: [
                            { id: input.forId },
                            {
                                OR: [
                                    { user: { id: userId } },
                                    {
                                        organization: {
                                            members: {
                                                some: {
                                                    id: userId,
                                                    role: { in: [MemberRole.Admin as any, MemberRole.Owner as any] },
                                                }
                                            }
                                        }
                                    }
                                ]
                            }
                        ]
                    }
                })
                if (routine) isOwn = true;
                break;
            case ViewFor.Standard:
                // Check if standard is owned by this user or by an organization they are a member of
                const standard = await prisma.standard.findFirst({
                    where: {
                        AND: [
                            { id: input.forId },
                            {
                                OR: [
                                    { createdByUser: { id: userId } },
                                    {
                                        createdByOrganization: {
                                            members: {
                                                some: {
                                                    id: userId,
                                                    role: { in: [MemberRole.Admin as any, MemberRole.Owner as any] },
                                                }
                                            }
                                        }
                                    }
                                ]
                            }
                        ]
                    }
                })
                if (standard) isOwn = true;
                break;
            case ViewFor.User:
                isOwn = userId === input.forId;
                break;
        }
        // If user is owner, don't do anything else
        if (isOwn) return true;
        // Check the last time the user viewed this object
        const redisKey = `view:${userId}_${input.forId}_${input.viewFor}`
        const client = await initializeRedis();
        const lastViewed = await client.get(redisKey);
        // If object viewed more than 1 hour ago, update view count
        if (!lastViewed || new Date(lastViewed).getTime() < new Date().getTime() - 3600000) {
            await prismaFor.update({
                where: { id: input.forId },
                data: { views: viewFor.views + 1 }
            })
            // Log view
            Log.collection.insertOne({
                timestamp: Date.now(),
                userId: userId,
                action: LogType.View,
                object1Type: input.viewFor,
                object1Id: input.forId,
            })
        }
        // Update last viewed time
        await client.set(redisKey, new Date().toISOString());
        return true;
    },
    /**
     * Deletes views from user's view list, but does not affect view count or logs.
     */
    async deleteViews(userId: string, ids: string[]): Promise<Count> {
        return await prisma.view.deleteMany({
            where: {
                AND: [
                    { id: { in: ids } },
                    { byId: userId },
                ]
            }
        })
    },
    /**
     * Removes all of user's views, but does not affect view count or logs.
     */
    async clearViews(userId: string): Promise<Count> {
        return await prisma.view.deleteMany({
            where: { byId: userId }
        })
    },
    async getIsVieweds(
        userId: string,
        ids: string[],
        viewFor: keyof typeof ViewFor
    ): Promise<Array<boolean | null>> {
        // Create result array that is the same length as ids
        const result = new Array(ids.length).fill(null);
        // Filter out nulls and undefineds from ids
        const idsFiltered = ids.filter(id => id !== null && id !== undefined);
        const fieldName = `${viewFor.toLowerCase()}Id`;
        const isViewedArray = await prisma.view.findMany({ where: { byId: userId, [fieldName]: { in: idsFiltered } } });
        // Replace the nulls in the result array with true if viewed
        for (let i = 0; i < ids.length; i++) {
            // Try to find this id in the isViewed array
            result[i] = Boolean(isViewedArray.find((view: any) => view[fieldName] === ids[i]));
        }
        return result;
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function ViewModel(prisma: PrismaType) {
    const prismaObject = prisma.view;
    const format = viewFormatter();

    return {
        prisma,
        prismaObject,
        ...format,
        ...viewer(prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================