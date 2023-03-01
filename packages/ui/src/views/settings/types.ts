import { Session, User } from "@shared/consts";

interface SettingsBaseProps {
    profile: User | undefined;
    onUpdated: (profile: User) => void;
    session: Session | undefined;
    zIndex: number;
}
export interface SettingsAuthenticationProps extends SettingsBaseProps {}
export interface SettingsDisplayProps extends SettingsBaseProps {}
export interface SettingsNotificationsProps extends SettingsBaseProps {}
export interface SettingsProfileProps extends SettingsBaseProps {}