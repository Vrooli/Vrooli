import { ListObject, ReactionFor } from "@local/shared";
import { Box, Stack, Typography, useTheme } from "@mui/material";
import { ReportsLink } from "components/buttons/ReportsLink/ReportsLink";
import { VoteButton } from "components/buttons/VoteButton/VoteButton";
import { VisibleIcon } from "icons";
import { useMemo } from "react";
import { getCounts, getYou } from "utils/display/listTools";
import { StatsCompactProps } from "../types";

/**
 * Displays basic stats about an object, in a short format.
 * Displays votes, views, date created, and reports
 */
export const StatsCompact = <T extends ListObject>({
    handleObjectUpdate,
    object,
}: StatsCompactProps<T>) => {
    const { palette } = useTheme();

    const you = useMemo(() => getYou(object), [object]);
    const counts = useMemo(() => getCounts(object), [object]);
    console.log("stats you", you, object);

    return (
        <Stack
            direction="row"
            spacing={1}
            sx={{
                marginTop: 1,
                marginBottom: 1,
                display: "flex",
                alignItems: "center",
                color: palette.background.textSecondary,
            }}
        >
            {/* Votes. Show even if you can't vote */}
            {object && object.__typename.replace("Version", "") in ReactionFor && <VoteButton
                disabled={!you.canReact}
                emoji={you.reaction}
                objectId={object.id ?? ""}
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
            <Box display="flex" alignItems="center">
                <VisibleIcon />
                <Typography variant="body2">
                    {(object as any)?.views ?? 1}
                </Typography>
            </Box>
            {/* Reports */}
            {object?.id && <ReportsLink object={object as any} />}
        </Stack>
    );
};
