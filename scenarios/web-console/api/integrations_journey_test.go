//go:build journey

package main

import "testing"

func TestJ14_CapabilityAndDependencyViewsAgree(t *testing.T) { launcherJourney(t, t.Name()) }
func TestJ14_InstalledVersionsShown(t *testing.T)            { launcherJourney(t, t.Name()) }
func TestJ14_EachIntegrationSaysWhatItDoes(t *testing.T)     { launcherJourney(t, t.Name()) }
func TestJ14_AgenticFeaturesShown(t *testing.T)              { launcherJourney(t, t.Name()) }
func TestJ14_OldMappingDeleted(t *testing.T)                 { launcherJourney(t, t.Name()) }
func TestJ14_CreditsRowShown(t *testing.T)                   { launcherJourney(t, t.Name()) }
func TestJ14_SecondCapabilitySamePattern(t *testing.T)       { launcherJourney(t, t.Name()) }
