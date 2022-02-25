import { CODE, StarFor } from "@local/shared";
import { CustomError } from "../error";
import { Star, StarInput } from "../schema/types";
import { PartialSelectConvert, PrismaType, RecursivePartial } from "../types";
import { BaseType, deconstructUnion, FormatConverter, FormatterMap, infoToPartialSelect, InfoType, MODEL_TYPES, relationshipFormatter } from "./base";
import _ from "lodash";
import { star } from "@prisma/client";
import { commentDBFields } from "./comment";
import { projectDBFields } from "./project";
import { routineDBFields } from "./routine";
import { standardDBFields } from "./standard";
import { organizationDBFields } from "./organization";
import { tagDBFields } from "./tag";
import { userDBFields } from "./user";

//==============================================================
/* #region Custom Components */
//==============================================================

type StarFormatterType = FormatConverter<Star, star>;
/**
 * Component for formatting between graphql and prisma types
 */
export const starFormatter = (): StarFormatterType => {
    return {
        dbShape: (partial: PartialSelectConvert<Star>): PartialSelectConvert<star> => {
            let modified = partial;
            console.log('star toDB', modified);
            // Deconstruct GraphQL unions
            modified = deconstructUnion(modified, 'to', [
                ['comment', {
                    ...Object.fromEntries(commentDBFields.map(f => [f, true]))
                }],
                ['organization', {
                    ...Object.fromEntries(organizationDBFields.map(f => [f, true]))
                }],
                ['project', {
                    ...Object.fromEntries(projectDBFields.map(f => [f, true]))
                }],
                ['routine', {
                    ...Object.fromEntries(routineDBFields.map(f => [f, true]))
                }],
                ['standard', {
                    ...Object.fromEntries(standardDBFields.map(f => [f, true]))
                }],
                ['tag', {
                    ...Object.fromEntries(tagDBFields.map(f => [f, true]))
                }],
                ['user', {
                    ...Object.fromEntries(userDBFields.map(f => [f, true]))
                }],
            ]);
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['from', FormatterMap.User.dbShapeUser],
            ]);
            return modified;
        },
        dbPrune: (info: InfoType): PartialSelectConvert<star> => {
            // Convert GraphQL info object to a partial select object
            let modified = infoToPartialSelect(info);
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['from', FormatterMap.User.dbPruneUser],
            ]);
            return modified;
        },
        selectToDB: (info: InfoType): PartialSelectConvert<star> => {
            return starFormatter().dbShape(starFormatter().dbPrune(info));
        },
        selectToGraphQL: (obj: RecursivePartial<any>): RecursivePartial<Star> => {
            console.log('starFormatter.toGraphQL start', obj);
            // Create unions
            let { comment, organization, project, routine, standard, tag, user, ...modified } = obj;
            if (comment) modified.to = FormatterMap.Comment.selectToGraphQL(comment);
            else if (organization) modified.to = FormatterMap.Organization.selectToGraphQL(organization);
            else if (project) modified.to = FormatterMap.Project.selectToGraphQL(project);
            else if (routine) modified.to = FormatterMap.Routine.selectToGraphQL(routine);
            else if (standard) modified.to = FormatterMap.Standard.selectToGraphQL(standard);
            else if (tag) modified.to = FormatterMap.Tag.selectToGraphQL(tag);
            else if (user) modified.to = FormatterMap.User.selectToGraphQLUser(user);
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['from', FormatterMap.User.selectToGraphQLUser],
            ]);
            console.log('starFormatter.toGraphQL finished', modified);
            return modified;
        },
    }
}

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
    async getIsStarreds(
        userId: string,
        ids: string[],
        starFor: StarFor
    ): Promise<boolean[]> {
        const fieldName = `${starFor.toLowerCase()}Id`;
        const isStarredArray = await prisma.star.findMany({ where: { byId: userId, [fieldName]: { in: ids } } });
        return ids.map(id => {
            const isStarred = isStarredArray.find((star: any) => star[fieldName] === id);
            return Boolean(isStarred);
        });
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function StarModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Star;
    const format = starFormatter();

    return {
        prisma,
        model,
        ...format,
        ...starrer(prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================