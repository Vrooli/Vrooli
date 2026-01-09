import type { Root } from 'react-dom/client';
import mountApp from './renderApp';

export interface BootstrapOptions {
  /**
   * Render the application inside React.StrictMode. Defaults to false for embedded contexts
   * where double invocation of lifecycle effects can be disruptive.
   */
  strictMode?: boolean;
}

export function bootstrap(container: HTMLElement, options: BootstrapOptions = {}): Root {
  if (!container) {
    throw new Error('A valid container element is required to bootstrap Vrooli Ascension');
  }

  return mountApp(container, {
    strictMode: options.strictMode ?? false,
  });
}

export default bootstrap;
