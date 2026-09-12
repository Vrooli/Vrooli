import { sleep } from '../../../src/utils/timing';

describe('timing utilities', () => {
  afterEach(() => {
    jest.useRealTimers();
  });

  describe('sleep', () => {
    it('resolves immediately for zero and negative durations', async () => {
      await expect(sleep(0)).resolves.toBeUndefined();
      await expect(sleep(-25)).resolves.toBeUndefined();
    });

    it('waits for positive durations before resolving', async () => {
      jest.useFakeTimers();

      const done = jest.fn();
      const promise = sleep(250).then(done);

      await Promise.resolve();
      expect(done).not.toHaveBeenCalled();

      jest.advanceTimersByTime(249);
      await Promise.resolve();
      expect(done).not.toHaveBeenCalled();

      jest.advanceTimersByTime(1);
      await promise;
      expect(done).toHaveBeenCalledTimes(1);
    });
  });
});
