import { ReportsViewProps } from "../types";
import { useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { OperationVariables, useQuery } from "@apollo/client";
import { reports, reportsVariables } from "graphql/generated/reports";
import { reportsQuery } from "graphql/query";

export const ReportsView = ({
    session,
    type,
}: ReportsViewProps): JSX.Element => {
    switch (type) {
        case 'comment':
            return <CommentReportsView />
        case 'organization':
            return <OrganizationReportsView />
        case 'project':
            return <ProjectReportsView />
        case 'routine':
            return <RoutineReportsView />
        case 'standard':
            return <StandardReportsView />
        case 'tag':
            return <TagReportsView />
        case 'user':
            return <UserReportsView />
        default:
            console.error('Invalid type in ReportsView');
            return <></>
    }
}

const CommentReportsView = (): JSX.Element => {
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

const OrganizationReportsView = (): JSX.Element => {
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

const ProjectReportsView = (): JSX.Element => {
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

const RoutineReportsView = (): JSX.Element => {
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

const StandardReportsView = (): JSX.Element => {
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

const TagReportsView = (): JSX.Element => {
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

const UserReportsView = (): JSX.Element => {
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
    const [, params] = useRoute(`${appLink}/reports/:id`);
    const id = params?.id;
    console.log(id);

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
    const edges = props.data.reports.edges;
    return <>
        {edges.map((edge, i) => {
            const report = edge.node;
            return <div 
                key={i} 
                style={{ 
                    background: "white",
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