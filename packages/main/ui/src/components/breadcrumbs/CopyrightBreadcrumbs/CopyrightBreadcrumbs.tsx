import { BUSINESS_NAME, LINKS } from "@local/shared;";
import { useTranslation } from "react-i18next";
import { BreadcrumbsBase } from "../BreadcrumbsBase/BreadcrumbsBase";
import { CopyrightBreadcrumbsProps } from "../types";

export const CopyrightBreadcrumbs = ({
    sx,
    ...props
}: CopyrightBreadcrumbsProps) => {
    const { t } = useTranslation();
    return BreadcrumbsBase({
        paths: [
            [`© ${new Date().getFullYear()} ${BUSINESS_NAME}`, LINKS.Home],
            [t("Privacy"), LINKS.Privacy],
            [t("Terms"), LINKS.Terms],
        ].map(row => ({ text: row[0], link: row[1] })),
        ariaLabel: "Copyright breadcrumb",
        sx: {
            ...sx,
            display: "flex",
            justifyContent: "center",
            alignItems: "center",
        },
        ...props,
    });
};
