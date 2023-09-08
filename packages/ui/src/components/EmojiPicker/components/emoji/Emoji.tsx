import { DataEmoji } from "../../dataUtils/DataTypes";
import { emojiHasVariations, emojiNames } from "../../dataUtils/emojiSelectors";
import { BaseEmojiProps } from "./BaseEmojiProps";
import { ClickableEmojiButton } from "./ClickableEmojiButton";
import "./Emoji.css";
import { ViewOnlyEmoji } from "./ViewOnlyEmoji";

type ClickableEmojiProps = Readonly<
    BaseEmojiProps & {
        hidden?: boolean;
        showVariations?: boolean;
        hiddenOnSearch?: boolean;
        emoji: DataEmoji;
    }
>;

export function ClickableEmoji({
    emoji,
    unified,
    hidden,
    hiddenOnSearch,
    showVariations = true,
    size,
    lazyLoad,
}: ClickableEmojiProps) {
    const hasVariations = emojiHasVariations(emoji);

    return (
        <ClickableEmojiButton
            hasVariations={hasVariations}
            showVariations={showVariations}
            hidden={hidden}
            hiddenOnSearch={hiddenOnSearch}
            emojiNames={emojiNames(emoji)}
            unified={unified}
        >
            <ViewOnlyEmoji
                unified={unified}
                emoji={emoji}
                size={size}
                lazyLoad={lazyLoad}
            />
        </ClickableEmojiButton>
    );
}
