import { ListObject, ReactionFor } from "@local/shared";
import { Box, Typography, styled } from "@mui/material";
import { useCallback, useMemo } from "react";
import { VisibleIcon } from "../../icons/common.js";
import { getCounts, getYou } from "../../utils/display/listTools.js";
import { ReportsLink } from "../buttons/ReportsLink/ReportsLink.js";
import { VoteButton } from "../buttons/VoteButton.js";
import { StatsCompactProps } from "./types.js";

const OuterBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    gap: theme.spacing(2),
    alignItems: "center",
    color: theme.palette.background.textSecondary,
}));

/**
 * Displays basic stats about an object, in a short format.
 * Displays votes, views, date created, and reports
 */
export function StatsCompact<T extends ListObject>({
    handleObjectUpdate,
    object,
}: StatsCompactProps<T>) {

    const you = useMemo(() => getYou(object), [object]);
    const counts = useMemo(() => getCounts(object), [object]);

    const handleChange = useCallback(function handleChangeCallback(newEmoji: string | null, newScore: number) {
        if (!object) return;
        handleObjectUpdate({
            ...object,
            reaction: newEmoji, // TODO
            score: newScore, //TODO
        });
    }, [object, handleObjectUpdate]);

    return (
        <OuterBox>
            {/* Views */}
            <Box display="flex" alignItems="center">
                <VisibleIcon width={32} height={32} />
                <Typography variant="body2">
                    {(object as any)?.views ?? 1}
                </Typography>
            </Box>
            {/* Reports */}
            {object?.id && <ReportsLink object={object as any} />}
            {/* Votes. Show even if you can't vote */}
            {object && object.__typename.replace("Version", "") in ReactionFor && <VoteButton
                disabled={!you.canReact}
                emoji={you.reaction}
                objectId={object.id ?? ""}
                voteFor={object.__typename as ReactionFor}
                score={counts.score}
                onChange={handleChange}
            />}
        </OuterBox>
    );
}
