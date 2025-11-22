import { Link, Outlet, useLocation } from "react-router-dom";
import {
  Layers,
  Code2,
  GitBranch,
  Moon,
  Sun,
  Search,
} from "lucide-react";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { useUIStore } from "../store/ui-store";
import { cn } from "../lib/utils";

export function Layout() {
  const location = useLocation();
  const { theme, toggleTheme, searchQuery, setSearchQuery } = useUIStore();

  const navItems = [
    { path: "/", label: "Library", icon: Layers },
    { path: "/adoptions", label: "Adoptions", icon: GitBranch },
  ];

  return (
    <div className="flex h-screen flex-col bg-slate-950 text-slate-50">
      {/* Header */}
      <header className="border-b border-white/10 bg-slate-900/50 backdrop-blur">
        <div className="flex h-16 items-center px-6">
          <div className="flex items-center space-x-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-600">
              <Code2 className="h-5 w-5" />
            </div>
            <div>
              <h1 className="text-lg font-semibold">Component Library</h1>
              <p className="text-xs text-slate-400">Design, preview, and track shared components</p>
            </div>
          </div>

          <div className="ml-auto flex items-center space-x-4">
            {/* Search - only show on library page */}
            {location.pathname === "/" && (
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                <Input
                  type="search"
                  placeholder="Search components..."
                  className="w-64 pl-9"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                />
              </div>
            )}

            <Button
              variant="ghost"
              size="icon"
              onClick={toggleTheme}
              aria-label="Toggle theme"
            >
              {theme === "dark" ? (
                <Sun className="h-5 w-5" />
              ) : (
                <Moon className="h-5 w-5" />
              )}
            </Button>
          </div>
        </div>

        {/* Navigation */}
        <nav className="flex space-x-1 px-6">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = location.pathname === item.path;
            return (
              <Link key={item.path} to={item.path}>
                <Button
                  variant="ghost"
                  className={cn(
                    "flex items-center space-x-2 rounded-b-none",
                    isActive &&
                      "border-b-2 border-blue-500 bg-slate-800/50 text-blue-400"
                  )}
                >
                  <Icon className="h-4 w-4" />
                  <span>{item.label}</span>
                </Button>
              </Link>
            );
          })}
        </nav>
      </header>

      {/* Main Content */}
      <main className="flex-1 overflow-hidden">
        <Outlet />
      </main>
    </div>
  );
}
