import { Link, Tooltip } from "@mui/material";
import { getObjectSlug, getObjectUrlBase } from "utils";
import { ReportsLinkProps } from "../types";

/**
 * Renders a link that says 'Reports (x)' where x is the number of reports.
 * When clicked, navigates to href.
 */
export const ReportsLink = ({
    object,
}: ReportsLinkProps) => {
    if (!object?.reportsCount || object.reportsCount <= 0) return null;
    return (
        <Tooltip title="Press to view and repond to reports.">
            <Link
                href={`${getObjectUrlBase(object)}/reports/${getObjectSlug(object)}`}
                underline="hover"
            >
                Reports ({object.reportsCount})
            </Link>
        </Tooltip>
    )
}