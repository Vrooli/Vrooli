import { PrismaType } from "../types";
import { Count, LogSearchInput, LogSortBy, Log } from "../schema/types";
import { CUDInput, CUDResult, Searcher, selectHelper, modelToGraphQL, FormatConverter } from "./base";
import { CustomError } from "../error";
import { CODE } from "@local/shared";

//==============================================================
/* #region Custom Components */
//==============================================================

export const logFormatter = (): FormatConverter<Log> => ({
    relationshipMap: {},
})

export const logSearcher = (): Searcher<LogSearchInput> => ({
    defaultSort: LogSortBy.DateCreatedDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [LogSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [LogSortBy.DateCreatedDesc]: { created_at: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string, languages?: string[]): any => {
        //TODO
        return ({ })
    },
    customQueries(input: LogSearchInput): { [x: string]: any } {
        const table1Query = input.table1 ? { table1: input.table1 } : {};
        const table2Query = input.table2 ? { table2: input.table2 } : {};
        const object1IdQuery = input.object1Id ? { object1Id: input.object1Id } : {};
        const object2IdQuery = input.object2Id ? { object2Id: input.object2Id } : {};
        return { ...table1Query, ...table2Query, ...object1IdQuery, ...object2IdQuery };
    },
})

export const logMutater = (prisma: PrismaType) => ({
    /**
     * Performs adds and deletes of activities.
     */
    async cud({ partial, userId, createMany, deleteMany }: CUDInput<any, any>): Promise<CUDResult<Log>> {
        if (!userId) throw new CustomError(CODE.InternalError, 'User id is required in order to create/delete activities.');
        // Perform mutations. NOTE: Cannot update activities
        let created: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                const data = input;
                // Create object
                const currCreated = await prisma.log.create({ data, ...selectHelper(partial) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partial);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (deleteMany) {
            deleted = await prisma.log.deleteMany({
                where: {
                    AND: [
                        { id: { in: deleteMany } },
                        { userId },
                    ]
                }
            })
        }
        return {
            created: createMany ? created : undefined,
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

export function LogModel(prisma: PrismaType) {
    const prismaObject = prisma.log;
    const format = logFormatter();
    const mutate = logMutater(prisma);

    return {
        prisma,
        prismaObject,
        ...format,
        ...mutate,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================