import { GqlModelType, snakeCase } from "@local/shared";

type PrismaSelect = Record<string, any>;

function mockFindUnique<T>(records: T[], args: { where: { id: string }, select?: PrismaSelect }): Promise<T | null> {
    const item = records.find(item => (item as any).id === args.where.id);
    if (!item) return Promise.resolve(null);

    // Return only the fields that are requested in the select clause
    const result = constructSelectResponse(item, args.select);
    return Promise.resolve(result) as any;
}

function constructSelectResponse<T>(item: T, select?: Record<string, boolean>): Partial<T> {
    if (!select) return item; // If no select clause, return the whole item

    const constructNestedResponse = (nestedItem: any, nestedSelect: Record<string, any>) => {
        return Object.keys(nestedSelect).reduce((acc, key) => {
            if (nestedSelect[key] === true) {
                if (nestedItem[key] !== undefined) {
                    acc[key] = nestedItem[key];
                }
            } else if (typeof nestedSelect[key] === "object" && nestedItem[key] !== undefined) {
                // Check for Prisma pattern where relations are wrapped in "select"
                if (nestedSelect[key].select) {
                    acc[key] = constructNestedResponse(nestedItem[key], nestedSelect[key].select);
                } else {
                    acc[key] = constructNestedResponse(nestedItem[key], nestedSelect[key]);
                }
            }
            return acc;
        }, {});
    };

    const result = constructNestedResponse(item, select);
    return result as Partial<T>;
}

/**
 * Instead of mocking the prisma package directly, this 
 * returns a mock object which can be passed in as a PrismaType
 */
export const mockPrisma = (data: { [key in GqlModelType]?: any[] }) => {
    const prismaMock = {};

    Object.entries(data).forEach(([modelType, records]) => {
        const modelName = snakeCase(modelType);

        prismaMock[modelName] = {
            findUnique: jest.fn((args) => mockFindUnique(records, args)),
            // Add other methods here as needed
        };
    });

    return prismaMock;
};
