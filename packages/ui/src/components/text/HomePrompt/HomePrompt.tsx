import { CommonKey } from "@local/shared";
import { Typography } from "@mui/material";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";

/**
 * Main prompt for HomeView. Displays "What would you 
 * like to do?" most of the time, but every once in a while 
 * displays a random inspirational quote.
 */
export const HomePrompt = () => {
    const { t } = useTranslation();

    // There is a 1 in 20 chance that a quote will be displayed. 
    // Quotes are numbered from 1 to 20.
    const quoteNumber = useMemo(() => {
        // Pick a number between 1 and 20
        const random = Math.floor(Math.random() * 20) + 1;
        // If the number is 1, return a random number between 1 and 20
        if (random === 1) {
            return Math.floor(Math.random() * 20) + 1;
        }
        // Otherwise, return -1
        return -1;
    }, []);

    return (
        <Typography component="h1" variant="h4" textAlign="center">
            {quoteNumber > 0 ? t(`Inspirational${quoteNumber}` as CommonKey) : t("WhatWouldYouLikeToDo")}
        </Typography>
    );
};
