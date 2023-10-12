import { DotNotation } from "@local/shared";
import { Displayer, ModelLogic, ModelLogicType } from "../models/types";

/**
 * Finds the most appropriate version to display for a root object.
 * Order of preference:
 * 1. Version marked isLatest
 * 2. Public version with highest version index
 * 3. Private version with highest version index
 * @param versions 
 * @returns 
 */
const findBestVersion = (versions: any[]): any | undefined => {
    if (versions.length === 0) return undefined;
    // 1. Version marked isLatest
    let latest = versions.find(v => v.isLatest);
    if (latest) return latest;
    // 2. Public version with highest version index
    // Sort by version index, descending
    versions.sort((a, b) => b.versionIndex - a.versionIndex);
    latest = versions.find(v => !v.isPrivate);
    if (latest) return latest;
    // 3. Private version with highest version index
    latest = versions[0];
    return latest;
};

/**
 * All root objects share the same logic to display their label for push notifications.
 * This is that logic.
 * @param versionModelLogic The logic for the version model associated with the root object.
 * @returns Displayer object
 */
export const rootObjectDisplay = <
    RootModel extends ModelLogicType,
    VersionModel extends ModelLogicType,
    VersionSuppFields extends readonly DotNotation<VersionModel["GqlModel"]>[]
>(
    versionModelLogic: ModelLogic<VersionModel, VersionSuppFields>,
): ReturnType<Displayer<RootModel>> => ({
    label: {
        select: () => ({
            versions: {
                select: {
                    ...versionModelLogic.display().label.select(),
                    isLatest: true,
                    isPrivate: true,
                    versionIndex: true,
                },
            },
        }),
        get: (select, languages) => {
            const version = findBestVersion(select.versions);
            if (!version) return "";
            return versionModelLogic.display().label.get(version, languages);
        },
    },
});
