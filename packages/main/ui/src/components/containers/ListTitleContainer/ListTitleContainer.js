import { jsx as _jsx } from "react/jsx-runtime";
import { List, Typography } from "@mui/material";
import { useTranslation } from "react-i18next";
import { TitleContainer } from "../TitleContainer/TitleContainer";
export function ListTitleContainer({ children, emptyText, isEmpty, ...props }) {
    const { t } = useTranslation();
    return (_jsx(TitleContainer, { ...props, children: isEmpty ?
            _jsx(Typography, { variant: "h6", sx: {
                    textAlign: "center",
                    paddingTop: "8px",
                }, children: emptyText ?? t("NoResults", { ns: "error" }) }) :
            _jsx(List, { sx: { overflow: "hidden" }, children: children }) }));
}
//# sourceMappingURL=ListTitleContainer.js.map