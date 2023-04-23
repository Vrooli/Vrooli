import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { CancelIcon, RefreshIcon, SearchIcon } from "@local/icons";
import { Box, Button, Grid, useTheme } from "@mui/material";
import { Formik } from "formik";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { generateDefaultProps, generateYupSchema } from "../../../forms/generators";
import { parseSearchParams } from "../../../utils/route";
import { convertFormikForSearch, convertSearchForFormik } from "../../../utils/search/inputToSearch";
import { searchTypeToParams } from "../../../utils/search/objectToSearch";
import { GridActionButtons } from "../../buttons/GridActionButtons/GridActionButtons";
import { GeneratedGrid } from "../../inputs/generated";
import { TopBar } from "../../navigation/TopBar/TopBar";
import { LargeDialog } from "../LargeDialog/LargeDialog";
const titleId = "advanced-search-dialog-title";
export const AdvancedSearchDialog = ({ handleClose, handleSearch, isOpen, searchType, zIndex, }) => {
    const theme = useTheme();
    const { t } = useTranslation();
    const [schema, setSchema] = useState(null);
    useEffect(() => {
        async function getSchema() {
            setSchema(searchType in searchTypeToParams ? (await searchTypeToParams[searchType]()).advancedSearchSchema : null);
        }
        getSchema();
    }, [searchType]);
    const initialValues = useMemo(() => {
        const fieldInputs = generateDefaultProps(schema?.fields ?? []);
        const urlValues = schema ? convertSearchForFormik(parseSearchParams(), schema) : {};
        const values = {};
        fieldInputs.forEach((field) => {
            values[field.fieldName] = field.props.defaultValue;
        });
        Object.keys(urlValues).forEach((key) => {
            const currValue = urlValues[key];
            if (currValue !== undefined)
                values[key] = currValue;
        });
        return values;
    }, [schema]);
    const validationSchema = useMemo(() => schema ? generateYupSchema(schema) : undefined, [schema]);
    return (_jsxs(LargeDialog, { id: "advanced-search-dialog", isOpen: isOpen, onClose: handleClose, titleId: titleId, zIndex: zIndex, children: [_jsx(TopBar, { display: "dialog", onClose: handleClose, titleData: { titleId, titleKey: "AdvancedSearch" } }), _jsx(Formik, { enableReinitialize: true, initialValues: initialValues, onSubmit: (values) => {
                    if (schema) {
                        const searchValue = convertFormikForSearch(values, schema);
                        handleSearch(searchValue);
                    }
                    handleClose();
                }, validationSchema: validationSchema, children: (formik) => _jsxs(_Fragment, { children: [_jsx(Button, { onClick: () => { formik.resetForm(); }, startIcon: _jsx(RefreshIcon, {}), sx: {
                                display: "flex",
                                margin: "auto",
                                marginTop: 2,
                                marginBottom: 2,
                            }, children: t("Reset") }), _jsx(Box, { sx: {
                                margin: "auto",
                                display: "flex",
                                justifyContent: "center",
                                alignItems: "center",
                                paddingBottom: "64px",
                            }, children: schema && _jsx(GeneratedGrid, { childContainers: schema.containers, fields: schema.fields, layout: schema.formLayout, onUpload: () => { }, theme: theme, zIndex: zIndex }) }), _jsxs(GridActionButtons, { display: "dialog", children: [_jsx(Grid, { item: true, xs: 6, p: 1, sx: { paddingTop: 0 }, children: _jsx(Button, { fullWidth: true, startIcon: _jsx(SearchIcon, {}), type: "submit", onClick: formik.handleSubmit, children: t("Search") }) }), _jsx(Grid, { item: true, xs: 6, p: 1, sx: { paddingTop: 0 }, children: _jsx(Button, { fullWidth: true, startIcon: _jsx(CancelIcon, {}), onClick: handleClose, children: t("Cancel") }) })] })] }) })] }));
};
//# sourceMappingURL=AdvancedSearchDialog.js.map