import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { uuid } from "@local/uuid";
import { Stack } from "@mui/material";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getDeviceInfo } from "../../../../utils/display/device";
import { translateSnackMessage } from "../../../../utils/display/translationTools";
import { PubSub } from "../../../../utils/pubsub";
import { BasicSnack } from "../BasicSnack/BasicSnack";
import { CookiesSnack } from "../CookiesSnack/CookiesSnack";
export const SnackStack = () => {
    const { t } = useTranslation();
    const [snacks, setSnacks] = useState([]);
    const [isCookieSnackOpen, setIsCookieSnackOpen] = useState(false);
    const handleClose = (id) => {
        setSnacks((prev) => [...prev.filter((snack) => snack.id !== id)]);
    };
    useEffect(() => {
        const snackSub = PubSub.get().subscribeSnack((o) => {
            setSnacks((snacks) => {
                const id = o.id ?? uuid();
                let newSnacks = [...snacks, {
                        buttonClicked: o.buttonClicked,
                        buttonText: o.buttonKey ? t(o.buttonKey, { ...o.buttonVariables, defaultValue: o.buttonKey }) : undefined,
                        data: o.data,
                        handleClose: () => handleClose(id),
                        id,
                        message: o.message ?
                            o.message :
                            translateSnackMessage(o.messageKey, o.messageVariables).message,
                        severity: o.severity,
                    }];
                newSnacks = newSnacks.filter((snack, index, self) => { return self.findIndex((s) => s.id === snack.id) === index; });
                if (newSnacks.length > 3) {
                    newSnacks = newSnacks.slice(1);
                }
                return newSnacks;
            });
        });
        const cookiesSub = PubSub.get().subscribeCookies(() => {
            if (getDeviceInfo().isStandalone)
                return;
            setIsCookieSnackOpen(true);
        });
        return () => {
            PubSub.get().unsubscribe(snackSub);
            PubSub.get().unsubscribe(cookiesSub);
        };
    }, [t]);
    const visible = useMemo(() => snacks.length > 0 || isCookieSnackOpen, [snacks, isCookieSnackOpen]);
    return (_jsxs(Stack, { direction: "column", spacing: 1, sx: {
            display: visible ? "flex" : "none",
            position: "fixed",
            bottom: { xs: "calc(64px + env(safe-area-inset-bottom))", md: "calc(8px + env(safe-area-inset-bottom))" },
            left: "calc(8px + env(safe-area-inset-left))",
            zIndex: 20000,
            pointerEvents: "none",
        }, children: [snacks.map((snack) => (_jsx(BasicSnack, { ...snack }, snack.id))), isCookieSnackOpen && _jsx(CookiesSnack, { handleClose: () => setIsCookieSnackOpen(false) })] }));
};
//# sourceMappingURL=SnackStack.js.map