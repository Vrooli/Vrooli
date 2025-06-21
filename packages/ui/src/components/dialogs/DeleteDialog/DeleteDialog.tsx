import { useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "../../buttons/Button.js";
import { IconCommon } from "../../../icons/Icons.js";
import { TextInput } from "../../inputs/TextInput/TextInput.js";
import { Dialog, DialogContent, DialogActions } from "../Dialog/Dialog.js";
import { DialogTitle } from "../DialogTitle/DialogTitle.js";
import { FormTip } from "../../inputs/form/FormTip.js";
import { type DeleteDialogProps } from "../types.js";
import { cn } from "../../../utils/tailwind-theme.js";

const SUCCESS_ANIMATION_DELAY = 800;

export function DeleteDialog({
    handleClose,
    handleDelete,
    isOpen,
    objectName,
    objectIcon,
}: DeleteDialogProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Stores user-inputted name of object to be deleted
    const [nameInput, setNameInput] = useState<string>("");
    const [isDeleting, setIsDeleting] = useState(false);
    const [showSuccessAnimation, setShowSuccessAnimation] = useState(false);
    
    useEffect(() => {
        setNameInput("");
        setIsDeleting(false);
        setShowSuccessAnimation(false);
    }, [isOpen]);
    
    function handleInputChange(event: React.ChangeEvent<HTMLInputElement>) {
        setNameInput(event.target.value);
    }

    const close = useCallback((wasDeleted?: boolean) => {
        handleClose(wasDeleted ?? false);
    }, [handleClose]);

    function onClose() {
        close();
    }

    const handleDeleteClick = useCallback(async () => {
        setIsDeleting(true);
        try {
            await handleDelete();
            setShowSuccessAnimation(true);
            setTimeout(() => {
                close(true);
            }, SUCCESS_ANIMATION_DELAY);
        } catch (error) {
            setIsDeleting(false);
        }
    }, [handleDelete, close]);

    const isInputValid = nameInput.trim() === objectName.trim();
    const showError = nameInput.length > 0 && !isInputValid;
    const inputProgress = nameInput.length > 0 ? (nameInput.length / objectName.length) * 100 : 0;

    // FormTip data for the warning message
    const warningTip = useMemo(() => ({
        id: "delete-warning",
        label: t("DeleteWarningMessage"),
        icon: "Warning" as const,
        isMarkdown: false,
    }), [t]);

    // Memoized function for copying to clipboard
    const handleCopyToClipboard = useCallback(() => {
        navigator.clipboard.writeText(objectName);
    }, [objectName]);

    // Memoized style for text input
    const textInputStyle = useMemo(() => ({
        "& .MuiOutlinedInput-root": {
            fontFamily: "monospace",
            fontSize: "0.875rem",
            "&.Mui-focused": {
                "& fieldset": {
                    borderColor: isInputValid ? palette.success.main : palette.primary.main,
                },
            },
            "&.Mui-error": {
                animation: "shake 0.3s ease-in-out",
                "& fieldset": {
                    borderColor: palette.error.main,
                },
            },
        },
        "@keyframes shake": {
            "0%, 100%": { transform: "translateX(0)" },
            "25%": { transform: "translateX(-5px)" },
            "75%": { transform: "translateX(5px)" },
        },
    }), [isInputValid, palette]);

    // Memoized style for progress bar
    const progressBarStyle = useMemo(() => ({
        width: `${inputProgress}%`,
    }), [inputProgress]);

    return (
        <Dialog
            isOpen={isOpen}
            onClose={onClose}
            size="md"
            variant="danger"
            showCloseButton={false}
            closeOnEscape={false}
            closeOnOverlayClick={false}
            aria-labelledby="delete-dialog-title"
            aria-describedby="delete-dialog-description"
            className="tw-overflow-visible"
        >
            {/* Custom DialogTitle with just close button */}
            <DialogTitle
                id="delete-dialog-title"
                onClose={onClose}
            />
            
            {/* Custom header with delete icon and text */}
            <div className="tw-relative tw-overflow-hidden tw-bg-gradient-to-r tw-from-red-500/10 tw-to-orange-500/10 tw-px-6 tw-py-5 -tw-mt-4">
                {/* Animated background pattern */}
                <div className="tw-absolute tw-inset-0 tw-opacity-10">
                    <div className="tw-absolute tw-inset-0 tw-bg-gradient-to-br tw-from-red-500/20 tw-to-transparent tw-animate-pulse" />
                    <div className="tw-absolute tw-top-0 tw-right-0 tw-w-32 tw-h-32 tw-bg-red-500/10 tw-rounded-full tw-blur-3xl tw-animate-float" />
                    <div className="tw-absolute tw-bottom-0 tw-left-0 tw-w-40 tw-h-40 tw-bg-orange-500/10 tw-rounded-full tw-blur-3xl tw-animate-float-delayed" />
                </div>
                
                {/* Header content */}
                <div className="tw-relative tw-flex tw-items-center tw-gap-4">
                    <div className={cn(
                        "tw-flex tw-items-center tw-justify-center tw-w-14 tw-h-14 tw-rounded-xl",
                        "tw-bg-gradient-to-br tw-from-red-500 tw-to-red-600",
                        "tw-shadow-lg tw-shadow-red-500/25",
                        "tw-transform tw-transition-all tw-duration-300",
                        showSuccessAnimation ? "tw-scale-0 tw-rotate-180" : "tw-scale-100 tw-rotate-0",
                    )}>
                        {objectIcon ? (
                            <IconCommon 
                                name={objectIcon.name} 
                                size={24} 
                                className="tw-text-white"
                            />
                        ) : (
                            <IconCommon 
                                name="Delete" 
                                size={24} 
                                className="tw-text-white"
                            />
                        )}
                    </div>
                    
                    <div className="tw-flex-1">
                        <h2 className="tw-text-2xl tw-font-bold tw-text-red-600 tw-mb-1">
                            {t("Delete")} {objectName}?
                        </h2>
                        <p className="tw-text-sm tw-text-gray-600">
                            {t("DeleteConfirmDescription")}
                        </p>
                    </div>
                </div>
            </div>

            <DialogContent variant="danger" className="tw-px-6 tw-py-6">
                <div className="tw-space-y-6">
                    {/* Warning message using FormTip */}
                    <div 
                        role="alert" 
                        aria-live="polite"
                        id="delete-dialog-description"
                        className={cn(
                            "tw-transform tw-transition-all tw-duration-300",
                            showSuccessAnimation ? "tw-scale-95 tw-opacity-50" : "tw-scale-100 tw-opacity-100",
                        )}
                    >
                        <FormTip 
                            element={warningTip}
                            isEditing={false}
                        />
                    </div>
                    
                    {/* Type to confirm section */}
                    <div className="tw-space-y-3">
                        <div className="tw-flex tw-items-center tw-justify-between">
                            <label 
                                htmlFor="delete-confirmation-input"
                                className="tw-text-sm tw-font-medium tw-text-gray-700"
                            >
                                {t("TypeToConfirm")}
                            </label>
                            {nameInput.length > 0 && (
                                <span className={cn(
                                    "tw-text-xs tw-font-medium tw-transition-colors tw-duration-200",
                                    isInputValid ? "tw-text-green-600" : "tw-text-gray-400",
                                )}>
                                    {nameInput.length} / {objectName.length}
                                </span>
                            )}
                        </div>
                        
                        {/* Object name to type */}
                        <div className="tw-relative tw-mb-2">
                            <div className={cn(
                                "tw-px-4 tw-py-3 tw-rounded-lg tw-font-mono tw-text-sm",
                                "tw-bg-gray-100 tw-border tw-border-gray-200",
                                "tw-flex tw-items-center tw-justify-between tw-group",
                                "tw-transition-all tw-duration-200",
                                isInputValid && "tw-border-green-400 tw-bg-green-50",
                            )}>
                                <span className={cn(
                                    "tw-font-semibold tw-select-none",
                                    isInputValid ? "tw-text-green-700" : "tw-text-gray-700",
                                )}>
                                    {objectName}
                                </span>
                                <button
                                    type="button"
                                    onClick={handleCopyToClipboard}
                                    className={cn(
                                        "tw-p-1 tw-rounded tw-transition-all tw-duration-200",
                                        "tw-opacity-0 tw-group-hover:tw-opacity-100",
                                        "hover:tw-bg-gray-200 active:tw-scale-95",
                                    )}
                                    aria-label={t("CopyToClipboard")}
                                >
                                    <IconCommon name="Copy" size={16} className="tw-text-gray-500" />
                                </button>
                            </div>
                            
                            {/* Progress bar */}
                            <div className="tw-absolute -tw-bottom-0.5 tw-left-0 tw-right-0 tw-h-1 tw-bg-gray-200 tw-rounded-full tw-overflow-hidden">
                                <div 
                                    className={cn(
                                        "tw-h-full tw-transition-all tw-duration-300 tw-ease-out",
                                        isInputValid 
                                            ? "tw-bg-gradient-to-r tw-from-green-400 tw-to-green-500" 
                                            : showError 
                                                ? "tw-bg-gradient-to-r tw-from-red-400 tw-to-red-500"
                                                : "tw-bg-gradient-to-r tw-from-blue-400 tw-to-blue-500",
                                    )}
                                    style={progressBarStyle}
                                />
                            </div>
                        </div>
                        
                        {/* Input field */}
                        <div className="tw-relative">
                            <TextInput
                                id="delete-confirmation-input"
                                variant="outlined"
                                fullWidth
                                value={nameInput}
                                onChange={handleInputChange}
                                error={showError}
                                placeholder={t("EnterObjectName", { objectName })}
                                autoComplete="off"
                                spellCheck={false}
                                aria-label={t("ConfirmationInput")}
                                className={cn(
                                    "tw-transition-all tw-duration-200",
                                    isInputValid && "tw-ring-2 tw-ring-green-400 tw-ring-offset-2",
                                )}
                                sx={textInputStyle}
                            />
                            
                            {/* Error/success icon */}
                            <div className={cn(
                                "tw-absolute tw-right-3 tw-top-1/2 -tw-translate-y-1/2",
                                "tw-transition-all tw-duration-200",
                                nameInput.length === 0 && "tw-opacity-0 tw-scale-0",
                            )}>
                                {isInputValid ? (
                                    <IconCommon 
                                        name="Check" 
                                        size={20} 
                                        className="tw-text-green-500 tw-animate-scale-in"
                                    />
                                ) : showError ? (
                                    <IconCommon 
                                        name="Close" 
                                        size={20} 
                                        className="tw-text-red-500 tw-animate-scale-in"
                                    />
                                ) : null}
                            </div>
                        </div>
                        
                        {/* Helper text */}
                        <p className={cn(
                            "tw-text-xs tw-transition-all tw-duration-200",
                            showError 
                                ? "tw-text-red-600 tw-font-medium" 
                                : isInputValid 
                                    ? "tw-text-green-600 tw-font-medium"
                                    : "tw-text-gray-500",
                        )}>
                            {showError 
                                ? t("NameDoesNotMatch") 
                                : isInputValid 
                                    ? t("ReadyToDelete")
                                    : t("TypeExactName")}
                        </p>
                    </div>
                </div>
            </DialogContent>
            
            <DialogActions variant="danger" className="tw-px-6 tw-py-4">
                <Button
                    onClick={onClose}
                    variant="ghost"
                    size="md"
                    disabled={isDeleting}
                    className="tw-min-w-[100px]"
                    aria-label={t("CancelDelete")}
                >
                    {t("Cancel")}
                </Button>
                <Button
                    startIcon={showSuccessAnimation ? (
                        <IconCommon name="Check" className="tw-animate-scale-in" />
                    ) : (
                        <IconCommon name="Delete" />
                    )}
                    onClick={handleDeleteClick}
                    disabled={!isInputValid || isDeleting}
                    variant={showSuccessAnimation ? "primary" : "danger"}
                    size="md"
                    isLoading={isDeleting && !showSuccessAnimation}
                    className={cn(
                        "tw-min-w-[140px] tw-transition-all tw-duration-300",
                        isInputValid && !isDeleting && "hover:tw-shadow-lg hover:tw-shadow-red-500/25 hover:tw-scale-105",
                        showSuccessAnimation && "tw-bg-green-600 hover:tw-bg-green-700",
                    )}
                    aria-label={t("ConfirmDelete", { objectName })}
                >
                    {showSuccessAnimation ? t("Deleted") : t("Delete")}
                </Button>
            </DialogActions>
            
            {/* Success animation overlay */}
            {showSuccessAnimation && (
                <div className="tw-absolute tw-inset-0 tw-pointer-events-none tw-flex tw-items-center tw-justify-center tw-z-50">
                    <div className="tw-w-24 tw-h-24 tw-bg-green-500 tw-rounded-full tw-flex tw-items-center tw-justify-center tw-animate-success-pop">
                        <IconCommon name="Check" size={48} className="tw-text-white" />
                    </div>
                </div>
            )}
        </Dialog>
    );
}
