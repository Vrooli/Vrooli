import { parseNativeEmoji } from "components/EmojiPicker/dataUtils/parseNativeEmoji";
import * as React from "react";
import { emojiByUnified } from "../../dataUtils/emojiSelectors";
import { BaseEmojiProps } from "./BaseEmojiProps";

export function ViewOnlyEmoji({
    emoji,
    unified,
    size,
}: BaseEmojiProps) {
    const style = {} as React.CSSProperties;
    if (size) {
        style.width = style.height = style.fontSize = `${size}px`;
    }

    const emojiToRender = emoji ? emoji : emojiByUnified(unified);

    if (!emojiToRender) {
        return null;
    }

    return (
        <span data-unified={unified}>
            {parseNativeEmoji(unified)}
        </span>
    );
}
