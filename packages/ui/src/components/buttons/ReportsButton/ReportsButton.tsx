import { Box, ListItemText, Stack, Tooltip } from '@mui/material';
import { useCallback } from 'react';
import { ReportsButtonProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useLocation } from '@shared/route';
import { ReportIcon } from '@shared/icons';
import { getObjectUrlBase, getObjectSlug } from 'utils';

export const ReportsButton = ({
    reportsCount = 0,
    object,
    tooltipPlacement = "left",
}: ReportsButtonProps) => {
    const [, setLocation] = useLocation();

    // When clicked, navigate to reports page
    const handleClick = useCallback((event) => {
        if (!object) return;
        // Do not want the click to propogate to the routine list item, which would cause the routine page to open.
        event.stopPropagation();
        setLocation(`${getObjectUrlBase(object)}/reports/${getObjectSlug(object)}`)
    }, [object, setLocation]);

    return (
        <Stack
            direction="row"
            spacing={0.5}
            sx={{
                marginRight: 0,
            }}
        >
            <Tooltip placement={tooltipPlacement} title={'View reports'}>
                <Box onClick={handleClick} sx={{ display: 'contents', cursor: 'pointer' }}>
                    <ReportIcon fill='#a96666' />
                </Box>
            </Tooltip>
            <ListItemText
                primary={reportsCount}
                sx={{ ...multiLineEllipsis(1) }}
            />
        </Stack>
    )
}