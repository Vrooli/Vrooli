package main

import "sync"

var runtimeInitOnce sync.Once

func initializeScenarioAuditorRuntime() {
	runtimeInitOnce.Do(func() {
		ruleStateStore.enablePersistence()
		protectedScenariosStore.enablePersistence()
		standardsStore.enablePersistence()
		vulnStore.enablePersistence()
		automatedFixStore.enablePersistence()
		registerDefaultExternalProviders()
	})
}
