import { FormInputBase, FormSchema, generateInitialValues, generateYupSchema, parseSearchParams, ParseSearchParamsResult, SearchType, TranslationFuncCommon } from "@local/shared";
import { Box, Button, Grid, styled, Tooltip, Typography, useTheme } from "@mui/material";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog.js";
import { TopBar } from "components/navigation/TopBar/TopBar.js";
import { Formik } from "formik";
import { FormRunView } from "forms/FormView/FormView.js";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { addSearchParams, removeSearchParams, useLocation } from "route";
import { convertFormikForSearch, convertSearchForFormik } from "utils/search/inputToSearch.js";
import { searchTypeToParams } from "utils/search/objectToSearch.js";
import { BuildIcon, CancelIcon, RefreshIcon, SearchIcon } from "../../../icons/common.js";
import { searchButtonStyle } from "../../../styles.js";
import { ELEMENT_IDS } from "../../../utils/consts.js";
import { BottomActionsGrid } from "../BottomActionsGrid/BottomActionsGrid.js";
import { AdvancedSearchButtonProps } from "../types.js";

function createTopBarOptions(resetForm: (() => unknown), t: TranslationFuncCommon) {
    return [
        {
            Icon: RefreshIcon,
            label: t("Reset"),
            onClick: resetForm,
        },
    ];
}

const FormContainer = styled(Box)(({ theme }) => ({
    margin: "auto",
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
    padding: theme.spacing(2),
    paddingBottom: "64px",
}));

function AdvancedSearchDialog({
    handleClose,
    handleSearch,
    isOpen,
    searchType,
}: {
    handleClose: () => unknown;
    handleSearch: (searchQuery: ParseSearchParamsResult) => unknown;
    isOpen: boolean;
    searchType: SearchType | `${SearchType}`;
}) {
    const { t } = useTranslation();

    const [searchParams, setSearchParams] = useState<ParseSearchParamsResult>(parseSearchParams());
    // Search schema to use
    const [schema, setSchema] = useState<FormSchema | null>(null);
    useEffect(() => {
        setSchema(searchType in searchTypeToParams ? searchTypeToParams[searchType]().advancedSearchSchema : null);
    }, [searchType]);

    const initialValues = useMemo(function initialValuesMemo() {
        // Calculate initial values from schema, to use for values not in URL
        const initialValues = generateInitialValues(schema?.elements);
        // Parse search params from URL
        const urlValues = convertSearchForFormik(searchParams, schema);
        // Replace default values with URL values
        for (const key in urlValues) {
            if (urlValues[key] === undefined) continue;
            initialValues[key] = urlValues[key] as never;
        }
        return initialValues;
    }, [schema, searchParams]);
    const validationSchema = useMemo(function validationSchemaMemo() {
        return schema ? generateYupSchema(schema) : undefined;
    }, [schema]);

    const onSubmit = useCallback(function onSubmitCallback(values: ParseSearchParamsResult) {
        if (schema) {
            const searchValue = convertFormikForSearch(values, schema);
            handleSearch(searchValue);
        }
        handleClose();
        setSearchParams(parseSearchParams());
    }, [handleSearch, schema, handleClose]);

    return (
        <LargeDialog
            id={ELEMENT_IDS.AdvancedSearchDialog}
            isOpen={isOpen}
            onClose={handleClose}
            titleId={ELEMENT_IDS.AdvancedSearchDialogTitle}
        >
            <Formik<ParseSearchParamsResult>
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={onSubmit}
                validationSchema={validationSchema}
            >
                {(formik) => {
                    function onSubmit() {
                        formik.handleSubmit();
                    }
                    function resetForm() {
                        formik.setValues(generateInitialValues(schema?.elements));
                    }
                    const topBarOptions = createTopBarOptions(resetForm, t);

                    return (
                        <>
                            <TopBar
                                display="dialog"
                                onClose={handleClose}
                                title={t("AdvancedSearch")}
                                titleId={ELEMENT_IDS.AdvancedSearchDialogTitle}
                                options={topBarOptions}
                            />
                            <FormContainer>
                                {/* Search options */}
                                {schema && <FormRunView
                                    disabled={false}
                                    schema={schema}
                                />}
                            </FormContainer>
                            {/* Search/Cancel buttons */}
                            <BottomActionsGrid display="dialog">
                                <Grid item xs={6}>
                                    <Button
                                        fullWidth
                                        startIcon={<SearchIcon />}
                                        type="submit"
                                        onClick={onSubmit}
                                        variant="contained"
                                    >{t("Search")}</Button>
                                </Grid>
                                <Grid item xs={6}>
                                    <Button
                                        fullWidth
                                        startIcon={<CancelIcon />}
                                        onClick={handleClose}
                                        variant="outlined"
                                    >{t("Cancel")}</Button>
                                </Grid>
                            </BottomActionsGrid>
                        </>
                    );
                }}
            </Formik>
        </LargeDialog>
    );
}

