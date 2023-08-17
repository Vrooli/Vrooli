import { ShapeModel } from "types";

/**
 * Shapes versioning relation data for a GraphQL update input
 * @param root The root object, which contains the version data
 * @param shape The version's shape object
 * @returns The shaped object, ready to be passed to the mutation endpoint
 */
export const updateVersion = <
    Root extends { id: string, versions?: Record<string, unknown>[] | null | undefined },
    VersionCreateInput extends object,
    VersionUpdateInput extends object,
>(
    originalRoot: Root,
    updatedRoot: Root,
    shape: ShapeModel<any, VersionCreateInput, VersionUpdateInput>,
): ({ versionsCreate?: VersionCreateInput[], versionsUpdate?: VersionUpdateInput[] }) => {
    // Return empty object if no updated version data. We don't handle deletes here
    if (!updatedRoot.versions) return {};
    // Find every version in the updated root that is not in the original root (using the version's id)
    const newVersions = updatedRoot.versions.filter((v) => !originalRoot.versions?.find((ov) => ov.id === v.id));
    // Every other version in the updated root is an update
    const updatedVersions = updatedRoot.versions.filter((v) => !newVersions.find((nv) => nv.id === v.id));
    // Shape and return version data
    return {
        versionsCreate: newVersions.map((version) => shape.create({
            ...version,
            root: { id: originalRoot.id },
        })) as VersionCreateInput[],
        versionsUpdate: updatedVersions.map((version) => shape.update(originalRoot.versions?.find((ov) => ov.id === version.id), {
            ...version,
            root: { id: originalRoot.id },
        })) as VersionUpdateInput[],
    };
};
