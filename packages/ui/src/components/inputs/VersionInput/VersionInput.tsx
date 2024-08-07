import { calculateVersionsFromString, getMinVersion, meetsMinVersion } from "@local/shared";
import { IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { useField } from "formik";
import { BumpMajorIcon, BumpMinorIcon, BumpModerateIcon } from "icons";
import { useCallback, useMemo, useRef, useState } from "react";
import { TextInput } from "../TextInput/TextInput";
import { VersionInputProps } from "../types";

const COMPONENT_HEIGHT_PX = 56;

export function VersionInput({
    autoFocus = false,
    fullWidth = true,
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
    console.log("in version input", field.value, name, internalValue);
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
        console.log("in version input bump minor", changedVersion);
        setInternalValue(changedVersion);
        helpers.setValue(changedVersion);
    }, [major, moderate, minor, helpers]);


    /**
     * On blur, update value
     */
    const handleBlur = useCallback((ev: any) => {
        const changedVersion = `${major}.${moderate}.${minor}`;
        helpers.setValue(changedVersion);
        field.onBlur(ev);
    }, [major, moderate, minor, helpers, field]);

    return (
        <Stack direction="row" spacing={0}>
            <TextInput
                {...props}
                autoFocus={autoFocus}
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
                sx={{
                    "& .MuiInputBase-root": {
                        borderRadius: "5px 0 0 5px",
                    },
                }}
            />
            <Tooltip placement="top" title="Major bump (increment the first number)">
                <IconButton
                    aria-label='major-bump'
                    onClick={bumpMajor}
                    sx={{
                        borderRadius: "0",
                        background: palette.secondary.main,
                        borderRight: `1px solid ${palette.secondary.contrastText}`,
                        height: `${textFieldRef.current?.clientHeight ?? COMPONENT_HEIGHT_PX}px)`,
                    }}>
                    <BumpMajorIcon fill="white" />
                </IconButton>
            </Tooltip>
            <Tooltip placement="top" title="Moderate bump (increment the middle number)">
                <IconButton
                    aria-label='moderate-bump'
                    onClick={bumpModerate}
                    sx={{
                        borderRadius: "0",
                        background: palette.secondary.main,
                        borderRight: `1px solid ${palette.secondary.contrastText}`,
                        height: `${textFieldRef.current?.clientHeight ?? COMPONENT_HEIGHT_PX}px)`,
                    }}>
                    <BumpModerateIcon fill="white" />
                </IconButton>
            </Tooltip>
            <Tooltip placement="top" title="Minor bump (increment the last number)">
                <IconButton
                    aria-label='minor-bump'
                    onClick={bumpMinor}
                    sx={{
                        borderRadius: "0 5px 5px 0",
                        background: palette.secondary.main,
                        height: `${textFieldRef.current?.clientHeight ?? COMPONENT_HEIGHT_PX}px)`,
                    }}>
                    <BumpMinorIcon fill="white" />
                </IconButton>
            </Tooltip>
        </Stack>
    );
}
