import { Card, CardContent, Typography, useTheme } from "@mui/material";
import { CompletionBar } from "components/CompletionBar/CompletionBar";
import { useMemo } from "react";
import { AwardCardProps } from "../types";

export const AwardCard = ({
    award,
    isEarned,
}: AwardCardProps) => {
    const { palette } = useTheme();

    // Display highest earned tier or next tier,
    // depending on isEarned
    const { title, description, level } = useMemo(() => {
        // If not earned, display next tier
        if (!isEarned) {
            if (award.nextTier) return award.nextTier;
            // Default to earned tier if no next tier
            if (award.earnedTier) return award.earnedTier;
        }
        // If earned, display earned tier
        else if (award.earnedTier) return award.earnedTier;
        // If here, award invalid
        return { title: "", description: "", level: 0 };
    }, [award.earnedTier, award.nextTier, isEarned]);

    // Calculate percentage complete
    const percentage = useMemo(() => {
        if (award.progress === 0) return 0;
        if (level === 0) return -1;
        return Math.round((award.progress / level) * 100);
    }, [award.progress, level]);

    const isAlmostThere = useMemo(() => {
        return percentage > 90 && percentage < 100 || level - award.progress === 1;
    }, [percentage, award.progress, level]);

    return (
        <Card sx={{
            width: "100%",
            height: "100%",
            background: isEarned ? palette.secondary.main : palette.primary.light,
            color: palette.primary.contrastText,
            borderRadius: "16px",
            margin: 0,
        }}>
            <CardContent sx={{
                display: "flex",
                flexDirection: "column",
                justifyContent: "space-between",
                height: "100%",
            }}>
                {/* TODO add brone, silver, gold, etc. AwardIcon depending on tier */}
                <Typography
                    variant="h6"
                    component="h2"
                    textAlign="center"
                    mb={2}
                >{title}</Typography>
                <Typography
                    variant="body2"
                    component="p"
                    textAlign="center"
                    mb={4}
                >{description}</Typography>
                {/* Display progress */}
                {percentage >= 0 && <>
                    <CompletionBar
                        showLabel={false}
                        value={percentage}
                    />
                    <Typography
                        variant="body2"
                        component="p"
                        textAlign="center"
                        mt={1}
                    >
                        {award.progress} / {level} ({percentage}%)
                    </Typography>
                    {isAlmostThere &&
                        <Typography
                            variant="body2"
                            component="p"
                            textAlign="center"
                            mt={1}
                            style={{ fontStyle: "italic" }}
                        >
                            Almost there!
                        </Typography>
                    }
                </>}
            </CardContent>
        </Card>
    );
};
