import { FormControl, FormHelperText, IconButton, InputAdornment, InputLabel, LinearProgress, OutlinedInput, useTheme } from "@mui/material";
import { useField } from "formik";
import { InvisibleIcon, LockIcon, VisibleIcon } from "icons";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import zxcvbn from "zxcvbn";
import { PasswordTextInputProps } from "../types";

export const PasswordTextInput = ({
    autoComplete = "current-password",
    autoFocus = false,
    fullWidth = true,
    label,
    name,
    ...props
}: PasswordTextInputProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [field, meta] = useField(name);

    const [showPassword, setShowPassword] = useState<boolean>(false);

    const handleClickShowPassword = useCallback(() => {
        setShowPassword(!showPassword);
    }, [showPassword]);

    const getPasswordStrengthProps = useCallback((password: string) => {
        const result = zxcvbn(password);
        const score = result.score;
        switch (score) {
            case 0:
            case 1:
                return { label: "Weak", primary: palette.error.main, secondary: palette.error.light, score };
            case 2:
                return { label: "Moderate", primary: palette.warning.main, secondary: palette.warning.light, score };
            case 3:
                return { label: "Strong", primary: palette.success.main, secondary: palette.success.light, score };
            case 4:
                return { label: "Very Strong", primary: palette.success.dark, secondary: palette.success.light, score };
            default:
                return { label: "N/A", primary: palette.info.main, secondary: palette.info.light, score };
        }
    }, [palette]);
    const strengthProps = getPasswordStrengthProps(field.value);

    return (
        <FormControl fullWidth={fullWidth} variant="outlined" {...props as any}>
            <InputLabel htmlFor={name}>{label ?? "Password"}</InputLabel>
            <OutlinedInput
                id={name}
                name={name}
                type={showPassword ? "text" : "password"}
                value={field.value}
                onBlur={field.onBlur}
                onChange={field.onChange}
                autoComplete={autoComplete}
                autoFocus={autoFocus}
                error={meta.touched && !!meta.error}
                startAdornment={
                    <InputAdornment position="start">
                        <LockIcon />
                    </InputAdornment>
                }
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
                label={label ?? t("Password")}
                placeholder={t("PasswordPlaceholder")}
                sx={{
                    borderRadius: autoComplete === "new-password" && field.value.length > 0 ? "4px 4px 0 0" : "4px",
                }}
            />
            {
                autoComplete === "new-password" && (
                    <LinearProgress
                        value={strengthProps.score * 25}  // Convert score to percentage
                        variant="determinate"
                        sx={{
                            marginTop: 0,
                            height: "6px",
                            borderRadius: "0 0 4px 4px",
                            backgroundColor: field.value.length === 0 ? "transparent" : strengthProps.secondary,
                            "& .MuiLinearProgress-bar": {
                                backgroundColor: strengthProps.primary,
                            },
                        }}
                    />
                )
            }
            <FormHelperText id="adornment-password-error-text" sx={{ color: palette.error.main }}>
                {meta.touched && meta.error}
            </FormHelperText>
        </FormControl>
    );
};