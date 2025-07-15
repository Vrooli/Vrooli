import { IconButton } from "../../buttons/IconButton.js";
// eslint-disable-next-line import/extensions
import InputAdornment from "@mui/material/InputAdornment";
import { Tooltip } from "../../Tooltip/Tooltip.js";
import { styled, useTheme } from "@mui/material";
import { calculateVersionsFromString, getMinVersion, meetsMinVersion } from "@vrooli/shared";
import { useField } from "formik";
import React, { forwardRef, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { IconCommon } from "../../../icons/Icons.js";
import { TextInput, TextInputBase } from "../TextInput/TextInput.js";
import { type VersionInputProps, type VersionInputBaseProps, type VersionInputFormikProps } from "../types.js";

const textInputWithSideButtonStyle = {
    "& .MuiInputBase-root": {
        backgroundColor: "transparent",
    },
    "& .MuiOutlinedInput-notchedOutline": {
        border: "none",
    },
} as const;

const Outer = styled("div")(({ theme }) => ({
    backgroundColor: theme.palette.background.paper,
    borderRadius: theme.spacing(3),
    position: "relative",
    display: "flex",
    alignItems: "stretch",
    overflow: "overlay",
}));

/**
 * Base version input component without Formik integration.
 * This is the pure visual component that handles all styling and interaction logic.
 */
export const VersionInputBase = forwardRef<HTMLDivElement, VersionInputBaseProps>(({
    value,
    onChange,
    onBlur,
    error,
    helperText,
    label,
    name = "versionLabel",
    isRequired = false,
    disabled,
    versions,
    InputProps,
    sx,
    ...props
}, ref) => {
    const { palette } = useTheme();

    const textFieldRef = useRef<HTMLDivElement | null>(null);

    const [internalValue, setInternalValue] = useState<string>(value);
    
    // Sync internal value with external value
    useEffect(() => {
        setInternalValue(value);
    }, [value]);

    const handleChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        const newValue = event.target.value;
        setInternalValue(newValue);
        // If value is a valid version (e.g. 1.0.0, 1.0, 1) and is at least the minimum value, then call onChange
        if (newValue.match(/^[0-9]+(\.[0-9]+){0,2}$/)) {
            if (meetsMinVersion(newValue, getMinVersion(versions))) {
                onChange(newValue);
            }
        }
    }, [onChange, versions]);

    // Calculate major, moderate, and minor versions. 
    // Ex: 1.2.3 => major = 1, moderate = 2, minor = 3
    // Ex: 1 => major = 1, moderate = 0, minor = 0
    // Ex: 1.2 => major = 1, moderate = 2, minor = 0
    // Ex: asdfasdf (or any other invalid number) => major = minMajor, moderate = minModerate, minor = minMinor
    const { major, moderate, minor } = useMemo(() => calculateVersionsFromString(internalValue ?? ""), [internalValue]);

    const bumpMajor = useCallback(() => {
        const changedVersion = `${major + 1}.${moderate}.${minor}`;
        setInternalValue(changedVersion);
        onChange(changedVersion);
    }, [major, moderate, minor, onChange]);

    const bumpModerate = useCallback(() => {
        const changedVersion = `${major}.${moderate + 1}.${minor}`;
        setInternalValue(changedVersion);
        onChange(changedVersion);
    }, [major, moderate, minor, onChange]);

    const bumpMinor = useCallback(() => {
        const changedVersion = `${major}.${moderate}.${minor + 1}`;
        setInternalValue(changedVersion);
        onChange(changedVersion);
    }, [major, moderate, minor, onChange]);

    /**
     * On blur, update value
     */
    const handleBlur = useCallback(function handleBlurCallback(event: React.FocusEvent<HTMLInputElement>) {
        const changedVersion = `${major}.${moderate}.${minor}`;
        onChange(changedVersion);
        onBlur?.(event);
    }, [major, moderate, minor, onChange, onBlur]);

    const bumpMajorIconButtonStyle = useMemo(function bumpMajorIconButtonStyleMemo() {
        return {
            borderRadius: "0",
            background: palette.secondary.main,
            borderRight: `1px solid ${palette.secondary.contrastText}`,
            height: "100%",
            alignSelf: "stretch",
        } as const;
    }, [palette.secondary.main, palette.secondary.contrastText]);

    const bumpModerateIconButtonStyle = useMemo(function bumpModerateIconButtonStyleMemo() {
        return {
            borderRadius: "0",
            background: palette.secondary.main,
            borderRight: `1px solid ${palette.secondary.contrastText}`,
            height: "100%",
            alignSelf: "stretch",
        } as const;
    }, [palette.secondary.main, palette.secondary.contrastText]);

    const bumpMinorIconButtonStyle = useMemo(function bumpMinorIconButtonStyleMemo() {
        return {
            borderRadius: "0",
            background: palette.secondary.main,
            height: "100%",
            alignSelf: "stretch",
        } as const;
    }, [palette.secondary.main]);

    // Memoize InputProps to fix linter warning
    const textInputProps = useMemo(() => ({
        ...InputProps,
        startAdornment: (
            <InputAdornment position="start">
                v
            </InputAdornment>
        ),
    }), [InputProps]);

    // Style for the button container
    const buttonContainerStyle = useMemo(() => ({
        display: "flex",
        alignSelf: "stretch",
    }), []);

    return (
        <Outer data-testid="version-input" ref={ref}>
            <TextInputBase
                {...props}
                id={name}
                name={name}
                label={label}
                isRequired={isRequired}
                value={internalValue}
                onBlur={handleBlur}
                onChange={handleChange}
                error={error}
                helperText={helperText}
                disabled={disabled}
                ref={textFieldRef}
                sx={{ ...textInputWithSideButtonStyle, ...sx }}
                InputProps={textInputProps}
                variant="outlined"
            />
            {/* Wrap buttons in a stretching container */}
            <div style={buttonContainerStyle} data-testid="version-bump-buttons">{/* Use memoized style */}
                <Tooltip placement="top" title="Major bump (increment the first number)">
                    <IconButton
                        onClick={bumpMajor}
                        variant="transparent"
                        sx={bumpMajorIconButtonStyle}
                        aria-label="Major bump (increment the first number)"
                        data-testid="major-bump-button"
                        disabled={disabled}>
                        <IconCommon decorative name="BumpMajor" />
                    </IconButton>
                </Tooltip>
                <Tooltip placement="top" title="Moderate bump (increment the middle number)">
                    <IconButton
                        onClick={bumpModerate}
                        variant="transparent"
                        sx={bumpModerateIconButtonStyle}
                        aria-label="Moderate bump (increment the middle number)"
                        data-testid="moderate-bump-button"
                        disabled={disabled}>
                        <IconCommon decorative name="BumpModerate" />
                    </IconButton>
                </Tooltip>
                <Tooltip placement="top" title="Minor bump (increment the last number)">
                    <IconButton
                        onClick={bumpMinor}
                        variant="transparent"
                        sx={bumpMinorIconButtonStyle}
                        aria-label="Minor bump (increment the last number)"
                        data-testid="minor-bump-button"
                        disabled={disabled}>
                        <IconCommon decorative name="BumpMinor" />
                    </IconButton>
                </Tooltip>
            </div>
        </Outer>
    );
});

VersionInputBase.displayName = "VersionInputBase";

/**
 * Formik-integrated version input component.
 * Automatically connects to Formik context using the field name.
 * 
 * @example
 * ```tsx
 * // Inside a Formik form
 * <VersionInput name="version" label="Version" versions={["1.0.0", "1.1.0"]} />
 * 
 * // With validation
 * <VersionInput 
 *   name="versionLabel" 
 *   label="Version"
 *   versions={existingVersions}
 *   validate={(value) => !value ? "Version is required" : undefined}
 * />
 * ```
 */
export const VersionInput = forwardRef<HTMLDivElement, VersionInputFormikProps>(({
    name = "versionLabel",
    validate,
    ...props
}, ref) => {
    const [field, meta, helpers] = useField({ name, validate });

    const handleChange = useCallback((value: string) => {
        helpers.setValue(value);
    }, [helpers]);

    return (
        <VersionInputBase
            {...props}
            ref={ref}
            name={name}
            value={field.value || ""}
            onChange={handleChange}
            onBlur={field.onBlur}
            error={meta.touched && Boolean(meta.error)}
            helperText={meta.touched && meta.error}
        />
    );
});

VersionInput.displayName = "VersionInput";
