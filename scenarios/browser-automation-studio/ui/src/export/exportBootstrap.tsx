import React from 'react';
import ReactDOM from 'react-dom/client';
import ReplayExportPage from './ReplayExportPage';

export function mountReplayExport(target: HTMLElement): ReactDOM.Root {
  const root = ReactDOM.createRoot(target);
  root.render(
    <React.StrictMode>
      <ReplayExportPage />
    </React.StrictMode>,
  );
  return root;
}

export default mountReplayExport;
