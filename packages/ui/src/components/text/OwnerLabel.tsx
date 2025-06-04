import { getObjectUrl, getTranslation } from "@local/shared";
import { Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useContext, useMemo } from "react";
import { SessionContext } from "../../contexts/session.js";
import { useLocation } from "../../route/router.js";
import { firstString } from "../../utils/display/stringTools.js";
import { getUserLanguages } from "../../utils/display/translationTools.js";
import { type OwnerLabelProps } from "./types.js";

/**
 * Gets name of user or team that owns/created this object
 * @params owner Owner object
 * @params languages Languages preferred by user
 * @returns String of owner, or empty string if no owner
 */
function getLabel(
    owner: {
        __typename: "Team" | "User",
        handle?: string | null,
        name?: string | null,
        translations?: { language: string, name?: string }[],
    } | null | undefined,
    languages: readonly string[],
): string {
    if (!owner) return "";
    return firstString(owner.name, owner.handle, getTranslation(owner, languages, true).name);
}

export function OwnerLabel({
    confirmOpen,
    language,
    objectType,
    owner,
    sxs,
}: OwnerLabelProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const ownerLabel = useMemo(() => getLabel(owner, language ? [language] : getUserLanguages(session)), [language, owner, session]);

    // We set href and onClick so users can open in new tab, while also supporting single-page app navigation TODO not working
    const link = useMemo<string>(() => owner ? getObjectUrl(owner) : "", [owner]);
    const toOwner = useCallback(() => {
        if (link.length === 0) return;
        setLocation(link);
    }, [link, setLocation]);
    const onClick = useCallback((e: any) => {
        if (typeof confirmOpen === "function") {
            confirmOpen(toOwner);
        } else {
            toOwner();
        }
        // Prevent default so we don't use href
        e.preventDefault();
    }, [confirmOpen, toOwner]);

    return (
        <a
            href={link}
            onClick={onClick}
            style={{
                minWidth: "auto",
                padding: 0,
            }}
        >
            <Tooltip title={`Press to view ${objectType === "Standard" ? "creator" : "owner"}`}>
                <Typography
                    variant="body1"
                    sx={{
                        color: palette.primary.contrastText,
                        cursor: "pointer",
                        // Highlight text like a link
                        textDecoration: "none",
                        "&:hover": {
                            textDecoration: "underline",
                        },
                        ...(sxs?.label ?? {}),
                    }}
                >
                    {ownerLabel}
                </Typography>
            </Tooltip>
        </a>
    );
}
