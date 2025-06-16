/**
 * Cookie settings dialog for managing user privacy preferences
 */
import { useFormik } from "formik";
import { useTranslation } from "react-i18next";
import { Z_INDEX } from "../../../utils/consts.js";
import { type CookiePreferences, setCookie, getCookie } from "../../../utils/localStorage.js";
import { Button } from "../../buttons/Button.js";
import { HelpButton } from "../../buttons/HelpButton.js";
import { Switch } from "../../inputs/Switch/Switch.js";
import { TopBar } from "../../navigation/TopBar.js";
import { LargeDialog } from "../LargeDialog/LargeDialog.js";
import { type CookieSettingsDialogProps } from "../types.js";
import { cn } from "../../../utils/tailwind-theme.js";

const titleId = "cookie-settings-dialog-title";

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

const largeDialogSxs = {
    paper: { width: "min(100vw - 64px, 700px)" },
    root: { zIndex: Z_INDEX.CookieSettingsDialog },
} as const;

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
        <LargeDialog
            id="cookie-settings-dialog"
            isOpen={isOpen}
            onClose={onCancel}
            titleId={titleId}
            sxs={largeDialogSxs}
        >
            <TopBar
                display="Dialog"
                onClose={onCancel}
                title={t("CookieSettings")}
                titleId={titleId}
            />
            
            <div className="tw-p-6 tw-space-y-6">
                {/* Introduction text */}
                <div className="tw-mb-6">
                    <p className="tw-text-base tw-text-gray-600 dark:tw-text-gray-300 tw-leading-relaxed">
                        {t("CookieSettingsIntro", "We use cookies to enhance your experience. Choose which types of cookies you'd like to allow.")}
                    </p>
                </div>

                <form onSubmit={formik.handleSubmit} className="tw-space-y-6">
                    {/* Cookie categories */}
                    {cookieCategories.map((category, index) => (
                        <div key={category.key} className={cn(
                            "tw-border tw-border-gray-200 dark:tw-border-gray-700 tw-rounded-lg tw-p-4",
                            "tw-transition-all tw-duration-200",
                            "hover:tw-border-gray-300 dark:hover:tw-border-gray-600",
                            "hover:tw-shadow-sm"
                        )}>
                            {/* Category header */}
                            <div className="tw-flex tw-items-start tw-justify-between tw-gap-4">
                                <div className="tw-flex-1 tw-min-w-0">
                                    <div className="tw-flex tw-items-center tw-gap-2 tw-mb-2">
                                        <h3 className="tw-font-semibold tw-text-lg tw-text-gray-900 dark:tw-text-gray-100">
                                            {t(category.titleKey)}
                                        </h3>
                                        {category.required && (
                                            <span className="tw-inline-flex tw-items-center tw-px-2 tw-py-1 tw-rounded-md tw-text-xs tw-font-medium tw-bg-blue-100 tw-text-blue-800 dark:tw-bg-blue-900 dark:tw-text-blue-200">
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
                    <div className="tw-flex tw-flex-col sm:tw-flex-row tw-gap-3 tw-pt-4 tw-border-t tw-border-gray-200 dark:tw-border-gray-700">
                        {/* Primary actions */}
                        <div className="tw-flex tw-flex-col sm:tw-flex-row tw-gap-3 tw-flex-1">
                            <Button
                                type="submit"
                                variant="primary"
                                size="md"
                                fullWidth
                                className="sm:tw-flex-1"
                            >
                                {t("SavePreferences")}
                            </Button>
                            
                            <Button
                                onClick={handleAcceptAllCookies}
                                variant="secondary"
                                size="md"
                                fullWidth
                                className="sm:tw-flex-1"
                            >
                                {t("AcceptAll")}
                            </Button>
                        </div>
                        
                        {/* Secondary actions */}
                        <div className="tw-flex tw-flex-col sm:tw-flex-row tw-gap-3">
                            <Button
                                onClick={handleRejectAllOptional}
                                variant="outline"
                                size="md"
                                fullWidth
                                className="sm:tw-w-auto"
                            >
                                {t("RejectOptional")}
                            </Button>
                            
                            <Button
                                onClick={onCancel}
                                variant="ghost"
                                size="md"
                                fullWidth
                                className="sm:tw-w-auto"
                            >
                                {t("Cancel")}
                            </Button>
                        </div>
                    </div>
                </form>
            </div>
        </LargeDialog>
    );
}
