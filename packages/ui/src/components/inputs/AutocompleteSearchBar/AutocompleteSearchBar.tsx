import { Autocomplete, AutocompleteChangeDetails, AutocompleteChangeReason, AutocompleteHighlightChangeReason, CircularProgress, IconButton, Input, ListItemIcon, ListItemText, MenuItem, Paper, Tooltip, useTheme } from '@mui/material';
import { AutocompleteSearchBarProps } from '../types';
import { ChangeEvent, useCallback, useState, useEffect, useMemo } from 'react';
import { AutocompleteOption } from 'types';
import { ActionIcon, DeleteIcon, HistoryIcon, OrganizationIcon, PlayIcon, ProjectIcon, RoutineIcon, SearchIcon, ShortcutIcon, StandardIcon, StarFilledIcon, SvgComponent, UserIcon, VisibleIcon } from '@shared/icons';
import { getLocalStorageKeys, performAction, useDebounce } from 'utils';
import { StarFor } from '@shared/consts';
import { StarButton } from 'components/buttons';
import { getCurrentUser } from 'utils/authentication';

type OptionHistory = { timestamp: number, option: AutocompleteOption };

/**
 * Gets search history from local storage, unique by search bar and account
 * @param searchBarId The search bar ID
 * @param userId The user ID
 */
