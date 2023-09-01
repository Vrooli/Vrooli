import { Box, Button, Grid, Tooltip, Typography, useTheme } from "@mui/material";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { GeneratedGrid } from "components/inputs/generated";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { generateDefaultProps, generateYupSchema } from "forms/generators";
import { FieldData, FormSchema } from "forms/types";
import { BuildIcon, CancelIcon, RefreshIcon, SearchIcon } from "icons";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { addSearchParams, parseSearchParams, removeSearchParams, useLocation } from "route";
import { convertFormikForSearch, convertSearchForFormik } from "utils/search/inputToSearch";
import { SearchType, searchTypeToParams } from "utils/search/objectToSearch";
import { BottomActionsGrid } from "../BottomActionsGrid/BottomActionsGrid";
import { searchButtonStyle } from "../styles";
import { AdvancedSearchButtonProps } from "../types";

type SearchQuery = { [x: string]: object | string | number | boolean | null };

const titleId = "advanced-search-dialog-title";

const AdvancedSearchDialog = ({
    handleClose,
    handleSearch,
    isOpen,
    searchType,
}: {
    handleClose: () => unknown;
    handleSearch: (searchQuery: SearchQuery) => unknown;
    isOpen: boolean;
    searchType: SearchType | `${SearchType}`;
}) => {
    const theme = useTheme();
    const { t } = useTranslation();

    // Search schema to use
    const [schema, setSchema] = useState<FormSchema | null>(null);
    useEffect(() => {
        setSchema(searchType in searchTypeToParams ? searchTypeToParams[searchType]().advancedSearchSchema : null);
    }, [searchType]);

    // Parse default values to use in formik
    const initialValues = useMemo(() => {
        // Calculate initial values from schema, to use if values not already in URL
        const fieldInputs: FieldData[] = generateDefaultProps(schema?.fields ?? []);
        // Parse search params from URL, and filter out search fields that are not in schema
        const urlValues = schema ? convertSearchForFormik(parseSearchParams(), schema) : {} as { [key: string]: object | string | number | boolean | null };
        // Filter out search params that are not in schema
        const values: SearchQuery = {};
        // Add fieldInputs to values
        fieldInputs.forEach((field) => {
            values[field.fieldName] = field.props.defaultValue;
        });
        // Add or replace urlValues to values
        Object.keys(urlValues).forEach((key) => {
            const currValue = urlValues[key];
            if (currValue !== undefined) values[key] = currValue;
        });
        return values;
    }, [schema]);

    // Generate yup validation schema
    const validationSchema = useMemo(() => schema ? generateYupSchema(schema) : undefined, [schema]);

    return (
        <LargeDialog
            id="advanced-search-dialog"
            isOpen={isOpen}
            onClose={handleClose}
            titleId={titleId}
        >
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values) => {
                    if (schema) {
                        const searchValue = convertFormikForSearch(values, schema);
                        handleSearch(searchValue);
                    }
                    handleClose();
                }}
                validationSchema={validationSchema}
            >
                {(formik) => <>
                    <TopBar
                        display="dialog"
                        onClose={handleClose}
                        title={t("AdvancedSearch")}
                        titleId={titleId}
                        options={[{
                            Icon: RefreshIcon,
                            label: t("Reset"),
                            onClick: () => { formik.resetForm(); },
                        }]}
                    />
                    <Box sx={{
                        margin: "auto",
                        display: "flex",
                        justifyContent: "center",
                        alignItems: "center",
                        paddingBottom: "64px",
                    }}>
                        {/* Search options */}
                        {schema && <GeneratedGrid
                            childContainers={schema.containers}
                            fields={schema.fields}
                            layout={schema.formLayout}
                            // eslint-disable-next-line @typescript-eslint/no-empty-function
                            onUpload={() => { }}
                            theme={theme}
                        />}
                    </Box>
                    {/* Search/Cancel buttons */}
                    <BottomActionsGrid display="dialog">
                        <Grid item xs={6} p={1} sx={{ paddingTop: 0 }}>
                            <Button
                                fullWidth
                                startIcon={<SearchIcon />}
                                type="submit"
                                onClick={formik.handleSubmit as (() => void)}
                                variant="contained"
                            >{t("Search")}</Button>
                        </Grid>
                        <Grid item xs={6} p={1} sx={{ paddingTop: 0 }}>
                            <Button
                                fullWidth
                                startIcon={<CancelIcon />}
                                onClick={handleClose}
                                variant="outlined"
                            >{t("Cancel")}</Button>
                        </Grid>
                    </BottomActionsGrid>
                </>}
            </Formik>
        </LargeDialog>
    );
};

export const AdvancedSearchButton = ({
    advancedSearchParams,
    advancedSearchSchema,
    searchType,
    setAdvancedSearchParams,
}: AdvancedSearchButtonProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    // Update params when schema changes
    useEffect(() => {
        const searchParams = parseSearchParams();
        if (!advancedSearchSchema?.fields) {
            console.log("setting advanced search params 1");
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
        console.log("setting advanced search params 2", advancedData, otherParams);
        setAdvancedSearchParams(advancedData.reduce((acc, k) => ({ ...acc, [k]: otherParams[k] }), {}));
    }, [advancedSearchSchema?.fields, setAdvancedSearchParams]);

    const [advancedSearchDialogOpen, setAdvancedSearchDialogOpen] = useState<boolean>(false);
    const handleAdvancedSearchDialogOpen = useCallback(() => { setAdvancedSearchDialogOpen(true); }, []);
    const handleAdvancedSearchDialogClose = useCallback(() => {
        setAdvancedSearchDialogOpen(false);
    }, []);
    const handleAdvancedSearchDialogSubmit = useCallback((values: SearchQuery) => {
        // Remove 0 values
        const valuesWithoutBlanks = Object.fromEntries(Object.entries(values).filter(([_, v]) => v !== 0));
        // Remove schema fields from search params
        removeSearchParams(setLocation, advancedSearchSchema?.fields?.map(f => f.fieldName) ?? []);
        // Add set fields to search params
        addSearchParams(setLocation, valuesWithoutBlanks);
        console.log("setting advanced search params 3", valuesWithoutBlanks);
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
