import { ResourceUsedFor, getObjectUrl } from "@local/shared";
import { Box, IconButton, InputAdornment, List, ListItem, Popover, TextField, Typography } from "@mui/material";
import { KeyboardEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Icon, IconCommon, IconInfo } from "../../../icons/Icons.js";
import { noSelect } from "../../../styles.js";
import { getResourceIcon } from "../../../utils/display/getResourceIcon.js";

export interface ListObject {
    id: string;
    name: string;
    type: "Note" | "Routine" | "Project" | "Bookmark" | "Web";
}

interface Action {
    type: ListObject["type"];
    label: string;
    iconInfo: IconInfo;
    handler: (query: string) => ListObject;
}

interface Category {
    type: ListObject["type"];
    label: string;
    iconInfo: IconInfo;
    items: ListObject[];
}

export interface AdvancedInputTagDropdownProps {
    anchorEl: HTMLElement | null;
    onClose: () => void;
    onSelect: (item: ListObject) => void;
    /** Optional initial category to be selected when the dropdown opens */
    initialCategory?: ListObject["type"];
}

// Define actions separately from categories
const actions: Action[] = [
    {
        type: "Web",
        label: "Web",
        iconInfo: { name: "Language", type: "Common" },
        handler: (query: string) => ({ id: "web", name: query || "Search Web", type: "Web" }),
    },
];

const categories: Category[] = [
    {
        type: "Note",
        label: "Notes",
        iconInfo: { name: "Note", type: "Common" },
        items: [
            { id: "note1", name: "Test Note", type: "Note" },
        ],
    },
    {
        type: "Routine",
        label: "Routines",
        iconInfo: { name: "Routine", type: "Routine" },
        items: [
            { id: "routine1", name: "Test Routine", type: "Routine" },
        ],
    },
    {
        type: "Project",
        label: "Projects",
        iconInfo: { name: "Project", type: "Common" },
        items: [
            { id: "project1", name: "Test Project", type: "Project" },
        ],
    },
    {
        type: "Bookmark",
        label: "Bookmarks",
        iconInfo: { name: "BookmarkFilled", type: "Common" },
        items: [
            { id: "bookmark1", name: "Test Bookmark", type: "Bookmark" },
        ],
    },
];

const anchorOrigin = {
    vertical: "top",
    horizontal: "left",
} as const;

const transformOrigin = {
    vertical: "bottom",
    horizontal: "left",
} as const;

const listItemStyle = {
    display: "flex",
    alignItems: "center",
    gap: 1,
} as const;

const popoverPaperStyle = {
    maxHeight: "400px",
    width: "300px",
} as const;

const textFieldStyle = { mb: 1 } as const;

const textFieldProps = {
    autoComplete: "off",
    form: {
        autoComplete: "off",
    },
} as const;
const textFieldInputProps = {
    endAdornment: (
        <InputAdornment position="end">
            <IconCommon
                fill="background.textSecondary"
                name="Search"
                size={20}
            />
        </InputAdornment>
    ),
} as const;

const categoryTypographyStyle = {
    px: 1,
    py: 0.5,
    cursor: "pointer",
    borderRadius: 1,
    display: "inline-flex",
    alignItems: "center",
    gap: 0.5,
    width: "100%",
    justifyContent: "space-between",
} as const;

const categoryLabelStyle = {
    display: "inline-flex",
    alignItems: "center",
    gap: 0.5,
} as const;

const listItemSxStyle = {
    ...noSelect,
    ...listItemStyle,
    cursor: "pointer",
    "&:hover": {
        backgroundColor: "action.hover",
    },
    justifyContent: "space-between",
} as const;

const itemContentStyle = {
    display: "flex",
    alignItems: "center",
    gap: 1,
} as const;

const fallbackIconInfo: IconInfo = { name: "Note", type: "Common" };

const backButtonStyle = {
    mr: 1,
} as const;

