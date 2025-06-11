import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { styled } from "@mui/material/styles";
import { ReactionFor, setDotNotationValue, type ListObject } from "@vrooli/shared";
import { useCallback, useMemo } from "react";
import { IconCommon } from "../../icons/Icons.js";
import { getCounts, getYou, getYouDot } from "../../utils/display/listTools.js";
import { ReportsLink } from "../buttons/ReportsLink.js";
import { VoteButton } from "../buttons/VoteButton.js";
import { type StatsCompactProps } from "./types.js";

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
        // Try to find existing reaction path
        let reactionPath = getYouDot(object, "reaction");
        
        // If no path found, determine where to store the reaction based on object type
        if (!reactionPath) {
            // For versioned objects, reactions go in root.you.reaction
            if (object.root?.id) {
                reactionPath = "root.you.reaction";
            } else {
                // For direct objects, reactions go in you.reaction  
                reactionPath = "you.reaction";
            }
        }
        
        const updatedObject = setDotNotationValue(object, reactionPath, newEmoji);
        handleObjectUpdate(updatedObject);
    }, [object, handleObjectUpdate]);

    return (
        <OuterBox>
            {/* Views */}
            <Box display="flex" alignItems="center">
                <IconCommon
                    decorative
                    name="Visible"
                    size={32}
                />
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
