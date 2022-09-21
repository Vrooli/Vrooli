import { Box, ListItemText, Stack, Tooltip } from '@mui/material';
import { useCallback } from 'react';
import { ReportsButtonProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { APP_LINKS } from '@shared/consts';
import { useLocation } from '@shared/route';
import { ReportIcon } from '@shared/icons';

function determineLink(typename?: string) {
    switch (typename) {
        case 'Organization': return APP_LINKS.Organization;
        case 'Routine': return APP_LINKS.Routine;
        case 'Standard': return APP_LINKS.Standard;
        case 'User': return APP_LINKS.User;
        default:
            console.error('Invalid typename');
            return '';
    }
}

export const ReportsButton = ({
    reportsCount = 0,
    object,
    tooltipPlacement = "left",
}: ReportsButtonProps) => {
    const [, setLocation] = useLocation();

    // When clicked, navigate to reports page
    const handleClick = useCallback((event) => {
        // Do not want the click to propogate to the routine list item, which would cause the routine page to open.
        event.stopPropagation();

        setLocation(`${determineLink(object?.__typename)}/reports/${object?.id}`)
    }, [object?.__typename, object?.id, setLocation]);

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