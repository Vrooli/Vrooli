import { CODE, reportCreate, ReportFor, reportUpdate } from "@local/shared";
import { CustomError } from "../error";
import { Count, Report, ReportCreateInput, ReportUpdateInput } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { hasProfanity } from "../utils/censor";
import { addSupplementalFields, CUDInput, CUDResult, FormatConverter, infoToPartialSelect, InfoType, modelToGraphQL, selectHelper, ValidateMutationsInput } from "./base";

//==============================================================
/* #region Custom Components */
//==============================================================

export const reportFormatter = (): FormatConverter<Report> => ({
    removeCalculatedFields: (partial) => {
        let { isOwn, ...rest } = partial;
        // Add userId field so we can calculate isOwn
        return { ...rest, userId: true }
    },
    removeJoinTables: (data) => {
        // Remove userId to hide who submitted the report
        let { userId, ...rest } = data;
        return rest;
    },
    async addSupplementalFields(
        prisma: PrismaType,
        userId: string | null, // Of the user making the request
        objects: RecursivePartial<any>[],
        info: InfoType, // GraphQL select info
    ): Promise<RecursivePartial<Report>[]> {
        // Convert GraphQL info object to a partial select object
        const partial = infoToPartialSelect(info);
        // Query for isOwn
        if (partial.isOwn) objects = objects.map((x) => ({ ...x, isOwn: Boolean(userId) && x.fromId === userId }));
        // Convert Prisma objects to GraphQL objects
        return objects as RecursivePartial<Report>[];
    },
})

export const reportVerifier = () => ({
    profanityCheck(data: ReportCreateInput | ReportUpdateInput): void {
        if (hasProfanity(data.reason, data.details)) throw new CustomError(CODE.BannedWord);
    },
})

const forMapper = {
    [ReportFor.Comment]: 'commentId',
    [ReportFor.Organization]: 'organizationId',
    [ReportFor.Project]: 'projectId',
    [ReportFor.Routine]: 'routineId',
    [ReportFor.Standard]: 'standardId',
    [ReportFor.Tag]: 'tagId',
    [ReportFor.User]: 'userId',
}

export const reportMutater = (prisma: PrismaType, verifier: any) => ({
    async toDBShapeAdd(userId: string | null, data: ReportCreateInput): Promise<any> {
        return {
            reason: data.reason,
            details: data.details,
            from: { connect: { id: userId } },
            [forMapper[data.createdFor]]: data.createdForId,
        }
    },
    async toDBShapeUpdate(userId: string | null, data: ReportUpdateInput): Promise<any> {
        return {
            reason: data.reason ?? undefined,
            details: data.details,
        }
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<ReportCreateInput, ReportUpdateInput>): Promise<void> {
        if ((createMany || updateMany || deleteMany) && !userId) throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations');
        if (createMany) {
            createMany.forEach(input => reportCreate.validateSync(input, { abortEarly: false }));
            createMany.forEach(input => verifier.profanityCheck(input));
            // Check if report already exists by user on object TODO
        }
        if (updateMany) {
            updateMany.forEach(input => reportUpdate.validateSync(input, { abortEarly: false }));
            updateMany.forEach(input => verifier.profanityCheck(input));
        }
    },
    async cud({ info, userId, createMany, updateMany, deleteMany }: CUDInput<ReportCreateInput, ReportUpdateInput>): Promise<CUDResult<Report>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                const data = await this.toDBShapeAdd(userId, input);
                // Create object
                const currCreated = await prisma.report.create({ data, ...selectHelper(info) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, info);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Call createData helper function
                const data = await this.toDBShapeUpdate(userId, input);
                // Find in database
                let object = await prisma.report.findFirst({
                    where: {
                        id: input.id,
                        userId,
                    }
                })
                if (!object) throw new CustomError(CODE.ErrorUnknown);
                // Update object
                const currUpdated = await prisma.report.update({
                    where: { id: object.id },
                    data,
                    ...selectHelper(info)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, info);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            deleted = await prisma.report.deleteMany({
                where: {
                    AND: [
                        { id: { in: deleteMany } },
                        { userId },
                    ]
                }
            })
        }
        // Format and add supplemental/calculated fields
        const createdLength = created.length;
        const supplemental = await addSupplementalFields(prisma, userId, [...created, ...updated], info);
        created = supplemental.slice(0, createdLength);
        updated = supplemental.slice(createdLength);
        return {
            created: createMany ? created : undefined,
            updated: updateMany ? updated : undefined,
            deleted: deleteMany ? deleted : undefined,
        };
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function ReportModel(prisma: PrismaType) {
    const prismaObject = prisma.report;
    const format = reportFormatter();
    const verify = reportVerifier();
    const mutate = reportMutater(prisma, verify);

    return {
        prismaObject,
        ...format,
        ...verify,
        ...mutate,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================