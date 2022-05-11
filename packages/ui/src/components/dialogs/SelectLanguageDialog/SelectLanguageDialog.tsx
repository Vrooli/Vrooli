import { IconButton, ListItem, Popover, Stack, TextField, Tooltip, Typography, useTheme } from '@mui/material';
import { SelectLanguageDialogProps } from '../types';
import {
    ArrowDropDown as ArrowDropDownIcon,
    ArrowDropUp as ArrowDropUpIcon,
    Close as DeleteIcon,
    Language as LanguageIcon,
} from '@mui/icons-material';
import { MouseEvent, useCallback, useMemo, useRef, useState } from 'react';
import { AllLanguages, getUserLanguages } from 'utils';
import { FixedSizeList } from 'react-window';

export const SelectLanguageDialog = ({
    availableLanguages,
    canDelete = false,
    canDropdownOpen = true,
    color = 'default',
    handleDelete,
    handleSelect,
    language,
    onClick,
    session,
    sxs,
}: SelectLanguageDialogProps) => {
    const { palette } = useTheme();
    const [searchString, setSearchString] = useState('');
    const updateSearchString = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        setSearchString(event.target.value);
    }, []);

    const userLanguages = useMemo(() => getUserLanguages(session), [session]);
    const languageOptions = useMemo<Array<[string, string]>>(() => {
        // Handle restricted languages
        let options: Array<[string, string]> = availableLanguages ?
            availableLanguages.map(l => [l, AllLanguages[l]]) : Object.entries(AllLanguages);
        // Handle search string
        if (searchString.length > 0) {
            options = options.filter((o: [string, string]) => o[1].toLowerCase().includes(searchString.toLowerCase()));
        }
        // Reorder so user's languages are first
        options = options.sort((a, b) => {
            const aIndex = userLanguages.indexOf(a[0]);
            const bIndex = userLanguages.indexOf(b[0]);
            if (aIndex === -1 && bIndex === -1) {
                return 0;
            } else if (aIndex === -1) {
                return 1;
            } else if (bIndex === -1) {
                return -1;
            } else {
                return aIndex - bIndex;
            }
        });
        return options;
    }, [availableLanguages, searchString, userLanguages]);

    // Popup for selecting language
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
    const anchorRef = useRef<HTMLElement | null>(null);
    const open = Boolean(anchorEl || language === null);
    const onOpen = useCallback((event: MouseEvent<HTMLDivElement>) => {
        if (onClick) onClick(event);
        if (canDropdownOpen) setAnchorEl(anchorRef.current);
    }, [canDropdownOpen, onClick]);
    const onClose = useCallback(() => setAnchorEl(null), []);

    const onDelete = useCallback((e: MouseEvent<HTMLButtonElement>) => {
        e.preventDefault();
        e.stopPropagation();
        if (handleDelete) handleDelete()
    }, [handleDelete]);

    return (
        <>
            {/* Language select popover */}
            <Popover
                open={open}
                anchorEl={anchorEl}
                onClose={onClose}
                sx={{
                    '& .MuiPopover-paper': {
                        background: 'transparent',
                        boxShadow: 'none',
                        border: 'none',
                        paddingBottom: 1,
                    }
                }}
                anchorOrigin={{
                    vertical: 'bottom',
                    horizontal: 'center',
                }}
                transformOrigin={{
                    vertical: 'top',
                    horizontal: 'center',
                }}
            >
                {/* Search bar and list of languages */}
                <Stack direction="column" spacing={2} sx={{
                    width: 'min(100vw, 400px)',
                    maxHeight: 'min(100vh, 600px)',
                    overflowX: 'auto',
                    overflowY: 'hidden',
                    background: palette.background.paper,
                    borderRadius: '8px',
                    padding: '8px',
                    "&::-webkit-scrollbar": {
                        width: 10,
                    },
                    "&::-webkit-scrollbar-track": {
                        backgroundColor: '#dae5f0',
                    },
                    "&::-webkit-scrollbar-thumb": {
                        borderRadius: '100px',
                        backgroundColor: "#409590",
                    },
                }}>
                    <TextField
                        placeholder="Enter language..."
                        autoFocus={true}
                        value={searchString}
                        onChange={updateSearchString}
                    />
                    <FixedSizeList
                        height={600}
                        width={384}
                        itemSize={46}
                        itemCount={languageOptions.length}
                        overscanCount={5}
                        style={{
                            scrollbarColor: '#409590 #dae5f0',
                            scrollbarWidth: 'thin',
                        }}
                    >
                        {(props) => {
                            const { index, style } = props;
                            const option: [string, string] = languageOptions[index];
                            return (
                                <ListItem
                                    key={index}
                                    style={style}
                                    disablePadding
                                    button
                                    onClick={() => { handleSelect(option[0]); onClose(); }}
                                >
                                    {option[1]}
                                </ListItem>
                            );
                        }}
                    </FixedSizeList>
                </Stack>
            </Popover>
            {/* Selected language label */}
            <Tooltip title={AllLanguages[language] ?? ''} placement="top">
                <Stack direction="row" ref={anchorRef} spacing={0} onClick={onOpen} sx={{
                    ...(sxs?.root ?? {}),
                    display: availableLanguages === undefined || availableLanguages.length > 0 ? 'flex' : 'none',
                    justifyContent: 'center',
                    alignItems: 'center',
                    borderRadius: '50px',
                    cursor: canDropdownOpen ? 'pointer' : 'default',
                    background: color === 'default' ? '#4e7d31' : color,
                    '&:hover': {
                        filter: canDropdownOpen ? 'brightness(120%)' : 'brightness(100%)',
                    },
                    transition: 'all 0.2s ease-in-out',
                }}>
                    {canDelete && <IconButton onClick={onDelete} size="large" sx={{ padding: '4px', marginRight: '-8px' }}>
                        <DeleteIcon sx={{ fill: 'red' }} />
                    </IconButton>}
                    <IconButton disabled={!canDropdownOpen} size="large" sx={{ padding: '4px' }}>
                        <LanguageIcon sx={{ fill: 'white' }} />
                    </IconButton>
                    <Typography variant="body2" sx={{ color: 'white', marginRight: '8px' }}>
                        {language?.toLocaleUpperCase()}
                    </Typography>
                    {/* Drop down or drop up icon */}
                    {canDropdownOpen && <IconButton size="large" aria-label="language-select" sx={{ padding: '4px', marginLeft: '-8px' }}>
                        {open ? <ArrowDropUpIcon sx={{ fill: 'white' }} /> : <ArrowDropDownIcon sx={{ fill: 'white' }} />}
                    </IconButton>}
                </Stack>
            </Tooltip>
        </>
    )
}