// Used to display popular/search results of a particular object type
import { Box, CircularProgress, Link, Stack, Tooltip, Typography } from '@mui/material';
import { TitleContainerProps } from '../types';
import { clickSize, containerShadow } from 'styles';
import { HelpButton } from 'components';

export function TitleContainer({
    title = 'Popular Items',
    onClick,
    loading = false,
    tooltip = '',
    helpText = '',
    options = [],
    sx,
    children,
}: TitleContainerProps) {
    return (
        <Tooltip placement="bottom" title={tooltip}>
            <Box display="flex" justifyContent="center">
                <Box
                    onClick={onClick}
                    sx={{
                        ...containerShadow,
                        borderRadius: '8px',
                        background: (t) => t.palette.background.default,
                        width: 'min(100%, 700px)',
                        cursor: 'pointer',
                        '&:hover': {
                            filter: `brightness(102%)`,
                        },
                        ...sx
                    }}
                >
                    {/* Title container */}
                    <Box sx={{
                        background: (t) => t.palette.primary.dark,
                        color: (t) => t.palette.primary.contrastText,
                        borderRadius: '8px 8px 0 0',
                        padding: 0.5,
                    }}>
                        {/* Title */}
                        <Stack direction="row" justifyContent="center" alignItems="center">
                            <Typography component="h2" variant="h4" textAlign="center">{title}</Typography>
                            {Boolean(helpText) ? <HelpButton markdown={helpText} sx={{ fill: '#a0e7c4' }} /> : null}
                        </Stack>
                    </Box>
                    {/* Main content */}
                    <Stack direction="column">
                        <Box sx={{
                            minHeight: 'min(300px, 25vh)',
                            ...(loading ? {
                                display: 'flex',
                                justifyContent: 'center',
                                alignItems: 'center',
                            } : {
                                display: 'block',
                            })
                        }}>
                            {loading ? <CircularProgress color="secondary" /> : children}
                        </Box>
                        {/* Links */}
                        {
                            options.length > 0 && (
                                <Stack direction="row" sx={{
                                    ...clickSize,
                                    justifyContent: 'end',
                                }}
                                >
                                    {options.map(([label, onClick], index) => (
                                        <Link key={index} onClick={onClick}>
                                            <Typography sx={{ marginRight: 2, color: (t) => t.palette.secondary.dark }}>{label}</Typography>
                                        </Link>
                                    ))}
                                </Stack>
                            )
                        }
                    </Stack>
                </Box>
            </Box>
        </Tooltip>
    );
}