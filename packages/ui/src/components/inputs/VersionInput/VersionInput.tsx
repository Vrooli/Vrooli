import { Stack, TextField, Tooltip, useTheme } from '@mui/material';
import { BumpMajorIcon, BumpMinorIcon, BumpModerateIcon } from "@shared/icons";
import { calculateVersionsFromString, meetsMinVersion } from "@shared/validation";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { useField } from 'formik';
import { useCallback, useMemo, useRef, useState } from "react";
import { getMinimumVersion } from "utils/shape/general";
import { VersionInputProps } from "../types";

export const VersionInput = ({
    autoFocus = false,
    fullWidth = true,
    label = 'Version',
    name = 'versionLabel',
    versions,
    ...props
}: VersionInputProps) => {
    const { palette } = useTheme();

    const textFieldRef = useRef<HTMLDivElement | null>(null);

    const [field, meta, helpers] = useField(name);
    const [internalValue, setInternalValue] = useState<string>(field.value);
    const handleChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        const newValue = event.target.value;
        setInternalValue(newValue);
        // If value is a valid version (e.g. 1.0.0, 1.0, 1) and is at least the minimum value, then call onChange
        if (newValue.match(/^[0-9]+(\.[0-9]+){0,2}$/)) {
            if (meetsMinVersion(newValue, getMinimumVersion(versions))) {
                helpers.setValue(newValue);
            }
        }
    }, [helpers, versions]);

    // Calculate major, moderate, and minor versions. 
    // Ex: 1.2.3 => major = 1, moderate = 2, minor = 3
    // Ex: 1 => major = 1, moderate = 0, minor = 0
    // Ex: 1.2 => major = 1, moderate = 2, minor = 0
    // Ex: asdfasdf (or any other invalid number) => major = minMajor, moderate = minModerate, minor = minMinor
    const { major, moderate, minor } = useMemo(() => calculateVersionsFromString(internalValue ?? ''), [internalValue]);

    const bumpMajor = useCallback(() => {
        const changedVersion = `${major + 1}.${moderate}.${minor}`;
        helpers.setValue(changedVersion);
    }, [major, moderate, minor, helpers]);

    const bumpModerate = useCallback(() => {
        const changedVersion = `${major}.${moderate + 1}.${minor}`;
        helpers.setValue(changedVersion);
    }, [major, moderate, minor, helpers]);

    const bumpMinor = useCallback(() => {
        const changedVersion = `${major}.${moderate}.${minor + 1}`;
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
            <TextField
                autoFocus={autoFocus}
                fullWidth
                id="versionLabel"
                name="versionLabel"
                label={label}
                value={internalValue}
                onBlur={handleBlur}
                onChange={handleChange}
                error={meta.touched && !!meta.error}
                helperText={meta.touched && meta.error}
                ref={textFieldRef}
                sx={{
                    '& .MuiInputBase-root': {
                        borderRadius: '5px 0 0 5px',
                    }
                }}
            />
            <Tooltip placement="top" title="Major bump (increment the first number)">
                <ColorIconButton
                    aria-label='major-bump'
                    background={palette.secondary.main}
                    onClick={bumpMajor}
                    sx={{
                        borderRadius: '0',
                        borderRight: `1px solid ${palette.secondary.contrastText}`,
                        height: `${textFieldRef.current?.clientHeight ?? 56}px)`,
                    }}>
                    <BumpMajorIcon />
                </ColorIconButton>
            </Tooltip>
            <Tooltip placement="top" title="Moderate bump (increment the middle number)">
                <ColorIconButton
                    aria-label='moderate-bump'
                    background={palette.secondary.main}
                    onClick={bumpModerate}
                    sx={{
                        borderRadius: '0',
                        borderRight: `1px solid ${palette.secondary.contrastText}`,
                        height: `${textFieldRef.current?.clientHeight ?? 56}px)`,
                    }}>
                    <BumpModerateIcon />
                </ColorIconButton>
            </Tooltip>
            <Tooltip placement="top" title="Minor bump (increment the last number)">
                <ColorIconButton
                    aria-label='minor-bump'
                    background={palette.secondary.main}
                    onClick={bumpMinor}
                    sx={{
                        borderRadius: '0 5px 5px 0',
                        height: `${textFieldRef.current?.clientHeight ?? 56}px)`,
                    }}>
                    <BumpMinorIcon />
                </ColorIconButton>
            </Tooltip>
        </Stack>
    )
}