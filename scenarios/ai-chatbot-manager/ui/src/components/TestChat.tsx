import { useEffect, useMemo, useRef, useState, KeyboardEvent } from 'react';
import { useParams } from 'react-router-dom';
import {
  AlertTriangle,
  Bot,
  CheckCircle2,
  Loader2,
  MessageCircle,
  Send,
  UserRound,
} from 'lucide-react';

import apiClient from '../utils/api';
import type { Chatbot } from '../types';

interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  metadata?: {
    confidence?: number;
    should_escalate?: boolean;
    conversation_id?: string;
  };
}

function TestChat() {
  const { id } = useParams();
  const [chatbot, setChatbot] = useState<Chatbot | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId] = useState(() => `test-session-${Date.now()}`);
  const messagesEndRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    let isMounted = true;

    const initialise = async () => {
      if (!id) return;
      try {
        const response = await apiClient.get(`/api/v1/chatbots/${id}`);
        if (!response.ok) {
          alert('Unable to load chatbot profile.');
          return;
        }

        const profile = (await response.json()) as Chatbot;
        if (isMounted) {
          setChatbot(profile);
          setMessages([
            {
              id: `${Date.now()}-welcome`,
              role: 'assistant',
              content: "Hi! I'm ready to explore conversations with you. How should we get started?",
              timestamp: new Date(),
            },
          ]);
        }
      } catch (error) {
        console.error('Failed to load chatbot:', error);
        alert('Unable to connect to the chatbot API.');
      }
    };

    initialise();
    return () => {
      isMounted = false;
    };
  }, [id]);

  useEffect(() => {
    if (messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages]);

  const formatTime = (timestamp: Date) =>
    timestamp.toLocaleTimeString(undefined, {
      hour: '2-digit',
      minute: '2-digit',
    });

  const sendMessage = async () => {
    if (!id || !inputValue.trim() || isLoading) {
      return;
    }

    const userMessage: ChatMessage = {
      id: `${Date.now()}-user`,
      role: 'user',
      content: inputValue.trim(),
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setInputValue('');
    setIsLoading(true);

    try {
      const response = await apiClient.post(`/api/v1/chat/${id}`, {
        message: userMessage.content,
        session_id: sessionId,
        context: { source: 'test-interface' },
      });

      if (response.ok) {
        const payload = (await response.json()) as {
          response: string;
          confidence?: number;
          should_escalate?: boolean;
          conversation_id?: string;
        };

        const assistantMessage: ChatMessage = {
          id: `${Date.now()}-assistant`,
          role: 'assistant',
          content: payload.response,
          timestamp: new Date(),
          metadata: {
            confidence: payload.confidence,
            should_escalate: payload.should_escalate,
            conversation_id: payload.conversation_id,
          },
        };

        setMessages((prev) => [...prev, assistantMessage]);
      } else {
        setMessages((prev) => [
          ...prev,
          {
            id: `${Date.now()}-error`,
            role: 'assistant',
            content: 'The API responded with an error. Retry or inspect server logs.',
            timestamp: new Date(),
          },
        ]);
      }
    } catch (error) {
      console.error('Failed to send message:', error);
      setMessages((prev) => [
        ...prev,
        {
          id: `${Date.now()}-network`,
          role: 'assistant',
          content: "We couldn't reach the API. Please confirm the scenario is running and try again.",
          timestamp: new Date(),
        },
      ]);
    } finally {
      setIsLoading(false);
    }
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      sendMessage();
    }
  };

  const clearChat = () => {
    if (window.confirm('Clear this conversation history?')) {
      setMessages([
        {
          id: `${Date.now()}-welcome`,
          role: 'assistant',
          content: 'Chat cleared. What scenario should we test next?',
          timestamp: new Date(),
        },
      ]);
    }
  };

  const statusPill = useMemo(() => {
    if (!chatbot) return null;
    return chatbot.is_active ? (
      <span className="status-pill is-success">
        <CheckCircle2 size={14} /> Active
      </span>
    ) : (
      <span className="status-pill is-muted">
        <AlertTriangle size={14} /> Inactive
      </span>
    );
  }, [chatbot]);

  if (!chatbot) {
    return (
      <div className="card loading-card">
        <div className="loading-spinner" />
        <p>Loading chatbot playground…</p>
      </div>
    );
  }

  return (
    <div className="page-stack">
      <header className="page-header">
        <div>
          <h2>Conversation sandbox</h2>
          <p>Prototype new prompts, escalation sequences, and tone before deploying to production.</p>
        </div>
        <button className="ghost-button" onClick={clearChat}>
          <MessageCircle size={16} />
          <span>Reset chat</span>
        </button>
      </header>

      <section className="chat-layout">
        <aside className="card chatbot-summary">
          <header>
            <div className="chatbot-avatar" aria-hidden>
              <Bot size={20} />
            </div>
            <div>
              <h3>{chatbot.name}</h3>
              {statusPill}
            </div>
          </header>
          <dl>
            <div>
              <dt>Model</dt>
              <dd>{(chatbot.model_config?.model as string) || 'llama3.2'}</dd>
            </div>
            <div>
              <dt>Created</dt>
              <dd>{new Date(chatbot.created_at).toLocaleDateString()}</dd>
            </div>
            {chatbot.description && (
              <div>
                <dt>Mission</dt>
                <dd>{chatbot.description}</dd>
              </div>
            )}
          </dl>
          <div className="persona-block">
            <h4>Personality</h4>
            <p>{chatbot.personality?.slice(0, 240) || 'No personality configured yet.'}</p>
          </div>
        </aside>

        <div className="card chat-surface">
          <div className="chat-stream">
            {messages.map((message) => (
              <ChatBubble key={message.id} message={message} formatTime={formatTime} />
            ))}
            {isLoading && (
              <div className="chat-bubble assistant">
                <div className="avatar">
                  <Bot size={16} />
                </div>
                <div className="bubble typing">
                  <span />
                  <span />
                  <span />
                </div>
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>
          <div className="chat-composer">
            <textarea
              value={inputValue}
              onChange={(event) => setInputValue(event.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Draft the next message…"
              rows={2}
              disabled={isLoading}
            />
            <button className="primary-action" onClick={sendMessage} disabled={!inputValue.trim() || isLoading}>
              {isLoading ? <Loader2 size={16} className="spin" /> : <Send size={16} />}
              <span>Send</span>
            </button>
            <p className="composer-hint">Press Enter to send · Shift + Enter for newline</p>
          </div>
        </div>
      </section>
    </div>
  );
}

interface ChatBubbleProps {
  message: ChatMessage;
  formatTime: (timestamp: Date) => string;
}

function ChatBubble({ message, formatTime }: ChatBubbleProps) {
  const isUser = message.role === 'user';
  const Icon = isUser ? UserRound : Bot;
  const confidence = message.metadata?.confidence;
  const shouldEscalate = message.metadata?.should_escalate;

  return (
    <div className={`chat-bubble ${isUser ? 'user' : 'assistant'}`}>
      <div className="avatar">
        <Icon size={16} />
      </div>
      <div className="bubble">
        <p>{message.content}</p>
        <div className="bubble-meta">
          <span>{formatTime(message.timestamp)}</span>
          {typeof confidence === 'number' && (
            <span>Confidence: {(confidence * 100).toFixed(0)}%</span>
          )}
          {shouldEscalate && (
            <span className="escalate">
              <AlertTriangle size={12} /> Escalate recommended
            </span>
          )}
        </div>
      </div>
    </div>
  );
}

export default TestChat;
