import { ReactNode, useState, useRef, useEffect } from 'react';
import { useLocation, useNavigate, Link } from 'react-router-dom';
import { LogOut, ChevronRight, ChevronDown } from 'lucide-react';
import { Button } from '../../../shared/ui/button';
import { adminLogout } from '../../../shared/api';
import { NAVIGATION_CONFIG } from '../config/navigation';
import { buildBreadcrumbs, isGroupActive } from '../config/navigation.utils';
import type { NavGroup } from '../config/navigation.types';
import { LAYOUT } from '../config/layout.constants';

export type MaxWidthPreset = 'narrow' | 'default' | 'wide' | 'extraWide' | 'full';

interface AdminLayoutProps {
  children: ReactNode;
  /** Content max-width preset. When set, constrains main content width. */
  maxWidth?: MaxWidthPreset;
}

// Dropdown component for nav groups
function NavDropdown({ group, currentPath, alignRight }: {
  group: NavGroup;
  currentPath: string;
  alignRight?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const Icon = group.icon;

  // Close on click outside
  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('click', handleClick);
    return () => document.removeEventListener('click', handleClick);
  }, []);

  const isActive = isGroupActive(group, currentPath);

  return (
    <div ref={ref} className="relative">
      <Button
        variant="ghost"
        size="sm"
        className={`gap-2 ${isActive ? 'bg-slate-800' : ''}`}
        onClick={() => setOpen(!open)}
        data-testid={`nav-group-${group.id}`}
      >
        <Icon className="h-4 w-4" />
        {group.label}
        <ChevronDown className={`h-3 w-3 transition-transform ${open ? 'rotate-180' : ''}`} />
      </Button>
      {open && (
        <div className={`absolute top-full ${alignRight ? 'right-0' : 'left-0'} mt-1 bg-slate-800 border border-slate-700 rounded-md shadow-lg py-1 z-50 min-w-[180px]`}>
          {group.items.map((item) => {
            const ItemIcon = item.icon;
            return (
              <Link
                key={item.path}
                to={item.path}
                onClick={() => setOpen(false)}
                data-testid={item.testId}
                className={`flex items-center gap-2 px-3 py-2 text-sm hover:bg-slate-700 transition-colors ${
                  currentPath.startsWith(item.path) ? 'text-blue-400' : 'text-slate-200'
                }`}
              >
                <ItemIcon className="h-4 w-4" />
                <span className="flex-1">{item.name}</span>
                {item.isStub && (
                  <span className="text-[10px] text-slate-500 uppercase">Soon</span>
                )}
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}

export function AdminLayout({ children, maxWidth }: AdminLayoutProps) {
  const location = useLocation();
  const navigate = useNavigate();

  const handleLogout = async () => {
    try {
      await adminLogout();
      navigate('/admin/login');
    } catch (error) {
      console.error('Logout failed:', error);
    }
  };

  const breadcrumbs = buildBreadcrumbs(location.pathname);

  // Compute max-width class from preset
  const maxWidthClass = maxWidth && maxWidth !== 'full'
    ? LAYOUT.maxWidth[maxWidth]
    : '';

  // Split groups into left-aligned and right-aligned
  const leftGroups = NAVIGATION_CONFIG.groups.filter(g => !g.rightAligned);
  const rightGroups = NAVIGATION_CONFIG.groups.filter(g => g.rightAligned);

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50">
      {/* Top Navigation Bar */}
      <header className="border-b border-white/10 bg-slate-900/50 backdrop-blur">
        <div className="container mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-6">
              <Link to="/admin" className="text-xl font-semibold hover:text-blue-400 transition-colors">
                Landing Manager
              </Link>
              <nav className="hidden md:flex gap-1">
                {/* Direct links */}
                {NAVIGATION_CONFIG.directLinks.map((link) => {
                  const LinkIcon = link.icon;
                  return (
                    <Link key={link.id} to={link.path} data-testid={link.testId}>
                      <Button variant="ghost" size="sm" className="gap-2">
                        <LinkIcon className="h-4 w-4" />
                        {link.name}
                      </Button>
                    </Link>
                  );
                })}
                {/* Left-aligned dropdown groups */}
                {leftGroups.map((group) => (
                  <NavDropdown
                    key={group.id}
                    group={group}
                    currentPath={location.pathname}
                  />
                ))}
              </nav>
            </div>
            <div className="flex items-center gap-1">
              {/* Right-aligned dropdown groups */}
              {rightGroups.map((group) => (
                <NavDropdown
                  key={group.id}
                  group={group}
                  currentPath={location.pathname}
                  alignRight
                />
              ))}
              <Button
                variant="ghost"
                size="sm"
                onClick={handleLogout}
                className="gap-2"
                data-testid="nav-logout"
              >
                <LogOut className="h-4 w-4" />
                Logout
              </Button>
            </div>
          </div>
        </div>
      </header>

      {/* Breadcrumb Navigation */}
      <div className="border-b border-white/5 bg-slate-900/30">
        <div className="container mx-auto px-6 py-3">
          <nav className="flex items-center gap-2 text-sm" data-testid="admin-breadcrumb">
            {breadcrumbs.map((segment, index) => (
              <div key={index} className="flex items-center gap-2">
                {index > 0 && <ChevronRight className="h-4 w-4 text-slate-500" />}
                {segment.path ? (
                  <Link
                    to={segment.path}
                    className="text-slate-400 hover:text-slate-200 transition-colors"
                    data-testid={`breadcrumb-${index}`}
                  >
                    {segment.label}
                  </Link>
                ) : (
                  <span className="text-slate-200" data-testid={`breadcrumb-${index}`}>
                    {segment.label}
                  </span>
                )}
              </div>
            ))}
          </nav>
        </div>
      </div>

      {/* Main Content */}
      <main className={`mx-auto px-6 py-8 ${maxWidthClass}`}>
        {children}
      </main>
    </div>
  );
}
