import { endpointGetRunProject, RunProject } from "@local/shared";
import { useTheme } from "@mui/material";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getDisplay } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { firstString } from "utils/display/stringTools";
import { RunProjectViewProps } from "../types";

export const RunProjectView = ({
    isOpen,
    onClose,
    zIndex,
}: RunProjectViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const display = toDisplay(isOpen);

    const { object: existing, isLoading, setObject: setRunProject } = useObjectFromUrl<RunProject>({
        ...endpointGetRunProject,
        objectType: "RunProject",
    });

    const { title } = useMemo(() => ({ title: getDisplay(existing).title ?? "" }), [existing]);

    const actionData = useObjectActions({
        object: existing,
        objectType: "RunProject",
        setLocation,
        setObject: setRunProject,
    });

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={firstString(title, t("Run", { count: 1 }))}
                zIndex={zIndex}
            />
            <>
                {/* TODO */}
            </>
        </>
    );
};
