import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { ArrowLeftIcon, ArrowRightIcon, CompleteIcon } from "@local/icons";
import { Box, IconButton, Stack, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { useLocation } from "../../../utils/route";
export const TutorialView = ({ display = "page", }) => {
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const [page, setPage] = useState(0);
    const previousPage = useCallback(() => setPage(Math.max(0, page - 1)), [page, setPage]);
    const nextPage = useCallback(() => setPage(Math.min(5, page + 1)), [page, setPage]);
    const goToPage = useCallback((page) => setPage(page), [setPage]);
    const complete = useCallback(() => setLocation("/"), [setLocation]);
    const currentPage = useMemo(() => {
        return null;
    }, [page]);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Tutorial",
                } }), currentPage, _jsxs(Stack, { direction: "row", spacing: 2, sx: { justifyContent: "center", alignItems: "center", textAlign: "center", marginTop: "1em" }, children: [page > 0 ? _jsx(IconButton, { onClick: previousPage, children: _jsx(ArrowLeftIcon, { fill: palette.background.textPrimary }) }) : _jsx(Box, { sx: { width: "40px" } }), [0, 1, 2, 3, 4, 5].map((p) => (_jsx(Box, { sx: {
                            width: "1em",
                            height: "1em",
                            borderRadius: "100%",
                            background: p === page ? palette.primary.main : palette.background.textSecondary,
                            cursor: p === page ? "default" : "pointer",
                            transition: "0.3s ease-in-out",
                            "&:hover": {
                                transform: p === page ? "none" : "scale(1.2)",
                            },
                        }, onClick: () => goToPage(p) }, p))), page < 5 ? _jsx(IconButton, { onClick: nextPage, children: _jsx(ArrowRightIcon, { fill: palette.background.textPrimary }) }) : _jsx(IconButton, { onClick: complete, children: _jsx(CompleteIcon, { fill: palette.background.textPrimary }) })] })] }));
};
//# sourceMappingURL=TutorialPage.js.map