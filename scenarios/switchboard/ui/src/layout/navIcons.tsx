import { LayoutGrid, MessagesSquare, Bot, Users, Radio, Settings } from "lucide-react";
import type { ReactNode } from "react";

import type { NavItem } from "./navItems";

/** One icon per nav key; shared by the sidebar and the bottom nav. */
export function navIcon(key: NavItem["key"], className = "h-5 w-5"): ReactNode {
  switch (key) {
    case "dashboard":
      return <LayoutGrid aria-hidden className={className} />;
    case "conversations":
      return <MessagesSquare aria-hidden className={className} />;
    case "agents":
      return <Bot aria-hidden className={className} />;
    case "contacts":
      return <Users aria-hidden className={className} />;
    case "channels":
      return <Radio aria-hidden className={className} />;
    case "settings":
      return <Settings aria-hidden className={className} />;
  }
}
