import { ViewOnlyEmoji } from "./ViewOnlyEmoji";

export function ExportedEmoji({
    unified,
    size = 32,
    lazyLoad = false,
}: {
    unified: string;
    size?: number;
    lazyLoad?: boolean;
}) {
    if (!unified) {
        return null;
    }

    return (
        <ViewOnlyEmoji
            unified={unified}
            size={size}
            lazyLoad={lazyLoad}
        />
    );
}
