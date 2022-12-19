import { useCallback, useEffect, useMemo, useState } from "react";
import { Stack, TextField, Tooltip, useTheme } from '@mui/material';
import { VersionInputProps } from "../types";
import { BumpMajorIcon, BumpMinorIcon, BumpModerateIcon } from "@shared/icons";
import { calculateVersionsFromString, meetsMinVersion } from "@shared/validation";
import { ColorIconButton } from "components/buttons";

export const VersionInput = ({
    autoFocus = false,
    error = false,
    fullWidth = true,
    helperText = undefined,
    id = 'version',
    label = 'Version',
    minimum = '0.0.1',
    name = 'version',
    onBlur = () => { },
    onChange,
    value,
    ...props
}: VersionInputProps) => {
    const { palette } = useTheme();

    const [internalValue, setInternalValue] = useState<string>(value);
    useEffect(() => {
        setInternalValue(value);
    }, [value]);
    const handleChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        const newValue = event.target.value;
        setInternalValue(newValue);
        // If value is a valid version (e.g. 1.0.0, 1.0, 1) and is at least the minimum value, then call onChange
        if (newValue.match(/^[0-9]+(\.[0-9]+){0,2}$/)) {
            if (meetsMinVersion(newValue, minimum)) {
                onChange(newValue);
            }
        }
    }, [minimum, onChange]);

    // Calculate major, moderate, and minor versions. 
    // Ex: 1.2.3 => major = 1, moderate = 2, minor = 3
    // Ex: 1 => major = 1, moderate = 0, minor = 0
    // Ex: 1.2 => major = 1, moderate = 2, minor = 0
    // Ex: asdfasdf (or any other invalid number) => major = minMajor, moderate = minModerate, minor = minMinor
    const { major, moderate, minor } = useMemo(() => calculateVersionsFromString(internalValue), [internalValue]);

    const bumpMajor = useCallback(() => {
        onChange(`${major + 1}.${moderate}.${minor}`);
    }, [major, moderate, minor, onChange]);

    const bumpModerate = useCallback(() => {
        onChange(`${major}.${moderate + 1}.${minor}`);
    }, [major, moderate, minor, onChange]);

    const bumpMinor = useCallback(() => {
        onChange(`${major}.${moderate}.${minor + 1}`);
    }, [major, moderate, minor, onChange]);

    /**
     * On blur, update value
     */
    const handleBlur = useCallback((ev: any) => {
        onChange(`${major}.${moderate}.${minor}`);
        if (onBlur) { onBlur(ev); }
    }, [major, moderate, minor, onChange, onBlur]);

    return (
        <Stack direction="row" spacing={0}>
            <TextField
                autoFocus={autoFocus}
                fullWidth
                id={id}
                name={name}
                label={label}
                value={internalValue}
                onBlur={handleBlur}
                onChange={handleChange}
                error={error}
                helperText={helperText}
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
                        height: '56px',
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
                        height: '56px',
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
                        height: '56px',
                    }}>
                    <BumpMinorIcon />
                </ColorIconButton>
            </Tooltip>
        </Stack>
    )
}