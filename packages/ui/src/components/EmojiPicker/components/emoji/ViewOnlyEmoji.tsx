import * as React from "react";
import { emojiByUnified } from "../../dataUtils/emojiSelectors";
import { isCustomEmoji } from "../../typeRefinements/typeRefinements";
import { useEmojisThatFailedToLoadState } from "../context/PickerContext";
import { BaseEmojiProps } from "./BaseEmojiProps";
import { EmojiImg } from "./EmojiImg";
import { NativeEmoji } from "./NativeEmoji";

export function ViewOnlyEmoji({
    emoji,
    unified,
    size,
    lazyLoad,
}: BaseEmojiProps) {
    const [, setEmojisThatFailedToLoad] = useEmojisThatFailedToLoadState();

    const style = {} as React.CSSProperties;
    if (size) {
        style.width = style.height = style.fontSize = `${size}px`;
    }

    const emojiToRender = emoji ? emoji : emojiByUnified(unified);

    if (!emojiToRender) {
        return null;
    }

    if (isCustomEmoji(emojiToRender)) {
        return (
            <EmojiImg
                style={style}
                emojiName={unified}
                lazyLoad={lazyLoad}
                imgUrl={emojiToRender.imgUrl}
                onError={onError}
            />
        );
    }

    return (
        <NativeEmoji unified={unified} style={style} />
    );

    function onError() {
        setEmojisThatFailedToLoad(prev => new Set(prev).add(unified));
    }
}
