import emojis from "../data/emojis";

export enum EmojiProperties {
    name = "name",
    unified = "u",
    variations = "v",
    added_in = "a",
    imgUrl = "imgUrl"
}

export interface DataEmoji extends WithName {
    [EmojiProperties.unified]: string;
    [EmojiProperties.variations]?: string[];
    [EmojiProperties.added_in]: string;
    [EmojiProperties.imgUrl]?: string;
}

export type DataEmojis = DataEmoji[];

export type DataGroups = keyof typeof emojis;

export type WithName = {
    name: string[];
};
