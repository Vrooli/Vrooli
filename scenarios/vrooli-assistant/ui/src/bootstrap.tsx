import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';

export function bootstrap(container: HTMLElement): void {
  ReactDOM.createRoot(container).render(
    <React.StrictMode>
      <App />
    </React.StrictMode>,
  );
}
