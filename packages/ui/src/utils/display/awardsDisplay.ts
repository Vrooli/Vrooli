import { Award, awardNames } from "@shared/consts"
import { AwardDisplay } from "types"

/**
 * Converts a queried award object into an AwardDisplay object.
 * @param award The queried award object.
 * @param t The translation function.
 * @returns The AwardDisplay object.
 */
export const awardToDisplay = (award: Award, t: any): AwardDisplay => {
    // Find earned tier
    const earnedTierDisplay = awardNames[award.category](award.progress);
    let earnedTier: AwardDisplay['earnedTier'] | undefined = undefined;
    if (earnedTierDisplay.name && earnedTierDisplay.body) {
        earnedTier = {
            title: t(`${earnedTierDisplay.name}`, { ns: 'award', ...earnedTierDisplay.nameVariables }),
            description: t(`${earnedTierDisplay.body}`, { ns: 'award', ...earnedTierDisplay.bodyVariables }),
            level: earnedTierDisplay.level,
        }
    }
    // Find next tier
    const nextTierDisplay = awardNames[award.category](award.progress, true);
    let nextTier: AwardDisplay['nextTier'] | undefined = undefined;
    if (nextTierDisplay.name && nextTierDisplay.body) {
        nextTier = {
            title: t(`${nextTierDisplay.name}`, { ns: 'award', ...nextTierDisplay.nameVariables }),
            description: t(`${nextTierDisplay.body}`, { ns: 'award', ...nextTierDisplay.bodyVariables }),
            level: nextTierDisplay.level,
        }
    }
    return {
        category: award.category,
        categoryDescription: t(`${award.category}Title`, { ns: 'award' }),
        categoryTitle: t(`${award.category}Body`, { ns: 'award' }),
        earnedTier,
        nextTier,
        progress: award.progress,
    }
}