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
        if (award.nextTier)  return award.nextTier
        // If not, but earned tier exists, then we must be at the top tier already
        if (award.earnedTier) return award.earnedTier
        // Otherwise, award invalid
        return { title: '',  description: '', level: 0 }
    }, [award]);
    console.log('award card', award);

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
                <LinearProgress variant="determinate" value={level > 0 ? (award.progress / level) : 0} />
            </CardContent>
        </Card>
    );
}