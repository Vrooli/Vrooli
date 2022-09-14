import { Autocomplete, AutocompleteChangeDetails, AutocompleteChangeReason, AutocompleteHighlightChangeReason, CircularProgress, IconButton, Input, ListItemIcon, ListItemText, MenuItem, Paper, Tooltip, useTheme } from '@mui/material';
import { AutocompleteSearchBarProps } from '../types';
import AwesomeDebouncePromise from 'awesome-debounce-promise';
import { ChangeEvent, useCallback, useState, useEffect, useMemo } from 'react';
import { AutocompleteOption } from 'types';
import { HistoryIcon, OrganizationIcon, ProjectIcon, RoutineIcon, SearchIcon, ShortcutIcon, StandardIcon, SvgProps, UserIcon } from '@shared/icons';
import { Delete as DeleteIcon } from '@mui/icons-material';
import { ObjectType } from 'utils';

type OptionHistory = { timestamp: number, option: AutocompleteOption };

/**
 * Gets search history from local storage
 */
const getSearchHistory = (searchBarId: string): { [label: string]: OptionHistory } => {
    const existingHistoryString: string = localStorage.getItem(`search-history-${searchBarId}`) ?? '{}';
    let existingHistory: { [label: string]: OptionHistory } = {};
    // Try to parse existing history
    try {
        const parsedHistory: any = JSON.parse(existingHistoryString);
        // If it's not an object, set it to an empty object
        if (typeof existingHistory !== 'object') existingHistory = {};
        else existingHistory = parsedHistory;
    } catch (e) {
        existingHistory = {};
    }
    return existingHistory;
}

/**
 * Maps object types to icons
 */
const typeToIcon = (type: string, fill: string): JSX.Element | null => {
    let Icon: null | ((props: SvgProps) => JSX.Element) = null;
    switch (type) {
        case ObjectType.Organization:
            Icon = OrganizationIcon;
            break;
        case ObjectType.Project:
            Icon = ProjectIcon;
            break;
        case ObjectType.Routine:
            Icon = RoutineIcon;
            break;
        case 'Shortcut':
            Icon = ShortcutIcon;
            break;
        case ObjectType.Standard:
            Icon = StandardIcon;
            break;
        case ObjectType.User:
            Icon = UserIcon;
            break;
    }
    return Icon ? <Icon fill={fill} /> : null;
}


