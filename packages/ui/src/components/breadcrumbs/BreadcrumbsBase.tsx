import { Breadcrumbs, Link } from "@mui/material";
import { useMemo } from "react";
import { openLink } from "../../route/openLink.js";
import { useLocation } from "../../route/router.js";
import { noSelect } from "../../styles.js";
import { type BreadcrumbsBaseProps } from "./types.js";

const DEFAULT_SEPARATOR = "|";
const DEFAULT_ARIA_LABEL = "breadcrumb";

export function BreadcrumbsBase({
    ariaLabel = DEFAULT_ARIA_LABEL,
    onClick,
    paths,
    separator = DEFAULT_SEPARATOR,
    sx,
    textColor,
}: BreadcrumbsBaseProps) {
    const [, setLocation] = useLocation();

    const pathLinks = useMemo(() => (
        paths.map((p) => {
            function handleClick(e: React.MouseEvent<HTMLAnchorElement>) {
                onClick?.(e);
                e.preventDefault();
                openLink(setLocation, p.link);
            }

            return (
                <Link
                    key={p.text}
                    color={textColor}
                    href={p.link}
                    onClick={handleClick}
                >
                    {window.location.pathname === p.link ? <b>{p.text}</b> : p.text}
                </Link>
            );
        })
    ), [paths, textColor, onClick, setLocation]);

    const breadcrumbsStyle = useMemo(function breadcrumbsStyleMemo() {
        return {
            ...sx,
            "& .MuiBreadcrumbs-li > a": {
                color: sx?.color || "inherit",
                minHeight: "48px", // Lighthouse recommends this for SEO, as it is more clickable
                display: "flex",
                flexDirection: "row",
                alignItems: "center",
                cursor: "pointer",
                ...noSelect,
            },
        } as const;
    }, [sx]);

    return (
        <Breadcrumbs
            sx={breadcrumbsStyle}
            separator={separator}
            aria-label={ariaLabel}
        >
            {pathLinks}
        </Breadcrumbs>
    );
}
