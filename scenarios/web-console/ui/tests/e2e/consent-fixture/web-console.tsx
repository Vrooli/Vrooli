import React from "react";
import { createRoot } from "react-dom/client";
import "../../../src/styles.css";
import { RemoteAIConsentSettings } from "../../../src/components/launcher/RemoteAIConsentSettings";
import { generateAICommand } from "../../../src/api/ai";

function Fixture() {
  const [result, setResult] = React.useState("Ready to send a confidential test prompt.");
  const [secret, setSecret] = React.useState(false);
  return <main style={{ maxWidth: 720, margin: "2rem auto", fontFamily: "sans-serif", color: "#eee", background: "#171717", padding: 24 }}>
    <h1>Web Console privacy settings</h1>
    <RemoteAIConsentSettings />
    <label style={{ display: "block", marginTop: 20 }}><input data-testid="secret-toggle" type="checkbox" checked={secret} onChange={event => setSecret(event.target.checked)} /> Send a SECRET marker for the local-only check</label>
    <button data-testid="send-ai" onClick={() => void generateAICommand("fixture prompt", secret ? "SECRET_SENTINEL" : "confidential session fixture").then(value => setResult(`Dispatched: ${value.provider}`)).catch(error => setResult(`Rejected: ${error.message}`))}>Send confidential test prompt</button>
    <p data-testid="request-result" role="status">{result}</p>
  </main>;
}

createRoot(document.getElementById("root")!).render(<Fixture />);
