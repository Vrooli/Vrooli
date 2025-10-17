import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../../utils/api', () => ({
  apiJsonRequest: vi.fn(),
  apiVoidRequest: vi.fn(),
}));

import { updateIssue } from '../issues';
import { apiJsonRequest } from '../../utils/api';

describe('issues service', () => {
  beforeEach(() => {
    vi.mocked(apiJsonRequest).mockReset();
  });

  it('sends trimmed payload when updating an issue', async () => {
    vi.mocked(apiJsonRequest).mockResolvedValue({ data: { issue: null } });

    await updateIssue('http://localhost/api/v1', {
      issueId: 'ISSUE-42',
      title: '  Updated title  ',
      description: '  Details here  ',
      priority: 'High',
      status: 'active',
      appId: '  web-ui  ',
      tags: ['regression', ' ui '],
      reporterName: '  Morgan  ',
      reporterEmail: ' morgan@example.com ',
      attachments: [
        {
          name: 'log.txt',
          content: 'ZmlsZQ==',
          encoding: 'base64',
          contentType: 'text/plain',
        },
      ],
    });

    expect(apiJsonRequest).toHaveBeenCalledTimes(1);
    const [url, options] = vi.mocked(apiJsonRequest).mock.calls[0];
    expect(url).toBe('http://localhost/api/v1/issues/ISSUE-42');
    expect(options?.method).toBe('PATCH');
    expect(options?.headers).toMatchObject({ 'Content-Type': 'application/json' });

    const body = JSON.parse((options?.body as string) ?? '{}');
    expect(body).toMatchObject({
      title: 'Updated title',
      description: 'Details here',
      priority: 'high',
      status: 'active',
      app_id: 'web-ui',
      tags: ['regression', ' ui '],
      reporter: {
        name: 'Morgan',
        email: 'morgan@example.com',
      },
    });
    expect(body.artifacts).toEqual([
      {
        name: 'log.txt',
        category: 'attachment',
        content: 'ZmlsZQ==',
        encoding: 'base64',
        content_type: 'text/plain',
      },
    ]);
  });

  it('omits optional fields when not provided', async () => {
    vi.mocked(apiJsonRequest).mockResolvedValue({ data: { issue: null } });

    await updateIssue('http://localhost/api/v1', {
      issueId: 'ISSUE-101',
    });

    const [, options] = vi.mocked(apiJsonRequest).mock.calls[0];
    const body = JSON.parse((options?.body as string) ?? '{}');
    expect(body).toEqual({});
  });
});
