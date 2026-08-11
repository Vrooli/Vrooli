package release

import("testing";"github.com/stretchr/testify/require")
func TestProceduralReleaseDerivesDisclosureAndRequiresAltText(t *testing.T){s:=NewStore();_,err:=s.Release(Request{CandidateID:"c",StyleID:"s",Strategy:"procedural",Width:10,Height:10,LegibilityPasses:true});require.Error(t,err);b,err:=s.Release(Request{CandidateID:"c",StyleID:"s",Strategy:"procedural",Width:10,Height:10,AltText:"ambient",LegibilityPasses:true});require.NoError(t,err);require.False(t,b.AIGenerated)}
func TestReleaseRejectsDirectDisclosureAndGeometry(t *testing.T){s:=NewStore();_,err:=s.Release(Request{CandidateID:"c",StyleID:"s",Strategy:"guided",Width:9,Height:10,ExpectedWidth:10,ExpectedHeight:10,AIGeneratedSet:true,AltText:"x",LegibilityPasses:true});require.Error(t,err);require.Contains(t,err.Error(),"ai_generated")}
