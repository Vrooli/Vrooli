import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import FactoryHome from './pages/FactoryHome';

export default function App() {
  return (
    <BrowserRouter>
      {/* [REQ:A11Y-SKIP] Skip to main content link for keyboard navigation */}
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4 focus:z-50 focus:px-6 focus:py-3 focus:bg-emerald-600 focus:text-white focus:rounded-lg focus:shadow-2xl focus:font-semibold focus:text-base"
        tabIndex={0}
      >
        Skip to main content
      </a>

      <Routes>
        {/* Factory home (default entrypoint) */}
        <Route path="/" element={<FactoryHome />} />
        <Route path="/health" element={<SimpleHealth />} />

        {/* Preview disabled inside factory to avoid scope drift */}
        <Route path="/preview/*" element={<PreviewPlaceholder />} />
        <Route path="/admin/*" element={<Navigate to="/" replace />} />

        {/* 404 redirect */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}

function SimpleHealth() {
  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex items-center justify-center">
      <div className="text-center space-y-2">
        <div className="text-sm text-slate-400">landing-manager ui</div>
        <div className="text-xl font-semibold">healthy</div>
      </div>
    </div>
  );
}

function PreviewPlaceholder() {
  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex items-center justify-center px-6">
      <div className="max-w-xl text-center space-y-4">
        <p className="text-sm text-amber-200/80">Template preview moved</p>
        <h2 className="text-2xl font-semibold">Preview lives in the generated landing scenario</h2>
        <p className="text-slate-300">
          To review the public landing and admin portal, generate a scenario and run it directly
          (e.g. <code className="px-1 py-0.5 rounded bg-slate-900">landing-manager generate saas-landing-page --name "demo" --slug "demo"</code> then <code className="px-1 py-0.5 rounded bg-slate-900">cd generated/demo && make start</code>).
        </p>
      </div>
    </div>
  );
}
