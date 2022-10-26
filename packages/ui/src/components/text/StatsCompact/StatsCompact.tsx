import { Stack } from "@mui/material"
import { VoteFor } from "@shared/consts";
import { ReportsLink } from "components/buttons";
import { UpvoteDownvote } from "components/lists";
import { useMemo } from "react";
import { getListItemPermissions } from "utils";
import { DateDisplay } from "../DateDisplay/DateDisplay"
import { StatsCompactProps, StatsCompactPropsObject } from "../types"
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

    const permissions = useMemo(() => getListItemPermissions(object as any, session), [object, session]);

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
            <UpvoteDownvote
                direction="row"
                disabled={!permissions.canVote}
                session={session}
                objectId={object?.id ?? ''}
                voteFor={(object?.__typename as any) ?? VoteFor.Routine}
                isUpvoted={object?.isUpvoted}
                score={object?.score}
                onChange={(isUpvote, score) => { object && handleObjectUpdate({ 
                    ...object, 
                    isUpvoted: isUpvote,
                    score: score, 
                }); }}
            />
            {/* Views */}
            <ViewsDisplay views={(object as any)?.views} />
            {/* Date created */}
            <DateDisplay
                loading={loading}
                showIcon={true}
                timestamp={object?.created_at}
            />
            {/* Reports */}
            {object?.id && <ReportsLink object={object as any} />}
        </Stack>
    )
}