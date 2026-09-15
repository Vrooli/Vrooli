/** Explicit standalone development entry; never imported by the application. */
import { createRoot } from 'react-dom/client';
import { PresentationPage } from './PresentationPage';
import { parseProductPresentation } from './decode';
import signal from './fixtures/signal.json';
import studio from './fixtures/studio.json';
const fixture = new URLSearchParams(window.location.search).get('design') === 'studio' ? studio : signal;
const mount = document.getElementById('presentation-preview');
if (!mount) throw new Error('Missing presentation preview mount');
createRoot(mount).render(<PresentationPage presentation={parseProductPresentation(JSON.stringify(fixture.presentation))} />);
