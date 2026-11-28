import React from 'react';
import { createRoot } from 'react-dom/client';
import '../../../src/index.css';
import { AIPrivacyConsentSection } from '../../../src/views/SettingsView/sections/AIPrivacyConsentSection';
import { analyzeElementsWithAI, newGatewayRootOperationID } from '../../../src/api/ai';

function Fixture() {
  const [result, setResult] = React.useState('Ready to send a confidential test workflow request.');
  const [secret, setSecret] = React.useState(false);
  const send = async () => {
    try {
      await analyzeElementsWithAI({ url: 'https://fixture.invalid', intent: secret ? 'SECRET_SENTINEL' : 'confidential workflow fixture' }, newGatewayRootOperationID());
      setResult('Dispatched: fake-remote');
    } catch (error) {
      setResult(`Rejected: ${error instanceof Error ? error.message : String(error)}`);
    }
  };
  return <main style={{ maxWidth: 720, margin: '2rem auto', fontFamily: 'sans-serif', color: '#eee', background: '#171717', padding: 24 }}>
    <h1>Browser Automation Studio AI settings</h1>
    <AIPrivacyConsentSection />
    <label style={{ display: 'block', marginTop: 20 }}><input data-testid="secret-toggle" type="checkbox" checked={secret} onChange={event => setSecret(event.target.checked)} /> Send a SECRET marker for the local-only check</label>
    <button data-testid="send-ai" onClick={() => void send()}>Send confidential test workflow request</button>
    <p data-testid="request-result" role="status">{result}</p>
  </main>;
}

createRoot(document.getElementById('root')!).render(<Fixture />);
