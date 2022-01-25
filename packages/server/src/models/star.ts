import { CODE, StarFor } from "@local/shared";
import { CustomError } from "../error";
import { StarInput } from "schema/types";
import { PrismaType, RecursivePartial } from "../types";
import { BaseType, FormatConverter, MODEL_TYPES } from "./base";
import { UserDB } from "./user";
import { CommentDB, OrganizationDB, ProjectDB, RoutineDB, StandardDB, TagDB } from "../models";
import { comment } from "@prisma/client";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type StarRelationshipList = 'comment' | 'organization' | 'project' | 'routine' | 'standard' | 'starredUser' | 'tag' | 'user';
// Type 2. QueryablePrimitives
export type StarQueryablePrimitives = {};
// Type 3. AllPrimitives
export type StarAllPrimitives = StarQueryablePrimitives & {
    id: string;
    userId: string;
    commentId?: string;
    projectId?: string;
    routineId?: string;
    standardId?: string;
    tagId?: string;
    starredUserId?: string;
};
// type 4. Database shape
export type StarDB = StarAllPrimitives & {
    user: UserDB;
    comment?: CommentDB;
    organization?: OrganizationDB;
    project?: ProjectDB;
    routine?: RoutineDB;
    standard?: StandardDB;
    tag?: TagDB;
    starredUser?: UserDB;
}

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

const forMapper = {
    [StarFor.Comment]: 'comment',
    [StarFor.Organization]: 'organization',
    [StarFor.Project]: 'project',
    [StarFor.Routine]: 'routine',
    [StarFor.Standard]: 'standard',
    [StarFor.Tag]: 'tag',
    [StarFor.User]: 'user',
}

/**
 * Casts stars. Makes sure not to duplicate.
 * A user may star their own project/routine/etc, but why would you want to?
 * @returns True if cast correctly (even if skipped because of duplicate)
 */
const starrer = (prisma: PrismaType) => ({
    async star(userId: string, input: StarInput): Promise<boolean> {
        console.log('going to star', input)
        // Define prisma type for object being starred
        const prismaFor = (prisma[forMapper[input.starFor] as keyof PrismaType] as BaseType);
        console.log('prismaFor', prismaFor)
        // Check if object being starred exists
        const starringFor: null | { id: string, stars: number } = await prismaFor.findUnique({ where: { id: input.forId }, select: { id: true, stars: true } });
        if (!starringFor) throw new CustomError(CODE.ErrorUnknown);
        // Check if star already exists on object by this user
        const star = await prisma.star.findFirst({
            where: {
                byId: userId,
                [`${forMapper[input.starFor]}Id`]: input.forId
            }
        })
        console.log('existing star', star)
        // If star already existed and we want to star, 
        // or if star did not exist and we don't want to star, skip
        if ((star && input.isStar) || (!star && !input.isStar)) return true;
        // If star did not exist and we want to star, create
        if (!star && input.isStar) {
            // Create
            await prisma.star.create({
                data: {
                    byId: userId,
                    [`${forMapper[input.starFor]}Id`]: input.forId
                }
            })
            // Increment star count on object
            await prismaFor.update({
                where: { id: input.forId },
                data: { stars: starringFor.stars + 1 }
            })
        }
        // If star did exist and we don't want to star, delete
        else if (star && !input.isStar) {
            // Delete star
            await prisma.star.delete({ where: { id: star.id } })
            // Decrement star count on object
            await prismaFor.update({
                where: { id: input.forId },
                data: { stars: starringFor.stars - 1 }
            })
        }
        return true;
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function StarModel(prisma: PrismaType) {
    return {
        prisma,
        ...starrer(prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================