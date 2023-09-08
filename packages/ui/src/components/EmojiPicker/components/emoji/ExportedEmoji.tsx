import { ViewOnlyEmoji } from "./ViewOnlyEmoji";

export function ExportedEmoji({
    unified,
    size = 32,
}: {
    unified: string;
    size?: number;
}) {
    if (!unified) {
        return null;
    }

    return (
        <ViewOnlyEmoji
            unified={unified}
            size={size}
        />
    );
}
