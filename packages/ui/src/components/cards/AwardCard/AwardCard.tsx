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
}: AwardCardProps) => {
    const { palette } = useTheme();

    // Display next tier
    const { title, description, level } = useMemo(() => {
        // If next tier exists, display that
        if (award.nextTier) return award.nextTier
        // If not, but earned tier exists, then we must be at the top tier already
        if (award.earnedTier) return award.earnedTier
        // Otherwise, award invalid
        return { title: '', description: '', level: 0 }
    }, [award]);
    console.log('award card', award);

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
            <CardContent
                sx={{
                    display: 'contents',
                }}
            >
                <Typography
                    variant="h6"
                    component="h2"
                    textAlign="center"
                    sx={{
                        marginBottom: '10px'
                    }}
                >{title}</Typography>
                {/* Display progress */}
                {percentage >= 0 && <>
                    <LinearProgress
                        variant="determinate"
                        value={percentage}
                        sx={{
                            marginLeft: 2,
                            marginRight: 2,
                            height: '12px',
                            borderRadius: '12px',
                        }}
                    />
                    <Typography
                        variant="body2"
                        component="p"
                        textAlign="center"
                        sx={{
                            marginBottom: 1,
                        }}
                    >
                        {award.progress} / {level} ({percentage}%)
                    </Typography>
                </>}
            </CardContent>
        </Card>
    );
}