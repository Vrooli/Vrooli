import { jsx as _jsx } from "react/jsx-runtime";
import { Breadcrumbs, Link, } from "@mui/material";
import { useMemo } from "react";
import { noSelect } from "../../../styles";
import { openLink, useLocation } from "../../../utils/route";
export const BreadcrumbsBase = ({ paths, separator = "|", ariaLabel = "breadcrumb", textColor, sx, }) => {
    const [, setLocation] = useLocation();
    const pathLinks = useMemo(() => (paths.map(p => (_jsx(Link, { color: textColor, href: p.link, onClick: (e) => { e.preventDefault(); openLink(setLocation, p.link); }, children: window.location.pathname === p.link ? _jsx("b", { children: p.text }) : p.text }, p.text)))), [setLocation, paths, textColor]);
    return (_jsx(Breadcrumbs, { sx: {
            ...sx,
            "& .MuiBreadcrumbs-li > a": {
                color: sx?.color || "inherit",
                minHeight: "48px",
                display: "flex",
                flexDirection: "row",
                alignItems: "center",
                cursor: "pointer",
                ...noSelect,
            },
        }, separator: separator, "aria-label": ariaLabel, children: pathLinks }));
};
//# sourceMappingURL=BreadcrumbsBase.js.map