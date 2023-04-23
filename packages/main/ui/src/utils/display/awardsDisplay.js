import { awardNames } from "@local/consts";
export const awardToDisplay = (award, t) => {
    const earnedTierDisplay = awardNames[award.category](award.progress, false);
    let earnedTier = undefined;
    if (earnedTierDisplay.name && earnedTierDisplay.body) {
        earnedTier = {
            title: t(`${earnedTierDisplay.name}`, { ns: "award", ...earnedTierDisplay.nameVariables }),
            description: t(`${earnedTierDisplay.body}`, { ns: "award", ...earnedTierDisplay.bodyVariables }),
            level: earnedTierDisplay.level,
        };
    }
    const nextTierDisplay = awardNames[award.category](award.progress, true);
    let nextTier = undefined;
    if (nextTierDisplay.name && nextTierDisplay.body) {
        nextTier = {
            title: t(`${nextTierDisplay.name}`, { ns: "award", ...nextTierDisplay.nameVariables }),
            description: t(`${nextTierDisplay.body}`, { ns: "award", ...nextTierDisplay.bodyVariables }),
            level: nextTierDisplay.level,
        };
    }
    return {
        category: award.category,
        categoryDescription: t(`${award.category}Title`, { ns: "award" }),
        categoryTitle: t(`${award.category}Body`, { ns: "award" }),
        earnedTier,
        nextTier,
        progress: award.progress,
    };
};
//# sourceMappingURL=awardsDisplay.js.map