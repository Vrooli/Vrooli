// Suite/case identity and bounded activity for explicitly admitted native work.
// Per-case deadlines remain Jest-owned; activity is not evidence of passing work.
class NativeProgress {
  constructor(_config, options = {}) {
    this.write = options.write || ((line) => process.stderr.write(line));
    this.timer = undefined;
  }
  onTestStart(test) {
    this.stop();
    this.started = Date.now();
    this.active = test.path;
    this.write(`[native] suite started ${this.active}\n`);
    this.timer = setInterval(() => {
      this.write(`[native] active ${this.active} elapsed=${Math.floor((Date.now() - this.started) / 1000)}s\n`);
    }, 15000);
  }
  onTestCaseResult(_test, result) {
    this.write(`[native] case ${result.status} ${result.fullName}\n`);
  }
  onTestResult() { this.stop(); }
  onRunComplete() { this.stop(); }
  stop() { if (this.timer) clearInterval(this.timer); this.timer = undefined; }
}
module.exports = NativeProgress;
