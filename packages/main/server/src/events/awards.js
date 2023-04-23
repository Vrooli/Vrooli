import { AwardCategory, awardNames, awardVariants } from "@local/consts";
import i18next from "i18next";
import { Notify } from "../notify";
function closestLower(num, list) {
    for (let i = 0; i < list.length; i++) {
        if (list[i] > num)
            return list[i - 1] || null;
    }
    return null;
}
const shouldAward = (awardCategory, previousCount, currentCount) => {
    if (awardCategory === "AccountAnniversary")
        return currentCount > previousCount;
    if (awardCategory === "AccountNew")
        return false;
    const tiers = awardVariants[awardCategory];
    if (!tiers)
        return false;
    const previous = closestLower(previousCount, tiers);
    const current = closestLower(currentCount, tiers);
    return current !== null && current > (previous ?? 0);
};
export const objectAwardCategory = (objectType) => {
    return `${objectType}Create` in AwardCategory ? `${objectType}Create` : null;
};
export const Award = (prisma, userId, languages) => ({
    update: async (category, newProgress) => {
        const award = await prisma.award.upsert({
            where: { userId_category: { userId, category } },
            update: { progress: { increment: newProgress } },
            create: { userId, category, progress: newProgress },
        });
        const isNewTier = shouldAward(category, award.progress - newProgress, award.progress);
        if (isNewTier) {
            const lng = languages.length > 0 ? languages[0] : "en";
            const { name, nameVariables, body, bodyVariables } = awardNames[category](award.progress);
            const transTitle = name ? i18next.t(`award:${name}`, { lng, ...(nameVariables ?? {}) }) : null;
            const transBody = body ? i18next.t(`award:${body}`, { lng, ...(bodyVariables ?? {}) }) : null;
            if (transTitle && transBody) {
                await Notify(prisma, languages).pushAward(transTitle, transBody).toUser(userId);
            }
            await prisma.award.update({
                where: { userId_category: { userId, category } },
                data: { timeCurrentTierCompleted: new Date() },
            });
        }
        return award;
    },
});
//# sourceMappingURL=awards.js.map