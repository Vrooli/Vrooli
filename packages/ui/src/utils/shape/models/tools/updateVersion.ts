import { ShapeModel } from 'types';

/**
 * Shapes versioning relation data for a GraphQL update input
 * @param root The root object, which contains the version data
 * @param shape The version's shape object
 * @param preShape A function to convert the version data before passing it to the shape function
 * @returns The shaped object, ready to be passed to the mutation endpoint
 */
export const updateVersion = <
    Root extends { versionInfo?: { id: string, versionLabel: string | null | undefined } | null | undefined },
    VersionCreateInput extends {},
    VersionUpdateInput extends {},
>(
    originalRoot: Root,
    updatedRoot: Root,
    shape: ShapeModel<any, VersionCreateInput, VersionUpdateInput>,
    preShape?: (version: Record<string, any>, root: Record<string, any>) => any,
): ({ versionsCreate?: [VersionCreateInput], versionsUpdate?: [VersionUpdateInput] }) => {
    // Return empty object if no updated version data. We don't handle deletes here
    if (!updatedRoot.versionInfo) return {} as any;
    // Make preShape a function, if not provided 
    const preShaper = preShape ?? ((x: any) => x);
    // Determine if we're creating or updating
    const isCreate = !originalRoot.versionInfo || originalRoot.versionInfo.id !== updatedRoot.versionInfo.id;
    // Shape and return version data
    if (isCreate) {
        return {
            versionsCreate: [shape.create(preShaper({
                ...updatedRoot.versionInfo,
                versionLabel: updatedRoot.versionInfo.versionLabel ?? '0.0.1',
            }, updatedRoot))]
        } as any;
    } else {
        return {
            versionsUpdate: [shape.update(
                preShaper(originalRoot.versionInfo!, originalRoot),
                preShaper(updatedRoot.versionInfo, updatedRoot)
            )]
        } as any;
    }
};