const filterCountLabelStyle = { marginLeft: 0.5 } as const;

export function AdvancedSearchButton({
    advancedSearchParams,
    advancedSearchSchema,
    controlsUrl,
    searchType,
    setAdvancedSearchParams,
}: AdvancedSearchButtonProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    // Update params when schema changes
    useEffect(() => {
        const searchParams = parseSearchParams();
        if (!advancedSearchSchema?.elements) {
            setAdvancedSearchParams(null);
            return;
        }
        // Open advanced search dialog, if needed
        if (typeof searchParams.advanced === "boolean") setAdvancedSearchDialogOpen(searchParams.advanced);
        // Any search params that aren't advanced, search, sort, or time MIGHT be advanced search params
        const { advanced, search, sort, time, ...otherParams } = searchParams;
        // Find valid advanced search params
        const allAdvancedSearchParams = advancedSearchSchema.elements.filter(f => Object.prototype.hasOwnProperty.call(f, "fieldName")).map(f => (f as FormInputBase).fieldName);
        // fields in both otherParams and allAdvancedSearchParams should be the new advanced search params
        const advancedData = Object.keys(otherParams).filter(k => allAdvancedSearchParams.includes(k));
        setAdvancedSearchParams(advancedData.reduce((acc, k) => ({ ...acc, [k]: otherParams[k] }), {}));
    }, [advancedSearchSchema?.elements, setAdvancedSearchParams]);

    const [advancedSearchDialogOpen, setAdvancedSearchDialogOpen] = useState<boolean>(false);
    const handleAdvancedSearchDialogOpen = useCallback(() => { setAdvancedSearchDialogOpen(true); }, []);
    const handleAdvancedSearchDialogClose = useCallback(() => {
        setAdvancedSearchDialogOpen(false);
    }, []);
    const handleAdvancedSearchDialogSubmit = useCallback((values: ParseSearchParamsResult) => {
        if (!controlsUrl) return;
        // Remove schema fields from search params
        removeSearchParams(setLocation, advancedSearchSchema?.elements?.filter(f => Object.prototype.hasOwnProperty.call(f, "fieldName")).map(f => (f as FormInputBase).fieldName) ?? []);
        // Add set fields to search params
        addSearchParams(setLocation, values);
        setAdvancedSearchParams(values);
    }, [advancedSearchSchema?.elements, controlsUrl, setAdvancedSearchParams, setLocation]);

    // Set dialog open stats in url search params
    useEffect(() => {
        if (!controlsUrl) return;
        addSearchParams(setLocation, { advanced: advancedSearchDialogOpen });
    }, [advancedSearchDialogOpen, controlsUrl, setLocation]);

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
                    {Object.keys(advancedSearchParams).length > 0 && <Typography variant="body2" sx={filterCountLabelStyle}>
                        *{Object.keys(advancedSearchParams).length}
                    </Typography>}
                </Box>
            </Tooltip>}
        </>
    );
}
