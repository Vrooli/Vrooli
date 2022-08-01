import { ReportsViewProps } from "../types";
import { useRoute } from "wouter";
import { APP_LINKS, VoteFor } from "@local/shared";

export const ReportsView = ({
    session
}: ReportsViewProps) => {
    const [, params] = useRoute(`${APP_LINKS.Routine}/reports/:id`);
    const id = params?.id;
    console.log(id);

    return <p>Reports</p>
}