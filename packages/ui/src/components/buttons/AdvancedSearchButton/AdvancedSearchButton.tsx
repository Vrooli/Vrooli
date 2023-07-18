import { Box, Tooltip, Typography, useTheme } from "@mui/material";
import { AdvancedSearchDialog } from "components/dialogs/AdvancedSearchDialog/AdvancedSearchDialog";
import { BuildIcon } from "icons";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { addSearchParams, parseSearchParams, removeSearchParams, useLocation } from "route";
import { searchButtonStyle } from "../styles";
import { AdvancedSearchButtonProps } from "../types";

export const AdvancedSearchButton = ({
    advancedSearchParams,
    advancedSearchSchema,
    searchType,
    setAdvancedSearchParams,
    zIndex,
}: AdvancedSearchButtonProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    // Update params when schema changes
    useEffect(() => {
        const searchParams = parseSearchParams();
        if (!advancedSearchSchema?.fields) {
            setAdvancedSearchParams(null);
            return;
        }
        // Open advanced search dialog, if needed
        if (typeof searchParams.advanced === "boolean") setAdvancedSearchDialogOpen(searchParams.advanced);
        // Any search params that aren't advanced, search, sort, or time MIGHT be advanced search params
        const { advanced, search, sort, time, ...otherParams } = searchParams;
        // Find valid advanced search params
        const allAdvancedSearchParams = advancedSearchSchema.fields.map(f => f.fieldName);
        // fields in both otherParams and allAdvancedSearchParams should be the new advanced search params
        const advancedData = Object.keys(otherParams).filter(k => allAdvancedSearchParams.includes(k));
        setAdvancedSearchParams(advancedData.reduce((acc, k) => ({ ...acc, [k]: otherParams[k] }), {}));
    }, [advancedSearchSchema?.fields, setAdvancedSearchParams]);

    const [advancedSearchDialogOpen, setAdvancedSearchDialogOpen] = useState<boolean>(false);
    const handleAdvancedSearchDialogOpen = useCallback(() => { setAdvancedSearchDialogOpen(true); }, []);
    const handleAdvancedSearchDialogClose = useCallback(() => {
        setAdvancedSearchDialogOpen(false);
    }, []);
    const handleAdvancedSearchDialogSubmit = useCallback((values: any) => {
        // Remove 0 values
        const valuesWithoutBlanks: { [x: string]: any } = Object.fromEntries(Object.entries(values).filter(([_, v]) => v !== 0));
        // Remove schema fields from search params
        removeSearchParams(setLocation, advancedSearchSchema?.fields?.map(f => f.fieldName) ?? []);
        // Add set fields to search params
        addSearchParams(setLocation, valuesWithoutBlanks);
        setAdvancedSearchParams(valuesWithoutBlanks);
    }, [advancedSearchSchema?.fields, setAdvancedSearchParams, setLocation]);

    // Set dialog open stats in url search params
    useEffect(() => {
        addSearchParams(setLocation, { advanced: advancedSearchDialogOpen });
    }, [advancedSearchDialogOpen, setLocation]);

    return (
        <>
            <AdvancedSearchDialog
                handleClose={handleAdvancedSearchDialogClose}
                handleSearch={handleAdvancedSearchDialogSubmit}
                isOpen={advancedSearchDialogOpen}
                searchType={searchType}
                zIndex={zIndex + 1}
            />
            {advancedSearchParams && <Tooltip title={t("SeeAllSearchSettings")} placement="top">
                <Box
                    onClick={handleAdvancedSearchDialogOpen}
                    sx={searchButtonStyle(palette)}
                >
                    <BuildIcon fill={palette.secondary.main} />
                    {Object.keys(advancedSearchParams).length > 0 && <Typography variant="body2" sx={{ marginLeft: 0.5 }}>
                        *{Object.keys(advancedSearchParams).length}
                    </Typography>}
                </Box>
            </Tooltip>}
        </>
    );
};
