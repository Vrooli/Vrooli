import { EmojisKey } from "@local/shared";
import i18next from "i18next";
import { Categories } from "../config/categoryConfig";
import { CustomEmoji } from "../config/customEmojiConfig";
import emojis from "../data/emojis";
import skinToneVariations, { skinTonesMapped } from "../data/skinToneVariations";
import { SkinTones } from "../types";
import { indexEmoji } from "./alphaNumericEmojiIndex";
import { DataEmoji, DataEmojis, EmojiProperties, WithName } from "./DataTypes";

export function addedIn(emoji: DataEmoji): number {
    return parseFloat(emoji[EmojiProperties.added_in]);
}

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

    if (skinTonesMapped[skinTone]) {
        return splat.join("-");
    }

    return unified;
}

export function emojiUnified(emoji: DataEmoji, skinTone?: string): string {
    const unified = emoji[EmojiProperties.unified];

    if (!skinTone || !emojiHasVariations(emoji)) {
        return unified;
    }

    return emojiVariationUnified(emoji, skinTone) ?? unified;
}

export function emojisByCategory(category: Categories): DataEmojis {
    return emojis?.[category] ?? [];
}

export function emojiVariations(emoji: DataEmoji): string[] {
    return emoji[EmojiProperties.variations] ?? [];
}

export function emojiHasVariations(emoji: DataEmoji): boolean {
    return emojiVariations(emoji).length > 0;
}

export function emojiVariationUnified(
    emoji: DataEmoji,
    skinTone?: string,
): string | undefined {
    return skinTone
        ? emojiVariations(emoji).find(variation => variation.includes(skinTone))
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
console.log("got all emojis", allEmojis, i18next.t("Submit"), i18next.t("emojis:1f606", { ns: "emojis" }));

export function addCustomEmojis(customEmojis: CustomEmoji[]): void {
    customEmojis.forEach(emoji => {
        const emojiData = customToRegularEmoji(emoji);

        if (allEmojisByUnified[emojiData[EmojiProperties.unified]]) {
            return;
        }

        allEmojis.push(emojiData);
        allEmojisByUnified[emojiData[EmojiProperties.unified]] = emojiData;
        emojis[Categories.CUSTOM].push(emojiData as never);
        indexEmoji(emojiData);
    });
}

function customToRegularEmoji(emoji: CustomEmoji): DataEmoji {
    return {
        name: i18next.t(emoji.id.toLowerCase() as unknown as EmojisKey, { ns: "emojis" }).split(", "),
        u: emoji.id.toLowerCase(),
        a: "0",
        imgUrl: emoji.imgUrl,
    };
}

const allEmojisByUnified: {
    [unified: string]: DataEmoji;
} = {};

setTimeout(() => {
    allEmojis.reduce((allEmojis, Emoji) => {
        allEmojis[emojiUnified(Emoji)] = Emoji;
        return allEmojis;
    }, allEmojisByUnified);
});

export function activeVariationFromUnified(unified: string): SkinTones | null {
    const [, suspectedSkinTone] = unified.split("-") as [string, SkinTones];
    return skinToneVariations.includes(suspectedSkinTone)
        ? suspectedSkinTone
        : null;
}
