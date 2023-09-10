import { exists } from "./exists";

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
    return typeof reaction === "string" ? reaction.replace(/[\u{1F3FB}-\u{1F3FF}\u{200D}\u{FE0F}\u{20E3}]/gu, "") : "";
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
