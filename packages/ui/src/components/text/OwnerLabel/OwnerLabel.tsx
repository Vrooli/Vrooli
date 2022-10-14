import { getObjectSlug, getObjectUrlBase, getTranslation, getUserLanguages, ObjectType, openObject } from "utils";
import { Tooltip, Typography, useTheme } from "@mui/material"
import { OwnerLabelProps } from "../types";
import { Comment, Project, Routine, Standard, User } from "types";
import { Link, useLocation } from "@shared/route";
import { useCallback, useMemo } from "react";

/**
 * Gets name of user or organization that owns/created this object
 * @params owner Owner object
 * @params languages Languages preferred by user
 * @returns String of owner, or empty string if no owner
 */
const getLabel = (
    owner: Comment['creator'] | Project['owner'] | Routine['owner'] | Standard['creator'] | null | undefined,
    languages: readonly string[]
): string => {
    console.log('getting label', owner, languages)
    if (!owner) return '';
    // Check if user or organization. Only users have a non-translated name
    if (owner.__typename === 'User' || owner.hasOwnProperty('name')) {
        return (owner as User).name ?? owner.handle ?? '';
    } else {
        return getTranslation(owner, 'name', languages, true) ?? owner.handle ?? '';
    }
}

export const OwnerLabel = ({
    confirmOpen,
    language,
    objectType,
    owner,
    session,
    sxs,
}: OwnerLabelProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const ownerLabel = useMemo(() => getLabel(owner, language ? [language] : getUserLanguages(session)), [language, owner, session]);

    const ownerLink = useMemo<string>(() => {
        if (!owner) return '';
        return `${getObjectUrlBase(owner)}/${getObjectSlug(owner)}`
    }, [owner]);

    const toOwner = useCallback(() => { 
        if (!owner) return;
        openObject(owner, setLocation); 
    }, [owner, setLocation]);

    const handleClick = useCallback((event: any) => {
        if (typeof confirmOpen === 'function') {
            event.preventDefault();
            confirmOpen(toOwner);
        }
    }, [confirmOpen, toOwner]);

    return (
        <Link
            href={ownerLink}
            onClick={handleClick}
            style={{
                minWidth: 'auto',
                padding: 0,
            }}
        >
            <Tooltip title={`Press to view ${objectType === ObjectType.Standard ? 'creator' : 'owner'}`}>
                <Typography
                    variant="body1"
                    sx={{
                        color: palette.primary.contrastText,
                        cursor: 'pointer',
                        // Highlight text like a link
                        textDecoration: 'none',
                        '&:hover': {
                            textDecoration: 'underline',
                        },
                        ...(sxs?.label ?? {}),
                    }}
                >
                    {ownerLabel}
                </Typography>
            </Tooltip>
        </Link>
    )
}