import { FormControl, FormControlProps, FormHelperText, IconButton, InputAdornment, InputLabel, LinearProgress, OutlinedInput, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../../icons/Icons.js";
import { PasswordTextInputProps } from "../types.js";

type PasswordStrengthProps = {
    label: string;
    primary: string;
    secondary: string;
    score: number;
};

const passwordStartAdornment = (
    <InputAdornment position="start">
        <IconCommon
            decorative
            name="Lock"
        />
    </InputAdornment>
);

export function PasswordTextInput({
    autoComplete = "current-password",
    autoFocus = false,
    fullWidth = true,
    label,
    name,
    ...props
}: PasswordTextInputProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [field, meta] = useField(name);

    const [showPassword, setShowPassword] = useState<boolean>(false);

    const handleClickShowPassword = useCallback(() => {
        setShowPassword(!showPassword);
    }, [showPassword]);

    const getPasswordStrengthProps = useCallback(async (password: string) => {
        const defaultProps = { label: "N/A", primary: palette.info.main, secondary: palette.info.light };
        if (!password) {
            return { ...defaultProps, score: 0 };
        }
        const zxcvbn = (await import("zxcvbn")).default;
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
                return { ...defaultProps, score };
        }
    }, [palette]);

    const [strengthProps, setStrengthProps] = useState<PasswordStrengthProps>({ label: "N/A", primary: palette.info.main, secondary: palette.info.light, score: 0 });
    useEffect(() => {
        getPasswordStrengthProps(field.value).then(setStrengthProps);
    }, [field.value, getPasswordStrengthProps]);

    return (
        <FormControl fullWidth={fullWidth} variant="outlined" {...props as FormControlProps}>
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
                startAdornment={passwordStartAdornment}
                endAdornment={
                    <InputAdornment position="end">
                        <IconButton
                            aria-label="toggle password visibility"
                            onClick={handleClickShowPassword}
                            edge="end"
                            sx={{
                                "&:focus": {
                                    border: `2px solid ${palette.background.textPrimary}`,
                                },
                                borderRadius: "2px",
                            }}
                        >
                            <IconCommon
                                decorative
                                fill={palette.background.textSecondary}
                                name={showPassword ? "Invisible" : "Visible"}
                            />
                        </IconButton>
                    </InputAdornment>
                }
                label={label ?? t("Password")}
                placeholder={t("PasswordPlaceholder")}
                sx={{
                    borderRadius: "24px",
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
}
