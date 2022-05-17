import { useCallback, useState } from "react";
import { FormControl, FormHelperText, IconButton, InputAdornment, InputLabel, OutlinedInput, useTheme } from '@mui/material';
import {
    Visibility as VisibilityIcon,
    VisibilityOff as VisibilityOffIcon,
} from '@mui/icons-material';
import { PasswordTextFieldProps } from "../types";

export const PasswordTextField = ({
    autoComplete = 'current-password',
    autoFocus = false,
    error = false,
    fullWidth = true,
    helperText = undefined,
    id = 'password',
    label,
    name = 'password',
    onBlur = () => { },
    onChange,
    value,
    ...props
}: PasswordTextFieldProps) => {
    const { palette } = useTheme();

    const [showPassword, setShowPassword] = useState<boolean>(false);

    const handleClickShowPassword = useCallback(() => {
        setShowPassword(!showPassword);
    }, [showPassword]);

    return (
        <FormControl fullWidth={fullWidth} variant="outlined" {...props}>
            <InputLabel htmlFor={id}>{label ?? 'Password'}</InputLabel>
            <OutlinedInput
                id={id}
                name={name}
                type={showPassword ? 'text' : 'password'}
                value={value}
                onBlur={onBlur}
                onChange={onChange}
                autoComplete={autoComplete}
                autoFocus={autoFocus}
                error={error}
                endAdornment={
                    <InputAdornment position="end">
                        <IconButton
                            aria-label="toggle password visibility"
                            onClick={handleClickShowPassword}
                            edge="end"
                        >
                            {showPassword ? <VisibilityOffIcon /> : <VisibilityIcon />}
                        </IconButton>
                    </InputAdornment>
                }
                label={label ?? 'Password'}
            />
            <FormHelperText id="adornment-password-error-text" sx={{ color: palette.error.main }}>{helperText}</FormHelperText>
        </FormControl>
    )
}