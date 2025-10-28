import { mountApp } from './renderApp';

const container = document.getElementById('root');

if (!container) {
  throw new Error('Failed to locate root element for Browser Automation Studio UI');
}

mountApp(container, { strictMode: true });
