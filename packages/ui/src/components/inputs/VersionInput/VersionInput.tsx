import { calculateVersionsFromString, getMinVersion, meetsMinVersion } from "@local/shared";
import { IconButton, InputAdornment, Tooltip, styled, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useMemo, useRef, useState } from "react";
import { IconCommon } from "../../../icons/Icons.js";
import { TextInput } from "../TextInput/TextInput.js";
import { VersionInputProps } from "../types.js";

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

export function VersionInput({
    isRequired = false,
    name = "versionLabel",
    versions,
    ...props
}: VersionInputProps) {
    const { palette } = useTheme();

    const textFieldRef = useRef<HTMLDivElement | null>(null);

    const [field, meta, helpers] = useField(name);
    const [internalValue, setInternalValue] = useState<string>(field.value);
    const handleChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        const newValue = event.target.value;
        setInternalValue(newValue);
        // If value is a valid version (e.g. 1.0.0, 1.0, 1) and is at least the minimum value, then call onChange
        if (newValue.match(/^[0-9]+(\.[0-9]+){0,2}$/)) {
            if (meetsMinVersion(newValue, getMinVersion(versions))) {
                helpers.setValue(newValue);
            }
        }
    }, [helpers, versions]);

    // Calculate major, moderate, and minor versions. 
    // Ex: 1.2.3 => major = 1, moderate = 2, minor = 3
    // Ex: 1 => major = 1, moderate = 0, minor = 0
    // Ex: 1.2 => major = 1, moderate = 2, minor = 0
    // Ex: asdfasdf (or any other invalid number) => major = minMajor, moderate = minModerate, minor = minMinor
    const { major, moderate, minor } = useMemo(() => calculateVersionsFromString(internalValue ?? ""), [internalValue]);

    const bumpMajor = useCallback(() => {
        const changedVersion = `${major + 1}.${moderate}.${minor}`;
        setInternalValue(changedVersion);
        helpers.setValue(changedVersion);
    }, [major, moderate, minor, helpers]);

    const bumpModerate = useCallback(() => {
        const changedVersion = `${major}.${moderate + 1}.${minor}`;
        setInternalValue(changedVersion);
        helpers.setValue(changedVersion);
    }, [major, moderate, minor, helpers]);

    const bumpMinor = useCallback(() => {
        const changedVersion = `${major}.${moderate}.${minor + 1}`;
        setInternalValue(changedVersion);
        helpers.setValue(changedVersion);
    }, [major, moderate, minor, helpers]);

    /**
     * On blur, update value
     */
    const handleBlur = useCallback(function handleBlurCallback(event: React.FocusEvent<HTMLInputElement>) {
        const changedVersion = `${major}.${moderate}.${minor}`;
        helpers.setValue(changedVersion);
        field.onBlur(event);
    }, [major, moderate, minor, helpers, field]);

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
        ...props.InputProps,
        startAdornment: (
            <InputAdornment position="start">
                v
            </InputAdornment>
        ),
    }), [props.InputProps]);

    // Style for the button container
    const buttonContainerStyle = useMemo(() => ({
        display: "flex",
        alignSelf: "stretch",
    }), []);

    return (
        <Outer>
            <TextInput
                {...props}
                id="versionLabel"
                name="versionLabel"
                isRequired={isRequired}
                value={internalValue}
                onBlur={handleBlur}
                onChange={handleChange}
                error={meta.touched && !!meta.error}
                helperText={meta.touched && meta.error}
                ref={textFieldRef}
                sx={textInputWithSideButtonStyle}
                InputProps={textInputProps}
                variant="outlined"
            />
            {/* Wrap buttons in a stretching container */}
            <div style={buttonContainerStyle}>{/* Use memoized style */}
                <Tooltip placement="top" title="Major bump (increment the first number)">
                    <IconButton
                        onClick={bumpMajor}
                        sx={bumpMajorIconButtonStyle}>
                        <IconCommon decorative name="BumpMajor" />
                    </IconButton>
                </Tooltip>
                <Tooltip placement="top" title="Moderate bump (increment the middle number)">
                    <IconButton
                        onClick={bumpModerate}
                        sx={bumpModerateIconButtonStyle}>
                        <IconCommon decorative name="BumpModerate" />
                    </IconButton>
                </Tooltip>
                <Tooltip placement="top" title="Minor bump (increment the last number)">
                    <IconButton
                        onClick={bumpMinor}
                        sx={bumpMinorIconButtonStyle}>
                        <IconCommon decorative name="BumpMinor" />
                    </IconButton>
                </Tooltip>
            </div>
        </Outer>
    );
}
