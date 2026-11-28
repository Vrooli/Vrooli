import React from 'react';
import { createRoot } from 'react-dom/client';
import '../../src/index.css';
import { AIPrivacyConsentSection } from '../../src/views/SettingsView/sections/AIPrivacyConsentSection';
import { AI_CONSENT_RECEIPT_KEY, AI_CONSENT_SOURCE_KEY } from '../../src/api/aiPrivacyConsent';
import { analyzeElementsWithAI, newGatewayRootOperationID } from '../../src/api/ai';
import { visionNavigationClient } from '../../src/api/visionNavigation';

function Fixture() {
  const caller = new URLSearchParams(window.location.search).get('caller') || '';
  const productionBASCaller = caller === 'bas';
  const productionVisionCaller = caller === 'vision';
  const [result, setResult] = React.useState('Ready.');
  const [secret, setSecret] = React.useState(false);
  const [visionSessionID, setVisionSessionID] = React.useState('');
  React.useEffect(() => {
    if (!productionVisionCaller) return;
    void fetch('/__fixture/bas/session').then(async response => {
      if (!response.ok) throw new Error(`fixture session failed (${response.status})`);
      const body = await response.json() as { session_id: string };
      setVisionSessionID(body.session_id);
    }).catch(error => setResult(`Rejected: ${error instanceof Error ? error.message : String(error)}`));
  }, [productionVisionCaller]);
  const send = async () => {
    setResult('Sending…');
    if (productionVisionCaller && !secret) {
      try {
        if (!visionSessionID) throw new Error('fixture Playwright session is not ready');
        const started = await visionNavigationClient.startNavigation({
          sessionId: visionSessionID,
          prompt: 'Use the page and finish without changing anything.',
          model: 'remote_only',
          maxSteps: 1,
          clientSource: 'ui',
          effectPolicy: 'read_only',
        });
        setResult(`Navigation started: ${started.navigationId}`);
      } catch (error) {
        setResult(`Rejected: ${error instanceof Error ? error.message : String(error)}`);
      }
      return;
    }
    if (productionBASCaller && !secret) {
      try {
        await analyzeElementsWithAI({ url: 'https://fixture.invalid', intent: 'find the search button' }, newGatewayRootOperationID());
        setResult('Dispatched: succeeded via openrouter');
      } catch (error) {
        setResult(`Rejected: ${error instanceof Error ? error.message : String(error)}`);
      }
      return;
    }
    const tokenState = JSON.parse(sessionStorage.getItem('vrooli.web.access-token') || '{}');
    const headers = new Headers({ 'Content-Type': 'application/json' });
    if (tokenState.accessToken) headers.set('Authorization', `Bearer ${tokenState.accessToken}`);
    const receipt = sessionStorage.getItem(AI_CONSENT_RECEIPT_KEY);
    const source = sessionStorage.getItem(AI_CONSENT_SOURCE_KEY);
    if (receipt) headers.set('X-Vrooli-AI-Consent-Grant', receipt);
    if (source) headers.set('X-Vrooli-AI-Source', source);
    const response = await fetch('/vrooli.ai_gateway.v1.routing.RoutingService/ExecuteRoute', {
      method: 'POST', headers,
      body: JSON.stringify({
        request: {
          kind: 'REQUEST_KIND_TEXT_GENERATION', role: 'chat.default', profile: 'PROFILE_REMOTE_ONLY',
          privacyClass: secret ? 'PRIVACY_CLASS_SECRET' : 'PRIVACY_CLASS_CONFIDENTIAL',
          privacyProvenance: 'consent_gateway_browser_fixture', scenario: 'consent-fixture',
          operation: 'browser-consent-conformance', requestId: crypto.randomUUID(), maxOutputTokens: 16,
        },
        inputText: secret ? 'SECRET_SENTINEL browser fixture' : 'confidential browser fixture',
      }),
    });
    const body = await response.json().catch(() => ({}));
    const evidence = body.evidence || {};
    const rejection = body.issues?.[0]?.code || evidence.status || body.message || response.status;
    setResult(response.ok && evidence.status === 'succeeded'
      ? `Dispatched: ${evidence.status} via ${evidence.selectedProvider}`
      : `Rejected: ${rejection}`);
  };
  return <main style={{ maxWidth: 720, margin: '2rem auto', fontFamily: 'sans-serif', color: '#eee', background: '#171717', padding: 24 }}>
    <h1>{productionVisionCaller ? 'BAS Playwright sidecar consent journey' : productionBASCaller ? 'BAS AIService production caller consent journey' : 'AI Gateway consent integration fixture'}</h1>
    <AIPrivacyConsentSection />
    <label style={{ display: 'block', marginTop: 20 }}><input data-testid="secret-toggle" type="checkbox" checked={secret} onChange={event => setSecret(event.target.checked)} /> Send SECRET classification</label>
    <button data-testid="send-ai" disabled={productionVisionCaller && !visionSessionID} onClick={() => void send()}>{productionVisionCaller ? 'Send through BAS VisionNavigation' : productionBASCaller ? 'Send through BAS AIService' : 'Send to real Gateway router'}</button>
    <p data-testid="request-result" role="status">{result}</p>
  </main>;
}

createRoot(document.getElementById('root')!).render(<Fixture />);
