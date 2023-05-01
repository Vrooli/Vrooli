import { DotNotation } from "@local/shared";
import { Displayer, ModelLogic, ModelLogicType } from "../models/types";

/**
 * All root objects share the same logic to display their label for push notifications.
 * This is that logic.
 * @param versionModelLogic The logic for the version model associated with the root object.
 * @returns Displayer object
 */
export const rootObjectDisplay = <
    RootModel extends ModelLogicType,
    VersionModel extends ModelLogicType,
    VersionSuppFields extends readonly DotNotation<VersionModel['GqlModel']>[]
>(
    versionModelLogic: ModelLogic<VersionModel, VersionSuppFields>
): Displayer<RootModel> => ({
    select: () => ({
        id: true,
        versions: {
            orderBy: { versionIndex: 'desc' },
            take: 10,
            select: {
                ...versionModelLogic.display.select(),
                isLatest: true,
                isPrivate: true,
                versionIndex: true,
            }
        }
    }),
    label: (select, languages) => {
        if (select.versions.length === 0) return '';
        // Find most appropriate version to display. Order of preference: 
        // 1. Version marked isLatest
        // 2. Public version with highest versionIndex
        // 3. Private version with highest versionIndex
        let latest = select.versions.find(v => v.isLatest);
        if (!latest) {
            latest = select.versions.find(v => !v.isPrivate);
            if (!latest) latest = select.versions[0];
        }
        return versionModelLogic.display.label(latest as any, languages);
    }
})