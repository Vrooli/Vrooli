import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import type { LucideIcon } from 'lucide-react';
import {
  Bot,
  CheckCircle2,
  Clipboard,
  Filter,
  FolderPlus,
  ShieldOff,
  Sparkles,
} from 'lucide-react';

import apiClient from '../utils/api';
import type { Chatbot } from '../types';

const FILTERS: FilterOption[] = [
  { id: 'all', label: 'All chatbots', icon: Bot },
  { id: 'active', label: 'Active', icon: CheckCircle2 },
  { id: 'inactive', label: 'Inactive', icon: ShieldOff },
];

interface FilterOption {
  id: 'all' | 'active' | 'inactive';
  label: string;
  icon: LucideIcon;
}

function ChatbotList() {
  const [loading, setLoading] = useState(true);
  const [chatbots, setChatbots] = useState<Chatbot[]>([]);
  const [filter, setFilter] = useState<FilterOption['id']>('all');

  useEffect(() => {
    let isMounted = true;

    const loadChatbots = async () => {
      try {
        setLoading(true);
        const response = await apiClient.get('/api/v1/chatbots');
        if (!response.ok) {
          return;
        }

        const payload = (await response.json()) as Chatbot[];
        if (isMounted) {
          setChatbots(payload);
        }
      } catch (error) {
        console.error('Failed to load chatbots:', error);
      } finally {
        if (isMounted) {
          setLoading(false);
        }
      }
    };

    loadChatbots();
    return () => {
      isMounted = false;
    };
  }, []);

  const filteredChatbots = useMemo(() => {
    if (filter === 'active') {
      return chatbots.filter((chatbot) => chatbot.is_active);
    }
    if (filter === 'inactive') {
      return chatbots.filter((chatbot) => !chatbot.is_active);
    }
    return chatbots;
  }, [chatbots, filter]);

  const formatDate = (isoDate: string) => {
    if (!isoDate) return '—';
    return new Date(isoDate).toLocaleDateString(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  };

  const copyEmbedCode = (chatbotId: string) => {
    const embedSnippet = `<script src="${window.location.origin}/widget.js" data-chatbot-id="${chatbotId}"></script>`;
    navigator.clipboard
      .writeText(embedSnippet)
      .then(() => alert('Embed code copied. Paste it into your marketing site to deploy.'))
      .catch(() => alert('Unable to copy automatically. Please copy the snippet manually.'));
  };

  return (
    <div className="page-stack">
      <header className="page-header">
        <div>
          <h2>Chatbot portfolio</h2>
          <p>Deploy, optimize, and govern every AI assistant from a single command center.</p>
        </div>
        <Link to="/chatbots/new" className="primary-action">
          <FolderPlus size={16} />
          <span>Create chatbot</span>
        </Link>
      </header>

      <div className="filter-toolbar">
        <div className="filter-label">
          <Filter size={16} />
          <span>Filter by status</span>
        </div>
        <div className="filter-pills">
          {FILTERS.map((option) => {
            const isActive = option.id === filter;
            const Icon = option.icon;
            const count =
              option.id === 'active'
                ? chatbots.filter((chatbot) => chatbot.is_active).length
                : option.id === 'inactive'
                ? chatbots.filter((chatbot) => !chatbot.is_active).length
                : chatbots.length;

            return (
              <button
                key={option.id}
                className={`filter-pill ${isActive ? 'is-active' : ''}`}
                onClick={() => setFilter(option.id)}
              >
                <Icon size={14} />
                <span>{option.label}</span>
                <span className="pill-count">{count}</span>
              </button>
            );
          })}
        </div>
      </div>

      {loading ? (
        <div className="card loading-card">
          <div className="loading-spinner" />
          <p>Loading deployed chatbots…</p>
        </div>
      ) : filteredChatbots.length === 0 ? (
        <div className="card">
            <div className="empty-state">
              <div className="empty-icon">
                <Sparkles size={20} />
            </div>
            <h4>No chatbots match this filter</h4>
            <p>Adjust the filters or create a new chatbot aligned with your next GTM experiment.</p>
            <Link to="/chatbots/new" className="ghost-button">
              <FolderPlus size={16} />
              <span>Blueprint a chatbot</span>
            </Link>
          </div>
        </div>
      ) : (
        <section className="chatbot-grid">
          {filteredChatbots.map((chatbot) => (
            <article key={chatbot.id} className="card chatbot-card">
              <header className="card-header">
                <div className="chatbot-headline">
                  <div className="chatbot-avatar" aria-hidden>
                    <Bot size={18} />
                  </div>
                  <div>
                    <h3>{chatbot.name}</h3>
                    <p>{chatbot.description || 'No description provided yet.'}</p>
                  </div>
                </div>
                <span className={`status-pill ${chatbot.is_active ? 'is-success' : 'is-muted'}`}>
                  {chatbot.is_active ? 'Active' : 'Inactive'}
                </span>
              </header>

              <dl className="chatbot-meta">
                <div>
                  <dt>Model</dt>
                  <dd>{(chatbot.model_config?.model as string) || 'llama3.2'}</dd>
                </div>
                <div>
                  <dt>Created</dt>
                  <dd>{formatDate(chatbot.created_at)}</dd>
                </div>
                <div>
                  <dt>Updated</dt>
                  <dd>{formatDate(chatbot.updated_at)}</dd>
                </div>
              </dl>

              <div className="chatbot-persona">
                <h4>Personality</h4>
                <p>{chatbot.personality?.length > 160 ? `${chatbot.personality.slice(0, 160)}…` : chatbot.personality}</p>
              </div>

              <footer className="chatbot-footer">
                <div className="chatbot-actions">
                  <Link to={`/chatbots/${chatbot.id}/test`} className="pill-button">
                    Test
                  </Link>
                  <Link to={`/chatbots/${chatbot.id}/analytics`} className="pill-button">
                    Analytics
                  </Link>
                  <Link to={`/chatbots/${chatbot.id}/edit`} className="pill-button">
                    Edit
                  </Link>
                </div>
                <button className="ghost-button" onClick={() => copyEmbedCode(chatbot.id)}>
                  <Clipboard size={16} />
                  <span>Copy embed</span>
                </button>
              </footer>
            </article>
          ))}
        </section>
      )}
    </div>
  );
}

export default ChatbotList;
