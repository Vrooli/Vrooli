import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { ReactionFor } from "@local/consts";
import { Stack } from "@mui/material";
import { useMemo } from "react";
import { getCounts, getYou } from "../../../utils/display/listTools";
import { ReportsLink } from "../../buttons/ReportsLink/ReportsLink";
import { VoteButton } from "../../buttons/VoteButton/VoteButton";
import { ViewsDisplay } from "../ViewsDisplay/ViewsDisplay";
export const StatsCompact = ({ handleObjectUpdate, loading, object, }) => {
    const you = useMemo(() => getYou(object), [object]);
    const counts = useMemo(() => getCounts(object), [object]);
    return (_jsxs(Stack, { direction: "row", sx: {
            marginTop: 1,
            marginBottom: 1,
            display: "flex",
            alignItems: "center",
            "& > *:not(:first-child)": {
                marginLeft: 1,
            },
            "& > *:nth-child(3)": {
                marginLeft: 2,
            },
        }, children: [object && object.__typename in ReactionFor && _jsx(VoteButton, { direction: "row", disabled: !you.canReact, emoji: you.reaction, objectId: object.id, voteFor: object.__typename, score: counts.score, onChange: (newEmoji, newScore) => {
                    object && handleObjectUpdate({
                        ...object,
                        reaction: newEmoji,
                        score: newScore,
                    });
                } }), _jsx(ViewsDisplay, { views: object?.views }), object?.id && _jsx(ReportsLink, { object: object })] }));
};
//# sourceMappingURL=StatsCompact.js.map