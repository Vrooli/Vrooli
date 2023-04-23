import { exists } from "./exists";
const PositiveReactions = ["👍", "👏", "🎉", "🥳", "😊", "😃", "😄", "😁", "😇", "❤️", "🥰", "💖", "😍", "🚀", "👀", "🔥", "🎊", "🙌", "👌", "👊", "💯", "🤘", "🤙", "🤟", "🤝"];
const NegativeReactions = ["👎", "😕", "😡", "😠", "🤬", "😞", "😟", "😨", "🤮", "🤢", "🤧", "🤒", "🤕", "🤡", "🤥", "🤦", "🙅‍♂️"];
export function removeModifiers(reaction) {
    return reaction.replace(/[\u{1F3FB}-\u{1F3FF}\u{200D}\u{FE0F}\u{20E3}]/gu, "");
}
export function getReactionScore(reaction) {
    if (!exists(reaction))
        return 0;
    const baseReaction = removeModifiers(reaction);
    if (PositiveReactions.includes(baseReaction)) {
        return 1;
    }
    else if (NegativeReactions.includes(baseReaction)) {
        return -1;
    }
    else {
        return 0;
    }
}
//# sourceMappingURL=emotions.js.map