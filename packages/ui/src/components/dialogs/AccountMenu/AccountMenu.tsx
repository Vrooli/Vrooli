import { useCallback, useEffect, useState } from 'react';
import {
    Box,
    Dialog,
    DialogContent,
    IconButton,
    Palette,
    TextField,
    Tooltip,
    Typography,
    useTheme,
} from '@mui/material';
import { Stack } from '@mui/system';
import { ArrowDownIcon, ArrowUpIcon, CaseSensitiveIcon, CloseIcon, RegexIcon, WholeWordIcon } from '@shared/icons';
import { AccountMenuProps } from '../types';

export const AccountMenu = ({
    // handleClose,
    // isOpen,
}: AccountMenuProps) => {
    const { palette } = useTheme();

    return null;

    // return (
    //     <Dialog
    //         open={isOpen}
    //         onClose={handleClose}
    //         disableScrollLock={true}
    //         BackdropProps={{ invisible: true }}
    //         sx={{
    //             '& .MuiDialog-paper': {
    //                 background: palette.background.paper,
    //                 border: palette.mode === 'dark' ? `1px solid white` : 'unset',
    //                 minWidth: 'min(100%, 350px)',
    //                 position: 'absolute',
    //                 top: '0%',
    //                 right: '0%',
    //                 overflowY: 'visible',
    //                 margin: { xs: '8px', sm: '16px' },
    //             },
    //             '& .MuiDialogContent-root': {
    //                 padding: '12px 8px',
    //                 borderRadius: '4px',
    //             },
    //         }}
    //     >
    //         <DialogContent sx={{
    //             background: palette.background.default,
    //             position: 'relative',
    //             overflowY: 'visible',
    //         }}>
    //             <Stack direction="row">
    //                 <Stack direction="row" sx={{
    //                     background: palette.background.paper,
    //                     borderRadius: '4px',
    //                     border: `1px solid ${searchString.length > 0 && results.length === 0 ? 'red' : palette.background.textPrimary}`,
    //                 }}>
    //                     {/* Search bar */}
    //                     <TextField
    //                         id="command-palette-search"
    //                         autoComplete='off'
    //                         autoFocus={true}
    //                         placeholder='Find in page...'
    //                         value={searchString}
    //                         onChange={onSearchChange}
    //                         size="small"
    //                         sx={{
    //                             paddingLeft: '4px',
    //                             paddingTop: '4px',
    //                             paddingBottom: '4px',
    //                             width: '100%',
    //                             border: 'none',
    //                             borderRight: `1px solid ${palette.background.textPrimary}`,
    //                             '& .MuiInputBase-root': {
    //                                 height: '100%',
    //                             },
    //                         }}
    //                         variant="standard"
    //                         InputProps={{
    //                             disableUnderline: true,
    //                         }}
    //                     />
    //                     {/* Display resultIndex and total results */}
    //                     {results.length > 0 &&
    //                         <Typography variant="body2" sx={{
    //                             padding: '4px',
    //                             borderRight: `1px solid ${palette.background.textPrimary}`,
    //                             width: 100,
    //                             display: 'flex',
    //                             justifyContent: 'center',
    //                             alignItems: 'center',
    //                         }}>
    //                             {results.length > 0 ? `${resultIndex + 1}/${results.length}` : ''}
    //                         </Typography>
    //                     }
    //                     {/* Buttons for case-sensitive, match whole word, and regex */}
    //                     <Box display="flex" alignItems="center">
    //                         <Tooltip title="Match case (Alt+C)">
    //                             <IconButton aria-label="case-sensitive" sx={commonButtonSx(palette, isCaseSensitive)} onClick={onCaseSensitiveChange}>
    //                                 <CaseSensitiveIcon {...commonIconProps(palette)} />
    //                             </IconButton>
    //                         </Tooltip>
    //                         <Tooltip title="Match whole word (Alt+W)">
    //                             <IconButton aria-label="match whole word" sx={commonButtonSx(palette, isWholeWord)} onClick={onWholeWordChange}>
    //                                 <WholeWordIcon {...commonIconProps(palette)} />
    //                             </IconButton>
    //                         </Tooltip>
    //                         <Tooltip title="Use regular expression (Alt+R)">
    //                             <IconButton aria-label="match regex" sx={commonButtonSx(palette, isRegex)} onClick={onRegexChange}>
    //                                 <RegexIcon {...commonIconProps(palette)} />
    //                             </IconButton>
    //                         </Tooltip>
    //                     </Box>
    //                 </Stack>
    //                 {/* Up and down arrows, and close icon */}
    //                 <Box display="flex" alignItems="center" justifyContent="flex-end">
    //                     <Tooltip title="Previous result (Shift+Enter)">
    //                         <IconButton aria-label="previous result" sx={commonButtonSx(palette)} onClick={onPrevious}>
    //                             <ArrowUpIcon {...commonIconProps(palette)} />
    //                         </IconButton>
    //                     </Tooltip>
    //                     <Tooltip title="Next result (Enter)">
    //                         <IconButton aria-label="next result" sx={commonButtonSx(palette)} onClick={onNext}>
    //                             <ArrowDownIcon {...commonIconProps(palette)} />
    //                         </IconButton>
    //                     </Tooltip>
    //                     <Tooltip title="Close">
    //                         <IconButton aria-label="close" sx={commonButtonSx(palette)} onClick={close}>
    //                             <CloseIcon {...commonIconProps(palette)} />
    //                         </IconButton>
    //                     </Tooltip>
    //                 </Box>
    //             </Stack>
    //         </DialogContent>
    //     </Dialog >
    // );
}