export function AdvancedInputTagDropdown({
    anchorEl,
    onClose,
    onSelect,
    initialCategory,
}: AdvancedInputTagDropdownProps) {
    const [selectedCategory, setSelectedCategory] = useState<ListObject["type"] | null>(initialCategory ?? null);
    const [searchQuery, setSearchQuery] = useState("");
    const [focusedIndex, setFocusedIndex] = useState(-1);
    const listRef = useRef<HTMLDivElement>(null);
    const searchRef = useRef<HTMLInputElement>(null);

    // Reset selection when dropdown closes
    useEffect(() => {
        if (!anchorEl) {
            setSelectedCategory(initialCategory ?? null);
            setSearchQuery("");
            setFocusedIndex(-1);
        }
    }, [anchorEl, initialCategory]);

    // Update selection when initialCategory changes
    useEffect(() => {
        setSelectedCategory(initialCategory ?? null);
    }, [initialCategory]);

    const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setSearchQuery(e.target.value);
    }, []);

    const handleSearchFocus = useCallback(() => {
        setFocusedIndex(-1);
    }, []);

    const handleSearchKeyDown = useCallback((e: KeyboardEvent<HTMLInputElement>) => {
        switch (e.key) {
            case "Backspace":
                if (selectedCategory && !searchQuery) {
                    e.preventDefault();
                    setSelectedCategory(null);
                }
                break;
        }
    }, [selectedCategory, searchQuery]);

    // Handle action selection
    const handleActionClick = useCallback((action: Action) => {
        const result = action.handler(searchQuery);
        onSelect(result);
        onClose();
    }, [onClose, onSelect, searchQuery]);

    // Handle category selection
    const handleCategoryClick = useCallback((category: Category) => {
        setSelectedCategory(prevType => prevType === category.type ? null : category.type);
        searchRef.current?.focus();
    }, []);

    // Handle item selection
    const handleItemSelect = useCallback((item: ListObject) => {
        onSelect(item);
        onClose();
    }, [onClose, onSelect]);

    // Memoize style getters to avoid recreating objects
    const memoizedCategoryStyle = useMemo(() => {
        return (isSelected: boolean) => ({
            ...categoryTypographyStyle,
            fontWeight: isSelected ? "bold" : "normal",
            backgroundColor: isSelected ? "action.selected" : "transparent",
        });
    }, []);

    // Filter items based on search query and selected category
    const getFilteredCategories = useCallback(() => {
        if (!searchQuery && !selectedCategory) {
            return { categories, actions };
        }

        const filteredCategories = categories
            .map(category => ({
                ...category,
                items: category.items.filter(item =>
                    (!selectedCategory || item.type === selectedCategory) &&
                    (!searchQuery || item.name.toLowerCase().includes(searchQuery.toLowerCase())),
                ),
            }))
            .filter(category => category.items.length > 0 || category.type === selectedCategory);

        return {
            categories: filteredCategories,
            actions: searchQuery ? actions : [],
        };
    }, [searchQuery, selectedCategory]);

    const handleBackClick = useCallback(() => {
        setSelectedCategory(null);
    }, []);

    const { categories: filteredCategories, actions: filteredActions } = getFilteredCategories();

    // Memoize the Popover props to avoid recreating objects
    const popoverProps = useMemo(() => ({
        open: Boolean(anchorEl),
        anchorEl,
        onClose,
        anchorOrigin,
        transformOrigin,
        PaperProps: {
            sx: popoverPaperStyle,
        },
    }), [anchorEl, onClose]);

    // Memoize the flex box style
    const flexBoxStyle = useMemo(() => ({
        display: "flex",
        alignItems: "center",
    }), []);

    // Calculate total number of focusable items
    const totalItems = useMemo(() => {
        const categoryCount = filteredCategories.length;
        const itemsCount = selectedCategory
            ? filteredCategories.find(c => c.type === selectedCategory)?.items.length ?? 0
            : 0;
        const actionsCount = filteredActions.length;
        return categoryCount + itemsCount + actionsCount;
    }, [filteredCategories, filteredActions, selectedCategory]);

    // Handle keyboard navigation
    const handleKeyDown = useCallback((e: KeyboardEvent) => {
        switch (e.key) {
            case "ArrowDown":
                e.preventDefault();
                setFocusedIndex(prev => {
                    if (prev === totalItems - 1) return prev;
                    return prev + 1;
                });
                break;
            case "ArrowUp":
                e.preventDefault();
                setFocusedIndex(prev => {
                    if (prev <= 0) {
                        searchRef.current?.focus();
                        return -1;
                    }
                    return prev - 1;
                });
                break;
            case "ArrowRight":
                e.preventDefault();
                // Only handle right arrow if we're focused on a category
                if (focusedIndex >= 0 && focusedIndex < filteredCategories.length) {
                    const category = filteredCategories[focusedIndex];
                    setSelectedCategory(category.type);
                    // Focus the search field after selecting
                    searchRef.current?.focus();
                    setFocusedIndex(-1);
                }
                break;
            case "ArrowLeft":
                e.preventDefault();
                // Only handle left arrow if we're not in the search field and a category is selected
                if (document.activeElement !== searchRef.current && selectedCategory) {
                    setSelectedCategory(null);
                    // Focus the previously selected category
                    const categoryIndex = filteredCategories.findIndex(c => c.type === selectedCategory);
                    if (categoryIndex >= 0) {
                        setFocusedIndex(categoryIndex);
                    }
                }
                break;
            case "Enter":
            case " ":
                e.preventDefault();
                if (focusedIndex >= 0) {
                    const element = listRef.current?.querySelector(`[data-index="${focusedIndex}"]`) as HTMLElement;
                    element?.click();
                }
                break;
            case "Escape":
                e.preventDefault();
                onClose();
                break;
            case "Tab":
                if (e.shiftKey) {
                    if (focusedIndex === 0 || focusedIndex === -1) {
                        // Allow natural tab order to search field
                        return;
                    }
                    e.preventDefault();
                    setFocusedIndex(prev => prev - 1);
                } else {
                    if (focusedIndex === totalItems - 1) {
                        // Allow natural tab order out of the component
                        return;
                    }
                    e.preventDefault();
                    setFocusedIndex(prev => prev + 1);
                }
                break;
        }
    }, [focusedIndex, totalItems, onClose]);

    // Focus management
    useEffect(() => {
        if (focusedIndex >= 0) {
            const element = listRef.current?.querySelector(`[data-index="${focusedIndex}"]`) as HTMLElement;
            element?.focus();
        }
    }, [focusedIndex]);

    // Memoize click handlers to avoid recreating functions
    const categoryClickHandlers = useMemo(() =>
        filteredCategories.map(category => () => handleCategoryClick(category)),
        [filteredCategories, handleCategoryClick],
    );

    const itemClickHandlers = useMemo(() =>
        filteredCategories.flatMap(category =>
            category.items.map(item => () => handleItemSelect(item)),
        ),
        [filteredCategories, handleItemSelect],
    );

    const actionClickHandlers = useMemo(() =>
        filteredActions.map(action => () => handleActionClick(action)),
        [filteredActions, handleActionClick],
    );

    return (
        <Popover {...popoverProps}>
            <Box
                p={1}
                ref={listRef}
                onKeyDown={handleKeyDown}
                role="menu"
                aria-label="Advanced input options"
            >
                <Box sx={flexBoxStyle}>
                    {selectedCategory && (
                        <IconButton
                            size="small"
                            onClick={handleBackClick}
                            sx={backButtonStyle}
                            aria-label="Go back"
                        >
                            <IconCommon
                                name="ArrowLeft"
                                size={20}
                            />
                        </IconButton>
                    )}
                    <TextField
                        // eslint-disable-next-line jsx-a11y/no-autofocus
                        autoFocus
                        fullWidth
                        size="small"
                        placeholder="Search..."
                        value={searchQuery}
                        onChange={handleSearchChange}
                        onFocus={handleSearchFocus}
                        onKeyDown={handleSearchKeyDown}
                        inputRef={searchRef}
                        sx={textFieldStyle}
                        autoComplete="off"
                        inputProps={textFieldProps}
                        InputProps={textFieldInputProps}
                    />
                </Box>

                {/* Render Categories */}
                {filteredCategories.map((category, categoryIndex) => {
                    const isSelected = selectedCategory === category.type;
                    const categoryStyle = memoizedCategoryStyle(isSelected);
                    const isFocused = focusedIndex === categoryIndex;

                    return (
                        <Box key={category.type} mb={1}>
                            <Typography
                                variant="caption"
                                component="div"
                                sx={categoryStyle}
                                onClick={categoryClickHandlers[categoryIndex]}
                                tabIndex={isFocused ? 0 : -1}
                                role="button"
                                data-index={categoryIndex}
                                aria-haspopup="menu"
                                aria-expanded={isSelected}
                            >
                                <Box sx={categoryLabelStyle}>
                                    <Icon
                                        info={category.iconInfo}
                                        size={16}
                                    />
                                    {category.label}
                                </Box>
                                {!isSelected && (
                                    <IconCommon
                                        name="ChevronRight"
                                        size={16}
                                    />
                                )}
                            </Typography>

                            {(isSelected || searchQuery) && (
                                <List dense disablePadding role="menu">
                                    {category.items.map((item, itemIndex) => {
                                        const link = getObjectUrl(item);
                                        const Icon = getResourceIcon(ResourceUsedFor.Context, link);
                                        // Calculate global index based on all previous categories' items
                                        const previousItemsCount = filteredCategories
                                            .slice(0, categoryIndex)
                                            .reduce((sum, cat) => sum + (cat.items?.length || 0), 0);
                                        const globalItemIndex = filteredCategories.length + previousItemsCount + itemIndex;
                                        const isFocused = focusedIndex === globalItemIndex;

                                        return (
                                            <ListItem
                                                key={item.id}
                                                onClick={itemClickHandlers[itemIndex]}
                                                sx={listItemSxStyle}
                                                role="menuitem"
                                                tabIndex={isFocused ? 0 : -1}
                                                data-index={globalItemIndex}
                                                aria-current={isFocused ? "true" : undefined}
                                            >
                                                <Box sx={itemContentStyle}>
                                                    {Icon}
                                                    {item.name}
                                                </Box>
                                            </ListItem>
                                        );
                                    })}
                                </List>
                            )}
                        </Box>
                    );
                })}

                {/* Render Actions */}
                {filteredActions.map((action, actionIndex) => {
                    // Calculate global index after all categories and their items
                    const previousItemsCount = filteredCategories.reduce((sum, cat) => sum + (cat.items?.length || 0), 0);
                    const globalActionIndex = filteredCategories.length + previousItemsCount + actionIndex;
                    const isFocused = focusedIndex === globalActionIndex;

                    return (
                        <Box key={action.type} mb={1}>
                            <Typography
                                variant="caption"
                                component="div"
                                sx={categoryTypographyStyle}
                                onClick={actionClickHandlers[actionIndex]}
                                tabIndex={isFocused ? 0 : -1}
                                role="menuitem"
                                data-index={globalActionIndex}
                                aria-current={isFocused ? "true" : undefined}
                            >
                                <Box sx={categoryLabelStyle}>
                                    <Icon
                                        info={action.iconInfo}
                                        size={16}
                                    />
                                    {action.label}
                                </Box>
                            </Typography>
                        </Box>
                    );
                })}
            </Box>
        </Popover>
    );
}
