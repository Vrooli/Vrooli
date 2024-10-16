import { endpointGetRunRoutine, RunRoutine } from "@local/shared";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useObjectActions } from "hooks/objectActions";
import { useManagedObject } from "hooks/useManagedObject";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { RunRoutineViewProps } from "../types";

export function RunRoutineView({
    display,
    onClose,
}: RunRoutineViewProps) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { object: existing, isLoading, setObject: setRunRoutine } = useManagedObject<RunRoutine>({
        ...endpointGetRunRoutine,
        objectType: "RunRoutine",
    });

    const { title } = useMemo(() => ({ title: getDisplay(existing).title ?? "" }), [existing]);

    const actionData = useObjectActions({
        object: existing,
        objectType: "RunRoutine",
        setLocation,
        setObject: setRunRoutine,
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
