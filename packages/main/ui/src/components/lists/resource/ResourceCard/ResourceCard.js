import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { ResourceUsedFor } from "@local/consts";
import { DeleteIcon, EditIcon } from "@local/icons";
import { Box, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { forwardRef, useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { multiLineEllipsis, noSelect } from "../../../../styles";
import { getResourceIcon } from "../../../../utils/display/getResourceIcon";
import { getDisplay } from "../../../../utils/display/listTools";
import { firstString } from "../../../../utils/display/stringTools";
import { getUserLanguages } from "../../../../utils/display/translationTools";
import usePress from "../../../../utils/hooks/usePress";
import { getResourceType, getResourceUrl } from "../../../../utils/navigation/openObject";
import { PubSub } from "../../../../utils/pubsub";
import { openLink, useLocation } from "../../../../utils/route";
import { SessionContext } from "../../../../utils/SessionContext";
import { ColorIconButton } from "../../../buttons/ColorIconButton/ColorIconButton";
export const ResourceCard = forwardRef(({ canUpdate, data, dragProps, dragHandleProps, index, onContextMenu, onEdit, onDelete, }, ref) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [showIcons, setShowIcons] = useState(false);
    const { title, subtitle } = useMemo(() => {
        const { title, subtitle } = getDisplay(data, getUserLanguages(session));
        return {
            title: Boolean(title) ? title : t((data.usedFor ?? "Context")),
            subtitle,
        };
    }, [data, session, t]);
    const Icon = useMemo(() => {
        return getResourceIcon(data.usedFor ?? ResourceUsedFor.Related, data.link);
    }, [data]);
    const href = useMemo(() => getResourceUrl(data.link), [data]);
    const handleClick = useCallback((target) => {
        const targetId = target.id;
        if (targetId && targetId.startsWith("edit-")) {
            onEdit?.(index);
        }
        else if (targetId && targetId.startsWith("delete-")) {
            onDelete?.(index);
        }
        else {
            const resourceType = getResourceType(data.link);
            if (!resourceType || !href) {
                PubSub.get().publishSnack({ messageKey: "CannotOpenLink", severity: "Error" });
                return;
            }
            else
                openLink(setLocation, href);
        }
    }, [data.link, href, index, onDelete, onEdit, setLocation]);
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
    return (_jsx(Tooltip, { placement: "top", title: `${subtitle ? subtitle + " - " : ""}${data.link}`, children: _jsxs(Box, { ref: ref, ...dragProps, ...dragHandleProps, ...pressEvents, component: "a", href: href, onClick: (e) => e.preventDefault(), sx: {
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
            }, children: [showIcons && (_jsxs(_Fragment, { children: [_jsx(Tooltip, { title: t("Edit"), children: _jsx(ColorIconButton, { id: 'edit-icon-button', background: '#c5ab17', sx: { position: "absolute", top: 4, left: 4 }, children: _jsx(EditIcon, { id: 'edit-icon', fill: palette.secondary.contrastText }) }) }), _jsx(Tooltip, { title: t("Delete"), children: _jsx(ColorIconButton, { id: 'delete-icon-button', background: palette.error.main, sx: { position: "absolute", top: 4, right: 4 }, children: _jsx(DeleteIcon, { id: 'delete-icon', fill: palette.secondary.contrastText }) }) })] })), _jsxs(Stack, { direction: "column", justifyContent: "center", alignItems: "center", sx: {
                        height: "100%",
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                    }, children: [_jsx(Icon, { sx: { fill: "white" } }), _jsx(Typography, { gutterBottom: true, variant: "body2", component: "h3", sx: {
                                ...multiLineEllipsis(3),
                                textAlign: "center",
                                lineBreak: Boolean(title) ? "auto" : "anywhere",
                            }, children: firstString(title, data.link) })] })] }) }));
});
//# sourceMappingURL=ResourceCard.js.map