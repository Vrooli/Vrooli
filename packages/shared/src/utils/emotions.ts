import { exists } from "./exists.js";

// Reactions that increase the score of the object
const PositiveReactions = ["👍", "👏", "🎉", "🥳", "😊", "😃", "😄", "😁", "😇", "❤️", "🥰", "💖", "😍", "🚀", "👀", "🔥", "🎊", "🙌", "👌", "👊", "💯", "🤘", "🤙", "🤟", "🤝"];
// Reactions that decrease the score of the object
const NegativeReactions = ["👎", "😕", "😡", "😠", "🤬", "😞", "😟", "😨", "🤮", "🤢", "🤧", "🤒", "🤕", "🤡", "🤥", "🤦", "🙅‍♂️"];

/**
 * Removes skin tone and other modifiers from an emoji reaction. 
 * Example: '👍🏻' -> '👍'
 * @string reaction The emoji reaction to remove modifiers from
 */
export function removeModifiers(reaction: string): string {
    // Only remove skin tone modifiers (1F3FB-1F3FF), not variation selectors (FE0F) or ZWJ (200D)
    return typeof reaction === "string" ? reaction.replace(/[\u{1F3FB}-\u{1F3FF}]/gu, "") : "";
}

/**
 * Finds the score (+1, 0, -1) of a reaction. Ignores skin tone and other modifiers.
 * Example: '👍🏻' -> 1, '👎🏻' -> -1, '🐰' -> 0
 */
export function getReactionScore(reaction: string | null | undefined): number {
    if (!exists(reaction)) return 0;
    const baseReaction = removeModifiers(reaction);
    if (PositiveReactions.includes(baseReaction)) {
        return 1;
    } else if (NegativeReactions.includes(baseReaction)) {
        return -1;
    } else {
        return 0;
    }
}
