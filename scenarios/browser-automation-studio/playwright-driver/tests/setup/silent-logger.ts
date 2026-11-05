import { createNoOpLogger, setLogger } from '../../src/utils/logger';
import winston from 'winston';

let consoleSpies: jest.SpyInstance[] = [];

beforeEach(() => {
  setLogger(createNoOpLogger());

  if (process.env.BAS_JEST_VERBOSE_LOGS === '1') {
    return;
  }

  consoleSpies = [
    jest.spyOn(console, 'log').mockImplementation(() => undefined),
    jest.spyOn(console, 'warn').mockImplementation(() => undefined),
    jest.spyOn(console, 'error').mockImplementation(() => undefined),
    jest.spyOn(winston.transports.Console.prototype, 'log').mockImplementation((_info, next) => {
      next?.();
    }),
  ];
});

afterEach(() => {
  for (const spy of consoleSpies) {
    spy.mockRestore();
  }
  consoleSpies = [];
});
