import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { ApiIcon, DeleteIcon, HelpIcon, NoteIcon, OrganizationIcon, ProjectIcon, RoutineIcon, SmartContractIcon, StandardIcon } from "@local/icons";
import { Box, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { multiLineEllipsis, noSelect } from "../../../../styles";
import { getDisplay } from "../../../../utils/display/listTools";
import { getUserLanguages } from "../../../../utils/display/translationTools";
import usePress from "../../../../utils/hooks/usePress";
import { getObjectUrl } from "../../../../utils/navigation/openObject";
import { useLocation } from "../../../../utils/route";
import { SessionContext } from "../../../../utils/SessionContext";
import { ColorIconButton } from "../../../buttons/ColorIconButton/ColorIconButton";
export const DirectoryCard = ({ canUpdate, data, index, onContextMenu, onDelete, }) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [showIcons, setShowIcons] = useState(false);
    const { title, subtitle } = useMemo(() => getDisplay(data, getUserLanguages(session)), [data, session]);
    const Icon = useMemo(() => {
        if (!data || !data.__typename)
            return HelpIcon;
        if (data.__typename === "ApiVersion")
            return ApiIcon;
        if (data.__typename === "NoteVersion")
            return NoteIcon;
        if (data.__typename === "Organization")
            return OrganizationIcon;
        if (data.__typename === "ProjectVersion")
            return ProjectIcon;
        if (data.__typename === "RoutineVersion")
            return RoutineIcon;
        if (data.__typename === "SmartContractVersion")
            return SmartContractIcon;
        if (data.__typename === "StandardVersion")
            return StandardIcon;
        return HelpIcon;
    }, [data]);
    const href = useMemo(() => data ? getObjectUrl(data) : "#", [data]);
    const handleClick = useCallback((target) => {
        const targetId = target.id;
        if (targetId && targetId.startsWith("delete-")) {
            onDelete?.(index);
        }
        else {
            setLocation(href);
        }
    }, [href, index, onDelete, setLocation]);
    const handleContextMenu = useCallback((target) => {
        onContextMenu(target, index);
    }, [onContextMenu, index]);
    const handleHover = useCallback(() => {
        if (canUpdate) {
            setShowIcons(true);
        }
    }, [canUpdate]);
    const handleHoverEnd = useCallback(() => { setShowIcons(false); }, []);
    const pressEvents = usePress({
        onLongPress: handleContextMenu,
        onClick: handleClick,
        onHover: handleHover,
        onHoverEnd: handleHoverEnd,
        onRightClick: handleContextMenu,
        hoverDelay: 100,
    });
    return (_jsx(Tooltip, { placement: "top", title: `${subtitle ? subtitle + " - " : ""}${href}`, children: _jsxs(Box, { ...pressEvents, component: "a", href: href, onClick: (e) => e.preventDefault(), sx: {
                ...noSelect,
                boxShadow: 8,
                background: palette.primary.light,
                color: palette.secondary.contrastText,
                borderRadius: "16px",
                margin: 0,
                padding: 1,
                cursor: "pointer",
                width: "120px",
                minWidth: "120px",
                minHeight: "120px",
                height: "120px",
                position: "relative",
                "&:hover": {
                    filter: "brightness(120%)",
                    transition: "filter 0.2s",
                },
            }, children: [showIcons && (_jsx(_Fragment, { children: _jsx(Tooltip, { title: t("Delete"), children: _jsx(ColorIconButton, { id: 'delete-icon-button', background: palette.error.main, sx: { position: "absolute", top: 4, right: 4 }, children: _jsx(DeleteIcon, { id: 'delete-icon', fill: palette.secondary.contrastText }) }) }) })), _jsxs(Stack, { direction: "column", justifyContent: "center", alignItems: "center", sx: {
                        height: "100%",
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                    }, children: [_jsx(Icon, { fill: "white" }), _jsx(Typography, { gutterBottom: true, variant: "body2", component: "h3", sx: {
                                ...multiLineEllipsis(3),
                                textAlign: "center",
                                lineBreak: Boolean(title) ? "auto" : "anywhere",
                            }, children: title })] })] }) }));
};
//# sourceMappingURL=DirectoryCard.js.map