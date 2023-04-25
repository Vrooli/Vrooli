import { openLink, useLocation } from "@local/shared";
import {
    Breadcrumbs,
    Link
} from "@mui/material";
import { useMemo } from "react";
import { noSelect } from "styles";
import { BreadcrumbsBaseProps } from "../types";

export const BreadcrumbsBase = ({
    paths,
    separator = "|",
    ariaLabel = "breadcrumb",
    textColor,
    sx,
}: BreadcrumbsBaseProps) => {
    const [, setLocation] = useLocation();

    const pathLinks = useMemo(() => (
        paths.map(p => (
            <Link
                key={p.text}
                color={textColor}
                href={p.link}
                onClick={(e) => { e.preventDefault(); openLink(setLocation, p.link); }}
            >
                {window.location.pathname === p.link ? <b>{p.text}</b> : p.text}
            </Link>
        ))
    ), [setLocation, paths, textColor]);

    return (
        <Breadcrumbs
            sx={{
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
            }}
            separator={separator}
            aria-label={ariaLabel}
        >
            {pathLinks}
        </Breadcrumbs>
    );
};
