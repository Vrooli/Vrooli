import { snakeCase, uuid } from "@local/shared";

type PrismaSelect = Record<string, any>;

function mockFindUnique<T>(records: T[], args: { where: Record<string, any>, select?: PrismaSelect }): Promise<T | null> {
    const whereKeys = Object.keys(args.where);
    const item = records.find(record =>
        whereKeys.every(key => (record as any)[key] === args.where[key]),
    );

    if (!item) return Promise.resolve(null);

    // Return only the fields that are requested in the select clause
    const result = constructSelectResponse(item, args.select);
    return Promise.resolve(result) as any;
}

function mockFindMany<T>(records: T[], args: { where: Record<string, any>, select?: PrismaSelect }): Promise<T[]> {
    const whereKeys = Object.keys(args.where);
    const filteredItems = records.filter(record =>
        whereKeys.every(key => {
            const condition = args.where[key];
            if (typeof condition === "object" && condition !== null) {
                // Handle case-insensitive equality
                if ("equals" in condition) {
                    const valueToCompare = (record as any)[key];
                    if ("mode" in condition && condition.mode === "insensitive") {
                        return valueToCompare.toLowerCase() === condition.equals.toLowerCase();
                    } else {
                        return valueToCompare === condition.equals;
                    }
                }
                // Add more conditions here as needed
            } else {
                // Direct equality
                return (record as any)[key] === condition;
            }
        }),
    );

    const results = filteredItems.map(item => constructSelectResponse(item, args.select));
    return Promise.resolve(results) as any;
}

function mockCreate<T>(records: T[], args: { data: T }): Promise<T> {
    const newItem = { id: uuid(), ...args.data };
    records.push(newItem);
    return Promise.resolve(newItem);
}

function constructSelectResponse<T>(item: T, select?: Record<string, boolean>): Partial<T> {
    if (!select) return item; // If no select clause, return the whole item

    function constructNestedResponse(nestedItem: any, nestedSelect: Record<string, any>) {
        return Object.keys(nestedSelect).reduce((acc, key) => {
            if (nestedSelect[key] === true) {
                if (nestedItem[key] !== undefined) {
                    acc[key] = nestedItem[key];
                }
            } else if (typeof nestedSelect[key] === "object" && nestedItem[key] !== undefined) {
                // Check for Prisma pattern where relations are wrapped in "select"
                if (nestedSelect[key].select) {
                    if (Array.isArray(nestedItem[key])) {
                        // Handle array of relations
                        acc[key] = nestedItem[key].map(item => constructNestedResponse(item, nestedSelect[key].select));
                    } else {
                        // Handle single relation object
                        acc[key] = constructNestedResponse(nestedItem[key], nestedSelect[key].select);
                    }
                } else {
                    acc[key] = constructNestedResponse(nestedItem[key], nestedSelect[key]);
                }
            }
            return acc;
        }, {});
    }

    const result = constructNestedResponse(item, select);
    return result as Partial<T>;
}

function mockUpdate<T>(records: T[], args: { where: Record<string, any>, data: T }): Promise<T> {
    const whereKeys = Object.keys(args.where);
    const index = records.findIndex(record =>
        whereKeys.every(key => (record as any)[key] === args.where[key]),
    );

    if (index === -1) throw new Error("Record not found");

    records[index] = { ...records[index], ...args.data };
    return Promise.resolve(records[index]);
}

function mockUpsert<T>(records: T[], args: { where: Record<string, any>, create: T, update: T }): Promise<T> {
    const existingItem = mockFindUnique(records, { where: args.where });
    return existingItem.then(item => {
        if (item) {
            return mockUpdate(records, { where: args.where, data: args.update });
        } else {
            return mockCreate(records, { data: args.create });
        }
    });
}

/**
 * Instead of mocking the prisma package directly, this 
 * returns a mock object which can be passed in as a PrismaType
 */
export const mockPrisma = (data: { [key in string]: any[] }) => {
    const prismaMock = {};

    Object.entries(data).forEach(([modelType, records]) => {
        const modelName = snakeCase(modelType);

        prismaMock[modelName] = {
            findUnique: jest.fn((args) => mockFindUnique(records, args)),
            findMany: jest.fn((args) => mockFindMany(records, args)),
            create: jest.fn((args) => mockCreate(records, args)),
            update: jest.fn((args) => mockUpdate(records, args)),
            upsert: jest.fn((args) => mockUpsert(records, args)),
            // Add other methods here as needed
        };
    });

    return prismaMock;
};
