# Host requirement conformance mutations

The mutation fixtures live beside the suite tests. Each fixture changes one
contract rule at a time, and `TestSuiteRejectsEachMutant` records that the
shared suite rejects it before stamped handler tests are removed.
