import {
  Activity,
  BarChart3,
  ClipboardList,
  Play,
  Search,
  Settings2,
} from "lucide-react";

export type NavSection =
  | "dashboard"
  | "profiles"
  | "tasks"
  | "runs"
  | "investigations"
  | "stats";

interface MobileNavProps {
  activeSection: NavSection;
  onSectionChange: (section: NavSection) => void;
}

const navItems: Array<{
  id: NavSection;
  label: string;
  icon: typeof Activity;
}> = [
  { id: "dashboard", label: "Dashboard", icon: Activity },
  { id: "profiles", label: "Profiles", icon: Settings2 },
  { id: "tasks", label: "Tasks", icon: ClipboardList },
  { id: "runs", label: "Runs", icon: Play },
  { id: "investigations", label: "Investigate", icon: Search },
  { id: "stats", label: "Stats", icon: BarChart3 },
];

export function MobileNav({ activeSection, onSectionChange }: MobileNavProps) {
  return (
    <nav
      className="fixed bottom-0 left-0 right-0 z-40 border-t border-border bg-background/95 backdrop-blur-sm pb-safe"
      data-testid="mobile-nav"
    >
      <div className="flex items-center justify-around h-16">
        {navItems.map((item) => {
          const isActive = activeSection === item.id;
          const Icon = item.icon;

          return (
            <button
              key={item.id}
              type="button"
              onClick={() => onSectionChange(item.id)}
              className={`relative flex flex-col items-center justify-center gap-1 px-2 py-2 min-w-[48px] transition-colors ${
                isActive
                  ? "text-primary"
                  : "text-muted-foreground hover:text-foreground active:text-foreground"
              }`}
              aria-label={item.label}
              aria-current={isActive ? "page" : undefined}
              data-testid={`mobile-nav-${item.id}`}
            >
              <div className="relative">
                <Icon className="h-5 w-5" />
              </div>
              <span className="text-[10px] font-medium">{item.label}</span>
              {isActive && (
                <span
                  className="absolute bottom-0 left-1/2 -translate-x-1/2 w-8 h-0.5 rounded-full bg-primary"
                  aria-hidden="true"
                />
              )}
            </button>
          );
        })}
      </div>
    </nav>
  );
}
