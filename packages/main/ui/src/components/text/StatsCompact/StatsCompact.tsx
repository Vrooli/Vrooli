import { ReactionFor } from "@local/shared;";
import { Stack } from "@mui/material";
import { useMemo } from "react";
import { getCounts, getYou } from "../../../utils/display/listTools";
import { ReportsLink } from "../../buttons/ReportsLink/ReportsLink";
import { VoteButton } from "../../buttons/VoteButton/VoteButton";
import { ViewsDisplay } from "../ViewsDisplay/ViewsDisplay";
import { StatsCompactProps, StatsCompactPropsObject } from "../types";

/**
 * Displays basic stats about an object, in a short format.
 * Displays votes, views, date created, and reports
 */
export const StatsCompact = <T extends StatsCompactPropsObject>({
    handleObjectUpdate,
    loading,
    object,
}: StatsCompactProps<T>) => {
    const you = useMemo(() => getYou(object as any), [object]);
    const counts = useMemo(() => getCounts(object as any), [object]);

    return (
        <Stack
            direction="row"
            sx={{
                marginTop: 1,
                marginBottom: 1,
                display: "flex",
                alignItems: "center",
                // Space items out, with first item having less space than the rest
                "& > *:not(:first-child)": {
                    marginLeft: 1,
                },
                "& > *:nth-child(3)": {
                    marginLeft: 2,
                },
            }}
        >
            {/* Votes */}
            {object && object.__typename in ReactionFor && <VoteButton
                direction="row"
                disabled={!you.canReact}
                emoji={you.reaction}
                objectId={object.id}
                voteFor={object.__typename as ReactionFor}
                score={counts.score}
                onChange={(newEmoji, newScore) => {
                    object && handleObjectUpdate({
                        ...object,
                        reaction: newEmoji, //TODO
                        score: newScore, //TODO
                    });
                }}
            />}
            {/* Views */}
            <ViewsDisplay views={(object as any)?.views} />
            {/* Reports */}
            {object?.id && <ReportsLink object={object as any} />}
        </Stack>
    );
};
