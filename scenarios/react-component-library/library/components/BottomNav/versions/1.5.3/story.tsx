import { BottomNav } from "./BottomNav";
import { createElement } from "react";
import {
  Activity,
  CreditCard,
  Home,
  LayoutDashboard,
  ListTodo,
  Plug,
  Settings,
  Users,
  Workflow,
} from "lucide-react";
export function MobilePrimary({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(BottomNav, {
    ...args,
    items: [
      {
        active: true,
        href: "#home",
        icon: createElement(Home, { "aria-hidden": true }),
        id: "home",
        label: "Home",
      },
      {
        href: "#tasks",
        icon: createElement(ListTodo, { "aria-hidden": true }),
        id: "tasks",
        label: "Tasks",
      },
      {
        href: "#settings",
        icon: createElement(Settings, { "aria-hidden": true }),
        id: "settings",
        label: "Settings",
      },
    ],
  } as never);
}

export function DisabledItem({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(BottomNav, {
    ...args,
    items: [
      {
        active: true,
        icon: createElement(LayoutDashboard, { "aria-hidden": true }),
        id: "overview",
        label: "Overview",
      },
      {
        disabled: true,
        icon: createElement(CreditCard, { "aria-hidden": true }),
        id: "billing",
        label: "Billing",
      },
      {
        icon: createElement(Users, { "aria-hidden": true }),
        id: "team",
        label: "Team",
      },
    ],
  } as never);
}

export function LongLabels({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(BottomNav, {
    ...args,
    items: [
      {
        active: true,
        icon: createElement(Activity, { "aria-hidden": true }),
        id: "activity",
        label: "Activity",
      },
      {
        icon: createElement(Workflow, { "aria-hidden": true }),
        id: "automations",
        label: "Automations",
      },
      {
        icon: createElement(Plug, { "aria-hidden": true }),
        id: "integrations",
        label: "Integrations",
      },
    ],
  } as never);
}
