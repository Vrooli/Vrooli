import { ShapeModel } from "types";

/**
 * Shapes versioning relation data for a GraphQL create input
 * @param root The root object, which contains the version data
 * @param shape The version's shape object
 * @returns The shaped object, ready to be passed to the mutation endpoint
 */
export const createVersion = <
    Root extends { id: string, versions?: Record<string, unknown>[] | null | undefined },
    VersionCreateInput extends object,
>(
    root: Root,
    shape: ShapeModel<any, VersionCreateInput, null>,
): ({ versionsCreate?: VersionCreateInput[] }) => {
    // Return empty object if no version data
    if (!root.versions) return {};
    // Shape and return version data, injecting the root ID
    return {
        versionsCreate: root.versions.map((version) => shape.create({
            ...version,
            root: { id: root.id },
        })) as VersionCreateInput[],
    };
};
