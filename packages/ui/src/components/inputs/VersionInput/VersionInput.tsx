import { calculateVersionsFromString, getMinVersion, meetsMinVersion } from "@local/shared";
import { IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { useField } from "formik";
import { BumpMajorIcon, BumpMinorIcon, BumpModerateIcon } from "icons";
import { useCallback, useMemo, useRef, useState } from "react";
import { TextInput } from "../TextInput/TextInput";
import { VersionInputProps } from "../types";

const COMPONENT_HEIGHT_PX = 56;
const textInputWithSideButtonStyle = {
    "& .MuiInputBase-root": {
        borderRadius: "5px 0 0 5px",
    },
} as const;

export function VersionInput({
    isRequired = false,
    label = "Version",
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
            height: `${textFieldRef.current?.clientHeight ?? COMPONENT_HEIGHT_PX}px)`,
        } as const;
    }, [palette.secondary.main, palette.secondary.contrastText]);

    const bumpModerateIconButtonStyle = useMemo(function bumpModerateIconButtonStyleMemo() {
        return {
            borderRadius: "0",
            background: palette.secondary.main,
            borderRight: `1px solid ${palette.secondary.contrastText}`,
            height: `${textFieldRef.current?.clientHeight ?? COMPONENT_HEIGHT_PX}px)`,
        } as const;
    }, [palette.secondary.main, palette.secondary.contrastText]);

    const bumpMinorIconButtonStyle = useMemo(function bumpMinorIconButtonStyleMemo() {
        return {
            borderRadius: "0 5px 5px 0",
            background: palette.secondary.main,
            height: `${textFieldRef.current?.clientHeight ?? COMPONENT_HEIGHT_PX}px)`,
        } as const;
    }, [palette.secondary.main]);

    return (
        <Stack direction="row" spacing={0}>
            <TextInput
                {...props}
                id="versionLabel"
                name="versionLabel"
                isRequired={isRequired}
                label={label}
                value={internalValue}
                onBlur={handleBlur}
                onChange={handleChange}
                error={meta.touched && !!meta.error}
                helperText={meta.touched && meta.error}
                ref={textFieldRef}
                sx={textInputWithSideButtonStyle}
            />
            <Tooltip placement="top" title="Major bump (increment the first number)">
                <IconButton
                    aria-label='major-bump'
                    onClick={bumpMajor}
                    sx={bumpMajorIconButtonStyle}>
                    <BumpMajorIcon fill="white" />
                </IconButton>
            </Tooltip>
            <Tooltip placement="top" title="Moderate bump (increment the middle number)">
                <IconButton
                    aria-label='moderate-bump'
                    onClick={bumpModerate}
                    sx={bumpModerateIconButtonStyle}>
                    <BumpModerateIcon fill="white" />
                </IconButton>
            </Tooltip>
            <Tooltip placement="top" title="Minor bump (increment the last number)">
                <IconButton
                    aria-label='minor-bump'
                    onClick={bumpMinor}
                    sx={bumpMinorIconButtonStyle}>
                    <BumpMinorIcon fill="white" />
                </IconButton>
            </Tooltip>
        </Stack>
    );
}
