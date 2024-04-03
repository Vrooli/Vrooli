import { snakeCase, uuid } from "@local/shared";

type PrismaSelect = Record<string, any>;

let globalDataStore = {};

const mockFindUnique = <T>(model: string, args: { where: Record<string, any>, select?: PrismaSelect }): Promise<T | null> => {
    const records = globalDataStore[model] || [];
    const whereKeys = Object.keys(args.where);
    const item = records.find(record =>
        whereKeys.every(key => (record as any)[key] === args.where[key]),
    );

    if (!item) return Promise.resolve(null);

    // Return only the fields that are requested in the select clause
    const result = constructSelectResponse(item, args.select);
    return Promise.resolve(result) as any;
};

const evaluateCondition = (record, key: string, condition) => {
    if (typeof condition === "object" && condition !== null) {
        // Handle "in" condition
        if ("in" in condition) {
            return condition.in.includes(record[key]);
        }
        // Handle case-insensitive equality
        if ("equals" in condition) {
            const valueToCompare = record[key];
            if ("mode" in condition && condition.mode === "insensitive") {
                return valueToCompare.toLowerCase() === condition.equals.toLowerCase();
            } else {
                return valueToCompare === condition.equals;
            }
        }
        // Handle nested relation or object conditions
        const relatedModelName = key.charAt(0).toUpperCase() + key.slice(1); // Assuming model names are capitalized
        if (globalDataStore[relatedModelName]) {
            const relatedRecords = globalDataStore[relatedModelName];
            return relatedRecords.some(relatedRecord => {
                const relationConditionKeys = Object.keys(condition);
                return relationConditionKeys.every(relationKey =>
                    evaluateCondition(relatedRecord, relationKey, condition[relationKey]),
                );
            });
        }
    } else {
        // Direct equality
        return record[key] === condition;
    }
};

const mockFindFirst = <T>(model: string, args: { where: Record<string, any>, select?: PrismaSelect }): Promise<T | null> => {
    const records = globalDataStore[model] || [];
    const whereKeys = Object.keys(args.where);

    const item = records.find(record =>
        whereKeys.every(key => evaluateCondition(record, key, args.where[key])),
    );

    if (!item) return Promise.resolve(null);

    const result = constructSelectResponse(item, args.select);
    return Promise.resolve(result) as any;
};

const mockFindMany = <T>(model: string, args: { where: Record<string, any>, select?: PrismaSelect }): Promise<T[]> => {
    const records = globalDataStore[model] || [];
    const whereKeys = Object.keys(args.where);

    const filteredItems = records.filter(record =>
        whereKeys.every(key => evaluateCondition(record, key, args.where[key])),
    );

    const results = filteredItems.map(item => constructSelectResponse(item, args.select));
    return Promise.resolve(results) as unknown as Promise<T[]>;
};

const mockCreate = <T>(model: string, args: { data: T }): Promise<T> => {
    const records = globalDataStore[model] || [];
    const newItem = { id: uuid(), ...args.data };
    records.push(newItem);
    return Promise.resolve(newItem);
};

const constructSelectResponse = <T>(item: T, select?: Record<string, any>): Partial<T> => {
    if (!select) return item; // If no select clause, return the whole item

    function constructNestedResponse(nestedItem: any, nestedSelect: Record<string, any>, parentModel?: string) {
        return Object.keys(nestedSelect).reduce((acc, key) => {
            if (nestedSelect[key] === true && nestedItem[key] !== undefined) {
                acc[key] = nestedItem[key];
            } else if (typeof nestedSelect[key] === "object" && nestedItem[key] !== undefined) {
                // Derive the related model name and ensure it matches keys in globalDataStore
                const relatedModelName = key.charAt(0).toUpperCase() + key.slice(1); // Adjust as necessary

                if (nestedSelect[key].select) {
                    if (Array.isArray(nestedItem[key])) {
                        // Initialize relatedModel in globalDataStore if not present
                        if (!globalDataStore[relatedModelName]) globalDataStore[relatedModelName] = [];

                        acc[key] = nestedItem[key].map(relatedItem => {
                            const relatedData = globalDataStore[relatedModelName].find(r => r.id === relatedItem.id);
                            return constructNestedResponse(relatedData ?? relatedItem, nestedSelect[key].select, relatedModelName);
                        });
                    } else {
                        // For single relation object, safely fetch related data or fallback
                        const relatedData = parentModel
                            ? nestedItem[key]
                            : globalDataStore[relatedModelName]?.find(r => r.id === nestedItem[key].id) ?? nestedItem[key];
                        acc[key] = constructNestedResponse(relatedData, nestedSelect[key].select, relatedModelName);
                    }
                } else {
                    acc[key] = constructNestedResponse(nestedItem[key], nestedSelect[key], parentModel);
                }
            }
            return acc;
        }, {});
    }

    const result = constructNestedResponse(item, select);
    return result as Partial<T>;
};

const mockUpdate = <T>(model: string, args: { where: Record<string, any>, data: T }): Promise<T> => {
    const records = globalDataStore[model] || [];
    const whereKeys = Object.keys(args.where);
    const index = records.findIndex(record =>
        whereKeys.every(key => (record as any)[key] === args.where[key]),
    );

    if (index === -1) throw new Error("Record not found");

    records[index] = { ...records[index], ...args.data };
    return Promise.resolve(records[index]);
};

const mockUpsert = <T>(model: string, args: { where: Record<string, any>, create: T, update: T }): Promise<T> => {
    const records = globalDataStore[model] || [];
    const existingItem = mockFindUnique(records, { where: args.where });
    return existingItem.then(item => {
        if (item) {
            return mockUpdate(records, { where: args.where, data: args.update });
        } else {
            return mockCreate(records, { data: args.create });
        }
    });
};

class PrismaClientMock {
    static instance = new PrismaClientMock();

    constructor() {
        return PrismaClientMock.instance;
    }

    static injectData(data: { [key in string]: any[] }) {
        globalDataStore = data;

        Object.keys(data).forEach((modelType) => {
            const modelName = snakeCase(modelType);

            PrismaClientMock.instance[modelName] = {
                findFirst: jest.fn((args) => mockFindFirst(modelType, args)),
                findUnique: jest.fn((args) => mockFindUnique(modelType, args)),
                findMany: jest.fn((args) => mockFindMany(modelType, args)),
                create: jest.fn((args) => mockCreate(modelType, args)),
                update: jest.fn((args) => mockUpdate(modelType, args)),
                upsert: jest.fn((args) => mockUpsert(modelType, args)),
                // Add other methods here as needed
            };
        });
    }

    static clearData() {
        globalDataStore = {};
    }

    // Define $connect, $disconnect, and other methods as class properties
    $connect = jest.fn().mockResolvedValue(undefined);
    $disconnect = jest.fn().mockResolvedValue(undefined);
}

export default {
    PrismaClient: PrismaClientMock,
};
