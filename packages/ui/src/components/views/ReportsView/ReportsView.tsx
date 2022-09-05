import {  useQuery } from "@apollo/client";
import { reports, reportsVariables } from "graphql/generated/reports";
import { reportsQuery } from "graphql/query";
import { APP_LINKS } from "@shared/consts";
import { useMemo } from "react";
import { useTheme } from "@mui/material";
import { getLastUrlPart } from "utils";

export const CommentReportsView = (): JSX.Element => {
    const { loading, error, data } = useReportsQuery(APP_LINKS.Comment, 'commentId');

    if (loading) {
        return <p>Loading reports...</p>
    }
    if (error) {
        console.error(error);
        return <p>Error</p>;
    }
    if (!data || !data.reports) {
        return <></>
    }

    return <>
        <BaseReportsView data={data} />
    </>
}

export const OrganizationReportsView = (): JSX.Element => {
    const { loading, error, data } = useReportsQuery(APP_LINKS.Organization, 'organizationId');

    if (loading) {
        return <p>Loading reports...</p>
    }
    if (error) {
        console.error(error);
        return <p>Error</p>;
    }
    if (!data || !data.reports) {
        return <></>
    }

    return <>
        <BaseReportsView data={data} />
    </>
}

export const ProjectReportsView = (): JSX.Element => {
    const { loading, error, data } = useReportsQuery(APP_LINKS.Project, 'projectId');

    if (loading) {
        return <p>Loading reports...</p>
    }
    if (error) {
        console.error(error);
        return <p>Error</p>;
    }
    if (!data || !data.reports) {
        return <></>
    }

    return <>
        <BaseReportsView data={data} />
    </>
}

export const RoutineReportsView = (): JSX.Element => {
    const { loading, error, data } = useReportsQuery(APP_LINKS.Routine, 'routineId');

    if (loading) {
        return <p>Loading reports...</p>
    }
    if (error) {
        console.error(error);
        return <p>Error</p>;
    }
    if (!data || !data.reports) {
        return <></>
    }

    return <>
        <BaseReportsView data={data} />
    </>
}

export const StandardReportsView = (): JSX.Element => {
    const { loading, error, data } = useReportsQuery(APP_LINKS.Standard, 'standardId');

    if (loading) {
        return <p>Loading reports...</p>
    }
    if (error) {
        console.error(error);
        return <p>Error</p>;
    }
    if (!data || !data.reports) {
        return <></>
    }

    return <>
        <BaseReportsView data={data} />
    </>
}

export const TagReportsView = (): JSX.Element => {
    const { loading, error, data } = useReportsQuery(APP_LINKS.Tag, 'tagId');

    if (loading) {
        return <p>Loading reports...</p>
    }
    if (error) {
        console.error(error);
        return <p>Error</p>;
    }
    if (!data || !data.reports) {
        return <></>
    }

    return <>
        <BaseReportsView data={data} />
    </>
}

export const UserReportsView = (): JSX.Element => {
    const { loading, error, data } = useReportsQuery(APP_LINKS.User, 'userId');

    if (loading) {
        return <p>Loading reports...</p>
    }
    if (error) {
        console.error(error);
        return <p>Error</p>;
    }
    if (!data || !data.reports) {
        return <></>
    }

    return <>
        <BaseReportsView data={data} />
    </>
}

function useReportsQuery(appLink: string, queryField: string) {
    const id = useMemo(() => getLastUrlPart(), []);

    return useQuery<reports, reportsVariables>(
        reportsQuery,
        {
            variables: {
                input: { [queryField]: id },
            },
        },
    );
}

const BaseReportsView = (props: { data: reports }): JSX.Element => {
    const { palette } = useTheme();
    const edges = props.data.reports.edges;
    return <>
        {edges.map((edge, i) => {
            const report = edge.node;
            return <div 
                key={i} 
                style={{ 
                    background: palette.background.paper,
                    color: palette.background.textPrimary,
                    borderRadius: "16px",
                    boxShadow: "0 4px 16px 0 #00000050", 
                    margin: "16px 16px 0 16px",
                    padding: "1rem",
                }}
            >
                <p style={{ margin: "0" }}>
                    <b>Reason:</b> {report.reason}
                </p>
                <p style={{ margin: "1rem 0 0 0" }}>
                    <b>Details:</b>  {report.details}
                </p>
            </div>
        })}
    </>;
}