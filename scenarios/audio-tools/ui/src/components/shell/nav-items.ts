import { LayoutDashboard, Mic, Activity, Sliders, Volume2, BarChart3, BookOpen, UserCheck, Radio, Waves, type LucideIcon } from "lucide-react";
import { strings } from "../../consts/strings";

type NavKey = typeof strings.nav[keyof typeof strings.nav];

export interface NavItem {
  to: string;
  /** i18n key path consumed via `t(item.labelKey)` at render time. */
  labelKey: NavKey;
  icon: LucideIcon;
  /** Mobile bottom-nav slot. Items without this are desktop-sidebar only. */
  mobile?: boolean;
}

export const NAV_ITEMS: NavItem[] = [
  { to: "/", labelKey: strings.nav.overview, icon: LayoutDashboard, mobile: true },
  { to: "/diagnostics", labelKey: strings.nav.diagnostics, icon: Mic, mobile: true },
  { to: "/status", labelKey: strings.nav.status, icon: Activity, mobile: true },
  { to: "/configuration", labelKey: strings.nav.configure, icon: Sliders, mobile: true },
  { to: "/voices", labelKey: strings.nav.voices, icon: Volume2 },
  { to: "/usage", labelKey: strings.nav.usage, icon: BarChart3, mobile: true },
  { to: "/admin/speaker-verification", labelKey: strings.nav.speakerVerification, icon: UserCheck },
  { to: "/admin/wake-word", labelKey: strings.nav.wakeWord, icon: Radio },
  { to: "/admin/stream-config", labelKey: strings.nav.streamConfig, icon: Waves },
  { to: "/docs", labelKey: strings.nav.docs, icon: BookOpen },
];
