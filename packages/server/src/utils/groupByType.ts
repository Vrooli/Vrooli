import { GqlModelType } from "@shared/consts";

type GroupByTypeProps = {
    createMany: { [x: string]: any }[] | null | undefined,
    deleteMany: string[] | null | undefined,
    updateMany: { [x: string]: any }[] | null | undefined,
}

/**
 * Groups create/update/delete mutations inputs by type
 * @param createMany Create inputs
 * @param updateMany Update inputs
 * @param deleteMany Delete inputs
 * @returns Two objects. One with inputs grouped by type, and one with ids grouped by type
 */
export const groupByType = ({
    createMany,
    deleteMany,
    updateMany, 
}: GroupByTypeProps) => {
    // Initialize the return objects
    const inputsByType: { [key in GqlModelType]?: { [x: string]: any }[] } = {};
    const idsByType: { [key in GqlModelType]?: string[] } = {};
    // For every create input, add it to the return object
    for (const input of createMany ?? []) {
        const { __typename } = input;
        inputsByType[__typename] = inputsByType[__typename] ?? [];
        inputsByType[__typename].push(input);
    }
}