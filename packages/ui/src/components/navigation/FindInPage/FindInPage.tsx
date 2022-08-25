import { useCallback, useEffect, useState } from 'react';
import {
    Box,
    Dialog,
    DialogContent,
    IconButton,
    TextField,
    Tooltip,
    useTheme,
} from '@mui/material';
import {
    ArrowDownward as PreviousIcon,
    ArrowUpward as NextIcon,
    Close as CloseIcon,
} from '@mui/icons-material';
import { PubSub } from 'utils';
import { Stack } from '@mui/system';
import { CaseSensitiveIcon, RegexIcon, WholeWordIcon } from 'assets/img';

const FindInPage = () => {
    const { palette } = useTheme();

    const [open, setOpen] = useState(false);
    const close = useCallback(() => setOpen(false), []);

    useEffect(() => {
        let dialogSub = PubSub.get().subscribeFindInPage(() => {
            setOpen(o => !o);
        });
        return () => { PubSub.get().unsubscribe(dialogSub) };
    }, [])

    return (
        <Dialog
            open={open}
            sx={{
                '& .MuiDialog-container': {
                    color: 'transparent'
                },
                '& .MuiDialog-paper': {
                    border: palette.mode === 'dark' ? `1px solid white` : 'unset',
                    minWidth: 'min(100%, 400px)',
                    position: 'absolute',
                    top: '5%',
                    right: '5%',
                    overflowY: 'visible',
                }
            }}
        >
            <DialogContent sx={{
                background: palette.background.default,
                position: 'relative',
                overflowY: 'visible',
            }}>
                <Stack direction="row" spacing={1}>
                    {/* Search bar */}
                    <TextField
                        id="command-palette-search"
                        autoFocus={true}
                        placeholder='Find in page...'
                        sx={{
                            width: '100%',
                            background: palette.background.paper,
                            borderRadius: '4px',
                            border: '1px solid #e0e0e0',
                            padding: '8px',
                            marginBottom: '8px',
                        }}
                    />
                    {/* Buttons for case-sensitive, match whole word, and regex */}
                    <Box display="flex" alignItems="center">
                        <Tooltip title="Match case (Alt+C)">
                            <IconButton color="inherit" aria-label="case-sensitive" sx={{ width: '48px', height: '48px' }}>
                                <CaseSensitiveIcon fill={palette.background.textPrimary} width="48" height="48" />
                            </IconButton>
                        </Tooltip>
                        <Tooltip title="Match whole word (Alt+W)">
                            <IconButton color="inherit" aria-label="match whole word" sx={{ width: '48px', height: '48px' }}>
                                <WholeWordIcon fill={palette.background.textPrimary} width="48" height="48" />
                            </IconButton>
                        </Tooltip>
                        <Tooltip title="Use regular expression (Alt+R)">
                            <IconButton color="inherit" aria-label="match regex" sx={{ width: '48px', height: '48px' }}>
                                <RegexIcon fill={palette.background.textPrimary} width="48" height="48" />
                            </IconButton>
                        </Tooltip>
                    </Box>
                    {/* TODO */}
                    {/* Up and down arrows, and close icon */}
                    {/* TODO */}
                </Stack>
            </DialogContent>
        </Dialog >
    );
}

export { FindInPage };