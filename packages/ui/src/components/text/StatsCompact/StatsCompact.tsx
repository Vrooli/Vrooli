import { Stack } from "@mui/material"
import { VoteFor } from "@shared/consts";
import { ReportsLink } from "components/buttons";
import { UpvoteDownvote } from "components/lists";
import { useMemo } from "react";
import { getCounts, getYou } from "utils";
import { StatsCompactProps, StatsCompactPropsObject } from "../types";
import { ViewsDisplay } from "../ViewsDisplay/ViewsDisplay";

/**
 * Displays basic stats about an object, in a short format.
 * Displays votes, views, date created, and reports
 */
export const StatsCompact = <T extends StatsCompactPropsObject>({
    handleObjectUpdate,
    loading,
    object,
    session,
}: StatsCompactProps<T>) => {

    const you = useMemo(() => getYou(object as any), [object]);
    const counts = useMemo(() => getCounts(object as any), [object])

    return (
        <Stack
            direction="row"
            sx={{
                marginTop: 1,
                marginBottom: 1,
                display: 'flex',
                alignItems: 'center',
                // Space items out, with first item having less space than the rest
                '& > *:not(:first-child)': {
                    marginLeft: 1,
                },
                '& > *:nth-child(3)': {
                    marginLeft: 2,
                },
            }}
        >
            {/* Votes */}
            {object && object.__typename in VoteFor && <UpvoteDownvote
                direction="row"
                disabled={!you.canVote}
                session={session}
                objectId={object.id}
                voteFor={object.__typename as VoteFor}
                isUpvoted={you.isUpvoted}
                score={counts.score}
                onChange={(isUpvote, score) => { object && handleObjectUpdate({ 
                    ...object, 
                    isUpvoted: isUpvote, //TODO
                    score: score, //TODO
                }); }}
            />}
            {/* Views */}
            <ViewsDisplay views={(object as any)?.views} />
            {/* Reports */}
            {object?.id && <ReportsLink object={object as any} />}
        </Stack>
    )
}