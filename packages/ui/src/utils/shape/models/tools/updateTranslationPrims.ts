import { updatePrims } from "./updatePrims";

/**
 * Like updatePrims, but forces "language" field to be included in the return type
 * @param original The original object
 * @param updated The updated object
 * @param primary The primary key of the object, which is always returned
 * @param fields The fields to check, which are returned if they are different
 */
export const updateTranslationPrims = <T extends { language: string }, K extends keyof T, PK extends keyof T>(
    original: T | null | undefined,
    updated: T | null | undefined,
    primary: PK | null,
    ...fields: K[]
): ({ [F in K]: Exclude<T[F], null | undefined> } & { [F in PK]: T[F] } & { language: string }) => {
    const updatePrimsResult = updatePrims(original, updated, primary, ...fields);
    return { ...updatePrimsResult, language: original?.language ?? updated?.language } as any;
};
