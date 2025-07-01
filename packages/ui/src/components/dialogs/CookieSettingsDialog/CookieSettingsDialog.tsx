/**
 * Cookie settings dialog for managing user privacy preferences
 */
import { useFormik } from "formik";
import { useTranslation } from "react-i18next";
import { type CookiePreferences, setCookie, getCookie } from "../../../utils/localStorage.js";
import { Button } from "../../buttons/Button.js";
import { HelpButton } from "../../buttons/HelpButton.js";
import { Switch } from "../../inputs/Switch/Switch.js";
import { Dialog, DialogContent } from "../Dialog/Dialog.js";
import { type CookieSettingsDialogProps } from "../types.js";
import { cn } from "../../../utils/tailwind-theme.js";

const cookieCategories = [
    {
        key: "strictlyNecessary" as const,
        titleKey: "CookieStrictlyNecessary",
        descriptionKey: "CookieStrictlyNecessaryDescription",
        uses: ["Authentication"],
        required: true,
        variant: "default" as const,
    },
    {
        key: "functional" as const,
        titleKey: "Functional", 
        descriptionKey: "CookieFunctionalDescription",
        uses: ["DisplayCustomization", "Caching"],
        required: false,
        variant: "success" as const,
    },
    {
        key: "performance" as const,
        titleKey: "Performance",
        descriptionKey: "CookiePerformanceDescription", 
        uses: [],
        required: false,
        variant: "warning" as const,
    },
    {
        key: "targeting" as const,
        titleKey: "Targeting",
        descriptionKey: "CookieTargetingDescription",
        uses: [],
        required: false,
        variant: "danger" as const,
    },
] as const;

export function CookieSettingsDialog({
    handleClose,
    isOpen,
}: CookieSettingsDialogProps) {
    const { t } = useTranslation();

    // Get current preferences from localStorage
    const currentPreferences = getCookie("Preferences") as CookiePreferences | null;
    
    function setPreferences(preferences: CookiePreferences) {
        setCookie("Preferences", preferences);
        handleClose(preferences);
    }
    
    function onCancel() {
        handleClose();
    }

    const formik = useFormik({
        initialValues: {
            strictlyNecessary: true,
            performance: currentPreferences?.performance ?? false,
            functional: currentPreferences?.functional ?? true,
            targeting: currentPreferences?.targeting ?? false,
        },
        onSubmit: (values) => {
            setPreferences(values);
        },
    });

    function handleAcceptAllCookies() {
        const preferences: CookiePreferences = {
            strictlyNecessary: true,
            performance: true,
            functional: true,
            targeting: true,
        };
        setPreferences(preferences);
    }

    function handleRejectAllOptional() {
        const preferences: CookiePreferences = {
            strictlyNecessary: true,
            performance: false,
            functional: false,
            targeting: false,
        };
        setPreferences(preferences);
    }

    return (
        <Dialog
            isOpen={isOpen}
            onClose={onCancel}
            title={t("CookieSettings")}
            size="lg"
        >
            <DialogContent>
                {/* Introduction text */}
                <div className="tw-mb-4">
                    <p className="tw-text-sm tw-text-gray-600 dark:tw-text-gray-300">
                        {t("CookieSettingsIntro", "We use cookies to enhance your experience. Choose which types of cookies you'd like to allow.")}
                    </p>
                </div>

                <form onSubmit={formik.handleSubmit} className="tw-space-y-4">
                    {/* Cookie categories */}
                    {cookieCategories.map((category, index) => (
                        <div key={category.key} className={cn(
                            "tw-border tw-border-gray-200 dark:tw-border-gray-700 tw-rounded-lg tw-p-3",
                            "tw-transition-all tw-duration-200",
                            "hover:tw-border-gray-300 dark:hover:tw-border-gray-600",
                            "hover:tw-shadow-sm",
                        )}>
                            {/* Category header */}
                            <div className="tw-flex tw-items-start tw-justify-between tw-gap-3">
                                <div className="tw-flex-1 tw-min-w-0">
                                    <div className="tw-flex tw-items-center tw-gap-2 tw-mb-1">
                                        <h3 className="tw-font-semibold tw-text-base tw-text-gray-900 dark:tw-text-gray-100">
                                            {t(category.titleKey)}
                                        </h3>
                                        {category.required && (
                                            <span className="tw-inline-flex tw-items-center tw-px-1.5 tw-py-0.5 tw-rounded tw-text-xs tw-font-medium tw-bg-blue-100 tw-text-blue-800 dark:tw-bg-blue-900 dark:tw-text-blue-200">
                                                {t("Required")}
                                            </span>
                                        )}
                                        <HelpButton markdown={t(category.descriptionKey)} />
                                    </div>
                                    
                                    {/* Uses */}
                                    <div className="tw-text-sm tw-text-gray-600 dark:tw-text-gray-400">
                                        <span className="tw-font-medium">{t("CurrentUses")}:</span>
                                        {category.uses.length > 0 ? (
                                            <span className="tw-ml-1">
                                                {category.uses.map((use) => t(use)).join(", ")}
                                            </span>
                                        ) : (
                                            <span className="tw-ml-1 tw-font-medium">{t("None")}</span>
                                        )}
                                    </div>
                                </div>
                                
                                {/* Switch */}
                                <div className="tw-flex-shrink-0">
                                    <Switch
                                        variant={category.variant}
                                        size="md"
                                        checked={formik.values[category.key]}
                                        disabled={category.required}
                                        onChange={(checked) => {
                                            formik.setFieldValue(category.key, checked);
                                        }}
                                        aria-label={`${t("Toggle")} ${t(category.titleKey)}`}
                                    />
                                </div>
                            </div>
                        </div>
                    ))}

                    {/* Action buttons */}
                    <div className="tw-flex tw-flex-col sm:tw-flex-row tw-gap-2 tw-pt-3 tw-border-t tw-border-gray-200 dark:tw-border-gray-700">
                        <Button
                            type="submit"
                            variant="primary"
                            size="sm"
                            fullWidth
                            className="sm:tw-flex-1"
                        >
                            {t("Save")}
                        </Button>
                        
                        <Button
                            onClick={handleAcceptAllCookies}
                            variant="secondary"
                            size="sm"
                            fullWidth
                            className="sm:tw-flex-1"
                        >
                            {t("AcceptAll")}
                        </Button>
                        
                        <Button
                            onClick={onCancel}
                            variant="ghost"
                            size="sm"
                            fullWidth
                            className="sm:tw-w-auto"
                        >
                            {t("Cancel")}
                        </Button>
                    </div>
                </form>
            </DialogContent>
        </Dialog>
    );
}
