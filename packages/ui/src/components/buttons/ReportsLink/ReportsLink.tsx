import { Link } from "@mui/material";

/**
 * Renders a link that says 'Reports (x)' where x is the number of reports.
 * When clicked, navigates to href.
 */
export const ReportsLink = (props: { href?: string, reports?: number }): JSX.Element => {
    return <Link
        href={props.href}
        underline="hover"
    >
        Reports ({props.reports})
    </Link>
}