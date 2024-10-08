import { ListObject, ReactionFor } from "@local/shared";
import { Box, Typography, styled } from "@mui/material";
import { ReportsLink } from "components/buttons/ReportsLink/ReportsLink";
import { VoteButton } from "components/buttons/VoteButton/VoteButton";
import { VisibleIcon } from "icons";
import { useCallback, useMemo } from "react";
import { getCounts, getYou } from "utils/display/listTools";
import { StatsCompactProps } from "../types";

const OuterBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    gap: theme.spacing(1),
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
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
            {/* Votes. Show even if you can't vote */}
            {object && object.__typename.replace("Version", "") in ReactionFor && <VoteButton
                disabled={!you.canReact}
                emoji={you.reaction}
                objectId={object.id ?? ""}
                voteFor={object.__typename as ReactionFor}
                score={counts.score}
                onChange={handleChange}
            />}
            {/* Views */}
            <Box display="flex" alignItems="center">
                <VisibleIcon />
                <Typography variant="body2">
                    {(object as any)?.views ?? 1}
                </Typography>
            </Box>
            {/* Reports */}
            {object?.id && <ReportsLink object={object as any} />}
        </OuterBox>
    );
}
