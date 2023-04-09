import { Box, ListItemText, Stack, useTheme } from '@mui/material';
import { ReportIcon } from '@shared/icons';
import { useLocation } from '@shared/route';
import { useCallback, useMemo } from 'react';
import { multiLineEllipsis } from 'styles';
import { getObjectReportUrl } from 'utils/navigation/openObject';
import { ReportsButtonProps } from '../types';

export const ReportsButton = ({
    reportsCount = 0,
    object,
}: ReportsButtonProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const link = useMemo(() => object ? getObjectReportUrl(object) : '', [object]);
    const handleClick = useCallback((event: any) => {
        // Stop propagation to prevent list item from being selected
        event.stopPropagation();
        // Prevent default to stop href from being used
        event.preventDefault();
        if (link.length === 0) return;
        setLocation(link);
    }, [link, setLocation]);

    return (
        <Stack
            direction="row"
            spacing={0.5}
            sx={{
                marginRight: 0,
                pointerEvents: 'none',
            }}
        >
            <Box
                component="a"
                href={link}
                onClick={handleClick}
                sx={{
                    display: 'contents',
                    cursor: 'pointer',
                    pointerEvents: 'all',
                }}
            >
                <ReportIcon fill={palette.secondary.main} />
            </Box>
            <ListItemText
                primary={reportsCount}
                sx={{ ...multiLineEllipsis(1), pointerEvents: 'none' }}
            />
        </Stack>
    )
}