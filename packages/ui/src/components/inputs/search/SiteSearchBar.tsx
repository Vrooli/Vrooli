import Box from "@mui/material/Box";
import IconButton from "@mui/material/IconButton";
import Input from "@mui/material/Input";
import InputAdornment from "@mui/material/InputAdornment";
import Paper from "@mui/material/Paper";
import TextField from "@mui/material/TextField";
import { useTheme } from "@mui/material";
import { noop } from "@vrooli/shared";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useDebounce } from "../../../hooks/useDebounce.js";
import { IconCommon } from "../../../icons/Icons.js";
import { ELEMENT_CLASSES } from "../../../utils/consts.js";
import { MicrophoneButton } from "../../buttons/MicrophoneButton.js";

const DEFAULT_DEBOUNCE_MS = 200;

type EnterKeyHint = "search" | "enter" | "done" | "go" | "next" | "previous" | "send";

type SearchBarProps = {
    debounce?: number;
    enterKeyHint?: EnterKeyHint;
    id?: string;
    /** 
     * If true, we'll use a div instead of a form for the search bar component.
     * Nested forms are not allowed in HTML
     */
    isNested?: boolean;
    onChange: (value: string) => unknown;
    /**
     * Callback when input loses focus
     */
    onBlur?: () => void;
    /**
     * Callback when input is focused
     */
    onFocus?: () => void;
    /**
     * Callback for handling keyboard events
     */
    onKeyDown?: (event: React.KeyboardEvent) => void;
    placeholder?: string;
    value: string;
}

/**
 * A customized search bar for searching user-generated content, quick actions, and shortcuts.
 * Supports search history and starring items.
 */
export function BasicSearchBar({
    enterKeyHint,
    id = "search-bar",
    isNested,
    placeholder,
    value,
    onChange,
    onBlur,
    onFocus,
    onKeyDown,
    debounce,
}: SearchBarProps) {
    const { t } = useTranslation();
    const { palette, spacing } = useTheme();

    const [internalValue, setInternalValue] = useState<string>(value);
    const [onChangeDebounced] = useDebounce(onChange, debounce ?? DEFAULT_DEBOUNCE_MS);

    useEffect(function syncExternalValueEffect() {
        setInternalValue(value);
    }, [value]);

    const handleInputChange = useCallback(function handleInputChangeCallback(
        event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
    ) {
        const newValue = event.target.value;
        setInternalValue(newValue);
        onChangeDebounced(newValue);
    }, [onChangeDebounced]);

    const handleTranscriptChange = useCallback(
        (transcript: string) => {
            setInternalValue(transcript); // Update the input field
            onChange(transcript); // Notify parent immediately
        },
        [onChange],
    );

    const handleFocus = useCallback(() => {
        if (onFocus) onFocus();
    }, [onFocus]);

    const handleBlur = useCallback(() => {
        if (onBlur) onBlur();
    }, [onBlur]);

    const basicSearchBarSx = useMemo(function basicSearchBarSxMemo() {
        return {
            // Remove default MUI border
            "& .MuiOutlinedInput-root": {
                "& fieldset": { border: "none" }, // Normal state
                "&:hover fieldset": { border: "none" }, // Hover state
                "&.Mui-focused fieldset": { border: "none" }, // Focused state
            },
            // Add static 1px bottom border
            borderBottom: `1px solid ${palette.divider}`, // Consistent color from theme
            // Apply border radius to top corners
            borderTopLeftRadius: spacing(3),
            borderTopRightRadius: spacing(3),
        } as const;
    }, [palette, spacing]);

    const InputProps = useMemo(function InputPropsMemo() {
        return {
            endAdornment: (
                <InputAdornment position="end">
                    <MicrophoneButton onTranscriptChange={handleTranscriptChange} />
                    <IconButton
                        aria-label={t("Search")}
                        edge="end"
                        onClick={noop}
                    >
                        <IconCommon
                            decorative
                            name="Search"
                        />
                    </IconButton>
                </InputAdornment>
            ),
            enterKeyHint: enterKeyHint || "search",
        };
    }, [enterKeyHint, handleTranscriptChange, t]);

    return (
        <Box className={ELEMENT_CLASSES.SearchBar} component={isNested ? "div" : "form"}>
            <TextField
                fullWidth
                id={id}
                InputProps={InputProps}
                onChange={handleInputChange}
                onFocus={handleFocus}
                onBlur={handleBlur}
                onKeyDown={onKeyDown}
                placeholder={placeholder}
                value={internalValue}
                variant="outlined"
                sx={basicSearchBarSx}
            />
        </Box>
    );
}

