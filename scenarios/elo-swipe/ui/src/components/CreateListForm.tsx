import React, { useRef, useState } from 'react';
import type { CreateListPayload } from '../types';

interface CreateListFormProps {
  onSubmit: (payload: CreateListPayload) => Promise<void>;
  loading?: boolean;
}

export const CreateListForm = ({ onSubmit, loading }: CreateListFormProps) => {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [itemsText, setItemsText] = useState('');
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [error, setError] = useState<string | null>(null);

  const parseItems = (raw: string) =>
    raw
      .split('\n')
      .map((entry) => entry.trim())
      .filter(Boolean)
      .map((entry) => ({ content: entry }));

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError(null);

    const items = parseItems(itemsText);
    if (items.length < 2) {
      setError('Please provide at least two items.');
      return;
    }

    await onSubmit({ name, description, items });
    setName('');
    setDescription('');
    setItemsText('');
  };

  const handleImportClick = () => {
    fileInputRef.current?.click();
  };

  const handleImportFile = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    try {
      const text = await file.text();
      const data = JSON.parse(text);
      const list = Array.isArray(data)
        ? data
        : Array.isArray((data as { items?: unknown[] }).items)
        ? (data as { items: unknown[] }).items
        : [];

      const normalized = list
        .map((entry) => {
          if (typeof entry === 'string') return entry.trim();
          if (entry && typeof entry === 'object' && 'content' in entry) {
            const value = (entry as { content?: unknown }).content;
            return typeof value === 'string' ? value.trim() : '';
          }
          return JSON.stringify(entry);
        })
        .filter(Boolean);

      setItemsText(normalized.join('\n'));
      setError(null);
    } catch (err) {
      setError('We could not parse that JSON file. Provide an array or { "items": [] }.');
    }
  };

  return (
    <form className="create-list" onSubmit={handleSubmit}>
      <label className="field">
        <span className="field__label">List name</span>
        <input
          required
          value={name}
          onChange={(event) => setName(event.target.value)}
          placeholder="e.g. Q3 bet stack"
        />
      </label>
      <label className="field">
        <span className="field__label">Description</span>
        <textarea
          rows={2}
          value={description}
          onChange={(event) => setDescription(event.target.value)}
          placeholder="What are we prioritizing?"
        />
      </label>
      <label className="field field--full">
        <span className="field__label">Items (one per line)</span>
        <textarea
          required
          rows={10}
          value={itemsText}
          onChange={(event) => setItemsText(event.target.value)}
          placeholder={`Launch onboarding refresh\nRefine billing narrative\nShip AI concierge beta`}
        />
      </label>
      {error && <p className="form-error">{error}</p>}
      <div className="form-actions">
        <button type="button" className="btn btn-ghost" onClick={handleImportClick}>
          Import JSON
        </button>
        <button type="submit" className="btn btn-primary" disabled={loading}>
          {loading ? 'Creating...' : 'Create list'}
        </button>
      </div>
      <input ref={fileInputRef} type="file" accept="application/json" hidden onChange={handleImportFile} />
    </form>
  );
};
