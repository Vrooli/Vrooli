import { useCallback, useEffect, useMemo, useState } from "react";
import { Stack, TextField, Tooltip, useTheme } from '@mui/material';
import { VersionInputProps } from "../types";
import { BumpMajorIcon, BumpMinorIcon, BumpModerateIcon } from "@shared/icons";
import { calculateVersionsFromString, meetsMinVersion } from "@shared/validation";
import { ColorIconButton } from "components/buttons";
import { VersionInfo } from "types"; 
import { getMinimumVersion } from "utils";

export const VersionInput = ({
    autoFocus = false,
    error = false,
    fullWidth = true,
    helperText = undefined,
    id = 'version',
    label = 'Version',
    name = 'version',
    onBlur = () => { },
    onChange,
    versionInfo,
    versions,
    ...props
}: VersionInputProps) => {
    const { palette } = useTheme();

    const [internalValue, setInternalValue] = useState<Partial<VersionInfo>>(versionInfo);
    console.log('versioninputtttt', versionInfo, internalValue)
    useEffect(() => {
        setInternalValue(versionInfo);
    }, [versionInfo]);
    const handleChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        const newValue = event.target.value;
        setInternalValue({
            ...internalValue,
            versionLabel: newValue,
        });
        // If value is a valid version (e.g. 1.0.0, 1.0, 1) and is at least the minimum value, then call onChange
        if (newValue.match(/^[0-9]+(\.[0-9]+){0,2}$/)) {
            if (meetsMinVersion(newValue, getMinimumVersion(versions))) {
                onChange({
                    ...internalValue,
                    versionIndex: versions.length,
                    versionLabel: newValue,
                });
            }
        }
    }, [internalValue, onChange, versions]);

    // Calculate major, moderate, and minor versions. 
    // Ex: 1.2.3 => major = 1, moderate = 2, minor = 3
    // Ex: 1 => major = 1, moderate = 0, minor = 0
    // Ex: 1.2 => major = 1, moderate = 2, minor = 0
    // Ex: asdfasdf (or any other invalid number) => major = minMajor, moderate = minModerate, minor = minMinor
    const { major, moderate, minor } = useMemo(() => calculateVersionsFromString(internalValue.versionLabel ?? ''), [internalValue]);

    const bumpMajor = useCallback(() => {
        const changedVersion = `${major + 1}.${moderate}.${minor}`
        onChange({ 
            ...internalValue, 
            versionIndex: versions.length,
            versionLabel: changedVersion 
        });
    }, [major, moderate, minor, onChange, internalValue, versions.length]);

    const bumpModerate = useCallback(() => {
        const changedVersion = `${major}.${moderate + 1}.${minor}`
        onChange({
            ...internalValue,
            versionIndex: versions.length,
            versionLabel: changedVersion
        });
    }, [major, moderate, minor, onChange, internalValue, versions.length]);

    const bumpMinor = useCallback(() => {
        const changedVersion = `${major}.${moderate}.${minor + 1}`
        onChange({
            ...internalValue,
            versionIndex: versions.length,
            versionLabel: changedVersion
        });
    }, [major, moderate, minor, onChange, internalValue, versions.length]);

    /**
     * On blur, update value
     */
    const handleBlur = useCallback((ev: any) => {
        const changedVersion = `${major}.${moderate}.${minor}`
        onChange({
            ...internalValue,
            versionIndex: versions.length,
            versionLabel: changedVersion
        });
        if (onBlur) { onBlur(ev); }
    }, [major, moderate, minor, onChange, internalValue, versions.length, onBlur]);

    return (
        <Stack direction="row" spacing={0}>
            <TextField
                autoFocus={autoFocus}
                fullWidth
                id={id}
                name={name}
                label={label}
                value={internalValue.versionLabel}
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