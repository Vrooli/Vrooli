import { APP_LINKS, REPORTABLE } from '@local/shared';
import { Session } from 'types';

export interface OrganizationViewPageProps {
    session: Session
}

export interface ProjectViewPageProps {
    session: Session
}

export interface ReportsViewPageProps {
    session: Session,
    type: REPORTABLE,
}

export interface RoutineViewPageProps {
    session: Session,
}

export interface StandardViewPageProps {
    session: Session
}

export interface UserViewPageProps {
    session: Session
}