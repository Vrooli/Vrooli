import { RunStatus } from "@local/shared";
import { CompletionBar } from "components/CompletionBar/CompletionBar";
import { useMemo } from "react";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { RunRoutineListItemProps } from "../types";

export function RunRoutineListItem({
    data,
    loading,
    ...props
}: RunRoutineListItemProps) {

    /**
     * Run list items may get a progress bar
     */
    const progressBar = useMemo(() => {
        const completedComplexity = data?.completedComplexity ?? 0;
        const totalComplexity = data?.routineVersion?.complexity ?? null;
        const percentComplete = data?.status === RunStatus.Completed ? 100 :
            (completedComplexity && totalComplexity) ?
                Math.min(Math.round(completedComplexity / totalComplexity * 100), 100) :
                0;
        return (<CompletionBar
            color="secondary"
            variant={loading ? "indeterminate" : "determinate"}
            value={percentComplete}
            sx={{ height: "15px" }}
        />);
    }, [data?.completedComplexity, data?.routineVersion?.complexity, data?.status, loading]);

    return (
        <ObjectListItemBase
            {...props}
            belowSubtitle={progressBar}
            data={data}
            loading={loading}
            objectType="RunRoutine"
        />
    );
}
