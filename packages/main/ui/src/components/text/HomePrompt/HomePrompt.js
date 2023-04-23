import { jsx as _jsx } from "react/jsx-runtime";
import { Typography } from "@mui/material";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
export const HomePrompt = () => {
    const { t } = useTranslation();
    const quoteNumber = useMemo(() => {
        const random = Math.floor(Math.random() * 10) + 1;
        if (random === 1) {
            return Math.floor(Math.random() * 20) + 1;
        }
        return -1;
    }, []);
    return (_jsx(Typography, { component: "h1", variant: "h4", textAlign: "center", children: quoteNumber > 0 ? t(`Inspirational${quoteNumber}`) : t("WhatWouldYouLikeToDo") }));
};
//# sourceMappingURL=HomePrompt.js.map