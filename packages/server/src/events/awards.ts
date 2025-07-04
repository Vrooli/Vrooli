/**
 * Given a trigger and the trigger's data, determines if the user should receive an
 * award
 */

import { AwardCategory, awardNames, awardVariants, DEFAULT_LANGUAGE, type ModelType } from "@vrooli/shared";
import i18next from "i18next";
import { DbProvider } from "../db/provider.js";
import { Notify } from "../notify/notify.js";

/**
 * Given an ordered list of numbers, returns the closest lower number in the list
 * @param num The number to find the closest lower number for
 * @param list The list of numbers to search, from lowest to highest
 * @returns The closest lower number in the list, or null if there is none
 */
function closestLower(num: number, list: number[]): number | null {
    for (let i = 0; i < list.length; i++) {
        if (list[i] > num)
            return list[i - 1] || null;
    }
    return null;
}

/**
 * Checks if a user should receive an award
 * @param awardCategory The award category
 * @param previousCount The previous count of the award category
 * @param currentCount The current count of the award category
 * @returns True if the user should receive the award
 */
function shouldAward(awardCategory: `${AwardCategory}`, previousCount: number, currentCount: number): boolean {
    // Anniversary and new accounts are special cases
    if (awardCategory === "AccountAnniversary") return currentCount > previousCount;
    if (awardCategory === "AccountNew") return false;
    // Get tiers
    const tiers = awardVariants[awardCategory];
    if (!tiers) return false;
    // Check which tier previous count and new count are in
    const previous = closestLower(previousCount, tiers);
    const current = closestLower(currentCount, tiers);
    // Only award if moving to a new tier
    return current !== null && current > (previous ?? 0);
}

/**
 * Checks if an object type is tracked by the award system
 * @param objectType The object type to check
 * @returns `${objectType}Create` if it's a tracked award category
 */
export function objectAwardCategory<T extends keyof typeof ModelType>(objectType: T): `${T}Create` | null {
    return `${objectType}Create` in AwardCategory ? `${objectType}Create` : null;
}

/**
 * Handles tracking awards for a user. If a new award is earned, a notification
 * can be sent to the user (push or email)
 */
export function Award(userId: string, languages: string[] | undefined): {
    update: (category: `${AwardCategory}`, newProgress: number) => Promise<{ progress: number; timeReward: Date; userId: string; category: AwardCategory; }>;
} {
    return {
        /**
         * Upserts an award into the database. If the award progress reaches a new goal,
         * the user is notified
         * @param category The category of the award
         * @param newProgress The new progress of the award
         * @param languages Preferred languages for the award name and body
         * @returns The award
         */
        update: async (category: `${AwardCategory}`, newProgress: number) => {
            // Upsert the award into the database, with progress incremented
            // by the new progress
            const award = await DbProvider.get().award.upsert({
                where: { award_userId_category_unique: { userId, category } },
                update: { progress: { increment: newProgress } },
                create: { userId, category, progress: newProgress },
            });
            // Check if user should receive a new award (i.e. the progress has put them
            // into a new award tier)
            const isNewTier = shouldAward(category, award.progress - newProgress, award.progress);
            if (isNewTier) {
                // Get translated award name and body
                const lng = languages && languages.length > 0 ? languages[0] : DEFAULT_LANGUAGE;
                const { name, nameVariables, body, bodyVariables } = awardNames[category](award.progress);
                const transTitle = name ? i18next.t(`award:${name}`, { lng, ...(nameVariables ?? {}) }) : null;
                const transBody = body ? i18next.t(`award:${body}`, { lng, ...(bodyVariables ?? {}) }) : null;
                // Send a notification to the user
                if (transTitle && transBody) {
                    await Notify(languages).pushAward(transTitle, transBody).toUser(userId);
                }
                // Set "tierCompletedAt" to the current time
                await DbProvider.get().award.update({
                    where: { award_userId_category_unique: { userId, category } },
                    data: { tierCompletedAt: new Date() },
                });
            }
            return award;
        },
    };
}

// AI_CHECK: TASK_ID=fix-compound-unique-key-awards | LAST: 2025-01-29

