import { ReactNode } from 'react';
import { useLocation, useNavigate, Link } from 'react-router-dom';
import { Home, BarChart3, Palette, LogOut, ChevronRight, CreditCard, Download, Settings2 } from 'lucide-react';
import { Button } from '../../../shared/ui/button';
import { adminLogout } from '../../../shared/api';
import { RuntimeSignalStrip } from './RuntimeSignalStrip';

interface AdminLayoutProps {
  children: ReactNode;
}

interface BreadcrumbSegment {
  label: string;
  path?: string;
}

export function AdminLayout({ children }: AdminLayoutProps) {
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

  const getBreadcrumbs = (): BreadcrumbSegment[] => {
    const path = location.pathname;
    const segments: BreadcrumbSegment[] = [{ label: 'Admin', path: '/admin' }];

    if (path.startsWith('/admin/analytics')) {
      segments.push({ label: 'Analytics', path: '/admin/analytics' });

      const variantMatch = path.match(/\/admin\/analytics\/(.+)/);
      if (variantMatch) {
        segments.push({ label: `Variant ${variantMatch[1]}` });
      }
    } else if (path.startsWith('/admin/customization')) {
      segments.push({ label: 'Customization', path: '/admin/customization' });

      if (path.includes('/variants/new')) {
        segments.push({ label: 'New Variant' });
      } else {
        const variantMatch = path.match(/\/variants\/([^/]+)/);
        if (variantMatch) {
          segments.push({ label: `Variant ${variantMatch[1]}`, path: `/admin/customization/variants/${variantMatch[1]}` });

          const sectionMatch = path.match(/\/sections\/(\d+)/);
          if (sectionMatch) {
            segments.push({ label: `Section ${sectionMatch[1]}` });
          }
        }
      }
    } else if (path.startsWith('/admin/billing')) {
      segments.push({ label: 'Billing', path: '/admin/billing' });
    } else if (path.startsWith('/admin/downloads')) {
      segments.push({ label: 'Downloads', path: '/admin/downloads' });
    } else if (path.startsWith('/admin/branding')) {
      segments.push({ label: 'Branding', path: '/admin/branding' });
    }

    return segments;
  };

  const breadcrumbs = getBreadcrumbs();

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
                <Link to="/admin" data-testid="nav-home">
                  <Button variant="ghost" size="sm" className="gap-2">
                    <Home className="h-4 w-4" />
                    Home
                  </Button>
                </Link>
                <Link to="/admin/analytics" data-testid="nav-analytics">
                  <Button variant="ghost" size="sm" className="gap-2">
                    <BarChart3 className="h-4 w-4" />
                    Analytics
                  </Button>
                </Link>
                <Link to="/admin/customization" data-testid="nav-customization">
                  <Button variant="ghost" size="sm" className="gap-2">
                    <Palette className="h-4 w-4" />
                    Customization
                  </Button>
                </Link>
                <Link to="/admin/billing" data-testid="nav-billing">
                  <Button variant="ghost" size="sm" className="gap-2">
                    <CreditCard className="h-4 w-4" />
                    Billing
                  </Button>
                </Link>
                <Link to="/admin/downloads" data-testid="nav-downloads">
                  <Button variant="ghost" size="sm" className="gap-2">
                    <Download className="h-4 w-4" />
                    Downloads
                  </Button>
                </Link>
                <Link to="/admin/branding" data-testid="nav-branding">
                  <Button variant="ghost" size="sm" className="gap-2">
                    <Settings2 className="h-4 w-4" />
                    Branding
                  </Button>
                </Link>
              </nav>
            </div>
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
      <main className="container mx-auto px-6 py-8">
        <RuntimeSignalStrip />
        {children}
      </main>
    </div>
  );
}
