import { ShapeModel } from 'types';

/**
 * Shapes versioning relation data for a GraphQL create input
 * @param root The root object, which contains the version data
 * @param shape The version's shape object
 * @param preShape A function to convert the version data before passing it to the shape function
 * @returns The shaped object, ready to be passed to the mutation endpoint
 */
export const createVersion = <
    Root extends { versionInfo?: { versionLabel: string | null | undefined } | null | undefined },
    VersionCreateInput extends {},
>(
    root: Root,
    shape: ShapeModel<any, VersionCreateInput, null>,
    preShape?: (version: Record<string, any>) => any,
): ({ versionsCreate?: [VersionCreateInput] }) => {
    // Return empty object if no version data
    if (!root.versionInfo) return {} as any;
    // Make preShape a function, if not provided 
    const preShaper = preShape ?? ((x: any) => x);
    // Shape and return version data
    return {
        versionsCreate: [shape.create(preShaper({
            ...root.versionInfo,
            versionLabel: root.versionInfo.versionLabel ?? '0.0.1',
        }))]
    } as any;
};