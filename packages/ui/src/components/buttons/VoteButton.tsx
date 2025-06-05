import { Box, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { getReactionScore, removeModifiers } from "@vrooli/shared";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../contexts/session.js";
import { useVoter } from "../../hooks/objectActions.js";
import { IconCommon } from "../../icons/Icons.js";
import { ObjectActionComplete, type ActionCompletePayloads } from "../../utils/actions/objectActions.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { type VoteButtonProps } from "./types.js";

export function VoteButton({
    disabled = false,
    emoji,
    score,
    objectId,
    voteFor,
    onChange,
}: VoteButtonProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);
    const { t } = useTranslation();

    // Used to respond to user clicks immediately, without having 
    // to wait for the mutation to complete
    const [internalEmoji, setInternalEmoji] = useState<string | null | undefined>(emoji);
    useEffect(() => setInternalEmoji(emoji), [emoji]);

    // Internal state for score, based on difference between internal and external emoji
    const internalScore = useMemo(() => {
        const scoreNum = score ?? 0;
        const internalFeeling = getReactionScore(internalEmoji ? removeModifiers(internalEmoji) : null);
        const externalFeeling = getReactionScore(emoji ? removeModifiers(emoji) : null);
        // Examples:
        // If internal and external are the same, k + (x - x) => k + 0 => k
        // If internal is 1 point greater, k + ((x + 1) - x) => k + 1 => k + 1
        // If internal is 1 point less, k + ((x - 1) - x) => k - 1 => k - 1
        return scoreNum + internalFeeling - externalFeeling;
    }, [emoji, internalEmoji, score]);

    const onVoteComplete = useCallback(<T extends "VoteDown" | "VoteUp">(
        action: T,
        _data: ActionCompletePayloads[T],
    ) => {
        // If cancelling, subtracts existing feeling.
        // Otherwise, same logic as calculating the internal score
        const updatedFeeling = getReactionScore(action === ObjectActionComplete.VoteUp ? "👍" : "👎");
        const existingFeeling = getReactionScore(internalEmoji ? removeModifiers(internalEmoji) : null);
        const cancelsVote = existingFeeling === updatedFeeling;
        const newScore = cancelsVote ?
            internalScore - existingFeeling :
            internalScore + updatedFeeling - existingFeeling;
        const newEmoji = cancelsVote ? null : action === ObjectActionComplete.VoteUp ? "👍" : "👎";
        onChange(newEmoji, newScore);
    }, [internalEmoji, internalScore, onChange]);

    const { handleVote: mutate } = useVoter({
        objectId,
        objectType: voteFor,
        onActionComplete: onVoteComplete,
    });

    const handleVote = useCallback((event: any, emoji: string | null) => {
        // Prevent propagation of normal click event
        event.stopPropagation();
        event.preventDefault();
        // Send vote mutation
        mutate(emoji);
    }, [mutate]);

    const handleUpvoteClick = useCallback((event: any) => {
        if (!userId || disabled) return;
        // If already upvoted, cancel the vote
        const vote = getReactionScore(internalEmoji) > 0 ? null : "👍";
        setInternalEmoji(vote);
        handleVote(event, vote);
    }, [userId, disabled, internalEmoji, handleVote]);

    const handleDownvoteClick = useCallback((event: any) => {
        if (!userId || disabled) return;
        // If already downvoted, cancel the vote
        const vote = getReactionScore(internalEmoji) < 0 ? null : "👎";
        setInternalEmoji(vote);
        handleVote(event, vote);
    }, [userId, disabled, internalEmoji, handleVote]);

    const upvoteColor = (!userId || disabled) ?
        palette.background.textSecondary :
        getReactionScore(internalEmoji) > 0 ?
            "#34c38b" :
            "#687074";

    const downvoteColor = (!userId || disabled) ?
        palette.background.textSecondary :
        getReactionScore(internalEmoji) < 0 ?
            "#af2929" :
            "#687074";

    return (
        <Stack direction="row" sx={{ pointerEvents: "none" }}>
            {/* Upvote arrow */}
            <Tooltip title={t("VoteUp")}>
                <Box
                    display="inline-block"
                    onClick={handleUpvoteClick}
                    role="button"
                    aria-pressed={getReactionScore(internalEmoji) > 0}
                    sx={{
                        cursor: (userId && !disabled) ? "pointer" : "default",
                        pointerEvents: "all",
                        display: "flex",
                        "&:hover": {
                            filter: (userId && !disabled) ? "brightness(120%)" : "none",
                            transition: "filter 0.2s",
                        },
                    }}
                >
                    <IconCommon
                        decorative
                        fill={upvoteColor}
                        name="ArrowUp"
                        size={24}
                    />
                </Box>
            </Tooltip>
            {/* Score */}
            <Typography
                variant="body1"
                textAlign="center"
                sx={{
                    marginLeft: "4px",
                    marginRight: "4px",
                    pointerEvents: "none",
                    color: palette.background.textSecondary,
                }}
            >{internalScore}</Typography>
            {/* Downvote arrow */}
            <Tooltip title={t("VoteDown")}>
                <Box
                    display="inline-block"
                    onClick={handleDownvoteClick}
                    role="button"
                    aria-pressed={getReactionScore(internalEmoji) < 0}
                    sx={{
                        cursor: (userId && !disabled) ? "pointer" : "default",
                        pointerEvents: "all",
                        display: "flex",
                        "&:hover": {
                            filter: (userId && !disabled) ? "brightness(120%)" : "none",
                            transition: "filter 0.2s",
                        },
                    }}
                >
                    <IconCommon
                        decorative
                        fill={downvoteColor}
                        name="ArrowDown"
                        size={24}
                    />
                </Box>
            </Tooltip>
        </Stack>
    );
}