const inputStyle = {
    ml: 1,
    flex: 1,
    // Remove underline
    "&:before": {
        display: "none",
    },
    "&:after": {
        display: "none",
    },
    "&:hover:before": {
        display: "none",
    },
    // Drop down should be as large as the full width of the screen
    "& .MuiAutocomplete-popper": {
        width: "100vw!important",
        left: "0",
        right: "0",
        // The drop down should be below the search bar
        "& .MuiPaper-root": {
            marginTop: "0",
        },
    },
} as const;
const searchIconStyle = { width: "48px", height: "48px" } as const;

export function PaperSearchBar({
    enterKeyHint,
    id = "search-bar",
    isNested,
    placeholder,
    value,
    onChange,
    onBlur,
    onFocus,
    onKeyDown,
    debounce,
}: SearchBarProps) {
    const { t } = useTranslation();
    const { palette, spacing } = useTheme();

    const [internalValue, setInternalValue] = useState<string>(value);
    const [onChangeDebounced] = useDebounce(onChange, debounce ?? DEFAULT_DEBOUNCE_MS);

    useEffect(function syncExternalValueEffect() {
        setInternalValue(value);
    }, [value]);

    const handleInputChange = useCallback(function handleInputChangeCallback(
        event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
    ) {
        const newValue = event.target.value;
        setInternalValue(newValue);
        onChangeDebounced(newValue);
    }, [onChangeDebounced]);

    const handleTranscriptChange = useCallback(
        (transcript: string) => {
            setInternalValue(transcript); // Update the input field
            onChange(transcript); // Notify parent immediately
        },
        [onChange],
    );

    const handleFocus = useCallback(() => {
        if (onFocus) onFocus();
    }, [onFocus]);

    const handleBlur = useCallback(() => {
        if (onBlur) onBlur();
    }, [onBlur]);

    const paperSx = useMemo(() => ({
        p: "2px 4px",
        display: "flex",
        alignItems: "center",
        borderRadius: spacing(2),
        height: "48px",
    }), [spacing]);

    const InputProps = useMemo(function InputPropsMemo() {
        return {
            endAdornment: (
                <InputAdornment position="end">
                    <MicrophoneButton onTranscriptChange={handleTranscriptChange} />
                    <IconButton
                        aria-label={t("Search")}
                        edge="end"
                        onClick={noop}
                    >
                        <IconCommon
                            decorative
                            name="Search"
                        />
                    </IconButton>
                </InputAdornment>
            ),
            enterKeyHint: enterKeyHint || "search",
        };
    }, [enterKeyHint, handleTranscriptChange, t]);

    return (
        <Paper className={ELEMENT_CLASSES.SearchBar} component={isNested ? "div" : "form"} sx={paperSx}>
            <Input
                onChange={handleInputChange}
                onFocus={handleFocus}
                onBlur={handleBlur}
                onKeyDown={onKeyDown}
                placeholder={placeholder}
                inputProps={InputProps}
                sx={inputStyle}
                value={internalValue}
                disableUnderline={true}
            />
            <MicrophoneButton
                fill={palette.background.textSecondary}
                onTranscriptChange={handleTranscriptChange}
            />
            <IconButton
                sx={searchIconStyle}
            >
                <IconCommon
                    decorative
                    fill={palette.background.textSecondary}
                    name="Search"
                />
            </IconButton>
        </Paper>
    );
}

export function SiteSearchBarPaper() {
    return null;
}
