import { PrismaType } from "types";
import { TagHidden } from "../../schema/types";
import { FormatConverter, GraphQLModelType } from "./base";

//==============================================================
/* #region Custom Components */
//==============================================================

export const tagHiddenFormatter = (): FormatConverter<TagHidden> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.TagHidden,
        'user': GraphQLModelType.User,
    },
})

//TODO
export const tagHiddenMutater = (prisma: PrismaType) => ({
    // async toDBShape(userId: string | null, data: TagHiddenCreateInput | TagHiddenUpdateInput): Promise<any> {
    //     return {
    //         name: data.tag,
    //         createdByUserId: userId,
    //         translations: TranslationModel().relationshipBuilder(userId, data, { create: tagTranslationCreate, update: tagTranslationUpdate }, false),
    //     }
    // },
    // /**
    //  * Maps type of a tag's parent with the unique field
    //  */
    // parentMapper: {
    //     [GraphQLModelType.Organization]: 'organization_tags_taggedid_tagid_unique',
    //     [GraphQLModelType.Project]: 'project_tags_taggedid_tagid_unique',
    //     [GraphQLModelType.Routine]: 'routine_tags_taggedid_tagid_unique',
    //     [GraphQLModelType.Standard]: 'standard_tags_taggedid_tagid_unique',
    // },
    // async relationshipBuilder(
    //     userId: string | null,
    //     input: { [x: string]: any },
    //     parentType: keyof typeof this.parentMapper,
    //     relationshipName: string = 'tags',
    // ): Promise<{ [x: string]: any } | undefined> {
    //     // If any tag creates, check if they're supposed to be connects
    //     if (Array.isArray(input[`${relationshipName}Create`])) {
    //         // Query for all creating tags
    //         const existingTags = await prisma.tag.findMany({
    //             where: { tag: { in: input[`${relationshipName}Create`].map((c: any) => c.tag) } },
    //             select: { id: true, tag: true }
    //         });
    //         // All results should be connects
    //         const connectTags = existingTags.map(t => ({ id: t.id }));
    //         // All tags that didn't exist are creates
    //         const createTags = input[`${relationshipName}Create`].filter((c: any) => !connectTags.some(t => t.id === c.id));
    //         input[`${relationshipName}Connect`] = Array.isArray(input[`${relationshipName}Connect`]) ? [...input[`${relationshipName}Connect`], ...connectTags] : createTags;
    //         input[`${relationshipName}Create`] = createTags;
    //     }
    //     // Validate create
    //     if (Array.isArray(input[`${relationshipName}Create`])) {
    //         for (const tag of input[`${relationshipName}Create`]) {
    //             // Check for valid arguments
    //             tagCreate.validateSync(tag, { abortEarly: false });
    //             // Check for censored words
    //             verifier.profanityCheck(tag as TagCreateInput);
    //         }
    //     }
    //     // Convert input to Prisma shape
    //     // Updating/deleting tags is not supported. This must be done in its own query.
    //     let formattedInput = joinRelationshipToPrisma({
    //         data: input,
    //         joinFieldName: 'tag',
    //         uniqueFieldName: this.parentMapper[parentType],
    //         childIdFieldName: 'tagId',
    //         parentIdFieldName: 'taggedId',
    //         parentId: input.id ?? null,
    //         relationshipName,
    //         isAdd: !Boolean(input.id),
    //         relExcludes: [RelationshipTypes.update, RelationshipTypes.delete],
    //     })
    //     return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    // },
    // async validateMutations({
    //     userId, createMany, updateMany, deleteMany
    // }: ValidateMutationsInput<TagCreateInput, TagUpdateInput>): Promise<void> {
    //     if ((createMany || updateMany || deleteMany) && !userId) throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations');
    //     if (createMany) {
    //         createMany.forEach(input => tagCreate.validateSync(input, { abortEarly: false }));
    //         createMany.forEach(input => verifier.profanityCheck(input));
    //         // Check for max tags on object TODO
    //     }
    //     if (updateMany) {
    //         updateMany.forEach(input => tagUpdate.validateSync(input.data, { abortEarly: false }));
    //         updateMany.forEach(input => verifier.profanityCheck(input.data));
    //     }
    // },
    // async cud({ partial, userId, createMany, updateMany, deleteMany }: CUDInput<TagCreateInput, TagUpdateInput>): Promise<CUDResult<Tag>> {
    //     await this.validateMutations({ userId, createMany, updateMany, deleteMany });
    //     // Perform mutations
    //     let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
    //     if (createMany) {
    //         // Loop through each create input
    //         for (const input of createMany) {
    //             // Call createData helper function
    //             const data = await this.toDBShape(input.anonymous ? null : userId, input);
    //             // Create object
    //             const currCreated = await prisma.tag.create({ data, ...selectHelper(partial) });
    //             // Convert to GraphQL
    //             const converted = modelToGraphQL(currCreated, partial);
    //             // Add to created array
    //             created = created ? [...created, converted] : [converted];
    //         }
    //     }
    //     if (updateMany) {
    //         // Loop through each update input
    //         for (const input of updateMany) {
    //             // Find in database
    //             let object = await prisma.tag.findFirst({
    //                 where: { ...input.where, createdByUserId: userId },
    //             })
    //             if (!object) throw new CustomError(CODE.ErrorUnknown);
    //             // Update object
    //             const currUpdated = await prisma.tag.update({
    //                 where: input.where,
    //                 data: await this.toDBShape(input.data.anonymous ? null : userId, input.data),
    //                 ...selectHelper(partial)
    //             });
    //             // Convert to GraphQL
    //             const converted = modelToGraphQL(currUpdated, partial);
    //             // Add to updated array
    //             updated = updated ? [...updated, converted] : [converted];
    //         }
    //     }
    //     if (deleteMany) {
    //         const tags = await prisma.tag.findMany({
    //             where: { id: { in: deleteMany } },
    //         })
    //         if (tags.some(t => t.createdByUserId !== userId)) throw new CustomError(CODE.Unauthorized, "You can't delete tags you didn't create");
    //         deleted = await prisma.tag.deleteMany({
    //             where: { id: { in: tags.map(t => t.id) } },
    //         });
    //     }
    //     return {
    //         created: createMany ? created : undefined,
    //         updated: updateMany ? updated : undefined,
    //         deleted: deleteMany ? deleted : undefined,
    //     };
    // },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function TagHiddenModel(prisma: PrismaType) {
    const prismaObject = prisma.wallet;
    const format = tagHiddenFormatter();
    const mutate = tagHiddenMutater(prisma);

    return {
        prisma,
        prismaObject,
        ...format,
        ...mutate
    }
}

//==============================================================
/* #endregion Model */
//==============================================================