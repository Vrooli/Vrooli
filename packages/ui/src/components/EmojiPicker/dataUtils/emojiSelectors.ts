import { EmojisKey } from "@local/shared";
import i18next from "i18next";
import emojis from "../data/emojis";
import { SkinTones } from "../EmojiPicker";
import { DataEmoji, EmojiProperties, WithName } from "./DataTypes";

export function emojiNames(emoji: WithName): string[] {
    return emoji[EmojiProperties.name] ?? [];
}

export function emojiName(emoji?: WithName): string {
    if (!emoji) {
        return "";
    }

    return emojiNames(emoji)[0];
}

export function unifiedWithoutSkinTone(unified: string): string {
    const splat = unified.split("-");
    const [skinTone] = splat.splice(1, 1);

    if (SkinTones[skinTone]) {
        return splat.join("-");
    }

    return unified;
}

export function emojiUnified(emoji: DataEmoji, skinTone?: string): string {
    const unified = emoji[EmojiProperties.unified];

    if (!skinTone || !Array.isArray(emoji[EmojiProperties.variations]) || emoji[EmojiProperties.variations].length === 0) {
        return unified;
    }

    return emojiVariationUnified(emoji, skinTone) ?? unified;
}

export function emojiVariationUnified(
    emoji: DataEmoji,
    skinTone?: string,
): string | undefined {
    return skinTone
        ? (emoji[EmojiProperties.variations] ?? []).find(variation => variation.includes(skinTone))
        : emojiUnified(emoji);
}

export function emojiByUnified(unified?: string): DataEmoji | undefined {
    if (!unified) {
        return;
    }

    if (allEmojisByUnified[unified]) {
        return allEmojisByUnified[unified];
    }

    const withoutSkinTone = unifiedWithoutSkinTone(unified);
    return allEmojisByUnified[withoutSkinTone];
}

export const allEmojis = Object.values(emojis).map(category => category.map(emoji => ({
    ...emoji,
    name: (i18next.t(`emojis:${emoji.u.toLowerCase() as unknown as EmojisKey}`, { ns: "emojis" }) ?? "").split(", "),
}))).flat() as DataEmoji[];
console.log("got all emojis", allEmojis, i18next.t("common:Submit"), i18next.t("emojis:1f606", { ns: "emojis" }));

const allEmojisByUnified: {
    [unified: string]: DataEmoji;
} = {};
