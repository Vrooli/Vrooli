// Used to display popular/search results of a particular object type
import { Box, CircularProgress, Link, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import { TitleContainerProps } from '../types';
import { clickSize, containerShadow } from 'styles';
import { HelpButton } from 'components';

export function TitleContainer({
    id,
    title = 'Popular Items',
    onClick,
    loading = false,
    tooltip = '',
    helpText = '',
    options = [],
    sx,
    children,
}: TitleContainerProps) {
    const { palette } = useTheme();

    return (
        <Tooltip placement="bottom" title={tooltip}>
            <Box id={id} display="flex" justifyContent="center">
                <Box
                    onClick={(e) => { onClick && (onClick as any)(e) }}
                    sx={{
                        ...containerShadow,
                        borderRadius: '8px',
                        overflow: 'overlay',
                        background: palette.background.default,
                        width: 'min(100%, 700px)',
                        cursor: onClick ? 'pointer' : 'default',
                        '&:hover': {
                            filter: `brightness(onClick ? ${102} : ${100}%)`,
                        },
                        ...sx
                    }}
                >
                    {/* Title container */}
                    <Box sx={{
                        background: palette.primary.dark,
                        color: palette.primary.contrastText,
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
                            ...(loading ? {
                                minHeight: 'min(300px, 25vh)',
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
                                        <Link key={index} onClick={onClick} sx={{
                                            marginTop: 'auto',
                                            marginBottom: 'auto',
                                            marginRight: 2,
                                        }}>
                                            <Typography sx={{ color: palette.secondary.dark }}>{label}</Typography>
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