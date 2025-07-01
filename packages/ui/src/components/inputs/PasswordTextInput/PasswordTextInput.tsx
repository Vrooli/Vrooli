import { useField } from "formik";
import React, { forwardRef, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../../icons/Icons.js";
import { IconButton } from "../../buttons/IconButton.js";
import { TextInput, TextInputBase } from "../TextInput/TextInput.js";
import { type PasswordTextInputProps, type PasswordTextInputBaseProps, type PasswordTextInputFormikProps } from "../types.js";

// Constants for password strength scoring
const PASSWORD_LENGTH_THRESHOLDS = {
    VERY_STRONG: 12,
    STRONG: 10,
    MODERATE: 8,
    WEAK: 6,
} as const;

const SCORE_MULTIPLIER = 25; // Convert 0-4 score to 0-100 percentage

type PasswordStrengthProps = {
    label: string;
    primary: string;
    secondary: string;
    score: number;
};

// Custom progress bar component using Tailwind
interface ProgressBarProps {
    value: number; // 0-100
    primaryColor: string;
    secondaryColor: string;
    visible: boolean;
}

function ProgressBar({ value, primaryColor, secondaryColor, visible }: ProgressBarProps) {
    if (!visible) return null;

    return (
        <div
            role="progressbar"
            aria-valuenow={value}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-label="Password strength"
            className="tw-w-full tw-overflow-hidden tw-relative"
            style={{
                marginTop: "4px",  // Small positive margin to separate from input
                height: "6px",
                backgroundColor: secondaryColor || "#e5e5e5",
                borderRadius: "4px",  // Full rounding for a cleaner look
                position: "relative",
            }}
        >
            <div
                className="tw-transition-all tw-duration-300 tw-ease-out"
                style={{
                    height: "100%",
                    width: `${Math.max(value, 5)}%`, // Minimum 5% width so it's always visible
                    backgroundColor: primaryColor,
                    borderRadius: "4px",  // Full rounding to match container
                }}
            />
        </div>
    );
}

/**
 * Base password input component without Formik integration.
 * This is the pure visual component that handles all styling and interaction logic.
 */
export const PasswordTextInputBase = forwardRef<HTMLInputElement, PasswordTextInputBaseProps>(({
    autoComplete = "current-password",
    autoFocus = false,
    fullWidth = true,
    label,
    name,
    value,
    onChange,
    onBlur,
    error,
    helperText,
    id,
    ...props
}, ref) => {
    const { t } = useTranslation();
    const [showPassword, setShowPassword] = useState<boolean>(false);

    const handleClickShowPassword = useCallback(() => {
        setShowPassword(!showPassword);
    }, [showPassword]);

    const getPasswordStrengthProps = useCallback(async (password: string) => {
        const defaultProps = { label: "N/A", primary: "#3b82f6", secondary: "#bfdbfe" }; // blue colors
        if (!password) {
            return { ...defaultProps, score: 0 };
        }

        try {
            const zxcvbn = (await import("zxcvbn")).default;
            const result = zxcvbn(password);
            const score = result.score;
            switch (score) {
                case 0:
                case 1:
                    return { label: "Weak", primary: "#ef4444", secondary: "#fecaca", score }; // red
                case 2:
                    return { label: "Moderate", primary: "#f59e0b", secondary: "#fed7aa", score }; // amber
                case 3:
                    return { label: "Strong", primary: "#10b981", secondary: "#a7f3d0", score }; // emerald
                case 4:
                    return { label: "Very Strong", primary: "#059669", secondary: "#a7f3d0", score }; // dark emerald
                default:
                    return { ...defaultProps, score };
            }
        } catch (error) {
            console.warn("Failed to load password strength library:", error);
            // Fallback to simple length-based scoring
            let score = 0;
            if (password.length >= PASSWORD_LENGTH_THRESHOLDS.VERY_STRONG) score = 4;
            else if (password.length >= PASSWORD_LENGTH_THRESHOLDS.STRONG) score = 3;
            else if (password.length >= PASSWORD_LENGTH_THRESHOLDS.MODERATE) score = 2;
            else if (password.length >= PASSWORD_LENGTH_THRESHOLDS.WEAK) score = 1;

            switch (score) {
                case 0:
                case 1:
                    return { label: "Weak", primary: "#ef4444", secondary: "#fecaca", score };
                case 2:
                    return { label: "Moderate", primary: "#f59e0b", secondary: "#fed7aa", score };
                case 3:
                    return { label: "Strong", primary: "#10b981", secondary: "#a7f3d0", score };
                case 4:
                    return { label: "Very Strong", primary: "#059669", secondary: "#a7f3d0", score };
                default:
                    return { ...defaultProps, score };
            }
        }
    }, []);

    const [strengthProps, setStrengthProps] = useState<PasswordStrengthProps>({
        label: "N/A",
        primary: "#3b82f6",
        secondary: "#bfdbfe",
        score: 0,
    });

    useEffect(() => {
        let isMounted = true;

        getPasswordStrengthProps(value).then((props) => {
            if (isMounted) {
                setStrengthProps(props);
            }
        });

        return () => {
            isMounted = false;
        };
    }, [value, getPasswordStrengthProps]);

    // Start adornment - Lock icon
    const startAdornment = (
        <IconCommon
            decorative
            name="Lock"
            className="text-gray-500"
        />
    );

    // End adornment - Toggle visibility button
    const endAdornment = (
        <IconButton
            aria-label="toggle password visibility"
            onClick={handleClickShowPassword}
            variant="transparent"
            className="p-1 rounded hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
            <IconCommon
                decorative
                name={showPassword ? "Invisible" : "Visible"}
                className="text-gray-500"
            />
        </IconButton>
    );

    return (
        <div className="tw-w-full">
            <TextInputBase
                {...props}
                ref={ref}
                id={id || name}
                name={name}
                type={showPassword ? "text" : "password"}
                value={value}
                onChange={onChange}
                onBlur={onBlur}
                autoComplete={autoComplete}
                autoFocus={autoFocus}
                fullWidth={fullWidth}
                label={label ?? t("Password")}
                placeholder={t("PasswordPlaceholder")}
                error={error}
                helperText={helperText}
                startAdornment={startAdornment}
                endAdornment={endAdornment}
                variant="outline"
                size="md"
                data-testid="password-input"
                className="" // Remove extra rounding to use default from variant
            />
            <ProgressBar
                value={strengthProps.score * SCORE_MULTIPLIER} // Convert 0-4 score to 0-100 percentage
                primaryColor={strengthProps.primary}
                secondaryColor={strengthProps.secondary}
                visible={autoComplete === "new-password" && value.length > 0}
            />
        </div>
    );
});

PasswordTextInputBase.displayName = "PasswordTextInputBase";

/**
 * Formik-integrated password input component.
 * Automatically connects to Formik context using the field name.
 * 
 * @example
 * ```tsx
 * // Inside a Formik form
 * <PasswordTextInput name="password" label="Password" />
 * 
 * // With validation for new passwords
 * <PasswordTextInput 
 *   name="newPassword" 
 *   label="New Password"
 *   autoComplete="new-password"
 *   validate={(value) => value.length < 8 ? "Password must be at least 8 characters" : undefined}
 * />
 * ```
 */
export const PasswordTextInput = forwardRef<HTMLInputElement, PasswordTextInputFormikProps>(({
    name,
    validate,
    ...props
}, ref) => {
    const [field, meta] = useField({ name, validate });

    return (
        <PasswordTextInputBase
            {...props}
            ref={ref}
            name={name}
            value={field.value}
            onChange={field.onChange}
            onBlur={field.onBlur}
            error={meta.touched && !!meta.error}
            helperText={meta.touched && meta.error}
        />
    );
});

PasswordTextInput.displayName = "PasswordTextInput";
