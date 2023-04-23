import { jsx as _jsx } from "react/jsx-runtime";
import { Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useContext, useMemo } from "react";
import { firstString } from "../../../utils/display/stringTools";
import { getTranslation, getUserLanguages } from "../../../utils/display/translationTools";
import { getObjectUrl } from "../../../utils/navigation/openObject";
import { useLocation } from "../../../utils/route";
import { SessionContext } from "../../../utils/SessionContext";
const getLabel = (owner, languages) => {
    if (!owner)
        return "";
    return firstString(owner.name, owner.handle, getTranslation(owner, languages, true).name);
};
export const OwnerLabel = ({ confirmOpen, language, objectType, owner, sxs, }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const ownerLabel = useMemo(() => getLabel(owner, language ? [language] : getUserLanguages(session)), [language, owner, session]);
    const link = useMemo(() => owner ? getObjectUrl(owner) : "", [owner]);
    const toOwner = useCallback(() => {
        if (link.length === 0)
            return;
        setLocation(link);
    }, [link, setLocation]);
    const onClick = useCallback((e) => {
        if (typeof confirmOpen === "function") {
            confirmOpen(toOwner);
        }
        else {
            toOwner();
        }
        e.preventDefault();
    }, [confirmOpen, toOwner]);
    return (_jsx("a", { href: link, onClick: onClick, style: {
            minWidth: "auto",
            padding: 0,
        }, children: _jsx(Tooltip, { title: `Press to view ${objectType === "Standard" ? "creator" : "owner"}`, children: _jsx(Typography, { variant: "body1", sx: {
                    color: palette.primary.contrastText,
                    cursor: "pointer",
                    textDecoration: "none",
                    "&:hover": {
                        textDecoration: "underline",
                    },
                    ...(sxs?.label ?? {}),
                }, children: ownerLabel }) }) }));
};
//# sourceMappingURL=OwnerLabel.js.map