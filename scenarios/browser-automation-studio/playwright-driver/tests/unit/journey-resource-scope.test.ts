async function loadJourneyResourceScope(): Promise<typeof import('../journeys/support').JourneyResourceScope> {
  process.env.BAS_API_URL ??= 'http://127.0.0.1:1';
  process.env.BAS_DRIVER_URL ??= 'http://127.0.0.1:1';
  process.env.BAS_FIXTURE_ORIGIN ??= 'http://127.0.0.1:1';
  return (await import('../journeys/support')).JourneyResourceScope;
}

describe('JourneyResourceScope', () => {
  it('attempts every cleanup in reverse order and reports every failure', async () => {
    const JourneyResourceScope = await loadJourneyResourceScope();
    const scope = new JourneyResourceScope();
    const attempted: string[] = [];
    scope.add('first resource', async () => {
      attempted.push('first');
      throw new Error('first failed');
    });
    scope.add('second resource', async () => {
      attempted.push('second');
      throw new Error('second failed');
    });
    scope.add('third resource', async () => {
      attempted.push('third');
    });

    let failure: unknown;
    try {
      await scope.run(async () => 'body result');
    } catch (error) {
      failure = error;
    }

    expect(attempted).toEqual(['third', 'second', 'first']);
    expect(failure).toBeInstanceOf(AggregateError);
    expect((failure as AggregateError).errors).toHaveLength(2);
    expect((failure as AggregateError).errors.map((error: Error) => error.message)).toEqual([
      'Could not second resource',
      'Could not first resource',
    ]);
  });

  it('keeps a behavioral failure visible alongside cleanup failures', async () => {
    const JourneyResourceScope = await loadJourneyResourceScope();
    const scope = new JourneyResourceScope();
    const bodyFailure = new Error('behavior assertion failed');
    const cleanupFailure = new Error('cleanup failed');
    scope.add('recording session', async () => { throw cleanupFailure; });

    let failure: unknown;
    try {
      await scope.run(async () => { throw bodyFailure; });
    } catch (error) {
      failure = error;
    }

    expect(failure).toBeInstanceOf(AggregateError);
    expect((failure as AggregateError).errors).toEqual([
      bodyFailure,
      expect.objectContaining({ message: 'Could not recording session', cause: cleanupFailure }),
    ]);
  });
});
