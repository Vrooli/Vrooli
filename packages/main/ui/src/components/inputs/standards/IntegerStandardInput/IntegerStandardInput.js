import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Grid } from "@mui/material";
import { useTranslation } from "react-i18next";
import { IntegerInput } from "../../IntegerInput/IntegerInput";
export const IntegerStandardInput = ({ isEditing, }) => {
    const { t } = useTranslation();
    return (_jsxs(Grid, { container: true, spacing: 2, alignItems: "center", justifyContent: "center", children: [_jsx(Grid, { item: true, xs: 12, sm: 6, children: _jsx(IntegerInput, { disabled: !isEditing, label: t("DefaultValue"), name: "defaultValue", tooltip: "The default value of the input" }) }), _jsx(Grid, { item: true, xs: 12, sm: 6, children: _jsx(IntegerInput, { disabled: !isEditing, label: t("Min"), name: "min", tooltip: "The minimum value of the integer" }) }), _jsx(Grid, { item: true, xs: 12, sm: 6, children: _jsx(IntegerInput, { disabled: !isEditing, label: t("Max"), name: "max", tooltip: "The maximum value of the integer" }) }), _jsx(Grid, { item: true, xs: 12, sm: 6, children: _jsx(IntegerInput, { disabled: !isEditing, label: t("Step"), name: "step", tooltip: "How much to increment/decrement by" }) })] }));
};
//# sourceMappingURL=IntegerStandardInput.js.map