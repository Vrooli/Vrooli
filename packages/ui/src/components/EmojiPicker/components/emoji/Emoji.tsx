import { Button } from "@mui/material";
import { DataEmoji } from "../../dataUtils/DataTypes";
import { emojiNames } from "../../dataUtils/emojiSelectors";
import { BaseEmojiProps } from "./BaseEmojiProps";
import { ViewOnlyEmoji } from "./ViewOnlyEmoji";

type ClickableEmojiProps = Readonly<
    BaseEmojiProps & {
        hidden?: boolean;
        hiddenOnSearch?: boolean;
        emoji: DataEmoji;
    }
>;

export function ClickableEmoji({
    emoji,
    unified,
    hiddenOnSearch,
    size,
}: ClickableEmojiProps) {
    return (
        <Button
            data-unified={unified}
            aria-label={emojiNames[0]}
            data-full-name={emojiNames}
        >
            <ViewOnlyEmoji
                unified={unified}
                emoji={emoji}
                size={size}
            />
        </Button>
    );
}