const getSearchHistory = (searchBarId: string, userId: string): { [label: string]: OptionHistory } => {
    const existingHistoryString: string = localStorage.getItem(`search-history-${searchBarId}-${userId}`) ?? '{}';
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
 * For a list of options, checks if they are stored in local storage and need their stars/isStarred updated. If so, updates them.
 * @param searchBarId The search bar ID
 * @param userId The user ID
 * @param options The options to check
 */
const updateHistoryItems = (searchBarId: string, userId: string, options: AutocompleteOption[]) => {
    // Find all search history objects in localStorage
    const searchHistoryKeys = getLocalStorageKeys({
        prefix: 'search-history-',
        suffix: userId,
    });
    // For each search history object, perform update
    searchHistoryKeys.forEach(key => {
        // Get history data
        let existingHistory: { [label: string]: OptionHistory } = {};
        // Try to parse existing history
        try {
            const parsedHistory: any = JSON.parse(localStorage.getItem(key) ?? '{}');
            // If it's not an object, set it to an empty object
            if (typeof existingHistory !== 'object') existingHistory = {};
            else existingHistory = parsedHistory;
        } catch (e) {
            existingHistory = {};
        }
        // Find options that are in history, and update if stars or isStarred are different
        let updatedHistory = options.map(option => {
            // stars and isStarred are not in shortcuts or actions
            if (option.type === 'Shortcut' || option.type === 'Action') return null;
            const historyItem = existingHistory[option.label];
            if (historyItem && historyItem.option.type !== 'Shortcut' && historyItem.option.type !== 'Action') {
                const { stars, isStarred } = option;
                if (stars !== historyItem.option.stars || isStarred !== historyItem.option.isStarred) {
                    return {
                        timestamp: historyItem.timestamp,
                        option: {
                            ...option,
                            stars,
                            isStarred,
                        }
                    };
                }
            }
            return null;
        }).filter(Boolean) as OptionHistory[];
        // Update changed options
        if (updatedHistory.length > 0) {
            for (const historyItem of updatedHistory) {
                existingHistory[historyItem.option.label] = historyItem;
            }
            localStorage.setItem(key, JSON.stringify(existingHistory));
        }
    });
}

/**
 * Maps object types to icons
 */
const typeToIcon = (type: string, fill: string): JSX.Element | null => {
    let Icon: null | SvgComponent = null;
    switch (type) {
        case 'Action':
            Icon = ActionIcon;
            break;
        case 'Organization':
            Icon = OrganizationIcon;
            break;
        case 'Project':
            Icon = ProjectIcon;
            break;
        case 'Routine':
            Icon = RoutineIcon;
            break;
        case 'Run':
            Icon = PlayIcon;
            break;
        case 'Shortcut':
            Icon = ShortcutIcon;
            break;
        case 'Standard':
            Icon = StandardIcon;
            break;
        case 'Star':
            Icon = StarFilledIcon;
            break;
        case 'User':
            Icon = UserIcon;
            break;
        case 'View':
            Icon = VisibleIcon;
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
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    // Input internal value (since value passed back is debounced)
    const [internalValue, setInternalValue] = useState<string>(value);
    // Highlighted option (if navigated with keyboard)
    const [highlightedOption, setHighlightedOption] = useState<AutocompleteOption | null>(null);

    const onChangeDebounced = useDebounce(onChange, debounce);
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

    const [optionsWithHistory, setOptionsWithHistory] = useState<AutocompleteOption[]>(options);
    useEffect(() => {
        // Grab history from local storage
        const searchHistory = getSearchHistory(id, userId ?? '');
        // Filter out history keys that don't contain internal value
        let filteredHistory = Object.entries(searchHistory).filter(([key]) => key.toLowerCase().includes(internalValue.toLowerCase()));
        // Order remaining history keys by most recent. Value is stored as { timestamp: string, value: AutocompleteOption }
        filteredHistory = filteredHistory.sort((a, b) => { return b[1].timestamp - a[1].timestamp });
        // Convert history keys to options
        let historyOptions: AutocompleteOption[] = filteredHistory.map(([, value]) => ({ ...value.option, isFromHistory: true }));
        // Limit to 5 options
        historyOptions = historyOptions.slice(0, 5);
        // Filter out options that are in history (use id to check)
        const filteredOptions = options.filter(option => !historyOptions.some(historyOption => historyOption.id === option.id));
        // If any options have a stars/isStarred values which differs from history, update history
        updateHistoryItems(id, userId ?? '', filteredOptions);
        // Combine history and options
        let combinedOptions = [...historyOptions, ...filteredOptions];
        // In case there are bad options, filter out anything with: an empty or whitespace-only label, or no id
        combinedOptions = combinedOptions.filter(option => option.label && option.label.trim() !== '' && option.id);
        // Update state
        setOptionsWithHistory(combinedOptions);
    }, [options, internalValue, id, userId]);

    const removeFromHistory = useCallback((option: AutocompleteOption) => {
        // Get existing history
        const existingHistory = getSearchHistory(id, userId ?? '');
        // Remove the option from history
        delete existingHistory[option.label];
        // Save the new history
        localStorage.setItem(`search-history-${id}-${userId ?? ''}`, JSON.stringify(existingHistory));
        // Update options with history
        setOptionsWithHistory(optionsWithHistory.filter(o => o.id !== option.id));
    }, [id, optionsWithHistory, userId]);

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
        const existingHistory = getSearchHistory(id, userId ?? '');
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
        localStorage.setItem(`search-history-${id}-${userId ?? ''}`, JSON.stringify(existingHistory));
        // If action, perform action and clear input
        if (option.type === 'Action') {
            performAction(option, session);
            // Update state
            setInternalValue('');
            // Remove the highlight
            setHighlightedOption(null);
            // Debounce onChange
            onChangeDebounced('');
        }
        // Trigger onInputChange
        onInputChange(option);
    }, [id, onChangeDebounced, onInputChange, session, userId]);

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

    const handleStar = useCallback((option: AutocompleteOption, isStarred: boolean, event: any) => {
        if (option.type === 'Shortcut' || option.type === 'Action') return;
        // Stop propagation so that the option isn't selected
        event.stopPropagation();
        // Update the option's isStarred and stars value
        const updatedOption = { ...option, isStarred, stars: isStarred ? (option.stars ?? 0) + 1 : (option.stars ?? 1) - 1 };
        // Update history and state
        updateHistoryItems(id, userId ?? '', [updatedOption]);
        setOptionsWithHistory(optionsWithHistory.map(o => o.id === option.id ? updatedOption : o));
    }, [id, optionsWithHistory, userId]);

    // On key down, fill search input with highlighted option if right arrow is pressed
    const onKeyDown = useCallback((event: React.KeyboardEvent) => {
        if (event.key === 'ArrowRight' && highlightedOption) {
            console.log('right arrow pressed', highlightedOption);
            // Update state
            setInternalValue(highlightedOption.label);
            // Debounce onChange
            onChangeDebounced('');
        }
    }, [highlightedOption, onChangeDebounced]);

    return (
        <Autocomplete
            disablePortal
            id={id}
            options={optionsWithHistory}
            onHighlightChange={onHighlightChange}
            onKeyDown={onKeyDown}
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
                            <ListItemIcon sx={{ display: { xs: 'none', sm: 'block' } }}>
                                <HistoryIcon fill={optionColor(true, false)} />
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
                        {/* Star button */}
                        {option.type !== 'Shortcut' && option.type !== 'Action' && option.stars !== undefined && <StarButton
                            isStar={option.isStarred}
                            objectId={option.id}
                            onChange={(isStarred, event) => handleStar(option, isStarred, event)}
                            session={session}
                            showStars={true}
                            starFor={option.type as unknown as StarFor}
                            stars={option.stars}
                            sxs={{ root: { marginRight: 1 } }}
                            tooltipPlacement="right"
                        />}
                        {/* Object icon */}
                        <ListItemIcon>
                            {typeToIcon(option.type, optionColor(option.isFromHistory, true))}
                        </ListItemIcon>
                        {/* If history, show delete icon */}
                        {option.isFromHistory && <Tooltip placement='right' title='Remove'>
                            <IconButton size="small" onClick={(event) => {
                                event.stopPropagation();
                                removeFromHistory(option);
                            }}>
                                <DeleteIcon fill={optionColor(true, true)} />
                            </IconButton>
                        </Tooltip>}
                    </MenuItem>
                )
            }}
            renderInput={(params) => (
                <Paper
                    component="form"
                    sx={{ 
                        ...(sxs?.paper ?? {}), 
                        p: '2px 4px', 
                        display: 'flex', 
                        alignItems: 'center', 
                        borderRadius: '10px' 
                    }}
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