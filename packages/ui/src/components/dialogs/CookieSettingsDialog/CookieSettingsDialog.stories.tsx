import { useCallback, useState } from "react";
import { CookieSettingsDialog } from "./CookieSettingsDialog.js";

export default {
    title: "Components/Dialogs/CookieSettingsDialog",
    component: CookieSettingsDialog,
};

export function Default() {
    const [isOpen, setIsOpen] = useState(true);
    const handleClose = useCallback(() => {
        setIsOpen(false);
    }, [setIsOpen]);

    return (
        <CookieSettingsDialog
            handleClose={handleClose}
            isOpen={isOpen}
        />
    );
}
