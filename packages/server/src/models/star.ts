import { CODE, StarFor } from "@local/shared";
import { CustomError } from "../error";
import { Star, StarInput } from "../schema/types";
import { PrismaType } from "../types";
import { deconstructUnion, FormatConverter, GraphQLModelType } from "./base";
import _ from "lodash";

//==============================================================
/* #region Custom Components */
//==============================================================

export const starFormatter = (): FormatConverter<Star> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.Star,
        'from': GraphQLModelType.User,
        'to': {
            'Comment': GraphQLModelType.Comment,
            'Organization': GraphQLModelType.Organization,
            'Project': GraphQLModelType.Project,
            'Routine': GraphQLModelType.Routine,
            'Standard': GraphQLModelType.Standard,
            'Tag': GraphQLModelType.Tag,
            'User': GraphQLModelType.User,
        }
    },
    constructUnions: (data) => {
        let { comment, organization, project, routine, standard, tag, user, ...modified } = data;
        if (comment) modified.to = comment;
        else if (organization) modified.to = organization;
        else if (project) modified.to = project;
        else if (routine) modified.to = routine;
        else if (standard) modified.to = standard;
        else if (tag) modified.to = tag;
        else if (user) modified.to = user;
        return modified;
    },
    deconstructUnions: (partial) => {
        let modified = deconstructUnion(partial, 'to', [
            [GraphQLModelType.Comment, 'comment'],
            [GraphQLModelType.Organization, 'organization'],
            [GraphQLModelType.Project, 'project'],
            [GraphQLModelType.Routine, 'routine'],
            [GraphQLModelType.Standard, 'standard'],
            [GraphQLModelType.Tag, 'tag'],
            [GraphQLModelType.User, 'user'],
        ]);
        return modified;
    },
})

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
        // Define prisma type for object being starred
        const prismaFor = (prisma[forMapper[input.starFor] as keyof PrismaType] as any);
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
    async getIsStarreds(
        userId: string,
        ids: string[],
        starFor: StarFor
    ): Promise<boolean[]> {
        // Create result array that is the same length as ids
        const result = new Array(ids.length).fill(false);
        // Filter out nulls and undefineds from ids
        const idsFiltered = ids.filter(id => id !== null && id !== undefined);
        const fieldName = `${starFor.toLowerCase()}Id`;
        const isStarredArray = await prisma.star.findMany({ where: { byId: userId, [fieldName]: { in: idsFiltered } } });
        // Replace the nulls in the result array with true or false
        for (let i = 0; i < ids.length; i++) {
            // check if this id is in isStarredArray
            if (ids[i] !== null && ids[i] !== undefined && 
                isStarredArray.find((star: any) => star[fieldName] === ids[i])) {
                result[i] = true;
            }
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

export function StarModel(prisma: PrismaType) {
    const prismaObject = prisma.star;
    const format = starFormatter();

    return {
        prisma,
        prismaObject,
        ...format,
        ...starrer(prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================