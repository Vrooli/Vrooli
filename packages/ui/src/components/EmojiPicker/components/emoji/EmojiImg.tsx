import clsx from "clsx";
import * as React from "react";
import { ClassNames } from "../../DomUtils/classNames";

export function EmojiImg({
    emojiName,
    style,
    lazyLoad = false,
    imgUrl,
    onError,
}: {
    emojiName: string;
    style: React.CSSProperties;
    lazyLoad?: boolean;
    imgUrl: string;
    onError: () => void;
}) {
    return (
        <img
            src={imgUrl}
            alt={emojiName}
            className={clsx(ClassNames.external, "epr-emoji-img")}
            loading={lazyLoad ? "lazy" : "eager"}
            onError={onError}
            style={style}
        />
    );
}
