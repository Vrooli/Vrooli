import MenuItem from "@mui/material/MenuItem";
import Select from "@mui/material/Select";
import { useTheme } from "@mui/material";
import { exists } from "@vrooli/shared";
import { useField } from "formik";
import { useCallback, useMemo, useState, useRef } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../../icons/Icons.js";
import { InputContainer } from "../InputContainer/index.js";
import { type SelectorBaseProps, type SelectorProps } from "../types.js";

export function SelectorBase<T extends string | number | { [x: string]: unknown }>({
    addOption,
    autoFocus = false,
    options,
    error,
    getDisplayIcon,
    getOptionDescription,
    getOptionIcon,
    getOptionLabel,
    helperText,
    fullWidth = false,
    inputAriaLabel = "select-label",
    isRequired,
    onBlur,
    onChange,
    name,
    noneOption = false,
    noneText,
    label,
    disabled = false,
    placeholder,
    size = "md",
    sxs,
    tabIndex,
    value,
    variant = "filled",
}: SelectorBaseProps<T>) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [isFocused, setIsFocused] = useState(false);
    const selectRef = useRef<HTMLDivElement>(null);

    const getOptionStyle = useCallback((label: string) => {
        const isSelected = exists(value) && getOptionLabel(value, t) === label;
        return {
            fontWeight: isSelected ? "500" : "400",
        };
    }, [value, getOptionLabel, t]);

    const getLabelStyle = useCallback(function getLabelStyleCallback(label: string, description: string | null | undefined) {
        return {
            ...getOptionStyle(label),
            fontWeight: description ? "bold" : "400",
        };
    }, [getOptionStyle]);

    const menuItemSx = useMemo(() => ({ whiteSpace: "normal" }), []);

    // Render all labels
    const labels = useMemo(() => options.map((option) => {
        const labelText = getOptionLabel(option, t) ?? "";
        const description = getOptionDescription ? getOptionDescription(option, t) : null;
        const Icon = getOptionIcon ? getOptionIcon(option) : null;
        const labelStyle = getLabelStyle(labelText, description);

        return (
            <MenuItem key={labelText} value={labelText} sx={menuItemSx}>
                {
                    Icon ?
                        typeof Icon === "function" ?
                            <div className="tw-flex tw-items-center tw-justify-center tw-min-w-[32px]">
                                <Icon fill={palette.background.textSecondary} />
                            </div> :
                            Icon :
                        null
                }
                <div className="tw-flex tw-flex-col">
                    <span style={labelStyle}>{labelText}</span>
                    {description && <span style={getOptionStyle(labelText)} className="tw-text-sm tw-text-text-secondary">{description}</span>}
                </div>
            </MenuItem>
        );
    }), [options, getOptionLabel, t, getOptionDescription, getOptionIcon, getLabelStyle, palette.background.textSecondary, getOptionStyle, menuItemSx]);

    // Find option from label
    const findOption = useCallback(function findOptionCallback(label: string) {
        return options.find((option) => getOptionLabel(option, t) === label);
    }, [options, getOptionLabel, t]);

    const renderValue = useCallback(function renderValueCallback(value: string) {
        // If no value is selected, show placeholder (either provided or default)
        if (!value) {
            const placeholderText = placeholder || `${t("Select")}...`;
            return (
                <span style={{ 
                    color: palette.text.secondary, 
                    fontStyle: "italic",
                    opacity: 0.7,
                }}>
                    {placeholderText}
                </span>
            );
        }
        
        const option = findOption(value as string);
        if (!exists(option)) return null;
        const labelText = getOptionLabel(option, t) ?? "";
        // Icon function can be overridden by getDisplayIcon
        const Icon = typeof getDisplayIcon === "function"
            ? getDisplayIcon(option) :
            typeof getOptionIcon === "function"
                ? getOptionIcon(option) :
                null;
        return (
            <div className="tw-flex tw-items-center">
                {
                    Icon ?
                        typeof Icon === "function" ?
                            <div className="tw-flex tw-items-center tw-justify-center tw-min-w-[32px]">
                                <Icon fill={palette.background.textSecondary} />
                            </div> :
                            Icon :
                        null
                }
                <span style={getOptionStyle(labelText)}>{labelText}</span>
                {/* Note that we omit the description */}
            </div>
        );
    }, [findOption, getDisplayIcon, getOptionIcon, getOptionLabel, getOptionStyle, palette.background.textSecondary, palette.text.secondary, placeholder, t]);

    const handleChange = useCallback((e) => { 
        const found = findOption(e.target.value as string);
        if (found !== undefined) onChange(found as T); 
    }, [findOption, onChange]);

    const selectStyle = useMemo(function selectStyle() {
        return {
            width: "100%",
            height: "100%",
            "& .MuiSelect-select": {
                padding: 0,
                paddingRight: "32px", // Space for dropdown icon
                display: "flex",
                alignItems: "center",
                backgroundColor: "transparent",
                border: "none",
            },
            "& .MuiOutlinedInput-notchedOutline": {
                border: "none",
            },
            "& fieldset": {
                border: "none",
            },
            "& .MuiSelect-icon": {
                right: "8px",
                color: palette.text.secondary,
            },
            ...sxs?.root,
        };
    }, [palette.text.secondary, sxs]);

    const handleFocus = useCallback((e: React.FocusEvent<HTMLDivElement>) => {
        setIsFocused(true);
    }, []);

    const handleBlur = useCallback((e: React.FocusEvent<HTMLDivElement>) => {
        setIsFocused(false);
        onBlur?.(e);
    }, [onBlur]);

    const handleContainerClick = useCallback(() => {
        // Focus the select when clicking the container
        const selectElement = selectRef.current?.querySelector("input, [role=\"button\"]") as HTMLElement;
        selectElement?.focus();
        selectElement?.click();
    }, []);

    return (
        <InputContainer
            variant={variant}
            size={size}
            error={error}
            disabled={disabled}
            fullWidth={fullWidth}
            focused={isFocused}
            onClick={handleContainerClick}
            onFocus={handleFocus}
            onBlur={handleBlur}
            label={label}
            isRequired={isRequired}
            helperText={helperText}
            htmlFor={name}
        >
            <Select
                ref={selectRef}
                aria-describedby={`helper-text-${name}`}
                autoFocus={autoFocus}
                disabled={disabled}
                id={name}
                labelId={inputAriaLabel}
                name={name}
                onChange={handleChange}
                onFocus={handleFocus}
                onBlur={handleBlur}
                required={isRequired}
                value={exists(value) ? (getOptionLabel(value, t) ?? undefined) : ""}
                variant="standard"
                sx={selectStyle}
                tabIndex={tabIndex}
                renderValue={renderValue}
                displayEmpty
                disableUnderline
            >
                {
                    noneOption ? (
                        <MenuItem value="">
                            <em>{noneText || t("None")}</em>
                        </MenuItem>
                    ) : null
                }
                {
                    labels
                }
                {
                    addOption ? (
                        <MenuItem
                            aria-label={addOption.label}
                            value="addOption"
                            onClick={addOption.onSelect}
                        >
                            <IconCommon
                                decorative
                                fill="inherit"
                                name="Add"
                                className="tw-mr-2"
                            />
                            <em>{addOption.label}</em>
                        </MenuItem>
                    ) : null
                }
            </Select>
        </InputContainer>
    );
}

export function Selector<T extends string | number | { [x: string]: unknown }>({
    name,
    onChange,
    ...props
}: SelectorProps<T>) {
    const [field, meta, helpers] = useField(name);

    const handleChange = useCallback(function handleChangeCallback(newValue: T) {
        if (onChange) onChange(newValue);
        helpers.setValue(newValue);
    }, [onChange, helpers]);

    return (
        <SelectorBase
            {...props}
            name={name}
            value={field.value}
            error={meta.touched && !!meta.error}
            helperText={meta.touched && meta.error}
            onBlur={field.onBlur}
            onChange={handleChange}
        />
    );
}
