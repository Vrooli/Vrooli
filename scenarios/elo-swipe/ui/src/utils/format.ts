export const formatContent = (content: unknown): string => {
  if (typeof content === 'string') return content;
  if (!content) return '';

  if (typeof content === 'object') {
    const record = content as Record<string, unknown>;
    if (typeof record.title === 'string') return record.title;
    if (typeof record.name === 'string') return record.name;
    if (typeof record.text === 'string') return record.text;
    if (Array.isArray(record.tags)) return record.tags.join(', ');
  }

  return JSON.stringify(content);
};