export function AutocompleteSearchBar({
    id = 'search-bar',
    placeholder = 'Search...',
    options = [],
    value,
    onChange,
    onInputChange,
    debounce = 200,
    loading = false,
    session,
    showSecondaryLabel = false,
    sxs,
    ...props
}: AutocompleteSearchBarProps) {
    const { palette } = useTheme();

    // Input internal value (since value passed back is debounced)
    const [internalValue, setInternalValue] = useState<string>(value);
    // Highlighted option (if navigated with keyboard)
    const [highlightedOption, setHighlightedOption] = useState<AutocompleteOption | null>(null);

    const onChangeDebounced = useMemo(() => AwesomeDebouncePromise(
        onChange,
        debounce,
    ), [onChange, debounce]);
    useEffect(() => setInternalValue(value), [value]);
    const handleChange = useCallback((event: ChangeEvent<any>) => {
        // Get the new input string
        const { value } = event.target;
        // Update state
        setInternalValue(value);
        // Remove the highlight
        setHighlightedOption(null);
        // Debounce onChange
        onChangeDebounced(value);
    }, [onChangeDebounced]);

    // For testing purposes
    useEffect(() => {
        console.log('search history', getSearchHistory(id));
    }, [id]);

    const [optionsWithHistory, setOptionsWithHistory] = useState<AutocompleteOption[]>(options);
    useEffect(() => {
        // Grab history from local storage
        const searchHistory = getSearchHistory(id);
        // Filter out history keys that don't contain internal value
        let filteredHistory = Object.entries(searchHistory).filter(([key]) => key.includes(internalValue));
        // Order remaining history keys by most recent. Value is stored as { timestamp: string, value: AutocompleteOption }
        filteredHistory = filteredHistory.sort((a, b) => { return b[1].timestamp - a[1].timestamp });
        // Convert history keys to options
        let historyOptions: AutocompleteOption[] = filteredHistory.map(([, value]) => ({ ...value.option, isFromHistory: true }));
        // Limit to 5 options
        historyOptions = historyOptions.slice(0, 5);
        // Filter out options that are in history (use id to check)
        const filteredOptions = options.filter(option => !historyOptions.some(historyOption => historyOption.id === option.id));
        // Set options with history
        setOptionsWithHistory([...historyOptions, ...filteredOptions]);
    }, [options, internalValue, id]);

    const removeFromHistory = useCallback((option: AutocompleteOption) => {
        // Get existing history
        const existingHistory = getSearchHistory(id);
        // Remove the option from history
        delete existingHistory[option.label];
        // Save the new history
        localStorage.setItem(`search-history-${id}`, JSON.stringify(existingHistory));
        // Update options with history
        setOptionsWithHistory(optionsWithHistory.filter(o => o.id !== option.id));
    }, [id, optionsWithHistory]);

    /**
     * If no options but loading, display a loading indicator
     */
    const noOptionsText = useMemo(() => {
        if (loading || (internalValue !== value)) {
            return (
                <>
                    <CircularProgress
                        color="secondary"
                        size={20}
                        sx={{
                            marginRight: 1,
                        }}
                    />
                    Loading...
                </>
            )
        }
        return 'No options';
    }, [loading, value, internalValue]);

    const onHighlightChange = useCallback((event: React.SyntheticEvent<Element, Event>, option: AutocompleteOption | null, reason: AutocompleteHighlightChangeReason) => {
        if (option && option.label && reason === 'keyboard') {
            setHighlightedOption(option);
        }
    }, [])

    const handleSelect = useCallback((option: AutocompleteOption) => {
        // Add to search history
        const existingHistory = getSearchHistory(id);
        // If history has more than 500 entries, remove the oldest
        if (Object.keys(existingHistory).length > 500) {
            const oldestKey = Object.keys(existingHistory).sort((a, b) => existingHistory[a].timestamp - existingHistory[b].timestamp)[0];
            delete existingHistory[oldestKey];
        }
        // Add new entry
        existingHistory[option.label] = {
            timestamp: Date.now(),
            option: ({ ...option, isFromHistory: true }),
        };
        // Save to local storage
        localStorage.setItem(`search-history-${id}`, JSON.stringify(existingHistory));
        // Call onInputChange
        onInputChange(option);
    }, [id, onInputChange]);

    const onSubmit = useCallback((event: React.SyntheticEvent<Element, Event>, value: AutocompleteOption | null, reason: AutocompleteChangeReason, details?: AutocompleteChangeDetails<any> | undefined) => {
        // If there is a highlighted option, use that
        if (highlightedOption) {
            handleSelect(highlightedOption);
        }
        // Otherwise, don't submit
    }, [highlightedOption, handleSelect])

    const optionColor = useCallback((isFromHistory: boolean | undefined, isSecondary: boolean): string => {
        if (isFromHistory) return palette.mode === 'dark' ? 'hotpink' : 'purple';
        return isSecondary ? palette.background.textSecondary : palette.background.textPrimary;
    }, [palette]);

    return (
        <Autocomplete
            disablePortal
            id={id}
            options={optionsWithHistory}
            onHighlightChange={onHighlightChange}
            inputValue={internalValue}
            getOptionLabel={(option: AutocompleteOption) => option.label ?? ''}
            // Stop default onSubmit, since this reloads the page for some reason
            onSubmit={(event: any) => {
                event.preventDefault();
                event.stopPropagation();
            }}
            // The real onSubmit, since onSubmit is only triggered after 2 presses of the enter key (don't know why)
            onChange={onSubmit}
            renderOption={(props, option) => {
                // If loading, display spinner
                if (option.__typename === 'Loading') {
                    return (
                        <MenuItem key="loading">
                            {/* Object title */}
                            <ListItemText sx={{
                                '& .MuiTypography-root': {
                                    textOverflow: 'ellipsis',
                                    whiteSpace: 'nowrap',
                                    overflow: 'hidden',
                                },
                            }}>
                                <CircularProgress
                                    color="secondary"
                                    size={20}
                                    sx={{
                                        marginRight: 1,
                                    }}
                                />
                                {option.label ?? ''}
                            </ListItemText>
                        </MenuItem>
                    )
                }
                return (
                    <MenuItem
                        {...props}
                        key={option.id}
                        onClick={() => {
                            const label = option.label ?? '';
                            setInternalValue(label);
                            onChangeDebounced(label);
                            handleSelect(option);
                        }}
                        sx={{
                            color: optionColor(option.isFromHistory, false),
                        }}
                    >
                        {/* Show history icon if from history */}
                        {option.isFromHistory && (
                            <ListItemIcon>
                                <HistoryIcon fill='hotpink' />
                            </ListItemIcon>
                        )}
                        {/* Object title */}
                        <ListItemText sx={{
                            '& .MuiTypography-root': {
                                textOverflow: 'ellipsis',
                                whiteSpace: 'nowrap',
                                overflow: 'hidden',
                            },
                        }}>
                            {option.label}
                        </ListItemText>
                        {/* Object icon */}
                        <ListItemIcon>
                            {typeToIcon(option.__typename, optionColor(option.isFromHistory, true))}
                        </ListItemIcon>
                        {/* If history, show delete icon */}
                        {option.isFromHistory && <Tooltip placement='right' title='Remove'>
                            <IconButton size="small" onClick={(event) => {
                                event.stopPropagation();
                                removeFromHistory(option);
                            }}>
                                <DeleteIcon sx={{ fill: "hotpink" }} />
                            </IconButton>
                        </Tooltip>}
                    </MenuItem>
                )
            }}
            renderInput={(params) => (
                <Paper
                    component="form"
                    sx={{ ...(sxs?.paper ?? {}), p: '2px 4px', display: 'flex', alignItems: 'center', borderRadius: '10px' }}
                >
                    <Input
                        id={params.id}
                        disabled={params.disabled}
                        disableUnderline={true}
                        fullWidth={params.fullWidth}
                        value={internalValue}
                        onChange={handleChange}
                        placeholder={placeholder}
                        autoFocus={props.autoFocus ?? false}
                        // {...params.InputLabelProps}
                        inputProps={params.inputProps}
                        ref={params.InputProps.ref}
                        size={params.size}
                        sx={{
                            ml: 1,
                            flex: 1,
                            // Drop down/up icon
                            '& .MuiAutocomplete-endAdornment': {
                                width: '48px',
                                height: '48px',
                                top: '0',
                                position: 'relative',
                                '& .MuiButtonBase-root': {
                                    width: '48px',
                                    height: '48px',
                                }
                            }
                        }}
                    />
                    <IconButton sx={{
                        width: '48px',
                        height: '48px',
                    }} aria-label="main-search-icon">
                        <SearchIcon fill={palette.background.textSecondary} />
                    </IconButton>
                </Paper>
            )}
            noOptionsText={noOptionsText}
            sx={{
                ...(sxs?.root ?? {}),
                '& .MuiAutocomplete-inputRoot': {
                    paddingRight: '0 !important',
                },
            }}
        />
    );
}