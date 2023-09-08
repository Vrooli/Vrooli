import { CustomEmoji } from "../../config/customEmojiConfig";
import { DataEmoji } from "../../dataUtils/DataTypes";

export type BaseEmojiProps = {
    emoji?: DataEmoji | CustomEmoji;
    unified: string;
    size?: number;
    lazyLoad?: boolean;
};
