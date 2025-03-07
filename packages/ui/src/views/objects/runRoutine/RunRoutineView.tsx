import { endpointsRunRoutine, RunRoutine } from "@local/shared";
import { TopBar } from "components/navigation/TopBar/TopBar.js";
import { useObjectActions } from "hooks/objectActions.js";
import { useManagedObject } from "hooks/useManagedObject.js";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getDisplay } from "utils/display/listTools.js";
import { firstString } from "utils/display/stringTools.js";
import { RunRoutineViewProps } from "./types.js";

export function RunRoutineView({
    display,
    onClose,
}: RunRoutineViewProps) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { object: existing, isLoading, setObject: setRunRoutine } = useManagedObject<RunRoutine>({
        ...endpointsRunRoutine.findOne,
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
