import clsx from "clsx";
import * as React from "react";
import { parseNativeEmoji } from "../../dataUtils/parseNativeEmoji";
import { ClassNames } from "../../DomUtils/classNames";

export function NativeEmoji({
    unified,
    style,
}: {
    unified: string;
    style: React.CSSProperties;
}) {
    return (
        <span
            className={clsx(ClassNames.external, "epr-emoji-native")}
            data-unified={unified}
            style={style}
        >
            {parseNativeEmoji(unified)}
        </span>
    );
}
