import { Box, IconButton, ListItem, Popover, Stack, TextField, Tooltip, Typography, useTheme } from '@mui/material';
import { SelectLanguageDialogProps } from '../types';
import {
    ArrowDropDown as ArrowDropDownIcon,
    ArrowDropUp as ArrowDropUpIcon,
    Check as CheckIcon,
    Close as CloseIcon,
    Delete as DeleteIcon,
    Language as LanguageIcon,
} from '@mui/icons-material';
import { useCallback, useMemo, useRef, useState } from 'react';
import { AllLanguages, getUserLanguages } from 'utils';
import { FixedSizeList } from 'react-window';

export const SelectLanguageDialog = ({
    availableLanguages,
    canDropdownOpen = true,
    currentLanguage,
    handleDelete,
    handleCurrent,
    isEditing = false,
    selectedLanguages,
    session,
    sxs,
    zIndex,
}: SelectLanguageDialogProps) => {
    const { palette } = useTheme();
    const [searchString, setSearchString] = useState('');
    const updateSearchString = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        setSearchString(event.target.value);
    }, []);

    const languageOptions = useMemo<Array<[string, string]>>(() => {
        // Find user languages
        const userLanguages = getUserLanguages(session);
        const selected = selectedLanguages ?? [];
        // Sort selected languages. Selected languages which are also user languages are first.
        const sortedSelectedLanguages = selected.sort((a, b) => {
            const aIndex = userLanguages.indexOf(a);
            const bIndex = userLanguages.indexOf(b);
            if (aIndex === -1 && bIndex === -1) {
                return 0;
            } else if (aIndex === -1) {
                return 1;
            } else if (bIndex === -1) {
                return -1;
            } else {
                return aIndex - bIndex;
            }
        }) ?? [];
        // Filter selected languages from user languages
        const userLanguagesFiltered = userLanguages.filter(l => selected.indexOf(l) === -1);
        // Select selected and user languages from all languages
        const allLanguagesFiltered = (availableLanguages ?? Object.keys(AllLanguages)).filter(l => selected.indexOf(l) === -1).filter(l => userLanguagesFiltered.indexOf(l) === -1);
        // Create array with all available languages.
        // Selected and user languages first, then selected, then user languages which haven't been seleccted, then other available languages.
        const displayed = [...sortedSelectedLanguages, ...userLanguagesFiltered, ...allLanguagesFiltered];
        // Convert to array of [languageCode, languageDisplayName]
        let options: Array<[string, string]> = displayed.map(l => [l, AllLanguages[l]]);
        // Filter options with search string
        if (searchString.length > 0) {
            options = options.filter((o: [string, string]) => o[1].toLowerCase().includes(searchString.toLowerCase()));
        }
        return options;
    }, [availableLanguages, searchString, selectedLanguages, session]);

    // Popup for selecting language
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
    const anchorRef = useRef<HTMLElement | null>(null);
    const open = Boolean(anchorEl);
    const onOpen = useCallback((event: React.MouseEvent) => {
        if (canDropdownOpen) setAnchorEl(anchorRef.current);
        // Force parent to save current translation
        if (currentLanguage) handleCurrent(currentLanguage);
    }, [canDropdownOpen, currentLanguage, handleCurrent]);
    const onClose = useCallback(() => {
        // Chear text field
        setSearchString('');
        setAnchorEl(null)
    }, []);

    const onDelete = useCallback((e: React.MouseEvent, language: string) => {
        e.preventDefault();
        e.stopPropagation();
        if (handleDelete) handleDelete(language);
    }, [handleDelete]);

    /**
     * Title bar with close icon
     */
    const titleBar = useMemo(() => (
        <Box
            sx={{
                background: palette.primary.dark,
                color: palette.primary.contrastText,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: 1,
            }}
        >
            <Typography component="h2" variant="h5" textAlign="center" sx={{ marginLeft: 'auto' }}>
                Select Language
            </Typography>
            <Box sx={{ marginLeft: 'auto' }}>
                <IconButton
                    edge="start"
                    onClick={onClose}
                >
                    <CloseIcon sx={{ fill: palette.primary.contrastText }} />
                </IconButton>
            </Box>
        </Box>
    ), [onClose, palette])

    return (
        <>
            {/* Language select popover */}
            <Popover
                open={open}
                anchorEl={anchorEl}
                onClose={onClose}
                sx={{
                    zIndex: zIndex + 1,
                    '& .MuiPopover-paper': {
                        background: 'transparent',
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
                {/* Title */}
                {titleBar}
                {/* Search bar and list of languages */}
                <Stack direction="column" spacing={2} sx={{
                    width: 'min(100vw, 400px)',
                    maxHeight: 'min(100vh, 600px)',
                    maxWidth: '100%',
                    overflowX: 'auto',
                    overflowY: 'hidden',
                    background: palette.background.default,
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
                        sx={{ 
                            paddingLeft: 1,
                            paddingRight: 1,
                        }}
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
                            maxWidth: '100%',
                        }}
                    >
                        {(props) => {
                            const { index, style } = props;
                            const option: [string, string] = languageOptions[index];
                            const isSelected = selectedLanguages?.includes(option[0]) || option[0] === currentLanguage;
                            const isCurrent = option[0] === currentLanguage;
                            return (
                                <ListItem
                                    key={index}
                                    style={style}
                                    disablePadding
                                    button
                                    onClick={() => { handleCurrent(option[0]); onClose(); }}
                                    // Darken/lighten selected language (depending on light/dark mode)
                                    sx={{
                                        background: isCurrent ? palette.secondary.light : palette.background.default,
                                        color: isCurrent ? palette.secondary.contrastText : palette.background.textPrimary,
                                        '&:hover': {
                                            background: isCurrent ? palette.secondary.light : palette.background.default,
                                            filter: 'brightness(105%)',
                                        },
                                    }}
                                >
                                    {/* Display check mark if selected */}
                                    {isSelected && (
                                        <CheckIcon
                                            sx={{
                                                color: (isCurrent) ? palette.secondary.contrastText : palette.background.textPrimary,
                                                fontSize: '1.5rem',
                                            }}
                                        />
                                    )}
                                    <Typography variant="body2" style={{
                                        display: 'block',
                                        marginRight: 'auto',
                                    }}>{option[1]}</Typography>
                                    {/* Display delete icon if selected and editing */}
                                    {isSelected && (selectedLanguages?.length ?? 0) > 1 && isEditing && (
                                        <IconButton
                                            size="small"
                                            onClick={(e) => onDelete(e, option[0])}
                                        >
                                            <DeleteIcon
                                                sx={{
                                                    color: (isCurrent) ? palette.secondary.contrastText : palette.background.textPrimary,
                                                    '&:hover': {
                                                        color: palette.error.main,
                                                    },
                                                    fontSize: '1.5rem',
                                                }}
                                            />
                                        </IconButton>
                                    )}
                                </ListItem>
                            );
                        }}
                    </FixedSizeList>
                </Stack>
            </Popover>
            {/* Selected language label */}
            <Tooltip title={AllLanguages[currentLanguage] ?? ''} placement="top">
                <Stack direction="row" ref={anchorRef} spacing={0} onClick={onOpen} sx={{
                    ...(sxs?.root ?? {}),
                    display: availableLanguages === undefined || availableLanguages.length > 0 ? 'flex' : 'none',
                    justifyContent: 'center',
                    alignItems: 'center',
                    borderRadius: '50px',
                    cursor: canDropdownOpen ? 'pointer' : 'default',
                    background: '#4e7d31',
                    '&:hover': {
                        filter: canDropdownOpen ? 'brightness(120%)' : 'brightness(100%)',
                    },
                    transition: 'all 0.2s ease-in-out',
                    width: 'fit-content',
                }}>
                    <IconButton disabled={!canDropdownOpen} size="large" sx={{ padding: '4px' }}>
                        <LanguageIcon sx={{ fill: 'white' }} />
                    </IconButton>
                    <Typography variant="body2" sx={{ color: 'white', marginRight: '8px' }}>
                        {currentLanguage?.toLocaleUpperCase()}
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