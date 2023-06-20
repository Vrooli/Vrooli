/**
 * Displays all search options for an organization
 */
import { CancelIcon, parseSearchParams, RefreshIcon, SearchIcon } from "@local/shared";
import { Box, Button, Grid, useTheme } from "@mui/material";
import { GridActionButtons } from "components/buttons/GridActionButtons/GridActionButtons";
import { GeneratedGrid } from "components/inputs/generated";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { generateDefaultProps, generateYupSchema } from "forms/generators";
import { FieldData, FormSchema } from "forms/types";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { convertFormikForSearch, convertSearchForFormik } from "utils/search/inputToSearch";
import { searchTypeToParams } from "utils/search/objectToSearch";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { AdvancedSearchDialogProps } from "../types";

const titleId = "advanced-search-dialog-title";

export const AdvancedSearchDialog = ({
    handleClose,
    handleSearch,
    isOpen,
    searchType,
    zIndex,
}: AdvancedSearchDialogProps) => {
    const theme = useTheme();
    const { t } = useTranslation();

    // Search schema to use
    const [schema, setSchema] = useState<FormSchema | null>(null);
    useEffect(() => {
        async function getSchema() {
            setSchema(searchType in searchTypeToParams ? (await searchTypeToParams[searchType]()).advancedSearchSchema : null);
        }
        getSchema();
    }, [searchType]);

    // Parse default values to use in formik
    const initialValues = useMemo(() => {
        // Calculate initial values from schema, to use if values not already in URL
        const fieldInputs: FieldData[] = generateDefaultProps(schema?.fields ?? []);
        // Parse search params from URL, and filter out search fields that are not in schema
        const urlValues = schema ? convertSearchForFormik(parseSearchParams(), schema) : {} as { [key: string]: any };
        // Filter out search params that are not in schema
        const values: { [x: string]: any } = {};
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
            zIndex={zIndex}
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
                            zIndex={zIndex}
                        />}
                    </Box>
                    {/* Search/Cancel buttons */}
                    <GridActionButtons display="dialog">
                        <Grid item xs={6} p={1} sx={{ paddingTop: 0 }}>
                            <Button
                                fullWidth
                                startIcon={<SearchIcon />}
                                type="submit"
                                onClick={formik.handleSubmit as any}
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
                    </GridActionButtons>
                </>}
            </Formik>
        </LargeDialog>
    );
};
