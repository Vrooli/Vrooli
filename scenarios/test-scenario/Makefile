.PHONY: run test test-unit test-integration test-performance test-dependencies test-business logs stop clean

run:
	@echo "Starting test-scenario"

test:
	./test/run-tests.sh

test-unit:
	./test/phases/test-unit.sh

test-integration:
	./test/phases/test-integration.sh

test-performance:
	./test/phases/test-performance.sh

test-dependencies:
	./test/phases/test-dependencies.sh

test-business:
	./test/phases/test-business.sh

logs:
	@echo "Displaying logs for test-scenario"

stop:
	@echo "Stopping test-scenario"

clean:
	@echo "Cleaning test artifacts"
	rm -rf coverage/
	rm -f api/test-scenario api/test-scenario.test
	find . -name "*.test" -type f -delete