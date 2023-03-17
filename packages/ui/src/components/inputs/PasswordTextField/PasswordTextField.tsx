import { FormControl, FormHelperText, IconButton, InputAdornment, InputLabel, OutlinedInput, useTheme } from '@mui/material';
import { InvisibleIcon, VisibleIcon } from "@shared/icons";
import { useField } from "formik";
import { useCallback, useState } from "react";
import { PasswordTextFieldProps } from "../types";

export const PasswordTextField = ({
    autoComplete = 'current-password',
    autoFocus = false,
    fullWidth = true,
    label,
    name,
    ...props
}: PasswordTextFieldProps) => {
    const { palette } = useTheme();
    const [field, meta] = useField(name);

    const [showPassword, setShowPassword] = useState<boolean>(false);

    const handleClickShowPassword = useCallback(() => {
        setShowPassword(!showPassword);
    }, [showPassword]);

    return (
        <FormControl fullWidth={fullWidth} variant="outlined" {...props as any}>
            <InputLabel htmlFor={name}>{label ?? 'Password'}</InputLabel>
            <OutlinedInput
                id={name}
                name={name}
                type={showPassword ? 'text' : 'password'}
                value={field.value}
                onBlur={field.onBlur}
                onChange={field.onChange}
                autoComplete={autoComplete}
                autoFocus={autoFocus}
                error={meta.touched && !!meta.error}
                endAdornment={
                    <InputAdornment position="end">
                        <IconButton
                            aria-label="toggle password visibility"
                            onClick={handleClickShowPassword}
                            edge="end"
                        >
                            {
                                showPassword ?
                                    <InvisibleIcon fill={palette.background.textSecondary} /> :
                                    <VisibleIcon fill={palette.background.textSecondary} />
                            }
                        </IconButton>
                    </InputAdornment>
                }
                label={label ?? 'Password'}
            />
            <FormHelperText id="adornment-password-error-text" sx={{ color: palette.error.main }}>{meta.touched && meta.error}</FormHelperText>
        </FormControl>
    )
}