import { FormEvent, useEffect, useRef, useState } from "react";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";

type Message = {
  id: string;
  text: string;
  author: "human" | "agent";
  media?: { name: string; mime: string; size: number; url?: string }[];
};

const threadKey = "in-app-default";

function socketURL() {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${window.location.host}/api/v1/channels/socket?thread_key=${encodeURIComponent(threadKey)}`;
}

export function ConversationsPage() {
  const { t } = useTranslation();
  const [messages, setMessages] = useState<Message[]>([]);
  const [text, setText] = useState("");
  const [attachment, setAttachment] = useState<File>();
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string>();
  const socket = useRef<WebSocket>();

  useEffect(() => {
    const connection = new WebSocket(socketURL());
    socket.current = connection;
    connection.onopen = () => { setConnected(true); setError(undefined); };
    connection.onclose = () => setConnected(false);
    connection.onerror = () => setError(t(strings.console.conversations.disconnectError));
    connection.onmessage = (event) => {
      try {
        const payload = JSON.parse(String(event.data)) as { text?: string; media?: Message["media"]; error?: string };
        if (payload.error) { setError(payload.error); return; }
        if (payload.text || payload.media?.length) {
          setMessages((current) => [...current, { id: `agent-${Date.now()}`, text: payload.text ?? "", author: "agent", media: payload.media }]);
        }
      } catch { setError(t(strings.console.conversations.unreadableError)); }
    };
    return () => { connection.close(); socket.current = undefined; };
  }, [t]);

  const send = (event: FormEvent) => {
    event.preventDefault();
    const value = text.trim();
    if ((!value && !attachment) || !connected || socket.current?.readyState !== WebSocket.OPEN) return;
    const media = attachment ? [{ name: attachment.name, mime: attachment.type || "application/octet-stream", size: attachment.size, url: URL.createObjectURL(attachment) }] : undefined;
    socket.current.send(JSON.stringify({ channel_id: "in-app", remote_message_id: `${Date.now()}-${Math.random()}`, thread_key: threadKey, sender_address: "browser", author_kind: "human", text: value, media }));
    setMessages((current) => [...current, { id: `human-${Date.now()}`, text: value, author: "human", media }]);
    setText("");
    setAttachment(undefined);
  };

  const transcriptState = error ? "error" : connected ? (messages.length ? "ready" : "empty") : "loading";
  return <section aria-labelledby="conversations-heading" className="flex flex-col gap-4">
    <h2 id="conversations-heading" className="text-2xl font-semibold">{t(strings.console.conversations.title)}</h2>
    <p className="text-app-muted-foreground">{t(strings.console.conversations.description)}</p>
    <ExperienceSurface surfaceId="thread-list-region" state={messages.length ? "ready" : "empty"} className="rounded-lg border p-4">
    <ExperienceSurface surfaceId="transcript-region" state={transcriptState} className="rounded-lg border p-4">
      <div className="mb-3 flex items-center justify-between"><h3 className="font-semibold">{t(strings.console.conversations.thread)}</h3><span role="status">{connected ? t(strings.console.conversations.connected) : t(strings.console.conversations.offline)}</span></div>
      {error && <p role="alert" className="mb-3 rounded border border-red-300 p-2 text-sm">{error}</p>}
      <div data-testid="conversation-transcript" role="log" aria-live="polite" className="flex min-h-32 flex-col gap-2">
        {messages.length === 0 && <p data-experience-state="empty" className="text-sm text-app-muted-foreground">{t(strings.console.conversations.empty)}</p>}
        {messages.map((message) => <article key={message.id} data-testid={`conversation-message-${message.author}`} className={`rounded p-2 ${message.author === "agent" ? "bg-muted" : "bg-primary/10"}`}>
          <span className="mr-2 text-xs font-semibold uppercase">{message.author === "agent" ? t(strings.console.conversations.agent) : t(strings.console.conversations.you)}</span>{message.text}
          {message.media?.map((item) => <p key={item.name} className="text-xs">{t(strings.console.conversations.attachment, item)}</p>)}
        </article>)}
      </div>
    </ExperienceSurface>
    </ExperienceSurface>
    <form data-testid="conversation-composer" aria-label={t(strings.console.conversations.send)} onSubmit={send} className="flex flex-col gap-2 rounded-lg border p-4">
      <label htmlFor="conversation-input" className="text-sm font-semibold">{t(strings.console.conversations.message)}</label>
      <textarea id="conversation-input" value={text} onChange={(event) => setText(event.target.value)} disabled={!connected} className="min-h-20 rounded border p-2" placeholder={t(strings.console.conversations.writePlaceholder)} />
      <div className="flex flex-wrap items-center gap-2">
        <label className="rounded border px-3 py-2 text-sm"><span>{t(strings.console.conversations.attachFile)}</span><input type="file" className="sr-only" onChange={(event) => setAttachment(event.target.files?.[0])} disabled={!connected} /></label>
        {attachment && <span className="text-sm">{attachment.name}</span>}
        <button type="submit" disabled={!connected || (!text.trim() && !attachment)} className="rounded bg-primary px-3 py-2 text-primary-foreground">{t(strings.console.conversations.send)}</button>
      </div>
    </form>
  </section>;
}
