.PHONY: run test test-unit test-integration test-performance test-dependencies test-business logs stop clean

run:
	@echo "Starting test-scenario"

test:
	test-genie execute test-scenario --preset comprehensive

test-unit:
	test-genie execute test-scenario --phases unit

test-integration:
	test-genie execute test-scenario --phases integration

test-performance:
	test-genie execute test-scenario --phases performance

test-dependencies:
	test-genie execute test-scenario --phases dependencies

test-business:
	test-genie execute test-scenario --phases business

logs:
	@echo "Displaying logs for test-scenario"

stop:
	@echo "Stopping test-scenario"

clean:
	@echo "Cleaning test artifacts"
	rm -rf coverage/
	rm -f api/test-scenario api/test-scenario.test
	find . -name "*.test" -type f -delete
