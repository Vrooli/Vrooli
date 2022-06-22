import { ListItemText, Stack, Tooltip } from '@mui/material';
import { useCallback } from 'react';
import { ReportButtonProps } from '../types';
import { Flag as ReportsIcon } from '@mui/icons-material';
import { multiLineEllipsis } from 'styles';

export const ReportButton = ({
    reportsCount = 0,
    object,
    tooltipPlacement = "left",
}: ReportButtonProps) => {

    // When clicked, navigate to reports page
    const handleClick = useCallback(() => {
        //TODO
    }, []);

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