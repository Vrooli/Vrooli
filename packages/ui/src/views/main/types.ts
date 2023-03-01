import { Session } from "@shared/consts";

export interface CreateViewProps {
    session: Session | undefined;
}

export interface HomeViewProps {
    session: Session | undefined;
}

export interface HistoryViewProps {
    session: Session | undefined;
}

export interface NotificationsViewProps {
    session: Session | undefined;
}

export interface SettingsViewProps {
    session: Session | undefined;
}