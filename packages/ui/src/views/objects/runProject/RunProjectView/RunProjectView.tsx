import { endpointGetRunProject, RunProject } from "@local/shared";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useObjectActions } from "hooks/objectActions";
import { useManagedObject } from "hooks/useManagedObject";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { RunProjectViewProps } from "../types";

export function RunProjectView({
    display,
    onClose,
}: RunProjectViewProps) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { object: existing, isLoading, setObject: setRunProject } = useManagedObject<RunProject>({
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
            />
            <>
                {/* TODO */}
            </>
        </>
    );
}
