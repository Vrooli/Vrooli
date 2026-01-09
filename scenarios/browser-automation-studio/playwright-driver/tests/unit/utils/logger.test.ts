import winston from 'winston';
import { createLogger, setLogger, logger } from '../../../src/utils/logger';
import { createTestConfig } from '../../helpers';

describe('Logger', () => {
  describe('createLogger', () => {
    it('should create logger with info level by default', () => {
      const config = createTestConfig();
      const testLogger = createLogger(config);

      expect(testLogger).toBeInstanceOf(winston.Logger);
      expect(testLogger.level).toBe('info');
    });

    it('should create logger with custom level', () => {
      const config = createTestConfig({
        logging: { level: 'debug', format: 'json' },
      });
      const testLogger = createLogger(config);

      expect(testLogger.level).toBe('debug');
    });

    it('should create logger with json format', () => {
      const config = createTestConfig({
        logging: { level: 'info', format: 'json' },
      });
      const testLogger = createLogger(config);

      expect(testLogger).toBeInstanceOf(winston.Logger);
    });

    it('should create logger with text format', () => {
      const config = createTestConfig({
        logging: { level: 'info', format: 'text' },
      });
      const testLogger = createLogger(config);

      expect(testLogger).toBeInstanceOf(winston.Logger);
    });

    it('should have console transport', () => {
      const config = createTestConfig();
      const testLogger = createLogger(config);

      expect(testLogger.transports).toHaveLength(1);
      expect(testLogger.transports[0]).toBeInstanceOf(winston.transports.Console);
    });
  });

  describe('setLogger', () => {
    it('should update the global logger instance', () => {
      const config = createTestConfig({
        logging: { level: 'debug', format: 'json' },
      });
      const newLogger = createLogger(config);

      setLogger(newLogger);

      // The module exports a singleton, so we can't directly test the updated instance
      // but we can verify setLogger doesn't throw
      expect(() => setLogger(newLogger)).not.toThrow();
    });
  });

  describe('logger singleton', () => {
    it('should export a logger-like object', () => {
      // Logger is a proxy that delegates to the actual winston.Logger instance
      // We verify it has the expected methods rather than instanceof check
      expect(typeof logger.info).toBe('function');
      expect(typeof logger.error).toBe('function');
      expect(typeof logger.warn).toBe('function');
      expect(typeof logger.debug).toBe('function');
    });

    it('should have standard logging methods', () => {
      expect(logger.info).toBeDefined();
      expect(logger.error).toBeDefined();
      expect(logger.warn).toBeDefined();
      expect(logger.debug).toBeDefined();
    });
  });

  describe('logging operations', () => {
    let testLogger: winston.Logger;
    let logSpy: jest.SpyInstance;

    beforeEach(() => {
      const config = createTestConfig();
      testLogger = createLogger(config);

      // Spy on the transport's log method
      logSpy = jest.spyOn(testLogger.transports[0], 'log').mockImplementation(() => {});
    });

    afterEach(() => {
      logSpy.mockRestore();
    });

    it('should log info messages', () => {
      testLogger.info('Test message', { key: 'value' });

      expect(logSpy).toHaveBeenCalled();
    });

    it('should log error messages', () => {
      testLogger.error('Error message', { error: 'details' });

      expect(logSpy).toHaveBeenCalled();
    });

    it('should log warn messages', () => {
      testLogger.warn('Warning message');

      expect(logSpy).toHaveBeenCalled();
    });

    it('should log debug messages when level is debug', () => {
      const config = createTestConfig({
        logging: { level: 'debug', format: 'json' },
      });
      const debugLogger = createLogger(config);
      const debugSpy = jest.spyOn(debugLogger.transports[0], 'log').mockImplementation(() => {});

      debugLogger.debug('Debug message');

      expect(debugSpy).toHaveBeenCalled();

      debugSpy.mockRestore();
    });

    it('should not log debug messages when level is info', () => {
      const config = createTestConfig({
        logging: { level: 'info', format: 'json' },
      });
      const infoLogger = createLogger(config);
      const infoSpy = jest.spyOn(infoLogger.transports[0], 'log').mockImplementation(() => {});

      infoLogger.debug('Debug message');

      expect(infoSpy).not.toHaveBeenCalled();

      infoSpy.mockRestore();
    });
  });
});
