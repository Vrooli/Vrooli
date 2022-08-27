import { ListItemText, Stack, Tooltip } from '@mui/material';
import { useCallback } from 'react';
import { ReportButtonProps } from '../types';
import { Flag as ReportsIcon } from '@mui/icons-material';
import { multiLineEllipsis } from 'styles';
import { APP_LINKS, StarFor, VoteFor } from '@local/shared';
import { useLocation } from 'wouter';

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

export const ReportButton = ({
    reportsCount = 0,
    object,
    tooltipPlacement = "left",
}: ReportButtonProps) => {
    const [, setLocation] = useLocation();

    // When clicked, navigate to reports page
    const handleClick = useCallback((event) => {
        // Do not want the click to propogate to the routine list item, which would cause the routine page to open.
        event.stopPropagation();

        setLocation(`${determineLink(object?.__typename)}/reports/${object?.id}`)
    }, [setLocation]);

    return (
        <Stack
            direction="row"
            spacing={1}
            sx={{
                marginRight: 0,
            }}
        >
            <Tooltip placement={tooltipPlacement} title={'View reports'}>
                <ReportsIcon
                    onClick={handleClick}
                    sx={{
                        fill: '#a96666',
                        cursor: 'pointer'
                    }}
                />
            </Tooltip>
            <ListItemText
                primary={reportsCount}
                sx={{ ...multiLineEllipsis(1) }}
            />
        </Stack>
    )
}