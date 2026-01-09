import { mountReplayExport } from './exportBootstrap';

document.addEventListener('DOMContentLoaded', () => {
  const container = document.getElementById('root');
  if (!container) {
    throw new Error('Replay composer root element missing');
  }
  mountReplayExport(container);
});
