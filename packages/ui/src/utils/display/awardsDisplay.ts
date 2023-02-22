import { Award, awardNames } from "@shared/consts"
import { AwardDisplay } from "types"

/**
 * Converts a queried award object into an AwardDisplay object.
 * @param award The queried award object.
 * @param t The translation function.
 * @param lng The language to use for translations.
 * @returns The AwardDisplay object.
 */
export const awardToDisplay = (award: Award, t: any, lng: string): AwardDisplay => {
    // Find earned tier
    const earnedTierDisplay = awardNames[award.category](award.progress);
    let earnedTier: AwardDisplay['earnedTier'] | undefined = undefined;
    if (earnedTierDisplay.name && earnedTierDisplay.body) {
        earnedTier = {
            title: t(`award:${earnedTierDisplay.name}`, { lng, ...earnedTierDisplay.nameVariables }),
            description: t(`award:${earnedTierDisplay.body}`, { lng, ...earnedTierDisplay.bodyVariables }),
            level: earnedTierDisplay.level,
        }
    }
    // Find next tier
    const nextTierDisplay = awardNames[award.category](award.progress, true);
    let nextTier: AwardDisplay['nextTier'] | undefined = undefined;
    if (nextTierDisplay.name && nextTierDisplay.body) {
        nextTier = {
            title: t(`award:${nextTierDisplay.name}`, { lng, ...nextTierDisplay.nameVariables }),
            description: t(`award:${nextTierDisplay.body}`, { lng, ...nextTierDisplay.bodyVariables }),
            level: nextTierDisplay.level,
        }
    }
    return {
        category: award.category,
        categoryDescription: t(`award:${award.category}.description`, { lng }),
        categoryTitle: t(`award:${award.category}.title`, { lng }),
        earnedTier,
        nextTier,
        progress: award.progress,
    }
}