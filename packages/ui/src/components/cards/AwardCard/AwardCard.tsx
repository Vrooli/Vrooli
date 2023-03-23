import {
    Card,
    CardContent,
    LinearProgress,
    Typography,
    useTheme
} from '@mui/material';
import { useMemo } from 'react';
import { AwardCardProps } from '../types';

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
            if (award.nextTier) return award.nextTier
            // Default to earned tier if no next tier
            if (award.earnedTier) return award.earnedTier
        }
        // If earned, display earned tier
        else if (award.earnedTier) return award.earnedTier
        // If here, award invalid
        return { title: '', description: '', level: 0 }
    }, [award.earnedTier, award.nextTier, isEarned]);

    // Calculate percentage complete
    const percentage = useMemo(() => {
        if (award.progress === 0) return 0;
        if (level === 0) return -1;
        return Math.round((award.progress / level) * 100);
    }, [award.progress, level]);

    return (
        <Card sx={{
            width: '100%',
            height: '100%',
            boxShadow: 6,
            background: palette.primary.light,
            color: palette.primary.contrastText,
            borderRadius: '16px',
            margin: 0,
        }}>
            <CardContent sx={{
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'space-between',
                height: '100%',
            }}>
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
                    <LinearProgress
                        variant="determinate"
                        value={percentage}
                        sx={{
                            margin: 2,
                            marginBottom: 1,
                            height: '12px',
                            borderRadius: '12px',
                        }}
                    />
                    <Typography
                        variant="body2"
                        component="p"
                        textAlign="center"
                    >
                        {award.progress} / {level} ({percentage}%)
                    </Typography>
                </>}
            </CardContent>
        </Card>
    );
}